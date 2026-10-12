# First transfer

In this step you transfer a single file from GitHub (HTTP) into the SeaweedFS bucket from the [setup](setup.md).
All files are in [`examples/walk-through/01-first-transfer`](https://github.com/opendefensecloud/artifact-conduit/tree/main/examples/walk-through/01-first-transfer).

Five resources are involved:

| Resource           | Owned by | Purpose                                                            |
| ------------------ | -------- | ------------------------------------------------------------------ |
| `Secret`           | you      | Credentials for the destination                                    |
| `Endpoint`         | ARC      | Where to pull from and push to                                     |
| `WorkflowTemplate` | Argo     | **How** to transfer, the actual work                               |
| `ArtifactType`     | ARC      | Connects an artifact type to a `WorkflowTemplate` and allowed endpoints |
| `Order`            | ARC      | **What** to transfer                                               |

## Secret

The destination needs credentials. The keys of a Secret are up to you, the `WorkflowTemplate` is what reads them.

```yaml
{% include "../../examples/walk-through/01-first-transfer/1-secret.yaml" %}
```

    kubectl apply -f examples/walk-through/01-first-transfer/1-secret.yaml

## Endpoints

An `Endpoint` describes a remote location. Its `type` is free-form and is used to decide which artifact types may use
it. `usage` restricts it to pulling, pushing or both, and `secretRef` points to the credentials.

```yaml
{% include "../../examples/walk-through/01-first-transfer/2-endpoints.yaml" %}
```

    kubectl apply -f examples/walk-through/01-first-transfer/2-endpoints.yaml

## WorkflowTemplate

This is where the work happens. It is a plain Argo
[`WorkflowTemplate`](https://argo-workflows.readthedocs.io/en/stable/workflow-templates/) with a single
[script template](https://argo-workflows.readthedocs.io/en/stable/walk-through/scripts-and-results/) that downloads
the file with `curl` and streams it into the bucket with the AWS CLI.

```yaml
{% include "../../examples/walk-through/01-first-transfer/3-workflow-template.yaml" %}
```

    kubectl apply -f examples/walk-through/01-first-transfer/3-workflow-template.yaml

ARC provides two things to every workflow it creates:

* **Parameters**, accessible as `{{workflow.parameters.<name>}}` (see
  [workflow variables](https://argo-workflows.readthedocs.io/en/stable/variables/)). They describe both endpoints
  and the artifact. Every field of the artifact's `spec` in the `Order` becomes a parameter prefixed with `spec`, so
  `key` turns into `specKey`. See [Workflow Parameters](../operator-manual/workflow-parameters.md) for all the rules.
* **Volumes** named `src-secret-vol` and `dst-secret-vol` containing the endpoint secrets. If an `Endpoint` has no
  secret, an empty volume is used instead. The template decides where to mount them, here the destination
  credentials end up in `/secrets/dst`.

Parameters never contain secrets. Use `srcSecret` and `dstSecret` to find out whether credentials were provided.

!!! warning "Supply chain security"

    This workflow copies the file into the destination without looking at it. Never do this in production: anything
    you transfer, you also distribute. Verify artifacts before pushing them, e.g. by checking a checksum, scanning for
    viruses or creating an SBOM. The [next step](verify.md) adds a checksum and a virus scan.

## ArtifactType

The `ArtifactType` named `blob` connects the `WorkflowTemplate` with orders. Its `rules` define which endpoint types
are allowed as source and destination. An `Order` for a `blob` from e.g. an `oci` endpoint is still accepted by the
API, but ARC creates no workflow for it and reports the reason as a warning event, visible with
`kubectl describe orders.arc.opendefense.cloud <name>`.

```yaml
{% include "../../examples/walk-through/01-first-transfer/4-artifact-type.yaml" %}
```

By default, ARC deletes a workflow as soon as it succeeded, failed ones are kept for troubleshooting.
`ttlAfterFinished` keeps succeeded workflows around for a while, so you can inspect them and read their logs. See
[TTL-Based Cleanup](../operator-manual/ttl-based-cleanup.md) for all options.

    kubectl apply -f examples/walk-through/01-first-transfer/4-artifact-type.yaml

!!! note

    An `ArtifactType` only applies to its namespace. It references a namespaced `WorkflowTemplate` by default, or a
    `ClusterWorkflowTemplate` with `workflowTemplateRef.clusterScope: true`. Use a `ClusterArtifactType` to offer an
    artifact type to all namespaces. If both exist with the same name, the namespaced `ArtifactType` wins.

See [Core Concepts](../user-guide/core-concepts.md) for how the ARC resources relate to each other and
[Extending ArtifactTypes](../operator-manual/extending-artifact-types.md) for more on writing your own.

## Order

Finally, the `Order` describes what to transfer. `defaults` sets the endpoints for all artifacts, each artifact can
override them with its own `srcRef` and `dstRef`. The `spec` is free-form and only interpreted by the workflow.

```yaml
{% include "../../examples/walk-through/01-first-transfer/5-order.yaml" %}
```

Apply it last: the `Order` immediately creates a workflow, which fails if its `WorkflowTemplate` does not exist yet.

    kubectl apply -f examples/walk-through/01-first-transfer/5-order.yaml

!!! tip

    The files are numbered, so `kubectl apply -f examples/walk-through/01-first-transfer/` applies all of them in the
    right order. If an `Order` was applied too early, a failed workflow is not retried on its own. Fix the cause and
    [force a reconciliation](../operator-manual/force-reconciliation.md) to recreate it.

## Watch it run

For every artifact in the `Order`, ARC creates an `ArtifactWorkflow`, which in turn creates an Argo `Workflow`:

    kubectl get artifactworkflows
    kubectl get workflows

Wait for the workflow to finish and read its logs:

=== "kubectl"

    ```bash
    kubectl wait workflows --all --for jsonpath='{.status.phase}'=Succeeded --timeout 5m
    kubectl logs -l workflows.argoproj.io/workflow -c main --tail -1
    ```

=== "Argo CLI"

    ```bash
    argo watch @latest
    argo logs @latest
    ```

Once the workflow succeeded, the `Order` reports it in its status:

    kubectl get orders.arc.opendefense.cloud first-transfer -o jsonpath='{.status.artifactWorkflows}'

!!! note

    cert-manager also has a resource called `Order`, so plain `kubectl get order` may return cert-manager's. The
    fully qualified name `orders.arc.opendefense.cloud` always works.

Check that the file arrived in the bucket:

    kubectl run s3-check --rm -it --restart Never --image amazon/aws-cli:2.37.9 \
      --env AWS_ACCESS_KEY_ID=admin --env AWS_SECRET_ACCESS_KEY=adminadmin -- \
      --endpoint-url http://dst.seaweedfs:8333 s3 cp s3://test/walk-through/file.txt -

If something does not work, have a look at [Debugging](../operator-manual/debugging.md).

Next: [verify the file before you push it](verify.md).
