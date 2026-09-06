# Tasks: Layer 3 — Context Management

Execution starts only after the grill closes Q-L3-1…9 and the owner accepts. Every milestone review states three independent verdicts: architecture-conformant · test-evidenced (coverage-verified on claimed lines) · operationally proven. An editing command or commit message is never evidence.

## 0. Gate

- [ ] Grill session held; each Q-L3-n answered and recorded in design.md
- [ ] Design accepted by owner

## 1. L3-M1 — ManagementPolicy artifact (Class 3 — security-sensitive)

- [ ] Policy schema + fail-closed loader (L1/L2 artifact posture: hash, unknown fields, trailing bytes, floors)
- [ ] Per-slot limits, rank keys, dedup mode, drop order, undroppable marks, token budget
- [ ] Policy hash into the delivery trace
- [ ] Security review (drop-order semantics, undroppable protection)

## 2. L3-M2 — Manage: filter/rank/dedup/budget (Class 3 — security-sensitive)

- [ ] `Manage(policy, gathered) (managed *Gathered, trace SelectionTrace, err)` — deterministic, judgment-free execution
- [ ] Required/undroppable slots excluded from drop candidates; unmeetable budget ⇒ fail closed
- [ ] Typed drop states per Q-L3-3 decision; every outcome trace-recorded
- [ ] Hash-identity dedup + collapsed refs per Q-L3-8 decision
- [ ] Deterministic token estimator per Q-L3-5/D-L3-5
- [ ] Security review

## 3. L3-M3 — Retrieval sources (Class 2)

- [ ] ripgrep/lexical/metadata-filter sources as registered L2 sources (external-untrusted, confined)
- [ ] Table tests: classification, confinement, determinism

## 4. L3-M4 — Seams + proof (Class 2)

- [ ] Compressor registration seam defined (no v1 implementation) per Q-L3-6 decision
- [ ] Selection trace documented for L6; L7 policy-selection contract stated
- [ ] End-to-end: L2 delivery proof through Manage (no-pressure passthrough byte-identical; pressure scenario per Q-L3-9)
- [ ] Traceability table, coverage-verified

## 5. Deferred dependencies (NOT L3 scope)

- Semantic dedup/retrieval, embeddings, rerankers, compressors (implementation), vector stores
- L6 trace sink; L7 policy selection + budget override channel
- Provider-exact tokenizers (registered computation, later)

## 6. Close

- [ ] Architecture + security + test review, three-state verdicts
- [ ] Code map updated; green checkpoints pushed on owner approval; archive on close
