// Copyright 2025 BWI GmbH and Artifact Conduit contributors
// SPDX-License-Identifier: Apache-2.0

package arc_test

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.opendefense.cloud/arc/api/arc"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/validation/field"
)

var _ = Describe("ArtifactType Strategy", func() {
	var (
		ctx context.Context
	)

	BeforeEach(func() {
		ctx = context.Background()
	})

	Describe("Validate", func() {
		Context("when validating parameters", func() {
			It("should accept ArtifactType with no parameters", func() {
				artifactType := &arc.ArtifactType{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-type",
						Namespace: "default",
					},
					Spec: arc.ArtifactTypeSpec{
						Rules: arc.ArtifactTypeRules{
							SrcTypes: []string{"http"},
							DstTypes: []string{"s3"},
						},
						Parameters: []arc.ArtifactWorkflowParameter{},
						WorkflowTemplateRef: arc.ArtifactTypeTemplateRef{
							Name: "test-template",
						},
					},
				}

				errs := artifactType.Validate(ctx)
				Expect(errs).To(BeEmpty())
			})

			It("should accept ArtifactType with unique parameters", func() {
				artifactType := &arc.ArtifactType{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-type",
						Namespace: "default",
					},
					Spec: arc.ArtifactTypeSpec{
						Rules: arc.ArtifactTypeRules{
							SrcTypes: []string{"http"},
							DstTypes: []string{"s3"},
						},
						Parameters: []arc.ArtifactWorkflowParameter{
							{Name: "param1", Value: "value1"},
							{Name: "param2", Value: "value2"},
							{Name: "param3", Value: "value3"},
						},
						WorkflowTemplateRef: arc.ArtifactTypeTemplateRef{
							Name: "test-template",
						},
					},
				}

				errs := artifactType.Validate(ctx)
				Expect(errs).To(BeEmpty())
			})

			It("should reject ArtifactType with empty parameter name", func() {
				artifactType := &arc.ArtifactType{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-type",
						Namespace: "default",
					},
					Spec: arc.ArtifactTypeSpec{
						Rules: arc.ArtifactTypeRules{},
						Parameters: []arc.ArtifactWorkflowParameter{
							{Name: "", Value: "value1"},
						},
						WorkflowTemplateRef: arc.ArtifactTypeTemplateRef{
							Name: "test-template",
						},
					},
				}

				errs := artifactType.Validate(ctx)
				Expect(errs).To(HaveLen(1))
				Expect(errs[0].Type).To(Equal(field.ErrorTypeRequired))
				Expect(errs[0].Field).To(Equal("spec.parameters[0].name"))
			})

			It("should reject ArtifactType with two duplicate parameters", func() {
				artifactType := &arc.ArtifactType{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-type",
						Namespace: "default",
					},
					Spec: arc.ArtifactTypeSpec{
						Rules: arc.ArtifactTypeRules{},
						Parameters: []arc.ArtifactWorkflowParameter{
							{Name: "param1", Value: "value1"},
							{Name: "param1", Value: "value2"},
						},
						WorkflowTemplateRef: arc.ArtifactTypeTemplateRef{
							Name: "test-template",
						},
					},
				}

				errs := artifactType.Validate(ctx)
				Expect(errs).To(HaveLen(1))
				Expect(errs[0].Type).To(Equal(field.ErrorTypeDuplicate))
				Expect(errs[0].Field).To(Equal("spec.parameters[1].name"))
			})

			It("should reject ArtifactType with multiple different duplicate parameters", func() {
				artifactType := &arc.ArtifactType{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-type",
						Namespace: "default",
					},
					Spec: arc.ArtifactTypeSpec{
						Rules: arc.ArtifactTypeRules{},
						Parameters: []arc.ArtifactWorkflowParameter{
							{Name: "param1", Value: "value1"},
							{Name: "param2", Value: "value2"},
							{Name: "param1", Value: "value3"},
							{Name: "param2", Value: "value4"},
						},
						WorkflowTemplateRef: arc.ArtifactTypeTemplateRef{
							Name: "test-template",
						},
					},
				}

				errs := artifactType.Validate(ctx)
				Expect(errs).To(HaveLen(2))

				// Check that duplicate occurrences after the first are reported
				errorFields := []string{}
				for _, err := range errs {
					Expect(err.Type).To(Equal(field.ErrorTypeDuplicate))
					errorFields = append(errorFields, err.Field)
				}
				Expect(errorFields).To(ConsistOf(
					"spec.parameters[2].name",
					"spec.parameters[3].name",
				))
			})

			It("should reject ArtifactType with empty and duplicate parameters", func() {
				artifactType := &arc.ArtifactType{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-type",
						Namespace: "default",
					},
					Spec: arc.ArtifactTypeSpec{
						Rules: arc.ArtifactTypeRules{},
						Parameters: []arc.ArtifactWorkflowParameter{
							{Name: "", Value: "value1"},
							{Name: "param1", Value: "value2"},
							{Name: "param1", Value: "value3"},
						},
						WorkflowTemplateRef: arc.ArtifactTypeTemplateRef{
							Name: "test-template",
						},
					},
				}

				errs := artifactType.Validate(ctx)
				Expect(errs).To(HaveLen(2))

				// Check error types: empty name and duplicate
				errorTypes := []field.ErrorType{}
				for _, err := range errs {
					errorTypes = append(errorTypes, err.Type)
				}
				Expect(errorTypes).To(ConsistOf(
					field.ErrorTypeRequired,
					field.ErrorTypeDuplicate,
				))
			})
		})

		Context("when validating spec", func() {
			It("should accept ArtifactType with minimal spec", func() {
				artifactType := &arc.ArtifactType{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-type",
						Namespace: "default",
					},
					Spec: arc.ArtifactTypeSpec{
						WorkflowTemplateRef: arc.ArtifactTypeTemplateRef{
							Name: "test-template",
						},
					},
				}

				errs := artifactType.Validate(ctx)
				Expect(errs).To(BeEmpty())
			})

			It("should accept ArtifactType with complete spec", func() {
				artifactType := &arc.ArtifactType{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-type",
						Namespace: "default",
					},
					Spec: arc.ArtifactTypeSpec{
						Rules: arc.ArtifactTypeRules{
							SrcTypes: []string{"http", "oci"},
							DstTypes: []string{"s3", "oci"},
						},
						Parameters: []arc.ArtifactWorkflowParameter{
							{Name: "timeout", Value: "300"},
						},
						WorkflowTemplateRef: arc.ArtifactTypeTemplateRef{
							Name:         "test-template",
							ClusterScope: true,
						},
					},
				}

				errs := artifactType.Validate(ctx)
				Expect(errs).To(BeEmpty())
			})
		})
	})

	Describe("ValidateUpdate", func() {
		Context("when updating ArtifactType", func() {
			It("should reject update when spec is modified", func() {
				oldArtifactType := &arc.ArtifactType{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-type",
						Namespace: "default",
					},
					Spec: arc.ArtifactTypeSpec{
						Rules: arc.ArtifactTypeRules{
							SrcTypes: []string{"http"},
						},
						WorkflowTemplateRef: arc.ArtifactTypeTemplateRef{
							Name: "old-template",
						},
					},
				}

				newArtifactType := &arc.ArtifactType{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-type",
						Namespace: "default",
					},
					Spec: arc.ArtifactTypeSpec{
						Rules: arc.ArtifactTypeRules{
							SrcTypes: []string{"oci"},
						},
						WorkflowTemplateRef: arc.ArtifactTypeTemplateRef{
							Name: "new-template",
						},
					},
				}

				errs := newArtifactType.ValidateUpdate(ctx, oldArtifactType)
				Expect(errs).To(HaveLen(1))
				Expect(errs[0].Type).To(Equal(field.ErrorTypeForbidden))
				Expect(errs[0].Field).To(Equal("spec"))
			})

			It("should accept update when spec is unchanged", func() {
				oldArtifactType := &arc.ArtifactType{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-type",
						Namespace: "default",
					},
					Spec: arc.ArtifactTypeSpec{
						Rules: arc.ArtifactTypeRules{
							SrcTypes: []string{"http"},
						},
						WorkflowTemplateRef: arc.ArtifactTypeTemplateRef{
							Name: "test-template",
						},
					},
				}

				newArtifactType := &arc.ArtifactType{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-type",
						Namespace: "default",
					},
					Spec: arc.ArtifactTypeSpec{
						Rules: arc.ArtifactTypeRules{
							SrcTypes: []string{"http"},
						},
						WorkflowTemplateRef: arc.ArtifactTypeTemplateRef{
							Name: "test-template",
						},
					},
				}

				errs := newArtifactType.ValidateUpdate(ctx, oldArtifactType)
				Expect(errs).To(BeEmpty())
			})

			It("should reject update when rules are modified", func() {
				oldArtifactType := &arc.ArtifactType{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-type",
						Namespace: "default",
					},
					Spec: arc.ArtifactTypeSpec{
						Rules: arc.ArtifactTypeRules{
							SrcTypes: []string{"http"},
							DstTypes: []string{"s3"},
						},
						WorkflowTemplateRef: arc.ArtifactTypeTemplateRef{
							Name: "test-template",
						},
					},
				}

				newArtifactType := &arc.ArtifactType{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-type",
						Namespace: "default",
					},
					Spec: arc.ArtifactTypeSpec{
						Rules: arc.ArtifactTypeRules{
							SrcTypes: []string{"http"},
							DstTypes: []string{"oci"},
						},
						WorkflowTemplateRef: arc.ArtifactTypeTemplateRef{
							Name: "test-template",
						},
					},
				}

				errs := newArtifactType.ValidateUpdate(ctx, oldArtifactType)
				Expect(errs).To(HaveLen(1))
				Expect(errs[0].Type).To(Equal(field.ErrorTypeForbidden))
			})

			It("should reject update when parameters are modified", func() {
				oldArtifactType := &arc.ArtifactType{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-type",
						Namespace: "default",
					},
					Spec: arc.ArtifactTypeSpec{
						Parameters: []arc.ArtifactWorkflowParameter{
							{Name: "param1", Value: "value1"},
						},
						WorkflowTemplateRef: arc.ArtifactTypeTemplateRef{
							Name: "test-template",
						},
					},
				}

				newArtifactType := &arc.ArtifactType{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-type",
						Namespace: "default",
					},
					Spec: arc.ArtifactTypeSpec{
						Parameters: []arc.ArtifactWorkflowParameter{
							{Name: "param1", Value: "new-value"},
						},
						WorkflowTemplateRef: arc.ArtifactTypeTemplateRef{
							Name: "test-template",
						},
					},
				}

				errs := newArtifactType.ValidateUpdate(ctx, oldArtifactType)
				Expect(errs).To(HaveLen(1))
				Expect(errs[0].Type).To(Equal(field.ErrorTypeForbidden))
			})
		})
	})

	Describe("ConvertToTable", func() {
		Context("for single ArtifactType", func() {
			It("should convert ArtifactType to table with correct columns", func() {
				artifactType := &arc.ArtifactType{
					ObjectMeta: metav1.ObjectMeta{
						Name:              "test-type",
						Namespace:         "default",
						ResourceVersion:   "12345",
						CreationTimestamp: metav1.Now(),
					},
					Spec: arc.ArtifactTypeSpec{
						WorkflowTemplateRef: arc.ArtifactTypeTemplateRef{
							Name: "test-template",
						},
					},
					Status: arc.ArtifactTypeStatus{
						Phase:   "Running",
						Message: "Workflow executing",
					},
				}

				table, err := artifactType.ConvertToTable(ctx, nil)
				Expect(err).ToNot(HaveOccurred())
				Expect(table).ToNot(BeNil())

				// Verify column definitions
				Expect(table.ColumnDefinitions).To(HaveLen(4))
				Expect(table.ColumnDefinitions[0].Name).To(Equal("Name"))
				Expect(table.ColumnDefinitions[1].Name).To(Equal("Created At"))
				Expect(table.ColumnDefinitions[2].Name).To(Equal("Phase"))
				Expect(table.ColumnDefinitions[3].Name).To(Equal("Message"))

				// Verify rows
				Expect(table.Rows).To(HaveLen(1))
				row := table.Rows[0]
				Expect(row.Cells).To(HaveLen(4))
				Expect(row.Cells[0]).To(Equal("test-type"))
				Expect(row.Cells[1]).To(Equal(artifactType.CreationTimestamp))
				Expect(row.Cells[2]).To(Equal(arc.ArtifactTypePhase("Running")))
				Expect(row.Cells[3]).To(Equal("Workflow executing"))

				// Verify resource version
				Expect(table.ResourceVersion).To(Equal("12345"))
			})

			It("should convert ArtifactType with empty status", func() {
				artifactType := &arc.ArtifactType{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-type",
						Namespace: "default",
					},
					Spec: arc.ArtifactTypeSpec{
						WorkflowTemplateRef: arc.ArtifactTypeTemplateRef{
							Name: "test-template",
						},
					},
				}

				table, err := artifactType.ConvertToTable(ctx, nil)
				Expect(err).ToNot(HaveOccurred())
				Expect(table).ToNot(BeNil())
				Expect(table.Rows).To(HaveLen(1))

				row := table.Rows[0]
				Expect(row.Cells[2]).To(Equal(arc.ArtifactTypePhase("")))
				Expect(row.Cells[3]).To(Equal(""))
			})
		})

		Context("for ArtifactTypeList", func() {
			It("should convert empty list to table", func() {
				list := &arc.ArtifactTypeList{
					ListMeta: metav1.ListMeta{
						ResourceVersion: "100",
					},
					Items: []arc.ArtifactType{},
				}

				table, err := list.ConvertToTable(ctx, nil)
				Expect(err).ToNot(HaveOccurred())
				Expect(table).ToNot(BeNil())
				Expect(table.ColumnDefinitions).To(HaveLen(4))
				Expect(table.Rows).To(BeEmpty())
				Expect(table.ResourceVersion).To(Equal("100"))
			})

			It("should convert list with single item to table", func() {
				creationTime := metav1.Now()
				list := &arc.ArtifactTypeList{
					ListMeta: metav1.ListMeta{
						ResourceVersion: "200",
					},
					Items: []arc.ArtifactType{
						{
							ObjectMeta: metav1.ObjectMeta{
								Name:              "type-1",
								Namespace:         "default",
								CreationTimestamp: creationTime,
							},
							Status: arc.ArtifactTypeStatus{
								Phase:   "Pending",
								Message: "Workflow initializing",
							},
						},
					},
				}

				table, err := list.ConvertToTable(ctx, nil)
				Expect(err).ToNot(HaveOccurred())
				Expect(table).ToNot(BeNil())
				Expect(table.Rows).To(HaveLen(1))

				row := table.Rows[0]
				Expect(row.Cells[0]).To(Equal("type-1"))
				Expect(row.Cells[1]).To(Equal(creationTime))
				Expect(row.Cells[2]).To(Equal(arc.ArtifactTypePhase("Pending")))
				Expect(row.Cells[3]).To(Equal("Initializing"))

				Expect(table.ResourceVersion).To(Equal("200"))
			})

			It("should convert list with multiple items to table", func() {
				list := &arc.ArtifactTypeList{
					ListMeta: metav1.ListMeta{
						ResourceVersion: "300",
						Continue:        "eyJyZXNvdXJjZVZlcnNpb24iOiIzMDAifQ",
					},
					Items: []arc.ArtifactType{
						{
							ObjectMeta: metav1.ObjectMeta{
								Name:              "type-1",
								Namespace:         "default",
								CreationTimestamp: metav1.Now(),
							},
							Status: arc.ArtifactTypeStatus{
								Phase:   "Running",
								Message: "Workflow executing",
							},
						},
						{
							ObjectMeta: metav1.ObjectMeta{
								Name:              "type-2",
								Namespace:         "default",
								CreationTimestamp: metav1.Now(),
							},
							Status: arc.ArtifactTypeStatus{
								Phase:   "Failed",
								Message: "Workflow failed",
							},
						},
						{
							ObjectMeta: metav1.ObjectMeta{
								Name:              "type-3",
								Namespace:         "production",
								CreationTimestamp: metav1.Now(),
							},
							Status: arc.ArtifactTypeStatus{
								Phase:   "Succeeded",
								Message: "Workflow completed",
							},
						},
					},
				}

				table, err := list.ConvertToTable(ctx, nil)
				Expect(err).ToNot(HaveOccurred())
				Expect(table).ToNot(BeNil())

				// Verify column definitions
				Expect(table.ColumnDefinitions).To(HaveLen(4))

				// Verify all rows are present
				Expect(table.Rows).To(HaveLen(3))

				// Verify first row
				Expect(table.Rows[0].Cells[0]).To(Equal("type-1"))
				Expect(table.Rows[0].Cells[2]).To(Equal(arc.ArtifactTypePhase("Running")))
				Expect(table.Rows[0].Cells[3]).To(Equal("Workflow executing"))

				// Verify second row
				Expect(table.Rows[1].Cells[0]).To(Equal("type-2"))
				Expect(table.Rows[1].Cells[2]).To(Equal(arc.ArtifactTypePhase("Failed")))
				Expect(table.Rows[1].Cells[3]).To(Equal("Workflow failed"))

				// Verify third row
				Expect(table.Rows[2].Cells[0]).To(Equal("type-3"))
				Expect(table.Rows[2].Cells[2]).To(Equal(arc.ArtifactTypePhase("Succeeded")))
				Expect(table.Rows[2].Cells[3]).To(Equal("Workflow completed"))

				// Verify metadata
				Expect(table.ResourceVersion).To(Equal("300"))
				Expect(table.Continue).To(Equal("eyJyZXNvdXJjZVZlcnNpb24iOiIzMDAifQ"))
			})

			It("should handle RemainingItemCount in pagination", func() {
				remainingItems := int64(50)
				list := &arc.ArtifactTypeList{
					ListMeta: metav1.ListMeta{
						ResourceVersion:    "400",
						RemainingItemCount: &remainingItems,
					},
					Items: []arc.ArtifactType{
						{
							ObjectMeta: metav1.ObjectMeta{
								Name:      "type-page-1",
								Namespace: "default",
							},
						},
					},
				}

				table, err := list.ConvertToTable(ctx, nil)
				Expect(err).ToNot(HaveOccurred())
				Expect(table).ToNot(BeNil())
				Expect(table.RemainingItemCount).ToNot(BeNil())
				Expect(*table.RemainingItemCount).To(Equal(int64(50)))
			})
		})
	})

	Describe("ArtifactTypePhase", func() {
		Context("constants", func() {
			It("should have correct phase values", func() {
				Expect(arc.ArtifactTypeUnknown).To(Equal(arc.ArtifactTypePhase("")))
				Expect(arc.ArtifactTypePending).To(Equal(arc.ArtifactTypePhase("Pending")))
				Expect(arc.ArtifactTypeRunning).To(Equal(arc.ArtifactTypePhase("Running")))
				Expect(arc.ArtifactTypeSucceeded).To(Equal(arc.ArtifactTypePhase("Succeeded")))
				Expect(arc.ArtifactTypeFailed).To(Equal(arc.ArtifactTypePhase("Failed")))
				Expect(arc.ArtifactTypeError).To(Equal(arc.ArtifactTypePhase("Error")))
			})
		})

		Context("Completed", func() {
			It("should return true for completed phases", func() {
				Expect(arc.ArtifactTypeSucceeded.Completed()).To(BeTrue())
				Expect(arc.ArtifactTypeFailed.Completed()).To(BeTrue())
				Expect(arc.ArtifactTypeError.Completed()).To(BeTrue())
			})

			It("should return false for non-completed phases", func() {
				Expect(arc.ArtifactTypeUnknown.Completed()).To(BeFalse())
				Expect(arc.ArtifactTypePending.Completed()).To(BeFalse())
				Expect(arc.ArtifactTypeRunning.Completed()).To(BeFalse())
			})

			It("should return false for custom unknown phase", func() {
				customPhase := arc.ArtifactTypePhase("CustomPhase")
				Expect(customPhase.Completed()).To(BeFalse())
			})
		})
	})

	Describe("ClusterArtifactType Strategy", func() {
		Describe("Validate", func() {
			It("should accept ClusterArtifactType with no parameters", func() {
				clusterArtifactType := &arc.ClusterArtifactType{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test-type",
					},
					Spec: arc.ArtifactTypeSpec{
						Parameters: []arc.ArtifactWorkflowParameter{},
						WorkflowTemplateRef: arc.ArtifactTypeTemplateRef{
							Name:         "test-template",
							ClusterScope: true,
						},
					},
				}

				errs := clusterArtifactType.Validate(ctx)
				Expect(errs).To(BeEmpty())
			})

			It("should accept ClusterArtifactType with valid parameters", func() {
				clusterArtifactType := &arc.ClusterArtifactType{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test-type",
					},
					Spec: arc.ArtifactTypeSpec{
						Parameters: []arc.ArtifactWorkflowParameter{
							{Name: "param1", Value: "value1"},
							{Name: "param2", Value: "value2"},
						},
						WorkflowTemplateRef: arc.ArtifactTypeTemplateRef{
							Name: "test-template",
						},
					},
				}

				errs := clusterArtifactType.Validate(ctx)
				Expect(errs).To(BeEmpty())
			})

			It("should reject ClusterArtifactType with duplicate parameters", func() {
				clusterArtifactType := &arc.ClusterArtifactType{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test-type",
					},
					Spec: arc.ArtifactTypeSpec{
						Parameters: []arc.ArtifactWorkflowParameter{
							{Name: "param1", Value: "value1"},
							{Name: "param1", Value: "value2"},
						},
						WorkflowTemplateRef: arc.ArtifactTypeTemplateRef{
							Name: "test-template",
						},
					},
				}

				errs := clusterArtifactType.Validate(ctx)
				Expect(errs).To(HaveLen(1))
				Expect(errs[0].Type).To(Equal(field.ErrorTypeDuplicate))
				Expect(errs[0].Field).To(Equal("spec.parameters[1].name"))
			})
		})

		Describe("ValidateUpdate", func() {
			It("should validate src and dst types cannot be empty", func() {
				oldClusterArtifactType := &arc.ClusterArtifactType{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test-type",
					},
					Spec: arc.ArtifactTypeSpec{
						Rules: arc.ArtifactTypeRules{
							SrcTypes: []string{"http"},
							DstTypes: []string{"s3"},
						},
						WorkflowTemplateRef: arc.ArtifactTypeTemplateRef{
							Name: "test-template",
						},
					},
				}

				newClusterArtifactType := &arc.ClusterArtifactType{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test-type",
					},
					Spec: arc.ArtifactTypeSpec{
						Rules: arc.ArtifactTypeRules{
							SrcTypes: []string{"", "http"},
							DstTypes: []string{"s3", ""},
						},
						WorkflowTemplateRef: arc.ArtifactTypeTemplateRef{
							Name: "test-template",
						},
					},
				}

				errs := newClusterArtifactType.ValidateUpdate(ctx, oldClusterArtifactType)
				// Should have errors for empty srcTypes and dstTypes
				Expect(errs).To(HaveLen(2))
				for _, err := range errs {
					Expect(err.Type).To(Equal(field.ErrorTypeRequired))
				}
			})

			It("should validate parameters in update", func() {
				oldClusterArtifactType := &arc.ClusterArtifactType{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test-type",
					},
					Spec: arc.ArtifactTypeSpec{
						Parameters: []arc.ArtifactWorkflowParameter{
							{Name: "param1", Value: "value1"},
						},
						WorkflowTemplateRef: arc.ArtifactTypeTemplateRef{
							Name: "test-template",
						},
					},
				}

				newClusterArtifactType := &arc.ClusterArtifactType{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test-type",
					},
					Spec: arc.ArtifactTypeSpec{
						Parameters: []arc.ArtifactWorkflowParameter{
							{Name: "", Value: "value1"},
							{Name: "param1", Value: "value1"},
						},
						WorkflowTemplateRef: arc.ArtifactTypeTemplateRef{
							Name: "test-template",
						},
					},
				}

				errs := newClusterArtifactType.ValidateUpdate(ctx, oldClusterArtifactType)
				Expect(errs).ToNot(BeEmpty())
			})

			It("should require workflow template reference name", func() {
				oldClusterArtifactType := &arc.ClusterArtifactType{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test-type",
					},
					Spec: arc.ArtifactTypeSpec{
						WorkflowTemplateRef: arc.ArtifactTypeTemplateRef{
							Name: "test-template",
						},
					},
				}

				newClusterArtifactType := &arc.ClusterArtifactType{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test-type",
					},
					Spec: arc.ArtifactTypeSpec{
						WorkflowTemplateRef: arc.ArtifactTypeTemplateRef{
							Name: "",
						},
					},
				}

				errs := newClusterArtifactType.ValidateUpdate(ctx, oldClusterArtifactType)
				Expect(errs).To(HaveLen(1))
				Expect(errs[0].Type).To(Equal(field.ErrorTypeRequired))
				Expect(errs[0].Field).To(Equal("spec.workflowTemplateRef.name"))
			})
		})
	})
})
