# Endpoint status

Manifests that drive an `Endpoint` into each status the controller can report,
so you can see the feature work end to end on a local cluster.

## Run it

```console
$ kubectl apply -f examples/endpoint-status/endpoints.yaml
$ kubectl get endpoints.arc.opendefense.cloud -n endpoint-status-demo
```

Note the fully-qualified resource name. Plain `kubectl get endpoints` resolves
to **core/v1 Endpoints**, which is a different resource entirely.

Give the controller a few seconds. Each probe is bounded by a 5s timeout, so a
result can take up to that long to appear, an unreachable host that neither
answers nor refuses is what actually consumes the budget. Most cases resolve far
faster: `dns-failure` uses a `.invalid` name, which RFC 6761 reserves and
resolvers answer with an immediate NXDOMAIN, and `target-denied` is refused at
dial time without any network call at all.

To force a re-probe:

```console
$ for e in $(kubectl get endpoints.arc.opendefense.cloud -n endpoint-status-demo       -o name); do
    kubectl annotate -n endpoint-status-demo "$e"       arc.opendefense.cloud/force-at="$(date +%s)" --overwrite
  done
```

## What you should see

| Endpoint | Ready | Blocking condition | What it demonstrates |
| --- | --- | --- | --- |
| `ready-anonymous` | `True` | — | Reachable, credentials not applicable |
| `secret-missing` | `False` | `Validated=False/SecretNotFound` | Config problems are reported, not retried into a backoff |
| `type-unknown` | `False` | `Validated=False/UnknownType` | The type must be claimed by an ArtifactType |
| `dns-failure` | `False` | `Reachable=False/DNSFailure` | Transport errors are classified, not lumped together |
| `target-denied` | `False` | `Reachable=False/TargetDenied` | **SSRF guard.** `0.0.0.0:8081` is the manager's own health port |
| `scheme-unsupported` | `Unknown` | `Reachable=Unknown/UnsupportedScheme` | **Not `True`.** ARC never opened a socket, so it does not claim the endpoint is usable |
| `auth-rejected` | `False` | `Authenticated=False/Unauthorized` | The one-hop Bearer token exchange, with credentials the realm rejects |
| `auth-unverifiable` | `True` | — | `Authenticated=Unknown/UnsupportedAuthScheme`: ARC cannot check SigV4 keys, and says so instead of guessing |

`Unknown` means ARC did not check, which is a different claim from the check
failed. `Unknown` on `Authenticated` leaves an endpoint `Ready`; `Unknown`
on `Reachable` does not.

Full detail for any one of them:

```console
$ kubectl get endpoints.arc.opendefense.cloud auth-rejected \
    -n endpoint-status-demo -o yaml | yq '.status'
```

## Re-probing

There is no periodic re-probe, an untouched Endpoint generates no outbound
traffic at all. A probe runs when the spec changes, when the referenced Secret
changes, or on demand via annotation:

```console
$ kubectl annotate endpoints.arc.opendefense.cloud ready-anonymous \
    -n endpoint-status-demo \
    arc.opendefense.cloud/force-at="$(date +%s)" --overwrite
```

To see credentials being re-checked on rotation, edit `bad-creds` and watch
`auth-rejected` re-probe without you touching the Endpoint.

## What `Ready=True` does not promise

The probe runs from the controller-manager's network position and identity.
Workflow pods run under a different ServiceAccount and possibly different egress
rules, so `Ready=True` is strong evidence an Order will succeed, not a guarantee.

## Cleanup

```console
$ kubectl delete -f examples/endpoint-status/endpoints.yaml
```
