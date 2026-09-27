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

## D-N-3 — What a delivery intent contains (LOCKED 2026-09-27, owner: snapshot + references)

> Every delivery intent contains immutable lineage references plus an
> immutable snapshot of the Themis facts and the materialized
> destination payload at intent creation. The intent identifies the
> originating event and timestamp, references the Finding / release /
> Position version, captures the applicable facts, and stores
> content-addressed payload bytes for the destination. Destination is a
> governed name/configuration, never a credential. Delivery types:
> `jira_issue`, `ci_build`, `email`. Retries deliver the same snapshot;
> they never reconstruct from current state. Payloads derive only from
> Themis facts — never model output, keys, or workspace content.

"Regenerable from the snapshot", not from the current Finding:
deterministic reconstruction with the original bytes as the delivery
artifact. Same shape as Communication's `Publication` (snapshot
artifact + capped materialized payload + lineage + delivery outcome).

## Q-N-4 — The CI trigger and the result as evidence (OPEN)
## Q-N-5 — Mail: recipients, content, what it must never contain (OPEN)
## Q-N-6 — Credentials and the acting key holder (OPEN)
