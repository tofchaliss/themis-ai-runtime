# Design: Layer 7 — Orchestration

**Inputs:** `docs/architecture/harness/00-p0-architecture-v2.md` §10 (layer doc), archived L1–L6 designs (their L7-addressed IOUs land here), `ARCHITECTURE.md`, the existing `Server.invoke`/`Router` fragments (code map: EVOLVE).
**Decision IDs** `D-L7-n`; **open questions** `Q-L7-n`.

## 0. Position in the flow

```
Task envelope (governed half + untrusted payload)
        │
        ▼
┌───────────────────────────────────────────────────────────┐
│ L7 Orchestration — composition and pacing, never authority│
│  assemble (roots · ceilings · hashes)                     │
│  loop: L1 → L2/L3 → model → L4-in-L5, L6-recorded         │
│  terminate typed → seal → egress → teardown → record      │
└───────────────────────────────────────────────────────────┘
        │ everything it did is in the L6 record
        ▼
 Themis governance (system of record)
```

## 1. Hard invariants (inherited, not grillable)

- **L7 composes and paces; it never grants.** Every permission arrives in a governed artifact; L7 instantiates or narrows ceilings and can never define one (the locked cross-layer principle, fourth application).
- **Model output stays advisory through every iteration:** the loop converts model output into *requests to gates*, never into authority; nothing accumulates trust across turns.
- **A lower-level component never trusts the caller to have authorized** (Q-L5-3) — L7 being the caller changes nothing: every gate re-checks.
- **Record-before-effect is L7's wiring duty** (D-L6-10): audit before result delivery, delivery record before payload delivery, lifecycle event before projection.
- **Deterministic security decisions never consume L6 state** (Q-L6-10); L7 itself reads only the StatusView vocabulary, under declared workflow-owned dependencies.
- **Fail closed:** an unassemblable task never starts; an unwritable record fails the task; typed outcomes only.

## 2. Draft decisions (grill targets)

### D-L7-1 — The task envelope
One governed submission: {task_id, workflow ref, repo + pinned SHA, grant, provision spec, context contract/plan, task payload}. The governed half is loaded fail-closed with recorded hashes; the payload half is and stays external-untrusted. The envelope is the ONLY task input — no ambient configuration reaches the loop.

### D-L7-2 — Assembly is the ⊆-checkpoint
At assembly, the one place everything is known: grant ⊆ WorkflowCeiling (the recorded L4 deferral), plan ⊆ contract (L2's input), spec ⊆ execution ceiling (already L5's, invoked here), root pairwise disjointness (L6's check, at its designed call site), all artifact hashes into the L6 task record.

### D-L7-3 — The loop is deterministic mechanics around an advisory core
Fixed step structure: compose → model call → handle tool calls (each through L4, inside L5, recorded in L6, CallState incremented by L7 between calls — the L4-locked obligation) → append results to the conversation → iterate. The loop adds no judgment: what varies per step is model output, and model output only ever becomes gate requests.

### D-L7-4 — Typed termination, projected
Closed termination vocabulary mapped onto existing machines: model-completed → seal(task-complete) → egress → COMPLETED; call-budget/env-deadline exhausted → seal(deadline) → FAILED; fatal error → seal(fatal-breach) → FAILED; crash → L6 recovery's FAILED_PARTIAL. No termination state that lacks a home in L5 seal reasons + L6 lifecycle.

### D-L7-5 — Retry is a new identity, linked
Re-running a failed task = new task_id + `retry_of` (locked Q-L6-3), new environment identity (locked Q-L5-12), fresh assembly. L7 never resumes, reconciles, or reuses.

### D-L7-6 — Model selection behind a governed input
The existing Router capability evolves in; the selected model is a governed envelope/workflow input recorded in the task record — never a per-turn dynamic choice by model output.

### D-L7-7 — Approval seam defined, not implemented
The L4 `requires-approval` reservation gets its routing seam shape (decision surfaces to a human channel; the loop blocks typed); no channel implementation in v1.

## 3. Open questions for the grill (Q-L7-n)

1. **Q-L7-1 — What may the orchestrator decide?** The layer's central authority question: enumerate L7's actual decision surface (pacing, iteration count within budgets, termination timing, retry submission) vs what looks like a decision but must be envelope-determined (tool grants, model choice, context plan, ceilings). Where is the line that keeps "composition" from quietly becoming a policy engine?
2. **Q-L7-2 — Who authors the envelope, and what trusts what?** The envelope's governed half vs payload half; can a caller-supplied workflow reference expand anything; is the envelope itself a governed artifact with the full fail-closed posture?
3. **Q-L7-3 — Turn structure and context lifecycle:** what carries between iterations (conversation history vs re-gather/re-compose), how tool evidence enters subsequent turns, and how the L3 no-re-management rule interacts with a multi-turn loop.
4. **Q-L7-4 — Budgets and pacing:** which budgets L7 enforces (total calls locked in grants; wall-clock in specs) vs owns; does the loop have its own iteration ceiling; what the model sees on budget exhaustion.
5. **Q-L7-5 — Failure classes:** which errors continue the loop (typed tool errors are model-visible data) vs fail the task; is that boundary fixed or workflow-declared?
6. **Q-L7-6 — Recovery and startup:** when does the loop invoke L6 recovery; what does it do with FAILED_PARTIAL/CORRUPT records it finds; the live-writer exclusion in practice.
7. **Q-L7-7 — The approval seam shape** (D-L7-7): where the block sits in the loop, what state it leaves if the process dies while blocked.
8. **Q-L7-8 — What of the existing service/Router carries over** (EVOLVE discipline): invoke precedence, routing tables, healthz — what becomes the L7 seam and what stays L10-era.
9. **Q-L7-9 — Operational proof gate:** proposed — one live task through the production loop end-to-end (the L4/L5/L6 proofs' union, driven by the real loop), plus a kill-mid-loop run recovered cold.

## 4. Test plan (three-state discipline)

Fail-closed envelope loader; assembly ⊆-matrix (each containment violated in isolation, refused typed); loop determinism (scripted provider: identical inputs → identical records); CallState progression; termination vocabulary → seal/lifecycle mapping (every class); retry linking; record-before-effect ordering asserted in the loop (not just in tests' own wiring); kill-mid-loop recovery; live proof per Q-L7-9.
