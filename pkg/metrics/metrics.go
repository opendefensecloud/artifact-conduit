// Copyright 2025 BWI GmbH and Artifact Conduit contributors
// SPDX-License-Identifier: Apache-2.0

// Package metrics exposes ARC domain metrics on the controller manager's
// existing Prometheus endpoint. Current state is reported by a collector that
// reads the manager cache at scrape time; flow and failures are recorded from
// the reconcile path.
package metrics

import (
	"runtime"
	"runtime/debug"

	"github.com/prometheus/client_golang/prometheus"
	ctrlmetrics "sigs.k8s.io/controller-runtime/pkg/metrics"

	arcv1alpha1 "go.opendefense.cloud/arc/api/arc/v1alpha1"
)

// Result values for terminal ArtifactWorkflow phases.
const (
	ResultSucceeded = "succeeded"
	ResultFailed    = "failed"
	ResultError     = "error"
)

// UnknownArtifactType is reported for ArtifactWorkflows created before the
// artifact type label existed.
const UnknownArtifactType = "unknown"

var (
	completions = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "arc_artifactworkflow_completions_total",
		Help: "Total single mode ArtifactWorkflows that reached a terminal phase, by result.",
	}, []string{"namespace", "artifact_type", "result"})

	// Buckets are sized for artifact transfers rather than web requests. The
	// client_golang defaults top out at 10s, which would put every real run
	// into the +Inf bucket.
	duration = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "arc_artifactworkflow_duration_seconds",
		Help:    "Argo execution time of single mode ArtifactWorkflows, by result.",
		Buckets: []float64{10, 30, 60, 120, 300, 600, 1800, 3600, 7200},
	}, []string{"artifact_type", "result"})

	reconcileErrors = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "arc_reconcile_errors_total",
		Help: "Total classified reconcile failures. The reason label matches the Kubernetes Event reason for the same failure.",
	}, []string{"controller", "reason"})

	buildInfo = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "arc_build_info",
		Help: "Build information of the running controller manager. Always 1.",
	}, []string{"version", "revision", "go_version"})
)

func init() {
	ctrlmetrics.Registry.MustRegister(completions, duration, reconcileErrors, buildInfo)
}

// ResultFor maps a terminal ArtifactWorkflow phase onto a result label value.
// The second return value is false for phases that are not terminal outcomes,
// including Stopped, which is an operator action rather than a result.
func ResultFor(phase arcv1alpha1.WorkflowPhase) (string, bool) {
	switch phase {
	case arcv1alpha1.WorkflowSucceeded:
		return ResultSucceeded, true
	case arcv1alpha1.WorkflowFailed:
		return ResultFailed, true
	case arcv1alpha1.WorkflowError:
		return ResultError, true
	default:
		return "", false
	}
}

// RecordCompletion counts one ArtifactWorkflow reaching a terminal phase.
// Call it only after the status write that recorded the transition succeeded,
// so a conflicting write cannot be counted twice.
func RecordCompletion(namespace, artifactType, result string) {
	completions.WithLabelValues(namespace, artifactType, result).Inc()
}

// ObserveDuration records how long Argo took to run a workflow.
func ObserveDuration(artifactType, result string, seconds float64) {
	duration.WithLabelValues(artifactType, result).Observe(seconds)
}

// CompletionsCounterForTest exposes one completions series for assertions in
// controller tests. It is not part of the runtime API.
func CompletionsCounterForTest(namespace, artifactType, result string) prometheus.Counter {
	return completions.WithLabelValues(namespace, artifactType, result)
}

// RecordReconcileError counts one classified reconcile failure. The reason must
// be one of the Event reason constants in the controller package so the metric
// and the Kubernetes Event always agree.
func RecordReconcileError(controller, reason string) {
	reconcileErrors.WithLabelValues(controller, reason).Inc()
}

// ReconcileErrorsCounterForTest exposes one reconcile error series for
// assertions in controller tests. It is not part of the runtime API.
func ReconcileErrorsCounterForTest(controller, reason string) prometheus.Counter {
	return reconcileErrors.WithLabelValues(controller, reason)
}

// SetBuildInfo publishes build information read from the embedded build data.
func SetBuildInfo() {
	version, revision := "unknown", "unknown"

	if info, ok := debug.ReadBuildInfo(); ok {
		if info.Main.Version != "" {
			version = info.Main.Version
		}

		for _, setting := range info.Settings {
			if setting.Key == "vcs.revision" {
				revision = setting.Value
			}
		}
	}

	buildInfo.WithLabelValues(version, revision, runtime.Version()).Set(1)
}
