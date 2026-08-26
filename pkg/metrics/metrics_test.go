// Copyright 2025 BWI GmbH and Artifact Conduit contributors
// SPDX-License-Identifier: Apache-2.0

package metrics

import (
	"github.com/prometheus/client_golang/prometheus/testutil"

	arcv1alpha1 "go.opendefense.cloud/arc/api/arc/v1alpha1"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("ResultFor", func() {
	It("should map terminal phases to results", func() {
		for phase, want := range map[arcv1alpha1.WorkflowPhase]string{
			arcv1alpha1.WorkflowSucceeded: ResultSucceeded,
			arcv1alpha1.WorkflowFailed:    ResultFailed,
			arcv1alpha1.WorkflowError:     ResultError,
		} {
			got, ok := ResultFor(phase)
			Expect(ok).To(BeTrue(), "phase %q should be terminal", phase)
			Expect(got).To(Equal(want))
		}
	})

	It("should reject non-terminal phases", func() {
		for _, phase := range []arcv1alpha1.WorkflowPhase{
			arcv1alpha1.WorkflowUnknown,
			arcv1alpha1.WorkflowPending,
			arcv1alpha1.WorkflowRunning,
			arcv1alpha1.WorkflowActive,
			arcv1alpha1.WorkflowStopped,
		} {
			_, ok := ResultFor(phase)
			Expect(ok).To(BeFalse(), "phase %q should not be terminal", phase)
		}
	})
})

var _ = Describe("Recording helpers", func() {
	It("should count completions per namespace, type and result", func() {
		before := testutil.ToFloat64(completions.WithLabelValues("team-a", "oci", ResultSucceeded))

		RecordCompletion("team-a", "oci", ResultSucceeded)
		RecordCompletion("team-a", "oci", ResultSucceeded)

		after := testutil.ToFloat64(completions.WithLabelValues("team-a", "oci", ResultSucceeded))
		Expect(after - before).To(Equal(2.0))
	})

	It("should count reconcile errors per controller and reason", func() {
		before := testutil.ToFloat64(reconcileErrors.WithLabelValues("order", "InvalidSecret"))

		RecordReconcileError("order", "InvalidSecret")

		after := testutil.ToFloat64(reconcileErrors.WithLabelValues("order", "InvalidSecret"))
		Expect(after - before).To(Equal(1.0))
	})

	It("should record durations in the histogram", func() {
		ObserveDuration("oci", ResultSucceeded, 90)

		Expect(testutil.CollectAndCount(duration)).To(BeNumerically(">", 0))
	})

	It("should expose build info as a constant one", func() {
		SetBuildInfo()
		Expect(testutil.CollectAndCount(buildInfo)).To(Equal(1))
	})
})
