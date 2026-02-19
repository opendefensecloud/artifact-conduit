
# TTL-Based Cleanup

The `ArtifactWorkflowTTLSettings` controls automatic cleanup of completed and failed `ArtifactWorkflow` resources. These settings can be configured on both `ClusterArtifactWorkflow` and `ArtifactWorkflow` specs.

## TTL Settings

| Field                      | Description                                           |
| -------------------------- | ----------------------------------------------------- |
| `TTLDurationAfterFinished` | Time to live for succeeded workflows after completion |
| `TTLDurationAfterFailed`   | Time to live for failed/error workflows after failure |

## Cleanup Behavior

### TTLDurationAfterFinished

| TTL Value     | Behavior                                                                         |
| ------------- | -------------------------------------------------------------------------------- |
| Not set (nil) | ArtifactWorkflows deleted immediately after reaching `Succeeded` phase           |
| `0`           | ArtifactWorkflows retained indefinitely                                          |
| `> 0`         | ArtifactWorkflows retained for specified duration after completion, then deleted |

### TTLDurationAfterFailed

| TTL Value     | Behavior                                                                      |
| ------------- | ----------------------------------------------------------------------------- |
| Not set (nil) | ArtifactWorkflows retained indefinitely for troubleshooting                   |
| `0`           | ArtifactWorkflows retained indefinitely                                       |
| `> 0`         | ArtifactWorkflows retained for specified duration after failure, then deleted |

**Note:** Only `Succeeded`, `Failed`, and `Error` workflows are subject to TTL cleanup. Other workflows (Running, Pending) are retained until completion.
