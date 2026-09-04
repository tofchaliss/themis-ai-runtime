# Themis System Architecture

**Authoritative.** This file is the single authoritative description of the system architecture. `.claude/skills/themis-harness-engineering/SKILL.md` references and operationalizes it; nothing duplicates it. Architecture changes once, here — Claude Code instructions adapt to it. Detailed baselines: `docs/architecture/` (P0 baseline, layer documents) and `docs/harness-project-report.md` (the project reference document).

## The system

```
Themis
  │
  ├── Security Governance        Findings, Enterprise Positions,
  │                              security decisions and workflows
  ├── Knowledge Builder          CVE/CWE/CPE, vendor/product and
  │                              enterprise security knowledge
  ├── Communication              presentation only — establishes nothing
  │
  └── AI Harness                 execution capability owned by Themis
       ├── L1  Instructions               how the agent must behave
       ├── L2  Context Delivery           what information is available
       ├── L3  Context Management         what is relevant
       ├── L4  Tool Interface             what the agent can do
       ├── L5  Execution Environment      where it can work
       ├── L6  Durable State              what survives failure
       ├── L7  Orchestration              what happens next
       ├── L8  Subagents                  which specialist does what
       ├── L9  Skills & Procedures        how a task type is performed
       ├── L10 Verification/Observability did it succeed; what happened
       └── L11 Ratchet                    validated improvement only
```

Themis is the security application and system of record. The AI Harness is an execution capability of Themis — architecturally owned and governed by it, whether deployed in-process or as a separate Themis-controlled process/node (deployment topology is deliberately open). A layer **owns a capability**; it does not merely contain code related to it.

## Security-data authority (Tier hierarchy)

```
Tier 0  Immutable facts (e.g. SBOM)      ─┐ authoritative
Tier 1  Deterministic security feeds     ─┘
Tier 2  Derived knowledge (deterministic from Tiers 0–1)
        AI reasoning (advisory only — never writes upward)
        Presentation (displays; establishes nothing)
```

Sensitivity is independent of tier: any datum, whatever its tier, is separately assessed for storage, transmission, logging, and exposure.

## Deterministic vs probabilistic

Deterministic (never model-based): authorization, schema validation, state transitions, version comparison, hash calculation, policy enforcement, tool permissions, approval requirements, evidence provenance, audit records, verification, instruction resolution, budgets, injection flagging.

Probabilistic (model territory): interpretation, reasoning, hypothesis generation, classification assistance, summarization, contextual analysis, natural-language explanation.

Nothing probabilistic reaches Themis except through deterministic validation, policy, and verification. The model may *propose* crossing any boundary; only deterministic machinery may *effect* it.

## Model containment

```
Themis → Harness → Agent Runtime → Model Interface → DeepSeek (or any provider)
```

DeepSeek is a replaceable reasoning component behind the Harness model interface. Nothing above the Model Interface may know which provider is configured. The model is never told it is the security system; it reasons over supplied context with neutral role framing. All AI output is advisory; acceptance into security truth requires the appropriate human or governed Themis decision.

## Where the harness lives

This repository is a monorepo: the harness Go module is at `src/harness/` (root `go.work`); the Themis application is expected to join under `src/`. Existing components — `src/harness/internal/llm` (model layer) and `src/harness/benchmarks` (themis-bench) — are the foundations of the Model Interface and Layer 11 respectively.
