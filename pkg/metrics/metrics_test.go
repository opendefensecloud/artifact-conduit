// Copyright 2025 BWI GmbH and Artifact Conduit contributors
// SPDX-License-Identifier: Apache-2.0

package metrics

import (
	"github.com/prometheus/client_golang/prometheus/testutil"
	dto "github.com/prometheus/client_model/go"
	ctrlmetrics "sigs.k8s.io/controller-runtime/pkg/metrics"

	arcv1alpha1 "go.opendefense.cloud/arc/api/arc/v1alpha1"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// cumulativeBucket reads the cumulative count of the given bucket upper bound
// for the artifact_type="oci",result="succeeded" series of
// arc_artifactworkflow_duration_seconds, straight from ctrlmetrics.Registry.
func cumulativeBucket(upperBound float64) float64 {
	families, err := ctrlmetrics.Registry.Gather()
	Expect(err).NotTo(HaveOccurred())

	for _, mf := range families {
		if mf.GetName() != "arc_artifactworkflow_duration_seconds" {
			continue
		}

		for _, m := range mf.GetMetric() {
			if !hasLabel(m.GetLabel(), "artifact_type", "oci") || !hasLabel(m.GetLabel(), "result", ResultSucceeded) {
				continue
			}

			for _, b := range m.GetHistogram().GetBucket() {
				if b.GetUpperBound() == upperBound {
					return float64(b.GetCumulativeCount())
				}
			}
		}
	}

	return 0
}

func hasLabel(labels []*dto.LabelPair, name, value string) bool {
	for _, l := range labels {
		if l.GetName() == name && l.GetValue() == value {
			return true
		}
	}

	return false
}

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

	It("should place a 90 second observation in the 120 second bucket, not the 60 second bucket", func() {
		before60 := cumulativeBucket(60)
		before120 := cumulativeBucket(120)

		ObserveDuration("oci", ResultSucceeded, 90)

		after60 := cumulativeBucket(60)
		after120 := cumulativeBucket(120)

		Expect(after120-before120).To(Equal(1.0), "a 90 second observation should land in the le=\"120\" bucket")
		Expect(after60-before60).To(Equal(0.0), "a 90 second observation must not increment the le=\"60\" bucket")
	})

	It("should expose build info as a constant one", func() {
		SetBuildInfo()
		Expect(testutil.CollectAndCount(buildInfo)).To(Equal(1))
	})
})
