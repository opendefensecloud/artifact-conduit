// Copyright 2025 BWI GmbH and Artifact Conduit contributors
// SPDX-License-Identifier: Apache-2.0

package order_test

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.opendefense.cloud/arc/api/arc"
	"go.opendefense.cloud/arc/pkg/registry/order"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/structured-merge-diff/v6/fieldpath"
)

type testObjectTyper struct{}

func (testObjectTyper) ObjectKinds(runtime.Object) ([]schema.GroupVersionKind, bool, error) {
	return nil, false, nil
}

func (testObjectTyper) Recognizes(gvk schema.GroupVersionKind) bool {
	return false
}

var _ = Describe("Order Strategy", func() {
	var (
		ctx context.Context
	)

	BeforeEach(func() {
		ctx = context.Background()
	})

	Describe("NewStrategy", func() {
		It("should create a new strategy", func() {
			s := order.NewStrategy(testObjectTyper{})
			Expect(s).NotTo(BeNil())
		})
	})

	Describe("GetAttrs", func() {
		It("should return labels and fields for an Order", func() {
			o := &arc.Order{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-order",
					Namespace: "default",
					Labels: map[string]string{
						"app": "test",
					},
				},
			}

			lbls, flds, err := order.GetAttrs(o)
			Expect(err).NotTo(HaveOccurred())
			Expect(lbls).To(HaveKeyWithValue("app", "test"))
			Expect(flds.Has("metadata.name")).To(BeTrue())
			Expect(flds.Get("metadata.name")).To(Equal("test-order"))
		})

		It("should return error for non-Order object", func() {
			notAnOrder := &arc.Endpoint{
				ObjectMeta: metav1.ObjectMeta{
					Name: "not-an-order",
				},
			}

			_, _, err := order.GetAttrs(notAnOrder)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("not an Order"))
		})
	})

	Describe("MatchOrder", func() {
		It("should create a selection predicate", func() {
			labelSelector := labels.Everything()
			fieldSelector := fields.Everything()

			predicate := order.MatchOrder(labelSelector, fieldSelector)
			Expect(predicate.Label).To(Equal(labelSelector))
			Expect(predicate.Field).To(Equal(fieldSelector))
			Expect(predicate.GetAttrs).NotTo(BeNil())
		})
	})

	Describe("SelectableFields", func() {
		It("should return selectable fields for an Order", func() {
			o := &arc.Order{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-order",
					Namespace: "default",
				},
			}

			flds := order.SelectableFields(o)
			Expect(flds.Has("metadata.name")).To(BeTrue())
			Expect(flds.Has("metadata.namespace")).To(BeTrue())
		})
	})

	Describe("NamespaceScoped", func() {
		It("should return true", func() {
			strategy := order.NewStrategy(testObjectTyper{})
			Expect(strategy.NamespaceScoped()).To(BeTrue())
		})
	})

	Describe("PrepareForCreate", func() {
		It("should not modify the object", func() {
			o := &arc.Order{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-order",
					Namespace: "default",
				},
				Spec: arc.OrderSpec{
					Artifacts: []arc.OrderArtifact{
						{
							Type: "test-type",
						},
					},
				},
			}

			original := o.DeepCopy()
			strategy := order.NewStrategy(testObjectTyper{})
			strategy.PrepareForCreate(ctx, o)
			Expect(o).To(Equal(original))
		})
	})

	Describe("PrepareForUpdate", func() {
		It("should not modify the objects", func() {
			oldOrder := &arc.Order{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-order",
					Namespace: "default",
				},
				Spec: arc.OrderSpec{
					Artifacts: []arc.OrderArtifact{
						{Type: "test-type"},
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
						{Type: "test-type"},
						{Type: "test-type-2"},
					},
				},
			}

			originalNew := newOrder.DeepCopy()
			originalOld := oldOrder.DeepCopy()

			strategy := order.NewStrategy(testObjectTyper{})
			strategy.PrepareForUpdate(ctx, newOrder, oldOrder)
			Expect(newOrder).To(Equal(originalNew))
			Expect(oldOrder).To(Equal(originalOld))
		})
	})

	Describe("Validate", func() {
		It("should return empty error list", func() {
			o := &arc.Order{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-order",
					Namespace: "default",
				},
			}

			strategy := order.NewStrategy(testObjectTyper{})
			errs := strategy.Validate(ctx, o)
			Expect(errs).To(BeEmpty())
		})
	})

	Describe("WarningsOnCreate", func() {
		It("should return nil", func() {
			o := &arc.Order{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-order",
					Namespace: "default",
				},
			}

			strategy := order.NewStrategy(testObjectTyper{})
			warnings := strategy.WarningsOnCreate(ctx, o)
			Expect(warnings).To(BeNil())
		})
	})

	Describe("AllowCreateOnUpdate", func() {
		It("should return false", func() {
			strategy := order.NewStrategy(testObjectTyper{})
			Expect(strategy.AllowCreateOnUpdate()).To(BeFalse())
		})
	})

	Describe("AllowUnconditionalUpdate", func() {
		It("should return false", func() {
			strategy := order.NewStrategy(testObjectTyper{})
			Expect(strategy.AllowUnconditionalUpdate()).To(BeFalse())
		})
	})

	Describe("Canonicalize", func() {
		It("should not modify the object", func() {
			o := &arc.Order{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-order",
					Namespace: "default",
				},
			}

			original := o.DeepCopy()
			strategy := order.NewStrategy(testObjectTyper{})
			strategy.Canonicalize(o)
			Expect(o).To(Equal(original))
		})
	})

	Describe("ValidateUpdate", func() {
		It("should return empty error list", func() {
			oldOrder := &arc.Order{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-order",
					Namespace: "default",
				},
			}

			newOrder := oldOrder.DeepCopy()
			newOrder.Labels = map[string]string{"updated": "true"}

			strategy := order.NewStrategy(testObjectTyper{})
			errs := strategy.ValidateUpdate(ctx, newOrder, oldOrder)
			Expect(errs).To(BeEmpty())
		})
	})

	Describe("WarningsOnUpdate", func() {
		It("should return nil", func() {
			oldOrder := &arc.Order{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-order",
					Namespace: "default",
				},
			}

			newOrder := oldOrder.DeepCopy()

			strategy := order.NewStrategy(testObjectTyper{})
			warnings := strategy.WarningsOnUpdate(ctx, newOrder, oldOrder)
			Expect(warnings).To(BeNil())
		})
	})
})

var _ = Describe("Order Status Strategy", func() {
	var (
		ctx context.Context
	)

	BeforeEach(func() {
		ctx = context.Background()
	})

	Describe("NewStatusStrategy", func() {
		It("should create a new status strategy", func() {
			s := order.NewStatusStrategy(testObjectTyper{})
			Expect(s).NotTo(BeNil())
		})
	})

	Describe("GetResetFields", func() {
		It("should return spec as a reset field", func() {
			statusStrategy := order.NewStatusStrategy(testObjectTyper{})
			resetFields := statusStrategy.GetResetFields()
			Expect(resetFields).To(HaveKey(fieldpath.APIVersion("arc.bwi.de/v1alpha1")))

			fieldSet := resetFields["arc.bwi.de/v1alpha1"]
			Expect(fieldSet).NotTo(BeNil())
			// The field set should contain "spec"
			Expect(fieldSet.Has(fieldpath.MakePathOrDie("spec"))).To(BeTrue())
		})
	})

	Describe("PrepareForUpdate", func() {
		It("should preserve spec from old object", func() {
			oldOrder := &arc.Order{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-order",
					Namespace: "default",
				},
				Spec: arc.OrderSpec{
					Artifacts: []arc.OrderArtifact{
						{Type: "test-type"},
					},
				},
				Status: arc.OrderStatus{
					Message: "Pending",
				},
			}

			newOrder := &arc.Order{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-order",
					Namespace: "default",
				},
				Spec: arc.OrderSpec{
					Artifacts: []arc.OrderArtifact{
						{Type: "different-type"},
					},
				},
				Status: arc.OrderStatus{
					Message: "Running",
				},
			}

			statusStrategy := order.NewStatusStrategy(testObjectTyper{})
			statusStrategy.PrepareForUpdate(ctx, newOrder, oldOrder)

			// Spec should be preserved from old object
			Expect(newOrder.Spec).To(Equal(oldOrder.Spec))
			// Status should remain as set in new object
			Expect(newOrder.Status.Message).To(Equal("Running"))
		})
	})

	Describe("ValidateUpdate", func() {
		It("should return empty error list", func() {
			oldOrder := &arc.Order{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-order",
					Namespace: "default",
				},
			}

			newOrder := oldOrder.DeepCopy()
			newOrder.Status.Message = "Completed"

			statusStrategy := order.NewStatusStrategy(testObjectTyper{})
			errs := statusStrategy.ValidateUpdate(ctx, newOrder, oldOrder)
			Expect(errs).To(BeEmpty())
		})
	})

	Describe("WarningsOnUpdate", func() {
		It("should return nil", func() {
			oldOrder := &arc.Order{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-order",
					Namespace: "default",
				},
			}

			newOrder := oldOrder.DeepCopy()

			statusStrategy := order.NewStatusStrategy(testObjectTyper{})
			warnings := statusStrategy.WarningsOnUpdate(ctx, newOrder, oldOrder)
			Expect(warnings).To(BeNil())
		})
	})
})
