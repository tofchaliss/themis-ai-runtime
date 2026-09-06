# Tasks: Layer 3 — Context Management

Execution starts only after the grill closes Q-L3-1…9 and the owner accepts. Every milestone review states three independent verdicts: architecture-conformant · test-evidenced (coverage-verified on claimed lines) · operationally proven. An editing command or commit message is never evidence.

## 0. Gate

- [x] Grill session held 2026-09-06; Q-L3-1..9 closed (Q5/Q6 deliberately deferred to dedicated grills) and recorded in design.md §3
- [x] Owner ACCEPTED 2026-09-06 (all gate criteria GREEN); autonomous execution granted with the standing constraint: omitted_for_capacity never becomes silent degradation — unfittable required/undroppable set fails closed

## Binding constraints (from the grill — govern all implementation)

Deterministic policy execution, zero runtime judgment (rank/drop functions receive ItemRef only, never evidence bytes) · capacity is never authority (required never droppable; optional governed undroppable by default; droppability = explicit named contract relaxation; every governed slot declares one of four dispositions) · fifth availability state `omitted_for_capacity` (state-only model marker; counts/mechanics trace-only) · rank ≠ render (presentation stays contract slot order) · no truncation of evidence · dedup within class only, provenance multiplicity retained; cross-class never collapsed or promoted · budget lives in the ManagementPolicy; overrides are L7-era governance acts · model requests carry zero rank weight · no probabilistic component and no compressor without its own grill

## 1. L3-M1 — ManagementPolicy artifact (Class 3) — **DONE 2026-09-06**

- [x] Policy schema + fail-closed loader (L1/L2 artifact posture: hash, unknown fields, trailing bytes, floors)
- [x] Rank keys (closed legal set: size_asc/size_desc/kind — "version" removed per arch F2: lexicographic ≠ recency), dedup mode, drop order, token budget; droppability lives in the CONTRACT per Q-L3-2, not the policy (arch F4 record fix)
- [x] Policy hash into the delivery trace
- [x] Security review done (see §4a)

## 2. L3-M2 — Manage: filter/rank/dedup/budget (Class 3) — **DONE 2026-09-06**

- [x] `Manage(policy, gathered) (managed *Gathered, trace SelectionTrace, err)` — deterministic, judgment-free execution
- [x] Required/undroppable slots excluded from drop candidates; unmeetable budget ⇒ fail closed
- [x] `omitted_for_capacity` state (Q-L3-3): state-only model marker at slot granularity; counts/ranks/policy hash trace-only
- [x] Within-class hash-identity dedup with full collapsed source refs in trace; cross-class duplicates delivered separately (Q-L3-8)
- [x] Deterministic token estimator per Q-L3-5/D-L3-5
- [x] Security review done (see §4a)

## 3. L3-M3 — Retrieval sources (Class 2) — **DONE 2026-09-06**

- [x] Lexical search source (KindSearch: confined walk, regular-files-only, sorted, capped-with-refusal) + filesystem/metadata filtering via existing sources as registered L2 sources (external-untrusted, confined)
- [x] Table tests: classification, confinement (incl. symlink escape), determinism, over-cap refusal

## 4. L3-M4 — Seams + proof (Class 2) — **DONE 2026-09-06**

- [x] Compressor registration seam only (Q-L3-6 deferral recorded: no activation without a dedicated grill)
- [x] Selection trace documented for L6 (SelectionTrace in manage.go); L7 policy-selection contract stated
- [x] Q-L3-9 proofs done: no-pressure byte-identical passthrough + golden managed set (deterministic); live budget-pressure run with omitted_for_capacity marker + survivor citation
- [x] Traceability table (`traceability.md`)

## 4a. Security review record (M1+M2, 2026-09-06)

Class-3 review with compiled PoCs. Verdict: the authority boundary holds — no combination evicts required or undroppable content. Two HIGH integrity defects found and remediated: **HIGH-1** phantom-drop accounting (ranking pre-dedup refs while budgeting post-dedup let duplicate bytes falsify BudgetUsed and bypass ErrBudget; refs/items could diverge so PayloadHash attested undelivered items) → rewritten to index-bijective post-dedup accounting, refs rebuilt from kept items, regression test locks the exact PoC. **HIGH-2** KindSearch followed file symlinks outside the confinement root (host-file exfiltration into model context) → non-regular files hard-refuse ErrConfinement + reads routed through confinedPath; regression test. **MEDIUM-1** silent 64-match cap → over-cap now refuses deterministically (no-silent-caps rule). **LOW-1** collapse-order tie → Authority added to sort key. **LOW-2** collapsed items' Kind/Version now retained in CollapseRef. Informational accepted: estimator excludes framing (MaxComposedBytes is the backstop), policy↔contract binding is caller convention until L7, search scan skips oversized files.

## 5. Deferred dependencies (NOT L3 scope)

- Semantic dedup/retrieval, embeddings, rerankers, compressors (implementation), vector stores
- L6 trace sink; L7 policy selection + budget override channel
- Provider-exact tokenizers (registered computation, later)

## 5b. Close-out review record (2026-09-06)

**Test review three-state verdict:** architecture-conformant YES · test-evidenced PARTIAL→closed (HIGH-A re-entry erasing the omission marker — confirmed empirically, fixed by refusing re-management of a managed set + regression; HIGH-B all rank keys now table-tested; dedup+drop bijection under real pressure; multi-source collapse w/ Authority tiebreak; full-omission SlotState assertions; estimator pinned exactly (incl. used=111); golden managed-payload hash; search dir-symlink + oversized-scan-skip tests; policy loader edges) · operationally proven YES (live pressure proof hardened with sentinels: survivor token echoed by the model, dropped token provably absent from payload AND reply; ≥1 real drop asserted).

**Architecture review three-state verdict:** conformant, no boundary violations, live-re-verified independently. F2 version rank key removed (wisdom trap); F4/F5 record/comment drift fixed; F6 mechanism-stamp noted for the L4 era; F3 accepted (policy attestation lives in SelectionTrace until L6/L7 wire Compose↔Manage binding).

**Arch F1 RESOLVED by owner 2026-09-06 — CONFORM, no delta.** Shipped interpretation kept: every governed slot has a deterministic effective disposition; explicit `droppable` is required for capacity-driven omission; absence of that declaration resolves to undroppable. An accidental omission can only make the system LESS aggressive in dropping evidence, never more. Separately locked: a workflow contract may not rely on undeclared behavior to exclude governed evidence. Reject-if-undeclared was considered and rejected (brittle authoring without a security gain — capacity must never become the authority that decides governed evidence can disappear, and the default guarantees exactly that).

## 6. Close

- [x] Architecture + security + test review complete, three-state verdicts recorded (§4a, §5b)
- [x] Code map updated (L3 → done)
- [x] Owner resolved F1 (CONFORM, 2026-09-06)
- [ ] Green checkpoints pushed on owner approval; archive on close
