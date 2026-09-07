# Proposal: Layer 7 — Orchestration

**Change ID:** layer-07-orchestration · **Status:** DRAFT for grilling · 2026-09-07
**Owner gate:** ships only after the grill closes Q-L7-1..n (design.md) and the owner accepts.

## Why

Six shipped layers have been writing IOUs addressed to exactly one recipient. L4: "the loop must increment CallState between calls" and "the ceiling ⊆ check is L7-era." L2: the plan/assignments arrive from "the L7 plan (future)." L5: "L7 will own provisioning eventually"; retry is "an orchestration decision." L6: task assembly is "the one place all roots are known"; the floor-event wiring is "the L7-era caller's composition duty"; `retry_of`, recovery invocation, and StatusView reads all await their consumer. Today every one of those obligations is discharged by test harnesses. L7 is the component that discharges them for real: the deterministic loop that composes six governed layers into one accountable task execution — **without acquiring any authority of its own**.

## What

An orchestration layer at `src/harness/orchestration`:

- **Task envelope:** the governed task submission — {task identity, workflow reference, pinned repo+SHA, grant, spec, context plan, task payload} — loaded fail-closed; the payload half stays external-untrusted (locked since L1/L2).
- **Task assembly:** the one place all roots are known — disjointness check, artifact hashes into the L6 task record, WorkflowCeiling ⊆ validation of the grant, plan ⊆ contract hand-off to L2.
- **The loop:** resolve (L1) → gather/manage/compose (L2/L3) → model call → for each tool call: L4 Handle inside the L5 environment with L6 record-before-effect wiring → CallState increment → iterate; typed termination (completion, budget, deadline, fatal) mapped onto the L6 lifecycle and L5 seal reasons.
- **Provisioning ownership:** L7 calls L5's Provision/Seal/Egress/Teardown at the right lifecycle moments — taking over the role the test harnesses have been playing.
- **Failure and retry:** typed task failure vocabulary; retry = new task identity linked via `retry_of` (locked at Q-L6-3); recovery invoked cold at startup, never against live writers.
- **Reuse:** the existing `Router` model-routing capability evolves in behind a governed selection input (per the locked code-map sequence), not a new subsystem.

## What this change does NOT do

- No new authority: every permission the loop exercises arrives in a governed artifact; L7 instantiates and narrows ceilings, never defines them (the locked cross-layer principle).
- No approval-channel implementation (the L4 `requires-approval` reservation gets a seam definition at most).
- No subagents (L8 deferred), no run_command (OPEN-2 preconditions stand), no scheduling/queueing/parallel tasks, no L2 durable-record source kind.

## Success criteria

A real task runs end-to-end through the production loop — not a test harness — with every layer's obligations discharged: CallState counted, ceilings validated at assembly, floor events committed record-before-effect, environment provisioned/sealed/egressed/torn down by the loop, typed termination projected into the L6 record; a crashed loop leaves a recoverable typed record; the L4/L5/L6 live proofs re-expressed as one L7-driven live proof; three-state verdicts per milestone.
