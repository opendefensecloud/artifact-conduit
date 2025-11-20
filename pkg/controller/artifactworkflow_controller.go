// Copyright 2025 BWI GmbH and Artefact Conduit contributors
// SPDX-License-Identifier: Apache-2.0

package controller

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"slices"
	"strings"

	wfv1alpha1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	"github.com/go-logr/logr"
	"github.com/jastBytes/sprint"
	arcv1alpha1 "go.opendefense.cloud/arc/api/arc/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

const (
	artifactWorkflowFinalizer = "arc.bwi.de/artifact-workflow-finalizer"
)

// ArtifactWorkflowReconciler reconciles a ArtifactWorkflow object
type ArtifactWorkflowReconciler struct {
	client.Client
	ClientSet *kubernetes.Clientset
	Scheme    *runtime.Scheme
}

//+kubebuilder:rbac:groups=arc.bwi.de,resources=artifacttypes,verbs=get;list;watch
//+kubebuilder:rbac:groups=arc.bwi.de,resources=artifactworkflows/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=arc.bwi.de,resources=artifactworkflows/finalizers,verbs=update
//+kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=argoproj.io,resources=workflows,verbs=get;list;watch;create;update;patch;delete

// Reconcile moves the current state of the cluster closer to the desired state
func (r *ArtifactWorkflowReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := ctrl.LoggerFrom(ctx)

	aw := &arcv1alpha1.ArtifactWorkflow{}
	if err := r.Get(ctx, req.NamespacedName, aw); err != nil {
		if apierrors.IsNotFound(err) {
			// Object not found, return.
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, errLogAndWrap(log, err, "failed to get object")
	}

	if !aw.DeletionTimestamp.IsZero() {
		log.V(1).Info("ArtifactWorkflow is being deleted")
		// Cleanup workflow, if exists
		wf := wfv1alpha1.Workflow{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: aw.Namespace,
				Name:      aw.Name,
			},
		}
		if err := r.Delete(ctx, &wf); client.IgnoreNotFound(err) != nil {
			return ctrl.Result{}, errLogAndWrap(log, err, "workflow deletion failed")
		}
		// Remove finalizer
		if slices.Contains(aw.Finalizers, artifactWorkflowFinalizer) {
			log.V(1).Info("Removing finalizer from ArtifactWorkflow")
			aw.Finalizers = slices.DeleteFunc(aw.Finalizers, func(f string) bool {
				return f == artifactWorkflowFinalizer
			})
			if err := r.Update(ctx, aw); err != nil {
				return ctrl.Result{}, errLogAndWrap(log, err, "failed to remove finalizer")
			}
		}
		return ctrl.Result{}, nil
	}

	// Add finalizer if not present and not deleting
	if aw.DeletionTimestamp.IsZero() {
		if !slices.Contains(aw.Finalizers, artifactWorkflowFinalizer) {
			log.V(1).Info("Adding finalizer to ArtifactWorkflow")
			aw.Finalizers = append(aw.Finalizers, artifactWorkflowFinalizer)
			if err := r.Update(ctx, aw); err != nil {
				return ctrl.Result{}, errLogAndWrap(log, err, "failed to add finalizer")
			}
			// Return without requeue; the Update event will trigger reconciliation again
			return ctrl.Result{}, nil
		}
	}

	if aw.Status.Phase == arcv1alpha1.WorkflowUnknown {
		return r.createArgoWorkflow(ctx, log, aw)
	}

	if res, err := r.checkArgoWorkflow(ctx, log, aw); err != nil {
		return res, err
	}
	return ctrl.Result{}, nil
}

func (r *ArtifactWorkflowReconciler) createArgoWorkflow(ctx context.Context, log logr.Logger, aw *arcv1alpha1.ArtifactWorkflow) (ctrl.Result, error) {
	artifactType := arcv1alpha1.ArtifactType{}
	if err := r.Get(ctx, namespacedName("", aw.Spec.Type), &artifactType); err != nil {
		return ctrl.Result{}, errLogAndWrap(log, err, "failed to retrieve artifact type")
	}

	srcSecret := corev1.Secret{}
	if aw.Spec.SrcSecretRef.Name != "" {
		if err := r.Get(ctx, namespacedName(aw.Namespace, aw.Spec.SrcSecretRef.Name), &srcSecret); err != nil {
			return ctrl.Result{}, errLogAndWrap(log, err, "failed to fetch secret for source")
		}
	}

	dstSecret := corev1.Secret{}
	if aw.Spec.DstSecretRef.Name != "" {
		if err := r.Get(ctx, namespacedName(aw.Namespace, aw.Spec.DstSecretRef.Name), &dstSecret); err != nil {
			return ctrl.Result{}, errLogAndWrap(log, err, "failed to fetch secret for destination")
		}
	}

	wf := r.hydrateArgoWorkflow(aw, &artifactType, &srcSecret, &dstSecret)

	if err := controllerutil.SetControllerReference(aw, wf, r.Scheme); err != nil {
		return ctrl.Result{}, errLogAndWrap(log, err, "failed to set controller reference")
	}

	if err := r.Create(ctx, wf); client.IgnoreAlreadyExists(err) != nil {
		return ctrl.Result{}, errLogAndWrap(log, err, "failed to create argo workflow")
	}

	aw.Status.Phase = arcv1alpha1.WorkflowPending
	if err := r.Status().Update(ctx, aw); err != nil {
		return ctrl.Result{}, errLogAndWrap(log, err, "failed to update status")
	}
	return ctrl.Result{}, nil
}

func (r *ArtifactWorkflowReconciler) hydrateArgoWorkflow(aw *arcv1alpha1.ArtifactWorkflow, artifactType *arcv1alpha1.ArtifactType, srcSecret *corev1.Secret, dstSecret *corev1.Secret) *wfv1alpha1.Workflow {

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
	for _, p := range artifactType.Spec.Parameters {
		parameters = append(parameters, wfv1alpha1.Parameter{
			Name:  p.Name,
			Value: (*wfv1alpha1.AnyString)(&p.Value),
		})
	}

	wf := &wfv1alpha1.Workflow{
		ObjectMeta: metav1.ObjectMeta{
			Name:      aw.Name,
			Namespace: aw.Namespace,
		},
		Spec: wfv1alpha1.WorkflowSpec{
			WorkflowTemplateRef: &wfv1alpha1.WorkflowTemplateRef{
				Name:         artifactType.Spec.WorkflowTemplateRef.Name,
				ClusterScope: true,
			},
			Volumes: []corev1.Volume{
				srcVolume,
				dstVolume,
			},
			Arguments: wfv1alpha1.Arguments{
				Parameters: parameters,
			},
		},
	}

	return wf
}

func (r *ArtifactWorkflowReconciler) checkArgoWorkflow(ctx context.Context, log logr.Logger, aw *arcv1alpha1.ArtifactWorkflow) (ctrl.Result, error) {
	wf := wfv1alpha1.Workflow{}
	if err := r.Get(ctx, namespacedName(aw.Namespace, aw.Name), &wf); err != nil {
		return ctrl.Result{}, errLogAndWrap(log, err, "failed to get workflow")
	}

	if aw.Status.Message == "" {
		switch wf.Status.Phase {
		case wfv1alpha1.WorkflowFailed:
			r.generateWorkflowStatusMessage(ctx, wf, log, aw)
		case wfv1alpha1.WorkflowError:
			// TODO: Properly show why the workflow errored
			aw.Status.Message = wf.Status.Message
		}
	}

	aw.Status.Phase = arcv1alpha1.WorkflowPhase(wf.Status.Phase)
	if err := r.Status().Update(ctx, aw); err != nil {
		return ctrl.Result{}, errLogAndWrap(log, err, "failed to update status")
	}
	return ctrl.Result{}, nil
}

func (r *ArtifactWorkflowReconciler) generateWorkflowStatusMessage(ctx context.Context, wf wfv1alpha1.Workflow, log logr.Logger, aw *arcv1alpha1.ArtifactWorkflow) {
	reports := []struct {
		Name    string
		Pod     string
		Message string
		Logs    string
	}{}
	for _, node := range wf.Status.Nodes {
		if node.Phase == wfv1alpha1.NodeFailed && node.Type == wfv1alpha1.NodeTypePod {
			nr := struct {
				Name    string
				Pod     string
				Message string
				Logs    string
			}{
				Name:    node.DisplayName,
				Pod:     generatePodNameFromNodeStatus(node),
				Message: node.Message,
			}
			reports = append(reports, nr)
		}
	}

	for _, nr := range reports {
		logs, err := r.fetchPodLogs(ctx, aw.Namespace, nr.Pod)
		if err != nil {
			log.V(1).Info("failed to fetch pod logs", "pod", nr.Pod, "error", err)
		} else {
			nr.Logs = logs
		}
		aw.Status.Message += fmt.Sprintf("Step '%s' failed:\n%s\nLogs:\n%s\n\n", nr.Name, nr.Message, nr.Logs)
	}
}

func generatePodNameFromNodeStatus(node wfv1alpha1.NodeStatus) string {
	podId := node.ID[strings.LastIndex(node.ID, "-")+1:]
	return fmt.Sprintf("%s-%s-%s", node.BoundaryID, node.DisplayName, podId)
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

// SetupWithManager sets up the controller with the Manager.
func (r *ArtifactWorkflowReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&arcv1alpha1.ArtifactWorkflow{}).
		Owns(&wfv1alpha1.Workflow{}).
		Complete(r)
}
