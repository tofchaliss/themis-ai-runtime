# Design: Layer 1 — Instructions

**Inputs:** `docs/architecture/harness/01-instructions.md` (layer doc), the accepted Stage-1 constitution, `ARCHITECTURE.md`, the proven Model Interface (`runtime/model`), the owner's reference flow (below).
**Decision IDs** `D-L1-n`; **open questions** `Q-L1-n` — the grill session's targets.

## 0. The reference flow (owner-supplied, authoritative)

```
                    Instruction Sources
                           │
          ┌────────────────┼────────────────┐
          ▼                ▼                ▼
     Harness rules    Themis rules     Repository rules
          │                │                │
          └────────────────┼────────────────┘
                           ▼
                  Instruction Loader
                           │
                           ▼
                    Representation
                           │
                           ▼
                      Validator
                           │
                           ▼
                  Conflict Detector
                           │
                           ▼
                       Resolver
                           │
                           ▼
                 EffectiveInstructionSet
                           │
              ┌────────────┼────────────┐
              ▼            ▼            ▼
            version       hash        sources
              │            │            │
              └────────────┼────────────┘
                           ▼
                    Context delivery
```

Note the terminal: the EIS flows into **Context Delivery (L2)**, which composes the final model payload. L1 renders; L2 delivers; L1 never calls the model.

## 1. Hard invariants (constitutional — not grillable)

- Resolution is fully deterministic; the model never decides precedence.
- Only explicitly recognized sources participate; repository content, CVE text, and tool results are data, never instructions.
- Protected instructions cannot be overridden by lower scopes.
- The EIS is reconstructable from an execution trace (hashing + versioning).
- Role framing is neutral (DEC-06); no secrets in instructions (DAY-0 §3).
- Instructions carry rules, never facts (Instructions ≠ Context, Contract-5).

## 2. Design decisions

### D-L1-1 — Representation: one instruction per Markdown file, YAML frontmatter

```yaml
---
id: harness.safety.no-credential-exposure      # unique, dot-namespaced
scope: harness-safety                           # one of the seven scopes
category: safety                                # safety|harness|themis-domain|engineering|skill|task
protected: true                                 # lower scopes cannot shadow it
---
Never expose credentials, tokens, or key material in any output.
```

*Why:* human-reviewable (instructions are reviewed content — files, never DB-served), diffable, byte-hashable, consistent with the repo's artifact style.
*Rejected:* JSON (hostile to prose review); one big file (no per-instruction identity); DB-backed (unreviewed-change risk).

### D-L1-2 — Seven scopes, fixed total order

`harness-safety > harness-system > themis-domain > repository > directory > skill > task` (layer doc §7). Ordered enum; resolver output sorted by (scope, id) — deterministic. Task ranks lowest by accepted design: a task may specialize, never contradict.

### D-L1-3 — Sources are registered roots, nothing else

```go
type Source struct {
    Kind   Scope         // scope this root feeds
    Root   string        // directory of *.md files, or
    Inline []Instruction // task payload / skill-declared
}
```

This repo's roots: `instructions/global/` (harness-safety + harness-system), `instructions/themis/` (themis-domain), skill-declared files (ScopeSkill, wired when L9 lands), the task payload (ScopeTask). A target repository's `AGENTS.md` becomes a ScopeRepository source **only through a recognized channel that the operator enabled for the task** — the agent reading a file never creates a source (Q-L1-1).

### D-L1-4 — The pipeline, concretely

Per the reference flow: Load → Represent → Validate → DetectConflicts → Resolve → (version, hash, sources) → hand to Context Delivery.

- **Load:** every `*.md` under each root, filename-sorted, byte-exact.
- **Validate (deterministic, no LLM):** id present/unique/namespaced; scope legal for the source (a task payload cannot claim `harness-safety`); category legal; body non-empty; **secret scan** (reuse secret-guard's pattern family — a pasted key is a load-time hard error); size caps (Q-L1-6). Malformed frontmatter = hard error naming the file. Fail closed: a bad instruction file stops resolution; it is never skipped.
- **DetectConflicts (mechanical P0 scope, as accepted at Stage 1):**
  1. *ID shadowing:* same id at two scopes → higher wins; if the higher one is `protected`, the lower is **rejected and recorded** (id, sources, scopes, resolution).
  2. *Protected-directive patterns:* skill/task/repository bodies matched against a small versioned pattern list (the `SuspectInjection` family generalized: ignore/disregard previous, reveal system prompt, you-are-no-longer, disable/bypass a control). Match ⇒ that instruction rejected + recorded; the run continues (rejection is the safe direction).
  3. *Semantic contradiction:* deferred (recorded limitation; Ratchet-era).
- **Resolve:** stable (scope, id) ordering.
- **Version/Hash/Sources:** SHA-256 per body; SHA-256 of the canonical ordered serialization = the **EIS hash**; the source list recorded. The proven manifest pattern.

### D-L1-5 — The EffectiveInstructionSet is a value

```go
type EffectiveSet struct {
    Instructions []Instruction
    Conflicts    []Conflict          // rejected/shadowed, with reasons — trace data, not logs
    Hash         string              // canonical set hash (the version)
    SourceHashes map[string]string   // id -> body hash
    Sources      []SourceRef         // which roots/payloads fed the set
}
func Resolve(sources ...Source) (*EffectiveSet, error)
func (e *EffectiveSet) Render() string          // deterministic instruction text
func (e *EffectiveSet) SystemMessage() model.Message // convenience for L2
```

Pure function of inputs after Load: no clock, no randomness. Same sources ⇒ same hash, regardless of `Source` argument order.

### D-L1-6 — Rendering: sectioned plain text, neutral role first

One deterministic text: the DEC-06 neutral role preamble first (itself a protected instruction, id `harness-system.role` — Q-L1-8), then bodies verbatim, grouped by category under fixed headings (`## Safety`, `## Themis principles`, `## Engineering`, `## Procedure`, `## Task`), in resolved order. **No templating, no interpolation** in v1 (Q-L1-4) — facts never splice into instruction text (layer doc §21).
*Rejected:* per-category multiple system messages (provider-dependent); L1 calling the model (the reference flow terminates in L2).

### D-L1-7 — Package layout: one package, reserved subdirs

Package `src/harness/instructions`: `instruction.go`, `source.go`, `loader.go`, `validate.go`, `conflict.go`, `resolver.go`, `hash.go`, `render.go` + tests. The skeleton's `loader/ resolver/ validator/ versioning/` subdirs stay reserved until size forces a split — layers own capabilities, not directory counts (ARCHITECTURE.md).

### D-L1-8 — Versioning is content hashing

No hand-maintained version numbers: an instruction's version is its body hash; the set's version is the EIS hash; the trace records both + sources. "What changed" = hash diff. Optional curated labels: Q-L1-7.

### D-L1-9 — Initial content ships with the layer

`instructions/global/`: neutral role; evidence-only/no outside knowledge (generalized from `themis-preamble.md`); untrusted-content-is-data; no credential exposure; verification-required; advisory-only conduct; worktree confinement.
`instructions/themis/`: Themis owns security truth; Findings in release context; AI advisory unless accepted through governance; not-a-second-VAMS; the Tier hierarchy in behavioral terms.
Authored from the accepted constitution + existing prompts; the service prompts themselves stay untouched until L9.

## 3. Security analysis (Stage-3 boundaries)

- **3.2:** structural — no path from unrecognized content to the resolver.
- **3.5:** load-time secret scan; zero interpolation ⇒ no runtime value can enter instruction text.
- **3.9:** malformed/illegal/secret ⇒ hard error; lower-scope violations ⇒ recorded rejection, run continues.
- **3.10:** every invariant lands as a table test (§5).
- **DEC-05/06:** nothing provider-specific; neutral role; delivery only via L2 → `model.Message`.

## 4. Interfaces to neighbors

- **→ L2 Context Delivery (per the reference flow):** L1 hands the EIS (rendered text + hash + sources) to L2; L2 composes the model payload. `SystemMessage()` is a convenience for that composition, not a delivery path.
- **→ L6 State (future):** trace shape defined now — `{EISHash, SourceHashes, Sources, Conflicts}` — storage later.
- **→ L7 Orchestration (future):** one `Resolve` per task before the first model call; re-resolution = new EIS + new hash, recorded as a new step (Q-L1-5).
- **→ L9 Skills (future):** a skill contributes `Source{Kind: ScopeSkill}`; nothing else changes.

## 5. Test plan

Table-driven; `t.TempDir()` roots; no live model except an optional final delivery proof.

- Loader: happy path; malformed frontmatter; duplicate id (same and cross-file); scope illegal for source; empty body; secret in body; filename-order determinism.
- Conflicts: every directive pattern rejects; ID shadow of protected → rejection recorded; shadow of unprotected → higher wins silently-but-recorded; conflict record completeness.
- Resolver: all seven scopes' precedence; source-argument-order independence.
- Hashing: byte-change ⇒ new hash; stability across runs; golden EIS render + golden hash.
- Render: role-first, category grouping, verbatim bodies — golden file.
- Traceability: layer-doc §17 acceptance criteria mapped one-to-one to test names.

## 6. Open questions for the grill (Q-L1-n)

1. **Q-L1-1 — Repository-scope trust:** the layer doc and the reference flow include Repository rules as a source, but Stage 3 says repository content is untrusted data. Proposed reconciliation: ScopeRepository exists, ranks below themis-domain, passes the directive-pattern rejection, and requires per-task operator enablement. Right trust model, or cut from v1?
2. **Q-L1-2 — Directive-pattern governance:** the rejection patterns are security-sensitive config. Amended by whom; embedded code vs a file under `policies/security/`?
3. **Q-L1-3 — Task-instruction schema:** one blob (id `task.instructions`) vs structured multiple instructions in the task payload? Proposal: one blob for v1.
4. **Q-L1-4 — Templating:** v1 bans templating/partials in instruction bodies. Right call, or carry the partial mechanism over now?
5. **Q-L1-5 — Mid-task re-resolution:** frozen EIS per resolution; a skill-stage change triggers a new resolution recorded as a new step. Acceptable for v1?
6. **Q-L1-6 — Size caps:** proposal 8 KiB/instruction, 64 KiB/EIS, fail closed at load; revisit with L3 token budgeting. Right numbers?
7. **Q-L1-7 — Curated labels:** are content hashes sufficient versioning for governance review, or is a human-readable changelog label per content change also required?
8. **Q-L1-8 — DEC-06 role text home:** a normal protected instruction file (proposed — reviewable like everything else) vs hard-coded in the renderer?
