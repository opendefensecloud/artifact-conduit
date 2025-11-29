// Copyright 2025 BWI GmbH and Artifact Conduit contributors
// SPDX-License-Identifier: Apache-2.0

package endpoint_test

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.opendefense.cloud/arc/api/arc"
	"go.opendefense.cloud/arc/pkg/registry/endpoint"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

type testObjectTyper struct{}

func (testObjectTyper) ObjectKinds(runtime.Object) ([]schema.GroupVersionKind, bool, error) {
	return nil, false, nil
}

func (testObjectTyper) Recognizes(gvk schema.GroupVersionKind) bool {
	return false
}

var _ = Describe("Endpoint Strategy", func() {
	var (
		ctx context.Context
	)

	BeforeEach(func() {
		ctx = context.Background()
	})

	Describe("NewStrategy", func() {
		It("should create a new strategy", func() {
			s := endpoint.NewStrategy(testObjectTyper{})
			Expect(s).NotTo(BeNil())
		})
	})

	Describe("GetAttrs", func() {
		It("should return labels and fields for an Endpoint", func() {
			e := &arc.Endpoint{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-endpoint",
					Namespace: "default",
					Labels: map[string]string{
						"app": "test",
					},
				},
			}

			lbls, flds, err := endpoint.GetAttrs(e)
			Expect(err).NotTo(HaveOccurred())
			Expect(lbls).To(HaveKeyWithValue("app", "test"))
			Expect(flds.Has("metadata.name")).To(BeTrue())
			Expect(flds.Get("metadata.name")).To(Equal("test-endpoint"))
		})

		It("should return error for non-Endpoint object", func() {
			notAnEndpoint := &arc.Order{
				ObjectMeta: metav1.ObjectMeta{
					Name: "not-an-endpoint",
				},
			}

			_, _, err := endpoint.GetAttrs(notAnEndpoint)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("not an Endpoint"))
		})
	})

	Describe("MatchEndpoint", func() {
		It("should create a selection predicate", func() {
			labelSelector := labels.Everything()
			fieldSelector := fields.Everything()

			predicate := endpoint.MatchEndpoint(labelSelector, fieldSelector)
			Expect(predicate.Label).To(Equal(labelSelector))
			Expect(predicate.Field).To(Equal(fieldSelector))
			Expect(predicate.GetAttrs).NotTo(BeNil())
		})
	})

	Describe("SelectableFields", func() {
		It("should return selectable fields for an Endpoint", func() {
			e := &arc.Endpoint{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-endpoint",
					Namespace: "default",
				},
			}

			flds := endpoint.SelectableFields(e)
			Expect(flds.Has("metadata.name")).To(BeTrue())
			Expect(flds.Has("metadata.namespace")).To(BeTrue())
		})
	})

	Describe("NamespaceScoped", func() {
		It("should return true", func() {
			strategy := endpoint.NewStrategy(testObjectTyper{})
			Expect(strategy.NamespaceScoped()).To(BeTrue())
		})
	})

	Describe("PrepareForCreate", func() {
		It("should not modify the object", func() {
			e := &arc.Endpoint{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-endpoint",
					Namespace: "default",
				},
				Spec: arc.EndpointSpec{
					Type:      "oci",
					RemoteURL: "https://registry.example.com",
				},
			}

			original := e.DeepCopy()
			strategy := endpoint.NewStrategy(testObjectTyper{})
			strategy.PrepareForCreate(ctx, e)
			Expect(e).To(Equal(original))
		})
	})

	Describe("PrepareForUpdate", func() {
		It("should not modify the objects", func() {
			oldEndpoint := &arc.Endpoint{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-endpoint",
					Namespace: "default",
				},
				Spec: arc.EndpointSpec{
					Type:      "oci",
					RemoteURL: "https://registry.example.com",
				},
			}

			newEndpoint := &arc.Endpoint{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-endpoint",
					Namespace: "default",
				},
				Spec: arc.EndpointSpec{
					Type:      "helm",
					RemoteURL: "https://charts.example.com",
				},
			}

			originalNew := newEndpoint.DeepCopy()
			originalOld := oldEndpoint.DeepCopy()

			strategy := endpoint.NewStrategy(testObjectTyper{})
			strategy.PrepareForUpdate(ctx, newEndpoint, oldEndpoint)
			Expect(newEndpoint).To(Equal(originalNew))
			Expect(oldEndpoint).To(Equal(originalOld))
		})
	})

	Describe("Validate", func() {
		It("should return empty error list", func() {
			e := &arc.Endpoint{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-endpoint",
					Namespace: "default",
				},
			}

			strategy := endpoint.NewStrategy(testObjectTyper{})
			errs := strategy.Validate(ctx, e)
			Expect(errs).To(BeEmpty())
		})
	})

	Describe("WarningsOnCreate", func() {
		It("should return nil", func() {
			e := &arc.Endpoint{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-endpoint",
					Namespace: "default",
				},
			}

			strategy := endpoint.NewStrategy(testObjectTyper{})
			warnings := strategy.WarningsOnCreate(ctx, e)
			Expect(warnings).To(BeNil())
		})
	})

	Describe("AllowCreateOnUpdate", func() {
		It("should return false", func() {
			strategy := endpoint.NewStrategy(testObjectTyper{})
			Expect(strategy.AllowCreateOnUpdate()).To(BeFalse())
		})
	})

	Describe("AllowUnconditionalUpdate", func() {
		It("should return false", func() {
			strategy := endpoint.NewStrategy(testObjectTyper{})
			Expect(strategy.AllowUnconditionalUpdate()).To(BeFalse())
		})
	})

	Describe("Canonicalize", func() {
		It("should not modify the object", func() {
			e := &arc.Endpoint{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-endpoint",
					Namespace: "default",
				},
			}

			original := e.DeepCopy()
			strategy := endpoint.NewStrategy(testObjectTyper{})
			strategy.Canonicalize(e)
			Expect(e).To(Equal(original))
		})
	})

	Describe("ValidateUpdate", func() {
		It("should return empty error list", func() {
			oldEndpoint := &arc.Endpoint{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-endpoint",
					Namespace: "default",
				},
			}

			newEndpoint := oldEndpoint.DeepCopy()
			newEndpoint.Labels = map[string]string{"updated": "true"}

			strategy := endpoint.NewStrategy(testObjectTyper{})
			errs := strategy.ValidateUpdate(ctx, newEndpoint, oldEndpoint)
			Expect(errs).To(BeEmpty())
		})
	})

	Describe("WarningsOnUpdate", func() {
		It("should return nil", func() {
			oldEndpoint := &arc.Endpoint{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-endpoint",
					Namespace: "default",
				},
			}

			newEndpoint := oldEndpoint.DeepCopy()

			strategy := endpoint.NewStrategy(testObjectTyper{})
			warnings := strategy.WarningsOnUpdate(ctx, newEndpoint, oldEndpoint)
			Expect(warnings).To(BeNil())
		})
	})
})
