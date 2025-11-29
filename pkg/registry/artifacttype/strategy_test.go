// Copyright 2025 BWI GmbH and Artifact Conduit contributors
// SPDX-License-Identifier: Apache-2.0

package artifacttype_test

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.opendefense.cloud/arc/api/arc"
	"go.opendefense.cloud/arc/pkg/registry/artifacttype"
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

var _ = Describe("ArtifactType Strategy", func() {
	var (
		ctx context.Context
	)

	BeforeEach(func() {
		ctx = context.Background()
	})

	Describe("NewStrategy", func() {
		It("should create a new strategy", func() {
			s := artifacttype.NewStrategy(testObjectTyper{})
			Expect(s).NotTo(BeNil())
		})
	})

	Describe("GetAttrs", func() {
		It("should return labels and fields for an ArtifactType", func() {
			at := &arc.ArtifactType{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-artifacttype",
					Namespace: "default",
					Labels: map[string]string{
						"app": "test",
					},
				},
			}

			lbls, flds, err := artifacttype.GetAttrs(at)
			Expect(err).NotTo(HaveOccurred())
			Expect(lbls).To(HaveKeyWithValue("app", "test"))
			Expect(flds.Has("metadata.name")).To(BeTrue())
			Expect(flds.Get("metadata.name")).To(Equal("test-artifacttype"))
		})

		It("should return error for non-ArtifactType object", func() {
			notAnArtifactType := &arc.Order{
				ObjectMeta: metav1.ObjectMeta{
					Name: "not-an-artifacttype",
				},
			}

			_, _, err := artifacttype.GetAttrs(notAnArtifactType)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("not an ArtifactType"))
		})
	})

	Describe("MatchArtifactType", func() {
		It("should create a selection predicate", func() {
			labelSelector := labels.Everything()
			fieldSelector := fields.Everything()

			predicate := artifacttype.MatchArtifactType(labelSelector, fieldSelector)
			Expect(predicate.Label).To(Equal(labelSelector))
			Expect(predicate.Field).To(Equal(fieldSelector))
			Expect(predicate.GetAttrs).NotTo(BeNil())
		})
	})

	Describe("SelectableFields", func() {
		It("should return selectable fields for an ArtifactType", func() {
			at := &arc.ArtifactType{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-artifacttype",
					Namespace: "default",
				},
			}

			flds := artifacttype.SelectableFields(at)
			Expect(flds.Has("metadata.name")).To(BeTrue())
			Expect(flds.Has("metadata.namespace")).To(BeTrue())
		})
	})

	Describe("NamespaceScoped", func() {
		It("should return true", func() {
			strategy := artifacttype.NewStrategy(testObjectTyper{})
			Expect(strategy.NamespaceScoped()).To(BeTrue())
		})
	})

	Describe("PrepareForCreate", func() {
		It("should not modify the object", func() {
			at := &arc.ArtifactType{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-artifacttype",
					Namespace: "default",
				},
				Spec: arc.ArtifactTypeSpec{
					WorkflowTemplateRef: arc.ArtifactTypeTemplateRef{
						Name: "test-template",
					},
				},
			}

			original := at.DeepCopy()
			strategy := artifacttype.NewStrategy(testObjectTyper{})
			strategy.PrepareForCreate(ctx, at)
			Expect(at).To(Equal(original))
		})
	})

	Describe("PrepareForUpdate", func() {
		It("should not modify the objects", func() {
			oldArtifactType := &arc.ArtifactType{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-artifacttype",
					Namespace: "default",
				},
				Spec: arc.ArtifactTypeSpec{
					WorkflowTemplateRef: arc.ArtifactTypeTemplateRef{
						Name: "old-template",
					},
				},
			}

			newArtifactType := &arc.ArtifactType{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-artifacttype",
					Namespace: "default",
				},
				Spec: arc.ArtifactTypeSpec{
					WorkflowTemplateRef: arc.ArtifactTypeTemplateRef{
						Name: "new-template",
					},
				},
			}

			originalNew := newArtifactType.DeepCopy()
			originalOld := oldArtifactType.DeepCopy()

			strategy := artifacttype.NewStrategy(testObjectTyper{})
			strategy.PrepareForUpdate(ctx, newArtifactType, oldArtifactType)
			Expect(newArtifactType).To(Equal(originalNew))
			Expect(oldArtifactType).To(Equal(originalOld))
		})
	})

	Describe("Validate", func() {
		It("should return empty error list", func() {
			at := &arc.ArtifactType{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-artifacttype",
					Namespace: "default",
				},
			}

			strategy := artifacttype.NewStrategy(testObjectTyper{})
			errs := strategy.Validate(ctx, at)
			Expect(errs).To(BeEmpty())
		})
	})

	Describe("WarningsOnCreate", func() {
		It("should return nil", func() {
			at := &arc.ArtifactType{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-artifacttype",
					Namespace: "default",
				},
			}

			strategy := artifacttype.NewStrategy(testObjectTyper{})
			warnings := strategy.WarningsOnCreate(ctx, at)
			Expect(warnings).To(BeNil())
		})
	})

	Describe("AllowCreateOnUpdate", func() {
		It("should return false", func() {
			strategy := artifacttype.NewStrategy(testObjectTyper{})
			Expect(strategy.AllowCreateOnUpdate()).To(BeFalse())
		})
	})

	Describe("AllowUnconditionalUpdate", func() {
		It("should return false", func() {
			strategy := artifacttype.NewStrategy(testObjectTyper{})
			Expect(strategy.AllowUnconditionalUpdate()).To(BeFalse())
		})
	})

	Describe("Canonicalize", func() {
		It("should not modify the object", func() {
			at := &arc.ArtifactType{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-artifacttype",
					Namespace: "default",
				},
			}

			original := at.DeepCopy()
			strategy := artifacttype.NewStrategy(testObjectTyper{})
			strategy.Canonicalize(at)
			Expect(at).To(Equal(original))
		})
	})

	Describe("ValidateUpdate", func() {
		It("should return empty error list", func() {
			oldArtifactType := &arc.ArtifactType{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-artifacttype",
					Namespace: "default",
				},
			}

			newArtifactType := oldArtifactType.DeepCopy()
			newArtifactType.Labels = map[string]string{"updated": "true"}

			strategy := artifacttype.NewStrategy(testObjectTyper{})
			errs := strategy.ValidateUpdate(ctx, newArtifactType, oldArtifactType)
			Expect(errs).To(BeEmpty())
		})
	})

	Describe("WarningsOnUpdate", func() {
		It("should return nil", func() {
			oldArtifactType := &arc.ArtifactType{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-artifacttype",
					Namespace: "default",
				},
			}

			newArtifactType := oldArtifactType.DeepCopy()

			strategy := artifacttype.NewStrategy(testObjectTyper{})
			warnings := strategy.WarningsOnUpdate(ctx, newArtifactType, oldArtifactType)
			Expect(warnings).To(BeNil())
		})
	})
})
