# Verify before you push

The [first transfer](first-transfer.md) copied the file without looking at it. Anything you transfer, you also
distribute, so in this step the workflow checks the file before pushing it: it compares a checksum and scans it for
viruses. If a check fails, the file never reaches the destination.
All files are in [`examples/walk-through/02-verify`](https://github.com/opendefensecloud/artifact-conduit/tree/main/examples/walk-through/02-verify).

This step builds on the resources of the first transfer. The `Secret`, the `Endpoints` and the `ArtifactType` stay
as they are, only the `WorkflowTemplate` changes and a new `Order` asks for the checksum.

## WorkflowTemplate

The new `blob-transfer` replaces the one from the first transfer. Instead of a single script, it runs four
[steps](https://argo-workflows.readthedocs.io/en/stable/walk-through/steps/) one after another: `pull`, `checksum`,
`virus-scan` and `push`.

```yaml
{% include "../../examples/walk-through/02-verify/1-workflow-template.yaml" %}
```

```bash
kubectl apply -f examples/walk-through/02-verify/1-workflow-template.yaml
```

A few things are new compared to the first transfer:

* **Steps.** Each step runs in a pod of its own, and a step only starts when the previous one succeeded. That is what
  makes the checks a gate: a failing check stops the workflow before `push`. Steps in the same list item run in
  parallel, see [Steps](https://argo-workflows.readthedocs.io/en/stable/walk-through/steps/).
* **A shared volume.** Since the steps run in different pods, they pass the file through a volume. Argo creates the
  `PersistentVolumeClaim` from `volumeClaimTemplates` for every workflow and deletes it afterwards, see
  [Volumes](https://argo-workflows.readthedocs.io/en/stable/walk-through/volumes/). The checks mount it read-only.
  [Artifacts](https://argo-workflows.readthedocs.io/en/stable/walk-through/artifacts/) are the alternative, but need
  an artifact repository.
* **A checksum from the `Order`.** The expected SHA-256 is part of the artifact `spec` and reaches the workflow as
  `specSha256`. The parameter has no default, so from now on every `blob` order without a `sha256` fails. This
  includes `first-transfer`, if you [force a reconciliation](../operator-manual/force-reconciliation.md) of it.
* **A virus scan.** The [ClamAV](https://docs.clamav.net/) image ships with a virus database, `clamscan` exits with an
  error if it finds something.

More checks fit in as further steps, e.g. creating an SBOM or verifying a signature. The
[Blob example](https://github.com/opendefensecloud/artifact-conduit/tree/main/examples/blob) also sizes the volume
from the file instead of using a fixed size.

## Order

The `Order` adds the checksum to the artifact `spec`. In practice, it comes from the publisher of the file, e.g. a
`SHA256SUMS` file next to a release, not from whoever writes the `Order`. Quote it: YAML reads a checksum that
consists only of digits as a number.

```yaml
{% include "../../examples/walk-through/02-verify/2-order.yaml" %}
```

```bash
kubectl apply -f examples/walk-through/02-verify/2-order.yaml
```

The `Order` has a label. ARC copies the labels of an `Order` to its `ArtifactWorkflows` and Argo `Workflows`, which
makes it easy to find them among the workflows of the first transfer.

## Watch it run

Wait for the workflow to finish and read its logs:

=== "kubectl"

    ```bash
    export WORKFLOW="$(kubectl get workflows -l walk-through=verify -o jsonpath='{.items[0].metadata.name}')"
    kubectl wait workflow "${WORKFLOW}" --for jsonpath='{.status.phase}'=Succeeded --timeout 5m
    kubectl logs -l workflows.argoproj.io/workflow="${WORKFLOW}" -c main --prefix --tail -1
    ```

    The logs of all steps are printed, but not necessarily in the order the steps ran. As in the first transfer,
    lines with `argo=true` come from the Argo executor.

=== "Argo CLI"

    ```bash
    argo watch @latest
    argo logs @latest
    ```

The virus scan takes a moment, `clamscan` loads its database first. Check that the file arrived in the bucket:

```bash
kubectl run s3-check --rm -it --restart Never --image amazon/aws-cli:2.37.9 \
  --env AWS_ACCESS_KEY_ID=admin --env AWS_SECRET_ACCESS_KEY=adminadmin -- \
  --endpoint-url http://dst.seaweedfs:8333 s3 cp s3://test/walk-through/verified.txt -
```

## Watch it fail

Now order the same file with a checksum that does not match, as if the file had been tampered with:

```bash
kubectl apply -f - <<EOF
apiVersion: arc.opendefense.cloud/v1alpha1
kind: Order
metadata:
  name: tampered-transfer
  labels:
    walk-through: tampered
spec:
  defaults:
    srcRef:
      name: github-raw
    dstRef:
      name: seaweedfs
  artifacts:
    - type: blob
      spec:
        key: opendefensecloud/artifact-conduit/refs/tags/v0.3.0/examples/blob/file.txt
        override: test/walk-through/tampered.txt
        sha256: "0000000000000000000000000000000000000000000000000000000000000000"
EOF
```

The `checksum` step fails, and so does the workflow:

=== "kubectl"

    ```bash
    export WORKFLOW="$(kubectl get workflows -l walk-through=tampered -o jsonpath='{.items[0].metadata.name}')"
    kubectl wait workflow "${WORKFLOW}" --for jsonpath='{.status.phase}'=Failed --timeout 5m
    kubectl logs -l workflows.argoproj.io/workflow="${WORKFLOW}" -c main --prefix --tail -1
    ```

=== "Argo CLI"

    ```bash
    argo watch @latest
    argo logs @latest
    ```

The `Order` reports the failure as well, including the logs of the failed step:

```bash
kubectl get orders.arc.opendefense.cloud tampered-transfer -o jsonpath='{.status.artifactWorkflows}'
```

`push` never ran, so there is no `tampered.txt` in the bucket. This command fails with a `404`:

```bash
kubectl run s3-check --rm -it --restart Never --image amazon/aws-cli:2.37.9 \
  --env AWS_ACCESS_KEY_ID=admin --env AWS_SECRET_ACCESS_KEY=adminadmin -- \
  --endpoint-url http://dst.seaweedfs:8333 s3 cp s3://test/walk-through/tampered.txt -
```

ARC keeps failed workflows for troubleshooting and does not retry them on its own. Deleting the `Order` also deletes
its workflows:

```bash
kubectl delete orders.arc.opendefense.cloud tampered-transfer
```

Next: [pull from more than one kind of source](conditionals.md).
