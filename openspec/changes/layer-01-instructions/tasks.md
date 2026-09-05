# Tasks: Layer 1 — Instructions

Post-grill 2026-09-05: all grill questions closed in design.md §6, which governs where earlier decisions conflict. Every group ends at a green, reviewed checkpoint (Stage-2 pipeline; Class 2/3 as marked).

## 0. Gate

- [x] Grill session held 2026-09-05; Q-L1-1…8 plus the grill-added instruction-identity question closed and recorded in design.md §6
- [x] Owner reviews the folded proposal/design/tasks; acceptance clears this gate — **ACCEPTED 2026-09-05**, including the central safety property: Layer 1 is not a security enforcement dependency; its failure may produce incorrect or incomplete instruction resolution but must not provide a path around deterministic authorization, security controls, verification, or Themis governance

## Binding architecture changes (from the grill — govern all implementation below)

Not tasks; constraints. Recorded in design.md §6: atomic resolution (no partial/degraded/fallback EIS) · namespace ownership with a trusted registry · three conflict classes (`shadowed`, `rejected-pattern`, `rejected-prohibited`) · immutable resolution epochs, change as recorded succession · no runtime data may alter resolved instruction text · pattern policy as a versioned security artifact with dual corpora · exemption ≠ authorization · role text as protected instruction `harness.system.role` · `ScopeRepository` in the enum, repository Sources rejected in v1 · renderer owns furniture, never rules.

## 1. L1-M1 — Representation and loading (Class 2) — **DONE 2026-09-05**

- [x] `Instruction`, `Scope` (seven ordered values), `Category`, `Source`, `Conflict` (three classes), `ResolutionStatus` types
- [x] **Namespace registry** (trusted configuration): namespace → owning source kind; unknown namespace ⇒ abort
- [x] Frontmatter + body parser (deterministic restricted subset, fail-closed on malformed files)
- [x] Loader: path-sorted recursive root walking, byte-exact bodies
- [x] Validation: id namespaced **and namespace-owned by the declaring source**, scope-legal-for-source, category, non-empty body
- [x] Secret scan on bodies (secret-guard pattern family, with must-pass prose evidence) — load-time hard error
- [x] Size caps: 8 KiB/instruction (64 KiB EIS constant defined for M4) — fail closed
- [x] `Load` rejects `ScopeRepository`/`ScopeDirectory`/`ScopeSkill` sources as unrecognized (v1 decision, Q-L1-1)
- [x] Test review passed (99.2% coverage); blocking findings remediated: `\b`-anchored `sk-` pattern (prose false positive), untrusted-path abort table, multi-source atomicity, `ErrSourceUnavailable` sentinel, untrusted `protected:` declaration aborts (fail-closed decision), inline provenance overwrite pinned, scope-order regression guard

## 2. L1-M2 — Conflict detection and pattern policy (Class 3 — security-sensitive) — **DONE 2026-09-05**

- [x] Ownership-violation rejection: foreign-namespace claim ⇒ `shadowed` + Conflict record, **regardless of body content** (delivered in M1)
- [x] Trusted-root duplicate id / trusted-root namespace violation ⇒ abort (delivered in M1)
- [x] Pattern engine loading `policies/security/instruction-directive-patterns.json` (versioned; hash on every LoadResult); unloadable/invalid/hollow policy ⇒ abort; nil/unvalidated policy ⇒ abort
- [x] Initial pattern family authored: 3 hygiene + 3 boundary (disable/bypass-control, preset-authority-attributes, exfiltrate-secrets); tier maps to conflict class only — per accepted design, patterns of any tier are exemptable, controls never are
- [x] **Dual corpus** (12 must-reject / 12 must-pass, grill-seeded), versioned in the policy file, re-verified at every load — a corpus-violating policy cannot activate
- [x] Exemption record type: all pins mandatory (operator, task, pattern id, policy hash, instruction hash, reason, timestamp, expiry); Config carries TaskID + Now (clock as deterministic input); invalid/stale/expired/task-mismatched ⇒ abort; content-pin mismatch ⇒ does not apply, rejection stands
- [x] Security review done; all findings remediated: HIGH-1 exemption no longer short-circuits scanning (continue-scan, every match independently exempted), boundary tier evaluated first (evidence never downgraded), decoy duplicate-id abort, mandatory pins, expiry/task binding, policy floors, trailing-bytes rejection. **Recorded for owner at M4 content review (LOW-8):** boundary patterns are broad for security-analysis phrasings ("analyze whether an attacker could bypass X") — remedy exists via exemption; consider growing must_pass before tightening. Known accepted evasions (LOW-9): >50-char gap padding, zero-width chars, "requires human decision" spelled with spaces — defense-in-depth limitations, not sole enforcement.

## 3. L1-M3 — Versioning, hashing, atomic resolution (Class 2) — **DONE 2026-09-05**

- [x] Canonical serialization (`themis-eis-v1`, separator-injective via validate(); spec test independently derives the bytes)
- [x] Per-body SHA-256 + EIS hash; `SourceHashes`, `Sources` recorded; hash covers delivered content only (conflicts/exemptions are trace data)
- [x] `ResolutionStatus` wired: `Resolve` sets completed/completed_with_conflicts; `StatusOf(err)` maps `ErrIntake`-wrapped (payload/exemption) failures → failed_intake, trusted/registration failures → failed_resolution; per-sentinel both-direction table tests
- [x] Determinism: golden hash literal + independent canonical spec, byte/protected/id sensitivity, source-argument-order independence (hash, instructions, sources, conflicts, exemptions), same-kind-root tiebreaks
- [x] Trace shape = `EffectiveSet{Hash, SourceHashes, Sources, Conflicts, PolicyHash, ExemptionsApplied, Status}` documented for L6
- [x] Test review passed; findings closed: Resolve failure-path atomicity, exemptions-through-Resolve evidence, multi-conflict determinism, source-registration errors reclassified as resolution failures (orchestrator config ≠ task payload), zero-source resolution fails closed (empty EIS is not a valid environment)

## 4. L1-M4 — Rendering and shipped content (Class 2 + owner content review) — **DONE 2026-09-05** (owner content review pending)

- [x] Deterministic renderer: role-first, fixed category headings, verbatim bodies; golden file (`testdata/render.golden`) + section-order and completeness assertions
- [x] **Renderer furniture rule** enforced structurally: H2 lines reserved for the renderer — a body containing "## " refuses delivery at any scope (counterfeit-furniture defense); unknown categories refuse rather than silently omit; empty sets refuse
- [x] **Final rendered-EIS validation pass** (scoping per the recorded design §2 amendment): size + secret scan on full delivered bytes; directive patterns on untrusted-body composition; detect/refuse only, never rewrite
- [x] Delivered-payload hash surfaced (`Render`/`SystemMessage` return it) for per-call trace recording
- [x] `instructions/global/` authored: role (DEC-06 neutral) + 6 protected safety instructions
- [x] `instructions/themis/` authored: 5 themis-domain instructions (3 protected)
- [x] **Owner review of instruction content** — PASSED 2026-09-05 (incl. the M2 LOW-8 pattern-breadth note)

## 5. L1-M5 — Delivery seam (Class 2) — **DONE 2026-09-05**

- [x] `Render(p)` / `SystemMessage(p)` with policy-hash binding; EIS → L2 contract documented in resolver.go/render.go
- [x] Delivery proof: rendered EIS through `model.ExecutionRequest` to a mock Ollama provider, byte-identical system message, exactly system+user in the payload, unicode/escaping round-trip
- [x] Traceability table: `traceability.md` — §17 criteria and grill invariants → test names (spot-checked honest by test review)

## 6. Verification obligations (cross-cutting; test-reviewer gate) — **DONE 2026-09-05**

- [x] Namespace violations: foreign claim, byte-identical-body claim — rejected identically (content confers no ownership)
- [x] Aborts: unknown namespace, trusted-root duplicate, missing registered root, unloadable pattern policy, cap breach — all yield no EIS
- [x] Atomicity: partial source availability never yields an EIS; Resolve failure path atomic
- [x] All three conflict classes recorded with id, scope, source, body hash, reason
- [x] Dual-corpus CI gates green; corpus-violating policy cannot activate
- [x] Golden render + golden EIS hash + independent canonical spec; final-pass scoping per design §2 amendment

## 7. Deferred dependencies (recorded, NOT Layer-1 implementation scope)

- **L5:** pinned-ref provenance machinery — precondition for activating the repository-source contract (design §6 Q-L1-1)
- **L9:** immutable registered skill identity binding `skill.<id>.*` namespaces; skill sources wired then
- **L2/L7:** epoch context boundaries, succession triggers, resolution-boundary state machine, emergency epoch termination; **mandatory-root registration contract** (production tasks must register the shipped safety/system/themis roots — role presence is orchestrator configuration, per the recorded F2 decision in design §2)
- **L6:** durable storage of trace metadata (EIS/conflict/exemption records)
- **Governance plane:** authenticated exemption-granting channel with the full record fields (design §6 Q-L1-2)

## 8. Close

- [x] Architecture + test review on the full change (2026-09-05): architecture verdict "faithful, disciplined realization; no boundary violations"; both reviews' findings remediated or recorded (design §2 amendments F1/F2 pending owner confirmation)
- [x] Architecture-to-code map updated (L1 → DONE)
- [x] Owner confirmed the design §2 implementation-driven amendments and passed the M4 content review (2026-09-05)
- [x] Green checkpoint commits pushed to origin/main on owner approval (2026-09-05) — **LAYER 1 CLOSED**
- [x] **Owner-requested code-level readiness review (2026-09-05, post-close):** independent design→code→tests audit of all §6 invariants + both §2 amendments. Verdict: every invariant CONFORMS with file:line evidence; recorded deferrals check out. The audit caught a real evidence regression: an earlier scripted test edit silently no-opped, shipping the M4 structural render refusals (H2-spoof, unknown-category, empty-set) and several assertions implemented but untested. Remediated with direct edits: structural-refusal + role-absent-pin tests (Render/SystemMessage now 100% covered), section-order/completeness/unicode-delivery assertions, conflict/exemption sort-tiebreak evidence, `Load` zero-source guard, `\x1e` injectivity row, corpus floor 12/12. AGENTS.md question answered precisely: no filename-based recognition exists in either direction — a prose AGENTS.md inside a trusted root aborts resolution (ErrMalformed, accepted insider-DoS trade); repository/directory/skill sources hard-reject. Layer doc `01-instructions.md` now carries a supersession banner naming the deferred §17 criteria.
