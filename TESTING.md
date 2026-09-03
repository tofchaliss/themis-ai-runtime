# Testing

Every layer of the project is testable without a real model: unit tests
use `httptest` mock backends and temporary directories, and the full
benchmark pipeline has an end-to-end test that never leaves the test
process. A live model is only needed for the (optional) real-model
verification workflow at the end.

## Running the tests

From the repo root:

```bash
go test ./src/harness/...          # everything
go test -cover ./src/harness/...   # with coverage
go vet ./src/harness/...           # static analysis
gofmt -l .                         # formatting (CI fails on any output)
```

From `src/harness/benchmarks/`:

```bash
make check               # fmt + vet + test
```

## What is covered where

| Package | Coverage* | What the tests exercise |
|---------|----------:|-------------------------|
| `src/harness/internal/llm` | ~78% | Ollama and OpenAI clients against `httptest` servers: request shape (pinned `temperature 0, seed 42`), status/error/empty-choice handling, context cancellation; registry resolution (defaults, aliases, API-key env, option overrides, malformed files) |
| `src/harness/internal/service` | ~84% | Router (best-per-category, latest-run-wins, variant exclusion); guardrails (injection patterns, stance contract, required fields); full HTTP handler tests against a mock backend: happy paths, injection flagging, 502 on malformed model output, 400 on bad requests, forced `requires_human_decision` |
| `src/harness/benchmarks/internal/benchmark` | ~81% | Definition loading (missing id/prompt, duplicates, malformed JSON); prompt templating (partials, variant overlay, error cases); **end-to-end pipeline test** (see below) |
| `src/harness/benchmarks/internal/evaluator` | ~28%** | Run-envelope parsing, legacy raw-Ollama fallback, error payloads, incomplete runs |
| `src/harness/benchmarks/internal/validator` | ~53%** | All three validators: keyword (case-insensitivity, forbidden, scoring), regex (alternation, invalid patterns), json (code-fence tolerance, strict defaults, coerce/tolerance/case options, per-field credit) |
| `src/harness/benchmarks/internal/report` | ~89% | Aggregation (validated-only averaging), Markdown rendering (`-` for unvalidated), run discovery with slashed model names, cross-model comparison, slash-safe report writing |
| `src/harness/benchmarks/internal/gate` | ~85% | Pass/fail semantics: improvements, tolerated drops, missing-benchmark always fails, new benchmarks informational, report rendering |

\* Approximate, from `go test -cover`.
\*\* The uncovered portion of these packages is their `*All` pipeline
drivers, which the end-to-end test exercises through the benchmark
package instead.

## The end-to-end pipeline test

`src/harness/benchmarks/internal/benchmark/e2e_test.go` drives the entire pipeline
— run → evaluate → validate → report — in a temporary suite root
against an `httptest` mock Ollama server. It asserts:

- the run stage writes runtime-agnostic envelopes and a manifest
  recording options and rendered-prompt hashes,
- the evaluate stage skips the manifest and normalizes metrics,
- the validate stage scores against the expected spec,
- the report contains the right aggregate numbers.

This is the test that catches cross-stage contract breaks (file
formats, directory layout, manifest exclusion) that unit tests miss.

## Continuous integration

`.github/workflows/ci.yml` runs on every push to `main` and every pull
request: gofmt (fails on any unformatted file), `go vet`, `go test
./...`, and release builds of both binaries.

## The regression gate (testing *models and prompts*, not code)

Code tests protect the harness; the **gate** protects the quality of
what it measures. After any model, prompt, or quantization change:

```bash
cd src/harness/benchmarks
make bench MODEL=<model>                       # produce today's scores
./bin/themis-bench gate <model> --baseline <last-good-date> --max-drop 5
```

Exit code is non-zero when a benchmark that existed at the baseline is
missing, or the average score dropped more than `--max-drop` points.
Prompt changes are gated the same way via variants: run `make bench
MODEL=<model> VARIANT=<name>` and gate/compare `MODEL@<name>` against
the base series.

## Live verification (optional, needs a real model)

With Ollama running and a model pulled:

```bash
# 1. Benchmark it end to end
cd src/harness/benchmarks && make bench MODEL=<model>

# 2. Serve and exercise a routed request
cd .. && ./bin/themis-serve -addr :8080 -benchmarks-root src/harness/benchmarks &
curl -s localhost:8080/healthz                  # routing table present?
curl -s localhost:8080/v1/extract -d '{"evidence": "CVE: CVE-2025-1 ..."}'
```

Expected: `/healthz` lists per-category routes derived from the run you
just made, and `/v1/extract` returns `facts` plus `meta.routed: true`.

## Writing new tests

- Follow the existing table-driven `t.Run` style; helpers take
  `t.Helper()`.
- Anything that talks HTTP gets an `httptest` server — never a live
  endpoint in unit tests.
- Anything that touches the filesystem builds its fixture tree in
  `t.TempDir()` — never the repo's own data directories.
- A new benchmark needs no Go test: add `definitions/BXXX.json`,
  `prompts/BXXX.md`, `expected/BXXX.json`, and the loader validates it
  at run time (the e2e test covers the machinery). Malformed specs fail
  fast with the filename in the error.
- A new validator gets: happy path, scoring edge cases, malformed-spec
  error, and a case in `validator.go`'s dispatch — plus a line in the
  READMEs.
