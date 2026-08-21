# Themis Benchmark Suite

Deterministic benchmark suite for evaluating LLMs on security-related
tasks (CVE recall, CVSS interpretation, SBOM/VEX understanding, fact
extraction, hallucination resistance, prompt-injection resistance, …).

Each benchmark is a prompt plus a machine-checkable expectation; scoring
is fully deterministic — no LLM-as-judge. Generation is pinned to
`temperature 0, seed 42` by default for reproducibility.

Models can be served by [Ollama](https://ollama.com) (the default) or
any OpenAI-compatible endpoint (vLLM, llama.cpp server, LM Studio,
OpenRouter, hosted OpenAI).

## Requirements

- Go 1.24+
- A running Ollama instance (default `http://localhost:11434`)

## Build

```bash
make build
```

## Pipeline

The suite runs in four stages. Each stage reads the previous stage's
output from a date/model-scoped directory:

| Stage      | Command                       | Reads                     | Writes                        |
|------------|-------------------------------|---------------------------|-------------------------------|
| Run        | `themis-bench run MODEL`      | `definitions/`, `prompts/`| `runs/DATE/MODEL/`            |
| Evaluate   | `themis-bench evaluate MODEL` | `runs/DATE/MODEL/`        | `responses/DATE/MODEL/`       |
| Validate   | `themis-bench validate MODEL` | `responses/`, `expected/` | `validation/DATE/MODEL/`      |
| Report     | `themis-bench report MODEL`   | `responses/`, `validation/`| `reports/DATE/MODEL.md`      |
| Compare    | `themis-bench compare`        | all of `responses/`, `validation/` | `reports/comparison.md` |

Run the whole pipeline with one command:

```bash
make bench MODEL=cyberpal20b
# optional overrides:
make bench MODEL=cyberpal20b ENDPOINT=http://remote:11434 DATE=2026-08-19
```

Or run stages individually:

```bash
./bin/themis-bench run cyberpal20b
./bin/themis-bench evaluate cyberpal20b
./bin/themis-bench validate cyberpal20b
./bin/themis-bench report cyberpal20b
```

Compare every model you have benchmarked, across dates:

```bash
./bin/themis-bench compare
```

This scans all evaluated runs and writes `reports/comparison.md` with a
per-model summary, a per-benchmark score matrix (latest run of each
model), and score history for models benchmarked on multiple dates.

### Model registry

Without configuration, every model name resolves to local Ollama. To
benchmark models on other runtimes, copy `models.example.json` to
`models.json` (gitignored) and register them:

```json
{
  "models": {
    "gpt-4o": {
      "runtime": "openai",
      "endpoint": "https://api.openai.com/v1",
      "api_key_env": "OPENAI_API_KEY"
    },
    "qwen-vllm": {
      "runtime": "openai",
      "endpoint": "http://localhost:8000/v1",
      "model": "Qwen/Qwen3-30B-A3B-Instruct-2507"
    }
  }
}
```

Entry fields: `runtime` (`ollama` | `openai`), `endpoint`, `model`
(alias → real identifier), `api_key_env` (env var holding the key —
keys never go in the file), and optional `temperature`/`seed`
overrides of the deterministic defaults.

Run files record which runtime and generation options produced them.

### Flags

- `--root DIR` — benchmark suite root (default `.`; all commands)
- `--date YYYY-MM-DD` — operate on a past run (default today; all commands)
- `--endpoint URL` — Ollama endpoint (`run` only; default `$OLLAMA_HOST`
  or `http://localhost:11434`)
- `--timeout DURATION` — per-benchmark generation timeout (`run` only;
  default `10m`)

## Benchmarks

Each benchmark consists of three files sharing an ID:

- `definitions/BXXX.json` — metadata: id, name, category, prompt file,
  expected file, weight
- `prompts/BXXX.md` — the prompt sent to the model
- `expected/BXXX.json` — validation spec

Two validators are supported:

- `keyword` — the answer must contain every `required` keyword
  (case-insensitive); `forbidden` keywords are reported as violations.
  Score = passed required checks / total required checks.
- `json` — the answer must contain a JSON object whose top-level fields
  deep-equal the `expected` ground truth (code fences and surrounding
  prose are tolerated). Score = matching fields / total fields.

## Development

```bash
make check   # fmt + vet + test
```

## Layout

```
cmd/themis-bench/   CLI entry point
internal/benchmark/ definition loading and run execution
internal/runtime/   model runtimes (Ollama)
internal/evaluator/ raw-run normalization and metrics
internal/validator/ answer validation (keyword, json)
internal/report/    Markdown report generation
definitions/        benchmark definitions
prompts/            benchmark prompts
expected/           validation specs
runs/               raw model output (generated, gitignored)
responses/          normalized responses (generated)
validation/         validation results (generated)
reports/            Markdown reports (generated)
docs/               architecture documentation
```
