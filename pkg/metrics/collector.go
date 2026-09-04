// Copyright 2025 BWI GmbH and Artifact Conduit contributors
// SPDX-License-Identifier: Apache-2.0

package metrics

import (
	"context"
	"sync/atomic"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"sigs.k8s.io/controller-runtime/pkg/client"

	arcv1alpha1 "go.opendefense.cloud/arc/api/arc/v1alpha1"
)

// collectTimeout bounds a single scrape. Collect has no context of its own, and
// a lazily created or resyncing informer can block in List, so an unbounded
// read would pile up hung scrape goroutines instead of failing.
const collectTimeout = 5 * time.Second

const (
	modeSingle = "single"
	modeCron   = "cron"
)

var (
	ordersDesc = prometheus.NewDesc(
		"arc_orders",
		"Number of Orders currently in each aggregate phase. This is a current state count, not a cumulative total.",
		[]string{"namespace", "phase"}, nil,
	)

	workflowsDesc = prometheus.NewDesc(
		"arc_artifactworkflows",
		"Number of ArtifactWorkflows currently in each phase. This is a current state count, not a cumulative total.",
		[]string{"namespace", "artifact_type", "mode", "phase"}, nil,
	)

	lastScheduledDesc = prometheus.NewDesc(
		"arc_artifactworkflow_last_scheduled_timestamp_seconds",
		"Unix timestamp of the most recent scheduling of a cron ArtifactWorkflow. "+
			"Where several cron ArtifactWorkflows share a namespace and artifact type, the oldest of their timestamps is reported.",
		[]string{"namespace", "artifact_type"}, nil,
	)

	lastSuccessDesc = prometheus.NewDesc(
		"arc_artifactworkflow_last_success_timestamp_seconds",
		"Unix timestamp of the most recent successful run of a cron ArtifactWorkflow. "+
			"Where several cron ArtifactWorkflows share a namespace and artifact type, the oldest of their timestamps is reported.",
		[]string{"namespace", "artifact_type"}, nil,
	)
)

// Collector reports current ARC state by reading the manager cache at scrape
// time. Reading on demand means deleted objects stop being reported without any
// bookkeeping, which is the failure mode that makes reconcile-loop gauges lie.
//
// It reports nothing unless this replica holds leadership. Every replica serves
// the metrics endpoint but only the leader reconciles, so ungated gauges would
// be multiplied by the replica count.
type Collector struct {
	reader   client.Reader
	isLeader atomic.Bool
}

var _ prometheus.Collector = &Collector{}

// NewCollector returns a Collector reading through the given reader, which is
// normally the manager cache.
func NewCollector(reader client.Reader) *Collector {
	return &Collector{reader: reader}
}

// Describe implements prometheus.Collector.
func (c *Collector) Describe(ch chan<- *prometheus.Desc) {
	ch <- ordersDesc
	ch <- workflowsDesc
	ch <- lastScheduledDesc
	ch <- lastSuccessDesc
}

// Collect implements prometheus.Collector.
func (c *Collector) Collect(ch chan<- prometheus.Metric) {
	if !c.isLeader.Load() {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), collectTimeout)
	defer cancel()

	c.collectOrders(ctx, ch)
	c.collectWorkflows(ctx, ch)
}

func (c *Collector) collectOrders(ctx context.Context, ch chan<- prometheus.Metric) {
	orders := &arcv1alpha1.OrderList{}
	if err := c.reader.List(ctx, orders); err != nil {
		// Report the failure rather than reporting zeros. A broken collector
		// must not look like a healthy, idle system.
		ch <- prometheus.NewInvalidMetric(ordersDesc, err)

		return
	}

	type key struct{ namespace, phase string }

	counts := map[key]int{}
	for i := range orders.Items {
		order := &orders.Items[i]
		counts[key{order.Namespace, string(OrderPhase(order.Status.ArtifactWorkflows))}]++
	}

	for k, count := range counts {
		ch <- prometheus.MustNewConstMetric(ordersDesc, prometheus.GaugeValue, float64(count), k.namespace, k.phase)
	}
}

func (c *Collector) collectWorkflows(ctx context.Context, ch chan<- prometheus.Metric) {
	workflows := &arcv1alpha1.ArtifactWorkflowList{}
	if err := c.reader.List(ctx, workflows); err != nil {
		ch <- prometheus.NewInvalidMetric(workflowsDesc, err)

		return
	}

	type key struct{ namespace, artifactType, mode, phase string }

	counts := map[key]int{}

	// The cron gauges are labelled by namespace and artifact type only, which is
	// not unique per object, so they have to be aggregated before they are
	// emitted. Two metrics with the same name and labels make the registry
	// reject the whole scrape.
	scheduled := map[cronKey]int64{}
	succeeded := map[cronKey]int64{}

	for i := range workflows.Items {
		workflow := &workflows.Items[i]
		artifactType := ArtifactTypeOf(workflow)

		mode := modeSingle
		if workflow.Spec.Cron != nil {
			mode = modeCron
		}

		// A freshly created workflow has no phase yet. An explicit Unknown
		// beats an empty label value.
		phase := string(workflow.Status.Phase)
		if phase == "" {
			phase = "Unknown"
		}

		counts[key{workflow.Namespace, artifactType, mode, phase}]++

		if mode != modeCron {
			continue
		}

		cron := cronKey{workflow.Namespace, artifactType}

		if workflow.Status.LastScheduled != nil && !workflow.Status.LastScheduled.IsZero() {
			keepOldest(scheduled, cron, workflow.Status.LastScheduled.Unix())
		}

		// Succeeded is the count Argo reports for the cron workflow, so a
		// non zero value is what makes CompletionTime a success rather than a
		// stop.
		if workflow.Status.Succeeded > 0 && !workflow.Status.CompletionTime.IsZero() {
			keepOldest(succeeded, cron, workflow.Status.CompletionTime.Unix())
		}
	}

	for k, count := range counts {
		ch <- prometheus.MustNewConstMetric(workflowsDesc, prometheus.GaugeValue, float64(count),
			k.namespace, k.artifactType, k.mode, k.phase)
	}

	emitCronTimestamps(ch, lastScheduledDesc, scheduled)
	emitCronTimestamps(ch, lastSuccessDesc, succeeded)
}

// cronKey is the label tuple of the cron timestamp gauges. It is coarser than
// one object, so several cron ArtifactWorkflows can share it.
type cronKey struct{ namespace, artifactType string }

// keepOldest reduces a group of timestamps to its minimum. The oldest value is
// the one a staleness alert has to see: taking the newest would let a healthy
// workflow mask a stalled sibling in the same group, which is the failure these
// gauges exist to catch.
func keepOldest(timestamps map[cronKey]int64, key cronKey, seconds int64) {
	if current, ok := timestamps[key]; ok && current <= seconds {
		return
	}

	timestamps[key] = seconds
}

func emitCronTimestamps(ch chan<- prometheus.Metric, desc *prometheus.Desc, timestamps map[cronKey]int64) {
	for k, seconds := range timestamps {
		ch <- prometheus.MustNewConstMetric(desc, prometheus.GaugeValue, float64(seconds), k.namespace, k.artifactType)
	}
}

// ArtifactTypeOf reads the artifact type an ArtifactWorkflow was derived from.
// Workflows created before the label existed report UnknownArtifactType rather
// than being dropped from the metrics.
func ArtifactTypeOf(workflow *arcv1alpha1.ArtifactWorkflow) string {
	if value, ok := workflow.Labels[arcv1alpha1.LabelArtifactType]; ok && value != "" {
		return value
	}

	return UnknownArtifactType
}
