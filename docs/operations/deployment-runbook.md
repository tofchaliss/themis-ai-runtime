# Deployment Runbook — themis-ai-runtime

Step-by-step procedure for standing up a governed Themis AI Runtime
deployment. Written 2026-09-13 against the frozen architecture
(L1–L11 + G1 + G2).

**What this produces:** a Governance-ACTIVE Deployment Anchor and a
verified deployment whose every governing artifact is pinned.

**What this does NOT do:** grant production wiring. Wiring a
production caller to `orchestration.Open`/`SubmitTask` is a separate
owner decision (see Step 12).

**Read first:** `docs/architecture/harness/execution-chain.md` (what
the chain is), `policies/deployment/README.md` (the ceiling
contract). Validation belongs to
`docs/development/deployment-test-plan.md` — this runbook builds;
that plan proves.

**Standing rule:** if a step refuses, do not weaken a loader, infer a
default, or reinterpret an artifact. Classify the problem
(1 deployment/config · 2 implementation defect · 3 recorded residual
· 4 architectural gap) and fix it at its own level. Only category 4
touches a locked decision.

---

## Roles

| Role | Does |
|---|---|
| **Operator** | provisions the host, supplies deployment-specific configuration, runs the binary |
| **Governance** (human owner) | reviews the proposed anchor and performs the activation act |
| **Harness** | verifies and refuses; it never activates anything |

The operator prepares; Governance admits. A prepared anchor has no
authority until Governance acts (D-G1-1A).

---

## Inputs the operator must have before starting

| Input | Example | Notes |
|---|---|---|
| Deployment name | `prod-sec-a` | lowercase kebab-case |
| Deployment version | `1` | integer ≥ 1; a new version is a new identity |
| `mirror_root` | `/srv/themis/mirror` | absolute host path; the git mirror this deployment may provision from |
| Execution limits | wall 600s, 1 MiB/file, … | the ceiling's numeric bounds |
| Model allowlist | `["qwen2.5:7b"]` | exact names permitted in envelopes |
| Model registry | file, or `absent` | what those names resolve to; `absent` = local-only resolution |
| Steward | `security-engineering` | accountability metadata on the registration |

Everything else is computed from the repository.

---

## Step 1 — Provision the host

```bash
sudo apt-get update && sudo apt-get install -y git curl build-essential
curl -fsSL https://go.dev/dl/go1.24.0.linux-amd64.tar.gz -o /tmp/go.tgz
sudo rm -rf /usr/local/go && sudo tar -C /usr/local -xzf /tmp/go.tgz
export PATH=$PATH:/usr/local/go/bin
go version && which git
```

Record: OS/kernel, Go version, **absolute git path** (the harness
pins the git binary), hostname.

**Check the Go caches are not on a quota-limited filesystem.** Go
defaults `GOCACHE` and `GOMODCACHE` to `~/.cache` and `~/go`. On hosts
with a networked or quota'd home — common in corporate estates, where
`/home` is autofs while the bulk storage is local — the build cache
exhausts the quota mid-run:

```bash
df -hT /home "$(go env GOCACHE)" .
go env GOCACHE GOMODCACHE
```

If home is limited, relocate them to the local filesystem and persist
it, because the process that eventually runs governed tasks builds
under the same account:

```bash
sudo mkdir -p /srv/themis/build && sudo chown "$(id -u):$(id -g)" /srv/themis/build
mkdir -p /srv/themis/build/{gocache,gomodcache}
grep -q GOCACHE ~/.profile || {
  echo 'export GOCACHE=/srv/themis/build/gocache'
  echo 'export GOMODCACHE=/srv/themis/build/gomodcache'
} >> ~/.profile
```

**Why this is a provisioning step and not a footnote:** a quota
exhausted mid-run does not fail the suite cleanly. Packages that cannot
build simply do not run, their tests never report, and the shell still
exits 0. Observed 2026-09-14: a run that looked plausible had silently
executed five fewer packages than the one before it, detectable only
because the skip audit returned 5 lines instead of 10. A green result
from a host in this state means nothing.

Finally, confirm the host is ready:

```bash
scripts/themis-preflight --deploy "$DEPLOY"
```

It reports the conditions under which parts of the suite SKIP rather
than fail — git outside the probed paths, a reachable endpoint whose
models are absent, running as root, CPU-only inference. Resolve every
FAIL before Step 2.

## Step 2 — Check out and gate the build

```bash
export REPO=/opt/themis/themis-ai-runtime
git clone <repo-url> "$REPO" && cd "$REPO/src/harness"
git rev-parse HEAD          # record the SHA this deployment runs
go build ./... && go vet ./... && gofmt -l .
THEMIS_LIVE_OLLAMA=http://127.0.0.1:9 go test ./... -count=1
```

The dead endpoint makes the nine live proofs skip, so this run is purely
deterministic and must be **fully green** — any failure is real. Measured
on 24 vCPU: about 8 seconds.

Then run the live proofs SEPARATELY. Nine of them across nine packages
that `go test` runs in parallel will all drive one model server and time
out; this is concurrency, not host capacity (it is worse on a bigger
host). See the test plan's A5 for the full list:

```bash
go test ./context/ -run 'TestLiveOperationalProof|TestLivePressureProof' -count=1
go test ./verification/seam/ -run TestLiveRemediateWalk -count=1
```

The live proofs are **endpoint-gated, not opt-in**: they skip only when
nothing answers at `THEMIS_LIVE_OLLAMA` (default
`http://localhost:11434`). If this host runs a model endpoint, pull the
models the proofs name (`THEMIS_LIVE_TOOL_MODEL`, default `qwen2.5:7b`;
`THEMIS_LIVE_MODEL`) or point the variables at models you have.

A failure anywhere else stops the deployment.

## Step 3 — Create the deployment directory (outside the repo)

```bash
export DEPLOY=/srv/themis/prod-sec-a
mkdir -p "$DEPLOY"/{mirror,state,artifacts,provider}
chmod 700 "$DEPLOY"
```

`$DEPLOY` holds deployment-specific configuration and the record
plane. It is **never** inside `$REPO`: the ceiling is deployment
configuration, not a repository artifact.

## Step 4 — Seed the governed mirror

```bash
git clone --mirror <governed-source-repo> "$DEPLOY/mirror/<repo>"
git -C "$DEPLOY/mirror/<repo>" rev-parse HEAD   # a spec pins this SHA
```

The mirror is what L5 provisions workspaces from. Only repositories
placed here are reachable.

**The directory name under `mirror_root` IS the spec's `repo` value.**
L5 resolves `repo` directly against the mirror root
(`confine.ResolvePath(ceiling.MirrorRoot, spec.Repo)`), so a mirror
cloned as `<repo>.git` requires every spec to say `"repo": "<repo>.git"`.
Cloning without the suffix keeps specs readable; either is valid, but
the two must agree exactly or provisioning refuses.

**Mirror contents are not pinned by the anchor.** The ceiling pins
`mirror_root` as a path; what sits inside it is operational state. That
means a repository can be added later without a new anchor version —
and equally, **write access to `mirror_root` is the control** over what
this deployment can ever provision from. Keep `$DEPLOY` mode 700 and
treat that directory as governed infrastructure.

## Step 5 — Write the execution ceiling

```bash
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
sha256sum "$DEPLOY/execution-ceiling.json"
```

This hash is the `execution_ceiling` pin. **Verify it loads** before
going further — a ceiling that cannot instantiate must never be
pinned:

```bash
scripts/themis-status --ceiling "$DEPLOY/execution-ceiling.json"
```

**`LoadCeiling` requires `mirror_root` to be absolute, not to exist.**
A mistyped path therefore passes this step, pins cleanly, survives the
Governance act and `Open`, and fails at the first governed task. The
status script checks existence separately for exactly that reason;
if you verify by hand instead, check the directory yourself.

Equivalent by hand (`go run` does **not** read a program from stdin, so
the program goes in a file):

```bash
cat > /tmp/ceilingcheck.go <<'GO'
package main
import ("fmt";"os";"github.com/tofchaliss/themis/execution")
func main(){ if _,err:=execution.LoadCeiling(os.Args[1]);err!=nil{fmt.Println("REFUSED:",err);os.Exit(1)}; fmt.Println("ceiling loads") }
GO
cd "$REPO/src/harness" && go run /tmp/ceilingcheck.go "$DEPLOY/execution-ceiling.json"
```

## Step 6 — Decide the model registry

Either a real registry:

```bash
cp <models.json> "$DEPLOY/models.json"
sha256sum "$DEPLOY/models.json"        # → the model_registry pin
```

…or none, in which case the anchor declares the string `"absent"`
and `ModelRegistryPath` stays empty. `absent` is a **declaration**,
not a default: configuring a registry while the anchor says `absent`
is refused, and vice versa.

## Step 7 — Compute the artifact pins

The anchor pins exact content. Compute with the same functions the
loader uses — directory pins are `deployment.HashDir`, file pins are
`deployment.HashFile`, and the constitutions are code identities:

| Anchor field | Source |
|---|---|
| `instruction_root_safety` | `HashDir($REPO/instructions/global/safety)` |
| `instruction_root_system` | `HashDir($REPO/instructions/global/system)` |
| `instruction_root_themis` | `HashDir($REPO/instructions/themis)` |
| `instruction_policy` | `HashFile($REPO/policies/security/instruction-directive-patterns.json)` |
| `tool_registry` | `HashFile($REPO/policies/tools/registry-v4.json)` |
| `execution_ceiling` | `HashFile($DEPLOY/execution-ceiling.json)` |
| `constitution.state` | `state.ConstitutionHash()` |
| `constitution.orchestration` | `orchestration.ConstitutionHash()` |
| `workflows[]` | per workflow: `workflow`, `workflow_ceiling`, `context_contract` — an indivisible bundle |
| `models[]` | the allowlist names |
| `model_registry` | `HashFile($DEPLOY/models.json)` or `"absent"` |
| `skill_catalog` | `HashFile($REPO/policies/skills/catalog.json)` |
| `contract_registry` | `HashFile($REPO/policies/verification/contracts.json)` |
| `criteria_registry` | `HashFile($REPO/policies/ratchet/criteria.json)` |
| `regression_set_registry` | `HashFile($REPO/policies/ratchet/regression-sets.json)` |

> **Open choice (undecided):** whether a reviewed anchor-minting
> helper should exist, or whether pins are computed per deployment in
> session. Until decided, compute them explicitly and keep the
> transcript as evidence. Do not script this into an unreviewed tool.

## Step 8 — Emit the PROPOSED anchor and registration

```bash
$REPO/policies/deployment/<name>.json          # the anchor (all pins)
$REPO/policies/deployment/anchors.proposed.json # the registration
```

The registration entry is `{name, version, artifact_sha256, state:
"active", steward}` where `artifact_sha256` is the sha256 of the
anchor file's exact bytes.

**Parse-verify the anchor before proposing it** — a malformed anchor
must never reach Governance:

```bash
cat > /tmp/anchorcheck.go <<'GO'
package main
import ("fmt";"os";"github.com/tofchaliss/themis/deployment")
func main(){ b,_:=os.ReadFile(os.Args[1]); a,err:=deployment.ParseAnchor(b,os.Args[1])
 if err!=nil{fmt.Println("REFUSED:",err);os.Exit(1)}; fmt.Println("anchor ok:",a.Name,a.Deployment,a.SHA256) }
GO
cd "$REPO/src/harness" && go run /tmp/anchorcheck.go "$REPO/policies/deployment/<name>.json"
```

## Step 9 — Prove the proposal is INERT

Before the Governance act, admission **must refuse** — a prepared
anchor has no authority:

```bash
cat > /tmp/admitcheck.go <<'GO'
package main
import ("fmt";"os";"github.com/tofchaliss/themis/deployment")
func main(){ _,err:=deployment.AdmitAnchor(os.Args[1],os.Args[2],os.Args[3])
 fmt.Println("pre-activation admission:",err) }
GO
cd "$REPO/src/harness" && go run /tmp/admitcheck.go \
  "$REPO/policies/deployment/<name>.json" "<anchor-sha>" \
  "$REPO/policies/deployment/anchors.json"
```

Expected: a refusal (registry unavailable, or the anchor not
registered). **If this succeeds, stop** — the proposal path is not
inert and that is an implementation defect.

## Step 10 — Governance act: activate

Performed by the owner, recorded as a Governance act:

```bash
cp "$REPO/policies/deployment/anchors.proposed.json" \
   "$REPO/policies/deployment/anchors.json"
git add policies/deployment/ && git commit -m "Governance act: <name>@<v> ACTIVE"
```

Re-run Step 9's command: it must now print `<nil>`, and two-way
identity must hold (the registration's `name@version` equals the
anchor's self-declaration). Record the anchor hash — it is the
**deployment identity**.

## Step 11 — Verify the deployment opens

Open with the real anchor, registry, and ceiling. Required
configuration:

| Field | Value |
|---|---|
| `StateRoot` | `$DEPLOY/state` |
| `ArtifactDir` | `$DEPLOY/artifacts` |
| `GitPath` | absolute git path from Step 1 |
| `ProviderDir` | `$DEPLOY/provider` |
| `SafetyRoot` / `SystemRoot` / `ThemisRoot` | the three instruction roots |
| `PolicyPath` | the instruction-directive policy |
| `AnchorPath` / `AnchorSHA256` / `AnchorsRegistryPath` | the anchor, its hash, the governed registry |
| `ExecCeilingPath` | `$DEPLOY/execution-ceiling.json` |
| `SkillCatalogPath` | the governed catalog (required for skill-attributed tasks) |
| `ModelRegistryPath` | the registry, or empty when the anchor says `absent` |
| `Model` / `Verifier` | the model provider and the L10 evaluator |

`Unanchored` stays **false**. Open must succeed; on refusal, read the
message — it names the artifact that is not the anchored one.

## Step 12 — Production wiring (owner decision, NOT part of this runbook)

**There is no shipped production entry binary.** `src/harness/cmd/`
contains `themis-ratchet` (the L11 invocation surface) only; the
legacy HTTP service was decommissioned (audit R1). The process that
opens an orchestrator and submits governed tasks in production is
the remaining decision: what it is, who may call it, and how a
submitter is authenticated (submitter authentication is an explicit
recorded G1 residual — origin is recorded, never treated as
Governance authority).

Until that decision, a deployment is **prepared and verifiable but
not wired**.

## Step 13 — Validate

Run `docs/development/deployment-test-plan.md` against this
deployment: Phase C (the substitution attempts that must refuse),
Phase D (anchored positive path and kill/recovery), Phase E (the
governed chain end to end), Phase F (evidence).

## Step 14 — Record the evidence

```bash
tar czf "themis-deploy-$(date +%Y%m%d).tgz" "$DEPLOY/state" "$DEPLOY/artifacts"
```

Record: repo SHA, deployment name@version, **anchor hash**, ceiling
hash, Go/git versions, the activation commit, and every refusal
observed during validation.

---

## Changing a deployment

| Change | Procedure |
|---|---|
| Ceiling values (limits, mirror_root) | new ceiling bytes → new pin → **new anchor version** → propose → activate → restart |
| Instruction content, tool registry, workflow, contract | re-pin → new anchor version → propose → activate → restart |
| Model allowlist or registry | same — a model enters a deployment only by Governance act |
| Rebuilt binary with a different constitution | new anchor version (the constitution pins will refuse otherwise) |

**Adoption is by restart only.** There is no live anchor mutation:
one anchor per Open, frozen for the lifetime. Superseding an anchor
means registering a new version and withdrawing the old one —
`active → withdrawn`, never deletion, so past records stay
interpretable.

## Withdrawing a deployment

Set the registration's state to `withdrawn` and commit as a
Governance act. New Opens refuse; existing records remain
explicable, and `deployment.VerifyAnchorRecord` still re-establishes
what governed them.

---

## Troubleshooting

| Message | Meaning | Category |
|---|---|---|
| `no deployment anchor configured and Unanchored not explicitly set` | correct refusal — production has no silent unanchored path | — |
| `identifier, never an admission claim` | the anchor is not registered; run the Governance act | 1 |
| `<artifact> is not the anchored artifact` | the file changed after pinning, or the wrong path is configured | 1 |
| `non-regular entry in a pinned tree` | a symlink/device inside a pinned instruction root | 1 |
| `constitution is not the anchored one` | the binary was rebuilt with different control vocabularies | 1 (new anchor version) |
| `supplied at Open, never chosen per task` | an envelope named its own execution ceiling | 1 |
| `a submitter selects an anchored skill, never its constituent hashes` | the envelope's composition disagrees with the catalog | 1 |
| `deployment identity is immutable` / `disappeared` | the anchors registry was mutated out of band — investigate before proceeding | 1 or 2 |
| `the anchored execution ceiling does not load` | the pinned ceiling is malformed; fix the ceiling, re-pin, new version | 1 |

Anything not on this list, and not explained by a recorded residual,
is a finding: classify it before acting on it.
