# Failed Workflow Debugging

## Automatic Pod Log Collection

When Argo Workflows fail, the `ArtifactWorkflowReconciler` automatically collects logs from failed pods for debugging:

```mermaid
sequenceDiagram
    participant AWController as ArtifactWorkflowReconciler
    participant ArgoWF as Argo Workflow
    participant K8sAPI as Kubernetes API
    participant AW as ArtifactWorkflow Status
    
    AWController->>ArgoWF: Check workflow status
    ArgoWF-->>AWController: status.phase = Failed
    AWController->>AWController: generateWorkflowStatusMessage()
    
    Note over AWController: Identify failed pods<br/>from workflow.status.nodes
    
    loop For each failed pod
        AWController->>K8sAPI: fetchPodLogs(podName)
        K8sAPI-->>AWController: Last 30 lines of logs
        AWController->>AWController: Append to status.message
    end
    
    AWController->>AW: Update status.message with logs
    AW-->>AWController: Status updated
```

**Key Points**:

- Only the **last 30 lines** are collected (configurable via `TailLines`)
- Logs are retrieved from the `main` container
- Errors during log fetching are logged but don't fail the reconciliation

## Debugging Workflow Failures

When investigating failed workflows:

1. Check `ArtifactWorkflow.status.message` for collected pod logs
2. Inspect `Order.status.message` for validation errors
3. Review Kubernetes events: `kubectl describe order <name>` and `kubectl describe artifactworkflow <name>`
4. Query Argo Workflow status: `kubectl get workflow <name> -o yaml`
5. Check controller logs for detailed reconciliation traces (enable debug logging)
