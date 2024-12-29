/*
Copyright 2024.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controller

import (
	"context"
	"strings"
	"time"

	"errors"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	"dcnlab.ssu.ac.kr/kt-cloud-operator/api/v1beta1"
	infrastructurev1beta1 "dcnlab.ssu.ac.kr/kt-cloud-operator/api/v1beta1"
	"dcnlab.ssu.ac.kr/kt-cloud-operator/cmd/httpapi"
)

// KTMachineReconciler reconciles a KTMachine object
type KTMachineReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=infrastructure.dcnlab.ssu.ac.kr,resources=ktmachines,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=infrastructure.dcnlab.ssu.ac.kr,resources=ktmachines/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=infrastructure.dcnlab.ssu.ac.kr,resources=ktmachines/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the KTMachine object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.19.1/pkg/reconcile
func (r *KTMachineReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx, "LogFrom", "KTMachine")
	logger.V(1).Info("KTMachine Reconcile", "KTMachine", req)

	ktMachine := &v1beta1.KTMachine{}
	if err := r.Get(ctx, req.NamespacedName, ktMachine); err != nil {
		if apierrors.IsNotFound(err) {
			logger.Info("KTMachine resource not found. Ignoring since it must be deleted")
			return ctrl.Result{}, nil
		}
		logger.Error(err, "Failed to get KTMachine resource")
		return ctrl.Result{}, err
	}

	//first get the token associated for the cluster and find token
	subjectToken, err := r.getSubjectToken(ctx, ktMachine, req)
	if err != nil {
		logger.Error(err, "Failed to find KTSubject token matching cluster")
		return ctrl.Result{RequeueAfter: time.Minute}, nil
	}

	if subjectToken == "" {
		logger.Error(err, "We have to reconcile again to check the Subject token")
		return ctrl.Result{RequeueAfter: time.Minute}, nil
	}

	// check if current machine is control plane
	machineName := ktMachine.Name
	substring := "control-plane"

	cluster, err := r.GetMachineAssociatedCluster(ctx, ktMachine, req)
	if cluster == nil || err != nil {
		if cluster == nil {
			logger.Error(errors.New("cluster empty from get-associated-cluster for machine"), "Failed to retrieve cluster for Machine")
			return ctrl.Result{RequeueAfter: time.Minute}, nil
		} else if err != nil {
			logger.Error(err, "Failed to retrieve cluster for Machine")
			return ctrl.Result{RequeueAfter: time.Minute}, nil
		}
	}

	//trigger to create machine on KTCloud by calling API
	if ktMachine.Status.ID == "" && strings.Contains(machineName, substring) {
		logger.Info("Machine has no ID in the status field, create it on KT Cloud")

		err = httpapi.CreateVM(ktMachine, subjectToken)
		if err != nil {
			logger.Error(err, "Failed to create VM on KT Cloud during API Call")
			return ctrl.Result{RequeueAfter: time.Minute}, nil
		}

		//use the response to from the api and update the machine
		return ctrl.Result{RequeueAfter: time.Minute}, nil
	} else if strings.Contains(machineName, substring) {
		logger.Info("Machine already created and has ID")
		//call API and check if machine is ready
		serverResponse, err := httpapi.GetCreatedVM(ktMachine, subjectToken)
		if err != nil || serverResponse == nil {
			logger.Error(err, "Failed to query VM on KT Cloud during API Call")
			return ctrl.Result{RequeueAfter: time.Minute}, nil
		}

		logger.Info("Got the machine we have to update if the states dont match")
		if ktMachine.Status.Status != serverResponse.Status {
			ktMachine.Status = *serverResponse
			if err := r.Status().Update(ctx, ktMachine); err != nil {
				logger.Error(err, "Can't update for machine with status on cloud")
				return ctrl.Result{RequeueAfter: time.Minute}, nil
			}

		}

		logger.Info("Machine state is not creating")
		logger.Info("The status is the same on cloud and cluster")
		// logger.Info("Do we need to reconcile again when the machine is all ready?")
		//what happens if 404 on cloud but present on cluster?

		//check if machine is control plane and get kubeadm data
		//if we already kubeadm data, join worker nodes if not joined

		//we have to attach public IP to all control planes

		if strings.Contains(machineName, substring) {
			logger.Info("The machine name contains 'control-plane', therefore Control Plane.")
			//attach public IP

			//CHECK NETWORK STUFF ONLY IF THE MACHINE STATUS IS ACTIVE
			if ktMachine.Status.Status == "ACTIVE" {

				if cluster.Spec.ControlPlaneExternalNetworkEnable && len(ktMachine.Status.AssignedPublicIps) == 0 {
					err = httpapi.AttachPublicIP(ktMachine, subjectToken)
					if err != nil {
						logger.Error(err, "Failed to attach network to Machine")
						return ctrl.Result{RequeueAfter: time.Minute}, nil
					}
					//we have to fix firewall settings
					if cluster.Spec.ManagedSecurityGroups.ControlPlaneRules.Direction != "" {

						//rules to apply
						err = httpapi.AddFirewallSettings(ktMachine, subjectToken, cluster.Spec.ManagedSecurityGroups.ControlPlaneRules, cluster.Spec.ManagedSecurityGroups.EnableOutboundInternetTraffic)
						if err != nil {
							logger.Error(err, "Failed to add firewall settings for the cluster")
							return ctrl.Result{RequeueAfter: time.Minute}, nil
						}
					}

					//we are done with the control plane
					ktMachine.Status.ControlPlaneRef.Type = "BootstrapReady"
					ktMachine.Status.ControlPlaneRef.Status = false
					if err := r.Status().Update(ctx, ktMachine); err != nil {
						logger.Error(err, "Can't update for bootstrap ready control-plane to be false")
						return ctrl.Result{RequeueAfter: time.Minute / 2}, nil
					}

				}
				logger.Info("Skip adding public IP address to Machine already added")
				//check controlPlane state
				if ktMachine.Status.ControlPlaneRef.Type == "BootstrapReady" && !ktMachine.Status.ControlPlaneRef.Status {
					// try to curl the API server if it is ready for join
					if err := httpapi.CheckControlPlaneMachineReady(ktMachine); err != nil {
						logger.Error(err, "Control Plane not ready yet")
						return ctrl.Result{RequeueAfter: time.Minute / 2}, nil
					}

					ktMachine.Status.ControlPlaneRef.Type = "BootstrapReady"
					ktMachine.Status.ControlPlaneRef.Status = true
					if err := r.Status().Update(ctx, ktMachine); err != nil {
						logger.Error(err, "Can't update for bootstrap ready control-plane")
						return ctrl.Result{RequeueAfter: time.Minute / 2}, nil
					}

				}
				return ctrl.Result{}, nil
			}
			logger.Info("Machine is not yet ready, we have to reconcile")
			return ctrl.Result{RequeueAfter: time.Minute / 2}, nil
		} else {
			logger.Info("This is a worker machine, something wrong with configurations")
			//trigger to create machine on KTCloud by calling API

		}

		return ctrl.Result{RequeueAfter: time.Hour / 2}, nil
	} else {
		if ktMachine.Status.Status == "" {
			// we have to get one control-plane machine associated with the cluster
			//check if control machine is ready and join
			if !ktMachine.Status.WorkerRef.JoinedControlPlane {
				//available ready controlPlanes
				controlPlanesBootstrapList, err := r.GetBootstrapReadyMachineControlPlane(ctx, ktMachine, cluster, req)
				if err != nil {
					logger.Error(err, "failed to list bootstrap ready control-planes")
					return ctrl.Result{RequeueAfter: time.Minute}, nil
				}
				if len(controlPlanesBootstrapList) == 0 {
					logger.Error(errors.New("no ready available bootstrapped control planes"), "reconciling to check again...")
					return ctrl.Result{RequeueAfter: time.Minute}, nil
				}

				err = httpapi.JoinControlPlane(controlPlanesBootstrapList, *ktMachine, subjectToken)
				if err != nil {
					logger.Error(err, "worker machine failed joining control-plane")
					return ctrl.Result{RequeueAfter: time.Minute}, nil
				}

				//if assumed to have joined, we have to update the status of this machine
				ktMachine.Status.WorkerRef.JoinedControlPlane = true
				if err := r.Status().Update(ctx, ktMachine); err != nil {
					logger.Error(err, "Can't update status of machine to have joined cluster")
					return ctrl.Result{RequeueAfter: time.Minute / 2}, nil
				}

			}

			logger.Info("Machine Already joined control plane, nothing to do here, we have to check the machine again and update status")
			return ctrl.Result{RequeueAfter: time.Minute}, nil
		}
		logger.Info("Worker machine already initialized on cloud, we have to get the cloud status and update status on the local resource")
		// return ctrl.Result{RequeueAfter: time.Hour}, nil
		logger.Info("Worker Machine already created and has ID")
		//call API and check if machine is ready
		serverResponse, err := httpapi.GetCreatedVM(ktMachine, subjectToken)
		if err != nil || serverResponse == nil {
			logger.Error(err, "Failed to query worker VM on KT Cloud during API Call")
			return ctrl.Result{RequeueAfter: time.Minute}, nil
		}

		if !ktMachine.Status.WorkerRef.JoinedControlPlane {
			ktMachine.Status.WorkerRef.JoinedControlPlane = true
		}

		logger.Info("Got the machine we have to update if the states dont match")
		if ktMachine.Status.Status != serverResponse.Status {
			ktMachine.Status = *serverResponse
			if err := r.Status().Update(ctx, ktMachine); err != nil {
				logger.Error(err, "Can't update for machine with status on cloud")
				return ctrl.Result{RequeueAfter: time.Minute}, nil
			}

		}
	}

	//WORKER NODE CODE
	return ctrl.Result{RequeueAfter: time.Hour}, nil
}

func (r *KTMachineReconciler) getSubjectToken(ctx context.Context, ktMachine *infrastructurev1beta1.KTMachine, req ctrl.Request) (string, error) {

	logger := log.FromContext(ctx, "LogFrom", "Machine")

	cluster, err := r.GetMachineAssociatedCluster(ctx, ktMachine, req)
	if cluster == nil || err != nil {
		if cluster == nil {
			return "", errors.New("Failed to retrieve cluster for Machine")
		} else if err != nil {
			return "", err
		}
	}

	//ktsubjecttoken.name is always the same to cluster.name

	ktSubjectToken := &v1beta1.KTSubjectToken{}
	err = r.Get(ctx, types.NamespacedName{Name: cluster.Name, Namespace: ktMachine.Namespace}, ktSubjectToken)

	if err != nil {
		if apierrors.IsNotFound(err) {
			logger.Error(err, "Failed to get KTSubjectTokens associated with cluster", "Name", cluster.Name, "Namespace", cluster.Namespace)
			return "nil", err
		}
		return "", err
	}

	return ktSubjectToken.Spec.SubjectToken, nil

}

func (r *KTMachineReconciler) GetMachineAssociatedCluster(ctx context.Context, ktMachine *infrastructurev1beta1.KTMachine, req ctrl.Request) (*v1beta1.KTCluster, error) {
	logger := log.FromContext(ctx, "LogFrom", "Machine")

	ktMachineDeploymentList := &v1beta1.MachineDeploymentList{}
	err := r.List(ctx, ktMachineDeploymentList, client.InNamespace(ktMachine.Namespace))
	if err != nil {
		logger.Error(err, "failed to list MachineDeployments for this machine")
		return nil, err
	}

	// Filter by ownerReferences
	var ownerMachineDeployment v1beta1.MachineDeployment
	for _, machineDeployment := range ktMachineDeploymentList.Items {
		for _, ref := range ktMachine.OwnerReferences {
			if ref.UID == machineDeployment.UID {
				ownerMachineDeployment = machineDeployment
				logger.Info("Found owned MachineDeployment", "name", machineDeployment.Name)
				break
			}
		}
	}

	// we found matching machine deployment owner
	// we have to find the cluster from this
	// KTMachineTemplate is owned by KTCluster and KTMachineTemplate.Name = MachineDeployment.Name, KTMachineTemplate.NameSpace = MachineDeployment.NameSpace
	// therefore, use MachineDeployment to MachineTemplate to find Cluster then token for the cluster
	if ownerMachineDeployment.UID != "" {
		ktMachineTemplate := &v1beta1.KTMachineTemplate{}
		err := r.Get(ctx, types.NamespacedName{Name: ownerMachineDeployment.Name, Namespace: ownerMachineDeployment.Namespace}, ktMachineTemplate)

		if err != nil {
			if apierrors.IsNotFound(err) {
				logger.Error(err, "KTMachineTemplate not found no need to proceed for finding SubjectToken To Auth API", "Name", ownerMachineDeployment.Name, "Namespace", ownerMachineDeployment.Namespace)
				return nil, err
			}
			return nil, err
		}

		var clusterName string

		for _, ref := range ktMachineTemplate.OwnerReferences {
			if ref.Kind == "KTCluster" {
				clusterName = ref.Name
				break

			}
		}

		ktCluster := &v1beta1.KTCluster{}
		err = r.Get(ctx, types.NamespacedName{Name: clusterName, Namespace: ktMachine.Namespace}, ktCluster)
		if err != nil {
			return nil, err
		}

		return ktCluster, err
	}

	return nil, nil
}

func (r *KTMachineReconciler) GetBootstrapReadyMachineControlPlane(ctx context.Context, ktMachine *infrastructurev1beta1.KTMachine, associatedCluster *v1beta1.KTCluster, req ctrl.Request) ([]v1beta1.KTMachine, error) {
	logger := log.FromContext(ctx, "LogFrom", "Machine")

	ktMachineList := &v1beta1.KTMachineList{}
	err := r.List(ctx, ktMachineList, client.InNamespace(ktMachine.Namespace))
	if err != nil {
		logger.Error(err, "failed to list machines in same namespace as this worker")
		return nil, err
	}

	substring := "control-plane" // control-planes name have this string in their names

	// We just want to get ready bootstrap control planes
	var controlPlaneClusterMachines []v1beta1.KTMachine

	for _, machine := range ktMachineList.Items {
		if machine.Name != ktMachine.Name && strings.Contains(machine.Name, substring) && machine.Status.Status == "ACTIVE" && machine.Status.ControlPlaneRef.Type == "BootstrapReady" && machine.Status.ControlPlaneRef.Status {
			controlPlaneClusterMachines = append(controlPlaneClusterMachines, machine)
		}
	}
	return controlPlaneClusterMachines, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *KTMachineReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&infrastructurev1beta1.KTMachine{}).
		Named("ktmachine").
		Complete(r)
}
