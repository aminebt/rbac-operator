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
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/util/retry"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	rbacv1alpha1 "github.com/aminebt/rbac-operator/api/v1alpha1"
	"github.com/aminebt/rbac-operator/internal/controller/utils"
)

// GridOSGroupRoleBindingReconciler reconciles a GridOSGroupRoleBinding object
type GridOSGroupRoleBindingReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

var (
	groupKey    = ".spec.group.name"
	roleListKey = ".spec.roles"
)

// +kubebuilder:rbac:groups=rbac.security.gridos.gevernova.com,resources=gridosgrouprolebindings,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=rbac.security.gridos.gevernova.com,resources=gridosgrouprolebindings/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=rbac.security.gridos.gevernova.com,resources=gridosgrouprolebindings/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the GridOSGroupRoleBinding object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.20.4/pkg/reconcile
func (r *GridOSGroupRoleBindingReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := logf.FromContext(ctx)

	grb := &rbacv1alpha1.GridOSGroupRoleBinding{}
	if err := r.Client.Get(ctx, req.NamespacedName, grb); err != nil {
		if !k8serr.IsNotFound(err) {
			log.Error(err, "unable to fetch Group")
		}
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	msg := fmt.Sprintf("received reconcile request for %q (namespace: %q)", grb.GetName(), grb.GetNamespace())
	log.Info(msg)

	// is object marked for deletion ?
	if !grb.ObjectMeta.DeletionTimestamp.IsZero() {
		log.Info("grb is marked for deletion")
		if utils.ContainsString(grb.ObjectMeta.Finalizers, finalizerID) {
			// finalizer is present, so handle dependencies
			if err := r.reconcileDelete(ctx, grb); err != nil {
				// if fail to delete the grb, return with error so that it can be retried
				msg := "unable to delete Group"
				log.Error(err, msg)
				grb.Status.Update(rbacv1alpha1.DeletingStatusPhase, msg, err)

				if statusUpdateErr := r.Client.Status().Update(ctx, grb); statusUpdateErr != nil {
					log.Error(statusUpdateErr, fmt.Sprintf("Failed to update status for %v/%v ", grb.Namespace, grb.Spec.Group.Name))
					return ctrl.Result{}, statusUpdateErr
				}
				return ctrl.Result{}, err
			}

			// remove finalizer from the list and update it.
			grb.ObjectMeta.Finalizers = utils.RemoveString(grb.ObjectMeta.Finalizers, finalizerID)
			if err := r.Update(ctx, grb); err != nil {
				msg := "could not remove finalizer"
				log.Error(err, msg)
				grb.Status.Update(rbacv1alpha1.DeletingStatusPhase, msg, err)
				return ctrl.Result{}, errors.Wrap(r.Client.Status().Update(ctx, grb), "could not update status")
			}
		}
		// finalizer already removed, nothing to do
		return ctrl.Result{}, nil
	}

	// register finalizer if it does not exist
	if !utils.ContainsString(grb.ObjectMeta.Finalizers, finalizerID) {
		grb.ObjectMeta.Finalizers = append(grb.ObjectMeta.Finalizers, finalizerID)
		if err := r.Update(ctx, grb); err != nil {
			return ctrl.Result{}, errors.Wrap(err, "could not add finalizer")
		}
		log.Info("Appended Finalizer")
	}

	// error status if group does not exist - don't return result just yet - wait until updating roles status
	groupNamespacedName := types.NamespacedName{Namespace: grb.Namespace, Name: grb.Spec.Group.Name}
	gr := &rbacv1alpha1.GridOSGroup{}
	groupErr := r.Client.Get(ctx, groupNamespacedName, gr)

	//// update group status bindings

	// if error fetching group (other than notFound) return the error
	var groupAbsent bool = false
	if groupErr != nil {
		// if group not found, clean up roles status
		if k8serr.IsNotFound(groupErr) {
			log.Info(fmt.Sprintf("Group %v/%v not found", grb.Namespace, grb.Spec.Group.Name))
			groupAbsent = true
			// if other error, return error to requeue
		} else {
			log.Error(groupErr, fmt.Sprintf("unable to fetch Group %v/%v", grb.Namespace, grb.Spec.Group.Name))

			grb.Status.Update(rbacv1alpha1.ErrorStatusPhase, "Error fetching group", groupErr)
			if statusUpdateErr := r.Client.Status().Update(ctx, grb); statusUpdateErr != nil {
				log.Error(statusUpdateErr, fmt.Sprintf("Failed to update status for %v/%v ", grb.Namespace, grb.Spec.Group.Name))
			}
			return ctrl.Result{}, groupErr
		}
	}

	// if groupErr != nil {
	// 	var err error
	// 	if !k8serr.IsNotFound(groupErr) {
	// 		log.Error(groupErr, fmt.Sprintf("unable to fetch Group %v/%v", grb.Namespace, grb.Spec.Group.Name))
	// 		err = groupErr
	// 	} else {
	// 		log.Info(fmt.Sprintf("Group %v/%v not found", grb.Namespace, grb.Spec.Group.Name))
	// 		err = nil
	// 	}
	// 	grb.Status.Update(rbacv1alpha1.ErrorStatusPhase, "Error fetching group", groupErr)
	// 	if statusUpdateErr := r.Client.Status().Update(ctx, grb); statusUpdateErr != nil {
	// 		log.Error(statusUpdateErr, fmt.Sprintf("Failed to update status for %v/%v ", grb.Namespace, grb.Spec.Group.Name))
	// 	}
	// 	return ctrl.Result{}, err

	// }

	// pending if one of the roles doesn't exist
	roles := grb.Spec.Roles
	missingRoles := []string{}
	boundRoles := []string{}
	for _, roleName := range roles {
		roleNamespacedName := types.NamespacedName{Namespace: grb.Namespace, Name: roleName}
		role := &rbacv1alpha1.GridOSRole{}
		if err := r.Client.Get(ctx, roleNamespacedName, role); err != nil {
			if !k8serr.IsNotFound(err) {
				log.Error(err, fmt.Sprintf("unable to fetch Role %v/%v", grb.Namespace, roleName))
			} else {
				log.Info(fmt.Sprintf("role %v/%v not found", grb.Namespace, roleName))
			}
			missingRoles = append(missingRoles, roleName)
			continue
		}
		boundRoles = append(boundRoles, roleName)

		err := retry.RetryOnConflict(retry.DefaultBackoff, func() error {
			// fetch the latest role
			latestRole := &rbacv1alpha1.GridOSRole{}
			if err := r.Client.Get(ctx, roleNamespacedName, latestRole); err != nil {
				return err
			}

			latestRole.Status.UpdateBinding(grb.Name, grb.Spec.Group.Name, groupAbsent)
			return r.Client.Status().Update(ctx, latestRole)
		})

		if err != nil {
			log.Error(err, fmt.Sprintf("Failed to update role status bindings for %v/%v ", role.Namespace, role.Name))
			return ctrl.Result{}, err
		}

	}

	// now that roles are reconciled - if group not present return
	if groupAbsent {
		grb.Status.Update(rbacv1alpha1.ErrorStatusPhase, "Group Not Found", groupErr)
		if statusUpdateErr := r.Client.Status().Update(ctx, grb); statusUpdateErr != nil {
			log.Error(statusUpdateErr, fmt.Sprintf("Failed to update status for %v/%v ", grb.Namespace, grb.Spec.Group.Name))
		}
		return ctrl.Result{}, nil
	}

	// otherwise - update group status bindings
	// we use RetryOnConflict to avoid error "the object has been modified; please apply your changes to the latest version and try again"
	err := retry.RetryOnConflict(retry.DefaultBackoff, func() error {
		// fetch the latest group
		latestGroup := &rbacv1alpha1.GridOSGroup{}
		if err := r.Client.Get(ctx, groupNamespacedName, latestGroup); err != nil {
			return err
		}

		latestGroup.Status.UpdateBindings(grb.Name, boundRoles)
		return r.Client.Status().Update(ctx, latestGroup)
	})

	if err != nil {
		log.Error(err, fmt.Sprintf("Failed to update group status bindings for %v/%v ", gr.Namespace, gr.Name))
		return ctrl.Result{}, err
	}

	// update grb status
	var phase rbacv1alpha1.StatusPhase = rbacv1alpha1.ErrorStatusPhase
	var statusMsg string
	if len(boundRoles) == 0 {
		phase = rbacv1alpha1.ErrorStatusPhase
		statusMsg = "None of the referenced roles was found"
	} else if len(boundRoles) < len(roles) {
		phase = rbacv1alpha1.PendingStatusPhase
		statusMsg = fmt.Sprintf("The following referenced roles were not found: %v", missingRoles)
	} else {
		phase = rbacv1alpha1.ReadyStatusPhase
		statusMsg = successMessage
	}

	grb.Status.Update(phase, statusMsg, nil)
	if statusUpdateErr := r.Client.Status().Update(ctx, grb); statusUpdateErr != nil {
		log.Error(statusUpdateErr, fmt.Sprintf("Failed to update status for %v/%v", grb.Namespace, grb.Spec.Group.Name))
	}

	return ctrl.Result{}, nil
}

func (r *GridOSGroupRoleBindingReconciler) reconcileDelete(ctx context.Context, grb *rbacv1alpha1.GridOSGroupRoleBinding) error {
	log := logf.FromContext(ctx, "phase", "deleting group-role binding")
	log.Info(fmt.Sprintf("cleaning up before deleting group-role binding %v in namespace %v", grb.GetName(), grb.GetNamespace()))

	// error status if group does not exist - don't return result just yet - wait until updating roles status
	groupNamespacedName := types.NamespacedName{Namespace: grb.Namespace, Name: grb.Spec.Group.Name}
	gr := &rbacv1alpha1.GridOSGroup{}
	groupErr := r.Client.Get(ctx, groupNamespacedName, gr)

	//// update group status bindings

	// if error fetching group (other than notFound) return the error
	var groupAbsent bool = false
	if groupErr != nil {
		// if group not found, clean up roles status
		if k8serr.IsNotFound(groupErr) {
			log.Info(fmt.Sprintf("Group %v/%v not found", grb.Namespace, grb.Spec.Group.Name))
			groupAbsent = true
			// if other error, return error to requeue
		} else {
			log.Error(groupErr, fmt.Sprintf("unable to fetch Group %v/%v", grb.Namespace, grb.Spec.Group.Name))

			grb.Status.Update(rbacv1alpha1.ErrorStatusPhase, "Error fetching group", groupErr)
			if statusUpdateErr := r.Client.Status().Update(ctx, grb); statusUpdateErr != nil {
				log.Error(statusUpdateErr, fmt.Sprintf("Failed to update status for %v/%v ", grb.Namespace, grb.Spec.Group.Name))
			}
			return groupErr
		}
	}

	// pending if one of the roles doesn't exist
	roles := grb.Spec.Roles
	missingRoles := []string{}
	boundRoles := []string{}
	for _, roleName := range roles {
		roleNamespacedName := types.NamespacedName{Namespace: grb.Namespace, Name: roleName}
		role := &rbacv1alpha1.GridOSRole{}
		if err := r.Client.Get(ctx, roleNamespacedName, role); err != nil {
			if !k8serr.IsNotFound(err) {
				log.Error(err, fmt.Sprintf("unable to fetch Role %v/%v", grb.Namespace, roleName))
			} else {
				log.Info(fmt.Sprintf("role %v/%v not found", grb.Namespace, roleName))
			}
			missingRoles = append(missingRoles, roleName)
			continue
		}
		boundRoles = append(boundRoles, roleName)

		err := retry.RetryOnConflict(retry.DefaultBackoff, func() error {
			// fetch the latest role
			latestRole := &rbacv1alpha1.GridOSRole{}
			if err := r.Client.Get(ctx, roleNamespacedName, latestRole); err != nil {
				return err
			}

			latestRole.Status.UpdateBinding(grb.Name, grb.Spec.Group.Name, true)
			return r.Client.Status().Update(ctx, latestRole)
		})

		if err != nil {
			log.Error(err, fmt.Sprintf("Failed to update role status bindings for %v/%v ", role.Namespace, role.Name))
			return err
		}

	}

	// now that roles are reconciled - if group not present return nil (no further updates needed)
	if groupAbsent {
		return nil
	}
	// otherwise - update group status bindings
	// we use RetryOnConflict to avoid error "the object has been modified; please apply your changes to the latest version and try again"
	err := retry.RetryOnConflict(retry.DefaultBackoff, func() error {
		// fetch the latest group
		latestGroup := &rbacv1alpha1.GridOSGroup{}
		if err := r.Client.Get(ctx, groupNamespacedName, latestGroup); err != nil {
			return err
		}

		latestGroup.Status.DeleteBindings(grb.Name)
		return r.Client.Status().Update(ctx, latestGroup)
	})

	return err
}

// SetupWithManager sets up the controller with the Manager.
func (r *GridOSGroupRoleBindingReconciler) SetupWithManager(mgr ctrl.Manager) error {

	// add an index field on spec.group.name to facilitate querying grbs based on group
	if err := mgr.GetFieldIndexer().IndexField(
		context.Background(),
		&rbacv1alpha1.GridOSGroupRoleBinding{},
		groupKey,
		func(obj client.Object) []string {
			grb := obj.(*rbacv1alpha1.GridOSGroupRoleBinding)
			return []string{grb.Spec.Group.Name}
		}); err != nil {
		return err
	}

	// add an index field on spec.roles to facilitate querying grbs based on roles
	if err := mgr.GetFieldIndexer().IndexField(
		context.Background(),
		&rbacv1alpha1.GridOSGroupRoleBinding{},
		roleListKey,
		func(obj client.Object) []string {
			grb := obj.(*rbacv1alpha1.GridOSGroupRoleBinding)
			return grb.Spec.Roles // this is []string
		},
	); err != nil {
		return err
	}

	roleHandler := handler.EnqueueRequestsFromMapFunc(func(ctx context.Context, obj client.Object) []reconcile.Request {
		log := logf.FromContext(ctx, "phase", "watch related roles changes")
		role, ok := obj.(*rbacv1alpha1.GridOSRole)
		if !ok {
			return nil
		}
		var grbList rbacv1alpha1.GridOSGroupRoleBindingList

		if err := mgr.GetClient().List(ctx, &grbList, client.InNamespace(role.Namespace), client.MatchingFields{roleListKey: role.Name}); err != nil {
			log.Error(err, "unable to list related roles")
			return nil
		}

		var requests []reconcile.Request
		for _, grb := range grbList.Items {
			requests = append(requests, reconcile.Request{
				NamespacedName: types.NamespacedName{
					Namespace: grb.Namespace,
					Name:      grb.Name,
				},
			})
		}
		return requests

	})

	pred := predicate.GenerationChangedPredicate{}

	return ctrl.NewControllerManagedBy(mgr).
		For(&rbacv1alpha1.GridOSGroupRoleBinding{}, builder.WithPredicates(pred)).
		Named("gridosgrouprolebinding").
		Watches(
			&rbacv1alpha1.GridOSGroup{},
			handler.EnqueueRequestsFromMapFunc(func(ctx context.Context, obj client.Object) []reconcile.Request {
				log := logf.FromContext(ctx, "phase", "watch related resources changes")
				group, ok := obj.(*rbacv1alpha1.GridOSGroup)
				if !ok {
					return nil
				}

				var requests []reconcile.Request

				//// Handle group deletion : reconcile all referenced grb in status
				// if !group.DeletionTimestamp.IsZero() {
				// 	for binding := range group.Status.Bindings {
				// 		requests = append(requests, reconcile.Request{
				// 			NamespacedName: types.NamespacedName{
				// 				Namespace: group.Namespace,
				// 				Name:      binding,
				// 			},
				// 		})
				// 	}
				// 	return requests
				// }

				// Handle creation: find grbs that point to the created group in their spec
				var grbList rbacv1alpha1.GridOSGroupRoleBindingList
				// if err := r.Client.List(ctx, &grbList, client.InNamespace(group.Namespace)); err != nil {
				// 	log.Error(err, "unable to list related groups")
				// 	return nil
				// }
				// for _, grb := range grbList.Items {
				// 	if grb.Spec.Group.Name == group.Name && grb.Namespace == group.Namespace {
				// 		requests = append(requests, reconcile.Request{
				// 			NamespacedName: types.NamespacedName{
				// 				Namespace: grb.Namespace,
				// 				Name:      grb.Name,
				// 			},
				// 		})
				// 	}
				// }

				if err := r.List(ctx, &grbList, client.InNamespace(group.Namespace), client.MatchingFields{groupKey: group.Name}); err != nil {
					log.Error(err, "unable to list related groups")
					return nil
				}

				for _, grb := range grbList.Items {
					requests = append(requests, reconcile.Request{
						NamespacedName: types.NamespacedName{
							Namespace: grb.Namespace,
							Name:      grb.Name,
						},
					})
				}
				return requests
			}),
			builder.WithPredicates(predicate.Funcs{
				DeleteFunc:  func(e event.DeleteEvent) bool { return true },
				CreateFunc:  func(e event.CreateEvent) bool { return true },
				UpdateFunc:  func(e event.UpdateEvent) bool { return false },
				GenericFunc: func(e event.GenericEvent) bool { return false },
			}),
		).
		Watches(
			&rbacv1alpha1.GridOSRole{},
			roleHandler,
			builder.WithPredicates(predicate.Funcs{
				DeleteFunc:  func(e event.DeleteEvent) bool { return true },
				CreateFunc:  func(e event.CreateEvent) bool { return true },
				UpdateFunc:  func(e event.UpdateEvent) bool { return false },
				GenericFunc: func(e event.GenericEvent) bool { return false },
			}),
		).
		Complete(r)
}
