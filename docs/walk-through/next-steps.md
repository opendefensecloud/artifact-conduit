# Next steps

You built an artifact type `blob` from scratch: `Endpoints` for HTTP and S3, a `WorkflowTemplate` that checks every
file before pushing it, an `ArtifactType` that ties both together, and `Orders` that run once or on a schedule.
When you are done, [delete the cluster](setup.md#clean-up).

## Learn from the examples

The [`examples` directory](https://github.com/opendefensecloud/artifact-conduit/tree/main/examples) has more artifact
types. Each one is offered to all namespaces with a `ClusterArtifactType` and a `ClusterWorkflowTemplate`, and comes
with an `Order`:

| Example | What its workflow does |
| ------- | ---------------------- |
| [`blob`](https://github.com/opendefensecloud/artifact-conduit/tree/main/examples/blob) | HTTP and S3 sources like this walk-through, with an optional MD5 check, a virus scan, a storage quota and a volume sized to fit the file |
| [`oci`](https://github.com/opendefensecloud/artifact-conduit/tree/main/examples/oci) | Copies container images with skopeo, resolves tags from a semver range, scans with Trivy, attests the scan results and signs the image with cosign |
| [`helm`](https://github.com/opendefensecloud/artifact-conduit/tree/main/examples/helm) | Pulls a Helm chart from an OCI registry or a Helm repository, scans it with Trivy and pushes it to an OCI registry |
| [`ocm`](https://github.com/opendefensecloud/artifact-conduit/tree/main/examples/ocm) | Transfers an [OCM](https://ocm.software/) component and scans all of its images |

The `ArtifactType` named `blob` of this walk-through shadows the `ClusterArtifactType` of the `blob` example in the
`walk-through` namespace. Try the examples in another namespace.

## Learn more about Argo Workflows

ARC leaves the work to Argo, so the more you know about Argo Workflows, the more you can do with ARC. Topics this
walk-through did not cover:

* [Artifacts](https://argo-workflows.readthedocs.io/en/stable/walk-through/artifacts/) pass files between steps
  without a shared volume.
* [Retries](https://argo-workflows.readthedocs.io/en/stable/retries/) make transfers survive flaky sources.
* [Exit handlers](https://argo-workflows.readthedocs.io/en/stable/walk-through/exit-handlers/) run after a workflow,
  e.g. to send a notification when a transfer failed.
* [DAGs](https://argo-workflows.readthedocs.io/en/stable/walk-through/dag/) describe steps by their dependencies
  instead of their order.

## Run ARC in production

* Offer artifact types to every namespace with a `ClusterArtifactType` and a `ClusterWorkflowTemplate`, see
  [Extending ArtifactTypes](../operator-manual/extending-artifact-types.md).
* Keep the default [minimal schedule interval](../operator-manual/installation.md#cron-minimal-schedule-interval)
  of 10 minutes, or raise it.
* Decide how long to keep finished workflows with [TTL-Based Cleanup](../operator-manual/ttl-based-cleanup.md).
* Watch transfers with the metrics and dashboard from [Observability](../operator-manual/observability.md).
* When a transfer fails, start with [Debugging](../operator-manual/debugging.md) and
  [Force Reconciliation](../operator-manual/force-reconciliation.md).
