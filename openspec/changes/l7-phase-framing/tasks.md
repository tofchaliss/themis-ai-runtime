# Tasks — L7 phase framing (P-M1..P-M3)

Grill closed 2026-09-27 (D-P-1..6; D-P-6 pending one host line). Same
rules as I-M*: each milestone lands hermetically green with a Gate 1
note in `themis-v0/RESUME-HERE.md`; PROPOSED artifacts stay inert; the
host act is the owner's; no push without the owner's word.

## P-M1 — The phase rule and the phase-state slot (Class 3: model-visible framing) — LANDED 2026-09-27
- [x] L1: one harness-system instruction file under `instructions/global/system/`
      stating the phase rule (D-P-1); no phase names, no skill specifics
- [x] L2/L7: the loop's fixed plan becomes `task-payload` + `phase-state`
      (kind `phase-state`, authority `derived`, author `harness`,
      sensitivity public); content per D-P-2 (`workflow`, `phase`,
      `completed`, `capabilities` = narrowed phase grant); provenance =
      `cause_seq` of the entering transition (initial: RUNNING seq)
- [x] Plan ⊆ Contract: a contract without `phase-state` refuses at Gather
      (link-named); never skipped, never optional
- [x] Record: `l2-delivery` carries the slot like any other; byte-exact
      reconstruction (D-L7-11) re-derives the same phase-state from the
      transition chain — test asserts equality, not just presence
- [x] Tests: refused composition under a one-slot contract; delivered
      under a declaring contract; `capabilities` equals L4's phase grant
      in every phase; a fresh phase shows `completed` in order;
      constitution pins UNCHANGED (both hashes asserted)
- [x] Integration suites move from `@4` to `@5` where they walk; the
      existing `@4` fixtures stay untouched (real records under the old
      loop; Themis intake tests keep reconstructing them)

## P-M2 — remediate-dependency@5, rsys7 candidate, records (Class 2) — LANDED 2026-09-27
- [x] `policies/skills/remediate-dependency-5/`: contract declares
      `phase-state` (required, `[derived]`); phase-scoped procedure
      (D-P-4); workflow/ceiling/grant bytes identical to `@4`
- [x] Catalog entry `@5` + `reg-skill-remediate-dependency-5` citing this
      grill and the two host records; `@4` stays active
- [x] `policies/deployment/rsys7.proposed.json` derived from `rsys6.json`
      + new `instruction_root_system`, `skill_catalog`, the `@5` bundle,
      `skills: [@5]` (owner amendment 2026-09-27); parse-verified; admission refused (inert); `97599c96…`
- [x] `docs/operations/rsys7-host-sequence.md`: delta page (checks,
      the D-P-5 gate, act 3, then I-M5 steps 7–11 under `rsys@7`);
      demo document regenerated; matrix and RESUME-HERE updated

## P-M2b — D-P-7 (Class 2) — LANDED 2026-09-27
- [x] `phases.md` completion sentence; `remediate-dependency@6` +
      `reg-skill-remediate-dependency-6`; suites on `@6`; capture mirror
      with a real `require`; candidate re-derived `a5266c52…` (inert);
      full hermetic sweep green; pins unchanged

## P-M3 — Host gate and act (Class 4: host)
- [ ] `TestLiveCapture` with `@6` on the host under the test anchor:
      gpt-oss:20b must COMPLETE; qwen2.5:7b and cyberpal20b-v3 captured
      as compatibility evidence (D-P-5); raw turns archived
- [ ] Governance act 3: `rsys@7` ACTIVE (decision `rel-anchor-rsys-7`);
      `rsys@6` stays active until a Position exists under `rsys@7`, then
      the owner decides `rsys@5`/`rsys@6` withdrawal as separate acts
- [ ] I-M5 steps 7–11 resume under `rsys@7` (commission, walk, intake,
      proposal, decision, evidence, patch bundle)

## Consequences recorded while landing P-M1 (facts, not decisions)
- Every skill registered before D-P-1 (`remediate-dependency@1..4`,
  `investigate-cve@1`) is un-composable under the amended loop: its
  contract lacks `phase-state`, so a submission refuses at Gather. They
  stay `active` in the catalog as recognized history. `rsys@7`'s
  allowlist `[@1, @2, @5]` (D-P-4) therefore admits two entries that
  cannot compose — harmless (admission ≠ composability) but the owner
  may prefer `[@5]` at the mint; flagged in the Gate note.
- `investigate-cve@2` registered (`reg-skill-investigate-cve-2`): the
  runtime's own anchored-skill suites (~50 references) need a composable
  skill of that shape; `@2` = `@1` + the slot + a phase-scoped ASSESS.
- `policies/context/task-contract-v2.json` (v1 + the slot) for the
  generic-task suites; v1 kept, nothing pins it.
- `state/live_test.go` attached the L5 witness (a W-M2 leftover that
  failed on the pushed tree, unrelated to this change).
- The `workflow` field of phase-state is the WorkflowDef's own
  `name@version` (definition version, 1 for every current skill), not
  the skill version.
