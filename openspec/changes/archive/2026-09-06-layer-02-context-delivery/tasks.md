# Tasks: Layer 2 — Context Delivery

Post-grill 2026-09-05: all ten grill questions closed in design.md §3, which governs where the draft decisions conflict. **Implementation has NOT started and does not start until the owner accepts this folded change (gate 0).** Every milestone review states three independent verdicts: architecture-conformant · test-evidenced (coverage-verified on the claimed lines) · operationally proven. An editing command or commit message is never evidence.

## 0. Gate

- [x] Grill session held 2026-09-05; Q-L2-1…10 closed and recorded in design.md §3
- [x] Owner reviews the folded proposal/design/tasks — ACCEPTED 2026-09-06 with autonomous execution granted ("you dont need to wait for me each step")
- [x] Planning items resolved per the recorded recommendations (owner delegated):
  - **Filesystem confinement root:** orchestrator-supplied root per task (workspace dir); any path escaping it ⇒ hard refusal. (Recommend: yes, refuse symlink escapes too.)
  - **Caps (v1 constants):** recommend 256 KiB/item, 1 MiB total context, fail closed; revisit when L3 owns budgets.
  - **Themis connector:** recommend typed stub contract v1 (real wiring waits for the recorded `internal/service` → `integrations/themis` relocation).
  - **Operational-proof gate:** recommend mock-provider composition proof required + one live local-model run of a full L1+L2 payload (conversation only; tools-capable model still absent).

## Binding architecture constraints (from the grill — govern all implementation below)

Not tasks; constraints. Recorded in design.md §3: Themis workflow contract is the ceiling (`Plan ⊆ Contract`, fail closed) · L7 considers, L4 authorizes, L2 fulfills — no retrieval-time judgment, no second permission table, uniform L4 gate for planned and expanded context · contract-declared mechanical closure is the only expansion · typed absence (delivered/unavailable/not_applicable/withheld_by_contract; source-status ≠ delivery-status) · withholding governed truth only where the contract permits, marker = minimum epistemic metadata, never behavioral · verbatim evidence, no sanitization, no directive-pattern detection on the data plane · dual-reader framing (length-framed canonical record; delimiter-rendered model view with collision refusal; framing ≠ authentication) · L2 re-encodes, never creates propositions; derived propositions only from registered computations with complete computational provenance (incl. config hash) · four authority classes by authorship + governance treatment, fail-closed classification, immutable origin, transport carries zero authority weight · delivery + tool-execution traces jointly reconstruct every model-visible byte.

## 1. L2-M1 — ContextItem, envelope, classification (Class 2) — **DONE 2026-09-06**

- [x] `ContextItem` {kind, provenance{origin, source, author}, authority_class, producer, sensitivity, version/as-of, evidence bytes, hash}; `ContextEnvelope` {task_id, slot, source_status, delivery_status, delivery.mechanism}
- [x] Four-class `AuthorityClass` enum; classification derived from source/producer registration only; fail-closed default `external-untrusted`; caller-supplied class/hash overwritten
- [x] `ContextAvailability` vocabulary (4 states) + source-status/delivery-status split
- [x] Registered source kinds: inline task facts, confined filesystem, Themis connector (per gate-0 decision); unrecognized ⇒ hard error
- [x] Caps per gate-0 decision — fail closed
- [x] Table tests: negative paths, determinism, classification defaults, immutable-origin property

## 2. L2-M2 — Workflow context contract (Class 3 — security-sensitive) — **DONE 2026-09-06**

- [x] `ContextContract` artifact {version, hash, permitted classes, required/optional, authority classification, sensitivity ceiling} — versioned file, hash in trace. **Bounded claim (arch-review F4):** no production contract artifact ships in v1 (no real workflow exists yet); contracts are test fixtures until L7 registers real workflows — full pattern-policy parity lands then.
- [x] `Plan ⊆ Contract` enforcement, fail closed; required-missing ⇒ composition failure, no model call; optional-missing ⇒ typed unavailable
- [x] Withholding: `withheld_by_contract` only where the contract permits; marker carries existence + state only, never content or behavioral guidance
- [ ] ~~Mechanical closure rules executed; expansion ceiling enforced~~ **DEFERRED (arch-review F1, owner-visible):** v1 has no multi-hop connector and no capability fetch, so closure rules and the expansion ceiling are omitted from the contract schema rather than shipped as dead configuration; they join with a version bump at the L4/L7 era. This narrows the owner-locked artifact shape {…, closure rules, expansion ceiling} — deferral, not delivery.
- [x] Security review (contract enforcement, withholding, fail-closed paths)

## 3. L2-M3 — Composition, framing, delivery integrity (Class 3 — security-sensitive) — **DONE 2026-09-06**

- [x] Deterministic Compose: EIS system message verbatim (policy-hash-bound via L1 seam) + structured context rendering; sorted item order; byte-determinism golden + hash
- [x] Dual-reader framing: length-framed canonical record (ground truth for reconstruction) + delimiter-rendered model view with content-bound fences and collision refusal (refuse, never rewrite/encode; collision ≠ attack signal; refusal trace-visible only)
- [x] Frame metadata descriptive only — no behavioral text in L2 furniture (renderer-furniture rule, data-plane edition)
- [x] Hash-attributability: every delivered context byte belongs to an item's verbatim content or its lossless re-encoding — no L2-authored propositions (structural test)
- [x] No append-to-conversation API; fresh composition per epoch
- [x] Security review (framing integrity, proposition boundary, refusal paths)

## 4. L2-M4 — Trace + delivery proof (Class 2) — **DONE 2026-09-06**

- [x] Delivery trace {ContractHash, EISHash, RenderHash, PayloadHash, per-item: provenance/class/hashes/statuses/mechanism} documented for L6; joint delivery+tool-trace reconstruction contract stated for L7/L4 era. **Recorded deferral (arch-review F2):** with the candidate-skip fence, per-item framing refusal no longer occurs — fence exhaustion aborts the whole composition, so `DeliveryRefused` and `AvailabilityNotApplicable` are reserved vocabulary with no v1 producer; per-item refusal-status recording joins at the L6 trace-sink era.
- [x] Error classification consistent with L1's intake/resolution split
- [x] Mock-provider proof: composed payload byte-identical through `model.ExecutionRequest`
- [x] Operational proof per gate-0 decision
- [x] Traceability table: grill invariants → test names, coverage-verified

## 5. L1 content dependency — **DONE 2026-09-06; owner content review PASSED 2026-09-06** (accepted with push approval)

- [x] Revise `instructions/themis/tier-behavior.md` to the four-class vocabulary (`governed-external` placed explicitly: storage attestation governed, content external prose) — Class 2 + owner content review

## 5a. Security review record (M2+M3, 2026-09-06)

Class-3 review ran with adversarial PoCs. Three HIGHs found and remediated: **HIGH-1** metadata (Version) interpolated raw into frame headers enabled forged governed-record furniture → metadata charset/length validation (no control bytes, no brackets, bounded) refused as intake + fence redesign; **HIGH-2** per-item hash-prefix fences let one item's evidence close a sibling's frame → replaced with a **composition-wide content-derived candidate-skip fence** (deterministic, secret-free; embedding candidate N forces N+1; embedding all candidates requires a sha256 fixed point — not constructible; exhaustion refuses). This supersedes per-item collision refusal as the L2-5.5 mechanism — same invariant (deterministic, never rewrite/encode), stronger construction; flagged for owner visibility. **HIGH-3** evidence-byte caps allowed 23.8× framing-overhead amplification via many tiny items → item-count cap (512) + composed-size cap. **LOW-1** externally constructed Gathered bypassed validation → unexported validated flag, Compose refuses. Reviewer verified holding: classification spine, confinement (incl. mid-path symlinks), fail-closed gating, typed absence, verbatim evidence, canonical PayloadHash unambiguity, no DB/credentials, intake/resolution split, EIS verbatim.

## 6. Deferred dependencies (NOT L2 scope)

- **Contract schema growth (L4/L7, version bump):** closure rules + expansion ceiling (arch F1)
- **L6 trace sink:** per-item refusal-status recording; producers for `DeliveryRefused`/`AvailabilityNotApplicable` (arch F2)
- **First production contract artifact** under governance review (arch F4)

- **L3:** selection/rank/dedup/compression/token budgets
- **L4:** capability registry + authorization for model-requested expansion (L2 defines the fulfillment seam only)
- **L5:** worktrees/sandboxed retrieval; pinned-ref provenance
- **L7:** plan instantiation, sequencing, epoch succession, expansion consideration, mandatory registration policy
- **integrations/themis:** real Themis API read paths (per gate-0 stub decision)
- **Registered analyzers** (`derived`-class producers) beyond v1 built-ins; external web/browser connectors

## 7. Close

- [x] Reviews complete 2026-09-06 with three-state verdicts. **Security (Class 3):** 3 HIGHs remediated (§5a). **Test:** architecture-conformant YES / test-evidenced MOSTLY→closed (real fence-scan coverage, total-byte cap, non-inline metadata directions, Gather-path classification, order independence, golden payload hash d8ead1c5…, contract crumbs — all added; 97.9% coverage) / operationally proven YES (live run + evidence-echo starvation guard added). **Architecture:** conformant, no boundary violations; F1/F2/F4 record-honesty items resolved above; F3 contract slot-name checkMeta added; F5 withheld SourceStatus convention documented.
- [x] Architecture-to-code map updated (L2 → done)
- [x] Green checkpoints pushed on owner approval 2026-09-06; fence-mechanism supersession accepted with push approval — **LAYER 2 CLOSED**; change archived
