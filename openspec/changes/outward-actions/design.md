# Design record — outward actions (PROPOSED until each Q-N is locked)

## D-N-1 — Who triggers outward actions (LOCKED 2026-09-27, owner: option 1)

> Event-driven outward actions. External actions are projections /
> effects of already-established Themis events; they do not acquire
> Governance authority. The harness remains networkless.

`finding_opened` → Jira defect, automatically (a ticket is a mirror,
not a decision). `proposal_accepted` → CI image-build trigger,
automatically — the human decision has already happened. Build result
→ mail, automatically. Jira and mail are Communication outputs, never
sources of security truth. CI owns image building; Themis records the
trigger and the result as evidence.

```
Themis Governance event → Communication / CI integration → external system → result back to Themis as evidence
```
never `Harness → Jira / CI / Mail`.

Rejected: everything manual (safe, slow, no reason to lose the
events); the harness calling Jira/CI/mail as tools (breaks the sandbox
and "the harness never initiates a Governance act").

## D-N-2 — Failure of an outward action (LOCKED 2026-09-27, owner: option 1 + the isolation rule)

> Durable delivery intent with isolated per-system delivery workers.
> Themis durably records the outward delivery intent before attempting
> the external action. Jira, CI and mail delivery are independently
> retried and failure is isolated from the Governance event stream.
> Exhausted or continuously failing delivery becomes a visible
> dead-letter state requiring human retry or cancellation. Successful
> delivery records the external result as evidence. External
> availability never blocks or changes Themis security truth or
> Governance decisions.

Two boundaries: the delivery row is the durable obligation (never
inferred later from the event); dead-letter is operational state and
evidence ("Jira unreachable since 14:02, 37 attempts"), never a change
to Finding state. Not locked here: retry counts, backoff, dead-letter
schema — implementation questions. Fact that shaped it: the bus reader
poison-halts a stream after 5 failed attempts; outward failures must
never reach that path.

## Q-N-3 — What a Jira defect mirrors and who closes it (OPEN)
## Q-N-4 — The CI trigger and the result as evidence (OPEN)
## Q-N-5 — Mail: recipients, content, what it must never contain (OPEN)
## Q-N-6 — Credentials and the acting key holder (OPEN)
