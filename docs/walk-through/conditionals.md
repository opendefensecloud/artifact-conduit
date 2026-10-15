# Conditionals

So far, every file came from GitHub over HTTP. In this step the workflow also pulls from S3 and picks the right way
to pull for each artifact with an Argo [conditional](https://argo-workflows.readthedocs.io/en/stable/walk-through/conditionals/).
One `Order` transfers one file from each source.
All files are in [`examples/walk-through/03-conditionals`](https://github.com/opendefensecloud/artifact-conduit/tree/main/examples/walk-through/03-conditionals).

## Endpoint

The S3 source is the SeaweedFS from the setup again, this time as a `PullOnly` endpoint. Unlike `github-raw`, it
needs credentials and reuses the `Secret` of the first transfer.

```yaml
{% include "../../examples/walk-through/03-conditionals/1-endpoint.yaml" %}
```

```bash
kubectl apply -f examples/walk-through/03-conditionals/1-endpoint.yaml
```

## WorkflowTemplate

The new `blob-transfer` has two pull steps instead of one, `pull-http` and `pull-s3`. Both are in the same group of
steps, but each has a `when` condition on the `srcType` parameter, so only one runs and the other is skipped. The
checks and `push` stay as they are, they do not care where the file came from.

```yaml
{% include "../../examples/walk-through/03-conditionals/2-workflow-template.yaml" %}
```

```bash
kubectl apply -f examples/walk-through/03-conditionals/2-workflow-template.yaml
```

`pull-s3` shows the second kind of conditional: whether the source has credentials is decided inside the script,
with the `srcSecret` parameter. `src-secret-vol` is always there, but it is an empty volume if the `Endpoint` has no
`secretRef`. See [Conditionals](https://argo-workflows.readthedocs.io/en/stable/walk-through/conditionals/) for the
syntax of `when` and [Enhanced Depends Logic](https://argo-workflows.readthedocs.io/en/stable/enhanced-depends-logic/)
for conditions on the results of other steps.

## ArtifactType

The rules of the `ArtifactType` only allowed `http` sources. Add `s3`, otherwise ARC creates no workflow for an
artifact from `seaweedfs-source`:

```yaml
{% include "../../examples/walk-through/03-conditionals/3-artifact-type.yaml" %}
```

The `spec` of an `ArtifactType` is immutable, `kubectl apply` refuses to change it. Replace it instead, `--force`
deletes the `ArtifactType` and creates it again. Workflows that already exist are not affected.

```bash
kubectl replace --force -f examples/walk-through/03-conditionals/3-artifact-type.yaml
```

## Order

The `Order` has two artifacts. The first one uses the default source `github-raw`. The second one sets its own
`srcRef` and pulls the file that the previous step pushed. Since `seaweedfs-source` has `type: s3`, its workflow
gets `srcType=s3` and runs `pull-s3`.

```yaml
{% include "../../examples/walk-through/03-conditionals/4-order.yaml" %}
```

```bash
kubectl apply -f examples/walk-through/03-conditionals/4-order.yaml
```

## Watch it run

ARC creates one `ArtifactWorkflow` and one Argo `Workflow` per artifact, so there are two this time:

```bash
kubectl get workflows -l walk-through=conditionals
```

=== "kubectl"

    ```bash
    kubectl wait workflows -l walk-through=conditionals --for jsonpath='{.status.phase}'=Succeeded --timeout 5m
    kubectl get workflows -l walk-through=conditionals \
      -o jsonpath='{range .items[*]}{range .status.nodes.*}{.name}{": "}{.phase}{"\n"}{end}{end}' | sort
    ```

=== "Argo CLI"

    ```bash
    argo list -l walk-through=conditionals
    argo get <name>
    ```

    `argo get` shows the steps of a workflow as a tree, skipped steps included.

In each workflow, one of `pull-http` and `pull-s3` is `Skipped`. Besides the steps, the list contains the workflow
itself and the groups of steps, `[0]` to `[3]`. Check that both files arrived in the bucket:

```bash
kubectl run s3-check --rm -it --restart Never --image amazon/aws-cli:2.37.9 \
  --env AWS_ACCESS_KEY_ID=admin --env AWS_SECRET_ACCESS_KEY=adminadmin -- \
  --endpoint-url http://dst.seaweedfs:8333 s3 ls s3://test/walk-through/
```

Next: [schedule transfers](schedule.md).
