# ARC Helm Chart

This Helm chart deploys Artifact Conduit (ARC) - a Kubernetes-native system for artifact procurement and transfer across security zones with automated scanning and validation.

## Prerequisites

- Kubernetes 1.28+
- Helm 3.8+
- cert-manager (for TLS certificate management)
- Argo Workflows (for artifact processing workflows)

## Installation

### Install from OCI Registry

ARC Helm charts are published to GitHub Container Registry as OCI artifacts.

```bash
# Pull and install the latest version
helm install arc oci://ghcr.io/opendefensecloud/charts/arc \
  --namespace arc-system \
  --create-namespace

# Install a specific version
helm install arc oci://ghcr.io/opendefensecloud/charts/arc --version 0.1.0 \
  --namespace arc-system \
  --create-namespace

# Install with custom values
helm install arc oci://ghcr.io/opendefensecloud/charts/arc \
  --namespace arc-system \
  --create-namespace \
  -f custom-values.yaml
```

### Install from Source

For development or testing, install directly from the source repository:

```bash
# Clone the repository
git clone https://github.com/opendefensecloud/artifact-conduit.git
cd artifact-conduit

# Install from local chart
helm install arc ./charts/arc --namespace arc-system --create-namespace
```

### Install cert-manager (if not already installed)

```bash
kubectl apply -f https://github.com/cert-manager/cert-manager/releases/latest/download/cert-manager.yaml
```

### Install Argo Workflows (if not already installed)

```bash
kubectl create namespace argo
kubectl apply -n argo -f https://github.com/argoproj/argo-workflows/releases/latest/download/install.yaml
```

## Components

The chart deploys three main components:

1. **API Server** - Extension API server providing custom resources
2. **Controller Manager** - Reconciles Order and ArtifactWorkflow resources
3. **etcd** - Dedicated storage backend for the API server

## Examples

### Basic Installation

```bash
helm install arc oci://ghcr.io/opendefensecloud/charts/arc \
  --namespace arc-system \
  --create-namespace
```

### High Availability Setup

```yaml
# ha-values.yaml
controller:
  replicaCount: 3
  args:
    leaderElect: true
  affinity:
    podAntiAffinity:
      preferredDuringSchedulingIgnoredDuringExecution:
      - weight: 100
        podAffinityTerm:
          labelSelector:
            matchLabels:
              app.kubernetes.io/component: controller-manager
          topologyKey: kubernetes.io/hostname

etcd:
  replicaCount: 3
  affinity:
    podAntiAffinity:
      preferredDuringSchedulingIgnoredDuringExecution:
      - weight: 100
        podAffinityTerm:
          labelSelector:
            matchLabels:
              app.kubernetes.io/component: etcd
          topologyKey: kubernetes.io/hostname
```

```bash
helm install arc oci://ghcr.io/opendefensecloud/charts/arc -f ha-values.yaml
```

### Enable Metrics with Prometheus

```yaml
# metrics-values.yaml
controller:
  metrics:
    enabled: true
    serviceMonitor:
      enabled: true
      additionalLabels:
        prometheus: kube-prometheus
```

```bash
helm install arc oci://ghcr.io/opendefensecloud/charts/arc -f metrics-values.yaml
```

### Custom Storage Class

```yaml
# storage-values.yaml
global:
  storageClass: fast-ssd

etcd:
  persistence:
    size: 10Gi
```

```bash
helm install arc oci://ghcr.io/opendefensecloud/charts/arc -f storage-values.yaml
```

### Using External etcd

```yaml
# external-etcd-values.yaml
etcd:
  enabled: false

apiserver:
  args:
    etcdServers: "http://external-etcd:2379"
```

```bash
helm install arc oci://ghcr.io/opendefensecloud/charts/arc -f external-etcd-values.yaml
```

## Upgrading

```bash
# Upgrade to the latest version
helm upgrade arc oci://ghcr.io/opendefensecloud/charts/arc --namespace arc-system

# Upgrade to a specific version
helm upgrade arc oci://ghcr.io/opendefensecloud/charts/arc --version 0.2.0 --namespace arc-system
```

## Uninstalling

```bash
helm uninstall arc --namespace arc-system
```

## Post-Installation

After installing ARC, you need to:

1. **Verify Installation**
   ```bash
   kubectl get all -n arc-system
   kubectl get apiservice v1alpha1.arc.opendefense.cloud
   ```

2. **Create WorkflowTemplates**
   Deploy Argo WorkflowTemplates that define how artifacts are processed.
   See [examples/oci/cluster-workflow-template.yaml](https://github.com/opendefensecloud/artifact-conduit/blob/main/examples/oci/cluster-workflow-template.yaml)

3. **Create ArtifactTypes**
   Define supported artifact types (OCI images, Helm charts, etc.)
   ```bash
   kubectl apply -f examples/oci/artifact-type.yaml
   ```

4. **Create Endpoints**
   Define source and destination registries/repositories
   ```bash
   kubectl apply -f examples/oci/order-and-endpoints.yaml
   ```

5. **Submit Orders**
   Create Order resources to transfer artifacts
   ```bash
   kubectl apply -f examples/order.yaml
   ```

## Troubleshooting

### API Server not ready

Check certificate issuance:
```bash
kubectl get certificate -n arc-system
kubectl describe certificate arc-apiserver-cert -n arc-system
```

Check API Server logs:
```bash
kubectl logs -n arc-system -l app.kubernetes.io/component=apiserver
```

### Controller Manager issues

Check controller logs:
```bash
kubectl logs -n arc-system -l app.kubernetes.io/component=controller-manager -f
```

Check RBAC permissions:
```bash
kubectl auth can-i --as=system:serviceaccount:arc-system:arc-controller-manager --list
```

### etcd connection issues

Check etcd status:
```bash
kubectl get statefulset -n arc-system
kubectl logs -n arc-system arc-etcd-0
```

Test connectivity:
```bash
kubectl run -it --rm debug --image=busybox --restart=Never -- wget -O- http://arc-etcd:2379/health
```

## Migration from Kustomize

If you're currently using Kustomize to deploy ARC:

1. Export your current configuration:
   ```bash
   kubectl get deployment,service,configmap -n arc-system -o yaml > current-config.yaml
   ```

2. Map Kustomize overlays to Helm values:
   - Image overrides → `*.image.repository` and `*.image.tag`
   - Resource patches → `*.resources`
   - Replica counts → `*.replicaCount`

3. Create a values file with your customizations

4. Test with dry-run:
   ```bash
   helm install arc oci://ghcr.io/opendefensecloud/charts/arc --dry-run --debug -f your-values.yaml
   ```

5. Uninstall Kustomize deployment and install Helm chart

## Configuration
<!-- Auto generated by helm-docs -->
### Values

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| apiserver.affinity | object | `{}` | Affinity for pod assignment |
| apiserver.apiservice.groupPriorityMinimum | int | `2000` | Group priority minimum |
| apiserver.apiservice.versionPriority | int | `100` | Version priority |
| apiserver.args.auditLogMaxAge | int | `0` | Audit log max age |
| apiserver.args.auditLogMaxBackup | int | `0` | Audit log max backup |
| apiserver.args.auditLogPath | string | `"-"` | Audit log path ("-" for stdout) |
| apiserver.args.cronMinScheduleInterval | string | `""` | Cron minimum schedule interval (leave empty for default) |
| apiserver.args.enablePriorityAndFairness | bool | `false` | Enable priority and fairness |
| apiserver.args.etcdServers | string | `""` | etcd server URLs (auto-configured to internal etcd service if empty) |
| apiserver.args.securePort | int | `8443` | Secure port for HTTPS |
| apiserver.command | list | `["/arc-apiserver"]` | Command to run in the container |
| apiserver.enabled | bool | `true` | Enable API Server deployment |
| apiserver.extraArgs | object | `{}` | Additional command-line arguments as key-value pairs |
| apiserver.extraEnv | list | `[]` | Additional environment variables |
| apiserver.fullnameOverride | string | `""` | Override API Server full name |
| apiserver.image.pullPolicy | string | `"IfNotPresent"` | Image pull policy |
| apiserver.image.repository | string | `"ghcr.io/opendefensecloud/arc-apiserver"` | API Server image repository |
| apiserver.image.tag | string | `""` | API Server image tag (defaults to chart appVersion if not set) |
| apiserver.imagePullSecrets | list | `[]` | Image pull secrets for API Server |
| apiserver.livenessProbe | object | `{"httpGet":{"path":"/healthz","port":8443,"scheme":"HTTPS"},"initialDelaySeconds":20,"periodSeconds":20}` | Liveness probe configuration |
| apiserver.nameOverride | string | `""` | Override API Server name |
| apiserver.nodeSelector | object | `{}` | Node selector for pod assignment |
| apiserver.podAnnotations | object | `{}` | Pod annotations |
| apiserver.podLabels | object | `{}` | Pod labels |
| apiserver.podSecurityContext | object | `{"runAsNonRoot":true}` | Pod security context |
| apiserver.readinessProbe | object | `{"httpGet":{"path":"/readyz","port":8443,"scheme":"HTTPS"},"initialDelaySeconds":5,"periodSeconds":10}` | Readiness probe configuration |
| apiserver.replicaCount | int | `1` | Number of API Server replicas |
| apiserver.resources | object | `{"limits":{"cpu":"500m","memory":"128Mi"},"requests":{"cpu":"10m","memory":"64Mi"}}` | Resource limits and requests |
| apiserver.securityContext | object | `{"allowPrivilegeEscalation":false,"capabilities":{"drop":["ALL"]}}` | Container security context |
| apiserver.service.annotations | object | `{}` | Service annotations |
| apiserver.service.port | int | `443` | Service port |
| apiserver.service.targetPort | int | `8443` | Service target port |
| apiserver.service.type | string | `"ClusterIP"` | Service type |
| apiserver.serviceAccount.annotations | object | `{}` | Service account annotations |
| apiserver.serviceAccount.create | bool | `true` | Create service account |
| apiserver.serviceAccount.name | string | `""` | Service account name (auto-generated if not set) |
| apiserver.tolerations | list | `[]` | Tolerations for pod assignment |
| certManager.certificate.duration | string | `"2160h"` | Certificate duration |
| certManager.certificate.renewBefore | string | `"720h"` | Renew before duration |
| certManager.enabled | bool | `true` | Enable cert-manager integration (requires cert-manager to be installed) |
| certManager.issuer.acme.email | string | `""` | Email for ACME registration |
| certManager.issuer.acme.enabled | bool | `false` | Enable ACME issuer |
| certManager.issuer.acme.privateKeySecretRef | string | `""` | Private key secret reference |
| certManager.issuer.acme.server | string | `""` | ACME server URL |
| certManager.issuer.ca.enabled | bool | `false` | Enable CA issuer |
| certManager.issuer.ca.secretName | string | `""` | CA secret name |
| certManager.issuer.create | bool | `true` | Create Issuer resource |
| certManager.issuer.kind | string | `"Issuer"` | Issuer kind (Issuer or ClusterIssuer) |
| certManager.issuer.name | string | `""` | Issuer name (auto-generated if empty) |
| certManager.issuer.selfSigned | bool | `true` | Use self-signed issuer |
| commonAnnotations | object | `{}` | Common annotations applied to all resources |
| commonLabels | object | `{}` | Common labels applied to all resources |
| controller.affinity | object | `{}` | Affinity for pod assignment |
| controller.args.enableHTTP2 | bool | `false` | Enable HTTP/2 for metrics server |
| controller.args.healthProbeBindAddress | string | `":8081"` | Health probe bind address |
| controller.args.leaderElect | bool | `false` | Enable leader election (set to true for HA) |
| controller.args.metricsBindAddress | string | `"0"` | Metrics bind address (set to "0" to disable, ":8443" for HTTPS) |
| controller.args.metricsSecure | bool | `true` | Serve metrics securely via HTTPS |
| controller.args.pprofBindAddress | string | `""` | Pprof bind address (empty to disable) |
| controller.command | list | `["/arc-controller-manager"]` | Command to run in the container |
| controller.enabled | bool | `true` | Enable Controller Manager deployment |
| controller.extraArgs | object | `{}` | Additional command-line arguments as key-value pairs |
| controller.extraEnv | list | `[]` | Additional environment variables |
| controller.fullnameOverride | string | `""` | Override Controller Manager full name |
| controller.image.pullPolicy | string | `"IfNotPresent"` | Image pull policy |
| controller.image.repository | string | `"ghcr.io/opendefensecloud/arc-controller-manager"` | Controller Manager image repository |
| controller.image.tag | string | `""` | Controller Manager image tag (defaults to chart appVersion if not set) |
| controller.imagePullSecrets | list | `[]` | Image pull secrets for Controller Manager |
| controller.livenessProbe | object | `{"httpGet":{"path":"/healthz","port":8081},"initialDelaySeconds":15,"periodSeconds":20}` | Liveness probe configuration |
| controller.metrics.certManager.certKey | string | `"tls.key"` | Certificate key file name |
| controller.metrics.certManager.certName | string | `"tls.crt"` | Certificate file name |
| controller.metrics.certManager.certPath | string | `"/tmp/k8s-metrics-server/metrics-certs"` | Path to mount certificates |
| controller.metrics.certManager.enabled | bool | `false` | Enable cert-manager for metrics certificates |
| controller.metrics.enabled | bool | `false` | Enable metrics service |
| controller.metrics.service.annotations | object | `{}` | Metrics service annotations |
| controller.metrics.service.port | int | `8443` | Metrics service port |
| controller.metrics.service.type | string | `"ClusterIP"` | Metrics service type |
| controller.metrics.serviceMonitor.additionalLabels | object | `{}` | Additional labels for ServiceMonitor |
| controller.metrics.serviceMonitor.enabled | bool | `false` | Enable ServiceMonitor |
| controller.metrics.serviceMonitor.interval | string | `"30s"` | Scrape interval |
| controller.metrics.serviceMonitor.scrapeTimeout | string | `"10s"` | Scrape timeout |
| controller.nameOverride | string | `""` | Override Controller Manager name |
| controller.nodeSelector | object | `{}` | Node selector for pod assignment |
| controller.podAnnotations | object | `{}` | Pod annotations |
| controller.podLabels | object | `{}` | Pod labels |
| controller.podSecurityContext | object | `{"runAsNonRoot":true,"seccompProfile":{"type":"RuntimeDefault"}}` | Pod security context |
| controller.readinessProbe | object | `{"httpGet":{"path":"/readyz","port":8081},"initialDelaySeconds":5,"periodSeconds":10}` | Readiness probe configuration |
| controller.replicaCount | int | `1` | Number of Controller Manager replicas |
| controller.resources | object | `{"limits":{"cpu":"300m","memory":"128Mi"},"requests":{"cpu":"100m","memory":"64Mi"}}` | Resource limits and requests |
| controller.securityContext | object | `{"allowPrivilegeEscalation":false,"capabilities":{"drop":["ALL"]}}` | Container security context |
| controller.serviceAccount.annotations | object | `{}` | Service account annotations |
| controller.serviceAccount.create | bool | `true` | Create service account |
| controller.serviceAccount.name | string | `""` | Service account name (auto-generated if not set) |
| controller.tolerations | list | `[]` | Tolerations for pod assignment |
| createNamespace | bool | `false` | Create namespace if it doesn't exist |
| etcd.affinity | object | `{}` | Affinity for pod assignment |
| etcd.args.advertiseClientUrls | string | `"http://localhost:2379"` | Advertise client URLs |
| etcd.args.dataDir | string | `"/etcd-data-dir/default.etcd"` | Data directory |
| etcd.args.listenClientUrls | string | `"http://[::]:2379"` | Listen client URLs |
| etcd.enabled | bool | `true` | Enable etcd deployment |
| etcd.extraArgs | object | `{}` | Additional command-line arguments as key-value pairs |
| etcd.extraEnv | list | `[]` | Additional environment variables |
| etcd.image.pullPolicy | string | `"IfNotPresent"` | Image pull policy |
| etcd.image.repository | string | `"quay.io/coreos/etcd"` | etcd image repository |
| etcd.image.tag | string | `"v3.6.8"` | etcd image tag |
| etcd.imagePullSecrets | list | `[]` | Image pull secrets for etcd |
| etcd.livenessProbe | object | `{"httpGet":{"path":"/health","port":2379},"initialDelaySeconds":15,"periodSeconds":20}` | Liveness probe configuration |
| etcd.nodeSelector | object | `{}` | Node selector for pod assignment |
| etcd.persistence.accessMode | string | `"ReadWriteOnce"` | Access mode |
| etcd.persistence.annotations | object | `{}` | PVC annotations |
| etcd.persistence.enabled | bool | `true` | Enable persistence |
| etcd.persistence.size | string | `"1Gi"` | Storage size |
| etcd.persistence.storageClass | string | `""` | Storage class (uses default if empty) |
| etcd.podAnnotations | object | `{}` | Pod annotations |
| etcd.podLabels | object | `{}` | Pod labels |
| etcd.podSecurityContext | object | `{"seccompProfile":{"type":"RuntimeDefault"}}` | Pod security context |
| etcd.readinessProbe | object | `{"httpGet":{"path":"/health","port":2379},"initialDelaySeconds":5,"periodSeconds":10}` | Readiness probe configuration |
| etcd.replicaCount | int | `1` | Number of etcd replicas (single instance for non-HA) |
| etcd.resources | object | `{"limits":{"cpu":"500m","memory":"128Mi"},"requests":{"cpu":"10m","memory":"64Mi"}}` | Resource limits and requests |
| etcd.securityContext | object | `{"allowPrivilegeEscalation":false,"capabilities":{"drop":["ALL"]}}` | Container security context |
| etcd.service.annotations | object | `{}` | Service annotations |
| etcd.service.port | int | `2379` | Service port |
| etcd.service.type | string | `"ClusterIP"` | Service type |
| etcd.tolerations | list | `[]` | Tolerations for pod assignment |
| fullnameOverride | string | `""` | Override the full name of the chart |
| global | object | `{"imagePullSecrets":[],"storageClass":""}` | Global settings that can be shared with subcharts |
| global.imagePullSecrets | list | `[]` | Global image pull secrets |
| global.storageClass | string | `""` | Global storage class for persistent volumes |
| nameOverride | string | `""` | Override the name of the chart |
| namespaceOverride | string | `""` | Override the namespace to install into |
| rbac.additionalAPIServerRules | list | `[]` | Additional ClusterRole rules for apiserver |
| rbac.additionalControllerRules | list | `[]` | Additional ClusterRole rules for controller |
| rbac.create | bool | `true` | Create RBAC resources |
<!-- End Auto generated by helm-docs -->

## Contributing

Contributions are welcome! Please see the [Contributing Guide](https://github.com/opendefensecloud/artifact-conduit/blob/main/docs/CONTRIBUTING.md).

## License

Apache-2.0

## Resources

- [Documentation](https://arc.opendefense.cloud/)
- [GitHub Repository](https://github.com/opendefensecloud/artifact-conduit)
- [Issue Tracker](https://github.com/opendefensecloud/artifact-conduit/issues)
- [Examples](https://github.com/opendefensecloud/artifact-conduit/tree/main/examples)
