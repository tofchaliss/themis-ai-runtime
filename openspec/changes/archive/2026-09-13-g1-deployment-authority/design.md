# Design: G1 — Deployment Authority Anchoring

Grill OPEN 2026-09-13 → **CLOSED 2026-09-13: D-G1-1 LOCKED with
owner amendment D-G1-1A** (anchor admission). Source gap:
integration-audit synthesis G1.

## D-G1-1 — The Deployment Anchor (LOCKED 2026-09-13; owner
amendment D-G1-1A folded)

One governed artifact — the Deployment Anchor — pins the
deployment's authoritative artifact set by content hash, consumed
fail-closed at task assembly. Not a new authority class: the fourth
application of the proven Governance registration pattern.

**Q-G1-1 (owner LOCK):** Governance owns deployment identity and
admission; L7 enforces; the submitter owns neither.

**Q-G1-2 (owner LOCK):** the authoritative set is a CLOSED
enumeration, each entry an exact content hash: L4 tool registry;
grant ceiling(s); exec ceiling; workflow ceiling; skill catalog;
L10 contract registry; L11 criteria and regression-set registries;
instruction roots (safety/system/themis) as content hashes; the
MODEL ALLOWLIST (models.json becomes anchor-governed — an external
endpoint can enter a deployment only by Governance act; closes half
the R3 coupling); the door table. Closed schema; nothing defaulted;
unknown fields refused. (Owner: the model allowlist matters because
otherwise the deployment is governed everywhere except where the
model itself is selected; the door table belongs because it
establishes which governed mechanisms constitute the deployment.)

**Q-G1-3 (owner AMEND → D-G1-1A below):** anchor bytes live at
policies/deployment/anchor artifacts; the ACTIVE-registration
relationship is explicit in the Governance plane
(policies/deployment/anchors.json, registry format), entering by
*.proposed.* → owner act → ACTIVE.

**D-G1-1A — Deployment Anchor admission (owner amendment,
constitutional):** *a Deployment Anchor is executable as deployment
authority only when its exact bytes and content hash resolve to a
Governance-ACTIVE Deployment Anchor. A caller-supplied anchor path
or hash may IDENTIFY the requested deployment; it can never
establish its governed status.* Sequence at Open: resolve exact
anchor (bytes unavailable → refusal; hash mismatch → refusal) →
establish Governance admission against the anchors registry
(unregistered → refusal; withdrawn → refusal) → ACTIVE anchor →
immutable Open. Then SubmitTask: bundle artifacts hash-matched
against the ADMITTED anchor → assembly. Two distinct steps, never
collapsed: anchor admission answers "is this a Governance-approved
deployment definition?"; bundle validation answers "does this
submission conform to that definition?" Without this, an operator
manufactures an anchor pinning any artifact set, supplies its own
hash, and every bundle check passes — the bundle is unforged but
the anchor is forged: exactly the C-1 trust-error class (a caller
may identify evidence or configuration; the governed owner must
establish its authority).

**Q-G1-4 (owner LOCK, admission-before-consumption):** Open =
resolve/admit anchor + verify expected operator-supplied hash
(consumption pin) + freeze; SubmitTask = verify bundle ⊆ admitted
anchor. One anchor per Open; no live mutation.

**Q-G1-5 (owner LOCK):** the submitter chooses a task WITHIN the
deployment (workflow among anchored workflows, payload, skill refs
via the anchored catalog); the submitter does not define the
deployment. Ceilings, registries, instruction roots, model list are
structurally out of the submitter's hands.

**Q-G1-6 (owner LOCK):** the anchor's content hash is the
authoritative deployment identity; the version field is
human/reference identity only.

**Q-G1-7 (owner LOCK after amendment):** the assembly record
carries the anchor identity/hash, the admitted-anchor
evidence/reference, and the artifact hashes checked — so replay and
reconstruction establish "executed under this ADMITTED Deployment
Anchor", not merely "under a consistent bundle".

**Q-G1-8 (owner LOCK):** supersession = new anchor version via the
proposed→act discipline; prior anchors retained (append-only,
permanently interpretable); adoption by restart only.

**Q-G1-9 (owner LOCK):** bundle/anchor disagreement → typed
refusal, recorded. No closest match, no partial bundle, no
unpinned fallback; an absent/unreadable/unadmitted anchor fails
Open itself.

**Q-G1-10 (owner LOCK with the admission clarification):** no new
authority store — one more governed registry-format artifact under
the existing Governance act discipline, with the same walls
(read-only machinery, no write API, two-way identity where
applicable, AST-audited).

**Residual (owner-accepted, deliberately outside G1):** submitter
authentication. G1 establishes WHAT deployment is authorized, not
WHO authenticated the caller requesting it. Caller origin is
recorded-not-authenticated (the D-L6-4 structural-trust position),
pointing at the L8/deployment boundary. Owner tightening: *caller
origin may be recorded, but origin/authentication status must never
be interpreted as evidence of Governance authority* — the residual
itself must not become a trust channel.

**Constitutional sentence:** *a caller may identify the deployment
it requests; only Governance admission makes an anchor authority,
and only the admitted anchor makes a bundle governed.*

## Implementation note

Implementation follows the audit's defect pass and its own Class-3
review; the anchor loader reuses the proven registry patterns
(readGoverned, duplicate-key wall, two-way identity, append-only
check, consumption pin). Enforcement lands in
orchestration.Open/SubmitTask. No production wiring of SubmitTask
before the anchor enforcement exists.

## Implementation record (2026-09-13)

Realized in `src/harness/deployment/` (anchor schema + admission,
read-only, no write API) and the L7 seam
(`orchestration/orchestrator.go`):

- **Open** — when `Config.AnchorPath` is set: `AdmitAnchor` resolves
  the exact anchor bytes, verifies the operator's expected hash,
  then ESTABLISHES admission against the Governance-active anchors
  registry (unregistered → refusal naming the D-G1-1A rule;
  withdrawn → refusal; two-way identity checked). Only then are the
  instruction roots + policy verified against the ADMITTED anchor's
  pins (HashDir/HashFile), the themis root required, and the anchor
  frozen for the orchestrator lifetime (Q-G1-8: adoption by restart
  only).
- **SubmitTask** — every governing artifact must BE the anchored
  one: tool registry, workflow ceiling, exec ceiling, context
  contract by hash; workflow within the anchored set; envelope model
  within the anchored allowlist ("a model enters a deployment only
  by Governance act"). Fail closed, first mismatch refuses, no
  partial bundle (Q-G1-5/9).
- **Record** — `governed["deployment_anchor"]` carries the admitted
  anchor's hash into the task's governed-hash set, so replay and
  reconstruction establish "executed under this ADMITTED anchor"
  (Q-G1-7).
- **Unanchored mode** remains legal ONLY for the recorded
  test-harness caller role; production wiring requires the anchor.

Proofs: `deployment/anchor_test.go` (admission incl. the
forged-anchor attack — self-consistent bytes + correct self-computed
hash + no registration → refused with "identifier, never an
admission claim"; withdrawn; two-way identity; duplicate
registration; wrong registry kind; 9 schema refusals; HashDir
determinism) and `orchestration/verification_seam_test.go`
(TestAnchoredOpen: admitted open, unregistered refusal, instruction
root drift refusal, themis root required; TestAnchoredSubmit
RefusesUnanchoredBundle: bundle-artifact and model-allowlist
refusals).

### Class-3 close review + remediation (2026-09-13)

Two parallel Class-3 reviews (security; architecture + test
evidence) against the first implementation. **D-G1-1A's core held
in both**: the two-step is not collapsed, a forged anchor alone is
genuinely refused, admission requires registry resolution, the
freeze is sound, and the package is read-only with no cycle. The
findings were all about what the admitted anchor BINDS. Remediated:

- **CRITICAL (symlink bypass)** — HashDir skipped non-regular
  entries while the instruction loader follows symlinks: hashing and
  reading disagreed about the file set, so a symlink dropped into a
  pinned root left the pin unchanged while unpinned out-of-tree text
  reached the model. HashDir now REFUSES non-regular entries
  (skipping and reading must never disagree). Proof:
  TestHashDirRefusesNonRegularEntries.
- **CRITICAL (consume-before-admit)** — Open resolved and rendered
  the instruction set BEFORE admission, and verified afterwards
  through an independent second read (the R-L9-2 shape). Admission
  and pin verification now run BEFORE `instructions.Resolve`.
- **HIGH (per-task drift)** — resolveTaskEIS re-walks the roots
  every task; the anchor's instruction claim was a one-shot startup
  assertion. verifyAnchoredInstructionPlane now runs per task too.
- **HIGH (model plane)** — the allowlist governed a NAME while
  name→endpoint resolution lived in an unpinned models.json. The
  anchor gains `model_registry` (a pin, or the explicit declaration
  `"absent"`); Open verifies it. Q-G1-2's endpoint clause is now
  implemented, not merely stated. Proof: TestAnchoredModelRegistryPin.
- **HIGH (dead allowlist test)** — the model-allowlist "proof"
  refused at the exec ceiling and never reached the check (verified
  survivor: deleting the whole allowlist loop left the package
  green). Fixtures now satisfy every bundle pin first; the
  workflow-set and instruction-policy survivors got dedicated tests
  too; and a POSITIVE anchored path (assembly → completed walk) now
  exists at both the L7 and Phase C levels.
- **MEDIUM (silent unanchored bypass)** — an anchorless Open now
  REFUSES unless `Config.Unanchored` explicitly declares the
  recorded test-harness caller role; declaring both is a refusal;
  unanchored records carry `deployment_anchor: "unanchored"` so
  absence is never ambiguous. Phase C, the flagship end-to-end
  register, now runs ANCHORED.
- **MEDIUM (Q-G1-7 evidence)** — the anchor BYTES are now durable
  in the task record, not only the identifier.
- **MEDIUM (framing collision)** — HashDir is length-prefixed;
  `{a:"", b:"c"}` and `{a:"\0b\0c"}` no longer collide, and
  sibling deletion changes the pin. Proof:
  TestHashDirFramingIsInjective.
- **LOW** — duplicate artifact hashes across registrations refused
  (order-dependent admission); pinned-file reads bounded; loader
  negative space covered (oversize, directory-as-anchor, missing
  registry, version-zero, malformed entry).

**Findings NOT fixed by inference — recorded for owner decision:**
1. Four enumerated pins (`skill_catalog`, `contract_registry`,
   `criteria_registry`, `regression_set_registry`) are parsed but
   have no consumer; the package comment's "verified where those
   planes are consumed" is aspirational. In particular the skill
   procedure is still verified against a SUBMITTER-supplied hash
   (Q-G1-5's "skill refs via the anchored catalog" unimplemented).
2. No append-only check on the anchors registry; the design's
   implementation note claiming it is corrected here.
3. `governed["deployment_anchor"]` is write-only provenance —
   nothing on the read path re-establishes admission, and a record
   lacking the key is refused by nobody (Q-G1-7 half-met).
4. Door table and constitution hashes absent from the enumeration
   (Q-G1-2 arity); per-workflow vs per-deployment arity for
   contract/ceilings makes multi-workflow anchors unexpressible.
5. The shipped `local-dev@1` pins a spec TEMPLATE where an exec
   ceiling belongs (no exec-ceiling artifact exists under
   policies/execution/) — the ACTIVE anchor is un-runnable as
   written and nothing references it from code.

Governed registration: `policies/deployment/local-dev.json` (anchor,
sha256 88bdc772da01c288…) proposed via
`anchors.proposed.json` → owner act → `anchors.json` ACTIVE
(2026-09-13). Verified inert before activation (admission refused:
registry unavailable) and admitted after.


## Owner disposition of the five findings + remediation (2026-09-13)

Owner ruling: four MUST FIX, one implementation correction; **no
D-G1-1 reopen** — the architectural rule is settled and the
remaining question is anchor COMPLETENESS: *an admitted Deployment
Anchor must be a closed-world declaration of the artifacts that can
influence governed execution*, not merely "the artifacts the
validator currently checks". Production wiring stays BLOCKED until
these land.

**1. Skill resolution from the anchored catalog (MUST FIX) — DONE.**
Under an anchored deployment, a skill-attributed envelope resolves
its composition from the ANCHORED catalog: the catalog file must
hash to the anchor's `skill_catalog` pin, the skill ref must resolve
to an ACTIVE registration, and the envelope's sealed composition
must EQUAL that registration's `composition_sha256`. The submitter
selects an anchored skill identity and can no longer select one of
its constituent hashes. L7 still only compares hashes (skill-blind);
the identity it compares against now comes from governed bytes.
(`Orchestrator.verifyAnchoredSkill`, `Config.SkillCatalogPath`.)

**2. Anchors registry append-only (MUST FIX) — DONE.**
`deployment.Registry` + `CheckAppendOnly` (the ratified pattern):
registrations never disappear, `name@version` never rebinds,
state advances active→withdrawn only. The wall spans restarts — the
observed registry state is persisted under the record root and
re-checked at every Open, so out-of-band mutation is DETECTED
(Register T), not trusted. deployment@N → H is immutable.

**3. Read-path re-verification (MUST FIX) — DONE.**
`deployment.VerifyAnchorRecord(recordedHash, anchorBytes,
registryPath)`: the record IDENTIFIES the anchor; the registry and
the bytes PROVE what that identity meant — the G2 principle applied
to G1. The anchor bytes are durable in the record, so a cold reader
re-parses them, re-hashes to the recorded identity, and
re-establishes registration. A withdrawn anchor still explains past
execution (withdrawal stops new opens, never rewrites history); a
deregistered one makes the deployment uninterpretable and says so.
Exercised end-to-end in Phase C (`verifyDeployment`).

**4. Enumeration completeness + arity (MUST FIX) — DONE.**
- *Constitution pins*: the anchor pins the L6 and L7 constitution
  hashes — compiled control vocabularies that pass the owner's test
  ("can changing this artifact change the behavior or authority of
  an anchored deployment?"). A rebuilt binary with a different
  constitution cannot open under the old anchor.
- *Door table*: classified, not pinned — L11's door table is
  compiled reviewed code ("adding a door is a reviewed code change,
  never data"), and the four door REGISTRIES it names are pinned.
  Changing it is a code change, covered by the constitution/binary
  identity rather than by a file hash. Recorded classification, not
  an omission.
- *Arity*: `workflows` is now a list of INDIVISIBLE bundles
  (workflow + its workflow ceiling + exec ceiling + context
  contract), bounded at 64. Multi-workflow deployments are
  expressible, and a submitter cannot pair an anchored workflow with
  another bundle's ceiling.

**5. local-dev@1 (implementation correction) — DONE by WITHDRAWAL.**
The anchor pinned a spec TEMPLATE where an execution ceiling
belongs, and no execution-ceiling artifact exists
(`policies/execution/` is empty). Per the owner: do not reinterpret,
do not infer a default — refuse. `local-dev@1` is WITHDRAWN (not
deleted: the registration stays interpretable); its bytes are
retained as `local-dev.withdrawn.json`; `policies/deployment/
README.md` records the reason and what a runnable `local-dev@2`
requires. **No anchor is ACTIVE**, so orchestrators run only in the
explicitly declared Unanchored test-harness role.

**Negative-space re-run:** every newly anchored dependency now has a
submitter-alteration refusal test — bundle indivisibility,
workflow-set membership, model allowlist (reachable now), model
registry rewrite, instruction-policy pin, instruction-root drift,
constitution drift, catalog composition mismatch, registry rebinding
and deletion across Opens, and the read-path cases. Phase C runs
ANCHORED end-to-end.


## Execution-ceiling shape (owner decision 2026-09-13 — implementation/governance, NOT a new G1 decision)

**Deployment-supplied exact bytes, hash-bound by the admitted
anchor.** Rejected: a repo-committed ceiling with a placeholder
`mirror_root` — it would create a second resolution mechanism
(anchor → placeholder → environment → authority) and force a new
set of questions (which variables, who supplies them, authenticated?
mutable after Open? part of identity? reconstructible?) that G1
exists to avoid. Also rejected: pinning a ceiling TYPE resolved into
host values outside the anchor.

Implemented:
- `Anchor.ExecutionCeiling` pins the exact ceiling bytes,
  DEPLOYMENT-scoped (the ceiling describes where and under what
  limits this deployment executes — a deployment property, not a
  workflow property). Workflow bundles now pin workflow + workflow
  ceiling + context contract.
- `Config.ExecCeilingPath` supplies the bytes at Open. Open hashes
  them, compares to the pin, refuses on mismatch, ALSO refuses a
  pinned ceiling that does not load (a governance-artifact defect is
  caught at Open, not mid-walk), and freezes the hash.
- SubmitTask refuses any task whose ceiling is not those bytes:
  "supplied at Open, never chosen per task". Deployment-supplied
  never means submitter-selected. Proof:
  TestAnchoredSubmitRefusesUnanchoredBundle's foreign-ceiling arc.
- Phase C is now a CONCRETE DEPLOYMENT INSTANCE: it mints its own
  ceiling (real mirror_root), pins it, registers and admits the
  anchor, runs the full chain anchored, and re-establishes the
  deployment from the record.

Repository state: no ACTIVE anchor, by design. `local-dev@1` stays
WITHDRAWN and is not reactivated; a runnable anchor is
deployment-instance-specific and is created where that deployment
lives, not committed here (policies/deployment/README.md records the
contract). Production wiring therefore remains gated on a concrete
deployment, not on missing architecture.

Door-table classification, recorded explicitly per the owner: the
L11 door table is REVIEWED CODE ("adding a door is a reviewed code
change, never data"); its four referenced registries (skill catalog,
L10 contract registry, L11 criteria and regression-set registries)
are the governed DATA dependencies and are pinned by the anchor; the
constitution pins provide the binary/code integrity boundary.
