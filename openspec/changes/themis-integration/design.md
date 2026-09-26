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

## D-I-9 — Subject-bound capability scope (LOCKED 2026-09-26, owner; amends D-L9-4 and D-L9-5; implementation DEFERRED)

> The human-supplied Finding UUID becomes authorization state ONLY
> through deterministic L7 binding; it never becomes procedure text or
> model-controlled grant content.

- **L9** permits a grant template to reference a declared task input:
  `themis_scope: ["@input:finding"]`. L9 validates that `finding` is a
  declared, required, string input. L9 does not resolve the value and
  mints no scope.
- **L7** binds at SubmitTask: validated task input → `@input:finding` →
  exact Finding UUID → effective grant → sealed composition/hash. Before
  accepting the task, L7 verifies the value against the registry's
  `uuid` syntax class.
- **Strictly narrowing:** registry ceiling `themis_scope = uuid`; task
  binding `themis_scope = exactly UUID-X`; `UUID-X ⊂ uuid`. The model
  cannot widen to another UUID.
- **L4** permits `get_finding` only when `call.id == bound_finding_uuid`;
  any other id is denied deterministically and audited.
- **The grant is model-inaccessible:** the model cannot edit the grant,
  substitute or add a Finding, modify the scope, or regenerate the
  authorization hash.
- **L6** records the sealed effective grant, so the execution
  establishes "this execution was AUTHORIZED to access Finding X", not
  merely "the model was told to work on Finding X".

Ownership preserved: L9 declares the allowable binding form · L7 binds
validated task input · L4 enforces · L6 records · the model alters none
of them. No new scope interpreter.

**Sequencing (owner):** architecture LOCKED, implementation DEFERRED as
hardening after the integration lands. D-I-4's `uuid` registry scope is
sufficient for the demo; the demo depends on D-I-9 only if its explicit
objective includes "even if the model attempts to substitute another
Finding, L4 prevents it" (a hostile-control demonstration, valuable but
not required for the core integration).

Archive amendment recorded against D-L9-4 (task input may participate
in the closed placeholder mechanism, scope fields only) and D-L9-5 (the
grant template gains a formally bounded narrowing input).

## D-I-5 — Decision Proposal integration (LOCKED 2026-09-26, owner)

> The harness does not become the proposer. It produces evidence that a
> HUMAN uses to create a Themis proposal.

```
Harness execution → intake.Resolve → Evidence View → human invokes themis-intake
   → Governance Proposal (proposer_kind = human, proposer_id = authenticated principal)
   → separate human acceptance → Position
```

- **Themis's stance vocabulary is authoritative** (`affected`,
  `not_affected`, `under_investigation`, `mitigated`, `accepted_risk`,
  `deferred`). D-T-7's four dispositions are NOT carried into the
  proposal API; `remediation-planned` is not invented as a seventh
  stance; any domain mapping is a separate Themis-domain decision.
- **Evidence is a first-class proposal field**, never a rationale
  suffix: `rationale` stays the human-readable explanation; `evidence`
  is a separate immutable closed-schema `harness-execution/v1` object —
  anchor hash; anchor name@version + lifecycle state; task id;
  artifact-bound seq; artifact ObjectID; verified member path + hash;
  verification contract name@version + hash + state; reconstruction
  verdict; production witness; constitution hash; harness module
  version. Governance validates the SHAPE; it never re-runs
  `intake.Resolve` or reconstructs the harness record (D-I-1).
- **Business Verification refs come from the RECORDED Finding bytes**
  (dependency PURL + CVE the execution actually read): Finding read →
  recorded bytes → dependency + CVE → Business Verification → human
  proposal. The model never manufactures the identity being vouched.
- **Trust is never an API default and never a CLI choice (owner
  tightening):** `themis-intake` produces a DETERMINISTIC trust
  classification from the established evidence facts, and Governance
  validates that the submitted class is permitted for that evidence.
  L10 PASS establishes admissibility of the artifact; it does not turn a
  model-authored artifact into human-authored evidence — the class is
  `inferred` by derivation, and the constitutional human-acceptance
  boundary stays intact.
- **Accepted evidence is durable Position provenance:** the Position
  cites `AcceptedProposalID`, so the evidence must be immutable and
  self-contained at proposal creation.
- **Themis-side change, under Themis's own governance:** this closes
  EDR-TRUST-01's deferred Decision Proposal payload → an EDR, a
  `phase3-*` OpenSpec change, Themis's existing proposal/acceptance
  semantics, immutable evidence persistence, no second Position model
  in the harness.

Final authority chain: Model → Harness (verifies + reconstructs) →
themis-intake (human invokes; deterministic evidence) → Themis Proposal
(authenticated human proposer) → Business Verification → human
acceptance → Position. The model is never the decision-maker; the
harness is never a Governance actor.
