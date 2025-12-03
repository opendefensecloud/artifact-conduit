// Copyright 2025 BWI GmbH and Artifact Conduit contributors
// SPDX-License-Identifier: Apache-2.0

package controller

import (
	"time"

	wfv1alpha1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	arcv1alpha1 "go.opendefense.cloud/arc/api/arc/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

var _ = Describe("Helper Functions", func() {
	Describe("generatePodNameFromNodeStatus", func() {
		It("should generate pod name from node status", func() {
			node := wfv1alpha1.NodeStatus{
				ID:           "workflow-abc123-def456",
				BoundaryID:   "workflow-abc123",
				TemplateName: "step-name",
			}

			result := generatePodNameFromNodeStatus(node)
			Expect(result).To(Equal("workflow-abc123-step-name-def456"))
		})

		It("should handle node ID without dashes", func() {
			node := wfv1alpha1.NodeStatus{
				ID:           "simple",
				BoundaryID:   "boundary",
				TemplateName: "display",
			}

			result := generatePodNameFromNodeStatus(node)
			Expect(result).To(Equal("boundary-display-simple"))
		})
	})

	Describe("namespacedName", func() {
		It("should create a namespaced name", func() {
			result := namespacedName("test-namespace", "test-name")
			Expect(result.Namespace).To(Equal("test-namespace"))
			Expect(result.Name).To(Equal("test-name"))
		})

		It("should handle empty values", func() {
			result := namespacedName("", "")
			Expect(result.Namespace).To(BeEmpty())
			Expect(result.Name).To(BeEmpty())
		})
	})

	Describe("awName", func() {
		It("should generate artifact workflow name", func() {
			order := &arcv1alpha1.Order{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-order",
				},
			}

			result := awName(order, "abc123")
			Expect(result).To(Equal("test-order-abc123"))
		})
	})

	Describe("cloneObjectMeta", func() {
		It("should clone object metadata with new name", func() {
			meta := metav1.ObjectMeta{
				Namespace: "test-namespace",
				Name:      "original-name",
				Labels: map[string]string{
					"app": "test",
				},
				Annotations: map[string]string{
					"key": "value",
				},
			}

			result := cloneObjectMeta(meta, "new-name")
			Expect(result.Namespace).To(Equal("test-namespace"))
			Expect(result.Name).To(Equal("new-name"))
			Expect(result.Labels).To(HaveKeyWithValue("app", "test"))
			Expect(result.Annotations).To(HaveKeyWithValue("key", "value"))
		})

		It("should handle nil labels and annotations", func() {
			meta := metav1.ObjectMeta{
				Namespace: "ns",
				Name:      "original",
			}

			result := cloneObjectMeta(meta, "new")
			Expect(result.Labels).To(BeNil())
			Expect(result.Annotations).To(BeNil())
		})
	})

	Describe("awObjectMeta", func() {
		It("should create artifact workflow object metadata", func() {
			order := &arcv1alpha1.Order{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "test-ns",
					Name:      "test-order",
					Labels: map[string]string{
						"app": "test",
					},
				},
			}

			result := awObjectMeta(order, "sha123")
			Expect(result.Namespace).To(Equal("test-ns"))
			Expect(result.Name).To(Equal("test-order-sha123"))
			Expect(result.Labels).To(HaveKeyWithValue("app", "test"))
		})
	})

	Describe("workflowObjectMeta", func() {
		It("should create workflow object metadata from artifact workflow", func() {
			aw := &arcv1alpha1.ArtifactWorkflow{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "test-ns",
					Name:      "test-aw",
					Labels: map[string]string{
						"key": "value",
					},
					Annotations: map[string]string{
						"annotation": "data",
					},
				},
			}

			result := workflowObjectMeta(aw)
			Expect(result.Namespace).To(Equal("test-ns"))
			Expect(result.Name).To(Equal("test-aw"))
			Expect(result.Labels).To(HaveKeyWithValue("key", "value"))
			Expect(result.Annotations).To(HaveKeyWithValue("annotation", "data"))
		})
	})

	Describe("flattenMap", func() {
		It("should flatten simple key-value pairs", func() {
			src := map[string]any{
				"name": "test",
				"age":  30,
			}
			dst := map[string]any{}

			flattenMap("prefix", src, dst)

			Expect(dst).To(HaveKeyWithValue("prefixName", "test"))
			Expect(dst).To(HaveKeyWithValue("prefixAge", 30))
		})

		It("should flatten nested maps", func() {
			src := map[string]any{
				"config": map[string]any{
					"port": 8080,
					"host": "localhost",
				},
			}
			dst := map[string]any{}

			flattenMap("spec", src, dst)

			Expect(dst).To(HaveKeyWithValue("specconfigPort", 8080))
			Expect(dst).To(HaveKeyWithValue("specconfigHost", "localhost"))
		})

		It("should flatten arrays with indexed keys", func() {
			src := map[string]any{
				"items": []any{"first", "second", "third"},
			}
			dst := map[string]any{}

			flattenMap("spec", src, dst)

			Expect(dst).To(HaveKeyWithValue("specItems0", "first"))
			Expect(dst).To(HaveKeyWithValue("specItems1", "second"))
			Expect(dst).To(HaveKeyWithValue("specItems2", "third"))
		})

		It("should handle empty map", func() {
			src := map[string]any{}
			dst := map[string]any{}

			flattenMap("prefix", src, dst)

			Expect(dst).To(BeEmpty())
		})

		It("should handle deeply nested structures", func() {
			src := map[string]any{
				"level1": map[string]any{
					"level2": map[string]any{
						"value": "deep",
					},
				},
			}
			dst := map[string]any{}

			flattenMap("spec", src, dst)

			Expect(dst).To(HaveKeyWithValue("speclevel1level2Value", "deep"))
		})

		It("should capitalize first letter of keys", func() {
			src := map[string]any{
				"lowerCase": "value",
			}
			dst := map[string]any{}

			flattenMap("prefix", src, dst)

			Expect(dst).To(HaveKeyWithValue("prefixLowerCase", "value"))
		})
	})

	Describe("dawToParameters", func() {
		It("should generate parameters from desiredAW", func() {
			daw := &desiredAW{
				srcEndpoint: &arcv1alpha1.Endpoint{
					Spec: arcv1alpha1.EndpointSpec{
						Type:      "oci",
						RemoteURL: "https://src.example.com",
						SecretRef: corev1.LocalObjectReference{Name: "src-secret"},
					},
				},
				dstEndpoint: &arcv1alpha1.Endpoint{
					Spec: arcv1alpha1.EndpointSpec{
						Type:      "helm",
						RemoteURL: "https://dst.example.com",
					},
				},
				artifact: &arcv1alpha1.OrderArtifact{
					Spec: runtime.RawExtension{},
				},
				typeSpec: &arcv1alpha1.ArtifactTypeSpec{},
			}

			params, err := dawToParameters(daw)
			Expect(err).NotTo(HaveOccurred())

			paramMap := make(map[string]string)
			for _, p := range params {
				paramMap[p.Name] = p.Value
			}

			Expect(paramMap).To(HaveKeyWithValue("srcType", "oci"))
			Expect(paramMap).To(HaveKeyWithValue("srcRemoteURL", "https://src.example.com"))
			Expect(paramMap).To(HaveKeyWithValue("dstType", "helm"))
			Expect(paramMap).To(HaveKeyWithValue("dstRemoteURL", "https://dst.example.com"))
			Expect(paramMap).To(HaveKeyWithValue("srcSecret", "true"))
			Expect(paramMap).To(HaveKeyWithValue("dstSecret", "false"))
		})

		It("should include parameters from artifact spec", func() {
			daw := &desiredAW{
				srcEndpoint: &arcv1alpha1.Endpoint{
					Spec: arcv1alpha1.EndpointSpec{Type: "oci", RemoteURL: "https://example.com"},
				},
				dstEndpoint: &arcv1alpha1.Endpoint{
					Spec: arcv1alpha1.EndpointSpec{Type: "oci", RemoteURL: "https://example.com"},
				},
				artifact: &arcv1alpha1.OrderArtifact{
					Spec: runtime.RawExtension{
						Raw: []byte(`{"imageName": "nginx", "tag": "latest"}`),
					},
				},
				typeSpec: &arcv1alpha1.ArtifactTypeSpec{},
			}

			params, err := dawToParameters(daw)
			Expect(err).NotTo(HaveOccurred())

			paramMap := make(map[string]string)
			for _, p := range params {
				paramMap[p.Name] = p.Value
			}

			Expect(paramMap).To(HaveKeyWithValue("specImageName", "nginx"))
			Expect(paramMap).To(HaveKeyWithValue("specTag", "latest"))
		})

		It("should allow type parameters to override artifact spec", func() {
			daw := &desiredAW{
				srcEndpoint: &arcv1alpha1.Endpoint{
					Spec: arcv1alpha1.EndpointSpec{Type: "oci", RemoteURL: "https://example.com"},
				},
				dstEndpoint: &arcv1alpha1.Endpoint{
					Spec: arcv1alpha1.EndpointSpec{Type: "oci", RemoteURL: "https://example.com"},
				},
				artifact: &arcv1alpha1.OrderArtifact{
					Spec: runtime.RawExtension{
						Raw: []byte(`{"imageName": "nginx"}`),
					},
				},
				typeSpec: &arcv1alpha1.ArtifactTypeSpec{
					Parameters: []arcv1alpha1.ArtifactWorkflowParameter{
						{Name: "specImageName", Value: "overridden"},
						{Name: "customParam", Value: "custom-value"},
					},
				},
			}

			params, err := dawToParameters(daw)
			Expect(err).NotTo(HaveOccurred())

			paramMap := make(map[string]string)
			for _, p := range params {
				paramMap[p.Name] = p.Value
			}

			Expect(paramMap).To(HaveKeyWithValue("specImageName", "overridden"))
			Expect(paramMap).To(HaveKeyWithValue("customParam", "custom-value"))
		})

		It("should return error for invalid JSON in artifact spec", func() {
			daw := &desiredAW{
				srcEndpoint: &arcv1alpha1.Endpoint{
					Spec: arcv1alpha1.EndpointSpec{Type: "oci", RemoteURL: "https://example.com"},
				},
				dstEndpoint: &arcv1alpha1.Endpoint{
					Spec: arcv1alpha1.EndpointSpec{Type: "oci", RemoteURL: "https://example.com"},
				},
				artifact: &arcv1alpha1.OrderArtifact{
					Spec: runtime.RawExtension{
						Raw: []byte(`{invalid json`),
					},
				},
				typeSpec: &arcv1alpha1.ArtifactTypeSpec{},
			}

			_, err := dawToParameters(daw)
			Expect(err).To(HaveOccurred())
		})

		It("should handle empty artifact spec", func() {
			daw := &desiredAW{
				srcEndpoint: &arcv1alpha1.Endpoint{
					Spec: arcv1alpha1.EndpointSpec{Type: "oci", RemoteURL: "https://example.com"},
				},
				dstEndpoint: &arcv1alpha1.Endpoint{
					Spec: arcv1alpha1.EndpointSpec{Type: "oci", RemoteURL: "https://example.com"},
				},
				artifact: &arcv1alpha1.OrderArtifact{
					Spec: runtime.RawExtension{Raw: nil},
				},
				typeSpec: &arcv1alpha1.ArtifactTypeSpec{},
			}

			params, err := dawToParameters(daw)
			Expect(err).NotTo(HaveOccurred())
			Expect(params).NotTo(BeEmpty())
		})
	})

	Describe("GetForceAtAnnotationValue", func() {
		It("should return time from valid annotation", func() {
			expectedTime := time.Unix(1700000000, 0)
			obj := &metav1.ObjectMeta{
				Annotations: map[string]string{
					forceAtAnnotation: "1700000000",
				},
			}

			result, err := GetForceAtAnnotationValue(obj)
			Expect(err).NotTo(HaveOccurred())
			Expect(result).To(Equal(expectedTime))
		})

		It("should return zero time when annotation is missing", func() {
			obj := &metav1.ObjectMeta{
				Annotations: map[string]string{},
			}

			result, err := GetForceAtAnnotationValue(obj)
			Expect(err).NotTo(HaveOccurred())
			Expect(result.IsZero()).To(BeTrue())
		})

		It("should return zero time when annotations is nil", func() {
			obj := &metav1.ObjectMeta{}

			result, err := GetForceAtAnnotationValue(obj)
			Expect(err).NotTo(HaveOccurred())
			Expect(result.IsZero()).To(BeTrue())
		})

		It("should return error for invalid annotation value", func() {
			obj := &metav1.ObjectMeta{
				Annotations: map[string]string{
					forceAtAnnotation: "not-a-number",
				},
			}

			_, err := GetForceAtAnnotationValue(obj)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("invalid force reconcile annotation"))
		})
	})
})
