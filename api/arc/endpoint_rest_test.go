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

var _ = Describe("Endpoint Strategy", func() {
	var (
		ctx context.Context
	)

	BeforeEach(func() {
		ctx = context.Background()
	})

	Describe("Validate", func() {
		Context("when validating remoteURL", func() {
			It("should accept Endpoint with valid remoteURL", func() {
				endpoint := &arc.Endpoint{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-endpoint",
						Namespace: "default",
					},
					Spec: arc.EndpointSpec{
						RemoteURL: "https://example.com/api",
						Type:      "http",
					},
				}

				errs := endpoint.Validate(ctx)
				Expect(errs).To(BeEmpty())
			})

			It("should reject Endpoint with empty remoteURL", func() {
				endpoint := &arc.Endpoint{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-endpoint",
						Namespace: "default",
					},
					Spec: arc.EndpointSpec{
						RemoteURL: "",
						Type:      "http",
					},
				}

				errs := endpoint.Validate(ctx)
				Expect(errs).To(HaveLen(1))
				Expect(errs[0].Type).To(Equal(field.ErrorTypeRequired))
				Expect(errs[0].Field).To(Equal("spec.remoteURL"))
			})

			It("should accept Endpoint with various valid URLs", func() {
				testURLs := []string{
					"https://example.com",
					"http://localhost:8080",
					"oci://registry.example.com/path",
					"s3://bucket-name/key",
				}

				for _, url := range testURLs {
					endpoint := &arc.Endpoint{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "test-endpoint",
							Namespace: "default",
						},
						Spec: arc.EndpointSpec{
							RemoteURL: url,
						},
					}
					errs := endpoint.Validate(ctx)
					Expect(errs).To(BeEmpty(), "should accept URL: "+url)
				}
			})
		})

		Context("when validating optional fields", func() {
			It("should accept Endpoint with only remoteURL", func() {
				endpoint := &arc.Endpoint{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-endpoint",
						Namespace: "default",
					},
					Spec: arc.EndpointSpec{
						RemoteURL: "https://example.com",
					},
				}

				errs := endpoint.Validate(ctx)
				Expect(errs).To(BeEmpty())
			})

			It("should accept Endpoint with all fields populated", func() {
				endpoint := &arc.Endpoint{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-endpoint",
						Namespace: "default",
					},
					Spec: arc.EndpointSpec{
						RemoteURL: "https://example.com",
						Type:      "oci",
						SecretRef: corev1.LocalObjectReference{
							Name: "my-secret",
						},
						Usage: "push",
					},
				}

				errs := endpoint.Validate(ctx)
				Expect(errs).To(BeEmpty())
			})

			It("should accept Endpoint with Type set", func() {
				endpoint := &arc.Endpoint{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-endpoint",
						Namespace: "default",
					},
					Spec: arc.EndpointSpec{
						RemoteURL: "https://example.com",
						Type:      "http",
					},
				}

				errs := endpoint.Validate(ctx)
				Expect(errs).To(BeEmpty())
			})

			It("should accept Endpoint with SecretRef set", func() {
				endpoint := &arc.Endpoint{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-endpoint",
						Namespace: "default",
					},
					Spec: arc.EndpointSpec{
						RemoteURL: "https://example.com",
						SecretRef: corev1.LocalObjectReference{
							Name: "credentials",
						},
					},
				}

				errs := endpoint.Validate(ctx)
				Expect(errs).To(BeEmpty())
			})

			It("should accept Endpoint with Usage set", func() {
				endpoint := &arc.Endpoint{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-endpoint",
						Namespace: "default",
					},
					Spec: arc.EndpointSpec{
						RemoteURL: "https://example.com",
						Usage:     "pull",
					},
				}

				errs := endpoint.Validate(ctx)
				Expect(errs).To(BeEmpty())
			})
		})
	})

	Describe("ValidateUpdate", func() {
		Context("when updating Endpoint", func() {
			It("should accept update with valid remoteURL", func() {
				oldEndpoint := &arc.Endpoint{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-endpoint",
						Namespace: "default",
					},
					Spec: arc.EndpointSpec{
						RemoteURL: "https://old.example.com",
					},
				}

				newEndpoint := &arc.Endpoint{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-endpoint",
						Namespace: "default",
					},
					Spec: arc.EndpointSpec{
						RemoteURL: "https://new.example.com",
					},
				}

				errs := newEndpoint.ValidateUpdate(ctx, oldEndpoint)
				Expect(errs).To(BeEmpty())
			})

			It("should reject update with empty remoteURL", func() {
				oldEndpoint := &arc.Endpoint{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-endpoint",
						Namespace: "default",
					},
					Spec: arc.EndpointSpec{
						RemoteURL: "https://example.com",
					},
				}

				newEndpoint := &arc.Endpoint{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-endpoint",
						Namespace: "default",
					},
					Spec: arc.EndpointSpec{
						RemoteURL: "",
					},
				}

				errs := newEndpoint.ValidateUpdate(ctx, oldEndpoint)
				Expect(errs).To(HaveLen(1))
				Expect(errs[0].Type).To(Equal(field.ErrorTypeRequired))
				Expect(errs[0].Field).To(Equal("spec.remoteURL"))
			})

			It("should accept update changing other fields", func() {
				oldEndpoint := &arc.Endpoint{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-endpoint",
						Namespace: "default",
					},
					Spec: arc.EndpointSpec{
						RemoteURL: "https://example.com",
						Type:      "http",
					},
				}

				newEndpoint := &arc.Endpoint{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-endpoint",
						Namespace: "default",
					},
					Spec: arc.EndpointSpec{
						RemoteURL: "https://example.com",
						Type:      "oci",
						Usage:     "push",
					},
				}

				errs := newEndpoint.ValidateUpdate(ctx, oldEndpoint)
				Expect(errs).To(BeEmpty())
			})

			It("should accept update adding optional fields", func() {
				oldEndpoint := &arc.Endpoint{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-endpoint",
						Namespace: "default",
					},
					Spec: arc.EndpointSpec{
						RemoteURL: "https://example.com",
					},
				}

				newEndpoint := &arc.Endpoint{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-endpoint",
						Namespace: "default",
					},
					Spec: arc.EndpointSpec{
						RemoteURL: "https://example.com",
						Type:      "oci",
						SecretRef: corev1.LocalObjectReference{
							Name: "secret",
						},
						Usage: "push",
					},
				}

				errs := newEndpoint.ValidateUpdate(ctx, oldEndpoint)
				Expect(errs).To(BeEmpty())
			})

			It("should accept update removing optional fields", func() {
				oldEndpoint := &arc.Endpoint{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-endpoint",
						Namespace: "default",
					},
					Spec: arc.EndpointSpec{
						RemoteURL: "https://example.com",
						Type:      "oci",
						SecretRef: corev1.LocalObjectReference{
							Name: "secret",
						},
						Usage: "push",
					},
				}

				newEndpoint := &arc.Endpoint{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-endpoint",
						Namespace: "default",
					},
					Spec: arc.EndpointSpec{
						RemoteURL: "https://example.com",
					},
				}

				errs := newEndpoint.ValidateUpdate(ctx, oldEndpoint)
				Expect(errs).To(BeEmpty())
			})
		})
	})

	Describe("ConvertToTable", func() {
		Context("for single Endpoint", func() {
			It("should convert Endpoint to table with correct columns", func() {
				endpoint := &arc.Endpoint{
					ObjectMeta: metav1.ObjectMeta{
						Name:              "test-endpoint",
						Namespace:         "default",
						ResourceVersion:   "12345",
						CreationTimestamp: metav1.Now(),
					},
					Spec: arc.EndpointSpec{
						RemoteURL: "https://example.com/api",
						Type:      "http",
						Usage:     "push",
						SecretRef: corev1.LocalObjectReference{
							Name: "my-secret",
						},
					},
				}

				table, err := endpoint.ConvertToTable(ctx, nil)
				Expect(err).ToNot(HaveOccurred())
				Expect(table).ToNot(BeNil())

				// Verify column definitions
				Expect(table.ColumnDefinitions).To(HaveLen(5))
				Expect(table.ColumnDefinitions[0].Name).To(Equal("Name"))
				Expect(table.ColumnDefinitions[1].Name).To(Equal("Created At"))
				Expect(table.ColumnDefinitions[2].Name).To(Equal("Remote URL"))
				Expect(table.ColumnDefinitions[3].Name).To(Equal("Usage"))
				Expect(table.ColumnDefinitions[4].Name).To(Equal("Secret"))

				// Verify rows
				Expect(table.Rows).To(HaveLen(1))
				row := table.Rows[0]
				Expect(row.Cells).To(HaveLen(5))
				Expect(row.Cells[0]).To(Equal("test-endpoint"))
				Expect(row.Cells[1]).To(Equal(endpoint.CreationTimestamp))
				Expect(row.Cells[2]).To(Equal("https://example.com/api"))
				Expect(row.Cells[3]).To(Equal(arc.EndpointUsage("push")))
				Expect(row.Cells[4]).To(Equal("my-secret"))

				// Verify resource version
				Expect(table.ResourceVersion).To(Equal("12345"))
			})

			It("should convert Endpoint with minimal fields", func() {
				endpoint := &arc.Endpoint{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-endpoint",
						Namespace: "default",
					},
					Spec: arc.EndpointSpec{
						RemoteURL: "https://example.com",
					},
				}

				table, err := endpoint.ConvertToTable(ctx, nil)
				Expect(err).ToNot(HaveOccurred())
				Expect(table).ToNot(BeNil())
				Expect(table.Rows).To(HaveLen(1))

				row := table.Rows[0]
				Expect(row.Cells[0]).To(Equal("test-endpoint"))
				Expect(row.Cells[2]).To(Equal("https://example.com"))
				Expect(row.Cells[3]).To(Equal(arc.EndpointUsage(""))) // Empty usage
				Expect(row.Cells[4]).To(Equal(""))                    // Empty secret name
			})

			It("should convert Endpoint with only usage", func() {
				endpoint := &arc.Endpoint{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-endpoint",
						Namespace: "default",
					},
					Spec: arc.EndpointSpec{
						RemoteURL: "oci://registry.example.com/repo",
						Usage:     "pull",
					},
				}

				table, err := endpoint.ConvertToTable(ctx, nil)
				Expect(err).ToNot(HaveOccurred())
				Expect(table).ToNot(BeNil())
				Expect(table.Rows).To(HaveLen(1))

				row := table.Rows[0]
				Expect(row.Cells[0]).To(Equal("test-endpoint"))
				Expect(row.Cells[2]).To(Equal("oci://registry.example.com/repo"))
				Expect(row.Cells[3]).To(Equal(arc.EndpointUsage("pull")))
				Expect(row.Cells[4]).To(Equal("")) // Empty secret name
			})

			It("should convert Endpoint with only secret", func() {
				endpoint := &arc.Endpoint{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-endpoint",
						Namespace: "default",
					},
					Spec: arc.EndpointSpec{
						RemoteURL: "s3://bucket-name/path",
						SecretRef: corev1.LocalObjectReference{
							Name: "credentials",
						},
					},
				}

				table, err := endpoint.ConvertToTable(ctx, nil)
				Expect(err).ToNot(HaveOccurred())
				Expect(table).ToNot(BeNil())
				Expect(table.Rows).To(HaveLen(1))

				row := table.Rows[0]
				Expect(row.Cells[0]).To(Equal("test-endpoint"))
				Expect(row.Cells[2]).To(Equal("s3://bucket-name/path"))
				Expect(row.Cells[3]).To(Equal(arc.EndpointUsage(""))) // Empty usage
				Expect(row.Cells[4]).To(Equal("credentials"))
			})
		})
	})
})
