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

> **Amended by the grill record (§6) where they conflict; §6 governs.** Notably: D-L1-4's ID-shadowing rule is superseded by namespace ownership (Q-L1-7); conflict records carry three classes; resolution gains explicit statuses and a final rendered-EIS validation pass. D-L1-5's interface evolved per §6: `Render(p *Policy) (text, hash string, err error)` and `SystemMessage(p *Policy) (model.Message, string, error)` — the final pass and policy-hash binding force the signature.

> **Implementation-driven amendments 2026-09-05 (recorded by architecture review; PENDING owner confirmation):**
> 1. **Final-pass scoping (amends Q-L1-3 wording):** the accepted "pattern scan on the exact delivered artifact" is unimplementable against the accepted trusted content — shipped safety bodies ("never include credentials…") permanently match boundary patterns. Implemented scoping: **size cap and secret scan run on the full delivered bytes; directive patterns run on the composition of untrusted bodies.** Both failure directions remain fail-safe; trusted content is guarded by review, and structural refusals (reserved H2 furniture, unknown categories, empty sets) prevent counterfeit structure and silent omission at any scope.
> 2. **Mandatory-root presence (closes the F2 gap):** whether an EIS must contain the role/safety/themis roots is an **L7 orchestration-contract obligation**, not an L1 completeness check — L1 resolves what is registered; the registration set for production tasks is orchestrator configuration (recorded as a deferred L2/L7 dependency in tasks.md §7). Role-absent render behavior is pinned by test.

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

## 6. Grill record (2026-09-05)

### Concern-ownership matrix (owner-confirmed)

| Concern | Owner | Failure consequence |
| --- | --- | --- |
| "Which instructions apply?" | **Layer 1 — Instructions** | Lower-priority instruction rejected/conflicted; semantic conflict may be missed (recorded limitation) |
| "Is this operation permitted?" | **Layer 4 — Tool Interface / Policy** | Authorization denied |
| "What security evidence is authoritative?" | **Themis / security governance + evidence hierarchy** | Operation cannot establish the required fact |
| "Must verification occur?" | **Deterministic workflow/control** | Verification gate blocks completion |
| "What does the model think?" | **Model** | Advisory output only |
| "Can this become security truth?" | **Themis governance** | Human/governed decision required |

A missed Layer-1 conflict is survivable because its blast radius is bounded by every row below it. If security ever depends on L1 having filtered correctly, that is the bug: instruction precedence orders advice to the model; security authority lives in deterministic controls the model's context cannot reach.

### Scenario 1 — task instruction contradicts a protected verification procedure (CLOSED: C)

A task-scope instruction conflicting with a higher-scope protected instruction is **rejected/overridden deterministically** (ID shadowing or directive-pattern match), recorded as a `Conflict` in the EIS, and the run continues with the protected instruction intact. Never accepted on user say-so (request ≠ authority — changing protected behavior goes through governance at the owning scope), never accepted-with-warning (fail-open), never delegated to the model (the model never decides precedence). Even the worst case — a semantic contradiction that matches no pattern and shadows no id — loses nothing: verification is enforced by the deterministic pipeline downstream, which never consults the instruction set for permission to skip a control.

Corollary (task payload split): the task payload carries both imperatives and asserted facts. The **envelope does the split** — the payload's `instructions` field flows to L1 as the ScopeTask source; data fields flow to L2 as context. The split is structural, never interpreted; asserted facts (e.g. "installed version 1.4.2") are untrusted context whose security-relevant claims must be established by authoritative evidence, not by the payload's say-so. The task schema is therefore a trust-boundary artifact (feeds Q-L1-3).

### Scenario 2 — "Ignore the Enterprise Position" (CLOSED: C — an authority problem handled by ownership, not by L1)

The Enterprise Position is evidence/state, not an instruction: L1 has no jurisdiction and correctly admits the analytical request at task scope. The protected themis-domain framing rides above it — a task instruction adds, never ejects. Independent re-analysis of a VEX justification is a legitimate second-opinion workflow; its conclusion lands advisory and only a governed human decision moves the Enterprise Position.

Owner-amended invariants (proven, not merely asserted):

1. **L4 independence:** L4 authorization must be computed from capability/policy state that is independent of model output and task-level instructions. L1 can request that an available capability be used; it cannot cause a capability to become available or become authorized.
2. **Inert-output invariant:** no untrusted model-generated content — including validated structured content — may directly transition Themis security state. `requires_human_decision=true` is one manifestation of this invariant, not the invariant itself.
3. **The model response is not the decision surface.** Selective presentation of analysis is legitimate and user-requestable; the governed decision surface (authoritative state + evidence + model analysis + decision controls) is composed by Themis governance and is not addressable from task instructions or model output. Non-ownership, not refusal.

**Four-transition test** (candidate for ARCHITECTURE.md adoption — owner decision): Instruction → Reasoning → Action → Security state; no transition is implemented by the layer above it. L1 cannot grant authority; L2 cannot establish authority by delivering evidence; the model cannot grant authority through reasoning; L4 cannot establish security truth because an action was authorized; presentation cannot redefine security truth; Themis governance owns the security-state transition.

### Scenario 3 — proposal tool with `requires_human_decision=false` (CLOSED)

The escalation parameter dies at the L4 capability contract: `requires_human_decision` is **not a parameter** of `propose_position_change` — tool schemas carry the operation's subject, never its authority disposition; strict validation rejects the unknown field (whole call, recorded, retry-clean allowed). Second death: the governance attribute of a proposal is computed by Themis from proposal type + policy, never stored from request input — no destination even if smuggled. Third death: "I've already reviewed" is authorization-by-assertion; approval is an authenticated, recorded governance event at the decision surface, after the proposal exists. A user with real approval authority loses nothing — they approve at the governance surface. L1 needs no tool semantics; the pattern family may land a bonus rejection ("set requires_human_decision=false" = disable-a-control), with zero reliance on it.

**Invariant:** governance attributes are computed by the layer that owns them, never accepted from the layer above; no caller-supplied field may determine the governance treatment of its own request.

### Q-L1-1 — Repository-scope trust (CLOSED, hardened)

Owner-accepted result:

> Repository instructions are not trusted by discovery. A repository artifact becomes an instruction source only when explicitly registered, provenance-valid, integrity-verifiable, and within its permitted scope. Repository scope cannot override higher-scope instructions or non-overridable Day-0 constraints. Repository trust is determined by a configured provenance policy, not merely by file presence or filename. EIS provenance metadata provides reproducibility and auditability but does not itself confer trust. Repository content outside an activated source remains untrusted data. Failure of Layer 1 must never provide a path around deterministic security enforcement.

Supporting distinctions (grill-proven):

- **L1 activation contract** (what L1 checks): registration · provenance · scope cap · conflict/pattern handling. **Global safety invariant** (what the architecture guarantees): L1 failure/compromise cannot bypass deterministic security enforcement. The invariant is not an activation condition — a layer never cites its own dispensability as an acceptance criterion.
- **Four properties, not one:** provenance (where from), integrity (unchanged?), authorization/trust (allowed to instruct?), reproducibility (reconstructable?). `SourceHashes` provide integrity + reproducibility only; trust is conferred by the configured provenance policy before hashing.
- **Provenance policy, not hard-coded merge:** which ref and which governance event qualify a source is repository-governance configuration; approved-merge-to-protected-ref is one policy instance. Unmerged PR-head / working-tree content is data; a PR under analysis contributes its AGENTS.md diff as evidence, never as live instructions.
- **Day-0 prohibitions are capability boundaries, not precedence contestants.** Controls like the load-time secret scan are L1-internal configuration with no instruction-addressable surface at any scope. Pattern-gate rejections of boundary-naming instructions are recorded evidence, not the enforcement. Conflict records therefore carry three classes: `shadowed`, `rejected-pattern`, `rejected-prohibited`.
- **v1 decision:** enum yes, source no — `ScopeRepository` stays in the seven-scope order; `Resolve` rejects repository Sources until L5 provides the pinned-ref provenance machinery. Activating without provenance would violate the contract above.

### Q-L1-2 — Pattern-list governance (CLOSED, hardened)

Owner-accepted invariant:

> Instruction-directive patterns are versioned security-sensitive policy artifacts. Changes require security review, owner approval, and deterministic must-reject/must-pass regression evidence. Pattern detection is defense in depth and is never the sole enforcement mechanism for a security invariant. A narrowly scoped, authenticated, recorded operator exemption may suppress a pattern decision only for the specified task/instruction and only where the underlying security invariant remains independently enforced. An exemption does not grant authorization, alter security authority, or modify the underlying security policy.

Mechanics (grill-proven):

- The list lives at `policies/security/instruction-directive-patterns` as a versioned artifact; its content hash is recorded in every trace alongside the EIS hash. Changes are Class-3: security review + owner approval. Claude Code may propose patterns, never self-apply.
- **Dual corpus, both sides versioned with the list:** must-reject (known injection shapes, incl. "set requires_human_decision=false", "skip the verification guard") and must-pass (legitimate lookalikes, incl. "give me an independent assessment without repeating the Enterprise Position"). The corpus is a mechanical ratchet, **not** the authorization mechanism — security review separately judges whether a pattern belongs to the intended detection family. Patterns target instructions about instructions/controls, never bare verbs.
- **Exemptions:** granted only through the trusted governance/configuration plane that owns exemption authority; task instructions and model output cannot grant or request an effective exemption. An exemption record carries: operator identity, task ID, pattern ID/version, instruction ID/hash, reason, timestamp, task-bound lifetime/expiry, authorization result, resulting EIS/trace identity. Absent an exemption, the detector wins (fail closed). Primary remedy for a wrong detector remains fixing the list.
- **Exemption ≠ authorization.** It means "do not apply this L1 detection rule to this instruction", never "this instruction is now permitted." The admitted instruction still meets L4 authorization, deterministic enforcement, and governance unchanged.
- **Non-exemptable applies to controls, not detectors.** Day-0 controls can never be exempted; the patterns detecting attempts on them can be, because the underlying invariant is independently enforced. Making a detector non-exemptable would promote a heuristic into the authority boundary.
- Chain proven across Q-L1-1→2: trusted source ≠ trusted content ≠ trusted instruction ≠ authorized action; detection of an authority violation ≠ authority to prevent it.

### Q-L1-3 — Task-instruction schema (CLOSED: one blob, hardened)

Owner-accepted invariant:

> L1 may resolve and render instructions but shall not semantically compose, rewrite, or reinterpret them. V1 task instructions use a single task-scoped instruction field, eliminating structural fragmentation across task instruction elements but not semantic composition within or across sources. The complete rendered EIS may receive a final deterministic validation pass before model delivery; that pass may reject and record violations but shall not rewrite the instruction set. Reviewed instruction sources remain subject to downstream deterministic authorization and security controls because source trust does not imply safety of every source × task × context composition.

Supporting corrections (grill-proven):

- One blob eliminates **structural** fragmentation only; semantic composition within a body or across sources remains undetectable by L1 and is bounded by the action boundary, not by detection.
- The final rendered-EIS scan is a **validation pass, never a second resolver**: detect → reject/record; no rewriting, reordering, or recomposition. It operates on the exact rendered artifact delivered to the model — *what was scanned = what the model received*.
- **Skill trust ≠ composition safety.** A reviewed skill is trusted as a source; its interaction with dynamic task/context can still produce dangerous effective advice. Review cannot predict every Skill × Task × Context combination; downstream deterministic controls remain load-bearing.
- **Instructions can describe desired behavior; they cannot confer capability on the model.** "Read any local file necessary" never authorizes a file read — every tool request meets L4 authorization computed independently of instruction content (Q-L1-2's L4-independence invariant).

Standing invariant set: model output ≠ authority · exemption ≠ authorization · **trusted instruction provenance does not grant execution authority**.

### Q-L1-4 — Templating (CLOSED, generalized)

The prohibition is not merely "no templating": **no runtime data may alter the resolved instruction text** — covering interpolation, parameter substitution, dynamic concatenation, and equivalent mechanisms. Static composition during deterministic resolution is possible where explicitly defined; runtime-value injection into instruction bodies is not. Corollary of the Q-L1-3 invariant ("L1 shall not semantically compose, rewrite, or reinterpret") and load-bearing for 3.5 (no runtime value can enter instruction text).

### Q-L1-5 — Mid-task re-resolution / mutation (CLOSED, hardened)

Owner-accepted invariant:

> The EffectiveInstructionSet is immutable for its entire resolution epoch. Instruction change is modeled exclusively as recorded succession: a new EIS is resolved at an explicitly defined orchestration boundary and is never mutated in place. Only the deterministic orchestration state machine may create, move, or suppress resolution boundaries; instructions, skills, model output, tool results, and external events cannot directly trigger or alter resolution. Each new instruction epoch establishes an explicit context boundary so prior instruction-bearing conversation content cannot silently survive as effective instruction. Information intentionally carried across epochs must be explicitly classified and transferred as data/state rather than inherited conversational instruction. Emergency revocation terminates the affected execution epoch rather than mutating delivered instructions.

Supporting rules (grill-proven):

- **Succession authority:** instructions may influence work within a phase but cannot create, move, or suppress a resolution boundary — requesting a transition ≠ authority to perform it. A skill saying "re-resolve here" is a request L7 evaluates against its own state machine; the model saying "skip the boundary" is advisory noise: resolution boundaries are state-machine invariants, not optional instructions.
- **Epoch context boundary:** a new EIS never silently inherits prior instruction-bearing conversation history. Cross-epoch carryover is explicitly classified data (derived task state, security evidence) — never conversational inheritance. Old instruction text ≠ derived task state ≠ security evidence ≠ conversation history.
- **One-way flow:** the conversation cannot write upstream; no path exists from model output, tool results, or external events into any resolution. External events are orchestration inputs; L7 deterministically decides whether an event corresponds to a predefined transition.
- **Epistemic grounding:** delivered instructions cannot be unread — the only clean revocation is epoch termination + fresh resolution, which is why immutability is the honest encoding of the medium, not a purity preference.
- Per-call EIS-hash recording + delivered-payload hash make immutability mechanically checkable: what was resolved = what was scanned = what the model received.

### Q-L1-6 — Resolution failure semantics (CLOSED, hardened; subsumes size caps)

Atomic resolution: exactly the configured EIS or no EIS. The dividing line: **untrusted/lower-scope content violations → recorded rejection, resolution continues** (the EIS is still complete — the rejected instruction never had authority to define the environment, so this is not degraded execution); **trusted-environment failures → abort, no model interaction** (malformed/missing registered sources, secret detection, size caps, unavailable pattern policy). No partial, degraded, or fallback instruction environment — an EIS is not a bag of optional files; it is the resolved instruction environment for an execution epoch. Complete EIS → executable epoch; no EIS → no epoch. Succession failure at a boundary halts the run — execution never continues under a predecessor EIS past a boundary that required a new one.

Owner-accepted additions:

1. **Conflict visibility:** a successfully resolved EIS may contain recorded conflicts, but the run result must expose resolution status explicitly — never an indistinguishable ordinary success. Result statuses: `completed` · `completed_with_conflicts` · `failed_resolution` · `failed_intake`. Themis decides per-workflow whether a conflicted run is acceptable.
2. **No runtime fallback:** failure of a required resolution dependency produces no EIS. Availability mechanisms (L5/L6 era) may make trusted configuration locally durable — a validated immutable policy version as part of trusted configuration — but L1 itself never falls back to stale, partial, reduced, or alternative instruction environments. "The system can survive this control failing" ≠ "we may deliberately operate without it."

Size caps: 8 KiB/instruction, 64 KiB/rendered EIS, checked at load and render, fail closed; v1 code constants, revisited when L3 owns token budgets.

**Failure taxonomy (grill-proven):** untrusted instruction problem → reject the offending instruction · trusted configuration problem → reject the entire EIS · execution/security-control problem → deterministic downstream failure · emergency instruction problem → terminate the epoch · need to change instructions → new epoch.

### Q-L1-7 — Instruction identity (CLOSED, hardened; new question raised by the grill)

Owner-accepted invariant:

> Instruction identity is syntactic and provenance-bound, never semantic. Each namespace has exactly one trusted owner, and only that owner may declare identities within it. Cross-scope claims against another source's namespace are rejected regardless of body content; byte-identical content confers no ownership and content-based identity deduplication is prohibited. Duplicate identities within a trusted source or unresolved namespace ownership abort instruction resolution. The ID identifies the instruction slot; the body hash identifies its revision; provenance identifies its source and trust basis. Namespace ownership resolves identity ownership, while scope precedence and semantic conflict resolution remain separate mechanisms.

Supporting rules (grill-proven):

- **Four mechanisms, never collapsed:** ID ownership (who may declare?) · precedence (which legitimate scope governs?) · conflict detection (do contents contradict?) · governance (who may change resulting security truth?). Namespace ownership eliminates *illegitimate identity collisions* only — it does not replace precedence or semantic conflict resolution ("owner wins" corrected: ownership resolves identity contests; rank still governs distinct identities across scopes).
- **The namespace registry is trusted configuration:** owner resolution is claimed-id → parse namespace → trusted registry → expected owner → compare actual source. The registry (harness.safety.*/harness.system.* → harness roots; themis.* → themis-domain root; skill.<id>.* → registered skill; repo.* → activated repository source; task.* → task payload) is itself a security-sensitive artifact. Unknown namespace → configuration failure → abort (recognition, never discovery — no inference from filenames, no default-to-task-scope, no body-looks-harmless acceptance).
- **L9 dependency:** a skill namespace binds to an immutable registered skill identity, never a mutable display name or task-supplied name; skill identity is established before resolution and cannot be chosen dynamically by task content.
- **Trusted-root duplicates abort** — any winner rule (first/last/file-order/hash) is an accidental authority mechanism; internal ambiguity of trusted configuration means no EIS. Insider-DoS exposure accepted (same trade as Q-L1-6); mitigation lives outside L1: protected-branch governance, review, pre-merge duplicate-ID CI checks.

### Q-L1-8 — Role text (CLOSED, hardened)

Owner-accepted invariant (with refinement):

> Role text is an instruction: a protected harness-system instruction (`harness.system.role`), stored, reviewed, hashed, and versioned like all instruction content — never hard-coded in the renderer, because delivered text that shapes behavior while escaping instruction governance is a shadow instruction channel. Its content is constrained by DEC-06: neutral work framing, never a security-authority identity. A role confers no standing — no claimed or delivered persona grants approval, verification, or any governed capability, because nothing in instruction space confers capability. Lower scopes cannot declare or redefine the protected `harness.system.role` instruction identity or its authority-bearing semantics; perspective framing at lower scopes ("analyze this from the perspective of a security reviewer") is legitimate behavioral framing within the framer's own namespace, subject to conflict detection. The renderer owns structural furniture only; any delivered text carrying a rule lives in an instruction file.

Key distinction (owner): role text is instruction because it changes model behavior; role identity is not authority because instruction cannot confer authority. The boundary is protected role *semantics*, not the appearance of role-related words.

**Permanent review rule:** if delivered renderer text contains behavioral directives, the directive belongs in the instruction system, not renderer code. Renderer: headings, delimiters, structural formatting, section labels. Instruction files: safety rules, role framing, behavioral constraints, task directives.

### Q-L1-7-original — Curated version labels (CLOSED by the identity closure)

Content hashes are the versioning mechanism: the id names the slot, the body hash names the revision, and hash diffs answer "what changed" mechanically. Curated human-readable labels are optional governance additions — not required for v1, addable later without architectural change.

### Grill summary — the boundary distinctions (all owner-confirmed)

Discovery ≠ instruction recognition · provenance ≠ integrity ≠ authorization ≠ reproducibility · identity ≠ semantics · namespace ownership ≠ scope precedence · pattern detection ≠ security enforcement · exemption ≠ authorization · trusted source ≠ universally safe composition · instruction ≠ capability · role ≠ authority · model output ≠ security truth · EIS mutation ≠ instruction succession · conversation history ≠ new instruction source · renderer furniture ≠ behavioral instruction · no EIS ≠ degraded EIS · resolution boundary ≠ optional model decision.

## 7. Grill questions (ALL CLOSED 2026-09-05)

1. **Q-L1-1 — Repository-scope trust: CLOSED 2026-09-05** — see §6 "Q-L1-1 (CLOSED, hardened)". Enum yes, source no in v1; five-part hardened trust statement accepted; activation deferred to the L5 era.
2. **Q-L1-2 — Directive-pattern governance: CLOSED 2026-09-05** — see §6 "Q-L1-2 (CLOSED, hardened)". Versioned policy artifact under `policies/security/`, Class-3 changes, dual-corpus ratchet + security review, operator exemptions that suppress detection but never authorize.
3. **Q-L1-3 — Task-instruction schema: CLOSED 2026-09-05** — see §6 "Q-L1-3 (CLOSED: one blob, hardened)". One task-scoped blob; L1 never composes/rewrites; final rendered-EIS validation pass (reject/record only); source trust ≠ composition safety.
4. **Q-L1-4 — Templating: CLOSED 2026-09-05** — see §6. Generalized: no runtime data may alter resolved instruction text.
5. **Q-L1-5 — Mid-task re-resolution: CLOSED 2026-09-05** — see §6. Immutable EIS per resolution epoch; recorded succession at L7-owned boundaries only; explicit epoch context boundaries.
6. **Q-L1-6 — Resolution failure semantics (subsumed size caps): CLOSED 2026-09-05** — see §6. Atomic resolution, reject-vs-abort dividing line, explicit resolution statuses, no runtime fallback; caps 8 KiB/64 KiB confirmed.
7. **Q-L1-7 — CLOSED 2026-09-05** — split into two closures in §6: instruction identity (grill-added question: namespace ownership, registry, no content-based identity) and curated labels (hashes suffice; labels optional later).
8. **Q-L1-8 — Role text home: CLOSED 2026-09-05** — see §6. Protected instruction file `harness.system.role`; renderer owns furniture only; role ≠ authority.
