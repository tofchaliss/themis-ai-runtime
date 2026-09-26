# Tasks: commissioning (folded into the integration milestones)

Grill CLOSED 2026-09-26 (D-C-1..6). No milestone of its own: the
runtime half lands in I-M1, the Themis half in I-M3
(`themis-integration/tasks.md`). Gate 1 notes go to
`themis-v0/RESUME-HERE.md`.

## Runtime (into I-M1)
- [ ] `skills.Request.Commission` (UUID syntax; empty allowed) → L9 writes
      `origin["commission"]`; `themis-instantiate --commission`
- [ ] Test: the key lands in CREATED as `origin:commission`; absence leaves
      no key; a non-UUID refuses at instantiation; the model never sees it
      (not in payload, not in any L2 composition)

## Themis (into I-M3, under the Themis EDR)
- [ ] Domain: `Commission` on the Finding aggregate (D-C-2 fields; `open` →
      `withdrawn` with witness + rationale); refused on Archived (D-C-4);
      human actors only (D-C-6); events `FindingCommissioned`,
      `CommissionWithdrawn` (thin, internal)
- [ ] Store: migration `finding_commissions` (immutable rows + withdrawal
      columns); read view `commissions[]`
- [ ] API: `POST /findings/{id}/commissions`, `POST …/commissions/{cid}/withdraw`
      (write scope; server-derived actor); `GET` via FindingView
- [ ] `themis-intake`: extracts `origin:commission` from CREATED; evidence
      field `commission_id`; no override flag exists (D-C-5)
- [ ] `raiseProposal`: correspondence check in causal order with named
      refusals (D-C-5 §4); proposal records the commission id
- [ ] Tests: each refusal + positive twin; runtime cannot commission
      (read key refused); withdrawn-after-execution: execution stays
      inspectable, proposal refused; commissioner = proposer admitted;
      cold replay Position → proposal → commission
