## Force Reconciliation Annotation

## Annotation-Based Debugging

Both `Order` and `ArtifactWorkflow` resources support manual reconciliation triggering via the `arc.opendefense.cloud/forceAt` annotation:

```yaml
apiVersion: arc.bwi.de/v1alpha1
kind: Order
metadata:
  name: my-order
  annotations:
    arc.opendefense.cloud/forceAt: "1735689600"  # Unix timestamp
spec:
  # ...
```

## Force Reconcile Logic

Controllers check the annotation and compare it against the last force reconciliation timestamp:
For `Order`, force reconciliation deletes all `ArtifactWorkflows` to trigger fresh creation. For `ArtifactWorkflow`, it deletes the Argo Workflow.

## Reconciliation Timestamps

Both `Order` and `ArtifactWorkflow` resources track reconciliation timestamps in their status:

| Field                    | Type          | Purpose                                       |
| ------------------------ | ------------- | --------------------------------------------- |
| `status.lastReconcileAt` | `metav1.Time` | Timestamp of most recent reconciliation       |
| `status.lastForceAt`     | `metav1.Time` | Timestamp of most recent force reconciliation |
