# Proposal: L1–L11 Integration Audit (constitutional)

Status: OPEN 2026-09-13. Owner-selected next work category (#3
architecture integration). This is an AUDIT, not a layer design: no
individual layer grill reopens; the question is whether the COMPLETE
chain preserves its authority boundaries when exercised as one
end-to-end Themis workflow.

## The single audit question

Does the complete L1–L11 chain preserve its authority boundaries
when exercised end-to-end — task initiation → instructions → context
→ tools → execution → durable evidence → orchestration → procedure →
verification → Ratchet evidence → Governance — or does authority
leak between layers at the seams the individual grills never
examined together?

## Finding classification (fixed in advance, the D-L11-20 rule)

Every finding lands in exactly one class:
1. **Implementation defect** — a seam violates an already-locked
   decision; fix under the frozen constitution.
2. **Recorded residual** — a known, explicitly recorded gap; verify
   it is recorded, never silently widen or narrow it.
3. **Genuine cross-layer gap** — a hole no layer's constitution
   covers because it lives BETWEEN layers; only this class produces
   a new architecture decision, with its own grill.
Findings must not be reclassified downward for convenience.

## Leak taxonomy (what "authority leakage" means, closed)

- **Upward leakage:** a later layer writing or mutating an earlier
  layer's governed state (registries, catalogs, instructions,
  records).
- **Advisory→control leakage:** model-authored content reaching any
  control decision without a deterministic wall at the crossing.
- **Vocabulary leakage:** one layer's semantic tokens acquiring
  control meaning in another layer (COMPLETED≠PASS≠better≠promoted;
  L10 outcomes as L11 states; scores as authorization).
- **State leakage:** a second home for a truth some layer already
  owns (duplicated state demanding reconciliation).
- **Initiation leakage:** execution/initiation authority arising
  anywhere outside the governed entry points (including chains of
  individually-legal steps composing into an initiator).
- **Egress leakage:** data crossing the local boundary outside
  policy gates (including indirect routes: evidence embedded in
  artifacts that later egress).
- **Identity leakage:** an identity/hash minted in one layer being
  trusted in another without re-verification at the crossing.

## Seam inventory (the audit walk)

| Seam | Crossing | Authority question |
|---|---|---|
| S1 | task initiation → L1 instructions | who may initiate; instruction resolution authority; conflict handling |
| S2 | L1 → L2/L3 context | context contract binding; delivery policy; untrusted content marking |
| S3 | context → L4 capability authorization | registry authority; grants; model proposals vs deterministic authorization |
| S4 | L4 → L5 execution | confinement; environment; secrets isolation |
| S5 | execution → L6 durable evidence | record-before-effect; object identity; event vocabulary ownership |
| S6 | L6 → L7 orchestration | δ consumes typed events only; totality; replay |
| S7 | L7 → L9 skills | catalog resolution; composition pinning; two-gate discipline |
| S8 | walk → L10 verification | verifier eligibility; contract gates; outcome vocabulary containment |
| S9 | records → L11 comparison | established-facts-only; door observation; terminal output |
| S10 | L11 evidence → Governance doors | evidence carriage without obligation; *.proposed.* handoff; activation authority |
| X1 | model I/O across ALL seams | the one trust boundary, checked at every crossing |
| X2 | benchmark plane ↔ router ↔ L11 | Class-2 consumption conditions; single-home for compare semantics |
| X3 | egress (knowledge packages, reports) | policy gating; sensitivity inheritance residual boundaries |

## Method

Phase A — three parallel seam audits (Class-3 review agents) against
the archived constitutions + live code, S1–S4 / S5–S7 / S8–S10+X1–X3.
Phase B — synthesis: dedup, classify per the fixed classes, owner
review of any class-3 findings.
Phase C — integration proof: one live end-to-end walk exercising the
full chain in a single workflow, if Phase A/B leave it justified.

Authoritative sources: the ten archives under
openspec/changes/archive/, ARCHITECTURE.md, .claude/policy/DAY-0.md.
The audit assesses against them; it never redefines them.
