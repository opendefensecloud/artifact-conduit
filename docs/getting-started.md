# Quick Start

To try out ARC, you can install it and run example orders.

## Prerequisites

Before installing ARC, you need a Kubernetes cluster and `kubectl` configured to access it.
For quick testing, you can use a local cluster with [kind](https://kind.sigs.k8s.io/) or similar tools.

ARC has the following CNCF projects as dependencies:

- [Argo Workflows](https://argo-workflows.readthedocs.io/en/stable/installation/)
- [cert-manager](https://cert-manager.io/docs/installation/)

Please make sure you have these dependencies installed before you proceed.

!!! note

    These instructions are intended to help you get started quickly. They are not suitable for production. For production installs, please refer to the [installation documentation](./operator-manual/installation.md).

## Install ARC

## Kustomize

First, specify the version you want to install in an environment variable. Modify the command below:

    ARC_VERSION="main"

Then, copy the commands below to apply the kustomization:

    kubectl apply -k "https://github.com/opendefensecloud/artifact-conduit/examples/deployment?ref=${ARC_VERSION}"

## Submit an example order

// TODO
