# Overview

## Non-production installation¶

If you just want to try out ARC in a non-production environment (including on desktop via minikube/kind/k3d etc) follow the [quick-start guide](../getting-started.md).

## Production installation

### Prerequisites

ARC has the following CNCF projects as dependencies:

- [Argo Workflows](https://argo-workflows.readthedocs.io/en/stable/installation/)
- [cert-manager](https://cert-manager.io/docs/installation/)

Please make sure you have these dependencies installed before you proceed.

### Installation Methods

To install ARC, navigate to the [releases page](https://github.com/opendefensecloud/artifact-conduit/releases) and find the release you wish to use (the latest full release is preferred).

#### Kustomize

You can use Kustomize to patch your preferred configurations on top of the base manifests.

First, specify the version you want to install in an environment variable. Modify the command below:

    ARC_VERSION="main"

Then, copy the commands below to apply the kustomization:

    kubectl apply -k "https://github.com/opendefensecloud/artifact-conduit/examples/deployment?ref=${ARC_VERSION}"

#### Helm

See [Helm installation](./helm.md) for more information.


### Advanced Settings

The ARC's apiserver and controller-manager have some flags to configure behavior.

#### Cron Minimal Schedule Interval

Most operators don't want users to spawn workflows every few seconds, so a conservative default of 10 minutes as minimal schedule interval is set. The operator can change this by setting the following flag: `--cron-min-schedule-interval`.

This can be overridden through the helm chart by specifying `apiserver.args.cronMinScheduleInterval`, e.g.:
```yaml
apiserver:
  args:
    cronMinScheduleInterval: 1h
```
