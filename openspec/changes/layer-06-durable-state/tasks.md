# Tasks: Layer 6 — Durable State

Grill closed 2026-09-07 (Q-L6-1..11, all CLOSED; design.md §3 is the record, §2 the folded decisions). Execution starts only after owner Gate 0. Three-state verdicts per milestone: architecture-conformant · test-evidenced · operationally proven. An editing command or commit message is never evidence. The architecture defines the milestones — never the reverse.

## 0. Gate

- [x] Grill session held; Q-L6-1..11 answered and recorded
- [x] Decisions folded into design.md §2 (D-L6-1..11 + residuals)
- [x] Gate 0: design accepted by owner (2026-09-07, "Let us implement L6") (audit vs the eleven closures: dropped decisions, accidental new authority, dead configuration, implementation-over-architecture, L1–L5 contradictions)

## 1. L6-M1 — Object store + task manifest (Class 3)

- [x] **Crash-safe object publication FIRST** (D-L6-11) in the L6 object store — lands before any retry/idempotence claim
- [x] L5 ArtifactStore inherits the corrected publication discipline (closed in the review-remediation round)
- [x] `StoreObject(class, provenance)` primitive: closed two-class vocabulary, verify-on-read, algorithm-prefixed identity, no Update/Delete
- [x] Task manifest: single-use exclusive-create identity, closed monotonic machine, recovery-only states not caller-requestable, atomic replace, constitution hash recorded
- [x] Root pairwise-disjointness check at task assembly
- [x] Security review (combined L6 review: 2 HIGH + 3 MED + hardening, all remediated with regressions)

## 2. L6-M2 — Trace sink + event wiring (Class 3)

- [x] `AppendEvent`: sink-assigned sequence, length-framed + per-entry hash, reference validity at the door, terminal-on-failed-append, UTC annotation at append
- [x] Synchronous floor commits (D-L6-10 record-before-effect): L4 audit before result delivery, L2 delivery record before payload delivery, lifecycle event before manifest projection
- [x] Event classes + caller-side glue: L1/L4/L5 wired in the live proof; L2-delivery/L3-selection classes exist with unit coverage (full task-loop glue is the L7-era caller's, per D-L6-4); flag-only secret-pattern scan at the sink
- [x] Stream summary into manifest at terminal transitions
- [x] Security review (combined; frame cap, junk-in-frame, live-writer guard remediated)

## 3. L6-M3 — Recovery + cold verifier + read surface (Class 3)

- [x] Recovery: projection completion + reconciliation events, FAILED_PARTIAL (event-first), torn-tail preserve-aside, idempotent
- [x] Cold verifier: full re-derivation (objects, entries, summary, projection legality — unexplainable manifest state ⇒ CORRUPT), verdict events, **scan-completeness verdict**
- [x] Read surface: audit primitives with inseparable (bytes, provenance, classification); structurally content-free `StatusView`
- [x] Security review (combined; laundering gate, projection re-derivation remediated)

## 4. L6-M4 — Proof registers + close (Class 2/3)

- [x] Register A structural suite (API closure, deletion absence, no model verb, disjointness, edge products, relocatability)
- [x] Register B behavioral suite (branch-pinned adversarial matrix incl. recovery idempotence)
- [x] Register C: exhaustive fault-point sweep + real-kill live proof + byte-exact cold reconstruction
- [x] Reviews: security + test + architecture — all findings remediated (4 test-review HIGHs now branch-pinned; amendments 1-8 recorded in design.md); coverage 83.9%
- [ ] Code map + status doc + artifact updates; push/archive on owner approval

## 5. Deferred (NOT L6 scope — recorded IOUs, never silently promoted)

- GC implementation + maintenance stream + governed retention artifact (invariants locked; retain-all v1 ships no deletion path)
- Themis transfer protocol + external anchor (seam shapes recorded in D-L6-8)
- Write-behind persistence (per-class grill required)
- L2 "durable-record source" kind (L2's grill); L11 evaluation-from-record (its own grill)
- Read auditing (needs a real access boundary); monotonic/duration timing
- L8 differently-trusted principals (isolation belongs at the L8 boundary)
