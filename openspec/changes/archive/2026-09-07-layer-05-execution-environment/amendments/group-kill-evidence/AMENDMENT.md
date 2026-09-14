# L5 Amendment: group-kill evidence corrected (2026-09-14)

**Nature: evidence-record correction, not an architecture change.** No
L5 decision, declaration, or control is amended. D-L5-9, Q-L5-2/6, the
`TerminationGroupKill` declaration (design.md §3 item 5) and the
subprocess-tier termination guarantee all stand exactly as archived.
What changes is the *evidence* claimed for one traceability row, which
was overstated at archive.

## What the archive claimed

traceability.md row "Envelope wall-clock budget enforced … per-op group
kill (M1 MED-3, Q-L5-2)" cited `TestExecTimeoutGroupKill` as evidence
for the **group** kill.

## What that test actually proved

Two gaps, both found 2026-09-14 when CI was repaired and run on Linux
for the first time since 2026-09-06:

1. **It did not prove group semantics on any platform.** The test ran
   one `git log` process. A single process with no children makes
   `syscall.Kill(-pid, SIGKILL)` and `cmd.Process.Kill()`
   indistinguishable, so the row's distinguishing claim — the whole
   execution-owned group, not merely the direct child (Q-L5-2/6,
   design.md §3 item 143) — had no test behind it.

2. **On Linux it did not reach the timeout branch at all.** The test
   used a fixed 1ms deadline, commented "a deadline shorter than
   process spawn". Measured: `git log` takes 6.2–7.4ms on the darwin
   development host, so the deadline always fired there; on the Linux
   CI runner git finishes in under 1ms, so `cmd.Wait()` won the select
   and the call simply succeeded. The test asserted a typed timeout and
   received `<nil>`.

The control itself was never defective — `runGit`'s select/kill path
(local.go) is platform-neutral and reads correct. The defect was in the
evidence: a host-timing assumption standing in for a proof.

**Why this went unnoticed for a week:** CI was red from 2026-09-06
through 2026-09-13 for unrelated reasons (a gofmt failure, then a build
step pointing at the decommissioned `themis-serve`). The L5–L11 close
evidence was therefore darwin-local throughout; the Linux failure was
present but unread.

## What the evidence is now

`spawnOverride` (execution/local.go) — the deterministic
termination-proof seam, in the shape of the L6/L7 fault seams
(state/store.go, orchestration/loop.go). Nil in production; a test sets
it to substitute the child for one invocation. It changes only WHICH
binary runs: deadline computation, process group, environment
allowlist, endpoint refusal, audit record and group kill remain the
production path, so it cannot reach git with arguments the caller
vocabulary forbids (M1 MED-2) and it is not a second execution path.

`TestExecTimeoutGroupKill` now substitutes a child that forks a
**grandchild** which would create a marker after the deadline, and
asserts: the error is a typed timeout, the call is bounded, the audited
outcome is `timeout`, and the marker never appears — killing only the
direct child would leave the grandchild alive to create it.

Mutation-verified in a disposable git worktree (AGENTS.md probe
isolation):

| Mutant | Result |
| --- | --- |
| `syscall.Kill(-pid, SIGKILL)` → `cmd.Process.Kill()` | FAIL at 30.01s — the call is unbounded |
| grandchild writes the marker before the kill | FAIL on the marker assertion |

Both assertions are load-bearing. The first mutant also shows the group
kill is what *bounds the call at all*: the surviving grandchild holds
the inherited stdout pipe, so `cmd.Wait()` blocks until it exits.

## Scope

Unchanged: the Tier-0 cooperative-termination amendment (Q-L5-2), the
per-tier declaration residual (design.md §3 item 223 item 5), and every
other L5 invariant and its evidence. Nothing is weakened; one row's
claim is now backed by a test that distinguishes what it asserts.

Verified at amendment: `go build ./...`, `go vet ./...`, `gofmt -l .`
clean; `./execution` green (104.8s, live proofs included); CI green on
Linux — run 34797631381, the first green run since 2026-09-06.
