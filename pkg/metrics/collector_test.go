// Copyright 2025 BWI GmbH and Artifact Conduit contributors
// SPDX-License-Identifier: Apache-2.0

package metrics

import (
	"context"
	"strings"

	"github.com/prometheus/client_golang/prometheus/testutil"
	dto "github.com/prometheus/client_model/go"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	arcv1alpha1 "go.opendefense.cloud/arc/api/arc/v1alpha1"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func newScheme() *runtime.Scheme {
	scheme := runtime.NewScheme()
	Expect(arcv1alpha1.AddToScheme(scheme)).To(Succeed())

	return scheme
}

func aw(namespace, name, artifactType string, cron bool, phase arcv1alpha1.WorkflowPhase) *arcv1alpha1.ArtifactWorkflow {
	obj := &arcv1alpha1.ArtifactWorkflow{
		ObjectMeta: metav1.ObjectMeta{Namespace: namespace, Name: name},
		Status:     arcv1alpha1.ArtifactWorkflowStatus{WorkflowStatus: arcv1alpha1.WorkflowStatus{Phase: phase}},
	}
	if artifactType != "" {
		obj.Labels = map[string]string{arcv1alpha1.LabelArtifactType: artifactType}
	}
	if cron {
		obj.Spec.Cron = &arcv1alpha1.Cron{}
	}

	return obj
}

// cronWorkflow builds a cron ArtifactWorkflow that has already run, so both
// freshness timestamps are populated.
func cronWorkflow(namespace, name, artifactType string, scheduled, completed int64) *arcv1alpha1.ArtifactWorkflow {
	obj := aw(namespace, name, artifactType, true, arcv1alpha1.WorkflowActive)

	lastScheduled := metav1.Unix(scheduled, 0)
	obj.Status.LastScheduled = &lastScheduled
	obj.Status.CompletionTime = metav1.Unix(completed, 0)
	obj.Status.Succeeded = 1

	return obj
}

// seriesCount reports how many series a gathered metric family holds. A name
// that was not gathered at all counts as zero rather than failing.
func seriesCount(families []*dto.MetricFamily, name string) int {
	for _, family := range families {
		if family.GetName() == name {
			return len(family.GetMetric())
		}
	}

	return 0
}

var _ = Describe("Collector", func() {
	It("should emit nothing while not the leader", func() {
		client := fake.NewClientBuilder().WithScheme(newScheme()).
			WithObjects(aw("team-a", "one", "oci", false, arcv1alpha1.WorkflowRunning)).Build()

		collector := NewCollector(client)

		Expect(testutil.CollectAndCount(collector)).To(Equal(0))
	})

	It("should count workflows by namespace, type, mode and phase", func() {
		client := fake.NewClientBuilder().WithScheme(newScheme()).WithObjects(
			aw("team-a", "one", "oci", false, arcv1alpha1.WorkflowRunning),
			aw("team-a", "two", "oci", false, arcv1alpha1.WorkflowRunning),
			aw("team-a", "three", "helm", true, arcv1alpha1.WorkflowActive),
		).Build()

		collector := NewCollector(client)
		collector.isLeader.Store(true)

		expected := `
# HELP arc_artifactworkflows Number of ArtifactWorkflows currently in each phase. This is a current state count, not a cumulative total.
# TYPE arc_artifactworkflows gauge
arc_artifactworkflows{artifact_type="helm",mode="cron",namespace="team-a",phase="Active"} 1
arc_artifactworkflows{artifact_type="oci",mode="single",namespace="team-a",phase="Running"} 2
`
		Expect(testutil.CollectAndCompare(collector, strings.NewReader(expected), "arc_artifactworkflows")).To(Succeed())
	})

	It("should keep counts separate across namespaces", func() {
		client := fake.NewClientBuilder().WithScheme(newScheme()).WithObjects(
			aw("team-a", "one", "oci", false, arcv1alpha1.WorkflowRunning),
			aw("team-b", "two", "oci", false, arcv1alpha1.WorkflowRunning),
		).Build()

		collector := NewCollector(client)
		collector.isLeader.Store(true)

		expected := `
# HELP arc_artifactworkflows Number of ArtifactWorkflows currently in each phase. This is a current state count, not a cumulative total.
# TYPE arc_artifactworkflows gauge
arc_artifactworkflows{artifact_type="oci",mode="single",namespace="team-a",phase="Running"} 1
arc_artifactworkflows{artifact_type="oci",mode="single",namespace="team-b",phase="Running"} 1
`
		Expect(testutil.CollectAndCompare(collector, strings.NewReader(expected), "arc_artifactworkflows")).To(Succeed())
	})

	It("should report workflows without the type label as unknown", func() {
		client := fake.NewClientBuilder().WithScheme(newScheme()).WithObjects(
			aw("team-a", "legacy", "", false, arcv1alpha1.WorkflowSucceeded),
		).Build()

		collector := NewCollector(client)
		collector.isLeader.Store(true)

		expected := `
# HELP arc_artifactworkflows Number of ArtifactWorkflows currently in each phase. This is a current state count, not a cumulative total.
# TYPE arc_artifactworkflows gauge
arc_artifactworkflows{artifact_type="unknown",mode="single",namespace="team-a",phase="Succeeded"} 1
`
		Expect(testutil.CollectAndCompare(collector, strings.NewReader(expected), "arc_artifactworkflows")).To(Succeed())
	})

	It("should normalise an empty phase to Unknown", func() {
		client := fake.NewClientBuilder().WithScheme(newScheme()).WithObjects(
			aw("team-a", "fresh", "oci", false, arcv1alpha1.WorkflowUnknown),
		).Build()

		collector := NewCollector(client)
		collector.isLeader.Store(true)

		expected := `
# HELP arc_artifactworkflows Number of ArtifactWorkflows currently in each phase. This is a current state count, not a cumulative total.
# TYPE arc_artifactworkflows gauge
arc_artifactworkflows{artifact_type="oci",mode="single",namespace="team-a",phase="Unknown"} 1
`
		Expect(testutil.CollectAndCompare(collector, strings.NewReader(expected), "arc_artifactworkflows")).To(Succeed())
	})

	It("should roll orders up to an aggregate phase", func() {
		order := &arcv1alpha1.Order{
			ObjectMeta: metav1.ObjectMeta{Namespace: "team-a", Name: "nightly"},
			Status: arcv1alpha1.OrderStatus{
				ArtifactWorkflows: map[string]arcv1alpha1.OrderArtifactWorkflowStatus{
					"a": {WorkflowStatus: arcv1alpha1.WorkflowStatus{Phase: arcv1alpha1.WorkflowSucceeded}},
					"b": {WorkflowStatus: arcv1alpha1.WorkflowStatus{Phase: arcv1alpha1.WorkflowFailed}},
				},
			},
		}

		client := fake.NewClientBuilder().WithScheme(newScheme()).WithObjects(order).Build()

		collector := NewCollector(client)
		collector.isLeader.Store(true)

		expected := `
# HELP arc_orders Number of Orders currently in each aggregate phase. This is a current state count, not a cumulative total.
# TYPE arc_orders gauge
arc_orders{namespace="team-a",phase="Failed"} 1
`
		Expect(testutil.CollectAndCompare(collector, strings.NewReader(expected), "arc_orders")).To(Succeed())
	})

	It("should publish cron freshness timestamps only for successful cron workflows", func() {
		scheduled := metav1.NewTime(metav1.Unix(1700000000, 0).Time)
		completed := metav1.NewTime(metav1.Unix(1700000600, 0).Time)

		cronAW := aw("team-a", "sync", "oci", true, arcv1alpha1.WorkflowActive)
		cronAW.Status.LastScheduled = &scheduled
		cronAW.Status.CompletionTime = completed
		cronAW.Status.Succeeded = 3

		neverRan := aw("team-a", "new", "oci", true, arcv1alpha1.WorkflowPending)

		client := fake.NewClientBuilder().WithScheme(newScheme()).WithObjects(cronAW, neverRan).Build()

		collector := NewCollector(client)
		collector.isLeader.Store(true)

		expected := `
# HELP arc_artifactworkflow_last_success_timestamp_seconds Unix timestamp of the most recent successful run of a cron ArtifactWorkflow. Where several cron ArtifactWorkflows share a namespace and artifact type, the oldest of their timestamps is reported.
# TYPE arc_artifactworkflow_last_success_timestamp_seconds gauge
arc_artifactworkflow_last_success_timestamp_seconds{artifact_type="oci",namespace="team-a"} 1.7000006e+09
`
		Expect(testutil.CollectAndCompare(collector, strings.NewReader(expected),
			"arc_artifactworkflow_last_success_timestamp_seconds")).To(Succeed())
	})

	It("should report the oldest timestamp when cron workflows share a namespace and type", func() {
		older := cronWorkflow("team-a", "sync-a", "oci", 1700000000, 1700000600)
		newer := cronWorkflow("team-a", "sync-b", "oci", 1700009000, 1700009600)

		// A workflow outside the group must keep its own series rather than be
		// folded into the minimum.
		other := cronWorkflow("team-b", "sync-c", "helm", 1700020000, 1700020600)

		client := fake.NewClientBuilder().WithScheme(newScheme()).WithObjects(older, newer, other).Build()

		collector := NewCollector(client)
		collector.isLeader.Store(true)

		// One series per namespace and artifact type, carrying the older of the
		// two team-a timestamps, so a staleness alert fires on the stalest
		// workflow in the group.
		expectedScheduled := `
# HELP arc_artifactworkflow_last_scheduled_timestamp_seconds Unix timestamp of the most recent scheduling of a cron ArtifactWorkflow. Where several cron ArtifactWorkflows share a namespace and artifact type, the oldest of their timestamps is reported.
# TYPE arc_artifactworkflow_last_scheduled_timestamp_seconds gauge
arc_artifactworkflow_last_scheduled_timestamp_seconds{artifact_type="helm",namespace="team-b"} 1.70002e+09
arc_artifactworkflow_last_scheduled_timestamp_seconds{artifact_type="oci",namespace="team-a"} 1.7e+09
`
		Expect(testutil.CollectAndCompare(collector, strings.NewReader(expectedScheduled),
			"arc_artifactworkflow_last_scheduled_timestamp_seconds")).To(Succeed())

		expectedSuccess := `
# HELP arc_artifactworkflow_last_success_timestamp_seconds Unix timestamp of the most recent successful run of a cron ArtifactWorkflow. Where several cron ArtifactWorkflows share a namespace and artifact type, the oldest of their timestamps is reported.
# TYPE arc_artifactworkflow_last_success_timestamp_seconds gauge
arc_artifactworkflow_last_success_timestamp_seconds{artifact_type="helm",namespace="team-b"} 1.7000206e+09
arc_artifactworkflow_last_success_timestamp_seconds{artifact_type="oci",namespace="team-a"} 1.7000006e+09
`
		Expect(testutil.CollectAndCompare(collector, strings.NewReader(expectedSuccess),
			"arc_artifactworkflow_last_success_timestamp_seconds")).To(Succeed())
	})

	It("should gather cleanly with cron workflows sharing a namespace and type", func() {
		// Duplicate series are rejected by the registry for the whole response,
		// not just for the offending metric, so this asserts on a full Gather
		// rather than on one filtered metric name.
		client := fake.NewClientBuilder().WithScheme(newScheme()).WithObjects(
			cronWorkflow("team-a", "sync-a", "oci", 1700000000, 1700000600),
			cronWorkflow("team-a", "sync-b", "oci", 1700009000, 1700009600),
		).Build()

		collector := NewCollector(client)
		collector.isLeader.Store(true)

		registry := prometheus.NewPedanticRegistry()
		Expect(registry.Register(collector)).To(Succeed())

		families, err := registry.Gather()
		Expect(err).NotTo(HaveOccurred())

		for _, name := range []string{
			"arc_artifactworkflow_last_scheduled_timestamp_seconds",
			"arc_artifactworkflow_last_success_timestamp_seconds",
		} {
			Expect(seriesCount(families, name)).To(Equal(1), "%s should hold one series per namespace and artifact type", name)
		}
	})
})

var _ = Describe("LeaderRunnable", func() {
	It("should report only while running", func() {
		client := fake.NewClientBuilder().WithScheme(newScheme()).WithObjects(
			aw("team-a", "one", "oci", false, arcv1alpha1.WorkflowRunning),
		).Build()

		collector := NewCollector(client)
		Expect(testutil.CollectAndCount(collector)).To(Equal(0))

		ctx, cancel := context.WithCancel(context.Background())

		done := make(chan error, 1)
		go func() { done <- collector.LeaderRunnable().Start(ctx) }()

		Eventually(func() int { return testutil.CollectAndCount(collector) }).Should(BeNumerically(">", 0))

		cancel()
		Expect(<-done).To(Succeed())
		Eventually(func() int { return testutil.CollectAndCount(collector) }).Should(Equal(0))
	})
})
