// Copyright BWI GmbH and Artifact Conduit contributors
// SPDX-License-Identifier: Apache-2.0

package controller

import (
	"context"
	"errors"
	"math"
	"time"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/tools/events"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	arcv1alpha1 "go.opendefense.cloud/arc/api/arc/v1alpha1"
	"go.opendefense.cloud/arc/pkg/endpointprobe"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// The probe gate decides whether the consumer's credentials go over the network
// again, and it decides it from the Endpoint alone. These specs build each
// situation from fixed parts rather than waiting for a reconcile to race one.
var _ = Describe("EndpointReconciler probe gate", func() {
	const (
		epName    = "ep-gate"
		epNS      = "gate-ns"
		epType    = "oci"
		remoteURL = "https://registry.example/gate"
		secretRV  = "100"
	)

	// reachableResult is what the stub answers when a spec wants a probe to succeed.
	reachableResult := func() endpointprobe.Result {
		return endpointprobe.Result{
			Reachable: endpointprobe.Check{
				Status: metav1.ConditionTrue, Reason: endpointprobe.ReasonReachable, Message: "ok",
			},
			Authenticated: endpointprobe.Check{
				Status: metav1.ConditionTrue, Reason: endpointprobe.ReasonAuthenticated, Message: "ok",
			},
		}
	}

	// probedEndpoint is an Endpoint whose probe already ran and whose record says
	// so: the state every spec below perturbs one field of.
	probedEndpoint := func() *arcv1alpha1.Endpoint {
		// Computed per call: a value captured when Ginkgo builds the tree would
		// age by however long the suite takes to reach this spec.
		probedAt := metav1.NewTime(time.Now().Add(-time.Minute)).Rfc3339Copy()

		ep := &arcv1alpha1.Endpoint{
			ObjectMeta: metav1.ObjectMeta{
				Name:       epName,
				Namespace:  epNS,
				UID:        "11111111-2222-3333-4444-555555555555",
				Generation: 3,
			},
			Spec: arcv1alpha1.EndpointSpec{
				Type:      epType,
				RemoteURL: remoteURL,
				SecretRef: corev1.LocalObjectReference{Name: "creds"},
				Usage:     arcv1alpha1.EndpointUsageAll,
			},
			Status: arcv1alpha1.EndpointStatus{
				LastProbeTime:       &probedAt,
				ProbedGeneration:    3,
				ProbedSecretVersion: secretRV,
			},
		}
		meta.SetStatusCondition(&ep.Status.Conditions, condition(
			arcv1alpha1.EndpointConditionReachable, metav1.ConditionTrue, ReasonEndpointValid, "ok"))

		return ep
	}

	Describe("probeReason", func() {
		It("should not probe an Endpoint whose record matches its inputs", func() {
			Expect(probeReason(probedEndpoint(), secretRV, time.Time{}, 0)).To(BeEmpty())
		})

		It("should probe an Endpoint that has never been probed", func() {
			ep := probedEndpoint()
			ep.Status.LastProbeTime = nil

			Expect(probeReason(ep, secretRV, time.Time{}, 0)).To(Equal(reasonUnprobed))
		})

		It("should probe when the result is gone but the record remains", func() {
			ep := probedEndpoint()
			meta.RemoveStatusCondition(&ep.Status.Conditions, arcv1alpha1.EndpointConditionReachable)

			// A cached copy older than our own write has neither, because both are
			// written in one update: that reads as unprobed, not as lost.
			Expect(probeReason(ep, secretRV, time.Time{}, 0)).To(Equal(reasonResultLost))
		})

		It("should probe an Endpoint whose status predates the record", func() {
			ep := probedEndpoint()
			// What the previous version left behind: a probe time and a result,
			// but no record of what they were produced from.
			ep.Status.ProbedGeneration = 0
			ep.Status.ProbedSecretVersion = ""

			Expect(probeReason(ep, secretRV, time.Time{}, 0)).To(Equal(reasonUnrecorded))
		})

		It("should probe when the spec has moved on", func() {
			ep := probedEndpoint()
			ep.Generation = 4

			Expect(probeReason(ep, secretRV, time.Time{}, 0)).To(Equal(reasonSpecChanged))
		})

		It("should probe when the generation is behind the record, not only ahead", func() {
			ep := probedEndpoint()
			ep.Status.ProbedGeneration = 9

			// Compared for difference rather than order, so a record that somehow
			// runs ahead probes once instead of wedging forever.
			Expect(probeReason(ep, secretRV, time.Time{}, 0)).To(Equal(reasonSpecChanged))
		})

		It("should probe when the Secret has been rotated", func() {
			// The Secret is a separate object, so this can never be visible in the
			// Endpoint's generation.
			Expect(probeReason(probedEndpoint(), "101", time.Time{}, 0)).To(Equal(reasonSecretChanged))
		})

		It("should probe when a resourceVersion rolls over a digit", func() {
			ep := probedEndpoint()
			ep.Status.ProbedSecretVersion = "9"

			// "10" sorts before "9": ordering comparisons would read this as
			// unchanged and skip a probe that was owed.
			Expect(probeReason(ep, "10", time.Time{}, 0)).To(Equal(reasonSecretChanged))
		})

		It("should probe when the force annotation is set", func() {
			forced := time.Now().Truncate(time.Second)

			Expect(probeReason(probedEndpoint(), secretRV, forced, 0)).To(Equal(reasonForced))
		})

		It("should not probe again for a force value it already honoured", func() {
			forced := time.Now().Truncate(time.Second)
			ep := probedEndpoint()
			ep.Status.ProbedForceAt = forceAtRecord(forced)

			// The stored value has been through the API's precision, so this only
			// holds if the comparison is by instant rather than by struct.
			Expect(probeReason(ep, secretRV, forced, 0)).To(BeEmpty())
		})

		It("should only accept a force value the record can hold", func() {
			Expect(recordableForceAt(time.Time{})).To(BeTrue())
			Expect(recordableForceAt(time.Now().Truncate(time.Second))).To(BeTrue())
			Expect(recordableForceAt(time.Unix(253402300799, 0))).To(BeTrue(), "9999-12-31 is the edge")

			// A metav1.Time past year 9999 serialises to null, so the record would
			// read back absent no matter how often the annotation was honoured.
			Expect(recordableForceAt(time.Unix(253402300800, 0))).To(BeFalse())
			Expect(recordableForceAt(time.Unix(99999999999999, 0))).To(BeFalse())

			// Sub-second values are rejected too, deliberately. The annotation is
			// Unix seconds so this cannot happen today, but the gate compares the
			// value as parsed, and a record truncated away from it would look
			// unhandled forever. Ignoring such a force is the safe direction.
			Expect(recordableForceAt(time.Unix(1790000000, 500))).To(BeFalse())
		})

		It("should probe once the result is older than the TTL", func() {
			ep := probedEndpoint()

			Expect(probeReason(ep, secretRV, time.Time{}, time.Hour)).To(BeEmpty())
			Expect(probeReason(ep, secretRV, time.Time{}, 30*time.Second)).To(Equal(reasonStale))
		})
	})

	Describe("the TTL requeue", func() {
		It("should ask for no requeue when no TTL is configured", func() {
			Expect(untilProbeStale(probedEndpoint(), 0)).To(BeZero())
		})

		It("should ask to be called back when the result goes stale", func() {
			// Probed a minute ago with a two-minute TTL: roughly a minute left.
			Expect(untilProbeStale(probedEndpoint(), 2*time.Minute)).
				To(BeNumerically("~", time.Minute, 5*time.Second))
		})

		It("should not ask for an immediate requeue once the deadline has passed", func() {
			Expect(untilProbeStale(probedEndpoint(), time.Second)).To(Equal(probeRequeueFloor))
		})

		It("should spread the deadline per Endpoint, and put it in the same place every time", func() {
			first, second := probedEndpoint(), probedEndpoint()
			second.UID = "99999999-8888-7777-6666-555555555555"
			r := &EndpointReconciler{ProbeTTL: time.Hour}

			Expect(r.effectiveProbeTTL(first)).To(Equal(r.effectiveProbeTTL(first)))
			Expect(r.effectiveProbeTTL(first)).NotTo(Equal(r.effectiveProbeTTL(second)))

			for _, ep := range []*arcv1alpha1.Endpoint{first, second} {
				Expect(r.effectiveProbeTTL(ep)).To(And(
					BeNumerically(">=", time.Hour),
					BeNumerically("<=", time.Hour+time.Duration(probeJitterFraction*float64(time.Hour)))))
			}
		})

		It("should not spread anything when the TTL is disabled", func() {
			Expect((&EndpointReconciler{}).effectiveProbeTTL(probedEndpoint())).To(BeZero())
		})

		It("should not wrap a TTL too large to spread", func() {
			r := &EndpointReconciler{ProbeTTL: time.Duration(math.MaxInt64)}

			// Wrapping would give a negative deadline, which reads as stale on
			// every reconcile: a probe per reconcile, for every Endpoint.
			Expect(r.effectiveProbeTTL(probedEndpoint())).To(BeNumerically(">", 0))
		})
	})

	Describe("Reconcile", func() {
		var (
			ctx    context.Context
			scheme *runtime.Scheme
			secret *corev1.Secret
			cat    *arcv1alpha1.ClusterArtifactType
			stub   *probeStub
		)

		BeforeEach(func() {
			ctx = context.Background()
			scheme = runtime.NewScheme()
			Expect(corev1.AddToScheme(scheme)).To(Succeed())
			Expect(arcv1alpha1.AddToScheme(scheme)).To(Succeed())

			secret = &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name: "creds", Namespace: epNS, ResourceVersion: secretRV,
				},
				StringData: map[string]string{"username": "alice", "password": "s3cret"},
			}
			cat = &arcv1alpha1.ClusterArtifactType{
				ObjectMeta: metav1.ObjectMeta{Name: "cat-oci"},
				Spec: arcv1alpha1.ArtifactTypeSpec{
					Rules: arcv1alpha1.ArtifactTypeRules{
						SrcTypes: []string{epType},
						DstTypes: []string{epType},
					},
					WorkflowTemplateRef: arcv1alpha1.ArtifactTypeTemplateRef{Name: "dummy"},
				},
			}
			stub = &probeStub{}
		})

		reconcilerFor := func(c client.Client) *EndpointReconciler {
			return &EndpointReconciler{
				Client:   c,
				Scheme:   scheme,
				Recorder: events.NewFakeRecorder(20),
				Probe:    stub.Probe,
			}
		}

		It("should ask to be called back only when a TTL is configured", func() {
			// Nothing else wakes an Endpoint whose spec, Secret and annotations are
			// all untouched, so if this requeue is ever dropped the TTL silently
			// stops working and no other assertion here would notice.
			c := fake.NewClientBuilder().WithScheme(scheme).
				WithObjects(probedEndpoint(), secret, cat).
				WithStatusSubresource(&arcv1alpha1.Endpoint{}).Build()
			key := ctrl.Request{NamespacedName: namespacedName(epNS, epName)}

			withoutTTL, err := reconcilerFor(c).Reconcile(ctx, key)
			Expect(err).NotTo(HaveOccurred())
			Expect(withoutTTL.RequeueAfter).To(BeZero())

			r := reconcilerFor(c)
			r.ProbeTTL = time.Hour
			withTTL, err := r.Reconcile(ctx, key)
			Expect(err).NotTo(HaveOccurred())

			// Derived from the same function the gate uses, not from ProbeTTL: the
			// requeue and the staleness test have to agree on the spread deadline,
			// or the callback arrives to find nothing to do and asks again.
			Expect(withTTL.RequeueAfter).To(BeNumerically(
				"~", r.effectiveProbeTTL(probedEndpoint())-time.Minute, 10*time.Second))
			Expect(stub.CallsFor(remoteURL)).To(BeZero())
		})

		It("should not probe an Endpoint a previous process already probed", func() {
			c := fake.NewClientBuilder().WithScheme(scheme).
				WithObjects(probedEndpoint(), secret, cat).
				WithStatusSubresource(&arcv1alpha1.Endpoint{}).Build()

			// A fresh reconciler is what a restart or a leader failover produces.
			// The record is in the object, so it survives both.
			_, err := reconcilerFor(c).Reconcile(ctx, ctrl.Request{
				NamespacedName: namespacedName(epNS, epName),
			})

			Expect(err).NotTo(HaveOccurred())
			Expect(stub.CallsFor(remoteURL)).To(BeZero())
		})

		It("should not spend a probe on a status update that loses its race", func() {
			// The fixture carries no Ready condition, so setReadyCondition gives
			// this reconcile something to persist and it reaches the update — and
			// therefore the conflict — without needing a probe.
			ep := probedEndpoint()

			conflict := apierrors.NewConflict(
				schema.GroupResource{Group: arcv1alpha1.GroupName, Resource: "endpoints"},
				epName, errors.New("the object has been modified"))

			c := fake.NewClientBuilder().WithScheme(scheme).
				WithObjects(ep, secret, cat).
				WithStatusSubresource(&arcv1alpha1.Endpoint{}).
				WithInterceptorFuncs(interceptor.Funcs{
					SubResourceUpdate: func(
						_ context.Context, _ client.Client, _ string, _ client.Object,
						_ ...client.SubResourceUpdateOption,
					) error {
						return conflict
					},
				}).Build()

			r := reconcilerFor(c)
			key := ctrl.Request{NamespacedName: namespacedName(epNS, epName)}

			_, err := r.Reconcile(ctx, key)
			Expect(apierrors.IsConflict(errors.Unwrap(err))).To(BeTrue(), "expected the conflict to surface")

			// Nothing was forgotten by the failed write, so the retry reads the
			// same record and still owes no probe.
			_, err = r.Reconcile(ctx, key)
			Expect(err).To(HaveOccurred())
			Expect(stub.CallsFor(remoteURL)).To(BeZero())
		})

		It("should record what the probe ran against", func() {
			ep := probedEndpoint()
			ep.Generation = 7

			c := fake.NewClientBuilder().WithScheme(scheme).
				WithObjects(ep, secret, cat).
				WithStatusSubresource(&arcv1alpha1.Endpoint{}).Build()

			stub.SetResult(reachableResult())

			_, err := reconcilerFor(c).Reconcile(ctx, ctrl.Request{
				NamespacedName: namespacedName(epNS, epName),
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(stub.CallsFor(remoteURL)).To(Equal(1))

			stored := &arcv1alpha1.Endpoint{}
			Expect(c.Get(ctx, namespacedName(epNS, epName), stored)).To(Succeed())
			Expect(stored.Status.ProbedGeneration).To(Equal(int64(7)))
			Expect(stored.Status.ProbedSecretVersion).To(Equal(secretRV))
			Expect(stored.Status.LastProbeTime).NotTo(BeNil())
		})

		It("should not probe in a loop for an Endpoint without a generation", func() {
			ep := probedEndpoint()
			// The API never serves generation 0 — PrepareForCreate starts at 1 — so
			// this is unreachable today. It is asserted because recording a zero
			// against a zero reads as "no record" again, and the pass that writes
			// the record is what triggers the next one.
			ep.Generation = 0
			ep.Status.ProbedGeneration = 0

			c := fake.NewClientBuilder().WithScheme(scheme).
				WithObjects(ep, secret, cat).
				WithStatusSubresource(&arcv1alpha1.Endpoint{}).Build()

			stub.SetResult(reachableResult())
			r := reconcilerFor(c)
			key := ctrl.Request{NamespacedName: namespacedName(epNS, epName)}

			for range 5 {
				_, err := r.Reconcile(ctx, key)
				Expect(err).NotTo(HaveOccurred())
			}

			Expect(stub.CallsFor(remoteURL)).To(BeNumerically("<=", 1))
		})

		It("should not probe in a loop for a force value it cannot record", func() {
			ep := probedEndpoint()
			// Unix seconds far past year 9999: a typo, or a consumer pasting
			// milliseconds. The record cannot hold it, so honouring it would look
			// unhandled on every pass — and the probe's own status write is what
			// triggers the next pass.
			ep.Annotations = map[string]string{AnnotationForceAt: "99999999999999"}

			c := fake.NewClientBuilder().WithScheme(scheme).
				WithObjects(ep, secret, cat).
				WithStatusSubresource(&arcv1alpha1.Endpoint{}).Build()

			stub.SetResult(reachableResult())
			r := reconcilerFor(c)
			key := ctrl.Request{NamespacedName: namespacedName(epNS, epName)}

			for range 3 {
				_, err := r.Reconcile(ctx, key)
				Expect(err).NotTo(HaveOccurred())
			}

			Expect(stub.CallsFor(remoteURL)).To(BeZero())
		})

		It("should probe an upgraded Endpoint exactly once", func() {
			ep := probedEndpoint()
			ep.Status.ProbedGeneration = 0
			ep.Status.ProbedSecretVersion = ""

			c := fake.NewClientBuilder().WithScheme(scheme).
				WithObjects(ep, secret, cat).
				WithStatusSubresource(&arcv1alpha1.Endpoint{}).Build()

			stub.SetResult(reachableResult())
			r := reconcilerFor(c)
			key := ctrl.Request{NamespacedName: namespacedName(epNS, epName)}

			// The upgrade costs the fleet one probe each. It must not cost two:
			// the first reconcile writes the record the second one reads.
			for range 3 {
				_, err := r.Reconcile(ctx, key)
				Expect(err).NotTo(HaveOccurred())
			}

			Expect(stub.CallsFor(remoteURL)).To(Equal(1))
		})

		It("should never write a probe time without a result, or the reverse", func() {
			// The whole design rests on this pair being written together: it is
			// what lets a record without a result mean "lost" while a cached copy
			// older than our own write reads as "unprobed". Anything that writes
			// one without the other re-opens that ambiguity.
			starts := map[string]func() *arcv1alpha1.Endpoint{
				"unprobed": func() *arcv1alpha1.Endpoint {
					ep := probedEndpoint()
					clearProbeRecord(ep)
					meta.RemoveStatusCondition(&ep.Status.Conditions, arcv1alpha1.EndpointConditionReachable)

					return ep
				},
				"result lost": func() *arcv1alpha1.Endpoint {
					ep := probedEndpoint()
					meta.RemoveStatusCondition(&ep.Status.Conditions, arcv1alpha1.EndpointConditionReachable)

					return ep
				},
				"record predates the fields": func() *arcv1alpha1.Endpoint {
					ep := probedEndpoint()
					ep.Status.ProbedGeneration = 0

					return ep
				},
				"spec moved on": func() *arcv1alpha1.Endpoint {
					ep := probedEndpoint()
					ep.Generation = 9

					return ep
				},
				"current": probedEndpoint,
			}

			for name, start := range starts {
				By(name)

				c := fake.NewClientBuilder().WithScheme(scheme).
					WithObjects(start(), secret, cat).
					WithStatusSubresource(&arcv1alpha1.Endpoint{}).Build()

				stub.SetResult(reachableResult())

				_, err := reconcilerFor(c).Reconcile(ctx, ctrl.Request{
					NamespacedName: namespacedName(epNS, epName),
				})
				Expect(err).NotTo(HaveOccurred(), name)

				stored := &arcv1alpha1.Endpoint{}
				Expect(c.Get(ctx, namespacedName(epNS, epName), stored)).To(Succeed())

				hasTime := stored.Status.LastProbeTime != nil
				hasResult := meta.FindStatusCondition(
					stored.Status.Conditions, arcv1alpha1.EndpointConditionReachable) != nil
				Expect(hasTime).To(Equal(hasResult), name)
			}
		})

		It("should drop the record when validation stops resolving", func() {
			// No ClusterArtifactType, so the type no longer resolves: the result
			// goes, and the record has to go with it or it reads as "lost".
			c := fake.NewClientBuilder().WithScheme(scheme).
				WithObjects(probedEndpoint(), secret).
				WithStatusSubresource(&arcv1alpha1.Endpoint{}).Build()

			_, err := reconcilerFor(c).Reconcile(ctx, ctrl.Request{
				NamespacedName: namespacedName(epNS, epName),
			})
			Expect(err).NotTo(HaveOccurred())

			stored := &arcv1alpha1.Endpoint{}
			Expect(c.Get(ctx, namespacedName(epNS, epName), stored)).To(Succeed())
			Expect(stored.Status.LastProbeTime).To(BeNil())
			Expect(stored.Status.ProbedGeneration).To(BeZero())
			Expect(stored.Status.ProbedSecretVersion).To(BeEmpty())
			Expect(stored.Status.ProbedForceAt).To(BeNil())
			Expect(stub.CallsFor(remoteURL)).To(BeZero())
		})
	})

	Describe("endpointsForType", func() {
		var (
			ctx    context.Context
			scheme *runtime.Scheme
		)

		endpoint := func(name, namespace, endpointType string) *arcv1alpha1.Endpoint {
			return &arcv1alpha1.Endpoint{
				ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: namespace},
				Spec: arcv1alpha1.EndpointSpec{
					Type: endpointType, RemoteURL: remoteURL, Usage: arcv1alpha1.EndpointUsageAll,
				},
			}
		}

		names := func(requests []reconcile.Request) []string {
			out := make([]string, 0, len(requests))
			for _, req := range requests {
				out = append(out, req.Name)
			}

			return out
		}

		BeforeEach(func() {
			ctx = context.Background()
			scheme = runtime.NewScheme()
			Expect(arcv1alpha1.AddToScheme(scheme)).To(Succeed())
		})

		reconcilerWith := func(objs ...client.Object) *EndpointReconciler {
			return &EndpointReconciler{
				Client: fake.NewClientBuilder().WithScheme(scheme).WithObjects(objs...).Build(),
			}
		}

		It("should re-queue only the Endpoints whose type the rules mention", func() {
			r := reconcilerWith(
				endpoint("ep-oci", epNS, "oci"),
				endpoint("ep-helm", epNS, "helm"),
				endpoint("ep-blob", epNS, "blob"),
			)

			requests := r.endpointsForType(ctx, &arcv1alpha1.ClusterArtifactType{
				ObjectMeta: metav1.ObjectMeta{Name: "cat"},
				Spec: arcv1alpha1.ArtifactTypeSpec{
					Rules: arcv1alpha1.ArtifactTypeRules{
						SrcTypes: []string{"oci"},
						DstTypes: []string{"helm"},
					},
				},
			})

			Expect(names(requests)).To(ConsistOf("ep-oci", "ep-helm"))
		})

		It("should re-queue everything for rules that name no types", func() {
			r := reconcilerWith(
				endpoint("ep-oci", epNS, "oci"),
				endpoint("ep-helm", epNS, "helm"),
			)

			// Empty means "any type in that position", so such rules can change
			// the verdict for any Endpoint: see endpointTypeAccepted.
			requests := r.endpointsForType(ctx, &arcv1alpha1.ClusterArtifactType{
				ObjectMeta: metav1.ObjectMeta{Name: "cat"},
				Spec: arcv1alpha1.ArtifactTypeSpec{
					Rules: arcv1alpha1.ArtifactTypeRules{DstTypes: []string{"oci"}},
				},
			})

			Expect(names(requests)).To(ConsistOf("ep-oci", "ep-helm"))
		})

		It("should keep a namespaced ArtifactType to its own namespace", func() {
			r := reconcilerWith(
				endpoint("ep-here", epNS, "oci"),
				endpoint("ep-elsewhere", "other-ns", "oci"),
			)

			requests := r.endpointsForType(ctx, &arcv1alpha1.ArtifactType{
				ObjectMeta: metav1.ObjectMeta{Name: "at", Namespace: epNS},
				Spec: arcv1alpha1.ArtifactTypeSpec{
					Rules: arcv1alpha1.ArtifactTypeRules{
						SrcTypes: []string{"oci"},
						DstTypes: []string{"oci"},
					},
				},
			})

			Expect(names(requests)).To(ConsistOf("ep-here"))
		})

		It("should ignore an object that is not an artifact type", func() {
			Expect(reconcilerWith(endpoint("ep-oci", epNS, "oci")).
				endpointsForType(ctx, &corev1.Secret{})).To(BeEmpty())
		})
	})
})
