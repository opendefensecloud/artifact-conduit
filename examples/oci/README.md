# OCI Example

## Cosign

The example uses a generated key-pair with password `password`.

> **FOR TESTING ONLY. DO NOT USE IN PRODUCTION!**

```bash
cosign generate-key-pair
kubectl create secret generic cosign-key --from-file=cosign.key --from-file=cosign.pub -oyaml --dry-run > cosign-key.yaml
```
