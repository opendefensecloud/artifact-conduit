---
status: accepted
date: 2025-11-11
---

# Evaluate Future ARC Architecture Based on Predecessors

## Context and Problem Statement

This ADR is about finding the right architecture for the ARC suite of services based on the knowledge gained during the internal predecessors of this project.

### Glossary

- `arcctl`: Command line utility to interact with the ARC API.
- `Order`: Represents an order of one or more artifacts. Even ordering artifacts that may exist in the future can be reference here using semver expressions for example.
- `OrderArtifactWorkflow`: Represents a single artifact order which is part of an `Order`.
- `OrderTypeDefinition`: Defines rules and defaults for a specific order type like 'OCI'. References a certain workflow to use for that type.
- `Endpoint`: General term for source or destination. Can be a source or destination for artifacts. Includes optional credentials to access it.
- `WorkflowTemplate`: Argo Workflows, see <https://argo-workflows.readthedocs.io/en/latest/fields/#workflowtemplate>
- `Workflow`: Argo Workflows, see <https://argo-workflows.readthedocs.io/en/latest/fields/#workflow>
- `ARC API Server`: A Kubernetes Extension API Server which handles storage of ARC API
- `Order Controller`: A Kubernetes Controller which reconciles `Orders`, splits up `Order` resources into `OrderArtifactWorkflow` Resources, creates `Workflow` resources for necessary workload
- `ArtifactType`: Specifies the processing rules and workflow templates for artifact types (e.g. `oci`, `helm`).

## Considered Options

### Classic Kubernetes Operators

- [CRDs](https://kubernetes.io/docs/concepts/extend-kubernetes/api-extension/custom-resources/) are used to interact with ARC via the Kubernetes API Server.
- Several operators come into play which reconcile the different custom resources.
- A sharding mechanism is implemented to be able to scale the workers horizontally and give every worker a given chunk of resources to reconcile.

#### Pros

- CRDs and Kubernetes are relatively simple to implement

#### Cons

- Storage may be limited by `etcd` and can bring the control plane of Kubernetes into trouble if too many resources are present
- The necessity to implement sharding may be hard work and error prone
- Thus said it may not scale in way necessary for such a solution

### Extension API Server and CNCF Landscape Tooling

![Considered Options](img/0-arc-architecture-arcdb.drawio.svg "Considered Options")

#### Flavor A

Flavor uses several controllers to handle different parts of the API.
Orders are reconciled and converted to hydrated Orders which contain the source and destination information along with the artifact to process.
This information are published to some message queue which is subscribed by workers working on that queue.
Optionally [KEDA](https://keda.sh/) is used to scale, handle quotas and fairness.
The database to be used for the API Server is etcd.

#### Flavor B

Same as [Flavor A](#flavor-a) except the database for the API Server is something like Postgres instead of etcd.
Additionally the option is considered to let the controller access the Postgres directly to reconcile without using the Kubernetes API Server.

#### Flavor C

Same as [Flavor B](#flavor-b) except that no message queue is used but the job queue is stored directly in the Postgres database.

#### Flavor D

Same as [Flavor A](#flavor-a) except that no message queue is used.
The Order controller creates hydrated orders directly in `etcd` as readonly resource which is then consumed by workers directly.
This approach needs some kind of sharding mechanism to have the workers to know which shard of resources they need to handle.

#### Flavor E

This flavor is the most compelling due to the reduced amount of code that is necessary to bring this solution to live.
etcd is used as storage for the API server.
Argo Workflows is used to build workflows which do the steps necessary to process one artifact.
Kueue can be used to bring fairness, scaling, quotas into play which workflows.
The order controller creates "jobs" which are actually Argo Workflows.

This option is described in detail in the following document.

#### Technology

- Instead of [CRDs](https://kubernetes.io/docs/concepts/extend-kubernetes/api-extension/custom-resources/), `ARC` uses an [Extension API Server](https://kubernetes.io/docs/tasks/extend-kubernetes/setup-extension-api-server/) via the [Kubernetes API Aggregation Layer](https://kubernetes.io/docs/concepts/extend-kubernetes/api-extension/apiserver-aggregation/) to handle API requests.
- This gives it the possibility to use a dedicated `etcd` or a even more suitable storage backend for the high amount of resources and status information in case this is necessary.
- While `etcd` still can be used as storage backend, it is one separated from the `etcd` used by the Kubernetes control plane and reduces the risk of bringing the whole cluster into trouble.
- Additional links
  - <https://github.com/kubernetes-sigs/apiserver-runtime>
  - <https://github.com/kubernetes/sample-apiserver/tree/master>
- Utilize [Argo Workflows](https://argo-workflows.readthedocs.io) to handle the workflows necessary to process different artifact types
- Optionally use [Kueue](https://kueue.sigs.k8s.io/docs/overview/) to handle quotas and enhanced scheduling
- Namespaces are used to separate resources in a multi-tenant environment.

#### Architecture Diagram

**Overview Diagram**

![ARC Architecture Diagram](img/0-arc-architecture.drawio.svg "ARC Architecture")

**API Concept Diagram**

```mermaid
---
config:
  layout: elk
---
flowchart LR
 subgraph subGraph0["Declarative Layer"]
        Order["Order (User Input)"]
        Spec["spec:<br>- defaults<br>- artifacts[]"]
  end
 subgraph subGraph1["Generated Layer"]
        ArtifactWorkflow1["ArtifactWorkflow-1"]
        ArtifactWorkflow2["ArtifactWorkflow-2"]
        ArtifactWorkflowN["ArtifactWorkflow-N"]
  end
 subgraph Configuration["Configuration"]
        ArtifactTypeDef@{ label: "ArtifactType (e.g., 'oci')" }
        EndpointSrc["Endpoint (Source)"]
        EndpointDst["Endpoint (Destination)"]
        Secret["Secret (Credentials)"]
  end
 subgraph Execution["Execution"]
        WorkflowTemplate["WorkflowTemplate (Argo)"]
        WorkflowInstance1["Workflow Instance"]
        WorkflowInstance2["Workflow Instance"]
  end
    Order -- contains --> Spec
    Spec -- generates --> ArtifactWorkflow1 & ArtifactWorkflow2 & ArtifactWorkflowN
    ArtifactWorkflow1 -- type --> ArtifactTypeDef
    ArtifactWorkflow2 -- type --> ArtifactTypeDef
    ArtifactWorkflow1 -- srcRef --> EndpointSrc
    ArtifactWorkflow2 -- srcRef --> EndpointSrc
    ArtifactWorkflow1 -- dstRef --> EndpointDst
    ArtifactWorkflow2 -- dstRef --> EndpointDst
    ArtifactWorkflow1 -- references --> EndpointSrc & EndpointDst
    ArtifactWorkflow2 -- references --> EndpointSrc & EndpointDst
    EndpointSrc -- credentialRef --> Secret
    EndpointDst -- credentialRef --> Secret
    ArtifactTypeDef -- workflowTemplateRef --> WorkflowTemplate
    ArtifactWorkflow1 -- triggers --> WorkflowTemplate
    ArtifactWorkflow2 -- triggers --> WorkflowTemplate
    WorkflowTemplate -- instantiates --> WorkflowInstance1 & WorkflowInstance2
    ArtifactTypeDef@{ shape: rect}

```

The solution shows the ARC API Server which handles storage for the custom resources / API of ARC.
`etcd` is used as storage solution.
`Order Controller` is a classic Kubernetes controller implementation which reconciles `Orders` and `Endpoints`.
An `Order` contains the information what artifacts should be processed.
An `Endpoint` contains the information about a source or destination for artifacts.
The `Order Controller` creates `ArtifactWorkflow` resources which are single artifacts decomposed from an `Order`.
An `ArtifactType` specifies the processing rules and workflow templates for artifact types (e.g. `oci`, `helm`).

#### Pros

- Using dedicated `etcd` does not clutter the infra etcd
- Storage can be changed later on if necessary
- Keep the declarative style of Kubernetes while having complete freedom on the API implementation
- Argo Workflows allows us to focus on the domain of the product without reinventing the wheel
- Qotas and Fairness easy without writing code via Kueue

#### Cons

- Building addon apiservers directly on the raw api-machinery libraries requires non-trivial code that must be maintained and rebased as the raw libraries change.
- Steep learning curve when starting the project and steeper learning curve when joining the project.

## Decision Outcome

Chosen Option: Solution A.

Because the solution is the one that provides the most flexibility while the necessity to write own code for many parts is minimized.
The flexibility comes from utilizing the CNCF projects Argo Workflows and Kueue for building the workflow engine.
The project itself can focus on the order process and the handling of endpoints.
