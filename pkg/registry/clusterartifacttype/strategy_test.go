// Copyright 2025 BWI GmbH and Artifact Conduit contributors
// SPDX-License-Identifier: Apache-2.0

package clusterartifacttype_test

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.opendefense.cloud/arc/api/arc"
	"go.opendefense.cloud/arc/pkg/registry/clusterartifacttype"
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

var _ = Describe("ClusterArtifactType Strategy", func() {
	var (
		ctx context.Context
	)

	BeforeEach(func() {
		ctx = context.Background()
	})

	Describe("NewStrategy", func() {
		It("should create a new strategy", func() {
			s := clusterartifacttype.NewStrategy(testObjectTyper{})
			Expect(s).NotTo(BeNil())
		})
	})

	Describe("GetAttrs", func() {
		It("should return labels and fields for a ClusterArtifactType", func() {
			cat := &arc.ClusterArtifactType{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-clusterartifacttype",
					Labels: map[string]string{
						"app": "test",
					},
				},
			}

			lbls, flds, err := clusterartifacttype.GetAttrs(cat)
			Expect(err).NotTo(HaveOccurred())
			Expect(lbls).To(HaveKeyWithValue("app", "test"))
			Expect(flds.Has("metadata.name")).To(BeTrue())
			Expect(flds.Get("metadata.name")).To(Equal("test-clusterartifacttype"))
		})

		It("should return error for non-ClusterArtifactType object", func() {
			notAClusterArtifactType := &arc.Order{
				ObjectMeta: metav1.ObjectMeta{
					Name: "not-a-clusterartifacttype",
				},
			}

			_, _, err := clusterartifacttype.GetAttrs(notAClusterArtifactType)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("not an ArtifactType"))
		})
	})

	Describe("MatchArtifactType", func() {
		It("should create a selection predicate", func() {
			labelSelector := labels.Everything()
			fieldSelector := fields.Everything()

			predicate := clusterartifacttype.MatchArtifactType(labelSelector, fieldSelector)
			Expect(predicate.Label).To(Equal(labelSelector))
			Expect(predicate.Field).To(Equal(fieldSelector))
			Expect(predicate.GetAttrs).NotTo(BeNil())
		})
	})

	Describe("SelectableFields", func() {
		It("should return selectable fields for a ClusterArtifactType", func() {
			cat := &arc.ClusterArtifactType{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-clusterartifacttype",
				},
			}

			flds := clusterartifacttype.SelectableFields(cat)
			Expect(flds.Has("metadata.name")).To(BeTrue())
		})
	})

	Describe("NamespaceScoped", func() {
		It("should return false (cluster-scoped)", func() {
			strategy := clusterartifacttype.NewStrategy(testObjectTyper{})
			Expect(strategy.NamespaceScoped()).To(BeFalse())
		})
	})

	Describe("PrepareForCreate", func() {
		It("should not modify the object", func() {
			cat := &arc.ClusterArtifactType{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-clusterartifacttype",
				},
				Spec: arc.ArtifactTypeSpec{
					WorkflowTemplateRef: arc.ArtifactTypeTemplateRef{
						Name: "test-template",
					},
				},
			}

			original := cat.DeepCopy()
			strategy := clusterartifacttype.NewStrategy(testObjectTyper{})
			strategy.PrepareForCreate(ctx, cat)
			Expect(cat).To(Equal(original))
		})
	})

	Describe("PrepareForUpdate", func() {
		It("should not modify the objects", func() {
			oldClusterArtifactType := &arc.ClusterArtifactType{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-clusterartifacttype",
				},
				Spec: arc.ArtifactTypeSpec{
					WorkflowTemplateRef: arc.ArtifactTypeTemplateRef{
						Name: "old-template",
					},
				},
			}

			newClusterArtifactType := &arc.ClusterArtifactType{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-clusterartifacttype",
				},
				Spec: arc.ArtifactTypeSpec{
					WorkflowTemplateRef: arc.ArtifactTypeTemplateRef{
						Name: "new-template",
					},
				},
			}

			originalNew := newClusterArtifactType.DeepCopy()
			originalOld := oldClusterArtifactType.DeepCopy()

			strategy := clusterartifacttype.NewStrategy(testObjectTyper{})
			strategy.PrepareForUpdate(ctx, newClusterArtifactType, oldClusterArtifactType)
			Expect(newClusterArtifactType).To(Equal(originalNew))
			Expect(oldClusterArtifactType).To(Equal(originalOld))
		})
	})

	Describe("Validate", func() {
		It("should return empty error list", func() {
			cat := &arc.ClusterArtifactType{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-clusterartifacttype",
				},
			}

			strategy := clusterartifacttype.NewStrategy(testObjectTyper{})
			errs := strategy.Validate(ctx, cat)
			Expect(errs).To(BeEmpty())
		})
	})

	Describe("WarningsOnCreate", func() {
		It("should return nil", func() {
			cat := &arc.ClusterArtifactType{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-clusterartifacttype",
				},
			}

			strategy := clusterartifacttype.NewStrategy(testObjectTyper{})
			warnings := strategy.WarningsOnCreate(ctx, cat)
			Expect(warnings).To(BeNil())
		})
	})

	Describe("AllowCreateOnUpdate", func() {
		It("should return false", func() {
			strategy := clusterartifacttype.NewStrategy(testObjectTyper{})
			Expect(strategy.AllowCreateOnUpdate()).To(BeFalse())
		})
	})

	Describe("AllowUnconditionalUpdate", func() {
		It("should return false", func() {
			strategy := clusterartifacttype.NewStrategy(testObjectTyper{})
			Expect(strategy.AllowUnconditionalUpdate()).To(BeFalse())
		})
	})

	Describe("Canonicalize", func() {
		It("should not modify the object", func() {
			cat := &arc.ClusterArtifactType{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-clusterartifacttype",
				},
			}

			original := cat.DeepCopy()
			strategy := clusterartifacttype.NewStrategy(testObjectTyper{})
			strategy.Canonicalize(cat)
			Expect(cat).To(Equal(original))
		})
	})

	Describe("ValidateUpdate", func() {
		It("should return empty error list", func() {
			oldClusterArtifactType := &arc.ClusterArtifactType{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-clusterartifacttype",
				},
			}

			newClusterArtifactType := oldClusterArtifactType.DeepCopy()
			newClusterArtifactType.Labels = map[string]string{"updated": "true"}

			strategy := clusterartifacttype.NewStrategy(testObjectTyper{})
			errs := strategy.ValidateUpdate(ctx, newClusterArtifactType, oldClusterArtifactType)
			Expect(errs).To(BeEmpty())
		})
	})

	Describe("WarningsOnUpdate", func() {
		It("should return nil", func() {
			oldClusterArtifactType := &arc.ClusterArtifactType{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-clusterartifacttype",
				},
			}

			newClusterArtifactType := oldClusterArtifactType.DeepCopy()

			strategy := clusterartifacttype.NewStrategy(testObjectTyper{})
			warnings := strategy.WarningsOnUpdate(ctx, newClusterArtifactType, oldClusterArtifactType)
			Expect(warnings).To(BeNil())
		})
	})
})
