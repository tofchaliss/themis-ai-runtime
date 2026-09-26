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

## D-I-6 — Decision act and witness (LOCKED 2026-09-26, owner; retires D-T-8's observed witness on the Themis-integrated path)

> `acceptProposal` remains the decision door; no new decision
> capability. Themis derives the decider from the authenticated request
> (`key:<KeyID>`, server-derived; `actor_id` cannot override it when a
> principal is present) → accepted proposal → Position.

- **Production requires authenticated Themis operation.** The demo VM
  runs `THEMIS_AUTH_REQUIRED=1` (auth database required). Two distinct
  write-capable credentials: the proposer key raises; the decider key
  accepts. The runbook records that they are distinct principals.
- **Separation of duties is operational, not a Governance invariant.**
  The integration does not amend Themis to enforce proposer ≠ decider;
  mandatory enforcement would be a separate Themis-domain decision.
- **Witness vocabulary:** `key:<KeyID>` when a principal exists;
  `dev:<actor_id>` only in development/no-principal operation — a
  production demo must never produce it. D-T-8's
  `decider_authentication: observed-not-authenticated` is RETIRED for
  the Themis-integrated decision path (not retained beside a stronger,
  contradictory witness model).
- **Evidence is referenced, not duplicated:** Position →
  `AcceptedProposalID` → immutable `harness-execution/v1` → harness
  record plane. `themis-intake` may render the evidence view for the
  human/runbook; that rendering is presentation, not a security record.
- **`review_by` is an explicit operator choice** (useful for `deferred`
  and `accepted_risk`), never imposed by the integration.
- **Boundary:** Themis proves WHICH authenticated key accepted the
  proposal; the integration does not prove the person controlling that
  key is the intended human. Key issuance, ownership, rotation, and
  revocation stay within Themis's authentication administration.

Two-actor path: Human A (authenticated proposal) → Themis Proposal
(immutable harness evidence) → Human B (authenticated `acceptProposal`)
→ Themis Position. The harness is outside both acts.

## D-I-7 — Cross-repository placement and walls (LOCKED 2026-09-26, owner)

> Themis may consume a narrow, read/reconstruction-only harness surface;
> the harness never imports Themis; Governance never depends on the
> harness adapter; `themis-intake` is the human-operated integration
> boundary.

**Placement in Themis:**
```
internal/governance/domain            harness-execution/v1 value type (plain data, no harness import)
internal/governance/adapters/harness  intake.Resolve adapter; Resolution → domain evidence + derived trust
cmd/themis-intake                     CLI; Governance HTTP client; human-provided proposal arguments
```
The Governance service never links the harness adapter.

**Themis-side wall (adapter):** allow-list stdlib, kernel, governance
domain/app, and exactly `state`, `deployment`, `verification`,
`verification/seam`; deny `orchestration`, `tools`, `execution`,
`runtime/model`, `instructions`, `context`. No `os` writer, no
`os/exec` (arch test). The transitive reach through `verification/seam`
is documented as intentional (the T-M3 Wall 3 clarification). Property:
Themis intake can reconstruct evidence; it cannot acquire harness
execution authority.

**Whole-repository wall (owner strengthening):** EXACTLY ONE Themis
package may import the harness module —
`internal/governance/adapters/harness`. `cmd/themis-intake` consumes
the adapter's interface and never imports harness packages itself; a
transitive-dependency test covers the binary because `tests/architecture`
ignores `cmd/`.

**Harness-side wall:** Wall 1 becomes "no harness package, whole graph,
imports `github.com/themis-project/themis`". The Themis-facing seam is
an HTTP client (`net/http` → Themis API), never a Go dependency.

**Test split:**
- Harness repo: the live execution-side seam — projection, response
  identity, contract pin, unavailable/404, the HTTP boundary — against
  an httptest Governance stand-in, never a running Themis.
- Themis repo: fixture reconstruction, mapping to
  `harness-execution/v1`, refusal cases, domain persistence, dependency
  walls. The fixture's constitution hash must equal the expected
  witnessing constitution, so a constitution change makes the fixture
  visibly stale.
- **Fixture provenance (owner):** a fixture is not "checked in" until
  its provenance is explicit — real harness walk → fixture generation →
  fixture committed WITH provenance metadata → Themis fixture test.
  Forged-record tests stay separate and deliberately use L6 primitives.
  Bytes are not established merely because a fixture contains them (G2):
  the fixture proves Themis's mapping/reconstruction; the harness-side
  test proves the production walk.

**`src/themis` removal** only after: the Themis adapter compiles; harness
imports are renamed; both sides' walls pass; fixture provenance is
established; every old `src/themis` reference is eliminated. Then one
cleanup commit whose message maps each former component to its new
owner.

## D-I-8 — Demo VM topology and host sequence (LOCKED 2026-09-26, owner)

> **Invariant: `rsys@6` is the ONLY demo deployment mint; no intermediate
> anchor is created** (D-W-4's one-mint rule preserved; no `rsys@7` to
> separate the integration changes).

1. **Themis estate:** all six systemd nodes stay deployed (Registry 8082,
   Governance 8083, Knowledge 8085, Intelligence 8086, Evidence and the
   rest as deployed). The demo depends only on Registry → Evidence →
   Knowledge → Governance; Communication and Intelligence are outside
   the path. Governance and Registry run `THEMIS_AUTH_REQUIRED=1`;
   `THEMIS_GOVERNANCE_AI_ENABLED` off — the demo's proposal comes from
   the human-operated intake path, never Themis's AI mechanism.
2. **Three credentials, three holders:** read key (harness read door,
   harness environment only) · product-scoped write key (`themis-intake`,
   intake operator) · separate write key (`acceptProposal`, decider).
   Only key IDs enter the runbook/evidence; values never. No `dev:`
   decision witness may appear.
3. **Contract establishment before the mint:** Themis checkout →
   `contract.json` → running Themis commit verified → contract/spec
   hashes verified → eligible for `rsys@6` (the host-tree discipline).
   The contract identifies loopback endpoints and pinned interface
   specs; it does not pin live Finding data or attest the running
   service binary.
4. **Single mint `rsys@6`:** amended W-M1 constitution, `themis_contract`,
   UUID `themis_scope`, `remediate-dependency@4`, required catalog state,
   everything else inherited from `rsys@5`. `rsys@6` opens → validated →
   `rsys@5` withdrawn.
5. **Real Finding:** Product → Project → Release → CycloneDX SBOM →
   Evidence → Knowledge correlation → Governance Finding; the documented
   one-CVE fixture feed through Knowledge's injectable endpoint if OSV
   is unreachable. Runbook records Finding UUID, Product/Project/Release
   provenance, SBOM provenance, correlation source/fixture status. No
   manual Finding row.
6. **Host-act sequence:** THEMIS SIDE (auth configured; contract commit
   verified; Finding established) → HARNESS SIDE (W-M1..W-M3 landed and
   proven; module rename proven; `rsys@6` minted, opened, validated;
   demo execution) → DECISION SIDE (human raises; separate human
   accepts). Evidence checkpoints: vm-verify before/after, harness
   phasec, Addendum G, Themis proposal/Position records.

```
Human ─▶ themis-intake ─ local ─▶ harness record plane ─▶ intake.Resolve ─▶ Proposal ─HTTP─▶ Themis (Registry/Evidence/Knowledge/Governance)
                                                                                                   └─▶ human acceptance ─▶ Position
```

The harness supplies execution evidence, Themis supplies enterprise
authority, the authenticated human supplies the decision.

## Closure

Integration grill CLOSED 2026-09-26: D-I-1..9 locked (Q-I-1..9). The
remaining work is implementation milestones (`tasks.md`), not
architecture, unless implementation exposes a genuine category-4 gap.
