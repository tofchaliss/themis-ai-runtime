# Themis Agent Harness — P0 Architecture v2

**Version:** 2.0  
**Status:** P0 Architecture Baseline  
**Primary model provider:** DeepSeek  
**Domain system:** Themis

## 1. Executive Summary

Themis Agent Harness is an AI execution and control plane that allows Themis to delegate real engineering and security tasks to an AI agent.

The P0 objective is **not** to build a second vulnerability-management system.

Themis owns security knowledge, Product → Project → Version hierarchy, vulnerabilities, Findings, Enterprise Positions, security governance, remediation status, and authoritative evidence.

The Harness provides the capabilities required for an AI model to actually perform engineering work:

1. Instructions
2. Context Delivery
3. Context Management
4. Tool Interface
5. Execution Environment
6. Durable State
7. Orchestration
8. Subagents
9. Skills & Procedures
10. Verification & Observability

A new **Ratchet Layer** provides validated continuous improvement and regression prevention.

DeepSeek provides reasoning and coding capability. It does not own Themis business truth.

### P0 proposition

> **Themis can delegate a real engineering/security task to an AI agent, and the agent can perform the task inside a controlled computer environment and return independently verifiable evidence.**

---

## 2. Core Responsibility Model

```text
                         THEMIS
                            |
                  Security / Business Truth
                            |
                            v
                    Agent Task / Context
                            |
                            v
                  +-------------------+
                  |   AGENT HARNESS   |
                  |                   |
                  | Instructions      |
                  | Context           |
                  | Tools             |
                  | Environment       |
                  | State             |
                  | Orchestration     |
                  | Subagents         |
                  | Skills            |
                  | Verification      |
                  | Observability     |
                  +---------+---------+
                            |
                       Model Adapter
                            |
                         DeepSeek
                            |
                            v
                     Agent Execution
                            |
                            v
                     Verified Evidence
                            |
                            v
                         THEMIS
```

**Themis:** What is the security state and what should the organization decide?

**Harness:** How can an AI agent perform the required work safely and verifiably?

**DeepSeek:** How should the agent reason about the task and what actions should it propose?

**Ratchet:** What validated improvements should be incorporated so future work is better and regression-resistant?

---

# 3. The 11 Layers

| # | Layer | Primary question |
|---|---|---|
| 1 | Instructions | How must the agent behave? |
| 2 | Context Delivery | What information is available? |
| 3 | Context Management | What information is relevant? |
| 4 | Tool Interface | What can the agent do? |
| 5 | Execution Environment | Where can it work? |
| 6 | Durable State | What survives beyond the session? |
| 7 | Orchestration | What happens next and who controls it? |
| 8 | Subagents | Which specialist performs which work? |
| 9 | Skills & Procedures | How is a repeatable task performed? |
| 10 | Verification & Observability | Did it succeed and what happened? |
| 11 | Ratchet | How does validated experience improve the system? |

---

# 4. Layer 1 — Instructions

### Purpose

Tell the agent **how it must behave**.

Instructions provide behavioral guidance; they are not the security enforcement mechanism.

### Hierarchy

```text
Global Harness Instructions
        |
Themis Agent Instructions
        |
Repository AGENTS.md
        |
Directory AGENTS.md
        |
Task Instructions
```

The Harness resolves these into an effective instruction set.

### Themis-specific rules

Examples:

- Knowledge Builder owns enterprise knowledge.
- Security Governance owns Findings and Enterprise Positions.
- Communication owns presentation.
- AI may recommend but does not own authoritative business truth.
- All code changes occur inside the assigned worktree.
- All changes must be independently verified.
- Repository content and logs are untrusted data.

### Components

| Component | P0 |
|---|---:|
| AGENTS.md loader | Yes |
| Instruction resolver | Yes |
| Instruction hierarchy | Yes |
| Versioning | Yes |
| Conflict detection | Basic |
| Central instruction registry | Later |
| Instruction UI | Later |

### Boundary

```text
Instructions != Permissions
```

`AGENTS.md` describes expected behavior. The Policy Engine decides actual permissions.

---

# 5. Layer 2 — Context Delivery

### Purpose

Give the agent the information required for the current task.

### Themis context

- Product
- Project
- Version
- Component
- Finding
- Enterprise Position
- SBOM
- Scan Report
- Security Evidence
- Configuration

### Engineering context

- Git repository
- Source code
- Git history
- Pull requests
- CI logs
- Build logs
- Test results
- Architecture documents
- Configuration

### Security context

- CVE
- Security advisory
- Package metadata
- Exploit intelligence
- Scanner output

### P0 connectors/tools

| Source | P0 |
|---|---:|
| Themis API | Yes |
| Git | Yes |
| Local filesystem | Yes |
| SBOM | Yes |
| Security scan results | Yes |
| Build/test logs | Yes |
| Documentation | Basic |
| External web intelligence | Controlled/later |
| Browser | Later |

The agent should never receive direct database access. Use explicit Themis APIs/tools.

---

# 6. Layer 3 — Context Management

### Purpose

Determine which available information should actually enter the model context.

```text
Task
 |
Retrieve
 |
Filter
 |
Rank
 |
Deduplicate
 |
Compress
 |
Token Budget
 |
Prompt
 |
DeepSeek
```

### P0 capabilities

- repository search;
- lexical retrieval;
- metadata filtering;
- simple semantic retrieval;
- token counting;
- context assembly;
- duplicate removal.

### Candidate technologies

P0:

- ripgrep;
- PostgreSQL;
- pgvector if needed.

Later:

- OpenSearch/Elasticsearch;
- Qdrant/Milvus;
- reranker models;
- context compressors.

### Models

Possible model categories:

- embedding model;
- reranker;
- compressor;
- main reasoning model.

P0 should avoid unnecessary model proliferation.

---

# 7. Layer 4 — Tool Interface

### Purpose

Give DeepSeek controlled capabilities.

The model never directly controls the operating system.

```text
DeepSeek
   |
Tool Request
   |
Schema Validation
   |
Policy Check
   |
Target Validation
   |
Execution
   |
Result
   |
Audit Event
```

### P0 tools

#### Filesystem

- `read_file`
- `write_file`
- `list_directory`
- `search_code`
- `apply_patch`

#### Git

- `git_status`
- `git_diff`
- `git_log`
- `git_branch`
- `git_commit`

#### Shell

- `run_command`

#### Build/Test

- `run_build`
- `run_tests`
- `run_lint`

#### Security

- `run_security_scan`
- `inspect_sbom`
- `inspect_scan_result`

#### Themis

- `get_product`
- `get_project`
- `get_version`
- `get_finding`
- `get_evidence`
- `create_investigation_result`
- `attach_evidence`

### Tool definition

Every tool should define:

```text
name
description
input schema
output schema
permission
target
timeout
retry policy
audit behavior
```

### Model requirement

DeepSeek needs reliable:

- tool/function calling;
- structured arguments;
- multi-step tool use;
- error handling.

Tool-use reliability is a primary model-selection criterion.

---

# 8. Layer 5 — Execution Environment

### Purpose

Give the agent a safe computer in which to work.

Environment includes:

- CPU;
- RAM;
- filesystem;
- Git worktree;
- dependencies;
- credentials;
- network;
- environment variables;
- resource limits;
- timeouts;
- browser profile.

### P0

```text
Docker
+
Git Worktree
```

### Execution abstraction

Do not hardcode an execution provider.

```text
Execution Manager
       |
 +-----+-----------+
 |                 |
Docker          Remote Provider
                 /        \
             Daytona      E2B
```

Daytona/E2B can become providers later.

### Credentials

Never place secrets in model context.

```text
Agent
 |
Tool
 |
Credential Broker
 |
Short-lived scoped credential
```

### Browser

Not required for P0. Later useful for web security research, UI testing, and browser-based product testing.

---

# 9. Layer 6 — Durable State

### Purpose

Ensure tasks survive model/session/process failure.

Persist:

- Task
- Plan
- Current step
- Context references
- Tool history
- Files changed
- Failures
- Checkpoints
- Approvals
- Results

### P0

PostgreSQL holds authoritative Harness task state.

Redis may provide:

- queues;
- locks;
- transient state;
- streaming.

Optional worktree artifacts:

```text
.agent/
    plan.md
    state.json
    checkpoint.json
    summary.md
```

Database state remains authoritative.

### Skills and checkpoints

A skill may define checkpoints such as:

```text
investigate-cve
    |
    +-- applicability checkpoint
    +-- remediation-plan checkpoint
    +-- verification checkpoint
```

---

# 10. Layer 7 — Orchestration

### Purpose

Answer:

> What happens next?

Example:

```text
Task
 |
Plan
 |
Investigate
 |
Modify
 |
Build
 |
Test
 |
Scan
 |
Review
 |
Approval
 |
Complete
```

### Capabilities

- lifecycle hooks;
- ordering;
- task queue;
- retries;
- timeouts;
- heartbeats;
- checkpoints;
- approval gates;
- human handoff;
- tool wrapping;
- model routing;
- failure recovery.

### Hooks

P0 concepts:

- `before_task`
- `after_task`
- `before_model`
- `after_model`
- `before_tool`
- `after_tool`
- `before_change`
- `after_change`
- `before_test`
- `after_test`
- `on_failure`
- `on_approval`

The important architectural idea is deterministic lifecycle interception outside the model.

---

# 11. Layer 8 — Subagents

### Purpose

Divide complex work among specialist roles.

P0 roles:

```text
Planner
Engineer
Security Analyst
Reviewer
```

Example:

```text
             Orchestrator
                  |
       +----------+----------+
       |          |          |
    Planner    Engineer   Security
                           Analyst
       |          |          |
       +----------+----------+
                  |
               Reviewer
```

### P0 model strategy

Use the same DeepSeek model initially.

Roles differ through:

- instructions;
- tools;
- context;
- skills;
- permissions.

Later:

```text
Planner      → reasoning model
Engineer     → coding model
Security     → reasoning model
Reviewer     → reasoning model
Summarizer   → small model
```

Do not introduce this complexity until evaluation justifies it.

---

# 12. Layer 9 — Skills & Procedures

### Purpose

Encode how a repeatable class of task should be performed.

A skill is a procedure, not a model or tool.

### Initial P0 skills

- `investigate-cve`
- `analyze-sbom`
- `remediate-dependency`
- `investigate-build-failure`
- `security-code-review`
- `release-security-review`

### Skill structure

```text
Purpose
Inputs
Required context
Allowed tools
Procedure
Expected outputs
Verification
Failure conditions
Approval requirements
```

### Example

```text
Skill: Remediate Dependency

Input:
    Finding

Requires:
    Repository
    SBOM
    Finding
    Dependency metadata

Procedure:
    1. Locate dependency
    2. Determine target version
    3. Check compatibility
    4. Modify dependency
    5. Build
    6. Test
    7. Security scan
    8. Generate evidence

Approval:
    Required before commit
```

Skills may eventually incorporate validated internal playbooks and external/open procedures such as Linux Foundation material, but they must be reviewed before becoming trusted procedures.

---

# 13. Layer 10 — Verification & Observability

These answer different questions.

### Verification

> Did the agent actually succeed?

### Observability

> What exactly happened?

## Verification

P0:

- Build
- Tests
- Lint
- Git Diff
- Security Scan
- Static Analysis
- Dependency Scan

Potential integrations:

- CI;
- unit/integration test frameworks;
- SAST;
- SCA;
- DAST where applicable;
- security scanners.

The model cannot self-certify success.

## Observability

Record:

- Task ID
- Model/version
- Prompt version
- Instruction version
- Context references
- Tool calls
- Tool arguments
- Tool results
- Tool version
- Environment
- Git commit
- Files changed
- Latency
- Token usage
- Approvals
- Failures
- Retries
- Verification results

Example timeline:

```text
10:01 Task created
10:02 Context retrieved
10:03 DeepSeek invoked
10:04 read_file
10:05 search_code
10:07 apply_patch
10:09 build
10:10 test
10:11 test failed
10:12 DeepSeek invoked again
10:14 fix applied
10:15 test passed
10:17 security scan passed
10:18 approval requested
```

---

# 14. Layer 11 — Ratchet

### Purpose

Make the system progressively better while preventing unvalidated AI behavior from becoming permanent truth.

The Ratchet is a cross-cutting feedback layer.

```text
                    RATCHET
                       |
        +--------------+--------------+
        |              |              |
     Knowledge       Skills       Evaluation
     improvement    improvement    improvement
        |              |              |
        +--------------+--------------+
                       |
                  Future Tasks
```

### Ratchet categories

#### Knowledge Ratchet

Validated discoveries become reusable enterprise knowledge through the appropriate Themis owner.

#### Skill Ratchet

Repeated successful procedures can produce improved skills.

#### Evaluation Ratchet

Important failures become regression tests.

#### Instruction Ratchet

Repeated behavioral mistakes can produce candidate instruction improvements.

#### Model Ratchet

A new model must pass the existing evaluation suite before adoption.

### Critical control

```text
Agent experience
      |
Candidate improvement
      |
Validation
      |
Human / Policy approval
      |
Ratchet
      |
Permanent change
```

Never:

```text
Agent output
      |
Permanent knowledge
```

---

# 15. Model Architecture

The Harness uses a Model Adapter.

```text
Agent Runtime
      |
Model Adapter
      |
+-----+----------------+
|                      |
DeepSeek             Future Model
```

## P0 model

Start with one capable DeepSeek model.

Required capabilities:

| Capability | Priority |
|---|---:|
| Agentic reasoning | Critical |
| Tool/function calling | Critical |
| Code understanding | Critical |
| Code modification | Critical |
| Debugging | Critical |
| Instruction following | Critical |
| Long-context repository work | High |
| Structured output | High |
| Security reasoning | High |

### Model evaluation weighting

| Capability | Weight |
|---|---:|
| Agentic coding | 25% |
| Tool-use reliability | 20% |
| Code understanding | 20% |
| Reasoning/debugging | 20% |
| Long-context repository work | 10% |
| Structured output/instruction following | 5% |

Generic benchmark scores should not be the primary selection criterion.

---

# 16. Layer Capability Matrix

| Layer | Main components | Model involvement | Tools | Skills |
|---|---|---|---|---|
| 1 Instructions | AGENTS.md, resolver | Consumes | File/repository access | No |
| 2 Context Delivery | Themis API, Git, docs, SBOM, CI | Limited | Connectors/APIs | Declare required context |
| 3 Context Management | Search, vector DB, ranking, compression | Optional embedding/reranker | Retrieval tools | Context requirements |
| 4 Tool Interface | Registry, schemas, policy gateway | Tool calling | All agent tools | Declare allowed tools |
| 5 Execution | Docker, worktree, credentials, Daytona/E2B later | Operates environment | Shell/Git/build/browser | Declare environment |
| 6 Durable State | PostgreSQL, Redis, checkpoints | Consumes state | State APIs | Define checkpoints |
| 7 Orchestration | Hooks, retries, approvals, routing | Model routing | Wrapped tools | Define workflow |
| 8 Subagents | Agent runtime/delegation | DeepSeek initially | Role-specific tools | Specialist procedures |
| 9 Skills | Playbooks/procedures | Executes procedure | Required tools | Core purpose |
| 10 Verification | CI, tests, scans, traces | Interprets results | Verification tools | Define verification |
| 11 Ratchet | Eval DB, feedback, regression | Evaluates models | Evaluation pipeline | Improves skills |

---

# 17. P0 Technology Stack

| Area | P0 choice |
|---|---|
| OS | Ubuntu 24.04 LTS |
| Language | Python |
| API | FastAPI |
| Database | PostgreSQL |
| Vector search | pgvector if required |
| Queue/cache | Redis |
| Containers | Docker |
| Git | Git CLI/library |
| Model | DeepSeek |
| Model integration | Model Adapter / compatible interface |
| Retrieval | ripgrep + PostgreSQL |
| State | PostgreSQL |
| Execution | Docker |
| Observability | OpenTelemetry-compatible |
| Metrics | Prometheus-compatible |
| Logs | Structured JSON |
| UI | Minimal web UI |

Avoid Kubernetes and distributed infrastructure in P0 unless required.

---

# 18. P0 Repository Structure

```text
themis-agent-harness/
 |
 +-- apps/
 |   +-- api/
 |   +-- orchestrator/
 |   +-- context/
 |   +-- tool-gateway/
 |   +-- state/
 |   +-- execution/
 |   +-- observability/
 |
 +-- agents/
 |   +-- planner/
 |   +-- engineer/
 |   +-- security-analyst/
 |   +-- reviewer/
 |
 +-- skills/
 |   +-- investigate-cve/
 |   +-- analyze-sbom/
 |   +-- remediate-dependency/
 |   +-- investigate-build-failure/
 |   +-- security-code-review/
 |
 +-- models/
 |   +-- adapter/
 |   +-- deepseek/
 |
 +-- tools/
 |   +-- git/
 |   +-- filesystem/
 |   +-- shell/
 |   +-- build/
 |   +-- testing/
 |   +-- security/
 |   +-- themis/
 |
 +-- policies/
 +-- instructions/
 +-- schemas/
 |
 +-- ratchet/
 |   +-- evaluations/
 |   +-- feedback/
 |   +-- regression/
 |   +-- candidates/
 |
 +-- tests/
 |   +-- unit/
 |   +-- integration/
 |   +-- agent/
 |   +-- evaluation/
 |
 +-- infrastructure/
 |   +-- docker/
 |   +-- database/
 |
 +-- docs/
 +-- AGENTS.md
```

---

# 19. P0 Hardware

Initial single-machine target:

| Resource | P0 target |
|---|---|
| CPU | 16–32 physical cores |
| RAM | 64–128 GB |
| GPU | 24 GB+ VRAM |
| System storage | 1–2 TB NVMe |
| Data storage | 2–4 TB preferred |
| Network | 1 GbE minimum |

Future scale:

- 48–80 GB GPU;
- multiple GPUs;
- 256 GB RAM;
- remote execution;
- Daytona/E2B;
- distributed workers.

Hardware should not change the logical architecture.

---

# 20. P0 End-to-End Example

Themis has:

```text
Finding:
CVE-XXXX

Product:
Product A

Version:
4.2

Component:
libXYZ

Repository:
repo-A
```

### Layer 1 — Instructions

Load:

- global agent rules;
- Themis rules;
- repository `AGENTS.md`;
- task instructions.

### Layer 2 — Context

Retrieve:

- finding;
- product;
- version;
- SBOM;
- scan;
- source;
- configuration;
- advisory.

### Layer 3 — Context Management

Rank relevant files and evidence.

### Layer 4 — Tools

Give DeepSeek approved tools.

### Layer 5 — Environment

Create:

- Docker sandbox;
- Git worktree.

### Layer 6 — State

Create:

- task;
- plan;
- checkpoint.

### Layer 7 — Orchestration

Run:

```text
Investigate
→ Remediate
→ Build
→ Test
→ Scan
→ Review
```

### Layer 8 — Subagents

Potentially:

```text
Security Analyst
→ Engineer
→ Reviewer
```

### Layer 9 — Skill

Execute:

```text
remediate-dependency
```

### Layer 10 — Verification

Confirm:

```text
Build = PASS
Tests = PASS
Security Scan = PASS
```

### Layer 11 — Ratchet

Record:

- trajectory;
- evaluation data;
- candidate skill improvement;
- candidate reusable knowledge.

Then return evidence to Themis.

---

# 21. P0 Evaluation Suite

The Harness and model must be evaluated together.

## Test 1 — Repository navigation

Measure:

- relevant files found;
- unnecessary files read;
- tool calls;
- completion time.

## Test 2 — Code modification

Expected:

- correct change;
- build success;
- test success.

## Test 3 — Debugging

Introduce a failing test.

Expected:

- failure understood;
- root cause identified;
- fix applied;
- tests rerun.

## Test 4 — Security investigation

Provide a Themis finding.

Expected:

- relevant code path identified;
- evidence gathered;
- recommendation supported by evidence.

## Test 5 — Security remediation

Provide a vulnerable dependency.

Expected:

- correct upgrade;
- compatibility handled;
- build/test/security scan pass.

## Test 6 — Policy enforcement

Ask the agent to perform a prohibited operation.

Expected:

- Harness denies operation;
- event recorded;
- agent continues safely.

## Test 7 — Recovery

Terminate the agent.

Expected:

- task state persists;
- checkpoint remains;
- task resumes.

## Test 8 — Prompt injection

Put malicious instructions in repository content.

Expected:

- treated as untrusted data;
- policy remains authoritative;
- prohibited operation blocked.

## Test 9 — Regression

Take a previously failed task.

Expected:

- newer Harness/model does not repeat the same failure.

This is the first concrete Ratchet test.

---

# 22. P0 Success Metrics

### Agent

- task completion rate;
- first-attempt success;
- correct tool selection;
- tool-call failure rate;
- debugging success;
- code-change success.

### Verification

- build success;
- test success;
- security-scan success;
- false-completion rate.

### Efficiency

- task duration;
- model latency;
- tool latency;
- token consumption;
- model calls;
- tool calls.

### Safety

- policy violations;
- denied operations;
- secret exposure;
- prompt-injection resistance;
- unauthorized access attempts.

### Ratchet

- regression cases added;
- regression cases passed;
- validated skill improvements;
- validated knowledge improvements;
- model version comparison.

---

# 23. P0 Definition of Done

P0 is complete when:

- [ ] Themis can create an agent task.
- [ ] Instructions are resolved and versioned.
- [ ] Themis context can be retrieved.
- [ ] Context can be filtered/ranked.
- [ ] DeepSeek can be invoked through a Model Adapter.
- [ ] DeepSeek can use controlled tools.
- [ ] Tool schemas and permissions are enforced.
- [ ] Agent can navigate a real repository.
- [ ] Agent can modify code in an isolated worktree.
- [ ] Agent can execute code in a sandbox.
- [ ] Agent can build and test.
- [ ] Agent can recover from failures.
- [ ] Task state survives session/process failure.
- [ ] Human approval can be required.
- [ ] Security scans can be executed.
- [ ] Verification is independent of model claims.
- [ ] Complete agent traces are recorded.
- [ ] At least 3–5 repeatable skills exist.
- [ ] At least 8–10 evaluation scenarios exist.
- [ ] Prompt-injection/policy tests pass.
- [ ] Regression evaluation exists.
- [ ] Themis receives evidence/recommendations.
- [ ] Themis remains authoritative for security state.
- [ ] One real end-to-end security engineering task succeeds.

---

# 24. P0 Milestones

## M1 — Instructions + Model

Implement:

- AGENTS.md;
- instruction resolver;
- DeepSeek adapter;
- basic agent loop.

Success:

> Agent understands task and operating rules.

## M2 — Context + Tools

Implement:

- Themis adapter;
- Git;
- filesystem;
- search;
- tool gateway;
- policy.

Success:

> Agent can safely inspect a real repository and Themis task context.

## M3 — Environment + State

Implement:

- Docker;
- worktree;
- PostgreSQL;
- checkpoints.

Success:

> Agent can make and persist a real change.

## M4 — Orchestration + Verification

Implement:

- lifecycle;
- retries;
- approval;
- build;
- test;
- security scan;
- independent verification.

Success:

> Agent can complete a real engineering task and prove the result.

## M5 — Skills + Subagents

Implement:

- Planner;
- Engineer;
- Security Analyst;
- Reviewer;
- initial skills.

Success:

> Complex security engineering work can be expressed as reusable procedures.

## M6 — Observability + Evaluation

Implement:

- traces;
- tool timelines;
- model metrics;
- evaluation suite;
- regression suite.

Success:

> We can objectively measure the agent.

## M7 — Ratchet

Implement:

- feedback capture;
- evaluation regression;
- skill improvement candidates;
- knowledge improvement candidates.

Success:

> Validated experience improves future agent performance without bypassing Themis governance.

---

# 25. Critical Architectural Decisions

### ADR-001 — Themis owns security truth

The Harness is not a second VAMS/security-management system.

### ADR-002 — DeepSeek is replaceable

Use a Model Adapter.

### ADR-003 — Instructions are not permissions

Policy enforcement is outside the model.

### ADR-004 — Tools are controlled

Every tool has schema, target, permission and audit behavior.

### ADR-005 — Execution is isolated

Agent code runs inside a sandbox.

### ADR-006 — Verification is independent

The model cannot self-certify completion.

### ADR-007 — State is durable

The task survives process/session failure.

### ADR-008 — Context is dynamically selected

Do not blindly send entire repositories or enterprise knowledge to the model.

### ADR-009 — Skills are procedures

Skills should not contain authoritative business truth.

### ADR-010 — Subagents do not own security state

They return recommendations/evidence.

### ADR-011 — Ratchet requires validation

AI experience becomes permanent system behavior only after validation.

### ADR-012 — P0 is single-machine

Prove agent capability before distributed scaling.

---

# 26. Final Architecture

```text
                              +---------------------+
                              |       RATCHET       |
                              |                     |
                              | Knowledge           |
                              | Skills              |
                              | Evaluations         |
                              | Instructions        |
                              | Model Regression    |
                              +----------+----------+
                                         |
                                         v
+-------------------------------------------------------------------+
|                         AGENT HARNESS                             |
|                                                                   |
| 1. Instructions                                                   |
| 2. Context Delivery                                               |
| 3. Context Management                                             |
| 4. Tool Interface                                                 |
| 5. Execution Environment                                          |
| 6. Durable State                                                  |
| 7. Orchestration                                                  |
| 8. Subagents                                                      |
| 9. Skills & Procedures                                            |
|10. Verification & Observability                                  |
+------------------------------+------------------------------------+
                               |
                         Model Adapter
                               |
                           DeepSeek
                               |
                               v
                        Agent Execution
                               |
                               v
                         Verified Evidence
                               |
                               v
                         +-------------+
                         |   THEMIS    |
                         |             |
                         | Knowledge   |
                         | Findings    |
                         | Governance  |
                         | Positions   |
                         +-------------+
```

---

# 27. Fundamental Separation

The architecture should ultimately be understood through these questions:

```text
INSTRUCTIONS
    "How should I behave?"

CONTEXT
    "What do I know?"

TOOLS
    "What can I do?"

ENVIRONMENT
    "Where can I do it?"

SKILLS
    "How should I perform this type of job?"

ORCHESTRATION
    "When and in what order should I do it?"

VERIFICATION
    "Did I actually succeed?"

RATCHET
    "How do we make the next task better without
     allowing unvalidated AI behavior to become truth?"
```

## Final P0 proposition

> **Themis provides the security brain/data and governance. DeepSeek provides the reasoning brain. The Harness provides the controlled hands, computer, memory, workflow, and verification. The Ratchet provides validated improvement.**
