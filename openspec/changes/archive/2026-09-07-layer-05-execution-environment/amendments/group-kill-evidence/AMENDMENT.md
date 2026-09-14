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

---

# Addendum (2026-09-14): two further L5 evidence gaps closed

The archive sweep that followed this correction surfaced two more L5
rows whose evidence did not discriminate what they claimed. Same
character as the above: controls sound, evidence thin. No L5 decision
is amended.

## 1. Teardown anomaly had no Linux evidence

Row: *"Teardown unconditional …; DESTROYED only verified; anomaly
typed, never false success (Q-L5-12, M1 MED-5)"*. The clause "anomaly
typed, never false success" rested solely on `TestTeardownAnomalous`,
whose fixture used darwin `chflags uchg` and **skipped on Linux** —
so on the platform deployments run on, nothing proved that an
unverifiable teardown is typed rather than reported as DESTROYED.

`TestTeardownAnomalous` is now two fixtures:

- **`unremovable-parent` (portable, primary)** — the environment's
  parent directory is made non-writable, so unlinking `baseDir` itself
  fails. `teardown`'s permission-restore walk is scoped to `baseDir`
  and never chmods its parent, so unlike a 0555 directory *inside* the
  workspace this survives the walk on every platform.
- **`immutable-flag` (darwin)** — the original mechanism, retained: the
  immutable flag is a genuinely different failure, surviving the
  restore walk rather than side-stepping it.

Both are permission/flag based and so skip under root, following the
convention already used by the egress fixtures.

## 2. The observed-RSS unit conversion was undiscriminated

`local.go` multiplies `ru_maxrss` by 1024 on Linux, because getrusage
reports bytes on darwin and **kilobytes** on Linux. Nothing tested that
branch. `TestEgressMemObservedGate` cannot: its bound is
`mem_bytes: 1`, which every observation breaches, so a missing
conversion (1024× low) or a doubled one (1024× high) passed
identically.

This matters for an honesty claim, not an enforcement one. L5 refused
"the 'we support memory limits' lie" in writing (design.md §3 item
183); an *observation* wrong by three orders of magnitude is the same
kind of dishonesty in the other direction.

`TestObservedRSSIsInBytes` asserts the recorded peak lies in
[1 MiB, 1 GiB] — deliberately loose bounds that make no claim about
git's footprint, chosen as the widest window that still separates
bytes from kilobytes.

## 3. The mid-drain budget proof rode the same timing assumption

Row: *"Envelope wall-clock budget enforced: pre-exec exhaustion and
mid-op drain both seal `env-deadline`"*. `TestBudgetMidDrainAutoSeals`
set the remaining budget to 1ms and ran real `git log`, assuming git
would be slower — the same assumption that made
`TestExecTimeoutGroupKill` fail on Linux. It survived there only
because `time.Since(start)` also counts `cmd.Start()`, leaving a margin
of one fork/exec that nothing designed. Had git won that race the op
would have SUCCEEDED and the test's first assertion would have failed.

It now drives the drain through `spawnOverride` with a child outliving
the budget 100×, making the drain a consequence of the budget — which
is what the row claims — rather than of host process-spawn latency,
which it does not.

The test also now pins **which** branch sealed. Pre-exec exhaustion
(`TestBudgetExhaustionSeals`) seals with the *same* `SealDeadline`
reason but records no op at all, so the two were indistinguishable by
assertion: a regression collapsing mid-drain into pre-exec would have
left both tests green. The mid-drain test now requires the trace's last
op to exist and carry outcome `timeout`.

## Mutation verification (disposable worktree, AGENTS.md isolation)

| Mutant | Result |
| --- | --- |
| `MaxRSSByte = rss / 1024` (conversion missing) | FAIL — 5664 bytes, caught as kilobytes |
| `MaxRSSByte = rss * 1024` (conversion doubled) | FAIL — 5.9 GB, caught as over-converted |
| removal failure ignored (`verified = true`) | **SURVIVES** — equivalent mutant, see below |
| removal failure ignored **and** post-removal `Stat` assertion dropped | FAIL in both teardown subtests |
| mid-drain auto-seal removed | FAIL `TestBudgetMidDrainAutoSeals` (`ACTIVE ""`); `TestBudgetExhaustionSeals` still PASSES, confirming the two branches are now separately evidenced |
| remaining budget no longer caps the effective deadline | FAIL `TestBudgetMidDrainAutoSeals` at 5.22s — the op ran uncapped to completion and succeeded |

**Recorded equivalent mutant:** ignoring `removeErr` alone does not
produce a false DESTROYED, because `teardown` independently re-checks
with `os.Stat` ("asserted, not assumed"). That is defence in depth
working as designed, not a coverage hole — the property survives the
loss of either layer and fails only when both go. Recorded in the same
form as the L10 close's `verifState`-ordering equivalent mutant.

Full module suite green at addendum.
