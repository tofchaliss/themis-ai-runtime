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

## D-N-7 — Cross-release remediation tracking (LOCKED 2026-09-27, owner: option 1)

> Cross-release remediation is tracked by Themis from the CI delivery
> chain and the subsequent SBOM evaluation. The `ci_build` intent
> retains lineage to the original Finding and release. CI returns the
> produced release version, `git_ref` and image digest and registers
> the release through the Registry process. The resulting image SBOM
> cites the intent and is evaluated by Themis. Only the SBOM evaluation
> can establish that the fault is absent from the new release. Themis
> records `remediated_in` on the original Finding as governed-external
> evidence (new release, image digest, intent, evaluation time). The
> original release remains affected. A human performs the eventual
> Finding resolution decision.

> Terminology: `mitigated` in a Position means the governed remediation
> decision was accepted; `fixed` is reserved for a release whose
> evaluated SBOM demonstrates the fault is no longer present.

Three states, never conflated: fix exists ≠ release is fixed ≠ Finding
is resolved. R.x.y stays affected (its image never changed); the estate
graph answers "which deployments still run R.x.y". A clean R.x.y+1
SBOM establishes something about R.x.y+1 only. A failed attempt (z still
matched) records the same field as an attempt, and F stays as it is.
One more D-N-5 mail event: `remediated_in` written. Rejected:
auto-resolving F (resolution is a decision about exposure, not about the
existence of a fix); the link typed into Jira (Jira is a projection; the
authoritative relationship lives in Themis).

## The remediation cycle (D-N-8..D-N-12, owner feedback 2026-10-01)

The owner restated the workflow end to end: an SBOM is uploaded to
Themis under Product / Project / Release / SBOM id; Themis lists its
vulnerabilities; a Jira ticket tracks the fix; a Jenkins build produces
a new image and a new SBOM uploaded to the same Product / Project /
Release under a new SBOM id; Themis compares new against previous
(closed vs still open) and updates the ticket; the result goes out by
mail; the loop repeats until the vulnerabilities are fixed or a
configured maximum number of rebuilds is reached, after which it stops
and tells a person.

D-N-1..7 already decide who may act and what an outward action carries.
D-N-8..D-N-12 decide the shape of the LOOP those actions form. They are
**documentation only**: no code, API, schema or generated handler
changes with them, and nothing here is implemented yet.

## D-N-8 — Valuation-complete gate and the Themis→harness notification (owner feedback 2026-10-01)

> Outward actions fire only after Themis has finished evaluating the
> uploaded SBOM. SBOM receipt is not the trigger; "release posture
> evaluated" is. Jira creation and update, the CI rebuild intent and
> mail are all held until Themis publishes that signal for the Release.
> Themis owns and operates the notification: a Themis-side pub/sub
> seam the harness MAY subscribe to. The harness is a subscriber only
> — it still calls no Jira, no CI and no mail relay, and it still
> initiates no Governance act.

Ingestion is asynchronous: a document can be stored long before its
vulnerabilities are known. A ticket written from a half-evaluated SBOM
lists a subset and reads as truth, and a rebuild triggered on receipt
rebuilds against nothing. Gating on evaluation makes every outward
payload a projection of a settled posture, which is what D-N-1 already
requires of Jira and mail.

The notification is a new cross-repository seam and is therefore the
one genuinely undecided thing in this section: event name(s), delivery
semantics (at-least-once assumed), transport, subscriber
authentication and whether it lives in Communication or Governance are
NOT decided here. They need their own EDR and API change before any
implementation. Until then the existing Governance events (notably
`finding_opened`) remain the effective trigger in code — this section
describes the target design, not current behaviour.

What does not change: the harness read door stays the only
harness-facing Themis seam for data, the harness stays networkless for
external systems, and subscribing to a notification is not authority
(D-N-1).

## D-N-9 — Jira ticket semantics (owner feedback 2026-10-01)

> One Jira ticket per Release, not one per Finding. The ticket body
> carries the severity counts for Critical, High, Medium and Low, and
> lists the CVE ids for **Critical and High only**; Medium and Low are
> represented by their counts, with no CVE list. The ticket is updated
> in place across the cycle's attempts — updates are idempotent, keyed
> by delivery intent id and attempt index. Jira remains a projection
> of Themis facts and never a source of security truth; no Themis
> state follows from a Jira transition.

This supersedes "a Jira defect per Finding" in the original owner ask
(see `proposal.md`) for the remediation cycle: the unit a rebuild
addresses is a Release, so the unit a tracking ticket addresses is a
Release. Listing every CVE id at every severity makes the ticket
unreadable at estate scale, and the two severities a rebuild is
actually judged on are Critical and High.

## D-N-10 — `ci_rebuild`, a new delivery kind (APPROVED 2026-10-01, owner)

> A fourth delivery kind joins `jira_issue`, `ci_build` and `email`
> (D-N-3): `ci_rebuild`. It is a policy-controlled rebuild of a
> Release that does NOT require a fresh human proposal acceptance —
> the cycle's authority comes from the policy that started it, bounded
> by lineage and knobs. Its snapshot carries the destination pipeline
> name, the Product / Project / Release identity, the prior SBOM id,
> the targeted Finding set and the attempt index — and no credential,
> no key, no model output. The callback MUST carry the new SBOM id and
> the image digest alongside `{intent_id, build_id, git_ref}`. The
> callback is governed-external evidence on the intent and changes no
> Finding state: only the evaluation of the new SBOM can establish
> that a fault is absent (D-N-4, D-N-7). Asserted trust — a build
> system claiming a fix — is refused.

`ci_build` (D-N-4) is unchanged and keeps its governance-controlled
path: it carries an accepted change artifact and follows
`proposal_accepted`. `ci_rebuild` carries no artifact; it asks the
pipeline to rebuild the Release as configured, which is why it can be
policy-gated rather than proposal-gated. The two must not be
conflated: one materializes a human-accepted change, the other repeats
a build.

Approved here means approved as a decision of record. It is not
implemented, and the knobs below are documentation until the Themis
milestone that builds them.

## D-N-11 — Loop control and the stop condition (owner feedback 2026-10-01)

> After each `ci_rebuild` callback and the evaluation of the new SBOM,
> Themis compares the new SBOM against the previous one for the same
> Product / Project / Release: which of the targeted vulnerabilities
> are closed and which are still open. The Jira ticket is updated with
> that comparison and the mail is sent with it — both AFTER the
> comparison, never from the callback alone. Success is the targeted
> set closed, and the loop stops. Otherwise the loop repeats, bounded
> by a maximum number of rebuild attempts per Release: **default 2**,
> an operator-configurable policy knob on the Themis side (name and
> locus to be fixed by the implementing milestone). On exhaustion the
> loop stops and tells a person — mail to the governed audience plus a
> Jira update saying the attempts are exhausted. Findings are never
> auto-resolved: resolution stays a human decision (D-N-7).

Two is deliberately low. An automated rebuild loop that cannot fix a
Release in two attempts is not going to fix it in ten, and the failure
mode of a high limit is a pipeline hammering itself while nobody
reads the mail. The number is a knob precisely so it can rise when the
loop has a reliability record.

Failure here means "the targeted set is still open", which is an
outcome, not an error: the attempt is recorded, the Finding stays as
it is, and the original Release stays affected (D-N-7).

## D-N-12 — Ownership recap and the invariants this cycle does not touch

> **Themis owns** security truth and every outward effect: SBOM
> intake and evaluation, the posture and the Findings, the comparison
> of new against previous SBOM, the delivery intents and workers,
> Jira, CI (`ci_build` and `ci_rebuild`), mail, the loop counter, the
> stop condition and the valuation-complete notification it publishes.
> **The harness owns** runtime execution and orchestration of its own
> work, and MAY subscribe to the valuation-complete notification. It
> calls no Jira, no CI and no mail relay, holds no outward credential,
> and initiates no Governance act.

Unchanged by D-N-8..D-N-12, and restated because a loop is exactly the
place where they get eroded:

- Model output is advisory; it never enters a delivery payload and
  never leaves the sandbox as a message from Themis (D-N-5).
- Delivery payloads derive only from the immutable snapshot of Themis
  facts (D-N-3).
- Secrets are never carried in an intent; destinations are governed
  names (D-N-3, D-N-6).
- Controls fail closed: no evaluation signal → no outward action.
- External availability never blocks or changes Themis truth; outward
  failure is dead-letter state, not a Finding change (D-N-2).
- **N-M0 is unchanged.** Explicit per-route write-scope authorization
  stands as implemented: `delivery:callback` is refused on every
  Governance write, `product:<id>` is confined to its own product, and
  the scope vocabulary stays closed. `ci_rebuild` and the
  valuation-complete notification imply no new scope, no relaxation
  and no new write path; a `ci_rebuild` callback enters through the
  Communication boundary under `delivery:callback` exactly as D-N-6
  requires.

## Build steps for the cycle (each testable on its own)

The implementing milestones are Themis-side (`phase3-outward-actions`,
Group 5 onward). Each step is independently testable:

1. Evaluation-complete event published for a Release — assert it fires
   once per evaluated SBOM, after evaluation, never on receipt.
2. The notification seam (own EDR first) — a subscriber receives the
   signal; the harness subscribes and does nothing else.
3. SBOM-to-SBOM comparison for one Release — closed vs still open,
   from two stored SBOMs, no outward action involved.
4. `jira_issue` per Release with the D-N-9 body rule — counts for four
   severities, CVE ids for Critical and High only; update idempotent.
5. `ci_rebuild` intent + callback — snapshot members, callback
   requiring new SBOM id and image digest, no Finding change.
6. Mail after comparison — one mail per intent, facts from the
   snapshot, plain text.
7. Loop control — attempt counter, max-attempts knob (default 2),
   success stop, exhaustion stop with mail + Jira update, no
   auto-resolution.

## Grill state (2026-09-27, extended 2026-10-01)
Q-N-1..7 LOCKED. D-N-8..D-N-12 recorded 2026-10-01 from owner feedback
(documentation only; `ci_rebuild` approved, the notification seam
approved in principle with its transport deferred to its own EDR).
Implementation is Themis-side under `EDR-DELIVERY-01` and the
`phase3-outward-actions` change; the harness tree is untouched by
design (D-N-1, D-N-12). Milestones in `tasks.md`.

