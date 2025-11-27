# Core Concepts

This page serves as an introduction to the core concepts of Artficat Conduit (ARC).

## The `Order`

The `Order` is the primary user-facing Kubernetes resource for requesting artifact operations.
It declares intent to procure one or more artifacts with shared configuration defaults.

See the [spec documentation](../user-guide/api-reference.md#orderspec) for details on how to define an `Order`.

## The `Endpoint`

The `Endpoint` is a Kubernetes resource that defines the location and configuration for accessing an external service or system.
It can be used for both input and output artifacts, depending on the context in which it's referenced.

See the [spec documentation](../user-guide/api-reference.md#endpointspec) for details on how to define an `Endpoint`.

## The `ClusterArtifactType` and `ArtifactType`

`ClusterArtifactType` resources are cluster-wide, while `ArtifactType` resources are namespaced.
They specify processing rules and references an Argo `WorkflowTemplate` resource for processing artifacts of this type, e.g `oci` or `helm`.
Furthermore they allow specifying additional parameters passed to workflows.

See the [spec documentation](../user-guide/api-reference.md#artifacttypespec) for details on how to define an `ArtifactType`.

## The `ArtifactWorkflow`

`ArtifactWorkflow` resources are hydrated by the ARC Controller Manager from `Order` resources. They reference an Argo [Workflow Template](https://argo-workflows.readthedocs.io/en/stable/workflow-templates/) along with parameters that are passed to it.
These workflow templates are meant to be defined by operators of ARC along with `ClusterArtifactType` or `ArtifactType` resources, e.g. `oci` or `helm`.
They are then used as a template for creating workflows that process artifacts of a certain type.

!!! note
    This resource type is created by the ARC Controller Manager and not to be used directly.

See the [spec documentation](../user-guide/api-reference.md#artifactworkflowspec) for details on their specification.
