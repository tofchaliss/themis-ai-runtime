# Tasks: skill identity vs instantiated composition identity (L9/L7 admission)

Grill closed 2026-09-22 (Q-SA-1..12; design.md §2 D-SA-1..10, §5 Gate 0
whitelist + twin suite, §6 attack record A-SA-1..11, §7 owner closure).
Architecture CLOSED; implementation NOT CONFORMANT. Three-state
verdicts per milestone; an editing command or commit message is never
evidence. The C17 lesson is the acceptance criterion: the positive
twin is a hard Gate-0 test and every negative must name its gate.

Owner-mandated sequence (finding → decision → amendment → implementation
→ positive-path proof → mutation tests → anchored live run) is now at
"implementation".

## 0. Gate

- [x] Finding recorded (`docs/development/finding-anchored-skill-admission-2026-09-15.md`, issue #1)
- [x] Owner decisions D-SA-1..10 locked; D-L9-13 amended (L7 reads the
      anchor-pinned catalog's identities, hashes only)
- [x] Gate 1: implementation design reviewed against §5 before code
      (2026-09-22; recorded in `RESUME-HERE.md`)

## 1. SA-M1 — Envelope: first-class `skill`; `origin` attribution-only (Class 3; L9 + L7)

- [x] `Envelope.Skill string` (`name@version`, schema-validated, exact
      ref syntax); L9 `Instantiate` stamps it; `origin` keys retained
      as attribution only (D-SA-5)
- [x] Coherence matrix enforced in `LoadEnvelope` (D-SA-5 table incl.
      D-L9-11d rows; `origin.skill` ≠ `skill` → refuse)
- [x] D-SA-3: anchored assembly refuses `skill_procedure_path` on an
      unattributed envelope (`ErrAssembly`); unanchored unchanged;
      wall test: sole `ActivateSkillSource` call site is in the
      corresponded path
- [x] Twins: attributed+corresponding admitted; unattributed+procedure
      refused (anchored) / admitted (unanchored); mismatching
      attribution refused

## 2. SA-M2 — Anchor `skills[]` (Class 3/4; additive G1 amendment)

- [x] `Anchor.Skills []string` (exact `name@version`; closed schema);
      `SubmitTask` anchored: `skill ∈ a.Skills` before catalog
      resolution, message names the allowlist (D-SA-9); absence admits
      no skill-attributed task
- [ ] `rsys@4` PROPOSED → Governance act (shares the L8 constitution
      re-pin; `delegation_template_registry` lands with L8 M6)
- [ ] `themis-status`/`themis-preflight` print `skills[]` — DEFERRED:
      both scripts print repo-derived pins and never read an anchor
      file; the anchor is its own record. Revisit with `rsys@4`
- [x] Twins: allowlist includes → ACCEPT; excludes → `ErrAssembly`
      allowlist message (never a correspondence message)

## 3. SA-M3 — Correspondence: D-SA-2 replaces `Seal != entry.Composition` (Class 3; L7)

- [x] `verifyAnchoredSkill` → resolve `env.Skill` in the anchor-pinned
      catalog (hash vs `a.SkillCatalog` first) → ACTIVE → manifest
      bytes two-way → compare the seven manifest pins against the
      commitment's seven fixed fields, equality per member, each with
      its own refusal text (D-SA-2 v); withdrawn → refusal (D-SA-8)
- [x] Seal has exactly two consumers (`checkSeal`, record); it is never
      compared with catalog/manifest/anchor values (D-SA-6) — AST wall
- [x] Twins: exact composition → ACCEPT; each single substituted member
      → REFUSE naming that member; mixed-bundle (A-SA-2, all of X's
      members under Y's name) → REFUSE naming `workflow`; withdrawn →
      REFUSE with the withdrawal message

## 4. SA-M4 — Instantiation: D-SA-4 `tools.Instantiates` (Class 3; L4 vocabulary, L7 application, L9 mirror)

- [x] `tools.Instantiates(effectiveRaw, templateRaw, taskID) error` — tool set
      equal; `max_calls`/`total_max_calls` in `[1, tpl]`; `mutating`,
      `themis_scope`, `template_scope` equal; `workspace`/`task_id`
      governed bindings; spec: placeholders substituted,
      `wall_deadline_s` ≤ tpl, all else equal
- [x] L7 anchored: read `grant_template`/`spec_template` from the
      **manifest pin** (confined under the catalog root, hash verified)
      — never from the commitment claim (D-SA-4 reference-source
      rule); apply `Instantiates`; then `grantWithinCeiling`
- [x] L9 applies the same relation at instantiation
      (non-authoritative; replaces the override-only check)
- [x] `grantAuthorityDigest` includes `TemplateScope` (shared with L8
      M2)
- [x] Twins: narrowed quotas + equal scope → ACCEPT; scope +1, scope
      −1, tool added, tool removed, `mutating` flipped, `max_calls`
      > bound, `total` > bound → REFUSE, each coupled to one clause;
      A-SA-4 reference-source rule holds STRUCTURALLY — `ResolvePin` is
      keyed by member name and no hash-keyed template resolver exists
      in the package (wall assertion, not a mutant; arch review MED-2)

## 5. SA-M5 — Claim-2 evidence completeness (Class 2/3; L6/L7)

- [x] Consumed catalog bytes and manifest bytes stored as
      content-addressed objects (dedup), referenced from the task
      record (`skill_catalog`, `skill_manifest`); `skill` recorded as a
      governed key
- [ ] Reconstruction of "ran the registered composition" from the
      record alone; current catalog state not an input (D-SA-8) —
      bytes retained (M5); a reconstruction VIEW consuming them is
      L10-side and not yet written (gap, see RESUME-HERE)
- [ ] Fault sweep at the new store points; record body reproducible

## 6. SA-M6 — Positive-path proof, live run, reviews, close (Class 3/4)

- [x] **Register B first (test harness):** the exact positive envelope
      admitted under a TEST-HARNESS anchor (`anchoredSkillWorld`, not
      `rsys@4`, not `themis-run`); every A-SA-1..10 negative refused with
      its intended message; every D-SA-4 clause mutation-coupled both
      ways (`tools/instantiate_test.go`, `TestAnchoredSkillNegativeTwins`)
- [x] Phase C row C17 retained (selector + admitting anchor) + positive
      counterpart recorded as C17+; deployment test plan updated
- [ ] Live anchored run on `rsys@4` submitting `remediate-dependency@1`
      — the first anchored skill execution (owner sequence, final step)
- [x] Three Class-3 close reviews in isolated worktrees against
      `dc8034c` (2026-09-22/23); findings and dispositions:
      - architecture MED-1 (catalog double-read) → single load + hash
        compare; MED-2 (task wording) → reworded + A-SA-2 tuple-swap
        test; LOW-1 (L9 spec mirror) → added; LOW-3 (C17 asserts a
        member) → done; LOW-4 (template bytes retention) → owner
      - security CRITICAL-1 (case-variant `"Workspace"` key passed the
        exact-key placeholder guard and bound a literal host path via
        the case-insensitive decoder) → `internal/strictjson` key wall
        (exact lowercase keys, no duplicates) before every grant/spec
        decode at L4, L5, L7, L9; `Instantiates` requires `@workspace`
        or absent (literal refused at parse); `instantiateGrant` exact
        entry-key allowlist + post-bind `Workspace == wsRoot` assertion;
        LOW-1 (`governed["skill"]` only when resolved; unanchored records
        `skill_claimed`) → done; LOW-2 (duplicate keys) → done
      - test HIGH (members 2–7, spec clauses, catalog mismatch,
        themis_scope, per-tool vs total quota) → added; MED (unanchored
        twins for the key wall, positive twin in-test) → added; live
        opt-in gating → kept the existing `THEMIS_LIVE_OLLAMA`
        convention (skip when unreachable), not changed here
- [x] Amendment records: L9 archive (D-L9-13 amendment; D-L9-11d matrix
      extension), G1 design (`skills[]`), L7 archive (assembly
      sequence)
- [x] `traceability.md`; archived 2026-09-23 under
      `openspec/changes/archive/2026-09-23-l9-l7-skill-admission-identity/`
- [x] `docs/harness-layer-status.md` and `execution-chain.md` updated;
      L8 dependency line marked discharged
- [ ] Close issue #1 — owner act (comments posted; label untouched)

## Deferred / residual

- **F-SA-2 → L11:** witness/grounding records carry `deployment_anchor`;
  consumption classifies `unanchored` evidence explicitly (D-L11-15)
- Residual C3 (D-L9-11a): governance actor attestation via signed
  registration — unchanged, not designed
