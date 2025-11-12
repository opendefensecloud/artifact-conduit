# What is Artifact Conduit (ARC)?

Project ARC is an open-source system to bring a variety of artifact types into air-gapped environments.
It acts as a gateway to get artifacts from one zone to another while scans assure that only artifacts that match certain policies are transported.
These scans include malware, CVE, license scans including validation of signatures and the attestation of that process.

## System Capabilities

ARC provides a comprehensive artifact management platform with the following key capabilities:

### Artifact Procurement

- **Multiple Source Types**: OCI registries (container images, Helm charts, OCM packages), Helm repositories, S3-compatible storage, Google Cloud Storage, Azure Blob Storage, HTTP(S) endpoints
- **Flexible Ordering**: One-shot orders for specific versions, watchers using semver for continuous ordering, complete registry mirroring

### Endpoint Management

- **Ownership Tracking**: Define responsible contacts for endpoints and configurations
- **Shared Endpoints**: Global endpoint availability across the ARC instance
- **Lifecycle Management**: Endpoint expiry based on contract terms
- **Policy Enforcement**: Blocklist/allowlist policies for administrators

### Security and Validation

- **Signature Validation**: Verify artifact signatures before transport
- **Malware Scanning**: Plugin system supporting Trivy, Sysdig, ClamAV
- **CVE Scanning**: Security vulnerability detection
- **License Scanning**: License compliance verification
- **Attestation**: Sign and attest to the scanning process

### Transport and Optimization

- **Multi-Destination**: Deliver artifacts to one or more destinations after validation
- **Dry-Run Mode**: Test orders without actual delivery
- **Deduplication**: Avoid redundant transfers and scans unless necessary (e.g., CVE database updates)
- **TTL Management**: Automatic retirement of orders and resources
