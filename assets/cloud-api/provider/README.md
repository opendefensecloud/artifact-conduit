# provider assets

These manifests configure a kcp provider workspace to expose ARC APIs (`orders`
and `endpoints`) to consumer workspaces through the kcp
[`APIExport`][apiexport] marketplace. They define the
[`APIResourceSchema`][ars] for each resource, the export, an
`APIExportEndpointSlice` for virtual-workspace discovery, and the RBAC grants
that let consumers [`APIBind`][apibinding] the export.

[`PermissionClaims`][permissionclaims] for `Secrets`, `Namespaces`, `Events`,
and `LogicalClusters` are offered on the export so consumer workspaces can
accept them — the sync agent needs cross-workspace access to operate.

A [`DependencyRule`][dep] prevents deletion of `Endpoints` while `Orders` still
reference them.

The sync agent's own schema and export controllers are intentionally disabled;
schemas and exports are maintained here as plain Git-managed manifests for full
reproducibility.

Subdirectories:

- **`syncagent/`** — Flux-based deployment of the kcp api-syncagent
  (HelmRelease, RBAC, kubeconfig Secret).
- **`ui/`** — ProviderMetadata and ContentConfiguration manifests that surface
  ARC in the platform-mesh marketplace UI.

[apiexport]: https://docs.kcp.io/kcp/latest/reference/apis/apiexport/
[apibinding]: https://docs.kcp.io/kcp/latest/reference/apis/apibinding/
[ars]: https://docs.kcp.io/kcp/latest/reference/apis/apiresourceschema/
[permissionclaims]: https://docs.kcp.io/kcp/latest/reference/apis/apiexport/#permissionclaims
[dep]: https://github.com/opendefensecloud/dependency-controller
