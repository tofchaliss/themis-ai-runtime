# Phase F — Sign-off Record: deployment instance `rsys`

Date: 2026-09-14. First concrete Deployment Instance validated against
the frozen architecture (L1–L11 + G1 + G2), following
`docs/development/deployment-test-plan.md`.

**What this record establishes:** that *this* deployed instance was
governed by the intended artifact set under real host conditions.
It does not re-establish the architecture, and it does not grant
production wiring — runbook Step 12 remains an open owner decision.

Raw evidence lives with the deployment, not here:
`$DEPLOY/phase-{a,b,c,d,e}-*.txt` under `/srv/themis/rsys/`.
Tooling: `evidence/harness/` (main, `phasec/`, `phasee/`).

---

## 0. Host

| | |
|---|---|
| OS / kernel | Ubuntu, Linux 6.8.0-38-generic, x86_64 |
| Capacity | 24 vCPU, 62 GB, 948 GB free on `/srv` |
| Inference | **CPU-only** — no NVIDIA/AMD GPU |
| Go | 1.24.0 linux/amd64 |
| **git** | **2.43.0 at `/usr/bin/git`** (the harness pins this absolute path) |
| Account | uid 10290, non-root; `/home` is autofs and quota-limited |
| Go caches | relocated to `/srv/themis/build/{gocache,gomodcache}` — see finding F-3 |
| Models | `qwen2.5:7b` (tool-calling verified), WhiteRabbitNeo 2.5 7B |
| Deployment root | `/srv/themis/rsys`, mode 700 |

## 1. Identities

| Artifact | Hash |
|---|---|
| Execution ceiling | `fd8fcdc19e23d2833e2efb5c5323d1b68b16b647826e4baf88d430e31b696f13` |
| **Anchor `rsys@1`** | `04fcdfaf609457d76b7be467cea91eded7318bf575d6c224e16ad586b8909b30` — **WITHDRAWN** |
| **Anchor `rsys@2`** | `666fd868f34f54fad02eef104a5f8b5fce0b4e7958d42da64b8764741c4f5918` — **ACTIVE** |
| Criteria registry (post-act) | `d739002c16d9286712b8f7230f632de9faf66de5ba0564f19a614f171d13d2a4` |
| Criterion `walk-report-score-delta@1` | `32b014cd53cd755b0c047c0096402c583b03f728efa331be821e6d73e544b2c9` |
| Mirror fixture `demo-vuln-app` | commit `a5b36b651d401f6c57467aa8cd789b24e1569f74` |

Repo SHA at Phase A: `dbe22616e4f2d4750d3eab317d80055f4526b4c2`.
Anchor authored at `0c5fcdc`. Governance acts recorded as local commits
on the deployment host (`Governance act: rsys@1 ACTIVE …`, then
`… walk-report-score-delta@1 registered; rsys@2 ACTIVE, rsys@1
withdrawn`). Later phases ran at the harness commits named in each
evidence file.

**`rsys@1` and `rsys@2` differ in exactly two fields:**
`deployment_version` and `criteria_registry`. The second anchor was not
optional — registering a criterion changed the registry bytes, which
changed the pin, which changed the deployment identity.

## 2. Phase results

| Phase | Result |
|---|---|
| **A** build/static | 21/21 green hermetic, 7.6 s, `A4 exit: 0`; skip audit exactly 10 lines (9 live proofs + darwin-only `immutable-flag`); all live proofs green individually — 45.4 s, 22.7 s, 19.2 s, 3.6 s |
| **B** artifact + admission | ceiling loads, `mirror_root` exists, 1 mirror present; anchor parses with two-way identity; **refused before the act**, `<nil>` after |
| **C** negative space | **13 of 19 proven** against the real anchor, 2 n/a with reasons, 4 uncovered |
| **D** anchored path | 5 governed walks, all typed terminals, D4–D6 on every one |
| **E** governed chain | **all six rows** |

### Gate B6/B8 — the one Phase B exists for

Before the Governance act:

> `deployment anchor admission refused: anchor 04fcdfaf6094 is not a
> Governance-registered deployment anchor — a matching hash is an
> identifier, never an admission claim`

After: `<nil>`. Identity is not admission; the two steps never collapsed.

## 3. Governed walks

| Task | Anchor | Model | Status | Terminal reached via |
|---|---|---|---|---|
| `rsys-d1` | `rsys@1` | live `qwen2.5:7b` | FAILED | `REMEDIATE/tool-error`, exhausted |
| `rsys-d2` | `rsys@1` | live `qwen2.5:7b` | FAILED | `ANALYZE/turn-no-action`, exhausted |
| `rsys-e-base` | `rsys@1` | scripted, score 0.82 | COMPLETED | gated edge, `report-valid@1` PASS |
| `rsys-e-cand` | `rsys@1` | scripted, score 0.91 | COMPLETED | gated edge, `report-valid@1` PASS |
| `rsys-v2-w1` | `rsys@2` | scripted, score 0.95 | COMPLETED | gated edge, `report-valid@1` PASS |

Every walk: record verdict `VERIFIED`, every transition carrying a
declared `edge_id` and its `cause_seq`, `deployment_anchor` recorded and
equal to the operator-supplied hash, anchor bytes durable in the record,
`VerifyAnchorRecord` re-establishing the deployment from record plus
registry alone.

**The two FAILED walks are evidence, not defects.** Phase D requires a
*typed terminal*, not success. Both failed through the workflow's own
declared exhaustion edges — "workflow-declared failure" — which is the
system ending where its definition says to.

### Supersession, proven both directions

Against the *same* registry in the *same* moment:

- `AdmitAnchor(rsys@1)` → refused: *"anchor rsys@1 is withdrawn — a
  superseded deployment definition cannot open"*
- `VerifyAnchorRecord(rsys@1)` → re-established all four of its records

Withdrawal closes the future and preserves the past. Had the
registration been replaced rather than appended, four completed governed
walks would have become permanently unattributable.

## 4. Phase C — refusal texts (the reason is the evidence)

| Row | Refusal |
|---|---|
| C1 | `anchor bytes do not match the operator's expected hash` |
| C2 | `a matching hash is an identifier, never an admission claim` |
| C3 | `anchor rsys@1 is withdrawn — a superseded deployment definition cannot open` |
| C6 | `system instruction root is not the anchored artifact (deployment rsys@2)` |
| C7 | `non-regular entry in a pinned tree — content pins admit regular files only` |
| C8 | `instruction policy is not the anchored artifact (deployment rsys@2)` |
| C11 | `the anchor declares no model registry but one is configured (deployment rsys@2)` |
| C12 | `model "gpt-4o" is not in the anchored allowlist — a model enters a deployment only by Governance act` |
| C13 | `the task's execution ceiling is not the deployment's ceiling — the ceiling is supplied at Open, never chosen per task` |
| C14 | `workflow is not in the anchored workflow set (deployment rsys@2)` |
| C16 | `tool registry is not the anchored artifact — a mutually consistent bundle is not a governed bundle` |
| C18 | `no deployment anchor configured and Unanchored not explicitly set — a deployment governs by anchor or refuses to open (G1)` |
| C19 | `Unanchored declared alongside a deployment anchor — the caller role is ambiguous` |

Also observed at L11: `never-registered@1 is not registered —
unregistered artifacts are data and measure nothing`.

Every row corrupted a **byte-identical copy** of the pinned trees in its
own scratch state root; the live deployment and its record plane were
never written (AGENTS.md probe isolation). The anchor pins content
hashes rather than paths, which is what makes copies admit identically
and the refusals genuine.

**Not claimed:**

| Row | Why |
|---|---|
| C10 | n/a — this anchor declares `model_registry: absent`; C11 is its applicable form |
| C15 | n/a — tests bundle *indivisibility* and needs two anchored bundles; `rsys@2` has one. Substituting an unanchored ceiling was refused by the **workflow loader** before the bundle check ran; recording that would have claimed a control that never executed |
| C4, C5 | need two Opens with a persisted observed-registry between them |
| C9 | needs a binary rebuilt with a different constitution |
| C17 | needs a skill-attributed envelope |

## 5. Phase E — the governed chain

| Row | Result |
|---|---|
| E3 | both facts grounded; witness = each walk's own `l4-audit`, seq 15 |
| E4 | admission observed at the real `l9-catalog` door: `remediate-dependency@1` artifact `fd59500a97de` |
| E5 | `score_delta` 0.09; run identities **derived** `[task:rsys-e-base task:rsys-e-cand]`; relation `better-under-k` |
| E6 | cold reconstruction **confirmed** |
| E7 | catalog `eb66b311ced2` byte-identical after the chain |
| E8 | model-turn object **REFUSED**: `provenance-violation` |

Two cross-checks neither tool was told to make line up:

- E4's `fd59500a97de` **is** `remediate-dependency@1`'s
  `composition_sha256` in the catalog.
- E7's `eb66b311ced2` **is** the `skill_catalog` pin in the anchor.

So the door observed is the *anchored* door: deployment identity and
comparison provenance refer to the same bytes.

**E8 is the load-bearing one.** A `model-turn` object from the *same
genuine walk* that produced the accepted report, offered under the *same
selector*, was refused. The model's own words — durably recorded, in a
verified record, inside a history whose tool results were accepted as
facts — still cannot become a fact.

The criterion was **resolved from the Governance registry**, and the
registry hash recorded in the package is that same file's hash. The
package's provenance is therefore true of `rsys` specifically, not of a
fixture.

## 6. Findings and dispositions

| # | Finding | Cat. | Disposition |
|---|---|---|---|
| F-1 | `orchestration.loop.toolDefs` offers the phase's declared capabilities **without intersecting the grant**, though its comment says "granted capability subset". L4 re-checks and denies zero-detail, so not a security hole — but the model spends turns and counters on calls that could never be authorized, which exhausted `rsys-d2` | 2 | **CLOSED 2026-09-14** — `toolDefs` now intersects the grant, making the code match its own comment. Narrowing only; L4 authority unchanged. Pinned by `TestToolDefsIntersectGrant`, mutation-verified against the pre-fix behaviour |
| F-2 | No registered criterion could consume governed-walk evidence (`bench-score-delta@1` selects the benchmark plane) | 1 | **CLOSED** — `walk-report-score-delta@1` registered by Governance act; forced `rsys@2` |
| F-3 | Go caches default into a quota-limited autofs `/home`; exhaustion does **not** fail cleanly — unbuildable packages simply do not run, tests never report, shell exits 0 | 1 | **CLOSED** — relocated to `/srv`; recorded as runbook Step 1 provisioning requirement |
| F-4 | Runbook Steps 5/8/9 used `go run - <<'GO'`; `go run` does not read a program from stdin, so all three verification commands failed as written | 2 | **CLOSED** — rewritten to a file; each corrected form executed |
| F-5 | Test plan B2 claimed `LoadCeiling` checks `mirror_root` is "real"; it requires **absolute**, not existent. A mistyped mirror pins, admits, and fails at the first governed task | 2 | **CLOSED** — wording corrected; `themis-status` checks existence separately |
| F-6 | `themis-status` detected mirrors by `*.git` suffix, reporting "no source" for a valid mirror — the naming convention the runbook had already abandoned | 2 | **CLOSED** — detects by repository shape |
| F-7 | `verification-guard` could not be cleared by the env-prefixed `go test` its own documentation prescribes | 2 | **CLOSED** — clear path widened; mark/check untouched |
| F-8 | A proposed anchors registration containing only the new entry would have **deleted** `local-dev@1` on copy | — | **CLOSED before damage** — caught by a post-act readback gate, not by the human step; registry now carries all three entries |
| F-9 | `qwen2.5:7b` on CPU could not drive `remediate-dependency` in two attempts — never called `write_file`; once tried three times to verify a report it never wrote, once invented an ungranted tool and stalled | 1 | **RECORDED** — model-selection evidence, not a harness defect. Phase E used a scripted model, as `phasec_test.go` does |
| F-10 | C15 unexercisable with a one-bundle anchor | — | **RECORDED** as a coverage gap; needs a two-bundle anchor |

## 7. Residuals confirmed still recorded

- **Submitter authentication** (G1, explicit) — origin is recorded,
  never treated as Governance authority.
- **Sensitivity inheritance** (local-endpoint scope).
- **The four consumption-pinned registries** whose consumers live
  outside L7 — verified by each plane's own consumer, not at Open.
- **Live-proof concurrency** — nine live proofs across nine parallel
  packages contend for one model server; run them separately. Not a
  host-capacity property (worse on the bigger host).
- **Repository-level evidence sweep (2026-09-14)** — seven gaps closed
  in L5/L7/L10 archives; one known-unmeasured class remains (a control
  both untested *and* correctly cited), closable only by a systematic
  mutation pass, not run and not decided.

## 8. What this does not establish

- **Production wiring.** `evidence/harness/` is evidence tooling with no
  authority; runbook Step 12 is untouched and remains the owner's.
- **Deployment confidence in general.** This record covers `rsys` on one
  host. Another instance is another deployment identity and its own
  evidence.
- **Model suitability.** F-9 is a fact about `qwen2.5:7b` on CPU, not a
  selection decision. `themis-bench` is the instrument for that.
- **The four uncovered C rows** (C4, C5, C9, C17) and C15.

## 9. Sign-off

Phases A, B, D and E complete; C at 13 of 19 with every omission stated.
No category-4 finding: nothing here reopened a locked architectural
decision. All ten findings are closed or recorded; none remain open.

Owner acceptance: _pending_.
