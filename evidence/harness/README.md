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
`-git /usr/bin/git`, `-turn-timeout-sec 180`, `-wall-sec 300`.

`-model` must be in the anchor's allowlist and `-wall-sec` must be
within the ceiling, or assembly refuses — which is itself evidence.

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
