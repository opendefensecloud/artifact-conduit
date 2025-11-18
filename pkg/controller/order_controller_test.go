// Copyright 2025 BWI GmbH and Artifact Conduit contributors
// SPDX-License-Identifier: Apache-2.0

package controller

import (
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
		ctx = envtest.Context()
		ns  = SetupTest(ctx)
	)

	createEndpoints := func(names ...string) {
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
							Type:   "art-1",
							SrcRef: corev1.LocalObjectReference{Name: "src-1"},
							DstRef: corev1.LocalObjectReference{Name: "dst-1"},
							Spec:   runtime.RawExtension{Raw: []byte(`{"key":"value-1"}`)},
						},
						{
							Type:   "art-2",
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
				Expect(aw.Spec.Type).To(Or(Equal("art-1"), Equal("art-2")))
				suffix := "2"
				if aw.Spec.Type == "art-1" {
					suffix = "1"
				}
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
						Name:  "specKey",
						Value: "value-" + suffix,
					},
				}))
			}

			// Verify contents of secrets created by the controller
			secrets := &corev1.SecretList{}
			Eventually(func() int {
				err := k8sClient.List(ctx, secrets, client.InNamespace(ns.Name), client.MatchingLabels(secretLabels))
				if err != nil {
					return 0
				}
				return len(secrets.Items)
			}).Should(Equal(2))

			data1 := `{"type":"art-1","src":{"type":"src-1","remoteURL":"src-1","auth":{"testkey":"src-1"}},"dst":{"type":"dst-1","remoteURL":"dst-1","auth":{"testkey":"dst-1"}},"spec":{"key":"value-1"}}`
			data2 := `{"type":"art-2","src":{"type":"src-2","remoteURL":"src-2","auth":{"testkey":"src-2"}},"dst":{"type":"dst-2","remoteURL":"dst-2","auth":{"testkey":"dst-2"}},"spec":{"key":"value-2"}}`
			for _, secret := range secrets.Items {
				Expect(string(secret.Data["config.json"])).To(Or(Equal(data1), Equal(data2)))
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
							Type: "art-3",
							Spec: runtime.RawExtension{Raw: []byte(`{"key":"value"}`)},
						},
						{
							Type: "art-4",
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
				Expect(aw.Spec.Type).To(Or(Equal("art-3"), Equal("art-4")))
				Expect(aw.Spec.Parameters).To(ConsistOf([]arcv1alpha1.ArtifactWorkflowParameter{
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
							Type: "art-only-defaults",
							// Uses defaults for both refs
							Spec: runtime.RawExtension{Raw: []byte(`{"key":"value"}`)},
						},
						{
							Type: "art-mixed",
							// Specifies src, uses default dst
							SrcRef: corev1.LocalObjectReference{Name: "src-1"},
							Spec:   runtime.RawExtension{Raw: []byte(`{"key":"value"}`)},
						},
						{
							Type: "art-specified",
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
				case "art-only-defaults":
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
				case "art-mixed":
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
				case "art-specified":
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
						{Type: "art-1", SrcRef: corev1.LocalObjectReference{Name: "src-1"}, DstRef: corev1.LocalObjectReference{Name: "dst-1"}},
						{Type: "art-2", SrcRef: corev1.LocalObjectReference{Name: "src-2"}, DstRef: corev1.LocalObjectReference{Name: "dst-2"}},
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
						{Type: "art-1", SrcRef: corev1.LocalObjectReference{Name: "src-1"}, DstRef: corev1.LocalObjectReference{Name: "dst-1"}},
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
					Type:   "art-2",
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
	})
})
