# Workflow Parameters

When ARC creates an Argo `Workflow` from an `ArtifactWorkflow`, it automatically generates a set of parameters that your `WorkflowTemplate` can consume. This page explains exactly which parameters are passed and how their names and values are derived.

---

## Parameter Sources

Every workflow receives parameters from three sources:

| Source           | Parameters generated                   | Example                              |
| ---------------- | -------------------------------------- | ------------------------------------ |
| **Endpoints**    | Connection details for src and dst     | `srcType`, `srcRemoteURL`            |
| **Secrets**      | Boolean flags indicating availability  | `srcSecret`, `dstSecret`             |
| **Artifact spec** | All fields from the artifact definition | `specImage`, `specOverride`          |

---

## Endpoint Parameters

These parameters are always present. They come directly from the source and destination `Endpoint` resources referenced in the `Order`:

| Parameter      | Value comes from            | Example value     |
| -------------- | --------------------------- | ----------------- |
| `srcType`      | `Endpoint.spec.type`        | `oci`             |
| `srcRemoteURL` | `Endpoint.spec.remoteURL`   | `https://ghcr.io` |
| `dstType`      | `Endpoint.spec.type`        | `oci`             |
| `dstRemoteURL` | `Endpoint.spec.remoteURL`   | `https://my-registry.example.com` |

---

## Secret Flags

ARC checks whether each endpoint has a `secretRef` defined. The result is passed as a string boolean:

| Parameter   | Value    | Meaning                                        |
| ----------- | -------- | ---------------------------------------------- |
| `srcSecret` | `"true"` | Source endpoint has a secret attached           |
| `srcSecret` | `"false"`| Source endpoint has no secret                   |
| `dstSecret` | `"true"` | Destination endpoint has a secret attached      |
| `dstSecret` | `"false"`| Destination endpoint has no secret              |

These flags let your `WorkflowTemplate` conditionally use credentials:

**Note**: Even when no secret is provided, an `emptyDir` volume is mounted to maintain consistent volume mount paths across all workflow executions, preventing Argo Workflows from failing due to missing volume definitions.

---

## Artifact Spec Parameters

Fields from the artifact's `spec` block in the `Order` are flattened into individual parameters. Each field name is prefixed with `spec` and converted to **camelCase**.

### Naming Rules

| Spec field type   | Naming pattern            | Example input                         | Parameter name   | Parameter value             |
| ----------------- | ------------------------- | ------------------------------------- | ---------------- | --------------------------- |
| Simple field      | `spec` + `FieldName`      | `image: library/alpine:3.18`         | `specImage`      | `library/alpine:3.18`      |
| Nested field      | `spec` + `Parent` + `Child` | `foo.bar: baz`                      | `specFooBar`     | `baz`                       |
| Array element     | `spec` + `FieldName` + index | `tags: [latest, v1.0]`            | `specTags0`, `specTags1` | `latest`, `v1.0`  |

### Full Example

Given this artifact spec in an `Order`:

```json
{
  "image": "library/alpine:3.18",
  "override": "myteam/alpine:3.18-dev",
  "tags": ["latest", "v1.0"]
}
```

ARC generates these parameters:

| Parameter name  | Value                      |
| --------------- | -------------------------- |
| `specImage`     | `library/alpine:3.18`     |
| `specOverride`  | `myteam/alpine:3.18-dev`  |
| `specTags0`     | `latest`                   |
| `specTags1`     | `v1.0`                     |

!!! note

    All parameter values are strings. Arrays are expanded into individually numbered parameters
    rather than passed as a single list.

---

## ArtifactType Parameters

An `ArtifactType` (or `ClusterArtifactType`) can also define default parameters in its `spec.parameters` list. These are merged with the parameters above, with **ArtifactType parameters taking precedence** over Order-level parameters when names conflict.

```yaml
apiVersion: arc.opendefense.cloud/v1alpha1
kind: ClusterArtifactType
metadata:
  name: oci
spec:
  parameters:
    - name: scanSeverity
      value: HIGH,CRITICAL
  # ...
```

In this example, every workflow of type `oci` receives a `scanSeverity` parameter with value `HIGH,CRITICAL`.

---

## Complete Parameter Reference

Putting it all together, a workflow for an OCI artifact with the resources from the examples above would receive:

| Parameter       | Value                      | Source         |
| --------------- | -------------------------- | -------------- |
| `srcType`       | `oci`                      | Endpoint       |
| `srcRemoteURL`  | `https://...`              | Endpoint       |
| `srcSecret`     | `true`                     | Computed       |
| `dstType`       | `oci`                      | Endpoint       |
| `dstRemoteURL`  | `https://...`              | Endpoint       |
| `dstSecret`     | `true`                     | Computed       |
| `specImage`     | `library/alpine:3.18`     | Artifact spec  |
| `specOverride`  | `myteam/alpine:3.18-dev`  | Artifact spec  |
| `scanSeverity`  | `HIGH,CRITICAL`            | ArtifactType   |

---

## Secret Volume Mounts

In addition to parameters, ARC mounts credential secrets as volumes so your workflow steps can read them from the filesystem

See [Understanding Workflows](understanding-workflows.md) for a full walkthrough of how these resources are composed.
