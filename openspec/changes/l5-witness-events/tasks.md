# Tasks: L5 witness events (W-M1..W-M4)

Grill CLOSED 2026-09-25 (D-W-1..6, `design.md`). Milestones are gated
one at a time by the owner; each lands hermetically green, probes
killed, with a Gate 1 note in `RESUME-HERE.md` of `themis-v0` (the
amendment is on Themis v0's critical path). No push unless the owner
says so. `rsys@6` is a host act (W-M4), never minted from the laptop.

## W-M1 — Constitutional writer invariant (Class 3/4: constitution change) — LANDED 2026-09-26
- [x] `state`: closed `eventWriters` table — every class → its closed SET
      of owning writers (layer classes exactly one; `lifecycle` = {l6,
      l6-recovery}, `recovery` = {l6-recovery}); `append` refuses any pair
      outside it with `ErrConstitution` naming the pair; primitive-only
      classes keep their rule
- [x] Folded into `state.ConstitutionHash()` as `writer:<class>><writer>`
      parts; pin test records NEW `1df0e285…` and historical `33c6f6c5…`
- [x] `orchestration.ConstitutionHash()` asserted UNCHANGED by its pin test (D-W-4)
- [x] Every production emitter's pair admitted; every class × every other
      layer identity refused (`TestEventWriterInvariant`); the table covers
      the whole vocabulary and nothing else
- [x] Fixtures: no anchor fixture hard-codes the hash (all compute it);
      four sink tests that used an arbitrary `l2`/`l7`/`forger` writer for
      classes they do not own were corrected to the owner — the walk was
      never wrong, the fixtures were
- [x] Probes (restored after each): drop the sink check → killed; unfold
      from the hash → killed; widen `l2-delivery` to a second writer →
      killed (hash pin); let L7 write an L5 class → killed

## W-M2 — L5 emission handle and the witnesses (Class 3) — LANDED 2026-09-26
- [x] `state.L5Sink` (`TaskRecord.L5Sink()`): `Transition`, `Op`, `Egress`
      only; writer `l5` named in exactly one place; L5 classes are
      handle-only — `AppendEvent` refuses them under every identity;
      refuses after terminal; exported-API closure extended deliberately
- [x] `execution.Witness` interface declared by the consumer (execution
      imports nothing of `state`); `Env.Attach` once; pre-record
      emissions buffered and replayed in exact order at attach
- [x] Per-edge ordering table: PROVISIONING→ACTIVE after effect; seal
      after effect (workspace read-only first); SEALED→EGRESSING before
      effect; egress op after the store acknowledged and before return;
      →ACKNOWLEDGED after; →TEARDOWN before; terminal edge after
- [x] `l5-op` on all four subprocess paths (endpoint-refused,
      budget-exhausted, start-failed, completed incl. timeout/exit-error);
      egress `l5-op` acknowledged with address and totals; refused/failed
      egress witnessed with no address
- [x] Unwitnessed environment: ACTIVE-phase ops, clean seal, and egress
      refuse (`ErrUnwitnessed`); teardown always proceeds; a teardown-phase
      witness failure is kept in `Trace.WitnessErr`
- [x] Witness refusal on a completed op fails the op closed (output
      discarded); the sink's secret scan is the refusing case
- [x] Orchestrator attaches `task.L5Sink()` immediately after `CreateTask`
- [x] AST wall (`orchestration/l5_wall_test.go`): execution imports no
      state; no `"l5"` AppendEvent writer literal outside state; `L5Sink()`
      constructed only at the record boundary
- [x] Register B: a real walk's stream shows the full machine
      (PROVISIONING→ACTIVE→SEALED→EGRESSING→ACKNOWLEDGED→TEARDOWN→DESTROYED),
      provisioning and egress ops, the egress acknowledgement naming the
      bound address (`"sha256:"+addr` = binding ref), all before
      `artifact-bound` and COMPLETED (`src/themis/intake_test.go`)
- [x] Constitution hash UNCHANGED by W-M2 (pin test); probes: omit an
      edge, drop the egress witness, drop the unwitnessed guard, ignore a
      refused witness, never attach, drop the refused-egress witness —
      6/6 killed

## W-M3 — Themis five-link replay and the compatibility rule (Class 3)
- [ ] `intake`: closed `witnessingConstitutions` table seeded with the
      new hash; test asserts the compiled harness hash is in it
- [ ] Branch on the RECORD's `constitution_hash` (manifest + CREATED
      event agree): witnessing → five-link replay required
      (ACTIVE→SEALED task-complete → SEALED→EGRESSING → `l5-op` egress
      acknowledged A → `artifact-bound` A → object → COMPLETED), every
      failure `artifact-provenance-refused` naming the first failed
      link; `ProductionWitness = l5-witnessed`. Historical → the T-M3
      replay, `ProductionWitness = l6-record-only`, constitution hash
      rendered, Position-ineligible
- [ ] Register A: positive five-link; each refusal (absent, reordered,
      duplicated, unacknowledged, mismatched address, wrong seal reason,
      L5 class with wrong writer cannot even exist) with its positive
      twin; the four T-M3 fixtures replayed as historical negatives
- [ ] Probes: forge (hand-built witness via the L6 primitives — must be
      impossible after W-M1: prove it), omit, substitute (egress op naming
      B, binding A) → all killed

## W-M4 — `rsys@6` host mint and Governance record (Class 4: host act)
- [ ] Folded into T-M5's host sequence: ONE mint carrying the amended
      `constitution.state`, `themis_store`, catalog/skills; `rsys@5`
      withdrawn only after `rsys@6` ACTIVE and validated; registry
      append-only
- [ ] Host check: `constitution.state` in `rsys@6` == compiled
      `state.ConstitutionHash()` of the binary RUNNING on the host
- [ ] Signoff Addendum G; VM procedure extended with the five-link
      proof rows
- [ ] Exit conditions 1–7 of `proposal.md` ticked with evidence; then
      T-M4 UNBLOCKS

## Ordering against Themis v0
W-M1 → W-M2 → W-M3 → T-M4 (Position on a five-link resolution) → T-M5
with W-M4 folded in (one host act).
