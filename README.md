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
                    │   src/harness/internal/llm   │
                    │  Ollama + OpenAI-compatible  │
                    │  runtimes, model registry,   │
                    │  pinned generation options   │
                    └──────────┬─────────┬─────────┘
                               │         │
              ┌────────────────┴──┐   ┌──┴────────────────────┐
              │   themis-bench    │   │     model router      │
              │ benchmark harness │   │ verdict-gated select. │
              │  (measures)       │   │ (D-L11-8 Class 2)     │
              └────────┬──────────┘   └──────────▲────────────┘
                       │ validation scores       │
                       └───── routing table ─────┘
```

The **harness** measures models on security tasks with fully
deterministic scoring. The **router** deterministically selects among
admitted models using verdict-gated benchmark results. Governed
execution runs through the L7 orchestrator (L1–L11); the legacy HTTP
service was decommissioned 2026-09-13 (integration-audit R1). The
harness's regression gate blocks model, prompt, or quantization
changes that would degrade selection.

## Components

| Component | Binary | Purpose |
|-----------|--------|---------|
| [`src/harness/benchmarks/`](src/harness/benchmarks/) | `themis-bench` | 20 security benchmarks: run, evaluate, validate, report, compare, gate |
| `src/harness/internal/service/` | — | model router (benchmark-verdict-gated selection); legacy HTTP service decommissioned 2026-09-13 |
| `src/harness/internal/llm/` | — | Shared model layer used by both |

## Quick start

```bash
# Build the benchmark harness
go build -o bin/themis-bench ./src/harness/benchmarks/cmd/themis-bench

# Benchmark a local Ollama model (full pipeline, one command)
cd src/harness/benchmarks && make bench MODEL=<ollama-model-name>
```

See [INSTALLATION.md](INSTALLATION.md) for prerequisites and
configuration.

## themis-serve — DECOMMISSIONED (2026-09-13)

The legacy HTTP service (`/v1/extract`, `/v1/recommend-position`) was
removed by the L1–L11 integration-audit R1 disposition: it invoked
models entirely outside the governed L1–L11 chain (finding F-1,
CLOSED BY REMOVAL). Evidence recorded at decommission: no repository
consumer, no deployment manifest, no running instance — only a stale
git-ignored local build.

What remains: the **model router** (`src/harness/internal/service/
router.go`) — benchmark-verdict-gated model selection, the
constitutionally blessed Class-2 evidence consumer (D-L11-8). The
governed execution path is the L7 orchestrator
(`src/harness/orchestration`); production wiring awaits the G1
Deployment Anchor implementation. History (and any future rebase
onto the governed seam) lives in git.

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

Full documentation: [src/harness/benchmarks/README.md](src/harness/benchmarks/README.md).

## Repository layout

```
docs/architecture/       Themis Agent Harness architecture (P0 baseline, layer docs)
src/harness/             the harness Go module (go.mod lives here)
  cmd/themis-serve/      runtime service binary
  internal/llm/          shared model layer (runtimes, registry, options)
  internal/service/      service: handlers, routing, guardrails, prompts
  benchmarks/
    cmd/themis-bench/    benchmark CLI
    internal/            pipeline packages (benchmark, evaluator,
                         validator, report, gate)
    definitions/         benchmark definitions (id, category)
    prompts/             prompts, shared partials, A/B variants
    expected/            validation specs (keyword | regex | json)
    runs/ responses/     generated pipeline artifacts (gitignored)
    validation/ reports/
go.work                  workspace root (future modules join here, e.g. src/themis)
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
