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
