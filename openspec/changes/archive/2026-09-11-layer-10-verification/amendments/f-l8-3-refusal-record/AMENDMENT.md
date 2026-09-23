# L10 Amendment: pre-instance refusal text is recorded (F-L8-3, 2026-09-23)

Authorized by: L8 design C-L8-12 (gap F-L8-3) and §5.3 (owner closure,
LOCKED 2026-09-22). D-L10-8 stands: a pre-instance refusal creates no
evaluation instance, no outcome, and no `l10-verification` event.

## Gap

D-L7-11 promises byte-exact reconstruction of the model-visible
conversation, but the seam's pre-instance refusal text was returned to
the model AFTER the call's `l4-audit` had committed and was recorded
nowhere.

## Amendment (the C-L8-12 mechanism, applied to the L10 seam)

- `orchestration.VerificationEvaluator` gains `PreResolve(taskID, call,
  authRegistrySHA256) (refusal, err)`: the pre-instance stage on its
  own — argument shape, contract naming, authorizing-registry
  agreement, atomic registry resolution (append-only wall intact),
  capability binding, the v1 single-slot constraint. No evidence, no
  instance.
- `seam.Evaluator` factors stage 1 into `resolveCall`, shared by
  `PreResolve` and `EvaluateCall`, so the two cannot disagree on what
  refuses (`TestPreResolveMirrorsStageOne`).
- L7 (`loop.go`) calls `PreResolve` for an authorized verifier-eligible
  call before committing its `l4-audit`; a refusal rides in that audit
  body as `VerificationRefusal` and the model receives
  `verification refused: <recorded text>`; `EvaluateCall` is not
  called. Evidence: `TestVerificationRefusalIsNotAnOutcome` now asserts
  the recorded text; mutation: dropping the annotation → fails.

## Residual

`EvaluateCall` re-runs stage 1 on its own registry load. A refusal
arising only at that second load (a withdrawal between the two loads
inside one call) is still returned to the model unrecorded — a window
of one call's duration, recorded here rather than closed by carrying
resolved state across the seam.
