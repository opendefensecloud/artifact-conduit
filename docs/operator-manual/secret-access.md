# Secret Access

A workflow runs the template of an artifact type with the `Secret`s of its `Endpoint`s mounted, against the URLs the `Endpoint`s name. Whoever chooses that template or URL can read the `Secret`s. ARC does not check this at admission time; RBAC is the only boundary.

All references resolve in the namespace of the `Order`, except that an `Order` uses the `ClusterArtifactType` of a name when no `ArtifactType` of that name exists in its namespace. Cluster-scoped types and templates therefore reach the `Secret`s of every namespace that uses them.

## Equivalences

Resources are RBAC resource names in the API group `arc.opendefense.cloud`, unless marked (Argo) or part of the core group (`secrets`, `pods`).

| Permission                                                                                       | Equivalent to                                          | Reason |
| ---                                                                                              | ---                                                    | --- |
| `create`/`update`/`patch` on `endpoints`                                                         | `get` on `secrets`                                     | Chooses the URL the `Secret` is sent to, by workflows and by the [connection probe](../user-guide/core-concepts.md#what-ready-does-not-promise), and can reference any `Secret`. |
| `create`/`update`/`patch` on `artifacttypes`                                                     | `get` on `secrets`                                     | Chooses the template and overrides all parameters, including `srcRemoteURL` and `dstRemoteURL`. Shadows a `ClusterArtifactType` of the same name. |
| `create`/`update`/`patch` on `workflowtemplates` (Argo)                                          | `get` on `secrets`, use of every ServiceAccount        | The template is the code the `Secret`s are mounted into, and it names the ServiceAccount. |
| `create`/`update`/`patch` on `clusterartifacttypes`, `clusterworkflowtemplates` (Argo)           | as above, in every namespace                           | As for their namespaced counterparts. |
| `create`/`update`/`patch` on `artifactworkflows`                                                 | `get` on `secrets`                                     | Names the `Secret`s and the template directly. Only the controller needs it. |
| `create` on `workflows`, `cronworkflows` (Argo), `pods` or workloads                             | `get` on `secrets`                                     | A pod can mount any `Secret` of its namespace. |
| `create` on `pods/exec`, `pods/attach`; `patch` on `pods/ephemeralcontainers`                    | `get` on `secrets`                                     | Workflow pods have `Secret`s mounted. |
| `create`/`update`/`patch` on `orders`                                                            | none, if templates treat `spec*` parameters as data    | `spec*` parameters cannot override `srcRemoteURL`, `dstRemoteURL` or the mounted `Secret`s. Interpolated into a script, a shell command or a `when` expression, they run code chosen by the `Order` creator. |

## Workflow ServiceAccount

Workflow pods also hold the token of the ServiceAccount named in their template. `Endpoint` `Secret`s reach the workflow as mounts, not through this token, so the ServiceAccount needs read access only to `Secret`s that a template step fetches from the API itself, restricted with `resourceNames`. Every process in a workflow pod can use this token, so a template that interpolates `spec*` parameters into a script or shell command gives the `Order` creator every permission of the ServiceAccount.

## Controller-Manager and API Server

The ARC controller-manager can read the `Secret`s of every namespace. Control over its ServiceAccount, its Deployment, the ARC API server or its etcd is equivalent to `get` on all `secrets`, and belongs to the ARC Operator only.
