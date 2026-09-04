# Harness Layer Baseline — Mapping Existing Code to the 11 Layers

**Date:** 2026-09-04 · **Steps 1–2 of the build sequence** (baseline the repository; map existing code to the layers). Verdict at the end: does the architecture fit the existing code?

## 1. Repository baseline

| Component | LOC (of which tests) | What it is |
|---|---|---|
| `src/harness/internal/llm` | 970 (399) | Model layer: `Runtime` interface (single prompt in / answer out), Ollama `/api/generate` + OpenAI-compatible clients, registry (`models.json`, `api_key_env`, alias, per-model overrides), pinned deterministic options (temp 0, seed 42), tolerant `ExtractJSON` |
| `src/harness/internal/service` | 980 (387) | `themis-serve`: two operations (`/v1/extract`, `/v1/recommend-position`), embedded prompt templates, benchmark-driven `Router` (best model per category), deterministic guardrails (`SuspectInjection`, stance contract, required fields, forced `requires_human_decision`) |
| `src/harness/benchmarks` | 3,638 (1,243) | `themis-bench`: 20 benchmarks, pipeline run→evaluate→validate→report + compare + gate, deterministic validators (keyword/regex/json), prompt partials + variants, per-run manifests with SHA-256 of rendered prompts |
| Total | 5,713 Go LOC, 7 packages, all tests green | No live model needed to build or test |

## 2. Layer mapping

| Harness layer | Existing implementation | Gap | Action |
| --- | --- | --- | --- |
| Instructions (L1) | Fragments: embedded per-operation prompt templates (`internal/service/prompts/`); benchmark prompts with partials and **rendered-prompt SHA-256 hashing in manifests** — the versioning/hashing pattern L1 needs, already proven | No instruction sources, hierarchy, resolver, precedence, conflict detection, or runtime effective-set versioning | Build the L1 resolver (P0 M1 scope); reuse the manifest hashing pattern |
| Context Delivery (L2) | Minimal: evidence arrives as a request field; benchmarks read local files | No Themis API connector; no git/SBOM/scan/repo connectors | Define the Themis read interface + minimal connectors (Finding, repo files) for Slice 1 |
| Context Management (L3) | **None** — evidence is templated verbatim; no retrieval, ranking, or token budgeting | Full | Minimal lexical retrieval (ripgrep) + token-budgeted assembly for Slice 1 |
| Tool Interface (L4) | **None** — the model gets one prompt and returns text; it has no tools | Full: registry, schemas, policy, audit | Core of the deterministic foundation (sequence step 4) |
| Execution Environment (L5) | **None** — service runs in-process, nothing is sandboxed | Full | Docker + read-only mount (Slice 1); worktree writes (Slice 2) |
| Durable State (L6) | Partial analog: benchmark pipeline persists date/model-scoped artifacts on disk — durable, but batch artifacts, not task state; the service is stateless | Task/step/checkpoint state, crash resumability | Postgres task store (P0) |
| Orchestration (L7) | Fragments: fixed render→invoke→validate flow in service handlers; Makefile-sequenced pipeline; **model routing exists** (benchmark-driven Router) | Agent loop, budgets, approval gates, task API, lifecycle | Core P0 build; Router becomes the model-routing seam |
| Subagents (L8) | **None** | Full — deliberately deferred | Keep the task model recursive; post-P0 |
| Skills (L9) | Closest analog: the two service operations — fixed procedure + embedded prompt + JSON output contract each | No skill structure (inputs, required context, allowed tools, checkpoints, approval) | OPEN-1 design discussion, then `investigate-cve` (Slice 1) |
| Verification/Observability (L10) | **Strongest layer**: deterministic validators, regression gate, injection flagging, output contracts, forced `requires_human_decision`, per-request meta (model/runtime/latency/tokens/routed), manifests with prompt hashes | Independent build/test/scan verification of agent work; per-step traces; grounding checks; structured logs/OTel | Extend meta → full trace; verification tools in Slice 2 |
| Ratchet (L11) | **Substantial for the evaluation half**: `compare` (cross-model matrix + score history), `gate` (baseline regression, max-drop), prompt variants for A/B | Agentic benchmark category; failure→regression flow; knowledge/skill/instruction ratchets (deferred) | Phase 0: agentic category on themis-bench; wire `gate` as the model ratchet |

**Cross-cutting (the Model Interface/Adapter, below the layers):** exists and tested (Runtime interface, two providers, registry) but speaks single-prompt only — no conversations, no native tool calls. Phase 1 evolves it to a chat + tool-call interface. This is the one existing component that must *change shape* rather than merely grow.

## 3. Verdict: does the architecture fit the existing code?

**Yes — no conflicts, and the fit is telling.** The existing code concentrates at the two ends of the architecture: the model adapter below the layers, and L10/L11 (verification, evaluation, regression) at the top — plus proven patterns the middle will reuse (prompt hashing → L1 versioning; deterministic validators → L10 contracts; the Router → L7 model routing). The completely missing middle — L4 tools, L5 sandbox, L6 task state, L7 orchestration — is exactly the deterministic foundation the build sequence constructs first. Nothing existing fights the architecture; nothing needs demolition; one component (the model layer) needs evolution rather than addition.

The service's two operations are effectively **proto-skills running without a harness** — single-shot, tool-less, but already advisory-only, contract-validated, and injection-flagged. The harness build wraps machinery around an approach the repo already practices.

## 4. The build sequence (owner-defined, 2026-09-04)

1. ✅ Baseline the actual repository (this document, §1)
2. ✅ Map existing code to the 11 layers (§2)
3. Establish the Harness P0 skeleton — **open structural question:** the skeleton sketch places the harness at `src/themis/ai/` (inside Themis), while code today lives at `src/harness/` and the Themis application has not yet joined the repo. Owner decision needed before moving code (Class 4: structural ownership).
4. Implement the deterministic foundation first: Instructions → Context/Policy → Tool Interface → Authorization → Execute → Verify → Themis workflow
5. Bring DeepSeek in through the abstraction (Harness → Model Interface → Model Adapter → DeepSeek)
6. Add Claude Code enforcement — hooks and review agents. **Status: already implemented and live** (four guards registered in `.claude/settings.json`, 31/31 behavior tests; review agents installed). Remaining: keep them aligned as the code grows.
7. First end-to-end vertical slice: Task → Instruction resolution → Context retrieval → Model reasoning → Tool request → Deterministic authorization → Tool execution → Verification → Auditable result
