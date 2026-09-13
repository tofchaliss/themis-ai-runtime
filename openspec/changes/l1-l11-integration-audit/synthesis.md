# Phase B synthesis — L1–L11 integration audit (2026-09-13)

Three parallel Phase A audits (S1–S4, S5–S7, S8–S10+X1–X3) against
the ten archived constitutions. Per-seam scoreboard:

| Seam | Verdict |
|---|---|
| S1 initiation→L1 | LEAKS (G1, R1) |
| S2 L1→L2/L3 | HOLDS (defects D6, D7) |
| S3 →L4 authorization | HOLDS |
| S4 L4→L5 execution | HOLDS; residuals respected |
| S5 →L6 durability | HOLDS (residual scope caution R2) |
| S6 L6→L7 (3 consumers) | HOLDS for L7/L10 consumers; LEAKS at the L11 consumer (D1) |
| S7 L7→L9 skills | HOLDS; residuals verified, tension noted (R4) |
| S8 walk→L10 | HOLDS |
| S9 records→L11 | LEAKS (D1–D5) |
| S10 L11→Governance | HOLDS (activation verified) |
| X1 model I/O | LEAKS (D1) + genuine gap (G2) |
| X2 bench↔router↔L11 | LEAKS (D2); single-home HOLDS |
| X3 egress | RESIDUAL-COVERED (coupling R3) |

**The audit's one-sentence result:** the intra-layer walls hold
essentially everywhere; every serious leak is an ENTRY or
CONSUMPTION boundary trusting its caller for standing the
constitution assigns to mechanical verification — the submitter
bundle at S1, the legacy service beside the chain, and L11's
grounding inputs at S9/X1/X2. Two of the three auditors
independently converged on the identical S9 core finding.

## Class 3 — GENUINE CROSS-LAYER GAPS (owner decision required)

**G1 — Deployment authority anchoring (S1; auditor F-2).** The
governed entry point verifies internal consistency of a
SUBMITTER-ASSEMBLED bundle: grant ⊆ a ceiling the submitter chose,
spec ⊆ an exec-ceiling the submitter chose, every governing artifact
referenced by submitter-supplied path; no submitter identity exists
in the record. No layer owns "which ceiling/registry/catalog
artifacts are THE deployment's governed set" — the gap lives between
L7 assembly and Themis governance. Bounded today only because
nothing production-wires SubmitTask. Decision needed:
deployment-pinned governed-artifact roots / signed or
catalog-bound ceilings / submitter authentication at the adapter —
its own boundary grill.

**G2 — The "established fact" boundary inside the L6 object plane
(X1; auditor F3).** L6 mints existence-and-integrity, never
measurement meaning; nothing anywhere records which L6 objects
(executor-minted vs model-authored) may serve as which L11 fact
kind. Even with all grounding defects fixed, a criterion selecting
genuine model-authored run artifacts launders assertions into Δ; the
only wall is human registration review with no schema support. The
hole is BETWEEN L6's object vocabulary and L11's fact vocabulary —
neither constitution covers it. Decision needed: a fact-provenance
boundary mechanism (e.g. event-anchored establishment classes, or a
per-source "minting authority" declaration reviewed at criterion
registration) — its own grill; must not be folded into the D1
defect fix.

## Residual re-recordings (owner ratification required — scope has
moved; no silent widening/narrowing)

**R1 — Legacy service endpoints (S1; auditor F-1, HIGH).** The
shipped production binary (cmd/themis-serve → internal/service)
invokes models entirely outside the L1–L11 chain: caller evidence
template-interpolated into prompts, no EIS/contract/record, and
caller-selected models can route evidence to registered remote
endpoints ungated. Its recorded exemption ("untouched until L9")
EXPIRED at the L9 archive. Options: decommission / rebase onto the
L7 seam / re-record with explicit boundaries.

**R2 — Evidence-payload commingling (S5; auditor M-1).** The whole
L11 instance plane + L10 discrepancy artifacts are durable objects
with NO referencing event, hence no durable classification or
provenance — the enabling condition for D1, present-tense. The
recorded L6 GC-anchoring residual covers only future-GC retention:
narrower than the observed consequence. Re-record: the anchoring ADG
must restore event-carried classification, not only retention roots.

**R3 — Sensitivity residual deployment coupling (X3; auditor F9).**
The residual's acceptability rests on "local single-user store", but
ungoverned models.json can route conversations (including packages
the model reads as Class-6 data) to external endpoints with no
deterministic egress gate at that crossing. Re-record: "sensitivity
residual valid only under local-endpoint deployments" (or take the
egress-gate work).

**R4 — Two-gate phrasing (S7; auditor L-2).** Registration-precedes-
evaluation is enforceable only for skill-attributed execution; a
hand-assembled candidate can be run and compared with no
deterministic refusal (each individual wall holds; a candidate
registry is a FORBIDDEN component). Re-record D-L9-16's "no
evaluation backdoor" as a VISIBILITY guarantee, not structural.

## Implementation defects (fixable under the frozen constitutions)

The L11 grounding cluster (two auditors converged; remediate as one
Class-3 pass):
- **D1 (HIGH; S9/S6/X1 — sec F2 = S5S7 H-1):** L6-plane grounding is
  plane- and event-blind: any evidence-payload object grounds as
  l6_execution_record / l10_evaluation_record from a caller-asserted
  source label — including model-authored turn objects and L11's own
  packages (the terminal-output wall bypassed at the fact layer);
  reconstruction confirms the same shape. Fix: establishment via the
  event plane — L6-plane facts must be anchored to a committed event
  of a named task (EvidenceRef gains task/event binding; GroundFacts
  gets event access; refuse l11-* artifact bytes as facts;
  reconstruction re-verifies anchoring). Schema touch on a v1
  artifact under locked obligations (D-L11-15/D-L11-4 §1, D-L6-9) —
  Class-3 implementation, not new architecture.
- **D2 (HIGH; X2 — sec F1):** benchmark_validated_score /
  gate_verdict ground as bare files under --evidence-root with none
  of the gate-admission discipline the router honors. Fix: ground
  these sources through the verdict+digest+location predicate (the
  gatePassed pattern), or strip "validated" from the source name.
- **D3 (MED; S9 F4):** --door-registry is an arbitrary unpinned
  path. Fix: --door-registry-sha256 consumption pin (ratified
  pattern) and/or a closed door→location table.
- **D4 (MED; S9 F5):** run_identities are unverified caller text
  inside the authoritative proposition. Fix with D1's anchoring.
- **D5 (MED; S9 F6):** door.go leniency diverges from owning
  loaders (last-duplicate-wins; malformed siblings ignored). Fix:
  owning-format entry validation file-wide; leniency only for
  unknown fields. Plus: reword CurrentActive as an L11 derivation,
  not an observation.

Other defects:
- **D6 (MED; S2/S4 back-edge — F-3):** tool results re-enter the
  conversation unframed (D-L4-7): add fence + authority label at
  the result re-entry.
- **D7 (MED; S1/S2 — F-4):** instructions/themis root structurally
  unwireable; L2 authority labels reach the model without their
  interpretive half. Fix: add the root to orchestrator Config +
  mandatory-root set.
- **D8 (LOW; S7):** execute the recorded L9 loader hardening
  follow-up (duplicate-key wall — reuse the ratchet helper).
- **D9 (LOW; S4 — F-5):** add the CI lint Q-L5-5.2 claims (executor
  ambient-env prohibition).
- **D10 (LOW; S1 — F-6):** grant staging via shared host tmp —
  stage under the state root.
- **D11 (LOW; S8 note b):** loader error strings (incl. paths)
  surfaced to the model in refusal details — sanitize.
- **D12 (LOW; S6 — L-1):** reconstruction reports carry no marker
  when the task is TORN — add verdict consultation or a marker.
- **D13 (NOTE; S8 note a):** canonReport PASS = well-formedness —
  an authoring-surface caution to record in workflow-authoring
  guidance, not a code change.

## Verified-held negatives (carried for the record)

Deterministic authorization tables, confinement, seal, disjoint
roots (S3/S4); record-before-effect at every producer; single
identity algebra across all three L6 consumers; δ steered only by
declared gate semantics; replay unaffected by L11 objects;
cross-task gate satisfaction structurally impossible; catalog
walls + sealed composition + TOCTOU closure (S7); L10 seam drift
refusal + closed vocabularies (S8); Governance activation verified,
nothing machine-consumes L11 packages (S10); benchmark/L11
single-home (X2); no L5 verifier-execution creep; R-L9-1 not
widened; no third l2-delivery semantic.

## Proposed Phase B disposition (awaiting owner)

1. G1, G2 → each gets its own boundary grill (owner-led, the
   established pattern). G1 first — it gates any production wiring
   of the governed chain.
2. R1 → owner chooses decommission / rebase / re-record.
3. R2, R3, R4 → ratify the re-recordings as written above.
4. D1–D12 → remediation pass under the frozen constitutions
   (Class-3 review discipline for D1/D2), sequenced: L11 grounding
   cluster first (D1–D5), then D6/D7, then the LOWs.
5. Phase C (live end-to-end walk) runs AFTER the D-pass, so the
   walk exercises the corrected seams.

## Phase B disposition (OWNER VERDICTS, 2026-09-13)

| Item | Decision |
|---|---|
| G1 | GRILL FIRST — no production wiring until closed |
| G2 | GRILL SECOND — sharpened statement below |
| R1 | DECOMMISSION / REBASE — never re-record as accepted; removal preferred unless a real consumer requires the interface, in which case rebase onto the L7 seam, never preserve old semantics |
| R2 | RATIFIED — the ADG must restore event-carried classification; "object exists + retained = established" is prohibited |
| R3 | RATIFIED with explicit applicability: valid ONLY under local-endpoint deployments; cross-endpoint execution requires sensitivity inheritance before sensitive-plane selectors admit |
| R4 | RATIFIED — "no evaluation backdoor" stays classified a VISIBILITY guarantee, never upgraded to structural |
| D1–D13 | Remediate AFTER G1/G2 close — D1 must not pre-decide G2 (G2 defines who can establish a fact; implementation then enforces that boundary, never the reverse) |
| Phase C | Runs only after the defect pass |

**G2 sharpened problem statement (owner):** L11 has a vocabulary of
fact references, but the architecture has not established a
sufficiently authoritative mapping between an L6 object and the
fact class L11 may treat as established evidence. Durable storage
does not make an established fact; an ObjectID proves identity of
bytes, not epistemic authority of bytes. The grill must establish
the fact taxonomy (executor-minted, model-authored, L10 outcomes,
L6 execution records, external, derived, L11-produced) and WHICH
MECHANISM establishes that a particular object is eligible to
satisfy a particular L11 fact selector. PROHIBITED solution shape:
an `established_fact: true` boolean on L6 objects — that merely
moves the trust problem; authority must come from the minting/
establishing mechanism with provenance sufficient for L11 to
verify eligibility.

**Preserved headline (owner):** the audit did not invalidate
L1–L11. It demonstrated that the completed layers need two
additional cross-layer authority boundaries before the chain can be
considered production-wired.


## D1–D5 remediation record (2026-09-13, enforcement of D-G2-1)

Implemented strictly as G2 enforcement (owner order — the boundary
was defined first; no design work occurred in the defect pass):

- **D1** — witness grounding: EvidenceRef carries {task_id,
  event_seq}; GroundFacts resolves the witness in the event plane
  (exists → class per witnessClasses → names the object → bytes
  match), refuses model-turn-witnessed and l11-* bytes; Compare
  refuses L6-plane facts without witness fields; reconstruction
  re-walks every witness (ReconstructInputs gains Root/BenchRoot;
  absent planes = missing-inputs, refused witnesses =
  discrepancies). witnessClasses: l10_evaluation_record →
  l10-verification (record named in body — the anti-shadowing rule
  generalized); l6_execution_record → l4-audit (object in event
  refs; witness_tool selector param pins the minting capability).
- **D2** — benchmark witness: verifyBenchWitness applies the
  verdict + digest + location predicate (BenchRunDigest = the gate
  contract); unadmitted/failing/digest-broken/off-layout runs
  refuse; no L6 event invented for benchmark facts (owner
  instruction).
- **D3** — --door-registry-sha256 consumption pin at the CLI
  (mismatch = durable integrity-failure refusal).
- **D4** — run identities are DERIVED from witnesses (task:<id> /
  bench:<date>/<model>), never caller text; the --runs flag is
  gone; reconstruction re-derives and compares.
- **D5** — door.go strict file-wide validation (duplicate identity,
  malformed entry, unknown state → whole-file refusal; leniency
  only for unknown fields); CurrentActive documented as an L11
  derivation over observed entries.

Proofs: TestGroundFacts (7 witness cases incl. wrong-class,
un-named object, wrong tool, l11-bytes), TestGroundFactsBenchWitness
(admitted/failing/rewritten-after-gating/off-layout), CLI contract
re-run with bench fixtures, cross-process reconstruction CONFIRMED
against witnessed packages, full ratchet+CLI+state+verification
suites green, live proof re-run PASS.

Remaining: D6–D13 (next), full negative-space sweep, then Phase C.
