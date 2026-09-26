# `rsys@4` — host sequence (prepared 2026-09-23; NOT an activation)

> **Executed 2026-09-25.** `rsys@4` (`20637572…`) was activated, ran
> the Layer-8 evidence, and was withdrawn the same day: its
> `criteria_registry` pin came from origin's tree, not the host's
> governed one (the host had registered `walk-report-score-delta@1`
> in an unpushed act). `rsys@5` (`0dd1867f…`) is the re-mint over the
> host's tree and is ACTIVE. Record: `deployment-signoff-rsys.md`
> Addendum F. Lesson folded into step 3 below: verify the pins against
> the tree THAT CARRIES EVERY HOST GOVERNANCE ACT, never a tree that
> lacks one.

Governance decision (owner, 2026-09-23, Q3 of the L8 close): the four
registrations `report-valid@2`, `remediate-dependency@2`,
`dependency-triage@1`, and tool registry-v5 are **ratified as they
stand** (their files already read `active`; the act is the decision,
recorded in the L8 archive `tasks.md` §7). `rsys@4` is to be minted on
the deployment host from the current tree plus exactly two fields
taken from the ACTIVE `rsys@3`: `execution_ceiling` and `models`.
`skills` stays `["remediate-dependency@1","remediate-dependency@2"]`;
adding `investigate-cve@1` would change the authority surface and is a
separate deployment decision. **This page stops before the
hash-bound activation commit.** Steps 1–6 are preparation and proof;
step 7 is the Governance act and is not performed by this document.

Everything below runs on the host, from the governed checkout at the
commit that carries this file. `$REPO` is the governed root; `$DEPLOY`
the deployment root; `rsys3.json` the ACTIVE anchor file on the host.

## 1. Derive the proposed anchor from the tree + rsys@3

```bash
cd "$REPO"
python3 - <<'PY'
import json
p = json.load(open("policies/deployment/rsys4.proposed.json"))
r3 = json.load(open("policies/deployment/rsys3.json"))
p["execution_ceiling"] = r3["execution_ceiling"]   # deployment-supplied bytes, host-specific
p["models"] = r3["models"]                          # the admitted allowlist, unchanged
p["skills"] = ["remediate-dependency@1", "remediate-dependency@2"]
assert p["name"] == "rsys" and p["deployment_version"] == 4
json.dump(p, open("policies/deployment/rsys4.json", "w"), indent=1)
open("policies/deployment/rsys4.json", "a").write("\n")
PY
```

`rsys4.json` is now the candidate. Its bytes are the identity: do not
reformat it after this point.

## 2. Parse and schema validation

```bash
cat > /tmp/anchorcheck.go <<'GO'
package main
import ("fmt";"os";"github.com/tofchaliss/themis-ai-runtime/src/harness/deployment")
func main(){ b,_:=os.ReadFile(os.Args[1]); a,err:=deployment.ParseAnchor(b,os.Args[1])
 if err!=nil{fmt.Println("REFUSED:",err);os.Exit(1)}; fmt.Println("anchor ok:",a.Name,a.Deployment,a.SHA256) }
GO
cd "$REPO/src/harness" && go run /tmp/anchorcheck.go "$REPO/policies/deployment/rsys4.json"
```

Expected: `anchor ok: rsys 4 <sha>`. A refusal here means the tree
and the proposal disagree — stop and reconcile; never edit the
loader.

## 3. Verify every pinned artifact hash against the tree

```bash
cd "$REPO" && scripts/themis-status | grep '^PIN' | sort > /tmp/tree-pins.txt
python3 - <<'PY'
import json
a = json.load(open("policies/deployment/rsys4.json"))
pins = dict(l.split("\t")[1:3] for l in open("/tmp/tree-pins.txt").read().splitlines())
want = {
 "instruction_policy": a["instruction_policy"], "tool_registry": a["tool_registry"],
 "skill_catalog": a["skill_catalog"], "contract_registry": a["contract_registry"],
 "criteria_registry": a["criteria_registry"], "regression_set_registry": a["regression_set_registry"],
 "delegation_template_registry": a["delegation_template_registry"],
}
bad = {k: (v, pins.get(k)) for k, v in want.items() if pins.get(k) != v}
print("PINS OK" if not bad else f"PIN MISMATCH {bad}")
PY
```

Also confirm the instruction roots and constitution the same way
(`themis-status` prints `ROOT` and `CONSTITUTION` lines; each must
equal the anchor's field). `scripts/themis-preflight` must print
`PASS dependency-triage@1 (active): manifest and every pin verified`
and the `delegation_template_registry pin:` equal to the anchor field.

## 4. Verify the two inherited fields against rsys@3

```bash
cd "$REPO" && python3 - <<'PY'
import json, hashlib
a = json.load(open("policies/deployment/rsys4.json")); r3 = json.load(open("policies/deployment/rsys3.json"))
assert a["execution_ceiling"] == r3["execution_ceiling"], "ceiling differs from rsys@3"
assert a["models"] == r3["models"], "model allowlist differs from rsys@3"
h = hashlib.sha256(open("$DEPLOY/execution-ceiling.json".replace("$DEPLOY", __import__("os").environ["DEPLOY"]), "rb").read()).hexdigest()
assert h == a["execution_ceiling"], f"host ceiling bytes {h} are not the pinned ceiling"
print("rsys@3-derived fields verified against the host")
PY
```

## 5. Prove the candidate is INERT

```bash
cd "$REPO" && ANCHOR_SHA=$(shasum -a 256 policies/deployment/rsys4.json | cut -d' ' -f1); echo "$ANCHOR_SHA"
cat > /tmp/admitcheck.go <<'GO'
package main
import ("fmt";"os";"github.com/tofchaliss/themis-ai-runtime/src/harness/deployment")
func main(){ _,err:=deployment.AdmitAnchor(os.Args[1],os.Args[2],os.Args[3])
 fmt.Println("pre-activation admission:",err) }
GO
cd "$REPO/src/harness" && go run /tmp/admitcheck.go \
  "$REPO/policies/deployment/rsys4.json" "$ANCHOR_SHA" "$REPO/policies/deployment/anchors.json"
```

Expected: a refusal (`rsys@4` is not registered). **If this admits,
stop** — the proposal path is not inert and that is an implementation
defect, not something to proceed past.

## 6. Prepare the registration (still inert)

Append to `policies/deployment/anchors.proposed.json` — the ACTIVE
registry stays untouched:

```json
{"name": "rsys", "version": 4, "artifact_sha256": "<ANCHOR_SHA from step 5>", "state": "active", "steward": "security-engineering"}
```

Keep `rsys@3`'s entry as it is (a later deployment supersedes; it
does not withdraw). Re-run step 5: it must still refuse, because
`anchors.json` has not changed.

## 7. STOP — the Governance act (owner, on the host, not this page)

Only the owner performs: copy `anchors.proposed.json` over
`anchors.json`, commit `Governance act: rsys@4 ACTIVE`, restart the
service with `-anchor rsys4.json -anchor-sha256 $ANCHOR_SHA`, re-run
step 5 (must print `<nil>`), open the deployment, and record
`$ANCHOR_SHA` as the deployment identity in the signoff document.
`themis-run` now loads registry-v5 and wires the delegation seam over
`policies/delegation/registry.json`, whose root must be disjoint from
`$DEPLOY/{state,artifacts,provider,mirror}` (it refuses to open
otherwise).

## What this does not establish

Nothing here activates anything or proves a delegation on the host.
The first anchored delegation under `rsys@4` is the production
Register B/E run recorded in the L8 archive as owner-only.
