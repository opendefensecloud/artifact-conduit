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

## Configuration

### Global Parameters

| Parameter | Description | Default |
|-----------|-------------|---------|
| `global.imagePullSecrets` | Global image pull secrets | `[]` |
| `global.storageClass` | Global storage class | `""` |
| `namespaceOverride` | Override namespace | `""` |
| `createNamespace` | Create namespace if it doesn't exist | `false` |

### API Server Parameters

| Parameter | Description | Default |
|-----------|-------------|---------|
| `apiserver.enabled` | Enable API Server | `true` |
| `apiserver.replicaCount` | Number of replicas | `1` |
| `apiserver.image.repository` | Image repository | `ghcr.io/opendefensecloud/arc-apiserver` |
| `apiserver.image.tag` | Image tag | Chart appVersion |
| `apiserver.resources.limits.cpu` | CPU limit | `500m` |
| `apiserver.resources.limits.memory` | Memory limit | `128Mi` |
| `apiserver.service.type` | Service type | `ClusterIP` |
| `apiserver.service.port` | Service port | `443` |

### Controller Manager Parameters

| Parameter | Description | Default |
|-----------|-------------|---------|
| `controller.enabled` | Enable Controller Manager | `true` |
| `controller.replicaCount` | Number of replicas | `1` |
| `controller.image.repository` | Image repository | `ghcr.io/opendefensecloud/arc-controller-manager` |
| `controller.image.tag` | Image tag | Chart appVersion |
| `controller.args.leaderElect` | Enable leader election for HA | `false` |
| `controller.metrics.enabled` | Enable metrics endpoint | `false` |
| `controller.metrics.serviceMonitor.enabled` | Create ServiceMonitor for Prometheus | `false` |

### etcd Parameters

| Parameter | Description | Default |
|-----------|-------------|---------|
| `etcd.enabled` | Enable etcd | `true` |
| `etcd.replicaCount` | Number of replicas | `1` |
| `etcd.image.repository` | Image repository | `quay.io/coreos/etcd` |
| `etcd.image.tag` | Image tag | `v3.6.6` |
| `etcd.persistence.enabled` | Enable persistence | `true` |
| `etcd.persistence.size` | Volume size | `1Gi` |
| `etcd.persistence.storageClass` | Storage class | `""` |

### cert-manager Parameters

| Parameter | Description | Default |
|-----------|-------------|---------|
| `certManager.enabled` | Enable cert-manager integration | `true` |
| `certManager.issuer.kind` | Issuer kind (Issuer or ClusterIssuer) | `Issuer` |
| `certManager.issuer.selfSigned` | Use self-signed issuer | `true` |
| `certManager.certificate.duration` | Certificate duration | `2160h` |

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
   kubectl get apiservice v1alpha1.arc.bwi.de
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

## Contributing

Contributions are welcome! Please see the [Contributing Guide](https://github.com/opendefensecloud/artifact-conduit/blob/main/docs/CONTRIBUTING.md).

## License

Apache-2.0

## Resources

- [Documentation](https://arc.opendefense.cloud/)
- [GitHub Repository](https://github.com/opendefensecloud/artifact-conduit)
- [Issue Tracker](https://github.com/opendefensecloud/artifact-conduit/issues)
- [Examples](https://github.com/opendefensecloud/artifact-conduit/tree/main/examples)
