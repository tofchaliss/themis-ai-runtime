# Themis AI Runtime — Architecture

## Purpose
Themis is the security application and system of record. The AI Harness is an execution capability of Themis, architecturally owned and governed by Themis.

## Deployment Boundary
The AI Harness is an execution capability of Themis—architecturally owned and governed by Themis, whether deployed in-process or as a separate Themis-controlled process/node. Deployment topology is an explicit architecture decision.

## Harness Capability Layers
1. Instructions
2. Context Delivery
3. Context Management
4. Tool Interface
5. Execution Environment
6. Durable State
7. Orchestration
8. Subagents
9. Skills and Procedures
10. Verification and Observability
11. Ratchet

These are capability/ownership boundaries, not a requirement for eleven source-code directories.

## Ownership
### Themis owns
Security truth, Findings, Enterprise Positions, Security Governance, Enterprise Knowledge through the Knowledge Builder, and authorized security workflows.

### Harness owns
AI execution/runtime coordination, context handling, authorized tool invocation, orchestration, runtime execution controls, and verification/observability of Harness execution.

### Model adapter owns
Model-specific protocol integration and transport concerns.

### Model owns
Probabilistic reasoning only.

The Harness must not become a second vulnerability-management or security-governance system.

## Security Evidence and Authority
Authority and sensitivity are separate dimensions.

Authority hierarchy:
1. Tier 0 — immutable security facts
2. Tier 1 — deterministic security feeds
3. Tier 2 — derived enterprise knowledge
4. AI reasoning
5. Presentation

External content is untrusted input even when it is authoritative evidence for a specific security fact. Provenance must be preserved.

## Deterministic / Probabilistic Boundary
Deterministic mechanisms own authorization, permissions, policy, schema validation, state transitions, version comparisons, hashes, provenance, audit, verification, and security invariants.

Probabilistic reasoning may provide interpretation, contextual analysis, hypotheses, summarization, and explanation.

Model output is never security authority.

Required flow:
Model output -> validation -> policy -> authorization -> execution -> verification -> governed result.

## Authority Separation
- Instruction != Context
- Context != Authority
- Tool != Authority
- Model output != Authority
- Harness state != Themis security truth

The Harness may write to Themis only through explicit Themis-owned interfaces/workflows. It must not directly establish competing security truth.

## External Systems
External-system access is policy-gated and local-first by default. Data egress to a non-local system requires explicit, recorded operator authorization.

## Security Data Protection
Dependencies are minimal and individually justified. Sensitivity is independently considered for storage, transmission, logging, and exposure. Sensitive values are redacted before durable storage or logs.

## Failure and Safety
The system must remain safe when models, external content, tools, dependencies, or external systems behave unexpectedly or maliciously.

Security controls fail closed. Model confidence never substitutes for authorization. Secrets are not ordinary model context. Execution is isolated and constrained according to risk. Destructive or irreversible operations require deterministic policy controls, required approval, and verification.

## Decision Authority
All AI output is advisory. Acceptance into security truth requires the appropriate human or governed Themis decision. A governed automated decision is not AI self-authority.

## Architecture Changes
Changes to architectural ownership, security authority, trust boundaries, deployment topology, or fundamental model architecture are explicit architecture decisions and must not be inferred by development tooling.
