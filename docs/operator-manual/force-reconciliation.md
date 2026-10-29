## Force Reconciliation Annotation

## Annotation-Based Debugging

Both `Order` and `ArtifactWorkflow` resources support manual reconciliation triggering via the `arc.opendefense.cloud/force-at` annotation:

```yaml
apiVersion: arc.opendefense.cloud/v1alpha1
kind: Order
metadata:
  name: my-order
  annotations:
    arc.opendefense.cloud/force-at: "1735689600"  # Unix timestamp
spec:
  # ...
```

Set the annotation as follows:

```bash
kubectl annotate --overwrite orders.arc.opendefense.cloud/<order-name> \
"arc.opendefense.cloud/force-at"="$(date +%s)"
```

## Force Reconcile Logic

Controllers force a reconciliation when the annotation is later than `.status.lastForceAt`, then record the annotation value there, so each value forces exactly once.
The force happens as soon as the controller sees the new value: `force-at` is not a schedule, a future timestamp does not delay the reconciliation until that time.
Values that cannot be recorded (past year 9999, e.g. milliseconds instead of seconds) are logged and ignored.
For `Order`, force reconciliation deletes all `ArtifactWorkflows` to trigger fresh creation. For `ArtifactWorkflow`, it deletes the Argo Workflow.

## Reconciliation Timestamps

Both `Order` and `ArtifactWorkflow` resources track reconciliation timestamps in their status:

| Field                     | Type          | Purpose                                       |
| ------------------------- | ------------- | --------------------------------------------- |
| `.status.lastReconcileAt` | `metav1.Time` | Timestamp of most recent reconciliation       |
| `.status.lastForceAt`     | `metav1.Time` | Annotation value last honoured                |
