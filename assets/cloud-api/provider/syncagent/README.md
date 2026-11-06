# syncagent assets

These manifests deploy the kcp api-syncagent on the ARC service cluster
(`arc-system`) via Flux CD. A `HelmRelease` pulls the `api-syncagent` chart
from the kcp Helm repository and a `kcp-kubeconfig-secret.yaml` provides the
workspace connection — populate the placeholder values (server, CA, token)
before deploying.

[`PublishedResources`][pr] define the sync behavior for `Orders` and
`Endpoints`, including `CEL`-based mutations that handle field promotion,
cross-resource name rewriting, and status derivation for the UI. Object names
are prefixed with the cluster identity to prevent collisions when multiple
consumers sync into the same namespace.

Supplemental RBAC (`rbac.yaml`) grants the agent per-type access to ARC
resources and secrets on the service cluster beyond what the chart provides.

[pr]: https://github.com/kcp-dev/api-syncagent#publishedresources
