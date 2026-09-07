# Tasks: Layer 7 — Orchestration

Grill closed 2026-09-07 (Q-L7-1..12, all CLOSED; design.md §3 is the record, §2 the folded decisions D-L7-1..12 + constitution amendments + residuals). Execution starts only after owner Gate 0. Three-state verdicts per milestone: architecture-conformant · test-evidenced · operationally proven. An editing command or commit message is never evidence. The architecture defines the milestones — never the reverse.

## 0. Gate

- [x] Grill session held; Q-L7-1..12 answered and recorded (incl. the recovered turn-lifecycle orphan)
- [x] Decisions folded into design.md §2; fold-completeness cross-check vs the draft: no orphans
- [x] Gate 0: design accepted by owner (2026-09-07, "Approved and push it, then implement it") (audit vs the twelve closures: dropped decisions, accidental new authority, dead configuration, implementation-over-architecture, L1–L6 contradictions)

## 1. L7-M1 — Constitutions + workflow definition artifact (Class 3)

- [x] L7 constitution in code (hashed): control vocabulary (`declare_done` → `PHASE_COMPLETION_REQUESTED` only), structural turn facts, termination mapping, invariant-failure path, reserved approval vocabulary
- [x] L6 constitution amendments (deliberate, hash-changing): `workflow-transition` (cause-carrying: from/to/edge-id/causing-seq) + `model-turn` event classes
- [x] Workflow-definition artifact + fail-closed loader: static totality (coverage, no overlap), exhaustion edges per counter, counter-free cycles refused, finite worst-case walk computed, definition ⊆ ceilings, invariant/approval vocabulary undeclarable (approval-gated definitions refuse in v1)
- [x] Registry `control` classification with two-way constitution ⊆
- [x] Security review (2026-09-07: no CRIT/HIGH; 6 MED remediated with regressions in review_test.go)

## 2. L7-M2 — Envelope + assembly + seam (Class 3)

- [x] Task-envelope artifact, fail-closed loader, governed/payload split, no defaulting anywhere
- [x] Assembly ⊆-checkpoint: grant ⊆ WorkflowCeiling, plan ⊆ contract, spec ⊆ execution ceiling, root disjointness at its designed call site, hashes + submitter identity into the L6 record
- [x] Public seam exactly Open/SubmitTask/ReadStatus; adapter import lint
- [x] Security review (2026-09-07: no CRIT/HIGH; 6 MED remediated with regressions in review_test.go)

## 3. L7-M3 — The loop (Class 3)

- [x] δ evaluator over (definition-at-hash, cursor, declared typed event, counters); cursor derived from the record, never independent
- [x] Record-before-next-turn wiring: compose→commit→deliver; model output→object+model-turn event→commit→next turn; audit→execute→result→commit→delivery
- [x] Structural turn facts; `declare_done` executor; CallState monotonic increment; independent-gate budget behavior
- [x] Typed termination → seal reasons → L6 lifecycle; L5 provision/seal/egress/teardown driven by the loop
- [x] Invariant-failure fixed path (freeze → typed event → fatal-breach seal → teardown → FAILED)
- [x] Security review (2026-09-07: no CRIT/HIGH; 6 MED remediated with regressions in review_test.go)

## 4. L7-M4 — Startup, failure classes, routing (Class 2/3)

- [x] Startup sweep: every non-terminal → typed terminal via L6 recovery before accepting work; CORRUPT surfaced + preserved
- [x] Failure taxonomy wiring (declared execution failures to δ; undeclared to record+conversation only)
- [x] Retry_of submission path; duplicate-submission structural refusal
- [x] Router evolution: model selection is a fixed governed envelope input; no per-turn routing exists (the legacy Router remains a governance-side advisor, untouched)
- [x] Security review (2026-09-07: no CRIT/HIGH; 6 MED remediated with regressions in review_test.go)

## 5. L7-M5 — Proof registers + close (Class 2/3)

- [x] Register A structural suite · Register B adversarial suite (incl. no-defaulting proofs) · Register C fault sweep + real-kill (loop boundaries + meta-sync) · Register D deterministic-walk replayer with single-authority + what-the-model-saw reconstruction · Register E live proof with negative proofs
- [x] CallState record-recount verifier rule
- [x] Reviews with three-state verdicts (2026-09-07: test-evidenced YES — 80%+ coverage, all registers green; operationally-proven YES — live walk + real-kill vs qwen2.5:7b through the production loop; architecture-conformant YES — after arch HIGH 2a closed by owner decision "implement": full L2 composition wired into the loop)
- [x] Code map + status doc updated (2026-09-07)
- [ ] Traceability.md (written at archive) · artifact update · push/archive on owner approval

## 6. Deferred (NOT L7 scope — recorded IOUs, never silently promoted)

- Approval channel implementation (ingress authentication + the resumable-await question — its own grill)
- Cross-process orchestrator exclusion (v1: one orchestrator per state root, deployment invariant)
- run_command (OPEN-2 preconditions stand); subagents (L8); scheduling/queueing/parallel tasks
- Token/cost budget dimensions (declared-dimension vocabulary pattern); L2 durable-record source kind; Themis transfer/anchor
