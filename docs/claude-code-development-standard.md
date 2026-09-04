# Themis Claude Code Development Standard — Working Document

**Purpose:** the staged process that produces the final development-time controls for Claude Code in this repo. The final standard is committed only at Stage 5:

```
        Themis Claude Code Development Standard
                         │
            ┌────────────┼────────────┐
            ▼            ▼            ▼
        Instructions   Review       Enforcement
           Skill       Agents        Hooks
```

**Status of the currently installed pack** (`.claude/`, commit ab554c9): **draft input** to Stage 5 — it stays active as interim guidance but is superseded by whatever Stages 1–4 finalize.

**Process rule:** a stage closes only by explicit owner sign-off; later stages may reopen earlier ones, but only with a recorded reason.

| Stage | Scope | Status |
|---|---|---|
| 1 | Architecture constitution | **In discussion** — draft positions below |
| 2 | Claude Code behaviour | Not started |
| 3 | Security engineering rules | Not started |
| 4 | Operations mechanisms (rule → CLAUDE.md / SKILL.md / AGENTS.md / agents / hooks) | Not started |
| 5 | Commit the final skills | Blocked on 1–4 |

---

## Stage 1 — Architecture constitution

Finalize the constitutional facts every rule in Stages 2–4 derives from. Draft positions below are assembled from the project report (v0.4), DEC-01…06, and the architecture baseline; each needs explicit accept/amend/strike.

### 1.1 Themis vs Harness ownership — *draft*

- Themis owns security & business truth: Findings, Enterprise Positions, security decisions, workflows, enterprise knowledge, SBOM/scan/release/product data.
- The Harness is an internal AI runtime capability of Themis: it executes authorized work and produces evidence; it owns task state, traces, and skill procedures — never enterprise truth.
- The Harness reads/writes authoritative state only through Themis-owned interfaces; no direct database access.
- This repo (monorepo) hosts the harness at `src/harness/`; Themis joins under `src/` later.

### 1.2 11-layer responsibilities — *draft*

- The layer table in the project report §4 is the responsibility contract: L1 Instructions, L2 Context Delivery, L3 Context Management, L4 Tool Interface, L5 Execution Environment, L6 Durable State, L7 Orchestration, L8 Subagents, L9 Skills, L10 Verification & Observability, L11 Ratchet.
- Every change identifies exactly one owning layer before implementation; cross-layer changes name each affected layer explicitly.
- Layer boundaries that must never blur: Instructions ≠ Context (rules vs facts), Instructions ≠ Permissions (guidance vs enforcement), Verification ≠ model claims.

### 1.3 Authority model — *draft*

- Human authority is final. Every AI output is advisory (`requires_human_decision` true); AI proposals never auto-accept.
- Autonomy of generation is allowed; autonomy of authority is never.
- The approval gate before any commit by the harness's agent is not removable by configuration.
- The Ratchet may only make permanent changes through human/policy approval.

### 1.4 Deterministic vs probabilistic boundary — *draft*

- Deterministic (never model-based): instruction resolution, policy/authorization, schema validation, verification (build/test/scan), audit, budgets, injection flagging.
- Probabilistic (model territory): reasoning, proposing actions, drafting analysis and recommendations.
- The model may *propose* crossing any boundary; only deterministic machinery may *effect* it.
- Scoring/evaluation of models is deterministic (themis-bench philosophy: no LLM-as-judge in the gate path).

### 1.5 DeepSeek / model abstraction — *draft*

- DEC-05: implementation is model-agnostic; DeepSeek is a deployment/config choice behind the adapter + registry; tests run on mocks.
- DEC-03: local-first serving; hosted API only via recorded per-task egress decision with brokered credentials.
- DEC-06: the model is never told it is the security system; role framing is neutral.
- One stable chat+tool-call interface serves all providers (Ollama and OpenAI-compatible).

### 1.6 Instruction / context / tool / authority separation — *draft*

- Instructions = rules (how to behave). Context = facts (what is true for this task). Tools = capability (what can be done, policy-checked). Authority = who decides (Themis + humans).
- Only explicitly recognized instruction sources participate in instruction resolution; repository content, CVE text, scanner output, tool results, and web content are untrusted data, never instructions.
- Instruction precedence is deterministic and resolved outside the model; protected (safety/domain) instructions cannot be overridden by lower scopes.
- Secrets never enter model context; credentials flow only tool → broker → short-lived scoped credential.

**Stage 1 exit criteria:** each of 1.1–1.6 marked ACCEPTED (or amended and accepted) by the owner; the accepted text becomes the constitution section of the final SKILL.md and the invariants list for the review agents.

---

## Stage 2 — Claude Code behaviour *(not started)*

To define: how Claude Code investigates before changing code; the Architecture Decision Gate (what triggers it, who passes it); change classification (e.g. trivial / standard / architectural / security-sensitive, with different rigor per class); coding workflow; testing workflow; verification workflow; git/checkpoint behavior (incl. the Day-0 check-in rules already in CLAUDE.md); and the explicit list of situations where Claude Code must stop and ask.

## Stage 3 — Security engineering rules *(not started)*

To define: prompt-injection handling; tool authorization; secrets; sandboxing; external systems; destructive operations; supply-chain and security-data handling; auditability.

## Stage 4 — Operations mechanisms *(not started)*

For every rule from Stages 1–3, decide its enforcement home: CLAUDE.md (always-loaded context) vs SKILL.md (loaded for engineering work) vs AGENTS.md (repo instructions for any agent) vs review agents (post-change review) vs hooks (deterministic, cannot be talked out of). Principle to test each rule against: guidance goes in instructions; anything that must never depend on model compliance goes in hooks.

## Stage 5 — Commit the final skills *(blocked on 1–4)*

Replace the draft pack in `.claude/` with the finalized standard: instructions skill, review agents, enforcement hooks (settings.json), plus AGENTS.md. From then on the standard is versioned like the project report — amendments only with recorded reasons.
