# `rsys@7` — host sequence for the phase-framing amendment (prepared 2026-09-27; NOT an activation)

Delta page over `rsys6-host-sequence.md`: the host already carries the
Themis estate, the three keys, the demo Finding, and `rsys@6` ACTIVE
(act `f9b2c80` on the host). `rsys@7` carries the L7 phase-framing
amendment (`openspec/changes/l7-phase-framing/`, D-P-1..6): the L1
phase rule, the derived `phase-state` slot, `remediate-dependency@5`.
It is the deployment under which I-M5 steps 7–11 resume. Same rules:
`$REPO=/opt/themis/themis-ai-runtime`, `$THEMIS=/opt/themis/themis-core`,
`$DEPLOY=/srv/themis/rsys`; the two Governance acts are the owner's;
the host cannot push.

| Item | Identity |
|---|---|
| Candidate anchor `rsys7.proposed.json` | `97599c96…` on the pushed tree with `rsys@6`'s ceiling and models; the host must reproduce it |
| Delta from `rsys@6` | `deployment_version` 7; `instruction_root_system` `2377bcd8…` (adds `phases.md`); `skill_catalog` `62a05ac5…`; third bundle's `context_contract` `28e91112…` (`@5` declares `phase-state`); `skills: [@5]` (owner, 2026-09-27: a clean allowlist — no entry that is a known Gather refusal) |
| Unchanged | both constitution pins; `themis_contract` `c6a95f49…`; registry-v5; ceiling; models |
| `remediate-dependency@5` composition | `12463c8f…` (the value the commission carries) |

`@1`..`@4` and `investigate-cve@1` stay in the catalog as history and
are not admitted: under D-P-1 their contracts lack `phase-state`, so
admitting them would admit known Gather refusals (owner, D-P-4
amendment). `rsys@6` stays the authoritative evidence source for the
two pre-amendment FAILED records; they are never rewritten or
reclassified because `rsys@7` fixes the framing.

## 1. Tree at the governed commit, on top of the host's act

```bash
cd $REPO && git pull -q --no-rebase && git log --oneline -4
# expect: the merge of origin/main (carrying P-M1/P-M2) above f9b2c80 "Governance act: rsys@6 ACTIVE (I-M5)"
git status --short                                        # expect: clean
scripts/themis-status | grep -E "l6 |l7 |rsys@6|remediate-dependency@5|rsys7"
# expect: l6 1df0e285…, l7 008be050…; rsys@6 active decision=rel-anchor-rsys-6; @5 active; rsys7.proposed.json inert
```

## 2. The D-P-5 gate — three captures under the test anchor (no governance effect)

```bash
cd $REPO/src/harness
for m in gpt-oss:20b qwen2.5:7b cyberpal20b-v3; do
  d=/tmp/cap5-${m%%:*}; rm -rf "$d"; mkdir -p "$d"
  LIVECAPTURE_DIR="$d" LIVECAPTURE_MODEL="$m" LIVECAPTURE_SKILL=remediate-dependency@5 \
    go test -count=1 -run 'TestLiveCapture$' -v ./integration/ 2>&1 | grep -E "result:|workflow-transition|l10-verification|artifact-bound" | cut -c1-160
done
```

Gate (D-P-5): the gpt-oss:20b run must print `Status:COMPLETED Verdict:VERIFIED Artifact:sha256:…`
with `l10-verification` PASS and `artifact-bound`. The other two are
compatibility evidence, whatever they print. Archive all three `/tmp/cap5-*`
directories into the evidence bundle. **If gpt-oss:20b does not
COMPLETE, stop here** and send the per-turn `{content, thinking,
tool_calls}` lines as before; do not mint.

## 3. Re-derive the candidate on the host and prove it inert

```bash
cd $REPO/src/harness && sed -n "/rsys6check.go <<'GO'/,/^GO$/p" ../../docs/operations/rsys6-host-sequence.md | sed '1d;$d' \
  | sed 's/rsys6.proposed.json/rsys7.proposed.json/g' >| /tmp/rsys7check.go
go run /tmp/rsys7check.go "$REPO" "$DEPLOY"
# expect: PINS OK; candidate rsys 7 97599c96…   then   pre-activation admission: … not a Governance-registered deployment anchor
```

## 4. STOP — Governance act 3: `rsys@7` ACTIVE (owner)

Same shape as act 1, with `7`, decision `rel-anchor-rsys-7`:

```bash
cd $REPO
export ANCHOR7=97599c9611b314ba3b07ae68a8df52e668263fd7947e275126806dbe4e145f58   # replace with the host's <sha> if skills differ
cp policies/deployment/rsys7.proposed.json policies/deployment/rsys7.json
cat >| policies/decisions/rel-anchor-rsys-7.json <<'JSON'
{
 "id": "rel-anchor-rsys-7",
 "kind": "reliance",
 "target": {
  "name": "rsys",
  "version": 7,
  "hash": "97599c9611b314ba3b07ae68a8df52e668263fd7947e275126806dbe4e145f58"
 },
 "evidence": [],
 "rationale": "rsys@7 minted on the host for the L7 phase-framing amendment (openspec/changes/l7-phase-framing, D-P-1..6): the L1 phase rule, the derived phase-state slot, skills [remediate-dependency@5] only — a clean allowlist: @1..@4 remain catalog history and cannot compose under D-P-1 (not a withdrawal). Everything else inherited from rsys@6. Gate: gpt-oss:20b COMPLETED remediate-dependency@5 under the test anchor on this host before the act.",
 "decided_at": "2026-09-27",
 "actor": {
  "kind": "commit",
  "id": "tofchaliss"
 }
}
JSON
export DEC7=$(sha256sum policies/decisions/rel-anchor-rsys-7.json | cut -d' ' -f1)
python3 - "$ANCHOR7" "$DEC7" <<'PY'
import json, sys
p = "policies/deployment/anchors.proposed.json"; d = json.load(open(p))
assert not any(e["name"] == "rsys" and e["version"] == 7 for e in d["entries"])
d["entries"].append({"name": "rsys", "version": 7, "artifact_sha256": sys.argv[1], "state": "active",
                     "steward": "security-engineering", "decision_ref": "rel-anchor-rsys-7", "decision_sha256": sys.argv[2]})
json.dump(d, open(p, "w"), indent=1); open(p, "a").write("\n")
PY
cp policies/deployment/anchors.proposed.json policies/deployment/anchors.json
git add policies/deployment/rsys7.json policies/deployment/anchors.json policies/deployment/anchors.proposed.json policies/decisions/rel-anchor-rsys-7.json
git commit -q -m "Governance act: rsys@7 ACTIVE (phase framing, I-M5)" && git log --oneline -1
cd src/harness && go run /tmp/rsys7check.go "$REPO" "$DEPLOY" | tail -1        # expect: <nil>
```

`rsys@6` stays active: it produced the evidence this amendment rests
on. Withdrawals of `rsys@5` and `rsys@6` are separate acts after a
Position exists under `rsys@7` (page 6 §10 shape).

## 5. I-M5 steps 7–11 under `rsys@7`, skill `@5`

Exactly `rsys6-host-sequence.md` §7–§11 with these substitutions:
`rsys@7` / `$ANCHOR7` / `rsys7.json`; skill `remediate-dependency@5`
(the commission's `composition_sha256` comes from the catalog entry for
version 5); task ids `demo-remediate-0003`…; `-model gpt-oss:20b`. A new
commission is required: the `rsys@6` commission names `rsys@6`'s hash
and `@4`, and correspondence is equality (D-C-5 §4).

```bash
COMP5=$(python3 -c "import json;print([e for e in json.load(open('$REPO/policies/skills/catalog.json'))['entries'] if e['name']=='remediate-dependency' and e['version']==5][0]['composition_sha256'])")
export CID7=$(curl -s -H "X-API-Key: $THEMIS_API_KEY_WRITE" -H content-type:application/json "localhost:8083/api/v1/findings/$FID/commissions" \
  -d "{\"skill\":\"remediate-dependency@5\",\"composition_sha256\":\"$COMP5\",\"anchor\":\"rsys@7\",\"artifact_sha256\":\"$ANCHOR7\",\"rationale\":\"I-M5 demo under the phase-framing amendment\"}" | jq -r .commission_id)
```

Then instantiate with `-skill remediate-dependency@5 -task demo-remediate-0003 -commission "$CID7" -model gpt-oss:20b`,
run with `-anchor "$REPO/policies/deployment/rsys7.json" -anchor-sha256 "$ANCHOR7"`,
and continue with the seq helper, intake, proposal, decision, twins and
the evidence bundle as written on page 6. The two `rsys@6` FAILED records
(`demo-remediate-0001/0002`) travel in the bundle as the finding's evidence.
