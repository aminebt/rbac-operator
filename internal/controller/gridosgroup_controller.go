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
	"time"

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

const (
	finalizerID    = "security.gridos.gevernova.com/rbac"
	defaultRequeue = 20 * time.Second
	successMessage = "successfully reconciled"
)

// GridOSGroupReconciler reconciles a GridOSGroup object
type GridOSGroupReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=rbac.security.gridos.gevernova.com,resources=gridosgroups,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=rbac.security.gridos.gevernova.com,resources=gridosgroups/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=rbac.security.gridos.gevernova.com,resources=gridosgroups/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the GridOSGroup object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.20.4/pkg/reconcile
func (r *GridOSGroupReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := logf.FromContext(ctx)

	gr := &rbacv1alpha1.GridOSGroup{}
	if err := r.Client.Get(ctx, req.NamespacedName, gr); err != nil {
		if !k8serr.IsNotFound(err) {
			log.Error(err, "unable to fetch Group")
		}
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	msg := fmt.Sprintf("received reconcile request for %q (namespace: %q)", gr.GetName(), gr.GetNamespace())
	log.Info(msg)

	// is object marked for deletion ?
	if !gr.ObjectMeta.DeletionTimestamp.IsZero() {
		log.Info("Group is marked for deletion")
		if utils.ContainsString(gr.ObjectMeta.Finalizers, finalizerID) {
			// finalizer is present, so handle dependencies
			if err := r.deleteGroup(ctx, gr); err != nil {
				// if fail to delete the group, return with error so that it can be retried
				msg := "unable to delete Group"
				log.Error(err, msg)
				gr.Status.Update(rbacv1alpha1.DeletingStatusPhase, msg, err)
				_ = r.Client.Status().Update(ctx, gr)
				return ctrl.Result{}, err
			}

			// remove finalizer from the list and update it.
			gr.ObjectMeta.Finalizers = utils.RemoveString(gr.ObjectMeta.Finalizers, finalizerID)
			if err := r.Update(ctx, gr); err != nil {
				msg := "could not remove finalizer"
				log.Error(err, msg)
				gr.Status.Update(rbacv1alpha1.DeletingStatusPhase, msg, err)
				_ = r.Client.Status().Update(ctx, gr)
				return ctrl.Result{}, err
			}
		}
		// finalizer already removed, nothing to do
		return ctrl.Result{}, nil
	}

	// register finalizer if it does not exist
	if !utils.ContainsString(gr.ObjectMeta.Finalizers, finalizerID) {
		gr.ObjectMeta.Finalizers = append(gr.ObjectMeta.Finalizers, finalizerID)
		if err := r.Update(ctx, gr); err != nil {
			return ctrl.Result{}, errors.Wrap(err, "could not add finalizer")
		}
		log.Info("Appended Finalizer")
	}

	// //initialize status bindings
	// if gr.Status.Bindings == nil {
	// 	gr.Status.Bindings = make(map[string][]string)
	// }
	// if gr.Status.BoundRoles == nil {
	// 	gr.Status.BoundRoles = make([]string, 0)
	// }

	gr.Status.Update(rbacv1alpha1.ReadyStatusPhase, successMessage, nil)

	err := r.Client.Status().Update(ctx, gr)
	return ctrl.Result{}, errors.Wrap(err, "could not update status")
}

func (r *GridOSGroupReconciler) deleteGroup(ctx context.Context, gr *rbacv1alpha1.GridOSGroup) error {
	log := logf.FromContext(ctx, "phase", "deleting group dependencies")
	log.Info(fmt.Sprintf("cleaning up dependencies before deleting group %v in namespace %v", gr.GetName(), gr.GetNamespace()))
	return nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *GridOSGroupReconciler) SetupWithManager(mgr ctrl.Manager) error {

	// isObjectUpdated := func(e event.UpdateEvent) bool {
	// 	if e.ObjectOld == nil {
	// 		fmt.Printf("UpdateAction event has no old runtime object to update - event : %v \n", e)
	// 		return false
	// 	}
	// 	if e.ObjectNew == nil {
	// 		fmt.Printf("UpdateAction event has no new runtime object for update - event : %v \n", e)
	// 		return false
	// 	}
	// 	groupNew, isGroupNew := e.ObjectNew.(*rbacv1alpha1.GridOSGroup)
	// 	groupOld, isGroupOld := e.ObjectOld.(*rbacv1alpha1.GridOSGroup)
	// 	if !isGroupNew || !isGroupOld {
	// 		return false
	// 	}
	// 	fmt.Printf("old finalizers - %v \n", groupOld.ObjectMeta.Finalizers)
	// 	fmt.Printf("new finalizers - %v \n", groupNew.ObjectMeta.Finalizers)
	// 	isFinalizersEqual := reflect.DeepEqual(groupNew.ObjectMeta.Finalizers, groupOld.ObjectMeta.Finalizers)
	// 	if !isFinalizersEqual {
	// 		fmt.Println("Finalizers changed")
	// 		return false
	// 	}
	// 	return true
	// }

	// isObjectUpdated := func(e event.UpdateEvent) bool {
	// 	oldGeneration := e.ObjectOld.GetGeneration()
	// 	newGeneration := e.ObjectNew.GetGeneration()
	// 	fmt.Printf("old generation - %v \n", oldGeneration)
	// 	fmt.Printf("new generation - %v \n", newGeneration)

	// 	groupOld, isGroupOld := e.ObjectOld.(*rbacv1alpha1.GridOSGroup)
	// 	groupNew, isGroupNew := e.ObjectNew.(*rbacv1alpha1.GridOSGroup)

	// 	if !isGroupNew || !isGroupOld {
	// 		return false
	// 	}
	// 	if oldGeneration != newGeneration {
	// 		jsonOld, _ := json.Marshal(groupOld)
	// 		jsonNew, _ := json.Marshal(groupNew)
	// 		fmt.Printf("old group object - %+v \n", string(jsonOld))
	// 		fmt.Printf("new group object - %+v \n", string(jsonNew))
	// 	}
	// 	return oldGeneration != newGeneration
	// }

	// eventFilter := predicate.Funcs{
	// 	CreateFunc: func(e event.CreateEvent) bool {
	// 		fmt.Println("CreateEvent called")
	// 		return true
	// 	},
	// 	DeleteFunc: func(e event.DeleteEvent) bool {
	// 		fmt.Println("DeleteEvent called")
	// 		return true
	// 	}, // TODO: Discuss how reconcile in case of Delete Event
	// 	UpdateFunc: func(e event.UpdateEvent) bool {
	// 		fmt.Println("UpdateEvent called")
	// 		return false
	// 		//return isObjectUpdated(e)
	// 	},
	// 	GenericFunc: func(e event.GenericEvent) bool {
	// 		fmt.Println("GenericEvent called")
	// 		return true
	// 	},
	// }

	pred := predicate.GenerationChangedPredicate{}

	return ctrl.NewControllerManagedBy(mgr).
		For(&rbacv1alpha1.GridOSGroup{}).
		WithEventFilter(pred).
		Named("gridosgroup").
		Complete(r)
}
