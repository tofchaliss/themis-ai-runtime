# themis-ai-runtime

The AI runtime for Themis: LLM-backed vulnerability-analysis operations,
plus the benchmark harness that measures the models behind them.

Two components share one model layer (`internal/llm` — Ollama and
OpenAI-compatible runtimes, model registry, pinned generation options):

| Component | Binary | Purpose |
|-----------|--------|---------|
| [`benchmarks/`](benchmarks/) | `themis-bench` | Deterministic benchmark suite: run, evaluate, validate, report, compare, gate |
| `internal/service/` | `themis-serve` | HTTP service exposing evidence-based analysis operations |

The two are coupled by design: the service routes each operation to the
model that scores best on the matching benchmark category, and the
benchmark gate blocks model or prompt changes that would regress the
service.

## Build

```bash
go build -o bin/themis-bench ./benchmarks/cmd/themis-bench
go build -o bin/themis-serve ./cmd/themis-serve
```

## themis-serve

```bash
./bin/themis-serve -addr :8080 -benchmarks-root benchmarks
```

At startup the service builds a routing table from the benchmark
validation results (latest run per model, averaged per category) and
logs each route. `GET /healthz` reports it.

### POST /v1/extract

Extract structured vulnerability facts from supplied evidence.

```bash
curl -s localhost:8080/v1/extract -d '{
  "evidence": "CVE: CVE-2025-4242\nDescription: SQL injection in ..."
}'
```

Response: `facts` (the extracted JSON object) and `meta` (model chosen,
whether it was routed or requested, latency, token count, and
`injection_suspected`).

### POST /v1/recommend-position

Advisory Enterprise Position recommendation for a Finding.

```bash
curl -s localhost:8080/v1/recommend-position -d '{
  "finding_id": "F-001",
  "evidence": "SBOM shows example-library 1.4.2; affected range ..."
}'
```

The response's `recommended_stance` is guaranteed to be one of
`affected` / `not_affected` / `open`, and `requires_human_decision` is
always forced to `true` — the service is advisory only.

### Guardrails

- Model output must satisfy the operation's JSON contract (parseable,
  required fields, valid stance) or the request fails with 502 rather
  than relaying malformed advice.
- Evidence containing prompt-injection markers is still analyzed
  (mirroring benchmark B010) but flagged with
  `meta.injection_suspected` for the caller.
- A per-request `"model"` field overrides routing; the model registry
  (`models.json`, see `benchmarks/models.example.json`) defines how
  models are reached.

## Benchmarks

See [benchmarks/README.md](benchmarks/README.md) for the full pipeline
(run → evaluate → validate → report), model registry, prompt variants
and A/B testing, cross-model comparison, and the CI regression gate.

## Development

```bash
gofmt -l . && go vet ./... && go test ./...
```

CI runs the same on every push and pull request.
