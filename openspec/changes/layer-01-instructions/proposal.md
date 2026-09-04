# Proposal: Layer 1 — Instructions

**Change ID:** layer-01-instructions · **Status:** DRAFT for grilling · 2026-09-04
**Owner gate:** this change ships only after the grill session closes the design's open questions (Q-L1-1…8 in design.md).

## Why

The Harness has no instruction system: behavioral rules live as fixed prompt templates inside two service operations and a benchmark preamble. Every later layer (context, tools, orchestration, skills) assumes a deterministic, versioned, auditable Effective Instruction Set exists per task. Layer 1 is the first layer of the deterministic foundation in the owner-locked sequence (architecture-to-code map §4), now unblocked by the proven Model Interface.

## What

A deterministic instruction system at `src/harness/instructions`:

- Instruction representation (Markdown + YAML frontmatter, one instruction per file)
- Recognized sources only: harness rules, Themis rules, repository rules (operator-gated), skill and task instructions
- Pipeline per the owner's reference flow: Sources → Loader → Representation → Validator → Conflict Detector → Resolver → **EffectiveInstructionSet** {version, hash, sources} → Context Delivery
- Deterministic validation (ids, scopes, categories, secret scan, size caps) and mechanical conflict detection (ID shadowing, protected-directive patterns); semantic conflict detection stays deferred per the accepted Stage-1 limitation
- Content-hash versioning (per-instruction body hash + canonical set hash), the proven manifest pattern
- The shipped initial instruction content (`instructions/global/`, `instructions/themis/`), authored from the accepted constitution and existing prompts
- Delivery contract: L1 produces the EIS; **Context Delivery (L2) composes it into the model payload** — L1 never calls the model

## What this change does NOT do

- No semantic/NL contradiction detection (deferred, recorded)
- No directory-scoped sources inside target repos (needs L5 worktrees)
- No central registry/UI (layer doc "Later")
- No change to the existing service prompts or benchmark preamble (they belong to the proto-skills until L9 absorbs them)
- No mid-task re-resolution (frozen EIS per resolution; see Q-L1-5)

## Success criteria

Every acceptance criterion of `docs/architecture/harness/01-instructions.md` §17 that is in scope maps to a named test; resolution is byte-deterministic (same sources ⇒ same EIS hash); a protected instruction provably cannot be overridden by any lower scope; the EIS is reconstructable from its recorded hashes and sources.
