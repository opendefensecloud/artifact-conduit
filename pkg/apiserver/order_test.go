// Copyright 2025 BWI GmbH and Artefact Conduit contributors
// SPDX-License-Identifier: Apache-2.0

package apiserver_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	orderv1alpha1 "gitlab.opencode.de/bwi/ace/artifact-conduit/api/order/v1alpha1"
	"gitlab.opencode.de/bwi/ace/artifact-conduit/pkg/envtest"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var _ = Describe("Order", func() {
	var (
		ctx   = envtest.Context()
		ns    = SetupTest(ctx)
		order = &orderv1alpha1.Order{}
	)

	Context("Order", func() {
		It("should allow creating an order", func() {
			By("creating a test order")
			expectedRaw := []byte(`{"oci":[]}`)
			order = &orderv1alpha1.Order{
				ObjectMeta: metav1.ObjectMeta{
					Namespace:    ns.Name,
					GenerateName: "test-",
				},
				Spec: orderv1alpha1.OrderSpec{
					RawExtension: runtime.RawExtension{
						Raw: expectedRaw,
					},
				},
			}
			Expect(k8sClient.Create(ctx, order)).To(Succeed())
			Expect(k8sClient.Get(ctx, client.ObjectKeyFromObject(order), order)).To(Succeed())
			Expect(order.Spec.Raw).To(Equal(expectedRaw))
		})
		It("should allow deleting an order", func() {
			By("deleting a test order")
			Expect(k8sClient.Delete(ctx, order)).To(Succeed())
		})
	})

})
