# Architecture

ARC is implemented as a Kubernetes Extension API Server integrated with the Kubernetes API Aggregation Layer. This architectural approach provides several advantages over Custom Resource Definitions (CRDs), including dedicated storage isolation, custom API implementation flexibility, and reduced risk to the hosting cluster's control plane.

```mermaid
graph TB
    subgraph "User Interface Layer"
        User["User/Operator"]
        Kubectl["kubectl CLI"]
        GitOps["GitOps Tools"]
    end
    
    subgraph "Kubernetes Control Plane"
        K8sAPI["Kubernetes API Server"]
        APIAgg["API Aggregation Layer"]
        
        subgraph "ARC API Server"
            ARCAPI["ARC Extension API Server"]
            ETCDArc["ARC etcd<br/>Isolated Storage"]
        end
    end
    
    subgraph "ARC Controller Manager"
        OrderCtrl["Order Controller<br/>Manages Order resources<br/>Creates ArtifactWorkflows"]
        AWCtrl["ArtifactWorkflow Controller<br/>Manages ArtifactWorkflow resources<br/>Creates Argo Workflows"]
    end
    
    subgraph "Custom Resources (arc.opendefense.cloud/v1alpha1)"
        Order["Order<br/>User-facing orchestration"]
        ArtifactWF["ArtifactWorkflow<br/>Execution unit"]
        Endpoint["Endpoint<br/>Source/Destination config"]
        ArtifactType["ArtifactType<br/>Processing rules"]
    end
    
    subgraph "Workflow Execution Layer"
        ArgoWF["Argo Workflows<br/>Workflow Engine"]
        Kueue["Kueue (Optional)<br/>Job Queuing & Quotas"]
        Workers["Worker Pods<br/>Execute artifact operations"]
    end
    
    subgraph "External Systems"
        SrcReg["Source Systems<br/>OCI Registries, S3,<br/>Helm Repos, HTTP"]
        DstReg["Destination Systems<br/>Private Registries,<br/>Secure Storage"]
    end
    
    User -->|"Creates Orders"| Kubectl
    GitOps -->|"Declarative Config"| Kubectl
    Kubectl -->|"API Requests"| K8sAPI
    
    K8sAPI <-->|"Routes arc.opendefense.cloud"| APIAgg
    APIAgg <-->|"Custom Resources"| ARCAPI
    ARCAPI <-->|"Persists"| ETCDArc
    
    Order -->|"Watched by"| OrderCtrl
    Endpoint -->|"Referenced by"| OrderCtrl
    ArtifactType -->|"Referenced by"| OrderCtrl
    OrderCtrl -->|"Creates/Updates"| ArtifactWF
    
    ArtifactWF -->|"Watched by"| AWCtrl
    AWCtrl -->|"Instantiates"| ArgoWF
    
    ArgoWF -->|"Optionally scheduled by"| Kueue
    ArgoWF -->|"Creates"| Workers
    
    Workers -->|"Pull artifacts from"| SrcReg
    Workers -->|"Push artifacts to"| DstReg
    Workers -->|"Scan, validate, transform"| Workers
```

**Architecture: ARC System Components and Data Flow**

The system follows a layered architecture where users interact through `kubectl` (or GitOps tools), requests flow through the Kubernetes API aggregation layer to the ARC API Server, and the two-controller pattern orchestrates workflow execution: the Order Controller decomposes high-level Orders into ArtifactWorkflows, and the ArtifactWorkflow Controller instantiates Argo Workflows for execution.

**Key Design Decisions:**

- **Extension API Server architecture** provides dedicated storage isolation in a separate etcd instance
- **Two-controller pattern** separates orchestration (Order) from execution (ArtifactWorkflow)
- **Integration with existing CNCF projects** (Argo Workflows, optionally Kueue) rather than building custom execution engine
- **Declarative, Kubernetes-native API** for GitOps compatibility

## Resource Model and Dependencies

```mermaid
graph TB
    subgraph "User-Facing Resources"
        Order["Order<br/>---<br/>spec.defaults<br/>spec.artifacts[]<br/>spec.TTLSecondsAfterCompletion<br/>---<br/>status.artifactWorkflows<br/>status.message"]
    end
    
    subgraph "Configuration Resources"
        Endpoint["Endpoint<br/>---<br/>spec.type<br/>spec.remoteURL<br/>spec.secretRef<br/>spec.usage"]
        
        ArtifactType["ArtifactType /<br/>ClusterArtifactType<br/>---<br/>spec.rules.srcTypes[]<br/>spec.rules.dstTypes[]<br/>spec.workflowTemplateRef<br/>spec.parameters[]"]
        
        Secret["Kubernetes Secret<br/>---<br/>Credentials for<br/>Endpoints"]
    end
    
    subgraph "Execution Resources"
        ArtifactWF["ArtifactWorkflow<br/>---<br/>spec.workflowTemplateRef<br/>spec.parameters[]<br/>spec.srcSecretRef<br/>spec.dstSecretRef<br/>---<br/>status.phase<br/>status.completionTime"]
        
        ArgoWT["Argo WorkflowTemplate /<br/>ClusterWorkflowTemplate<br/>---<br/>Workflow execution logic"]
        
        ArgoWorkflow["Argo Workflow<br/>---<br/>Runtime instance"]
    end
    
    Order -->|"1:N generates"| ArtifactWF
    Order -->|"references (defaults)"| Endpoint
    Order -->|"per artifact"| Endpoint
    Order -->|"artifacts[].type"| ArtifactType
    
    ArtifactType -->|"spec.workflowTemplateRef"| ArgoWT
    ArtifactType -->|"validates srcTypes"| Endpoint
    ArtifactType -->|"validates dstTypes"| Endpoint
    
    Endpoint -->|"spec.secretRef"| Secret
    
    ArtifactWF -->|"instantiates with params"| ArgoWorkflow
    ArtifactWF -->|"references template from"| ArtifactType
    ArtifactWF -->|"srcSecretRef"| Secret
    ArtifactWF -->|"dstSecretRef"| Secret
    
    ArgoWorkflow -->|"executes steps from"| ArgoWT
    ArgoWorkflow -->|"mounts at runtime"| Secret
    
    Order -.->|"tracks generations of"| Endpoint
    Order -.->|"tracks generations of"| Secret
```

**Key Patterns:**

- **1:N relationship** from Order to ArtifactWorkflow enables parallel processing of multiple artifacts
- **Configuration resources** (Endpoint, ArtifactType) are reusable across multiple Orders
- **Generation tracking** (dashed lines) provides automatic reconciliation on configuration changes
- **Secrets are referenced** but not owned, supporting shared credential management

## Controller Reconciliation Flow

```mermaid
sequenceDiagram
    participant User
    participant Order
    participant OrderCtrl as "Order Controller"
    participant AWCache as "ArtifactWorkflow Cache"
    participant AWCtrl as "ArtifactWorkflow Controller"
    participant Argo as "Argo Workflows"
    participant Pods as "Worker Pods"
    
    User->>Order: "Create Order resource"
    Note over Order: "spec.defaults<br/>spec.artifacts[]"
    
    OrderCtrl->>Order: "Watch event triggered"
    OrderCtrl->>Order: "Reconcile Order"
    
    Note over OrderCtrl: "For each artifact in spec.artifacts[]"
    OrderCtrl->>OrderCtrl: "computeDesiredAW:<br/>- Resolve Endpoints<br/>- Validate types vs ArtifactType rules<br/>- Fetch Secrets<br/>- Generate parameters<br/>- Compute SHA hash"
    
    OrderCtrl->>AWCache: "Create ArtifactWorkflow"
    Note over AWCache: "spec.workflowTemplateRef<br/>spec.parameters[]<br/>spec.srcSecretRef<br/>spec.dstSecretRef"
    
    OrderCtrl->>Order: "Update status.artifactWorkflows"
    Note over Order: "Phase: Unknown → Pending"
    
    AWCtrl->>AWCache: "Watch ArtifactWorkflow"
    AWCtrl->>AWCtrl: "hydrateArgoWorkflow:<br/>- Build Workflow spec<br/>- Mount secrets as volumes<br/>- Set parameters"
    
    AWCtrl->>Argo: "Create Argo Workflow"
    AWCtrl->>AWCache: "Update status.phase = Pending"
    
    Argo->>Pods: "Create worker pods"
    Pods->>Pods: "Execute workflow steps:<br/>- Pull from source<br/>- Scan/validate<br/>- Push to destination"
    
    Pods-->>Argo: "Report completion"
    Argo-->>AWCtrl: "Watch event"
    
    AWCtrl->>AWCtrl: "checkArgoWorkflow:<br/>Compare phases"
    
    alt Succeeded
        AWCtrl->>AWCache: "Update status.phase = Succeeded"
        AWCtrl->>AWCache: "Set completionTime"
    else Failed
        AWCtrl->>Pods: "Fetch pod logs"
        AWCtrl->>AWCache: "Update status.phase = Failed<br/>Set status.message with logs"
    end
    
    AWCache-->>OrderCtrl: "Owner reference triggers"
    OrderCtrl->>Order: "Sync AW phase to status"
    
    Note over OrderCtrl: "Check TTLSecondsAfterCompletion"
    OrderCtrl->>AWCache: "Delete ArtifactWorkflow (if TTL expired)"
```

**Status Propagation**: Owner references ensure automatic cleanup and status flows upward: Argo Workflow → ArtifactWorkflow → Order

## Generation Tracking and Idempotency

ARC implements generation tracking to enable automatic reconciliation and idempotent operations:

**Generation Tracking Mechanism:**

| Tracked Resource         | Purpose                                    | Storage Location                         |
| ------------------------ | ------------------------------------------ | ---------------------------------------- |
| **Endpoint generations** | Detect when Endpoint configuration changes | `Order.status.endpointGenerations` (map) |
| **Secret generations**   | Detect when credentials change             | `Order.status.secretGenerations` (map)   |

**Idempotency Benefits:**

- **No duplicate processing**: Same artifact configuration always produces same ArtifactWorkflow name (SHA-256)
- **Automatic updates**: Endpoint/Secret changes trigger new ArtifactWorkflows with different names
- **Race condition prevention**: Generation numbers from Kubernetes API ensure consistent ordering
- **Minimal resource overhead**: Small ArtifactWorkflow resources with externalized configuration

**Example Scenario:**

1. Order created with `srcRef: docker-hub` (generation 1)
2. ArtifactWorkflow created with name `sha256(namespace + artifact + gen1)`
3. Endpoint `docker-hub` updated with new URL (generation 2)
4. Order controller detects generation change
5. New ArtifactWorkflow created with name `sha256(namespace + artifact + gen2)`
6. Old ArtifactWorkflow remains (for historical tracking) or cleaned up by TTL

## Workflow Integration

ARC leverages Argo Workflows for artifact processing execution, minimizing custom code by delegating workflow execution to a proven CNCF project:

**Integration Architecture:**

| Aspect                  | Implementation                                                                  |
| ----------------------- | ------------------------------------------------------------------------------- |
| **Workflow Engine**     | Argo Workflows handles artifact pull, scan, and push operations                 |
| **WorkflowTemplates**   | Define reusable processing logic per artifact type (referenced by ArtifactType) |
| **Decoupling**          | Separates orchestration (Order Controller) from execution (Argo Workflows)      |
| **Scalability**         | Argo's native horizontal scaling handles concurrent processing                  |
| **Flexibility**         | New artifact types via new WorkflowTemplates without modifying core ARC         |
| **Optional Scheduling** | Kueue can be integrated for advanced job queuing, quotas, and fairness          |

**Integration Flow:**

1. `ArtifactType` resource references a WorkflowTemplate name (e.g., `oci-workflow-template`)
2. `ArtifactWorkflowReconciler` creates Argo Workflow instances using that template
3. ArtifactWorkflow parameters are passed to Workflow as `inputs.parameters`
4. Secrets are mounted as volumes in workflow pods (e.g., `/secrets/src`, `/secrets/dst`)
5. Workflow execution is independent of controller lifecycle
6. Status propagates: Argo Workflow → ArtifactWorkflow → Order (via owner references)

**Key Benefits:**

- **No reinventing the wheel**: Leverages Argo's battle-tested workflow engine
- **Focus on domain logic**: ARC focuses on artifact orchestration, not workflow execution
- **Quotas and fairness**: Optional Kueue integration for multi-tenant resource management
- **Declarative workflows**: WorkflowTemplates are Kubernetes resources, GitOps-ready
