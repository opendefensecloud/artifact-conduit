// Copyright 2025 BWI GmbH and Artefact Conduit contributors
// SPDX-License-Identifier: Apache-2.0

package controller

import (
	"context"
	"slices"

	wfv1alpha1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	arcv1alpha1 "go.opendefense.cloud/arc/api/arc/v1alpha1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const fragmentFinalizer = "arc.bwi.de/fragment-finalizer"

// FragmentReconciler reconciles a Fragment object
type FragmentReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=argoproj.io,resources=workflows,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=arc.bwi.de,resources=fragments,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=arc.bwi.de,resources=fragments/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=arc.bwi.de,resources=fragments/finalizers,verbs=update

// Reconcile moves the current state of the cluster closer to the desired state
func (r *FragmentReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := ctrl.LoggerFrom(ctx)

	// Fetch the Fragment object
	frg := &arcv1alpha1.Fragment{}
	if err := r.Get(ctx, req.NamespacedName, frg); err != nil {
		if apierrors.IsNotFound(err) {
			// Object not found, return.
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	// Handle deletion: cleanup fragments, then remove finalizer
	if !frg.DeletionTimestamp.IsZero() {
		log.V(1).Info("Fragment is being deleted")
		// TODO: remove workflow and secret if exists
		// Workflow and secret was cleaned up, remove finalizer
		if slices.Contains(frg.Finalizers, fragmentFinalizer) {
			log.V(1).Info("Removing finalizer from Fragment")
			frg.Finalizers = slices.DeleteFunc(frg.Finalizers, func(f string) bool {
				return f == finalizer
			})
			if err := r.Update(ctx, frg); err != nil {
				log.Error(err, "Failed to remove finalizer from Fragment")
				return ctrl.Result{}, err
			}
		}
		return ctrl.Result{}, nil
	}

	// Add finalizer if not present and not deleting
	if frg.DeletionTimestamp.IsZero() {
		if !slices.Contains(frg.Finalizers, fragmentFinalizer) {
			log.V(1).Info("Adding finalizer to Fragment")
			frg.Finalizers = append(frg.Finalizers, fragmentFinalizer)
			if err := r.Update(ctx, frg); err != nil {
				log.Error(err, "Failed to add finalizer to Fragment")
				return ctrl.Result{}, err
			}
			// Return without requeue; the Update event will trigger reconciliation again
			return ctrl.Result{}, nil
		}
	}

	// TODO: Is fragment status "done" or "error", then check if secret is still referenced in status.
	//       If secret exists, clean up and update status.

	// TODO: Fragment is not finished, then check if workflow is referenced in status.

	// TODO: If no workflow referenced, create secret and workflow.

	// TODO: If workflow exists, check and update status if necessary.

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *FragmentReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&arcv1alpha1.Fragment{}).
		Owns(&wfv1alpha1.Workflow{}).
		Complete(r)
}
