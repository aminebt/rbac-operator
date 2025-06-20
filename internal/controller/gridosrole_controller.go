/*
Copyright 2025.

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
	"fmt"

	"github.com/pkg/errors"
	k8serr "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	rbacv1alpha1 "github.com/aminebt/rbac-operator/api/v1alpha1"
	"github.com/aminebt/rbac-operator/internal/controller/utils"
)

// GridOSRoleReconciler reconciles a GridOSRole object
type GridOSRoleReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=rbac.security.gridos.gevernova.com,resources=gridosroles,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=rbac.security.gridos.gevernova.com,resources=gridosroles/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=rbac.security.gridos.gevernova.com,resources=gridosroles/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the GridOSRole object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.20.4/pkg/reconcile
func (r *GridOSRoleReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := logf.FromContext(ctx)

	role := &rbacv1alpha1.GridOSRole{}
	if err := r.Client.Get(ctx, req.NamespacedName, role); err != nil {
		if !k8serr.IsNotFound(err) {
			log.Error(err, "unable to fetch Group")
		}
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	msg := fmt.Sprintf("received reconcile request for %q (namespace: %q)", role.GetName(), role.GetNamespace())
	log.Info(msg)

	// is object marked for deletion ?
	if !role.ObjectMeta.DeletionTimestamp.IsZero() {
		log.Info("Role is marked for deletion")
		if utils.ContainsString(role.ObjectMeta.Finalizers, finalizerID) {
			// finalizer is present, so handle dependencies
			if err := r.deleteRole(ctx, role); err != nil {
				// if fail to delete the role, return with error so that it can be retried
				msg := "unable to delete Group"
				log.Error(err, msg)
				role.Status.Update(rbacv1alpha1.DeletingStatusPhase, msg, err)
				return ctrl.Result{}, errors.Wrap(r.Client.Status().Update(ctx, role), "could not update status")
			}

			// remove finalizer from the list and update it.
			role.ObjectMeta.Finalizers = utils.RemoveString(role.ObjectMeta.Finalizers, finalizerID)
			if err := r.Update(ctx, role); err != nil {
				msg := "could not remove finalizer"
				log.Error(err, msg)
				role.Status.Update(rbacv1alpha1.DeletingStatusPhase, msg, err)
				return ctrl.Result{}, errors.Wrap(r.Client.Status().Update(ctx, role), "could not update status")
			}
		}
		// finalizer already removed, nothing to do
		return ctrl.Result{}, nil
	}

	// register finalizer if it does not exist
	if !utils.ContainsString(role.ObjectMeta.Finalizers, finalizerID) {
		role.ObjectMeta.Finalizers = append(role.ObjectMeta.Finalizers, finalizerID)
		if err := r.Update(ctx, role); err != nil {
			return ctrl.Result{}, errors.Wrap(err, "could not add finalizer")
		}
		log.Info("Appended Finalizer")
	}

	// //initialize status bindings
	// if role.Status.Bindings == nil {
	// 	//role.Status.Bindings = make(map[string]string)
	// 	role.Status.Bindings = map[string]string{}
	// 	fmt.Println("initialize Status.Bindings")
	// }
	// if role.Status.BoundGroups == nil {
	// 	role.Status.BoundGroups = make([]string, 0)
	// 	fmt.Println("initialize Status.BoundGroups")
	// }

	role.Status.Update(rbacv1alpha1.ReadyStatusPhase, successMessage, nil)
	return ctrl.Result{}, errors.Wrap(r.Client.Status().Update(ctx, role), "could not update status")
}

func (r *GridOSRoleReconciler) deleteRole(ctx context.Context, role *rbacv1alpha1.GridOSRole) error {
	log := logf.FromContext(ctx, "phase", "deleting role")
	log.Info(fmt.Sprintf("cleaning up before deleting role %v in namespace %v", role.GetName(), role.GetNamespace()))
	return nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *GridOSRoleReconciler) SetupWithManager(mgr ctrl.Manager) error {
	pred := predicate.GenerationChangedPredicate{}
	return ctrl.NewControllerManagedBy(mgr).
		For(&rbacv1alpha1.GridOSRole{}).
		WithEventFilter(pred).
		Named("gridosrole").
		Complete(r)
}
