# Contributors Guide

## Contributing

The ARC project uses Github to manage reviews of pull requests.
<https://devenv.sh> is used to provide reproducible developer environments.

## Overview

ARC uses standard Kubernetes code generation tools to automatically generate:

- **Client-go libraries**: Type-safe Go clients for programmatic access to ARC resources
- **OpenAPI specifications**: Machine-readable API schemas for validation and documentation
- **Kubernetes manifests**: CRD definitions and RBAC configurations

Code generation is triggered via Makefile targets and uses tools from the Kubernetes ecosystem, including controller-gen, openapi-gen, and k8s.io/code-generator.

The pipeline processes API type definitions through multiple generators. The [hack/update_codegen.sh](/hack/update-codegen.sh) script orchestrates helper generation and client library creation, while controller-gen produces Kubernetes manifests based on code markers.
