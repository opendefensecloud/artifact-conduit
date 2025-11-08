package apiserver_test

import (
	. "github.com/ironcore-dev/ironcore/utils/testing"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	orderv1alpha1 "gitlab.opencode.de/bwi/ace/artifact-conduit/api/order/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _ = Describe("Order", func() {
	var (
		ctx   = SetupContext()
		ns    = SetupTest(ctx)
		order = &orderv1alpha1.Order{}
	)

	Context("Order", func() {
		It("should allow creating an order", func() {
			By("creating a test order")
			order = &orderv1alpha1.Order{
				ObjectMeta: metav1.ObjectMeta{
					Namespace:    ns.Name,
					GenerateName: "test-",
				},
				Spec: orderv1alpha1.OrderSpec{},
			}
			Expect(k8sClient.Create(ctx, order)).To(Succeed())
		})
		It("should allow deleting an order", func() {
			By("deleting a test order")
			Expect(k8sClient.Delete(ctx, order)).To(Succeed())
		})
	})

})
