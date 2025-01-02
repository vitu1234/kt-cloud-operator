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

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	"dcnlab.ssu.ac.kr/kt-cloud-operator/api/v1beta1"
	infrastructurev1beta1 "dcnlab.ssu.ac.kr/kt-cloud-operator/api/v1beta1"
)

// KTSubjectTokenReconciler reconciles a KTSubjectToken object
type KTSubjectTokenReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=infrastructure.dcnlab.ssu.ac.kr,resources=ktsubjecttokens,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=infrastructure.dcnlab.ssu.ac.kr,resources=ktsubjecttokens/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=infrastructure.dcnlab.ssu.ac.kr,resources=ktsubjecttokens/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the KTSubjectToken object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.19.1/pkg/reconcile
func (r *KTSubjectTokenReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx, "LogFrom", "KTSubjectTokenReconciler")
	logger.V(1).Info("KTSubjectToken Reconcile", "KTSubjectToken", req)

	ktSubjectToken := &v1beta1.KTSubjectToken{}
	if err := r.Get(ctx, req.NamespacedName, ktSubjectToken); err != nil {
		if apierrors.IsNotFound(err) {
			logger.Info("KTSubjectToken resource not found. Ignoring since it must be deleted")
			return ctrl.Result{}, nil
		}
		logger.Error(err, "Failed to get KTSubjectToken resource")
		return ctrl.Result{}, err
	}

	// we have to add finalizers
	if ktSubjectToken.ObjectMeta.DeletionTimestamp.IsZero() {
		// The object is not being deleted, so lets add our finalizer if not already added
		if !controllerutil.ContainsFinalizer(ktSubjectToken, infrastructurev1beta1.KTSubjectTokenFinalizer) {
			controllerutil.AddFinalizer(ktSubjectToken, infrastructurev1beta1.KTSubjectTokenFinalizer)

			if err := r.Update(ctx, ktSubjectToken); err != nil {
				return ctrl.Result{}, err
			}
		}
	} else {
		// The object is being deleted
		if controllerutil.ContainsFinalizer(ktSubjectToken, infrastructurev1beta1.KTSubjectTokenFinalizer) {
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
		}
		// Stop reconciliation as the item is being deleted
		return ctrl.Result{}, nil
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *KTSubjectTokenReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&infrastructurev1beta1.KTSubjectToken{}).
		Named("ktsubjecttoken").
		Complete(r)
}
