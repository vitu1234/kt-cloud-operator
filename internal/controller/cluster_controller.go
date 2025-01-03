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

// ClusterReconciler reconciles a Cluster object
type ClusterReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=infrastructure.dcnlab.ssu.ac.kr,resources=clusters,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=infrastructure.dcnlab.ssu.ac.kr,resources=clusters/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=infrastructure.dcnlab.ssu.ac.kr,resources=clusters/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the Cluster object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.19.1/pkg/reconcile
func (r *ClusterReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	_ = log.FromContext(ctx)

	logger := log.FromContext(ctx)
	logger.V(1).Info("KTCluster Reconcile", "ktCluster", req)

	// Fetch the ktcluster instance
	cluster := &v1beta1.Cluster{}
	if err := r.Get(ctx, req.NamespacedName, cluster); err != nil {
		if apierrors.IsNotFound(err) {
			logger.Info("KTCluster resource not found. Ignoring since it must be deleted")
			return ctrl.Result{}, nil
		}
		logger.Error(err, "Failed to get KTCluster resource")
		return ctrl.Result{}, err
	}

	// Check if the child resources already exists to add ownerRef
	foundKTCluster := &v1beta1.KTCluster{}
	err := r.Get(ctx, types.NamespacedName{Name: cluster.Name, Namespace: cluster.Namespace}, foundKTCluster)
	if err != nil && apierrors.IsNotFound(err) {
		// Read through the cluster Object
		logger.Info("KTCluster not found matching Cluster resource, reque")
		// Requeue the request to ensure the Cluster is given Owner ref
		return ctrl.Result{RequeueAfter: time.Minute}, nil
	} else if err != nil {
		logger.Error(err, "Failed to get KTCluster, maybe dont have")
		return ctrl.Result{}, err
	}
	err = r.ktClusterForCluster(cluster, foundKTCluster, ctx, req)
	if err != nil {
		logger.Error(err, "Failed to add owner ref to ", "KTCluster.Namespace ", foundKTCluster.Namespace, "KTCluster.Name", foundKTCluster.Name)
		return ctrl.Result{RequeueAfter: time.Minute}, nil
	}
	logger.Info("Added owner ref to ", "KTCluster.Namespace ", foundKTCluster.Namespace, "KTCluster.Name", foundKTCluster.Name)

	return ctrl.Result{}, nil
}

// used to create Owner Refs
func (r *ClusterReconciler) ktClusterForCluster(cluster *v1beta1.Cluster, ktCluster *v1beta1.KTCluster, ctx context.Context, req ctrl.Request) error {
	logger := log.FromContext(ctx)
	logger.Info("Cluster Reconcile In ktClusterForCluster FN")

	// Set the ownerRef for the KTCluster
	// will be deleted when the Cluster CR is deleted.
	if err := controllerutil.SetControllerReference(cluster, ktCluster, r.Scheme); err != nil {
		return err
	}

	// ownerRef = &v1beta1.KTCluster{
	// 	ObjectMeta: metav1.ObjectMeta{
	// 		OwnerReferences: []metav1.OwnerReference{
	// 			{
	// 				APIVersion: cluster.APIVersion,
	// 				Kind:       cluster.Kind,
	// 				Name:       cluster.Name,
	// 				UID:        cluster.UID,
	// 			},
	// 		},
	// 		// Finalizers: []string{infrav1.IPClaimMachineFinalizer},
	// 	},
	// }
	// Save the changes
	if err := r.Client.Update(ctx, ktCluster); err != nil {
		return err
	}

	return nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *ClusterReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&infrastructurev1beta1.Cluster{}).
		Owns(&v1beta1.KTCluster{}).
		Owns(&v1beta1.MachineDeployment{}).
		Named("cluster").
		Complete(r)
}
