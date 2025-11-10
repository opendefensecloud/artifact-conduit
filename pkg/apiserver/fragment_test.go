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

var _ = Describe("Fragment", func() {
    var (
        ctx   = envtest.Context()
        ns    = SetupTest(ctx)
        frag  = &arcv1alpha1.Fragment{}
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
