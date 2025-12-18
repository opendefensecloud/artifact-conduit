---
status: draft
date: 2025-12-18
---

# Scheduled Orders Architecture

## Context and Problem Statement

As of now, we only support one-shot `Order` requests. This ADR documents the architecture for scheduled orders.

## Considered Solutions

### New Kind `CronOrder`

To have a clear separation for end-users between one-shot and scheduled orders, we need to introduce a new kind of order: `CronOrder`.
In addition to the fields already present on an `Order` resource, we add all options a cron workflow of argo workflows are possible, see <https://argo-workflows.readthedocs.io/en/latest/cron-workflows/#cronworkflow-options>.

### Extend existing Kind `Order`

Instead of creating a new kind, we can extend the existing `Order` kind to support scheduled orders. We add:

- `.spec.defaults.cron`: All options a cron workflow of argo workflows are possible, see <https://argo-workflows.readthedocs.io/en/latest/cron-workflows/#cronworkflow-options>.
- `.spec.artifacts.[].cron`: Same as in defaults, but only overrides the defaults.

Operators of ARC need to adapt their workflow templates and `.spec.artifacts.[].spec` to allow their workflows according to their needs if they want to leverage these new functionality.

Example Order:

```yaml
apiVersion: arc.opendefense.cloud/v1alpha1
kind: Order
metadata:
  name: example-helm-order
spec:
  defaults:
    srcRef:
      name: docker-hub
    dstRef:
      name: internal-registry
    cron: # used to instantiate cron workflow, (see https://argo-workflows.readthedocs.io/en/latest/cron-workflows/#cron-workflows)
      schedules:
        - "0 0 * * *" # every day at 00:00
        - "0 12 * * *" # every day at 12:00
      timezone: "Europe/Berlin" # IANA Timezone (see https://en.wikipedia.org/wiki/List_of_tz_database_time_zones)
      concurrencyPolicy: "Allow" # What to do if multiple Workflows are scheduled at the same time. Allow: allow all, Replace: remove all old before scheduling new, Forbid: do not allow any new while there are old
      startingDeadlineSeconds: 0 # Seconds after the last scheduled time during which a missed Workflow will still be run.
  artifacts:
    - type: oci
      spec:
        image: library/alpine
        tag: ">=3.18"
        override: myteam/test-img
        overrideTag: "{{tag}}-dev" # overrides origin tag
    - type: helm
      cron: {} # disable default for this artifact
      spec:
        repo: jetstack/charts
        chart: cert-manager
        version: ">=v1.19.1"
    - type: helm
      cron:
        schedules:
          - "0 3 * * *" # every day at 03:00, overrides the default schedules
      srcRef:
        name: helm-examples
      spec:
        repo: myexamples
        chart: hello-world
        version: ">=v0.1.0<=v1.0"
```

## Decision Outcome

- We will extend the existing `Order` kind with a new field, `.spec.defaults.cron`.
- All options a cron workflow of argo workflows are possible, see <https://argo-workflows.readthedocs.io/en/latest/cron-workflows/#cronworkflow-options>.