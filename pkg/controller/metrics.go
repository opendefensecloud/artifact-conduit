// Copyright 2025 BWI GmbH and Artifact Conduit contributors
// SPDX-License-Identifier: Apache-2.0

package controller

import (
	wfv1alpha1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"

	arcv1alpha1 "go.opendefense.cloud/arc/api/arc/v1alpha1"
	"go.opendefense.cloud/arc/pkg/metrics"
)

// completion carries what a finished ArtifactWorkflow contributes to the
// metrics, held between detecting the transition and the status write that
// commits it. Recording before that write would count twice whenever the write
// loses a conflict and the object is reconciled again from a stale cache.
type completion struct {
	namespace    string
	artifactType string
	result       string
	seconds      float64
	hasDuration  bool
}

// newCompletion returns the metrics contribution of a workflow that has just
// reached a terminal phase, or nil if the phase is not a terminal result.
func newCompletion(aw *arcv1alpha1.ArtifactWorkflow, wf *wfv1alpha1.Workflow) *completion {
	result, ok := metrics.ResultFor(aw.Status.Phase)
	if !ok {
		return nil
	}

	c := &completion{
		namespace:    aw.Namespace,
		artifactType: metrics.ArtifactTypeOf(aw),
		result:       result,
	}

	// Argo's own start and finish times are the honest measurement. ARC's
	// status timestamps are observation times taken on the controller clock at
	// reconcile, against a creation timestamp from the API server clock.
	//
	// elapsed can legitimately be zero: metav1.Time serialises at second
	// granularity, so a workflow that starts and finishes inside the same
	// second measures as zero and that is still a truthful observation. Only
	// a negative elapsed, from clock skew or a FinishedAt that precedes
	// StartedAt, is rejected.
	if !wf.Status.StartedAt.IsZero() && !wf.Status.FinishedAt.IsZero() {
		if elapsed := wf.Status.FinishedAt.Sub(wf.Status.StartedAt.Time); elapsed >= 0 {
			c.seconds = elapsed.Seconds()
			c.hasDuration = true
		}
	}

	return c
}

// record publishes the completion. Call it only after the status write succeeded.
func (c *completion) record() {
	metrics.RecordCompletion(c.namespace, c.artifactType, c.result)

	if c.hasDuration {
		metrics.ObserveDuration(c.artifactType, c.result, c.seconds)
	}
}
