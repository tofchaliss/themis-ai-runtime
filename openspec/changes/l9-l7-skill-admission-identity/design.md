# Design: skill identity vs instantiated composition identity (L9/L7 admission)

Grill OPEN 2026-09-22 → **CLOSED 2026-09-22: D-SA-1..10 LOCKED, Q-SA-1..12
CLOSED, attack record A-SA-1..11 complete, F-SA-1 closed, F-SA-2
deferred to L11. Architecture CLOSED; implementation NOT CONFORMANT
YET** (§5 whitelist, `tasks.md`, §7 closure). §2 holds the locked
constitution; §3 the question table; §6 the drill record. Proposal: `proposal.md`
(Q-SA-1..12, Gate 0 criteria). Raised by
`docs/development/finding-anchored-skill-admission-2026-09-15.md`,
GitHub issue #1. Blocks live-COMPLETED work and L8 Register B
(C-L8-16 dependency: `template_scope` fixed-by-skill,
equality-checked).

## 0. Position in the flow

At `SubmitTask` under an ANCHORED deployment, after the workflow bundle
is verified against the anchor and before the model allowlist check:
the point where L7 must establish that a skill-attributed envelope IS
the composition Governance registered under the claimed `name@version`,
without becoming a skill-semantics interpreter.

```
envelope {origin.skill, composition commitment, artifact paths}
   ─► anchored catalog (hash-pinned) ─► entry ─► manifest (skill.json)
   ─► correspondence check ─► admitted | refused
```

## 1. Hard invariants (inherited, not grillable)

- Model output advisory, never authority. A submitter selects an
  anchored skill, never its constituent hashes (owner finding 1, G1).
- Governance owns registration; L9 instantiates, it does not register
  (D-L9-3). L9 is the sole *semantic* resolver of a composition
  (D-L9-11a C2).
- L7 stays hash-comparing and learns nothing about skills' meaning
  (D-L9-13). **G1 supersedes the letter of "L7 does not consult the
  catalog":** under an anchored deployment L7 resolves the skill ref
  from the anchor-pinned catalog (G1 design §"Skill resolution from the
  anchored catalog — DONE"; `verifyAnchoredSkill`). The scope of that
  read is what this grill fixes.
- Two claims, never collapsed (D-L9-11a): Claim 1, execution
  integrity ("I executed exactly the submitted composition") — closed
  by the seal + per-artifact verification; Claim 2, governance identity
  ("the submitted composition is the one registered as X") — the
  subject of this grill.
- Skill attribution requires a commitment (D-L9-11d); the commitment
  seals seven Skill-fixed members + effective grant + effective spec
  (D-L9-11c); registry and exec ceiling are a different identity
  domain and are never sealed.
- Instantiation surface is closed: Class 1 skill-fixed, Class 2
  caller-narrowable (quotas, deadline), Class 3 task-state, Class 4
  deployment-governed (D-L9-7).
- Fail closed; nothing defaulted. Past records stay interpretable
  across supersession. Day-0 prohibitions.
- The C17 lesson: every negative test needs its positive twin.

## 2. Locked decisions (fold target)

### D-SA-1 — Ownership of the correspondence check: both layers, at named strengths (LOCKED 2026-09-22, Q-SA-1)

- **L9 at instantiation** is the sole *semantic* resolver (D-L9-11a
  C2): resolves the ref, verifies manifest bytes, materializes the
  seven fixed members, instantiates grant and spec from their
  templates. Early admission hygiene with zero trust discount
  (D-L9-13): real, not authoritative.
- **L7 at assembly, anchored deployments only,** is *authoritative*
  and *identity-only*: resolves `name@version` in the anchor-pinned
  catalog, verifies manifest bytes against the entry hash (two-way),
  compares the manifest's seven member pins against the commitment's
  seven fixed fields — hash equality per member, nothing else. L7
  never reads a workflow, procedure, or schema for meaning.
- **Governance, through the catalog,** owns what the identity is
  composed of.

Why not L9 alone: forged or hand-assembled envelopes never pass
through L9 (D-L9-11a: an internally consistent forged composition is
not refusable without catalog evidence); the admitting layer must
consult that evidence or owner finding 1 is unenforced. Why not L7
alone: L7 stays blind to skill semantics; instantiation and schema
validation are L9's.

**D-L9-13 amendment (locked):** *"L7 does not consult the Skill
catalog" → "L7 does not consult the Skill catalog for semantics; under
an anchored deployment it reads the anchor-pinned catalog's registered
identities for the correspondence check, comparing hashes only."*
(Records the supersession G1 already made: G1 design §"Skill
resolution from the anchored catalog — DONE".)

### D-SA-2 — Skill/composition correspondence (LOCKED 2026-09-22, owner; Q-SA-6 remains a separate dependency)

> For an envelope attributed to Skill `name@version`, the submitted
> Skill-fixed composition must correspond exactly to the
> Governance-approved composition registered for that Skill version.
> The caller-supplied commitment/seal establishes only
> self-consistency. It does not establish Governance identity or
> authority.

Admission requires: (i) commitment seal self-consistent; (ii)
`name@version` resolves through the anchor-pinned Skill catalog; (iii)
the entry is ACTIVE; (iv) the entry's manifest bytes match its
registered `composition_sha256`; (v) every Skill-fixed member in the
submitted commitment equals the corresponding manifest pin; (vi) the
referenced workflow/bundle is in the Deployment Anchor; (vii)
effective grant/spec satisfy the applicable narrowing rules, with
**Q-SA-6 governing grant/template correspondence** (open).

| Item | Authority |
|---|---|
| submitted commitment | caller claim / self-consistency |
| Skill `name@version` | Governance catalog |
| Skill composition | Governance manifest |
| deployment membership | Deployment Anchor |
| grant narrowing | L9/L4 admission rules |
| runtime authorization | L4 |
| workflow execution | L7 |

**Invariant:** *caller composition ≠ authority.* The caller can
describe a composition; only the Governance catalog establishes which
composition belongs to `security-review@1`.

**Refusal classes (kept distinct, separately testable):**
self-inconsistent seal → integrity violation → `ErrInvariant`; valid
seal but composition ≠ registered Skill → governance/admission
mismatch → `ErrAssembly`. Both pre-task: no durable fact, no L6
witness.

**Implementation residual:** today's code refuses legitimate and
attacker submissions alike. **Gate 0 requires the twin test:** valid
registered composition → ACCEPT; same Skill + caller-substituted fixed
member → REFUSE. The second alone is insufficient.

### D-SA-3 — Skill-scope instruction provenance (LOCKED 2026-09-22, owner; closes F-SA-1)

> Under an anchored deployment, a skill-scope instruction artifact may
> enter the effective instruction set only through a Skill-attributed
> envelope whose Skill composition satisfies D-SA-2. An unattributed
> envelope carrying `skill_procedure_path` is refused at assembly. The
> procedure's self-supplied hash may establish byte integrity, but it
> cannot establish Governance provenance or Skill-scope authority.

**Hash integrity ≠ Governance provenance.** `sha256(P1)` matching the
caller's hash proves the caller supplied those bytes; it does not
prove Governance approved them or that they belong to
`security-review@1`. The prior path let the first statement masquerade
as the second by placing the artifact at skill scope — violating the
L1/L9 ownership model even though L4 capability could not widen.

| Deployment | Unattributed `skill_procedure_path` |
|---|---|
| anchored | refuse (`ErrAssembly`, governance/admission; pre-task, no L6 witness) |
| unanchored | existing behaviour preserved (the L9 seam tests rely on it) |

**Additional invariant (owner):** *No caller-controlled path +
caller-controlled hash pair may independently establish a
higher-trust L1 source classification.* The path may identify bytes;
the hash may identify bytes; neither may establish skill-scope
provenance. (Prevents the same vulnerability reappearing under a
differently named field.)

### D-SA-4 — Effective grant instantiation (LOCKED 2026-09-22, owner; closes Q-SA-6)

> For an anchored Skill admission, the effective grant must be a valid
> instantiation of the Governance-approved grant template. The relation
> is field-specific: caller-narrowable numeric fields may narrow within
> bounds; Skill-fixed structural fields must remain equal. Critically:
> `effective.template_scope == template.template_scope`, not `⊆`.

`template_scope` is not a resource quota; it defines which delegation
templates the reviewed composition permits. Numeric capacity → caller
may narrow; structural authority → caller may not alter.

| Grant field | Effective relation |
|---|---|
| tool set | equality |
| per-tool `max_calls` | effective ≤ template |
| `total_max_calls` | effective ≤ template |
| `workspace` | governed task binding |
| `mutating` | equality |
| `themis_scope` | set equality |
| `template_scope` | set equality |
| `task_id` | governed task binding |

Spec likewise: `@task_id`/`@repo`/`@pinned_sha` substituted;
`wall_deadline_s` ≤ template; all else equal.

**Ownership:** Governance establishes the template; L4 grant vocabulary
defines the deterministic `Instantiates(effective, template)`; L7
assembly applies it at the anchored admission boundary (L7 does not
decide grant policy — it mechanically verifies a Governance-defined
relationship); L9 constructs valid instances by construction,
non-authoritatively; L4 runtime authorizes calls against the admitted
effective grant.

**Admission sequence (anchored):** seal self-consistency → catalog
resolution → manifest integrity → Skill/member correspondence (D-SA-2)
→ resolve governed grant template → verify template identity/hash →
`Instantiates(effective, template)` (D-SA-4) → `grantWithinCeiling` →
registry/capability admissibility → materialization. *A digest proves
what was submitted; it does not prove the submitted artifact is a
valid instantiation.*

**Reference-source rule (A-SA-4; owner: retain verbatim):** the
authority chain is `Skill name@version → Governance manifest →
grant_template = A → resolve A from the governed catalog/artifact →
verify A's bytes/hash → G_eff ⊑ A`. The caller's `B` has no authority
to redirect that resolution; it participates only in (1) seal
self-consistency and (2) D-SA-2's claim-versus-manifest equality.
**Reference-source rule (detail):** D-SA-4's reference
template is resolved from the **manifest pin**, never from the
commitment's claimed `grant_template_sha256`. The claim has exactly two
consumers — the seal (Claim 1) and D-SA-2 (v) equality (Claim 2) — and
is never a lookup key. Resolving by the claim would collapse Claim 2
into Claim 1, the original defect's shape.

**Failure:** `ErrAssembly` (governance/admission mismatch), pre-task,
no L6 witness. **Twin tests:** positive = `template_scope` exactly the
template's, quotas narrowed → ACCEPT; negatives = scope +1, scope −1,
tool added, tool removed, `mutating` changed, `max_calls` increased,
`total_max_calls` increased → REFUSE, each mutation-coupled: removing
one clause of `Instantiates()` lets exactly its mutation through.
Closes the L8 residual "caller-narrowed `template_scope`".

### D-SA-5 — First-class Skill admission identity (LOCKED 2026-09-22, owner; resolves Q-SA-2, Q-SA-3)

> The claimed Skill identity is a first-class, schema-validated
> envelope field `skill = name@version`. It is the sole admission
> selector for Skill correspondence under an anchored deployment.
> `origin` remains opaque attribution and is not an authority-bearing
> or admission-control namespace. `skill_procedure_path` and
> `skill_procedure_sha256` are composition locators, not independent
> declarations of a skill-scope instruction source; under an anchored
> deployment they are admissible only through a Skill-attributed
> envelope whose composition passes D-SA-2.

```
skill ──selects Governance relationship──► D-SA-2 correspondence
      ──► D-SA-4 instantiation ──► L1 ActivateSkillSource
origin ──attribution only; no admission authority
```

**Coherence matrix (LOCKED; extends D-L9-11d):**

| `skill` | `origin.skill` | commitment | procedure | anchored result |
|---|---|---|---|---|
| absent | absent | absent | absent | ordinary envelope |
| absent | absent | absent | present | refuse |
| absent | absent | present | absent | permitted; Claim 1 only |
| absent | absent | present | present | refuse |
| present | any | absent | any | refuse |
| present | mismatching attribution | present | any | refuse |
| present | matching / absent | present | manifest permits procedure | D-SA-2 → D-SA-4 |

**Nuance (owner):** `origin.skill` being present is not what grants
authority — the first-class `skill` field does. `origin.skill` is
retained because attribution/provenance has independent audit value,
and inconsistency between the two is itself detectable.

**Ownership:** Governance owns the Skill relationship · L7 determines
whether it exists and gates Skill-source activation · L1 activates
already-authorized, pinned instruction bytes · L9 constructs/stamps the
governed composition and creates no authority · `origin` = attribution
only. Closes the conceptual ambiguity behind F-SA-1.

### D-SA-6 — Composition seal (LOCKED 2026-09-22, owner; closes Q-SA-7)

> The composition seal establishes internal consistency of the
> submitted commitment and serves as the identity of the executed
> composition instance. It establishes neither authority nor Governance
> identity. Its only consumers are (1) `checkSeal` — self-consistency;
> failure is `ErrInvariant` before record creation — and (2) the L6
> task record — executed-instance identity for reconstruction and
> comparison. The seal is never compared with a Governance catalog,
> Skill manifest, Deployment Anchor, or other Governance identity.

```
Registration identity  = Governance manifest composition hash → D-SA-2 ("belongs to security-review@1?")
Instantiation identity = composition seal → checkSeal ("one coherent instance?")
effective grant        → D-SA-4 ("legitimate instantiation?")
artifact bytes         → per-artifact verification ("the committed bytes?")
```

Anchored: seal = instance integrity/identity; manifest = Governance
identity. Unanchored: seal = composition instance binding (no catalog
correspondence available; removing it would materially weaken the
model).

**Implementation-review invariant (owner):** *the composition seal must
have exactly two semantic consumers — self-consistency validation and
durable instance identity. It must never become a lookup key or
authority comparison against Governance state.*

### D-SA-7 — Unanchored execution contract (LOCKED 2026-09-22, owner; closes Q-SA-8)

> Unanchored execution is an explicitly declared test-harness role,
> unreachable from the production surface. It establishes Claim 1
> only: the task executed exactly the composition it submitted, subject
> to all deterministic controls below admission. It can never establish
> Claim 2: Governance correspondence is not performed and cannot be
> inferred. Its durable record carries `deployment_anchor =
> "unanchored"`. No deployment authority may subsequently be
> re-established from such a record. Under this role, anchored
> Governance correspondence rules do not execute; the caller composes
> the deployment.

**Architectural result (owner):** *anchored and unanchored are two
different admission contracts, not two confidence levels of the same
contract.* Unanchored is never "anchored execution with verification
skipped".

Existing mechanisms retained: sentinel prevents absence reading as
anchored · `VerifyAnchorRecord` refuses to establish deployment
authority · production cannot create an unanchored task · `Open`
rejects anchor + `Unanchored`.

**F-SA-2 — DEFERRED to L11 (consumption gap, not admission):** *L11
witness and grounding records must carry the task's
`deployment_anchor` status, and L11 consumption must explicitly
classify `unanchored` evidence; it must never be implicitly treated as
governed evidence for promotion or other Governance decisions unless
the applicable criterion explicitly permits it.* The admission-side
protections do not follow the record into an L11 package on their own.

### D-SA-8 — Forward-only Skill withdrawal (LOCKED 2026-09-22, owner; closes Q-SA-9)

> Withdrawal is a forward-only Governance act on the identity
> `name@version`, recorded append-only as `ACTIVE → WITHDRAWN`. It
> refuses all new admission and instantiation of that identity. It does
> not affect already-instantiated/running tasks or historical records.
> A running deployment observes catalog state only through its
> already-adopted Deployment Anchor; a withdrawal becomes effective for
> new execution only through a new anchor and restart. Historical
> reconstruction uses the task's durable closure and the anchor under
> which it was admitted, not the current catalog state. Under an
> anchored deployment, callers cannot bypass withdrawal by supplying an
> old manifest or catalog because Governance artifacts are resolved
> exclusively through the anchor-pinned catalog.

```
Skill artifact          — immutable forever
Skill identity name@ver — ACTIVE → WITHDRAWN (state on the identity, never mutation of the artifact)

ACTIVE ─ Task A admitted ─ WITHDRAW ─┬─ Task A remains historically valid
                                     └─ Task B → REFUSED
```

No mid-execution catalog race: the running deployment's anchor is
frozen. **Reconstruction rule (owner):** *current Governance state is
not historical execution evidence.* A currently withdrawn Skill can
have historically CONFIRMED executions; reporting its current status
is a new evaluation, never a rewrite. A = locked model; B prohibited;
C impossible under anchored admission.

### D-SA-9 — Deployment-level Skill admissibility (LOCKED 2026-09-22, owner; additive G1 amendment; closes Q-SA-10)

> Deployment admissibility of a Skill is a property of the Deployment
> Anchor, never of the global Governance catalog and never of mutable
> runtime policy. The anchor pins an exact `skills[]` set of permitted
> `name@version` identities alongside the existing `workflows[]`
> bundle set. **`workflows[]` remains an independent bundle-level
> gate; `skills[]` is an additional, finer-grained Skill-level gate.**

For an anchored Skill-attributed submission: (1) `skill ∈
anchor.skills`; (2) its workflow bundle ∈ `anchor.workflows`; (3) the
identity resolves ACTIVE in the anchor-pinned catalog; (4) D-SA-2
correspondence; (5) D-SA-4 instantiation. Absence from `skills[]` is
an assembly refusal. Why the pin is necessary: the bundle check cannot
distinguish two Skills sharing one workflow bundle with different
procedures/grants/schemas.

```
Catalog            "does this Skill exist and is it ACTIVE?"
Anchor.skills[]    "may this deployment run this Skill?"
Anchor.workflows[] "is its workflow bundle admitted?"
D-SA-2             "is this actually that Skill's composition?"
D-SA-4             "is its effective grant a legitimate instance?"
```

Deliberately separate questions. Global withdrawal (`ACTIVE →
WITHDRAWN`) affects new admission wherever governed; allowlist removal
(anchor N → N+1) affects only that deployment; neither rewrites
history. Twins: `skills[]` includes + valid composition → ACCEPT;
excludes + otherwise valid → `ErrAssembly: skill not admitted by
deployment` (never mistakable for a correspondence failure). Runtime
mutable allowlist prohibited; anchor/restart adoption retained.

## 3. Grill — question table

| Q | Status | Decision |
|---|---|---|
| Q-SA-1 | LOCKED 2026-09-22 | D-SA-1 |
| Q-SA-2 | LOCKED 2026-09-22 (attack A-SA-1 → D-SA-2) | D-SA-2 |
| Q-SA-3 | CLOSED 2026-09-22 — claimed identity travels in the load-bearing `skill` field; `origin` keys are attribution only | D-SA-4, D-SA-5 |
| Q-SA-4 | CLOSED 2026-09-22 (fixed vs instantiated members = D-L9-7 classes; D-SA-4 field table) | D-SA-2, D-SA-4 |
| Q-SA-5 | CLOSED 2026-09-22 (per-member equality retained; not replaced by one derived hash) | D-SA-2 (v), D-SA-4 |
| Q-SA-6 | CLOSED 2026-09-22 | D-SA-4 |
| Q-SA-7 | CLOSED 2026-09-22 | D-SA-6 |
| Q-SA-8 | CLOSED 2026-09-22 (D-SA-3 + D-SA-7) | D-SA-3, D-SA-7 |
| Q-SA-9 | CLOSED 2026-09-22 | D-SA-8 |
| Q-SA-10 | CLOSED 2026-09-22 | D-SA-9 |
| Q-SA-11 | CLOSED 2026-09-22 (consequence; no new authority decision) | D-SA-10 |
| Q-SA-12 | CLOSED 2026-09-22 — architecture PASS; implementation FAIL (conformance gap) | A-SA-11 |

## 5. Gate 0 — implementation whitelist and acceptance criterion (LOCKED 2026-09-22, owner)

**Implementation of already-locked decisions (no architecture
reopening):** D-SA-5 first-class `skill` envelope field (origin back to
attribution; coherence matrix enforced) · D-SA-9 anchor `skills[]`
(additive G1 pin; new anchor `rsys@4`, shared with the L8
constitution re-pin) · D-SA-2 seven-member correspondence against the
anchor-pinned catalog manifest (replaces `Seal != entry.Composition`)
· D-SA-4 governed template resolution from the manifest pin +
`tools.Instantiates(effective, template)` for grant and spec ·
D-SA-3 anchored refusal of unattributed `skill_procedure_path` ·
**Claim-2 evidence completeness:** the consumed `skill_catalog` and
`skill_manifest` bytes stored as content-addressed objects (globally
deduplicated) and referenced from the task record; `skill` recorded as
a governed key — the historical task references the exact objects
used for admission, never today's catalog representation.

**Acceptance criterion — the C17 twin suite is a hard Gate-0 test, not
a smoke test.** Positive: `security-review@1` + exact governed
composition + valid seal + narrowed quotas + exact `template_scope` →
ACCEPT. Negatives, each coupled to one locked clause and refused *for
its intended reason* (the test must name the responsible gate; "was
rejected" is insufficient — today's implementation rejects the
legitimate case and proves nothing): wrong skill · missing skill
allowlist · wrong workflow · wrong manifest member · wrong grant
template · widened numeric quota · changed structural grant field ·
unattributed procedure · wrong template scope · withdrawn Skill.
Mutation-coupled both ways: suppress a clause → its negative passes;
refuse unconditionally → the positive fails.

## 4. Assets inventory (factual, for the grill)

- **The defect (orchestrator.go `verifyAnchoredSkill`):** compares
  `env.Composition.Seal` (sha256 over nine members incl. per-task
  `grant`, `spec`) against `entry.Composition` (sha256 of `skill.json`
  bytes). Different domains; equal for no input. One caller, zero Go
  tests; Phase C row C17 asserts only refusal.
- **Catalog entry:** `{name, version, composition_sha256 =
  sha256(skill.json), manifest_path, state, steward}`; `Resolve(ref)`
  verifies manifest bytes two-way; append-only, rebind refused
  (D-L9-10).
- **Manifest (`skill.json`):** seven members pinned `{path, sha256}`:
  workflow, workflow_ceiling, context_contract, grant_template,
  spec_template, input_schema, procedure.
- **Envelope commitment (`CompositionCommitment`):** the same seven
  identities + `grant_sha256` (effective) + `spec_sha256` (effective)
  + `composition_sha256` (seal over all nine). L9 computes it in
  `sealedComposition`; L7 `checkSeal`s it and verifies every
  materialized artifact against it (`verify` at orchestrator.go:574,
  647, 859).
- **Origin (opaque, D-L9-13):** L9 stamps `skill` (`name@version`),
  `skill_composition` (= manifest hash = catalog `composition_sha256`),
  `skill_catalog` (catalog hash), `grant_template`, `grant_instantiated`,
  `spec_template`, … Envelope validator: any `skill`/`skill_*` key
  requires a commitment (D-L9-11d). `verifyAnchoredSkill` reads
  `origin["skill"]` — an opaque field used load-bearingly.
- **Instantiation (`instantiateGrantTemplate`):** substitutes
  `@task_id`, `@workspace`; quota overrides narrow within `[1,
  bound]`; unknown overrides refused. Spec template substitutes
  `@task_id`, `@repo`, `@pinned_sha`.
- **Anchor:** pins `skill_catalog` (authoritative for composition
  resolution), the `workflows[]` bundle set (workflow + ceiling +
  contract hashes), models, registries, constitutions. `SubmitTask`
  already refuses a workflow outside the anchored set (C14) and a
  bundle mismatch (C15) — one Skill = one workflow (D-L9-6), so the
  anchored workflow set already bounds which skills a deployment can
  run.
- **Unanchored deployments:** explicit test-harness `Unanchored` role;
  `verifyAnchoredSkill` runs only when anchored; seal + materialization
  checks (Claim 1) run always.
- **Residual C3 (D-L9-11a):** governance actor attestation via signed
  registration — recorded, not designed.

## 6. Skill Admission Drill — attack record (owner-led, 2026-09-22)

Format per attack: classification PASS / FAIL / AMEND / DEFER; attack ·
execution path · deterministic control · authoritative owner ·
evidence/witness · residual gap.

### A-SA-1 — Caller-supplied composition under a claimed skill (`skill=security-review@1`, `H2/W2/C2/G2`)

**Classification:** PASS → LOCK (owner). D-SA-2 locked; FAIL
under today's code for the wrong reason (refuses attacker and
legitimate submitter alike — the positive twin fails).

- *Premise correction:* `instructions = R1` is not a Skill member; the
  instruction roots are anchor-pinned deployment artifacts verified at
  Open. The Skill's only instruction artifact is the procedure `P1`.
- *Execution path:* `SubmitTask`, anchored, pre-record. (1) envelope
  validation — attribution requires a commitment (D-L9-11d);
  `checkSeal` accepts a consistently resealed `H2`, refuses an
  inconsistent one (`ErrInvariant`, pre-record). (2) G1 bundle check —
  `W2 ∉ anchored workflows` → refused (C14); if `W2` is another
  anchored workflow, C15 requires that bundle's ceiling/contract.
  (3) correspondence — resolve `security-review@1` in the anchor-pinned
  catalog → ACTIVE entry → manifest bytes hash to entry → seven pins vs
  seven commitment fields → `W2 ≠ W1` → refused. `H2` never consulted
  as authority. (4) materialization not reached.
- *Deterministic control:* **D-SA-2 (proposed):** admitted only if (i)
  seal self-consistent; (ii) `name@version` resolves in the anchor-
  pinned catalog to an ACTIVE entry whose manifest hashes to the entry;
  (iii) each of the seven Skill-fixed identities in the commitment
  EQUALS the manifest's pin; (iv) workflow bundle ∈ anchored set (G1);
  (v) effective grant/spec relate to their templates by the narrowing
  rule (Q-SA-6). The envelope's `composition_sha256` establishes
  self-consistency only; governance identity is established by the
  catalog's pins. "Invoking a skill" = invoking its registered
  composition; the seven fixed members are not parameters (D-L9-7
  Class 1); no field exists to carry "the workflow I want".
- *Authoritative owner:* Governance via the catalog manifest's pins;
  L7 applies equality. Envelope identities are claims.
- *Refusal vs invariant:* assembly refusal (`ErrAssembly`), no task, no
  record. Self-inconsistent seal = `ErrInvariant`, also pre-record;
  the two classes stay distinct (D-L9-11a proof-register consequence).
- *Evidence/witness:* none in L6 — pre-task refusals have no task to
  record into (true of every assembly refusal C1–C19). Observation,
  not a gap for this attack: nothing durable is forged.
- *Residual gap:* today's implementation cannot pass the positive twin;
  D-SA-2 (iii) is the correction; Gate 0 demands the positive path
  first.

### A-SA-2 — Composition substitution via a valid governed bundle (all of `incident-analysis@1`'s members under `skill=security-review@1`)

**Classification:** PASS by D-SA-2 (v). **Finding F-SA-1** on the
unattributed variant; **D-SA-3 proposed.**

- *Execution path:* envelope validation passes (attributed +
  self-consistent commitment); **G1 bundle check passes** (`W2` is an
  anchored workflow, `C2`/`CTX2` are its bundle) — G1 establishes "a
  governed bundle", never "the claimed skill's bundle"; correspondence
  (D-SA-2 v): `security-review@1` manifest pins `W1 C1 CTX1 G1 S1 I1
  P1` vs commitment `W2 … P2` → seven mismatches → `ErrAssembly`,
  pre-record.
- *Owner of the rule:* Governance, in the Skill manifest — D-L9-1's
  atomic composition invariant ("resolved as one governed set, cannot
  be independently mixed or substituted"). The manifest is one
  approval of a seven-tuple, not seven approvals. L9 enforces by
  construction (materializes from one resolved manifest); L7 enforces
  by equality at anchored assembly (D-SA-1).
- *Why "individually governed" is insufficient:* review judges the
  tuple and its properties are not additive — `G1` reviewed for `W1`'s
  phases with `CTX1`'s classes and `P1`'s procedure; `P1` written to
  drive `W1`'s lattice; `I2` validates inputs for `S2`'s spec; `CTX1`'s
  ceiling judged against `W1`'s purpose. Authority is a property of a
  reviewed relationship, never of an artifact (the C-L8-18 principle
  for evidence, applied to compositions). G1's C15 is the same rule one
  level down (anchored workflow + another bundle's ceiling/contract is
  refused though both are anchored). Attribution selects which tuple
  the equality runs against.
- *Refusal class:* `ErrAssembly` (governance mismatch), not integrity
  — the seal is consistent and every artifact is what it claims.
- *Evidence:* none in L6 (pre-task).
- *Residual gap — F-SA-1 (real):* an **unattributed** envelope (no
  `origin.skill`, no commitment — D-L9-11d's "ordinary hand-assembled
  envelope") may pair anchored bundle `W2/C2/CTX2` with a caller grant
  within `C2`, a caller spec, and **`skill_procedure_path = P1` (or any
  file) with the caller's own sha256**; L7 activates it as a
  **skill-scope instruction source** verified only against the
  caller's hash (`envelope.go`: pair travels together; activation at
  orchestrator.go ~858 skips commitment verification when `c == nil`).
  Capability cannot widen (grant ⊆ ceiling, tools ⊆ registry,
  protected roots unshadowable), but a submitter authors a
  trusted-scope instruction source under an anchored deployment — an
  L1 source that is neither a registered root nor a recognized payload.

**D-SA-3 (LOCKED by owner; answers Q-SA-8's unattributed-procedure half):** *under an anchored
deployment, a skill-scope instruction artifact enters only through a
skill-attributed envelope whose composition corresponds (D-SA-2); an
unattributed envelope carrying `skill_procedure_path` is refused at
assembly.* Unanchored deployments keep today's behaviour (the L9 seam
tests rely on it).

**A-SA-2 disposition (owner):** PASS. D-SA-3 LOCK; F-SA-1 CLOSED by
D-SA-3.

### A-SA-3 — Caller-widened `template_scope` with narrowed quotas (Q-SA-6)

**Classification:** FAIL under the architecture as locked through
D-SA-3 → **AMEND: D-SA-4 proposed.** Nothing refuses it today; the
`grant_authority` digest only makes it visible.

- *Execution path (D-SA-2 implemented):* seal self-consistent
  (`grant_template_sha256 = hash(G1)` true, `grant_sha256 =
  hash(G_effective)` true) → D-SA-2 (v) passes (the pinned template IS
  G1) → `grantWithinCeiling` passes → Claim 1 passes (the forged grant
  is exactly what was committed) → digest differs, visible, nothing
  refuses → at runtime L4 authorizes `delegate(incident-analysis@1)`
  against `G_effective`. **Attack succeeds.**
- *Why:* the commitment pins the template's identity and the effective
  grant's identity separately and honestly; the **relation** between
  them is never checked by the authoritative layer. Only L9's
  `instantiateGrantTemplate` enforces it, by construction, and L9 is
  non-authoritative (zero trust discount) — a hand-built envelope never
  calls it.
- *The relation `effective ⊑ template` ("instantiates"):* tools set
  equal (no add/remove; Class 1) · `max_calls` per tool `1 ≤ eff ≤
  tpl` (Class 2) · `total_max_calls` `1 ≤ eff ≤ tpl` (Class 2) ·
  `workspace` present iff template has `@workspace`, value = per-task
  binding (Class 3) · `mutating` equal (Class 1) · `themis_scope`
  set-equal (Class 1) · **`template_scope` set-EQUAL (Class 1; C-L8-16
  as locked in L8 — caller narrowing is a residual, not a
  permission)** · `task_id` = the task id where the template has
  `@task_id`. Spec likewise: `@task_id`/`@repo`/`@pinned_sha`
  substituted; `wall_deadline_s` in `[1, tpl]`; all else equal.
  `⊆` for `template_scope` is mandatory-but-insufficient: v1 = equality
  (numbers narrow, structure is fixed — the D-L9-7 line).
- *Digest visibility sufficient?* No — post-hoc detection with zero
  authority; admission must refuse. This is the Q-SA-6 lock C-L8-16
  waited for.

**D-SA-4 (LOCKED by owner):** *Under an anchored deployment, L7 at assembly
reads the grant template and spec template from the anchor-pinned
catalog at the manifest's pinned paths (confined; hash verified
against the pin), and refuses (`ErrAssembly`, pre-record) unless
`effective_grant ⊑ grant_template` and `effective_spec ⊑
spec_template` under the relation above. The relation is L4 grant
algebra (`tools.Instantiates(effective, template)`, the
`grantWithinCeiling` shape), not skill semantics; L9 applies the same
relation at instantiation, non-authoritatively.* Possible only where
required: the template bytes are reachable only through the pinned
catalog; unanchored deployments keep Claim 1 only. Order: seal →
D-SA-2 ii–iv → member equality → template read + `⊑` →
`grantWithinCeiling` (independent gate, D-L7-4) → registry → Claim 1.

**Twin tests:** positive = quotas narrowed, `template_scope` equal;
negatives = scope +1 entry, scope −1 entry, tool added, tool removed,
`mutating` flipped, `max_calls` > bound, `total` > bound — each
mutation-coupled to one clause of `⊑`.

**A-SA-3 disposition (owner):** PASS → LOCK. D-SA-4 LOCK; Q-SA-6
CLOSED; `template_scope` relation = equality; digest visibility alone
insufficient; positive-path twin mandatory.

### A-SA-4 — Grant template substitution (`grant_template_sha256 = B` = G2 under `skill=security-review@1`)

**Classification:** PASS — refused by D-SA-2 (v) before D-SA-4 runs
(A-SA-2 at single-member granularity). Adds the **reference-source
rule** to D-SA-4 (clarification, folded above).

- *Where `security-review@1 → G1` is established:* the Governance
  manifest's `grant_template` pin, enforced by D-SA-2 (v) equality.
  "G2 is a legitimate Governance template" is true and irrelevant: the
  anchor pins the catalog and workflow bundles, not free-standing
  templates; a template has no governance identity outside the tuple
  it was reviewed in (D-L9-1).
- *Covered by D-SA-2 or extra check?* Covered for identity; D-SA-4
  checks a different pair. Three values: `A` (manifest pin,
  Governance), `B` (commitment claim, caller), `G_eff` bytes (caller).
  D-SA-2 (v): `B == A`. D-SA-4: `G_eff ⊑ bytes(pin A)` — the template
  read from the catalog at the manifest's pinned path, hash-verified.
  Never `G_eff ⊑ bytes(claim B)`.
- *Execution path:* seal ok → catalog → manifest → D-SA-2 (v) `B ≠ A`
  → `ErrAssembly`, pre-record. D-SA-4/ceiling/materialization not
  reached; identity precedes relation by design.
- *Owner:* Governance manifest; L7 equality; L4 algebra vs the
  manifest-resolved template.
- *Evidence:* none (pre-task). *Residual:* none. Gate 0 test: a
  mutation making D-SA-4 resolve its reference from the claim must
  fail when `B` is a genuine other-skill template and `G_eff` a valid
  instantiation of it.

**A-SA-4 disposition (owner):** PASS → CLOSED. Q-SA-4 CLOSED, Q-SA-5
CLOSED.

**Two levels of correspondence (owner, Q-SA-5):** D-SA-2 (v) =
governance identity, `claimed member == manifest-pinned member` for
each Skill-fixed member independently; D-SA-4 = instantiation
correspondence, `effective ⊑ manifest-pinned template` under the
field-specific relation. The seven per-member checks are NOT replaced
by one derived composition-hash comparison: per-member checks make
failure semantics and mutation coverage explicit; the composition hash
establishes the manifest's atomic identity.

**Invariant (owner):** *Caller-supplied identities can establish
self-consistency; Governance-pinned identities establish authority;
deterministic relations establish whether caller-shaped artifacts are
legitimate instantiations of those Governance artifacts.*

### A-SA-5 — `origin["skill"]` as a load-bearing authority claim (Q-SA-2/Q-SA-3)

**Classification:** attack PASS (Case B refused by D-SA-3); question
**AMEND — D-SA-5 proposed** (the discriminator lives in a field
declared opaque).

- *Why the argument fails:* "already governed artifacts" = the A-SA-2
  fallacy at tuple level — governed as members of M1, not as files.
  Case B has the same seven byte strings and none of the relationship;
  the attribution is the claim that selects the governance relationship
  under which the seven are admitted as one tuple. Hash integrity ≠
  governance provenance (D-SA-3).
- *1. Metadata or discriminator:* the **claimed skill identity** is a
  load-bearing selector (selection authority only, like L8's
  `template` argument: a false claim can only refuse or be true);
  **`origin`** is opaque attribution (D-L9-13). A selector cannot live
  in an opaque map (the H-1 review already had to gate every `skill_*`
  key). → D-SA-5.
- *2.* No (D-SA-3). *3. Invariant:* skill-scope source activated only
  from a procedure committed in a corresponding composition (D-SA-2 v);
  `skill_procedure_path`/`sha256` are locators for committed bytes,
  never a source declaration; *skill-scope activation ⇔ Skill
  attribution ⇔ corresponding commitment*. L1 stays mechanism-only;
  the gate is L7's; wall test: sole `ActivateSkillSource` call site is
  in the corresponded path. *4.* Under anchors `skill_procedure_path`
  is a composition locator, not a caller-addressable instruction
  source; unanchored keeps today's affordance (sentinel-marked
  records). *5.* Anchored → `ErrAssembly`, pre-record; unanchored →
  existing behaviour.
- *Execution path:* Case B anchored → refused at envelope validation
  before catalog/bundle/task; Case A → `skill` selects M1 → D-SA-2 (v)
  incl. procedure → D-SA-4 → ceiling → materialization →
  `ActivateSkillSource(P1, pin)` verified against the commitment.

**D-SA-5 (LOCKED by owner; folded into §2):** *the claimed skill identity is a first-class,
schema-validated envelope field `skill = name@version` with exactly one
reader (the anchored correspondence check); `origin` returns to opaque
attribution; L9 stamps both. Coherence matrix (extends D-L9-11d):*

| `skill` | `origin.skill*` | commitment | `skill_procedure_path` | anchored |
|---|---|---|---|---|
| absent | absent | absent | absent | valid ordinary envelope |
| absent | absent | absent | present | **refuse** (D-SA-3) |
| absent | absent | present | absent | permitted; Claim 1 only, no authority |
| absent | absent | present | present | **refuse** — skill scope needs correspondence, correspondence needs attribution |
| present | any | absent | any | refuse (D-L9-11d) |
| present | present ≠ `skill` | present | any | refuse — attribution incoherent |
| present | any | present | iff manifest pins one | correspondence path D-SA-2 → D-SA-4 |

Record keeps two independently derived values (`skill` used by
admission; `origin` as L9 said) so `TestAttributionInconsistencyIsDetectable`
retains meaning (D-L10-10).

**A-SA-5 disposition (owner):** PASS → CLOSED. D-SA-5 LOCK; F-SA-1
CLOSED; Q-SA-2 and Q-SA-3 resolved. Next: the seal itself (Q-SA-7).

### A-SA-6 — What the seal proves (Q-SA-7)

**Classification:** A, B, C all PASS. **D-SA-6 proposed** (LOCK, no
amendment).

| Attack | Rejected by | Class | Without the seal | Seal's contribution |
|---|---|---|---|---|
| A. forged seal, correct members | `checkSeal` at envelope validation, pre-catalog, pre-record | `ErrInvariant` | not protected — and correctly: D-SA-2 would run the task *correctly*; the seal refuses a harmless but malformed submission | envelope-contract coherence: never record an executed-composition identity that mismatches its members (refuse before, not DISCREPANCY after) |
| B. valid seal, wrong composition | D-SA-2 (v) | `ErrAssembly` | protected | nothing (correctly) |
| C. committed `X`, bytes later `Y` | Claim 1 per-artifact `verify` at activation (D-L9-11: before executable; no re-read path after; R-L9-2 bytes run) | `ErrInvariant` | protected (hash-vs-bytes, not hash-vs-hash) | makes nine verifications one claim |

**Answers:** 1. authority — no. 2. Governance identity — no ("never
'the composition hash authenticates the Skill'", D-L9-11a). 3.
internal consistency — yes, plus: **the identity of the executed
composition instance.**

**Why necessary, not defensive:** (1) instance identity for the record
— `H1` names the registration, identical for every task running the
skill; the seal names *this task's* composition (effective grant/spec
differ per task): Merkle root of the instance vs Merkle root of the
registration, the two roots the original defect confused; (2)
atomicity — D-L9-11b survives D-SA-2: without a self-hash the
commitment is nine loose claims and "compare the instance to the
registration" is not well-formed; (3) the unanchored case — no
catalog, D-SA-2/4 do not run; seal + per-artifact verification are the
only binding (Claim 1 alone).

**D-SA-6 (LOCKED by owner; folded into §2):** *The composition seal establishes internal
consistency of the submitted commitment and serves as the identity of
the executed composition instance. It establishes neither authority
nor Governance identity. Its only consumers are `checkSeal`
(self-consistency; `ErrInvariant`, pre-record) and the L6 task record
(executed-instance identity for reconstruction). It is never compared
against a catalog, manifest, or anchor value. Governance
correspondence is D-SA-2's alone; instantiation legitimacy is
D-SA-4's alone; artifact integrity is the per-artifact verification's
alone. Retained as necessary for instance identity, claim atomicity,
and the unanchored case — not as a defensive redundancy for D-SA-2.*

**A-SA-6 disposition (owner):** A, B, C PASS. D-SA-6 LOCK; Q-SA-7
CLOSED.

### A-SA-7 — Unanchored admission (Q-SA-8, remaining half)

**Classification:** A PASS; B PASS (not an attack on governed
execution; it is the unanchored contract). **No architecture
amendment — explicit statement of the existing contract (D-SA-7
proposed). Finding F-SA-2 DEFERRED to L11.**

Facts: task record stamps `deployment_anchor = <hash> | "unanchored"`
(close-review MEDIUM-1: distinguishable from a dropped key);
`VerifyAnchorRecord` refuses the sentinel ("no deployment authority to
verify"); `themis-run` never sets `Unanchored` (production surface
cannot open one); `Open` refuses anchor + `Unanchored` together. L11
witness/grounding/package code never reads the sentinel.

1. *Legitimate mode:* yes — an explicitly declared test-harness role
   (G1), never a degraded production mode; the L9 seam/e2e tests depend
   on it. Skill-shaped, not Skill-governed.
2. *Seal authority:* none; Claim 1 only (instance binding + integrity)
   — and in this mode the only binding (D-SA-6).
3. *Arbitrary artifacts:* yes — the unanchored caller IS the
   deployment (what owner finding 1 forbids in production is what a
   test harness must do), bounded by every deterministic control
   below admission (grant ⊆ ceiling, tools ⊆ registry, workflow under
   ceiling, L4, L5, L6, floors). Unanchored = no Governance
   correspondence, never no controls.
4. *D-SA-6 implication:* exactly — unanchored establishes Claim 1 and
   can never establish Claim 2.
5. *What prevents confusion:* sentinel · `VerifyAnchorRecord` refusal ·
   production surface cannot open unanchored · `Open` refuses
   ambiguity. **Gap F-SA-2:** L11 witnesses/packages do not carry the
   sentinel and grounding does not check it → an unanchored task's
   records can ground a comparison package that reaches a Governance
   door unseen. A *consumption* gap, not an admission gap (admission's
   obligation — an unforgeable, refusal-backed sentinel — is met).
   **DEFER to L11 (D-L11-15):** the witness records `deployment_anchor`
   for every cited task; the consumption rule classifies `unanchored`
   evidence explicitly — admissible only where a criterion says so,
   never by default for promotion evidence.

**D-SA-7 (LOCKED by owner; folded into §2):** *Unanchored execution is an explicitly declared
test-harness role, unreachable from the production surface. It
establishes Claim 1 only — the task executed exactly the composition it
submitted, under every deterministic control below admission. It can
never establish Claim 2: no Governance correspondence is performed and
none may be inferred. Its record carries `deployment_anchor =
"unanchored"`, and no deployment authority may be re-established from
such a record. Under this role D-SA-2, D-SA-4, and the anchored rows of
D-SA-3/D-SA-5 do not run; the caller composes the deployment.*

**A-SA-7 disposition (owner):** A PASS, B PASS. D-SA-7 LOCK; Q-SA-8
CLOSED; F-SA-2 DEFER → L11. Status: D-SA-1..7 LOCK; Q-SA-1..8 CLOSED;
Q-SA-9..12 OPEN.

### A-SA-8 — Withdrawn skill (Q-SA-9)

**Classification:** A (forward-only) PASS — the model. B (retroactive)
prohibited by construction. C (lookup-only) impossible. **No
amendment; explicit clarification — D-SA-8 proposed.**

Code facts: `Resolve` refuses withdrawn ("new instantiation is
refused; historical records remain interpretable"); `CheckAppendOnly`
refuses un-withdrawal ("active→withdrawn only").

- *Caller's argument:* `ACTIVE` is not a recommendation attached to
  bytes; it is the Governance disposition of the identity. Bytes were
  never what admitted a task; the relationship was.
- *1. New admission:* refused at D-SA-2 (iii), `ErrAssembly`,
  pre-record; L9 likewise, non-authoritatively.
- *2. Running tasks:* unaffected twice — D-L9-11 (resolved once, no
  arrow back to the catalog) and G1 (catalog anchor-pinned; a
  withdrawal changes the catalog hash → not visible within one `Open`;
  adopted only by new anchor + restart).
- *3. Bypass by supplying the old manifest:* impossible anchored — the
  caller supplies no manifest (D-SA-4 reference-source rule); only
  `skill = name@version`, and the catalog answers WITHDRAWN; a
  different catalog file fails the `skill_catalog` pin. Unanchored:
  the caller composes the deployment (D-SA-7); question moot.
- *4. Withdrawn after instantiation, before execution:* same running
  deployment → frozen catalog still ACTIVE → admitted; the record
  names that anchor (correct). Across a new anchor + restart →
  refused; a created-but-non-terminal task is driven to
  FAILED_PARTIAL at startup (D-L7-8) — no task straddles a withdrawal.
- *5. Owner:* Governance, by append-only catalog act, adopted only via
  a new anchor. L7/L9 never withdraw; a running deployment never
  observes one mid-life.
- *6. Property of:* the Governance identity `name@version`, not the
  immutable artifact (state lives on the entry) — which is what makes
  B impossible.
- *7.* Clarification: every piece exists (append-only state, `Resolve`
  refusal, D-L9-11, G1 adoption, D-L7-8, D-SA-4); the composition is
  what defeats C.

**D-SA-8 (LOCKED by owner; folded into §2):** *Withdrawal is a forward-only Governance act on
the identity `name@version`, recorded append-only (`active →
withdrawn` only, never reversed, never removed). It refuses all new
admission and instantiation of that identity; it never affects running
tasks (immutable w.r.t. their Skill; no catalog change observable
within one `Open`); it never affects historical records, which
reconstruct from their own durable closure and their anchor as
admitted. A deployment adopts a withdrawal only through a new anchor
and a restart, which closes every non-terminal task before the new
catalog governs anything. No caller can supply a manifest or catalog
under an anchored deployment, so withdrawal cannot be bypassed by
supplying the old composition.*

**Reconstruction semantics:** a historical task under a withdrawn
skill reconstructs CONFIRMED against its stored bytes and recorded
anchor; current catalog state is not a reconstruction input (reporting
it is re-evaluation, D-L10-12). Retroactive doubt is evidence for
humans (D-L9-16), never a record rewrite.

**A-SA-8 disposition (owner):** A PASS / locked model; B prohibited; C
impossible. D-SA-8 LOCK; Q-SA-9 CLOSED; no new gap. Open: Q-SA-10,
Q-SA-11, Q-SA-12.

### A-SA-9 — Catalog membership vs deployment admissibility (Q-SA-10)

**Classification:** attack PASS (refused at G1 C14 — the skill's
bundle is not anchored — before D-SA-2 runs). Question **AMEND,
additive: D-SA-9 proposed** (skill-granularity anchor allowlist).

Fact: the anchor's `workflows[]` already IS a per-deployment allowlist
at **bundle** granularity (C14/C15; one Skill = one workflow,
D-L9-6). Insufficient only when two skills legitimately share a
bundle (same workflow bytes, different procedure/grant/schema — the
reuse pattern C-L8-15 blesses for templates).

- *Caller's argument:* D-G1-1A at skill level — registration
  identifies, the anchor admits (as for models: "a model enters a
  deployment only by Governance act").
- *Three facts, three owners, none implying another:* catalog
  membership (identity registered + ACTIVE; global; D-SA-8) ·
  deployment admissibility (may THIS deployment run it; anchor; G1) ·
  skill instantiation (is THIS envelope a legitimate instance;
  D-SA-2/4/5).
- 1. sufficient? no. 2. narrower than catalog? yes — `workflows[]`
  today; `skills[]` proposed. 3. owner: the Deployment Anchor
  (Governance-admitted deployment artifact); L7 applies. 4. identity
  stays ACTIVE globally; membership is orthogonal (inverted case
  changes only membership; other deployments untouched). 5. bypass
  via another ACTIVE skill: admitted → legitimate use requiring THAT
  skill's composition (A-SA-2 refuses tuple-swapping); not admitted →
  refused. 6. allowlist change = anchor change → new anchor + restart;
  running tasks frozen; non-terminal closed (D-L7-8); old-anchor
  instantiations refused after adoption (D-SA-8 §4). 7. anchor
  artifact, never runtime policy (owner finding 4). 8. clarification
  of G1 + one additive G1 pin.

**D-SA-9 (LOCKED by owner; folded into §2):**
*Deployment admissibility of a Skill is a property of the Deployment
Anchor, never of the Governance catalog and never of runtime policy.
The anchor pins `skills[]` — the exact `name@version` set this
deployment may run — alongside `workflows[]`. At anchored `SubmitTask`,
`skill ∈ anchor.skills` is checked by exact equality before catalog
resolution, mirroring the model allowlist; absence of the field admits
no skill-attributed task. Admission then continues: bundle ∈ anchored
set (G1) → catalog resolution and correspondence (D-SA-2) →
instantiation (D-SA-4). Catalog membership establishes identity; the
anchor establishes admissibility; the envelope establishes
instantiation; none implies another. Allowlist changes take effect
only through a new anchor and restart.* Refusal `ErrAssembly`,
pre-record; twin tests distinguish the allowlist message from the
correspondence message.

### D-SA-10 — The Deployment Anchor selects; it never composes (LOCKED 2026-09-22, owner; closes Q-SA-11)

> The Deployment Anchor selects and admits Governance-registered
> artifacts; it never authors, edits, specializes, or overrides a
> Governance artifact or Skill member. Every Anchor field is a hash
> pin or exact allowlist over governed artifacts. Deployment-level
> narrowing may constrain execution through deployment-governed
> mechanisms (execution ceiling, workflow ceiling, tool registry, model
> allowlist, Skill allowlist) but may not mutate a Skill's governed
> composition. A deployment-specific Skill variant is a new Governance
> identity `name@version`, registered through the Skill catalog and
> subsequently admitted by the Deployment Anchor.

```
Governance composes security-review@1 {W1 C1 CTX1 G1 S1 I1 P1}
   → Deployment Anchor selects/admits (skills[] + workflows[] + ceilings + registries)
   ✗ Deployment Anchor modifies composition (P1→P2, G1→G2) — would be a second Governance registry
```

Three complementary rejection layers: (1) anchor admission — closed
schema rejects `skill_overrides` (strongest practical gate); (2)
D-SA-2 — correspondence always against the catalog manifest, never an
Anchor-supplied replacement; (3) D-L9-10 — a changed composition is a
changed identity; `name@version` cannot rebind. **Uniform narrowing
(deployment ceiling → all Skills' governed grants bounded) is allowed;
per-Skill mutation is composition mutation, not deployment admission.**

**A-SA-9 disposition (owner):** PASS. D-SA-9 LOCK; Q-SA-10 CLOSED.

### A-SA-10 — Anchor authority over Skill composition (Q-SA-11)

**Classification:** PASS — direct consequence of G1 + D-SA-2 + D-SA-9;
no amendment. **D-SA-10 proposed** as a clarification lock.

Fact: `ParseAnchor` uses `DisallowUnknownFields` and refuses trailing
content — an anchor with `skill_overrides` cannot be parsed, proposed,
admitted, or opened.

- *The error:* "anchor-controlled allowlist ⇒ anchor controls the
  Skill" confuses two verbs. Every anchor field is a pin or an
  allowlist over Governance-registered artifacts; the anchor
  **selects**, never **authors**. `skill_overrides` would make the
  anchor a second, unreviewed Skill catalog — competing security
  truth; contradicts the D-SA-2 ownership table.
- *Rejected at:* (1) anchor admission — unknown field, parse refusal;
  (2) D-SA-2 (v) had it existed — L7 compares against the catalog
  manifest, never anything anchor-supplied (D-SA-4 reference-source
  rule generalized: the anchor pins the catalog, it does not replace
  its contents); `P2 ≠ P1` → `ErrAssembly`; (3) D-L9-10 — a
  composition with `P2/G2` has a different manifest hash and is not
  `security-review@1` by definition.
- 1. overridable fields: none. 2. narrowing beyond D-SA-4: not per
  skill — the anchor narrows every skill **uniformly** through
  deployment-governed Class 4 inputs (execution ceiling, workflow
  ceiling, registry, model/skill allowlists; D-L9-11c kept these
  outside the composition); a per-skill quota edit = grant-template
  edit = new version. 3. replace structural members: no (A-SA-2,
  A-SA-4). 4. variant = new `name@version`, registered by Governance,
  then admitted by `skills[]` — author ≠ governance ≠ machinery. 5.
  consequence, not amendment.

**D-SA-10 (LOCKED by owner; folded into §2):** *The Deployment Anchor selects;
it never composes. Every anchor field is a hash pin or an exact
allowlist over Governance-registered artifacts; no anchor field
authors, edits, specializes, or overrides any Governance artifact or
Skill member. Deployment-level narrowing acts uniformly through
deployment-governed inputs (execution ceiling, workflow ceiling,
registry, model and skill allowlists), never per skill. A
deployment-specific Skill variant is a new `name@version` registered
by Governance and then admitted by the anchor.* Twins: unknown
top-level anchor field → parse refusal; anchor admitting a registered
`@2` admits a genuine `@2` instance and refuses a `@1` envelope
carrying `@2`'s members.

**A-SA-10 disposition (owner):** PASS. D-SA-10 LOCK; Q-SA-11 CLOSED;
anchor schema stays closed; `skills[]` is an allowlist, not a
composition mechanism. Remaining: Q-SA-12.

### A-SA-11 — The positive twin (Q-SA-12)

**Classification:** architecture **PASS**; implementation **FAIL as an
explicit implementation gap** (the twin is refused today at
`verifyAnchoredSkill`). No decision changes; every gap is conformance
work under a locked decision.

| # | Question | Where | Code today |
|---|---|---|---|
| 1 | field selecting the Skill | first-class `skill` field (D-SA-5), read before catalog resolution | MISSING (reads `origin["skill"]`) |
| 2 | authoritative composition | catalog manifest for the entry, from the anchor-pinned catalog, bytes hash-verified | EXISTS (`Resolve` two-way; catalog hash vs anchor first) |
| 3 | seal | `LoadEnvelope` → `checkSeal`, top of `SubmitTask`, before any anchor/catalog read; `ErrInvariant` | EXISTS |
| 4 | D-SA-2 seven members | anchored block after the bundle check; equality per member; `ErrAssembly` | MISSING (`Seal != entry.Composition` refuses everything) |
| 5 | D-SA-4 template source | manifest `grant_template`/`spec_template` pins read from the catalog root (confined, hash-verified) → `tools.Instantiates` | MISSING (envelope has no template path; only the catalog route exists) |
| 6 | narrow vs equal | quotas/total/deadline ≤; tool set, `mutating`, `themis_scope`, `template_scope` equal; `workspace`/`task_id` governed bindings | defined (D-SA-4), MISSING with 5 |
| 7 | `skills[]` | first anchored gate after instruction plane + registry; exact; allowlist message | MISSING (no anchor field) |
| 8 | `workflows[]` | existing bundle gate + C15 | EXISTS |
| 9 | materialization condition | all gates passed AND Claim 1 per artifact (workflow/ceiling/contract/spec at assembly; grant at binding; procedure at activation) → exact bytes stored, record created | EXISTS (orchestrator.go:574, 647, 859) |
| 10 | final evidence | governed hashes (`envelope`, artifacts, `grant_envelope/effective/authority`, `origin:*`, `deployment_anchor`) + stored artifact and anchor bytes | EXISTS; **Claim 2 byte gap**: consumed catalog + manifest bytes not stored |

**Refusal ordering (each negative fails for its intended reason):**
seal (`ErrInvariant`) → instruction plane + registry → `skills[]` →
`workflows[]` → catalog hash → resolve/ACTIVE/manifest integrity →
D-SA-2 members → template read + D-SA-4 → `grantWithinCeiling` →
Claim 1 per artifact (`ErrInvariant`) → record. Twin suite asserts
message class, not mere refusal (C17 lesson).

**Claim 2 evidence completeness (implementation, per R-L9-2 /
D-L10-10):** store the consumed catalog and manifest bytes as
content-addressed objects (dedup across tasks), referenced from the
record as `skill_catalog` / `skill_manifest`; record `skill` as a
governed key. Reconstruction of "ran the registered composition" must
not depend on the current catalog.

**Implementation gaps (whitelist for the change):** `skill` field
(D-SA-5) · `skills[]` anchor pin (D-SA-9) · per-member correspondence
(D-SA-2) · template resolution + `Instantiates` (D-SA-4) · anchored
refusal of unattributed procedures (D-SA-3) · Claim 2 byte
completeness · the twin suite as acceptance criterion: the exact
positive envelope admitted under `rsys@4`; every A-SA-1..10 negative
refused with its intended message; every clause of D-SA-2 (v), D-SA-4,
D-SA-5's matrix, D-SA-9's gate mutation-coupled to exactly one
negative.

## 7. Closure record (owner judgment, 2026-09-22)

| Decision | Status |
|---|---|
| D-SA-1 ownership of the correspondence check | LOCK |
| D-SA-2 Skill/composition correspondence | LOCK |
| D-SA-3 skill-scope instruction provenance | LOCK |
| D-SA-4 effective grant instantiation (`template_scope` equality) | LOCK |
| D-SA-5 first-class Skill admission identity | LOCK |
| D-SA-6 composition seal (two consumers) | LOCK |
| D-SA-7 unanchored execution contract | LOCK |
| D-SA-8 forward-only withdrawal | LOCK |
| D-SA-9 deployment-level Skill admissibility (`skills[]`) | LOCK |
| D-SA-10 the anchor selects, never composes | LOCK |
| Q-SA-1 → Q-SA-12 | CLOSED |
| F-SA-1 | CLOSED by D-SA-3 |
| F-SA-2 | DEFER → L11 (witness carries `deployment_anchor`; consumption classifies `unanchored`) |
| positive implementation path | NOT YET IMPLEMENTED |

**Overall:** Architecture CLOSED / LOCKED. Implementation NOT
CONFORMANT YET. No category-4 architecture gap identified by Q-SA-12
or the drill. Remaining work is implementation/conformance against the
locked decisions plus the deferred L11 finding.

**The invariant the drill produced (owner, A-SA-4):** *Caller-supplied
identities can establish self-consistency; Governance-pinned
identities establish authority; deterministic relations establish
whether caller-shaped artifacts are legitimate instantiations of those
Governance artifacts.* And its corollary (A-SA-7): *anchored and
unanchored are two different admission contracts, not two confidence
levels of the same contract.*

**Discharges:** the L8 dependency C-L8-16 / Q-SA-6 (`template_scope`
fixed-by-skill, equality-checked — D-SA-4); GitHub issue #1 is decided
(not fixed): implementation proceeds under this record.
