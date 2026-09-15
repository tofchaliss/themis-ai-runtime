# Systematic mutation pass — refusal suppression (2026-09-14)

Closes the one class every other method used that day structurally could
not reach: **a control that is untested AND correctly cited.** The
archive sweep finds dead citations. The deployment finds controls that
fire. Neither can find a guard that is correct, referenced, and never
exercised.

Tool: `evidence/harness/mutate`. Operator: for every
`if <cond> { … return <error> … }`, replace the condition with `false`
and run that package's tests. If nothing fails, that control is not
covered by its package's tests.

Run on the deployment host, 20m35s, in an isolated worktree.

## Result

| | |
|---|---|
| Refusal guards found | 971 |
| Killed (a test failed) | 433 |
| Did not compile | 130 — cannot ship, not a gap |
| **Survived** | **408** |

### Survivors by category

| Category | Count | Reading |
|---|---|---|
| I/O & parse error propagation (`if err != nil`) | 191 | error plumbing; reachable only with filesystem/serialisation fault injection. `state/` has fault points for this; most packages do not. A coverage observation, not a security finding. |
| Semantic controls | 157 | the real output |
| Schema / format validation | 60 | genuinely untested; mostly low severity because malformed input usually fails elsewhere too |

Of the 157 semantic controls, 115 are in security-relevant packages.

## Two corrections to that number

**`confine` (10 guards) is a tool false positive.** The package has no
`_test.go` of its own — its tests live in `context/confine_test.go` — so
`go test ./confine/` reports "no test files" and exits 0, making every
mutant survive trivially. Fixed in the tool (commit `32b3b58`); those
controls are tested. Same for `benchmarks/cmd/themis-bench`.

**"Survived" means "not covered by its own package's unit tests",**
which is narrower than "untested". Several survivors are proven by the
Phase C matrix, which lives outside the module in `evidence/harness`:
`orchestrator.go:168` (C19), `:474` (C16). The converse also holds — a
survivor that Phase C does not cover either is genuinely unguarded.

## Tier 1 — where a disabled guard permits something promised against

| Control | Exposure |
|---|---|
| `orchestration/orchestrator.go:763` — grant digest changed between attribution and execution | TOCTOU on authority itself |
| `orchestration/orchestrator.go:547` — artifact changed between loading and durable capture | TOCTOU on the governed record |
| `orchestration/orchestrator.go:215` — supplied ceiling ≠ anchored ceiling **at Open** | G1. The C matrix covers only the SubmitTask side (C13) |
| `orchestration/orchestrator.go:372` — **L7** constitution pin | C9 exercised the **L6** pin at `:369`; this is a separate line |
| `state/task.go:211` — a record may reference only already-durable objects | record-before-effect |
| `execution/local.go:243` — HEAD ≠ pinned SHA post-condition | workspace could sit at the wrong commit |
| `skills/catalog.go:193` — manifest self-declaration ≠ registration | L9 two-way identity |
| `skills/catalog.go:85` — manifest_path traversal (`..`, absolute) | path containment |
| `orchestration/loop.go:474` — verification event undeclared yet produced | L7 invariant |
| `orchestration/loop.go:382` — verifier-eligible call with no evaluator wired | fail-closed |
| `ratchet/package.go:129` — criterion bytes ≠ constituent conditioning tuple | L11 binding |
| `verification/contract.go:249/257` — provenance completeness | "not contract-relaxable" |

### One that deserves separate attention

**`execution/local.go:153`** — endpoint refusal (`://`, `::`).
`TestEndpointRefusal` exists and is cited in the L5 traceability, yet
this guard survives. The likely cause is that `refuseEndpoints` carries
several conditions and every test case is caught by a *different* one,
leaving this branch unexercised.

That is the same shape as Phase C row C15: a test that passes while the
specific control beneath it never fires. It is worth confirming, because
if correct it means a cited, passing test does not exercise the line it
is cited for — the exact defect this pass exists to find, occurring in a
control the L5 archive lists as evidenced.

## What this pass does not establish

- **It does not say 408 controls are broken.** Every one of them is
  present and correct in the source; the finding is about coverage.
- **It does not rank by exploitability.** Tier 1 above is a reading, not
  a computation. Some entries may prove to be equivalent mutants.
- **The operator is narrow by design.** It suppresses refusal guards. It
  does not mutate arithmetic, boundaries, or control flow, so a clean
  result here would not mean the suite is complete.

## Disposition

Open. The Tier-1 list is a work item, not something to close in one
sitting; the ~190 error-propagation survivors are a coverage map and
triaging them wholesale would be poor use of attention.

Recorded rather than fixed, deliberately: which of these deserve tests is
an owner judgement about where evidence is worth buying.
