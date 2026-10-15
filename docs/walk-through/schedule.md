# Schedule transfers

So far, every `Order` transferred its files once. With a `cron` schedule, ARC transfers an artifact again and again,
which keeps the copy in the destination in sync: if someone changes or deletes it, the next run restores it.
All files are in [`examples/walk-through/04-schedule`](https://github.com/opendefensecloud/artifact-conduit/tree/main/examples/walk-through/04-schedule).

## Order

Nothing but the `Order` changes. Its artifact gets a `cron` with one or more `schedules` in the usual
[cron format](https://en.wikipedia.org/wiki/Cron). `defaults.cron` sets a schedule for all artifacts of an `Order`.

```yaml
{% include "../../examples/walk-through/04-schedule/1-order.yaml" %}
```

```bash
kubectl apply -f examples/walk-through/04-schedule/1-order.yaml
```

!!! note

    ARC rejects this schedule if it was installed without `apiserver.args.cronMinScheduleInterval=1m`, e.g. if you
    skipped the [setup](setup.md#install-arc). Lower the interval with:

    ```bash
    helm upgrade arc oci://ghcr.io/opendefensecloud/charts/arc \
      --version "$(git describe --tags --abbrev=0)" \
      --namespace arc-system \
      --reuse-values \
      --set apiserver.args.cronMinScheduleInterval=1m
    ```

For a scheduled artifact, ARC creates an Argo [`CronWorkflow`](https://argo-workflows.readthedocs.io/en/stable/cron-workflows/)
instead of a `Workflow`, and the `CronWorkflow` creates a `Workflow` for every run. The `WorkflowTemplate` is the same
for both, it can tell them apart with the `cron` parameter. See [Cron-Scheduled Orders](../operator-manual/cron-orders.md)
for all options.

Every run checks the file again before pushing it. If the file changes at the source, its checksum no longer matches
and the run fails. You notice, and can review the new file before you update the `sha256` in the `Order`.

## Watch it run

The `CronWorkflow` exists right away, the first `Workflow` only once the schedule is due:

```bash
kubectl get cronworkflows -l walk-through=schedule
```

Wait for the first run to finish. There is no `Workflow` to wait for before the schedule is due, so wait for the
`ArtifactWorkflow` instead:

```bash
kubectl wait artifactworkflows -l walk-through=schedule --for jsonpath='{.status.phase}'=Succeeded --timeout 5m
```

=== "kubectl"

    ```bash
    kubectl get workflows -l walk-through=schedule
    ```

=== "Argo CLI"

    ```bash
    argo cron list -l walk-through=schedule
    argo list -l walk-through=schedule
    ```

The `Order` shows when the last run was scheduled and how it went, and counts the `succeeded` and `failed` runs. The
counts are updated a little after a run finished, so right after the first run they may still be `0`:

```bash
kubectl get orders.arc.opendefense.cloud scheduled-transfer -o jsonpath='{.status.artifactWorkflows}'
```

Argo keeps the last succeeded and the last failed `Workflow`, older ones are deleted. Unlike other workflows, the
`ArtifactWorkflow` of a scheduled artifact is not cleaned up by `ttlAfterFinished`, it stays as long as the `Order`.

Check that the file arrived in the bucket:

```bash
kubectl run s3-check --rm -it --restart Never --image amazon/aws-cli:2.37.9 \
  --env AWS_ACCESS_KEY_ID=admin --env AWS_SECRET_ACCESS_KEY=adminadmin -- \
  --endpoint-url http://dst.seaweedfs:8333 s3 cp s3://test/walk-through/scheduled.txt -
```

The `Order` transfers the file every two minutes until you delete it:

```bash
kubectl delete orders.arc.opendefense.cloud scheduled-transfer
```

Next: [where to go from here](next-steps.md).
