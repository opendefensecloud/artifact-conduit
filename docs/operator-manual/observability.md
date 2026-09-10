# Observability & Monitoring

ARC exposes Prometheus metrics and Kubernetes health probes. The controller manager emits ARC's own metrics for Orders and ArtifactWorkflows alongside the standard [controller-runtime](https://book.kubebuilder.io/reference/metrics-reference.html) reconciler signals, and the API Server exposes the usual Kubernetes API server metrics. A reference dashboard ships with the Helm chart.

## Enabling the Metrics Endpoint

Metrics are **disabled by default**. Enable them via Helm:

> **Note:** `serviceMonitor.enabled` requires the [Prometheus Operator](https://prometheus-operator.dev/) CRDs to be present in the cluster. If you are not using the Prometheus Operator, omit it.

```yaml
controller:
  metrics:
    enabled: true
    service:
      port: 8443
    # Optional: use cert-manager
    certManager:
      enabled: true
      # Optional overrides — defaults to the shared certManager issuer and certificate values
      issuerRef:
        kind: Issuer          # or ClusterIssuer
        name: my-metrics-issuer
      duration: 2160h
      renewBefore: 720h
    serviceMonitor:
      enabled: true
      # Required only with certManager (HTTPS) — see "Securing the scrape" below.
      tokenSecret:
        name: arc-metrics-scrape-token
  args:
    metricsBindAddress: ":8443"
```

> **HTTP vs HTTPS:** with `certManager.enabled: false` the endpoint is served over plain HTTP and is unauthenticated — no token wiring is needed and `serviceMonitor.tokenSecret` can be omitted. The steps below apply only to the secured (cert-manager/HTTPS) path.

### Securing the scrape

With cert-manager enabled the controller enforces RBAC authn/authz on `GET /metrics`, so Prometheus must present a bearer token. The chart creates the `{release-name}-controller-manager-metrics-reader` ClusterRole (grants `GET /metrics` and nothing else) but does **not** bind it — you wire up a least-privilege scrape identity:

1. Create a ServiceAccount in the ARC release namespace and bind it to the `{release-name}-controller-manager-metrics-reader` ClusterRole (substitute your release name for `arc`):

```yaml
apiVersion: v1
kind: ServiceAccount
metadata:
  name: arc-metrics-scraper
  namespace: arc-system          # the release namespace
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: arc-metrics-scraper-reader
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: arc-controller-manager-metrics-reader
subjects:
  - kind: ServiceAccount
    name: arc-metrics-scraper
    namespace: arc-system
```

2. Create a token for it as a Secret in the ARC release namespace (the ServiceMonitor and its credentials Secret must share a namespace):

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: arc-metrics-scrape-token
  namespace: arc-system           # the release namespace
  annotations:
    kubernetes.io/service-account.name: arc-metrics-scraper
type: kubernetes.io/service-account-token
```

3. Point the ServiceMonitor at it via `controller.metrics.serviceMonitor.tokenSecret.name` (shown above).

## Health Probes

Always exposed on port `8081`:

| Path       | Purpose                              |
| ---------- | ------------------------------------ |
| `/healthz` | Liveness — restart on failure        |
| `/readyz`  | Readiness — remove from service      |

## Top Health Signals

Start here. These answer whether ARC is doing its job.

| Metric | Signal |
|--------|--------|
| `arc_orders` | Orders sitting in `Failed`, or a `Pending` count that never drains |
| `arc_artifactworkflow_completions_total` | Success ratio dropping, by `result` |
| `arc_reconcile_errors_total` | Which failure `reason` dominates |
| `arc_artifactworkflow_last_success_timestamp_seconds` | `time() - <metric> > threshold` catches a cron sync that stopped running |
| `arc_collector_errors_total` | Non zero means the gauges above are stale, not that ARC is idle |

The controller-runtime signals below are second line diagnostics, useful once you
know something is wrong. Each carries a `controller` label; watch both `order` and
`artifactworkflow`.

| Metric | Signal |
|--------|--------|
| `controller_runtime_reconcile_errors_total` | Sustained error rate — reconciler is repeatedly failing |
| `workqueue_depth` | Queue not draining — reconciler is falling behind |
| `controller_runtime_reconcile_time_seconds` | High p99 latency — reconciles are slow |
| `controller_runtime_active_workers` | Equals `controller_runtime_max_concurrent_reconciles` — worker pool saturated |

## ARC Metric Catalog

Emitted by the controller manager. No label carries an object name or a message, so
cardinality stays bounded by namespaces and artifact types.

| Metric | Type | Labels | Description |
|--------|------|--------|-------------|
| `arc_orders` | Gauge | `namespace`, `phase` | Orders currently in each aggregate phase |
| `arc_artifactworkflows` | Gauge | `namespace`, `artifact_type`, `mode`, `phase` | ArtifactWorkflows currently in each phase |
| `arc_artifactworkflow_completions_total` | Counter | `namespace`, `artifact_type`, `result` | Single run workflows reaching a terminal phase |
| `arc_artifactworkflow_duration_seconds` | Histogram | `artifact_type`, `result` | Argo execution time of single run workflows |
| `arc_artifactworkflow_last_scheduled_timestamp_seconds` | Gauge | `namespace`, `artifact_type` | Cron only, when the group was last scheduled |
| `arc_artifactworkflow_last_success_timestamp_seconds` | Gauge | `namespace`, `artifact_type` | Cron only, when the group last succeeded |
| `arc_reconcile_errors_total` | Counter | `controller`, `reason` | Classified reconcile failures |
| `arc_collector_errors_total` | Counter | `resource` | Cache reads that failed while collecting the gauges |

The gauges are read from the controller's cache at scrape time rather than tracked in
the reconcile loop, so a deleted Order stops being counted with no bookkeeping. They
are emitted sparsely: a phase with no objects produces no series at all, so a panel
must not read an absent series as zero.

### Things that look like bugs and are not

- **An Order containing a cron artifact never settles.** It reads `Running` while a run
  is in flight and `Succeeded` between runs. `Running` means "has work in flight", not
  "unhealthy".
- **Completions and duration cover single run workflows only.** Cron completion data in
  ARC is a copy of Argo's cumulative counters, which jump by more than one and reset on
  a force reconcile, so counting them would be dishonest. Use the freshness timestamps
  for cron health instead.
- **`arc_reconcile_errors_total` is retry inflated.** A permanently failing reconcile
  increments its reason on every backoff. Fine for `rate()` and for "which reason
  dominates", misleading as a count of distinct failures.
- **The freshness gauges reduce to the oldest** in each `(namespace, artifact_type)`
  group, so a stalled workflow is never masked by a healthy sibling. A group that has
  never succeeded contributes no series.
- **ArtifactWorkflows created before this feature report `artifact_type="unknown"`.**
  The label is stamped at creation and existing objects are never relabelled. For cron
  workflows, which are never recreated, this is permanent.
- **`reason` matches the Kubernetes Event.** The same word appears in
  `kubectl describe order`.

## Full controller-runtime Metric Catalog

All metrics are emitted per controller (`order`, `artifactworkflow`) or workqueue (`name=order|artifactworkflow`).

| Metric | Type | Labels | Description |
|--------|------|--------|-------------|
| `controller_runtime_reconcile_total` | Counter | `controller`, `result` (`success`, `error`, `requeue`, `requeue_after`) | Total reconcile attempts |
| `controller_runtime_reconcile_errors_total` | Counter | `controller` | Reconciles that returned an error |
| `controller_runtime_reconcile_time_seconds` | Histogram | `controller` | Reconcile duration |
| `controller_runtime_active_workers` | Gauge | `controller` | Workers currently reconciling |
| `controller_runtime_max_concurrent_reconciles` | Gauge | `controller` | Worker pool size |
| `workqueue_depth` | Gauge | `name`, `priority` | Items waiting in the queue |
| `workqueue_adds_total` | Counter | `name` | Items added to the queue |
| `workqueue_queue_duration_seconds` | Histogram | `name` | Wait time before processing |
| `workqueue_work_duration_seconds` | Histogram | `name` | Processing time per item |
| `workqueue_retries_total` | Counter | `name` | Requeues due to error/rate-limit |
| `rest_client_requests_total` | Counter | `code`, `method`, `host` | Kubernetes API calls from the controller |

## Scraping the API Server

The ARC API Server always serves `/metrics` on its secure port. Nothing scrapes it until
you turn the ServiceMonitor on:

```yaml
apiserver:
  metrics:
    serviceMonitor:
      enabled: true
      tokenSecret:
        name: arc-metrics-scrape-token
```

TLS verification is on by default and resolves the cert-manager issued CA and the
service DNS name on its own. Rendering fails with an explicit message if you enable
verification while running without cert-manager, rather than producing a ServiceMonitor
that silently cannot connect.

A scrape reaches the Service directly rather than through the aggregation layer, so the
API Server authenticates the bearer token itself. The chart binds `system:auth-delegator`
to the API Server ServiceAccount when this monitor is enabled, and creates an unbound
`{release-name}-apiserver-metrics-reader` ClusterRole. Wire a scrape identity to it the
same way as for the controller above, using `arc-apiserver-metrics-reader` as the role.

`apiserver_request_total` and `workqueue_depth` are also emitted by the Kubernetes
control plane. Both ServiceMonitors set `targetLabels`, so every ARC series carries
`app_kubernetes_io_part_of="arc"` and `app_kubernetes_io_component`. Filter on those or
your queries will mix ARC together with the control plane.

Both monitors also set `honorLabels: true`. ARC's `namespace` label names the Order's
namespace, and without this Prometheus renames it to `exported_namespace`.

## Reference Dashboard

A Grafana dashboard ships with the chart, off by default:

```yaml
dashboards:
  enabled: true
```

It renders as a ConfigMap labelled for the Grafana sidecar. **The kube-prometheus-stack
sidecar only watches its own namespace by default**, so either install Grafana with
`sidecar.dashboards.searchNamespace=ALL` or set `dashboards.namespace` to the namespace
Grafana runs in. Without one of those the ConfigMap is created, nothing appears in
Grafana, and there is no error to go on.

Four sections: whether ARC is up, whether work is getting through, reconciler internals,
and runtime. It is a starting point rather than a finished observability product, so
copy it into your own folder before editing.

There is no container restart panel. That needs
`kube_pod_container_status_restarts_total` from kube-state-metrics, which a chart
shipped dashboard cannot assume is installed. If you run it, the query is
`sum by (pod) (kube_pod_container_status_restarts_total{namespace="arc-system"})`.

## Workflow Execution Failures

ARC's controller-runtime metrics cover controller health only — whether ARC successfully handed work to Argo Workflows. Workflow-level failures (a workflow that ran but produced a `Failed` or `Error` outcome) are owned by Argo and are visible through its own Prometheus metrics, which include per-namespace workflow counts broken down by phase.

Add Argo's metrics endpoint to your Prometheus scrape config alongside ARC's, then monitor workflow failures there rather than here. Consult the [Argo Workflows metrics documentation](https://argo-workflows.readthedocs.io/en/latest/metrics/) for details.

See [Failed Workflow Debugging](debugging.md) for log collection details.
