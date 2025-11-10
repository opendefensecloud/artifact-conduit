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

var _ = Describe("Endpoint", func() {
    var (
        ctx   = envtest.Context()
        ns    = SetupTest(ctx)
        ep    = &arcv1alpha1.Endpoint{}
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
