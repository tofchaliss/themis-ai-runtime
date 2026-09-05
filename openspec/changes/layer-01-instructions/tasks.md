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

## 3. L1-M3 — Versioning, hashing, atomic resolution (Class 2)

- [ ] Canonical serialization of the ordered set
- [ ] Per-body SHA-256 + EIS hash; `SourceHashes`, `Sources` recorded
- [ ] `ResolutionStatus`: `completed` / `completed_with_conflicts` / `failed_resolution` / `failed_intake` — conflicts never look like ordinary success
- [ ] Determinism property tests; golden hash; byte-change sensitivity; source-argument-order independence
- [ ] Trace-metadata shape documented for L6: `{EISHash, SourceHashes, Sources, Conflicts, PatternPolicyHash, Exemptions}`

## 4. L1-M4 — Rendering and shipped content (Class 2 + owner content review)

- [ ] Deterministic renderer: role-first, fixed category headings, verbatim bodies; golden file
- [ ] **Renderer furniture rule** enforced in review: no behavioral directives in renderer-owned text
- [ ] **Final rendered-EIS validation pass**: secret + pattern scan on the exact delivered artifact; reject/record only — never rewrite (what was scanned = what the model received)
- [ ] Delivered-payload hash surfaced for per-call trace recording
- [ ] `instructions/global/` authored: `harness.system.role` (DEC-06 neutral), evidence-only, untrusted-data, no-credential, verification, advisory-only, worktree confinement
- [ ] `instructions/themis/` authored: security truth, release context, advisory-unless-governed, not-a-second-VAMS, Tier behavior
- [ ] Owner review of instruction content (the agent's constitution-in-prompt)

## 5. L1-M5 — Delivery seam (Class 2)

- [ ] `Render()` / `SystemMessage()` convenience; EIS → Context Delivery contract documented, incl. per-call EIS-hash recording
- [ ] Delivery proof: EIS through `model.ExecutionRequest` against the mock providers (optional live proof if a model is available)
- [ ] Traceability table: layer-doc §17 acceptance criteria **and design §6 grill invariants** → test names

## 6. Verification obligations (cross-cutting; test-reviewer gate)

- [ ] Namespace violations: foreign claim, byte-identical-body claim, attacker-source-identical-body claim — all rejected identically
- [ ] Aborts: unknown namespace, trusted-root duplicate, missing registered root, unloadable pattern policy, cap breach — all yield no EIS
- [ ] Atomicity: partial source availability never yields an EIS
- [ ] All three conflict classes recorded with id, sources, scopes, body hashes, resolution class
- [ ] Dual-corpus CI gates green; adding a corpus-violating pattern fails CI
- [ ] Golden render + golden EIS hash; final-pass scan operates on delivered bytes

## 7. Deferred dependencies (recorded, NOT Layer-1 implementation scope)

- **L5:** pinned-ref provenance machinery — precondition for activating the repository-source contract (design §6 Q-L1-1)
- **L9:** immutable registered skill identity binding `skill.<id>.*` namespaces; skill sources wired then
- **L2/L7:** epoch context boundaries, succession triggers, resolution-boundary state machine, emergency epoch termination
- **L6:** durable storage of trace metadata (EIS/conflict/exemption records)
- **Governance plane:** authenticated exemption-granting channel with the full record fields (design §6 Q-L1-2)

## 8. Close

- [ ] Architecture + test review on the full change
- [ ] Architecture-to-code map updated (L1 status EVOLVE+BUILD → done)
- [ ] Green checkpoint commits pushed on owner approval
