# L7 Amendment: two cited-but-absent tests written (2026-09-14)

**Nature: evidence-record correction, not an architecture change.** No
L7 decision is amended. D-L7-1, D-L7-3/4, Q-L7-9 and Q-L7-10 stand as
archived, and both controls were present and correct in the code
throughout. What was missing was any test exercising them.

## What was wrong

Two traceability rows cited test names that do not exist:

| Row | Control | Enforced at | Cited as |
| --- | --- | --- | --- |
| Envelope is the sole task input … payload capped 64KiB (D-L7-1, Q-L7-10) | 64KiB envelope payload cap | `orchestration/envelope.go` | `TestPayloadCap` |
| Static totality + boundedness at load … finite worst-case walk ≤ ceiling (D-L7-3/4, Q-L7-9) | worst-case walk bound vs ceiling | `orchestration/workflow.go` | `TestWorstCaseWalkBound` |

Neither name existed anywhere in the module. Verified by mutation in a
disposable worktree: with `if len(e.Payload) > 64<<10` and
`if w.WorstCaseLen > ceiling.MaxWalkLength` each replaced by `if
false`, **the entire module suite stayed green** — both controls could
be deleted without any test noticing.

The rows' other citations (`TestEnvelopeNoDefaulting`,
`TestLoaderRefusals` → now `TestWorkflowLoaderFailsClosed`) are real
and cover the rest of each row; only the bounded-ness clause of each
was unevidenced.

## How it was found

A cross-check of every test name cited across all archive
traceability files (312 distinct names) against the tests that
actually exist (447). Ten citations were dead. Most are renames
(`TestDuplicateSubmit` → `TestDuplicateSubmitRefused`,
`TestStartupSweep` → `TestStartupSweepHermetic`,
`TestEmptyEvidenceGradesUnavailable` → `…GradesThroughCanonicalization`,
recorded at the L10 close), and `TestUndeclaredEventInvariant` is
covered under `TestStepInvariantBranches`. These two were real holes.

The sweep followed the L5 group-kill evidence correction
(`../../2026-09-07-layer-05-execution-environment/amendments/group-kill-evidence/`),
which raised the question of what else was claimed but unproven.

## What the evidence is now

`TestPayloadCap` — a payload of exactly 64KiB loads; 64KiB+1 refuses
typed (`ErrEnvelope`) and names the cap. The at-cap case matters: a
refusal alone could come from any other envelope rule and still look
like the cap working.

`TestWorstCaseWalkBound` — the computed bound is read back from a
permissive load rather than hardcoded, then asserted from both sides:
a ceiling of exactly `WorstCaseLen` loads (the bound is `>`, not
`>=`), and `WorstCaseLen-1` refuses typed naming both numbers. Reading
the bound back means the assertion follows the fixture instead of
drifting away from it — the failure mode that produced this gap.

Mutation-verified in a disposable git worktree (AGENTS.md probe
isolation); both previously-surviving mutants are now killed:

| Mutant | Before | After |
| --- | --- | --- |
| payload cap → `if false` | survived full suite | FAIL `TestPayloadCap` |
| worst-case ceiling refusal → `if false` | survived full suite | FAIL `TestWorstCaseWalkBound` (worst case 40, ceiling 39) |

## Scope

Nothing weakened; no control changed; no decision reopened. Two rows
that asserted more than their evidence supported are now backed by
tests that distinguish what they claim. Full module suite green.
