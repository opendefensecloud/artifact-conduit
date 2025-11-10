// Copyright 2025 BWI GmbH and Artefact Conduit contributors
// SPDX-License-Identifier: Apache-2.0

package apiserver_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	arcv1alpha1 "gitlab.opencode.de/bwi/ace/artifact-conduit/api/arc/v1alpha1"
	"gitlab.opencode.de/bwi/ace/artifact-conduit/pkg/envtest"
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
