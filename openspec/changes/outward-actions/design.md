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

## D-N-4 — The CI trigger and the result as evidence (LOCKED 2026-09-27, owner: option 1)

> `ci_build` delivery carries the accepted change artifact, not merely
> references. For a `proposal_accepted` whose evidence is a harness
> execution, the intent snapshot contains the recorded artifact members
> and their hashes, the artifact identity, release and Finding lineage.
> CI applies those bytes to a branch of the release repository and
> builds from that branch. The branch is never automatically merged;
> merge remains a human / repository-controlled act. CI returns
> `{intent_id, build_id, image_digest, git_ref}`; Themis records the
> callback as governed-external evidence on the delivery intent, not as
> Finding truth. The image digest alone changes no Finding state; the
> subsequent SBOM must be ingested and evaluated before security truth
> changes.

Boundaries made explicit: "CI applies those bytes" never means CI
reinterprets the artifact — the snapshot names exactly what CI is
authorized to materialize; CI is the build executor, not a security
decision-maker. `image digest ≠ fixed Finding`. Branch creation is
automated; merge is human-controlled. Rejected: references only with a
human PR first (the code path exists, it is just slower — kept as the
fallback when a repository forbids machine branches); Themis pushing
to git itself (Themis is not a code author and holds no repo write key).

## D-N-5 — Mail (LOCKED 2026-09-27, owner: option 1)

> Mail uses governed Communication audiences and contains facts from
> the immutable delivery snapshot only. Recipients are named
> Communication audiences, resolved to addresses by the mail worker;
> addresses and credentials are never carried in the intent. Mail
> materializes the event, Finding identity, CVE/PURL, release, Position
> version and verbatim stance, the applicable build result, dead-letter
> status where applicable, and a Themis link. It contains no model
> output, report text, workspace content, key identifiers/values, or
> attachments. Delivery is plain text and idempotent by intent id.

Event mapping: `finding_opened` → Jira; `proposal_accepted` → mail
(decision notification); CI callback → mail (build result); delivery
dead-letter → mail (operations audience). Position stance is carried
verbatim; Communication never rewrites it. One mail per intent; a retry
is a delivery retry, not a new communication event. Rejected: free-form
mail with the model's report "for context" (model output would leave
the sandbox as a message from Themis); per-user subscriptions (a
separate product capability with its own grill).

## D-N-6 — Credentials and acting identities (LOCKED 2026-09-27, owner: option 1, HMAC as transport variant)

> Outbound credentials are environment-referenced secrets owned by the
> Communication deployment; inbound CI callbacks use a dedicated
> `delivery:callback` authorization scope. Jira, SMTP and CI trigger
> credentials are referenced by destination name and never persisted
> in Themis data or Governance records. The callback endpoint accepts
> only `delivery:callback`. Governance write endpoints no longer treat
> every non-read key as universally write-capable; they explicitly
> require `admin` or `product:<id>`. Where a CI system cannot present a
> Themis key, an HMAC-authenticated callback may be used as a transport
> variant, its secret held in Communication's environment. External
> actions carry service identities, not human decision-maker credentials.

Scope table after this decision: `read` → reads; `product:<id>` →
permitted product-scoped Governance writes; `admin` → administrative
Governance writes; `delivery:callback` → the Communication callback
only, never a Governance write. This is the closure of the row-14 gap's
first half ("any non-read key → any write"). The second half —
confining `product:<id>` to that product's Findings — stays the Themis
security EDR item and is a prerequisite of N-M0 below. HMAC is
transport authentication, not an alternative authorization
architecture: the callback still enters only through the Communication
boundary. Rejected: reusing an `admin` key for CI (a build system would
hold the decider's power).

## Grill state (2026-09-27)
Q-N-1..6 LOCKED. Implementation is Themis-side under a new EDR
(`EDR-DELIVERY-01`) and a `phase3-outward-actions` change; the harness
tree is untouched by design (D-N-1). Milestones in `tasks.md`.

