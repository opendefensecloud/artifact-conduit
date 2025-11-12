# What is Artifact Conduit (ARC)?

ARC (Artifact Conduit) is an open-source system that acts as a gateway for procuring various artifact types and transferring them across security zones while ensuring policy compliance through automated scanning and validation. The system addresses the challenge of bringing external artifacts—container images, Helm charts, software packages, and other resources—into restricted environments where direct internet access is prohibited.

**Primary Goals:**

- **Artifact Procurement**: Pull artifacts from diverse sources including OCI registries, Helm repositories, S3-compatible storage, and HTTP endpoints
- **Security Validation**: Perform malware scanning, CVE analysis, license verification, and signature validation before artifact transfer
- **Policy Enforcement**: Ensure only artifacts meeting defined security and compliance policies cross security boundaries
- **Declarative Management**: Leverage Kubernetes-native declarative configuration for artifact lifecycle management
- **Auditability**: Provide attestation and traceability of all artifact processing operations

**Out of Scope:** ARC does not replace existing registry solutions or artifact repositories. It functions as an orchestration layer that coordinates artifact transfer and validation between existing infrastructure components.

## System Architecture

ARC is implemented as a Kubernetes Extension API Server integrated with the Kubernetes API Aggregation Layer. This architectural approach provides several advantages over Custom Resource Definitions (CRDs), including dedicated storage isolation, custom API implementation flexibility, and reduced risk to the hosting cluster's control plane.

```mermaid
graph TB
    subgraph "User Layer"
        Users["Users/Operators"]
        arcctl["arcctl CLI<br/>cmd/arcctl/main.go"]
    end
    
    subgraph "Kubernetes Control Plane"
        K8sAPI["Kubernetes API Server<br/>API Aggregation Layer"]
    end
    
    subgraph "ARC Control Plane"
        ARCAPI["ARC API Server<br/>pkg/apiserver/<br/>Extension API Server"]
        etcdStore["Dedicated etcd<br/>Isolated Storage"]
        OrderCtrl["Order Controller<br/>pkg/controller/<br/>Reconciliation Logic"]
    end
    
    subgraph "API Resources (api/arc.bwi.de/v1alpha1)"
        Order["Order<br/>High-level request"]
        Fragment["Fragment<br/>Single artifact op"]
        Endpoint["Endpoint<br/>Source/Destination"]
        ATD["ArtifactTypeDefinition<br/>Type rules"]
    end
    
    subgraph "Execution Layer"
        ArgoWorkflows["Argo Workflows<br/>Workflow Execution"]
        WorkflowTemplates["WorkflowTemplate<br/>Processing Logic"]
    end
    
    subgraph "External Systems"
        OCIReg["OCI Registries"]
        HelmRepo["Helm Repositories"]
        S3Storage["S3 Compatible Storage"]
        Scanners["Security Scanners<br/>Trivy, ClamAV"]
    end
    
    Users -->|"CLI commands"| arcctl
    arcctl -->|"API requests"| K8sAPI
    
    K8sAPI -->|"forwards arc.bwi.de/*"| ARCAPI
    ARCAPI -->|"stores/retrieves"| etcdStore
    
    OrderCtrl -->|"watches"| etcdStore
    OrderCtrl -->|"creates"| Fragment
    OrderCtrl -->|"creates"| ArgoWorkflows
    
    Order -->|"references"| Endpoint
    Order -->|"uses"| ATD
    Fragment -->|"references"| Endpoint
    
    ATD -->|"specifies"| WorkflowTemplates
    ArgoWorkflows -->|"instantiates"| WorkflowTemplates
    
    ArgoWorkflows -->|"pulls from"| OCIReg
    ArgoWorkflows -->|"pulls from"| HelmRepo
    ArgoWorkflows -->|"pulls from"| S3Storage
    ArgoWorkflows -->|"scans with"| Scanners
    ArgoWorkflows -->|"pushes to"| OCIReg
```

**Architecture: ARC System Components and Data Flow**

The system follows a layered architecture where users interact through the `arcctl` CLI tool, requests flow through the Kubernetes API aggregation layer to the ARC API Server, and the Order Controller orchestrates workflow execution by decomposing high-level Orders into executable Fragments.

## Core Concepts

ARC introduces four primary custom resource types under the `arc.bwi.de/v1alpha1` API group:

| Resource                   | Purpose                                                                                    | Scope                            |
| -------------------------- | ------------------------------------------------------------------------------------------ | -------------------------------- |
| **Order**                  | Declares intent to procure one or more artifacts with shared configuration defaults        | User-facing, high-level          |
| **Fragment**               | Represents a single artifact operation decomposed from an Order                            | System-generated, execution unit |
| **Endpoint**               | Defines a source or destination location with credentials                                  | Configuration, reusable          |
| **ArtifactTypeDefinition** | Specifies processing rules and workflow templates for artifact types (e.g., "oci", "helm") | Configuration, system-wide       |

```mermaid
graph LR
    subgraph "Declarative Layer"
        Order["Order<br/>(User Input)"]
        OrderSpec["spec:<br/>- defaults<br/>- artifacts[]"]
    end
    
    subgraph "Generated Layer"
        Fragment1["Fragment-1"]
        Fragment2["Fragment-2"]
        FragmentN["Fragment-N"]
    end
    
    subgraph "Configuration"
        SrcEndpoint["Endpoint<br/>(Source)"]
        DstEndpoint["Endpoint<br/>(Destination)"]
        ATD["ArtifactTypeDefinition<br/>(e.g., 'oci')"]
        Secret["Secret<br/>(Credentials)"]
    end
    
    subgraph "Execution"
        WorkflowTemplate["WorkflowTemplate<br/>(Argo)"]
        Workflow1["Workflow Instance"]
        Workflow2["Workflow Instance"]
    end
    
    Order -->|"contains"| OrderSpec
    OrderSpec -->|"generates"| Fragment1
    OrderSpec -->|"generates"| Fragment2
    OrderSpec -->|"generates"| FragmentN
    
    Fragment1 -->|"srcRef"| SrcEndpoint
    Fragment1 -->|"dstRef"| DstEndpoint
    Fragment1 -->|"type"| ATD
    Fragment2 -->|"references"| SrcEndpoint
    Fragment2 -->|"references"| DstEndpoint
    
    SrcEndpoint -->|"credentialRef"| Secret
    DstEndpoint -->|"credentialRef"| Secret
    
    ATD -->|"workflowTemplateRef"| WorkflowTemplate
    Fragment1 -.->|"triggers"| Workflow1
    Fragment2 -.->|"triggers"| Workflow2
    WorkflowTemplate -.->|"instantiates"| Workflow1
    WorkflowTemplate -.->|"instantiates"| Workflow2
```

## Key Components

### ARC API Server

The ARC API Server is a Kubernetes Extension API Server implemented using the `k8s.io/apiserver` library. Key characteristics:

- **Implementation Path**: `pkg/apiserver/`
- **Storage Backend**: Dedicated etcd instance (isolated from Kubernetes control plane etcd)
- **Registry Pattern**: Uses `pkg/registry/` for custom storage strategies per resource type
- **API Group**: `arc.bwi.de` with version `v1alpha1`
- **Integration**: Registered with Kubernetes API Aggregation Layer to handle requests to `arc.bwi.de/*` paths

The dedicated etcd approach provides:

- Isolation from the hosting cluster's control plane
- Flexibility to change storage backends if needed
- Protection against resource volume impacting cluster stability

### Order Controller

The Order Controller implements the reconciliation loop for Order resources:

- **Implementation Path**: `pkg/controller/`
- **Framework**: Uses `sigs.k8s.io/controller-runtime` (version 0.22.4)
- **Reconciliation Logic**:

    1. Watch for Order create/update/delete events
    2. Validate endpoint references exist
    3. Apply defaults from Order.spec.defaults
    4. Generate Fragment resources (one per artifact entry)
    5. Lookup ArtifactTypeDefinition for each fragment's type
    6. Create Argo Workflow instances with appropriate WorkflowTemplate references
    7. Update Order status based on Fragment and Workflow statuses
    8. Handle finalizers for cleanup operations

```mermaid
sequenceDiagram
    participant User
    participant arcctl
    participant K8sAPI as "Kubernetes API"
    participant ARCAPI as "ARC API Server<br/>pkg/apiserver/"
    participant etcd as "Dedicated etcd"
    participant OrderCtrl as "Order Controller<br/>pkg/controller/"
    participant ArgoCtrl as "Argo Controller"
    participant Workflow as "Workflow Pod"
    participant Registry as "External Registry"
    
    User->>arcctl: "arcctl oci pull alpine:3.18"
    arcctl->>K8sAPI: "Create Order CR"
    K8sAPI->>ARCAPI: "Forward to arc.bwi.de"
    ARCAPI->>etcd: "Store Order"
    
    etcd-->>OrderCtrl: "Watch notification"
    OrderCtrl->>OrderCtrl: "Reconcile()"
    OrderCtrl->>ARCAPI: "Create Fragment CRs"
    ARCAPI->>etcd: "Store Fragments"
    
    OrderCtrl->>ARCAPI: "Get ArtifactTypeDefinition"
    ARCAPI-->>OrderCtrl: "ATD with workflowTemplateRef"
    
    OrderCtrl->>K8sAPI: "Create Workflow CR"
    K8sAPI-->>ArgoCtrl: "Workflow created"
    
    ArgoCtrl->>Workflow: "Start workflow pods"
    Workflow->>ARCAPI: "Read Endpoint configs"
    ARCAPI-->>Workflow: "Endpoint details + secrets"
    
    Workflow->>Registry: "Pull artifact"
    Registry-->>Workflow: "Artifact data"
    
    Workflow->>Workflow: "Security scan"
    Workflow->>Registry: "Push to destination"
    
    Workflow-->>ArgoCtrl: "Workflow complete"
    ArgoCtrl->>K8sAPI: "Update Workflow status"
    
    OrderCtrl->>ARCAPI: "Update Order status"
    ARCAPI->>etcd: "Store status update"
```
