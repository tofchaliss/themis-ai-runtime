# L7 Amendment: L8 prerequisites F-L8-2 and F-L8-4 (2026-09-23)

Authorized by: L8 design §5.3 (owner closure, LOCKED 2026-09-22). The
L7 archive stands; both changes are additive and change no constitution
vocabulary.

## F-L8-2 — execution identity on `model-turn` (C-L8-10)

`recordTurn` writes, beside the governed `model` name, the adapter's
`identity {wire_model, runtime, reported}` and the provider `endpoint`
for every answered turn (absent on `provider-error`, where no response
exists). L6 validates the envelope, never the content: no constitution
change. Purpose: a parent turn and an `l8-delegation` are comparable at
reconstruction. Evidence: `TestModelTurnRecordsExecutionIdentity`.

## F-L8-4 — parent turn deadline (C-L8-19)

The model call's context deadline is `min(turn_timeout_sec, remaining
wall budget)`: a provider call can never outlive the floor it runs
under. Evidence: `TestParentTurnDeadlineIsMinOfTurnAndWall` (a blocking
model under a 2 s wall budget and a 180 s turn timeout terminates in
seconds). Mutation: removing the `min` → the test fails (probed
2026-09-23).

## F-L8-3 (L10-owned, wired here)

For an AUTHORIZED verifier-eligible call, L7 calls the seam's
`PreResolve` BEFORE the call's `l4-audit` commits and records a
non-empty refusal in the audit body (`VerificationRefusal`); the
model-visible message is the recorded text, verbatim. See the L10
amendment record.
