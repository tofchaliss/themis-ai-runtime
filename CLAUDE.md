# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Harness engineering rules

Before implementing or changing any harness code, read and follow `.claude/skills/themis-harness-engineering/SKILL.md` (identify the owning layer, the affected Themis authority boundary, and the smallest change first). The reviewer subagents in `.claude/agents/` (architecture-reviewer, security-reviewer, test-reviewer) review harness changes after implementation.

Non-negotiable, always:
- Themis is the security application and authority; the Harness is an internal AI runtime; DeepSeek is a replaceable model provider behind the model abstraction.
- Never allow model output to become authoritative security truth.
- Never bypass deterministic authorization because an agent or model requests an action.
- Prefer the smallest tested change.

## Commands

```bash
# Build both binaries (from repo root)
go build -o bin/themis-bench ./src/harness/benchmarks/cmd/themis-bench
go build -o bin/themis-serve ./src/harness/cmd/themis-serve

# Test / lint (one Go module at src/harness — the root go.work makes these work from the repo root)
go test ./src/harness/...                      # all tests
go test ./src/harness/internal/service/ -run TestRouter    # single test
go vet ./src/harness/...
gofmt -l .                                     # CI fails on any output

# From src/harness/benchmarks/: fmt + vet + test in one shot
make check

# Full benchmark pipeline against a model (needs Ollama running)
cd src/harness/benchmarks && make bench MODEL=<model> [ENDPOINT=<url>] [DATE=YYYY-MM-DD] [VARIANT=<name>]

# Regression gate after model/prompt/quantization changes
./bin/themis-bench gate <model> --baseline YYYY-MM-DD [--max-drop 5]

# Run the service
./bin/themis-serve -addr :8080 -benchmarks-root src/harness/benchmarks
```

No live model is needed to build or test: all unit tests use `httptest` mock backends and `t.TempDir()` fixtures.

## Architecture

This repo is being grown into the **Themis Agent Harness** — an 11-layer AI execution control plane. The architecture baseline and layer docs live in `docs/architecture/` (start with `harness-p0-architecture-v2.md`; decisions of 2026-09-03 supersede its §17 Python stack: the harness is Go, grown in this repo, local-default models with policy-gated hosted DeepSeek, thin-vertical-slice P0). The main themis application is expected to join the workspace later under `src/` (see the root `go.work`).

The existing runtime components below predate the harness architecture and become its foundations (`internal/llm` → model adapter, `benchmarks/` → Layer 11 evaluation/regression).

Two components share one model layer and are coupled by design:

- **`src/harness/internal/llm/`** — shared model layer: `Runtime` interface with Ollama and OpenAI-compatible implementations, model registry (`models.json`, gitignored; resolves unknown names to local Ollama at `localhost:11434`), and pinned deterministic generation options (`temperature 0, seed 42`).
- **`src/harness/benchmarks/`** (`themis-bench` CLI, cobra) — 20 security benchmarks with fully deterministic scoring (no LLM-as-judge).
- **`src/harness/internal/service/`** (`themis-serve`) — HTTP service (`/healthz`, `/v1/extract`, `/v1/recommend-position`). Prompts are embedded Go templates in `src/harness/internal/service/prompts/`.

The coupling: the service's `Router` reads the harness's `src/harness/benchmarks/validation/` results and routes each request category (`Extraction`, `Reasoning` — matching benchmark definitions' `category` field) to the model with the highest average score in its latest validated run. Variant results (`model@variant`) are excluded from routing. The harness's `gate` command protects the service from quality regressions.

### Benchmark pipeline

Stages are separate commands, each reading the previous stage's output from date/model-scoped directories:

```
run → runs/DATE/MODEL/ → evaluate → responses/ → validate → validation/ → report → reports/
```

Pipeline packages live in `src/harness/benchmarks/internal/` (`benchmark`, `evaluator`, `validator`, `report`, `gate`). Cross-stage contracts (file formats, directory layout, manifest exclusion) are covered by the end-to-end test in `src/harness/benchmarks/internal/benchmark/e2e_test.go` — the test to check when changing anything about the on-disk pipeline format.

Each benchmark = three files sharing an ID: `definitions/BXXX.json` (metadata, category, weight), `prompts/BXXX.md`, `expected/BXXX.json` (validator spec: `keyword` | `regex` | `json`). Adding a benchmark needs no Go test — the loader validates specs at run time.

Every run writes a `manifest.json` with the runtime, options, and SHA-256 of the *rendered* prompts (prompts may use Go template directives to include `prompts/partials/*.md`; `prompts/variants/<name>/` overlays base prompts for A/B testing, recorded as `MODEL@<variant>`).

### Service guardrails (`src/harness/internal/service/guardrails.go`)

Deterministic, not model-based: prompt-injection markers in evidence set `meta.injection_suspected` but the evidence is still analyzed (mirrors benchmark B010); model output must satisfy the operation's JSON contract or the request fails with 502; `recommended_stance` must be one of `affected`/`not_affected`/`open`; `requires_human_decision` is always forced `true` — the service is advisory only.

## Conventions

- Table-driven `t.Run` tests; helpers take `t.Helper()`. HTTP tests use `httptest` servers; filesystem tests build fixtures in `t.TempDir()` — never the repo's own data directories.
- API keys never go in `models.json` — entries name an env var via `api_key_env`.
- A new validator needs: happy path, scoring edge cases, malformed-spec error tests, a dispatch case in `validator.go`, and a line in the READMEs.
- The harness fails fast on model errors rather than saving error payloads as runs.
- Docs to keep in sync: `README.md` (overview), `INSTALLATION.md`, `TESTING.md`, `src/harness/benchmarks/README.md`, and `docs/architecture/` (harness architecture baseline — amend, don't silently override).
