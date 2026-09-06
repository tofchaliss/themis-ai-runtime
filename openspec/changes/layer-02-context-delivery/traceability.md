# Layer 2 Traceability — grill invariants → tests

All tests in `src/harness/context`. Invariants from `design.md` §3 (owner-locked closures).

| Invariant | Evidence |
| --- | --- |
| Plan ⊆ Contract, fail closed (Q-L2-1) | `TestGatherFailsClosed` (unknown slot, withheld-slot assignment, double assignment, class not permitted, uncovered slot), `TestContractFailsClosed` |
| Contract is a versioned hashed artifact | `TestContractFailsClosed` (unknown fields, trailing bytes, hollow, invalid enums), hash asserted in `TestComposeShape` |
| L2 has no retrieval-time judgment | Structural: `Gather`/`collect` contain no selection logic; assignments fully determine retrieval — `TestGatherDeterministic` |
| Classification registration-derived; no field determines its own treatment (Q-L2-10) | `TestClassificationForcedFromRegistration` (forged class/provenance/producer/hash/sensitivity all overwritten), `TestSourceClassConstraints` (kind→class table incl. inline/filesystem barred from governed classes) |
| Fail-closed classification default | `TestSourceClassConstraints` (unrecognized kind rejected outright — nothing mints unclassified items) |
| Immutable origin / transport carries no authority | `TestGatherHappyPath` (origin fields), `ItemRef.Mechanism` recorded as transport metadata only; no code path reads Mechanism for classification (structural) |
| Typed absence; four availability states; source-status ≠ delivery-status (Q-L2-4) | `TestOptionalUnavailableTyped`, `TestGatherHappyPath` (withheld) |
| Required-missing fails closed (Q-L2-7) | `TestGatherFailsClosed/required slot unavailable`, `/required slot stub-unwired` |
| Withholding: existence + state only, no content, no guidance (Q-L2-3) | `TestWithheldMarkerMinimal`, `TestComposeShape` |
| Verbatim evidence; no sanitization; data plane never pattern-checked (Q-L2-5) | `TestImperativeEvidenceDeliveredVerbatim` (PoC text with directive phrasing delivered byte-exact under the shipped-policy-compatible seam) |
| Framing integrity (L2-5.3/5.5, security-review hardened) | **Composition-wide content-derived candidate-skip fence**: `TestFenceSelection` (deterministic skip + exhaustion refusal), `TestSiblingFenceForgeryInert` (HIGH-2 PoC shape delivered verbatim, inert), `TestMetadataInjectionRefused` (HIGH-1: newline/bracket/oversized metadata refused as intake) |
| Amplification caps (security-review HIGH-3) | `TestAmplificationCaps` (item-count cap at gather; composed-size cap at compose) |
| Gathered unforgeable outside the package (LOW-1) | `TestComposeRefusesUnvalidatedGathered` |
| Frame metadata descriptive only (L2-5.7) | `TestHashAttributability` (every non-evidence line is fixed descriptive furniture), `TestWithheldMarkerMinimal` |
| L2 authors no propositions (Q-L2-6/9) | `TestHashAttributability` — stripping evidence leaves only furniture; any other byte fails the test |
| EIS verbatim, policy-hash-bound via L1 seam | `TestComposeEISVerbatim`, `TestComposeRefusesWithoutInputs` (nil policy refuses through the seam) |
| Dual-reader record; PayloadHash covers the length-framed canonical form (L2-5.3/5.4) | `payloadHash` length-frames every field; `TestComposeDeterministic` (byte-identical + hash-stable) |
| No append-to-conversation API (epoch boundary) | Structural: package exposes `Gather`/`Compose` only — no conversation type exists |
| Filesystem confinement (gate-0) | `TestConfinement` (absolute, dotdot, symlink escape, empty path, missing root — all `ErrConfinement`; confined read works) |
| Caps fail closed (gate-0) | `TestGatherFailsClosed/oversized inline item`; total cap enforced in `Gather` (`ErrContextTooLarge`) |
| No DB relationship; typed Themis seam (Q-L2-8) | Structural: only `ThemisReader` interface; no driver imports; stub-unwired path in `TestGatherFailsClosed` |
| Intake/resolution split consistent with L1 | `TestGatherErrorClassification` (task-payload cap breach → failed_intake; plan-outside → failed_resolution) |
| Delivery proof: payload byte-identical through the Model Interface | `TestDeliveryProofMockProvider` |
| Operational proof (gate-0) | `TestLiveOperationalProof` — full shipped L1 roots + policy + contract + gathered evidence executed against a live local model (run PASS 2026-09-06, 41.65s, clean stop; skips hermetically when no endpoint) |
