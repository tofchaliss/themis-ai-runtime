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

Still open, deliberately: the notification seam's event names,
delivery semantics, transport, subscriber authentication and owning
context need their own EDR and API change before implementation; the
configuration locus and name of the max-attempts knob; and whether the
comparison baseline is strictly the immediately-previous SBOM id for
the Release or a configured window.
