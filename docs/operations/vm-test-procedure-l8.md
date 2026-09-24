# Linux VM test procedure — Layer 8 (delegation) on a fresh deployment

Every line below is a bash command or a `#` comment. Run top to
bottom in one shell on a clean Ubuntu 22.04+ VM. `$REPO` is the
checkout, `$DEPLOY` a directory OUTSIDE it. Where a step must be
performed by the owner (a Governance act), the comment says so; the
commands still run, they just carry that meaning.

Expected outcomes are stated as comments beginning `# expect:`. A
refusal with the wrong reason, or a positive row that refuses, is a
finding — record it, do not "fix" it by weakening a loader.

```bash
# =====================================================================
# 0. Identity of this run — record everything this prints
# =====================================================================
export REPO="$HOME/themis-ai-runtime"           # the checkout
export DEPLOY="/srv/themis"                     # deployment root, outside the repo
export GIT_BIN="$(command -v git || echo /usr/bin/git)"
date -u; uname -a; id -un

# =====================================================================
# 1. Provision the VM (base tooling, Go, optional local model)
# =====================================================================
sudo apt-get update && sudo apt-get install -y git curl build-essential python3
curl -fsSL https://go.dev/dl/go1.24.0.linux-amd64.tar.gz -o /tmp/go.tgz
sudo rm -rf /usr/local/go && sudo tar -C /usr/local -xzf /tmp/go.tgz
export PATH="$PATH:/usr/local/go/bin"
go version                                        # expect: go1.24.x
git --version; which git                          # the harness pins the ABSOLUTE git path

# a local model endpoint — only if the live walk (step 11) is in scope
curl -fsSL https://ollama.com/install.sh | sh
ollama pull qwen2.5:7b
curl -fsS http://localhost:11434/api/tags | python3 -c 'import sys,json;print([m["name"] for m in json.load(sys.stdin)["models"]])'

# =====================================================================
# 2. Checkout and the static gate (Phase A)
# =====================================================================
git clone https://github.com/tofchaliss/themis-ai-runtime.git "$REPO"
cd "$REPO" && git rev-parse HEAD                  # record the SHA under test (fdbc8e7 or later)
cd "$REPO/src/harness"
go build ./... && go vet ./... && test -z "$(gofmt -l .)" && echo STATIC-OK
# hermetic suite — point the live endpoint at a closed port so live proofs SKIP here
THEMIS_LIVE_OLLAMA=http://127.0.0.1:9 go test ./... -count=1 2>&1 | grep -v '^ok' | tail -20
# expect: no FAIL lines (only "no test files" noise); ~10 minutes
cd "$REPO/evidence/harness" && GOWORK=off go build ./... && GOWORK=off go vet ./... && echo EVIDENCE-OK

# =====================================================================
# 3. Preflight — the host conditions under which evidence would SKIP
# =====================================================================
cd "$REPO" && scripts/themis-preflight --deploy "$DEPLOY" 2>&1 | tee /tmp/preflight.txt
# expect: no FAIL; the "Delegation templates" section prints
#   PASS dependency-triage@1 (active): manifest and every pin verified
#   delegation_template_registry pin: <sha>
# Resolve any FAIL before continuing.

# =====================================================================
# 4. Deployment root, mirror, execution ceiling (Phase B1–B3)
# =====================================================================
sudo mkdir -p "$DEPLOY" && sudo chown "$(id -un)" "$DEPLOY"
mkdir -p "$DEPLOY"/{mirror,state,artifacts,provider,envelopes,submissions}

# the governed mirror: the directory name IS the spec's `repo` value
git init -q "$DEPLOY/mirror/demo-vuln-app"
cd "$DEPLOY/mirror/demo-vuln-app"
printf 'module demo // vulnerable-dep v1\n' > go.mod
git -c user.name=t -c user.email=t@t add . && git -c user.name=t -c user.email=t@t commit -q -m seed
export PINNED_SHA="$(git rev-parse HEAD)"; echo "$PINNED_SHA"   # record

# the deployment's execution ceiling — exact bytes, pinned by the anchor
cat > "$DEPLOY/execution-ceiling.json" <<JSON
{"version":1,"mirror_root":"$DEPLOY/mirror","max_wall_deadline_sec":600,"max_file_bytes":1048576,"max_total_bytes":10485760,"max_file_count":500,"max_mem_bytes":1073741824,"max_cpu_time_sec":600,"max_proc_count":64}
JSON
sha256sum "$DEPLOY/execution-ceiling.json"        # record

# =====================================================================
# 5. Mint rsys@4 for THIS deployment (Phase B4–B5)
#    On a production host the ceiling and models come from the ACTIVE
#    rsys@3 (docs/operations/rsys4-host-sequence.md). This VM has no
#    rsys@3, so the ceiling is the one just written and the allowlist is
#    the model this VM serves. Everything else is computed from the tree.
# =====================================================================
cd "$REPO"
python3 - <<'PY'
import json, hashlib, os
R = os.getcwd(); D = os.environ["DEPLOY"]
def sha(p): return hashlib.sha256(open(p, "rb").read()).hexdigest()
a = json.load(open("policies/deployment/rsys4.proposed.json"))
a["execution_ceiling"] = sha(D + "/execution-ceiling.json")
a["models"] = ["qwen2.5:7b"]
a["skills"] = ["remediate-dependency@1", "remediate-dependency@2"]
assert a["name"] == "rsys" and a["deployment_version"] == 4
open("policies/deployment/rsys4.json", "w").write(json.dumps(a, indent=1) + "\n")
print("rsys4.json written")
PY
export ANCHOR_SHA="$(sha256sum policies/deployment/rsys4.json | cut -d' ' -f1)"; echo "$ANCHOR_SHA"   # record: the deployment identity

# parse + schema validation
cat > /tmp/anchorcheck.go <<'GO'
package main
import ("fmt";"os";"github.com/tofchaliss/themis/deployment")
func main(){ b,_:=os.ReadFile(os.Args[1]); a,err:=deployment.ParseAnchor(b,os.Args[1])
 if err!=nil{fmt.Println("REFUSED:",err);os.Exit(1)}; fmt.Println("anchor ok:",a.Name,a.Deployment,a.SHA256) }
GO
cd "$REPO/src/harness" && go run /tmp/anchorcheck.go "$REPO/policies/deployment/rsys4.json"
# expect: anchor ok: rsys 4 <ANCHOR_SHA>

# every tree pin equals the anchor's field (themis-status renders
# "  <name>  <hash>" lines; the constitution lines are named l6 / l7)
cd "$REPO" && scripts/themis-status 2>&1 | sed 's/\x1b\[[0-9;]*m//g' | tee /tmp/status.txt >/dev/null
python3 - <<'PY'
import json, re
a = json.load(open("policies/deployment/rsys4.json"))
pins = {}
for l in open("/tmp/status.txt"):
    m = re.match(r"\s+(\S+)\s+([0-9a-f]{64})\s*$", l)
    if m: pins[m.group(1)] = m.group(2)
want = {k: a[k] for k in ("instruction_root_safety","instruction_root_system","instruction_root_themis",
        "instruction_policy","tool_registry","skill_catalog","contract_registry","criteria_registry",
        "regression_set_registry","delegation_template_registry")}
want["l6"] = a["constitution"]["state"]; want["l7"] = a["constitution"]["orchestration"]
bad = {k: (v, pins.get(k)) for k, v in want.items() if pins.get(k) != v}
print("PINS OK" if not bad else f"PIN MISMATCH {bad}")
PY
# expect: PINS OK  (a mismatch means the tree and the proposal disagree — stop and reconcile)

# =====================================================================
# 6. Prove the proposal is INERT, then the Governance act (Phase B6–B8)
# =====================================================================
cat > /tmp/admitcheck.go <<'GO'
package main
import ("fmt";"os";"github.com/tofchaliss/themis/deployment")
func main(){ _,err:=deployment.AdmitAnchor(os.Args[1],os.Args[2],os.Args[3])
 fmt.Println("admission:",err) }
GO
cd "$REPO/src/harness" && go run /tmp/admitcheck.go "$REPO/policies/deployment/rsys4.json" "$ANCHOR_SHA" "$REPO/policies/deployment/anchors.json"
# expect: a refusal — rsys@4 is not registered. If this admits, STOP: the proposal path is not inert (a defect).

# the registration (still inert: anchors.json is untouched)
cd "$REPO" && python3 - <<'PY'
import json, os
reg = json.load(open("policies/deployment/anchors.json"))
reg["entries"].append({"name": "rsys", "version": 4, "artifact_sha256": os.environ["ANCHOR_SHA"], "state": "active", "steward": "security-engineering"})
open("policies/deployment/anchors.proposed.json", "w").write(json.dumps(reg, indent=1) + "\n")
print("anchors.proposed.json written")
PY
cd "$REPO/src/harness" && go run /tmp/admitcheck.go "$REPO/policies/deployment/rsys4.json" "$ANCHOR_SHA" "$REPO/policies/deployment/anchors.json"
# expect: still a refusal

# --- GOVERNANCE ACT (owner) — activation. Nothing before this line admitted anything.
cd "$REPO" && cp policies/deployment/anchors.proposed.json policies/deployment/anchors.json
git add policies/deployment/ && git -c user.name="$(id -un)" -c user.email="$(id -un)@vm" commit -q -m "Governance act: rsys@4 ACTIVE (VM $(hostname))"
cd "$REPO/src/harness" && go run /tmp/admitcheck.go "$REPO/policies/deployment/rsys4.json" "$ANCHOR_SHA" "$REPO/policies/deployment/anchors.json"
# expect: admission: <nil>

# =====================================================================
# 7. Negative space against the REAL anchor — the full matrix (Phase C)
#    The tool copies the governed trees and never writes the deployment.
# =====================================================================
cd "$REPO/evidence/harness" && GOWORK=off go build -o /tmp/phasec ./phasec
/tmp/phasec -repo "$REPO" -deploy "$DEPLOY" -anchor-file rsys4.json -anchor-sha256 "$ANCHOR_SHA" \
  -pinned-sha "$PINNED_SHA" -mirror-repo demo-vuln-app -git "$GIT_BIN" 2>&1 | tee /tmp/phase-c.txt
# expect: the last lines read "25 refused correctly, 0 findings, 0 skipped";
#   C1..C24 "refused" each with its reason; C20+, C25, C26, C27 "admitted".
# One row at a time, when investigating:
/tmp/phasec -repo "$REPO" -deploy "$DEPLOY" -anchor-file rsys4.json -anchor-sha256 "$ANCHOR_SHA" \
  -pinned-sha "$PINNED_SHA" -mirror-repo demo-vuln-app -git "$GIT_BIN" -only C20+

# =====================================================================
# 8. D8 — the delegating Skill through the real binary (Phase D)
# =====================================================================
cd "$REPO/src/harness"
ENVELOPE="$(go run ./cmd/themis-instantiate \
  -catalog "$REPO/policies/skills/catalog.json" -skill remediate-dependency@2 \
  -task rsys4-deleg-1 -repo demo-vuln-app -pinned-sha "$PINNED_SHA" \
  -input dependency=vulnerable-dep -input advisory=ADV-2026-1 \
  -model qwen2.5:7b -turn-timeout 180 -wall-deadline 300 \
  -registry "$REPO/policies/tools/registry-v5.json" -exec-ceiling "$DEPLOY/execution-ceiling.json" \
  -state "$DEPLOY/state" -artifacts "$DEPLOY/artifacts" -workspaces "$DEPLOY/provider" \
  -out "$DEPLOY/envelopes")"; echo "$ENVELOPE"
# expect: the envelope path; its effective grant and spec sit beside it

go run ./cmd/themis-run -deploy "$DEPLOY" -governed-root "$REPO" \
  -anchor "$REPO/policies/deployment/rsys4.json" -anchor-sha256 "$ANCHOR_SHA" \
  -anchors-registry "$REPO/policies/deployment/anchors.json" \
  -envelope "$ENVELOPE" -model-endpoint http://localhost:11434 -git "$GIT_BIN" -json 2>&1 | tee /tmp/d8-receipt.json
# expect: "status" COMPLETED or FAILED (a typed terminal, never a refusal);
#   "deployment_anchor" equals $ANCHOR_SHA. With a live 7B model the walk
#   MAY complete without delegating — that is model behaviour, recorded,
#   not a finding. The guaranteed delegation is row C20+ (same walk,
#   scripted parent) — its record is under $DEPLOY/ctest/C20+/state.

# what the record holds
ls "$DEPLOY/state/tasks/rsys4-deleg-1/"
grep -o '"class":"[a-z0-9-]*"' "$DEPLOY"/state/tasks/rsys4-deleg-1/*.log 2>/dev/null | sort | uniq -c
# expect (if the model delegated): one "l8-delegation"; always: l4-audit, model-turn, lifecycle

# =====================================================================
# 9. D9/D10 — cold reconstruction and the L10 view (record only, no registry files)
# =====================================================================
cat > /tmp/recon.go <<'GO'
package main
import ("encoding/json";"fmt";"os";"path/filepath"
 hctx "github.com/tofchaliss/themis/context";"github.com/tofchaliss/themis/state"
 "github.com/tofchaliss/themis/subagents/delegation/seam";"github.com/tofchaliss/themis/tools"
 vseam "github.com/tofchaliss/themis/verification/seam")
func main(){ root,err:=state.OpenRoot(os.Args[1]); if err!=nil{panic(err)}
 reg,err:=tools.LoadRegistry(filepath.Join(os.Args[3],"policies/tools/registry-v5.json")); if err!=nil{panic(err)}
 cfg:=seam.ReconstructConfig{RegistryHash:reg.Hash,ToolTrust:func(n string)(hctx.AuthorityClass,bool){for _,t:=range reg.Tools{if t.Name==n{return t.Trust,true}};return "",false}}
 recs,err:=seam.ReconstructTask(root,os.Args[2],cfg); if err!=nil{panic(err)}
 b,_:=json.MarshalIndent(recs,""," "); fmt.Println(string(b))
 view,err:=vseam.TaskVerificationHistory(root,os.Args[2]); if err!=nil{panic(err)}
 v,_:=json.MarshalIndent(view,""," "); fmt.Println(string(v)) }
GO
# the C20+ record (guaranteed delegation)
cd "$REPO/src/harness" && go run /tmp/recon.go "$DEPLOY/ctest/C20+/state" c20-pos "$REPO" | tee /tmp/d9-c20.json
# expect: one reconstruction, "verdict": "CONFIRMED"; the history view's
#   "delegations" lists one entry; "history"/"latest_per_contract" carry
#   only the report-valid@2 evaluation (a delegation is never a verification)
# the D8 record (delegation only if the live model delegated)
go run /tmp/recon.go "$DEPLOY/state" rsys4-deleg-1 "$REPO" | tee /tmp/d9-d8.json

# =====================================================================
# 10. D11 — crash before the witness (stage E: an orphan is not a fact)
#     Deterministic form: the hermetic fault-point test, run here on the
#     VM's own record plane.
# =====================================================================
cd "$REPO/src/harness" && go test ./subagents/delegation/seam/ -run 'TestDelegationFaultPoints' -count=1 -v 2>&1 | tail -8
# expect: PASS for delegation.pre-composition-store, pre-output-store,
#   pre-event-commit; the last asserts the composition object is
#   retained and UNREACHABLE and no l8-delegation exists.

# =====================================================================
# 11. Register E — live delegation walk through the unmodified loop (VM-8)
# =====================================================================
export THEMIS_LIVE_OLLAMA=http://localhost:11434
export THEMIS_LIVE_TOOL_MODEL=qwen2.5:7b
cd "$REPO/src/harness" && go test ./subagents/delegation/seam/ -run TestLiveDelegationWalk -count=1 -v -timeout 900s 2>&1 | tee /tmp/vm8-live.txt | grep -E 'live:|--- (PASS|FAIL|SKIP)'
# expect: PASS; the "live:" line reports status, delegate calls, and
#   witnessed delegations. Zero delegations is RECORDED (the model chose
#   not to), never a finding; any delegation that occurred must
#   reconstruct CONFIRMED (asserted by the test).
ollama show qwen2.5:7b 2>/dev/null | head -5      # record the model digest beside the result

# =====================================================================
# 12. Evidence capture (Phase F) — before any cleanup
# =====================================================================
mkdir -p "$HOME/evidence" && cd "$HOME/evidence"
cp /tmp/preflight.txt /tmp/phase-c.txt /tmp/d8-receipt.json /tmp/d9-c20.json /tmp/d9-d8.json /tmp/vm8-live.txt . 2>/dev/null
( cd "$REPO" && git rev-parse HEAD ) > commit.txt
echo "anchor rsys@4 $ANCHOR_SHA" > anchor.txt
sha256sum "$DEPLOY/execution-ceiling.json" >> anchor.txt
tar czf "themis-l8-evidence-$(date -u +%Y%m%d).tgz" "$DEPLOY/state" "$DEPLOY/artifacts" "$DEPLOY/ctest" ./*.txt ./*.json
ls -la "$HOME/evidence"
# Record in the sign-off document: the commit SHA, the anchor identity
# and ceiling hash, every Phase C refusal text (from phase-c.txt), the
# C20+ reconstruction verdict, the D8 receipt, the VM-8 delegate-call
# count and model digest, and any finding with its triage category.
```

## Reading the results

| Step | Green means | A finding means |
|---|---|---|
| 2 | static gate and hermetic suite pass at the SHA under test | category 2 (implementation defect) unless a documented flake |
| 5–6 | pins match the tree; the proposal is inert until the act; admission is `<nil>` after it | category 1 (deployment) for a pin mismatch; category 2 if the proposal path admits |
| 7 | 25 rows, 0 findings | any "WRONG REASON" or "ADMITTED" on C1–C24, any "REFUSED" on C20+/C25/C26/C27 |
| 8–9 | a typed terminal with the anchor in the record; C20+ reconstructs CONFIRMED | a refusal at submission is category 1 or 2; DISCREPANCY is a category-2 defect signal |
| 10 | three fault points PASS | category 2 |
| 11 | admitted, typed terminal, CONFIRMED for any delegation | a FAIL is real only when run in isolation as here |

Nothing in this procedure authorizes production wiring. It produces
the evidence the owner decides on.
