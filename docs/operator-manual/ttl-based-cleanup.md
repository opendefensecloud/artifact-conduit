
# TTL-Based Cleanup

The `Order.spec.TTLSecondsAfterCompletion` field controls automatic cleanup of completed `ArtifactWorkflow` resources. Understanding this mechanism is important for status visibility.

## Cleanup Behavior

| TTL Value     | Behavior                                                                        |
| ------------- | ------------------------------------------------------------------------------- |
| Not set (nil) | ArtifactWorkflows deleted immediately after reaching `Succeeded` phase          |
| `0`           | ArtifactWorkflows deleted immediately after reaching `Succeeded` phase          |
| `> 0`         | ArtifactWorkflows retained for specified seconds after completion, then deleted |

**Note:** Only `Succeeded` workflows are subject to TTL cleanup. `Failed` and `Error` workflows are retained indefinitely for troubleshooting unless explicitly deleted.
