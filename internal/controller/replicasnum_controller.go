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

	webv1 "github.com/charleswan/replicas-num/api/v1"
	"github.com/google/uuid"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// ReplicasNumReconciler reconciles a ReplicasNum object
type ReplicasNumReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=web.github.com,resources=replicasnums,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=web.github.com,resources=replicasnums/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=web.github.com,resources=replicasnums/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the ReplicasNum object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.18.2/pkg/reconcile
func (r *ReplicasNumReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	// Get web API instance object.
	web := &webv1.ReplicasNum{}
	err := r.Get(ctx, req.NamespacedName, web)
	if err != nil {
		logger.Error(err, "failed to get instance object")
		return ctrl.Result{}, err
	}

	// Get current API object Pod.
	podList := &corev1.PodList{}
	err = r.List(ctx, podList, client.InNamespace(req.Namespace), client.MatchingLabels{"app": web.Name})
	if err != nil {
		logger.Error(err, "failed to list Pods")
		return ctrl.Result{}, err
	}
	currentPodCount := len(podList.Items)

	// Determine whether to create or delete.
	if web.Spec.Replicas > currentPodCount {
		// The desired number is larger than the running pods, so create new ones.
		for i := 0; i < web.Spec.Replicas-currentPodCount; i++ {
			// Create new Pod object.
			pod := &corev1.Pod{
				ObjectMeta: ctrl.ObjectMeta{
					Name:      web.Name + "-" + uuid.New().String()[0:8],
					Namespace: web.Namespace,
					Labels:    web.Labels,
				},
				Spec: web.Spec.Template.Spec,
			}

			// Link Pod and web instance.
			err = ctrl.SetControllerReference(web, pod, r.Scheme)
			if err != nil {
				logger.Error(err, "unable to set ownerReference for Pod")
				return ctrl.Result{}, err
			}

			// Create Pod now.
			err = r.Create(ctx, pod)
			if err != nil {
				logger.Error(err, "unable to create Pod")
				return ctrl.Result{}, err
			}
		}
	} else {
		// If the desired number of pods is less than or equal to the running pods, remove the redundant ones.
		deletePods := podList.Items[:currentPodCount-web.Spec.Replicas]
		for _, pod := range deletePods {
			err = r.Delete(ctx, &pod)
			if err != nil {
				logger.Error(err, "unable to delete Pod")
				return ctrl.Result{}, err
			}
		}
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *ReplicasNumReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&webv1.ReplicasNum{}).
		Complete(r)
}
