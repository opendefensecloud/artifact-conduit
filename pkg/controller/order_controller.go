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

	"go.opendefense.cloud/arc/api/arc/v1alpha1"
	arcv1alpha1 "go.opendefense.cloud/arc/api/arc/v1alpha1"
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
	wf     *arcv1alpha1.ArtifactWorkflow
	secret *corev1.Secret
	index  int
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
			for sha, _ := range order.Status.ArtifactWorkflows {
				// Remove Secret and ArtifactWorkflow
				aw := &arcv1alpha1.ArtifactWorkflow{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: order.Namespace,
						Name:      awName(order, sha),
					},
				}
				_ = r.Delete(ctx, aw) // ignore errors
				s := &corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: order.Namespace,
						Name:      awName(order, sha),
					},
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
		srcEndpoint := &arcv1alpha1.Endpoint{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: order.Namespace,
				Name:      srcRefName,
			},
		}
		if err := r.Get(ctx, client.ObjectKeyFromObject(srcEndpoint), srcEndpoint); err != nil {
			// TODO: should we set status to something and not error here?
			log.Error(err, "Failed to fetch Endpoint (srcRef) for Order")
			return ctrl.Result{}, err
		}

		dstEndpoint := &arcv1alpha1.Endpoint{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: order.Namespace,
				Name:      dstRefName,
			},
		}
		if err := r.Get(ctx, client.ObjectKeyFromObject(dstEndpoint), dstEndpoint); err != nil {
			// TODO: should we set status to something and not error here?
			log.Error(err, "Failed to fetch Endpoint (dstRef) for Order")
			return ctrl.Result{}, err
		}

		// Next, we need the secret contents

		// Create a hash based on fragment fields for idempotency and compute the fragment name
		h := sha256.New()
		data := map[string]any{
			"type": artifact.Type,
			"src":  srcRef.Name,
			"dst":  dstRef.Name,
			"spec": artifact.Spec.Raw,
		}
		jsonData, err := json.Marshal(data)
		if err != nil {
			return ctrl.Result{}, fmt.Errorf("failed to marshal fragment data: %w", err)
		}
		h.Write(jsonData)
		sha := hex.EncodeToString(h.Sum(nil))[:16]

		// Let's collect the necessary data for the fragment from the artifact and order
		aw := &arcv1alpha1.ArtifactWorkflow{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: order.Namespace,
				Name:      awName(order, sha),
			},
			Spec: arcv1alpha1.ArtifactWorkflowSpec{
				Type: artifact.Type,
				// TODO
				// SrcRef: artifact.SrcRef,
				// DstRef: artifact.DstRef,
				// Spec:   spec,
			},
		}

		desiredAWs[sha] = desiredAW{
			wf:     aw,
			secret: nil, // TODO
			index:  i,
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
		order.Status.ArtifactWorkflows = map[string]corev1.LocalObjectReference{}
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
		frag := desiredAWs[sha]

		// Set owner reference so ArtifactWorkflow is garbage-collected with the Order
		if err := controllerutil.SetControllerReference(order, frag, r.Scheme); err != nil {
			return ctrl.Result{}, err
		}

		if err := r.Create(ctx, frag); err != nil {
			if apierrors.IsAlreadyExists(err) {
				// Already created by a previous reconcile — that's fine
				continue
			}
			return ctrl.Result{}, err
		}

		// Update status
		order.Status.ArtifactWorkflows[sha] = corev1.LocalObjectReference{Name: frag.Name}
	}

	// Delete obsolete fragments
	for _, sha := range deleteAWs {
		// Does not exist anymore, let's clean up!
		if err := r.Delete(ctx, &arcv1alpha1.ArtifactWorkflow{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: order.Namespace,
				Name:      order.Status.ArtifactWorkflows[sha].Name,
			},
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

func awName(order *v1alpha1.Order, sha string) string {
	return fmt.Sprintf("%s-%s", order.Name, sha)
}
