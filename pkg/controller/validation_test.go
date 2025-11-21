// Copyright 2025 BWI GmbH and Artifact Conduit contributors
// SPDX-License-Identifier: Apache-2.0

package controller

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	arcv1alpha1 "go.opendefense.cloud/arc/api/arc/v1alpha1"
)

var _ = Describe("ArtifactWorkflow Parameter Validation", func() {
	var reconciler *ArtifactWorkflowReconciler

	BeforeEach(func() {
		reconciler = &ArtifactWorkflowReconciler{}
	})

	Describe("validateNoDuplicateParameters", func() {
		Context("when parameters have no duplicates", func() {
			It("should return nil for unique parameters", func() {
				params := []arcv1alpha1.ArtifactWorkflowParameter{
					{Name: "param1", Value: "value1"},
					{Name: "param2", Value: "value2"},
					{Name: "param3", Value: "value3"},
				}
				err := reconciler.validateNoDuplicateParameters(params)
				Expect(err).To(BeNil())
			})

			It("should return nil for empty parameter list", func() {
				params := []arcv1alpha1.ArtifactWorkflowParameter{}
				err := reconciler.validateNoDuplicateParameters(params)
				Expect(err).To(BeNil())
			})

			It("should return nil for single parameter", func() {
				params := []arcv1alpha1.ArtifactWorkflowParameter{
					{Name: "param1", Value: "value1"},
				}
				err := reconciler.validateNoDuplicateParameters(params)
				Expect(err).To(BeNil())
			})

			It("should treat case-sensitive names as different parameters", func() {
				params := []arcv1alpha1.ArtifactWorkflowParameter{
					{Name: "Param1", Value: "value1"},
					{Name: "param1", Value: "value2"},
					{Name: "PARAM1", Value: "value3"},
				}
				err := reconciler.validateNoDuplicateParameters(params)
				Expect(err).To(BeNil())
			})
		})

		Context("when parameters have duplicates", func() {
			It("should return error for duplicate parameter names", func() {
				params := []arcv1alpha1.ArtifactWorkflowParameter{
					{Name: "param1", Value: "value1"},
					{Name: "param2", Value: "value2"},
					{Name: "param1", Value: "value3"},
				}
				err := reconciler.validateNoDuplicateParameters(params)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(Equal("duplicate parameter name found: param1"))
			})

			It("should detect the first duplicate when multiple duplicates exist", func() {
				params := []arcv1alpha1.ArtifactWorkflowParameter{
					{Name: "param1", Value: "value1"},
					{Name: "param2", Value: "value2"},
					{Name: "param2", Value: "value3"},
					{Name: "param1", Value: "value4"},
				}
				err := reconciler.validateNoDuplicateParameters(params)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(Equal("duplicate parameter name found: param2"))
			})

			It("should detect duplicate empty names", func() {
				params := []arcv1alpha1.ArtifactWorkflowParameter{
					{Name: "", Value: "value1"},
					{Name: "", Value: "value2"},
				}
				err := reconciler.validateNoDuplicateParameters(params)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(Equal("duplicate parameter name found: "))
			})
		})

		Context("when handling Order-created ArtifactWorkflows", func() {
			It("should detect srcType duplicates from Order spec flattening", func() {
				// This scenario happens when an Order's artifact spec contains
				// a parameter that conflicts with the default parameters added
				// by dawToParameters function
				params := []arcv1alpha1.ArtifactWorkflowParameter{
					{Name: "srcType", Value: "s3"},
					{Name: "srcRemoteURL", Value: "https://example.com"},
					{Name: "dstType", Value: "gcs"},
					{Name: "dstRemoteURL", Value: "https://example.org"},
					{Name: "srcSecret", Value: "true"},
					{Name: "dstSecret", Value: "true"},
					{Name: "srcType", Value: "override"}, // Duplicate from spec flattening
				}
				err := reconciler.validateNoDuplicateParameters(params)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(Equal("duplicate parameter name found: srcType"))
			})

			It("should allow Order parameters that don't conflict with defaults", func() {
				params := []arcv1alpha1.ArtifactWorkflowParameter{
					{Name: "srcType", Value: "s3"},
					{Name: "srcRemoteURL", Value: "https://example.com"},
					{Name: "dstType", Value: "gcs"},
					{Name: "dstRemoteURL", Value: "https://example.org"},
					{Name: "specCustomParam", Value: "customValue"},
					{Name: "specAnotherParam", Value: "anotherValue"},
				}
				err := reconciler.validateNoDuplicateParameters(params)
				Expect(err).To(BeNil())
			})
		})
	})
})
