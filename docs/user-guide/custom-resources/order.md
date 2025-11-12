# Order Resource

The `Order` resource is the primary user-facing interface for requesting artifact operations. It allows users to declare multiple artifacts with shared default configurations.

## Structure

```yaml
apiVersion: arc.bwi.de/v1alpha1
kind: Order
metadata:
  name: example-order
  namespace: default
spec:
  defaults:
    srcRef:
      name: docker-hub
    dstRef:
      name: internal-registry
  artifacts:
    - type: oci
      spec:
        image: library/alpine:3.18
    - type: oci
      dstRef:
        name: other-registry
      spec:
        image: library/ubuntu:1.0
status:
  fragments:
    "abc123": {name: "example-order-abc123"}
    "def456": {name: "example-order-def456"}
```
