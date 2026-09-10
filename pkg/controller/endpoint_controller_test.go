// Copyright 2025 BWI GmbH and Artifact Conduit contributors
// SPDX-License-Identifier: Apache-2.0

package controller

import (
	"strconv"
	"time"

	"go.opendefense.cloud/kit/envtest"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	arcv1alpha1 "go.opendefense.cloud/arc/api/arc/v1alpha1"
	"go.opendefense.cloud/arc/pkg/endpointprobe"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("EndpointController", func() {
	var (
		ctx = envtest.Context()
		ns  = setupTest(ctx)
	)

	// verdictOf reads one condition back from the API as "Status/Reason", which
	// keeps the assertions readable and the failure output useful.
	verdictOf := func(ep *arcv1alpha1.Endpoint, conditionType string) func() string {
		return func() string {
			fresh := &arcv1alpha1.Endpoint{}
			if err := k8sClient.Get(ctx, client.ObjectKeyFromObject(ep), fresh); err != nil {
				return "get-failed"
			}
			c := meta.FindStatusCondition(fresh.Status.Conditions, conditionType)
			if c == nil {
				return "absent"
			}

			return string(c.Status) + "/" + c.Reason
		}
	}

	createTypeAccepting := func(endpointType string) *arcv1alpha1.ClusterArtifactType {
		cat := &arcv1alpha1.ClusterArtifactType{
			ObjectMeta: metav1.ObjectMeta{GenerateName: "cat-"},
			Spec: arcv1alpha1.ArtifactTypeSpec{
				Rules: arcv1alpha1.ArtifactTypeRules{
					SrcTypes: []string{endpointType},
					DstTypes: []string{endpointType},
				},
				WorkflowTemplateRef: arcv1alpha1.ArtifactTypeTemplateRef{Name: "dummy"},
			},
		}
		Expect(k8sClient.Create(ctx, cat)).To(Succeed())
		DeferCleanup(k8sClient.Delete, ctx, cat)

		return cat
	}

	// createNamespacedTypeAccepting is createTypeAccepting's namespaced sibling:
	// it creates an ArtifactType, not a ClusterArtifactType, so specs can exercise
	// the InNamespace List branch of typeIsKnown and the ArtifactType watch.
	createNamespacedTypeAccepting := func(endpointType string) *arcv1alpha1.ArtifactType {
		at := &arcv1alpha1.ArtifactType{
			ObjectMeta: metav1.ObjectMeta{GenerateName: "at-", Namespace: ns.Name},
			Spec: arcv1alpha1.ArtifactTypeSpec{
				Rules: arcv1alpha1.ArtifactTypeRules{
					SrcTypes: []string{endpointType},
					DstTypes: []string{endpointType},
				},
				WorkflowTemplateRef: arcv1alpha1.ArtifactTypeTemplateRef{Name: "dummy"},
			},
		}
		Expect(k8sClient.Create(ctx, at)).To(Succeed())
		DeferCleanup(k8sClient.Delete, ctx, at)

		return at
	}

	createSecret := func(name string) *corev1.Secret {
		secret := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: ns.Name},
			StringData: map[string]string{"username": "alice", "password": "s3cret"},
		}
		Expect(k8sClient.Create(ctx, secret)).To(Succeed())

		return secret
	}

	createEndpoint := func(endpointType, secretName, remoteURL string) *arcv1alpha1.Endpoint {
		ep := &arcv1alpha1.Endpoint{
			ObjectMeta: metav1.ObjectMeta{GenerateName: "ep-", Namespace: ns.Name},
			Spec: arcv1alpha1.EndpointSpec{
				Type:      endpointType,
				RemoteURL: remoteURL,
				SecretRef: corev1.LocalObjectReference{Name: secretName},
				Usage:     arcv1alpha1.EndpointUsageAll,
			},
		}
		Expect(k8sClient.Create(ctx, ep)).To(Succeed())

		return ep
	}

	BeforeEach(func() {
		stubProbe.SetResult(endpointprobe.Result{
			Reachable: endpointprobe.Check{
				Status: metav1.ConditionTrue, Reason: endpointprobe.ReasonReachable, Message: "ok",
			},
			Authenticated: endpointprobe.Check{
				Status: metav1.ConditionTrue, Reason: endpointprobe.ReasonAuthenticated, Message: "ok",
			},
		})
	})

	It("should validate an endpoint whose secret and type resolve", func() {
		createTypeAccepting("oci")
		createSecret("creds-ok")
		ep := createEndpoint("oci", "creds-ok", "https://registry.example/ok")

		Eventually(verdictOf(ep, arcv1alpha1.EndpointConditionValidated)).
			Should(Equal("True/Valid"))
	})

	It("should record observedGeneration", func() {
		createTypeAccepting("oci")
		createSecret("creds-gen")
		ep := createEndpoint("oci", "creds-gen", "https://registry.example/gen")

		Expect(ep.Generation).To(Equal(int64(1)))

		Eventually(func() int64 {
			fresh := &arcv1alpha1.Endpoint{}
			if err := k8sClient.Get(ctx, client.ObjectKeyFromObject(ep), fresh); err != nil {
				return 0
			}

			return fresh.Status.ObservedGeneration
		}).Should(Equal(ep.Generation))
	})

	It("should report a missing secret and clear once it appears", func() {
		createTypeAccepting("oci")
		ep := createEndpoint("oci", "creds-later", "https://registry.example/later")

		Eventually(verdictOf(ep, arcv1alpha1.EndpointConditionValidated)).
			Should(Equal("False/SecretNotFound"))

		createSecret("creds-later")

		Eventually(verdictOf(ep, arcv1alpha1.EndpointConditionValidated)).
			Should(Equal("True/Valid"))
	})

	It("should report an unknown type and clear once an ArtifactType accepts it", func() {
		createSecret("creds-unknown-type")
		ep := createEndpoint("not-a-real-type", "creds-unknown-type", "https://registry.example/unknown-type")

		Eventually(verdictOf(ep, arcv1alpha1.EndpointConditionValidated)).
			Should(Equal("False/UnknownType"))

		createTypeAccepting("not-a-real-type")

		Eventually(verdictOf(ep, arcv1alpha1.EndpointConditionValidated)).
			Should(Equal("True/Valid"))
	})

	It("should respect usage when matching the type", func() {
		cat := &arcv1alpha1.ClusterArtifactType{
			ObjectMeta: metav1.ObjectMeta{GenerateName: "cat-src-only-"},
			Spec: arcv1alpha1.ArtifactTypeSpec{
				// The type is only ever a source. DstTypes has to be a non-empty
				// list that excludes it: an empty list matches any type (mirroring
				// order_controller.go), so leaving DstTypes unset would make this
				// endpoint acceptable as a destination too and defeat the test.
				Rules: arcv1alpha1.ArtifactTypeRules{
					SrcTypes: []string{"pull-only-type"},
					DstTypes: []string{"some-other-type"},
				},
				WorkflowTemplateRef: arcv1alpha1.ArtifactTypeTemplateRef{Name: "dummy"},
			},
		}
		Expect(k8sClient.Create(ctx, cat)).To(Succeed())
		DeferCleanup(k8sClient.Delete, ctx, cat)
		createSecret("creds-usage")

		ep := createEndpoint("pull-only-type", "creds-usage", "https://registry.example/usage")

		Eventually(func() error {
			fresh := &arcv1alpha1.Endpoint{}
			if err := k8sClient.Get(ctx, client.ObjectKeyFromObject(ep), fresh); err != nil {
				return err
			}
			fresh.Spec.Usage = arcv1alpha1.EndpointUsagePushOnly

			return k8sClient.Update(ctx, fresh)
		}).Should(Succeed())

		Eventually(verdictOf(ep, arcv1alpha1.EndpointConditionValidated)).
			Should(Equal("False/UnknownType"))
	})

	It("should accept any type when the ArtifactType's rules are unrestricted", func() {
		cat := &arcv1alpha1.ClusterArtifactType{
			ObjectMeta: metav1.ObjectMeta{GenerateName: "cat-open-"},
			Spec: arcv1alpha1.ArtifactTypeSpec{
				// Empty SrcTypes/DstTypes means "any type", matching
				// order_controller.go's `len(...) > 0 && !slices.Contains(...)`
				// semantics: an empty list imposes no restriction.
				Rules:               arcv1alpha1.ArtifactTypeRules{},
				WorkflowTemplateRef: arcv1alpha1.ArtifactTypeTemplateRef{Name: "dummy"},
			},
		}
		Expect(k8sClient.Create(ctx, cat)).To(Succeed())
		DeferCleanup(k8sClient.Delete, ctx, cat)
		createSecret("creds-open")

		ep := createEndpoint("some-arbitrary-type", "creds-open", "https://registry.example/open")

		Eventually(verdictOf(ep, arcv1alpha1.EndpointConditionValidated)).
			Should(Equal("True/Valid"))
	})

	It("should mark a valid endpoint Ready", func() {
		createTypeAccepting("oci")
		createSecret("creds-ready")
		ep := createEndpoint("oci", "creds-ready", "https://registry.example/ready")

		Eventually(verdictOf(ep, arcv1alpha1.EndpointConditionReady)).
			Should(HavePrefix("True/"))
	})

	It("should relay a blocking condition's reason onto Ready", func() {
		createTypeAccepting("oci")
		ep := createEndpoint("oci", "creds-never-created", "https://registry.example/never-created")

		Eventually(verdictOf(ep, arcv1alpha1.EndpointConditionValidated)).
			Should(Equal("False/SecretNotFound"))

		Eventually(verdictOf(ep, arcv1alpha1.EndpointConditionReady)).
			Should(Equal("False/SecretNotFound"))
	})

	It("should validate an endpoint whose type is accepted by a namespaced ArtifactType", func() {
		createNamespacedTypeAccepting("namespaced-type")
		createSecret("creds-namespaced-type")
		ep := createEndpoint("namespaced-type", "creds-namespaced-type", "https://registry.example/namespaced-type")

		Eventually(verdictOf(ep, arcv1alpha1.EndpointConditionValidated)).
			Should(Equal("True/Valid"))
	})

	It("should preserve status across a spec-only update", func() {
		createTypeAccepting("oci")
		createSecret("creds-preserve")
		ep := createEndpoint("oci", "creds-preserve", "https://registry.example/preserve")

		Eventually(verdictOf(ep, arcv1alpha1.EndpointConditionValidated)).
			Should(Equal("True/Valid"))

		fresh := &arcv1alpha1.Endpoint{}
		Expect(k8sClient.Get(ctx, client.ObjectKeyFromObject(ep), fresh)).To(Succeed())
		fresh.Spec.RemoteURL = "https://registry.example/changed"
		Expect(k8sClient.Update(ctx, fresh)).To(Succeed())

		// A main-resource update must not blank the status: that is CopyStatusTo
		// doing its job in the apiserver strategy.
		Consistently(verdictOf(ep, arcv1alpha1.EndpointConditionValidated)).
			ShouldNot(Equal("absent"))
	})

	It("should record the probe result as conditions", func() {
		createTypeAccepting("oci")
		createSecret("creds-probe")
		url := "https://registry.example/probe"
		ep := createEndpoint("oci", "creds-probe", url)

		Eventually(verdictOf(ep, arcv1alpha1.EndpointConditionReachable)).
			Should(Equal("True/Reachable"))
		Eventually(verdictOf(ep, arcv1alpha1.EndpointConditionAuthenticated)).
			Should(Equal("True/Authenticated"))
		Eventually(func() *metav1.Time {
			fresh := &arcv1alpha1.Endpoint{}
			if err := k8sClient.Get(ctx, client.ObjectKeyFromObject(ep), fresh); err != nil {
				return nil
			}

			return fresh.Status.LastProbeTime
		}).ShouldNot(BeNil())
		Expect(stubProbe.CallsFor(url)).To(BeNumerically(">=", 1))
	})

	It("should not mark an unreachable endpoint Ready", func() {
		stubProbe.SetResult(endpointprobe.Result{
			Reachable: endpointprobe.Check{
				Status: metav1.ConditionFalse, Reason: endpointprobe.ReasonDNSFailure, Message: "no such host",
			},
			Authenticated: endpointprobe.Check{
				Status: metav1.ConditionUnknown, Reason: endpointprobe.ReasonInconclusive, Message: "not reached",
			},
		})
		createTypeAccepting("oci")
		createSecret("creds-unreachable")
		ep := createEndpoint("oci", "creds-unreachable", "https://registry.example/down")

		Eventually(verdictOf(ep, arcv1alpha1.EndpointConditionReady)).
			Should(Equal("False/DNSFailure"))
	})

	It("should mark Ready Unknown, not True, when the target could not be probed", func() {
		// An unprobeable scheme (s3://) never opens a socket: Reachable comes
		// back Unknown/UnsupportedScheme. ARC verified nothing, so Ready must
		// say Unknown rather than claim usability it never checked.
		stubProbe.SetResult(endpointprobe.Result{
			Reachable: endpointprobe.Check{
				Status: metav1.ConditionUnknown, Reason: endpointprobe.ReasonUnsupportedScheme,
				Message: "only http and https targets can be probed: s3",
			},
			Authenticated: endpointprobe.Check{
				Status: metav1.ConditionUnknown, Reason: endpointprobe.ReasonInconclusive, Message: "target was not probed",
			},
		})
		createTypeAccepting("blob")
		createSecret("creds-unprobeable")
		ep := createEndpoint("blob", "creds-unprobeable", "s3://bucket")

		Eventually(verdictOf(ep, arcv1alpha1.EndpointConditionReady)).
			Should(Equal("Unknown/UnsupportedScheme"))
	})

	It("should stay Ready when credentials cannot be verified", func() {
		stubProbe.SetResult(endpointprobe.Result{
			Reachable: endpointprobe.Check{
				Status: metav1.ConditionTrue, Reason: endpointprobe.ReasonReachable, Message: "ok",
			},
			Authenticated: endpointprobe.Check{
				Status: metav1.ConditionUnknown,
				Reason: endpointprobe.ReasonUnsupportedAuthScheme, Message: "cannot verify",
			},
		})
		createTypeAccepting("blob")
		createSecret("creds-s3")
		ep := createEndpoint("blob", "creds-s3", "https://s3.example/bucket")

		// Unknown must not block Ready: ARC's inability to speak SigV4 is not
		// the consumer's problem.
		Eventually(verdictOf(ep, arcv1alpha1.EndpointConditionReady)).
			Should(Equal("True/Valid"))
	})

	It("should not re-probe when nothing has changed", func() {
		createTypeAccepting("oci")
		createSecret("creds-stable")
		url := "https://registry.example/stable"
		createEndpoint("oci", "creds-stable", url)

		Eventually(func() int { return stubProbe.CallsFor(url) }).Should(Equal(1))

		createTypeAccepting("unrelated-type")

		Consistently(func() int { return stubProbe.CallsFor(url) }).Should(Equal(1))
	})

	It("should re-probe when the probe result is lost from status", func() {
		createTypeAccepting("oci")
		createSecret("creds-lost-result")
		url := "https://registry.example/lost-result"
		ep := createEndpoint("oci", "creds-lost-result", url)

		Eventually(verdictOf(ep, arcv1alpha1.EndpointConditionReachable)).
			Should(Equal("True/Reachable"))
		Eventually(func() int { return stubProbe.CallsFor(url) }).Should(Equal(1))

		// Simulate a lost or overwritten status write: Reachable and Authenticated
		// disappear even though nothing about the spec or secret changed, so the
		// in-memory stamp still matches. Validated is left in place, since the
		// real failure mode loses only the probe's own conditions.
		//
		// This races the reconciler, which is concurrently writing status for the
		// same Endpoint; retry the read-modify-write on conflict rather than
		// racing it with a single Get+Update, matching the pattern in "should
		// respect usage when matching the type".
		Eventually(func() error {
			fresh := &arcv1alpha1.Endpoint{}
			if err := k8sClient.Get(ctx, client.ObjectKeyFromObject(ep), fresh); err != nil {
				return err
			}
			meta.RemoveStatusCondition(&fresh.Status.Conditions, arcv1alpha1.EndpointConditionReachable)
			meta.RemoveStatusCondition(&fresh.Status.Conditions, arcv1alpha1.EndpointConditionAuthenticated)

			return k8sClient.Status().Update(ctx, fresh)
		}).Should(Succeed())

		Eventually(verdictOf(ep, arcv1alpha1.EndpointConditionReachable)).
			Should(Equal("True/Reachable"))
		Eventually(func() int { return stubProbe.CallsFor(url) }).Should(BeNumerically(">", 1))
	})

	It("should re-probe when the spec changes", func() {
		createTypeAccepting("oci")
		createSecret("creds-specchange")
		url := "https://registry.example/specchange"
		ep := createEndpoint("oci", "creds-specchange", url)

		Eventually(func() int { return stubProbe.CallsFor(url) }).Should(Equal(1))

		fresh := &arcv1alpha1.Endpoint{}
		Expect(k8sClient.Get(ctx, client.ObjectKeyFromObject(ep), fresh)).To(Succeed())
		fresh.Spec.Usage = arcv1alpha1.EndpointUsagePullOnly
		Expect(k8sClient.Update(ctx, fresh)).To(Succeed())

		Eventually(func() int { return stubProbe.CallsFor(url) }).Should(Equal(2))
	})

	It("should re-probe when the secret changes", func() {
		createTypeAccepting("oci")
		secret := createSecret("creds-rotate")
		url := "https://registry.example/rotate"
		createEndpoint("oci", "creds-rotate", url)

		Eventually(func() int { return stubProbe.CallsFor(url) }).Should(Equal(1))

		fresh := &corev1.Secret{}
		Expect(k8sClient.Get(ctx, client.ObjectKeyFromObject(secret), fresh)).To(Succeed())
		fresh.StringData = map[string]string{"username": "alice", "password": "rotated"}
		Expect(k8sClient.Update(ctx, fresh)).To(Succeed())

		Eventually(func() int { return stubProbe.CallsFor(url) }).Should(Equal(2))
	})

	It("should re-probe when the force annotation is set", func() {
		createTypeAccepting("oci")
		createSecret("creds-force")
		url := "https://registry.example/force"
		ep := createEndpoint("oci", "creds-force", url)

		Eventually(func() int { return stubProbe.CallsFor(url) }).Should(Equal(1))

		fresh := &arcv1alpha1.Endpoint{}
		Expect(k8sClient.Get(ctx, client.ObjectKeyFromObject(ep), fresh)).To(Succeed())
		fresh.Annotations = map[string]string{
			AnnotationForceAt: strconv.FormatInt(time.Now().Unix(), 10),
		}
		Expect(k8sClient.Update(ctx, fresh)).To(Succeed())

		Eventually(func() int { return stubProbe.CallsFor(url) }).Should(Equal(2))
	})
})
