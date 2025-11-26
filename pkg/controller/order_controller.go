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
	"time"

	"github.com/go-logr/logr"
	arcv1alpha1 "go.opendefense.cloud/arc/api/arc/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

const (
	orderFinalizer = "arc.bwi.de/order-finalizer"
)

// OrderReconciler reconciles a Order object
type OrderReconciler struct {
	client.Client
	Scheme   *runtime.Scheme
	Recorder record.EventRecorder
}

type desiredAW struct {
	index       int
	objectMeta  metav1.ObjectMeta
	artifact    *arcv1alpha1.OrderArtifact
	typeSpec    *arcv1alpha1.ArtifactTypeSpec
	srcEndpoint *arcv1alpha1.Endpoint
	dstEndpoint *arcv1alpha1.Endpoint
	srcSecret   *corev1.Secret
	dstSecret   *corev1.Secret
	sha         string
}

//+kubebuilder:rbac:groups=arc.bwi.de,resources=endpoints,verbs=get;list;watch
//+kubebuilder:rbac:groups=arc.bwi.de,resources=artifacttypes,verbs=get;list;watch
//+kubebuilder:rbac:groups=arc.bwi.de,resources=clusterartifacttypes,verbs=get;list;watch
//+kubebuilder:rbac:groups=arc.bwi.de,resources=artifactworkflows,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=arc.bwi.de,resources=orders,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=arc.bwi.de,resources=orders/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=arc.bwi.de,resources=orders/finalizers,verbs=update
//+kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=events,verbs=create;patch

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

	// Handle deletion: cleanup artifact workflows, then remove finalizer
	if !order.DeletionTimestamp.IsZero() {
		log.V(1).Info("Order is being deleted")
		r.Recorder.Event(order, corev1.EventTypeWarning, "Deleting", "Order is being deleted, cleaning up artifact workflows")

		// Cleanup all artifact workflows
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

			// Requeue until all artifact workflows are gone
			return ctrl.Result{}, nil
		}
		// All artifact workflows are gone, remove finalizer
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

	// Make sure status is initialized
	if order.Status.ArtifactWorkflows == nil {
		order.Status.ArtifactWorkflows = map[string]arcv1alpha1.OrderArtifactWorkflowStatus{}
	}

	// Before we compare to our status, let's fetch all necessary information
	// to compute desired state:
	desiredAWs := map[string]desiredAW{}
	for i, artifact := range order.Spec.Artifacts {
		daw, err := r.computeDesiredAW(ctx, log, order, &artifact, i)
		if err != nil {
			r.Recorder.Event(order, corev1.EventTypeWarning, "ComputationFailed", fmt.Sprintf("Failed to compute desired artifact workflow for artifact index %d: %v", i, err))
			order.Status.Message = fmt.Sprintf("Failed to compute desired artifact workflow for artifact index %d: %v", i, err)
			if err := r.Status().Update(ctx, order); err != nil {
				return ctrl.Result{}, errLogAndWrap(log, err, "failed to update status")
			}
			return ctrl.Result{}, errLogAndWrap(log, err, "failed to compute desired artifact workflow")
		}
		desiredAWs[daw.sha] = *daw
	}

	// List missing artifact workflows
	createAWs := []string{}
	for sha := range desiredAWs {
		_, exists := order.Status.ArtifactWorkflows[sha]
		if exists {
			continue
		}
		createAWs = append(createAWs, sha)
	}

	// Find obsolete artifact workflows
	deleteAWs := []string{}
	for sha := range order.Status.ArtifactWorkflows {
		_, exists := desiredAWs[sha]
		if exists {
			continue
		}
		deleteAWs = append(deleteAWs, sha)
	}

	// Find finished artifact workflows to clean up
	finishedAWs := []string{}
	for sha := range order.Status.ArtifactWorkflows {
		awStatus := order.Status.ArtifactWorkflows[sha]

		// Only consider succeeded workflows for TTL cleanup
		if awStatus.Phase != arcv1alpha1.WorkflowSucceeded {
			continue
		}

		// If TTL is set, check if it has expired
		if order.Spec.TTLSecondsAfterCompletion != nil && *order.Spec.TTLSecondsAfterCompletion > 0 {
			if time.Since(awStatus.CompletionTime.Time) > time.Duration(*order.Spec.TTLSecondsAfterCompletion)*time.Second {
				finishedAWs = append(finishedAWs, sha)
			}
			continue
		}

		// No TTL set, cleanup immediately
		finishedAWs = append(finishedAWs, sha)
	}

	// Create missing artifact workflows
	for _, sha := range createAWs {
		daw := desiredAWs[sha]
		aw, err := r.hydrateArtifactWorkflow(&daw)
		if err != nil {
			r.Recorder.Event(order, corev1.EventTypeWarning, "HydrationFailed", fmt.Sprintf("Failed to hydrate artifact workflow for artifact index %d: %v", daw.index, err))
			return ctrl.Result{}, errLogAndWrap(log, err, "failed to hydrate artifact workflow")
		}

		// Set owner references
		if err := controllerutil.SetControllerReference(order, aw, r.Scheme); err != nil {
			r.Recorder.Event(order, corev1.EventTypeWarning, "HydrationFailed", fmt.Sprintf("Failed to set controller reference for artifact workflow: %v", err))
			return ctrl.Result{}, errLogAndWrap(log, err, "failed to set controller reference")
		}

		// Create artifact workflow
		if err := r.Create(ctx, aw); err != nil {
			if apierrors.IsAlreadyExists(err) {
				// Already created by a previous reconcile — that's fine
				continue
			}
			r.Recorder.Event(order, corev1.EventTypeWarning, "CreationFailed", fmt.Sprintf("Failed to create artifact workflow for artifact index %d: %v", daw.index, err))
			return ctrl.Result{}, errLogAndWrap(log, err, "failed to create artifact workflow")
		}

		// Update status
		order.Status.ArtifactWorkflows[sha] = arcv1alpha1.OrderArtifactWorkflowStatus{
			ArtifactIndex: daw.index,
			Phase:         arcv1alpha1.WorkflowUnknown,
		}

		r.Recorder.Event(order, corev1.EventTypeNormal, "ArtifactWorkflowCreated", fmt.Sprintf("Created artifact workflow '%s' for artifact index %d", aw.Name, daw.index))
		log.V(1).Info("Created artifact workflow", "artifactWorkflow", aw.Name)
	}

	// Delete obsolete artifact workflows
	for _, sha := range deleteAWs {
		// Does not exist anymore, let's clean up!
		if err := r.Delete(ctx, &arcv1alpha1.ArtifactWorkflow{
			ObjectMeta: awObjectMeta(order, sha),
		}); client.IgnoreNotFound(err) != nil {
			r.Recorder.Event(order, corev1.EventTypeWarning, "DeletionFailed", fmt.Sprintf("Failed to delete obsolete artifact workflow '%s': %v", sha, err))
			return ctrl.Result{}, errLogAndWrap(log, err, "failed to delete artifact workflow")
		}

		// Update status
		delete(order.Status.ArtifactWorkflows, sha)
		log.V(1).Info("Deleted obsolete artifact workflow", "artifactWorkflow", sha)
		r.Recorder.Event(order, corev1.EventTypeNormal, "ArtifactWorkflowDeleted", fmt.Sprintf("Deleted obsolete artifact workflow '%s'", sha))
	}

	// Delete finished artifact workflows
	for _, sha := range finishedAWs {
		// Finished, let's clean up!
		if err := r.Delete(ctx, &arcv1alpha1.ArtifactWorkflow{
			ObjectMeta: awObjectMeta(order, sha),
		}); client.IgnoreNotFound(err) != nil {
			r.Recorder.Event(order, corev1.EventTypeWarning, "DeletionFailed", fmt.Sprintf("Failed to delete finished artifact workflow '%s': %v", sha, err))
			return ctrl.Result{}, errLogAndWrap(log, err, "failed to delete artifact workflow")
		}

		log.V(1).Info("Deleted finished artifact workflow", "artifactWorkflow", sha)
		r.Recorder.Event(order, corev1.EventTypeNormal, "ArtifactWorkflowDeleted", fmt.Sprintf("Deleted finished artifact workflow '%s'", sha))
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
			awStatus.CompletionTime = aw.Status.CompletionTime
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
			WorkflowTemplateRef: daw.typeSpec.WorkflowTemplateRef,
			Parameters:          params,
			SrcSecretRef:        daw.srcEndpoint.Spec.SecretRef,
			DstSecretRef:        daw.dstEndpoint.Spec.SecretRef,
		},
	}

	return aw, nil
}

func (r *OrderReconciler) computeDesiredAW(ctx context.Context, log logr.Logger, order *arcv1alpha1.Order, artifact *arcv1alpha1.OrderArtifact, i int) (*desiredAW, error) {
	log = log.WithValues("artifactIndex", i)

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
		r.Recorder.Event(order, corev1.EventTypeWarning, "InvalidEndpoint", fmt.Sprintf("Failed to fetch source endpoint '%s': %v", srcRefName, err))
		return nil, errLogAndWrap(log, err, "failed to fetch endpoint for source")
	}
	dstEndpoint := &arcv1alpha1.Endpoint{}
	if err := r.Get(ctx, namespacedName(order.Namespace, dstRefName), dstEndpoint); err != nil {
		r.Recorder.Event(order, corev1.EventTypeWarning, "InvalidEndpoint", fmt.Sprintf("Failed to fetch destination endpoint '%s': %v", dstRefName, err))
		return nil, errLogAndWrap(log, err, "failed to fetch endpoint for destination")
	}

	// Validate that the endpoint usage is correct
	if srcEndpoint.Spec.Usage != arcv1alpha1.EndpointUsagePullOnly && srcEndpoint.Spec.Usage != arcv1alpha1.EndpointUsageAll {
		err := fmt.Errorf("endpoint '%s' usage '%s' is not compatible with source usage", srcEndpoint.Name, srcEndpoint.Spec.Usage)
		r.Recorder.Event(order, corev1.EventTypeWarning, "InvalidEndpoint", fmt.Sprintf("Source endpoint '%s' has incompatible usage '%s'", srcEndpoint.Name, srcEndpoint.Spec.Usage))
		return nil, errLogAndWrap(log, err, "artifact validation failed")
	}
	if dstEndpoint.Spec.Usage != arcv1alpha1.EndpointUsagePushOnly && dstEndpoint.Spec.Usage != arcv1alpha1.EndpointUsageAll {
		err := fmt.Errorf("endpoint '%s' usage '%s' is not compatible with destination usage", dstEndpoint.Name, dstEndpoint.Spec.Usage)
		r.Recorder.Event(order, corev1.EventTypeWarning, "InvalidEndpoint", fmt.Sprintf("Destination endpoint '%s' has incompatible usage '%s'", dstEndpoint.Name, dstEndpoint.Spec.Usage))
		return nil, errLogAndWrap(log, err, "artifact validation failed")
	}

	// Validate against ArtifactType rules
	artifactType := &arcv1alpha1.ArtifactType{}
	if err := r.Get(ctx, namespacedName(order.Namespace, artifact.Type), artifactType); client.IgnoreNotFound(err) != nil {
		r.Recorder.Event(order, corev1.EventTypeWarning, "InvalidArtifactType", fmt.Sprintf("Failed to fetch ArtifactType '%s': %v", artifact.Type, err))
		return nil, errLogAndWrap(log, err, "failed to fetch referenced ArtifactType")
	}
	var (
		artifactTypeGen  int64
		artifactTypeSpec *arcv1alpha1.ArtifactTypeSpec
	)
	if artifactType.Name == "" { // was not found, let's check ClusterArtifactType
		clusterArtifactType := &arcv1alpha1.ClusterArtifactType{}
		if err := r.Get(ctx, namespacedName("", artifact.Type), clusterArtifactType); err != nil {
			return nil, errLogAndWrap(log, err, "failed to fetch ArtifactType or ClusterArtifactType")
		}
		artifactTypeSpec = &clusterArtifactType.Spec
		artifactTypeGen = clusterArtifactType.Generation
		// NOTE: ClusterArtifactTypes can only referes ClusterWorkflowTemplates, so we enforce this here:
		artifactTypeSpec.WorkflowTemplateRef.ClusterScope = true
	} else {
		artifactTypeSpec = &artifactType.Spec
		artifactTypeGen = artifactType.Generation
	}

	if len(artifactTypeSpec.Rules.SrcTypes) > 0 && !slices.Contains(artifactTypeSpec.Rules.SrcTypes, srcEndpoint.Spec.Type) {
		err := fmt.Errorf("source endpoint type '%s' is not allowed by ArtifactType rules", srcEndpoint.Spec.Type)
		r.Recorder.Event(order, corev1.EventTypeWarning, "InvalidArtifactType", fmt.Sprintf("Source endpoint type '%s' is not allowed by ArtifactType '%s' rules", srcEndpoint.Spec.Type, artifact.Type))
		return nil, errLogAndWrap(log, err, "artifact validation failed")
	}
	if len(artifactTypeSpec.Rules.DstTypes) > 0 && !slices.Contains(artifactTypeSpec.Rules.DstTypes, dstEndpoint.Spec.Type) {
		err := fmt.Errorf("destination endpoint type '%s' is not allowed by ArtifactType rules", dstEndpoint.Spec.Type)
		r.Recorder.Event(order, corev1.EventTypeWarning, "InvalidArtifactType", fmt.Sprintf("Destination endpoint type '%s' is not allowed by ArtifactType '%s' rules", dstEndpoint.Spec.Type, artifact.Type))
		return nil, errLogAndWrap(log, err, "artifact validation failed")
	}

	// Next, we need the secret contents
	srcSecret := &corev1.Secret{}
	if srcEndpoint.Spec.SecretRef.Name != "" {
		if err := r.Get(ctx, namespacedName(order.Namespace, srcEndpoint.Spec.SecretRef.Name), srcSecret); err != nil {
			r.Recorder.Event(order, corev1.EventTypeWarning, "InvalidSecret", fmt.Sprintf("Failed to fetch source secret '%s': %v", srcEndpoint.Spec.SecretRef.Name, err))
			return nil, errLogAndWrap(log, err, "failed to fetch secret for source")
		}
	}

	dstSecret := &corev1.Secret{}
	if dstEndpoint.Spec.SecretRef.Name != "" {
		if err := r.Get(ctx, namespacedName(order.Namespace, dstEndpoint.Spec.SecretRef.Name), dstSecret); err != nil {
			r.Recorder.Event(order, corev1.EventTypeWarning, "InvalidSecret", fmt.Sprintf("Failed to fetch destination secret '%s': %v", dstEndpoint.Spec.SecretRef.Name, err))
			return nil, errLogAndWrap(log, err, "failed to fetch secret for destination")
		}
	}

	// Create a hash based on all related data for idempotency and compute the workflow name
	h := sha256.New()
	data := []any{
		order.Namespace,
		artifact.Type, artifact.Spec.Raw, artifactTypeGen,
		srcEndpoint.Name,
		dstEndpoint.Name,
	}
	jsonData, err := json.Marshal(data)
	if err != nil {
		return nil, errLogAndWrap(log, err, "failed to marshal artifact workflow data")
	}
	h.Write(jsonData)
	sha := hex.EncodeToString(h.Sum(nil))[:16]

	// We gave all the information to further process this artifact workflow.
	// Let's store it to compare it to the current status!
	return &desiredAW{
		index:       i,
		objectMeta:  awObjectMeta(order, sha),
		artifact:    artifact,
		typeSpec:    artifactTypeSpec,
		srcEndpoint: srcEndpoint,
		dstEndpoint: dstEndpoint,
		srcSecret:   srcSecret,
		dstSecret:   dstSecret,
		sha:         sha,
	}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *OrderReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&arcv1alpha1.Order{}).
		Owns(&arcv1alpha1.ArtifactWorkflow{}).
		Complete(r)
}
