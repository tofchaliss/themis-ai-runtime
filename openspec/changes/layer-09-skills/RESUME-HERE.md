# L9 — Resume state (paused 2026-09-08)

Work paused mid-gate to fix a bug in the Themis application. This file is the
handoff: where L9 stopped, what is decided, and exactly what happens next.

## One-line status

**L9 is IMPLEMENTED and NOT COMPLETE.** All code and tests are written, all
three Class-3 reviews have been re-run against the final implementation, and
every CRITICAL and HIGH finding is remediated and mutation-verified. What
remains is the owner's acceptance and a governance act — not engineering.

## Gate state

| Gate | State |
| --- | --- |
| Architecture-conformant | ✓ (final review: conformant on all locked decisions) |
| Security | ✓ (2 HIGH + 2 MED found and remediated) |
| Operationally-proven | ✓ (live walk vs qwen2.5:7b through the unmodified L7 loop) |
| D-L9-11 / C2 | ✓ decided and implemented |
| R-L9-1 (reconstruction provenance) | ✓ CLOSED |
| R-L9-2 (grant + procedure bytes) | ✓ implemented, cross-layer proven |
| Test-evidence | **awaiting owner acceptance** — last blocker (C-1) fixed and mutation-verified |
| Governance registration | **OPEN — owner act, never the machinery's** |

`investigate-cve@1` remains a **proposed** Skill in
`policies/skills/catalog.proposed.json`. It is NOT registered and must not be
treated as executable governed material.

## Verification at the pause point

Whole module green — every package, both live proofs included. Coverage:
skills 84.8% · orchestration 82.4% · instructions 95.2% · state 83.7%.
`gofmt` clean, `go vet` clean.

Note: `context/` can fail under full-suite CPU contention (a pre-existing L2
live test with a 120s client timeout, unrelated to L9). It passes in isolation.
Not touched.

## Git

**41 commits unpushed on main**, tree clean. The L9 sequence runs
`415ce1d` → `e9314d9`. Nothing has been pushed at any point; every push in this
project requires explicit owner approval.

## The four things that remain

1. **Owner acceptance of the three-state verdicts** (architecture-conformant ·
   test-evidenced · operationally-proven), challenged independently.
2. **Governance registration** of `investigate-cve@1` — moving it from
   `catalog.proposed.json` into the governed catalog. This is a Themis
   governance act; the skills package has no write API and an AST audit
   enforces that it never gains one.
3. **traceability.md** — written at archive, mapping each decision to its
   mechanism, register, test, and evidence (see the L7 archive for the format).
4. **Code map + status doc + artifact updates**, then push/archive on approval.

## Decisions a newcomer must not re-derive

Read `design.md` §2 for D-L9-0..17. The four amendments that cost the most to
reach, and which must not be quietly reinterpreted:

- **D-L9-11a (C2)** — L9 is the sole semantic resolver; L7 stays skill-blind.
  Execution integrity is closed; **governance identity is explicitly NOT**.
  The phrase "reviewed composition = executed composition" is unqualified-false.
  The truthful form: *the executed artifacts are cryptographically consistent
  with the composition submitted by L9.*
- **D-L9-11b/c/d** — the commitment carries a self-seal (without it the design
  degrades to the rejected option A); scope is all Skill-fixed material
  INCLUDING input_schema, EXCLUDING registry and exec_ceiling; skill
  attribution REQUIRES a commitment, so C2 is not submitter-elective.
- **R-L9-1 / ADG-L9/L6-1** — reconstruction needs a source the envelope cannot
  supply. L6 derives object addresses by hashing bytes itself; L7 stores the
  bytes it verified. No L6 vocabulary amendment was needed.
- **R-L9-2 single-home clarification** — single-home governs SEMANTIC/IDENTITY
  ownership, not physical durability. L1 keeps procedure identity (BodyHash),
  L6 owns durable bytes (ObjectID); these are different things even when both
  are SHA-256 of the same bytes.

## Recorded residuals (never silently promote these)

C3 signed governance attestation · governed multi-Skill pipelines (L8-vs-L9
ownership OPEN) · L8 delegation · runtime revocation/cancellation · approval
channel · repository/external skill activation · model-property predicate ·
richer input schema · evaluation-purpose attribution · proposal lifecycle ·
deterministic verification capabilities (OPEN-2/L10) · expected-outputs
contract · the event-class boundary (a THIRD meaning on the delivery class
forces an L6 vocabulary revisit rather than more consumer-side filtering).

## The methodological lesson, recorded as evidence

Three review rounds each found defects that passing tests concealed. In every
round the decisive tool was **mutation testing** — breaking the code and
confirming the test still passed. Green tests did not establish the invariants
they claimed; several documented intent rather than verifying it. Specifically:
a "the procedure reached the model" test passed with the procedure removed from
the render entirely; a catalog-write guard was defeated by a renamed function,
then by an import alias; and the equivalence register missed a provenance-driven
budget change because its script saturated the wrong dimension.

Treat "the suite is green" as the beginning of the evidence question, not the
end of it.
