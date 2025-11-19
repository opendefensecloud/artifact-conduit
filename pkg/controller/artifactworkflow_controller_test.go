// Copyright 2025 BWI GmbH and Artifact Conduit contributors
// SPDX-License-Identifier: Apache-2.0

package controller

import (
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
		ctx = envtest.Context()
		ns  = SetupTest(ctx)

		createArtifactType = func(name string) {
			at := &arcv1alpha1.ArtifactType{
				ObjectMeta: metav1.ObjectMeta{
					Name: name,
				},
				Spec: arcv1alpha1.ArtifactTypeSpec{
					Parameters: []arcv1alpha1.ArtifactWorkflowParameter{
						{
							Name:  name,
							Value: name,
						},
					},
					WorkflowTemplateRef: corev1.LocalObjectReference{
						Name: name,
					},
				},
			}
			Expect(k8sClient.Create(ctx, at)).To(Succeed())
			DeferCleanup(k8sClient.Delete, ctx, at)
		}
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
			atName := "art"
			awName := "no-secrets"
			createArtifactType(atName)
			aw := &arcv1alpha1.ArtifactWorkflow{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: ns.Name,
					Name:      awName,
				},
				Spec: arcv1alpha1.ArtifactWorkflowSpec{
					Type: atName,
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

			Expect(wf.Spec.Arguments.Parameters).To(HaveLen(2))
			Expect(wf.Spec.Arguments.Parameters).To(ConsistOf([]wfv1alpha1.Parameter{
				{Name: atName, Value: (*wfv1alpha1.AnyString)(&atName)},
				{Name: aw.Name, Value: (*wfv1alpha1.AnyString)(&aw.Name)},
			}))
			Expect(wf.Spec.Volumes).To(HaveLen(2))
			Expect(wf.Spec.Volumes[0].EmptyDir).ToNot(BeNil())
			Expect(wf.Spec.Volumes[1].EmptyDir).ToNot(BeNil())
		})

		It("should create Workflow for ArtifactWorkflow with secrets", func() {
			atName := "art"
			awName := "with-secrets"
			srcSecret := "src"
			dstSecret := "dst"
			createArtifactType(atName)
			createSecrets(srcSecret, dstSecret)
			aw := &arcv1alpha1.ArtifactWorkflow{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: ns.Name,
					Name:      awName,
				},
				Spec: arcv1alpha1.ArtifactWorkflowSpec{
					Type: atName,
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

			Expect(wf.Spec.Arguments.Parameters).To(HaveLen(2))
			Expect(wf.Spec.Arguments.Parameters).To(ConsistOf([]wfv1alpha1.Parameter{
				{Name: atName, Value: (*wfv1alpha1.AnyString)(&atName)},
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
			// TODO
		})
	})
})
