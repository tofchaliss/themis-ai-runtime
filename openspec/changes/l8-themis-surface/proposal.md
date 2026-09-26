# Proposal: L8 × Themis — the integration surface of subagent delegation

Status: **OPEN 2026-09-26 — grill open (owner-led, one question per
turn).** Completion-matrix row 11. **Not a re-grill of L8**: D-L8-1..21
and C-L8-1..21 are locked, archived (2026-09-23), and live-proven
(Addendum F). This change asks only what Themis integration needs from
delegation, against the stable parent established by Row 4 (D-C-1..6).

## What the L8 record already settles (facts, not questions)

- **A delegate cannot read Themis.** D-L8-2: one isolated, tool-less,
  single-call model execution. D-L8-9..13: zero capabilities, no grant,
  no ceiling, no workspace, no verification authority, **no Governance
  access**; nesting impossible by reachability. "Does a delegate
  inherit the parent's Themis scope?" is therefore moot: it has no
  capability interface to scope.
- **Themis bytes can reach a delegate only through the parent's
  authorized reads.** A delegation's `evidence_refs` name prior events
  in the same task; the seam refuses unless each names an AUTHORIZED
  call under the registry in force whose output object is the one
  referenced, prior to the authorizing `delegate` call, and accepted by
  a slot of the template's context contract (C-L8-6/8/9). So a
  `get_finding` result the parent was authorized to read may be shown
  to a delegate; nothing else can be.
- **Delegation output is data at the floor.** D-L8-11 and C-L8-18:
  the result is model-authored bytes, `external-untrusted`, a paired
  tool result; the class derivation reads the witnessing event, never
  the class of anything referenced. No laundering of governed-record
  through a delegate is possible.
- **The record witnesses every delegation:** `l8-delegation` (writer
  l8) with parent call seq, template identity, composition, template
  object refs, model identity, evidence refs, output ref, outcome.
- **Delegation is inside the commissioned method and deployment:** the
  `delegate` capability is in the skill's grant (method) and the
  delegation template registry is pinned by the anchor (deployment).
  No separate commission per delegation is needed or possible.
- **L10 never evaluates L8 output** (D-L8-18); a delegation cannot be
  a verification fact.

## What is genuinely open

| # | Question | Recommendation (PROPOSED) | State |
|---|---|---|---|
| Q-L-1 | How do delegations appear in the intake evidence view and the proposal evidence? Today D-T-7's "model turns" lists `model-turn` events only. | render delegations as part of the MODEL-REASONING fact (never the harness or verification fact): per delegation — seq, parent call seq, template name@version, model identity, evidence refs with the class each carried, output object id, outcome; `harness-execution/v1` gains `delegations` (count + seqs); Themis interprets none of it | OPEN |
| Q-L-2 | Does Themis need any delegation-specific admissibility rule at proposal time? | no: the commissioned method and deployment already cover the `delegate` grant and the template registry; a delegation that violated its template would have failed in the runtime record (outcome ≠ completed) and is visible in Q-L-1's rendering; adding a Themis rule would make Themis interpret runtime reasoning structure | OPEN |
| Q-L-3 | What must be recorded as the prohibition that keeps rows 3 and 11 stable? | "a tool-capable L8 is the decision that reopens Themis scope inheritance"; until then Themis scope reaches a delegate only through the parent's authorized reads (D-I-4/D-I-9 stay intact by construction) | OPEN |

## Not in scope

No tool-capable delegation · no delegate access to Themis · no L8
verification · no change to D-L8-* · no demo-specific field.
