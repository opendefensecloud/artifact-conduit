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
				Expect(table.ColumnDefinitions).To(HaveLen(7))
				Expect(table.ColumnDefinitions[0].Name).To(Equal("Name"))
				Expect(table.ColumnDefinitions[1].Name).To(Equal("Created At"))
				Expect(table.ColumnDefinitions[2].Name).To(Equal("Remote URL"))
				Expect(table.ColumnDefinitions[3].Name).To(Equal("Usage"))
				Expect(table.ColumnDefinitions[4].Name).To(Equal("Secret"))
				Expect(table.ColumnDefinitions[5].Name).To(Equal("Ready"))
				Expect(table.ColumnDefinitions[6].Name).To(Equal("Message"))

				// Verify rows
				Expect(table.Rows).To(HaveLen(1))
				row := table.Rows[0]
				Expect(row.Cells).To(HaveLen(7))
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

	Describe("PrepareForCreate", func() {
		It("should set generation to 1", func() {
			endpoint := &arc.Endpoint{
				ObjectMeta: metav1.ObjectMeta{Name: "test-endpoint", Namespace: "default"},
				Spec:       arc.EndpointSpec{RemoteURL: "https://example.com"},
			}

			endpoint.PrepareForCreate(ctx)

			Expect(endpoint.Generation).To(Equal(int64(1)))
		})
	})

	Describe("PrepareForUpdate", func() {
		It("should increment generation when the spec changed", func() {
			old := &arc.Endpoint{
				ObjectMeta: metav1.ObjectMeta{Name: "test-endpoint", Namespace: "default", Generation: 3},
				Spec:       arc.EndpointSpec{RemoteURL: "https://old.example.com"},
			}
			updated := &arc.Endpoint{
				ObjectMeta: metav1.ObjectMeta{Name: "test-endpoint", Namespace: "default", Generation: 3},
				Spec:       arc.EndpointSpec{RemoteURL: "https://new.example.com"},
			}

			updated.PrepareForUpdate(ctx, old)

			Expect(updated.Generation).To(Equal(int64(4)))
		})

		It("should not increment generation when the spec is unchanged", func() {
			old := &arc.Endpoint{
				ObjectMeta: metav1.ObjectMeta{Name: "test-endpoint", Namespace: "default", Generation: 3},
				Spec:       arc.EndpointSpec{RemoteURL: "https://example.com"},
			}
			updated := &arc.Endpoint{
				ObjectMeta: metav1.ObjectMeta{Name: "test-endpoint", Namespace: "default", Generation: 3},
				Status:     arc.EndpointStatus{ObservedGeneration: 3},
				Spec:       arc.EndpointSpec{RemoteURL: "https://example.com"},
			}

			updated.PrepareForUpdate(ctx, old)

			Expect(updated.Generation).To(Equal(int64(3)))
		})

		It("should not let a stale or zeroed incoming generation reset the count", func() {
			old := &arc.Endpoint{
				ObjectMeta: metav1.ObjectMeta{Name: "test-endpoint", Namespace: "default", Generation: 5},
				Spec:       arc.EndpointSpec{RemoteURL: "https://example.com"},
			}

			unchangedSpec := &arc.Endpoint{
				ObjectMeta: metav1.ObjectMeta{Name: "test-endpoint", Namespace: "default", Generation: 0},
				Spec:       arc.EndpointSpec{RemoteURL: "https://example.com"},
			}
			unchangedSpec.PrepareForUpdate(ctx, old)
			Expect(unchangedSpec.Generation).To(Equal(int64(5)))

			changedSpec := &arc.Endpoint{
				ObjectMeta: metav1.ObjectMeta{Name: "test-endpoint", Namespace: "default", Generation: 0},
				Spec:       arc.EndpointSpec{RemoteURL: "https://changed.example.com"},
			}
			changedSpec.PrepareForUpdate(ctx, old)
			Expect(changedSpec.Generation).To(Equal(int64(6)))
		})
	})

	Describe("CopyStatusTo", func() {
		It("should copy status onto the target Endpoint", func() {
			src := &arc.Endpoint{
				Status: arc.EndpointStatus{
					ObservedGeneration: 7,
					Conditions: []metav1.Condition{{
						Type:   arc.EndpointConditionReady,
						Status: metav1.ConditionTrue,
						Reason: "Valid",
					}},
				},
			}
			dst := &arc.Endpoint{}

			src.CopyStatusTo(dst)

			Expect(dst.Status.ObservedGeneration).To(Equal(int64(7)))
			Expect(dst.Status.Conditions).To(HaveLen(1))
			Expect(dst.Status.Conditions[0].Type).To(Equal(arc.EndpointConditionReady))
		})

		It("should ignore a target of a different type", func() {
			src := &arc.Endpoint{Status: arc.EndpointStatus{ObservedGeneration: 7}}
			other := &arc.Order{}

			Expect(func() { src.CopyStatusTo(other) }).NotTo(Panic())
		})
	})

	Describe("ConvertToTable", func() {
		newEndpoint := func(conds ...metav1.Condition) *arc.Endpoint {
			return &arc.Endpoint{
				ObjectMeta: metav1.ObjectMeta{Name: "test-endpoint", Namespace: "default"},
				Spec: arc.EndpointSpec{
					RemoteURL: "https://example.com",
					Type:      "oci",
					Usage:     arc.EndpointUsageAll,
				},
				Status: arc.EndpointStatus{Conditions: conds},
			}
		}

		It("should report Unknown when no Ready condition is present", func() {
			table, err := newEndpoint().ConvertToTable(ctx, nil)

			Expect(err).NotTo(HaveOccurred())
			Expect(table.ColumnDefinitions).To(HaveLen(7))
			Expect(table.ColumnDefinitions[5].Name).To(Equal("Ready"))
			Expect(table.ColumnDefinitions[6].Name).To(Equal("Message"))
			Expect(table.Rows[0].Cells[5]).To(Equal("Unknown"))
			Expect(table.Rows[0].Cells[6]).To(Equal(""))
		})

		It("should surface the Ready condition status and message", func() {
			table, err := newEndpoint(metav1.Condition{
				Type:    arc.EndpointConditionReady,
				Status:  metav1.ConditionFalse,
				Reason:  "Unauthorized",
				Message: "auth failed: 401",
			}).ConvertToTable(ctx, nil)

			Expect(err).NotTo(HaveOccurred())
			Expect(table.Rows[0].Cells[5]).To(Equal("False"))
			Expect(table.Rows[0].Cells[6]).To(Equal("auth failed: 401"))
		})
	})
})
