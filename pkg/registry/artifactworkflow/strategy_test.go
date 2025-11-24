// Copyright 2025 BWI GmbH and Artifact Conduit contributors
// SPDX-License-Identifier: Apache-2.0

package artifactworkflow_test

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.opendefense.cloud/arc/api/arc"
	"go.opendefense.cloud/arc/pkg/registry/artifactworkflow"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/validation/field"
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
						Type:       "test-type",
						Parameters: []arc.ArtifactWorkflowParameter{},
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
						Type: "test-type",
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
						Type: "test-type",
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
						Type: "test-type",
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
						Type: "test-type",
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
						Type: "test-type",
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
					Type: "test-type",
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
					Type: "test-type",
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
					Type: "test-type",
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
})
