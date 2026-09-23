# Harness Layer Status — L1 · L2 · L3 · L4 · L5 · L6 · L7 · L9 · L10 · L11 · cross-layer (G1/G2)

Status date: 2026-09-14 (last updated when INSTALLATION/TESTING were rebuilt against the as-built tree; content through the L11 close, the integration audit, and G1/G2). L1–L4 SHIPPED and ARCHIVED: grilled (openspec, owner-closed), implemented, security/test/architecture reviewed with three-state verdicts, live-proven against a local model. Archived changes: `openspec/changes/archive/2026-09-0{5,6}-layer-0{1,2,3,4}-*`. **L5 SHIPPED and ARCHIVED 2026-09-07** (openspec/changes/archive/2026-09-07-layer-05-execution-environment). **L6 SHIPPED and ARCHIVED 2026-09-07** (openspec/changes/archive/2026-09-07-layer-06-durable-state). **L7 SHIPPED and ARCHIVED 2026-09-07** (openspec/changes/archive/2026-09-07-layer-07-orchestration): grill closed (Q-L7-1..12), three Class-3 reviews remediated, arch HIGH 2a closed by owner decision (full L2 composition in the loop), live-proven vs qwen2.5:7b. **L9 Skills SHIPPED and ARCHIVED 2026-09-11** (openspec/changes/archive/2026-09-11-layer-09-skills): grill closed (D-L9-0..17 + amendments D-L9-11a/b/c/d), Gate 0 passed, M1–M5 implemented, residuals R-L9-1 and R-L9-2 closed, all three Class-3 reviews re-run against the final implementation with every CRITICAL/HIGH remediated and mutation-verified, live-proven vs qwen2.5:7b through the unmodified L7 loop. Three-state verdicts accepted by owner 2026-09-11; **investigate-cve@1 REGISTERED 2026-09-11** (owner-directed promotion to policies/skills/catalog.json; resolution verified ACTIVE with two-way manifest agreement); traceability.md written at archive. **L10 Verification & Observability SHIPPED and ARCHIVED 2026-09-11** (openspec/changes/archive/2026-09-11-layer-10-verification): grill closed same-day (D-L10-1..18 + D-L10-1a, owner-led), Gate 0 passed, M1–M6 implemented, three Class-3 close reviews remediated (both independent HIGHs = the same reconstruction record-selection forgery, closed by explicit record naming), five proof registers green incl. **live Register E vs qwen2.5:7b through the unmodified L7 loop with the real seam** (verify_report → report-valid@1 → PASS gates completion → cold reconstruction consistent). Three-state verdicts accepted; Governance registrations ACTIVE (registry-v4, L10 contract registry, catalog incl. remediate-dependency@1) as owner acts. Archived-layer amendments formally recorded in the L7, L4, and L6 archives (D-L10-17 discipline). Key residuals: L5 process-execution amendment for run_go_*-class verifiers (own gate), live telemetry grill (Q-L10-15), L6 audit-scope GC anchoring (ADG follow-up), one recorded equivalent mutant (verifState ordering). v1 observability = record-derived views inside the verification package; the D-L10-6 stage order carries an owner-blessed v1 clarification for read-only verifiers. Next: L11 (generalize compare/gate/variants — no second evaluation subsystem). **L11 Ratchet SHIPPED and ARCHIVED 2026-09-13** (openspec/changes/archive/2026-09-13-layer-11-ratchet): grill closed 2026-09-12 (D-L11-1..20 ALL LOCKED, 20/20, owner-led), Gate 0 PASS, M0–M7 implemented under the frozen constitution (src/harness/ratchet + cmd/themis-ratchet: criteria/regression-set registries, registered comparators, comparison packages with full conditioning tuples, door-resolved admission observations, grounded evidence, cold reconstruction, stateless derivations, Candidate/Evaluation Plan, structural walls). Three Class-3 close reviews 2026-09-13 converged on one seam (invocation boundary trusting its caller — sec CRITICAL C-1 admission forgery = arch H-1; evidence self-grounding); ALL CRITICAL/HIGH remediated (ObserveAdmission door resolution, GroundFacts, acceptance boundary with durable refusals, registry consumption-pin, reconstruction parity, ~19 mutants killed, CLI contract suite with cross-process cold reconstruction). Owner closure judgment 2026-09-13: CLOSED, three-state PASS (operationally-proven machine-local); three classifications RATIFIED (no-discard scope, consumption-pin wall shape, sensitivity residual as admission restriction). **Governance activated bench-score-delta@1 + core-regression@1 (policies/ratchet/, owner-directed act 2026-09-13).** Live proof PASS vs qwen2.5:7b (model authors candidate substance as Class-6 data-reader; every wall held). L11 residuals: knowledge-family handoff (Themis door unavailable — explicit, no substitute authority), meta-comparison / Δ-in-routing / multi-baseline (each = fresh architecture decision), benchmark-plane physical unification not performed (bench stays bench-owned), sensitivity inheritance. Owner closing sentence: no remaining L11 architecture work to design — failures default to implementation defect or recorded residual; architecture reopens only on a genuine D-L11-20 §7 gap.

---

## Checkpoint — 2026-09-15

State of the project at the end of the session. Read this before the
long status paragraph above, which is cumulative history.

| Area | State |
|---|---|
| L1–L7, L9–L11 | implemented, archived, frozen |
| **L8 Subagents** | **ARCHITECTURE CLOSED 2026-09-22 (owner LOCK), Gate 0 PASS, UNIMPLEMENTED** — D-L8-1..21 + challenge record C-L8-1..21 (`openspec/changes/l8-subagents/design.md`); milestone plan `tasks.md`; scaffold collapses to `subagents/delegation/` at M1; production authorization not implied |
| G1, G2 | closed and implemented |
| Deployment `rsys@3` | **ACTIVE** (`b5f551ab7076`); `rsys@2`, `rsys@1`, `local-dev@1` withdrawn and interpretable |
| Model allowlist | `qwen2.5:7b`, `gpt-oss:20b`, `cyberpal20b-v3` — three admitted, **none selected** |
| Model selection | **OPEN**. Admitting is not selecting; selection is per-task in the envelope |
| **L9 skills under G1** | **IMPLEMENTED IN CODE, NOT YET IN PRODUCTION** — D-SA-1..10 built as SA-M1..M5 (`dc8034c`, 2026-09-22) + review remediation (2026-09-23: `internal/strictjson` key wall after security CRITICAL-1); positive twin + A-SA-1..10 negatives green; first anchored skill execution PASSED under a test-harness anchor (`TestLiveAnchoredSkillWalk`, qwen2.5:7b). Production still unreachable until the owner's `rsys@4` Governance act lists `skills[]` |
| Live-model COMPLETED | **achieved in the test harness only** (anchored `investigate-cve@1`, COMPLETED/VERIFIED); not under a production anchor |
| Mutation Tier 1–4 | closed, 37 controls; ~190 error-propagation survivors deliberately untriaged |
| CI | green, dual-platform (ubuntu CI + darwin development) |

### The one blocking defect

**Anchored skill admission refuses every legitimate submission.**
`verifyAnchoredSkill` compares the registered manifest identity
(`sha256` of `skill.json`) against the instantiated composition seal
(`sha256` over nine members, two of them per-task). Different identity
domains; the comparison succeeds for no input. **Both catalog skills are
ACTIVE and unreachable under an anchored deployment**, which is the
production configuration.

Found live, not by reading. No test could have caught it: the only
coverage is Phase C row C17, which asserts a *doctored* composition is
refused — and an unconditional refusal satisfies that. It established
"bad input is refused" and never the required pair.

- Evidence: `docs/development/finding-anchored-skill-admission-2026-09-15.md`
- Tracked: GitHub issue #1 (`needs-triage`)
- Decision: `openspec/changes/archive/2026-09-23-l9-l7-skill-admission-identity/proposal.md`
- **Not fixed.** The correspondence between registered skill identity and
  instantiated composition identity is an owner decision spanning L9 and
  L7 with G1 as context. Phase C row C17 is **unproven** pending a
  positive counterpart.

### What today established

- **35 + 2 controls closed** across the mutation Tier-1 list, the G1
  anchor pin surface, L7 static boundedness, the L9 substitution
  boundary, the L5 execution floors, and egress discipline. One real code
  defect underneath them: `grantAuthorityDigest` was blind to
  `ThemisScope`, so grants with materially different themis-id authority
  recorded an identical `grant_authority`.
- **Records survive a binary change.** All six `rsys` tasks
  re-establish identity → durable bytes → registry under the rebuilt
  binary, four of them under a *withdrawn* anchor (Addendum D).
- **`gpt-oss:20b` benchmarked** — first candidate to move fabrication
  resistance off zero (B009 0% → 50%), and confirmation of Addendum B's
  quantisation hypothesis (recall 16% → 83%). Addendum E.
- **A live model drove the governed workflow to the L10 gate** for the
  first time: explored, transitioned phases, wrote files, called
  `verify_report`. It does **not** establish that `gpt-oss:20b` can
  complete the task end to end.

### Two grills queued, both owner-led

1. **L8 Subagents** — grill CLOSED 2026-09-22, Gate 0 PASS (see the
   2026-09-22 checkpoint below). Implementation not started; M0
   prerequisite P-L8-1 (provider-response hard ceiling) precedes L8
   code.
2. **L9/L7 skill-admission identity** —
   `openspec/changes/archive/2026-09-23-l9-l7-skill-admission-identity/proposal.md`.
   Built 2026-09-22/23 (see the table); remaining: `rsys@4` act and a
   production `themis-run` (owner), traceability + archive, issue #1.

### The session's recurring finding

Every defect found today — in the harness, in the evidence tooling, and
in the instruments written to look for defects — was the same shape: **a
check whose message claimed more than the check established.** Mutual
cover (two adjacent guards, one test tripping both) accounted for six
instances; the rechecker misreported a working append-only registry as
broken; the benchmark evaluator misdiagnosed a model's repetition loop
as a transport fault; a report rendered an unvalidated run as 0%. None
was found by reading. Each was found by running something that could
disagree.

---

## Architecture flow

```
                         Themis (system of record)
                                   │
        ┌──────────────────────────┼──────────────────────────┐
        │                          │                          │
  workflow context           instruction roots          governed records
  contract (ceiling)         (global/ themis/)          (via typed seam)
        │                          │                          │
        │                          ▼                          │
        │              ┌───────────────────────┐              │
        │              │ L1  INSTRUCTIONS      │              │
        │              │ resolve → EIS (hash)  │              │
        │              │ render (final pass)   │              │
        │              └───────────┬───────────┘              │
        │                          │ system message,          │
        │                          │ verbatim + hash-bound    │
        ▼                          │                          ▼
┌───────────────────────┐          │            ┌───────────────────────┐
│ L7  ORCHESTRATION     │          │            │ L2  GATHER            │
│ governed lattice · δ  │──────────┼───────────►│ registered sources    │
└───────────────────────┘          │            │ classify + provenance │
                                   │            └───────────┬───────────┘
                                   │                        │ typed items
                                   │                        ▼
                                   │            ┌───────────────────────┐
                                   │            │ L3  MANAGE            │
                                   │            │ policy-executed       │
                                   │            │ dedup · rank · budget │
                                   │            └───────────┬───────────┘
                                   │                        │ managed set +
                                   │                        │ selection trace
                                   ▼                        ▼
                              ┌─────────────────────────────────┐
                              │ L2  COMPOSE                     │
                              │ system msg + framed context     │
                              │ payload hash (canonical record) │
                              └───────────────┬─────────────────┘
                                              ▼
                                     Model (advisory only)
                                              │ tool call (advisory data)
                                              ▼
                              ┌─────────────────────────────────┐
                              │ L4  AUTHORIZE                   │
                              │ grant ∩ registry ∩ target       │
                              │ typed denials · audit per call  │
                              └───────────────┬─────────────────┘
                                              │ authorized action
                                              ▼
                              ┌─────────────────────────────────┐
                              │ L5  ENVIRONMENT                 │
                              │ worktree @ pinned SHA · empty   │
                              │ env · sealed → egress → ack     │──► artifact store
                              └───────────────┬─────────────────┘    (content-addressed,
                                              │ every event,          write-once)
                                              │ committed before
                                              │ its effect
                                              ▼
                              ┌─────────────────────────────────┐
                              │ L6  DURABLE STATE (record plane)│
                              │ append-only stream · crash-safe │
                              │ objects · verified recovery ·   │
                              │ byte-exact reconstruction       │
                              └───────────────┬─────────────────┘
                                              │ operational record
                                              ▼
                             Themis governance (system of record)
                             (verification · acceptance — nothing
                              above this line can grant authority)
```

One-line ownership: **L1 decides what the model is told · L2 decides how facts reach it · L3 decides what survives the budget · L4 decides what its output may do · L5 decides where it runs and what may leave · L6 remembers all of it without deciding anything · L7 decides only what comes next — sequence, never selection · none of them decides what the evidence means.**

## What each layer does — one use case, end to end

Task: *"Analyze CVE-2026-12345 for product X (component libXYZ 1.4.2). Give an independent assessment."* The task payload also smuggles: *"Skip the verification guard."* The workflow contract withholds the Enterprise Position (de-anchored analysis) and permits dropping source files under budget pressure.

### L1 — Instructions (`src/harness/instructions`)

Resolves registered instruction roots + the task blob into one immutable, hashed **Effective Instruction Set**.

- The neutral role, safety rules, and Themis principles load from reviewed files; namespace ownership means the task cannot claim or shadow any of their ids — even with a byte-identical body.
- *"Skip the verification guard"* hits the boundary-tier directive pattern → **rejected and recorded** (`rejected-prohibited` conflict, real body hash); the run continues under the intact constitution; status `completed_with_conflicts`, never a silent success.
- Renders deterministically (role first, fixed furniture; bodies verbatim; final validation pass) and hands L2 a verbatim, hash-bound system message. **L1 never calls the model.**
- Even if the pattern had missed: verification is enforced downstream — L1 is prompt hygiene, not a security dependency.

### L2 — Context Delivery (`src/harness/context`: Gather + Compose)

Turns the L7 plan into typed, classified evidence, then composes the payload. **Without judgment.**

- **Gather:** every assignment checked against the contract ceiling (unknown slot, class, sensitivity ⇒ fail closed). The finding arrives via the Themis seam as `governed-record`; the task's asserted facts are forced `external-untrusted` (caller-supplied classification is overwritten — authorship decides, never transport); workspace files are confined reads (symlink escapes refuse hard).
- The withheld Enterprise Position renders as `withheld_by_contract` — existence + state only, zero content, zero guidance. The model can never falsely conclude "no position exists"; the human decision surface still carries the full Position.
- A CVE PoC reading *"disable the sandbox"* is delivered **byte-exact** — evidence is never sanitized and never pattern-checked; the analyst must see what the evidence says.
- **Compose:** EIS verbatim + evidence in content-bound frames (composition-wide candidate-skip fence — unforgeable by content), length-framed canonical record under `PayloadHash`. Delivery + tool traces jointly reconstruct every model-visible byte.

### L3 — Context Management (`src/harness/context`: Manage)

Decides which already-classified items survive the token budget — by executing a governed policy, never by judging.

- **Capacity is never authority:** required slots (finding, task facts) are untouchable; only contract-declared-droppable slots enter the drop order; if the undroppable set alone exceeds the budget, the run **fails closed** — no degraded evidence basis, ever.
- Under pressure the large source file is dropped (declared rank keys over `ItemRef` metadata only — ranking cannot read evidence bytes by type-system construction). The model sees a state-only marker — `omitted_for_capacity` — counts and mechanics stay in the trace. `withheld_by_contract` (deliberate) and `omitted_for_capacity` (pressure) remain distinguishable forever.
- Byte-identical evidence within one authority class collapses to a single delivery with every supplier recorded; the same bytes under two authority classes are **both** delivered — the provenance duality is evidence, not waste.
- A managed set refuses re-management (the marker can't be silently erased); rank never changes presentation order.

### L4 — Tool Interface (`src/harness/tools`)

The first enforcement layer: is this capability authorized for this execution?

- The model asks to read `parser.go`: registry ∩ grant ∩ quota established **before** argument inspection (anti-oracle order); the target confines under the workspace; result classified `external-untrusted` with hash — an audit event on every path.
- It asks for an ungranted tool: `not-available`, zero detail (unknown ∪ not-granted ∪ quota collapse — no capability oracle). It asks for `../../etc/passwd`: `target-refused`. Extra argument: whole-call `invalid-args`, never silent stripping.
- Live-proven six ways against qwen2.5:7b.

### L5 — Execution Environment (`src/harness/execution` + `src/harness/confine`)

The safe computer: provisioned, isolated, provenance-pinned, torn down verified.

- The task's repository provisions from a **local mirror at a pinned SHA** (post-condition verified, binary attested) into a single-use environment: empty-env allowlist (no PATH), inherited identity declared honestly, denied-by-construction network/host-services, wall-clock budget enforced at the envelope.
- Mutating tools (`write_file`, transactional `apply_patch` — registry-v2, mutating-visible grants) pass CreateMode confinement: no symlink anywhere in a write path, broad `.git*` deny-list; the L4 gate is identical inside — containment, never a second permission system.
- The lifecycle is one monotonic machine — PROVISIONING → ACTIVE → **SEALED** (irreversible) → EGRESSING → ACKNOWLEDGED → TEARDOWN → DESTROYED | TEARDOWN_ANOMALOUS — and its invariants hold by reachability: no egress from an unsealed workspace, no partial artifacts, no unverified "destroyed".
- The hand-off is a diff-against-the-pin manifest in a content-addressed, write-once ArtifactStore (custody ends at acknowledgment; the store survives teardown; retention is L6's). No push credential, no remote write path — hand-off, never push.
- Registered repositories may speak at repository scope through the four-control chain (registration + pinned provenance + scope cap + pattern gate); an unregistered `AGENTS.md` is data, not instructions.
- Live-proven vs qwen2.5:7b: model-driven mutation landed in the worktree, escapes refused typed, artifact acknowledged, teardown verified clean.

### L6 — Durable State (`src/harness/state`)

The record plane: what the harness did, kept honestly.

- Every layer's typed events land in an **append-only, per-entry-hashed stream per task** — the authoritative record; the manifest is a derived projection, re-verified claim by claim by a cold verifier that trusts nothing. Objects (evidence payloads, egress artifacts) are content-addressed, write-once, published crash-safe (an address is only ever absent or complete).
- **Durable means fsync-committed before acknowledgment**, at honestly-declared local-substrate strength; a durable record references only already-durable material — orphans are waste, dangles are lies, and the ordering makes lies unproducible. **Record-before-effect:** the audit event commits before a tool result reaches the model; the delivery record commits before the payload does.
- Crash yields a typed `FAILED_PARTIAL` established by recovery, which holds projection authority and structural-fact authority and **zero origination authority** — it can finish projecting what the record already says, never invent an outcome; corruption is a verdict about the record, never a rewrite of what the task did.
- Retention is graph reachability (retain-all v1, no deletion path in code — a structural absence proof); the model has no L6 verb; L7 gets a status view that structurally cannot carry contents; Themis remains the system of record — `durable(local)` / `transferred` / `accepted` are permanently distinct words.
- Proven three ways: an exhaustive 9-point fault-injection sweep, a real SIGKILL mid-task with cold recovery, and a live full-stack task (qwen2.5:7b) reconstructed **byte-exactly** from disk alone.

### L7 — Orchestration (`src/harness/orchestration`)

The deterministic executor of a governed workflow lattice — sequence, never selection.

- The task arrives as one governed **envelope** (the only input; nothing defaulted, every reference named on refusal, payload capped and external-untrusted forever). Assembly is a ⊆-checkpoint that trusts the submitter for nothing: workflow ⊆ ceiling ⊆ registry, grant ⊆ workflow ceiling, spec ⊆ execution ceiling, contract bound to the workflow, task_id bound across every artifact, all hashes + both constitutions into the L6 attribution.
- The **workflow definition** is statically verified at load: total edge maps (every declared event, exactly once, per phase), counter-free cycles unloadable, every counter with a strictly-forward exhaustion edge, finite worst-case walk ≤ ceiling. Runtime-producible events are load-mandatory, so legal model behavior can never reach the invariant path.
- **δ is a function, not a planner**: it consumes (definition-at-hash, cursor, declared typed event, counters) — structural turn facts and typed L4 gate outcomes only. Model content never reaches control; the sole completion mechanism is a registered zero-argument control verb (`declare_done`) whose authorized execution emits a constitution-owned signal onto a governed edge. Prose, JSON-in-prose, invented arguments: all inert.
- Each phase entry composes fresh **through the full L2 pipeline** (owner decision on the one review HIGH: implement, not narrow) — fenced, provenance-labeled, recorded byte-exact before delivery; every turn, tool result, and transition commits before the next turn (record-before-next-turn); phase capabilities narrow the grant at the gate.
- Floors are constitution-owned and undeclarable: wall-clock exhaustion seals the environment and fails the task — never a workflow edge a definition could reroute. Startup closes every non-terminal record before accepting work; resume has no object, by reachability. Retry is a new identity carrying `retry_of`.
- Proven five ways: structural API closure, adversarial (every laundering path pinned), fault sweep + real SIGKILL child with no continuation, a deterministic replayer that re-derives every transition from the record and checks the recorded cause against its own derivation (single-authority), and a live walk vs qwen2.5:7b through the production loop — including negative live proofs.

### What the model then gets — and cannot do

A system message stating its rules (minus the rejected injection), evidence labeled by who authored it, honest markers for everything absent, tools that refuse everything ungranted, an execution environment whose blast radius is one disposable worktree, and a durable record plane that remembers every decision without ever making one — and no path anywhere in L1–L6 that converts its output into authority. Its reply is advisory input to deterministic verification and Themis governance — the layers that come next.

## Standing safety property (owner-accepted, applies to all three)

Failure of L1/L2/L3 can garble or starve model input; it can never bypass authorization, verification, or governance. Deferred behind dedicated grills: probabilistic selection, compression, repository/skill instruction sources (L5/L9 provenance), capability-fetch expansion (L4), trace persistence (L6), the approval channel + subagents + run_command (L7 residuals / L8 / OPEN-2).

## Cross-layer integration audit + G1/G2 (2026-09-13, pushed at 9de4958)

**L1–L11 integration audit COMPLETE** (charter → three parallel seam
audits S1–S10+X1–X3 → synthesis → remediation → Phase C):
openspec/changes/archive/2026-09-13-l1-l11-integration-audit/. Result: the intra-layer
walls held essentially everywhere; every serious leak was an entry or
consumption boundary trusting its caller. Two genuine cross-layer
gaps were surfaced, grilled, closed, implemented, reviewed, and
re-remediated.

**G1 — Deployment Authority Anchoring: CLOSED and IMPLEMENTED**
(openspec/changes/archive/2026-09-13-g1-deployment-authority/). D-G1-1 + owner
amendment D-G1-1A: a caller-supplied anchor path/hash IDENTIFIES a
requested deployment; only resolution against the Governance-active
anchors registry ADMITS it. Implemented in src/harness/deployment/
(read-only, no write API) and the L7 seam: admission before the
instruction plane is consumed, per-task re-verification, indivisible
per-workflow bundles (bounded), constitution pins, model-registry
pin, append-only anchors registry across restarts, durable anchor
bytes plus a read-path VerifyAnchorRecord, and an explicit
`Unanchored` opt-in (no silent bypass; unanchored records carry a
sentinel). Execution ceiling: DEPLOYMENT-scoped, deployment-supplied
exact bytes, hash-bound by the anchor, verified and frozen at Open —
no placeholders, no environment indirection, and an anchored task can
never choose its own ceiling. **No ACTIVE anchor ships in the repo by
design**: local-dev@1 is WITHDRAWN (it pinned a spec template where a
ceiling belongs; the loader was not weakened), and a runnable anchor
is deployment-instance-specific — see policies/deployment/README.md.
Production wiring is therefore gated on a concrete deployment, not on
missing architecture.

**G2 — The Established-Fact Boundary: CLOSED and ENFORCED.**
D-G2-1: storage proves bytes; EVENTS prove establishment. An L6
object is a fact of kind F only when a committed event of F's minting
class names it. Model-authored bytes are not a fact kind; L11's own
packages have no witnessing event and so are unestablishable —
closing the laundering path (model assertion → object → selector → Δ)
and the P1→K2→P2 bypass structurally. Enforced across L11 grounding,
Compare, and cold reconstruction.

**R1 executed:** the legacy themis-serve HTTP surface
(/v1/extract, /v1/recommend-position) is DECOMMISSIONED — it invoked
models entirely outside the governed chain. The model router
(internal/service/router.go) is retained as the D-L11-8 Class-2
consumer.

**Phase C register:** one workflow across the whole chain, twice,
under a concrete anchored deployment — walks' own committed events
witness the L11 facts, admission observed at the real catalog door,
cold reconstruction CONFIRMED, the deployment re-established from the
record, and the door byte-identical afterward; a model-turn object
from that same genuine history refuses to ground.

**Process invariant (AGENTS.md):** reviewers never run mutating or
destructive probes against the live working tree — use a copy or a
git worktree. Recorded after a second lost-edit incident.

**CI restored 2026-09-14 (run 34797631381) — first green run since
2026-09-06.** CI was red for the whole L5→L11 build: a gofmt failure
(09-06), then `TestExecTimeoutGroupKill` failing on Linux (09-07), then
a build step pointing at the decommissioned `themis-serve` (09-13). All
three closed. Consequence for the record: every "full suite green"
claim in the L5–L11 archives was **darwin-local evidence**; Linux is
independently verified only from 2026-09-14 onward.

**Archive evidence sweep (2026-09-14).** Prompted by the L5 finding
below, all 312 test names cited across archive traceability files were
cross-checked against the 447 tests that exist, and every skip in the
suite was enumerated. Three gaps found and closed — in each case the
control was sound and only the evidence was missing:

1. **L10 Register T #5** — `TestEventTamperDetectedAtReadBoundary` read
   `events.jsonl` where L6 writes `events.log`, so it silently skipped
   and had never executed on any platform. The L10 close recorded T #5
   CLOSED on it. Path fixed, both skip guards converted to failures.
   (`archive/2026-09-11-layer-10-verification/amendments/register-t5-evidence/`)
2. **L7 envelope 64KiB payload cap** (D-L7-1, Q-L7-10) — cited as
   `TestPayloadCap`, which did not exist. Deleting the cap left the
   whole suite green. Test written.
3. **L7 worst-case walk ≤ ceiling** (D-L7-3/4, Q-L7-9) — cited as
   `TestWorstCaseWalkBound`, which did not exist. Same: mutant survived
   the full suite. Test written.
   (2 and 3: `archive/2026-09-07-layer-07-orchestration/amendments/untested-controls-evidence/`)

Three further gaps, closed 2026-09-14 in the same sweep:

4. **L7 real-kill proof raced the host** — `TestRealKillNoContinuation`
   killed its child on a "stream > 2048 bytes" trigger polled every
   25ms, with a zero-cost scripted model. On Linux the walk finished
   inside one poll interval, so the kill landed after a typed terminal
   and the proof's premise was void (`Terminal:[t-kill]`, not
   `Recovered`). Real Linux CI failures 2026-09-07 and 2026-09-14. The
   child now costs 150ms/turn, giving the kill about a second of
   window, and the void-premise case is self-describing.
5. **L5 teardown anomaly had no Linux evidence** — the only fixture used
   darwin `chflags uchg`. A portable `unremovable-parent` fixture now
   runs everywhere; the darwin one is retained as a distinct mechanism.
6. **L5 observed-RSS conversion was undiscriminated** — `Maxrss` is
   bytes on darwin, KB on Linux; `TestEgressMemObservedGate`'s 1-byte
   bound passes identically whether the conversion is missing, correct,
   or doubled. `TestObservedRSSIsInBytes` asserts the magnitude.
   (4–6: L7 and L5 amendment records.)

7. **L5 mid-drain budget proof rode the same timing assumption** —
   `TestBudgetMidDrainAutoSeals` set a 1ms budget and assumed real git
   was slower, surviving on Linux only by the width of one fork/exec.
   It now drives the drain through the `spawnOverride` seam, and pins
   which branch sealed: pre-exec exhaustion seals with the *same*
   reason but records no op, so the two were indistinguishable by
   assertion.

Remaining known-thin evidence, recorded not closed: two egress tests
skip under root (void there by construction — the mechanism they prove
is permissions, which root does not have to obey).

**Method note.** Five of these seven surfaced only because CI went
green on a second platform after eight dark days; the cited-test
cross-check found two more. Neither method finds a control that is both
untested and correctly cited.

**That class was measured 2026-09-14** by a systematic refusal-
suppression mutation pass (`evidence/harness/mutate`, 20m35s, isolated
worktree): 971 refusal guards, 433 killed, 130 uncompilable, **408
survived**. Decomposed: 191 I/O error propagation, 60 schema validation,
157 semantic controls (115 in security-relevant packages). A Tier-1
shortlist of twelve — including two TOCTOU guards, the Open-side G1
ceiling check, the L7 constitution pin, and L9 two-way identity — is
recorded with dispositions in
`docs/development/mutation-pass-2026-09-14.md`.

**Tier 1 CLOSED 2026-09-15** — all twelve worked through individually,
each mutation-verified in a disposable worktree. Seven were real
coverage gaps now pinned by tests (endpoint refusal, the durable-capture
TOCTOU, the Open-side G1 ceiling pin, both constitution pins, the
HEAD-at-pin post-condition, L9 two-way identity and manifest_path
containment, the L11 criterion binding, provenance completeness). Five
were equivalent mutants — and **two of those rested on a premise that
was wrong**: `grantAuthorityDigest` omitted `GrantEntry.ThemisScope`, so
two grants with materially different themis-id authority recorded an
identical `grant_authority` (fixed); and `loop.go:474`'s "unreachable
when assembly held" rested on two assembly gates nothing tested (now
tested).

Three findings could not have come from reading. The `ThemisScope`
omission was a defect in a shipping control. **Mutual cover** — two
adjacent guards with one test tripping both, green and evidencing
neither half — accounted for three of the twelve. And the Tier-1
ranking was itself incomplete: the two gates beneath `loop.go:474` were
not on the list, they were the premise of an item that was.

The rule this earned: an equivalent mutant is not a closed question.
Ask "unreachable because of what, and is that tested?"

The ~190 error-propagation survivors remain a recorded coverage map,
deliberately untriaged. Remaining open: which of those deserve tests is
an owner judgement.

**L5 evidence correction (2026-09-14):** the Linux failure exposed an
overstated traceability row, not a control defect — the archived
`TestExecTimeoutGroupKill` ran a single git process, so it could not
distinguish a group kill from a child kill on any platform, and its
fixed 1ms deadline never reached the timeout branch on Linux. The test
now forks a grandchild through a production-inert seam and is
mutation-verified. Recorded as
`openspec/changes/archive/2026-09-07-layer-05-execution-environment/amendments/group-kill-evidence/AMENDMENT.md`.
No L5 decision or declaration changed.

Standing residuals: submitter authentication (G1, explicit);
sensitivity inheritance (local-endpoint scope); the four
consumption-pinned registries whose consumers live outside L7; plus
the pre-existing layer residuals.

**L8 Subagents — UNIMPLEMENTED, grill required (proposal written
2026-09-15).** The only unimplemented layer of the eleven. Proposal at
`openspec/changes/l8-subagents/proposal.md`: governing constraint ("a
subagent is a delegated reasoning execution, not a delegated
authority"), eleven prohibitions in force unless a locked decision
lifts each by name, twenty draft grill questions, and Gate 0 criteria.
Nothing is locked; the grill has not started and is owner-led.

`src/harness/subagents/{delegation,isolation,roles,runtime}/` stay
scaffold-only until it closes. The hazard is that "subagent" ordinarily
implies a second orchestration authority, another runtime, another
tool-selection mechanism, independent durable state or instructions,
role-based authority, uncontrolled delegation, or an approval path
around L7 — and L8 sits between the model's reasoning and the governed
execution machinery, so a second orchestrator built there would sit in
FRONT of L7 rather than beside it. The chain to preserve is L7 → L8 →
model → L8 result → L7 → L4/L5/L6: L8 returns to L7 and never reaches
L4.

The approval channel and `run_command` / OPEN-2 remain SEPARATE
decisions. They were deferred alongside L8, not merged into it; they
may constrain its design but must not be absorbed by it.

Two tracks, neither blocking the other: L8 is the next major
architecture activity, deployment validation the next major empirical
one.

**G1 pin determinism confirmed cross-platform (2026-09-14).** At the
same commit, `scripts/themis-status` produced byte-identical values on
darwin/arm64 and linux/amd64 for all eleven pinned identities: both
constitution hashes (L6, L7), the three instruction-root `HashDir`
values, and the six `HashFile` registry/policy pins. An anchor minted on
one host therefore admits on another — a property G1 depends on and
which had never been checked, since CI runs only Linux and development
ran only on darwin. Had `HashDir` varied with filesystem ordering or
path semantics, anchors would have been silently host-specific.

**Live-proof concurrency, corrected 2026-09-14.** This was recorded as
a "contention flake" affecting two `context` proofs under memory
pressure. First run on the deployment host (62 GB, 24-core, CPU-only)
disproved that characterisation: **six** live proofs failed in the full
sweep there — more, not fewer, than on the 16 GB laptop — and every one
passed alone minutes earlier. The mechanism is concurrency, not
capacity: nine live proofs live in nine packages, `go test` runs
packages in parallel, and all nine drive one model server. A larger host
makes it worse by running more of them at once.

It is therefore not a flake to tolerate but a scheduling rule: run the
suite hermetic (`THEMIS_LIVE_OLLAMA` at a closed port) and the live
proofs separately. Recorded in TESTING.md, the runbook Step 2, test-plan
A4/A5, and enforced as a warning by `scripts/themis-preflight`.

The live proofs remain endpoint-gated, not opt-in: they run whenever a
model endpoint answers, so a build host with Ollama up must have
`THEMIS_LIVE_TOOL_MODEL` / `THEMIS_LIVE_MODEL` pulled.

## First deployment instance — `rsys` (2026-09-14)

The first concrete Deployment Instance was stood up and validated on a
real host (Ubuntu 6.8, x86_64, 24 vCPU, 62 GB, CPU-only inference,
quota-limited autofs home). Deployment confidence is no longer unrated
for this instance; it remains unrated in general.

| Phase | Result |
|---|---|
| A build/static | 21/21 green hermetic, 7.6s; skip audit exactly 10; all live proofs green individually |
| B artifact + admission | ceiling pinned `fd8fcdc1`; anchor `rsys@1` = `04fcdfaf`; **refused before the act**, admitted after |
| C negative space | **18 of 19 rows proven against the REAL anchor**; C10 n/a (see below) |
| D anchored path | 5 governed walks, all typed terminals, D4–D6 on every one |
| E governed chain | **all six rows** — E3 grounding, E4 real-door admission, E5 Δ with derived run identities, E6 cold reconstruction CONFIRMED, E7 door byte-identical, E8 laundering refused |

**Supersession proven end to end.** Registering
`walk-report-score-delta@1` changed the criteria-registry pin, which
*forced* a second deployment identity — `rsys@2` (`666fd868`) — with
`rsys@1` withdrawn. Both then coexist in one record plane: four tasks
attributable to `rsys@1`, one to `rsys@2`. Against the *same* registry
in the *same* moment, `AdmitAnchor(rsys@1)` refuses ("a superseded
deployment definition cannot open") while `VerifyAnchorRecord(rsys@1)`
re-establishes all four of its records. Withdrawal closes the future and
preserves the past; a deletion would have made four completed governed
walks permanently unattributable.

**Phase C against the real anchor** (`evidence/harness/phasec`, 11
rows in one run plus C2/C3 proven earlier): C1 hash mismatch, C2 "a
matching hash is an identifier, never an admission claim", C3 "anchor
rsys@1 is withdrawn", C6 mutated instruction root, C7 "non-regular
entry in a pinned tree", C8 swapped policy, C11 registry against an
`absent` declaration, C12 "a model enters a deployment only by
Governance act", C13 "the ceiling is supplied at Open, never chosen per
task", C14 unanchored workflow, C16 "a mutually consistent bundle is
not a governed bundle", C18 "a deployment governs by anchor or refuses
to open", C19 ambiguous caller role. Also at L11: "unregistered
artifacts are data and measure nothing".

Every row corrupts a byte-identical COPY of the pinned trees in its own
scratch state root — the live deployment and its record plane are never
written, per the AGENTS.md probe-isolation invariant. The injected model
is inert and errors if reached, since a row that gets to a model turn
has already failed.

Not claimed: **C15** is n/a because `rsys@2` anchors one workflow bundle
and the row tests bundle indivisibility, which needs two; substituting
an unanchored ceiling was refused by the workflow loader before the
bundle check could speak, and recording that as a pass would have
claimed a control that never ran. **C10** is n/a because the anchor
declares `model_registry: absent`, making C11 its applicable form.
**C4, C5** (registry rebound/deleted between Opens), **C9** (rebuilt
constitution) and **C17** (submitter-chosen composition hash) need setup
the tool does not do.

**Two findings from the deployment, neither a defect:**

1. `orchestration.loop.toolDefs` offered the phase's declared
   capabilities **without intersecting the task grant**, though its
   comment said "granted capability subset". Not a security hole — L4
   re-checks and denies zero-detail — but the model spent turns and
   counters on calls that could never be authorized, which is what
   exhausted one walk. **CLOSED 2026-09-14:** toolDefs now intersects
   the grant (narrowing only; L4 authority unchanged), pinned by
   `TestToolDefsIntersectGrant` and mutation-verified.
2. No registered criterion could consume governed-walk evidence
   (`bench-score-delta@1` selects the benchmark plane). Closed by
   registering `walk-report-score-delta@1` as a Governance act.

**Model selection is now measured, not asserted** (sign-off Addendum B):
`qwen2.5:7b` benchmarks at **73.4%** across the 20-benchmark suite, with
**0% on B002 "Unknown CVE" and 0% on B009 "Hallucination Resistance"** —
the two that measure not fabricating — and 66% on B008 structured JSON
output. Those three predict exactly the governed-walk failure: the model
asserted a report it had never written. A model scoring zero on
fabrication still produced a governed FAILED rather than a false
success, because the gate required the report's BYTES graded under a
registered contract, not the claim to have written one. CyberPal 2.0 20B
was attempted and abandoned on a GGUF packaging defect (corrupted
harmony template, `<|channel|>` in the stop list), not a capability
judgement.

**Live-model result, recorded because it is evidence about models
rather than about the harness:** `qwen2.5:7b` on CPU could not drive
`remediate-dependency` in two attempts — it never called `write_file`,
once trying three times to verify a report it had never written, once
inventing a tool it was not granted and then stalling. Both runs still
reached typed terminals through declared edges. Phase E therefore used a
scripted model, as `integration/phasec_test.go` does, since Phase E
tests the chain and not the model.

Evidence under the deployment root: `phase-a-evidence.txt`,
`phase-b-evidence.txt`, `phase-d-evidence.txt`, `phase-d2-evidence.txt`,
`phase-e-walks.txt`, `phase-e-evidence.txt`,
`phase-d-rsys2-evidence.txt`. Tooling: `evidence/harness/`.

**Phase F sign-off record:**
`docs/development/deployment-signoff-rsys.md` — identities, per-phase
results, every Phase C refusal text, ten findings with triage categories
and dispositions, residuals, and an explicit list of what the record
does NOT establish. **Owner ACCEPTED 2026-09-14** — including its §8
limits: one host, no production-wiring grant, model selection unsettled,
five C rows unexercised. Deployment confidence is established for `rsys`
and remains unrated in general.

**Production wiring decided and built 2026-09-14** (Q-PW-1..4,
owner-locked): `cmd/themis-run` — a CLI invoked per task, not a service,
following the D-L11-14 discipline that nothing watches, schedules,
retries or continues. It holds no authority: a submitter chooses a task
WITHIN the deployment, `Unanchored` is not exposed as a flag, and access
control is the OS's. Submitter origin is **observed, not asserted** —
the runner authors the submitted envelope, stamping `submitter_*` into
the opaque `origin` map (D-L9-13), and **refuses a request that sets
those keys itself**, because the record must carry an observation rather
than a claim about who submitted. Skill attribution on the same map is
preserved. Exit 0 covers any governed terminal including FAILED; exit 1
means the deployment refused the submission. Submitter *authentication*
remains a recorded residual — bounded, since the submitter holds no
authority by construction.

**Exercised on the deployment host 2026-09-14:** `themis-run` against
`rsys@2` produced a governed terminal (FAILED/VERIFIED, exit 0) with
`origin:submitter_uid/user/host` recorded beside `deployment_anchor`;
a request asserting `submitter_uid: 0 / root` was **refused at exit 2
before the deployment opened**, so no record anywhere claims root
submitted anything. Sign-off Addendum A.

**Phase C completed 2026-09-14** (sign-off Addendum C): C4, C5, C9, C15
and C17 all now refuse for the right reason — 16 rows in one isolated
run, exit 0, plus C2/C3 proven at admission and supersession. C10 stays
n/a (this anchor declares `model_registry: absent`; C11 is its form).
C15 required three attempts: twice it produced a refusal from the WRONG
control, which a looser check would have scored as a pass — bundle
indivisibility had never actually been exercised against a real anchor
until the third. Production wiring (runbook Step 12) remains an open
owner decision — the evidence harness is deliberately not it.

## Where to look

- As-built chain diagram: `docs/architecture/harness/execution-chain.md`
- Deployment test plan incl. the real-VM scenario:
  `docs/development/deployment-test-plan.md`
- Deployment anchoring contract: `policies/deployment/README.md`
- Deployment runbook (step-by-step procedure):
  `docs/operations/deployment-runbook.md`
- Build and install: `INSTALLATION.md`
- Verification (suite, live proofs, per-layer evidence): `TESTING.md`

---

## Checkpoint — 2026-09-22

**L8 Subagents grill CLOSED, Gate 0 PASS, ARCHITECTURE LOCKED (owner-led,
one session; 21 decisions + 21 adversarial challenges C-L8-1..21).**
Record: `openspec/changes/l8-subagents/design.md` (D-L8-1..21, §5
inventory + five registers), `tasks.md` (M0–M6). The shape in one
paragraph: L8 v1 is *a deterministic delegation mechanism for isolated
model inference* — one tool-less, single-call model execution over a
parent-supplied L2 composition, requested by the parent model through
the existing L4 gate as a `delegate` capability (verifier-seam
precedent), composed by L7 from a Governance-registered **delegation
template** (a new artifact family with its own append-only registry,
disjoint from Skills, linked only through the grant's `template_scope`,
anchor-pinned), charged entirely to the parent's budget, recorded as a
new `l8-delegation` event in the parent's L6 stream (constitution
amendment → `rsys@4`), and returned as untrusted data with no fact
kind. Failure is stage-indexed on the instance boundary
(composition durably stored); no delegation event reaches δ; L10
observes delegations through the record and never evaluates them;
depth is exactly one by reachability. All eleven prohibitions in
force; `roles/`, `runtime/`, `isolation/` have no referent and are
deleted at M1.

**Finding (codebase, 2026-09-22):** no hard capture ceiling exists for
any model response — both adapters `io.ReadAll` the provider body and
the object store has no size cap. Pre-existing parent-side gap,
recorded as prerequisite **P-L8-1** (M0) rather than L8 scope.

**Grill-found corrections folded (C-L8-12/13/20):** stage B
instantiation runs inside the `delegate` executor so refusals are
reconstructable `l4-audit{error}` and the audited evidence is an
identities-only *instantiation capture*; the post-hook re-derives and
never reads the capture; establishment is strictly the `l8-delegation`
witness; identity is `(task_id, l8-delegation seq)`, no
`delegation_id`. Findings: P-L8-1 (no provider-response ceiling; both
adapters `io.ReadAll`), P-L8-2 (adapters echo the requested model, never
the reported one), F-L8-2 (parent `model-turn` records name only),
F-L8-3 (L10 seam pre-instance refusal text not reconstructable),
F-L8-4 (parent turn timeout not bounded by remaining deadline).
**Dependency:** Q-SA-6 in the skill-admission grill must treat
`template_scope` as fixed-by-skill, equality-checked; the anchored
positive-path proof (Register B) cannot run until issue #1 closes.

**Authoritative L8 statement (owner, design.md §7):** L8 is a bounded,
isolated, tool-less delegated reasoning mechanism subordinate to the
parent governed execution; no independent governance, capability,
workflow, durable execution state, verification authority,
security-truth authority, or recursive delegation authority;
establishment is witness-based through `l8-delegation`; inputs derive
through L1/L2 and explicit parent-owned evidence references; outputs
remain `external-untrusted` until a separate governed door establishes
a stronger meaning. Six prerequisites before implementation is declared
safe: P-L8-1, P-L8-2, F-L8-2, F-L8-3, F-L8-4, Q-SA-6.

Skill-admission identity (issue #1) remains the blocking defect for
live-COMPLETED work; unchanged today.

---

## Checkpoint — 2026-09-23: skill-admission identity BUILT and ARCHIVED

**`l9-l7-skill-admission-identity` — SA-M1..M6 implemented (`dc8034c`,
`7be7116`), three Class-3 reviews remediated, archived at
`openspec/changes/archive/2026-09-23-l9-l7-skill-admission-identity/`
with `traceability.md`.** The first anchored skill execution passed
(`TestLiveAnchoredSkillWalk`, qwen2.5:7b, COMPLETED/VERIFIED) under a
test-harness anchor pinning the real catalog. The security review of
`dc8034c` found a CRITICAL the twins had not: a case-variant
`"Workspace"` grant key passed L7's exact-key `@workspace` guard and
bound a literal host path through the case-insensitive decoder — the
recurring shape again (a guard reading something other than what the
decoder binds). Closed by `src/harness/internal/strictjson` (exact
lowercase keys, no duplicates) before every grant/spec decode at L4,
L5, L7, L9, plus a post-load bound-root assertion. L8 dependency Q-SA-6
discharged. Still the owner's: `rsys@4` listing `skills[]`, a
production `themis-run`, issue #1 closure. Residuals in the archived
`traceability.md`.

## Checkpoint — 2026-09-22 (second entry): skill-admission grill CLOSED

**`l9-l7-skill-admission-identity` — D-SA-1..10 LOCKED, Q-SA-1..12
CLOSED, drill record A-SA-1..11, owner closure.** Architecture CLOSED;
implementation NOT CONFORMANT. The finding of 2026-09-15 (issue #1) is
now decided: `verifyAnchoredSkill` compared two identity domains; the
locked correspondence is per-member equality of the seven Skill-fixed
identities against the anchor-pinned catalog manifest (D-SA-2), plus
`Instantiates(effective, template)` for grant and spec with
`template_scope` **equality** (D-SA-4, reference template always from
the manifest pin, never the caller's claim). New: first-class `skill`
envelope field, `origin` returned to attribution (D-SA-5); anchored
refusal of unattributed `skill_procedure_path` (D-SA-3 — closes finding
F-SA-1, a submitter could author a skill-scope instruction source under
an anchored deployment); seal has exactly two consumers (D-SA-6);
unanchored = a different admission contract, Claim 1 only (D-SA-7);
withdrawal forward-only on the identity (D-SA-8); anchor `skills[]`
allowlist as an additive G1 pin (D-SA-9); the anchor selects, never
composes (D-SA-10). Deferred: F-SA-2 → L11 (witness must carry
`deployment_anchor`; consumption must classify `unanchored`).
Discharges L8's Q-SA-6 dependency. Implementation plan: SA-M1..M6;
acceptance = the C17 twin suite naming the responsible gate.
