// Copyright BWI GmbH and Artifact Conduit contributors
// SPDX-License-Identifier: Apache-2.0

package controller

import (
	"strings"
	"time"

	wfv1alpha1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/validation"

	arcv1alpha1 "go.opendefense.cloud/arc/api/arc/v1alpha1"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
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
		const sha = "0123456789abcdef"

		It("should generate artifact workflow name", func() {
			order := &arcv1alpha1.Order{
				Name: "test-order",
			}

			result := awName(order, sha)
			Expect(result).To(Equal("test-order-" + sha))
		})

		It("should shorten long order names to a valid DNS-1123 subdomain", func() {
			order := &arcv1alpha1.Order{
				// A valid 253 character DNS-1123 subdomain.
				Name: strings.Repeat("a", 63) + "." +
					strings.Repeat("b", 63) + "." +
					strings.Repeat("c", 63) + "." +
					strings.Repeat("d", 61),
			}
			Expect(validation.IsDNS1123Subdomain(order.Name)).To(BeEmpty())

			result := awName(order, sha)
			Expect(len(result)).To(BeNumerically("<=", validation.DNS1123SubdomainMaxLength))
			Expect(validation.IsDNS1123Subdomain(result)).To(BeEmpty())
			Expect(result).To(HaveSuffix("-" + sha))
		})

		It("should not join the name hash to a trailing dot", func() {
			maxlen := validation.DNS1123SubdomainMaxLength - len(sha) - 1
			keep := maxlen - awNameHashLength - 1
			// Create a name longer than maxlen but still valid as DNS-1123 subdomain
			name := strings.Repeat("a", keep-1) + "." + strings.Repeat("b", maxlen-keep+10)
			order := &arcv1alpha1.Order{Name: name}
			Expect(validation.IsDNS1123Subdomain(name)).To(BeEmpty())

			result := awName(order, sha)
			Expect(result).NotTo(ContainSubstring(".-"))
			Expect(result).To(HavePrefix(strings.Repeat("a", keep-1) + "-"))
			Expect(validation.IsDNS1123Subdomain(result)).To(BeEmpty())
		})

		It("should keep shortened names deterministic and distinct", func() {
			prefix := strings.Repeat("a", 20)
			first := &arcv1alpha1.Order{Name: prefix + strings.Repeat("b", 40)}
			second := &arcv1alpha1.Order{Name: prefix + strings.Repeat("c", 40)}

			firstResult := awName(first, sha)
			Expect(awName(first, sha)).To(Equal(firstResult))
			Expect(awName(second, sha)).NotTo(Equal(firstResult))
			Expect(validation.IsDNS1123Subdomain(firstResult)).To(BeEmpty())
			Expect(validation.IsDNS1123Subdomain(awName(second, sha))).To(BeEmpty())
		})
	})

	Describe("shortenName", func() {
		It("should shorten a long name by keeping prefix and appending hash", func() {
			name := strings.Repeat("a", 100)
			result := shortenName(name, 50)
			Expect(result).To(HaveLen(50))
			Expect(validation.IsDNS1123Subdomain(result)).To(BeEmpty())
		})

		It("should be deterministic for the same input", func() {
			name := strings.Repeat("b", 80)
			result1 := shortenName(name, 40)
			result2 := shortenName(name, 40)
			Expect(result1).To(Equal(result2))
		})

		It("should produce different results for different names with same prefix", func() {
			prefix := strings.Repeat("c", 20)
			first := prefix + strings.Repeat("d", 60)
			second := prefix + strings.Repeat("e", 60)
			result1 := shortenName(first, 50)
			result2 := shortenName(second, 50)
			Expect(result1).NotTo(Equal(result2))
			Expect(validation.IsDNS1123Subdomain(result1)).To(BeEmpty())
			Expect(validation.IsDNS1123Subdomain(result2)).To(BeEmpty())
		})

		It("should not leave trailing dot or dash before the hash", func() {
			// Force trimming by creating name that would end with separator
			name := strings.Repeat("a.", 50)
			result := shortenName(name, 40)
			Expect(result).NotTo(ContainSubstring(".-"))
			Expect(result).NotTo(MatchRegexp(`-\-[0-9a-f]+$`))
			Expect(validation.IsDNS1123Subdomain(result)).To(BeEmpty())
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
				Namespace: "test-ns",
				Name:      "test-order",
				Labels: map[string]string{
					"app": "test",
				},
			}

			result := awObjectMeta(order, "sha123", "oci")
			Expect(result.Namespace).To(Equal("test-ns"))
			Expect(result.Name).To(Equal("test-order-sha123"))
			Expect(result.Labels).To(HaveKeyWithValue("app", "test"))
			Expect(result.Labels).To(HaveKeyWithValue(arcv1alpha1.LabelArtifactType, "oci"))
		})

		It("should stamp the artifact type label", func() {
			order := &arcv1alpha1.Order{
				Namespace: "team-a",
				Name:      "nightly",
				Labels:    map[string]string{"owner": "platform"},
			}

			meta := awObjectMeta(order, "abc123", "oci")

			Expect(meta.Namespace).To(Equal("team-a"))
			Expect(meta.Name).To(Equal("nightly-abc123"))
			Expect(meta.Labels).To(HaveKeyWithValue("owner", "platform"))
			Expect(meta.Labels).To(HaveKeyWithValue(arcv1alpha1.LabelArtifactType, "oci"))
		})

		It("should not mutate the order's own labels", func() {
			order := &arcv1alpha1.Order{
				Namespace: "team-a",
				Name:      "nightly",
				Labels:    map[string]string{"owner": "platform"},
			}

			awObjectMeta(order, "abc123", "oci")

			Expect(order.Labels).To(HaveLen(1))
			Expect(order.Labels).NotTo(HaveKey(arcv1alpha1.LabelArtifactType))
		})

		It("should skip the artifact type label when the value is not a valid label", func() {
			order := &arcv1alpha1.Order{
				Namespace: "team-a",
				Name:      "nightly",
			}

			// 64 characters, one over the label value limit. The workflow still
			// has to be creatable, so the label is left off rather than
			// truncated to a type that does not exist.
			meta := awObjectMeta(order, "abc123", strings.Repeat("a", 64))

			Expect(meta.Name).To(Equal("nightly-abc123"))
			Expect(meta.Labels).NotTo(HaveKey(arcv1alpha1.LabelArtifactType))
		})

		It("should work when the order has no labels", func() {
			order := &arcv1alpha1.Order{
				Namespace: "team-a",
				Name:      "nightly",
			}

			meta := awObjectMeta(order, "abc123", "helm")

			Expect(meta.Labels).To(HaveKeyWithValue(arcv1alpha1.LabelArtifactType, "helm"))
		})

		It("should drop inherited artifact type label when artifact type is invalid", func() {
			longType := strings.Repeat("x", 64)
			order := &arcv1alpha1.Order{
				Namespace: "team-a",
				Name:      "nightly",
				Labels: map[string]string{
					"owner":                       "platform",
					arcv1alpha1.LabelArtifactType: "inherited-type",
				},
			}

			meta := awObjectMeta(order, "abc123", longType)

			Expect(meta.Namespace).To(Equal("team-a"))
			Expect(meta.Name).To(Equal("nightly-abc123"))
			Expect(meta.Labels).To(HaveKeyWithValue("owner", "platform"))
			Expect(meta.Labels).NotTo(HaveKey(arcv1alpha1.LabelArtifactType))
		})
	})

	Describe("workflowObjectMeta", func() {
		It("should create workflow object metadata from artifact workflow", func() {
			aw := &arcv1alpha1.ArtifactWorkflow{
				Namespace: "test-ns",
				Name:      "test-aw",
				Labels: map[string]string{
					"key": "value",
				},
				Annotations: map[string]string{
					"annotation": "data",
				},
			}

			result := workflowObjectMeta(aw)
			Expect(result.Namespace).To(Equal("test-ns"))
			Expect(result.Name).To(Equal("test-aw"))
			Expect(result.Labels).To(HaveKeyWithValue("key", "value"))
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

			Expect(dst).To(HaveKeyWithValue("specConfigPort", 8080))
			Expect(dst).To(HaveKeyWithValue("specConfigHost", "localhost"))
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

			Expect(dst).To(HaveKeyWithValue("specLevel1Level2Value", "deep"))
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
					AnnotationForceAt: "1700000000",
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
					AnnotationForceAt: "not-a-number",
				},
			}

			_, err := GetForceAtAnnotationValue(obj)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("invalid force reconcile annotation"))
		})

		It("should return error for a value the status cannot record", func() {
			// Milliseconds pasted for seconds: roughly year 3170000.
			obj := &metav1.ObjectMeta{
				Annotations: map[string]string{
					AnnotationForceAt: "99999999999999",
				},
			}

			_, err := GetForceAtAnnotationValue(obj)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("invalid force reconcile annotation"))
		})
	})

	DescribeTable("recordableForceAt judges the year 9999 boundary in UTC whatever the zone",
		func(offset int, unix int64, recordable bool) {
			forceAt := time.Unix(unix, 0).In(time.FixedZone("test", offset))
			Expect(recordableForceAt(forceAt)).To(Equal(recordable))
		},
		// 253402300799 is 9999-12-31T23:59:59Z, the last second metav1.Time can hold.
		Entry("last recordable second, east of UTC", 3600, int64(253402300799), true),
		Entry("first unrecordable second, west of UTC", -5*3600, int64(253402300800), false),
	)

	Describe("earliestRequeue", func() {
		DescribeTable("should return the soonest positive duration",
			func(a, b, want time.Duration) {
				Expect(earliestRequeue(a, b)).To(Equal(want))
			},
			Entry("both zero", time.Duration(0), time.Duration(0), time.Duration(0)),
			Entry("first zero", time.Duration(0), time.Minute, time.Minute),
			Entry("second zero", time.Minute, time.Duration(0), time.Minute),
			Entry("first negative", -time.Second, time.Minute, time.Minute),
			Entry("second negative", time.Minute, -time.Second, time.Minute),
			Entry("first sooner", time.Second, time.Minute, time.Second),
			Entry("second sooner", time.Minute, time.Second, time.Second),
			Entry("equal", time.Minute, time.Minute, time.Minute),
		)
	})
})
