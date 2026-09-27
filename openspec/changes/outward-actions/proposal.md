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
