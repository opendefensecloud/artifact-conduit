// Copyright 2025 BWI GmbH and Artifact Conduit contributors
// SPDX-License-Identifier: Apache-2.0

package arc_test

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.opendefense.cloud/arc/api/arc"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/validation/field"
)

var _ = Describe("Order Strategy", func() {
	var (
		ctx context.Context
	)

	BeforeEach(func() {
		ctx = context.Background()
	})

	Describe("Validate", func() {
		Context("when validating artifacts without defaults", func() {
			It("should accept Order with all required artifact fields", func() {
				order := &arc.Order{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-order",
						Namespace: "default",
					},
					Spec: arc.OrderSpec{
						Artifacts: []arc.OrderArtifact{
							{
								Type: "container-image",
								SrcRef: corev1.LocalObjectReference{
									Name: "docker-endpoint",
								},
								DstRef: corev1.LocalObjectReference{
									Name: "registry-endpoint",
								},
							},
						},
					},
				}

				errs := order.Validate(ctx)
				Expect(errs).To(BeEmpty())
			})

			It("should reject Order with artifact missing type", func() {
				order := &arc.Order{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-order",
						Namespace: "default",
					},
					Spec: arc.OrderSpec{
						Artifacts: []arc.OrderArtifact{
							{
								SrcRef: corev1.LocalObjectReference{
									Name: "docker-endpoint",
								},
								DstRef: corev1.LocalObjectReference{
									Name: "registry-endpoint",
								},
							},
						},
					},
				}

				errs := order.Validate(ctx)
				Expect(errs).To(HaveLen(1))
				Expect(errs[0].Type).To(Equal(field.ErrorTypeRequired))
				Expect(errs[0].Field).To(Equal("spec.artifacts[0].type"))
			})

			It("should reject Order with artifact missing srcRef when no default src", func() {
				order := &arc.Order{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-order",
						Namespace: "default",
					},
					Spec: arc.OrderSpec{
						Artifacts: []arc.OrderArtifact{
							{
								Type: "container-image",
								DstRef: corev1.LocalObjectReference{
									Name: "registry-endpoint",
								},
							},
						},
					},
				}

				errs := order.Validate(ctx)
				Expect(errs).To(HaveLen(1))
				Expect(errs[0].Type).To(Equal(field.ErrorTypeRequired))
				Expect(errs[0].Field).To(Equal("spec.artifacts[0].srcRef"))
			})

			It("should reject Order with artifact missing dstRef when no default dst", func() {
				order := &arc.Order{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-order",
						Namespace: "default",
					},
					Spec: arc.OrderSpec{
						Artifacts: []arc.OrderArtifact{
							{
								Type: "container-image",
								SrcRef: corev1.LocalObjectReference{
									Name: "docker-endpoint",
								},
							},
						},
					},
				}

				errs := order.Validate(ctx)
				Expect(errs).To(HaveLen(1))
				Expect(errs[0].Type).To(Equal(field.ErrorTypeRequired))
				Expect(errs[0].Field).To(Equal("spec.artifacts[0].dstRef"))
			})

			It("should reject Order with artifact missing multiple required fields", func() {
				order := &arc.Order{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-order",
						Namespace: "default",
					},
					Spec: arc.OrderSpec{
						Artifacts: []arc.OrderArtifact{
							{
								// Missing type, srcRef, and dstRef
							},
						},
					},
				}

				errs := order.Validate(ctx)
				Expect(errs).To(HaveLen(3))

				errorFields := []string{}
				for _, err := range errs {
					Expect(err.Type).To(Equal(field.ErrorTypeRequired))
					errorFields = append(errorFields, err.Field)
				}
				Expect(errorFields).To(ConsistOf(
					"spec.artifacts[0].type",
					"spec.artifacts[0].srcRef",
					"spec.artifacts[0].dstRef",
				))
			})
		})

		Context("when validating artifacts with defaults", func() {
			It("should accept artifact without srcRef when default src is set", func() {
				order := &arc.Order{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-order",
						Namespace: "default",
					},
					Spec: arc.OrderSpec{
						Defaults: arc.OrderDefaults{
							SrcRef: corev1.LocalObjectReference{
								Name: "default-src-endpoint",
							},
						},
						Artifacts: []arc.OrderArtifact{
							{
								Type: "container-image",
								DstRef: corev1.LocalObjectReference{
									Name: "registry-endpoint",
								},
							},
						},
					},
				}

				errs := order.Validate(ctx)
				Expect(errs).To(BeEmpty())
			})

			It("should accept artifact without dstRef when default dst is set", func() {
				order := &arc.Order{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-order",
						Namespace: "default",
					},
					Spec: arc.OrderSpec{
						Defaults: arc.OrderDefaults{
							DstRef: corev1.LocalObjectReference{
								Name: "default-dst-endpoint",
							},
						},
						Artifacts: []arc.OrderArtifact{
							{
								Type: "container-image",
								SrcRef: corev1.LocalObjectReference{
									Name: "docker-endpoint",
								},
							},
						},
					},
				}

				errs := order.Validate(ctx)
				Expect(errs).To(BeEmpty())
			})

			It("should accept artifact without srcRef and dstRef when both defaults are set", func() {
				order := &arc.Order{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-order",
						Namespace: "default",
					},
					Spec: arc.OrderSpec{
						Defaults: arc.OrderDefaults{
							SrcRef: corev1.LocalObjectReference{
								Name: "default-src-endpoint",
							},
							DstRef: corev1.LocalObjectReference{
								Name: "default-dst-endpoint",
							},
						},
						Artifacts: []arc.OrderArtifact{
							{
								Type: "container-image",
							},
						},
					},
				}

				errs := order.Validate(ctx)
				Expect(errs).To(BeEmpty())
			})

			It("should accept artifact with explicit srcRef overriding default", func() {
				order := &arc.Order{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-order",
						Namespace: "default",
					},
					Spec: arc.OrderSpec{
						Defaults: arc.OrderDefaults{
							SrcRef: corev1.LocalObjectReference{
								Name: "default-src-endpoint",
							},
							DstRef: corev1.LocalObjectReference{
								Name: "default-dst-endpoint",
							},
						},
						Artifacts: []arc.OrderArtifact{
							{
								Type: "container-image",
								SrcRef: corev1.LocalObjectReference{
									Name: "specific-src-endpoint",
								},
							},
						},
					},
				}

				errs := order.Validate(ctx)
				Expect(errs).To(BeEmpty())
			})
		})

		Context("when validating multiple artifacts", func() {
			It("should accept Order with multiple valid artifacts", func() {
				order := &arc.Order{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-order",
						Namespace: "default",
					},
					Spec: arc.OrderSpec{
						Artifacts: []arc.OrderArtifact{
							{
								Type: "container-image",
								SrcRef: corev1.LocalObjectReference{
									Name: "docker-endpoint",
								},
								DstRef: corev1.LocalObjectReference{
									Name: "registry-endpoint",
								},
							},
							{
								Type: "helm-chart",
								SrcRef: corev1.LocalObjectReference{
									Name: "helm-repo",
								},
								DstRef: corev1.LocalObjectReference{
									Name: "helm-registry",
								},
							},
							{
								Type: "sbom",
								SrcRef: corev1.LocalObjectReference{
									Name: "scanner",
								},
								DstRef: corev1.LocalObjectReference{
									Name: "storage",
								},
							},
						},
					},
				}

				errs := order.Validate(ctx)
				Expect(errs).To(BeEmpty())
			})

			It("should report errors for all invalid artifacts", func() {
				order := &arc.Order{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-order",
						Namespace: "default",
					},
					Spec: arc.OrderSpec{
						Artifacts: []arc.OrderArtifact{
							{
								// Missing type and references
							},
							{
								Type: "container-image",
								// Missing srcRef and dstRef
							},
							{
								SrcRef: corev1.LocalObjectReference{
									Name: "src",
								},
								// Missing type
							},
						},
					},
				}

				errs := order.Validate(ctx)
				Expect(errs).ToNot(BeEmpty())

				// Verify errors for each artifact
				typeErrors := 0
				srcErrors := 0
				dstErrors := 0
				for _, err := range errs {
					if err.Type == field.ErrorTypeRequired {
						switch err.Field {
						case "spec.artifacts[0].type":
							typeErrors++
						case "spec.artifacts[0].srcRef":
							srcErrors++
						case "spec.artifacts[0].dstRef":
							dstErrors++
						case "spec.artifacts[1].srcRef":
							srcErrors++
						case "spec.artifacts[1].dstRef":
							dstErrors++
						case "spec.artifacts[2].type":
							typeErrors++
						}
					}
				}
				Expect(typeErrors).To(Equal(2))
				Expect(srcErrors).To(Equal(2))
				Expect(dstErrors).To(Equal(2))
			})
		})

		Context("when validating empty Order", func() {
			It("should accept Order with no artifacts", func() {
				order := &arc.Order{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-order",
						Namespace: "default",
					},
					Spec: arc.OrderSpec{
						Artifacts: []arc.OrderArtifact{},
					},
				}

				errs := order.Validate(ctx)
				Expect(errs).To(BeEmpty())
			})

			It("should accept Order with nil artifacts", func() {
				order := &arc.Order{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-order",
						Namespace: "default",
					},
					Spec: arc.OrderSpec{},
				}

				errs := order.Validate(ctx)
				Expect(errs).To(BeEmpty())
			})
		})
	})

	Describe("ValidateUpdate", func() {
		Context("when updating Order", func() {
			It("should use same validation rules as Validate", func() {
				oldOrder := &arc.Order{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-order",
						Namespace: "default",
					},
					Spec: arc.OrderSpec{
						Artifacts: []arc.OrderArtifact{
							{
								Type: "container-image",
								SrcRef: corev1.LocalObjectReference{
									Name: "docker-endpoint",
								},
								DstRef: corev1.LocalObjectReference{
									Name: "registry-endpoint",
								},
							},
						},
					},
				}

				newOrder := &arc.Order{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-order",
						Namespace: "default",
					},
					Spec: arc.OrderSpec{
						Artifacts: []arc.OrderArtifact{
							{
								// Missing required fields
								Type: "container-image",
							},
						},
					},
				}

				errs := newOrder.ValidateUpdate(ctx, oldOrder)
				Expect(errs).To(HaveLen(2))
				for _, err := range errs {
					Expect(err.Type).To(Equal(field.ErrorTypeRequired))
				}
			})

			It("should accept valid update", func() {
				oldOrder := &arc.Order{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-order",
						Namespace: "default",
					},
					Spec: arc.OrderSpec{
						Artifacts: []arc.OrderArtifact{
							{
								Type: "container-image",
								SrcRef: corev1.LocalObjectReference{
									Name: "docker-endpoint",
								},
								DstRef: corev1.LocalObjectReference{
									Name: "registry-endpoint",
								},
							},
						},
					},
				}

				newOrder := &arc.Order{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-order",
						Namespace: "default",
					},
					Spec: arc.OrderSpec{
						Artifacts: []arc.OrderArtifact{
							{
								Type: "container-image",
								SrcRef: corev1.LocalObjectReference{
									Name: "docker-endpoint",
								},
								DstRef: corev1.LocalObjectReference{
									Name: "registry-endpoint",
								},
								Spec: runtime.RawExtension{
									Raw: []byte(`{"timeout": 300}`),
								},
							},
						},
					},
				}

				errs := newOrder.ValidateUpdate(ctx, oldOrder)
				Expect(errs).To(BeEmpty())
			})

			It("should reject update with invalid artifact", func() {
				oldOrder := &arc.Order{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-order",
						Namespace: "default",
					},
					Spec: arc.OrderSpec{
						Artifacts: []arc.OrderArtifact{
							{
								Type: "container-image",
								SrcRef: corev1.LocalObjectReference{
									Name: "docker-endpoint",
								},
								DstRef: corev1.LocalObjectReference{
									Name: "registry-endpoint",
								},
							},
						},
					},
				}

				newOrder := &arc.Order{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-order",
						Namespace: "default",
					},
					Spec: arc.OrderSpec{
						Artifacts: []arc.OrderArtifact{
							{
								// Missing type
								SrcRef: corev1.LocalObjectReference{
									Name: "docker-endpoint",
								},
								DstRef: corev1.LocalObjectReference{
									Name: "registry-endpoint",
								},
							},
						},
					},
				}

				errs := newOrder.ValidateUpdate(ctx, oldOrder)
				Expect(errs).To(HaveLen(1))
				Expect(errs[0].Type).To(Equal(field.ErrorTypeRequired))
				Expect(errs[0].Field).To(Equal("spec.artifacts[0].type"))
			})

			It("should validate updated artifact list", func() {
				oldOrder := &arc.Order{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-order",
						Namespace: "default",
					},
					Spec: arc.OrderSpec{
						Artifacts: []arc.OrderArtifact{
							{
								Type: "container-image",
								SrcRef: corev1.LocalObjectReference{
									Name: "docker-endpoint",
								},
								DstRef: corev1.LocalObjectReference{
									Name: "registry-endpoint",
								},
							},
						},
					},
				}

				newOrder := &arc.Order{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-order",
						Namespace: "default",
					},
					Spec: arc.OrderSpec{
						Artifacts: []arc.OrderArtifact{
							{
								Type: "container-image",
								SrcRef: corev1.LocalObjectReference{
									Name: "docker-endpoint",
								},
								DstRef: corev1.LocalObjectReference{
									Name: "registry-endpoint",
								},
							},
							{
								// New artifact missing required fields
								Type: "helm-chart",
							},
						},
					},
				}

				errs := newOrder.ValidateUpdate(ctx, oldOrder)
				Expect(errs).ToNot(BeEmpty())

				// Should have errors for the new artifact
				for _, err := range errs {
					Expect(err.Field).To(ContainSubstring("spec.artifacts[1]"))
				}
			})
		})
	})
})
