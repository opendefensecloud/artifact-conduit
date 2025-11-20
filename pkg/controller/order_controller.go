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
	"strconv"
	"strings"

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

const (
	orderFinalizer          = "arc.bwi.de/order-finalizer"
	workflowConfigSecretKey = "config.json"
)

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
//+kubebuilder:rbac:groups=arc.bwi.de,resources=artifactworkflows,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=arc.bwi.de,resources=orders,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=arc.bwi.de,resources=orders/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=arc.bwi.de,resources=orders/finalizers,verbs=update
//+kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch;create;update;patch;delete

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
		return ctrl.Result{}, errLogAndWrap(log, err, "failed to get object")
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
				_ = r.Delete(ctx, aw) // Ignore errors
				delete(order.Status.ArtifactWorkflows, sha)
			}
			if err := r.Status().Update(ctx, order); err != nil {
				return ctrl.Result{}, errLogAndWrap(log, err, "failed to update order status")
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
				return ctrl.Result{}, errLogAndWrap(log, err, "failed to remove finalizer")
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
				return ctrl.Result{}, errLogAndWrap(log, err, "failed to add finalizer")
			}
			// Return without requeue; the Update event will trigger reconciliation again
			return ctrl.Result{}, nil
		}
	}

	// Before we compare to our status, let's fetch all necessary information
	// to compute desired state:
	desiredAWs := map[string]desiredAW{}
	for i, artifact := range order.Spec.Artifacts {
		// TODO: When a endpoint or secret fetch fails, we stop the reconciliation of the whole order.
		//       Should we instead not fail but skip invalid artifacts?
		log := log.WithValues("artifactIndex", i)

		// We need the referenced src- and dst-endpoints for the artifact
		srcRefName := artifact.SrcRef.Name
		if srcRefName == "" {
			srcRefName = order.Spec.Defaults.SrcRef.Name
		}
		dstRefName := artifact.DstRef.Name
		if dstRefName == "" {
			dstRefName = order.Spec.Defaults.DstRef.Name
		}
		srcEndpoint := &arcv1alpha1.Endpoint{}
		if err := r.Get(ctx, namespacedName(order.Namespace, srcRefName), srcEndpoint); err != nil {
			return ctrl.Result{}, errLogAndWrap(log, err, "failed to fetch endpoint for source")
		}
		dstEndpoint := &arcv1alpha1.Endpoint{}
		if err := r.Get(ctx, namespacedName(order.Namespace, dstRefName), dstEndpoint); err != nil {
			return ctrl.Result{}, errLogAndWrap(log, err, "failed to fetch endpoint for destination")
		}

		// Next, we need the secret contents
		srcSecret := &corev1.Secret{}
		if srcEndpoint.Spec.SecretRef.Name != "" {
			if err := r.Get(ctx, namespacedName(order.Namespace, srcEndpoint.Spec.SecretRef.Name), srcSecret); err != nil {
				return ctrl.Result{}, errLogAndWrap(log, err, "failed to fetch secret for source")
			}
		}

		dstSecret := &corev1.Secret{}
		if dstEndpoint.Spec.SecretRef.Name != "" {
			if err := r.Get(ctx, namespacedName(order.Namespace, dstEndpoint.Spec.SecretRef.Name), dstSecret); err != nil {
				return ctrl.Result{}, errLogAndWrap(log, err, "failed to fetch secret for destination")
			}
		}

		// Create a hash based on all related data for idempotency and compute the workflow name
		h := sha256.New()
		data := []any{
			order.Namespace,
			artifact.Type, artifact.Spec.Raw,
			srcEndpoint.Name, srcEndpoint.Generation,
			dstEndpoint.Name, dstEndpoint.Generation,
			srcSecret.Name, srcSecret.Generation,
			dstSecret.Name, dstSecret.Generation,
		}
		jsonData, err := json.Marshal(data)
		if err != nil {
			return ctrl.Result{}, errLogAndWrap(log, err, "failed to marshal fragment data")
		}
		h.Write(jsonData)
		sha := hex.EncodeToString(h.Sum(nil))[:16]

		// We gave all the information to further process this artifact workflow.
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
		aw, err := r.hydrateArtifactWorkflow(&daw)
		if err != nil {
			return ctrl.Result{}, errLogAndWrap(log, err, "failed to hydrate artifact workflow")
		}

		// Set owner references
		if err := controllerutil.SetControllerReference(order, aw, r.Scheme); err != nil {
			return ctrl.Result{}, errLogAndWrap(log, err, "failed to set controller reference")
		}

		// Create artifact workflow
		if err := r.Create(ctx, aw); err != nil {
			if apierrors.IsAlreadyExists(err) {
				// Already created by a previous reconcile — that's fine
				continue
			}
			return ctrl.Result{}, errLogAndWrap(log, err, "failed to create artifact workflow")
		}

		// Update status
		order.Status.ArtifactWorkflows[sha] = arcv1alpha1.OrderArtifactWorkflowStatus{
			ArtifactIndex: daw.index,
			Phase:         arcv1alpha1.WorkflowUnknown,
		}
	}

	// Delete obsolete fragments
	for _, sha := range deleteAWs {
		// Does not exist anymore, let's clean up!
		if err := r.Delete(ctx, &arcv1alpha1.ArtifactWorkflow{
			ObjectMeta: awObjectMeta(order, sha),
		}); client.IgnoreNotFound(err) != nil {
			return ctrl.Result{}, errLogAndWrap(log, err, "failed to delete artifact workflow")
		}

		// Update status
		delete(order.Status.ArtifactWorkflows, sha)
	}

	anyPhaseChanged := false
	for sha, daw := range desiredAWs {
		if slices.Contains(createAWs, sha) {
			// If it was just created we skip the update
			continue
		}
		aw := arcv1alpha1.ArtifactWorkflow{}
		if err := r.Get(ctx, namespacedName(daw.objectMeta.Namespace, daw.objectMeta.Name), &aw); err != nil {
			return ctrl.Result{}, errLogAndWrap(log, err, "failed to get artifact workflow")
		}
		if order.Status.ArtifactWorkflows[sha].Phase != aw.Status.Phase {
			awStatus := order.Status.ArtifactWorkflows[sha]
			awStatus.Phase = aw.Status.Phase
			order.Status.ArtifactWorkflows[sha] = awStatus
			anyPhaseChanged = true
		}
	}

	// Update status
	if len(createAWs) > 0 || len(deleteAWs) > 0 || anyPhaseChanged {
		log.V(1).Info("Updating order status")
		// Make sure ArtifactIndex is up to date
		for sha, daw := range desiredAWs {
			aws := order.Status.ArtifactWorkflows[sha]
			aws.ArtifactIndex = daw.index
			order.Status.ArtifactWorkflows[sha] = aws
		}
		if err := r.Status().Update(ctx, order); err != nil {
			return ctrl.Result{}, errLogAndWrap(log, err, "failed to update status")
		}
	}

	return ctrl.Result{}, nil
}

func (r *OrderReconciler) hydrateArtifactWorkflow(daw *desiredAW) (*arcv1alpha1.ArtifactWorkflow, error) {
	params, err := dawToParameters(daw)
	if err != nil {
		return nil, err
	}

	// Next we create the ArtifactWorkflow instance
	aw := &arcv1alpha1.ArtifactWorkflow{
		ObjectMeta: daw.objectMeta,
		Spec: arcv1alpha1.ArtifactWorkflowSpec{
			Type:         daw.artifact.Type,
			Parameters:   params,
			SrcSecretRef: daw.srcEndpoint.Spec.SecretRef,
			DstSecretRef: daw.dstEndpoint.Spec.SecretRef,
		},
	}

	return aw, nil
}

// generateReconcileRequestsForEndpoint generates reconcile requests for all Endpoints referenced by an Order
func (r *OrderReconciler) generateReconcileRequestsForEndpoint(ctx context.Context, endpoint client.Object) []reconcile.Request {
	resourcesReferencingEndpoint := &arcv1alpha1.OrderList{}
	listOps := &client.ListOptions{
		FieldSelector: fields.SelectorFromSet(fields.Set{".spec.srcRef.name": endpoint.GetName(), ".spec.dstRef.name": endpoint.GetName()}),
		Namespace:     endpoint.GetNamespace(),
	}
	err := r.List(ctx, resourcesReferencingEndpoint, listOps)
	if err != nil {
		return []reconcile.Request{}
	}

	requests := make([]reconcile.Request, len(resourcesReferencingEndpoint.Items))
	for i, item := range resourcesReferencingEndpoint.Items {
		log := ctrl.LoggerFrom(ctx)
		log.V(1).Info("Generating reconcile request for resource because referenced endpoint has changed...")
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
			&arcv1alpha1.Endpoint{},
			handler.EnqueueRequestsFromMapFunc(r.generateReconcileRequestsForEndpoint),
			builder.WithPredicates(predicate.ResourceVersionChangedPredicate{}),
		).
		Owns(&arcv1alpha1.ArtifactWorkflow{}).
		Complete(r)
}

func namespacedName(namespace, name string) types.NamespacedName {
	return types.NamespacedName{
		Namespace: namespace,
		Name:      name,
	}
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

// TODO: add unit tests
func dawToParameters(daw *desiredAW) ([]arcv1alpha1.ArtifactWorkflowParameter, error) {
	params := []arcv1alpha1.ArtifactWorkflowParameter{
		{
			Name:  paramName("src", "type"),
			Value: string(daw.srcEndpoint.Spec.Type),
		},
		{
			Name:  paramName("src", "remoteURL"),
			Value: daw.srcEndpoint.Spec.RemoteURL,
		},
		{
			Name:  paramName("dst", "type"),
			Value: string(daw.dstEndpoint.Spec.Type),
		},
		{
			Name:  paramName("dst", "remoteURL"),
			Value: daw.dstEndpoint.Spec.RemoteURL,
		},
		{
			Name:  "srcSecret",
			Value: fmt.Sprintf("%v", daw.srcEndpoint.Spec.SecretRef.Name != ""),
		},
		{
			Name:  "dstSecret",
			Value: fmt.Sprintf("%v", daw.dstEndpoint.Spec.SecretRef.Name != ""),
		},
	}

	spec := map[string]any{}
	raw := daw.artifact.Spec.Raw
	if len(raw) == 0 {
		raw = []byte("{}")
	}
	if err := json.Unmarshal(raw, &spec); err != nil {
		return nil, err
	}
	flattened := map[string]any{}
	flattenMap("spec", spec, flattened)
	for name, value := range flattened {
		params = append(params, arcv1alpha1.ArtifactWorkflowParameter{
			Name:  name,
			Value: fmt.Sprintf("%v", value),
		})
	}

	return params, nil
}

// TODO: add unit tests
func paramName(prefix, suffix string) string {
	return prefix + strings.ToUpper(suffix[:1]) + suffix[1:]
}

// TODO: add unit tests
func flattenMap(prefix string, src map[string]any, dst map[string]any) {
	for k, v := range src {
		kt := strings.ToUpper(k[:1]) + k[1:]
		switch child := v.(type) {
		case map[string]any:
			flattenMap(prefix+k, child, dst)
		case []any:
			for i, av := range child {
				dst[prefix+kt+strconv.Itoa(i)] = av
			}
		default:
			dst[prefix+kt] = v
		}
	}
}
