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
In addition to the fields already present on an `Order` resource, we add:

- `.spec.defaults.cron`: A cron expression that defines when the order should be executed.
- `.spec.artifacts.[].cron`: A cron expression that defines when the single artifact should be executed. This overrides the default cron expression for the entire `Order` resource.

### Extend existing Kind `Order`

Instead of creating a new kind, we can extend the existing `Order` kind to support scheduled orders. We add:

- `.spec.defaults.cron`: A cron expression that defines when the order should be executed.
- `.spec.artifacts.[].cron`: A cron expression that defines when the single artifact should be executed. This overrides the default cron expression for the entire `Order` resource.

Operators of ARC need to adapt their workflow templates and `.spec.artifacts.[].spec` to allow their workflows according to their needs if they want to leverage these new functionality.

Example Order:

```yaml
apiVersion: arc.opendefense.cloud/v1alpha1
kind: Order
metadata:
  name: example-helm-order
spec:
  defaults:
    cron: */5 * * * * # every 5 minutes
    srcRef:
      name: docker-hub
    dstRef:
      name: internal-registry
  artifacts:
    - type: oci
      spec:
        image: library/alpine
        tag: ">=3.18"
        override: myteam/test-img
        overrideTag: "{{tag}}-dev" # overrides origin tag
    - type: helm
      cron: "0 1 * * *" # every day at midnight, overrides default cron
      spec:
        repo: jetstack/charts
        chart: cert-manager
        version: ">=v1.19.1"
    - type: helm
      cron: "" # no cron, overrides default cron
      srcRef:
        name: helm-examples
      spec:
        repo: myexamples
        chart: hello-world
        version: ">=v0.1.0<=v1.0"
```

## Decision Outcome

- We will extend the existing `Order` kind with a new field, `.spec.defaults.cron`, which defines when the order should be executed.
- We will also add a new field, `.spec.artifacts.[].cron`, which allows for overriding the default cron expression for individual artifacts within an order.
- Operators of ARC will need to adapt their workflow templates and `.spec.artifacts.[].spec` accordingly if they want to leverage these new functionality.
