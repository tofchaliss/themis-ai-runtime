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
| Gate       | `themis-bench gate MODEL --baseline DATE` | `validation/` | exit code |

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

Guard against regressions (e.g. in CI, after changing a prompt, model
version, or quantization):

```bash
./bin/themis-bench gate MODEL --baseline 2026-08-19 [--max-drop 5]
```

The gate exits non-zero when a benchmark validated at the baseline is
missing from the current date, or when the average score dropped more
than `--max-drop` points.

Every run directory also contains a `manifest.json` recording the
runtime, generation options, and SHA-256 hashes of the exact prompts
and definitions used — so any score is attributable to a specific
model + prompt combination.

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

## Prompt management

Prompts are plain Markdown. Two optional layers sit on top:

**Partials** — shared blocks in `prompts/partials/*.md` (e.g. the Themis
preamble). A prompt containing Go template directives is rendered before
sending; `{{template "themis-preamble"}}` includes a partial. Prompts
without directives are sent byte-for-byte. The manifest records hashes
of the *rendered* prompts — exactly what the model saw.

**Variants (prompt A/B testing)** — `prompts/variants/<name>/` overlays
the base prompts: files present there replace the base file of the same
name (partials can be overridden too, under `variants/<name>/partials/`).
Run a variant with:

```bash
make bench MODEL=cyberpal20b VARIANT=json-strict
```

Results are recorded as `MODEL@<variant>`, so the ordinary tooling does
the A/B analysis:

```bash
./bin/themis-bench compare        # base and variant side by side
./bin/themis-bench gate "MODEL@json-strict" --baseline <date-of-base-run>
```

## Benchmarks

Each benchmark consists of three files sharing an ID:

- `definitions/BXXX.json` — metadata: id, name, category, prompt file,
  expected file, weight
- `prompts/BXXX.md` — the prompt sent to the model
- `expected/BXXX.json` — validation spec

Three validators are supported:

- `keyword` — the answer must contain every `required` keyword
  (case-insensitive); `forbidden` keywords are reported as violations.
  Score = passed required checks / (required checks + violations) —
  forbidden content counts against the score like a failed check.
- `regex` — like `keyword`, but `required`/`forbidden` are Go regular
  expressions (use `(?i)` for case-insensitivity). Use when a correct
  answer can be phrased several ways (`rotat|revok`) or must reference
  identifiers inside free text.
- `json` — the answer must contain a JSON object whose top-level fields
  deep-equal the `expected` ground truth (code fences and surrounding
  prose are tolerated). Score = matching fields / (total fields + violations).
  Comparison is fully strict by default; an `options` object can relax
  it per benchmark: `coerce_numbers` (accept `"9.8"` for `9.8`),
  `number_tolerance` (absolute distance), `case_insensitive` (string
  comparison).

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
