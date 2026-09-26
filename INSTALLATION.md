# Installation

Status: current as of 2026-09-14, against the frozen architecture
(L1–L11 + G1 + G2).

**Installing is not deploying.** This document gets the code built and
the tools running on a machine. Standing up a *governed* deployment —
an admitted Deployment Anchor, a deployment execution ceiling, the
Governance act — is a separate procedure in
[`docs/operations/deployment-runbook.md`](docs/operations/deployment-runbook.md).
Nothing here grants any execution authority.

## What actually installs

| Surface | Form | Notes |
|---|---|---|
| Governed harness (L1–L11) | **Go library**, module `github.com/tofchaliss/themis-ai-runtime/src/harness` | entered through `orchestration.Open` / `SubmitTask`; no production entry binary ships |
| `themis-ratchet` | binary, `src/harness/cmd/themis-ratchet` | L11 comparative-evidence CLI (`compare`, `set`, `reconstruct`, `derive`, `candidate`, `plan`) |
| `themis-bench` | binary, `src/harness/benchmarks/cmd/themis-bench` | deterministic model benchmark suite |
| model router | package `src/harness/internal/service` | verdict-gated selection among admitted models (D-L11-8 Class 2) |

`themis-serve` **no longer exists.** The legacy HTTP surface
(`/v1/extract`, `/v1/recommend-position`) was decommissioned on
2026-09-13 by integration-audit R1 — it invoked models outside the
governed chain. Any script or runbook that builds or starts it is
stale.

## Prerequisites

| Requirement | Version | Needed for |
|---|---|---|
| Go | **1.24+** (`src/harness/go.mod` declares `go 1.24`) | everything; the only build dependency |
| `git` | any recent | required at **runtime**: L5 provisions workspaces from a git mirror, and the harness pins the **absolute path** of the git binary |
| [Ollama](https://ollama.com) | any recent | only to *run* benchmarks or live proofs; not needed to build or test |
| An OpenAI-compatible endpoint | optional | vLLM, llama.cpp server, LM Studio, OpenRouter, hosted OpenAI |
| `make` | optional | convenience targets in `src/harness/benchmarks/` |

The only third-party Go dependency is `spf13/cobra` (the `themis-bench`
CLI). Everything else is the standard library.

Record `which git` and `go version` on any host that will run a
deployment — both are inputs the runbook asks for.

## Linux server install — verified sequence

Performed end to end on 2026-09-14: Ubuntu 6.8.0-38, x86_64, 24 vCPU,
62 GB, no GPU, corporate account with a quota-limited autofs home.
Every step below is one that ran; the notes record what actually went
wrong, because three of these failed silently rather than loudly.

### 1. Go toolchain

```bash
curl -fsSL https://go.dev/dl/go1.24.0.linux-amd64.tar.gz -o /tmp/go.tgz
sudo rm -rf /usr/local/go && sudo tar -C /usr/local -xzf /tmp/go.tgz
echo 'export PATH=$PATH:/usr/local/go/bin' >> ~/.profile
export PATH=$PATH:/usr/local/go/bin
go version        # expect go1.24.0 linux/amd64
```

Persist `PATH` in `~/.profile`. An `export` alone dies with the shell,
and the next login fails the preflight again.

### 2. Relocate the Go caches if `/home` is quota-limited

**Do this before the first build, not after it fails.**

```bash
df -hT /home /opt /srv
go env GOCACHE GOMODCACHE
```

If `/home` is autofs, networked, or quota'd while the bulk storage is
local, move both caches to the local filesystem:

```bash
sudo mkdir -p /srv/themis/build && sudo chown "$(id -u):$(id -g)" /srv/themis/build
mkdir -p /srv/themis/build/{gocache,gomodcache}
grep -q GOCACHE ~/.profile || {
  echo 'export GOCACHE=/srv/themis/build/gocache'
  echo 'export GOMODCACHE=/srv/themis/build/gomodcache'
} >> ~/.profile
export GOCACHE=/srv/themis/build/gocache
export GOMODCACHE=/srv/themis/build/gomodcache
go env GOCACHE GOMODCACHE        # confirm BOTH point at the new location
tail -3 ~/.profile               # confirm the append actually wrote
```

Two traps here, both hit on the real host:

- **An exhausted quota does not fail the build cleanly.** Packages that
  cannot build simply do not run, their tests never report, and the
  shell still exits 0. A corrupted suite run looked entirely plausible
  and had silently executed five fewer packages than the run before it.
- **A full quota also breaks `~/.profile`.** The append fails with
  `write error: Disk quota exceeded` while the surrounding commands
  succeed, so the variables never persist. Clear space first
  (`rm -rf ~/.cache/go-build`), then confirm with `tail`.

### 3. Clone WITHOUT sudo

```bash
sudo mkdir -p /opt/themis && sudo chown "$(id -u):$(id -g)" /opt/themis
git clone https://github.com/tofchaliss/themis-ai-runtime.git /opt/themis/themis-ai-runtime
cd /opt/themis/themis-ai-runtime
```

Cloning as root leaves root-owned files and git then refuses every
operation with `detected dubious ownership`. If that has already
happened, fix the ownership rather than adding git's `safe.directory`
exception — the exception silences git while leaving files you cannot
write, which resurfaces as a build failure instead:

```bash
sudo chown -R "$(id -u):$(id -g)" /opt/themis/themis-ai-runtime
```

Use numeric IDs. `chown -R $USER:$USER` assumes a group matching your
username, which is false on most corporate accounts (`invalid group`).

### 4. Model endpoint (optional, but needed for live evidence)

```bash
curl -fsSL https://ollama.com/install.sh | sh
ollama pull qwen2.5:7b                                                  # THEMIS_LIVE_TOOL_MODEL
ollama pull WhiteRabbitNeo/WHiteRabbitNeo-2.5-Qwen-2.5-Coder-7B:latest  # THEMIS_LIVE_MODEL
ollama list
```

Tags must match **exactly** — note the irregular `WHiteRabbitNeo`
capitalisation, which is the real registry tag. The live proofs are
endpoint-gated, so a reachable endpoint with the wrong tags makes them
fail rather than skip.

Verify tool calling, which is a capability gate rather than a
preference and is not visible in `ollama list`:

```bash
curl -s localhost:11434/api/chat -d '{
  "model":"qwen2.5:7b","stream":false,
  "messages":[{"role":"user","content":"read the file x"}],
  "tools":[{"type":"function","function":{"name":"read_file",
    "parameters":{"type":"object","properties":{"path":{"type":"string"}}}}}]}'
```

A `tool_calls` field means capable; HTTP 400 means not.
`THEMIS_LIVE_TOOL_MODEL` must be tool-capable; `THEMIS_LIVE_MODEL` is
conversation-only and need not be.

### 5. Preflight

```bash
scripts/themis-preflight --deploy /srv/themis/<name>
```

Resolve every FAIL. Warnings do not block but change what a green run
means — record them. On a GPU-less server expect one warning about
CPU-only inference; that is correct, not a fault.

### 6. Build and verify

```bash
cd src/harness
go build ./... && go vet ./... && gofmt -l .
THEMIS_LIVE_OLLAMA=http://127.0.0.1:9 go test ./... -count=1
```

The dead endpoint makes the nine live proofs skip, giving a purely
deterministic run where any failure is real. About **8 seconds** on 24
vCPU.

Audit what actually ran, rather than trusting the "ok" lines:

```bash
THEMIS_LIVE_OLLAMA=http://127.0.0.1:9 go test ./... -count=1 -v 2>&1 \
  | grep -E '^\s*--- SKIP' | sort
```

Expect exactly 10 lines: nine live proofs plus
`TestTeardownAnomalous/immutable-flag` (a darwin-only fixture,
correctly skipped on Linux). Fewer means packages did not run.

### 7. Live proofs — one at a time

```bash
go test ./context/ -run 'TestLiveOperationalProof|TestLivePressureProof' -count=1 -v
go test ./verification/seam/ -run TestLiveRemediateWalk -count=1 -v
go test ./ratchet/ -run TestLiveModelAuthorsCandidate -count=1 -v
go test ./tools/ -run TestLiveToolProof -count=1 -v
go test ./execution/ -run TestLiveExecutionProof -count=1 -v
go test ./state/ -run TestLiveTaskReconstruction -count=1 -v
go test ./orchestration/ -run 'TestLiveWalkProof|TestLiveSkillWalk' -count=1 -v
```

**Never run the full suite with live proofs enabled.** Nine of them
live in nine packages that `go test` runs in parallel, all driving one
model server; they queue and time out. This is concurrency, not
capacity — on the 62 GB / 24-core host six failed in the sweep and
every one passed alone minutes earlier, which is worse than on a 16 GB
laptop because more of them run at once.

Reference timings on this host, CPU-only: 45 s, 23 s, **19 s for a full
governed walk**, 4 s. That walk figure is the empirical input for
`turn_timeout_sec` and the ceiling's `max_wall_deadline_sec`.

### 8. Governance state

```bash
scripts/themis-status
```

Expect **ACTIVE: none** — correct, not a fault. No anchor ships in the
repository; see *Running governed execution* below.

## Build

```bash
git clone https://github.com/tofchaliss/themis-ai-runtime.git
cd themis-ai-runtime

# go.work puts the module on the workspace, so repo-root paths work
go build -o bin/themis-ratchet ./src/harness/cmd/themis-ratchet
go build -o bin/themis-bench   ./src/harness/benchmarks/cmd/themis-bench
```

The Go module root is `src/harness/`, not the repository root. Working
inside the module (the form CI and the runbook use):

```bash
cd src/harness
go build ./...          # the whole module, library packages included
```

For the benchmark CLI only:

```bash
cd src/harness/benchmarks && make build   # -> src/harness/benchmarks/bin/themis-bench
```

Verify:

```bash
./bin/themis-bench --help     # cobra usage: run, evaluate, validate, report, compare, gate
./bin/themis-ratchet          # usage line naming the six subcommands; exit 1 by design
```

`themis-ratchet` with no arguments prints usage and exits 1 — that is
the acceptance boundary, not an install problem. Every real invocation
requires exact pins; nothing is defaulted or resolved to "latest".

`bin/` is gitignored.

## Model setup

Model configuration is only needed to *run* models — benchmarks, live
proofs, or a deployment whose anchor admits a model registry. Building
and running the deterministic test suite needs none of it.

### Local models (Ollama)

```bash
ollama pull qwen2.5:7b          # the tool-calling model the live proofs default to
ollama serve                    # if not already running
```

With no registry, **every model name resolves to local Ollama at
`http://localhost:11434`** (override with `$OLLAMA_HOST`, or
`--endpoint` for `themis-bench run`).

### Model registry (`models.json`)

To reach OpenAI-compatible endpoints, alias model names, or override
per-model options, copy the example registry and edit it:

```bash
cp src/harness/benchmarks/models.example.json src/harness/benchmarks/models.json
```

`models.json` is gitignored under `benchmarks/` (it is machine-specific).
API keys never go in the file — only the *name* of the environment
variable holding one:

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

| Field | Meaning |
|---|---|
| `runtime` | `ollama` (default) or `openai` |
| `endpoint` | API base URL |
| `model` | identifier sent to the runtime when it differs from the registry name (alias) |
| `api_key_env` | environment variable holding the API key |
| `temperature`, `seed` | override the deterministic defaults (`0` / `42`) |

The same loader serves both surfaces: `internal/llm.LoadRegistry`,
re-exported as `runtime/model.LoadRegistry` for the governed path.

**In a governed deployment the registry is not free configuration.**
The Deployment Anchor pins the exact `models.json` bytes, or declares
`"absent"`; changing the file after the pin is a Governance act, and
`Open` refuses a registry that is not the anchored one. See
[`policies/deployment/README.md`](policies/deployment/README.md).

## Running the benchmark suite

From `src/harness/benchmarks/`:

```bash
# Full pipeline in one command: run -> evaluate -> validate -> report
make bench MODEL=<model-name>

# Optional overrides
make bench MODEL=<model> ENDPOINT=http://remote:11434   # remote Ollama
make bench MODEL=<model> DATE=2026-08-19                # operate on a past date
make bench MODEL=<model> VARIANT=json-strict            # prompt A/B run
```

Stage by stage:

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

Validation results under `src/harness/benchmarks/validation/` are what
the model router consumes — no run, no routing table.

## Running governed execution

There is **no service to start.** Governed execution is entered
programmatically:

```
orchestration.Open(Config{...})   ->  admits the Deployment Anchor,
                                      verifies the instruction plane,
                                      freezes deployment authority
orchestration.SubmitTask(...)     ->  the submitter chooses a task
                                      WITHIN the deployment
```

`Open` **refuses to open without an admitted anchor** unless the caller
explicitly declares the `Unanchored` test-harness role — and those
records carry a sentinel so they can never be mistaken for governed
ones. There is no silent unanchored path.

**No anchor is ACTIVE in this repository, by design.** `local-dev@1` is
WITHDRAWN. An anchor is deployment-instance-specific: its execution
ceiling carries host paths, so it is created where the deployment
lives, never committed here with placeholders.

To stand one up, follow
[`docs/operations/deployment-runbook.md`](docs/operations/deployment-runbook.md)
(14 steps: provision → pin → propose → prove inert → Governance act →
verify `Open` → validate → evidence), then prove it with
[`docs/development/deployment-test-plan.md`](docs/development/deployment-test-plan.md).

Production wiring of a caller to `Open`/`SubmitTask` is a separate
owner decision and is not granted by installing anything.

## Where things live

```
src/harness/                  the Go module (go.mod)
  cmd/themis-ratchet/         L11 evidence CLI
  benchmarks/cmd/themis-bench/ benchmark CLI
  instructions/ context/      L1, L2+L3
  tools/ execution/ confine/  L4, L5
  state/                      L6 durable record plane
  orchestration/              L7 (Open / SubmitTask / the walk)
  skills/ verification/       L9, L10
  ratchet/                    L11
  deployment/                 G1 Deployment Anchor
  runtime/model/              governed model interface
  internal/llm/               shared model layer (runtimes, registry)
  internal/service/           model router
  integration/                Phase C end-to-end register
instructions/                 shipped instruction roots (pinned by anchor)
policies/                     governed registries and policies
  deployment/                 anchors.json + the ceiling contract
docs/operations/              deployment runbook
docs/development/             deployment test plan
docs/architecture/harness/    execution-chain.md — the as-built diagram
bin/                          build output (gitignored)
```

## Troubleshooting

**Build and tooling**

- **`no required module provides package`** — you are outside the
  workspace. Build from the repository root (`go.work`) or from
  `src/harness/`.
- **Go version errors** — `go.mod` requires 1.24; a 1.23 toolchain will
  not build the module.
- **`disk quota exceeded` under `~/.cache/go-build`** — relocate
  `GOCACHE`/`GOMODCACHE` (Linux install step 2). Critically, this does
  **not** fail cleanly: unbuildable packages simply do not run and the
  shell still exits 0, so a corrupted suite looks green. Re-run the
  skip audit after fixing; fewer than 10 skip lines means packages were
  silently missed.
- **`write error: Disk quota exceeded` from `echo >> ~/.profile`** — the
  same quota. The append fails while surrounding commands succeed, so
  environment variables never persist. Clear space, then `tail ~/.profile`
  to confirm the write.
- **`detected dubious ownership in repository`** — the repo was cloned
  as root. Fix ownership with numeric IDs
  (`sudo chown -R "$(id -u):$(id -g)" <repo>`) rather than adding git's
  `safe.directory` exception, which leaves files you cannot write.
- **`chown: invalid group`** — `$USER:$USER` assumes a group named after
  the user; untrue on most corporate/LDAP accounts. Use
  `"$(id -u):$(id -g)"`.

**Benchmarks and models**

- **`connection refused` on run** — Ollama is not running or the
  endpoint is wrong; check `curl $OLLAMA_HOST/api/tags` or pass
  `--endpoint`.
- **`ollama error: model 'x' not found`** — pull it (`ollama pull x`) or
  fix the registry alias. The harness fails fast rather than saving
  error payloads as runs.
- **`environment variable X is not set`** — the registry entry names an
  `api_key_env` missing from your environment.
- **`no runs found` on evaluate** — the run stage was skipped for that
  date/model, or you need `--date` for a past run.
- **Gate: `no validation results`** — `validate` has not been run for
  the baseline or the current date.
- **Router: `no model available for category`** — no validation data
  under `benchmarks/validation/`; run the pipeline once.

**Governed execution**

- **`no deployment anchor configured and Unanchored not explicitly set`**
  — correct behaviour, not a bug. Supply an admitted anchor, or declare
  the test-harness role deliberately.
- **Admission refuses a hand-written anchor** — correct: caller-supplied
  bytes *identify* a deployment; only `anchors.json` *admits* it
  (D-G1-1A).
- **`not the anchored artifact`** — a pinned instruction root, policy,
  tool registry, or registry file changed after pinning. Restore the
  bytes or perform a Governance act; never weaken the loader.
- **`non-regular entry in a pinned tree`** — a symlink or device node
  appeared inside a pinned instruction root.

**When a governed step refuses, classify before fixing** (owner rule):
1 deployment/configuration · 2 implementation defect · 3 recorded
residual · 4 architectural gap. Only category 4 touches a locked
decision. Never fix a refusal by weakening a loader, inferring a
default, or reinterpreting an artifact.

## Related documents

- [`TESTING.md`](TESTING.md) — how the tree is verified
- [`ARCHITECTURE.md`](ARCHITECTURE.md) — authority and ownership (binding)
- [`docs/architecture/harness/execution-chain.md`](docs/architecture/harness/execution-chain.md) — the as-built chain
- [`docs/operations/deployment-runbook.md`](docs/operations/deployment-runbook.md) — standing up a deployment
- [`docs/development/deployment-test-plan.md`](docs/development/deployment-test-plan.md) — proving one
