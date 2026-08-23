# themis-ai-runtime

The AI runtime for Themis: LLM-backed vulnerability-analysis operations,
plus the deterministic benchmark harness that measures the models behind
them.

- **[INSTALLATION.md](INSTALLATION.md)** — prerequisites, build, model
  registry, running both binaries
- **[TESTING.md](TESTING.md)** — unit tests, end-to-end tests, live
  verification, CI, the regression gate

## Why one repo

Two components share one model layer and are coupled by design:

```
                    ┌──────────────────────────────┐
                    │        internal/llm          │
                    │  Ollama + OpenAI-compatible  │
                    │  runtimes, model registry,   │
                    │  pinned generation options   │
                    └──────────┬─────────┬─────────┘
                               │         │
              ┌────────────────┴──┐   ┌──┴────────────────────┐
              │   themis-bench    │   │     themis-serve      │
              │ benchmark harness │   │   runtime service     │
              │  (measures)       │   │   (serves)            │
              └────────┬──────────┘   └──────────▲────────────┘
                       │ validation scores       │
                       └───── routing table ─────┘
```

The **harness** measures models on security tasks with fully
deterministic scoring. The **service** exposes analysis operations over
HTTP and routes each request to the model that scores best on the
matching benchmark category. The harness's regression gate blocks
model, prompt, or quantization changes that would degrade the service.

## Components

| Component | Binary | Purpose |
|-----------|--------|---------|
| [`benchmarks/`](benchmarks/) | `themis-bench` | 20 security benchmarks: run, evaluate, validate, report, compare, gate |
| `internal/service/` | `themis-serve` | HTTP service: evidence-based extraction and recommendation |
| `internal/llm/` | — | Shared model layer used by both |

## Quick start

```bash
# Build both binaries
go build -o bin/themis-bench ./benchmarks/cmd/themis-bench
go build -o bin/themis-serve ./cmd/themis-serve

# Benchmark a local Ollama model (full pipeline, one command)
cd benchmarks && make bench MODEL=<ollama-model-name>

# Start the runtime service (routing driven by benchmark results)
./bin/themis-serve -addr :8080 -benchmarks-root benchmarks
```

See [INSTALLATION.md](INSTALLATION.md) for prerequisites and
configuration.

## themis-serve — the runtime service

### Endpoints

**`GET /healthz`** — liveness plus the current routing table
(category → per-model average scores).

**`POST /v1/extract`** — extract structured vulnerability facts from
supplied evidence.

```bash
curl -s localhost:8080/v1/extract -d '{
  "evidence": "CVE: CVE-2025-4242\nDescription: SQL injection in AcmeCorp WebPortal ...\nCWE: CWE-89\nAffected versions: 2.0.0 through 2.3.1\nFixed version: 2.3.2"
}'
```

Response: `facts` (the extracted JSON object: `cve`, `cwe`,
`description`, `affected_component`, `affected_versions`,
`fixed_versions`, `cvss`, `exploitability`, `references`,
`unknown_fields`) and `meta` (model chosen, `routed` true/false,
runtime, latency, tokens, `injection_suspected`).

**`POST /v1/recommend-position`** — advisory Enterprise Position
recommendation for a Finding.

```bash
curl -s localhost:8080/v1/recommend-position -d '{
  "finding_id": "F-001",
  "evidence": "SBOM shows example-library 1.4.2; vulnerability knowledge states 1.0.0 through 1.4.2 affected ..."
}'
```

The response's `recommended_stance` is guaranteed to be one of
`affected` / `not_affected` / `open`, and `requires_human_decision` is
always forced to `true` — the service is advisory only.

Both endpoints accept an optional `"model"` field that overrides
routing for that request.

### Benchmark-driven routing

At startup the service reads the benchmark suite's validation results
(latest run per model, averaged per benchmark category) and routes each
operation to the best-scoring model: extraction requests go to the
model that best proved it can extract, recommendations to the model
that best proved it can reason. Prompt-variant result series
(`model@variant`) are excluded. The table is logged at startup and
served on `/healthz`.

### Guardrails

- **Output contracts** — model output must be parseable JSON with the
  operation's required fields and a valid stance, or the request fails
  with `502` instead of relaying malformed advice.
- **Advisory only** — `requires_human_decision` is forced `true`
  regardless of what the model returns.
- **Injection flagging** — evidence containing prompt-injection markers
  ("ignore all previous instructions", …) is still analyzed (mirroring
  benchmark B010) but flagged via `meta.injection_suspected`.

## themis-bench — the benchmark harness

Twenty benchmarks across security categories (CVE recall, CVSS
interpretation and vector decoding, SBOM/VEX understanding, fact
extraction, hallucination resistance, prompt-injection resistance,
CWE classification, patch analysis, secrets detection, IaC review,
enterprise-position reasoning, …). Scoring is fully deterministic —
keyword, regex, and JSON-ground-truth validators; no LLM-as-judge.
Generation is pinned to `temperature 0, seed 42`.

Pipeline: `run → evaluate → validate → report`, plus `compare`
(cross-model matrix and score history) and `gate` (CI regression
check). Prompt variants enable A/B testing of prompt changes with the
same tooling. Every run writes a manifest recording the runtime,
options, and SHA-256 of the exact rendered prompts, making every score
attributable and reproducible.

Full documentation: [benchmarks/README.md](benchmarks/README.md).

## Repository layout

```
cmd/themis-serve/        runtime service binary
internal/llm/            shared model layer (runtimes, registry, options)
internal/service/        service: handlers, routing, guardrails, prompts
benchmarks/
  cmd/themis-bench/      benchmark CLI
  internal/              pipeline packages (benchmark, evaluator,
                         validator, report, gate)
  definitions/           benchmark definitions (id, category, weight)
  prompts/               prompts, shared partials, A/B variants
  expected/              validation specs (keyword | regex | json)
  runs/ responses/       generated pipeline artifacts (gitignored)
  validation/ reports/
.github/workflows/       CI (fmt, vet, test, build both binaries)
```

## Status and roadmap

Implemented: multi-model runtime layer (Ollama + OpenAI-compatible),
model registry, prompt templating/partials/variants, run manifests,
regression gate, cross-model comparison, CI, the runtime service with
benchmark-driven routing and guardrails.

Candidate next steps: streaming responses; restricting routing to
registry-listed models; precedent retrieval for recommendations;
semantic and LLM-judge validator tiers; HTML reports and historical
trend dashboards.

## License

See [LICENSE](LICENSE).
