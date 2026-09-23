// Copyright BWI GmbH and Artifact Conduit contributors
// SPDX-License-Identifier: Apache-2.0

package controller

import (
	"context"
	"fmt"
	"hash/fnv"
	"math"
	"slices"
	"time"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/events"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	ctrlcontroller "sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	arcv1alpha1 "go.opendefense.cloud/arc/api/arc/v1alpha1"
	"go.opendefense.cloud/arc/pkg/endpointprobe"
	"go.opendefense.cloud/arc/pkg/metrics"
)

// EndpointReconciler reconciles an Endpoint object.
type EndpointReconciler struct {
	client.Client
	Scheme   *runtime.Scheme
	Recorder events.EventRecorder

	// Probe tests the endpoint's target. It is a field rather than a direct
	// call so the envtest suite can substitute a stub and never open a socket.
	Probe func(context.Context, endpointprobe.Target) endpointprobe.Result

	// ProbeTTL is how long a probe result is treated as current. Reachability is
	// a property of the world rather than of the cluster, so nothing here
	// observes a target going down; without a TTL a result stays as it was until
	// the spec, the Secret or the force annotation changes. Zero disables it,
	// which is the default and matches the documented "no periodic re-probe".
	ProbeTTL time.Duration
}

// endpointMaxConcurrentReconciles bounds how many Endpoints this controller
// reconciles at once. The probe is network-bound and capped at 5s, so with the
// controller-runtime default of one worker, reconciliation is strictly serial:
// N Endpoints that all time out converge only after roughly 5*N seconds.
const endpointMaxConcurrentReconciles = 4

// Why an Endpoint is being probed, reported in the log line that precedes the
// probe so that "why did this re-probe?" is answerable from the logs.
const (
	reasonUnprobed      = "unprobed"
	reasonResultLost    = "result-lost"
	reasonSpecChanged   = "spec-changed"
	reasonSecretChanged = "secret-changed"
	reasonForced        = "forced"
	reasonStale         = "stale"
)

const (
	// probeJitterFraction is how much of the TTL a per-Endpoint offset may add,
	// so that Endpoints created together do not all go stale together.
	probeJitterFraction = 0.2

	// probeRequeueFloor keeps a requeue from becoming a busy loop if the
	// deadline has already passed by the time it is computed.
	probeRequeueFloor = time.Second
)

//+kubebuilder:rbac:groups=arc.opendefense.cloud,resources=endpoints,verbs=get;list;watch
//+kubebuilder:rbac:groups=arc.opendefense.cloud,resources=endpoints/status,verbs=get;update;patch

// Reconcile reports the observed state of an Endpoint in its status.
func (r *EndpointReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := ctrl.LoggerFrom(ctx)

	endpoint := &arcv1alpha1.Endpoint{}
	if err := r.Get(ctx, req.NamespacedName, endpoint); err != nil {
		if apierrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}

		return ctrl.Result{}, errLogAndWrap(log, err, "failed to get endpoint")
	}

	if !endpoint.DeletionTimestamp.IsZero() {
		return ctrl.Result{}, nil
	}

	original := endpoint.DeepCopy()

	secret, validated, err := r.validate(ctx, log, endpoint)
	if err != nil {
		return ctrl.Result{}, err
	}
	meta.SetStatusCondition(&endpoint.Status.Conditions, validated)

	var requeueAfter time.Duration
	if validated.Status == metav1.ConditionTrue {
		requeueAfter = r.probeEndpoint(ctx, log, endpoint, secret)
	} else {
		meta.RemoveStatusCondition(&endpoint.Status.Conditions, arcv1alpha1.EndpointConditionReachable)
		meta.RemoveStatusCondition(&endpoint.Status.Conditions, arcv1alpha1.EndpointConditionAuthenticated)
		// The result is gone, so the record of what produced it must go with it:
		// they are one fact, and a record without a result reads as "lost".
		clearProbeRecord(endpoint)
	}

	setReadyCondition(endpoint)
	endpoint.Status.ObservedGeneration = endpoint.Generation

	if equality.Semantic.DeepEqual(original.Status, endpoint.Status) {
		return ctrl.Result{RequeueAfter: requeueAfter}, nil
	}

	if err := r.Status().Update(ctx, endpoint); err != nil {
		metrics.RecordReconcileError(ControllerEndpoint, ReasonUpdateFailed)

		// Nothing to unwind: the probe's record travels with its result in the
		// same status write, so a write that did not land leaves both absent and
		// the retry reads that as "unprobed".
		return ctrl.Result{}, errLogAndWrap(log, err, "failed to update endpoint status")
	}

	r.recordReadyTransition(original, endpoint)

	return ctrl.Result{RequeueAfter: requeueAfter}, nil
}

// validate resolves the Endpoint's references.
//
// A missing Secret or an unknown type is a consumer configuration problem, so it
// is reported through the condition and not returned as an error: retrying with
// backoff would not fix it, it would pollute arc_reconcile_errors_total, and the
// Secret and ArtifactType watches already bring the reconcile back when the
// missing object appears.
func (r *EndpointReconciler) validate(
	ctx context.Context, log logr.Logger, ep *arcv1alpha1.Endpoint,
) (*corev1.Secret, metav1.Condition, error) {
	var secret *corev1.Secret

	if name := ep.Spec.SecretRef.Name; name != "" {
		found := &corev1.Secret{}
		err := r.Get(ctx, namespacedName(ep.Namespace, name), found)

		switch {
		case apierrors.IsNotFound(err):
			return nil, condition(arcv1alpha1.EndpointConditionValidated, metav1.ConditionFalse,
				ReasonSecretNotFound,
				fmt.Sprintf("secret %q not found in namespace %q", name, ep.Namespace)), nil
		case err != nil:
			metrics.RecordReconcileError(ControllerEndpoint, ReasonValidationFailed)

			return nil, metav1.Condition{}, errLogAndWrap(log, err, "failed to get secret")
		}

		secret = found
	}

	known, err := r.typeIsKnown(ctx, log, ep)
	if err != nil {
		return nil, metav1.Condition{}, err
	}
	if !known {
		return secret, condition(arcv1alpha1.EndpointConditionValidated, metav1.ConditionFalse,
			ReasonUnknownType,
			fmt.Sprintf("no ArtifactType or ClusterArtifactType accepts an endpoint of type %q for usage %s",
				ep.Spec.Type, ep.Spec.Usage)), nil
	}

	return secret, condition(arcv1alpha1.EndpointConditionValidated, metav1.ConditionTrue,
		ReasonEndpointValid, "secret and type resolve"), nil
}

func (r *EndpointReconciler) typeIsKnown(ctx context.Context, log logr.Logger, ep *arcv1alpha1.Endpoint) (bool, error) {
	clusterTypes := &arcv1alpha1.ClusterArtifactTypeList{}
	if err := r.List(ctx, clusterTypes); err != nil {
		metrics.RecordReconcileError(ControllerEndpoint, ReasonValidationFailed)

		return false, errLogAndWrap(log, err, "failed to list cluster artifact types")
	}
	for i := range clusterTypes.Items {
		if endpointTypeAccepted(ep, clusterTypes.Items[i].Spec.Rules) {
			return true, nil
		}
	}

	namespacedTypes := &arcv1alpha1.ArtifactTypeList{}
	if err := r.List(ctx, namespacedTypes, client.InNamespace(ep.Namespace)); err != nil {
		metrics.RecordReconcileError(ControllerEndpoint, ReasonValidationFailed)

		return false, errLogAndWrap(log, err, "failed to list artifact types")
	}
	for i := range namespacedTypes.Items {
		if endpointTypeAccepted(ep, namespacedTypes.Items[i].Spec.Rules) {
			return true, nil
		}
	}

	return false, nil
}

// endpointTypeAccepted reports whether the rules use this endpoint's type in a
// position its usage permits: a PullOnly endpoint has to appear as a source and
// a PushOnly one as a destination, or it can never take part in a workflow.
func endpointTypeAccepted(ep *arcv1alpha1.Endpoint, rules arcv1alpha1.ArtifactTypeRules) bool {
	asSrc := len(rules.SrcTypes) == 0 || slices.Contains(rules.SrcTypes, ep.Spec.Type)
	asDst := len(rules.DstTypes) == 0 || slices.Contains(rules.DstTypes, ep.Spec.Type)

	switch ep.Spec.Usage {
	case arcv1alpha1.EndpointUsagePullOnly:
		return asSrc
	case arcv1alpha1.EndpointUsagePushOnly:
		return asDst
	default:
		return asSrc || asDst
	}
}

// setReadyCondition summarises the other conditions.
//
// Authenticated Unknown deliberately does not block Ready. ARC cannot verify
// every credential shape — an S3 endpoint's key pair, for instance — and holding
// such an endpoint back from Ready would penalise the consumer for ARC's
// limitation rather than tell them something true.
//
// Reachable Unknown, unlike Authenticated Unknown, DOES block Ready (as
// Unknown, not False): an unsupported scheme or any other case where the
// probe never opened a socket means ARC verified nothing, and reporting
// Ready=True would claim a check that never happened.
func setReadyCondition(ep *arcv1alpha1.Endpoint) {
	for _, conditionType := range []string{
		arcv1alpha1.EndpointConditionValidated,
		arcv1alpha1.EndpointConditionReachable,
		arcv1alpha1.EndpointConditionAuthenticated,
	} {
		c := meta.FindStatusCondition(ep.Status.Conditions, conditionType)
		if c == nil || c.Status != metav1.ConditionFalse {
			continue
		}

		meta.SetStatusCondition(&ep.Status.Conditions, condition(
			arcv1alpha1.EndpointConditionReady, metav1.ConditionFalse, c.Reason, c.Message))

		return
	}

	validated := meta.FindStatusCondition(ep.Status.Conditions, arcv1alpha1.EndpointConditionValidated)
	reachable := meta.FindStatusCondition(ep.Status.Conditions, arcv1alpha1.EndpointConditionReachable)

	if validated == nil || validated.Status != metav1.ConditionTrue || reachable == nil || reachable.Status != metav1.ConditionTrue {
		reason, message := ReasonEndpointNotReady, "endpoint has not been fully checked yet"

		switch {
		case validated != nil && validated.Status != metav1.ConditionTrue:
			reason, message = validated.Reason, validated.Message
		case reachable != nil && reachable.Status != metav1.ConditionTrue:
			reason, message = reachable.Reason, reachable.Message
		}

		meta.SetStatusCondition(&ep.Status.Conditions, condition(
			arcv1alpha1.EndpointConditionReady, metav1.ConditionUnknown, reason, message))

		return
	}

	meta.SetStatusCondition(&ep.Status.Conditions, condition(
		arcv1alpha1.EndpointConditionReady, metav1.ConditionTrue,
		ReasonEndpointValid, "endpoint is usable"))
}

// probeEndpoint probes the target when the Endpoint's own record says the result
// it carries is no longer the one its inputs call for, and records what the probe
// ran against. It reports when to look at this Endpoint again, or zero for never.
//
// The gate is what keeps a quiet cluster quiet: without it every resync and every
// ArtifactType event would send the consumer's credentials over the network
// again. It is a cache, so a miss costs one redundant probe and nothing else —
// notably, a reconcile reading a cached copy older than our own last write sees
// no record, probes once more, and its stale write then loses the conflict.
func (r *EndpointReconciler) probeEndpoint(
	ctx context.Context, log logr.Logger, ep *arcv1alpha1.Endpoint, secret *corev1.Secret,
) time.Duration {
	forceAt, err := GetForceAtAnnotationValue(ep)
	if err != nil {
		log.V(1).Error(err, "Invalid force reconcile annotation, ignoring")
	}

	ttl := r.effectiveProbeTTL(ep)

	if reason := probeReason(ep, secretVersion(secret), forceAt, ttl); reason != "" {
		log.V(1).Info("Probing endpoint", "reason", reason)

		result := r.Probe(ctx, targetFor(ep, secret))

		now := metav1.Now().Rfc3339Copy()
		ep.Status.LastProbeTime = &now
		ep.Status.ProbedGeneration = ep.Generation
		ep.Status.ProbedSecretVersion = secretVersion(secret)
		ep.Status.ProbedForceAt = forceAtRecord(forceAt)

		meta.SetStatusCondition(&ep.Status.Conditions,
			checkCondition(arcv1alpha1.EndpointConditionReachable, result.Reachable))
		meta.SetStatusCondition(&ep.Status.Conditions,
			checkCondition(arcv1alpha1.EndpointConditionAuthenticated, result.Authenticated))
	}

	return untilProbeStale(ep, ttl)
}

// probeReason reports why this Endpoint needs probing, or "" for not at all.
// Every case is an independent question; the order decides only which reason is
// reported when more than one applies, so the most specific comes first.
func probeReason(ep *arcv1alpha1.Endpoint, secretRV string, forceAt time.Time, ttl time.Duration) string {
	if ep.Status.LastProbeTime == nil {
		// Never probed, or the record was cleared because the type or the Secret
		// stopped resolving.
		return reasonUnprobed
	}

	if meta.FindStatusCondition(ep.Status.Conditions, arcv1alpha1.EndpointConditionReachable) == nil {
		// Probed, but the result is no longer in status. LastProbeTime and the
		// condition are written in one update, so this pair can only mean the
		// result was lost — never that our own write is not visible yet.
		return reasonResultLost
	}

	if ep.Generation != ep.Status.ProbedGeneration {
		// Compared for difference, not order: a generation that somehow runs
		// ahead of the object (restore from backup, delete and recreate) must
		// probe once rather than wedge.
		return reasonSpecChanged
	}

	if secretRV != ep.Status.ProbedSecretVersion {
		// resourceVersion is opaque, so this too is only ever compared for
		// equality. The Secret is a separate object, so a rotation can never
		// move this Endpoint's generation.
		return reasonSecretChanged
	}

	if !recordedForceAt(ep).Equal(forceAt) {
		// Equal rather than ==: the latter compares a time.Time's monotonic
		// reading and location, not the instant. Annotations do not move the
		// generation either.
		return reasonForced
	}

	if ttl > 0 && time.Since(ep.Status.LastProbeTime.Time) > ttl {
		return reasonStale
	}

	return ""
}

// untilProbeStale reports how long this Endpoint's result stays current, which is
// how long until the reconcile has to come back. Zero means no TTL is configured
// and nothing has to come back: a spec, Secret or annotation change arrives as a
// watch event, but a target going down does not.
func untilProbeStale(ep *arcv1alpha1.Endpoint, ttl time.Duration) time.Duration {
	if ttl <= 0 || ep.Status.LastProbeTime == nil {
		return 0
	}

	return max(ttl-time.Since(ep.Status.LastProbeTime.Time), probeRequeueFloor)
}

// effectiveProbeTTL spreads the configured TTL by a per-Endpoint offset, so that
// a fleet applied together — and therefore probed together — does not expire
// together and saturate the workers with one registry's worth of probes. Derived
// from the UID rather than drawn at random, so the deadline stays put across
// reconciles and restarts, and so probeReason and untilProbeStale always agree
// on it: if they disagreed, a requeue would fire and find nothing to do.
func (r *EndpointReconciler) effectiveProbeTTL(ep *arcv1alpha1.Endpoint) time.Duration {
	if r.ProbeTTL <= 0 {
		return 0
	}

	h := fnv.New32a()
	_, _ = h.Write([]byte(ep.UID))
	offset := float64(r.ProbeTTL) * probeJitterFraction * (float64(h.Sum32()) / float64(math.MaxUint32))

	return r.ProbeTTL + time.Duration(offset)
}

// clearProbeRecord drops the probe result's provenance. It is called wherever the
// result itself is removed, because a record left behind without a result reads
// as "the result was lost".
func clearProbeRecord(ep *arcv1alpha1.Endpoint) {
	ep.Status.LastProbeTime = nil
	ep.Status.ProbedGeneration = 0
	ep.Status.ProbedSecretVersion = ""
	ep.Status.ProbedForceAt = nil
}

// recordedForceAt is the force annotation value the last probe honoured, or the
// zero time if it never honoured one.
func recordedForceAt(ep *arcv1alpha1.Endpoint) time.Time {
	if ep.Status.ProbedForceAt == nil {
		return time.Time{}
	}

	return ep.Status.ProbedForceAt.Time
}

// forceAtRecord stores a force annotation value, truncated to the precision the
// API round-trips at so that the value read back compares equal to this one.
func forceAtRecord(forceAt time.Time) *metav1.Time {
	if forceAt.IsZero() {
		return nil
	}

	stored := metav1.NewTime(forceAt).Rfc3339Copy()

	return &stored
}

// secretVersion is the resourceVersion of the Secret a probe would use, or "" for
// an Endpoint that references none.
func secretVersion(secret *corev1.Secret) string {
	if secret == nil {
		return ""
	}

	return secret.ResourceVersion
}

// targetFor builds the probe input. The username and password keys are the
// convention the shipped OCI and Helm workflow templates use; a Secret in any
// other shape yields empty credentials, which the probe reports as
// UnsupportedAuthScheme rather than as a failure.
func targetFor(ep *arcv1alpha1.Endpoint, secret *corev1.Secret) endpointprobe.Target {
	target := endpointprobe.Target{RemoteURL: ep.Spec.RemoteURL}
	if secret == nil {
		return target
	}

	target.HasSecret = true
	target.Username = string(secret.Data["username"])
	target.Password = string(secret.Data["password"])

	return target
}

func checkCondition(conditionType string, c endpointprobe.Check) metav1.Condition {
	return condition(conditionType, c.Status, c.Reason, c.Message)
}

func condition(conditionType string, status metav1.ConditionStatus, reason, message string) metav1.Condition {
	return metav1.Condition{
		Type:    conditionType,
		Status:  status,
		Reason:  reason,
		Message: message,
	}
}

// recordReadyTransition emits an Event only when Ready flips, so a busy cluster
// does not fill the event stream with restatements of a steady condition.
func (r *EndpointReconciler) recordReadyTransition(old, current *arcv1alpha1.Endpoint) {
	before := meta.FindStatusCondition(old.Status.Conditions, arcv1alpha1.EndpointConditionReady)
	after := meta.FindStatusCondition(current.Status.Conditions, arcv1alpha1.EndpointConditionReady)

	if after == nil || (before != nil && before.Status == after.Status) {
		return
	}

	eventType := corev1.EventTypeWarning
	if after.Status == metav1.ConditionTrue {
		eventType = corev1.EventTypeNormal
	}

	r.Recorder.Eventf(current, nil, eventType, after.Reason, "Reconcile",
		"Endpoint Ready=%s: %s", after.Status, after.Message)
}

// SetupWithManager sets up the controller with the Manager.
//
// The Secret watch means the manager caches every Secret it is permitted to
// read. That is the cost of re-probing on credential rotation; if it becomes a
// memory problem, the cache can be narrowed with a label selector on Secrets
// that Endpoints reference.
func (r *EndpointReconciler) SetupWithManager(mgr ctrl.Manager) error {
	if r.Probe == nil {
		r.Probe = endpointprobe.New(endpointprobe.DefaultDenyCIDRs).Probe
	}

	return ctrl.NewControllerManagedBy(mgr).
		For(&arcv1alpha1.Endpoint{}).
		Watches(&corev1.Secret{}, handler.EnqueueRequestsFromMapFunc(r.endpointsForSecret)).
		Watches(&arcv1alpha1.ClusterArtifactType{}, handler.EnqueueRequestsFromMapFunc(r.endpointsForType)).
		Watches(&arcv1alpha1.ArtifactType{}, handler.EnqueueRequestsFromMapFunc(r.endpointsForType)).
		WithOptions(ctrlcontroller.Options{MaxConcurrentReconciles: endpointMaxConcurrentReconciles}).
		Complete(r)
}

func (r *EndpointReconciler) endpointsForSecret(ctx context.Context, obj client.Object) []reconcile.Request {
	endpoints := &arcv1alpha1.EndpointList{}
	if err := r.List(ctx, endpoints, client.InNamespace(obj.GetNamespace())); err != nil {
		return nil
	}

	var requests []reconcile.Request
	for i := range endpoints.Items {
		ep := &endpoints.Items[i]
		if ep.Spec.SecretRef.Name == obj.GetName() {
			requests = append(requests, reconcile.Request{
				NamespacedName: namespacedName(ep.Namespace, ep.Name),
			})
		}
	}

	return requests
}

// endpointsForType re-queues the Endpoints whose validation an artifact type's
// rules can actually change, rather than every Endpoint in the cluster: a type
// appearing, changing or going away can make an unknown type known or the
// reverse, but only for the types its rules mention.
//
// This is correct across updates because controller-runtime maps both the old and
// the new object of an update event, so rules that stop mentioning a type still
// re-queue the Endpoints that were relying on them.
func (r *EndpointReconciler) endpointsForType(ctx context.Context, obj client.Object) []reconcile.Request {
	rules, ok := artifactTypeRules(obj)
	if !ok {
		return nil
	}

	var opts []client.ListOption
	if namespace := obj.GetNamespace(); namespace != "" {
		// A namespaced ArtifactType is only consulted for Endpoints beside it.
		opts = append(opts, client.InNamespace(namespace))
	}

	endpoints := &arcv1alpha1.EndpointList{}
	if err := r.List(ctx, endpoints, opts...); err != nil {
		return nil
	}

	// Rules that name no types accept every type in that position, so they can
	// affect any Endpoint: see endpointTypeAccepted.
	anyType := len(rules.SrcTypes) == 0 || len(rules.DstTypes) == 0

	requests := make([]reconcile.Request, 0, len(endpoints.Items))
	for i := range endpoints.Items {
		ep := &endpoints.Items[i]

		if !anyType &&
			!slices.Contains(rules.SrcTypes, ep.Spec.Type) &&
			!slices.Contains(rules.DstTypes, ep.Spec.Type) {
			continue
		}

		requests = append(requests, reconcile.Request{
			NamespacedName: namespacedName(ep.Namespace, ep.Name),
		})
	}

	return requests
}

// artifactTypeRules reads the validation rules off either kind of artifact type.
func artifactTypeRules(obj client.Object) (arcv1alpha1.ArtifactTypeRules, bool) {
	switch t := obj.(type) {
	case *arcv1alpha1.ArtifactType:
		return t.Spec.Rules, true
	case *arcv1alpha1.ClusterArtifactType:
		return t.Spec.Rules, true
	default:
		return arcv1alpha1.ArtifactTypeRules{}, false
	}
}
