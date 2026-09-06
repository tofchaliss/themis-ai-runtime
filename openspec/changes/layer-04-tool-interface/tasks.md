# Tasks: Layer 4 — Tool Interface

Execution starts only after the grill closes Q-L4-1…9 and the owner accepts. Every milestone review states three independent verdicts: architecture-conformant · test-evidenced (coverage-verified on claimed lines) · operationally proven. An editing command or commit message is never evidence.

## 0. Gate

- [x] Grill session held 2026-09-06; Q-L4-1..9 closed and recorded in design.md §3
- [ ] Design accepted by owner
- [x] Owner decision: pull qwen2.5-coder:7b for the live seam proof (Q-L4-9, authorized 2026-09-06; test equipment, not standardization)

## 1. L4-M1 — Registry + grants (Class 3 — security-sensitive)

- [ ] `ToolRegistry` artifact + fail-closed loader (schemas, permission class, target rules, timeout, audit behavior, result trust class)
- [ ] `TaskGrant` artifact; grant ⊆ registry enforcement, fail closed
- [ ] Registry/grant hashes into the trace
- [ ] Security review (vocabulary closure, no authority-bearing verbs, registration jurisdiction)

## 2. L4-M2 — Authorization pipeline (Class 3 — security-sensitive)

- [ ] `Authorize(registry, grant, request)` — pure, deterministic, input-bounded by construction
- [ ] Strict schema validation: unknown field ⇒ whole-call rejection, recorded (Scenario-3 regression: authority-disposition args die here)
- [ ] Target validation per class (workspace-path via confinedPath; themis-id shape)
- [ ] Typed denials per Q-L4-5 (reason class model-visible, full reason trace-only)
- [ ] Decision vocabulary incl. `requires-approval` (reserved per Q-L4-6 decision)
- [ ] Security review

## 3. L4-M3 — Executors + result evidence (Class 3 — security-sensitive)

- [ ] Read-only executors: `read_file`, `list_directory`, `search_code` (L2 confinement), Themis read stubs (typed seam)
- [ ] Dispatch table (absent ⇒ impossible, not denied); no mutating/shell entries exist
- [ ] Results as classified evidence (registration trust class, verbatim bytes, hash) framed via the L2 discipline; typed error results per Q-L4-8
- [ ] Audit event per call/denial (D-L4-8 shape), documented for L6
- [ ] Security review folded into M2's or standalone as scope demands

## 4. L4-M4 — Seams + proof (Class 2)

- [ ] Model Interface loop fixture: ToolCall → Authorize → execute → tool Message round-trip against the scripted mock providers
- [ ] Capability-fetch mechanism wired to the L2 envelope (`MechanismCapabilityFetch` gets its producer); joint trace reconstruction stated
- [ ] Operational proof per Q-L4-9 owner decision
- [ ] Traceability table, coverage-verified

## 5. Deferred dependencies (NOT L4 scope)

- **OPEN-2 + L5:** `run_command`, sandboxed execution, all mutating tools (write/patch/commit/create/attach), approval-gate execution path
- **L6:** durable audit sink
- **L7:** request loop, retries, quota pacing, considers-step, epoch placement of tool results
- **Governance plane:** human approval channel for `requires-approval` decisions

## 6. Close

- [ ] Architecture + security + test review, three-state verdicts
- [ ] Code map updated; layer-status doc extended to L4; green checkpoints pushed on owner approval; archive on close
