# Quick Start

To try out ARC, you can install it and run example orders.

## Prerequisites

Before installing ARC, you need a Kubernetes cluster and `kubectl` configured to access it.
For quick testing, you can use a local cluster with [kind](https://kind.sigs.k8s.io/) or similar tools.

ARC has the following CNCF projects as dependencies:

- [Argo Workflows](https://argo-workflows.readthedocs.io/en/stable/installation/) is used as workflow engine by ARC.
- [cert-manager](https://cert-manager.io/docs/installation/) provides the certificates for the extension apiserver and etcd.

Please make sure you have these dependencies installed before you proceed.

!!! note

    These instructions are intended to help you get started quickly. They are not suitable for production. For production installs, please refer to the [installation documentation](./operator-manual/installation.md).

## Install ARC via Helm

First, specify the version you want to install in an environment variable. Modify the command below:

    export ARC_VERSION="vX.X.X"

You can also retrieve the latest version from the GitHub API:

     export ARC_VERSION="$(curl -s https://api.github.com/repos/opendefensecloud/artifact-conduit/releases/latest | jq -r '.tag_name')"

Then, copy the commands below to install ARC with `helm`:

    helm install \
      arc oci://ghcr.io/opendefensecloud/charts/arc \
      --version $ARC_VERSION \
      --namespace arc-system \
      --create-namespace \
      --set fullnameOverride=arc

## Checkout the examples

Keep in mind that multiple resources have to be created to be able to run workflows for your orders, so the best place to start is the [examples](https://github.com/opendefensecloud/artifact-conduit/tree/main/examples).

For an `Order` to complete you need to define `ArtifactType` or `ClusterArtifactType` for your desired artifact types, which in turn reference the workflow templates.
