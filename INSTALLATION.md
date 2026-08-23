# Installation

## Prerequisites

| Requirement | Version | Notes |
|-------------|---------|-------|
| Go | 1.24+ | the only build dependency |
| [Ollama](https://ollama.com) | any recent | needed to *run* benchmarks or serve local models; not needed to build or test |
| An OpenAI-compatible endpoint | optional | vLLM, llama.cpp server, LM Studio, OpenRouter, or hosted OpenAI |
| `make` | optional | convenience targets in `benchmarks/` |

Everything builds with the standard Go toolchain; the only third-party
Go dependency is `spf13/cobra` (CLI).

## Build

```bash
git clone https://github.com/tofchaliss/themis-ai-runtime.git
cd themis-ai-runtime

go build -o bin/themis-bench ./benchmarks/cmd/themis-bench
go build -o bin/themis-serve ./cmd/themis-serve
```

Or, for the benchmark CLI only, from `benchmarks/`:

```bash
cd benchmarks && make build     # -> benchmarks/bin/themis-bench
```

Verify:

```bash
./bin/themis-bench --help
./bin/themis-serve -h
```

## Model setup

### Local models (Ollama)

```bash
ollama pull <model>             # e.g. a local security-tuned model
ollama serve                    # if not already running
```

With no configuration, **every model name resolves to local Ollama at
`http://localhost:11434`** (override with `$OLLAMA_HOST` or
`--endpoint`). Nothing more is needed for the local workflow.

### Model registry (other runtimes, aliases, per-model options)

To reach OpenAI-compatible endpoints or customize per-model behavior,
copy the example registry and edit it:

```bash
cp benchmarks/models.example.json benchmarks/models.json   # for themis-bench
cp benchmarks/models.example.json models.json              # for themis-serve (config-root ".")
```

`models.json` is gitignored (it is machine-specific; API keys never go
in the file — only the *name* of the environment variable holding one):

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
  },
  "defaults": { "runtime": "ollama", "endpoint": "http://localhost:11434" }
}
```

Entry fields:

| Field | Meaning |
|-------|---------|
| `runtime` | `ollama` (default) or `openai` |
| `endpoint` | API base URL |
| `model` | identifier sent to the runtime, when it differs from the registry name (alias) |
| `api_key_env` | environment variable holding the API key |
| `temperature`, `seed` | override the deterministic defaults (`0` / `42`) |

## Running the benchmark suite

From `benchmarks/`:

```bash
# Full pipeline in one command: run -> evaluate -> validate -> report
make bench MODEL=<model-name>

# Optional overrides
make bench MODEL=<model> ENDPOINT=http://remote:11434   # remote Ollama
make bench MODEL=<model> DATE=2026-08-19                # operate on a past date
make bench MODEL=<model> VARIANT=json-strict            # prompt A/B run
```

Stage-by-stage and analysis commands:

```bash
./bin/themis-bench run <model>        # execute benchmarks, write runs/
./bin/themis-bench evaluate <model>   # normalize -> responses/
./bin/themis-bench validate <model>   # score -> validation/
./bin/themis-bench report <model>     # Markdown report -> reports/
./bin/themis-bench compare            # cross-model matrix + history
./bin/themis-bench gate <model> --baseline YYYY-MM-DD [--max-drop 5]
```

Common flags: `--root DIR` (suite root, default `.`), `--date
YYYY-MM-DD` (default today); `run` also takes `--endpoint`, `--timeout`
(default 10m), and `--variant`.

## Running the runtime service

From the repo root:

```bash
./bin/themis-serve -addr :8080 -benchmarks-root benchmarks
```

| Flag | Default | Meaning |
|------|---------|---------|
| `-addr` | `:8080` | listen address |
| `-benchmarks-root` | `benchmarks` | suite root whose definitions + validation results drive routing |
| `-config-root` | `.` | directory containing `models.json` |
| `-default-model` | — | fallback when routing has no data for a category |
| `-timeout` | `10m` | per-request model invocation timeout |

On startup the service logs its routing table (category → chosen
model). Sanity-check with:

```bash
curl -s localhost:8080/healthz
```

The service needs at least one completed `validate` run under
`benchmarks/validation/` to build routes; otherwise pass
`-default-model` or a per-request `"model"` field.

## Troubleshooting

- **`connection refused` on run** — Ollama is not running or the
  endpoint is wrong; check `curl $OLLAMA_HOST/api/tags` or pass
  `--endpoint`.
- **`ollama error: model 'x' not found`** — pull it (`ollama pull x`)
  or fix the registry alias. The harness fails fast rather than saving
  error payloads as runs.
- **`environment variable X is not set`** — the registry entry names an
  `api_key_env` that is missing from your environment.
- **`no runs found` on evaluate** — the run stage was skipped for that
  date/model, or you need `--date` for a past run.
- **Gate: `no validation results`** — `validate` has not been run for
  the baseline or current date.
- **Service: `no model available for category`** — no validation data
  and no `-default-model`; run the benchmark pipeline once or pass a
  model per request.
