---
status: in decission
date: 2025-11-12
---

# Define an Optimal API for the Project Beginning

## Context and Problem Statement

This ADR is about finding the right API for ARC.

## Proposed Solution

Options were discussed and documented here: <https://app.bwi.conceptboard.com/board/u9c0-4nk5-rrhd-knre-6cfn>

### `Order`
```yaml
apiVersion: arc.opendefense.cloud/v1alpha1
kind: Order
metadata:
  name: example-order
spec:
  defaults:
    srcRef:
      name: docker-hub
      namespace: default # optional
    dstRef:
      name: internal-registry
  artifacts:
    - type: oci # artifactType, correcesponds to workflow
      dstRef:
        name: other-internal-registry
        namespace: default # optional
      spec:
        image: library/alpine:3.18
        override: myteam/alpine:3.18-dev # default alpine:3.18; support CEL?
    - type: oci
      spec:
        image: library/ubuntu:1.0
    - type: helm
      srcRef:
        name: jetstack-helm
      dstRef:
        name: internal-helm-registry
      spec:
        name: cert-manager
        version: "47.11"
        override: helm-charts/cert-manager:47.11
```

### `ArtifactWorkflow`
```yaml
apiVersion: arc.opendefense.cloud/v1alpha1
kind: ArtifactWorkflow
metadata:
  name: example-order-1 # sha256 for procedural
spec:
  workflowTemplateRef:
    name: foo
  srcSecretRef:
    name: lala
  dstSecretRef:
    name: other-internal-registry
  parameters: # input from order used to hydrate parameters for workflow
    - name: srcType
      value: oci
```

### `Endpoint`
```yaml
apiVersion: arc.opendefense.cloud/v1alpha1
kind: Endpoint
metadata:
  name: internal-registry
spec:
  type: oci # Endpoint Type! set valid types on controller manager?
  remoteURL: https://artifactory.example.com/artifactory/ace-oci-local
  secretRef: # STANDARDIZED!
    name: internal-registry-credentials
  usage: PullOnly | PushOnly | All # enum
```

### `ArtifactType` and `ClusterArtifactType`
```yaml
apiVersion: arc.opendefense.cloud/v1alpha1
kind: ArtifactType # or ClusterArtifactType
metadata:
  name: oci
spec:
  rules:
    srcTypes:
      - s3 # Endpoint Types!
      - oci
      - helm
    dstTypes:
      - oci
  workflowTemplateRef: # argo.Workflow
```
