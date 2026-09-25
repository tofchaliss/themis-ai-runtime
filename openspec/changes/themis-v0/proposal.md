# Proposal: Themis v0 — the read door and the decision door

Status: grill OPEN 2026-09-25 (owner-led). Scope LOCKED by the owner
before the grill: **A — read door + decision door**, minimal Themis.

## Why

The harness (L1–L11, G1, G2) is complete and operationally proven, but
`src/themis/` is empty. The harness holds Themis only as seams: the
`ThemisSeam`/`ThemisReader` read boundaries (nil today), the
`get_finding`/`get_product` capabilities at `governed-record` trust, and
the four-class authority vocabulary whose top three classes only Themis
can mint. Nothing can produce a Governance-established fact; the G2
table has no Themis row because the door does not exist.

The demo's core claim is not that a model can consume governed data. It
is that **Themis remains the authority for security truth while the
harness remains bounded execution infrastructure.** That claim needs a
real Themis, however small, on both sides of the boundary.

## What Themis v0 is

A system of record for Findings, Products, and Enterprise Positions,
with two doors:

1. **The read door.** A Themis-owned store, anchor-pinned, served to the
   harness through the existing `ThemisSeam` and `ThemisReader`. A
   Finding reaches a model as `governed-record`; whatever the model says
   about it re-enters at the floor (C-L8-18 laundering rule, live).
2. **The decision door.** A Themis-owned intake through which a
   **human** accepts a harness task's verified egress artifact into an
   Enterprise Position. The harness has no capability, executor, seam,
   or import for it; the model never touches it; `themis-run` never
   touches it. Only an operator command does, and the act is witnessed.

Three facts stay separate and are never collapsed into one "approved
report" object: the **harness fact** (this task bound this artifact —
`artifact-bound`), the **verification fact** (L10 evaluated it —
`l10-verification`), and the **governance fact** (an authorized human
accepted it — the Themis decision record).

## What Themis v0 is not (owner, locked)

Vulnerability-feed ingestion · Knowledge Builder · CVE enrichment ·
automated remediation · AI-generated security truth · generic harness
write access · a second workflow engine · a second verification engine
· Themis-owned execution · automatic acceptance. The demo proves the
authority boundary; nothing more is built.

## Grill questions

| # | Question | State |
|---|---|---|
| Q-T-1 | What constitutes a Themis-referencable harness execution? | LOCKED 2026-09-25 (D-T-1) |
| Q-T-2 | Which deployment identity is authoritative, and how does Themis resolve it? | LOCKED 2026-09-25 (D-T-2) |
| Q-T-3 | Which L6 object/event combination identifies the artifact? | folded into D-T-1 |
| Q-T-4 | How does Themis verify the artifact was produced by that governed execution? | LOCKED 2026-09-25 (D-T-4) |
| Q-T-5 | How does Themis verify the L10 result rather than trusting a claim? | LOCKED 2026-09-25 (D-T-5) |
| Q-T-6 | What happens when the task, anchor, artifact, or verification record is withdrawn or unavailable? | open |
| Q-T-7 | What exact act creates the Enterprise Position? | open |
| Q-T-8 | How is the human decision itself witnessed? | open |
| Q-T-9 | The read door: what is served, under which class, pinned how? | open |
| Q-T-10 | Where does Themis live (package, store, command) and what walls prove the harness cannot write? | open |

## Boundaries locked with Q-T-1 (owner, 2026-09-25)

- **Human decision only.** The decision door is structurally
  `HumanDecision`, not `Decision = Human | Automated`. A governed
  automated decision is a separate mechanism with its own grill.
- **L10 PASS is admissibility, not correctness.** `report-valid@2 =
  PASS` establishes that the artifact satisfies the registered
  contract; the human decision is the semantic acceptance act. The
  acceptance view shows model turns, artifact bytes, and the L10
  outcome as three independent things.
- **The write boundary is proven structurally.** No Position
  capability, executor, or seam in the harness, and an import/AST wall
  proving harness packages never reference the Themis write path.
  Capability absence is a consequence of ownership, not the proof.
