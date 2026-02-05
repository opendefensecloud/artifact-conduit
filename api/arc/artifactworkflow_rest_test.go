// Copyright 2025 BWI GmbH and Artifact Conduit contributors
// SPDX-License-Identifier: Apache-2.0

package arc_test

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/validation/field"

	"go.opendefense.cloud/arc/api/arc"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

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

				errs := workflow.Validate(ctx)
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

				errs := workflow.Validate(ctx)
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

				errs := workflow.Validate(ctx)
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

				errs := workflow.Validate(ctx)
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

				errs := workflow.Validate(ctx)
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

				errs := workflow.Validate(ctx)
				Expect(errs).To(BeEmpty(), "Should accept parameters with same value but different names")
			})
		})

	})

	Describe("NamespaceScoped", func() {
		It("should return true for ArtifactWorkflow", func() {
			Expect((&arc.ArtifactWorkflow{}).NamespaceScoped()).To(BeTrue())
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

				errs := newWorkflow.ValidateUpdate(ctx, oldWorkflow)
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

				errs := newWorkflow.ValidateUpdate(ctx, oldWorkflow)
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

				errs := newWorkflow.ValidateUpdate(ctx, oldWorkflow)
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

				errs := newWorkflow.ValidateUpdate(ctx, oldWorkflow)
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

				errs := newWorkflow.ValidateUpdate(ctx, oldWorkflow)
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

				errs := newWorkflow.ValidateUpdate(ctx, oldWorkflow)
				Expect(errs).ToNot(BeEmpty())
				Expect(errs[0].Type).To(Equal(field.ErrorTypeForbidden))
				Expect(errs[0].Field).To(Equal("spec"))
				Expect(errs[0].Detail).To(ContainSubstring("spec is immutable"))
			})
		})

		Context("when validating object type", func() {
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

				errs := newWorkflow.ValidateUpdate(ctx, notAWorkflow)
				Expect(errs).To(HaveLen(1))
				Expect(errs[0].Type).To(Equal(field.ErrorTypeInternal))
				Expect(errs[0].Detail).To(ContainSubstring("old object is not an ArtifactWorkflow"))
			})
		})
	})
})
