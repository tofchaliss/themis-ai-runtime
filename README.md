# themis-ai-runtime

The AI execution capability of Themis: a **governed harness** that runs
model-backed work under deterministic control, plus the deterministic
**benchmark suite** that measures the models behind it.

Themis is the security system of record. The harness executes; it does
not own security truth. Model output is advisory at every crossing.

- **[INSTALLATION.md](INSTALLATION.md)** — prerequisites, build, model
  registry, what actually installs
- **[TESTING.md](TESTING.md)** — the suite, live proofs, what each layer
  proves, the regression gate
- **[ARCHITECTURE.md](ARCHITECTURE.md)** — authority and ownership (binding)
- **[docs/architecture/harness/execution-chain.md](docs/architecture/harness/execution-chain.md)**
  — the as-built chain, end to end

Status: L1–L7 and L9–L11 shipped and archived; **L8 (Subagents) is
reserved and scaffolded, not implemented** — it requires its own
architecture grill before any code
(`openspec/changes/l8-subagents/proposal.md`). G1 (Deployment Authority
Anchoring) and G2 (Established-Fact Boundary) closed and implemented;
the legacy ungoverned HTTP surface removed. **The architecture is
frozen.**

First concrete deployment instance validated 2026-09-14 (`rsys`,
Phases A–F; `docs/development/deployment-signoff-rsys.md`). That record
establishes that *that* instance was governed by the intended artifact
set under real host conditions — it does not grant production wiring,
which remains runbook Step 12, an open owner decision.

## The governed chain

```
GOVERNANCE (humans) ──registers──► anchors · catalogs · contracts · criteria
        │
        ▼
  DEPLOYMENT OPEN ──admits the Deployment Anchor, verifies the
        │            instruction plane, freezes deployment authority
        ▼
  TASK ASSEMBLY ──── the submitter chooses a task WITHIN the
        │            deployment, never the deployment
        ▼
  L1 instructions → L2/L3 context → L4 tools → L5 execution
        → L6 durable state → L7 orchestration → L9 skills
        → L10 verification → L11 ratchet
        │
        ▼
GOVERNANCE (humans) ── evidence arrives; only a door makes a change real
```

Two cross-layer boundaries hold the chain together:

- **G1 — what governs an execution?** A caller-supplied anchor
  path/hash *identifies* a requested deployment; only resolution
  against the Governance-ACTIVE anchors registry *admits* it. The two
  steps are never collapsed.
- **G2 — what is a fact?** Storage proves bytes; **events** prove
  establishment. An L6 object is a fact of kind F only if a committed
  event of F's minting class names it.

## Components

| Component | Form | Purpose |
|---|---|---|
| [`src/harness/`](src/harness/) | Go library (`github.com/tofchaliss/themis`) | the governed chain L1–L11 + G1; entered via `orchestration.Open` / `SubmitTask` |
| `src/harness/cmd/themis-ratchet/` | `themis-ratchet` | L11 comparative-evidence CLI: `compare`, `set`, `reconstruct`, `derive`, `candidate`, `plan` |
| [`src/harness/benchmarks/`](src/harness/benchmarks/) | `themis-bench` | 20 security benchmarks: run, evaluate, validate, report, compare, gate |
| `src/harness/internal/service/` | package | model router — benchmark-verdict-gated selection among admitted models (D-L11-8 Class 2) |
| `src/harness/internal/llm/` | package | shared model layer (runtimes, registry, pinned options) |

There is **no production entry binary** and no service to start.
Wiring a production caller to `Open`/`SubmitTask` is a separate owner
decision.

## Quick start

```bash
# Build the two CLIs
go build -o bin/themis-ratchet ./src/harness/cmd/themis-ratchet
go build -o bin/themis-bench   ./src/harness/benchmarks/cmd/themis-bench

# Verify the tree
cd src/harness && go build ./... && go vet ./... && gofmt -l . && go test ./... -count=1

# Benchmark a local Ollama model (full pipeline, one command)
cd benchmarks && make bench MODEL=<ollama-model-name>
```

Read [INSTALLATION.md](INSTALLATION.md) before running the suite on a
machine with Ollama up — the live proofs are endpoint-gated, not
opt-in.

## Deploying

Installing is not deploying. A governed deployment needs a
Governance-ACTIVE Deployment Anchor pinning every artifact it executes
under, plus a deployment-supplied execution ceiling.

- [`docs/operations/deployment-runbook.md`](docs/operations/deployment-runbook.md)
  — 14 steps: provision → pin → propose → prove inert → Governance act
  → verify `Open` → validate → evidence
- [`docs/development/deployment-test-plan.md`](docs/development/deployment-test-plan.md)
  — Phases A–F plus a concrete real-VM scenario (VM-0…VM-9)
- [`policies/deployment/README.md`](policies/deployment/README.md)
  — the execution-ceiling contract

**No anchor is ACTIVE in this repository, by design.** An anchor's
ceiling carries host-specific paths, so it is created where the
deployment lives — never committed here with placeholders. Absent an
admitted anchor, `Open` refuses unless the caller explicitly declares
the `Unanchored` test-harness role, and those records carry a sentinel.

## themis-serve — DECOMMISSIONED (2026-09-13)

The legacy HTTP service (`/v1/extract`, `/v1/recommend-position`) was
removed by the L1–L11 integration-audit R1 disposition: it invoked
models entirely outside the governed chain (finding F-1, CLOSED BY
REMOVAL). Evidence recorded at decommission: no repository consumer, no
deployment manifest, no running instance — only a stale git-ignored
local build.

What remains is the **model router**
(`src/harness/internal/service/router.go`): benchmark-verdict-gated
model selection, the constitutionally blessed Class-2 evidence consumer
(D-L11-8). It selects *within* an admitted set; it never extends one.
History lives in git.

## themis-bench — the benchmark suite

Twenty benchmarks across security categories (CVE recall, CVSS
interpretation and vector decoding, SBOM/VEX understanding, fact
extraction, hallucination resistance, prompt-injection resistance, CWE
classification, patch analysis, secrets detection, IaC review,
enterprise-position reasoning, …). Scoring is fully deterministic —
keyword, regex, and JSON-ground-truth validators; **no LLM-as-judge**.
Generation is pinned to `temperature 0, seed 42`.

Pipeline: `run → evaluate → validate → report`, plus `compare`
(cross-model matrix and score history) and `gate` (CI regression
check). Prompt variants enable A/B testing with the same tooling. Every
run writes a manifest recording the runtime, options, and SHA-256 of
the exact rendered prompts, making every score attributable and
reproducible.

Full documentation:
[src/harness/benchmarks/README.md](src/harness/benchmarks/README.md).

## Repository layout

```
ARCHITECTURE.md          authority, ownership, prohibitions (binding)
AGENTS.md                repository conventions and review invariants
CONTEXT-MAP.md           multi-context layout
src/harness/             the Go module (go.mod lives here)
  cmd/themis-ratchet/    L11 evidence CLI
  instructions/          L1 instruction plane
  context/ confine/      L2/L3 context delivery and management
  tools/                 L4 tool interface (deterministic authorization)
  execution/             L5 sealed execution environment
  state/                 L6 durable record plane
  orchestration/         L7 orchestration (Open / SubmitTask / the walk)
  skills/                L9 sealed skill composition
  verification/          L10 verification + the L7 seam
  ratchet/               L11 comparative evidence
  deployment/            G1 Deployment Anchor
  runtime/model/         governed model interface
  internal/llm/          shared model layer
  internal/service/      model router
  integration/           Phase C end-to-end register
  benchmarks/            benchmark suite (CLI, pipeline, definitions)
instructions/            shipped instruction roots (pinned by anchor)
policies/                governed registries and policies
  deployment/            anchors.json + the ceiling contract
openspec/changes/archive/ per-layer locked decisions (why, not what)
docs/architecture/       architecture documents and the as-built chain
docs/operations/         deployment runbook
docs/development/        deployment test plan
go.work                  workspace root
.github/workflows/       CI (fmt, vet, test, build both CLIs)
```

## Reading order for a newcomer

1. [`ARCHITECTURE.md`](ARCHITECTURE.md) — ownership, authority, prohibitions
2. [`docs/architecture/harness/execution-chain.md`](docs/architecture/harness/execution-chain.md) — how the pieces connect as built
3. [`docs/harness-layer-status.md`](docs/harness-layer-status.md) — what shipped, when, with residuals
4. `openspec/changes/archive/<layer>/design.md` — why each layer is shaped the way it is
5. `openspec/changes/archive/2026-09-13-g1-deployment-authority/design.md` and `g2-established-fact-boundary/design.md` — the two cross-layer boundaries

## License

See [LICENSE](LICENSE).
