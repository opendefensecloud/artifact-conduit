# Observability & Monitoring

ARC exposes [controller-runtime](https://book.kubebuilder.io/reference/metrics-reference.html) Prometheus metrics and Kubernetes health probes. There are no ARC-specific custom metrics; alerting is built on the standard reconciler signals for the `order` and `artifactworkflow` controllers.

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

Monitor these metrics to catch a degraded controller. Each metric carries a `controller` label; watch both `order` and `artifactworkflow`.

| Metric | Signal |
|--------|--------|
| `controller_runtime_reconcile_errors_total` | Sustained error rate — reconciler is repeatedly failing |
| `workqueue_depth` | Queue not draining — reconciler is falling behind |
| `controller_runtime_reconcile_time_seconds` | High p99 latency — reconciles are slow |
| `controller_runtime_active_workers` | Equals `controller_runtime_max_concurrent_reconciles` — worker pool saturated |

## Full Metric Catalog

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

## Workflow Execution Failures

ARC's controller-runtime metrics cover controller health only — whether ARC successfully handed work to Argo Workflows. Workflow-level failures (a workflow that ran but produced a `Failed` or `Error` outcome) are owned by Argo and are visible through its own Prometheus metrics, which include per-namespace workflow counts broken down by phase.

Add Argo's metrics endpoint to your Prometheus scrape config alongside ARC's, then monitor workflow failures there rather than here. Consult the [Argo Workflows metrics documentation](https://argo-workflows.readthedocs.io/en/latest/metrics/) for details.

See [Failed Workflow Debugging](debugging.md) for log collection details.
