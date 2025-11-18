// Copyright 2025 BWI GmbH and Artefact Conduit contributors
// SPDX-License-Identifier: Apache-2.0

package controller

import (
	"context"
	"slices"

	wfv1alpha1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	arcv1alpha1 "go.opendefense.cloud/arc/api/arc/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	fragmentFinalizer = "arc.bwi.de/artifact-workflow-finalizer"
)

// ArtifactWorkflowReconciler reconciles a ArtifactWorkflow object
type ArtifactWorkflowReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=arc.bwi.de,resources=artifactworkflows,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=arc.bwi.de,resources=artifactworkflows/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=arc.bwi.de,resources=artifactworkflows/finalizers,verbs=update
//+kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=argoproj.io,resources=workflows,verbs=get;list;watch;create;update;patch;delete

// Reconcile moves the current state of the cluster closer to the desired state
func (r *ArtifactWorkflowReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := ctrl.LoggerFrom(ctx)

	// Fetch the ArtifactWorkflow object
	frag := &arcv1alpha1.ArtifactWorkflow{}
	if err := r.Get(ctx, req.NamespacedName, frag); err != nil {
		if apierrors.IsNotFound(err) {
			// Object not found, return.
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	// Handle deletion: cleanup fragments, then remove finalizer
	if !frag.DeletionTimestamp.IsZero() {
		log.V(1).Info("ArtifactWorkflow is being deleted")
		// TODO: remove workflow and secret if exists
		// Workflow and secret was cleaned up, remove finalizer
		if slices.Contains(frag.Finalizers, fragmentFinalizer) {
			log.V(1).Info("Removing finalizer from ArtifactWorkflow")
			frag.Finalizers = slices.DeleteFunc(frag.Finalizers, func(f string) bool {
				return f == fragmentFinalizer
			})
			if err := r.Update(ctx, frag); err != nil {
				log.Error(err, "Failed to remove finalizer from ArtifactWorkflow")
				return ctrl.Result{}, err
			}
		}
		return ctrl.Result{}, nil
	}

	// Add finalizer if not present and not deleting
	if frag.DeletionTimestamp.IsZero() {
		if !slices.Contains(frag.Finalizers, fragmentFinalizer) {
			log.V(1).Info("Adding finalizer to ArtifactWorkflow")
			frag.Finalizers = append(frag.Finalizers, fragmentFinalizer)
			if err := r.Update(ctx, frag); err != nil {
				log.Error(err, "Failed to add finalizer to ArtifactWorkflow")
				return ctrl.Result{}, err
			}
			// Return without requeue; the Update event will trigger reconciliation again
			return ctrl.Result{}, nil
		}
	}

	// TODO: track status if workload exists
	// TODO: create workflow if not exists (and not status done|error)

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *ArtifactWorkflowReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&arcv1alpha1.ArtifactWorkflow{}).
		Owns(&wfv1alpha1.Workflow{}).
		Owns(&corev1.Secret{}).
		Complete(r)
}
