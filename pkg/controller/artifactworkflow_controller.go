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
	fragmentFinalizer       = "arc.bwi.de/fragment-finalizer"
	workflowConfigSecretKey = "config.json"
)

// ArtifactWorkflowReconciler reconciles a ArtifactWorkflow object
type ArtifactWorkflowReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=arc.bwi.de,resources=endpoints,verbs=get;list;watch
//+kubebuilder:rbac:groups=arc.bwi.de,resources=fragments,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=arc.bwi.de,resources=fragments/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=arc.bwi.de,resources=fragments/finalizers,verbs=update
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

	// TODO: Is fragment status "done" or "error", then check if secret is still referenced in status.
	//       If secret exists, clean up and update status.

	// TODO: ArtifactWorkflow is not finished, then check if workflow is referenced in status.

	// TODO: If no workflow referenced, create secret and workflow.
	// log.V(1).Info("Checking if workflowConfigSecret already exists...")
	// configSecretName := types.NamespacedName{Name: frag.Name, Namespace: frag.Namespace}
	// foundWorkflowConfig := &corev1.Secret{}
	// if err := r.Get(ctx, configSecretName, foundWorkflowConfig); err != nil && apierrors.IsNotFound(err) {
	// 	log.Info("Creating new workflow config", "namespace", configSecretName.Namespace, "name", configSecretName.Name)
	// 	// Create configuration secret
	// 	workflowConfig, err := r.createWorkflowConfig(ctx, frag)
	// 	if err != nil {
	// 		log.Error(err, "Failed to create workflow config from fragment")
	// 		return ctrl.Result{}, err
	// 	}
	// 	json, err := workflowConfig.ToJson()
	// 	if err != nil {
	// 		log.Error(err, "Failed to marshal json from workflow config")
	// 		return ctrl.Result{}, err
	// 	}
	// 	configSecret := &corev1.Secret{
	// 		ObjectMeta: v1.ObjectMeta{
	// 			Name:      frag.Name,
	// 			Namespace: frag.Namespace,
	// 		},
	// 	}
	// 	configSecret.StringData = map[string]string{
	// 		"config.json": string(json),
	// 	}

	// 	// Set owner reference so Secret is garbage-collected with the ArtifactWorkflow
	// 	if err := controllerutil.SetControllerReference(frag, configSecret, r.Scheme); err != nil {
	// 		return ctrl.Result{}, err
	// 	}

	// 	// Create the Secret in the namespace of the ArtifactWorkflow
	// 	if err := r.Create(ctx, configSecret); err != nil {
	// 		log.Error(err, "Failed to create new workflow config", "namespace", configSecret.Namespace, "name", configSecret.Name)
	// 		return ctrl.Result{}, err
	// 	}

	// 	// Requeue the request to ensure the secret is created
	// 	return ctrl.Result{}, err
	// } else if err != nil {
	// 	log.Error(err, "Failed to get workflow config")
	// 	return ctrl.Result{}, err
	// }

	// TODO: If workflow exists, check and update status if necessary.

	return ctrl.Result{}, nil
}

// // createWorkflowConfig creates a new workflow config for the given fragment.
// func (r *ArtifactWorkflowReconciler) createWorkflowConfig(ctx context.Context, f *arcv1alpha1.ArtifactWorkflow) (*config.ArcctlConfig, error) {
// 	c := &config.ArcctlConfig{}
// 	c.Type = config.ArtifactType(f.Spec.Type)
// 	c.Spec = f.Spec.Spec

// 	srcEp, err := r.createWorkflowEndpoint(ctx, f.Namespace, f.Spec.SrcRef.Name)
// 	if err != nil {
// 		return nil, fmt.Errorf("failed to create source endpoint: %w", err)
// 	}
// 	c.Src = *srcEp

// 	dstEp, err := r.createWorkflowEndpoint(ctx, f.Namespace, f.Spec.DstRef.Name)
// 	if err != nil {
// 		return nil, fmt.Errorf("failed to create source endpoint: %w", err)
// 	}
// 	c.Dst = *dstEp

// 	return c, nil
// }

// // createWorkflowEndpoint creates a new workflow endpoint for the given reference.
// func (r *ArtifactWorkflowReconciler) createWorkflowEndpoint(ctx context.Context, namespace, name string) (*config.Endpoint, error) {
// 	ep, err := r.resolveEndpoint(ctx, namespace, name)
// 	if err != nil {
// 		return nil, fmt.Errorf("failed to resolve endpoint: %w", err)
// 	}

// 	confEp := &config.Endpoint{
// 		Type:      config.ArtifactType(ep.Spec.Type),
// 		RemoteURL: ep.Spec.RemoteURL,
// 	}

// 	if ep.Spec.SecretRef.Name != "" {
// 		secret, err := r.resolveSecret(ctx, ep.Namespace, ep.Spec.SecretRef.Name)
// 		if err != nil {
// 			return nil, fmt.Errorf("failed to resolve endpoint secret: %w", err)
// 		}
// 		confEp.Auth = secret.Data
// 	}

// 	return confEp, nil
// }

// SetupWithManager sets up the controller with the Manager.
func (r *ArtifactWorkflowReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&arcv1alpha1.ArtifactWorkflow{}).
		Owns(&wfv1alpha1.Workflow{}).
		Owns(&corev1.Secret{}).
		Complete(r)
}

// resolveEndpoint resolves the endpoint for a given reference.
// func (r *ArtifactWorkflowReconciler) resolveEndpoint(ctx context.Context, namespace, name string) (*arcv1alpha1.Endpoint, error) {
// 	ep := &arcv1alpha1.Endpoint{}
// 	if err := r.Get(ctx, types.NamespacedName{Name: name, Namespace: namespace}, ep); err != nil {
// 		return nil, fmt.Errorf("failed to get endpoint: %w", err)
// 	}
// 	return ep, nil
// }

// // resolveSecret resolves the secret for a given reference.
// func (r *ArtifactWorkflowReconciler) resolveSecret(ctx context.Context, namespace, name string) (*corev1.Secret, error) {
// 	secret := &corev1.Secret{}
// 	if err := r.Get(ctx, types.NamespacedName{Name: name, Namespace: namespace}, secret); err != nil {
// 		return nil, fmt.Errorf("failed to get secret: %w", err)
// 	}
// 	return secret, nil
// }
