// Copyright 2025 BWI GmbH and Artifact Conduit contributors
// SPDX-License-Identifier: Apache-2.0

package controller

// import (
// 	. "github.com/onsi/ginkgo/v2"
// 	. "github.com/onsi/gomega"
// 	arcv1alpha1 "go.opendefense.cloud/arc/api/arc/v1alpha1"
// 	"go.opendefense.cloud/arc/pkg/envtest"
// 	corev1 "k8s.io/api/core/v1"
// 	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
// 	"k8s.io/apimachinery/pkg/runtime"
// 	"k8s.io/apimachinery/pkg/types"
// )

// var _ = Describe("ArtifactWorkflowController", func() {
// 	var (
// 		ctx = envtest.Context()
// 		ns  = SetupTest(ctx)
// 	)

// 	Context("when reconciling ArtifactWorkflows", func() {
// 		It("should create workflowConfig secret for a fragment and a workflow", func() {
// 			srcEndpointAuth := &corev1.Secret{
// 				ObjectMeta: metav1.ObjectMeta{
// 					Name:      "test-endpoint-auth-a",
// 					Namespace: ns.Name,
// 				},
// 			}
// 			Expect(k8sClient.Create(ctx, srcEndpointAuth)).To(Succeed())

// 			srcEndpoint := &arcv1alpha1.Endpoint{
// 				ObjectMeta: metav1.ObjectMeta{
// 					Name:      "test-endpoint-a",
// 					Namespace: ns.Name,
// 				},
// 				Spec: arcv1alpha1.EndpointSpec{
// 					Type:      "test-type-1",
// 					RemoteURL: "example.com",
// 					Usage:     arcv1alpha1.EndpointUsageAll,
// 					SecretRef: corev1.LocalObjectReference{
// 						Name: srcEndpointAuth.Name,
// 					},
// 				},
// 			}
// 			Expect(k8sClient.Create(ctx, srcEndpoint)).To(Succeed())

// 			dstEndpoint := &arcv1alpha1.Endpoint{
// 				ObjectMeta: metav1.ObjectMeta{
// 					Name:      "test-endpoint-ab",
// 					Namespace: ns.Name,
// 				},
// 				Spec: arcv1alpha1.EndpointSpec{
// 					Type:      "test-type-1",
// 					RemoteURL: "example.com",
// 					Usage:     arcv1alpha1.EndpointUsageAll,
// 				},
// 			}
// 			Expect(k8sClient.Create(ctx, dstEndpoint)).To(Succeed())

// 			// Create test Order with multiple artifacts, no defaults
// 			fragment := &arcv1alpha1.ArtifactWorkflow{
// 				ObjectMeta: metav1.ObjectMeta{
// 					Name:      "test-fragment",
// 					Namespace: ns.Name,
// 				},
// 				Spec: arcv1alpha1.ArtifactWorkflowSpec{
// 					Type:   "test-type-1",
// 					SrcRef: corev1.LocalObjectReference{Name: srcEndpoint.Name},
// 					DstRef: corev1.LocalObjectReference{Name: dstEndpoint.Name},
// 					Spec:   runtime.RawExtension{Raw: []byte(`{"key":"value1"}`)},
// 				},
// 			}
// 			Expect(k8sClient.Create(ctx, fragment)).To(Succeed())

// 			// Verify workflowConfig secret was created
// 			workflowConfig := &corev1.Secret{}
// 			Eventually(func() error {
// 				return k8sClient.Get(ctx, types.NamespacedName{Namespace: fragment.Namespace, Name: fragment.Name}, workflowConfig)
// 			}).Should(Succeed())

// 			// Verify workflowConfig contents
// 			s, ok := workflowConfig.Data[workflowConfigSecretKey]
// 			Expect(ok).To(BeTrue())
// 			GinkgoWriter.Println(string(s))
// 		})
// 	})
// })
