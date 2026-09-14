# Deployment Test Plan — first concrete Deployment Instance

Status: written 2026-09-13 against the frozen architecture (L1–L11 +
G1 + G2). This plan tests an INSTANTIATION, not the architecture. It
produces evidence for the production-wiring decision; it does not
grant it.

**Triage rule in force for anything this plan surfaces** (owner):
1. deployment/configuration issue → fix the deployment
2. implementation defect → fix under the frozen constitutions
3. existing documented residual → verify recorded, do not widen
4. genuine architectural gap → only this reopens a locked decision

Nothing here may be "fixed" by weakening a loader, inferring a
default, or reinterpreting an artifact.

---

## 0. Preconditions (owner-supplied)

| Input | Who supplies | Notes |
|---|---|---|
| Deployment instance identity (`name@version`) | owner | e.g. `prod-sec-a@1` — kebab-case name |
| `mirror_root` | owner | absolute host path to the git mirror this deployment may provision from |
| Execution limits | owner | wall deadline, file/total bytes, file count, memory, CPU, process count |
| Model allowlist | owner | exact model names permitted |
| Model registry | owner | the models.json bytes, or the explicit declaration that the deployment ships none |
| Steward | owner | accountability metadata on the registration |

Everything else (artifact pins, anchor bytes, hashes) is computed
from the repository and the supplied ceiling.

---

## Phase A — Build and static verification (no deployment yet)

| # | Step | Expected |
|---|---|---|
| A1 | `cd src/harness && go build ./...` | clean |
| A2 | `go vet ./...` | clean |
| A3 | `gofmt -l .` | no output |
| A4 | `go test ./... -count=1` | all green except the documented live-contention flakes (`context.TestLivePressureProof`, `context.TestLiveOperationalProof`) |
| A5 | `go test ./context/ -run 'TestLiveOperationalProof\|TestLivePressureProof' -count=1` | green in isolation — confirms contention, not a regression |
| A6 | `go test ./deployment/ ./orchestration/ ./integration/ -count=1` | green — anchor, seam, and Phase C |

**Live proofs are endpoint-gated, not opt-in.** They skip only when
nothing answers at `THEMIS_LIVE_OLLAMA` (default
`http://localhost:11434`). If a model endpoint is up on the build host,
they run, and they need their models present:
`THEMIS_LIVE_TOOL_MODEL` (default `qwen2.5:7b`) and
`THEMIS_LIVE_MODEL` (default
`WhiteRabbitNeo/WHiteRabbitNeo-2.5-Qwen-2.5-Coder-7B:latest`). A live
failure caused by a missing model or by memory pressure is category 1;
a live failure **in isolation** is not.

Gate: A1–A6 pass before any deployment artifact is created.

---

## Phase B — Deployment artifact creation (governance act, owner-driven)

| # | Step | Expected |
|---|---|---|
| B1 | Write the execution ceiling with the owner's exact values to a deployment-local path (NOT the repo) | file exists, valid JSON |
| B2 | Confirm it LOADS: it must satisfy `execution.LoadCeiling` (mirror_root present and real, all bounds positive) | loads |
| B3 | Compute `sha256` of the exact ceiling bytes | the `execution_ceiling` pin |
| B4 | Compute the artifact pins: instruction roots (dir hashes), instruction policy, tool registry, per-workflow bundles (workflow + workflow ceiling + context contract), skill catalog, L10 contract registry, L11 criteria + regression-set registries, model registry (or `"absent"`), L6 + L7 constitution hashes | all sha256 |
| B5 | Emit `policies/deployment/<name>.json` (anchor) and `<...>.proposed.json` (registration) | proposed only — NOT active |
| B6 | **Verify the proposal is INERT**: attempt `AdmitAnchor` against the governed registry before the act | REFUSED (unregistered / registry unavailable) |
| B7 | Owner Governance act: proposed → governed `anchors.json` | registration ACTIVE |
| B8 | Re-attempt admission | ADMITTED, two-way identity holds |

Gate: B6 must refuse and B8 must admit. If B6 admits, stop — the
proposal path is not inert and that is a category-2 defect.

---

## Phase C — Anchored negative space (the substitution attempts)

Each row is an attempt to govern execution with something the anchor
did not admit. **Every one must REFUSE**, and the refusal must name
the right reason — a refusal for the wrong reason is a finding.

| # | Attempt | Expected refusal |
|---|---|---|
| C1 | Anchor bytes ≠ operator-supplied hash | hash mismatch at Open |
| C2 | Self-consistent anchor that is not registered | "identifier, never an admission claim" |
| C3 | Withdrawn anchor | withdrawn — no new opens |
| C4 | Registration rebound to different bytes between Opens | "deployment identity is immutable" |
| C5 | Registration deleted between Opens | "disappeared — anchors are append-only" |
| C6 | Instruction root mutated after registration | "not the anchored artifact" |
| C7 | Symlink dropped into a pinned instruction root | "non-regular entry in a pinned tree" |
| C8 | Instruction policy swapped | "instruction policy is not the anchored artifact" |
| C9 | Rebuilt binary with a different constitution | "constitution is not the anchored one" |
| C10 | models.json rewritten after pinning | "only by Governance act" |
| C11 | Model registry configured while the anchor declares `absent` | "declares no model registry" |
| C12 | Envelope names a model outside the allowlist | "not in the anchored allowlist" |
| C13 | Envelope supplies its own execution ceiling | "supplied at Open, never chosen per task" |
| C14 | Workflow not in the anchored set | "not in the anchored workflow set" |
| C15 | Anchored workflow paired with another bundle's ceiling/contract | "not the artifact this anchored workflow bundles" |
| C16 | Tool registry swapped | "tool registry is not the anchored artifact" |
| C17 | Skill composition hash chosen by the submitter, differing from the catalog's | "a submitter selects an anchored skill, never its constituent hashes" |
| C18 | `Open` with no anchor and no explicit `Unanchored` | "a deployment governs by anchor or refuses to open" |
| C19 | `Unanchored` declared alongside an anchor | "the caller role is ambiguous" |

Automated coverage today: C1–C6, C8–C19 in
`orchestration/verification_seam_test.go` +
`deployment/anchor_test.go`; C7 in `deployment/anchor_test.go`. On a
real deployment they are re-run against the REAL anchor rather than a
test-minted one (Phase E).

---

## Phase D — Anchored positive path

| # | Step | Expected |
|---|---|---|
| D1 | `Open` with the real anchor, real registry, real ceiling | opens; anchor frozen |
| D2 | Submit a task whose bundle is fully anchored | assembly passes |
| D3 | Walk runs to a typed terminal | COMPLETED (or a declared terminal) |
| D4 | Task record carries `deployment_anchor = <anchor hash>` | present, not `"unanchored"` |
| D5 | Anchor BYTES are durable in the record | object retrievable |
| D6 | Read-path re-verification (`deployment.VerifyAnchorRecord`) | re-establishes the deployment from record + registry alone |
| D7 | Kill the process mid-walk; reopen | startup sweep drives the task to a typed terminal; no continuation |

---

## Phase E — Governed chain end to end (the Phase C register, on real artifacts)

Mirrors `integration/phasec_test.go`, run against the real deployment:

| # | Step | Expected |
|---|---|---|
| E1 | Two governed walks (baseline, candidate) under the real anchor | both COMPLETED |
| E2 | L10 gate opened on a real registered contract | PASS recorded, gate fired |
| E3 | L11 facts witnessed by the walks' OWN `l4-audit` events | grounding accepts |
| E4 | Admission observed at the real catalog door | observation grounded in door bytes |
| E5 | Comparison produces Δ; run identities DERIVED (`task:<id>`) | no caller-supplied run text |
| E6 | Cold reconstruction from package + criterion + door + record plane | CONFIRMED |
| E7 | Governance doors byte-identical after the whole chain | unchanged |
| E8 | A model-turn object from that history offered as a fact | REFUSED (G2) |

---

## Phase F — Sign-off evidence

Record, do not summarize away:
- Phase A–E results with dates and commit SHA.
- The anchor identity (`name@version` + hash) and the ceiling hash.
- Every refusal message observed in Phase C (the exact text).
- Any finding, with its triage category (1–4) and its disposition.
- Residuals confirmed still-recorded: submitter authentication;
  sensitivity inheritance (local-endpoint scope); the four
  consumption-pinned registries whose consumers live outside L7;
  `TestLivePressureProof` flake.

Production wiring proceeds only on the owner's decision after this
evidence exists. The plan produces evidence; it does not grant the
gate.

---

# Real-VM scenario — concrete steps

A clean VM (Ubuntu 22.04+ or equivalent), no prior Themis state.
Commands assume `$REPO` = the checkout and `$DEPLOY` = a directory
**outside** the repo holding deployment-specific configuration.

## VM-0 — Provision

```bash
# base tooling
sudo apt-get update && sudo apt-get install -y git curl build-essential
# Go (match go.mod; adjust version as needed)
curl -fsSL https://go.dev/dl/go1.24.0.linux-amd64.tar.gz -o /tmp/go.tgz
sudo rm -rf /usr/local/go && sudo tar -C /usr/local -xzf /tmp/go.tgz
export PATH=$PATH:/usr/local/go/bin && go version

# a local model endpoint, only if live proofs are in scope
curl -fsSL https://ollama.com/install.sh | sh
ollama pull qwen2.5:7b
```

Record: OS, kernel, Go version, git version and absolute path
(`which git` — the harness pins the git binary), model name/digest.

## VM-1 — Checkout and static gate (Phase A)

```bash
git clone <repo-url> "$REPO" && cd "$REPO/src/harness"
git rev-parse HEAD                       # record the SHA under test
go build ./... && go vet ./... && gofmt -l .
go test ./... -count=1                   # expect the documented flakes only (~10 min)
go test ./context/ -run 'TestLiveOperationalProof|TestLivePressureProof' -count=1
```

If a model endpoint is running on this host, the live proofs execute
rather than skip — pull `qwen2.5:7b` and whatever `THEMIS_LIVE_MODEL`
names, or the suite fails for a category-1 reason.

## VM-2 — Deployment directory and execution ceiling (Phase B1–B3)

```bash
mkdir -p "$DEPLOY"/{mirror,state,artifacts,provider}
# seed the mirror this deployment may provision from
git clone --mirror <governed-source-repo> "$DEPLOY/mirror/<name>.git"

cat > "$DEPLOY/execution-ceiling.json" <<JSON
{"version":1,
 "mirror_root":"$DEPLOY/mirror",
 "max_wall_deadline_sec":600,
 "max_file_bytes":1048576,
 "max_total_bytes":10485760,
 "max_file_count":500,
 "max_mem_bytes":1073741824,
 "max_cpu_time_sec":600,
 "max_proc_count":64}
JSON
sha256sum "$DEPLOY/execution-ceiling.json"   # → the execution_ceiling pin
```

The ceiling lives in `$DEPLOY`, never in the repo. Its bytes are the
deployment's identity contribution.

## VM-3 — Compute pins and emit the proposal (Phase B4–B5)

Pins are computed with the same functions the loader uses
(`deployment.HashDir` for instruction roots, `HashFile` for single
artifacts, and the L6/L7 `ConstitutionHash()` values). Produce:

- `$REPO/policies/deployment/<name>.json` — the anchor
- `$REPO/policies/deployment/anchors.proposed.json` — the registration

The anchor must contain every pin listed in Phase B4; the loader
refuses anything defaulted, unknown, or missing.

## VM-4 — Prove the proposal is inert, then activate (Phase B6–B8)

```bash
# BEFORE the act: admission must refuse
# (governed anchors.json absent or lacking the registration)
# AFTER the owner's act:
cp "$REPO/policies/deployment/anchors.proposed.json" \
   "$REPO/policies/deployment/anchors.json"
```

Re-run admission: it must now succeed, with two-way identity
(`name@version` in the registration equals the anchor's
self-declaration).

## VM-5 — Negative space against the REAL anchor (Phase C)

Run each C-row as a deliberate attempt. Suggested VM-specific ones,
since they exercise the host rather than a fixture:

```bash
# C7 — symlink into a pinned instruction root
ln -s /etc/hostname "$REPO/instructions/global/safety/evil.md"
#   → Open must refuse: "non-regular entry in a pinned tree"
rm "$REPO/instructions/global/safety/evil.md"

# C6 — mutate a pinned instruction file
echo " " >> "$REPO/instructions/global/system/<some>.md"
#   → Open must refuse: "not the anchored artifact"
git checkout -- "$REPO/instructions/global/system"

# C10 — rewrite models.json after pinning
#   → Open must refuse: "only by Governance act"

# C13 — task envelope naming a different exec ceiling
#   → SubmitTask must refuse: "supplied at Open, never chosen per task"
```

Capture the exact refusal text for each. A refusal with the wrong
reason is a finding, not a pass.

## VM-6 — Anchored positive path and kill/recovery (Phase D)

```bash
# open with the real anchor + registry + ceiling, submit an anchored
# task, let it run to a typed terminal, then:
kill -9 <pid>            # mid-walk
# reopen: the startup sweep must drive the task to a typed terminal
```

Verify in the record: `deployment_anchor` present and equal to the
anchor hash; the anchor bytes retrievable; `VerifyAnchorRecord`
re-establishes the deployment from record + registry alone.

## VM-7 — Full governed chain (Phase E)

Two walks, L10 gate, witnessed L11 facts, admission observed at the
real catalog door, Δ with derived run identities, cold reconstruction
CONFIRMED, doors byte-identical, and the model-turn laundering
attempt refused.

## VM-8 — Live-model proofs (optional, if a model endpoint is in scope)

```bash
export THEMIS_LIVE_OLLAMA=http://localhost:11434
export THEMIS_LIVE_TOOL_MODEL=qwen2.5:7b
cd "$REPO/src/harness"
go test ./verification/seam/ -run TestLiveRemediateWalk -count=1 -v
go test ./ratchet/ -run TestLiveModelAuthorsCandidate -count=1 -v
```

These are machine-local evidence: record the model name and digest
alongside the result.

## VM-9 — Teardown and evidence

```bash
# preserve the record plane as evidence BEFORE any cleanup
tar czf themis-evidence-$(date +%Y%m%d).tgz "$DEPLOY/state" "$DEPLOY/artifacts"
```

Record everything Phase F lists. Do not delete the state root until
the evidence archive exists — it is the only durable record of what
the deployment actually did.

---

## Known-good failure modes (expected, not findings)

| Symptom | Meaning |
|---|---|
| `TestLivePressureProof` / `TestLiveOperationalProof` fail in a full sweep, pass alone | documented contention flake (Ollama `500 timed out waiting for llama-server to start` under memory pressure) |
| Live tests skip | no model endpoint reachable — endpoint-gated by design |
| Live tests fail with "model not found" | the default `THEMIS_LIVE_MODEL` / `THEMIS_LIVE_TOOL_MODEL` is not pulled — category 1 |
| `Open` refuses with "no deployment anchor configured and Unanchored not explicitly set" | correct: production has no silent unanchored path |
| Admission refuses a hand-written anchor | correct: identity is not admission |

---

**Building a deployment (as opposed to validating one):**
`docs/operations/deployment-runbook.md`.
