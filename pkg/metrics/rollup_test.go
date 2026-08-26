// Copyright 2025 BWI GmbH and Artifact Conduit contributors
// SPDX-License-Identifier: Apache-2.0

package metrics

import (
	arcv1alpha1 "go.opendefense.cloud/arc/api/arc/v1alpha1"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func awStatuses(phases ...arcv1alpha1.WorkflowPhase) map[string]arcv1alpha1.OrderArtifactWorkflowStatus {
	out := map[string]arcv1alpha1.OrderArtifactWorkflowStatus{}
	for i, phase := range phases {
		out[string(rune('a'+i))] = arcv1alpha1.OrderArtifactWorkflowStatus{
			WorkflowStatus: arcv1alpha1.WorkflowStatus{Phase: phase},
			ArtifactIndex:  i,
		}
	}

	return out
}

var _ = Describe("OrderPhase", func() {
	It("should report Pending for an order with no workflows yet", func() {
		Expect(OrderPhase(nil)).To(Equal(arcv1alpha1.WorkflowPending))
		Expect(OrderPhase(awStatuses())).To(Equal(arcv1alpha1.WorkflowPending))
	})

	It("should report Failed when any workflow failed or errored", func() {
		Expect(OrderPhase(awStatuses(
			arcv1alpha1.WorkflowSucceeded, arcv1alpha1.WorkflowFailed,
		))).To(Equal(arcv1alpha1.WorkflowFailed))

		Expect(OrderPhase(awStatuses(
			arcv1alpha1.WorkflowRunning, arcv1alpha1.WorkflowError,
		))).To(Equal(arcv1alpha1.WorkflowFailed))
	})

	It("should prefer Failed over Stopped", func() {
		Expect(OrderPhase(awStatuses(
			arcv1alpha1.WorkflowStopped, arcv1alpha1.WorkflowFailed,
		))).To(Equal(arcv1alpha1.WorkflowFailed))
	})

	It("should report Stopped rather than laundering it into Pending", func() {
		Expect(OrderPhase(awStatuses(
			arcv1alpha1.WorkflowSucceeded, arcv1alpha1.WorkflowStopped,
		))).To(Equal(arcv1alpha1.WorkflowStopped))
	})

	It("should report Running while any workflow is in progress", func() {
		for _, phase := range []arcv1alpha1.WorkflowPhase{
			arcv1alpha1.WorkflowPending,
			arcv1alpha1.WorkflowRunning,
			arcv1alpha1.WorkflowActive,
		} {
			Expect(OrderPhase(awStatuses(
				arcv1alpha1.WorkflowSucceeded, phase,
			))).To(Equal(arcv1alpha1.WorkflowRunning))
		}
	})

	It("should report Succeeded only when every workflow succeeded", func() {
		Expect(OrderPhase(awStatuses(
			arcv1alpha1.WorkflowSucceeded, arcv1alpha1.WorkflowSucceeded,
		))).To(Equal(arcv1alpha1.WorkflowSucceeded))
	})

	It("should report Pending for workflows with no phase set", func() {
		Expect(OrderPhase(awStatuses(
			arcv1alpha1.WorkflowUnknown,
		))).To(Equal(arcv1alpha1.WorkflowPending))
	})
})
