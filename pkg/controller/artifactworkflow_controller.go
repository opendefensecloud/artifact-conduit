// Copyright 2025 BWI GmbH and Artefact Conduit contributors
// SPDX-License-Identifier: Apache-2.0

package controller

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"slices"

	wfv1alpha1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	"github.com/go-logr/logr"
	"github.com/jastBytes/sprint"
	arcv1alpha1 "go.opendefense.cloud/arc/api/arc/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	artifactWorkflowFinalizer = "arc.opendefense.cloud/artifact-workflow-finalizer"
)

// ArtifactWorkflowReconciler reconciles a ArtifactWorkflow object
type ArtifactWorkflowReconciler struct {
	client.Client
	ClientSet kubernetes.Interface
	Scheme    *runtime.Scheme
	Recorder  record.EventRecorder
}

//+kubebuilder:rbac:groups=arc.opendefense.cloud,resources=clusterartifacttypes,verbs=get;list;watch
//+kubebuilder:rbac:groups=arc.opendefense.cloud,resources=artifactworkflows/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=arc.opendefense.cloud,resources=artifactworkflows/finalizers,verbs=update
//+kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=argoproj.io,resources=workflows,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=core,resources=events,verbs=create;patch
//+kubebuilder:rbac:groups="",resources=pods;pods/log,verbs=get;list

// Reconcile moves the current state of the cluster closer to the desired state
func (r *ArtifactWorkflowReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := ctrl.LoggerFrom(ctx)
	ctrlResult := ctrl.Result{}

	aw := &arcv1alpha1.ArtifactWorkflow{}
	if err := r.Get(ctx, req.NamespacedName, aw); err != nil {
		if apierrors.IsNotFound(err) {
			// Object not found, return.
			return ctrlResult, nil
		}
		return ctrlResult, errLogAndWrap(log, err, "failed to get object")
	}

	// Two different behaviors are implemented depending on whether we have
	// a "one-shot" order or cron order, so let's instantiate the corresponding handler:
	var handler WorkflowHandler
	if aw.Spec.Cron != nil {
		handler = NewCronWorkflowHandler(r, log, aw)
	} else {
		handler = NewSingleWorkflowHandler(r, log, aw)
	}

	// Update last reconcile time
	aw.Status.LastReconcileAt = metav1.Now()

	// Handle deletion
	if !aw.DeletionTimestamp.IsZero() {
		log.V(1).Info("ArtifactWorkflow is being deleted")
		// Cleanup workflow, if exists
		if err := handler.DeleteArgoResources(ctx); err != nil {
			return ctrlResult, errLogAndWrap(log, err, "workflow deletion failed")
		}

		// Remove finalizer
		if slices.Contains(aw.Finalizers, artifactWorkflowFinalizer) {
			log.V(1).Info("Removing finalizer from ArtifactWorkflow")
			aw.Finalizers = slices.DeleteFunc(aw.Finalizers, func(f string) bool {
				return f == artifactWorkflowFinalizer
			})
			if err := r.Update(ctx, aw); err != nil {
				return ctrlResult, errLogAndWrap(log, err, "failed to remove finalizer")
			}
		}
		return ctrlResult, nil
	}

	// Add finalizer if not present and not deleting
	if aw.DeletionTimestamp.IsZero() {
		if !slices.Contains(aw.Finalizers, artifactWorkflowFinalizer) {
			log.V(1).Info("Adding finalizer to ArtifactWorkflow")
			aw.Finalizers = append(aw.Finalizers, artifactWorkflowFinalizer)
			if err := r.Update(ctx, aw); err != nil {
				return ctrlResult, errLogAndWrap(log, err, "failed to add finalizer")
			}
			// Return without requeue; the Update event will trigger reconciliation again
			return ctrlResult, nil
		}
	}

	// Handle force reconcile annotation
	forceAt, err := GetForceAtAnnotationValue(aw)
	if err != nil {
		log.V(1).Error(err, "Invalid force reconcile annotation, ignoring")
	}
	if !forceAt.IsZero() && (aw.Status.LastForceAt.IsZero() || forceAt.After(aw.Status.LastForceAt.Time)) {
		log.V(1).Info("Force reconcile requested")
		r.Recorder.Event(aw, corev1.EventTypeNormal, "ForceReconcile", "Force reconcile requested via annotation")
		// Delete existing workflow, if any
		if err := handler.DeleteArgoResources(ctx); err != nil {
			return ctrlResult, errLogAndWrap(log, err, "failed to delete existing workflow for force reconcile")
		}
		// Reset phase so workflow gets recreated, and update last force time
		aw.Status.Phase = arcv1alpha1.WorkflowUnknown
		aw.Status.LastForceAt = metav1.Now()
		if err := r.Status().Update(ctx, aw); err != nil {
			return ctrlResult, errLogAndWrap(log, err, "failed to update last force time")
		}
		// Return without requeue; the update event will trigger reconciliation again
		return ctrlResult, nil
	}

	if aw.Status.Phase == arcv1alpha1.WorkflowUnknown {
		return ctrlResult, handler.CreateArgoResources(ctx)
	}

	if aw.Status.Phase.InProgress() {
		return ctrlResult, handler.CheckArgoResources(ctx)
	}

	return ctrlResult, nil
}

func (r *ArtifactWorkflowReconciler) setStatusFromWorkflow(ctx context.Context, log logr.Logger, aw *arcv1alpha1.ArtifactWorkflow, wf *wfv1alpha1.Workflow) bool {
	if aw.Status.Phase == arcv1alpha1.WorkflowPhase(wf.Status.Phase) {
		return false // nothing updated
	}
	aw.Status.Phase = arcv1alpha1.WorkflowPhase(wf.Status.Phase)

	switch aw.Status.Phase {
	case arcv1alpha1.WorkflowSucceeded:
		aw.Status.CompletionTime = metav1.Now()
	case arcv1alpha1.WorkflowError, arcv1alpha1.WorkflowFailed:
		// If workflow has errored or failed, fetch logs and update status message
		switch aw.Status.Phase {
		case arcv1alpha1.WorkflowFailed:
			r.generateWorkflowStatusMessage(ctx, wf, log, aw)
		case arcv1alpha1.WorkflowError:
			// TODO: Properly show why the workflow errored
			aw.Status.Message = wf.Status.Message
		}
	}
	return true
}

func (r *ArtifactWorkflowReconciler) generateWorkflowStatusMessage(ctx context.Context, wf *wfv1alpha1.Workflow, log logr.Logger, aw *arcv1alpha1.ArtifactWorkflow) {
	failedNodes := []struct {
		Name    string
		Pod     string
		Message string
	}{}
	for _, node := range wf.Status.Nodes {
		if node.Phase == wfv1alpha1.NodeFailed && node.Type == wfv1alpha1.NodeTypePod {
			nr := struct {
				Name    string
				Pod     string
				Message string
			}{
				Name:    node.DisplayName,
				Pod:     generatePodNameFromNodeStatus(node),
				Message: node.Message,
			}
			failedNodes = append(failedNodes, nr)
		}
	}

	for _, nr := range failedNodes {
		logs, err := r.fetchPodLogs(ctx, aw.Namespace, nr.Pod)
		if err != nil {
			log.V(1).Info("failed to fetch pod logs", "pod", nr.Pod, "error", err)
			continue
		}
		aw.Status.Message += fmt.Sprintf("Step '%s' failed:\n%s\nLogs:\n%s\n\n", nr.Name, nr.Message, logs)
	}
}

func (r *ArtifactWorkflowReconciler) fetchPodLogs(ctx context.Context, namespace, podName string) (string, error) {
	podLogOptions := corev1.PodLogOptions{
		Container: "main", // Assuming the main container
		Follow:    false,
		TailLines: sprint.ToPointer(int64(30)), // Fetch last 30 lines
	}
	req := r.ClientSet.CoreV1().Pods(namespace).GetLogs(podName, &podLogOptions)
	podLogs, err := req.Stream(ctx)
	if err != nil {
		return "", err
	}
	defer sprint.PanicOnErrorFunc(podLogs.Close) // Close the stream when done

	buf := new(bytes.Buffer)
	_, err = io.Copy(buf, podLogs)
	if err != nil {
		return "", err
	}
	return buf.String(), nil
}

func (r *ArtifactWorkflowReconciler) retrieveSecrets(ctx context.Context, aw *arcv1alpha1.ArtifactWorkflow) (*corev1.Secret, *corev1.Secret, error) {
	srcSecret := corev1.Secret{}
	if aw.Spec.SrcSecretRef.Name != "" {
		if err := r.Get(ctx, namespacedName(aw.Namespace, aw.Spec.SrcSecretRef.Name), &srcSecret); err != nil {
			r.Recorder.Event(aw, corev1.EventTypeWarning, "InvalidSecret", fmt.Sprintf("Failed to fetch source secret '%s': %v", aw.Spec.SrcSecretRef.Name, err))
			return nil, nil, fmt.Errorf("failed to fetch secret for source: %w", err)
		}
	}

	dstSecret := corev1.Secret{}
	if aw.Spec.DstSecretRef.Name != "" {
		if err := r.Get(ctx, namespacedName(aw.Namespace, aw.Spec.DstSecretRef.Name), &dstSecret); err != nil {
			r.Recorder.Event(aw, corev1.EventTypeWarning, "InvalidSecret", fmt.Sprintf("Failed to fetch destination secret '%s': %v", aw.Spec.DstSecretRef.Name, err))
			return nil, nil, fmt.Errorf("failed to fetch secret for destination: %w", err)
		}
	}

	return &srcSecret, &dstSecret, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *ArtifactWorkflowReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&arcv1alpha1.ArtifactWorkflow{}).
		Owns(&wfv1alpha1.Workflow{}).
		Owns(&wfv1alpha1.CronWorkflow{}).
		Complete(r)
}
