# consumer assets

These manifests configure a kcp consumer workspace to subscribe to the ARC
`APIExport`. An `APIBinding` references the export and explicitly accepts the
provider's `PermissionClaims` for `Secrets`, `Namespaces`, `Events`, and
`LogicalClusters` so the sync agent can operate across workspaces.
