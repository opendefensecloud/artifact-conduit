// Copyright 2025 BWI GmbH and Artifact Conduit contributors
// SPDX-License-Identifier: Apache-2.0

package artifactworkflow_test

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.opendefense.cloud/arc/api/arc"
	"go.opendefense.cloud/arc/pkg/registry/artifactworkflow"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"sigs.k8s.io/structured-merge-diff/v6/fieldpath"
)

type testObjectTyper struct{}

func (testObjectTyper) ObjectKinds(runtime.Object) ([]schema.GroupVersionKind, bool, error) {
	return nil, false, nil
}

func (testObjectTyper) Recognizes(gvk schema.GroupVersionKind) bool {
	return false
}

var _ = Describe("ArtifactWorkflow Strategy", func() {
	var (
		ctx context.Context
	)

	BeforeEach(func() {
		ctx = context.Background()
	})

	Describe("Validate", func() {
		Context("when validating parameters", func() {
			It("should accept ArtifactWorkflow with no parameters", func() {
				workflow := &arc.ArtifactWorkflow{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-workflow",
						Namespace: "default",
					},
					Spec: arc.ArtifactWorkflowSpec{
						WorkflowTemplateRef: arc.ArtifactTypeTemplateRef{Name: "test"},
						Parameters:          []arc.ArtifactWorkflowParameter{},
					},
				}

				strategy := artifactworkflow.NewStrategy(testObjectTyper{})
				errs := strategy.Validate(ctx, workflow)
				Expect(errs).To(BeEmpty())
			})

			It("should accept ArtifactWorkflow with unique parameters", func() {
				workflow := &arc.ArtifactWorkflow{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-workflow",
						Namespace: "default",
					},
					Spec: arc.ArtifactWorkflowSpec{
						WorkflowTemplateRef: arc.ArtifactTypeTemplateRef{Name: "test"},
						Parameters: []arc.ArtifactWorkflowParameter{
							{Name: "param1", Value: "value1"},
							{Name: "param2", Value: "value2"},
							{Name: "param3", Value: "value3"},
						},
					},
				}

				strategy := artifactworkflow.NewStrategy(testObjectTyper{})
				errs := strategy.Validate(ctx, workflow)
				Expect(errs).To(BeEmpty())
			})

			It("should reject ArtifactWorkflow with two duplicate parameters", func() {
				workflow := &arc.ArtifactWorkflow{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-workflow",
						Namespace: "default",
					},
					Spec: arc.ArtifactWorkflowSpec{
						WorkflowTemplateRef: arc.ArtifactTypeTemplateRef{Name: "test"},
						Parameters: []arc.ArtifactWorkflowParameter{
							{Name: "param1", Value: "value1"},
							{Name: "param1", Value: "value2"},
						},
					},
				}

				strategy := artifactworkflow.NewStrategy(testObjectTyper{})
				errs := strategy.Validate(ctx, workflow)
				Expect(errs).To(HaveLen(2))

				// Check that both duplicate occurrences are reported
				errorFields := []string{}
				for _, err := range errs {
					Expect(err.Type).To(Equal(field.ErrorTypeDuplicate))
					errorFields = append(errorFields, err.Field)
				}
				Expect(errorFields).To(ConsistOf(
					"spec.parameters[0].name",
					"spec.parameters[1].name",
				))
			})

			It("should reject ArtifactWorkflow with three duplicate parameters", func() {
				workflow := &arc.ArtifactWorkflow{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-workflow",
						Namespace: "default",
					},
					Spec: arc.ArtifactWorkflowSpec{
						WorkflowTemplateRef: arc.ArtifactTypeTemplateRef{Name: "test"},
						Parameters: []arc.ArtifactWorkflowParameter{
							{Name: "param1", Value: "value1"},
							{Name: "param2", Value: "value2"},
							{Name: "param1", Value: "value3"},
							{Name: "param1", Value: "value4"},
						},
					},
				}

				strategy := artifactworkflow.NewStrategy(testObjectTyper{})
				errs := strategy.Validate(ctx, workflow)
				Expect(errs).To(HaveLen(3))

				// Check that all duplicate occurrences are reported
				errorFields := []string{}
				for _, err := range errs {
					Expect(err.Type).To(Equal(field.ErrorTypeDuplicate))
					errorFields = append(errorFields, err.Field)
				}
				Expect(errorFields).To(ConsistOf(
					"spec.parameters[0].name",
					"spec.parameters[2].name",
					"spec.parameters[3].name",
				))
			})

			It("should reject ArtifactWorkflow with multiple different duplicate parameters", func() {
				workflow := &arc.ArtifactWorkflow{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-workflow",
						Namespace: "default",
					},
					Spec: arc.ArtifactWorkflowSpec{
						WorkflowTemplateRef: arc.ArtifactTypeTemplateRef{Name: "test"},
						Parameters: []arc.ArtifactWorkflowParameter{
							{Name: "param1", Value: "value1"},
							{Name: "param2", Value: "value2"},
							{Name: "param1", Value: "value3"},
							{Name: "param2", Value: "value4"},
							{Name: "param3", Value: "value5"},
						},
					},
				}

				strategy := artifactworkflow.NewStrategy(testObjectTyper{})
				errs := strategy.Validate(ctx, workflow)
				Expect(errs).To(HaveLen(4))

				// Check that all duplicate occurrences are reported
				duplicateCount := 0
				for _, err := range errs {
					if err.Type == field.ErrorTypeDuplicate {
						duplicateCount++
					}
				}
				Expect(duplicateCount).To(Equal(4))
			})

			It("should handle parameters with same value but different names", func() {
				workflow := &arc.ArtifactWorkflow{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-workflow",
						Namespace: "default",
					},
					Spec: arc.ArtifactWorkflowSpec{
						WorkflowTemplateRef: arc.ArtifactTypeTemplateRef{Name: "test"},
						Parameters: []arc.ArtifactWorkflowParameter{
							{Name: "param1", Value: "samevalue"},
							{Name: "param2", Value: "samevalue"},
							{Name: "param3", Value: "samevalue"},
						},
					},
				}

				strategy := artifactworkflow.NewStrategy(testObjectTyper{})
				errs := strategy.Validate(ctx, workflow)
				Expect(errs).To(BeEmpty(), "Should accept parameters with same value but different names")
			})
		})

		Context("when validating object type", func() {
			It("should return internal error for non-ArtifactWorkflow object", func() {
				notAWorkflow := &arc.ArtifactType{
					ObjectMeta: metav1.ObjectMeta{
						Name: "not-a-workflow",
					},
				}

				strategy := artifactworkflow.NewStrategy(testObjectTyper{})
				errs := strategy.Validate(ctx, notAWorkflow)
				Expect(errs).To(HaveLen(1))
				Expect(errs[0].Type).To(Equal(field.ErrorTypeInternal))
				Expect(errs[0].Detail).To(ContainSubstring("not an ArtifactWorkflow"))
			})
		})
	})

	Describe("NamespaceScoped", func() {
		It("should return true for ArtifactWorkflow", func() {
			strategy := artifactworkflow.NewStrategy(testObjectTyper{})
			Expect(strategy.NamespaceScoped()).To(BeTrue())
		})
	})

	Describe("PrepareForCreate", func() {
		It("should not modify the object during create", func() {
			workflow := &arc.ArtifactWorkflow{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-workflow",
					Namespace: "default",
				},
				Spec: arc.ArtifactWorkflowSpec{
					WorkflowTemplateRef: arc.ArtifactTypeTemplateRef{Name: "test"},
					Parameters: []arc.ArtifactWorkflowParameter{
						{Name: "param1", Value: "value1"},
					},
				},
			}

			originalWorkflow := workflow.DeepCopy()
			strategy := artifactworkflow.NewStrategy(testObjectTyper{})
			strategy.PrepareForCreate(ctx, workflow)
			Expect(workflow).To(Equal(originalWorkflow))
		})
	})

	Describe("PrepareForUpdate", func() {
		It("should not modify objects during update", func() {
			oldWorkflow := &arc.ArtifactWorkflow{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-workflow",
					Namespace: "default",
				},
				Spec: arc.ArtifactWorkflowSpec{
					WorkflowTemplateRef: arc.ArtifactTypeTemplateRef{Name: "test"},
					Parameters: []arc.ArtifactWorkflowParameter{
						{Name: "param1", Value: "value1"},
					},
				},
			}

			newWorkflow := &arc.ArtifactWorkflow{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-workflow",
					Namespace: "default",
				},
				Spec: arc.ArtifactWorkflowSpec{
					WorkflowTemplateRef: arc.ArtifactTypeTemplateRef{Name: "test"},
					Parameters: []arc.ArtifactWorkflowParameter{
						{Name: "param1", Value: "value2"},
						{Name: "param2", Value: "value3"},
					},
				},
			}

			originalNew := newWorkflow.DeepCopy()
			originalOld := oldWorkflow.DeepCopy()

			strategy := artifactworkflow.NewStrategy(testObjectTyper{})
			strategy.PrepareForUpdate(ctx, newWorkflow, oldWorkflow)
			Expect(newWorkflow).To(Equal(originalNew))
			Expect(oldWorkflow).To(Equal(originalOld))
		})
	})

	Describe("ValidateUpdate", func() {
		Context("when validating spec immutability", func() {
			It("should accept update when spec is unchanged", func() {
				oldWorkflow := &arc.ArtifactWorkflow{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-workflow",
						Namespace: "default",
					},
					Spec: arc.ArtifactWorkflowSpec{
						WorkflowTemplateRef: arc.ArtifactTypeTemplateRef{Name: "test"},
						Parameters: []arc.ArtifactWorkflowParameter{
							{Name: "param1", Value: "value1"},
						},
					},
				}

				newWorkflow := oldWorkflow.DeepCopy()
				// Only change metadata, not spec
				newWorkflow.Labels = map[string]string{"updated": "true"}

				strategy := artifactworkflow.NewStrategy(testObjectTyper{})
				errs := strategy.ValidateUpdate(ctx, newWorkflow, oldWorkflow)
				Expect(errs).To(BeEmpty())
			})

			It("should reject update when spec.Type is changed", func() {
				oldWorkflow := &arc.ArtifactWorkflow{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-workflow",
						Namespace: "default",
					},
					Spec: arc.ArtifactWorkflowSpec{
						WorkflowTemplateRef: arc.ArtifactTypeTemplateRef{Name: "test"},
						Parameters: []arc.ArtifactWorkflowParameter{
							{Name: "param1", Value: "value1"},
						},
					},
				}

				newWorkflow := oldWorkflow.DeepCopy()
				newWorkflow.Spec.WorkflowTemplateRef.Name = "different-type"

				strategy := artifactworkflow.NewStrategy(testObjectTyper{})
				errs := strategy.ValidateUpdate(ctx, newWorkflow, oldWorkflow)
				Expect(errs).To(HaveLen(1))
				Expect(errs[0].Type).To(Equal(field.ErrorTypeForbidden))
				Expect(errs[0].Field).To(Equal("spec"))
				Expect(errs[0].Detail).To(ContainSubstring("spec is immutable"))
			})

			It("should reject update when parameters are changed", func() {
				oldWorkflow := &arc.ArtifactWorkflow{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-workflow",
						Namespace: "default",
					},
					Spec: arc.ArtifactWorkflowSpec{
						WorkflowTemplateRef: arc.ArtifactTypeTemplateRef{Name: "test"},
						Parameters: []arc.ArtifactWorkflowParameter{
							{Name: "param1", Value: "value1"},
						},
					},
				}

				newWorkflow := oldWorkflow.DeepCopy()
				newWorkflow.Spec.Parameters = []arc.ArtifactWorkflowParameter{
					{Name: "param1", Value: "value2"}, // Changed value
				}

				strategy := artifactworkflow.NewStrategy(testObjectTyper{})
				errs := strategy.ValidateUpdate(ctx, newWorkflow, oldWorkflow)
				Expect(errs).To(HaveLen(1))
				Expect(errs[0].Type).To(Equal(field.ErrorTypeForbidden))
				Expect(errs[0].Field).To(Equal("spec"))
				Expect(errs[0].Detail).To(ContainSubstring("spec is immutable"))
			})

			It("should reject update when parameters are added", func() {
				oldWorkflow := &arc.ArtifactWorkflow{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-workflow",
						Namespace: "default",
					},
					Spec: arc.ArtifactWorkflowSpec{
						WorkflowTemplateRef: arc.ArtifactTypeTemplateRef{Name: "test"},
						Parameters: []arc.ArtifactWorkflowParameter{
							{Name: "param1", Value: "value1"},
						},
					},
				}

				newWorkflow := oldWorkflow.DeepCopy()
				newWorkflow.Spec.Parameters = append(newWorkflow.Spec.Parameters,
					arc.ArtifactWorkflowParameter{Name: "param2", Value: "value2"})

				strategy := artifactworkflow.NewStrategy(testObjectTyper{})
				errs := strategy.ValidateUpdate(ctx, newWorkflow, oldWorkflow)
				Expect(errs).To(HaveLen(1))
				Expect(errs[0].Type).To(Equal(field.ErrorTypeForbidden))
				Expect(errs[0].Field).To(Equal("spec"))
				Expect(errs[0].Detail).To(ContainSubstring("spec is immutable"))
			})

			It("should reject update when parameters are removed", func() {
				oldWorkflow := &arc.ArtifactWorkflow{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-workflow",
						Namespace: "default",
					},
					Spec: arc.ArtifactWorkflowSpec{
						WorkflowTemplateRef: arc.ArtifactTypeTemplateRef{Name: "test"},
						Parameters: []arc.ArtifactWorkflowParameter{
							{Name: "param1", Value: "value1"},
							{Name: "param2", Value: "value2"},
						},
					},
				}

				newWorkflow := oldWorkflow.DeepCopy()
				newWorkflow.Spec.Parameters = []arc.ArtifactWorkflowParameter{
					{Name: "param1", Value: "value1"},
				}

				strategy := artifactworkflow.NewStrategy(testObjectTyper{})
				errs := strategy.ValidateUpdate(ctx, newWorkflow, oldWorkflow)
				Expect(errs).To(HaveLen(1))
				Expect(errs[0].Type).To(Equal(field.ErrorTypeForbidden))
				Expect(errs[0].Field).To(Equal("spec"))
				Expect(errs[0].Detail).To(ContainSubstring("spec is immutable"))
			})

			It("should reject update when any part of spec is modified", func() {
				oldWorkflow := &arc.ArtifactWorkflow{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-workflow",
						Namespace: "default",
					},
					Spec: arc.ArtifactWorkflowSpec{
						WorkflowTemplateRef: arc.ArtifactTypeTemplateRef{Name: "test"},
						Parameters: []arc.ArtifactWorkflowParameter{
							{Name: "param1", Value: "value1"},
						},
						SrcSecretRef: corev1.LocalObjectReference{Name: "src-secret"},
						DstSecretRef: corev1.LocalObjectReference{Name: "dst-secret"},
					},
				}

				newWorkflow := oldWorkflow.DeepCopy()
				newWorkflow.Spec.SrcSecretRef = corev1.LocalObjectReference{Name: "new-src-secret"}

				strategy := artifactworkflow.NewStrategy(testObjectTyper{})
				errs := strategy.ValidateUpdate(ctx, newWorkflow, oldWorkflow)
				Expect(errs).ToNot(BeEmpty())
				Expect(errs[0].Type).To(Equal(field.ErrorTypeForbidden))
				Expect(errs[0].Field).To(Equal("spec"))
				Expect(errs[0].Detail).To(ContainSubstring("spec is immutable"))
			})
		})

		Context("when validating object type", func() {
			It("should return internal error for non-ArtifactWorkflow new object", func() {
				oldWorkflow := &arc.ArtifactWorkflow{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-workflow",
						Namespace: "default",
					},
				}

				notAWorkflow := &arc.ArtifactType{
					ObjectMeta: metav1.ObjectMeta{
						Name: "not-a-workflow",
					},
				}

				strategy := artifactworkflow.NewStrategy(testObjectTyper{})
				errs := strategy.ValidateUpdate(ctx, notAWorkflow, oldWorkflow)
				Expect(errs).To(HaveLen(1))
				Expect(errs[0].Type).To(Equal(field.ErrorTypeInternal))
				Expect(errs[0].Detail).To(ContainSubstring("not an ArtifactWorkflow"))
			})

			It("should return internal error for non-ArtifactWorkflow old object", func() {
				newWorkflow := &arc.ArtifactWorkflow{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-workflow",
						Namespace: "default",
					},
				}

				notAWorkflow := &arc.ArtifactType{
					ObjectMeta: metav1.ObjectMeta{
						Name: "not-a-workflow",
					},
				}

				strategy := artifactworkflow.NewStrategy(testObjectTyper{})
				errs := strategy.ValidateUpdate(ctx, newWorkflow, notAWorkflow)
				Expect(errs).To(HaveLen(1))
				Expect(errs[0].Type).To(Equal(field.ErrorTypeInternal))
				Expect(errs[0].Detail).To(ContainSubstring("old object is not an ArtifactWorkflow"))
			})
		})
	})

	Describe("GetAttrs", func() {
		It("should return labels and fields for an ArtifactWorkflow", func() {
			aw := &arc.ArtifactWorkflow{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-workflow",
					Namespace: "default",
					Labels: map[string]string{
						"app": "test",
					},
				},
			}

			lbls, flds, err := artifactworkflow.GetAttrs(aw)
			Expect(err).NotTo(HaveOccurred())
			Expect(lbls).To(HaveKeyWithValue("app", "test"))
			Expect(flds.Has("metadata.name")).To(BeTrue())
			Expect(flds.Get("metadata.name")).To(Equal("test-workflow"))
		})

		It("should return error for non-ArtifactWorkflow object", func() {
			notAWorkflow := &arc.Order{
				ObjectMeta: metav1.ObjectMeta{
					Name: "not-a-workflow",
				},
			}

			_, _, err := artifactworkflow.GetAttrs(notAWorkflow)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("not an ArtifactWorkflow"))
		})
	})

	Describe("MatchArtifactWorkflow", func() {
		It("should create a selection predicate", func() {
			labelSelector := labels.Everything()
			fieldSelector := fields.Everything()

			predicate := artifactworkflow.MatchArtifactWorkflow(labelSelector, fieldSelector)
			Expect(predicate.Label).To(Equal(labelSelector))
			Expect(predicate.Field).To(Equal(fieldSelector))
			Expect(predicate.GetAttrs).NotTo(BeNil())
		})
	})

	Describe("SelectableFields", func() {
		It("should return selectable fields for an ArtifactWorkflow", func() {
			aw := &arc.ArtifactWorkflow{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-workflow",
					Namespace: "default",
				},
			}

			flds := artifactworkflow.SelectableFields(aw)
			Expect(flds.Has("metadata.name")).To(BeTrue())
			Expect(flds.Has("metadata.namespace")).To(BeTrue())
		})
	})

	Describe("AllowCreateOnUpdate", func() {
		It("should return false", func() {
			strategy := artifactworkflow.NewStrategy(testObjectTyper{})
			Expect(strategy.AllowCreateOnUpdate()).To(BeFalse())
		})
	})

	Describe("AllowUnconditionalUpdate", func() {
		It("should return false", func() {
			strategy := artifactworkflow.NewStrategy(testObjectTyper{})
			Expect(strategy.AllowUnconditionalUpdate()).To(BeFalse())
		})
	})

	Describe("Canonicalize", func() {
		It("should not modify the object", func() {
			aw := &arc.ArtifactWorkflow{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-workflow",
					Namespace: "default",
				},
			}

			original := aw.DeepCopy()
			strategy := artifactworkflow.NewStrategy(testObjectTyper{})
			strategy.Canonicalize(aw)
			Expect(aw).To(Equal(original))
		})
	})

	Describe("WarningsOnCreate", func() {
		It("should return nil", func() {
			aw := &arc.ArtifactWorkflow{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-workflow",
					Namespace: "default",
				},
			}

			strategy := artifactworkflow.NewStrategy(testObjectTyper{})
			warnings := strategy.WarningsOnCreate(ctx, aw)
			Expect(warnings).To(BeNil())
		})
	})

	Describe("WarningsOnUpdate", func() {
		It("should return nil", func() {
			oldWorkflow := &arc.ArtifactWorkflow{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-workflow",
					Namespace: "default",
				},
			}

			newWorkflow := oldWorkflow.DeepCopy()

			strategy := artifactworkflow.NewStrategy(testObjectTyper{})
			warnings := strategy.WarningsOnUpdate(ctx, newWorkflow, oldWorkflow)
			Expect(warnings).To(BeNil())
		})
	})
})

var _ = Describe("ArtifactWorkflow Status Strategy", func() {
	var (
		ctx context.Context
	)

	BeforeEach(func() {
		ctx = context.Background()
	})

	Describe("NewStatusStrategy", func() {
		It("should create a new status strategy", func() {
			s := artifactworkflow.NewStatusStrategy(testObjectTyper{})
			Expect(s).NotTo(BeNil())
		})
	})

	Describe("GetResetFields", func() {
		It("should return spec as a reset field", func() {
			statusStrategy := artifactworkflow.NewStatusStrategy(testObjectTyper{})
			resetFields := statusStrategy.GetResetFields()
			Expect(resetFields).To(HaveKey(fieldpath.APIVersion("arc.bwi.de/v1alpha1")))

			fieldSet := resetFields["arc.bwi.de/v1alpha1"]
			Expect(fieldSet).NotTo(BeNil())
			Expect(fieldSet.Has(fieldpath.MakePathOrDie("spec"))).To(BeTrue())
		})
	})

	Describe("PrepareForUpdate", func() {
		It("should preserve spec from old object", func() {
			oldWorkflow := &arc.ArtifactWorkflow{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-workflow",
					Namespace: "default",
				},
				Spec: arc.ArtifactWorkflowSpec{
					WorkflowTemplateRef: arc.ArtifactTypeTemplateRef{Name: "original-template"},
					Parameters: []arc.ArtifactWorkflowParameter{
						{Name: "original-param", Value: "original-value"},
					},
				},
				Status: arc.ArtifactWorkflowStatus{
					Phase: "Pending",
				},
			}

			newWorkflow := &arc.ArtifactWorkflow{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-workflow",
					Namespace: "default",
				},
				Spec: arc.ArtifactWorkflowSpec{
					WorkflowTemplateRef: arc.ArtifactTypeTemplateRef{Name: "modified-template"},
					Parameters: []arc.ArtifactWorkflowParameter{
						{Name: "modified-param", Value: "modified-value"},
					},
				},
				Status: arc.ArtifactWorkflowStatus{
					Phase: "Running",
				},
			}

			statusStrategy := artifactworkflow.NewStatusStrategy(testObjectTyper{})
			statusStrategy.PrepareForUpdate(ctx, newWorkflow, oldWorkflow)

			// Spec should be preserved from old object
			Expect(newWorkflow.Spec).To(Equal(oldWorkflow.Spec))
			// Status should remain as set in new object
			Expect(string(newWorkflow.Status.Phase)).To(Equal("Running"))
		})
	})

	Describe("ValidateUpdate", func() {
		It("should return empty error list", func() {
			oldWorkflow := &arc.ArtifactWorkflow{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-workflow",
					Namespace: "default",
				},
			}

			newWorkflow := oldWorkflow.DeepCopy()
			newWorkflow.Status.Phase = "Completed"

			statusStrategy := artifactworkflow.NewStatusStrategy(testObjectTyper{})
			errs := statusStrategy.ValidateUpdate(ctx, newWorkflow, oldWorkflow)
			Expect(errs).To(BeEmpty())
		})
	})

	Describe("WarningsOnUpdate", func() {
		It("should return nil", func() {
			oldWorkflow := &arc.ArtifactWorkflow{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-workflow",
					Namespace: "default",
				},
			}

			newWorkflow := oldWorkflow.DeepCopy()

			statusStrategy := artifactworkflow.NewStatusStrategy(testObjectTyper{})
			warnings := statusStrategy.WarningsOnUpdate(ctx, newWorkflow, oldWorkflow)
			Expect(warnings).To(BeNil())
		})
	})
})
