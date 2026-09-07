# Tasks: Layer 6 — Durable State

Execution starts only after the grill closes Q-L6-1..9 and the owner accepts (Gate 0). Three-state verdicts per milestone: architecture-conformant · test-evidenced · operationally proven. An editing command or commit message is never evidence.

## 0. Gate

- [ ] Grill session held; Q-L6-1..9 answered and recorded
- [ ] Decisions folded; Gate 0: design accepted by owner

## 1. L6-M1 — Task record + state root (Class 3)

- [ ] Governed state-root/retention artifacts, fail-closed loaders
- [ ] Task-record envelope (manifest-first, governed-artifact hashes, status vocabulary)
- [ ] Security review

## 2. L6-M2 — Trace sink (Class 3)

- [ ] Append-only typed-event sink per the integrity model (Q-L6-3), timing at the envelope
- [ ] Persistence boundaries per event class (Q-L6-4); fail-closed unavailability behavior
- [ ] L1/L2/L3/L4/L5 event wiring
- [ ] Security review

## 3. L6-M3 — Reconstruction + hand-off (Class 3)

- [ ] Cold-reconstruction verifier (Q-L6-5 contract)
- [ ] Themis hand-off shape (Q-L6-8); ArtifactStore address binding
- [ ] Security review

## 4. L6-M4 — Proof + close (Class 2)

- [ ] Live proof per Q-L6-9 (complete run + killed run, both verified cold)
- [ ] Traceability, coverage-verified; reviews with three-state verdicts
- [ ] Code map + status doc + artifact updates; push/archive on owner approval

## 5. Deferred (NOT L6 scope)

- Checkpoint/resume; database substrates; metrics/telemetry pipeline; query API beyond the hand-off seam; GC implementation (vocabulary only); L7 loop
