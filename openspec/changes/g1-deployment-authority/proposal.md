# Proposal: G1 — Deployment Authority Anchoring

Status: GRILL OPEN 2026-09-13. Source: L1–L11 integration audit
(openspec/changes/l1-l11-integration-audit/synthesis.md, G1;
owner-classified genuine cross-layer gap, grill-first). No
production wiring of the governed chain before this closes.

## The gap (owner formulation)

The governed entry point verifies that the submitter-assembled
bundle is internally consistent, but it does not establish that
this bundle is the authoritative artifact set governing this
deployment. That is an authority question, not an implementation
defect: grant ⊆ ceiling is checked against a ceiling the submitter
chose; no layer owns "which artifacts are THE deployment's governed
set"; the task record carries no submitter identity.

## Grill questions (owner-set)

| Q | Question |
|---|---|
| Q-G1-1 | Who owns deployment identity? |
| Q-G1-2 | What constitutes the authoritative deployment artifact set? |
| Q-G1-3 | Where is that set declared? |
| Q-G1-4 | How is the set bound to the execution? |
| Q-G1-5 | Can a submitter choose workflow/ceiling/instructions/registry artifacts independently? |
| Q-G1-6 | What immutable identity represents the deployment? |
| Q-G1-7 | How does the chain prove executed artifacts are the governed deployment, not merely a mutually consistent bundle? |
| Q-G1-8 | How are deployment changes/supersession represented? |
| Q-G1-9 | What happens when the submitter bundle and the deployment anchor disagree? |
| Q-G1-10 | Does the anchor introduce another authority store, or attach to an existing Governance mechanism? |

## Constraints inherited (not grillable)

Fail closed; nothing defaulted; append-only governed registrations
with immutable hash bindings; *.proposed.* → owner act → ACTIVE;
admission enforced at consumption (the verdict/digest pattern);
model output advisory never authority; no new authority class
without a Governance owner; Day-0 prohibitions.
