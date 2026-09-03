# Themis Agent Harness — P0 Project Plan

**Status:** Draft for discussion
**Baseline:** [`architecture/harness-p0-architecture-v2.md`](architecture/harness-p0-architecture-v2.md), as amended by the 2026-09-03 decisions (Go; grown in this repo; local-default models with policy-gated hosted DeepSeek; thin-vertical-slice P0)
**Assumed staffing:** one senior engineer working with AI assistance (estimates below are in that unit; scale accordingly)

---

## 1. Strategy: two vertical slices, then build out

The architecture doc's M1–M7 milestones build the 11 layers horizontally. The locked P0 decision inverts that: prove the P0 proposition first — *one real task, performed in a controlled environment, returning independently verifiable evidence* — through the thinnest slice that touches every layer, then widen.

Two slices, chosen so the second reuses everything the first builds:

- **Slice 1 — `investigate-cve` (read-only).** The agent receives a Themis Finding, explores a repository in a sandbox, gathers evidence, and returns a grounded applicability recommendation. No code modification, no build/test verification — the verification is evidence-grounding. This exercises Layers 1–7 + 10 with the smallest tool surface.
- **Slice 2 — `remediate-dependency` (read-write).** The agent modifies a dependency in an isolated worktree, builds, tests, scans, and returns evidence. This adds write tools, build/test/scan verification, and the approval gate — the full P0 proposition.

Everything after Slice 2 is widening: more skills, subagents, the Ratchet, the hosted-model egress path.

---

## 2. Phase flow

```
Phase 0          Phase 1           Phase 2            Phase 3         Phase 4          Phase 5
Model spike ──▶  Model layer  ──▶  Slice-1 core  ──▶  Slice 1    ──▶  Slice 2     ──▶  Eval &
& decisions      (chat+tools)      (state, loop,      complete        (write path,     hardening
                                   tools, sandbox)    (skill, trace,  verification,
                                                      Themis return)  approval)
                                        │
                              Increment 2 (parallel after Phase 3):
                              hosted DeepSeek egress path, credential broker
```

### Phase 0 — Model spike and blocking decisions (1 week)

The single biggest project risk is **tool-calling reliability of local models** (the architecture doc itself names it the primary model-selection criterion). Resolve it empirically before building anything on it.

| Work item | Output |
|---|---|
| Extend themis-bench with a first **"Agentic" benchmark category**: 3–5 scripted multi-turn tool-call scenarios scored deterministically (correct tool chosen, valid JSON args, recovery from an error result) | Tool-call reliability scores per candidate model |
| Run candidates: `deepseek-coder-v2:16b-lite`, DeepSeek-R1-Distill-Qwen-14B/32B (q4), plus 1–2 non-DeepSeek controls (e.g. `qwen2.5-coder:32b`) via Ollama `/api/chat` | Model shortlist; go/no-go on local-only Slice 1 |
| Decide the two open architecture forks: **L9 skill format** (Markdown procedure vs executable state machine) and **L4 `run_command` policy** (allowlist vs sandbox-and-allow) | ADR entries in `docs/architecture/` |
| Amend the v2 doc (§17 stack, §15 model, §18 layout, §24 milestones) per the locked decisions | v2.1 baseline |

**Gate:** at least one local model passes the agentic scenarios at an acceptable rate. If none does, the hosted-DeepSeek increment moves *forward* in the plan (with its egress-policy cost) — better to know in week 1.

### Phase 1 — Model layer evolution (1.5 weeks)

`src/harness/internal/llm` today speaks stateless `/api/generate` with a single prompt. The harness needs conversations and native tool calls.

| Work item | Notes |
|---|---|
| New `ChatRuntime` interface: messages (system/user/assistant/tool), tool definitions, tool-call responses | Sits beside the existing `Runtime`; themis-bench keeps using the old one unchanged |
| Ollama `/api/chat` implementation with `tools` | Existing pinned options (temp 0, seed 42) carry over |
| OpenAI-compatible chat implementation with `tools` | Same interface serves vLLM later and the hosted DeepSeek API in Increment 2 |
| Registry extension: per-model chat capability, context window size | `models.json` schema addition |
| Scripted-conversation `httptest` mocks + table-driven tests | Same test philosophy as today; no live model needed |

### Phase 2 — Slice-1 core: state, loop, tools, sandbox (3.5 weeks)

The minimum machinery under any agent task. All new code under `src/harness/internal/` (public API surface extracted later, once stable — themis integrates over HTTP, so nothing needs to be importable yet).

| Work item | Layer | Est. |
|---|---|---|
| Postgres task state: `tasks`, `steps`, `tool_calls`, `checkpoints`, `approvals` tables + migrations; store package | L6 | 4 d |
| Task API (HTTP): create task, get status, list steps, approve/reject | L7 | 2 d |
| Agent loop: build messages → chat call → dispatch tool calls → persist every step → termination (goal/step-cap/token-budget/timeout) | L7 | 4 d |
| Instruction resolver (M1 scope): harness rules + Themis domain rules + task instructions; deterministic precedence; SHA-256 of effective set | L1 | 2 d |
| Tool gateway: registry, JSON-schema validation of args, policy check (per-skill allowlist), audit event per call | L4 | 3 d |
| Read-only toolset: `read_file`, `list_directory`, `search_code` (ripgrep), `git_log`, `git_diff`, `inspect_sbom`, `get_finding` (Themis API stub) | L2/L4 | 2 d |
| Execution environment: Docker container + read-only repo mount for Slice 1; container lifecycle tied to task | L5 | 3 d |

**Gate:** an agent task can run end-to-end against a toy repo with scripted-model tests, and against a real local model manually, with every step persisted and resumable after process kill (architecture Test 7).

### Phase 3 — Slice 1 complete: skill, trace, Themis return (2 weeks)

| Work item | Layer | Est. |
|---|---|---|
| `investigate-cve` skill (format per Phase-0 decision): required context, allowed tools, procedure, expected output schema, failure conditions | L9 | 3 d |
| Context assembly (M-scope of L3): fetch Finding/SBOM/advisory, rank files by lexical relevance, token-budgeted prompt assembly — no vector DB yet | L2/L3 | 3 d |
| Result contract: schema-validated advisory output, `requires_human_decision` forced true, grounding check (cited evidence must exist in gathered context) | L10 | 2 d |
| Full trace: per-step model/options/tokens/latency, tool timeline, instruction + prompt hashes; JSON export | L10 | 2 d |
| Evidence return to Themis: signed result envelope posted to a Themis inbox endpoint (coordinate with the themis repo's `readapi.ProposalWriter` seam) | integration | 2 d |

**Gate = Slice-1 demo:** a real Finding, investigated by a local model in a sandbox, returning a grounded recommendation with a complete trace — the first half of the P0 Definition of Done.

### Phase 4 — Slice 2: write path, verification, approval (2.5 weeks)

| Work item | Layer | Est. |
|---|---|---|
| Write tools: `write_file`, `apply_patch`, `git_branch`, `git_commit` — worktree-scoped, policy-gated | L4 | 2 d |
| Git worktree isolation inside the sandbox; diff capture | L5 | 2 d |
| Verification tools: `run_build`, `run_tests`, `run_lint`, `run_security_scan` (start with Grype/`osv-scanner` on the SBOM) — results recorded independently of model claims | L10 | 3 d |
| Approval gate: task pauses before commit; human approve/reject via API; resume | L7 | 2 d |
| `remediate-dependency` skill with checkpoints (locate → target version → modify → build → test → scan → evidence) | L9 | 3 d |
| Prompt-injection defenses: repo content marked untrusted in prompts; injection scan on tool outputs (reuse existing `SuspectInjection`); policy test (architecture Test 8) | L1/L4 | 1 d |

**Gate = the P0 proposition:** one real dependency remediation, performed in the sandbox, independently verified, human-approved — architecture §23's final checkbox.

### Phase 5 — Evaluation and hardening (1.5 weeks)

| Work item | Layer | Est. |
|---|---|---|
| Grow the Agentic benchmark category to 8–10 scenarios incl. the doc's Tests 1–8 (navigation, modification, debugging, policy denial, recovery, injection) | L11 | 4 d |
| Wire `themis-bench gate` as the **Model Ratchet**: a model/prompt change must pass the agentic suite vs baseline | L11 | 1 d |
| Failure-to-regression flow (manual): a failed real task becomes a new benchmark scenario | L11 | 1 d |
| Observability polish: structured JSON logs, OTel spans for task/step/tool | L10 | 2 d |

### Increment 2 — Hosted DeepSeek, policy-gated (2 weeks, parallel-safe after Phase 3)

| Work item | Notes |
|---|---|
| Credential broker: API keys held by the harness, never in model context; short-lived injection into the OpenAI-compatible client | doc §8 |
| Egress policy: per-task provider clearance; hosted provider requires an explicit flag + recorded operator decision | honors themis's no-silent-egress EDR |
| DeepSeek API endpoint in the registry; agentic suite run against it for the routing comparison | reuses Phase-1 client |

### Deferred (post-P0, in likely order)

Subagents (Planner/Engineer/Analyst/Reviewer — the loop's task model is kept recursive so this is additive) → more skills (`analyze-sbom`, `investigate-build-failure`, `security-code-review`) → Knowledge/Skill/Instruction Ratchet → semantic retrieval (pgvector only if lexical ranking proves insufficient — themis precedent says in-memory Go cosine won before) → minimal web UI → Redis queue (only when concurrent task volume demands it) → remote execution providers (Daytona/E2B).

---

## 3. Dependency graph

```
Phase 0 (model spike) ─── blocks everything (go/no-go on local-only)
   │
Phase 1 (chat+tools llm) ─── blocks Phase 2 loop
   │
Phase 2: Postgres state ──┬── loop needs state
         Docker sandbox ──┤── tools need sandbox
         tool gateway  ───┘
   │
Phase 3 (Slice 1) ─── needs Themis inbox endpoint  ◀── ONLY external dependency:
   │                                                   a small change in the themis repo
Phase 4 (Slice 2) ─── needs scanner choice (Grype/osv-scanner)
   │
Phase 5 (eval) ─── needs Slices 1–2 as scenario templates

Increment 2 ─── needs only Phase 1 + tool gateway; independent of Phases 4–5
```

Decision dependencies (resolve in Phase 0): L9 skill format blocks Phase 3; L4 `run_command` policy blocks Phase 4; both are cheap to decide now and expensive to reverse later.

External coordination: the **Themis inbox endpoint** for evidence return is the only cross-repo dependency in P0. The themis repo already has the outbound-write seam (`readapi.ProposalWriter`, landed Δ4b); the harness needs the receiving contract agreed — schedule that conversation during Phase 2 so it doesn't block Phase 3.

---

## 4. Dependencies — hardware

### Development machine (now)

Your current Mac is sufficient for all coding and scripted-model testing (no model needed). For *live* local-model work during development:

| Need | Requirement |
|---|---|
| Apple Silicon unified memory | 32 GB minimum runs 14–16B q4 models; **48–64 GB recommended** for 32B q4 (the likely sweet spot for tool-call reliability) |
| Disk | ~50 GB for a working set of 3–4 models |
| Software | Ollama (Metal acceleration is native; Docker Desktop for the sandbox — note containers on macOS have **no GPU access**, which is fine: the model runs on the host via Ollama, only the *agent workspace* is containerized) |

### P0 target box (needed by Phase 3–4 for realistic throughput; per architecture §19)

| Resource | Target | Concrete example |
|---|---|---|
| GPU | 24 GB+ VRAM | RTX 4090 24 GB (one) — runs 32B q4 fully on-GPU; 2× for headroom/concurrency |
| CPU | 16–32 physical cores | Ryzen 9 7950X / Threadripper |
| RAM | 64–128 GB | 128 GB preferred (parallel sandboxes + Postgres + model KV overflow) |
| Storage | 1–2 TB NVMe system + 2–4 TB data | model files, task workspaces, Postgres |
| OS | Ubuntu 24.04 LTS | matches §17 |
| Network | 1 GbE | LAN access from dev machine + Themis |

Rough cost if buying: a single-4090 workstation of this spec lands around €4–6k; a Mac Studio (M-series, 128 GB unified) is an alternative that trades raw CUDA throughput for simplicity. **Decision can wait until Phase 3** — Phases 0–2 run entirely on the dev Mac.

### Model candidates (validated, not assumed — that's Phase 0)

| Model | VRAM (q4) | Why |
|---|---|---|
| `deepseek-coder-v2:16b-lite` | ~10 GB | doc's provider preference; strong coding |
| DeepSeek-R1-Distill-Qwen-14B / 32B | ~9 / ~20 GB | reasoning strength; **tool-calling reliability unproven — must pass the spike** |
| `qwen2.5-coder:32b` (control) | ~20 GB | best-in-class local tool calling; keeps the DeepSeek choice honest |
| DeepSeek API (`deepseek-chat`/`-reasoner`) | — | Increment 2 only, policy-gated |

## 5. Dependencies — software

| Dependency | Version | Used from | New? |
|---|---|---|---|
| Go | 1.24+ | everywhere | already |
| Ollama | current | Phase 0 | already in use |
| Docker Engine / Desktop | current | Phase 2 | **new** |
| PostgreSQL | 16+ | Phase 2 | **new** (runs fine in Docker for dev) |
| ripgrep | any recent | Phase 2 (`search_code`) | **new** (binary dep of the sandbox image) |
| Grype or osv-scanner | current | Phase 4 | **new**; themis already speaks Grype formats — prefer it for reuse |
| Go libs: `pgx` (Postgres), OTel SDK | — | Phases 2, 5 | **new** — keep the doc's "minimal dependencies" spirit; no framework, `net/http` stdlib for the API |
| Redis | — | deferred | **not in P0** |
| pgvector | — | deferred | only if lexical ranking fails |

## 6. Effort summary

| Phase | Effort | Cumulative |
|---|---|---|
| 0 — Model spike + decisions | 1.0 wk | 1.0 |
| 1 — Model layer (chat + tools) | 1.5 wk | 2.5 |
| 2 — Slice-1 core | 3.5 wk | 6.0 |
| 3 — Slice 1 complete | 2.0 wk | 8.0 |
| 4 — Slice 2 (P0 proposition ✓) | 2.5 wk | 10.5 |
| 5 — Eval + hardening | 1.5 wk | **12.0 wk** |
| Increment 2 — hosted egress | +2.0 wk | 14.0 wk |

**≈ 12 engineer-weeks to the P0 Definition of Done** (10.5 to the headline demo), one senior engineer with AI assistance, full-time. Part-time scales linearly. The deferred list (subagents, Ratchet build-out, UI) is roughly another 8–12 weeks and should be re-planned after P0 ships — the eval data from Phase 5 will change its priorities.

**Estimate confidence:** Phases 0–1 high (small, well-understood code). Phase 2 medium (sandbox lifecycle is where unknowns hide). Phases 3–4 medium-low — they depend on how much prompt/skill iteration the chosen model needs, which is exactly what Phase 0 de-risks. The classic failure mode here is not code complexity but *model iteration burn*: budget the gates strictly and let a failing gate trigger a model change, not weeks of prompt tuning.

## 7. Top risks

| Risk | Mitigation |
|---|---|
| Local models can't do reliable native tool calling | Phase-0 spike with deterministic scoring; control model in the lineup; hosted increment can move forward |
| Sandbox lifecycle complexity (cleanup, resource caps, macOS/Linux drift) | Read-only mount in Slice 1 first; worktree writes only in Slice 2; CI runs Linux from day one |
| Scope creep toward the full 11-layer build-out | The two-slice gates are the scope contract; anything not needed by the current slice goes to the deferred list |
| Cross-repo drift on the evidence-return contract | Agree the inbox schema with the themis repo during Phase 2 |
| Model iteration burn (endless prompt tuning) | Every prompt/skill change must move the Phase-5 agentic benchmark score — no untracked tuning |
