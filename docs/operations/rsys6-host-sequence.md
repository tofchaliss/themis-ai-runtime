# `rsys@6` — host sequence for I-M5 (prepared 2026-09-27; NOT an activation)

I-M5 is the host act that closes the Themis integration
(`openspec/changes/themis-integration/tasks.md`, D-I-8, D-W-4): the
Themis estate runs on the same host as the harness (D-I-1), the read
door reads a real Finding (D-I-3), one mint of `rsys@6` carries the
W-M1 constitution, the `themis_contract` pin and catalog @4, and a
human decides a Position on a five-link, commissioned execution
(D-I-5, D-I-6). This page runs on the host from the governed checkout
at the commit that carries it. `$REPO` is the harness checkout,
`$THEMIS` the Themis checkout, `$DEPLOY` the deployment root
(`/srv/themis/rsys` on this host). **This page stops before every
Governance act (steps 6 and 10); the owner performs them.** The host
cannot push (Addendum F): Governance-act commits come back to origin
as a patch bundle.

Lesson folded in from `rsys@4` (Addendum F): every pin is verified
against the tree THAT CARRIES EVERY HOST GOVERNANCE ACT. Step 1 proves
the host's `main` and origin agree before anything is derived.

## What the laptop prepared (inert)

| Item | Where | Identity |
|---|---|---|
| Candidate anchor | `policies/deployment/rsys6.proposed.json` | `795b364a…` when derived over the pushed tree at `b219ad5`+ with `rsys@5`'s ceiling and models; the host re-derives and must reproduce it |
| Themis contract | `policies/themis/contract.json` | pin `c6a95f49…`; `themis_commit` `b4dfaf0…` (Themis `feat/harness-integration`) |
| Harness at | `main` (the commit carrying this page) | L6 `1df0e285…`, L7 `008be050…` |
| Themis at | `feat/harness-integration` `b4dfaf0` (I-M3 `1414997` + the intake decoder fix and the three walls) | pins the harness at `v0.0.0-20260926130325-331d326a4172` |

The candidate differs from `rsys@5` in exactly: `deployment_version`
6; `constitution.state` `1df0e285…` (W-M1: `eventWriters` folded in);
`skill_catalog` `bc5625eb…` (catalog v2, decision-bound);
`themis_contract` `c6a95f49…` (was absent: the binary from `d977720`
refuses `rsys5.json` at parse — "themis_contract … nothing is
defaulted" — so `rsys@6` is mandatory, not optional); a third
workflow bundle and `remediate-dependency@4` in `skills` (the demo
method; @1 and @2 retained so nothing withdraws by omission).
`execution_ceiling` (`fd8fcdc1…`, unchanged since `rsys@2`) and
`models` are inherited from `rsys@5` and verified against the host in
step 5. Whether `skills` keeps @1/@2 is the owner's call at step 6; a
different list changes the bytes and the hash, and that is fine —
the hash is recorded, never predicted.

## 0. Identity of this run — record everything this prints

```bash
export REPO="$HOME/themis-ai-runtime" THEMIS="$HOME/themis" DEPLOY="/srv/themis/rsys"
export GIT_BIN="$(command -v git)"
date -u; uname -a; id -un; go version                     # expect: go1.25.x (Themis needs 1.25)
```

## 1. Both trees at the governed commits; no unpushed host act

```bash
cd "$REPO" && git fetch -q origin && git status --short && git rev-parse HEAD origin/main
# expect: clean; HEAD == origin/main, at or after the commit that carries this page
git log --oneline origin/main..HEAD                       # expect: empty — no unpushed host act
scripts/themis-status | sed -n '/Code identity/,/Deployment anchors/p'
# expect: l6 1df0e28548a4…  l7 008be050c297…

cd "$THEMIS" && git fetch -q origin && git checkout -q feat/harness-integration && git status --short
git rev-parse HEAD                                        # expect: b4dfaf0d02d3e4c02756b1acc2b8ad035a4ddc88
grep themis-ai-runtime go.mod                             # expect: …/src/harness v0.0.0-20260926130325-331d326a4172
```

## 2. Contract commit check (D-I-8) — the estate is at the pinned interface

```bash
cd "$REPO" && python3 - "$THEMIS" <<'PY'
import json, hashlib, subprocess, sys
c = json.load(open("policies/themis/contract.json")); T = sys.argv[1]
head = subprocess.check_output(["git", "-C", T, "rev-parse", "HEAD"], text=True).strip()
h = lambda p: hashlib.sha256(open(f"{T}/{p}", "rb").read()).hexdigest()
bad = {k: v for k, v in {
  "themis_commit": (c["themis_commit"], head),
  "governance_spec": (c["governance_spec_sha256"], h("api/governance.openapi.yaml")),
  "registry_spec":   (c["registry_spec_sha256"],   h("api/registry.openapi.yaml")),
}.items() if v[0] != v[1]}
print("CONTRACT OK" if not bad else f"CONTRACT MISMATCH {bad}")
PY
# expect: CONTRACT OK. A mismatch means the estate is not the interface the anchor will pin — stop.
```

## 3. The Themis estate on this host

Build, databases, and the pipeline follow `$THEMIS/INSTALLATION.md`
Part A §1–§4 (Registry `:8082`, Evidence `:8081`, Knowledge `:8085`,
Governance `:8083`, Communication `:8084` optional; all four pipeline
services share `THEMIS_BUS_DATABASE_DSN`). Governance applies
migration `000014_harness_commissions` on start with
`THEMIS_GOVERNANCE_MIGRATE=1`. Then:

```bash
cd "$THEMIS" && GOWORK=off make build && ls bin/           # expect: governance registry evidence knowledge … present
# start per INSTALLATION.md §4 (systemd per §7 for a durable estate); then:
ss -ltn 'sport = :8081 or sport = :8082 or sport = :8083 or sport = :8085'   # expect: 4× LISTEN
psql "$PGBASE/governance?sslmode=disable" -c "select version from schema_migrations;"   # expect: 14
```

### 3a. Three key holders (D-I-8, D-C-6) — values never printed, never recorded

```bash
export THEMIS_AUTH_DATABASE_DSN="$PGBASE/auth?sslmode=disable"
THEMIS_AUTH_MIGRATE=1 ./bin/authadmin create-key --name harness-read --scopes read
./bin/authadmin create-key --name operator --scopes product:<PID>      # PID: the demo product, step 4
./bin/authadmin create-key --name decider  --scopes admin
```

Record the three **key ids** only. Restart Governance and Registry
with `THEMIS_AUTH_DATABASE_DSN` and `THEMIS_AUTH_REQUIRED=1` exported
(INSTALLATION.md §4a): reads need any key, writes need a write-capable
key (`RequireWriteScope`). Wire the holders:

| Holder | Key | Where it lives | Acts |
|---|---|---|---|
| harness read door | `harness-read` (`read`) | `THEMIS_API_KEY_READ` in `themis-run`'s environment | `GET /findings/{id}` only; a write with it must return 403 |
| operator | `operator` (`product:<PID>`) | `THEMIS_API_KEY_WRITE` in `themis-intake`'s environment; the commission `curl` | commission, withdraw, raise proposal |
| decider | `decider` (`admin`) | the decider's shell only | `acceptProposal` |

Known gap kept visible (matrix row 14): `product:<id>` is
write-capable but not confined to that product's Findings. Separation
of duties here is operational, not a Governance invariant (D-I-6).

Negative twin, now: `curl -H "X-API-Key: <harness-read>" -X POST …/findings/x/commissions -d '{}'`
must answer **403**. The runtime's credential cannot commission.

## 4. A real Finding through the pipeline (UC1) — no manual row

The demo module is a real Go module with a real dependency that has
a real OSV record. Recommended: `golang.org/x/text@v0.3.7`
(CVE-2022-32149, OSV `GO-2022-1059`), fixed in `v0.3.8`. Keep the
mirror repo tiny: the walk edits `go.mod`.

```bash
mkdir -p "$DEPLOY"/{mirror,state,artifacts,provider,envelopes,submissions}
git init -q "$DEPLOY/mirror/demo-xtext-app" && cd "$DEPLOY/mirror/demo-xtext-app"
cat > go.mod <<'MOD'
module demo

go 1.22

require golang.org/x/text v0.3.7
MOD
git -c user.name=t -c user.email=t@t add . && git -c user.name=t -c user.email=t@t commit -q -m seed
export PINNED_SHA="$(git rev-parse HEAD)"; echo "$PINNED_SHA"      # record

# CycloneDX SBOM of the same module, one component, the purl OSV keys on
cat > /tmp/demo-sbom.json <<'SBOM'
{"bomFormat":"CycloneDX","specVersion":"1.5","version":1,
 "metadata":{"component":{"type":"application","name":"demo-xtext-app","version":"1.0.0"}},
 "components":[{"type":"library","name":"golang.org/x/text","version":"v0.3.7",
   "purl":"pkg:golang/golang.org/x/text@v0.3.7","bom-ref":"x-text"}]}
SBOM
sha256sum /tmp/demo-sbom.json                                      # record: SBOM hash (provenance)
cd "$THEMIS" && ./scripts/gf-upload-sbom.sh -f /tmp/demo-sbom.json -p demo-xtext-app -j demo -v 1.0.0
export RID=<printed release id>; export PID=<printed product id>
sleep 12
curl -s -H "X-API-Key: $THEMIS_API_KEY_READ" "localhost:8083/api/v1/releases/$RID/posture" | jq .
export FID=<finding_id>
curl -s -H "X-API-Key: $THEMIS_API_KEY_READ" "localhost:8083/api/v1/findings/$FID" | jq '{id,cve,stage,components:[.components[].purl],positions,commissions}'
# expect: id UUID, cve "CVE-2022-32149", stage "identified", the x/text purl, positions [], commissions []
```

If the host cannot reach `api.osv.dev`, start Knowledge with
`THEMIS_OSV_URL` pointing at a local stub that serves the real OSV
record for that purl, and write "feed source: local OSV stub of
GO-2022-1059" into the provenance. The Finding is still born through
Evidence → Knowledge → Governance; only the feed transport changed.
Record for provenance: SBOM hash, release id, product id, finding id,
feed source, `PINNED_SHA`.

## 5. Re-derive the candidate anchor on the host and prove it inert

```bash
cd "$REPO/src/harness" && cat > /tmp/rsys6check.go <<'GO'
package main
import ("encoding/json";"fmt";"os";"path/filepath"
 "github.com/tofchaliss/themis-ai-runtime/src/harness/deployment"
 "github.com/tofchaliss/themis-ai-runtime/src/harness/orchestration"
 "github.com/tofchaliss/themis-ai-runtime/src/harness/state")
func main(){
 repo, deploy := os.Args[1], os.Args[2]
 raw, _ := os.ReadFile(filepath.Join(repo, "policies/deployment/rsys6.proposed.json"))
 a, err := deployment.ParseAnchor(raw, "rsys6.proposed.json"); if err != nil { fmt.Println("REFUSED:", err); os.Exit(1) }
 var m map[string]any; _ = json.Unmarshal(raw, &m)
 h := func(p string) string { s, e := deployment.HashFile(filepath.Join(repo, p)); if e != nil { return e.Error() }; return s }
 d := func(p string) string { s, e := deployment.HashDir(filepath.Join(repo, p)); if e != nil { return e.Error() }; return s }
 ceiling, _ := deployment.HashFile(filepath.Join(deploy, "execution-ceiling.json"))
 want := map[string]string{
  "instruction_root_safety": d("instructions/global/safety"), "instruction_root_system": d("instructions/global/system"),
  "instruction_root_themis": d("instructions/themis"), "instruction_policy": h("policies/security/instruction-directive-patterns.json"),
  "tool_registry": h("policies/tools/registry-v5.json"), "skill_catalog": h("policies/skills/catalog.json"),
  "contract_registry": h("policies/verification/contracts.json"), "criteria_registry": h("policies/ratchet/criteria.json"),
  "regression_set_registry": h("policies/ratchet/regression-sets.json"), "delegation_template_registry": h("policies/delegation/registry.json"),
  "themis_contract": h("policies/themis/contract.json"), "execution_ceiling": ceiling,
 }
 bad := 0
 for k, v := range want { if m[k] != v { fmt.Printf("PIN MISMATCH %s: anchor %v tree %s\n", k, m[k], v); bad++ } }
 c := m["constitution"].(map[string]any)
 if c["state"] != state.ConstitutionHash() || c["orchestration"] != orchestration.ConstitutionHash() { fmt.Println("CONSTITUTION MISMATCH: anchor", c, "binary", state.ConstitutionHash(), orchestration.ConstitutionHash()); bad++ }
 if bad > 0 { os.Exit(1) }
 fmt.Println("PINS OK; candidate", a.Name, a.Deployment, a.SHA256)
 _, err = deployment.AdmitAnchor(filepath.Join(repo, "policies/deployment/rsys6.proposed.json"), a.SHA256, filepath.Join(repo, "policies/deployment/anchors.json"))
 fmt.Println("pre-activation admission:", err)
}
GO
go run /tmp/rsys6check.go "$REPO" "$DEPLOY"
# expect: PINS OK; candidate rsys 6 <sha>   then   pre-activation admission: … is not a Governance-registered deployment anchor
```

`PINS OK` on the host means the tree the anchor pins is the tree the
host carries (step 1) and the ceiling bytes the anchor inherits are
the host's. The `CONSTITUTION` line compares against the binary being
run **on the host** — the W-M4 host check. `<sha>` is the deployment
identity; `795b364a…` only if `skills` and both inherited fields are
exactly as prepared. **If admission does not refuse, stop**: the
proposal path is not inert and that is a defect.

Also confirm `scripts/themis-preflight` passes and prints
`themis_contract pin: c6a95f49…` and `THEMIS_API_KEY_READ is set`.

## 6. STOP — Governance act 1: `rsys@6` ACTIVE (owner)

Only the owner performs, in this order, each a commit on the host:

1. `cp policies/deployment/rsys6.proposed.json policies/deployment/rsys6.json` —
   the bytes are the identity; never reformat.
2. Write `policies/decisions/rel-anchor-rsys-6.json` (`kind:
   reliance`, target `rsys@6` with `<sha>`, `evidence: []`, rationale
   naming I-M5 and this page, `decided_at`, actor `commit:tofchaliss`),
   then `decision_sha256=$(sha256sum … | cut -d' ' -f1)`.
3. Append to `policies/deployment/anchors.proposed.json`
   `{"name":"rsys","version":6,"artifact_sha256":"<sha>","state":"active","steward":"security-engineering","decision_ref":"rel-anchor-rsys-6","decision_sha256":"<decision_sha256>"}`
   — `rsys@5` stays `active` here (supersession never withdraws).
   Re-run step 5: must still refuse (`anchors.json` unchanged).
4. `cp anchors.proposed.json anchors.json`; commit
   `Governance act: rsys@6 ACTIVE (I-M5)`.
5. Restart with `-anchor "$REPO/policies/deployment/rsys6.json"
   -anchor-sha256 <sha>`; re-run step 5: admission prints `<nil>`.

`rsys@5` is withdrawn only at step 10, after `rsys@6` has opened and
completed the demo execution (W-M4: "withdrawn only after `rsys@6`
ACTIVE and validated").

## 7. Commission (UC2) — the operator, before any execution (D-C-1)

```bash
CATALOG="$REPO/policies/skills/catalog.json"
COMP=$(python3 -c "import json;print([e for e in json.load(open('$CATALOG'))['entries'] if e['name']=='remediate-dependency' and e['version']==4][0]['composition_sha256'])")
export CID=$(curl -s -H "X-API-Key: $THEMIS_API_KEY_WRITE" -H content-type:application/json \
  "localhost:8083/api/v1/findings/$FID/commissions" \
  -d "{\"skill\":\"remediate-dependency@4\",\"composition_sha256\":\"$COMP\",\"anchor\":\"rsys@6\",\"artifact_sha256\":\"<sha>\",\"rationale\":\"I-M5 demo: remediate x/text\"}" | jq -r .commission_id)
echo "$CID"                                                     # expect: a UUID (Themis-minted)
curl -s -H "X-API-Key: $THEMIS_API_KEY_READ" "localhost:8083/api/v1/findings/$FID" | jq '{stage, commissions}'
# expect: stage still "identified" (D-C-4); commissions[0] state "open", commissioned_id "key:<operator key id>"
```

## 8. Execute under `rsys@6` (UC3)

```bash
cd "$REPO/src/harness" && go build -o "$DEPLOY/bin/themis-instantiate" ./cmd/themis-instantiate && go build -o "$DEPLOY/bin/themis-run" ./cmd/themis-run
"$DEPLOY/bin/themis-instantiate" -catalog "$CATALOG" -skill remediate-dependency@4 -task demo-remediate-0001 \
  -repo demo-xtext-app -pinned-sha "$PINNED_SHA" -model qwen2.5:7b -commission "$CID" \
  -registry "$REPO/policies/tools/registry-v5.json" -exec-ceiling "$DEPLOY/execution-ceiling.json" \
  -state "$DEPLOY/state" -artifacts "$DEPLOY/artifacts" -workspaces "$DEPLOY/provider" -out "$DEPLOY/envelopes/demo-0001" \
  -input finding="$FID" -input dependency="golang.org/x/text" -input advisory="CVE-2022-32149" -input target-version="v0.3.8"
# expect: envelope + effective grant/spec written; the commission id is in origin, not in the payload (D-C-5)
THEMIS_API_KEY_READ=<harness-read key> "$DEPLOY/bin/themis-run" -deploy "$DEPLOY" -governed-root "$REPO" \
  -anchor "$REPO/policies/deployment/rsys6.json" -anchor-sha256 <sha> -anchors-registry "$REPO/policies/deployment/anchors.json" \
  -envelope "$DEPLOY/envelopes/demo-0001/envelope-demo-remediate-0001.json" -git "$GIT_BIN" -json | tee /tmp/receipt.json
# expect: status COMPLETED, verdict VERIFIED, artifact sha256:<hex>, deployment <sha>
# A FAILED task is a governed terminal, not an error: record it and the reason; do not retry blindly.
```

The artifact-bound seq (tuple element 3):

```bash
cat > /tmp/seq.go <<'GO'
package main
import ("fmt";"os";"github.com/tofchaliss/themis-ai-runtime/src/harness/state")
func main(){ r,_ := state.OpenRoot(os.Args[1]); evs,_ := r.ReadEvents(os.Args[2])
 for _, e := range evs { if e.Class == state.EvArtifact { fmt.Println(e.Seq) } } }
GO
export SEQ=$(cd "$REPO/src/harness" && go run /tmp/seq.go "$DEPLOY/state" demo-remediate-0001); echo "$SEQ"
```

## 9. Intake (UC4), proposal (UC5), decision (UC6)

```bash
cd "$THEMIS" && GOWORK=off go build -o "$DEPLOY/bin/themis-intake" ./cmd/themis-intake   # GOWORK=off: the pinned module version travels into the evidence
"$DEPLOY/bin/themis-intake" -state-root "$DEPLOY/state" -anchors "$REPO/policies/deployment/anchors.json" \
  -contracts "$REPO/policies/verification/contracts.json" -anchor <sha> -task demo-remediate-0001 -seq "$SEQ" \
  -out /tmp/evidence-view.json
# expect: the evidence view — execution.commission_id == $CID, artifact.production_witness "l5-witnessed",
#         verification.reconstructed_outcome "PASS", reconstruction_consistent true, trust_class "inferred",
#         business_verification_refs contains the x/text purl and CVE-2022-32149; harness_module names …@v0.0.0-20260926130325-331d326a4172

THEMIS_API_KEY_WRITE=<operator key> "$DEPLOY/bin/themis-intake" -state-root "$DEPLOY/state" -anchors "$REPO/policies/deployment/anchors.json" \
  -contracts "$REPO/policies/verification/contracts.json" -anchor <sha> -task demo-remediate-0001 -seq "$SEQ" \
  -finding "$FID" -stance mitigated -rationale "x/text bumped to v0.3.8; report-valid@2 PASS reproduced"
# expect: proposal raised: <PROP> (finding …, commission $CID, trust inferred)
export PROP=<proposal id>

# Decider — a different human, a different key (D-I-6):
curl -s -o /dev/null -w '%{http_code}\n' -H "X-API-Key: <decider key>" -H content-type:application/json \
  -X POST "localhost:8083/api/v1/findings/$FID/proposals/$PROP/accept" -d '{}'          # expect: 204
curl -s -H "X-API-Key: $THEMIS_API_KEY_READ" "localhost:8083/api/v1/findings/$FID" \
  | jq '{stage, positions:[.positions[]|{version,stance,accepted_proposal_id,actor_id,actor_kind}], proposals:[.proposals[]|{id,status,evidence_trust,commission_id}]}'
# expect: positions[0].accepted_proposal_id == $PROP, actor_id "key:<decider key id>"; proposals[0].evidence_trust "inferred",
#         commission_id == $CID. No "dev:" anywhere in the Finding (three authenticated humans, D-I-8).
```

Hostile twins to run live, each must refuse with the named reason:

```bash
# UC5-neg: the operator asserts a different trust class → 400 (D-I-5)
# (edit the raised JSON by hand: POST …/proposals with the same evidence and "evidence_trust":"asserted")
# UC2-neg: the harness read key commissions → 403 (step 3a)
# UC4-neg: intake with a wrong seq / another anchor hash → REFUSED: <link-named>, exit 1, nothing raised
"$DEPLOY/bin/themis-intake" -state-root "$DEPLOY/state" -anchors "$REPO/policies/deployment/anchors.json" \
  -contracts "$REPO/policies/verification/contracts.json" -anchor <sha> -task demo-remediate-0001 -seq 1; echo "exit=$?"
```

UC8 (verify A, egress B) is proven by the Themis intake tests over
the `…-verify-a-egress-b` fixture; it is not reproduced on the host.

## 10. STOP — Governance act 2: `rsys@5` WITHDRAWN (owner)

Only after step 8 COMPLETED under `rsys@6` and step 9 recorded a
Position: set `rsys@5`'s entry to `"state":"withdrawn"` in
`anchors.proposed.json`, copy to `anchors.json`, commit
`Governance act: rsys@5 WITHDRAWN (superseded by rsys@6)`. Keep
`rsys5.json`. `rsys@3` stays as it is unless the owner decides
otherwise (it is a separate deployment identity, not superseded by
this act).

## 11. Evidence to bring back

```bash
cd "$THEMIS" && PGBASE="$PGBASE" THEMIS_API_KEY=<read key> ./scripts/vm-verify.sh "$RID" > /tmp/vm-verify-after.txt
cd "$REPO" && scripts/themis-status > /tmp/status-after.txt
tar czf "$HOME/evidence/themis-i-m5-$(date -u +%Y%m%d).tgz" "$DEPLOY/state" "$DEPLOY/artifacts" "$DEPLOY/envelopes" "$DEPLOY/submissions" \
  /tmp/receipt.json /tmp/evidence-view.json /tmp/vm-verify-*.txt /tmp/status-after.txt /tmp/demo-sbom.json
git -C "$REPO" format-patch -o "$HOME/evidence/acts" origin/main     # the Governance-act commits, oldest first
```

Run `vm-verify.sh` before step 4 as well (`/tmp/vm-verify-before.txt`).
Record: the three key **ids**; `PID`, `RID`, `FID`, `CID`, `PROP`,
task id, `SEQ`, artifact object id, `<sha>`; every refusal text from
the hostile twins; the exact `constitution` line from step 5.

On return, the laptop side: apply the patch bundle to `main` and push
on the owner's word; write `deployment-signoff-rsys.md` Addendum G
from the recorded identities; extend `vm-test-procedure-l8.md` with
the five-link rows (W-M4); flip matrix rows 2, 4, 5, 7–10, 14, 15 to
🟢 with this evidence; run the three reviews in worktrees; archive
`themis-v0`, `l5-witness-events`, `themis-integration`,
`themis-commissioning`, `l8-themis-surface`,
`l11-governance-promotion`.

## What this page does not establish

It mints nothing and decides nothing. A green run establishes that
one commissioned, L5-witnessed, reproducibly verified execution under
a Governance-registered deployment became the evidence of one human
decision. It does not establish that a model chose well: the artifact
stays model-authored and the evidence class stays `inferred`.
