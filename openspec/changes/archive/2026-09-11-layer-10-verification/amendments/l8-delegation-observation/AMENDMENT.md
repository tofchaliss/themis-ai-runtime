# L10 Amendment: delegation observation in the history view (L8 M5, 2026-09-23)

Authorized by: L8 design D-L8-18 and §5.2 (owner LOCK 2026-09-22).
Additive, read-only.

`verification.HistoryView` gains `Delegations []DelegationObservation
{seq, parent_call_seq, template, outcome}`, derived by
`seam.TaskVerificationHistory` from `l8-delegation` events. L10 sees
that a delegation occurred, when, under which template, with which
recorded outcome — no evidence kind, no contract, no outcome
vocabulary of its own, no authority. An `l8-delegation` is never a
verification instance (`TestL10HistoryObservesDelegation`: the
verification history and latest-per-contract projection stay empty).
