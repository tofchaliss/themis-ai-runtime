# L7 Amendment: L10 verification seam (2026-09-11)

Authorized by: D-L10-13 (LOCKED), D-L10-17 (the amendment protocol),
Gate 0 (owner, 2026-09-11). This record is evidence of change to an
archived layer — never a second L7 specification. The original archive
stands; this amendment is additive only.

## Amendment definition

1. **Constitution (orchestration/constitution.go):** five verification
   outcome events added to the transition vocabulary —
   verification-pass / -fail / -inconclusive / -unavailable / -invalid
   — each carrying an opaque contract-identity token in its recorded
   body. ConstitutionHash changes as a consequence (deliberate,
   hash-changing, per the constitution's own header).
2. **Workflow schema (orchestration/workflow.go):** Edge gains an
   optional `gate` — (exact opaque contract token, required outcome
   from the closed five-value vocabulary). Loader rules added: exact
   token syntax (name@canonical-version; no floating/latest/ranges);
   outcome from the closed vocabulary (Governance vocabulary refused);
   gated edges counter-free, forward/terminal/@stay only, must precede
   and be backed by exactly one mandatory ungated fallback per event
   (totality stays static under the gate ladder); verification events
   are conditionally mandatory — totality over REACHABLE events (owner
   amendment 2 to D-L10-13), enforced per phase at assembly.
3. **δ (orchestration/loop.go step()):** gate-aware edge selection —
   gated edges in definition order against the latest-per-contract
   walk state, first satisfied wins, ungated fallback otherwise. Gated
   edge identity is "<phase>/<on>#g<idx>"; ungated identity is
   unchanged "<phase>/<on>". The fired gate (token + outcome) is
   recorded in the transition body.
4. **Walk state (loop.go):** verifState — latest-per-contract outcome
   (the CallState pattern), updated only after the EvVerification
   record commits; gate satisfaction re-derived statelessly at each
   transition evaluation (D-L10-9).
5. **Verification result path (loop.go):** an executed
   verifier-eligible capability result flows through the injected
   VerificationEvaluator (Config.Verifier). Durable stores (contract
   bytes — R-L9-2 pattern; raw; canonical; evaluation record) precede
   the EvVerification event commit, which precedes the typed event
   entering δ (record-before-event, D-L10-6 stage 5). Pre-instance
   refusals are recorded as evidence and surfaced to the model as
   data; they produce no event (D-L10-8). Evaluator machinery failure
   follows the existing invariant path and mints nothing.
6. **Assembly (orchestration/orchestrator.go):** declaration-gated
   exposure — a phase exposing a verifier-eligible capability refuses
   at assembly unless the evaluator hook is wired, the workflow
   declares all five verification events, and THAT phase maps each
   (reachability totality). L7 reads eligibility from the decoded L4
   registry it already loads; it never consults the L10 registry.
7. **L4 registry schema (tools/registry.go):** additive optional
   `verifier_eligible` classification on ToolDef — a registration
   property, never an authorization branch.
8. **L6 event vocabulary (state/constitution.go):** additive event
   class `l10-verification` (EvVerification) — a new semantic gets its
   own class rather than a third meaning on an existing one (the L9
   event-class lesson). L6 validates the envelope, never the content.
9. **API-closure allowlists (orchestration_test.go):** deliberate,
   commented extensions — the five event constants; types GateCond,
   VerificationEvaluator, VerificationOutcome.

## Affected invariants — none weakened

- Pre-amendment workflow definitions load and walk identically (no
  gates, no verification vocabulary → all new paths unreachable);
  pinned by TestPreAmendmentWorkflowUnchanged and the archived suite.
- δ still consumes only recorded typed events; model content still
  never reaches control; totality is still static (the gate ladder
  keeps exactly one mandatory ungated edge per event).
- L7 remains contract-blind: zero dependency on the verification
  package (go list -deps = 0 hits), no registry lookup, opaque token
  equality only. A nonexistent/withdrawn contract is an unsatisfiable
  gate, never an L7 error.
- Record-before-effect extends to verification: no outcome enters δ
  before its evaluation record is durable.
- No new execution-initiation semantics: verification is
  model-proposed only (D-L10-6); the hook fires solely in the existing
  governed result-processing path.

## New conformance proofs (orchestration/verification_seam_test.go)

- TestVerificationGateVocabularyLoads — the vocabulary loads.
- TestVerificationSeamLoaderRefusals — nine doctored definitions
  refused: floating/latest/non-canonical gate tokens, Governance
  vocabulary and unknown outcomes as gate outcomes, gated edge without
  fallback, gated edge with counter, gated edge after fallback,
  verification edges without declaration.
- TestVerificationEventsConditionallyTotal — reachability-based
  totality accepts a verifier-free phase without artificial edges.
- TestGateLadderDeterminism — first-satisfied-gate-wins, opaque token
  matching (foreign-token PASS opens nothing), downgrade closes the
  gate again, fired-branch edge identity.
- TestPreAmendmentWorkflowUnchanged — legacy definitions unaffected.

## Archived-suite result

Full `go test ./orchestration/ -count=1` GREEN post-amendment
(105.9s — fault sweeps, kill-recovery, skill seam/reconstruction/
equivalence registers, API closure all re-passed). tools/ and state/
suites green. The two API-closure allowlist extensions were caught by
the guards first and then deliberately recorded — the mechanism
constrained the implementation (the L9 process precedent).

## Deferred to the L10 change (recorded, not hidden)

- Replay/Register-R proofs over gate-bearing walks (need M4's verifier
  capability + M6's scripted walks); fault-point coverage of the
  verification store/commit window.
- Scoped architecture review of this amendment: batched into the L10
  close reviews (M7), scope-tagged to this record.
- Resulting artifact hashes: recorded at the L10 archive alongside the
  final tree state.
