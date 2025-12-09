// Copyright 2025 BWI GmbH and Artifact Conduit contributors
// SPDX-License-Identifier: Apache-2.0

package arc_test

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.opendefense.cloud/arc/api/arc"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/validation/field"
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
})
