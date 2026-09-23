# Delegation templates (Layer 8)

Governance's registration record for delegation templates
(`openspec/changes/archive/2026-09-23-layer-08-subagents/design.md`, D-L8-5/6). A delegation
template is a distinct governed artifact family: not a Skill, not a
Skill member, never executable. It defines only the bounded composition
available to one isolated, tool-less, single-call model execution that
a parent task may request through the `delegate` capability.

## Layout

- `registry.json` — append-only: `name`, `version`, `template_sha256`,
  `manifest_path`, `state` (`active` | `withdrawn`), `steward`. A
  `name@version` binds to one hash forever; any change is a new
  version; withdrawal is forward-only.
- `<name>/template.json` — the manifest: `context_contract {path,
  sha256}`, optional `instruction {path, sha256}`, `eis_carry_scopes[]`
  ⊆ `{repository, directory, skill}`, `brief {slot, max_bytes}`,
  `max_output_bytes`. The hash of these bytes is the template's
  identity and covers every pinned byte.
- `<name>/contract.json` — an ordinary Layer 2 context contract. Its
  slots are kind-routed: the brief slot (kind `delegation-brief`), and
  evidence slots by the referenced item's event-derived kind —
  `tool:*` (any recorded tool result), `model-turn` (a parent model
  turn), `delegation-output` (a prior delegation's output). A
  reference that fits zero or more than one non-withheld slot refuses.
- `<name>/instruction.md` — at most one skill-scope instruction file.

The harness only reads this directory (`src/harness/subagents/
delegation`, no write API; an AST wall keeps it so). Registration is an
owner act: a commit that adds an entry here. `scripts/themis-preflight`
verifies every registered template's bytes against its pin.

## What a template may not contain

A workflow, a workflow ceiling, a grant, a spec, an input schema,
tools, a model, a scope, a procedure, or another template. The loader
refuses these by name. Requiredness and slot classes live in the pinned
contract, never in `template.json`.

## Registration-review checklist (C-L8-14)

The loader establishes admissibility, identity, and accountability. It
cannot read prose. Before registering, a reviewer confirms:

1. **No directive to disregard a higher scope.** The instruction file
   specializes; it never says to ignore, override, or reinterpret
   harness-safety, harness-system, themis-domain, or repository rules.
2. **No factual claims about the world in the instruction file.** An
   instruction is a rule of work, never a fact (a claim such as "CVE-X
   is not exploitable" is a content defect; facts arrive as evidence).
3. **Permitted classes and sensitivity ceiling justified** against the
   template's purpose: the evidence slots admit only the classes the
   reasoning needs; the ceiling is the lowest that serves it.
4. **Brief slot bound and `max_output_bytes` justified**: the brief
   slot permits exactly `external-untrusted`; both bounds are as small
   as the purpose allows and never exceed Layer 2's per-item cap.
5. **Carry filter minimal**: `eis_carry_scopes` names only the scopes
   the reasoning needs; the mandatory roots carry regardless.
6. **Steward named** and accountable for the content.

## Registered templates

| Template | Purpose | Steward | State |
| --- | --- | --- | --- |
| `dependency-triage@1` | one triage pass over recorded evidence for `remediate-dependency@2` (Governance-ratified 2026-09-23) | security-engineering | active |
