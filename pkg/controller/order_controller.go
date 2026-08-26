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
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/events"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	arcv1alpha1 "go.opendefense.cloud/arc/api/arc/v1alpha1"
	"go.opendefense.cloud/arc/pkg/metrics"
)

const (
	orderFinalizer = "arc.opendefense.cloud/order-finalizer"
)

// OrderReconciler reconciles a Order object
type OrderReconciler struct {
	client.Client
	Scheme   *runtime.Scheme
	Recorder events.EventRecorder
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
	cron        *arcv1alpha1.Cron
}

//+kubebuilder:rbac:groups=arc.opendefense.cloud,resources=endpoints,verbs=get;list;watch
//+kubebuilder:rbac:groups=arc.opendefense.cloud,resources=artifacttypes,verbs=get;list;watch
//+kubebuilder:rbac:groups=arc.opendefense.cloud,resources=clusterartifacttypes,verbs=get;list;watch
//+kubebuilder:rbac:groups=arc.opendefense.cloud,resources=artifactworkflows,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=arc.opendefense.cloud,resources=orders,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=arc.opendefense.cloud,resources=orders/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=arc.opendefense.cloud,resources=orders/finalizers,verbs=update
//+kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=events.k8s.io,resources=events,verbs=create;patch

// Reconcile moves the current state of the cluster closer to the desired state
func (r *OrderReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := ctrl.LoggerFrom(ctx)
	ctrlResult := ctrl.Result{}

	// Fetch the Order instance
	order := &arcv1alpha1.Order{}
	if err := r.Get(ctx, req.NamespacedName, order); err != nil {
		if apierrors.IsNotFound(err) {
			// Object not found, return. Created objects are automatically garbage collected.
			return ctrlResult, nil
		}

		return ctrlResult, errLogAndWrap(log, err, "failed to get object")
	}

	// Update last reconcile time
	order.Status.LastReconcileAt = metav1.Now()

	// Handle deletion: cleanup artifact workflows, then remove finalizer
	if !order.DeletionTimestamp.IsZero() {
		log.V(1).Info("Order is being deleted")
		r.Recorder.Eventf(order, nil, corev1.EventTypeWarning, ReasonDeleting, "Delete", "Order is being deleted, cleaning up artifact workflows")

		// Cleanup all artifact workflows
		if len(order.Status.ArtifactWorkflows) > 0 {
			for sha := range order.Status.ArtifactWorkflows {
				// Remove ArtifactWorkflow. The artifact type label is not needed here since
				// Delete matches targets by namespace and name, not labels.
				aw := &arcv1alpha1.ArtifactWorkflow{
					ObjectMeta: awObjectMeta(order, sha, ""),
				}
				_ = r.Delete(ctx, aw) // Ignore errors
				delete(order.Status.ArtifactWorkflows, sha)
			}
			if err := r.Status().Update(ctx, order); err != nil {
				return ctrlResult, errLogAndWrap(log, err, "failed to update order status")
			}
			log.V(1).Info("Order artifact workflows cleaned up")

			// Requeue until all artifact workflows are gone
			return ctrlResult, nil
		}
		// All artifact workflows are gone, remove finalizer
		if slices.Contains(order.Finalizers, orderFinalizer) {
			log.V(1).Info("No artifact workflows, removing finalizer from Order")
			order.Finalizers = slices.DeleteFunc(order.Finalizers, func(f string) bool {
				return f == orderFinalizer
			})
			if err := r.Update(ctx, order); err != nil {
				return ctrlResult, errLogAndWrap(log, err, "failed to remove finalizer")
			}
		}

		return ctrlResult, nil
	}

	// Add finalizer if not present and not deleting
	if order.DeletionTimestamp.IsZero() {
		if !slices.Contains(order.Finalizers, orderFinalizer) {
			log.V(1).Info("Adding finalizer to Order")
			order.Finalizers = append(order.Finalizers, orderFinalizer)
			if err := r.Update(ctx, order); err != nil {
				return ctrlResult, errLogAndWrap(log, err, "failed to add finalizer")
			}
			// Return without requeue; the Update event will trigger reconciliation again
			return ctrlResult, nil
		}
	}

	// Handle force reconcile annotation
	forceAt, err := GetForceAtAnnotationValue(order)
	if err != nil {
		log.V(1).Error(err, "Invalid force reconcile annotation, ignoring")
	}
	if !forceAt.IsZero() && (order.Status.LastForceAt.IsZero() || forceAt.After(order.Status.LastForceAt.Time)) {
		log.V(1).Info("Force reconcile requested")
		r.Recorder.Eventf(order, nil, corev1.EventTypeNormal, "ForceReconcile", "Reconcile", "Force reconcile requested via annotation")
		// Delete existing artifact workflows to force re-creation
		for sha := range order.Status.ArtifactWorkflows {
			// Remove Secret and ArtifactWorkflow
			aw := &arcv1alpha1.ArtifactWorkflow{
				ObjectMeta: awObjectMeta(order, sha, ""),
			}
			_ = r.Delete(ctx, aw) // Ignore errors
			delete(order.Status.ArtifactWorkflows, sha)
			r.Recorder.Eventf(order, aw, corev1.EventTypeNormal, "ForceReconcile", "Reconcile", "Deleted artifact workflow '%s' with sha %s", aw.Name, sha)
		}
		// Update last force time
		order.Status.LastForceAt = metav1.Now()
		if err := r.Status().Update(ctx, order); err != nil {
			return ctrlResult, errLogAndWrap(log, err, "failed to update last force time")
		}
		// Return without requeue; the update event will trigger reconciliation again
		return ctrlResult, nil
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
			r.Recorder.Eventf(order, nil, corev1.EventTypeWarning, ReasonComputationFailed, "Compute", "Failed to compute desired artifact workflow for artifact index %d: %v", i, err)
			metrics.RecordReconcileError(ControllerOrder, ReasonComputationFailed)
			order.Status.Message = fmt.Sprintf("Failed to compute desired artifact workflow for artifact index %d: %v", i, err)
			if err := r.Status().Update(ctx, order); err != nil {
				return ctrlResult, errLogAndWrap(log, err, "failed to update status")
			}

			return ctrlResult, errLogAndWrap(log, err, "failed to compute desired artifact workflow")
		}
		desiredAWs[daw.sha] = *daw
	}
	order.Status.Message = "" // Clear any previous error message

	// List missing artifact workflows
	var createAWs []string
	for sha := range desiredAWs {
		if _, exists := order.Status.ArtifactWorkflows[sha]; exists {
			continue
		}

		createAWs = append(createAWs, sha)
	}

	// Find obsolete artifact workflows
	var deleteAWs []string
	for sha := range order.Status.ArtifactWorkflows {
		if _, exists := desiredAWs[sha]; exists {
			continue
		}

		deleteAWs = append(deleteAWs, sha)
	}

	// Find finished artifact workflows to clean up
	var finishedAWs []string
	for sha := range order.Status.ArtifactWorkflows {
		awStatus := order.Status.ArtifactWorkflows[sha]

		// Do not clean up ArtifactWorkflows with cron specified
		if daw, ok := desiredAWs[sha]; ok && daw.cron != nil {
			continue
		}

		// Do not clean up workflows that are still running or pending
		switch awStatus.Phase {
		case arcv1alpha1.WorkflowSucceeded:
		case arcv1alpha1.WorkflowFailed:
		case arcv1alpha1.WorkflowError:
		default:
			continue
		}

		// Get ArtifactWorkflow object and check TTLs.
		artifactWorkflow := &arcv1alpha1.ArtifactWorkflow{}
		if err := r.Get(ctx, types.NamespacedName{Namespace: order.Namespace, Name: awName(order, sha)}, artifactWorkflow); err != nil && !apierrors.IsNotFound(err) {
			r.Recorder.Eventf(order, nil, corev1.EventTypeWarning, ReasonInvalid, "Fetch", "Failed to fetch ArtifactWorkflow: %v", sha)
			metrics.RecordReconcileError(ControllerOrder, ReasonInvalid)

			return ctrlResult, errLogAndWrap(log, err, "")
		}
		if artifactWorkflow.Name != "" {
			// Cleanup finished workflows if TTLAfterFinished is set.
			if awStatus.Phase == arcv1alpha1.WorkflowSucceeded {
				// If TTL is set, check if it has expired
				if artifactWorkflow.Spec.TTLAfterFinished != nil {
					if artifactWorkflow.Spec.TTLAfterFinished.Seconds() == 0 {
						// If TTL is zero keep the workflow.
						continue
					}
					if time.Since(awStatus.CompletionTime.Time) < artifactWorkflow.Spec.TTLAfterFinished.Duration {
						// If TTL is set but not expired keep the workflow.
						// Requeue when the next TTL expires
						ctrlResult.RequeueAfter = artifactWorkflow.Spec.TTLAfterFinished.Duration - time.Since(awStatus.CompletionTime.Time)
						continue
					}
				}
			}

			// Cleanup failed workflows if TTLAfterFailed is set.
			if awStatus.Phase == arcv1alpha1.WorkflowFailed || awStatus.Phase == arcv1alpha1.WorkflowError {
				// If TTL is set, check if it has expired
				if artifactWorkflow.Spec.TTLAfterFailed != nil {
					if artifactWorkflow.Spec.TTLAfterFailed.Seconds() == 0 {
						// If TTL is zero keep the workflow.
						continue
					}
					if time.Since(awStatus.FailureTime.Time) < artifactWorkflow.Spec.TTLAfterFailed.Duration {
						// If TTL is set but not expired keep the workflow.
						ctrlResult.RequeueAfter = artifactWorkflow.Spec.TTLAfterFailed.Duration - time.Since(awStatus.FailureTime.Time)
						continue
					}
				} else {
					// If no TTL is set keep the workflow.
					continue
				}
			}
		}

		// Cleanup finished or not existing workflows
		finishedAWs = append(finishedAWs, sha)
	}

	// Create missing artifact workflows
	for _, sha := range createAWs {
		daw := desiredAWs[sha]
		aw, err := r.hydrateArtifactWorkflow(&daw)
		if err != nil {
			r.Recorder.Eventf(order, nil, corev1.EventTypeWarning, ReasonHydrationFailed, "Hydrate", "Failed to hydrate artifact workflow for artifact index %d: %v", daw.index, err)
			metrics.RecordReconcileError(ControllerOrder, ReasonHydrationFailed)

			return ctrlResult, errLogAndWrap(log, err, "failed to hydrate artifact workflow")
		}

		// Set owner references
		if err := controllerutil.SetControllerReference(order, aw, r.Scheme); err != nil {
			r.Recorder.Eventf(order, aw, corev1.EventTypeWarning, ReasonHydrationFailed, "Hydrate", "Failed to set controller reference for artifact workflow: %v", err)
			metrics.RecordReconcileError(ControllerOrder, ReasonHydrationFailed)

			return ctrlResult, errLogAndWrap(log, err, "failed to set controller reference")
		}

		// Create artifact workflow
		if err := r.Create(ctx, aw); err != nil {
			if apierrors.IsAlreadyExists(err) {
				// Already created by a previous reconcile — that's fine
				continue
			}
			r.Recorder.Eventf(order, nil, corev1.EventTypeWarning, ReasonCreationFailed, "Create", "Failed to create artifact workflow for artifact index %d: %v", daw.index, err)
			metrics.RecordReconcileError(ControllerOrder, ReasonCreationFailed)

			return ctrlResult, errLogAndWrap(log, err, "failed to create artifact workflow")
		} else {
			r.Recorder.Eventf(order, aw, corev1.EventTypeNormal, "Created", "Create", "Created artifact workflow '%s' for artifact index %d", aw.Name, daw.index)
			log.V(1).Info("Created artifact workflow", "artifactWorkflow", aw.Name)
		}

		// Update status
		order.Status.ArtifactWorkflows[sha] = arcv1alpha1.OrderArtifactWorkflowStatus{
			ArtifactIndex: daw.index,
			WorkflowStatus: arcv1alpha1.WorkflowStatus{
				Phase: arcv1alpha1.WorkflowUnknown,
			},
		}
	}

	// Delete obsolete artifact workflows
	for _, sha := range deleteAWs {
		// Does not exist anymore, let's clean up!
		aw := &arcv1alpha1.ArtifactWorkflow{
			ObjectMeta: awObjectMeta(order, sha, ""),
		}
		if err := r.Delete(ctx, aw); client.IgnoreNotFound(err) != nil {
			r.Recorder.Eventf(order, aw, corev1.EventTypeWarning, ReasonDeletionFailed, "Delete", "Failed to delete obsolete artifact workflow '%s': %v", sha, err)
			metrics.RecordReconcileError(ControllerOrder, ReasonDeletionFailed)

			return ctrlResult, errLogAndWrap(log, err, "failed to delete artifact workflow")
		}

		// Update status
		delete(order.Status.ArtifactWorkflows, sha)
		log.V(1).Info("Deleted obsolete artifact workflow", "artifactWorkflow", sha)
		r.Recorder.Eventf(order, aw, corev1.EventTypeNormal, "Deleted", "Delete", "Deleted obsolete artifact workflow '%s'", sha)
	}

	// Delete finished artifact workflows
	for _, sha := range finishedAWs {
		// Finished, let's clean up!
		aw := &arcv1alpha1.ArtifactWorkflow{
			ObjectMeta: awObjectMeta(order, sha, ""),
		}
		if err := r.Delete(ctx, aw); client.IgnoreNotFound(err) != nil {
			r.Recorder.Eventf(order, aw, corev1.EventTypeWarning, ReasonDeletionFailed, "Delete", "Failed to delete finished artifact workflow '%s': %v", sha, err)
			metrics.RecordReconcileError(ControllerOrder, ReasonDeletionFailed)

			return ctrlResult, errLogAndWrap(log, err, "failed to delete artifact workflow")
		}

		log.V(1).Info("Deleted finished artifact workflow", "artifactWorkflow", sha)
		r.Recorder.Eventf(order, aw, corev1.EventTypeNormal, "Deleted", "Delete", "Deleted finished artifact workflow '%s'", sha)
	}

	anyStatusChanged := false
	for sha, daw := range desiredAWs {
		if slices.Contains(createAWs, sha) {
			// If it was just created we skip the update
			continue
		}
		if daw.cron == nil && order.Status.ArtifactWorkflows[sha].Phase.Completed() {
			// We do not need to check for updates if the workflow is completed and is NOT cron
			continue
		}
		aw := arcv1alpha1.ArtifactWorkflow{}
		if err := r.Get(ctx, namespacedName(daw.objectMeta.Namespace, daw.objectMeta.Name), &aw); err != nil {
			delete(order.Status.ArtifactWorkflows, sha)
			log.V(1).Info("Artifact workflow not found, deleting from status.", "artifactWorkflow", sha)
			if err := r.Status().Update(ctx, order); err != nil {
				return ctrlResult, errLogAndWrap(log, err, "failed to update status")
			}

			return ctrlResult, errLogAndWrap(log, err, "failed to get artifact workflow")
		}
		orderAWStatus := order.Status.ArtifactWorkflows[sha]

		phaseChanged := orderAWStatus.Phase != aw.Status.Phase
		succeededChanged := orderAWStatus.Succeeded != aw.Status.Succeeded
		failedChanged := orderAWStatus.Failed != aw.Status.Failed
		lastScheduledChanged := !orderAWStatus.LastScheduled.Equal(aw.Status.LastScheduled)

		if phaseChanged || succeededChanged || failedChanged || lastScheduledChanged {
			orderAWStatus.WorkflowStatus = aw.Status.WorkflowStatus
			order.Status.ArtifactWorkflows[sha] = orderAWStatus
			anyStatusChanged = true
		}
	}

	// Update status
	if len(createAWs) > 0 || len(deleteAWs) > 0 || anyStatusChanged {
		log.V(1).Info("Updating order status")
		// Make sure ArtifactIndex is up to date
		for sha, daw := range desiredAWs {
			aws := order.Status.ArtifactWorkflows[sha]
			aws.ArtifactIndex = daw.index
			order.Status.ArtifactWorkflows[sha] = aws
		}
		if err := r.Status().Update(ctx, order); err != nil {
			return ctrlResult, errLogAndWrap(log, err, "failed to update status")
		}
	}

	return ctrlResult, nil
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
			WorkflowTemplateRef:         daw.typeSpec.WorkflowTemplateRef,
			Parameters:                  params,
			SrcSecretRef:                daw.srcEndpoint.Spec.SecretRef,
			DstSecretRef:                daw.dstEndpoint.Spec.SecretRef,
			Cron:                        daw.cron,
			ArtifactWorkflowTTLSettings: daw.typeSpec.ArtifactWorkflowTTLSettings,
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
		r.Recorder.Eventf(order, nil, corev1.EventTypeWarning, ReasonInvalidEndpoint, "FetchEndpoint", "Failed to fetch source endpoint '%s': %v", srcRefName, err)
		metrics.RecordReconcileError(ControllerOrder, ReasonInvalidEndpoint)

		return nil, errLogAndWrap(log, err, "failed to fetch endpoint for source")
	}
	dstEndpoint := &arcv1alpha1.Endpoint{}
	if err := r.Get(ctx, namespacedName(order.Namespace, dstRefName), dstEndpoint); err != nil {
		r.Recorder.Eventf(order, nil, corev1.EventTypeWarning, ReasonInvalidEndpoint, "FetchEndpoint", "Failed to fetch destination endpoint '%s': %v", dstRefName, err)
		metrics.RecordReconcileError(ControllerOrder, ReasonInvalidEndpoint)

		return nil, errLogAndWrap(log, err, "failed to fetch endpoint for destination")
	}

	// Validate that the endpoint usage is correct
	if srcEndpoint.Spec.Usage != arcv1alpha1.EndpointUsagePullOnly && srcEndpoint.Spec.Usage != arcv1alpha1.EndpointUsageAll {
		err := fmt.Errorf("endpoint '%s' usage '%s' is not compatible with source usage", srcEndpoint.Name, srcEndpoint.Spec.Usage)
		r.Recorder.Eventf(order, srcEndpoint, corev1.EventTypeWarning, ReasonInvalidEndpoint, "ValidateEndpoint", "Source endpoint '%s' has incompatible usage '%s'", srcEndpoint.Name, srcEndpoint.Spec.Usage)
		metrics.RecordReconcileError(ControllerOrder, ReasonInvalidEndpoint)

		return nil, errLogAndWrap(log, err, "artifact validation failed")
	}
	if dstEndpoint.Spec.Usage != arcv1alpha1.EndpointUsagePushOnly && dstEndpoint.Spec.Usage != arcv1alpha1.EndpointUsageAll {
		err := fmt.Errorf("endpoint '%s' usage '%s' is not compatible with destination usage", dstEndpoint.Name, dstEndpoint.Spec.Usage)
		r.Recorder.Eventf(order, dstEndpoint, corev1.EventTypeWarning, ReasonInvalidEndpoint, "ValidateEndpoint", "Destination endpoint '%s' has incompatible usage '%s'", dstEndpoint.Name, dstEndpoint.Spec.Usage)
		metrics.RecordReconcileError(ControllerOrder, ReasonInvalidEndpoint)

		return nil, errLogAndWrap(log, err, "artifact validation failed")
	}

	// Validate against ArtifactType rules
	artifactType := &arcv1alpha1.ArtifactType{}
	if err := r.Get(ctx, namespacedName(order.Namespace, artifact.Type), artifactType); client.IgnoreNotFound(err) != nil {
		r.Recorder.Eventf(order, nil, corev1.EventTypeWarning, ReasonInvalidArtifactType, "FetchArtifactType", "Failed to fetch ArtifactType '%s': %v", artifact.Type, err)
		metrics.RecordReconcileError(ControllerOrder, ReasonInvalidArtifactType)

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
		// NOTE: ClusterArtifactTypes can only reference ClusterWorkflowTemplates, so we enforce this here:
		artifactTypeSpec.WorkflowTemplateRef.ClusterScope = true
	} else {
		artifactTypeSpec = &artifactType.Spec
		artifactTypeGen = artifactType.Generation
	}

	if len(artifactTypeSpec.Rules.SrcTypes) > 0 && !slices.Contains(artifactTypeSpec.Rules.SrcTypes, srcEndpoint.Spec.Type) {
		err := fmt.Errorf("source endpoint type '%s' is not allowed by ArtifactType rules", srcEndpoint.Spec.Type)
		r.Recorder.Eventf(order, artifactType, corev1.EventTypeWarning, ReasonInvalidArtifactType, "ValidateArtifactType", "Source endpoint type '%s' is not allowed by ArtifactType '%s' rules", srcEndpoint.Spec.Type, artifact.Type)
		metrics.RecordReconcileError(ControllerOrder, ReasonInvalidArtifactType)

		return nil, errLogAndWrap(log, err, "artifact validation failed")
	}
	if len(artifactTypeSpec.Rules.DstTypes) > 0 && !slices.Contains(artifactTypeSpec.Rules.DstTypes, dstEndpoint.Spec.Type) {
		err := fmt.Errorf("destination endpoint type '%s' is not allowed by ArtifactType rules", dstEndpoint.Spec.Type)
		r.Recorder.Eventf(order, artifactType, corev1.EventTypeWarning, ReasonInvalidArtifactType, "ValidateArtifactType", "Destination endpoint type '%s' is not allowed by ArtifactType '%s' rules", dstEndpoint.Spec.Type, artifact.Type)
		metrics.RecordReconcileError(ControllerOrder, ReasonInvalidArtifactType)

		return nil, errLogAndWrap(log, err, "artifact validation failed")
	}

	// Next, we need the secret contents
	srcSecret := &corev1.Secret{}
	if srcEndpoint.Spec.SecretRef.Name != "" {
		if err := r.Get(ctx, namespacedName(order.Namespace, srcEndpoint.Spec.SecretRef.Name), srcSecret); err != nil {
			r.Recorder.Eventf(order, nil, corev1.EventTypeWarning, ReasonInvalidSecret, "FetchSecret", "Failed to fetch source secret '%s': %v", srcEndpoint.Spec.SecretRef.Name, err)
			metrics.RecordReconcileError(ControllerOrder, ReasonInvalidSecret)

			return nil, errLogAndWrap(log, err, "failed to fetch secret for source")
		}
	}

	dstSecret := &corev1.Secret{}
	if dstEndpoint.Spec.SecretRef.Name != "" {
		if err := r.Get(ctx, namespacedName(order.Namespace, dstEndpoint.Spec.SecretRef.Name), dstSecret); err != nil {
			r.Recorder.Eventf(order, nil, corev1.EventTypeWarning, ReasonInvalidSecret, "FetchSecret", "Failed to fetch destination secret '%s': %v", dstEndpoint.Spec.SecretRef.Name, err)
			metrics.RecordReconcileError(ControllerOrder, ReasonInvalidSecret)

			return nil, errLogAndWrap(log, err, "failed to fetch secret for destination")
		}
	}

	// Cron schedule if any
	cron := artifact.Cron
	if cron == nil {
		cron = order.Spec.Defaults.Cron
	}

	// Create a hash based on all related data for idempotency and compute the workflow name
	h := sha256.New()
	data := []any{
		order.Namespace,
		artifact.Type,
		artifact.Spec.Raw,
		artifactTypeGen,
		srcEndpoint.Name,
		dstEndpoint.Name,
		order.Status.LastForceAt,
		cron,
	}

	if err := json.NewEncoder(h).Encode(data); err != nil {
		return nil, errLogAndWrap(log, err, "failed to marshal artifact workflow data")
	}

	sha := hex.EncodeToString(h.Sum(nil))[:16]

	// We gave all the information to further process this artifact workflow.
	// Let's store it to compare it to the current status!
	return &desiredAW{
		index:       i,
		objectMeta:  awObjectMeta(order, sha, artifact.Type),
		artifact:    artifact,
		typeSpec:    artifactTypeSpec,
		srcEndpoint: srcEndpoint,
		dstEndpoint: dstEndpoint,
		srcSecret:   srcSecret,
		dstSecret:   dstSecret,
		sha:         sha,
		cron:        cron,
	}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *OrderReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&arcv1alpha1.Order{}).
		Owns(&arcv1alpha1.ArtifactWorkflow{}).
		Complete(r)
}
