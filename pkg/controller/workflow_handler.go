// Copyright 2025 BWI GmbH and Artifact Conduit contributors
// SPDX-License-Identifier: Apache-2.0

package controller

import (
	"context"
	"fmt"

	wfv1alpha1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	arcv1alpha1 "go.opendefense.cloud/arc/api/arc/v1alpha1"
	"go.opendefense.cloud/arc/pkg/metrics"
)

type WorkflowHandler interface {
	DeleteArgoResources(ctx context.Context) error
	CreateArgoResources(ctx context.Context) error
	CheckArgoResources(ctx context.Context) error
}

var _ WorkflowHandler = &SingleWorkflowHandler{}

type SingleWorkflowHandler struct {
	*ArtifactWorkflowReconciler
	log logr.Logger
	aw  *arcv1alpha1.ArtifactWorkflow
}

func NewSingleWorkflowHandler(r *ArtifactWorkflowReconciler, log logr.Logger, aw *arcv1alpha1.ArtifactWorkflow) *SingleWorkflowHandler {
	return &SingleWorkflowHandler{r, log, aw}
}

func (h *SingleWorkflowHandler) DeleteArgoResources(ctx context.Context) error {
	wf := wfv1alpha1.Workflow{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: h.aw.Namespace,
			Name:      h.aw.Name,
		},
	}
	if err := h.Delete(ctx, &wf); client.IgnoreNotFound(err) != nil {
		h.Recorder.Eventf(h.aw, nil, corev1.EventTypeWarning, ReasonDeletionFailed, "Delete", fmt.Sprintf("Failed to delete associated workflow '%s': %v", h.aw.Name, err))
		metrics.RecordReconcileError(ControllerArtifactWorkflow, ReasonDeletionFailed)

		return errLogAndWrap(h.log, err, "workflow deletion failed")
	}
	h.Recorder.Eventf(h.aw, nil, corev1.EventTypeNormal, "Deleted", "Delete", fmt.Sprintf("Deleted workflow '%s'", h.aw.Name))

	return nil
}

func (h *SingleWorkflowHandler) CreateArgoResources(ctx context.Context) error {
	srcSecret, dstSecret, err := h.retrieveSecrets(ctx, h.aw)
	if err != nil {
		return errLogAndWrap(h.log, err, "failed to fetch secrets for artifact workflow")
	}

	wf := hydrateArgoWorkflow(h.aw, srcSecret, dstSecret)

	if err := controllerutil.SetControllerReference(h.aw, wf, h.Scheme); err != nil {
		return errLogAndWrap(h.log, err, "failed to set controller reference")
	}

	if err := h.Create(ctx, wf); client.IgnoreAlreadyExists(err) != nil {
		h.Recorder.Eventf(h.aw, nil, corev1.EventTypeWarning, ReasonCreationFailed, "Create", fmt.Sprintf("Failed to create workflow '%s': %v", wf.GetName(), err))
		metrics.RecordReconcileError(ControllerArtifactWorkflow, ReasonCreationFailed)

		return errLogAndWrap(h.log, err, "failed to create argo workflow")
	}
	h.Recorder.Eventf(h.aw, nil, corev1.EventTypeNormal, "Created", "Create", fmt.Sprintf("Created workflow '%s'", wf.GetName()))

	h.aw.Status.Phase = arcv1alpha1.WorkflowPending
	if err := h.Status().Update(ctx, h.aw); err != nil {
		return errLogAndWrap(h.log, err, "failed to update status")
	}

	return nil
}

func (h *SingleWorkflowHandler) CheckArgoResources(ctx context.Context) error {
	wf := wfv1alpha1.Workflow{}
	if err := h.Get(ctx, namespacedName(h.aw.Namespace, h.aw.Name), &wf); err != nil {
		return errLogAndWrap(h.log, err, "failed to get workflow")
	}

	updated, done := h.setStatusFromWorkflow(ctx, h.log, h.aw, &wf)

	if wf.Status.Phase == wfv1alpha1.WorkflowSucceeded && h.aw.Status.Succeeded != 1 {
		h.aw.Status.Succeeded = 1
		updated = true
	}

	failed := wf.Status.Phase == wfv1alpha1.WorkflowError ||
		wf.Status.Phase == wfv1alpha1.WorkflowFailed

	if failed && h.aw.Status.Failed != 1 {
		h.aw.Status.Failed = 1
		updated = true
	}

	if !updated {
		return nil
	}

	if err := h.Status().Update(ctx, h.aw); err != nil {
		return errLogAndWrap(h.log, err, "failed to update status")
	}

	if done != nil {
		done.record()
	}

	return nil
}

var _ WorkflowHandler = &CronWorkflowHandler{}

type CronWorkflowHandler struct {
	*ArtifactWorkflowReconciler
	log logr.Logger
	aw  *arcv1alpha1.ArtifactWorkflow
}

func NewCronWorkflowHandler(r *ArtifactWorkflowReconciler, log logr.Logger, aw *arcv1alpha1.ArtifactWorkflow) *CronWorkflowHandler {
	return &CronWorkflowHandler{r, log, aw}
}

func (h *CronWorkflowHandler) DeleteArgoResources(ctx context.Context) error {
	cwf := wfv1alpha1.CronWorkflow{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: h.aw.Namespace,
			Name:      h.aw.Name,
		},
	}
	if err := h.Delete(ctx, &cwf); client.IgnoreNotFound(err) != nil {
		h.Recorder.Eventf(h.aw, nil, corev1.EventTypeWarning, ReasonDeletionFailed, "Delete", fmt.Sprintf("Failed to delete associated cron workflow '%s': %v", h.aw.Name, err))
		metrics.RecordReconcileError(ControllerArtifactWorkflow, ReasonDeletionFailed)

		return errLogAndWrap(h.log, err, "cron workflow deletion failed")
	}
	h.Recorder.Eventf(h.aw, nil, corev1.EventTypeNormal, "Deleted", "Delete", fmt.Sprintf("Deleted cron workflow '%s'", h.aw.Name))

	return nil
}

func (h *CronWorkflowHandler) CreateArgoResources(ctx context.Context) error {
	srcSecret, dstSecret, err := h.retrieveSecrets(ctx, h.aw)
	if err != nil {
		return errLogAndWrap(h.log, err, "failed to fetch secrets for artifact workflow")
	}

	cwf := hydrateArgoCronWorkflow(h.aw, srcSecret, dstSecret)

	if err := controllerutil.SetControllerReference(h.aw, cwf, h.Scheme); err != nil {
		return errLogAndWrap(h.log, err, "failed to set controller reference")
	}

	if err := h.Create(ctx, cwf); err != nil {
		if client.IgnoreAlreadyExists(err) != nil {
			h.Recorder.Eventf(h.aw, nil, corev1.EventTypeWarning, ReasonCreationFailed, "Create", fmt.Sprintf("Failed to create cron workflow '%s': %v", cwf.GetName(), err))
			metrics.RecordReconcileError(ControllerArtifactWorkflow, ReasonCreationFailed)

			return errLogAndWrap(h.log, err, "failed to create argo cron workflow")
		}
	} else {
		h.Recorder.Eventf(h.aw, nil, corev1.EventTypeNormal, "Created", "Create", fmt.Sprintf("Created cron workflow '%s'", cwf.GetName()))
	}

	h.aw.Status.Phase = arcv1alpha1.WorkflowPending
	if err := h.Status().Update(ctx, h.aw); err != nil {
		return errLogAndWrap(h.log, err, "failed to update status")
	}

	return nil
}

func (h *CronWorkflowHandler) CheckArgoResources(ctx context.Context) error {
	cwf := wfv1alpha1.CronWorkflow{}
	if err := h.Get(ctx, namespacedName(h.aw.Namespace, h.aw.Name), &cwf); err != nil {
		return errLogAndWrap(h.log, err, "failed to get cron workflow")
	}

	updated := false

	if !h.aw.Status.LastScheduled.Equal(cwf.Status.LastScheduledTime) {
		h.aw.Status.LastScheduled = cwf.Status.LastScheduledTime
		updated = true
	}
	if h.aw.Status.Failed != cwf.Status.Failed {
		h.aw.Status.Failed = cwf.Status.Failed
		updated = true
	}
	if h.aw.Status.Succeeded != cwf.Status.Succeeded {
		h.aw.Status.Succeeded = cwf.Status.Succeeded
		updated = true
	}

	// If the active workflow is not the same as the current one, update the reference
	if len(cwf.Status.Active) > 0 {
		// Should only contain a single element at most (expected to be in the same namespace!)
		ref := cwf.Status.Active[len(cwf.Status.Active)-1]

		if h.aw.Status.ActiveWorkflowRef.Name != ref.Name {
			h.log.V(1).Info("Updating reference for cron workflow", "cronWorkflow", cwf.Name, "activeWorkflow", ref.Name)

			// Get the active workflow
			wf := wfv1alpha1.Workflow{}
			if err := h.Get(ctx, namespacedName(h.aw.Namespace, ref.Name), &wf); err != nil {
				return errLogAndWrap(h.log, err, "failed to fetch active workflow")
			}

			h.aw.Status.ActiveWorkflowRef = corev1.LocalObjectReference{
				Name: wf.Name,
			}
			h.aw.Status.Message = ""
			h.aw.Status.Phase = arcv1alpha1.WorkflowActive

			changed, _ := h.setStatusFromWorkflow(ctx, h.log, h.aw, &wf)
			updated = changed || updated
		}
	}

	// If there is an active workflow, check its status
	if h.aw.Status.ActiveWorkflowRef.Name != "" {
		wf := wfv1alpha1.Workflow{}
		if err := h.Get(ctx, namespacedName(h.aw.Namespace, h.aw.Status.ActiveWorkflowRef.Name), &wf); err != nil {
			return errLogAndWrap(h.log, err, "failed to fetch active workflow")
		}

		changed, _ := h.setStatusFromWorkflow(ctx, h.log, h.aw, &wf)
		updated = changed || updated

		if wf.Status.Phase.Completed() {
			h.aw.Status.ActiveWorkflowRef.Name = ""
			updated = true
		}
	}

	if !updated {
		return nil
	}

	h.log.V(1).Info("Updating status from active workflow", "cronWorkflow", cwf.Name)

	if err := h.Status().Update(ctx, h.aw); err != nil {
		return errLogAndWrap(h.log, err, "failed to update status")
	}

	return nil
}

func hydrateArgoWorkflowSpec(aw *arcv1alpha1.ArtifactWorkflow, srcSecret *corev1.Secret, dstSecret *corev1.Secret) wfv1alpha1.WorkflowSpec {
	srcVolume := corev1.Volume{
		Name: "src-secret-vol",
		VolumeSource: corev1.VolumeSource{
			EmptyDir: &corev1.EmptyDirVolumeSource{},
		},
	}
	if srcSecret.Name != "" {
		srcVolume.VolumeSource = corev1.VolumeSource{
			Secret: &corev1.SecretVolumeSource{
				SecretName: srcSecret.Name,
			},
		}
	}

	dstVolume := corev1.Volume{
		Name: "dst-secret-vol",
		VolumeSource: corev1.VolumeSource{
			EmptyDir: &corev1.EmptyDirVolumeSource{},
		},
	}
	if dstSecret.Name != "" {
		dstVolume.VolumeSource = corev1.VolumeSource{
			Secret: &corev1.SecretVolumeSource{
				SecretName: dstSecret.Name,
			},
		}
	}

	parameters := []wfv1alpha1.Parameter{}
	for _, p := range aw.Spec.Parameters {
		parameters = append(parameters, wfv1alpha1.Parameter{
			Name:  p.Name,
			Value: (*wfv1alpha1.AnyString)(&p.Value),
		})
	}

	return wfv1alpha1.WorkflowSpec{
		WorkflowTemplateRef: &wfv1alpha1.WorkflowTemplateRef{
			Name:         aw.Spec.WorkflowTemplateRef.Name,
			ClusterScope: aw.Spec.WorkflowTemplateRef.ClusterScope,
		},
		Volumes: []corev1.Volume{
			srcVolume,
			dstVolume,
		},
		Arguments: wfv1alpha1.Arguments{
			Parameters: parameters,
		},
	}
}

func hydrateArgoWorkflow(aw *arcv1alpha1.ArtifactWorkflow, srcSecret *corev1.Secret, dstSecret *corev1.Secret) *wfv1alpha1.Workflow {
	return &wfv1alpha1.Workflow{
		ObjectMeta: workflowObjectMeta(aw),
		Spec:       hydrateArgoWorkflowSpec(aw, srcSecret, dstSecret),
	}
}

func hydrateArgoCronWorkflow(aw *arcv1alpha1.ArtifactWorkflow, srcSecret *corev1.Secret, dstSecret *corev1.Secret) *wfv1alpha1.CronWorkflow {
	om := workflowObjectMeta(aw)
	wf := &wfv1alpha1.CronWorkflow{
		ObjectMeta: om,
		Spec: wfv1alpha1.CronWorkflowSpec{
			WorkflowSpec:               hydrateArgoWorkflowSpec(aw, srcSecret, dstSecret),
			Schedules:                  aw.Spec.Cron.Schedules,
			ConcurrencyPolicy:          wfv1alpha1.ReplaceConcurrent,
			StartingDeadlineSeconds:    aw.Spec.Cron.StartingDeadlineSeconds,
			Timezone:                   aw.Spec.Cron.Timezone,
			When:                       aw.Spec.Cron.When,
			SuccessfulJobsHistoryLimit: new(int32(1)),
			FailedJobsHistoryLimit:     new(int32(1)),
			WorkflowMetadata:           &om,
		},
	}

	return wf
}
