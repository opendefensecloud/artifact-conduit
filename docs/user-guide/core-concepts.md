# Core Concepts

This page serves as an introduction to the core concepts of Artficat Conduit (ARC).

## The `Order`

The `Order` resource is the primary Custom Resource Definition (CRD) in the ARC (Artifact Conduit) system for declaring high-level artifact transfer operations. An `Order` specifies one or more artifacts to be processed, along with default source and destination endpoints. The `OrderReconciler` decomposes each Order into individual `ArtifactWorkflow` resources, which represent atomic artifact operations that can be executed independently.

The status map is populated by the `OrderReconciler` as `ArtifactWorkflows` are created. Each key is a truncated SHA-256 hash computed from the artifact's type, source endpoint (including generation), destination endpoint (including generation), source secret (including generation), destination secret (including generation), and spec fields, ensuring idempotent workflow generation and change detection.

See the [spec documentation](../user-guide/api-reference.md#orderspec) for details on how to define an `Order`.

## The `Endpoint`

An `Endpoint` defines the configuration for artifact sources and destinations, including authentication credentials and usage constraints.
The `Endpoint` resource is a namespace-scoped configuration object that represents a location where artifacts can be pulled from or pushed to.

See the [spec documentation](../user-guide/api-reference.md#endpointspec) for details on how to define an `Endpoint`.

### Usage Modes

The `usage` field controls how an Endpoint can be used in artifact workflows:

#### PullOnly

The Endpoint can only be used as a source (`srcRef`) in ArtifactWorkflows. Attempts to use it as a destination will be rejected during validation.

**Use Case**: Public or third-party registries where ARC has read-only access.

#### PushOnly

The Endpoint can only be used as a destination (`dstRef`) in ArtifactWorkflows. Attempts to use it as a source will be rejected during validation.

**Use Case**: Internal registries in air-gapped environments where artifacts are pushed but never pulled.

#### All

The Endpoint can be used as both a source and a destination. This is the most flexible mode and is the default if `usage` is not specified.

**Use Case**: Internal registries that serve as intermediate storage or cache layers.

### Status

An `Endpoint` reports what ARC has observed about it in `status.conditions`.

| Condition | Meaning |
| --- | --- |
| `Validated` | The referenced `Secret` exists and the `Endpoint`'s type is accepted by some `ArtifactType` or `ClusterArtifactType` in a position its usage allows. |
| `Reachable` | The target answered. Any HTTP non 404 / 5xx response counts. The point of this condition is that something is listening. |
| `Authenticated` | `True` means the target demanded credentials and accepted the configured ones. ARC never sends credentials on the first request — only once the target's own response challenges for them. |
| `Ready` | A summary of the others, and the column `kubectl get endpoints.arc.opendefense.cloud` prints. |

`Authenticated` is `Unknown` rather than `False` when ARC cannot verify
credentials of that shape — an S3 `Secret` holding `AWS_ACCESS_KEY_ID` and
`AWS_SECRET_ACCESS_KEY`, for example; or the target served the probe request
without ever asking for credentials, so the configured ones were never exercised
(a public registry that ignores credentials entirely is the common case).
`Unknown` does not prevent an `Endpoint` from becoming `Ready`: it records
that ARC did not check, or had nothing to check, not that the check failed.

```console
$ kubectl get endpoints.arc.opendefense.cloud
NAME       CREATED AT   REMOTE URL           USAGE      SECRET       READY   MESSAGE
registry   2d           gcr.io               PullOnly   gcr-creds    True
mirror     5h           https://zot.local    PushOnly   zot-creds    False   credentials rejected (401)
```

#### When ARC probes

The connection test runs when there is a reason to believe the answer changed:
on creation, when `spec` changes, and when the referenced `Secret` changes.
There is no periodic re-probe, so an `Endpoint` nobody touches produces no
outbound traffic.

To re-check on demand, set the force annotation to the current Unix timestamp:

```console
$ kubectl annotate endpoints.arc.opendefense.cloud registry \
    arc.opendefense.cloud/force-at="$(date +%s)" --overwrite
```

What the last probe ran against is recorded in `status.probedGeneration`,
`status.probedSecretVersion` and `status.probedForceAt`, next to the
`status.lastProbeTime` it produced. Those are what the decision above is made
from, so `Reachable` and its provenance always travel together: ARC never
re-probes because it forgot, and a restart or a leader failover does not
re-present every consumer's credentials.

##### Re-probing on a schedule

Reachability is a property of the world, not of the cluster: a target can go
down or come back without anything here to observe, so by default a `Reachable`
condition is only ever as fresh as the last reason to probe. Set
`--endpoint-probe-ttl` to bound that staleness — the controller then re-probes an
`Endpoint` whose result is older than the TTL, and nothing else changes:

```yaml
controller:
  args:
    endpointProbeTTL: 15m
```

Expiry is spread across `Endpoint`s by up to 20% of the TTL, derived from each
one's UID, so a set applied together does not come due together. The default is
`0`, which disables it.

#### What Ready does not promise

The probe runs from ARC's controller-manager, using its network position and
its identity. The workflow pods that carry out an `Order` run under a different
ServiceAccount and may be subject to different egress rules. `Ready=True` is
therefore strong evidence that an `Order` will succeed, not a guarantee of it.

Operators should also note that the probe connects to a URL supplied by the
consumer. ARC refuses loopback and link-local addresses by default — replace that
list with `controller.args.probeDenyCIDRs`, or set it to an empty string to turn
the check off entirely — but a NetworkPolicy on the controller-manager Deployment
is the boundary to rely on.

The probe also reads the `username`/`password` keys of the `Secret` an
`Endpoint` references and sends them as Basic auth to the `remoteURL` the
`Endpoint` names — both chosen by whoever created the `Endpoint`. The first
probe is always sent anonymously, even if the `Secret` contains credentials.
Credentials are only sent if the initial anonymous probe responds with a
status indicating that authentication is required. This is to close the gap
between a public api that responds successfully no matter the credentials,
which can therefore not be checked on validity, and private endpoints that 
require authentication and therefore validate the credentials by necessity.
`create` on `endpoints.arc.opendefense.cloud` must therefore be treated as
equivalent to `get` on `Secrets` in the same namespace: anyone who can
create an `Endpoint` can point it at a server they control and have that 
Secret's credentials delivered to it. RBAC that grants `Endpoint` creation
without also granting Secret read is not a safe boundary. Egress
`NetworkPolicy` on the controller-manager is the control for where those
credentials may travel.

## The `ClusterArtifactType` and `ArtifactType`

The `[Cluster]ArtifactType` resource defines artifact processing capabilities within ARC by:

1. Specifying validation rules for source and destination endpoint types
2. Declaring parameters required by the underlying WorkflowTemplate
3. Referencing an Argo WorkflowTemplate that implements the artifact processing logic

`ClusterArtifactType` resources are cluster-scoped configuration objects. `ArtifactType` resources are namespaced. Both establish the contract between ARC's orchestration layer and Argo Workflows' execution layer. When an ArtifactWorkflow is created with a specific type (e.g., `oci`), ARC queries the corresponding `ArtifactType` to validate endpoints and construct workflow parameters.

See the [spec documentation](../user-guide/api-reference.md#artifacttypespec) for details on how to define an `ArtifactType`.

## The `ArtifactWorkflow`

The ArtifactWorkflow resource represents a single, executable artifact operation within the ARC system. ArtifactWorkflows are the execution layer counterpart to the declarative Order resource - while an Order may specify multiple artifacts to process, each artifact is decomposed into an individual ArtifactWorkflow that can be independently executed by Argo Workflows.

!!! note
    This resource type is created by the ARC Controller Manager and not to be used directly.

### Integration with Argo Workflows

The ArtifactType acts as a bridge between ARC's declarative resource model and Argo Workflows' imperative execution model. When the ArtifactWorkflow reconciler processes a resource, it:

1. Queries the ArtifactType by name (matching `ArtifactWorkflow.spec.type`)
2. Validates source and destination endpoints against `rules.srcTypes` and `rules.dstTypes`
3. Constructs workflow parameters by merging ArtifactType defaults with flattened artifact specifications
4. Creates an Argo Workflow referencing the specified WorkflowTemplate

See the [spec documentation](../user-guide/api-reference.md#artifactworkflowspec) for details on their specification.
