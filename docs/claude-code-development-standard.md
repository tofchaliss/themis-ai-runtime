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
| 1 | Architecture constitution | **Accepted by owner 2026-09-04** — constitution text below |
| 2 | Claude Code behaviour | Not started |
| 3 | Security engineering rules | Not started |
| 4 | Operations mechanisms (rule → CLAUDE.md / SKILL.md / AGENTS.md / agents / hooks) | Not started |
| 5 | Commit the final skills | Blocked on 1–4 |

---

## Stage 1 — Architecture constitution — ACCEPTED 2026-09-04

### The constitution (owner-accepted text, verbatim)

> Themis is the security application and system of record. The AI Harness is an execution capability of Themis—architecturally owned and governed by it, whether deployed in-process or as a separate Themis-controlled process/node. The Harness may reason over Themis data and invoke authorized Themis capabilities, but it does not own security truth. DeepSeek is a replaceable reasoning component behind the Harness model interface. Deterministic mechanisms enforce authorization, policy, state, verification, and security invariants; probabilistic reasoning is used for interpretation and analysis. All AI output is advisory; acceptance into security truth requires the appropriate human or governed Themis decision.

Constitutional points and where each is detailed:

| # | Point | Detail |
|---|---|---|
| 1 | Themis ownership and Harness subordination | §1.1 |
| 2 | Deployment topology remains open (in-process or Themis-controlled node — an authority statement, not a deployment constraint) | §1.1 |
| 3 | 11-layer Harness responsibility boundaries | §1.2 |
| 4 | Security evidence/authority hierarchy | §1.2a |
| 5 | Deterministic vs probabilistic boundary | §1.4 |
| 6 | Instruction / Context / Tool / Authority separation | §1.6 |
| 7 | Human/governed decision authority over AI output | §1.3 |

The sections below carry the accepted detail; reopening any of them requires a recorded reason (process rule).

> **Vocabulary note:** Contract-3's "Layer 0/1/2" are **data-authority tiers** (where security truth comes from), *not* the harness's 11 runtime layers. To keep the two vocabularies apart, this document writes them as **Tier 0/1/2**.

### 1.1 Themis vs Harness ownership (Contract-1) — *accepted 2026-09-04*

- Themis owns security & business truth: Findings, Enterprise Positions, security decisions, workflows, enterprise knowledge, SBOM/scan/release/product data.
- The Harness is an internal AI runtime capability of Themis: it executes authorized work and produces evidence; it owns task state, traces, and skill procedures — never enterprise truth.
- The Harness reads/writes authoritative state only through Themis-owned interfaces; no direct database access.
- This repo (monorepo) hosts the harness at `src/harness/`; Themis joins under `src/` later.

### 1.2 11-layer responsibilities (Contract-2) — *accepted 2026-09-04*

- The layer table in the project report §4 is the responsibility contract: L1 Instructions, L2 Context Delivery, L3 Context Management, L4 Tool Interface, L5 Execution Environment, L6 Durable State, L7 Orchestration, L8 Subagents, L9 Skills, L10 Verification & Observability, L11 Ratchet.
- **A layer owns a capability; it does not merely contain code related to that capability.** If a capability's decisions are made outside its layer, the boundary is broken regardless of where the files sit.
- Every change identifies exactly one owning layer before implementation; cross-layer changes name each affected layer explicitly.
- Layer boundaries that must never blur: Instructions ≠ Context (rules vs facts), Instructions ≠ Permissions (guidance vs enforcement), Verification ≠ model claims.

### 1.2a Security-data authority hierarchy (Contract-3) — *accepted 2026-09-04*

Where security truth comes from, in strict order — AI reasoning sits *below* every deterministic tier and above only presentation:

```
                 SECURITY AUTHORITY
                       ▲
                       │
              ┌────────┴────────┐
              │                 │
           Tier 0            Tier 1
       Immutable facts   Deterministic feeds
        (e.g. SBOM)      (security data)
              │                 │
              └────────┬────────┘
                       │
                       ▼
                    Tier 2
              Derived knowledge
                       │
                       ▼
                 AI reasoning
                       │
                       ▼
                 Presentation
```

- Tier 0 (immutable facts, e.g. SBOM contents) and Tier 1 (deterministic security feeds) are authoritative; Tier 2 (derived knowledge) is built from them deterministically.
- AI reasoning consumes the tiers and produces advisory output; it can never write upward into any tier.
- Presentation (Communication) displays; it establishes nothing.

### 1.3 Authority model — *accepted 2026-09-04*

- Human authority is final. Every AI output is advisory (`requires_human_decision` true); AI proposals never auto-accept.
- Autonomy of generation is allowed; autonomy of authority is never.
- The approval gate before any commit by the harness's agent is not removable by configuration.
- The Ratchet may only make permanent changes through human/policy approval.

### 1.4 Deterministic vs probabilistic boundary (Contract-4) — *accepted 2026-09-04*

**Deterministic — never model-based:**
authorization · schema validation · state transitions · version comparison · hash calculation · policy enforcement · tool permissions · approval requirements · evidence provenance · audit records · verification. *(Plus, from prior drafts: instruction resolution, budgets, injection flagging.)*

**Probabilistic — model territory:**
interpretation · reasoning · hypothesis generation · classification assistance · summarization · contextual analysis · natural-language explanation.

The flow between them is one-directional and always mediated:

```
             DeepSeek (or any provider)
                │
                │ probabilistic
                ▼
          Agent reasoning
                │
                ▼
       ┌─────────────────┐
       │ Deterministic   │
       │ validation      │
       │ policy          │
       │ verification    │
       └────────┬────────┘
                │
                ▼
             Themis
```

- Nothing probabilistic reaches Themis except through the deterministic block.
- The model may *propose* crossing any boundary; only deterministic machinery may *effect* it.
- Scoring/evaluation of models is deterministic (themis-bench philosophy: no LLM-as-judge in the gate path).

### 1.5 DeepSeek / model abstraction (Contract: DeepSeek is behind the harness) — *accepted 2026-09-04*

The containment chain — each level reaches the model only through the next:

```
Themis
  │
  ▼
Harness
  │
  ▼
Agent Runtime
  │
  ▼
Model Interface
  │
  └── DeepSeek (or any provider)
```

- DEC-05: implementation is model-agnostic; DeepSeek is a deployment/config choice behind the adapter + registry; tests run on mocks. Nothing above the Model Interface may know which provider is configured.
- DEC-03: local-first serving; hosted API only via recorded per-task egress decision with brokered credentials.
- DEC-06: the model is never told it is the security system; role framing is neutral.
- One stable chat+tool-call interface serves all providers (Ollama and OpenAI-compatible).

### 1.6 Instruction / context / tool / authority separation (Contract-5) — *accepted 2026-09-04*

Four distinct concepts, defined by what they do:

| Concept | Role |
|---|---|
| **Instruction** | Tells the agent what/how to behave |
| **Context** | Provides information to reason about |
| **Tool** | Provides capability to act or retrieve information |
| **Authority** | Determines what the system accepts as truth/state |

Therefore, as inviolable inequalities:

- **Instruction ≠ Context** — rules are not facts
- **Context ≠ Authority** — knowing something does not make it true for the enterprise
- **Tool ≠ Authority** — being able to act does not decide what is accepted
- **Model output ≠ Authority** — ever

Supporting rules:

- Only explicitly recognized instruction sources participate in instruction resolution; repository content, CVE text, scanner output, tool results, and web content are untrusted data, never instructions.
- Instruction precedence is deterministic and resolved outside the model; protected (safety/domain) instructions cannot be overridden by lower scopes.
- Secrets never enter model context; credentials flow only tool → broker → short-lived scoped credential.

**Stage 1 exit criteria — MET 2026-09-04:** all of 1.1–1.6 (incl. 1.2a) accepted by the owner. The constitution text above becomes the constitution section of the final SKILL.md, and the seven constitutional points become the invariants list for the review agents (Stage 4 decides the exact placement).

---

## Stage 2 — Claude Code behaviour *(not started)*

To define: how Claude Code investigates before changing code; the Architecture Decision Gate (what triggers it, who passes it); change classification (e.g. trivial / standard / architectural / security-sensitive, with different rigor per class); coding workflow; testing workflow; verification workflow; git/checkpoint behavior (incl. the Day-0 check-in rules already in CLAUDE.md); and the explicit list of situations where Claude Code must stop and ask.

## Stage 3 — Security engineering rules *(not started)*

To define: prompt-injection handling; tool authorization; secrets; sandboxing; external systems; destructive operations; supply-chain and security-data handling; auditability.

## Stage 4 — Operations mechanisms *(not started)*

For every rule from Stages 1–3, decide its enforcement home: CLAUDE.md (always-loaded context) vs SKILL.md (loaded for engineering work) vs AGENTS.md (repo instructions for any agent) vs review agents (post-change review) vs hooks (deterministic, cannot be talked out of). Principle to test each rule against: guidance goes in instructions; anything that must never depend on model compliance goes in hooks.

## Stage 5 — Commit the final skills *(blocked on 1–4)*

Replace the draft pack in `.claude/` with the finalized standard: instructions skill, review agents, enforcement hooks (settings.json), plus AGENTS.md. From then on the standard is versioned like the project report — amendments only with recorded reasons.
