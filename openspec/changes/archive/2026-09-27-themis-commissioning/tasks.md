# Tasks: commissioning (folded into the integration milestones)

Grill CLOSED 2026-09-26 (D-C-1..6). No milestone of its own: the
runtime half lands in I-M1, the Themis half in I-M3
(`themis-integration/tasks.md`). Gate 1 notes go to
`themis-v0/RESUME-HERE.md`.

## Runtime (into I-M1) — LANDED 2026-09-26
- [x] `skills.Request.Commission` (UUID syntax; empty allowed) → L9 writes
      `origin["commission"]`; `themis-instantiate --commission`
- [x] Test: the key lands in the envelope's sealed origin (L7 records every
      origin key as `origin:<key>`); absence leaves no key; a non-UUID
      refuses at instantiation; the commission is not in the payload

## Themis (into I-M3, under the Themis EDR)
- [x] Domain: `Commission` on the Finding aggregate (D-C-2 fields; `open` →
      `withdrawn` with witness + rationale); refused on Archived (D-C-4);
      human actors only (D-C-6); events `FindingCommissioned`,
      `CommissionWithdrawn` (thin, internal)
- [x] Store: migration `finding_commissions` (000014) (immutable rows + withdrawal
      columns); read view `commissions[]`
- [x] API: `POST /findings/{id}/commissions`, `POST …/commissions/{cid}/withdraw`
      (write scope; server-derived actor); `GET` via FindingView
- [x] `themis-intake`: extracts `origin:commission` from CREATED; evidence
      field `commission_id`; no override flag exists (D-C-5)
- [x] `raiseProposal`: correspondence check in causal order with named
      refusals (D-C-5 §4); proposal records the commission id
- [x] Tests: each refusal + positive twin; withdrawn-after-execution
      (execution inspectable, proposal refused); commissioner = proposer
      admitted (Themis `1414997`); read key cannot commission — 403 from
      `RequireWriteScope` before the handler, nothing recorded, the
      product-scoped operator admitted (`TestReadKeyCannotCommission`);
      cold replay Position → AcceptedProposalID → proposal → evidence →
      commission, in-memory (`TestHarnessPositionReplaysToCommission`) and
      from persisted rows under embedded Postgres (store integration test)
      — Themis 2026-09-27, uncommitted until asked
