# Upgrading

## Helm

See [Helm installation](./helm.md#upgrading) for more information.

## Version notes

### To v0.4.0 — nested spec parameters follow the naming rules

Earlier versions did not capitalize the keys of nested objects when flattening an artifact's
`spec` into workflow parameters, so `spec.foo.bar` became `specfooBar` instead of `specFooBar`
as [Workflow parameters](./workflow-parameters.md#naming-rules) documents. This is fixed.
Any reference to the lowercase names must switch to the documented ones.

### To v0.3.0 — every `Endpoint` is probed once

The `EndpointReconciler` used to keep in memory what each probe had run against, and now
records it in `status`. Endpoints carrying a result from an older version have no such record,
so each is probed once on the first reconcile after the upgrade, and that probe re-sends the
Endpoint's credentials to its target. It happens once per `Endpoint`, not per reconcile.

Expect a burst proportional to how many Endpoints the cluster has, and a corresponding burst of
requests at their registries. Nothing has to be done about it; the probes stop as soon as each
record is written. [When ARC probes](../user-guide/core-concepts.md#when-arc-probes) lists what
causes a probe from then on.
