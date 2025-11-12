---
status: in decission
date: 2025-11-12
---

# Define an Optimal API for the Project Beginning

## Context and Problem Statement

This ADR is about finding the right API for ARC.

## Proposed Solution

Options were discussed and documented here: <https://app.bwi.conceptboard.com/board/u9c0-4nk5-rrhd-knre-6cfn>

```yaml
apiVersion: arc.bwi.de/v1alpha1
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

```yaml
apiVersion: arc.bwi.de/v1alpha1
kind: Fragment
metadata:
  name: example-order-1 # sha256 for procedural
spec:
  type: oci # artifactType, correcesponds to workflow
  srcRef: # required
    name: lala
  dstRef: #required
    name: other-internal-registry
    namespace: default # optional
  spec:
    image: library/alpine:3.18
    override: myteam/alpine:3.18-dev # default alpine:3.18; support CEL?
```

```yaml
apiVersion: arc.bwi.de/v1alpha1
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

```yaml
apiVersion: arc.bwi.de/v1alpha1
kind: ArtifactTypeDefinition
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
  defaults:
    dstRef: internal-registry
  workflowTemplateRef: # argo.Workflow
```
