# Roles

A blueprint of four roles. They may be combined or adapted, but the permissions listed in [Secret Access](secret-access.md) are equivalent to reading `Secret`s and belong only to principals trusted with the `Secret`s of the namespace.

| Role                        | Primary Goal                                                                        | Scope                | Cardinality |
| ---                         | ---                                                                                 | ---                  | ---         |
| ARC Operator                | Install and run ARC; restrict where the controller and workflows send credentials.  | Cluster              | Singleton   |
| Artifact Type Provider      | Define how artifacts are transferred: workflow templates and artifact types.        | Cluster or Namespace | Many        |
| Endpoint Provider           | Provide the `Endpoint`s of a namespace and their credentials.                       | Namespace            | Many        |
| Artifact Transfer Requester | Request transfers between the `Endpoint`s of a namespace.                           | Namespace            | Many        |

## Permissions

| Resource Type                     | Scope     | Artifact Type Provider (cluster-wide) | Artifact Type Provider (namespaced) | Endpoint Provider | Artifact Transfer Requester |
| ---                               | ---       | ---                                   | ---                                 | ---               | ---                         |
| `ClusterArtifactType`             | Cluster   | CRUD                                  | R                                   | R                 | R                           |
| `ClusterWorkflowTemplate` (Argo)  | Cluster   | CRUD                                  | R                                   | -                 | -                           |
| `ArtifactType`                    | Namespace | CRUD (every namespace)                | CRUD                                | R                 | R                           |
| `WorkflowTemplate` (Argo)         | Namespace | CRUD (every namespace)                | CRUD                                | -                 | -                           |
| `Secret`                          | Namespace | -                                     | -                                   | CRUD              | -                           |
| `Endpoint`                        | Namespace | R (every namespace)                   | R                                   | CRUD              | R                           |
| `Order`                           | Namespace | -                                     | -                                   | -                 | CRUD                        |
| `ArtifactWorkflow`                | Namespace | R (every namespace)                   | R                                   | -                 | R                           |
| `Workflow`, `CronWorkflow` (Argo) | Namespace | R (every namespace)                   | R                                   | -                 | R                           |

The ARC Operator holds `cluster-admin`.

## Manifests

Example bindings for the namespace `team-a`. A `RoleBinding` cannot grant cluster-scoped resources, so each namespaced role comes with a read-only `ClusterRole` for them. The Artifact Type Provider `ClusterRole` serves both scopes: bound with a `ClusterRoleBinding` it covers every namespace, bound with a `RoleBinding` only one.

### Artifact Type Provider

```yaml
--8<-- "docs/operator-manual/manifests/arc-artifact-type-provider.yaml"
```

### Artifact Type Provider (cluster-wide)

```yaml
--8<-- "docs/operator-manual/manifests/arc-artifact-type-provider-cluster-wide.yaml"
```

### Endpoint Provider

```yaml
--8<-- "docs/operator-manual/manifests/arc-endpoint-provider.yaml"
```

### Artifact Transfer Requester

```yaml
--8<-- "docs/operator-manual/manifests/arc-artifact-transfer-requester.yaml"
```
