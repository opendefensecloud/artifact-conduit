---
status: proposed
date: 2026-11-04
---

# Define the Re-transfer Contract

## Context

[Issue #518](https://github.com/opendefensecloud/artifact-conduit/issues/518)
asks which configuration changes should trigger another transfer. The Order
controller uses a hash to identify desired ArtifactWorkflows: a changed hash
creates a new execution; an unchanged hash preserves the recorded execution.
Today, `ArtifactType.spec` is immutable, `ClusterArtifactType.spec` is mutable,
and their generations do not advance automatically. The hash includes the
ineffective artifact type generation but excludes Endpoint and Secret
generations and contents.

## Questions and Arguments

### 1. Should artifact types be immutable?

The answer should apply equally to `ArtifactType` and `ClusterArtifactType`.

#### Yes

##### Pro

- Stable type definitions.
- Changes roll out explicitly through new type names.
- No generation tracking needed.

##### Contra

- New names and Order updates are needed for every spec change.
- Making ClusterArtifactType immutable changes existing behavior.

#### No

##### Pro

- Operators can maintain shared types in place.
- Fewer versioned resources.

##### Contra

- Shared edits can affect many Orders.
- Requires a clear policy for existing executions and consistent generation
  handling.

Immutability covers the type spec, not the contents of its referenced workflow
template.

### 2. If artifact types are mutable, should their generation be part of the hash?

#### Yes

##### Pro

- Updated processing rules, parameters, and template references lead to fresh
  executions.
- Dependent Orders converge on the updated type.

##### Contra

- Any spec change can cause bulk re-transfers, including changes such as TTL
  settings.
- Needs generation increments and dependency watches.

#### No

##### Pro

- Type maintenance does not implicitly rerun existing Orders.
- Operators control execution timing.

##### Contra

- Existing executions retain old configuration until explicitly rerun.
- The same type name can represent different processing behavior over time.

For immutable types, generation should neither advance on metadata updates nor
influence the hash. For mutable types, including generation only works if the
API server increments it on spec changes and the controller reconciles
dependent Orders.

### 3. Should Endpoint and Secret generations be part of the hash?

[Commit 6307cce8](https://github.com/opendefensecloud/artifact-conduit/commit/6307cce8492613cc3cdded1192d6f7e53e5befa3)
removed Endpoint generations, Secret names and generations, and the Endpoint
watch. Its commit message describes the removal but gives no rationale. The
arguments below assess that change rather than attributing intent to its
author.

#### Yes

##### Pro

- Endpoint changes can automatically refresh transfers, for example when the
  destination URL changes.
- Less manual coordination.

##### Contra

- Shared edits can trigger many transfers.
- Endpoint generation also changes for spec edits unrelated to transfer
  contents.
- Dependency watches are required.

#### No

##### Pro

- Supports the commit's behavior: credential rotation and Endpoint maintenance
  do not implicitly rerun Orders.
- Avoids dependency tracking and transfer fan-out.

##### Contra

- A changed destination or source does not automatically receive a fresh
  transfer.
- Operators must explicitly rerun affected Orders.

Endpoint generation currently advances on spec changes, so it can detect
Endpoint edits without hashing their contents. However, Kubernetes Secrets do
not automatically increment generation on data changes; Secret generation would
not reliably detect credential rotation (see the [Secret update
strategy](https://github.com/kubernetes/kubernetes/blob/master/pkg/registry/core/secret/strategy.go)).
Thus, restoring the old hash inputs alone would not provide automatic
re-transfers for all dependency changes, and restoring only the hash without
watches would delay detection until another reconciliation.

### 4. Should Endpoint and Secret contents trigger another transfer?

#### Yes

##### Pro

- Changes to URLs or credentials can automatically retry transfers against the
  new configuration.
- Selected content fields can avoid reacting to irrelevant edits.

##### Contra

- Credential rotation can rerun completed transfers unnecessarily.
- Requires field-selection rules, dependency watches, and careful handling of
  Secret-derived fingerprints.

#### No

##### Pro

- Separates configuration maintenance from execution.
- Avoids bulk transfers on credential rotation and avoids hashing Secret
  contents.

##### Contra

- Operators must request reruns after meaningful changes.
- Completed transfers do not continuously converge on current configuration.

For Endpoints, this question is largely answered by including generation: every
spec edit would already change the hash, making an additional content hash
redundant. It remains a separate question for Secrets because their data
changes are not tracked by generation.

## Proposed Result

1. **Make both artifact type specs immutable.** Roll out changed definitions
   under new names and update the Orders that should adopt them.
2. **Exclude artifact type generation from execution identity.** If mutable
   types are chosen instead, include generation and implement spec-based
   increments and dependency watches consistently for both scopes.
3. **Keep Endpoint and Secret generations out of the hash**, retaining the
   behavior introduced by commit 6307cce8.
4. **Do not automatically re-transfer on Endpoint or Secret content changes.**
   Use the existing [force
   annotation](../../operator-manual/force-reconciliation.md) on affected
   Orders when a rerun is intended.

This favors deliberate execution over automatic propagation of shared
configuration changes, especially credential rotation. The tradeoff is
operational responsibility: changing an Endpoint URL alone will not transfer
previously completed artifacts to the new destination. On acceptance, this
decision supersedes the Endpoint/Secret generation tracking and automatic
re-transfer claims in [ADR 001](001-ARC-Architecture.md).

### Follow-up Work

- Enforce ClusterArtifactType spec immutability consistently with ArtifactType
  and document the migration to versioned names.
- Replace the ineffective artifact type generation hash input with a reserved
  zero to preserve existing hashes and avoid upgrade-triggered re-transfers.
- Add regression coverage for the chosen triggers and align ADR 001 and the
  architecture guide with this contract.
