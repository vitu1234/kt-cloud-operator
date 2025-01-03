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
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	"dcnlab.ssu.ac.kr/kt-cloud-operator/api/v1beta1"
	infrastructurev1beta1 "dcnlab.ssu.ac.kr/kt-cloud-operator/api/v1beta1"
)

// KTMachineTemplateReconciler reconciles a KTMachineTemplate object
type KTMachineTemplateReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=infrastructure.dcnlab.ssu.ac.kr,resources=ktmachinetemplates,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=infrastructure.dcnlab.ssu.ac.kr,resources=ktmachinetemplates/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=infrastructure.dcnlab.ssu.ac.kr,resources=ktmachinetemplates/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the KTMachineTemplate object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.19.1/pkg/reconcile
func (r *KTMachineTemplateReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	_ = log.FromContext(ctx)

	// TODO(user): your logic here
	logger := log.FromContext(ctx, "LogFrom", "KTMachineTemplate")
	logger.V(1).Info("KTMachineTemplate Reconcile", "ktMachineTemplate", req)

	// Fetch the ktMachineTemplate instance
	ktMachineTemplate := &v1beta1.KTMachineTemplate{}

	if err := r.Get(ctx, req.NamespacedName, ktMachineTemplate); err != nil {
		if apierrors.IsNotFound(err) {
			logger.Info("ktMachineTemplate resource not found. Ignoring since it must be deleted")
			return ctrl.Result{}, nil
		}
		logger.Error(err, "Failed to get ktMachineTemplate resource")
		return ctrl.Result{}, err
	}

	// check child resources and add owner references
	foundMachineDeployment := &v1beta1.MachineDeployment{}

	err := r.Get(ctx, types.NamespacedName{Name: ktMachineTemplate.Name, Namespace: ktMachineTemplate.Namespace}, foundMachineDeployment)
	if err != nil && apierrors.IsNotFound(err) {
		logger.Info("MachineDeployment not found matching machine template")
		// Requeue the request to ensure the Cluster is given Owner ref
		return ctrl.Result{RequeueAfter: time.Minute}, nil
	} else if err != nil {
		logger.Error(err, "Failed to get MachineDeployment, maybe dont have")
		return ctrl.Result{}, err
	}

	// Read through the cluster Object
	err = r.ktMachineTemplateForMachineDeployment(ktMachineTemplate, foundMachineDeployment, ctx, req)
	if err != nil {
		logger.Error(err, "Failed to add owner ref to ", "MachineDeployment.Namespace ", ktMachineTemplate.Namespace, "MachineDeployment.Name", ktMachineTemplate.Name)
		return ctrl.Result{RequeueAfter: time.Minute}, nil
	}
	logger.Info("Added owner ref to ", "MachineDeployment.Namespace ", foundMachineDeployment.Namespace, "MachineDeployment.Name", foundMachineDeployment.Name)

	return ctrl.Result{}, nil
}

// used to create Owner Refs
func (r *KTMachineTemplateReconciler) ktMachineTemplateForMachineDeployment(ktMachineTemplate *v1beta1.KTMachineTemplate, machineDeployment *v1beta1.MachineDeployment, ctx context.Context, req ctrl.Request) error {
	logger := log.FromContext(ctx, "LogFrom", "KTMachineTemplate")
	log.FromContext(ctx, "KTMachineTemplate", ktMachineTemplate.Name, "KTMachineTemplate Namespace", ktMachineTemplate.Namespace)

	logger.Info("adding owner ref for machine ", "MachineDeployment.Namespace ", ktMachineTemplate.Namespace, " MachineDeployment.Name ", ktMachineTemplate.Name)

	// Set the ownerRef for the KTCluster
	// will be deleted when the Cluster CR is deleted.
	// controllerutil.SetControllerReference(ktMachineTemplate, machineDeployment, r.Scheme)
	if err := controllerutil.SetControllerReference(ktMachineTemplate, machineDeployment, r.Scheme); err != nil {
		return err
	}

	if err := r.Client.Update(ctx, machineDeployment); err != nil {
		return err
	}
	return nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *KTMachineTemplateReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&infrastructurev1beta1.KTMachineTemplate{}).
		Owns(&infrastructurev1beta1.MachineDeployment{}).
		Named("ktmachinetemplate").
		Complete(r)
}
