## Context and Problem Statement

This ADR is about finding the right architecture for the ARC suite of services based on the knowledge gained during the internal predecessors of this project.

### Glossary

- `arcctl`: Command line utility to interact with the ARC API.
- `Order`: Represents an order of one or more artifacts. Even ordering artifacts that may exist in the future can be reference here using semver expressions for example.
- `OrderFragment`: Represents a single artifact order which is part of an `Order`.
- `OrderTypeDefinition`: Defines rules and defaults for a specific order type like 'OCI'. References a certain workflow to use for that type.
- `Endpoint`: General term for source or destination. Can be a source or destination for artifacts. Includes optional credentials to access it.
- `WorkflowTemplate`: Argo Workflows, see <https://argo-workflows.readthedocs.io/en/latest/fields/#workflowtemplate>
- `Workflow`: Argo Workflows, see <https://argo-workflows.readthedocs.io/en/latest/fields/#workflow>
- `ARC API Server`: A Kubernetes Extension API Server which handles storage of ARC API
- `Order Controller`: A Kubernetes Controller which reconciles `Orders`, splits up `Order` resources into `OrderFragment` Resources, creates `Workflow` resources for necessary workload

### Design Advantages of utilizing Kubernetes Operators

- **Native Declarative Management**
  - End users manage artifact lifecycle via `kubectl` or GitOps like any other Kubernetes resource.
- **Resiliency**
  - If a worker Pod fails mid-transfer, Kubernetes reschedules the job automatically.
- **Horizontal Scalability**
  - Controller and workers can scale independently.

## Considered Options

### Solution A

#### Technology

- Instead of [CRDs](https://kubernetes.io/docs/concepts/extend-kubernetes/api-extension/custom-resources/), `ARC` uses an [Extension API Server](https://kubernetes.io/docs/tasks/extend-kubernetes/setup-extension-api-server/) via the [Kubernetes API Aggregation Layer](https://kubernetes.io/docs/concepts/extend-kubernetes/api-extension/apiserver-aggregation/) to handle API requests.
- This gives it the possibility to use a dedicated etcd or a even more suitable storage backend for the high amount of resources and status information in case this is necessary.
- While `etcd` still can be used as storage backend, it is one separated from the `etcd` used by the Kubernetes control plane and reduces the risk of bringing the whole cluster into trouble.
- Additional links
  - <https://github.com/kubernetes-sigs/apiserver-runtime>
  - <https://github.com/kubernetes/sample-apiserver/tree/master>

#### Architecture Diagram

![ARC Architecture Diagram](img/0-arc-architecture.drawio.svg "ARC Architecture")

#### Pros

- Usin dedicated etcd does not clutter the infra etcd
- Storage can be changed later on if necessary
- Keep the declarative style of Kubernetes while having complete freedom on the API implementation
- Utilize [Argo Workflows](https://argo-workflows.readthedocs.io) to handle the workflows necessary to process different artifact types without reinventing the wheel
- Optionally use [Kueue](https://kueue.sigs.k8s.io/docs/overview/) to handle quotas and enhanced scheduling

#### Cons

- Building addon apiservers directly on the raw api-machinery libraries requires non-trivial code that must be maintained and rebased as the raw libraries change.
- Steep learning curve when starting the project and steeper learning curve when joining the project.

### Solution B

- [CRDs](https://kubernetes.io/docs/concepts/extend-kubernetes/api-extension/custom-resources/) are used to interact with ARC via the Kubernetes API Server.
- Namespaces are used to separate resources in a multi-tenant environment.
- Several operators come into play which reconcile the different custom resources.
- A sharding mechanism is implemented to be able to scale the workers horizontally and give every worker a given chunk of resources to reconcile.

#### Pros

- CRDs and Kubernetes are relatively simple to implement
- AuthN & AuthZ are included with Kubernetes
- Storage via etcd included with Kubernetes

#### Cons

- Storage may be limited by etcd and can bring the control plane of Kubernetes into trouble if too many resources are present
- The necessity to implement sharding may be hard work and error prone
- Thus said it may not scale in way necessary for such a solution
- The concept of Kubernetes Operators might not fit the problem
  - ARC handles only things outside of Kubernetes
  - Tight coupling between the software and the hosting

### Solution C

- This solution doesn't stick to declarative Kubernetes in any way and provides it's own opinionated API.
- Runs based on Kubernetes without being part of it's API
- AuthN & AuthZ based on OIDC and is independent of Kubernetes cluster hosting it
- Uses message queueing for being able to have multiple workers scaling horizontally

#### Pros

- Free from Kubernetes boundaries
  - Storage
  - AuthN & AuthZ
  - Scaling issues

#### Cons

- AuthN & AuthZ must be implemented
- Own Database must be used

### Solution C.A

- Uses a classical CRUD based approach to store data

### Solution C.B

- Uses Event Sourcing to give full auditability of every change at any given time in the system

## Decision Outcome

Chosen Option: Solution A

> //TODO explain in detail why
