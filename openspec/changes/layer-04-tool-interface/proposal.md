# Proposal: Layer 4 — Tool Interface

**Change ID:** layer-04-tool-interface · **Status:** DRAFT for grilling · 2026-09-06
**Owner gate:** ships only after the grill closes Q-L4-1…9 (design.md) and the owner accepts.

## Why

L1–L3 shipped the advisory side: what the model is told, how facts reach it, what fits. Nothing yet enforces the other half of every locked invariant — *"model output is inert data"* is currently true only because no execution path exists at all. L4 is the first major security boundary (architecture-to-code map, Phase C): the layer where a tool request either becomes an authorized, validated, audited operation or dies as recorded data. Every proven rule finally gets its enforcement counterpart: L4-independence (authorization computed from capability/policy state, never from model desire or task instructions), the closed vocabulary (`update_enterprise_position` does not exist; schemas carry subject, never authority disposition), and the three verbs (L7 considers · **L4 authorizes** · execution fulfills).

## What

A deterministic tool-interface layer at `src/harness/tools`:

- **Capability registry** as a governed artifact: name, input/output schemas, permission class, target rules, timeout, audit behavior, result trust class — versioned, hashed into the trace
- **Per-task grants** strictly ⊆ registry (the contract-ceiling pattern applied to capabilities)
- **Pipeline per the layer doc:** tool request → strict schema validation (unknown field ⇒ whole-call rejection, recorded) → policy/grant authorization → target validation → registered executor → result → audit event
- **Results are evidence:** executor output re-enters the conversation as classified data (trust class fixed at tool registration — the L2 classification discipline), never as instructions or authority
- **Typed denials:** every refusal recorded and model-visible as data; registry/policy load failure ⇒ no tool execution at all (fail closed)
- **v1 executors: read-only** — confined filesystem reads (`read_file`, `list_directory`, `search_code` reusing L2 confinement), Themis read seam stubs (`get_finding`, `get_product`, …). Everything mutating or shell-executing is deferred.

## What this change does NOT do

- No `run_command` (needs OPEN-2 policy grill + L5 sandbox)
- No mutating tools (`write_file`, `apply_patch`, `git_commit`, `create_investigation_result`, `attach_evidence`) — deferred to L5 worktrees + the approval-gate design
- No execution sandbox (L5); no request loop or retry orchestration (L7); no durable audit sink (L6)
- No authority-bearing verbs, ever: nothing in the vocabulary mutates Themis security state

## Success criteria

Authorization is provably independent of request content beyond (tool, args, target); an ungranted or unregistered tool is denied identically regardless of how the model asks; unknown schema fields reject the whole call; no caller-suppliable field alters governance treatment; every execution and every denial is audit-recorded and reconstructable; the L1 scenario-3 attack (`propose_position_change` with `requires_human_decision=false`) dies at schema validation exactly as the grill specified; live proof per Q-L4-9 (requires a tools-capable local model — owner decision outstanding).
