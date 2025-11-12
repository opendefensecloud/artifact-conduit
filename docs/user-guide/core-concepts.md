# Core Concepts

ARC is built as a Kubernetes-native system that extends the Kubernetes API through the API Aggregation Layer. Unlike traditional Kubernetes operators that use Custom Resource Definitions (CRDs), ARC implements an Extension API Server with dedicated storage. This architectural decision provides isolation from the cluster's control plane etcd and enables future flexibility in storage backend selection.

The system consists of three primary runtime components:

- **ARC API Server** - Extends the Kubernetes API to handle arc.bwi.de/v1alpha1 resources
- **Order Controller** - Reconciles high-level artifact requests into executable units
- **Argo Workflows** - Executes artifact processing workflows defined by `ArtifactTypeDefinition` resources

## Resource Type Hierarchy

| **Resource Type**      | **API Group**       | **Purpose**                                               | **Lifecycle Owner** |
| ---------------------- | ------------------- | --------------------------------------------------------- | ------------------- |
| Order                  | arc.bwi.de/v1alpha1 | High-level artifact request containing multiple artifacts | User/GitOps         |
| Fragment               | arc.bwi.de/v1alpha1 | Single artifact operation unit, child of Order            | Order Controller    |
| Endpoint               | arc.bwi.de/v1alpha1 | Configuration for artifact source/destination             | User/Admin          |
| ArtifactTypeDefinition | arc.bwi.de/v1alpha1 | Type rules and workflow reference for artifact processing | Admin               |

The resource hierarchy establishes a declarative model where users create `Order` resources that reference shared `Endpoint` configurations. The Order Controller decomposes each Order into individual `Fragment` resources based on the `ArtifactTypeDefinition` rules, which then trigger corresponding Argo Workflows.

## Order Reconciliation Sequence

The `OrderController` watches for `Order` resources and implements the reconciliation logic. When an `Order` is created, the controller:

1. Retrieves the `Order` and referenced `Endpoint` resources
1. Looks up the appropriate `ArtifactTypeDefinition` for each artifact
1. Creates individual `Fragment` resources
1. Creates Argo Workflow instances based on the `workflowTemplateRef`
