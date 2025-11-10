// Copyright 2025 BWI GmbH and Artefact Conduit contributors
// SPDX-License-Identifier: Apache-2.0

package controller

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	arcv1alpha1 "gitlab.opencode.de/bwi/ace/artifact-conduit/api/arc/v1alpha1"
	"gitlab.opencode.de/bwi/ace/artifact-conduit/pkg/envtest"
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

	Context("when reconciling Orders", func() {
		It("should create fragments for an order with multiple artifacts and no defaults", func() {
			// Create test Order with multiple artifacts, no defaults
			order := &arcv1alpha1.Order{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-order-no-defaults",
					Namespace: ns.Name,
				},
				Spec: arcv1alpha1.OrderSpec{
					Artifacts: []arcv1alpha1.OrderArtifact{
						{
							Type:   "test-type-1",
							SrcRef: corev1.LocalObjectReference{Name: "src-1"},
							DstRef: corev1.LocalObjectReference{Name: "dst-1"},
							Spec:   runtime.RawExtension{Raw: []byte(`{"key":"value1"}`)},
						},
						{
							Type:   "test-type-2",
							SrcRef: corev1.LocalObjectReference{Name: "src-2"},
							DstRef: corev1.LocalObjectReference{Name: "dst-2"},
							Spec:   runtime.RawExtension{Raw: []byte(`{"key":"value2"}`)},
						},
					},
				},
			}
			Expect(k8sClient.Create(ctx, order)).To(Succeed())

			// Verify fragments were created
			fragmentList := &arcv1alpha1.FragmentList{}
			Eventually(func() int {
				err := k8sClient.List(ctx, fragmentList, client.InNamespace(ns.Name))
				if err != nil {
					return 0
				}
				return len(fragmentList.Items)
			}).Should(Equal(2))

			// Verify fragment contents
			for _, fragment := range fragmentList.Items {
				Expect(fragment.Spec.Type).To(Or(Equal("test-type-1"), Equal("test-type-2")))
				if fragment.Spec.Type == "test-type-1" {
					Expect(fragment.Spec.SrcRef.Name).To(Equal("src-1"))
					Expect(fragment.Spec.DstRef.Name).To(Equal("dst-1"))
				} else {
					Expect(fragment.Spec.SrcRef.Name).To(Equal("src-2"))
					Expect(fragment.Spec.DstRef.Name).To(Equal("dst-2"))
				}
			}
		})

		It("should create fragments for an order with multiple artifacts using defaults", func() {
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
							Type: "test-type-3",
							Spec: runtime.RawExtension{Raw: []byte(`{"key":"value1"}`)},
						},
						{
							Type: "test-type-4",
							Spec: runtime.RawExtension{Raw: []byte(`{"key":"value2"}`)},
						},
					},
				},
			}
			Expect(k8sClient.Create(ctx, order)).To(Succeed())

			// Verify fragments were created
			fragmentList := &arcv1alpha1.FragmentList{}
			Eventually(func() int {
				err := k8sClient.List(ctx, fragmentList, client.InNamespace(ns.Name))
				if err != nil {
					return 0
				}
				return len(fragmentList.Items)
			}).Should(Equal(2))

			// Verify fragment contents - should use default refs
			for _, fragment := range fragmentList.Items {
				Expect(fragment.Spec.Type).To(Or(Equal("test-type-3"), Equal("test-type-4")))
				Expect(fragment.Spec.SrcRef.Name).To(Equal("default-src"))
				Expect(fragment.Spec.DstRef.Name).To(Equal("default-dst"))
			}
		})

		It("should create fragments for an order with mixed default usage", func() {
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
							Type: "test-type-5",
							// Uses defaults for both refs
							Spec: runtime.RawExtension{Raw: []byte(`{"key":"value1"}`)},
						},
						{
							Type: "test-type-6",
							// Specifies src, uses default dst
							SrcRef: corev1.LocalObjectReference{Name: "custom-src"},
							Spec:   runtime.RawExtension{Raw: []byte(`{"key":"value2"}`)},
						},
						{
							Type: "test-type-7",
							// Specifies both refs
							SrcRef: corev1.LocalObjectReference{Name: "custom-src-2"},
							DstRef: corev1.LocalObjectReference{Name: "custom-dst-2"},
							Spec:   runtime.RawExtension{Raw: []byte(`{"key":"value3"}`)},
						},
					},
				},
			}
			Expect(k8sClient.Create(ctx, order)).To(Succeed())

			// Verify fragments were created
			fragmentList := &arcv1alpha1.FragmentList{}
			Eventually(func() int {
				err := k8sClient.List(ctx, fragmentList, client.InNamespace(ns.Name))
				if err != nil {
					return 0
				}
				return len(fragmentList.Items)
			}).Should(Equal(3))

			// Verify fragment contents
			for _, fragment := range fragmentList.Items {
				switch fragment.Spec.Type {
				case "test-type-5":
					// Should use defaults for both
					Expect(fragment.Spec.SrcRef.Name).To(Equal("default-src"))
					Expect(fragment.Spec.DstRef.Name).To(Equal("default-dst"))
				case "test-type-6":
					// Should use custom src, default dst
					Expect(fragment.Spec.SrcRef.Name).To(Equal("custom-src"))
					Expect(fragment.Spec.DstRef.Name).To(Equal("default-dst"))
				case "test-type-7":
					// Should use custom refs for both
					Expect(fragment.Spec.SrcRef.Name).To(Equal("custom-src-2"))
					Expect(fragment.Spec.DstRef.Name).To(Equal("custom-dst-2"))
				}
			}

			// Verify status contains all fragments
			Eventually(func() int {
				if err := k8sClient.Get(ctx, client.ObjectKeyFromObject(order), order); err != nil {
					return 0
				}
				return len(order.Status.Fragments)
			}).Should(Equal(3))
		})

		It("should delete fragments when order is deleted", func() {
			order := &arcv1alpha1.Order{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-order-delete",
					Namespace: ns.Name,
				},
				Spec: arcv1alpha1.OrderSpec{
					Artifacts: []arcv1alpha1.OrderArtifact{
						{Type: "type-a", SrcRef: corev1.LocalObjectReference{Name: "src-a"}, DstRef: corev1.LocalObjectReference{Name: "dst-a"}},
						{Type: "type-b", SrcRef: corev1.LocalObjectReference{Name: "src-b"}, DstRef: corev1.LocalObjectReference{Name: "dst-b"}},
					},
				},
			}
			Expect(k8sClient.Create(ctx, order)).To(Succeed())
			fragmentList := &arcv1alpha1.FragmentList{}
			Eventually(func() int {
				_ = k8sClient.List(ctx, fragmentList, client.InNamespace(ns.Name))
				return len(fragmentList.Items)
			}).Should(Equal(2))
			// Delete order
			Expect(k8sClient.Delete(ctx, order)).To(Succeed())
			// Eventually all fragments should be gone
			Eventually(func() int {
				_ = k8sClient.List(ctx, fragmentList, client.InNamespace(ns.Name))
				return len(fragmentList.Items)
			}).Should(Equal(0))
		})

		It("should create a new fragment and update status when an artifact is added", func() {
			order := &arcv1alpha1.Order{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-order-add-artifact",
					Namespace: ns.Name,
				},
				Spec: arcv1alpha1.OrderSpec{
					Artifacts: []arcv1alpha1.OrderArtifact{
						{Type: "type-c", SrcRef: corev1.LocalObjectReference{Name: "src-c"}, DstRef: corev1.LocalObjectReference{Name: "dst-c"}},
					},
				},
			}
			Expect(k8sClient.Create(ctx, order)).To(Succeed())
			fragmentList := &arcv1alpha1.FragmentList{}
			Eventually(func() int {
				_ = k8sClient.List(ctx, fragmentList, client.InNamespace(ns.Name))
				return len(fragmentList.Items)
			}).Should(Equal(1))
			// Add a new artifact
			Expect(k8sClient.Get(ctx, client.ObjectKeyFromObject(order), order)).To(Succeed()) // fetch reconciled updates
			order.Spec.Artifacts = append(order.Spec.Artifacts, arcv1alpha1.OrderArtifact{
				Type:   "type-d",
				SrcRef: corev1.LocalObjectReference{Name: "src-d"},
				DstRef: corev1.LocalObjectReference{Name: "dst-d"},
			})
			Expect(k8sClient.Update(ctx, order)).To(Succeed())
			// Eventually two fragments should exist
			Eventually(func() int {
				_ = k8sClient.List(ctx, fragmentList, client.InNamespace(ns.Name))
				return len(fragmentList.Items)
			}).Should(Equal(2))
			// Status should be updated
			Eventually(func() int {
				_ = k8sClient.Get(ctx, client.ObjectKeyFromObject(order), order)
				return len(order.Status.Fragments)
			}).Should(Equal(2))
		})

		It("should delete a fragment and update status when an artifact is removed", func() {
			order := &arcv1alpha1.Order{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-order-remove-artifact",
					Namespace: ns.Name,
				},
				Spec: arcv1alpha1.OrderSpec{
					Artifacts: []arcv1alpha1.OrderArtifact{
						{Type: "type-e", SrcRef: corev1.LocalObjectReference{Name: "src-e"}, DstRef: corev1.LocalObjectReference{Name: "dst-e"}},
						{Type: "type-f", SrcRef: corev1.LocalObjectReference{Name: "src-f"}, DstRef: corev1.LocalObjectReference{Name: "dst-f"}},
					},
				},
			}
			Expect(k8sClient.Create(ctx, order)).To(Succeed())
			fragmentList := &arcv1alpha1.FragmentList{}
			Eventually(func() int {
				_ = k8sClient.List(ctx, fragmentList, client.InNamespace(ns.Name))
				return len(fragmentList.Items)
			}).Should(Equal(2))
			// Remove one artifact
			order.Spec.Artifacts = order.Spec.Artifacts[:1]
			Expect(k8sClient.Update(ctx, order)).To(Succeed())
			// Eventually only one fragment should exist
			Eventually(func() int {
				_ = k8sClient.List(ctx, fragmentList, client.InNamespace(ns.Name))
				return len(fragmentList.Items)
			}).Should(Equal(1))
			// Status should be updated
			Eventually(func() int {
				_ = k8sClient.Get(ctx, client.ObjectKeyFromObject(order), order)
				return len(order.Status.Fragments)
			}).Should(Equal(1))
		})
	})
})
