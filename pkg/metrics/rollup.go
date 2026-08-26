// Copyright 2025 BWI GmbH and Artifact Conduit contributors
// SPDX-License-Identifier: Apache-2.0

package metrics

import (
	arcv1alpha1 "go.opendefense.cloud/arc/api/arc/v1alpha1"
)

// OrderPhase derives an aggregate phase for an Order from the phases of the
// ArtifactWorkflows it owns. Orders have no phase of their own.
//
// Precedence is deliberate. A failure anywhere makes the whole Order failed. A
// deliberate stop is reported rather than folded into Pending. Otherwise
// in progress work wins over completed work, so an Order reads Succeeded only
// once every workflow has succeeded.
//
// An Order containing a cron artifact moves between phases for as long as it
// exists rather than settling in one. It reads Running while a run is in
// flight, Succeeded between runs, and stays Failed after a failed run until the
// next run reports otherwise. Running therefore means "has work in flight", not
// "is unhealthy", and Succeeded means "nothing outstanding right now", not
// "finished for good".
func OrderPhase(statuses map[string]arcv1alpha1.OrderArtifactWorkflowStatus) arcv1alpha1.WorkflowPhase {
	if len(statuses) == 0 {
		return arcv1alpha1.WorkflowPending
	}

	var stopped, inProgress bool

	succeeded := 0

	for _, status := range statuses {
		switch {
		case status.Phase == arcv1alpha1.WorkflowFailed, status.Phase == arcv1alpha1.WorkflowError:
			return arcv1alpha1.WorkflowFailed
		case status.Phase == arcv1alpha1.WorkflowStopped:
			stopped = true
		case status.Phase.InProgress():
			inProgress = true
		case status.Phase == arcv1alpha1.WorkflowSucceeded:
			succeeded++
		}
	}

	switch {
	case stopped:
		return arcv1alpha1.WorkflowStopped
	case inProgress:
		return arcv1alpha1.WorkflowRunning
	case succeeded == len(statuses):
		return arcv1alpha1.WorkflowSucceeded
	default:
		return arcv1alpha1.WorkflowPending
	}
}
