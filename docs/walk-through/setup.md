# Setup

This step prepares a local cluster for the walk-through. If you already have a cluster with ARC and Argo Workflows,
you can skip to [creating the namespace](#create-a-namespace-for-the-walk-through).

You need `git`, `kubectl`, [`helm`](https://helm.sh/docs/intro/install/) and a Kubernetes cluster, e.g. one created
with [kind](https://kind.sigs.k8s.io/).

All files used in the walk-through live in the ARC repository, so clone it first:

    git clone https://github.com/opendefensecloud/artifact-conduit.git
    cd artifact-conduit

## Create a cluster

Any Kubernetes cluster works. For a local one, use [kind](https://kind.sigs.k8s.io/):

    kind create cluster --name arc-walk-through

## Install Argo Workflows

ARC creates Argo `Workflows`, so Argo Workflows has to be installed. See the
[Argo Workflows installation](https://argo-workflows.readthedocs.io/en/stable/installation/) for details. The short
version, using the Argo Workflows version ARC is built and tested with:

    export ARGO_WORKFLOWS_VERSION="$(awk '/^[ \t]+github.com\/argoproj\/argo-workflows/ {print $2}' go.mod)"
    kubectl create namespace argo
    kubectl apply --server-side -n argo -f "https://github.com/argoproj/argo-workflows/releases/download/${ARGO_WORKFLOWS_VERSION}/install.yaml"
    kubectl wait deployment/workflow-controller -n argo --for condition=Available --timeout 5m

Optionally install the [Argo CLI](https://argo-workflows.readthedocs.io/en/stable/walk-through/argo-cli/). It is not
required, but makes it easier to follow workflows and read their logs. The walk-through shows commands for both the
Argo CLI and plain `kubectl`, pick a tab once and all pages follow.

## Install ARC

ARC needs [cert-manager](https://cert-manager.io/docs/installation/) for the certificates of its API server. Install
the version ARC is tested with:

    export CERTMANAGER_VERSION="$(awk '/^export CERTMANAGER_VERSION/ {print $NF}' Makefile)"
    kubectl apply -f "https://github.com/cert-manager/cert-manager/releases/download/${CERTMANAGER_VERSION}/cert-manager.yaml"
    kubectl wait deployment -n cert-manager --all --for condition=Available --timeout 5m

Then install ARC with the release your checkout is based on (see the [Quick Start](../getting-started.md#install-arc-via-helm)
for other ways to pick a version):

    export ARC_VERSION="$(git describe --tags --abbrev=0)"
    helm install arc oci://ghcr.io/opendefensecloud/charts/arc \
      --version "${ARC_VERSION}" \
      --namespace arc-system \
      --create-namespace \
      --set fullnameOverride=arc \
      --set apiserver.args.cronMinScheduleInterval=1m

By default, ARC rejects schedules that run more often than every 10 minutes. The walk-through lowers that to one
minute, so you do not have to wait long when you [schedule transfers](schedule.md). See
[Cron Minimal Schedule Interval](../operator-manual/installation.md#cron-minimal-schedule-interval).

ARC's API is served by an [extension API server](https://kubernetes.io/docs/concepts/extend-kubernetes/api-extension/apiserver-aggregation/).
Wait until it is available, otherwise `kubectl` does not know the ARC resources yet:

    kubectl wait apiservice/v1alpha1.arc.opendefense.cloud --for condition=Available --timeout 5m

## Create a namespace for the walk-through

Everything in the walk-through is namespaced. Create a namespace and make it the default for `kubectl`:

    kubectl create namespace walk-through
    kubectl config set-context --current --namespace walk-through

## Create a ServiceAccount for workflows

Pods of an Argo Workflow run under a ServiceAccount that needs a few permissions, e.g. to report step results back to
Argo. See [Workflow RBAC](https://argo-workflows.readthedocs.io/en/stable/workflow-rbac/) for details.
The workflows in this walk-through use a ServiceAccount named `arc-workflow`:

```yaml
{% include "../../test/fixtures/service-account.yaml" %}
```

    kubectl apply -f test/fixtures/service-account.yaml

## Install an S3 store as destination

The walk-through transfers files into an S3 bucket. [SeaweedFS](https://github.com/seaweedfs/seaweedfs) provides a
small S3-compatible store with a bucket named `test`:

```yaml
{% include "../../test/fixtures/dst-seaweedfs.yaml" %}
```

    kubectl apply -f test/fixtures/dst-seaweedfs.yaml
    kubectl wait deployment/dst -n seaweedfs --for condition=Available --timeout 5m

You are ready for [your first transfer](first-transfer.md).

## Clean up

When you are done with the walk-through, delete the cluster:

    kind delete cluster --name arc-walk-through
