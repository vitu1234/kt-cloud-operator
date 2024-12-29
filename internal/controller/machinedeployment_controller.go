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

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"dcnlab.ssu.ac.kr/kt-cloud-operator/api/v1beta1"
	infrastructurev1beta1 "dcnlab.ssu.ac.kr/kt-cloud-operator/api/v1beta1"
	utils "dcnlab.ssu.ac.kr/kt-cloud-operator/internal/utils"
)

// MachineDeploymentReconciler reconciles a MachineDeployment object
type MachineDeploymentReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

const (
	waitForClusterInfrastructureReadyDuration = 15 * time.Second
	waitForInstanceBecomeActiveToReconcile    = 60 * time.Second
	waitForBuildingInstanceToReconcile        = 10 * time.Second
	deleteServerRequeueDelay                  = 10 * time.Second
)

// +kubebuilder:rbac:groups=infrastructure.dcnlab.ssu.ac.kr,resources=machinedeployments,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=infrastructure.dcnlab.ssu.ac.kr,resources=machinedeployments/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=infrastructure.dcnlab.ssu.ac.kr,resources=machinedeployments/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the MachineDeployment object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.19.1/pkg/reconcile
func (r *MachineDeploymentReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	_ = log.FromContext(ctx)

	logger := log.FromContext(ctx, "LogFrom", "MachineDeployment")
	logger.V(1).Info("MachineDeployment Reconcile", "machineDeployment", req)

	// Fetch the MachineDeployment instance
	machineDeployment := &v1beta1.MachineDeployment{}
	if err := r.Get(ctx, req.NamespacedName, machineDeployment); err != nil {
		if apierrors.IsNotFound(err) {
			logger.Info("MachineDeployment resource not found. Ignoring since it must be deleted")
			return ctrl.Result{}, nil
		}
		logger.Error(err, "Failed to get MachineDeployment resource")
		return ctrl.Result{}, err
	}

	// create child resources and add owner references
	countMachinesAvailable, err := r.countChildMachinesByOwner(ctx, machineDeployment)
	if err != nil {
		logger.Error(err, "Failed to get Machines for deployment, maybe dont have")

		return ctrl.Result{RequeueAfter: time.Minute}, nil
	} else if countMachinesAvailable != machineDeployment.Spec.Replicas {
		logger.Info("KTMachines not found matching machine deployment replicas, we have to create a new one")
		err := r.ktMachineForMachineDeployment(ctx, machineDeployment, countMachinesAvailable)
		if err != nil {
			logger.Error(err, "Failed to find KTMachineTemplate to create Machine from MachineDeployment", "MachineDeployment.Namespace", machineDeployment.Namespace, "MachineDeployment.Name", machineDeployment.Name)
			return ctrl.Result{}, err
		}
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func (r *MachineDeploymentReconciler) ktMachineForMachineDeployment(ctx context.Context, machineDeployment *v1beta1.MachineDeployment, countMachinesAvailable int) error {
	logger := log.FromContext(ctx, "LogFrom", "MachineDeployment")

	foundKTMachineTemplate := &v1beta1.KTMachineTemplate{}
	err := r.Get(ctx, types.NamespacedName{Name: machineDeployment.Name, Namespace: machineDeployment.Namespace}, foundKTMachineTemplate)
	if err != nil {
		logger.Error(err, "Failed to get KTMachineTemplate", "Name", machineDeployment.Name, "Namespace", machineDeployment.Namespace)
		return err
	}

	machinesToCreate := 0
	if countMachinesAvailable < machineDeployment.Spec.Replicas {
		machinesToCreate = machineDeployment.Spec.Replicas - countMachinesAvailable
	} else if countMachinesAvailable > machineDeployment.Spec.Replicas {

	} else {
		machinesToCreate = machineDeployment.Spec.Replicas
	}

	for i := 0; i < machinesToCreate; i++ {
		machineName := machineDeployment.Name + "-" + strings.ToLower(utils.RandomString(10))

		machine := &v1beta1.KTMachine{
			ObjectMeta: metav1.ObjectMeta{
				Name:      machineName,
				Namespace: machineDeployment.Namespace,
			},
			Spec: v1beta1.KTMachineSpec{
				ControlPlaneNumber: i + 1,
				Flavor:             foundKTMachineTemplate.Spec.Template.Spec.Flavor,
				ClusterName:        machineDeployment.Spec.Template.Spec.ClusterName,
				AvailabilityZone:   machineDeployment.Spec.Template.Spec.FailureDomain,
				SSHKeyName:         foundKTMachineTemplate.Spec.Template.Spec.SSHKeyName,
				BlockDeviceMapping: foundKTMachineTemplate.Spec.Template.Spec.BlockDeviceMapping,
				NetworkTier:        foundKTMachineTemplate.Spec.Template.Spec.NetworkTier,
			},
		}

		// Set the owner reference for the Machine
		if err := controllerutil.SetControllerReference(machineDeployment, machine, r.Scheme); err != nil {
			logger.Error(err, "Failed to set controller reference", "KTMachine.Name", machineName)
			return err
		}

		logger.Info("Creating a new KTMachine", "KTMachine.Namespace", machine.Namespace, "KTMachine.Name", machine.Name)
		if err := r.Create(ctx, machine); err != nil {
			logger.Error(err, "Failed to create new KTMachine", "KTMachine.Namespace", machine.Namespace, "KTMachine.Name", machine.Name)
			return err
		}
	}

	// Ensure all machines are created before returning
	logger.Info("Successfully created all KTMachine replicas", "Replicas", machineDeployment.Spec.Replicas)
	return nil

}

func (r *MachineDeploymentReconciler) countChildMachinesByOwner(ctx context.Context, machineDeployment *v1beta1.MachineDeployment) (int, error) {
	logger := log.FromContext(ctx, "LogFrom", "MachineDeployment")

	ktMachineList := &v1beta1.KTMachineList{}
	err := r.List(ctx, ktMachineList, client.InNamespace(machineDeployment.Namespace))
	if err != nil {
		logger.Error(err, "failed to list KTMachines")
		return 0, err
	}

	// Filter by ownerReferences
	count := 0
	for _, machine := range ktMachineList.Items {
		for _, ownerRef := range machine.OwnerReferences {
			if ownerRef.Kind == "MachineDeployment" && ownerRef.Name == machineDeployment.Name {
				count++
				break
			}
		}
	}
	return count, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *MachineDeploymentReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&infrastructurev1beta1.MachineDeployment{}).
		Owns(&infrastructurev1beta1.KTMachine{}).
		Named("machinedeployment").
		Complete(r)
}
