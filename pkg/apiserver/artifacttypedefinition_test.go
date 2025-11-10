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

var _ = Describe("ArtifactTypeDefinition", func() {
    var (
        ctx   = envtest.Context()
        ns    = SetupTest(ctx)
        atd   = &arcv1alpha1.ArtifactTypeDefinition{}
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
