# Getting started

To try out ARC, you can install it and run example orders.

## Prerequisites

Before installing ARC, you need a Kubernetes cluster and `kubectl` configured to access it.
For quick testing, you can use a local cluster with [kind](https://kind.sigs.k8s.io/) or similar tools.

ARC relies heavily on Argo Workflows, so you **must** install it first, see [Argo Workflows documentation](https://argo-workflows.readthedocs.io/en/stable/installation/).

!!! note

    These instructions are intended to help you get started quickly. They are not suitable for production. For production installs, please refer to the [installation documentation](./operator-manual/installation.md).

## Install ARC

## Kustomize

First, specify the version you want to install in an environment variable. Modify the command below:

```bash
ARC_VERSION="main"
```

Then, copy the commands below to apply the kustomization:

```bash
kubectl apply -k "https://github.com/opendefensecloud/artifact-conduit/examples/deployment?ref=${ARC_VERSION}"
```

## Submit an example order

// TODO
