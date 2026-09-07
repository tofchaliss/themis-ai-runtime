# Proposal: Layer 9 — Skills & Procedures

**Change ID:** layer-09-skills · **Status:** GRILL OPEN (Q-L9-1..10) · 2026-09-07
**Owner gate:** ships only after owner Gate-0 acceptance of the folded design.

## Why

L7 shipped the executor: a deterministic walk over a governed workflow lattice, driven by one envelope referencing seven governed artifacts. But today every task submission hand-assembles that envelope — workflow, ceiling, contract, grant, spec, payload — and nothing captures *how a repeatable class of task should be performed*. OPEN-1 (harness-project-report) demands the full runtime-skill design; the L1 grill left a recorded IOU (skill instruction sources, deferred to L9); ADR-009 constrains it (skills are procedures, never authoritative business truth). The architecture's answer to "how is a repeatable task performed?" is L9: a skill is a **reviewed, versioned procedure** the harness can execute — not a model, not a tool, not a policy engine.

## What

A skills layer that turns the architecture's skill structure (Purpose / Inputs / Required context / Allowed tools / Procedure / Expected outputs / Verification / Failure conditions / Approval requirements) into governed artifacts executable by the shipped L7 loop. The standing OPEN-1 format recommendation — a state-machine skeleton whose step bodies are instruction text, so checkpoints stay deterministic and procedure prose stays cheap to edit — now has a shipped state machine to target: the L7 workflow lattice.

## What this change does NOT do

- No new execution authority: a skill instantiates and narrows governed ceilings, never defines them (the locked cross-layer principle, fourth application).
- No business truth inside skills (ADR-009); no subagents (L8); no run_command (OPEN-2 stands); no approval-channel implementation (L7 residual stands).
- Runtime skills are distinct from development-time Claude Code skills in `.claude/` (recorded in the project report); this layer never touches the latter.

## Success criteria

A registered P0 skill executes end-to-end through the unmodified L7 production loop against a live local model, with skill identity + version + hash in the L6 record; skill instruction text reaches the model only through governed channels with honest provenance; an unreviewed/tampered skill is structurally unexecutable; three-state verdicts per milestone.
