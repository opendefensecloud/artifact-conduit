// Copyright 2025 BWI GmbH and Artifact Conduit contributors
// SPDX-License-Identifier: Apache-2.0

package controller

import (
	"fmt"

	wfv1alpha1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	arcv1alpha1 "go.opendefense.cloud/arc/api/arc/v1alpha1"
	"go.opendefense.cloud/arc/pkg/envtest"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _ = Describe("ArtifactWorkflowController", func() {
	var (
		ctx           = envtest.Context()
		ns            = setupTest(ctx)
		at            = setupClusterArtifactType(ctx)
		createSecrets = func(names ...string) {
			for _, name := range names {
				secret := corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      name,
						Namespace: ns.Name,
					},
					StringData: map[string]string{
						"testkey": name,
					},
				}
				Expect(k8sClient.Create(ctx, &secret)).To(Succeed())
			}
		}
	)

	Context("when reconciling ArtifactWorkflows", func() {
		It("should create Workflow for ArtifactWorkflow without secrets", func() {
			awName := "no-secrets"
			aw := &arcv1alpha1.ArtifactWorkflow{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: ns.Name,
					Name:      awName,
				},
				Spec: arcv1alpha1.ArtifactWorkflowSpec{
					WorkflowTemplateRef: at.Spec.WorkflowTemplateRef,
					Parameters: []arcv1alpha1.ArtifactWorkflowParameter{
						{Name: awName, Value: awName},
					},
				},
			}
			Expect(k8sClient.Create(ctx, aw)).To(Succeed())

			wf := &wfv1alpha1.Workflow{}
			Eventually(func() error {
				return k8sClient.Get(ctx, namespacedName(ns.Name, aw.Name), wf)
			}).Should(Succeed())

			Expect(wf.Spec.Arguments.Parameters).To(HaveLen(1))
			Expect(wf.Spec.Arguments.Parameters).To(ConsistOf([]wfv1alpha1.Parameter{
				{Name: aw.Name, Value: (*wfv1alpha1.AnyString)(&aw.Name)},
			}))
			Expect(wf.Spec.Volumes).To(HaveLen(2))
			Expect(wf.Spec.Volumes[0].EmptyDir).ToNot(BeNil())
			Expect(wf.Spec.Volumes[1].EmptyDir).ToNot(BeNil())
		})

		It("should create Workflow for ArtifactWorkflow with secrets", func() {
			awName := "with-secrets"
			srcSecret := "src"
			dstSecret := "dst"
			createSecrets(srcSecret, dstSecret)
			aw := &arcv1alpha1.ArtifactWorkflow{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: ns.Name,
					Name:      awName,
				},
				Spec: arcv1alpha1.ArtifactWorkflowSpec{
					WorkflowTemplateRef: at.Spec.WorkflowTemplateRef,
					Parameters: []arcv1alpha1.ArtifactWorkflowParameter{
						{Name: awName, Value: awName},
					},
					SrcSecretRef: corev1.LocalObjectReference{Name: srcSecret},
					DstSecretRef: corev1.LocalObjectReference{Name: dstSecret},
				},
			}
			Expect(k8sClient.Create(ctx, aw)).To(Succeed())

			wf := &wfv1alpha1.Workflow{}
			Eventually(func() error {
				return k8sClient.Get(ctx, namespacedName(ns.Name, aw.Name), wf)
			}).Should(Succeed())

			Expect(wf.Spec.Arguments.Parameters).To(HaveLen(1))
			Expect(wf.Spec.Arguments.Parameters).To(ConsistOf([]wfv1alpha1.Parameter{
				{Name: aw.Name, Value: (*wfv1alpha1.AnyString)(&aw.Name)},
			}))
			Expect(wf.Spec.Volumes).To(HaveLen(2))
			Expect(wf.Spec.Volumes).To(ConsistOf([]corev1.Volume{
				{
					Name: "dst-secret-vol",
					VolumeSource: corev1.VolumeSource{
						Secret: &corev1.SecretVolumeSource{
							SecretName: dstSecret,
						},
					},
				},
				{
					Name: "src-secret-vol",
					VolumeSource: corev1.VolumeSource{
						Secret: &corev1.SecretVolumeSource{
							SecretName: srcSecret,
						},
					},
				},
			}))
		})

		It("should track Workflow status changes of created ArtifactWorkflows", func() {
			awName := "track-status"
			aw := &arcv1alpha1.ArtifactWorkflow{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: ns.Name,
					Name:      awName,
				},
				Spec: arcv1alpha1.ArtifactWorkflowSpec{
					WorkflowTemplateRef: at.Spec.WorkflowTemplateRef,
					Parameters: []arcv1alpha1.ArtifactWorkflowParameter{
						{Name: awName, Value: awName},
					},
				},
			}
			Expect(k8sClient.Create(ctx, aw)).To(Succeed())

			wf := &wfv1alpha1.Workflow{}
			Eventually(func() error {
				return k8sClient.Get(ctx, namespacedName(aw.Namespace, aw.Name), wf)
			}).Should(Succeed())

			// NOTE: Argo Workflows does not support the status resource atm:
			// https://github.com/argoproj/argo-workflows/issues/11082
			wf.Status.Phase = wfv1alpha1.WorkflowRunning
			Expect(k8sClient.Update(ctx, wf)).To(Succeed())

			Eventually(func() arcv1alpha1.WorkflowPhase {
				Expect(k8sClient.Get(ctx, namespacedName(aw.Namespace, aw.Name), aw)).To(Succeed())
				return aw.Status.Phase
			}).To(Equal(arcv1alpha1.WorkflowRunning))

			wf.Status.Phase = wfv1alpha1.WorkflowSucceeded
			Expect(k8sClient.Update(ctx, wf)).To(Succeed())

			Eventually(func() arcv1alpha1.WorkflowPhase {
				Expect(k8sClient.Get(ctx, namespacedName(aw.Namespace, aw.Name), aw)).To(Succeed())
				return aw.Status.Phase
			}).To(Equal(arcv1alpha1.WorkflowSucceeded))
		})

		It("should track failed Workflow information of created ArtifactWorkflows", func() {
			awName := "track-failed-status"
			aw := &arcv1alpha1.ArtifactWorkflow{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: ns.Name,
					Name:      awName,
				},
				Spec: arcv1alpha1.ArtifactWorkflowSpec{
					WorkflowTemplateRef: at.Spec.WorkflowTemplateRef,
					Parameters: []arcv1alpha1.ArtifactWorkflowParameter{
						{Name: awName, Value: awName},
					},
				},
			}
			Expect(k8sClient.Create(ctx, aw)).To(Succeed())

			wf := &wfv1alpha1.Workflow{}
			Eventually(func() error {
				return k8sClient.Get(ctx, namespacedName(aw.Namespace, aw.Name), wf)
			}).Should(Succeed())

			// NOTE: Argo Workflows does not support the status resource atm:
			// https://github.com/argoproj/argo-workflows/issues/11082
			wf.Status.Phase = wfv1alpha1.WorkflowFailed
			wf.Status.Nodes = map[string]wfv1alpha1.NodeStatus{
				"workflow-name-v66mf-1111": {
					ID:          "workflow-name-v66mf-1111",
					BoundaryID:  "workflow-name-v66mf",
					DisplayName: "step1",
					Phase:       wfv1alpha1.NodeFailed,
					Type:        wfv1alpha1.NodeTypePod,
				},
				"workflow-name-v66mf-2222": {
					ID:          "workflow-name-v66mf-2222",
					BoundaryID:  "workflow-name-v66mf",
					DisplayName: "step2",
					Message:     "Error (exit code 1): Scan failed",
					Phase:       wfv1alpha1.NodeFailed,
					Type:        wfv1alpha1.NodeTypePod,
				},
				"workflow-name-v66mf-3333": {
					ID:          "workflow-name-v66mf-3333",
					BoundaryID:  "workflow-name-v66mf",
					DisplayName: "step3",
					Phase:       wfv1alpha1.NodeSucceeded,
					Type:        wfv1alpha1.NodeTypePod,
				},
			}
			Expect(k8sClient.Update(ctx, wf)).To(Succeed())

			Eventually(func() arcv1alpha1.WorkflowPhase {
				Expect(k8sClient.Get(ctx, namespacedName(aw.Namespace, aw.Name), aw)).To(Succeed())
				return aw.Status.Phase
			}).To(Equal(arcv1alpha1.WorkflowFailed))
			Eventually(func() string {
				Expect(k8sClient.Get(ctx, namespacedName(aw.Namespace, aw.Name), aw)).To(Succeed())
				return aw.Status.Message
			}).To(ContainSubstring("Step 'step1' failed"))
		})

		It("should handle force reconcile annotation", func() {
			awName := "force-reconcile"
			aw := &arcv1alpha1.ArtifactWorkflow{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: ns.Name,
					Name:      awName,
				},
				Spec: arcv1alpha1.ArtifactWorkflowSpec{
					WorkflowTemplateRef: at.Spec.WorkflowTemplateRef,
					Parameters: []arcv1alpha1.ArtifactWorkflowParameter{
						{Name: awName, Value: awName},
					},
				},
			}
			Expect(k8sClient.Create(ctx, aw)).To(Succeed())

			// Verify that the argo workflow is re-created (i.e., a new one with a different UID)
			var originalUID string
			Eventually(func() error {
				wf := &wfv1alpha1.Workflow{}
				if err := k8sClient.Get(ctx, namespacedName(aw.Namespace, aw.Name), wf); err != nil {
					return err
				}
				originalUID = string(wf.UID)
				return nil
			}).Should(Succeed())

			// Update the annotation to trigger a re-reconcile
			Eventually(func() error {
				order := &arcv1alpha1.ArtifactWorkflow{}
				if err := k8sClient.Get(ctx, namespacedName(aw.Namespace, aw.Name), order); err != nil {
					return err
				}
				if order.Annotations == nil {
					order.Annotations = map[string]string{}
				}
				order.Annotations[forceAtAnnotation] = fmt.Sprintf("%v", metav1.Now().Unix())
				return k8sClient.Update(ctx, order)
			}).Should(Succeed())

			// Verify that a new workflow has been created
			Eventually(func() (string, error) {
				wf := &wfv1alpha1.Workflow{}
				if err := k8sClient.Get(ctx, namespacedName(aw.Namespace, aw.Name), wf); err != nil {
					return "", err
				}
				return string(wf.UID), nil
			}).ShouldNot(Equal(originalUID))
		})
	})
})
