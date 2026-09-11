# L4 Amendment: L10 verification seam (2026-09-11)

Additive changes to the archived L4 layer, authorized by D-L10-3/16
and Gate 0; full context in the L7 amendment record
(../../2026-09-07-layer-07-orchestration/amendments/
L10-verification-seam/AMENDMENT.md, items 7 and the M4 executor note).

1. `ToolDef.VerifierEligible` — optional registration property marking
   deterministic-verifier eligibility. A classification, never an
   authorization branch; verifier calls pass the identical L4 gate.
2. Loader refusal: a tool cannot be both `control` and
   `verifier_eligible` (close security review L-3) — the verifier
   result branch would swallow the control signal.
3. Executor `verify_report` (registry-v4 era): raw evidence capture
   only, identical confinement and caps to `read_file`; no parsing,
   no judgment. Process-execution verifier classes remain
   unregistrable pending an L5 amendment (recorded residual).

Invariants not weakened: absent field = not eligible; pre-amendment
registries load identically; authorization/argument validation
unchanged; the L4 suite re-runs green (close evidence, L10 tasks.md).
