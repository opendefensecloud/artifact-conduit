// Copyright BWI GmbH and Artifact Conduit contributors
// SPDX-License-Identifier: Apache-2.0

package main_test

import (
	"go.opendefense.cloud/kit/envtest"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"

	arcv1alpha1 "go.opendefense.cloud/arc/api/arc/v1alpha1"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Order", func() {
	var (
		ctx   = envtest.Context()
		ns    = SetupTest(ctx)
		order = &arcv1alpha1.Order{}
	)

	Context("Order", func() {
		It("should allow creating an order", func() {
			By("creating a test order")
			order = &arcv1alpha1.Order{
				Namespace:    ns.Name,
				GenerateName: "test-",
				Spec:         arcv1alpha1.OrderSpec{},
			}
			Expect(k8sClient.Create(ctx, order)).To(Succeed())
			Expect(k8sClient.Get(ctx, client.ObjectKeyFromObject(order), order)).To(Succeed())
		})
		It("should allow deleting an order", func() {
			By("deleting a test order")
			Expect(k8sClient.Delete(ctx, order)).To(Succeed())
		})
	})

	// Order has no NameValidator, so it exercises the apiserver-kit default
	// rules. This guards the kit v0.6.0 upgrade, which introduced validation
	// of metadata.name as a DNS-1123 subdomain on create.
	Context("Order name validation (kit default)", func() {
		It("should reject names that are not DNS-1123 subdomains", func() {
			for _, name := range []string{"has space", "under_score", "Upper-Case"} {
				err := k8sClient.Create(ctx, &arcv1alpha1.Order{
					Name:      name,
					Namespace: ns.Name,
					Spec:      arcv1alpha1.OrderSpec{},
				})
				Expect(apierrors.IsInvalid(err)).To(BeTrue(), "expected name %q to be invalid, got %v", name, err)
			}
		})

		It("should accept a dotted DNS-1123 subdomain", func() {
			// A dot is allowed in a subdomain, which distinguishes the default
			// rule from the stricter DNS-1123 label rule.
			dotted := &arcv1alpha1.Order{
				Name:      "my.order",
				Namespace: ns.Name,
				Spec:      arcv1alpha1.OrderSpec{},
			}
			Expect(k8sClient.Create(ctx, dotted)).To(Succeed())
			DeferCleanup(k8sClient.Delete, ctx, dotted)
		})
	})

})

var _ = Describe("ArtifactWorkflow", func() {
	var (
		ctx  = envtest.Context()
		ns   = SetupTest(ctx)
		frag = &arcv1alpha1.ArtifactWorkflow{}
	)

	Context("ArtifactWorkflow", func() {
		It("should allow creating an artifact workflow", func() {
			By("creating a test artifact workflow")
			frag = &arcv1alpha1.ArtifactWorkflow{
				Namespace:    ns.Name,
				GenerateName: "test-",
				Spec:         arcv1alpha1.ArtifactWorkflowSpec{},
			}
			Expect(k8sClient.Create(ctx, frag)).To(Succeed())
			Expect(k8sClient.Get(ctx, client.ObjectKeyFromObject(frag), frag)).To(Succeed())
		})
		It("should allow deleting an artifact workflow", func() {
			By("deleting a test artifact workflow")
			Expect(k8sClient.Delete(ctx, frag)).To(Succeed())
		})
	})

})

var _ = Describe("Endpoint", func() {
	var (
		ctx = envtest.Context()
		ns  = SetupTest(ctx)
		ep  = &arcv1alpha1.Endpoint{}
	)

	Context("Endpoint", func() {
		It("should allow creating an endpoint", func() {
			By("creating a test endpoint")
			ep = &arcv1alpha1.Endpoint{
				Namespace:    ns.Name,
				GenerateName: "test-",
				Spec: arcv1alpha1.EndpointSpec{
					RemoteURL: "test",
				},
			}
			Expect(k8sClient.Create(ctx, ep)).To(Succeed())
			Expect(k8sClient.Get(ctx, client.ObjectKeyFromObject(ep), ep)).To(Succeed())
		})
		It("should allow deleting an endpoint", func() {
			By("deleting a test endpoint")
			Expect(k8sClient.Delete(ctx, ep)).To(Succeed())
		})
	})

})

var _ = Describe("ArtifactType", func() {
	var (
		ctx = envtest.Context()
		ns  = SetupTest(ctx)
		atd = &arcv1alpha1.ArtifactType{}
	)

	Context("ArtifactType", func() {
		It("should allow creating an artifact type definition", func() {
			By("creating a test artifact type definition")
			atd = &arcv1alpha1.ArtifactType{
				Namespace:    ns.Name,
				GenerateName: "test-",
				Spec:         arcv1alpha1.ArtifactTypeSpec{},
			}
			Expect(k8sClient.Create(ctx, atd)).To(Succeed())
			Expect(k8sClient.Get(ctx, client.ObjectKeyFromObject(atd), atd)).To(Succeed())
		})
		It("should allow deleting an artifact type definition", func() {
			By("deleting a test artifact type definition")
			Expect(k8sClient.Delete(ctx, atd)).To(Succeed())
		})
	})

})

var _ = Describe("ClusterArtifactType", func() {
	var (
		ctx = envtest.Context()
		atd = &arcv1alpha1.ClusterArtifactType{}
	)

	Context("ClusterArtifactType", func() {
		It("should allow creating an artifact type definition", func() {
			By("creating a test artifact type definition")
			atd = &arcv1alpha1.ClusterArtifactType{
				GenerateName: "test-",
				Spec:         arcv1alpha1.ArtifactTypeSpec{},
			}
			Expect(k8sClient.Create(ctx, atd)).To(Succeed())
			Expect(k8sClient.Get(ctx, client.ObjectKeyFromObject(atd), atd)).To(Succeed())
		})
		It("should allow deleting an artifact type definition", func() {
			By("deleting a test artifact type definition")
			Expect(k8sClient.Delete(ctx, atd)).To(Succeed())
		})
	})

})
