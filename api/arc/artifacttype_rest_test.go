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
						Name:      "http-to-s3.converter",
						Namespace: "default",
					},
					Spec: arc.ArtifactTypeSpec{
						Rules: arc.ArtifactTypeRules{
							SrcTypes: []string{"http"},
							DstTypes: []string{"s3"},
						},
						Parameters: []arc.ArtifactWorkflowParameter{},
						WorkflowTemplateRef: arc.ArtifactTypeTemplateRef{
							Name: "artifact-processing.v2",
						},
					},
				}

				errs := artifactType.Validate(ctx)
				Expect(errs).To(BeEmpty())
			})

			It("should accept ArtifactType with unique parameters", func() {
				artifactType := &arc.ArtifactType{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "oci-image-converter.v1",
						Namespace: "default",
					},
					Spec: arc.ArtifactTypeSpec{
						Rules: arc.ArtifactTypeRules{
							SrcTypes: []string{"http"},
							DstTypes: []string{"s3"},
						},
						Parameters: []arc.ArtifactWorkflowParameter{
							{Name: "max_retries", Value: "3"},
							{Name: "timeout.seconds", Value: "300"},
							{Name: "enable_cache", Value: "true"},
						},
						WorkflowTemplateRef: arc.ArtifactTypeTemplateRef{
							Name: "workflow-template-v1.2",
						},
					},
				}

				errs := artifactType.Validate(ctx)
				Expect(errs).To(BeEmpty())
			})

			It("should reject ArtifactType with empty parameter name", func() {
				artifactType := &arc.ArtifactType{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "artifact-type-with.empty",
						Namespace: "default",
					},
					Spec: arc.ArtifactTypeSpec{
						Rules: arc.ArtifactTypeRules{},
						Parameters: []arc.ArtifactWorkflowParameter{
							{Name: "", Value: "https://example.com/path?query=1"},
						},
						WorkflowTemplateRef: arc.ArtifactTypeTemplateRef{
							Name: "validation-workflow",
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
						Name:      "duplicate.param-test",
						Namespace: "default",
					},
					Spec: arc.ArtifactTypeSpec{
						Rules: arc.ArtifactTypeRules{},
						Parameters: []arc.ArtifactWorkflowParameter{
							{Name: "storage_backend", Value: "s3://bucket1"},
							{Name: "storage_backend", Value: "s3://bucket2"},
						},
						WorkflowTemplateRef: arc.ArtifactTypeTemplateRef{
							Name: "storage.workflow-v2",
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
						Name:      "multiple-duplicates-test",
						Namespace: "default",
					},
					Spec: arc.ArtifactTypeSpec{
						Rules: arc.ArtifactTypeRules{},
						Parameters: []arc.ArtifactWorkflowParameter{
							{Name: "api.endpoint", Value: "https://api1.example.com"},
							{Name: "retry_count", Value: "5"},
							{Name: "api.endpoint", Value: "https://api2.example.com"},
							{Name: "retry_count", Value: "10"},
						},
						WorkflowTemplateRef: arc.ArtifactTypeTemplateRef{
							Name: "api-workflow.template",
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
						Name:      "empty-and-duplicate.test",
						Namespace: "default",
					},
					Spec: arc.ArtifactTypeSpec{
						Rules: arc.ArtifactTypeRules{},
						Parameters: []arc.ArtifactWorkflowParameter{
							{Name: "", Value: "some_value"},
							{Name: "compression.type", Value: "gzip"},
							{Name: "compression.type", Value: "bzip2"},
						},
						WorkflowTemplateRef: arc.ArtifactTypeTemplateRef{
							Name: "compression-workflow",
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
						Name:      "minimal.spec-type",
						Namespace: "default",
					},
					Spec: arc.ArtifactTypeSpec{
						WorkflowTemplateRef: arc.ArtifactTypeTemplateRef{
							Name: "minimal-workflow.v1",
						},
					},
				}

				errs := artifactType.Validate(ctx)
				Expect(errs).To(BeEmpty())
			})

			It("should accept ArtifactType with complete spec", func() {
				artifactType := &arc.ArtifactType{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "complete-spec-type.v2",
						Namespace: "default",
					},
					Spec: arc.ArtifactTypeSpec{
						Rules: arc.ArtifactTypeRules{
							SrcTypes: []string{"http", "oci"},
							DstTypes: []string{"s3", "oci"},
						},
						Parameters: []arc.ArtifactWorkflowParameter{
							{Name: "request.timeout_ms", Value: "30000"},
						},
						WorkflowTemplateRef: arc.ArtifactTypeTemplateRef{
							Name:         "cluster-workflow-template",
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
						Name:      "update-test.artifact",
						Namespace: "default",
					},
					Spec: arc.ArtifactTypeSpec{
						Rules: arc.ArtifactTypeRules{
							SrcTypes: []string{"http"},
						},
						WorkflowTemplateRef: arc.ArtifactTypeTemplateRef{
							Name: "legacy.workflow-v1",
						},
					},
				}

				newArtifactType := &arc.ArtifactType{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "update-test.artifact",
						Namespace: "default",
					},
					Spec: arc.ArtifactTypeSpec{
						Rules: arc.ArtifactTypeRules{
							SrcTypes: []string{"oci"},
						},
						WorkflowTemplateRef: arc.ArtifactTypeTemplateRef{
							Name: "modern.workflow-v2",
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
						Name:      "unchanged-spec.test",
						Namespace: "default",
					},
					Spec: arc.ArtifactTypeSpec{
						Rules: arc.ArtifactTypeRules{
							SrcTypes: []string{"http"},
						},
						WorkflowTemplateRef: arc.ArtifactTypeTemplateRef{
							Name: "stable.workflow",
						},
					},
				}

				newArtifactType := &arc.ArtifactType{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "unchanged-spec.test",
						Namespace: "default",
					},
					Spec: arc.ArtifactTypeSpec{
						Rules: arc.ArtifactTypeRules{
							SrcTypes: []string{"http"},
						},
						WorkflowTemplateRef: arc.ArtifactTypeTemplateRef{
							Name: "stable.workflow",
						},
					},
				}

				errs := newArtifactType.ValidateUpdate(ctx, oldArtifactType)
				Expect(errs).To(BeEmpty())
			})

			It("should reject update when rules are modified", func() {
				oldArtifactType := &arc.ArtifactType{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "rules-modification.test",
						Namespace: "default",
					},
					Spec: arc.ArtifactTypeSpec{
						Rules: arc.ArtifactTypeRules{
							SrcTypes: []string{"http"},
							DstTypes: []string{"s3"},
						},
						WorkflowTemplateRef: arc.ArtifactTypeTemplateRef{
							Name: "conversion.workflow",
						},
					},
				}

				newArtifactType := &arc.ArtifactType{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "rules-modification.test",
						Namespace: "default",
					},
					Spec: arc.ArtifactTypeSpec{
						Rules: arc.ArtifactTypeRules{
							SrcTypes: []string{"http"},
							DstTypes: []string{"oci"},
						},
						WorkflowTemplateRef: arc.ArtifactTypeTemplateRef{
							Name: "conversion.workflow",
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
						Name:      "params-change.test",
						Namespace: "default",
					},
					Spec: arc.ArtifactTypeSpec{
						Parameters: []arc.ArtifactWorkflowParameter{
							{Name: "cache.ttl_seconds", Value: "3600"},
						},
						WorkflowTemplateRef: arc.ArtifactTypeTemplateRef{
							Name: "cache-workflow.v1",
						},
					},
				}

				newArtifactType := &arc.ArtifactType{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "params-change.test",
						Namespace: "default",
					},
					Spec: arc.ArtifactTypeSpec{
						Parameters: []arc.ArtifactWorkflowParameter{
							{Name: "cache.ttl_seconds", Value: "7200"},
						},
						WorkflowTemplateRef: arc.ArtifactTypeTemplateRef{
							Name: "cache-workflow.v1",
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
						Name:              "table-conversion.type-test",
						Namespace:         "default",
						ResourceVersion:   "12345",
						CreationTimestamp: metav1.Now(),
					},
					Spec: arc.ArtifactTypeSpec{
						Parameters: []arc.ArtifactWorkflowParameter{
							{Name: "auth.api_key", Value: "sk-1234567890abcdef"},
							{Name: "storage.region", Value: "us-west-2"},
						},
						WorkflowTemplateRef: arc.ArtifactTypeTemplateRef{
							Name: "table.workflow-v3",
						},
					},
				}

				table, err := artifactType.ConvertToTable(ctx, nil)
				Expect(err).ToNot(HaveOccurred())
				Expect(table).ToNot(BeNil())

				// Verify column definitions
				Expect(table.ColumnDefinitions).To(HaveLen(4))
				Expect(table.ColumnDefinitions[0].Name).To(Equal("Name"))
				Expect(table.ColumnDefinitions[1].Name).To(Equal("Created At"))

				// Verify rows
				Expect(table.Rows).To(HaveLen(1))
				row := table.Rows[0]
				Expect(row.Cells).To(HaveLen(4))
				Expect(row.Cells[0]).To(Equal("table-conversion.type-test"))
				Expect(row.Cells[1]).To(Equal(artifactType.CreationTimestamp))
				Expect(row.Cells[2]).To(Equal(2))                   // Parameter count
				Expect(row.Cells[3]).To(Equal("table.workflow-v3")) // Workflow name

				// Verify resource version
				Expect(table.ResourceVersion).To(Equal("12345"))
			})

			It("should convert ArtifactType with no parameters", func() {
				artifactType := &arc.ArtifactType{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "no-params-artifact",
						Namespace: "default",
					},
					Spec: arc.ArtifactTypeSpec{
						WorkflowTemplateRef: arc.ArtifactTypeTemplateRef{
							Name: "simple.workflow",
						},
					},
				}

				table, err := artifactType.ConvertToTable(ctx, nil)
				Expect(err).ToNot(HaveOccurred())
				Expect(table).ToNot(BeNil())
				Expect(table.Rows).To(HaveLen(1))

				row := table.Rows[0]
				Expect(row.Cells[2]).To(Equal(0))                 // Parameter count
				Expect(row.Cells[3]).To(Equal("simple.workflow")) // Workflow name
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
							Spec: arc.ArtifactTypeSpec{
								Parameters: []arc.ArtifactWorkflowParameter{
									{Name: "log.level", Value: "debug"},
								},
								WorkflowTemplateRef: arc.ArtifactTypeTemplateRef{
									Name: "logging-workflow.alpha",
								},
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
				Expect(row.Cells[2]).To(Equal(1))                        // Parameter count
				Expect(row.Cells[3]).To(Equal("logging-workflow.alpha")) // Workflow name

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
							Spec: arc.ArtifactTypeSpec{
								WorkflowTemplateRef: arc.ArtifactTypeTemplateRef{
									Name: "batch.workflow-1",
								},
							},
						},
						{
							ObjectMeta: metav1.ObjectMeta{
								Name:              "type-2",
								Namespace:         "default",
								CreationTimestamp: metav1.Now(),
							},
							Spec: arc.ArtifactTypeSpec{
								Parameters: []arc.ArtifactWorkflowParameter{
									{Name: "concurrency.max", Value: "10"},
									{Name: "batch.size", Value: "100"},
								},
								WorkflowTemplateRef: arc.ArtifactTypeTemplateRef{
									Name: "batch.workflow-2",
								},
							},
						},
						{
							ObjectMeta: metav1.ObjectMeta{
								Name:              "type-3",
								Namespace:         "production",
								CreationTimestamp: metav1.Now(),
							},
							Spec: arc.ArtifactTypeSpec{
								Parameters: []arc.ArtifactWorkflowParameter{
									{Name: "connection.timeout_ms", Value: "30000"},
									{Name: "max_retries", Value: "5"},
									{Name: "processing.mode", Value: "async"},
								},
								WorkflowTemplateRef: arc.ArtifactTypeTemplateRef{
									Name: "batch.workflow-3",
								},
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

				// Verify second row
				Expect(table.Rows[1].Cells[0]).To(Equal("type-2"))

				// Verify third row
				Expect(table.Rows[2].Cells[0]).To(Equal("type-3"))

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
							Spec: arc.ArtifactTypeSpec{
								WorkflowTemplateRef: arc.ArtifactTypeTemplateRef{
									Name: "paginated-workflow.test",
								},
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
						Name: "cluster-artifact-no.params",
					},
					Spec: arc.ArtifactTypeSpec{
						Parameters: []arc.ArtifactWorkflowParameter{},
						WorkflowTemplateRef: arc.ArtifactTypeTemplateRef{
							Name:         "cluster.workflow-global",
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
						Name: "cluster-artifact.type-valid",
					},
					Spec: arc.ArtifactTypeSpec{
						Parameters: []arc.ArtifactWorkflowParameter{
							{Name: "registry.url", Value: "https://registry.example.com:5000"},
							{Name: "auth.token_expiry", Value: "3600"},
						},
						WorkflowTemplateRef: arc.ArtifactTypeTemplateRef{
							Name: "registry.workflow-template",
						},
					},
				}

				errs := clusterArtifactType.Validate(ctx)
				Expect(errs).To(BeEmpty())
			})

			It("should reject ClusterArtifactType with duplicate parameters", func() {
				clusterArtifactType := &arc.ClusterArtifactType{
					ObjectMeta: metav1.ObjectMeta{
						Name: "duplicate.cluster-type",
					},
					Spec: arc.ArtifactTypeSpec{
						Parameters: []arc.ArtifactWorkflowParameter{
							{Name: "image.tag", Value: "v1.0.0"},
							{Name: "image.tag", Value: "v2.0.0"},
						},
						WorkflowTemplateRef: arc.ArtifactTypeTemplateRef{
							Name: "image-workflow.template",
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
						Name: "empty-types-validation.test",
					},
					Spec: arc.ArtifactTypeSpec{
						Rules: arc.ArtifactTypeRules{
							SrcTypes: []string{"http"},
							DstTypes: []string{"s3"},
						},
						WorkflowTemplateRef: arc.ArtifactTypeTemplateRef{
							Name: "validation.workflow-cluster",
						},
					},
				}

				newClusterArtifactType := &arc.ClusterArtifactType{
					ObjectMeta: metav1.ObjectMeta{
						Name: "empty-types-validation.test",
					},
					Spec: arc.ArtifactTypeSpec{
						Rules: arc.ArtifactTypeRules{
							SrcTypes: []string{"", "http"},
							DstTypes: []string{"s3", ""},
						},
						WorkflowTemplateRef: arc.ArtifactTypeTemplateRef{
							Name: "validation.workflow-cluster",
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
						Name: "params-update-validation.test",
					},
					Spec: arc.ArtifactTypeSpec{
						Parameters: []arc.ArtifactWorkflowParameter{
							{Name: "webhook.url", Value: "https://webhook.example.com/notify"},
						},
						WorkflowTemplateRef: arc.ArtifactTypeTemplateRef{
							Name: "notification.workflow-template",
						},
					},
				}

				newClusterArtifactType := &arc.ClusterArtifactType{
					ObjectMeta: metav1.ObjectMeta{
						Name: "params-update-validation.test",
					},
					Spec: arc.ArtifactTypeSpec{
						Parameters: []arc.ArtifactWorkflowParameter{
							{Name: "", Value: "oci://registry.io/repo:tag"},
							{Name: "destination.path", Value: "/var/artifacts/"},
						},
						WorkflowTemplateRef: arc.ArtifactTypeTemplateRef{
							Name: "notification.workflow-template",
						},
					},
				}

				errs := newClusterArtifactType.ValidateUpdate(ctx, oldClusterArtifactType)
				Expect(errs).ToNot(BeEmpty())
			})

			It("should require workflow template reference name", func() {
				oldClusterArtifactType := &arc.ClusterArtifactType{
					ObjectMeta: metav1.ObjectMeta{
						Name: "template-ref-required.test",
					},
					Spec: arc.ArtifactTypeSpec{
						WorkflowTemplateRef: arc.ArtifactTypeTemplateRef{
							Name: "existing.workflow-ref",
						},
					},
				}

				newClusterArtifactType := &arc.ClusterArtifactType{
					ObjectMeta: metav1.ObjectMeta{
						Name: "template-ref-required.test",
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
