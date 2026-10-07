# Outward actions — Jira defect, CI image build, mail (grill opened 2026-09-27)

## What the owner asked

New SBOMs produce a list of Findings to fix. Track it; open a Jira
defect per Finding; when a fix is decided, start a container image build;
when the build is done, send mail.

## Facts from code (2026-09-27)

- The list exists: Findings without a Position (`GET /releases/{id}/posture`).
- Governance emits outbox events on the bus: `finding_opened`,
  `finding_resolved`, `finding_reopened`, `finding_archived`,
  `proposal_raised`, `proposal_accepted`, `finding_commissioned`,
  `commission_withdrawn`.
- Communication already consumes the Governance stream and publishes
  artifacts (`vex`, `advisory`, `notification`, `audit_report`); no
  Jira, mail, webhook or CI integration exists in the greenfield tree.
- The bus reader retries a failing event with backoff (5 attempts,
  100 ms doubling to 30 s) and then declares it POISON and halts the
  stream (`eventbus.ErrStreamHalted`) — a halted stream stops every
  later event for that consumer.
- The harness tool registry (v5) has no network tool; the read door is
  its only Themis-facing seam. It must stay that way.

## Scope of this grill
Q-N-1 who triggers outward actions · Q-N-2 failure of an outward action
· Q-N-3 what a Jira defect mirrors and who closes it · Q-N-4 what the CI
trigger carries and how the result comes back as evidence · Q-N-5 mail:
to whom, what, and what it must never contain · Q-N-6 where the
credentials live and which key holder acts · Q-N-7 who tracks a fix that
ships in another release, and the vocabulary (fix exists ≠ release fixed ≠
Finding resolved).

## What the owner restated (2026-10-01) — the remediation cycle

The outward actions are a LOOP, not a set of one-shot effects:

1. An SBOM is uploaded to Themis; Themis lists its vulnerabilities
   under Product, Project, Release and SBOM id. This starts the cycle.
2. A Jira ticket is created to track the fix, listing the critical,
   high, medium and low vulnerabilities.
3. A Jenkins build is started; it builds a new image and a new SBOM,
   uploaded to the same Product / Project / Release under a new SBOM id.
4. Themis compares the new SBOM with the previous one — which
   vulnerabilities are closed, which are still open — and updates the
   Jira ticket with the result.
5. The result is sent by mail.
6. The loop repeats until the vulnerabilities are fixed, bounded by a
   configurable maximum number of rebuilds; after that it stops and
   tells a person.

Recorded as **D-N-8..D-N-12** in `design.md`. The decisions the
feedback settles: outward actions are gated on Themis finishing its
SBOM evaluation ("release posture evaluated"), with a Themis-owned
pub/sub notification the harness may subscribe to (D-N-8); one Jira
ticket per Release, CVE ids listed for Critical and High only, counts
for Medium and Low (D-N-9 — this supersedes "a Jira defect per
Finding" above for the cycle); a new delivery kind `ci_rebuild`,
**approved**, policy-gated rather than proposal-gated, its callback
required to carry the new SBOM id and the image digest (D-N-10);
default maximum 2 rebuild attempts per Release, an operator knob
(D-N-11); and the ownership recap (D-N-12).

**Scope: documentation only.** No code, API, schema or generated
handler changes. No seam changes on the harness read door, no network
tool, and the harness stays networkless for Jira / CI / mail — it is a
subscriber to the notification and nothing more. **N-M0 is unchanged**:
explicit per-route write-scope authorization stands as implemented,
`delivery:callback` is still refused on every Governance write, and
nothing here relaxes it. Implementation remains Themis-side.

## What the owner decided (2026-10-07) — the seam, the loop and the steps

Everything the 2026-10-01 entry left open is now decided; recorded as
**D-N-13** in `design.md` and Themis-side as `EDR-DELIVERY-01`
**Revision 5 (2026-10-07) — N-M2** (the plan's "Revision 3 — N-M2",
M2-1..M2-9). Still documentation only — no code anywhere yet.

1. **Two events, in one direction.** Knowledge publishes
   `knowledge.release_correlation_completed.v1` **once per SBOM, after
   all its other events for that SBOM**; the bus delivers in `seq`
   order per source context, so every earlier event for that SBOM is
   already processed when Governance handles it. Governance publishes
   `governance.release_evaluated.v1` with product, project, release and
   SBOM ids, the four severity counts and a `cause` of `new_sbom` or
   `rediscovery`. **An SBOM with no matched vulnerabilities still emits
   both events with zero counts — that is the success case.** The
   re-discovery sweep sets `cause=rediscovery` and **never starts or
   advances the loop**. This replaces the `finding_opened` proxy Themis
   has been using (EDR M1a-3).
2. **The harness subscribes by POLLING a Governance cursor read API** —
   `GET /api/v1/governance/events/release-evaluated?after=<sequence>&limit=<n>`,
   `X-API-Key` at **read** scope, at-least-once, **deduplicated by
   event id**, events stored in a new Governance table. **No SSE, no
   long-lived connection.** The harness **only subscribes**, filters
   `cause=new_sbom`, and calls no Jira, no CI and no mail relay.
3. **Max rebuilds per Release:**
   `THEMIS_COMMUNICATION_REBUILD_MAX_ATTEMPTS`, default **2**, **no
   per-Release override**.
4. **Baseline:** each new SBOM is compared with the **previous SBOM of
   the same Release by upload order** — no configured window. The
   targeted Critical+High set is fixed at cycle start and does not grow.
5. **`ci_rebuild` starts a Jenkins job** with `buildWithParameters`,
   Basic auth (user + API token),
   `THEMIS_COMMUNICATION_JENKINS_{ENABLED,URL,USER,API_TOKEN,JOB}`, the
   URL required to be `https`. The job uploads the new SBOM under a
   **`product:<id>`**-scoped key and calls back
   `POST /api/v1/communication/callbacks/ci-rebuild` with a
   **`delivery:callback`** key, sending intent id, build id, git ref,
   image digest and the new SBOM id. **The callback is evidence on the
   intent only and changes no Finding.** No HMAC variant now.
6. **After the new SBOM is evaluated**, Themis compares, updates the
   Release's Jira ticket and sends the mail; it stops when the targeted
   set is closed or the maximum is reached, **telling a person**.
   **Findings are never auto-resolved.**

The build plan is renumbered **N-M2a..N-M2j** (`design.md`), with the
API/schema steps marked: **N-M2d** (Governance events table + cursor
read API) and **N-M2h** (the Communication callback route). **N-M2j is
the only harness-side step** — the poller, which reads and does nothing
else. **N-M0 is unchanged**: the cursor route is a read route under a
read-scoped key, the callback enters through Communication under
`delivery:callback`, and nothing relaxes a Governance write.

**No open questions remain in the cycle's design.**
