# RESUME-HERE — L8 implementation (M1 onward)

Updated at every green milestone. If context was compacted, start here.

## One-line status
2026-09-23 (reviews): architecture + security reviews remediated (see
tasks.md §7 dispositions); owner decisions pending on arch MED-2
(withdrawn-in-scope at assembly) and MED-3 (§3 "one method" text);
test review relaunched. Then: traceability finalize, archive.
Earlier (M6 code): anchor pin, `rsys@4` PROPOSED with its four
companion proposals, themis-run wiring, record-ref furniture, anchored
Register B green, live walk admitted (model did not delegate). Next:
three Class-3 reviews in worktrees → remediate → traceability →
archive. Earlier (M5): record/reconstruction/observation LANDED (Register C
verdicts from L6 alone, projection, static bound, L10 observation).
Next: M6 — anchor pin `delegation_template_registry` + `rsys@4`
PROPOSED, themis-run wiring (registry-v5 + seam), Register B anchored
positive path, live proof, reviews, archive. OPEN at M6: the model
cannot see event seqs today, so it cannot form `<seq>:<id>` from a
framed result alone (gap 13) — decide the furniture before the live
proof. Earlier (M4): seam + L7 wiring LANDED — positive walk, stage A/B,
assembly, post-instance outcomes, fault points, carry filter, walls;
17/17 probes killed. Next: M5 (record/reconstruction/observation,
Register C/D) then M6 (anchor `rsys@4`, live proof, reviews, archive).
Earlier (M3): L6 `l8-delegation` class + body closure LANDED. Next:
M4 (seam + L7 wiring) after its Gate 1 section here — the largest
milestone; read D-L8-2/3, C-L8-5..10, C-L8-17..19 before designing.
Earlier: M2 LANDED (L4 `delegate`: target class,
exact-scope gate, evidence shape, closed error classes, instantiation
executor over an injected compose half, registry-v5). Next: M3 (L6
`l8-delegation` event class) then M4 (seam + L7 wiring) after their
Gate 1 sections here. Earlier: M0 + M0b LANDED (provider ceiling, Reported
identity, F-L8-2/3/4; amendment records in the L4/L7/L10 archives).
Next: M2 (L4 `delegate` capability) after its Gate 1 section here.
Earlier: M1 LANDED (package, loaders, first template PROPOSED,
README checklist, preflight check, Register A + 13/13 probes killed).
Next: M0 (provider ceiling, P-L8-2) and M0b (F-L8-2/3/4) — they gate
M4+, not M2; M2 (L4 `delegate` capability) needs its Gate 1 section
here first.

## Decisions a newcomer must not re-derive
`design.md` §2 D-L8-1..21 (locked), §5 the implementation whitelist,
§6 C-L8-1..21. Standing rule (D-L8-21): anything not in §5 stops the
work and is classified before code. M0/M0b prerequisites gate M4+, not
M1.

## Gate 1 — implementation design against §5 (M1 portion; later
## milestones add their own section here before their code)

**Registry schema** (`policies/delegation/registry.json`, mirrors
`policies/verification/contracts.json`): `{version, entries[{name,
version, template_sha256, manifest_path, state, steward}]}`. Loader
mirrors `verification.LoadRegistry`: bounded read (4 MiB), exact
lowercase keys + no duplicate keys (`internal/strictjson`), closed
schema, trailing-content refusal, identity syntax, immutable duplicate
refusal, sha syntax, relative confined `manifest_path`, state ∈
{active, withdrawn}. `ParseRef` exact only; `Resolve(ref)` → entry +
loaded template with hash verified against the pin and two-way
self-declaration (`name`, `template_version`); `CheckAppendOnly`
(deletion / rebind / un-withdrawal). No write API — AST wall + the
function-value-binding catch (verification pattern).

**Template schema** (`policies/delegation/<name>/template.json`,
closed): `version`, `name`, `template_version`, `context_contract
{path, sha256}`, optional `instruction {path, sha256}`,
`eis_carry_scopes[]` ⊆ {repository, directory, skill} (absent = carry
nothing beyond the roots; a root name, `task`, an unknown scope, or a
duplicate → named refusal), `brief {slot, max_bytes}` (1 ≤ max_bytes ≤
L2 `MaxItemBytes`), `max_output_bytes` (1 ≤ … ≤ L2 `MaxItemBytes`,
C-L8-7). **Disjointness at load:** a top-level key in {workflow,
workflow_ceiling, ceiling, grant, grant_template, spec, spec_template,
input_schema, template, templates, tools, model, scope, procedure} is
refused by name ("a delegation template may not pin …", D-L8-5/6)
before the closed-schema decoder runs; every other unknown key is the
closed-schema refusal. Pins resolve through `confine.ResolvePath`
under the template directory, regular files only, bytes verified
against the sha at load (C-L8-15 E: the template hash is the hash of
the manifest bytes, which pin every governed byte). The contract is
loaded by L2's own `LoadContract` (ordinary L2 contract; C-L8-14 D);
cross-checks: the brief slot exists, is not withheld, permits exactly
`[external-untrusted]`; no requiredness/slot semantics in
`template.json`. Loaded template retains `Raw`, `ContractRaw`,
`InstructionRaw`, `Hash`, `Dir` for later milestones (C-L8-9
durability; nothing is re-read at composition without re-verification).

**First template** (PROPOSED → owner Governance act):
`dependency-triage@1` for `remediate-dependency@1`'s triage need —
contract `dependency-triage` with slots `brief` (kind
`delegation-brief`, required, `[external-untrusted]`) and `evidence`
(kind `evidence-payload`, optional, `[governed-record, derived]`),
ceiling `public`; carry `[repository, skill]`; instruction file:
rules only, no factual claims, no directive against a higher scope.
Registration in `registry.json` is written by the owner's act; the
loader reads it. (The file lands PROPOSED with `state: active` so the
Register A positive twin can load it; the act is the owner's commit
that keeps or changes it.)

**Preflight** (C-L8-14 F): a "Delegation templates" section verifies
each registry entry's manifest bytes against `template_sha256` and each
manifest pin's bytes against its sha (read-only, python3 + shasum).

**Not in M1:** `Config.Delegator`, `delegate` capability, event
class, anchor pin — M2+. Scaffold dirs `roles/`, `runtime/`,
`isolation/` deleted with M1 (§5.1).

## Gate 1 — M2 (L4 `delegate`) design against §5.2

`TargetDelegationTemplate` + exact-scope gate in `Authorize`;
`ParseEvidenceRefs` shape (`<seq>:sha256:<hex>`, comma-separated, ≤256);
closed error classes with `DelegationRefused(reason)` constructor and
`KnownErrorClass` predicate; `DelegationInstantiator` interface
(`Instantiate(template, refs, brief) (capture, error)`) injected per
task through `NewExecutorTableWith`; `execDelegate` returns the capture
as evidence, `*ErrDelegationRefusal` → stage-B class, other errors →
seam-unavailable, nil → `delegation-refused:seam-unavailable`.
Registry-v5 = v4 + `delegate`. **Seq rule at instantiation:** every
reference's seq must already exist in the parent's stream when the
executor runs — which is strictly before the authorizing `l4-audit`
commits, so `seq < parent_call_seq` holds by construction (C-L8-8).
`themis-run` stays on v4 until `rsys@4`.

## Gate 1 — M3 (L6 `l8-delegation`) design against §5.2

One new caller-appendable class; body owned by `subagents/delegation`
(`Event` with closed `Outcome`, `Validate` closure: refs strictly
ascending and < parent_call_seq, template bytes referenced, output
present iff the outcome could produce one; `Encode`/`Decode`
canonical; `SortEvidence` refuses duplicates, never dedups).
Constitution hash changes → `rsys@4` re-pin at M6.

## Gate 1 — M4 (seam + L7 wiring) design against §5.1/§5.2

Recorded in full in the L7 archive amendment
`amendments/l8-delegation-seam/AMENDMENT.md` (boundary types, assembly,
post-hook, seam steps, walls). Classifications made at this gate:
(1) the seam has three entry points (Registered / Instantiate /
Delegate) — C-L8-13's instantiation half needs its own entry;
(2) L2 `Gather` accepts several distinct sources per slot (C-L8-6
assumed it; code refused it) — recorded as part of the §5.2 L2
amendment, parent path unchanged; (3) declared-absence record-object
sources so the seam names every non-withheld slot; (4) the P0
contract's slots are kind-routed (`tool:*`, `model-turn`,
`delegation-output`) — re-pinned in place because no registration act
had occurred; (5) `derived_sensitivity` = floor rank (residual).

## Gate 1 — M6 classifications

(1) **Record-ref furniture:** the model could not form `<seq>:<id>`
from a framed result (gap 13); L7 now appends `record-ref:
<seq>:<objectID>` after the frame of every authorized evidence-bearing
tool result and after a completed delegation's frame (the l8 seq +
output id). Deterministic, record-derived, outside the fence, part of
the projection. Implementation detail of L7 furniture; no authority.
(2) **`report-valid@2`:** an L10 contract pins the L4 registry hash it
is eligible under, so registry-v5 needs its own contract — a
consequence, not a decision; PROPOSED beside `rsys@4`.
(3) **`remediate-dependency@2`:** the delegating Skill is a new version
(immutable @1); only ceiling `allowed_tools`, ANALYZE capabilities,
the grant template, the gate token, and the procedure's method text
changed. @1's procedure uses `## ` headings the renderer refuses —
@1 cannot instantiate today (pre-existing, owner).

## Milestone log
- [x] M0 — green 2026-09-23 (ceiling at both adapters; registry narrows; mutation-probed)
- [x] M0b — green 2026-09-23 (F-L8-2/3/4; F-L8-3 residual: second-load window)
- [~] M6 — code green 2026-09-23; reviews, traceability, archive pending; owner acts listed in tasks.md §7
- [x] M5 — green 2026-09-23 (reconstruct.go, projection.go, L10 view; probes killed)
- [x] M4 — green 2026-09-23 (subagents/delegation/seam e2e + walls; L2/L1 amendments)
- [x] M3 — green 2026-09-23 (state vocabulary 16; delegation event tests)
- [x] M2 — green 2026-09-23 (tools/delegate_test.go; digest cases; skills surface test)
- [x] M1 — green 2026-09-23 (full hermetic suite green; owner
  registration act for `dependency-triage@1` and three Class-3 reviews
  still owed before M1 is called closed)

## Gaps found while testing
1. **Trailing content:** the exact-key wall (`strictjson.Check`)
   refuses trailing content ("after top-level value") before the
   loader's own trailing-content check runs; the loader's check is
   unreachable on that path (same shape as skills/verification, whose
   duplicate-key walls precede theirs). Harmless; recorded.
2. **Contract `workflow` field:** L2 requires it; for a template it is
   a label only. L7 compares `contract.Workflow` to the workflow name
   at assembly — the L8 seam (M3) MUST NOT apply that compare to the
   template contract (there is no workflow); note for the M3 Gate 1.
3. **Evidence slot kind:** the P0 contract's `evidence` slot uses kind
   `evidence-payload` (the L6 object class name). C-L8-14 C routes
   references by event-derived kind at the seam; the kind vocabulary
   for evidence references is fixed at M3 and may rename this slot's
   kind (a new template version, by construction).
4. **Registration is PROPOSED:** `registry.json` lists the template
   `active` so the positive twin loads through the real registry; the
   Governance act is the owner's commit. No anchor pins the registry
   yet (`delegation_template_registry` lands with `rsys@4`).
5. **Preflight counters:** the first draft piped the Python result
   into `while read`, which ran in a subshell and lost the PASS/FAIL
   counts (exit code would have been 0 on a FAIL). Fixed with a
   here-string before commit; the class — a check whose report cannot
   affect the verdict — is the recurring one.
6. **F-L8-3 first attempt tested nothing:** the existing refusal test
   named no contract, which L4 DENIES (required param) — the seam's
   pre-instance path was never reached and the test passed anyway.
   Fixed by naming an unregistered contract (authorized, then refused).
   The check-claims-more-than-it-establishes class again.
7. **F-L8-4 terminal race:** with a 2 s wall budget the cut turn may
   terminate via δ's turn-provider-error edge before the loop's floor
   check runs; the test asserts the cut (seconds, not minutes), not
   which of the two terminals fired.
8. **Commit 6f08824 carried a non-compiling test** (unused import) —
   a chained command's `&&` broke before the vet step and the commit
   still ran. Fixed in the follow-up commit; lesson: never chain a
   commit behind steps whose failure the chain can skip past.
9. **L2 refused two sources per slot** ("assigned twice") while C-L8-6
   requires one source per evidence reference in one slot; amended
   (distinct sources merge, same source twice still refused).
10. **L2 requires every non-withheld slot assigned** — the seam
   declares absence with an empty record-object source; the parent
   path never needed this because its contract has one slot.
11. **Two probes survived as mutual cover** (existence vs selectable
   witness; selectable map vs deriveClass default) — folded into one
   predicate each; now killed. The class again: a rule stated twice
   lets either copy rot.
12. **Sensitivity derivation is degenerate** (floor rank for every
   reference) — no per-item sensitivity exists on l4-audit/model-turn.
13. **The dynamic test model reads the record to build references** —
   a test convenience; a real model derives `<seq>:<id>` from what it
   saw (the framed hash equals the object hash; the seq is not shown
   to the model today — the live proof at M6 will tell whether a model
   can form a reference without it; likely an L7 furniture gap).
14. **Reconstruction without the registry:** an l4-audit-witnessed
   reference's class needs the registry at the audit's hash; when the
   caller cannot serve it, the class is taken from the witness and the
   registry is reported as a missing input (UNREPRODUCIBLE) — never
   assumed CONFIRMED.
15. **Verdict precedence:** a discrepancy and a missing input can both
   occur; DISCREPANCY wins (a defect signal must not be hidden by an
   availability fact). First draft let the missing input win.
16. **remediate-dependency@1 cannot instantiate:** its `procedure.md`
   carries `## ANALYZE` / `## REMEDIATE`, which the L1 renderer refuses
   as reserved H2 furniture. Never exercised by a test (the verification
   e2e uses a hand-written workflow). Owner: re-register as @2's
   sibling or accept @2 as its replacement.
17. **Live model did not delegate** (qwen2.5:7b declared done in
   ANALYZE, then stalled in REMEDIATE): admission + typed terminal
   evidenced; delegation-through-a-live-model not. Same class as
   `TestLiveWalkProof`'s coupling to model behaviour.
18. **Anchor field is mandatory:** pre-amendment anchors (`rsys2.json`)
   no longer load — by the "declaration, never a default" rule; the
   loader test worlds gained the field.
19. **Test hermeticity:** the seam suite mutated a repository safety
   file in one test; the unanchored world now runs on a private copy
   of the safety root (anchored worlds still hash the real roots and
   never mutate them).
20. **Reported ≠ WireModel on providers that report dated ids**
   (OpenAI-style) mints `model-identity-mismatch` on every delegation —
   fail-closed and correct per C-L8-10, but L8 is unusable on such a
   runtime until governance decides how a reported id maps to the
   requested one (residual).
21. **Live rerun after MED-5:** qwen2.5:7b still declares done in
   ANALYZE without reading or delegating (1m27s, FAILED via
   turn-no-action exhaustion in REMEDIATE). Model behaviour; the
   anchored scripted positive path is the Register B evidence.
22. **Reviewer worktree branches remain** (`worktree-agent-*`): the
   worktrees were pruned from the index and ignored, but branch
   deletion is a topology change the git-guard blocks — owner act.
