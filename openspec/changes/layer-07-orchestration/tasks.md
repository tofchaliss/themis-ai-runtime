# Tasks: Layer 7 — Orchestration

Execution starts only after the grill closes Q-L7-1..9 and the owner accepts (Gate 0). Three-state verdicts per milestone: architecture-conformant · test-evidenced · operationally proven. An editing command or commit message is never evidence.

## 0. Gate

- [ ] Grill session held; Q-L7-1..9 answered and recorded
- [ ] Decisions folded; Gate 0: design accepted by owner

## 1. L7-M1 — Task envelope + assembly (Class 3)

- [ ] Envelope artifact, fail-closed loader, governed/payload split
- [ ] Assembly ⊆-checkpoint: grant ⊆ ceiling, plan ⊆ contract, disjointness at its designed call site, hashes into the L6 record
- [ ] Security review

## 2. L7-M2 — The loop (Class 3)

- [ ] Deterministic step structure; CallState increment (the L4 obligation); record-before-effect wiring (the D-L6-10 obligation); L5 lifecycle driven by the loop
- [ ] Typed termination → seal reasons → L6 lifecycle projection
- [ ] Security review

## 3. L7-M3 — Failure, retry, recovery, routing (Class 2/3)

- [ ] Failure-class boundary per the Q-L7-5 closure; retry_of submission path
- [ ] Startup recovery invocation; live-writer discipline
- [ ] Router evolution behind the governed model-selection input
- [ ] Security review

## 4. L7-M4 — Proof + close (Class 2)

- [ ] Live proof per Q-L7-9 (production loop end-to-end + kill-mid-loop cold recovery)
- [ ] Traceability, coverage-verified; reviews with three-state verdicts
- [ ] Code map + status doc + artifact updates; push/archive on owner approval

## 5. Deferred (NOT L7 scope)

- Approval-channel implementation (seam only); subagents (L8); run_command (OPEN-2); scheduling/queueing/parallel tasks; durable-record L2 source kind; Themis transfer/anchor
