// Copyright 2025 BWI GmbH and Artifact Conduit contributors
// SPDX-License-Identifier: Apache-2.0

package controller

import (
	"context"
	"fmt"
	"slices"
	"sync"
	"time"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/events"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
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

	// probed remembers what each Endpoint looked like when it was last probed.
	// in-memory, so a restart re-probes every Endpoint once
	mu     sync.Mutex
	probed map[types.NamespacedName]probeStamp
}

// probeStamp is the set of inputs that can change a probe's answer.
type probeStamp struct {
	generation int64
	secretRV   string
	forceAt    time.Time
}

//+kubebuilder:rbac:groups=arc.opendefense.cloud,resources=endpoints,verbs=get;list;watch
//+kubebuilder:rbac:groups=arc.opendefense.cloud,resources=endpoints/status,verbs=get;update;patch

// Reconcile reports the observed state of an Endpoint in its status.
func (r *EndpointReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := ctrl.LoggerFrom(ctx)

	endpoint := &arcv1alpha1.Endpoint{}
	if err := r.Get(ctx, req.NamespacedName, endpoint); err != nil {
		if apierrors.IsNotFound(err) {
			r.forgetProbe(req.NamespacedName)

			return ctrl.Result{}, nil
		}

		return ctrl.Result{}, errLogAndWrap(log, err, "failed to get endpoint")
	}

	if !endpoint.DeletionTimestamp.IsZero() {
		r.forgetProbe(req.NamespacedName)

		return ctrl.Result{}, nil
	}

	original := endpoint.DeepCopy()

	secret, validated, err := r.validate(ctx, log, endpoint)
	if err != nil {
		return ctrl.Result{}, err
	}
	meta.SetStatusCondition(&endpoint.Status.Conditions, validated)

	if validated.Status == metav1.ConditionTrue {
		r.probeEndpoint(ctx, log, endpoint, secret)
	}

	setReadyCondition(endpoint)
	endpoint.Status.ObservedGeneration = endpoint.Generation

	if equality.Semantic.DeepEqual(original.Status, endpoint.Status) {
		return ctrl.Result{}, nil
	}

	if err := r.Status().Update(ctx, endpoint); err != nil {
		metrics.RecordReconcileError(ControllerEndpoint, ReasonUpdateFailed)
		// The probe's result did not reach the API, so do not remember having
		// probed: the retry has to run it again.
		r.forgetProbe(req.NamespacedName)

		return ctrl.Result{}, errLogAndWrap(log, err, "failed to update endpoint status")
	}

	r.recordReadyTransition(original, endpoint)

	return ctrl.Result{}, nil
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

func (r *EndpointReconciler) probeEndpoint(
	ctx context.Context, log logr.Logger, ep *arcv1alpha1.Endpoint, secret *corev1.Secret,
) {
	forceAt, err := GetForceAtAnnotationValue(ep)
	if err != nil {
		log.V(1).Error(err, "Invalid force reconcile annotation, ignoring")
	}

	if !r.shouldProbe(ep, secret, forceAt) {
		return
	}

	result := r.Probe(ctx, targetFor(ep, secret))

	now := metav1.Now()
	ep.Status.LastProbeTime = &now
	meta.SetStatusCondition(&ep.Status.Conditions,
		checkCondition(arcv1alpha1.EndpointConditionReachable, result.Reachable))
	meta.SetStatusCondition(&ep.Status.Conditions,
		checkCondition(arcv1alpha1.EndpointConditionAuthenticated, result.Authenticated))
}

// shouldProbe reports whether anything that could change the probe's answer has
// changed since the last probe, or the object has no probe result to show for
// one, and records the new state if so.
//
// This gate is what keeps an idle cluster silent. Without it every informer
// resync would send the consumer's credentials over the network again.
func (r *EndpointReconciler) shouldProbe(ep *arcv1alpha1.Endpoint, secret *corev1.Secret, forceAt time.Time) bool {
	current := probeStamp{generation: ep.Generation, forceAt: forceAt}
	if secret != nil {
		current.secretRV = secret.ResourceVersion
	}

	key := namespacedName(ep.Namespace, ep.Name)

	r.mu.Lock()
	defer r.mu.Unlock()

	missingResult := meta.FindStatusCondition(ep.Status.Conditions, arcv1alpha1.EndpointConditionReachable) == nil

	if last, seen := r.probed[key]; !missingResult && seen && last == current {
		return false
	}

	if r.probed == nil {
		r.probed = map[types.NamespacedName]probeStamp{}
	}
	r.probed[key] = current

	return true
}

func (r *EndpointReconciler) forgetProbe(key types.NamespacedName) {
	r.mu.Lock()
	defer r.mu.Unlock()

	delete(r.probed, key)
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
		Watches(&arcv1alpha1.ClusterArtifactType{}, handler.EnqueueRequestsFromMapFunc(r.allEndpoints)).
		Watches(&arcv1alpha1.ArtifactType{}, handler.EnqueueRequestsFromMapFunc(r.allEndpoints)).
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

// allEndpoints re-queues every Endpoint. ArtifactType rules are cluster-wide
// inputs to validation, so a new one can make a previously unknown type known.
func (r *EndpointReconciler) allEndpoints(ctx context.Context, _ client.Object) []reconcile.Request {
	endpoints := &arcv1alpha1.EndpointList{}
	if err := r.List(ctx, endpoints); err != nil {
		return nil
	}

	requests := make([]reconcile.Request, 0, len(endpoints.Items))
	for i := range endpoints.Items {
		ep := &endpoints.Items[i]
		requests = append(requests, reconcile.Request{
			NamespacedName: namespacedName(ep.Namespace, ep.Name),
		})
	}

	return requests
}
