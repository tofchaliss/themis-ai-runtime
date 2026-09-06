# Design: Layer 3 — Context Management

**Inputs:** `docs/architecture/harness/00-p0-architecture-v2.md` §6 (layer doc: Filter → Rank → Deduplicate → Compress → Token Budget), the closed L1+L2 designs (archived changes — their invariants bind), `ARCHITECTURE.md`, the architecture-to-code map note "L3: assembly/limits/provenance/lifecycle — **not security authority**".
**Decision IDs** `D-L3-n`; **open questions** `Q-L3-n` — the grill's targets. DEC-05 applies: the layer doc's DeepSeek terminal is illustrative; nothing here is provider-specific.

## 0. Position in the flow

```
L2 Gather ─► Gathered (typed, classified, contract-validated)
                 │
                 ▼
             L3 Manage: filter → rank → dedup → compress → budget
                 │            (governed policy, deterministic-first)
                 ▼
             Managed set (same typed vocabulary + selection trace)
                 │
                 ▼
             L2 Compose ─► payload
```

L3 transforms between Gather and Compose. It never gathers (L2), never authorizes (L4), never calls the reasoning model (L7).

## 1. Hard invariants (inherited, not grillable)

- **Selection ≠ withholding:** contract-permitted withholding is the only way governed truth is deliberately excluded; L3 dropping is a capacity decision, must be typed and trace-recorded, and can never touch required slots.
- **Verbatim originals:** compression never rewrites an item; it produces a NEW `derived`-class item (registered computation, full provenance incl. config hash) referencing the original hashes; the original remains trace-reconstructable.
- **Proposition boundary:** L3 authors no security propositions; a summary that adjudicates ("the VEX overrides the scanner") is outside every summarizer's permitted vocabulary.
- **Probabilistic components never decide availability:** embeddings/rerankers (if introduced) are registered computations whose output is advisory ordering; the deterministic policy decides what is dropped.
- **Typed states only:** every L3 outcome extends the L2 availability/delivery vocabulary — nothing is silently absent.
- **L3 failure is bounded:** it can starve or misorder model input, never bypass authorization, verification, or governance.
- Determinism: same inputs + same policy artifacts ⇒ same managed set ⇒ same payload hash.

## 2. Draft decisions (grill targets)

### D-L3-1 — ManagementPolicy is a governed artifact

`{version, hash, per-slot: max items/bytes, rank keys, dedup mode, compression eligibility, token budget, drop order}` — versioned file, hash recorded in the delivery trace beside the contract hash. Policy is Themis/owner-authored (contract posture); L7 selects which policy applies; L3 executes it without judgment beyond the policy's own rules.

### D-L3-2 — Drop order is declared, never inferred

Budget pressure drops in policy-declared order (e.g. optional slots by rank ascending), never by any runtime heuristic. Required slots and any slot the policy marks `undroppable` are excluded from drop candidates; if the budget cannot be met without them ⇒ fail closed (`ErrBudget`), no degraded package.

### D-L3-3 — Typed drop states

New delivery-status value `dropped_by_budget` (trace) with model-visible availability `unavailable`? — or a new model-visible state `omitted_for_capacity`? (Q-L3-3). Either way: recorded per item with reason, rank, and policy hash.

### D-L3-4 — Dedup is hash-identity only in v1

Byte-identical evidence (same hash) across items/slots collapses to one delivery + trace records the collapsed refs. Semantic near-dup detection deferred (would be a probabilistic judgment).

### D-L3-5 — Token counting is deterministic and provider-neutral

v1 counts bytes/estimated tokens via a fixed deterministic estimator (recorded in the policy); provider-exact tokenizers are a later registered computation. Budgets bind on the estimate; the composed-size caps remain the hard backstop.

### D-L3-6 — Compression enters as a registered computation only

No compressor ships in v1 (layer-doc "later"). The seam is defined now: a `Compressor` registration {producer id, version, algorithm/config hash, permitted proposition vocabulary: descriptive summary only}, output = `derived` item {refs: original hashes}. Original items remain in the trace even when only the summary is delivered — delivered-summary + trace-originals must reconstruct what the model saw AND what it stood for.

### D-L3-7 — Retrieval capabilities are L2 sources, not L3 features

Repo search (ripgrep), lexical retrieval, metadata filtering produce candidate items through registered L2 sources under existing classification (external-untrusted for repo content). L3 then selects among them. Semantic retrieval (pgvector/embeddings) deferred.

## 3. Grill record (2026-09-06)

### Q-L3-1 — Where L3's judgment ends (CLOSED)

Owner-locked rule:

> L3 runtime selection is deterministic policy execution. Runtime MUST NOT introduce semantic judgment, policy, or security preference not declared by an owned, versioned context policy. Every ranking dimension, priority, eligibility rule, and tie-break must be declared by the applicable policy/contract and mechanically computable from authorized metadata.

Accepted sharpenings: **(1) type-system boundary** — rank/drop functions receive `ItemRef` only, never evidence bytes; content-dependent relevance can only ever arrive as a registered-computation-produced metadata field with Q-L2-6 provenance. **(2) Structural-vs-wisdom split** — the validator proves the policy mechanically valid (declared dimensions, total drop order, complete deterministic tiebreaks); whether the priorities are *sensible security policy* is Class-3 human review of the artifact. **(3) Model requests carry zero rank weight** — the L7→L4→L2 path retrieves; the retrieved item ranks by policy dimensions like any other; request history lives in the trace. No human-like tie-breaks at runtime; declared tiebreak chain only.

### Q-L3-2 — Governed truth vs capacity (CLOSED WITH AMENDMENTS)

Owner-locked formulation:

> **L3 controls capacity, not authority. A context item may leave the model's view because the contract explicitly permits its exclusion, never merely because capacity became insufficient.**

The twelve principles: required context never droppable · required exceeding capacity ⇒ fail closed · optional governed records undroppable by default · droppable governed records require explicit contract authorization · capacity never grants authority to omit governed truth · capacity omission gets its own typed state · source availability ≠ delivery status · omission decisions deterministic and traceable · droppability is a governed-artifact relaxation, never an L3 runtime choice · model requests contribute zero ranking weight · omission counts and drop mechanics are trace-only by default · L3 does not truncate evidence as an incidental capacity optimization.

Amendments (owner-added):

- **Explicit governed-inclusion declaration:** the contract validator requires every governed slot to resolve explicitly to one of `required/undroppable` · `optional/undroppable` · `optional/droppable` · `excluded_by_contract`. No implicit default based on capacity; undeclared droppability is rejected. (No universal at-least-one-undroppable floor — some analytical workflows legitimately run without governed records.)
- **Droppability is a named reviewable relaxation** (Class-3 posture, artifact hash in trace) — "everything droppable" must be visible at review, never emergent from runtime pressure. Validator rejects undeclared droppability but not many-declared droppability.
- **No truncation:** L3 drops whole items or contract-defined units; it MUST NOT truncate evidence bytes unless the truncation is itself an explicitly defined lossless/registered representation transformation. Doesn't-fit ⇒ omitted, never first-N-bytes (hash-attributability applied to capacity).
- **Covert-selection risk recorded:** a mechanically valid policy can still systematically shape evidence (governed history omitted, external advisories retained) — that is precisely what the Class-3 wisdom review of the policy artifact exists to catch.

### Q-L3-3 — Capacity-omission vocabulary (CLOSED; amends the locked Q-L2-4 enum)

Owner sign-off: fifth semantic state **`omitted_for_capacity`** joins delivered/unavailable/not_applicable/withheld_by_contract (exact enum naming finalized at implementation). `unavailable` would be factually wrong (source available, item retrieved, capacity omitted). Applied uniformly across authority classes. Distinction that must survive into the trace: `withheld_by_contract` = contract intentionally excluded; `omitted_for_capacity` = contract permitted dropping, deterministic capacity policy selected it. **Minimum disclosure:** the model-visible marker is state-only (no counts); omitted counts, rank boundaries, policy hash, and drop decisions are trace-only; a workflow needing model-visible cardinality declares it as an explicit contract field. This section is the recorded owner amendment to the archived L2 design's Q-L2-4 closure.

### Q-L3-4 — Ranking vs cognition (CLOSED)

**Rank ≠ render.** L3 ranking MUST NOT alter model-facing presentation order; rank feeds the drop decision only; presentation stays contract-declared slot order. Eliminates the rank-high → appears-first → attention-bias pathway entirely in P0. Rank-driven presentation, if ever wanted, is an explicit contract/rendering decision.

### Q-L3-5 — Probabilistic selection (DEFERRED — dedicated grill required)

P0 excludes embeddings, semantic retrieval, learned reranking, probabilistic relevance, and model-based context selection entirely. Only the registered-computation seam ships. **The existence of the seam does not pre-authorize probabilistic selection** — a dedicated architectural grill gates any future entry.

### Q-L3-6 — Compression (DEFERRED — seam only)

Compression ≠ normalization: a summarizer selects, changes emphasis, and may introduce propositions; "descriptive-only summarizer" safety is unestablished and P0 does not need it. Ships only the registration seam {producer, version, configuration hash, original item hashes, derived-output hash} with originals always reconstructable in the canonical trace. **No compression implementation may activate without a dedicated grill of its proposition vocabulary.**

### Q-L3-7 — Budget ownership (CLOSED)

Contract = what evidence matters · ManagementPolicy = how much fits (budget, ranking/drop policy, caps, version/hash) · L7 = which policy governs this execution. Different owners, different change cadence. A runtime budget override is a governance-plane operation belonging to the future L7 override mechanism, never an ordinary L3 parameter.

### Q-L3-8 — Cross-class dedup (CLOSED WITH AMENDMENT)

Owner-locked rule:

> Deduplication is permitted within the same authority class when evidence bytes are identical, but all collapsed provenance references remain trace-visible. Evidence with different authority classes MUST NOT be collapsed into one class or have one class promoted over another. P0 does not physically deduplicate cross-class evidence in model-facing rendering; any future lossless representation optimization must preserve every authority/provenance relationship without changing evidence bytes.

Grounding: same bytes + different provenance = different evidence relationships (Q-L2-6); the duality is analytically meaningful and the model must be able to see it. **Dedup removes delivery redundancy, never provenance multiplicity** — within-class collapse retains every source ref in the trace (two sources independently supplying identical evidence is itself information). Budget consequence accepted for P0 (both copies cost capacity; ordinary drop policy applies); the multi-attestation rendering optimization is future-permitted but not built.

### Q-L3-9 — Operational proof (CLOSED)

Two required proofs: **deterministic** — golden managed set, plus the no-pressure passthrough (byte-identical through Manage, proving L3 doesn't quietly transform when capacity isn't binding); **live** — a budget-pressure run where declared-droppable items are deterministically reduced and the model demonstrably receives surviving evidence + the `omitted_for_capacity` marker and cites survivors.

### Grill board — P0 boundary CLOSED 2026-09-06

Q1 ✅ · Q2 ✅ · Q3 ✅ (amends Q-L2-4 vocabulary) · Q4 ✅ · Q5 ⏸ dedicated grill · Q6 ⏸ dedicated grill · Q7 ✅ · Q8 ✅ amended · Q9 ✅. Closing picture: **L2 determines what an item means as delivered data; L3 determines which already-defined items survive the context budget; neither layer decides what the security evidence means.**

## 4. Draft open questions (SUPERSEDED — resolved by the grill record above)

1. **Q-L3-1 — Where does L3's judgment legitimately end?** The L2 grill banned delivery-layer judgment; L3 exists to judge. Proposed line: L3 executes *governed policy* (like L1's resolver executes precedence) — all discretion lives in the policy artifact, none at runtime. Is that line right, and is rank-by-policy-keys genuinely judgment-free?
2. **Q-L3-2 — Can L3 ever drop a governed-record item?** Proposal: only if its slot is optional AND the policy marks it droppable — but is dropping governed truth for capacity ever acceptable, or must governed-record items be undroppable by class?
3. **Q-L3-3 — Model-visible vocabulary for capacity omission:** reuse `unavailable` (false: the source delivered) vs new `omitted_for_capacity` state (honest, but grows the locked Q-L2-4 vocabulary — needs owner amendment).
4. **Q-L3-4 — Ranking and cognition:** order affects model attention. Is rank order a pure policy concern, or does any ordering rule (e.g. governed-first) belong in the contract/instructions instead?
5. **Q-L3-5 — Probabilistic selection components:** admit embeddings/rerankers later as registered computations with advisory-ordering-only outputs — right seam, or should P0 architecture exclude them entirely until a dedicated grill?
6. **Q-L3-6 — Compression proposition vocabulary:** can any summarizer be trusted with "descriptive summary only," given summaries inherently select and emphasize? Is delivered-summary-plus-trace-original sufficient epistemic honesty?
7. **Q-L3-7 — Budget ownership:** policy artifact (proposed) vs contract vs L7 per-task — who sets the number, who may override per task, and is an override an exemption-like recorded act?
8. **Q-L3-8 — Dedup across authority classes:** byte-identical evidence appearing as both governed-external and external-untrusted — collapse to which class? (Proposal: deliver the higher-provenance copy, record both refs — but "higher" is an adjudication smell.)
9. **Q-L3-9 — Operational proof gate:** what proves L3 — budget-pressure scenario against the live model, or deterministic evidence only?

## 4. Interfaces to neighbors

- **← L2:** consumes `Gathered`, emits a managed `Gathered` (same type, same validation flag discipline) + selection trace; Compose unchanged.
- **→ L6:** selection trace {PolicyHash, per-item: kept/dropped/collapsed/compressed + reasons + ranks}.
- **→ L7:** policy selection per task; budget-pressure surfaced as typed failure, never silent degradation.

## 5. Test plan (three-state discipline)

Determinism/golden managed-set hash; drop-order table tests; required/undroppable protection (budget cannot evict them — fail closed); typed drop states; dedup collapse + trace refs; policy artifact fail-closed loading (L1/L2 posture); end-to-end: L2 delivery proof through L3 unchanged when no pressure; live budget-pressure proof per Q-L3-9.
