# G1 anchor pin sweep — completeness, 2026-09-15

The 2026-09-14 mutation pass surfaced `orchestration/orchestrator.go:474`
(the anchored tool-registry pin) as a survivor. Closing it alone would
have repeated the ranking mistake that produced it: the Tier-1 list
ranked LINES, and a line is the wrong unit here. The deployment anchor
is one surface — the highest-authority one in the system — and its pins
are enforced in three different places by three different mechanisms.

This sweep takes the whole surface at once, with a completeness
criterion that can be checked rather than asserted: **every field of the
`Anchor` struct that L7 enforces has a drift test, and each test fails
when its own guard alone is suppressed.**

## The surface

| Anchor field | Enforced at | Before | Now |
|---|---|---|---|
| `instruction_root_safety/system/themis` | `verifyAnchoredInstructionPlane` (one loop) | covered | covered |
| `instruction_policy` | same | covered | covered |
| `constitution.state` / `.orchestration` | same | closed 2026-09-15 | covered |
| `model_registry` — hash mismatch | same | covered | covered |
| `model_registry` — pinned, none configured | same | **UNCOVERED** | closed |
| `model_registry` — absent, one configured | same | **UNCOVERED** | closed |
| `execution_ceiling` | `Open` | closed 2026-09-15 | covered |
| `tool_registry` | `SubmitTask` | **UNCOVERED** | closed |
| `workflows` (bundle set) | `SubmitTask` | covered | covered |
| `models` (allowlist) | `SubmitTask` | covered | covered |
| `skill_catalog` — hash mismatch | `verifyAnchoredSkill` | covered | covered |
| `skill_catalog` — none configured | same | **UNCOVERED** | closed |
| registry admission + record identity | `deployment/anchor.go` | **UNCOVERED (4)** | closed |
| `contract_registry`, `criteria_registry`, `regression_set_registry` | *not L7* | recorded residual | recorded residual |

## What was open, and why each matters

**`tool_registry` (`orchestrator.go:474`) — the widest.** The registry
decides what every tool IS: its parameters, its target kind, whether it
mutates, whether it is verifier-eligible. A deployment running under an
unpinned registry has no stable meaning for any capability it grants,
so every other pin is being enforced over an undefined vocabulary. The
test satisfies every other pin and moves only this one.

**`model_registry`, both configuration branches.** The allowlist governs
NAMES; this pin governs what those names RESOLVE TO — runtime, endpoint,
credential env. They are two controls, and pinning one does not pin the
other. The mismatch branch was covered; the two configuration branches
were not, and they are the ones that fail OPEN: an anchor pinning a
registry while the deployment supplies none would resolve names by local
default, which is precisely what the explicit `"absent"` declaration
exists to prevent being implicit.

**`skill_catalog` with none configured.** A skill-attributed task under
an anchored deployment resolves its composition from the ANCHORED
catalog. With no catalog there is no anchored authority to resolve
against, and the failure must be a refusal rather than a fallback to the
envelope's own claim — the substitution the anchored-catalog rule exists
to prevent. The test supplies a genuine composition commitment first,
because the envelope validator refuses skill keys without one and that
refusal would otherwise stand in for the catalog check.

**`VerifyAnchorRecord`'s three entry guards.** Mutual cover, three ways.
Subtests existed for "unanchored" and "missing bytes", and all three
guards survived anyway: suppress the unanchored check and `"unanchored"`
fails the sha-syntax check; suppress the syntax check and nothing else
reaches it; suppress the missing-bytes check and `hashBytes(nil)` fails
the identity comparison. Every case was caught by a neighbour.

The fix is not cosmetic. "This run declared itself unanchored", "this
record is malformed", and "the anchor bytes are missing" are three
different facts about a record, and an auditor acts differently on each.
The subtests now assert WHICH refusal, plus one new case — a malformed
identity carried with well-formed bytes — that only the syntax guard
can catch.

**Registry-vs-anchor two-way identity (`anchor.go:578`).** The same
shape as the L9 manifest-vs-registration check closed earlier today, one
layer up. The registry says which anchor it admitted by ARTIFACT HASH;
the anchor says what it is. A registration whose name or version
disagrees with the bytes it admits leaves the record unable to say which
deployment governed the task — registry naming one, bytes naming
another. Covered now for both name and version disagreement.

## Verification

All eight guards mutation-verified in a disposable worktree (AGENTS.md
probe isolation). Each was suppressed individually and failed **only**
its own subtest — the check that matters here, since four of the eight
were survivors precisely because a neighbour stood in for them.

Full module green: build, vet, gofmt, 22 packages.

## Out of scope, deliberately

The three consumption-pinned registries (`contract_registry`,
`criteria_registry`, `regression_set_registry`) are declared in the
anchor and syntax-checked at parse, but L7 never compares them against
an artifact — each is verified by its own plane's consumer. That is a
recorded, owner-ratified residual ("consumption-pin wall shape",
RATIFIED at the L11 close), not a finding, and re-litigating it from a
mutation result would be the wrong way to reopen it.

## What this does not establish

That the anchor pins the right set of artifacts. This sweep proves every
pinned field is enforced and every enforcement is evidenced; whether the
field list is COMPLETE — whether some artifact that can change an
anchored deployment's behaviour or authority is absent from the anchor —
is the owner-finding-4 question, and it is a governance judgement, not
something a test can answer.
