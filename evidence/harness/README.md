# evidence/harness — Phase D evidence tooling

## What this is

The smallest program that can exercise `orchestration.Open` /
`SubmitTask` against a **real admitted Deployment Anchor**, so that a
concrete deployment instance can produce Phase D evidence
(`docs/development/deployment-test-plan.md`).

It opens under the real anchor, submits one governed task, and then
re-establishes the deployment from the durable record alone — D1
through D6.

## What this is NOT

**It is not production wiring, and must not be read as settling that
decision.** Runbook Step 12 — what process opens an orchestrator in
production, who may call it, how a submitter is authenticated — is an
open owner decision. This program exists because Phase D needs *a*
caller to produce evidence, and building one for evidence is cheaper
than prematurely fixing the production entry point.

Consequences of that boundary, deliberately:

- It lives in its **own module, outside `go.work`**, so
  `go build ./...` and CI never see it. Build with `GOWORK=off`.
- It has **no authority of its own**. Every refusal it reports comes
  from the harness under test. It validates nothing and admits nothing;
  it reports what happened.
- It takes the anchor hash as an **operator-supplied flag**, exactly as
  a real caller must. It never derives the hash from the anchor file —
  that would collapse the D-G1-1A two-step into self-certification.

## Usage

```bash
cd evidence/harness
GOWORK=off go build -o /tmp/themis-evidence .

/tmp/themis-evidence \
  -repo /opt/themis/themis-ai-runtime \
  -deploy /srv/themis/rsys \
  -anchor-sha256 <the operator's expected anchor hash> \
  -pinned-sha <40-hex commit in the mirrored repo> \
  -mirror-repo demo-vuln-app \
  -task rsys-d1
```

Flags with defaults: `-anchor-name rsys`, `-anchor-version 1`,
`-model qwen2.5:7b`, `-endpoint http://localhost:11434`,
`-git /usr/bin/git`, `-turn-timeout-sec 180`, `-wall-sec 300`,
`-payload-file` (empty uses the built-in brief).

`-model` must be in the anchor's allowlist and `-wall-sec` must be
within the ceiling, or assembly refuses — which is itself evidence.

## Scripted mode (`-scripted`)

```bash
/tmp/themis-evidence ... -scripted -score 0.82 -task rsys-e-base
/tmp/themis-evidence ... -scripted -score 0.91 -task rsys-e-cand
```

Drives the walk with a deterministic model instead of a live one.

**Why, for Phase E.** Phase E tests the *chain* — two comparable walks,
an L10 gate, witnessed L11 facts, cold reconstruction. A live model
introduces variance in the one thing not under test, and two live runs
cannot be relied upon to differ only in the score.
`integration/phasec_test.go` uses a scripted model for exactly this
reason. `-score` is what makes the two walks comparable: it lands in the
report as an extra field (the contract permits unknown fields) and is
what L11 derives a Δ from.

**It grants nothing.** The scripted model proposes tool calls like any
other model. L4 authorizes or refuses them, L10 grades the report
against `report-valid@1`, and the completion gate still requires PASS. A
script proposing an ungranted call is denied identically to a live one.

**Live mode remains the honest test of a model.** Two live runs with
`qwen2.5:7b` on CPU both reached governed terminals without ever calling
`write_file` — see the findings note below. Scripted mode is for
evidencing the chain, not for pretending a model can drive it.

## A finding worth knowing about grants

`orchestration.loop.toolDefs` offers the model **the phase's declared
capabilities**, without intersecting the task's grant — its comment says
"granted capability subset", but the code does not consult the grant.

That is not a security hole: L4 re-checks everything, and the denial is
correctly zero-detail (`not-available`, revealing nothing about whether
the tool exists). But it has a real cost. In run `rsys-d2` the model was
offered `list_directory` (declared by ANALYZE), called it, was denied
because the grant omitted it, and then stalled into no-action exhaustion.

**So a grant narrower than the workflow's declared capabilities spends
the model's turn budget on calls that could never have been authorized.**
This harness therefore grants every capability the phases declare. Worth
deciding whether `toolDefs` should intersect the grant, or whether the
comment should be corrected to match the code.

## Reading the result

**A non-COMPLETED status is still Phase D evidence.** What D3 requires
is a *typed terminal*, not success. A walk that reaches FAILED through
a declared edge has demonstrated governed behaviour; a walk that hangs,
or ends in an undeclared state, has not.

D4–D6 are the load-bearing checks: the record must carry the anchor
identity (not the sentinel `unanchored`), the anchor bytes must be
recoverable from the record, and `VerifyAnchorRecord` must re-establish
the deployment from record plus registry alone — with no help from the
running process.

## Related

- `docs/development/deployment-test-plan.md` — Phases A–F
- `docs/operations/deployment-runbook.md` — how the anchor was built
- `src/harness/integration/phasec_test.go` — the same chain against
  test-minted artifacts; this program is its real-anchor counterpart
