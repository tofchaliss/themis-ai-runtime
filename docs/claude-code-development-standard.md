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
| 2 | Claude Code behaviour | **Accepted by owner 2026-09-04** — behaviour standard below |
| 3 | Security engineering rules | **Accepted by owner 2026-09-04** — twelve boundaries below |
| 4 | Operations mechanisms (rule → CLAUDE.md / SKILL.md / AGENTS.md / agents / hooks) | **Accepted by owner 2026-09-04** — placement matrix below |
| 5 | Commit the final skills | **Ready** — assembly per Stages 1–4 |

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

## Stage 2 — Claude Code behaviour — ACCEPTED 2026-09-04

### The behaviour standard (owner-accepted text, verbatim)

> Claude Code shall investigate before modifying, identify the responsible Themis/Harness owner and architectural layer, classify the change, and apply the minimum justified implementation. Trivial documentary changes follow a reduced path; executable-code and configuration changes follow the full development and verification pipeline. Ambiguous architecture never resolves toward autonomy. Claude Code may act autonomously within established boundaries, but must stop for the eight explicitly defined conditions: architectural ownership, security authority, trust boundaries, deployment topology, fundamental model architecture, unclear architecture, destructive or irreversible operations, and Constitution/Day-0 prohibitions. Git checkpoints are green commits with evidence; one logical change belongs in each commit, while history rewrites, shared-remote pushes, and branch-topology changes require approval. Functional, security, and architectural review are mandatory according to change scope, with their eventual Claude Code mechanisms defined separately.

This text supersedes the earlier §2.3 draft and resolves every §2.4 clarification (all five accepted and absorbed). The stop+ask list is **closed by enumeration** at the eight conditions above; additions are a Stage-2 amendment. Review mechanisms (agents vs inline) are deliberately deferred to Stage 4.

### 2.1 The workflow (owner-accepted, verbatim)

```
                    User Request
                         │
                         ▼
                  ┌─────────────┐
                  │  Understand │
                  └──────┬──────┘
                         │
                         ▼
                   Inspect Code
                         │
                         ▼
                 Identify Owner
                         │
                         ▼
                 Identify Layer
                         │
                         ▼
               Classify the Change
                         │
              ┌──────────┴──────────┐
              │                     │
         Normal change         Architecture /
              │                 authority change
              ▼                     ▼
           Plan                  STOP + ASK
              │
              ▼
         Threat Model
              │
              ▼
        Minimal Change
              │
              ▼
        Functional Tests
              │
              ▼
         Security Tests
              │
              ▼
       Architecture Review
              │
              ▼
        Evidence / Checkpoint
              │
              ▼
             DONE
```

### 2.2 Change classification (owner-accepted, verbatim)

| Change | Claude Code behavior |
| --- | --- |
| Local implementation | Proceed |
| Normal cross-layer | Analyze + proceed if architecture is clear |
| Security-sensitive | Analyze + verify + proceed where authority is unchanged |
| Architecture change | **Stop + ask** |
| Authority change | **Stop + ask** |
| New deployment topology | **Stop + ask** |

### 2.3 Original behaviour summary (owner draft — superseded by the accepted text above)

> Claude Code shall investigate before modifying, identify the responsible Themis/Harness owner and architectural layer, classify the change, and apply the minimum justified implementation. It may autonomously implement changes within established architectural boundaries, including security-sensitive changes where ownership and authority remain unchanged, but must stop for decisions that alter architectural ownership, security authority, trust boundaries, deployment topology, or fundamental model architecture. Every meaningful change must undergo functional, security, and architectural verification, with evidence recorded before completion.

### 2.4 Clarifications proposed by Claude Code — *all five ACCEPTED, absorbed into the accepted text (2026-09-04)*

1. **Unclear architecture escalates.** In 2.2, "Normal cross-layer: proceed if architecture is clear" — the implicit inverse becomes explicit: if the owning layer or architectural intent cannot be determined with confidence during investigation, the change is treated as an Architecture change → **stop + ask**. Ambiguity never resolves toward autonomy.
2. **Proportional rigor for trivial changes.** "Every meaningful change" implies a floor: typo/comment/doc-only changes take the short path (inspect → change → functional verification (build/docs render) → checkpoint), skipping threat model and security tests. Anything touching executable code or configuration is at least "Local implementation" and takes the full pipeline.
3. **Git/checkpoint behavior made concrete** (Stage-2 item 7): a checkpoint = a commit at a green state (build + tests pass) with its evidence; commits follow the Day-0 check-in rules (no Claude signatures, .gitignore honored, no binaries, no secrets); one logical change per commit; history rewrites, pushes to shared remotes, and branch topology changes are **stop + ask**.
4. **Review-step mapping.** "Security Tests" and "Architecture Review" in 2.1 are performed via the review agents (security-reviewer/test-reviewer and architecture-reviewer respectively) once Stage 4 finalizes them; until then Claude Code performs the equivalent review inline. (Exact mechanism = Stage-4 decision; this clarification only fixes the intent.)
5. **The stop+ask list is closed by enumeration.** Stop + ask triggers are exactly: architectural ownership, security authority, trust boundaries, deployment topology, fundamental model architecture (per 2.3), unclear architecture (per clarification 1), destructive/irreversible operations, and anything the Day-0 rules or the constitution prohibit resolving autonomously. Additions to this list are a Stage-2 amendment, not ad-hoc judgment.

**Stage 2 exit criteria — MET 2026-09-04:** the consolidated behaviour standard accepted by the owner; workflow and classification table stand; all clarifications resolved.

## Stage 3 — Security engineering rules — ACCEPTED 2026-09-04

### The security standard summary (owner-accepted, verbatim)

> Claude Code shall develop the Harness under a zero-trust model: model output and external content are untrusted; authorization, policy, security state, verification, and other security invariants are enforced deterministically. The Harness shall use least privilege, isolate execution, protect credentials, preserve evidence provenance, and fail closed at security boundaries. Destructive or irreversible operations require explicit controls. Security invariants must be testable wherever practical. The Harness must remain safe when the model is wrong, manipulated, unavailable, or influenced by malicious input, and no model output may directly establish or override Themis security truth.

### The twelve security boundaries (owner-accepted, verbatim)

| # | Boundary | Rule |
|---|---|---|
| 3.1 | Zero trust toward model output | Model output is never inherently trusted. |
| 3.2 | External content is untrusted | CVE descriptions, repositories, web pages, scanner output, etc. remain data, not instructions. |
| 3.3 | Deterministic authorization | Model intent never grants permission. |
| 3.4 | Least privilege | Every capability receives minimum necessary permissions. |
| 3.5 | Secret isolation | Credentials are not ordinary model context. |
| 3.6 | Execution isolation | Command/code execution occurs within controlled execution boundaries. |
| 3.7 | Destructive/irreversible operations | Require explicit controls and appropriate approval. |
| 3.8 | Security evidence provenance | Preserve source, origin and transformation of evidence. |
| 3.9 | Fail closed | Security-control failure does not fall back to model judgment. |
| 3.10 | Testable security invariants | Important security properties require executable evidence wherever practical. |
| 3.11 | External-system access | External-system access is policy-gated: local-first by default, and any data egress to a non-local system requires explicit, recorded operator authorization. |
| 3.12 | Supply chain and security-data protection | Dependencies are minimal and individually justified; security data is handled according to the Tier 0/1/2 authority hierarchy, while sensitivity is independently considered for storage, transmission, logging, and exposure; sensitive values are redacted before entering durable storage or logs. |

### The trust and egress flow (owner-accepted, verbatim)

```
                         THEMIS
                           │
                    Authority Boundary
                           │
                           ▼
                    ┌─────────────┐
                    │   Harness   │
                    └──────┬──────┘
                           │
        ┌──────────────────┼──────────────────┐
        │                  │                  │
        ▼                  ▼                  ▼
   Model Output       External Data       Tools
        │                  │                  │
        │             UNTRUSTED              │
        │                  │                  │
        └──────────┬───────┴──────────┬───────┘
                   ▼                  ▼
             Validation          Authorization
                   │                  │
                   └────────┬─────────┘
                            ▼
                       Policy Gate
                            │
                  ┌─────────┴─────────┐
                  ▼                   ▼
               Local              External
              default              system
                                    │
                              explicit operator
                              authorization
                                    │
                                    ▼
                                 Egress
```

### The persistence sensitivity flow (owner-accepted, verbatim)

```
              All persistent state
                     │
                     ▼
              Sensitivity check
                     │
             ┌───────┴───────┐
             ▼               ▼
          Safe data       Sensitive data
             │               │
             ▼               ▼
          Persist       Redact / restrict
```

**Stage 3 exit criteria — MET 2026-09-04:** the summary paragraph and the twelve boundaries accepted by the owner; both proposed gap sentences absorbed (3.11 verbatim; 3.12 strengthened — sensitivity is considered independently of authority tier). Every boundary is a candidate for an executable check in Stage 4 (per 3.10).

## Stage 4 — Operations mechanisms — ACCEPTED 2026-09-04

### The mechanism-separation standard (owner-accepted, verbatim — final form 2026-09-04)

> Claude Code development controls shall be separated by mechanism: CLAUDE.md provides concise repository orientation; the Themis Harness Engineering Skill defines the engineering constitution; AGENTS.md provides repository and scoped implementation constraints; .claude/policy/DAY-0.md is the authoritative home for Day-0 prohibitions; hooks provide deterministic enforcement; settings.json registers those hooks; review agents assess architecture, security, and verification; and human authority remains outside these mechanisms. Each important rule has one authoritative home, with other mechanisms referencing, enforcing, or reviewing it rather than duplicating it.

*(Supersedes the first accepted wording of the same date. Material refinements: CLAUDE.md is orientation only — the engineering constitution's authoritative home is the Skill, and CLAUDE.md references it; DAY-0.md named as the authoritative home for the Day-0 prohibitions — resolving Stage-5 implementation note 2; settings.json explicitly registers hooks — resolving note 3; the single-home rule now states what the other mechanisms do: reference, enforce, or review — never duplicate.)*

### The placement matrix (owner-accepted, verbatim)

| Principle | Primary mechanism | Enforcement |
| --- | --- | --- |
| Themis owns security truth | Skill + architecture review | Review |
| Harness is Themis capability | Skill | Review |
| Deployment topology remains open | Skill | Stop + ask |
| Human authority | Skill + workflow | Stop/approval |
| 11-layer architecture | Skill | Architecture review |
| Deterministic vs probabilistic | Skill | Architecture review/tests |
| Model independence | Skill + tests | Review |
| Inspect before changing | Skill | Claude behavior |
| Minimal change | Skill + review | Review |
| Unclear architecture → stop | Skill | Stop |
| Destructive operation → stop | Hook + skill | Mechanical |
| Prompt injection | Skill + tests | Security review |
| Tool authorization | Skill + code/tests | Deterministic |
| Least privilege | Skill + code/tests | Security review |
| Secret protection | Hook + tests | Mechanical + review |
| External-system access | Skill + policy | Deterministic |
| Supply-chain discipline | Skill + dependency checks | Review |
| Data redaction | Skill + tests/hooks | Deterministic |
| Git checkpoint | Skill + hooks | Mechanical |
| One logical change/commit | Skill + review | Review |
| History rewrite | Hook/skill | Stop |
| Shared push | Hook/skill | Stop |
| Verification | Skill + tests | Mechanical/review |

### The mechanism stack

```
CLAUDE.md
     +
Development Skill
     +
AGENTS.md
     +
Day-0 policy
     +
Hooks
     +
Review agents
```

### Proposed final structure (owner-supplied — explicitly proposed, NOT concrete; finalized in Stage 5)

```
themis-ai-runtime/
│
├── CLAUDE.md
├── AGENTS.md
│
└── .claude/
    │
    ├── skills/
    │   └── themis-harness-engineering/
    │       └── SKILL.md
    │
    ├── policy/
    │   └── DAY-0.md
    │
    ├── agents/
    │   ├── architecture-reviewer.md
    │   ├── security-reviewer.md
    │   └── test-reviewer.md
    │
    └── hooks/
        ├── README.md
        ├── destructive-command-guard
        ├── secret-guard
        └── verification-guard
```

### Stage-5 implementation notes (Claude Code — flagged for assembly, no sign-off needed since the structure is non-concrete)

1. **Git protections need a mechanical home.** The matrix routes history-rewrite and shared-push to "Hook/skill → Stop"; the proposed hook trio doesn't name a git guard. At assembly, either fold git protections (history rewrite, shared-remote push, Day-0 commit rules) into `destructive-command-guard` or add a fourth `git-guard` — decided by whichever keeps each hook single-purpose.
2. **Single-home applied to Day-0.** `.claude/policy/DAY-0.md` becomes the authoritative home of the Day-0 rules; CLAUDE.md keeps a one-line reference instead of the full list (currently duplicated there).
3. **Hook wiring.** Claude Code executes hooks via `settings.json` entries pointing at the scripts in `.claude/hooks/`; the scripts are the enforcement, settings.json is the registration — both land in Stage 5.

**Stage 4 exit criteria — MET 2026-09-04:** the separation standard, placement matrix, and mechanism stack accepted; directory structure recorded as proposed, to be finalized during Stage-5 assembly.

## Stage 5 — Commit the final skills *(blocked on 1–4)*

Replace the draft pack in `.claude/` with the finalized standard: instructions skill, review agents, enforcement hooks (settings.json), plus AGENTS.md. From then on the standard is versioned like the project report — amendments only with recorded reasons.
