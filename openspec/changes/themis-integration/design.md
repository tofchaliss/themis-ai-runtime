# Design record: Themis integration (D-I-*)

Locked decisions, in the order the owner disposed them. `proposal.md`
holds the questions; this file is the record. PROPOSED until LOCKed.

## D-I-1 — Intake topology (LOCKED 2026-09-25, owner; deployment fact: same host)

> `cmd/themis-intake` is a Themis-owned CLI running on the same
> enterprise VM as the harness. It receives the execution TUPLE (never
> a record path or an ObjectID), resolves it through the locked
> D-T-1/D-T-2 machinery against the local, read-only harness record
> plane (manifest, event stream, object store), renders the evidence
> view, and raises a Decision Proposal over the authenticated
> Governance API. An authenticated human decider accepts or rejects;
> acceptance establishes the Position.

```
Human ─ tuple ─▶ cmd/themis-intake ─ local read-only ─▶ harness record plane
                        │
                        ▼ intake.Resolve → Evidence View
                        │ authenticated Governance API
                        ▼
                 Themis Governance ─▶ Decision Proposal ─▶ authenticated human ─▶ Position
```

Ownership boundary (locked):
- Harness owns execution records, evidence, provenance, verification
  reconstruction.
- `themis-intake` owns translating an already-existing harness
  execution into a Governance Proposal.
- Governance owns the proposal lifecycle and the authenticated decision.
- The human decider owns accept/reject.
- The harness does not initiate a Governance act. Governance does not
  read the harness record plane. Filesystem access to the record is
  confined to the Themis-owned intake CLI.

No second Position mechanism: the harness's output is the evidence
basis for the EXISTING Proposal → authenticated acceptance → Position
flow. The CLI boundary is `tuple → Resolve → Evidence View → Proposal`,
never `file path → read files → manufacture proposal`.

## D-I-2 — Harness module identity (LOCKED 2026-09-25, owner)

> Rename the harness module `github.com/tofchaliss/themis` →
> `github.com/tofchaliss/themis-ai-runtime/src/harness`, as ONE
> mechanical harness-repo commit (104 Go imports, 3 `go.mod`, `go.work`,
> ~10 doc/procedure references). No architecture or anchor movement: the
> module path is not folded into any constitution hash and no governed
> artifact identity depends on it.

Themis consumes the harness as a normal Go dependency:

```
Themis ── require github.com/tofchaliss/themis-ai-runtime/src/harness
                    └── exact pseudo-version pinned to the rsys@6 source commit
```

No `replace` to a sibling checkout (build would depend on filesystem
topology, not a reproducible identity). No source copying (a second
implementation of the record contract). The pinned harness commit is
part of Themis's build/dependency provenance; the harness's deployment
identity stays governed by G1/anchor pinning. Semver tags
(`src/harness/v0.x`) may come later; not required.

**The Go module path identifies the software dependency; the deployment
anchor identifies the governed executable artifact. Never conflated.**

Sequencing (locked): D-I-2 → mechanical rename → T-M3 intake imports
move → harness tests + CI → Themis requires the pinned commit → Themis
intake implementation. The rename lands BEFORE any Themis-repo change.

## D-I-3 — The Themis read door and contract pinning (LOCKED 2026-09-25, owner; amends D-T-9)

> The harness reads LIVE Themis authority through the existing seam.
> `ThemisSeam.Read(kind, id)` remains the boundary. `get_finding`
> exposes only id, release, faultline, CVE, stage, components —
> `current_position`, `positions[]`, `proposals[]` are excluded.
> `get_product` exposes id and name. L4 verifies response identity
> before minting `governed-record`.

**`themis_store` becomes `themis_contract`**: the anchor pins the hash
of `policies/themis/contract.json` — Governance base URL, Registry base
URL, SHA-256 of the two OpenAPI specifications, and the Themis commit
identifying them. Open verifies the contract hash and configures the
seam from it.

**The contract pins the interface, not the live data.** `themis_contract`
establishes the authorized endpoint and interface contract; it does not
establish the runtime Themis binary identity or pin Finding/Product
bytes. The harness intentionally reads CURRENT Themis state, never a
snapshot.

The API credential is seam-local: the read-scope key comes from the
environment and is never exposed through the anchor, registry data,
model context, or governed-record content. HTTP failure and 404 remain
`ErrUnavailable`, fail-closed.

**Execution-time data becomes immutable evidence once captured:**

```
EXECUTION       Live Themis → ThemisSeam.Read → L4 governed-record → L6 record
RECONSTRUCTION  L6 record plane → intake.Resolve → Evidence View
```

`intake.Resolve` never calls Themis again; it reconstructs from the
recorded bytes. Freshness is deliberately Themis's property: two
executions against the same tuple may observe different Finding
bytes, and each records exactly what it observed rather than
pretending a live mutable authority was immutable. The two paths
never merge, so historical evidence is never replaced by whatever
Themis returns today.

**D-T-9 amended (visible in `themis-v0/design.md`):** the deployment
anchor pins `themis_contract`, which identifies the authorized Themis
endpoints and interface contract. Findings and Products are obtained
from the live Themis authority and captured as execution-time governed
records. Positions remain outside the pinned/read context.

## D-I-4 — Finding identity and demo establishment (LOCKED 2026-09-26, owner)

> Finding identity is UUID-based, never prefix-based. `themis_scope`
> gains a `uuid` syntax; `remediate-dependency@4` uses UUID-scoped
> `get_finding`. The security property is `requested Finding UUID →
> exact governed-record identity`, not a prefix real identifiers cannot
> satisfy. The tool-registry change is part of the rsys@6 deployment
> state.

- **`get_product` is removed from the demo skill.** The seam may keep
  the capability for future workflows; `remediate-dependency@4` does not
  receive authority it does not require — capability availability
  follows workflow need, never implementation availability.
- **The fix version is human-provided task input.** The projected
  Finding does not expose Knowledge's `fixed_versions`. The demo task
  carries `finding` = Themis Finding UUID, `dependency` = component from
  the Finding, `advisory` = the Finding's CVE, `target-version` =
  operator-supplied fix. The model's `get_finding` read corroborates
  dependency/CVE against the governed record.

```
Knowledge → determines available fix information
Human     → supplies the target version for this task
Harness   → verifies governed Finding identity/context
Model     → reasons about remediation
```

- **The demo Finding is established through the real Themis pipeline:**
  real vulnerable Go module → CycloneDX SBOM → Evidence → Knowledge
  correlation → Governance Finding → Finding UUID → harness demo task.
  No manually inserted Finding JSON or database row. If OSV/NVD are
  unreachable from the VM, a DOCUMENTED local fixture feed for the one
  advisory is acceptable: the Finding still emerges through the normal
  Evidence → Knowledge → Governance path.
- **Required evidence:** the runbook preserves the exact Finding UUID and
  its provenance (pipeline execution → Finding UUID → release/product →
  CVE/component → demo task tuple), while the model still obtains the
  Finding BYTES through the live read door — the runbook never becomes a
  source of security truth.
- **Kept separate (owner):** whether L9 may derive/narrow a capability
  scope from task input, and what deterministic binding stops the model
  from widening or substituting the target, is an L9 architectural
  question with its own decision — Q-I-9, taken next; NOT folded here.
