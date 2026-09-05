# Tasks: Layer 2 — Context Delivery

Execution starts only after the grill closes Q-L2-1…8 and the owner accepts. Every milestone review states **three independent verdicts**: architecture-conformant · test-evidenced (coverage-verified on the claimed lines) · operationally proven. An editing command or commit message is never evidence.

## 0. Gate

- [ ] Grill session held; each Q-L2-n answered and recorded in design.md
- [ ] Design accepted by owner

## 1. L2-M1 — ContextItem, trust classes, registered sources (Class 2)

- [ ] `ContextItem`, `TrustClass`, `ContextSource` types; hash on construction; byte-exact content
- [ ] Source registration + unrecognized-kind hard error; trust class fixed per source (per Q-L2-1/4 decisions)
- [ ] Inline task-fact source (envelope data half); filesystem source with confinement root (per Q-L2-5)
- [ ] Caps per Q-L2-7 — fail closed
- [ ] Table tests: negative paths, determinism, caller-supplied hash/trust overwritten

## 2. L2-M2 — Composition + delimiter integrity (Class 3 — security-sensitive)

- [ ] Deterministic Compose: EIS system message verbatim (hash-bound), single structured user message, sorted item order
- [ ] Hash-derived fences; collision ⇒ refuse (detect/refuse only, never rewrite) per Q-L2-3 decision
- [ ] Structural separation proof: no context byte reachable in instruction text
- [ ] PayloadHash canonical serialization + golden; item-order independence
- [ ] Security review (framing, spoof defense, fail-closed paths)

## 3. L2-M3 — Trace + epoch boundary (Class 2)

- [ ] Delivery trace {EISHash, RenderHash, PayloadHash, ContextHashes, Sources, TrustClasses} documented for L6
- [ ] No append-to-conversation API; fresh-composition epoch test (no instruction-bearing inheritance)
- [ ] StatusOf-style error classification consistent with L1's intake/resolution split

## 4. L2-M4 — Delivery proof (Class 2)

- [ ] Mock-provider proof: composed payload byte-identical through `model.ExecutionRequest`
- [ ] Operational proof per Q-L2-8 decision (live local model if required)
- [ ] Traceability table: design invariants → test names, coverage-verified

## 5. Deferred dependencies (NOT L2 scope)

- **L3:** selection/rank/dedup/compression/token budgets
- **L4/L5:** tool execution, sandboxed retrieval, worktrees
- **L7:** epoch succession, mandatory-root + source registration policy, the model-call loop
- **integrations/themis:** real Themis API read paths (per Q-L2-6 decision)
- **External web/browser connectors:** controlled/later per layer doc

## 6. Close

- [ ] Architecture + security + test review, three-state verdicts each
- [ ] Architecture-to-code map updated (L2 → done)
- [ ] Green checkpoints pushed on owner approval
