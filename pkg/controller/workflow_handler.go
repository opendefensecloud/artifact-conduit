// Copyright 2025 BWI GmbH and Artifact Conduit contributors
// SPDX-License-Identifier: Apache-2.0

package controller

import (
	"context"
	"fmt"

	wfv1alpha1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	"github.com/go-logr/logr"
	arcv1alpha1 "go.opendefense.cloud/arc/api/arc/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
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
		h.Recorder.Event(h.aw, corev1.EventTypeWarning, "DeletionFailed", fmt.Sprintf("Failed to delete associated workflow '%s': %v", h.aw.Name, err))
		return errLogAndWrap(h.log, err, "workflow deletion failed")
	}
	h.Recorder.Event(h.aw, corev1.EventTypeNormal, "Deleted", fmt.Sprintf("Deleted workflow '%s'", h.aw.Name))
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
		h.Recorder.Event(h.aw, corev1.EventTypeWarning, "CreationFailed", fmt.Sprintf("Failed to create workflow '%s': %v", wf.GetName(), err))
		return errLogAndWrap(h.log, err, "failed to create argo workflow")
	}
	h.Recorder.Event(h.aw, corev1.EventTypeNormal, "Created", fmt.Sprintf("Created workflow '%s'", wf.GetName()))

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

	if updated := h.setStatusFromWorkflow(ctx, h.log, h.aw, &wf); !updated {
		return nil // nothing updated
	}

	if err := h.Status().Update(ctx, h.aw); err != nil {
		return errLogAndWrap(h.log, err, "failed to update status")
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
		h.Recorder.Event(h.aw, corev1.EventTypeWarning, "DeletionFailed", fmt.Sprintf("Failed to delete associated cron workflow '%s': %v", h.aw.Name, err))
		return errLogAndWrap(h.log, err, "cron workflow deletion failed")
	}
	h.Recorder.Event(h.aw, corev1.EventTypeNormal, "Deleted", fmt.Sprintf("Deleted cron workflow '%s'", h.aw.Name))
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

	if err := h.Create(ctx, cwf); client.IgnoreAlreadyExists(err) != nil {
		h.Recorder.Event(h.aw, corev1.EventTypeWarning, "CreationFailed", fmt.Sprintf("Failed to create cron workflow '%s': %v", cwf.GetName(), err))
		return errLogAndWrap(h.log, err, "failed to create argo cron workflow")
	}
	h.Recorder.Event(h.aw, corev1.EventTypeNormal, "Created", fmt.Sprintf("Created cron workflow '%s'", cwf.GetName()))

	h.aw.Status.Phase = arcv1alpha1.WorkflowPending
	if err := h.Status().Update(ctx, h.aw); err != nil {
		return errLogAndWrap(h.log, err, "failed to update status")
	}
	return nil
}

func (h *CronWorkflowHandler) CheckArgoResources(ctx context.Context) error {
	if h.aw.Status.Phase == arcv1alpha1.WorkflowStopped {
		return nil
	}

	if h.aw.Status.ActiveWorkflowRef.Name != "" {
		wf := wfv1alpha1.Workflow{}
		if err := h.Get(ctx, namespacedName(h.aw.Namespace, h.aw.Status.ActiveWorkflowRef.Name), &wf); err != nil {
			return errLogAndWrap(h.log, err, "failed to fetch active workflow")
		}

		if updated := h.setStatusFromWorkflow(ctx, h.log, h.aw, &wf); !updated {
			return nil // nothing updated
		}

		if h.aw.Status.Phase.Completed() {
			h.aw.Status.ActiveWorkflowRef.Name = ""
		}

		if err := h.Status().Update(ctx, h.aw); err != nil {
			return errLogAndWrap(h.log, err, "failed to update status")
		}

		return nil
	}

	cwf := wfv1alpha1.CronWorkflow{}
	if err := h.Get(ctx, namespacedName(h.aw.Namespace, h.aw.Name), &cwf); err != nil {
		return errLogAndWrap(h.log, err, "failed to get cron workflow")
	}

	if cwf.Status.Phase == wfv1alpha1.StoppedPhase {
		h.aw.Status.Phase = arcv1alpha1.WorkflowStopped

		if err := h.Status().Update(ctx, h.aw); err != nil {
			return errLogAndWrap(h.log, err, "failed to update status")
		}

		return nil
	}

	if len(cwf.Status.Active) > 0 {
		// Should only contain a single element at most
		ref := cwf.Status.Active[0]
		wf := wfv1alpha1.Workflow{}
		if err := h.Get(ctx, namespacedName(ref.Namespace, ref.Name), &wf); err != nil {
			return errLogAndWrap(h.log, err, "failed to fetch active workflow")
		}

		h.aw.Status.ActiveWorkflowRef = corev1.LocalObjectReference{
			Name: wf.Name,
		}
		h.aw.Status.Message = ""

		if err := h.Status().Update(ctx, h.aw); err != nil {
			return errLogAndWrap(h.log, err, "failed to update status")
		}
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
	wf := &wfv1alpha1.CronWorkflow{
		ObjectMeta: workflowObjectMeta(aw),
		Spec: wfv1alpha1.CronWorkflowSpec{
			WorkflowSpec:            hydrateArgoWorkflowSpec(aw, srcSecret, dstSecret),
			Schedules:               aw.Spec.Cron.Schedules,
			ConcurrencyPolicy:       wfv1alpha1.ReplaceConcurrent,
			StartingDeadlineSeconds: aw.Spec.Cron.StartingDeadlineSeconds,
			Timezone:                aw.Spec.Cron.Timezone,
			When:                    aw.Spec.Cron.When,
		},
	}

	return wf
}
