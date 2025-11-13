// Copyright 2025 BWI GmbH and Artifact Conduit contributors
// SPDX-License-Identifier: Apache-2.0

package apiserver_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	arcv1alpha1 "go.opendefense.cloud/arc/api/arc/v1alpha1"
	"go.opendefense.cloud/arc/pkg/envtest"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
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
				ObjectMeta: metav1.ObjectMeta{
					Namespace:    ns.Name,
					GenerateName: "test-",
				},
				Spec: arcv1alpha1.OrderSpec{},
			}
			Expect(k8sClient.Create(ctx, order)).To(Succeed())
			Expect(k8sClient.Get(ctx, client.ObjectKeyFromObject(order), order)).To(Succeed())
		})
		It("should allow deleting an order", func() {
			By("deleting a test order")
			Expect(k8sClient.Delete(ctx, order)).To(Succeed())
		})
	})

})

var _ = Describe("Fragment", func() {
	var (
		ctx  = envtest.Context()
		ns   = SetupTest(ctx)
		frag = &arcv1alpha1.Fragment{}
	)

	Context("Fragment", func() {
		It("should allow creating a fragment", func() {
			By("creating a test fragment")
			frag = &arcv1alpha1.Fragment{
				ObjectMeta: metav1.ObjectMeta{
					Namespace:    ns.Name,
					GenerateName: "test-",
				},
				Spec: arcv1alpha1.FragmentSpec{},
			}
			Expect(k8sClient.Create(ctx, frag)).To(Succeed())
			Expect(k8sClient.Get(ctx, client.ObjectKeyFromObject(frag), frag)).To(Succeed())
		})
		It("should allow deleting a fragment", func() {
			By("deleting a test fragment")
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
				ObjectMeta: metav1.ObjectMeta{
					Namespace:    ns.Name,
					GenerateName: "test-",
				},
				Spec: arcv1alpha1.EndpointSpec{},
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

var _ = Describe("ArtifactTypeDefinition", func() {
	var (
		ctx = envtest.Context()
		ns  = SetupTest(ctx)
		atd = &arcv1alpha1.ArtifactTypeDefinition{}
	)

	Context("ArtifactTypeDefinition", func() {
		It("should allow creating an artifact type definition", func() {
			By("creating a test artifact type definition")
			atd = &arcv1alpha1.ArtifactTypeDefinition{
				ObjectMeta: metav1.ObjectMeta{
					Namespace:    ns.Name,
					GenerateName: "test-",
				},
				Spec: arcv1alpha1.ArtifactTypeDefinitionSpec{},
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
