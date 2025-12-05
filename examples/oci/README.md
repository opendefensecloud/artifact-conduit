# OCI Example

## Cosign

The example uses a generated key-pair with password `password`.

> **FOR TESTING ONLY. DO NOT USE IN PRODUCTION!**

```bash
cosign generate-key-pair k8s://default/cosign-key
k get secret -n default cosign-key -oyaml > cosign-key.yaml
k delete secret -n default cosign-key
```
