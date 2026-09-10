// Copyright 2025 BWI GmbH and Artifact Conduit contributors
// SPDX-License-Identifier: Apache-2.0

package metrics

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// arcMetrics is every series ARC emits, including the ones Prometheus derives
// from a histogram. Keep it in step with metrics.go and collector.go.
var arcMetrics = []string{
	"arc_orders",
	"arc_artifactworkflows",
	"arc_artifactworkflow_last_scheduled_timestamp_seconds",
	"arc_artifactworkflow_last_success_timestamp_seconds",
	"arc_artifactworkflow_completions_total",
	"arc_artifactworkflow_duration_seconds",
	"arc_artifactworkflow_duration_seconds_bucket",
	"arc_artifactworkflow_duration_seconds_count",
	"arc_artifactworkflow_duration_seconds_sum",
	"arc_reconcile_errors_total",
	"arc_collector_errors_total",
}

// upstreamMetrics are series ARC does not emit but the dashboard may query.
var upstreamMetrics = []string{
	"up",
	"leader_election_master_status",
	"apiserver_request_total",
	"apiserver_request_duration_seconds_bucket",
	"controller_runtime_reconcile_errors_total",
	"controller_runtime_reconcile_time_seconds_bucket",
	"controller_runtime_active_workers",
	"controller_runtime_max_concurrent_reconciles",
	"workqueue_depth",
	"go_goroutines",
	"process_resident_memory_bytes",
}

// promqlBuiltins and labelNames are identifiers that are not metric names. Label
// names have to be skipped explicitly: the scan otherwise treats anything
// containing an underscore as a metric, and artifact_type would fail the test
// against a perfectly correct dashboard.
var promqlBuiltins = map[string]bool{
	"sum": true, "rate": true, "irate": true, "avg": true, "max": true, "min": true,
	"count": true, "topk": true, "bottomk": true, "by": true, "without": true,
	"histogram_quantile": true, "clamp_min": true, "clamp_max": true, "time": true,
	"increase": true, "delta": true, "vector": true, "scalar": true, "absent": true,
	"on": true, "ignoring": true, "group_left": true, "group_right": true,
	"and": true, "or": true, "unless": true, "offset": true,
}

var labelNames = map[string]bool{
	"artifact_type": true, "namespace": true, "result": true, "phase": true,
	"controller": true, "reason": true, "code": true, "pod": true, "name": true,
	"resource": true, "job": true, "instance": true, "le": true, "mode": true,
	"app_kubernetes_io_part_of": true, "app_kubernetes_io_component": true,
}

var identifier = regexp.MustCompile(`[a-zA-Z_][a-zA-Z0-9_]*`)

// interpolate replaces the Grafana template constructs, which are not PromQL.
var interpolate = strings.NewReplacer(
	"$__rate_interval", "5m",
	"$namespace", "default",
	"$artifact_type", "oci",
	"${datasource}", "prometheus",
)

// dashboardExprs returns every panel expression in a dashboard document.
func dashboardExprs(raw []byte) []string {
	var doc any
	Expect(json.Unmarshal(raw, &doc)).To(Succeed())

	var exprs []string

	var walk func(node any)
	walk = func(node any) {
		switch typed := node.(type) {
		case map[string]any:
			for key, value := range typed {
				if key == "expr" {
					if s, ok := value.(string); ok && s != "" {
						exprs = append(exprs, s)
					}
				}

				walk(value)
			}
		case []any:
			for _, value := range typed {
				walk(value)
			}
		}
	}
	walk(doc)

	return exprs
}

var _ = Describe("Shipped dashboards", func() {
	var files []string

	BeforeEach(func() {
		var err error
		files, err = filepath.Glob(filepath.Join("..", "..", "charts", "arc", "files", "dashboards", "*.json"))
		Expect(err).NotTo(HaveOccurred())
		Expect(files).NotTo(BeEmpty(), "no dashboards found, has the chart path moved?")
	})

	It("should only query metrics that exist", func() {
		known := map[string]bool{}
		for _, name := range append(append([]string{}, arcMetrics...), upstreamMetrics...) {
			known[name] = true
		}

		for _, file := range files {
			raw, err := os.ReadFile(file)
			Expect(err).NotTo(HaveOccurred())

			exprs := dashboardExprs(raw)
			Expect(exprs).NotTo(BeEmpty(), "%s has no panel expressions", file)

			for _, expr := range exprs {
				for _, candidate := range identifier.FindAllString(interpolate.Replace(expr), -1) {
					if promqlBuiltins[candidate] || labelNames[candidate] || !strings.Contains(candidate, "_") {
						continue
					}

					Expect(known).To(HaveKey(candidate),
						"%s queries unknown metric %q in: %s", file, candidate, expr)
				}
			}
		}
	})

	It("should reference a datasource variable rather than a fixed uid", func() {
		for _, file := range files {
			raw, err := os.ReadFile(file)
			Expect(err).NotTo(HaveOccurred())

			var doc map[string]any
			Expect(json.Unmarshal(raw, &doc)).To(Succeed())
			Expect(doc).To(HaveKey("uid"))
			Expect(string(raw)).To(ContainSubstring("${datasource}"))
		}
	})
})
