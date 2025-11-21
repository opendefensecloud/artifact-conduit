// Copyright 2025 BWI GmbH and Artifact Conduit contributors
// SPDX-License-Identifier: Apache-2.0

package controller

import (
	"maps"
	"slices"

	wfv1alpha1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	arcv1alpha1 "go.opendefense.cloud/arc/api/arc/v1alpha1"
	"go.opendefense.cloud/arc/pkg/envtest"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var _ = Describe("OrderController", func() {
	var (
		ctx             = envtest.Context()
		ns              = setupTest(ctx)
		at1             = setupArtifactType(ctx)
		at2             = setupArtifactType(ctx)
		at3             = setupArtifactType(ctx)
		createEndpoints = func(names ...string) {
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
				endpoint := arcv1alpha1.Endpoint{
					ObjectMeta: metav1.ObjectMeta{
						Name:      name,
						Namespace: ns.Name,
					},
					Spec: arcv1alpha1.EndpointSpec{
						Type:      name,
						RemoteURL: name,
						SecretRef: corev1.LocalObjectReference{
							Name: name,
						},
						Usage: arcv1alpha1.EndpointUsageAll,
					},
				}
				Expect(k8sClient.Create(ctx, &endpoint)).To(Succeed())
			}
		}
	)

	Context("when reconciling Orders", func() {
		It("should create ArtifactWorkflows for an order with multiple artifacts and no defaults", func() {
			createEndpoints("src-1", "dst-1", "src-2", "dst-2")
			// Create test Order with multiple artifacts, no defaults
			order := &arcv1alpha1.Order{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-order-no-defaults",
					Namespace: ns.Name,
				},
				Spec: arcv1alpha1.OrderSpec{
					Artifacts: []arcv1alpha1.OrderArtifact{
						{
							Type:   at1.Name,
							SrcRef: corev1.LocalObjectReference{Name: "src-1"},
							DstRef: corev1.LocalObjectReference{Name: "dst-1"},
							Spec:   runtime.RawExtension{Raw: []byte(`{"key":"value-1"}`)},
						},
						{
							Type:   at2.Name,
							SrcRef: corev1.LocalObjectReference{Name: "src-2"},
							DstRef: corev1.LocalObjectReference{Name: "dst-2"},
							Spec:   runtime.RawExtension{Raw: []byte(`{"key":"value-2"}`)},
						},
					},
				},
			}
			Expect(k8sClient.Create(ctx, order)).To(Succeed())

			// Verify artifact workflows were created
			awList := &arcv1alpha1.ArtifactWorkflowList{}
			Eventually(func() int {
				err := k8sClient.List(ctx, awList, client.InNamespace(ns.Name))
				if err != nil {
					return 0
				}
				return len(awList.Items)
			}).Should(Equal(2))

			// Verify artifact workflows contents
			for _, aw := range awList.Items {
				Expect(aw.Spec.Type).To(Or(Equal(at1.Name), Equal(at2.Name)))
				suffix := "1"
				if aw.Spec.Type == at2.Name {
					suffix = "2"
				}
				Expect(aw.Spec.SrcSecretRef.Name).To(Equal("src-" + suffix))
				Expect(aw.Spec.DstSecretRef.Name).To(Equal("dst-" + suffix))
				Expect(aw.Spec.Parameters).To(ConsistOf([]arcv1alpha1.ArtifactWorkflowParameter{
					{
						Name:  "srcType",
						Value: "src-" + suffix,
					},
					{
						Name:  "srcRemoteURL",
						Value: "src-" + suffix,
					},
					{
						Name:  "dstType",
						Value: "dst-" + suffix,
					},
					{
						Name:  "dstRemoteURL",
						Value: "dst-" + suffix,
					},
					{
						Name:  "srcSecret",
						Value: "true",
					},
					{
						Name:  "dstSecret",
						Value: "true",
					},
					{
						Name:  "specKey",
						Value: "value-" + suffix,
					},
				}))
			}
		})

		It("should create artifact workflows for an order with multiple artifacts using defaults", func() {
			createEndpoints("default-src", "default-dst")
			// Create test Order with defaults
			order := &arcv1alpha1.Order{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-order-with-defaults",
					Namespace: ns.Name,
				},
				Spec: arcv1alpha1.OrderSpec{
					Defaults: arcv1alpha1.OrderDefaults{
						SrcRef: corev1.LocalObjectReference{Name: "default-src"},
						DstRef: corev1.LocalObjectReference{Name: "default-dst"},
					},
					Artifacts: []arcv1alpha1.OrderArtifact{
						{
							Type: at1.Name,
							Spec: runtime.RawExtension{Raw: []byte(`{"key":"value"}`)},
						},
						{
							Type: at2.Name,
							Spec: runtime.RawExtension{Raw: []byte(`{"key":"value"}`)},
						},
					},
				},
			}
			Expect(k8sClient.Create(ctx, order)).To(Succeed())

			// Verify fragments were created
			awList := &arcv1alpha1.ArtifactWorkflowList{}
			Eventually(func() int {
				err := k8sClient.List(ctx, awList, client.InNamespace(ns.Name))
				if err != nil {
					return 0
				}
				return len(awList.Items)
			}).Should(Equal(2))

			// Verify artifact workflow parameters - should use default refs
			for _, aw := range awList.Items {
				Expect(aw.Spec.Type).To(Or(Equal(at1.Name), Equal(at2.Name)))
				Expect(aw.Spec.Parameters).To(ContainElements([]arcv1alpha1.ArtifactWorkflowParameter{
					{
						Name:  "srcType",
						Value: "default-src",
					},
					{
						Name:  "srcRemoteURL",
						Value: "default-src",
					},
					{
						Name:  "dstType",
						Value: "default-dst",
					},
					{
						Name:  "dstRemoteURL",
						Value: "default-dst",
					},
					{
						Name:  "specKey",
						Value: "value",
					},
				}))
			}
		})

		It("should create artifact workflows for an order with mixed default usage", func() {
			createEndpoints("default-src", "default-dst", "src-1", "dst-1")
			// Create test Order with some artifacts using defaults, others specifying refs
			order := &arcv1alpha1.Order{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-order-mixed-defaults",
					Namespace: ns.Name,
				},
				Spec: arcv1alpha1.OrderSpec{
					Defaults: arcv1alpha1.OrderDefaults{
						SrcRef: corev1.LocalObjectReference{Name: "default-src"},
						DstRef: corev1.LocalObjectReference{Name: "default-dst"},
					},
					Artifacts: []arcv1alpha1.OrderArtifact{
						{
							Type: at1.Name,
							// Uses defaults for both refs
							Spec: runtime.RawExtension{Raw: []byte(`{"key":"value"}`)},
						},
						{
							Type: at2.Name,
							// Specifies src, uses default dst
							SrcRef: corev1.LocalObjectReference{Name: "src-1"},
							Spec:   runtime.RawExtension{Raw: []byte(`{"key":"value"}`)},
						},
						{
							Type: at3.Name,
							// Specifies both refs
							SrcRef: corev1.LocalObjectReference{Name: "src-1"},
							DstRef: corev1.LocalObjectReference{Name: "dst-1"},
							Spec:   runtime.RawExtension{Raw: []byte(`{"key":"value"}`)},
						},
					},
				},
			}
			Expect(k8sClient.Create(ctx, order)).To(Succeed())

			// Verify fragments were created
			awList := &arcv1alpha1.ArtifactWorkflowList{}
			Eventually(func() int {
				err := k8sClient.List(ctx, awList, client.InNamespace(ns.Name))
				if err != nil {
					return 0
				}
				return len(awList.Items)
			}).Should(Equal(3))

			// Verify artifact workflow contents
			for _, aw := range awList.Items {
				switch aw.Spec.Type {
				case at1.Name:
					// Should use defaults for both
					Expect(aw.Spec.Parameters).To(ContainElements([]arcv1alpha1.ArtifactWorkflowParameter{
						{
							Name:  "srcType",
							Value: "default-src",
						},
						{
							Name:  "dstType",
							Value: "default-dst",
						},
					}))
				case at2.Name:
					// Should use custom src, default dst
					Expect(aw.Spec.Parameters).To(ContainElements([]arcv1alpha1.ArtifactWorkflowParameter{
						{
							Name:  "srcType",
							Value: "src-1",
						},
						{
							Name:  "dstType",
							Value: "default-dst",
						},
					}))
				case at3.Name:
					// Should use custom refs for both
					Expect(aw.Spec.Parameters).To(ContainElements([]arcv1alpha1.ArtifactWorkflowParameter{
						{
							Name:  "srcType",
							Value: "src-1",
						},
						{
							Name:  "dstType",
							Value: "dst-1",
						},
					}))
				}
			}

			// Verify status contains all artifact workflows
			Eventually(func() int {
				if err := k8sClient.Get(ctx, client.ObjectKeyFromObject(order), order); err != nil {
					return 0
				}
				return len(order.Status.ArtifactWorkflows)
			}).Should(Equal(3))
		})

		It("should delete artifact workflows and secrets when order is deleted", func() {
			createEndpoints("src-1", "dst-1", "src-2", "dst-2")
			order := &arcv1alpha1.Order{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-order-delete",
					Namespace: ns.Name,
				},
				Spec: arcv1alpha1.OrderSpec{
					Artifacts: []arcv1alpha1.OrderArtifact{
						{Type: at1.Name, SrcRef: corev1.LocalObjectReference{Name: "src-1"}, DstRef: corev1.LocalObjectReference{Name: "dst-1"}},
						{Type: at2.Name, SrcRef: corev1.LocalObjectReference{Name: "src-2"}, DstRef: corev1.LocalObjectReference{Name: "dst-2"}},
					},
				},
			}
			Expect(k8sClient.Create(ctx, order)).To(Succeed())
			awList := &arcv1alpha1.ArtifactWorkflowList{}
			Eventually(func() int {
				_ = k8sClient.List(ctx, awList, client.InNamespace(ns.Name))
				return len(awList.Items)
			}).Should(Equal(2))
			// Delete order
			Expect(k8sClient.Delete(ctx, order)).To(Succeed())
			// Eventually all fragments should be gone
			Eventually(func() int {
				_ = k8sClient.List(ctx, awList, client.InNamespace(ns.Name))
				return len(awList.Items)
			}).Should(Equal(0))
		})

		It("should create a new artifact workflow and update status when an artifact is added", func() {
			createEndpoints("src-1", "dst-1", "src-2", "dst-2")
			order := &arcv1alpha1.Order{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-order-add-artifact",
					Namespace: ns.Name,
				},
				Spec: arcv1alpha1.OrderSpec{
					Artifacts: []arcv1alpha1.OrderArtifact{
						{Type: at1.Name, SrcRef: corev1.LocalObjectReference{Name: "src-1"}, DstRef: corev1.LocalObjectReference{Name: "dst-1"}},
					},
				},
			}
			Expect(k8sClient.Create(ctx, order)).To(Succeed())
			awList := &arcv1alpha1.ArtifactWorkflowList{}
			Eventually(func() int {
				_ = k8sClient.List(ctx, awList, client.InNamespace(ns.Name))
				return len(awList.Items)
			}).Should(Equal(1))
			// Add a new artifact with retry on conflict
			Eventually(func() error {
				if err := k8sClient.Get(ctx, client.ObjectKeyFromObject(order), order); err != nil {
					return err
				}
				order.Spec.Artifacts = append(order.Spec.Artifacts, arcv1alpha1.OrderArtifact{
					Type:   at2.Name,
					SrcRef: corev1.LocalObjectReference{Name: "src-2"},
					DstRef: corev1.LocalObjectReference{Name: "dst-2"},
				})
				return k8sClient.Update(ctx, order)
			}).Should(Succeed())
			// Eventually two fragments should exist
			Eventually(func() int {
				_ = k8sClient.List(ctx, awList, client.InNamespace(ns.Name))
				return len(awList.Items)
			}).Should(Equal(2))
			// Status should be updated
			Eventually(func() int {
				_ = k8sClient.Get(ctx, client.ObjectKeyFromObject(order), order)
				return len(order.Status.ArtifactWorkflows)
			}).Should(Equal(2))
		})

		It("should delete an artifact workflow and update status when an artifact is removed", func() {
			createEndpoints("src-1", "dst-1", "src-2", "dst-2")
			order := &arcv1alpha1.Order{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-order-remove-artifact",
					Namespace: ns.Name,
				},
				Spec: arcv1alpha1.OrderSpec{
					Artifacts: []arcv1alpha1.OrderArtifact{
						{Type: "type-e", SrcRef: corev1.LocalObjectReference{Name: "src-1"}, DstRef: corev1.LocalObjectReference{Name: "dst-1"}},
						{Type: "type-f", SrcRef: corev1.LocalObjectReference{Name: "src-2"}, DstRef: corev1.LocalObjectReference{Name: "dst-2"}},
					},
				},
			}
			Expect(k8sClient.Create(ctx, order)).To(Succeed())
			awList := &arcv1alpha1.ArtifactWorkflowList{}
			Eventually(func() int {
				_ = k8sClient.List(ctx, awList, client.InNamespace(ns.Name))
				return len(awList.Items)
			}).Should(Equal(2))
			// Remove one artifact with retry on conflict
			Eventually(func() error {
				if err := k8sClient.Get(ctx, client.ObjectKeyFromObject(order), order); err != nil {
					return err
				}
				order.Spec.Artifacts = order.Spec.Artifacts[:1]
				return k8sClient.Update(ctx, order)
			}).Should(Succeed())
			// Eventually only one fragment should exist
			Eventually(func() int {
				_ = k8sClient.List(ctx, awList, client.InNamespace(ns.Name))
				return len(awList.Items)
			}).Should(Equal(1))
			// Status should be updated
			Eventually(func() int {
				_ = k8sClient.Get(ctx, client.ObjectKeyFromObject(order), order)
				return len(order.Status.ArtifactWorkflows)
			}).Should(Equal(1))
		})

		It("should work with endpoints without a secret", func() {
			endpoint := arcv1alpha1.Endpoint{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "no-secret",
					Namespace: ns.Name,
				},
				Spec: arcv1alpha1.EndpointSpec{
					Type:      "no-secret",
					RemoteURL: "no-secret",
					Usage:     arcv1alpha1.EndpointUsageAll,
				},
			}
			Expect(k8sClient.Create(ctx, &endpoint)).To(Succeed())

			// Create order using the no-secret endpoint as src and dst for artifacts
			order := &arcv1alpha1.Order{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-order-no-secret",
					Namespace: ns.Name,
				},
				Spec: arcv1alpha1.OrderSpec{
					Artifacts: []arcv1alpha1.OrderArtifact{
						{
							Type:   "art",
							SrcRef: corev1.LocalObjectReference{Name: "no-secret"},
							DstRef: corev1.LocalObjectReference{Name: "no-secret"},
							Spec:   runtime.RawExtension{Raw: []byte(`{"key":"value"}`)},
						},
					},
				},
			}
			Expect(k8sClient.Create(ctx, order)).To(Succeed())

			// Verify artifact workflows were created
			awList := &arcv1alpha1.ArtifactWorkflowList{}
			Eventually(func() int {
				err := k8sClient.List(ctx, awList, client.InNamespace(ns.Name))
				if err != nil {
					return 0
				}
				return len(awList.Items)
			}).Should(Equal(1))
			aw := awList.Items[0]
			Expect(aw.Spec.Type).To(Equal("art"))
			Expect(aw.Spec.SrcSecretRef.Name).To(Equal(""))
			Expect(aw.Spec.DstSecretRef.Name).To(Equal(""))
			Expect(aw.Spec.Parameters).To(ConsistOf([]arcv1alpha1.ArtifactWorkflowParameter{
				{
					Name:  "srcType",
					Value: "no-secret",
				},
				{
					Name:  "srcRemoteURL",
					Value: "no-secret",
				},
				{
					Name:  "dstType",
					Value: "no-secret",
				},
				{
					Name:  "dstRemoteURL",
					Value: "no-secret",
				},
				{
					Name:  "srcSecret",
					Value: "false",
				},
				{
					Name:  "dstSecret",
					Value: "false",
				},
				{
					Name:  "specKey",
					Value: "value",
				},
			}))
		})

		It("should update the artifact status if the workflow is updated", func() {
			createEndpoints("src-1", "dst-1")
			// Create test Order with multiple artifacts, no defaults
			order := &arcv1alpha1.Order{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-order-status-updates",
					Namespace: ns.Name,
				},
				Spec: arcv1alpha1.OrderSpec{
					Artifacts: []arcv1alpha1.OrderArtifact{
						{
							Type:   at1.Name,
							SrcRef: corev1.LocalObjectReference{Name: "src-1"},
							DstRef: corev1.LocalObjectReference{Name: "dst-1"},
							Spec:   runtime.RawExtension{Raw: []byte(`{"key":"value-1"}`)},
						},
					},
				},
			}
			Expect(k8sClient.Create(ctx, order)).To(Succeed())

			// Verify artifact workflows were created
			awList := &arcv1alpha1.ArtifactWorkflowList{}
			Eventually(func() int {
				err := k8sClient.List(ctx, awList, client.InNamespace(ns.Name))
				if err != nil {
					return 0
				}
				return len(awList.Items)
			}).Should(Equal(1))

			aw := &awList.Items[0]

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

			Eventually(func() arcv1alpha1.WorkflowPhase {
				Expect(k8sClient.Get(ctx, namespacedName(order.Namespace, order.Name), order)).To(Succeed())
				shas := slices.Collect(maps.Keys(order.Status.ArtifactWorkflows))
				Expect(shas).To(HaveLen(1))
				return order.Status.ArtifactWorkflows[shas[0]].Phase
			}).To(Equal(arcv1alpha1.WorkflowRunning))

		})

		It("should handle orders with duplicate parameters in artifact spec", func() {
			createEndpoints("src", "dst")
			// Create test Order with duplicate parameters in the spec
			// The JSON will create duplicate parameters when flattened
			order := &arcv1alpha1.Order{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-order-duplicate-params",
					Namespace: ns.Name,
				},
				Spec: arcv1alpha1.OrderSpec{
					Defaults: arcv1alpha1.OrderDefaults{
						SrcRef: corev1.LocalObjectReference{Name: "src"},
						DstRef: corev1.LocalObjectReference{Name: "dst"},
					},
					Artifacts: []arcv1alpha1.OrderArtifact{
						{
							Type: at1.Name,
							// Create a spec with potential duplicate parameters
							// after flattening
							Spec: runtime.RawExtension{Raw: []byte(`{"srcType":"override"}`)},
						},
					},
				},
			}
			Expect(k8sClient.Create(ctx, order)).To(Succeed())

			// The ArtifactWorkflow should not progress due to a validation error,
			// as srcType has already been added
			awList := &arcv1alpha1.ArtifactWorkflowList{}
			Eventually(func() int {
				err := k8sClient.List(ctx, awList, client.InNamespace(ns.Name))
				if err != nil {
					return 0
				}
				// ArtifactWorkflow should be created
				return len(awList.Items)
			}).Should(Equal(1))

			// But it should not create an Argo Workflow due to validation failure
			wfList := &wfv1alpha1.WorkflowList{}
			Consistently(func() int {
				err := k8sClient.List(ctx, wfList, client.InNamespace(ns.Name))
				if err != nil {
					return 0
				}
				return len(wfList.Items)
			}, "2s").Should(Equal(0))
		})
	})
})
