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
| A4 | `THEMIS_LIVE_OLLAMA=http://127.0.0.1:9 go test ./... -count=1` | **all green, no exceptions.** The dead endpoint makes the nine live proofs skip, so this is purely deterministic and any failure is real |
| A5 | each live proof run SEPARATELY (see below) | each green, with model name and duration recorded |
| A6 | `go test ./deployment/ ./orchestration/ ./integration/ -count=1` | green — anchor, seam, and Phase C |

**Live proofs are endpoint-gated, not opt-in.** They skip only when
nothing answers at `THEMIS_LIVE_OLLAMA` (default
`http://localhost:11434`). If a model endpoint is up on the build host
they run, and they need their models present: `THEMIS_LIVE_TOOL_MODEL`
(default `qwen2.5:7b`) and `THEMIS_LIVE_MODEL` (default
`WhiteRabbitNeo/WHiteRabbitNeo-2.5-Qwen-2.5-Coder-7B:latest`).

**A5 — run each one separately.** Nine live proofs across nine packages
that `go test` runs in parallel will all drive one model server and time
out. Measured on 62 GB / 24-core / CPU-only: six failed in the sweep,
every one passing alone minutes earlier. Record model name and duration
for each:

```bash
go test ./context/ -run 'TestLiveOperationalProof|TestLivePressureProof' -count=1 -v
go test ./verification/seam/ -run TestLiveRemediateWalk -count=1 -v
go test ./ratchet/ -run TestLiveModelAuthorsCandidate -count=1 -v
go test ./tools/ -run TestLiveToolProof -count=1 -v
go test ./execution/ -run TestLiveExecutionProof -count=1 -v
go test ./state/ -run TestLiveTaskReconstruction -count=1 -v
go test ./orchestration/ -run 'TestLiveWalkProof|TestLiveSkillWalk' -count=1 -v
```

A live failure from a missing model, or from running them concurrently,
is category 1. A live failure **in isolation** is not.

Gate: A1–A6 pass before any deployment artifact is created.

---

## Phase B — Deployment artifact creation (governance act, owner-driven)

| # | Step | Expected |
|---|---|---|
| B1 | Write the execution ceiling with the owner's exact values to a deployment-local path (NOT the repo) | file exists, valid JSON |
| B2 | Confirm it LOADS: `scripts/themis-status --ceiling <file>`. `execution.LoadCeiling` requires `mirror_root` **absolute** and all bounds positive — it does **not** check that the directory exists, so the script checks that separately | loads; mirror_root exists; at least one bare mirror present |
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
| C17 | Skill composition member chosen by the submitter, differing from the catalog's pin (per-member correspondence, D-SA-2; the row's anchor admits the skill so the correspondence gate — not the D-SA-9 allowlist — is the one that refuses) | "a submitter selects an anchored skill, never its constituent hashes" |
| C17+ | **Positive twin (Q-SA-12 / A-SA-11):** a genuine L9-instantiated skill envelope under an anchor that lists the skill is ADMITTED and executes; every A-SA-1..10 negative refuses with its own gate's message. Lives in `orchestration/skill_admission_test.go` (`TestAnchoredSkillPositiveTwin`, `TestAnchoredSkillNegativeTwins`, `TestAnchoredWithdrawnSkillRefused`, `TestAnchoredSkillAllowlist`, `TestAnchoredUnattributedProcedureRefused`, `TestAnchoredSkillCatalogMismatchRefused`; key-wall twins `TestGrantKeyWallAtAssembly`, `TestInstantiateGrantPostBindAssertion` in `review_test.go`). Without it, "refuse everything" satisfied C17. | admitted (COMPLETED under the scripted model); refusals name their gate |
| C18 | `Open` with no anchor and no explicit `Unanchored` | "a deployment governs by anchor or refuses to open" |
| C19 | `Unanchored` declared alongside an anchor | "the caller role is ambiguous" |
| C20+ | **Layer 8 positive twin (first, the C17 lesson):** `remediate-dependency@2` (L9-instantiated) under an anchor pinning the delegation-template registry; a scripted parent reads `go.mod`, forms its evidence reference from the `record-ref` furniture it saw, delegates to `dependency-triage@1`, and completes | admitted; `l4-audit{authorized}` + `l8-delegation` witness; exactly one delegated model execution; `ReconstructTask` → CONFIRMED from the record alone |
| C20 | Delegation-template registry rewritten after pinning | "delegation-template registry is not the anchored artifact" |
| C21 | Delegation seam configured while the anchor declares `absent` | "declares no delegation-template registry" |
| C22 | Wired seam built over bytes other than the pinned registry (configured path still hashes to the pin) | "wired delegation seam holds a registry" |
| C23 | Skill's `template_scope` names a template the pinned registry never registered (the registry side; a caller-widened scope trips D-SA-4 first and is C17's family) | "does not resolve" |
| C24 | Phase exposes `delegate` but no seam is wired | "no L8 delegator is wired" |
| C25 | **Twin:** template withdrawn under the pinned registry (C-L8-14 G) | admitted at assembly; the `delegate` call refuses stage B `delegation-refused:template-withdrawn` in its audit; no witness; the walk COMPLETES |
| C26 | **Twin:** `delegate` names a template outside the Skill's scope | admitted; L4 `denied` with `template-outside-grant-scope`; no witness; the walk COMPLETES |
| C27 | **Twin:** `delegate` references evidence beyond the task's record | admitted; stage B `delegation-refused:evidence-unreachable`; no witness; the walk COMPLETES |

Automated coverage today: C1–C6, C8–C19 in
`orchestration/verification_seam_test.go` +
`deployment/anchor_test.go`; C7 in `deployment/anchor_test.go`;
C20–C27 in `subagents/delegation/seam/{anchored,e2e}_test.go`. On a
real deployment they are re-run against the REAL anchor rather than a
test-minted one (Phase E) by `evidence/harness/phasec`, which now
carries C20+–C27 (positive rows print `admitted`, and a positive row
that refuses is a finding). Smoke-run 2026-09-24 on the development
host against a scratch `rsys@4`: 25 rows, 0 findings.

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
| D8 | **L8:** `themis-run` submits the L9-instantiated `remediate-dependency@2` envelope under the ACTIVE `rsys@4` with a model that delegates (scripted stand-in or a capable live model) | COMPLETED; `l8-delegation` in the record; the manifest carries `l8_execution_bound`; `deployment_anchor` = the rsys@4 hash |
| D9 | **L8:** cold reconstruction of that delegation from the host record (`seam.ReconstructTask` under the anchored registry hash) | CONFIRMED; the composition object is the exact `[system, user]` input; template bytes referenced |
| D10 | **L8:** L10 history view of the task | `delegations[]` lists the witness read-only; no verification instance minted for it |
| D11 | **L8:** kill the process between the delegated model call and the witness commit (fault point `delegation.pre-event-commit`, or `kill -9` timed by the record) | task FAILED_PARTIAL by the sweep; the composition object retained and unreachable (orphan, not a fact); no `l8-delegation` |

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

**Worked example:** `docs/development/deployment-signoff-rsys.md` is the
Phase F record for the first instance (`rsys`, 2026-09-14) — what a
completed sign-off looks like, including how omissions and
not-applicable rows are stated rather than dropped.

Record, do not summarize away:
- Phase A–E results with dates and commit SHA.
- The anchor identity (`name@version` + hash) and the ceiling hash.
- Every refusal message observed in Phase C (the exact text).
- Any finding, with its triage category (1–4) and its disposition.
- Residuals confirmed still-recorded: submitter authentication;
  sensitivity inheritance (local-endpoint scope); the four
  consumption-pinned registries whose consumers live outside L7;
  `TestLivePressureProof` flake.
- **L8:** the C20+ record (witness seq, composition and output object
  ids, reconstruction verdict), the D8 `themis-run` receipt, the D11
  orphan scan, the VM-8 delegate-call count; residuals still-recorded:
  live model may not delegate (Register E admitted-not-delegated),
  per-item `derived_sensitivity` = parent ceiling, providers reporting
  dated model ids mint `model-identity-mismatch`,
  `remediate-dependency@1` cannot instantiate (reserved H2 headings).

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

Then run the preflight, which records most of that for you and, more
importantly, reports the conditions under which parts of the suite
SKIP rather than fail — git outside the two probed paths, a running
model endpoint whose models are absent, running as root:

```bash
scripts/themis-preflight --deploy "$DEPLOY"
```

Any FAIL there either blocks the build or silently removes evidence
from a green run. Resolve before Phase A, not after.

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

The whole matrix, C1–C27, runs from the evidence tool against the REAL
anchor (it copies the governed trees and never writes the deployment):

```bash
cd "$REPO/evidence/harness" && GOWORK=off go build -o /tmp/phasec ./phasec
/tmp/phasec -repo "$REPO" -deploy "$DEPLOY" \
  -anchor-file rsys4.json -anchor-sha256 "$ANCHOR_SHA" \
  -pinned-sha "$PINNED_SHA" -mirror-repo demo-vuln-app -git /usr/bin/git
#   expected: every C-row "refused" with its reason; C20+, C25, C26,
#   C27 "admitted"; "0 findings". Add -only C20+ to run one row.
```

`-mirror-repo` names the repository under the ceiling's `mirror_root`
whose `go.mod` the delegating walk reads; `-pinned-sha` is a commit
of that repository.

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

**L8 (D8–D11):**

```bash
# D8 — instantiate the delegating Skill through L9 for THIS deployment
#      and submit it through the real binary
cd "$REPO/src/harness" && go run ./cmd/themis-instantiate \
  -catalog "$REPO/policies/skills/catalog.json" -skill remediate-dependency@2 \
  -task rsys4-deleg-1 -repo demo-vuln-app -pinned-sha "$PINNED_SHA" \
  -input dependency=vulnerable-dep -input advisory=ADV-2026-1 \
  -model qwen2.5:7b -turn-timeout 180 -wall-deadline 300 \
  -registry "$REPO/policies/tools/registry-v5.json" -exec-ceiling "$DEPLOY/execution-ceiling.json" \
  -state "$DEPLOY/state" -artifacts "$DEPLOY/artifacts" -workspaces "$DEPLOY/provider" \
  -out "$DEPLOY/envelopes"
#   prints the envelope path; the effective grant/spec sit beside it.
#   L9's disjointness wall requires every task-writable root to EXIST
#   (mkdir -p "$DEPLOY"/{state,artifacts,provider,envelopes} first).
go run ./cmd/themis-run -deploy "$DEPLOY" -governed-root "$REPO" \
  -anchor "$REPO/policies/deployment/rsys4.json" -anchor-sha256 "$ANCHOR_SHA" \
  -anchors-registry "$REPO/policies/deployment/anchors.json" \
  -envelope "$DEPLOY/envelopes/<envelope>.json" -model-endpoint http://localhost:11434 -json
#   with a live model the model MAY not delegate (recorded, not a
#   finding); for a guaranteed delegation, run the C20+ row instead
#   (scripted parent) — it is the same walk under the same anchor.

# D9 — cold reconstruction from the host record
cat > /tmp/recon.go <<'GO'
package main
import ("encoding/json";"fmt";"os";"path/filepath"
 hctx "github.com/tofchaliss/themis/context";"github.com/tofchaliss/themis/state"
 "github.com/tofchaliss/themis/subagents/delegation/seam";"github.com/tofchaliss/themis/tools")
func main(){ root,err:=state.OpenRoot(os.Args[1]); if err!=nil{panic(err)}
 reg,err:=tools.LoadRegistry(filepath.Join(os.Args[3],"policies/tools/registry-v5.json")); if err!=nil{panic(err)}
 cfg:=seam.ReconstructConfig{RegistryHash:reg.Hash,ToolTrust:func(n string)(hctx.AuthorityClass,bool){for _,t:=range reg.Tools{if t.Name==n{return t.Trust,true}};return "",false}}
 recs,err:=seam.ReconstructTask(root,os.Args[2],cfg); if err!=nil{panic(err)}
 b,_:=json.MarshalIndent(recs,""," "); fmt.Println(string(b)) }
GO
cd "$REPO/src/harness" && go run /tmp/recon.go "$DEPLOY/state" rsys4-deleg-1 "$REPO"
#   expected: one entry, "verdict": "CONFIRMED", every check listed

# D10 — L10 observes, never evaluates
#   seam.TaskVerificationHistory(root, task).Delegations has one entry;
#   History/Latest carry only the report-valid@2 evaluation

# D11 — crash before the witness
#   start themis-run with a scripted-or-live model, `kill -9` the
#   process once `l4-audit{Tool:delegate, authorized}` is in the stream
#   and before `l8-delegation` appears; reopen → the sweep closes the
#   task FAILED_PARTIAL; state.ScanReachable() lists the composition
#   object as present and NOT reachable; no l8-delegation exists.
#   (Deterministic form: the hermetic TestDelegationFaultPoints.)
```

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
# L8 (Register E): the anchored @2 walk through the unmodified loop
go test ./subagents/delegation/seam/ -run TestLiveDelegationWalk -count=1 -v
#   asserted: admission + typed terminal + CONFIRMED reconstruction of
#   any delegation that occurred. Whether the model delegates is
#   recorded (the -v log prints delegate calls and witnessed
#   delegations), never asserted — the live-register discipline.
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
| Live proofs fail in a full sweep, pass alone | nine live proofs across nine parallel packages driving one model server; run them separately (A5). Not a host-capacity issue — reproduces on 16GB/8-core and worse on 62GB/24-core |
| Live tests skip | no model endpoint reachable — endpoint-gated by design |
| Live tests fail with "model not found" | the default `THEMIS_LIVE_MODEL` / `THEMIS_LIVE_TOOL_MODEL` is not pulled — category 1 |
| `Open` refuses with "no deployment anchor configured and Unanchored not explicitly set" | correct: production has no silent unanchored path |
| Admission refuses a hand-written anchor | correct: identity is not admission |

---

**Building a deployment (as opposed to validating one):**
`docs/operations/deployment-runbook.md`.
