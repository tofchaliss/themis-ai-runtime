# L10 Amendment: Register T #5 evidence never executed (2026-09-14)

**Nature: evidence-record correction, not an architecture change.** No
L10 decision is amended and the control is sound. What is corrected is
the basis on which a test-review finding was recorded CLOSED.

## What was recorded

tasks.md, at the L10 close:

> Register T #8 cross-task replay → CLOSED
> (TestCrossTaskReplayCannotSatisfyGate). **T #5 event tamper → CLOSED
> (TestEventTamperDetectedAtReadBoundary).**

## What actually happened

`TestEventTamperDetectedAtReadBoundary` (added in af6599e, the L10
close remediation) read the raw event stream at
`tasks/verif-tamper/events.jsonl`. L6 writes `events.log`
(`state/task.go`, `state/view.go`). The read therefore always failed,
and the guard

```go
if rerr != nil {
    t.Skipf("stream file not at expected path: %v", rerr)
}
```

turned that into a silent skip. The test reported green in every run
since it was written, on every platform, **without ever executing its
assertion**. Register T #5 was closed on evidence that never ran.

It was still skipping on 2026-09-14, found by enumerating skips across
the whole suite (`go test ./... -v`, live proofs disabled) while
sweeping the archives for platform-sensitive evidence gaps after the
L5 group-kill correction.

## Is the control sound?

Yes — verified before changing anything. In a disposable worktree with
only the path corrected, the test **passes**: a tampered
`EvVerification` outcome is detected at the authoritative read
boundary. The gap was in the evidence, not the mechanism.

## What the evidence is now

Path corrected to `events.log`, and **both skip guards converted to
failures**. An escape hatch is precisely wrong here: if the stream's
storage shape moves out from under this test, the tamper evidence must
fail loudly rather than evaporate, which is the exact failure this
amendment exists to correct. The second guard now also reports the
stream length, so a "nothing was tampered" failure is diagnosable.

Mutation-verified in a disposable git worktree (AGENTS.md probe
isolation):

| Probe | Result |
| --- | --- |
| no tamper applied (write back original bytes) | FAIL — the assertion depends on the tamper |
| both detectors suppressed (`ReadEvents` corruption + `VerdictCorrupt`) | FAIL — detection loss is caught |
| `ReadEvents` corruption suppressed **only** | SURVIVES — see below |

**Recorded equivalent mutant:** suppressing corruption in
`state.ReadEvents` alone does not fail this test, because the test
asserts the property "detected at the authoritative read boundary by
`ReadEvents` **or** a `CORRUPT` verdict from `ReadStatus`" — and
`ReadStatus` still detects. That is the property Register T #5 asked
for, so the surviving mutant is equivalent with respect to this test's
claim, not a coverage hole. Recorded here in the same form as the L10
close's `verifState`-ordering equivalent mutant.

## Scope

Register T #5 is now genuinely closed. No L10 decision, contract, or
vocabulary changes; no other Register T item is affected (T #8 and the
fault-sweep closures cite tests that exist and run). Full module suite
green.
