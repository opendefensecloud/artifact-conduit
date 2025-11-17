// Copyright 2025 BWI GmbH and Artifact Conduit contributors
// SPDX-License-Identifier: Apache-2.0

package controller

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"slices"

	arcv1alpha1 "go.opendefense.cloud/arc/api/arc/v1alpha1"
	"go.opendefense.cloud/arc/pkg/workflow/config"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const orderFinalizer = "arc.bwi.de/order-finalizer"

// OrderReconciler reconciles a Order object
type OrderReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

type desiredAW struct {
	index       int
	objectMeta  metav1.ObjectMeta
	artifact    *arcv1alpha1.OrderArtifact
	srcEndpoint *arcv1alpha1.Endpoint
	dstEndpoint *arcv1alpha1.Endpoint
	srcSecret   *corev1.Secret
	dstSecret   *corev1.Secret
}

//+kubebuilder:rbac:groups=arc.bwi.de,resources=endpoints,verbs=get;list;watch
//+kubebuilder:rbac:groups=arc.bwi.de,resources=fragments,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=arc.bwi.de,resources=orders,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=arc.bwi.de,resources=orders/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=arc.bwi.de,resources=orders/finalizers,verbs=update
//+kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch

// Reconcile moves the current state of the cluster closer to the desired state
func (r *OrderReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := ctrl.LoggerFrom(ctx)

	// Fetch the Order instance
	order := &arcv1alpha1.Order{}
	if err := r.Get(ctx, req.NamespacedName, order); err != nil {
		if apierrors.IsNotFound(err) {
			// Object not found, return. Created objects are automatically garbage collected.
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	// Handle deletion: cleanup fragments, then remove finalizer
	if !order.DeletionTimestamp.IsZero() {
		log.V(1).Info("Order is being deleted")
		if len(order.Status.ArtifactWorkflows) > 0 {
			for sha := range order.Status.ArtifactWorkflows {
				// Remove Secret and ArtifactWorkflow
				aw := &arcv1alpha1.ArtifactWorkflow{
					ObjectMeta: awObjectMeta(order, sha),
				}
				_ = r.Delete(ctx, aw) // ignore errors
				s := &corev1.Secret{
					ObjectMeta: awObjectMeta(order, sha),
				}
				_ = r.Delete(ctx, s) // ignore errors
				delete(order.Status.ArtifactWorkflows, sha)
			}
			if err := r.Status().Update(ctx, order); err != nil {
				log.Error(err, "Failed to update artifact workflows in Order.Status")
				return ctrl.Result{}, err
			}
			log.V(1).Info("Order artifact workflows cleaned up")
			// Requeue until all fragments are gone
			return ctrl.Result{Requeue: true}, nil
		}
		// All fragments are gone, remove finalizer
		if slices.Contains(order.Finalizers, orderFinalizer) {
			log.V(1).Info("No artifact workflows, removing finalizer from Order")
			order.Finalizers = slices.DeleteFunc(order.Finalizers, func(f string) bool {
				return f == orderFinalizer
			})
			if err := r.Update(ctx, order); err != nil {
				log.Error(err, "Failed to remove finalizer from Order")
				return ctrl.Result{}, err
			}
		}
		return ctrl.Result{}, nil
	}

	// Add finalizer if not present and not deleting
	if order.DeletionTimestamp.IsZero() {
		if !slices.Contains(order.Finalizers, orderFinalizer) {
			log.V(1).Info("Adding finalizer to Order")
			order.Finalizers = append(order.Finalizers, orderFinalizer)
			if err := r.Update(ctx, order); err != nil {
				log.Error(err, "Failed to add finalizer to Order")
				return ctrl.Result{}, err
			}
			// Return without requeue; the Update event will trigger reconciliation again
			return ctrl.Result{}, nil
		}
	}

	desiredAWs := map[string]desiredAW{}
	for i, artifact := range order.Spec.Artifacts {
		log := log.WithValues("artifactIndex", i)

		// Let's get the endpoint names
		srcRefName := artifact.SrcRef.Name
		if srcRefName == "" {
			srcRefName = order.Spec.Defaults.SrcRef.Name
		}
		dstRefName := artifact.DstRef.Name
		if dstRefName == "" {
			dstRefName = order.Spec.Defaults.DstRef.Name
		}

		// Let's fetch the endpoints
		srcEndpoint := &arcv1alpha1.Endpoint{}
		if err := r.Get(ctx, namespacedName(order.Namespace, srcRefName), srcEndpoint); err != nil {
			// TODO: should we set status to something and not error here?
			log.Error(err, "Failed to fetch Endpoint (srcRef) for Order")
			return ctrl.Result{}, err
		}

		dstEndpoint := &arcv1alpha1.Endpoint{}
		if err := r.Get(ctx, namespacedName(order.Namespace, dstRefName), dstEndpoint); err != nil {
			// TODO: should we set status to something and not error here?
			log.Error(err, "Failed to fetch Endpoint (dstRef) for Order")
			return ctrl.Result{}, err
		}

		// Next, we need the secret contents
		srcSecret := &corev1.Secret{}
		if err := r.Get(ctx, namespacedName(order.Namespace, srcEndpoint.Spec.SecretRef.Name), srcSecret); err != nil {
			// TODO: should we set status to something and not error here?
			log.Error(err, "Failed to fetch Secret for source of Order")
			return ctrl.Result{}, err
		}

		dstSecret := &corev1.Secret{}
		if err := r.Get(ctx, namespacedName(order.Namespace, dstEndpoint.Spec.SecretRef.Name), dstSecret); err != nil {
			// TODO: should we set status to something and not error here?
			log.Error(err, "Failed to fetch Secret for source of Order")
			return ctrl.Result{}, err
		}

		// Create a hash based on all related data for idempotency and compute the workflow name
		h := sha256.New()
		data := map[string]any{
			"type":        artifact.Type,
			"spec":        artifact.Spec.Raw,
			"srcEndpoint": srcEndpoint.Generation,
			"dstEndpoint": dstEndpoint.Generation,
			"srcSecret":   srcSecret.Generation,
			"dstSecret":   dstSecret.Generation,
		}
		jsonData, err := json.Marshal(data)
		if err != nil {
			return ctrl.Result{}, fmt.Errorf("failed to marshal fragment data: %w", err)
		}
		h.Write(jsonData)
		sha := hex.EncodeToString(h.Sum(nil))[:16]

		// We gave all the information to furhter process this artifact workflow.
		// Let's store it to compare it to the current status!
		desiredAWs[sha] = desiredAW{
			index:       i,
			objectMeta:  awObjectMeta(order, sha),
			artifact:    &artifact,
			srcEndpoint: srcEndpoint,
			dstEndpoint: dstEndpoint,
			srcSecret:   srcSecret,
			dstSecret:   dstSecret,
		}
	}

	// List missing fragments
	createAWs := []string{}
	for sha := range desiredAWs {
		_, exists := order.Status.ArtifactWorkflows[sha]
		if exists {
			continue
		}
		createAWs = append(createAWs, sha)
	}

	// Make sure status is initialized
	if order.Status.ArtifactWorkflows == nil {
		order.Status.ArtifactWorkflows = map[string]arcv1alpha1.OrderArtifactWorkflowStatus{}
	}

	// Find obsolete fragments
	deleteAWs := []string{}
	for sha := range order.Status.ArtifactWorkflows {
		_, exists := desiredAWs[sha]
		if exists {
			continue
		}
		deleteAWs = append(deleteAWs, sha)
	}

	// Create missing fragments
	for _, sha := range createAWs {
		daw := desiredAWs[sha]
		aw, secret, err := r.createArtifactWorkflowSecretPair(&daw)
		if err != nil {
			return ctrl.Result{}, err
		}

		// Set owner references
		if err := controllerutil.SetControllerReference(order, aw, r.Scheme); err != nil {
			return ctrl.Result{}, err
		}
		if err := controllerutil.SetControllerReference(order, secret, r.Scheme); err != nil {
			return ctrl.Result{}, err
		}

		// Create objects
		if err := r.Create(ctx, secret); err != nil {
			if apierrors.IsAlreadyExists(err) {
				// Already created by a previous reconcile — that's fine
				continue
			}
			return ctrl.Result{}, err
		}
		if err := r.Create(ctx, aw); err != nil {
			if apierrors.IsAlreadyExists(err) {
				// Already created by a previous reconcile — that's fine
				continue
			}
			return ctrl.Result{}, err
		}

		// Update status
		order.Status.ArtifactWorkflows[sha] = arcv1alpha1.OrderArtifactWorkflowStatus{
			ArtifactIndex: daw.index,
			Phase:         arcv1alpha1.WorkflowPhaseInit,
		}
	}

	// Delete obsolete fragments
	for _, sha := range deleteAWs {
		// Does not exist anymore, let's clean up!
		if err := r.Delete(ctx, &arcv1alpha1.ArtifactWorkflow{
			ObjectMeta: awObjectMeta(order, sha),
		}); err != nil {
			return ctrl.Result{}, err
		}

		// Update status
		delete(order.Status.ArtifactWorkflows, sha)
	}

	// Update status
	if len(createAWs) > 0 || len(deleteAWs) > 0 {
		log.V(1).Info("Updating Order.Status")
		if err := r.Status().Update(ctx, order); err != nil {
			return ctrl.Result{}, err
		}
	}

	return ctrl.Result{}, nil
}

func (r *OrderReconciler) createArtifactWorkflowSecretPair(daw *desiredAW) (*arcv1alpha1.ArtifactWorkflow, *corev1.Secret, error) {
	// Let's create the secret object first
	wc := &config.ArcctlConfig{
		Type: config.ArtifactType(daw.artifact.Type),
		Spec: daw.artifact.Spec,
		Src: config.Endpoint{
			Type:      config.EndpointType(daw.srcEndpoint.Spec.Type),
			RemoteURL: daw.srcEndpoint.Spec.RemoteURL,
			Auth:      daw.srcSecret.Data,
		},
		Dst: config.Endpoint{
			Type:      config.EndpointType(daw.dstEndpoint.Spec.Type),
			RemoteURL: daw.dstEndpoint.Spec.RemoteURL,
			Auth:      daw.dstSecret.Data,
		},
	}
	json, err := wc.ToJson()
	if err != nil {
		return nil, nil, err
	}
	secret := &corev1.Secret{
		ObjectMeta: daw.objectMeta,
	}
	secret.StringData = map[string]string{
		"config.json": string(json),
	}

	// Next we create the ArtifactWorkflow instance
	aw := &arcv1alpha1.ArtifactWorkflow{
		ObjectMeta: daw.objectMeta,
		Spec: arcv1alpha1.ArtifactWorkflowSpec{
			// TODO: Parameters
			// Parameters: []wfv1alpha1.Parameter{},
			SecretRef: corev1.LocalObjectReference{
				Name: daw.objectMeta.Name,
			},
		},
	}

	return aw, secret, nil
}

// generateReconcileRequestsForSecret generates reconcile requests for all secrets referenced by an Order
func (r *OrderReconciler) generateReconcileRequestsForSecret(ctx context.Context, secret client.Object) []reconcile.Request {
	resourcesReferencingSecret := &arcv1alpha1.OrderList{}
	listOps := &client.ListOptions{
		FieldSelector: fields.SelectorFromSet(fields.Set{".spec.srcRef.name": secret.GetName(), ".spec.dstRef.name": secret.GetName()}),
		Namespace:     secret.GetNamespace(),
	}
	err := r.List(ctx, resourcesReferencingSecret, listOps)
	if err != nil {
		return []reconcile.Request{}
	}

	requests := make([]reconcile.Request, len(resourcesReferencingSecret.Items))
	for i, item := range resourcesReferencingSecret.Items {
		log := ctrl.LoggerFrom(ctx)
		log.V(1).Info("Generating reconcile request for resource because referenced secret has changed...")
		requests[i] = reconcile.Request{
			NamespacedName: types.NamespacedName{
				Name:      item.GetName(),
				Namespace: item.GetNamespace(),
			},
		}
	}
	return requests
}

// SetupWithManager sets up the controller with the Manager.
func (r *OrderReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&arcv1alpha1.Order{}).
		Watches(
			&corev1.Secret{},
			handler.EnqueueRequestsFromMapFunc(r.generateReconcileRequestsForSecret),
			builder.WithPredicates(predicate.ResourceVersionChangedPredicate{}),
		).
		Owns(&arcv1alpha1.ArtifactWorkflow{}).
		Complete(r)
}

func awName(order *arcv1alpha1.Order, sha string) string {
	return fmt.Sprintf("%s-%s", order.Name, sha)
}

func awObjectMeta(order *arcv1alpha1.Order, sha string) metav1.ObjectMeta {
	return metav1.ObjectMeta{
		Namespace: order.Namespace,
		Name:      awName(order, sha),
	}
}

func namespacedName(namespace, name string) types.NamespacedName {
	return types.NamespacedName{
		Namespace: namespace,
		Name:      name,
	}
}
