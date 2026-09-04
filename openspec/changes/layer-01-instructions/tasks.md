# Tasks: Layer 1 — Instructions

Execution starts only after the grill session closes Q-L1-1…8 and the owner accepts the design. Every group ends at a green, reviewed checkpoint (Stage-2 pipeline; Class 2/3 as marked).

## 0. Gate

- [ ] Grill session held; each Q-L1-n answered and recorded in design.md
- [ ] Design accepted by owner; deviations from the layer doc (if any) recorded

## 1. L1-M1 — Representation and loading (Class 2)

- [ ] `Instruction`, `Scope` (seven ordered values), `Category`, `Source`, `Conflict` types
- [ ] Frontmatter + body parser (deterministic, fail-closed on malformed files)
- [ ] Loader: filename-sorted root walking, byte-exact bodies
- [ ] Validation: id uniqueness/namespace, scope-legal-for-source, category, non-empty body
- [ ] Secret scan on bodies (secret-guard pattern family) — load-time hard error
- [ ] Size caps per Q-L1-6 decision
- [ ] Table tests: all loader/validation negative paths; determinism

## 2. L1-M2 — Conflict detection and resolution (Class 3 — security-sensitive)

- [ ] ID shadowing: higher scope wins; protected shadow → rejection + Conflict record
- [ ] Protected-directive pattern rejection for skill/task/repository scopes (list per Q-L1-2 decision)
- [ ] Resolver: stable (scope, id) ordering; source-argument-order independence
- [ ] Table tests: every pattern, every precedence pair, conflict-record completeness
- [ ] Security review (rejection semantics, pattern list, fail-closed paths)

## 3. L1-M3 — Versioning and hashing (Class 2)

- [ ] Canonical serialization of the ordered set
- [ ] Per-body SHA-256 + EIS hash; `SourceHashes`, `Sources` recorded
- [ ] Determinism property tests; golden hash; byte-change sensitivity
- [ ] Trace-metadata shape documented for L6 (`{EISHash, SourceHashes, Sources, Conflicts}`)

## 4. L1-M4 — Rendering and shipped content (Class 2 + content review)

- [ ] Deterministic renderer: role-first, fixed category headings, verbatim bodies; golden file
- [ ] `instructions/global/` authored (role, evidence-only, untrusted-data, no-credential, verification, advisory-only, worktree confinement)
- [ ] `instructions/themis/` authored (security truth, release context, advisory-unless-governed, not-a-second-VAMS, Tier behavior)
- [ ] Owner review of instruction content (it is the agent's constitution-in-prompt)

## 5. L1-M5 — Delivery seam (Class 2)

- [ ] `Render()` / `SystemMessage()` convenience; EIS → Context Delivery contract documented
- [ ] Delivery proof: EIS through `model.ExecutionRequest` against the mock providers (optional live proof if a model is available)
- [ ] Layer-doc §17 acceptance criteria → test-name traceability table

## 6. Close

- [ ] Architecture + test review on the full change
- [ ] Architecture-to-code map updated (L1 status EVOLVE+BUILD → done)
- [ ] Green checkpoint commits pushed on owner approval
