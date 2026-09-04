# Themis System Architecture

**Authoritative.** This file is the single authoritative description of the system architecture — components, boundaries, ownership, and data/control flows. It does not contain Claude Code operating procedure (that is `.claude/skills/themis-harness-engineering/SKILL.md`, which references and operationalizes this file without duplicating it). Architecture changes once, here — development instructions adapt to it. Detailed baselines: `docs/architecture/` and `docs/harness-project-report.md`.

## A. System purpose

The Themis Agent Harness exists so that Themis — the security application and system of record — can delegate real engineering and security tasks to an AI agent, have them performed inside a controlled computer environment, and receive independently verifiable evidence back. The Harness supplies everything the model cannot be trusted to own: rules, context selection, controlled tools, an isolated workplace, durable state, orchestration, verification, and audit.

## B. System boundary

```
                    ┌─────────────────────────┐
                    │         Themis          │
                    │ Security System of       │
                    │ Record / Authority       │
                    └────────────┬────────────┘
                                 │
                         authorized boundary
                                 │
                    ┌────────────▼────────────┐
                    │      AI Harness          │
                    │                          │
                    │  11 capability layers   │
                    │                          │
                    │  Instructions            │
                    │  Context Delivery        │
                    │  Context Management      │
                    │  Tool Interface          │
                    │  Execution Environment   │
                    │  Durable State           │
                    │  Orchestration           │
                    │  Subagents               │
                    │  Skills / Procedures     │
                    │  Verification / Obs.     │
                    │  Ratchet                 │
                    └────────────┬────────────┘
                                 │
                         model abstraction
                                 │
                    ┌────────────▼────────────┐
                    │       Model(s)           │
                    │ replaceable reasoning    │
                    │ component                │
                    └─────────────────────────┘
```

**This diagram does not imply deployment topology.** The Harness may eventually be in-process, a separate process, or another Themis-controlled deployment unit. That remains an architectural decision (a stop+ask condition) — never something to infer.

A layer **owns a capability**; it does not merely contain code related to that capability. If a capability's decisions are made outside its layer, the boundary is broken regardless of where the files sit.

## C. Security authority

```
                    SECURITY AUTHORITY
                           │
                           ▼
              ┌────────────────────────┐
              │ Tier 0                 │
              │ Immutable security     │
              │ facts                   │
              │ SBOM / artifacts /     │
              │ observed facts         │
              └───────────┬────────────┘
                          ▼
              ┌────────────────────────┐
              │ Tier 1                 │
              │ Deterministic security │
              │ feeds                   │
              │ CVE / VEX / scanner /  │
              │ vendor security data   │
              └───────────┬────────────┘
                          ▼
              ┌────────────────────────┐
              │ Tier 2                 │
              │ Derived enterprise     │
              │ knowledge              │
              └───────────┬────────────┘
                          ▼
              ┌────────────────────────┐
              │ AI reasoning           │
              │ interpretation /       │
              │ hypothesis / analysis  │
              └───────────┬────────────┘
                          ▼
              ┌────────────────────────┐
              │ Presentation            │
              └────────────────────────┘
```

Tiers 0–1 are authoritative; Tier 2 is derived from them deterministically. AI reasoning consumes the tiers and produces advisory output; it can never write upward into any tier. Presentation displays; it establishes nothing. Sensitivity is independent of tier: every datum is separately assessed for storage, transmission, logging, and exposure.

## D. Architectural ownership

```
Themis
 ├── Security Truth
 ├── Findings
 ├── Enterprise Positions
 ├── Security Governance
 ├── Knowledge Builder
 └── Authorized Security Workflows
          │
          ▼
       Harness
          │
          ├── reasoning
          ├── orchestration
          ├── tool invocation
          ├── context management
          ├── verification
          └── runtime execution
          │
          ▼
        Model
```

The Harness reads/writes authoritative state only through Themis-owned interfaces — never direct database access. Human authority is final: all AI output is advisory, and acceptance into security truth requires the appropriate human or governed Themis decision.

## E. Model boundary

```
Harness
   │
   │ Model Interface
   ▼
┌───────────────┐
│ Model Adapter │
└───────┬───────┘
        │
        ▼
     Model
```

The model (DeepSeek or any provider) is a replaceable reasoning component. Nothing above the Model Interface may know which provider is configured; provider-specific behavior lives only in the adapter/registry. The model is never told it is the security system — it reasons over supplied context with neutral role framing.

## F. Deterministic / probabilistic boundary

One of the strongest architecture rules.

**Deterministic** (never model-based): authorization · permissions · policy · schema validation · state transitions · version comparison · hashing · provenance · audit · verification · security invariants.

**Probabilistic** (model territory): interpretation · contextual reasoning · hypothesis generation · summarization · explanation · analytical assistance.

Therefore every model output passes the deterministic pipeline before anything Themis-owned exists:

```
Model output
     │
     ▼
Validate
     │
     ▼
Policy
     │
     ▼
Authorize
     │
     ▼
Execute
     │
     ▼
Verify
     │
     ▼
Themis-owned result
```

The model may *propose* crossing any boundary; only deterministic machinery may *effect* it. Model scoring/evaluation is deterministic (no LLM-as-judge in any gate path).

## G. External boundary system

External-system access is policy-gated, local-first by default, and data egress to non-local systems requires explicit, recorded operator authorization. This applies broadly:

- remote models
- web intelligence
- external APIs
- repositories
- scanners
- external security services
- remote execution environments

This boundary exists so the system is never accidentally redesigned into a web-enabled autonomous agent without recognizing that a trust boundary was crossed.

## H. Where this lives

This repository is a monorepo: the harness Go module is at `src/harness/` (root `go.work`); the Themis application is expected to join under `src/`. Existing components — `src/harness/internal/llm` (model layer) and `src/harness/benchmarks` (themis-bench) — are the foundations of the Model Adapter and the Ratchet's evaluation half (L11) respectively.
