/*
Copyright 2024. DCN

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
	"time"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	"dcnlab.ssu.ac.kr/kt-cloud-operator/api/v1beta1"
	infrastructurev1beta1 "dcnlab.ssu.ac.kr/kt-cloud-operator/api/v1beta1"
)

// KTClusterReconciler reconciles a KTCluster object
type KTClusterReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=infrastructure.dcnlab.ssu.ac.kr,resources=ktclusters,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=infrastructure.dcnlab.ssu.ac.kr,resources=ktclusters/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=infrastructure.dcnlab.ssu.ac.kr,resources=ktclusters/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the KTCluster object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.19.1/pkg/reconcile
func (r *KTClusterReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)
	logger.V(1).Info("KTCluster Reconcile", "ktCluster", req)

	// Fetch the KTCluster instance
	ktcluster := &v1beta1.KTCluster{}
	if err := r.Get(ctx, req.NamespacedName, ktcluster); err != nil {
		if apierrors.IsNotFound(err) {
			logger.Info("KTCluster resource not found. Ignoring since it must be deleted")
			return ctrl.Result{}, nil
		}
		logger.Error(err, "Failed to get KTCluster resource")
		return ctrl.Result{}, err
	}

	// Fetch child resources
	ktSubjectToken, err := r.fetchKTSubjectToken(ctx, ktcluster, req)
	if err != nil {
		logger.Error(err, "Failed to find KTSubjectToken")
		return ctrl.Result{}, nil // Or return an error if this is critical
	}

	// we have to add finalizers
	if ktcluster.ObjectMeta.DeletionTimestamp.IsZero() {
		// The object is not being deleted, so lets add our finalizer if not already added
		if !controllerutil.ContainsFinalizer(ktcluster, infrastructurev1beta1.KTClusterFinalizer) {
			controllerutil.AddFinalizer(ktcluster, infrastructurev1beta1.KTClusterFinalizer)

			if err := r.Update(ctx, ktcluster); err != nil {
				return ctrl.Result{}, err
			}
		}
	} else {
		// The object is being deleted
		if controllerutil.ContainsFinalizer(ktcluster, infrastructurev1beta1.KTClusterFinalizer) {
			// our finalizer is present, so lets handle any external dependency
			// we have to delete the machine on the cloud
			// we have to remove the finalizer and update the machine
			// remove our finalizer from the list and update it.
			// our finalizer is present, so lets handle any external dependency
			//update the machine status to deleting
			// ktMachine.Status.Status = "DELETING"
			// if err := r.Status().Update(ctx, ktMachine); err != nil {
			// 	return ctrl.Result{}, err
			// }

			// if err := r.deleteExternalResources(ctx, ktMachine, subjectToken); err != nil {
			// 	// if fail to delete the external dependency here, return with error
			// 	// so that it can be retried.
			// 	return ctrl.Result{}, err
			// }
			// remove our finalizer from the list and update it.
			// controllerutil.RemoveFinalizer(ktSubjectToken, infrastructurev1beta1.KTSubjectTokenFinalizer)
			// if err := r.Update(ctx, ktSubjectToken); err != nil {
			// 	return ctrl.Result{}, err
			// }

			machines, err := r.getClusterAssociatedKTMachines(ctx, ktcluster)
			if err != nil {
				logger.Error(err, "Failed to get machines associated with the cluster")
				return ctrl.Result{}, err

			}
			logger.Info("Machines associated with the cluster", "machines", len(machines))
			if len(machines) == 0 {
				logger.Info("There are no machines associated with the cluster. We have to remove clusterwide finalizers...")
				controllerutil.RemoveFinalizer(ktSubjectToken, infrastructurev1beta1.KTSubjectTokenFinalizer)
				if err := r.Update(ctx, ktSubjectToken); err != nil {
					return ctrl.Result{}, err
				}

				controllerutil.RemoveFinalizer(ktcluster, infrastructurev1beta1.KTClusterFinalizer)
				if err := r.Update(ctx, ktcluster); err != nil {
					return ctrl.Result{}, err
				}
			}

		}
		// Stop reconciliation as the item is being deleted
		return ctrl.Result{}, nil
	}

	foundKTMachineTemplateCP, err := r.fetchMachineTemplate(ctx, ktcluster, "-control-plane", req)
	if err != nil {
		logger.Error(err, "Failed to find control-plane machine template")
		return ctrl.Result{}, nil // Or return an error if this is critical
	}

	foundKTMachineTemplateMD, err := r.fetchMachineTemplate(ctx, ktcluster, "-md-0", req)
	if err != nil {
		logger.Error(err, "Failed to find -md-0 machine template", "ktCluster", ktcluster.Name, "namespace", ktcluster.Namespace)
		return ctrl.Result{}, nil // Or return an error if this is critical
	}

	// Check if any required machine template is missing
	if foundKTMachineTemplateCP == nil || foundKTMachineTemplateMD == nil {
		logger.Info("One or more machine templates are missing. Requeuing...")
		return ctrl.Result{RequeueAfter: 10 * time.Second}, nil
	}

	logger.Info("Successfully added owner references", "KTCluster.Name", ktcluster.Name)
	return ctrl.Result{}, nil
}

func (r *KTClusterReconciler) fetchMachineTemplate(ctx context.Context, ktcluster *v1beta1.KTCluster, suffix string, req ctrl.Request) (*v1beta1.KTMachineTemplate, error) {
	logger := log.FromContext(ctx, "LogFrom", "KTCluster")

	templateName := string(ktcluster.Name + suffix)
	machineTemplate := &v1beta1.KTMachineTemplate{}
	err := r.Get(ctx, types.NamespacedName{Name: templateName, Namespace: ktcluster.Namespace}, machineTemplate)

	if err != nil {
		if apierrors.IsNotFound(err) {
			logger.Info("MachineTemplate not found for "+templateName, "Name", templateName, "Namespace", ktcluster.Namespace)
			return &v1beta1.KTMachineTemplate{}, err
		}
		return &v1beta1.KTMachineTemplate{}, err
	}

	// Add owner references
	if err := r.ktClusterForMachineTemplate(ktcluster, machineTemplate, ctx, req); err != nil {
		logger.Error(err, "Failed to add owner reference to control-plane machine template")
	}

	return machineTemplate, nil
}

func (r *KTClusterReconciler) fetchKTSubjectToken(ctx context.Context, ktcluster *v1beta1.KTCluster, req ctrl.Request) (*v1beta1.KTSubjectToken, error) {
	logger := log.FromContext(ctx, "LogFrom", "KTCluster")

	ktSubjectToken := &v1beta1.KTSubjectToken{}
	err := r.Get(ctx, types.NamespacedName{Name: ktcluster.Name, Namespace: ktcluster.Namespace}, ktSubjectToken)

	if err != nil {
		if apierrors.IsNotFound(err) {
			logger.Error(err, "KTSubjectToken not found no need to proceed for this", "Name", ktcluster.Name, "Namespace", ktcluster.Namespace)
			return &v1beta1.KTSubjectToken{}, err
		}
		return &v1beta1.KTSubjectToken{}, err
	}

	// Add owner references
	if err := r.ktClusterForKTSubjectTokenOnwer(ktcluster, ktSubjectToken, ctx, req); err != nil {
		logger.Error(err, "Failed to add owner reference to KTSubject token")
	}

	return ktSubjectToken, nil
}

func (r *KTClusterReconciler) ktClusterForMachineTemplate(ktCluster *v1beta1.KTCluster, ktMachineTemplate *v1beta1.KTMachineTemplate, ctx context.Context, req ctrl.Request) error {
	logger := log.FromContext(ctx, "LogFrom", "KTCluster")

	logger.Info("adding owner ref for machine ", "KTMachineTemplate.Namespace ", ktMachineTemplate.Namespace, " KTMachineTemplate.Name ", ktMachineTemplate.Name)

	// Set the ownerRef for the KTCluster
	// will be deleted when the Cluster CR is deleted.
	// controllerutil.SetControllerReference(ktCluster, ktMachineTemplate, r.Scheme)
	if err := controllerutil.SetControllerReference(ktCluster, ktMachineTemplate, r.Scheme); err != nil {
		logger.Error(err, "Failed to set ktmachine template owner reference")
		return err
	}

	if err := r.Client.Update(ctx, ktMachineTemplate); err != nil {
		logger.Error(err, "Can't update for ktmachine template owner reference")
		return err
	}

	return nil
}

func (r *KTClusterReconciler) ktClusterForKTSubjectTokenOnwer(ktCluster *v1beta1.KTCluster, ktSubjectToken *v1beta1.KTSubjectToken, ctx context.Context, req ctrl.Request) error {
	logger := log.FromContext(ctx, "LogFrom", "KTCluster")

	logger.Info("adding owner ref for ktSubjectToken ", "KTSubjectToken.Namespace ", ktSubjectToken.Namespace, " KTSubjectToken.Name ", ktSubjectToken.Name)

	// Set the ownerRef for the KTCluster
	// will be deleted when the Cluster CR is deleted.
	// controllerutil.SetControllerReference(ktCluster, ktMachineTemplate, r.Scheme)
	if err := controllerutil.SetControllerReference(ktCluster, ktSubjectToken, r.Scheme); err != nil {
		logger.Error(err, "Failed to set ktsubjecttoken owner reference")
		return err
	}

	if err := r.Client.Update(ctx, ktSubjectToken); err != nil {
		logger.Error(err, "Can't update for ktsubjecttoken owner reference")
		return err
	}

	return nil
}

func (r *KTClusterReconciler) getClusterAssociatedKTMachines(ctx context.Context, ktCluster *v1beta1.KTCluster) ([]v1beta1.KTMachine, error) {
	logger := log.FromContext(ctx, "LogFrom", "KTCluster")

	// List all the machines in the same namespace
	machineList := &v1beta1.KTMachineList{}
	clusterMachineList := &v1beta1.KTMachineList{}

	if err := r.List(ctx, machineList, client.InNamespace(ktCluster.Namespace)); err != nil {
		logger.Error(err, "Failed to list machines")
		return nil, err
	}

	// Filter out the machines that are associated with the same cluster
	for _, machine := range machineList.Items {
		if machine.Spec.ClusterName == ktCluster.Name {
			clusterMachineList.Items = append(clusterMachineList.Items, machine)
		}
	}
	return clusterMachineList.Items, nil

}

// SetupWithManager sets up the controller with the Manager.
func (r *KTClusterReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&infrastructurev1beta1.KTCluster{}).
		Owns(&v1beta1.KTSubjectToken{}).
		Owns(&v1beta1.KTMachineTemplate{}).
		Named("ktcluster").
		Complete(r)
}
