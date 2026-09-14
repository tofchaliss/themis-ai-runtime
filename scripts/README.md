# scripts/

Operational scripts. Read-only diagnostics; none of them change
governance state.

| Script | Purpose |
|---|---|
| `themis-preflight` | Host readiness before a deployment. Run before Phase A. |
| `themis-status` | Governance and deployment state of a checkout. Run any time. |

## Why these exist

Both answer the same question from opposite ends: **what would a green
run actually mean on this host right now?**

Several evidence-producing tests SKIP rather than fail when the host is
not what they expect — git outside the two paths `gitBin()` probes, no
model endpoint, running as root. A skipped proof reports green. The
preflight makes those conditions visible before a green suite is read as
evidence.

`themis-status` reports the governance side: whether any Deployment
Anchor is ACTIVE, what the registries hold, the L6/L7 constitution
hashes, and what a ceiling would do if pinned.

## What they deliberately do not do

**Neither emits a Deployment Anchor.** `themis-status` displays artifact
pins for diagnosis, and stops there. Whether a reviewed anchor-minting
helper should exist is an open owner decision (deployment runbook Step
7); emitting one from a status script would settle that decision by
accident and put a convenience path immediately next to a Governance
act. Copy pins deliberately, or raise the minting question on its own
terms.

Neither writes under `policies/`, proposes anything, or activates
anything. Activation is a human act, by construction.

## Usage

```bash
scripts/themis-preflight [--deploy DIR]

scripts/themis-status [--ceiling FILE] [--state-root DIR] [--full]
```

`themis-preflight` exits non-zero when a check FAILs. `themis-status`
is informational and always exits 0 — it reports state, it does not
grade it.

### Notes

- `--ceiling` checks that the file satisfies `execution.LoadCeiling`
  **and** that its `mirror_root` exists. The loader itself requires only
  that the path be absolute, so a mistyped mirror pins and admits
  cleanly and fails at the first governed task; the script closes that
  gap rather than the loader being changed to guess.
- `--state-root` summarises a record plane by reading manifests. A task
  with no manifest never reached a typed terminal; the startup sweep
  drives it to one on the next `Open`. That is not corruption.
- `--full` adds `go build`, `go vet`, and `gofmt`.

## Related

- `docs/operations/deployment-runbook.md` — the deployment procedure
- `docs/development/deployment-test-plan.md` — how a deployment is proven
- `TESTING.md` — what the suite establishes, and the live-proof gating
