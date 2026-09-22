# Design: G2 — The Established-Fact Boundary

Grill OPEN 2026-09-13 → **CLOSED 2026-09-13: D-G2-1 LOCKED (owner)**.

## D-G2-1 — Establishment is event-witnessed minting (LOCKED)

**Principle (Q-G2-1):** storage proves bytes; events prove
establishment. An L6 object is an established fact of kind F iff a
committed typed event of the class that mints kind F names that
object. The event class is the establishing mechanism's signature;
the event body is the minting provenance. Nothing about the object
itself — content, shape, address, or any field — establishes
anything.

**Owner consequence, fixed for implementation:** *L11 does not
determine whether an object is an established fact from the object
itself. It determines eligibility by resolving and verifying the
authoritative witness belonging to the mechanism that established
that fact.* D1–D5 are strictly implementation ENFORCEMENT of this
decision — no further design work, no reopening.

**Taxonomy (Q-G2-2):**
| Fact kind | Established by | Witness |
|---|---|---|
| l10_evaluation_record | deterministic L10 evaluator | committed EvVerification event whose body's `record` field names the ObjectID |
| l6_execution_record | L4/L5 executor under a registered capability | committed EvL4Audit event referencing the ObjectID; body carries Tool + RegistryHash (selectors may pin the tool) |
| model-authored bytes | nothing — EvModelTurn witnesses authorship (advisory) | NOT a fact kind; no selector source exists structurally |
| benchmark_validated_score / gate_verdict | the benchmark plane's own admission mechanism | verdict + digest + location predicate (the gatePassed discipline) — the external plane's witness; do NOT invent an L6 event for benchmark facts |
| derived (better-under-K, views) | not facts | derivations (locked) |
| L11-produced packages | terminal, never facts | no witnessing event exists by construction — unwitnessed is unestablishable |

**Mapping home (Q-G2-3):** the fact-kind → witness-class table is
architecture, implemented as a closed in-code table (reviewed Go,
never data). No per-object declarations anywhere.

**Mechanical verification (Q-G2-4):** EvidenceRef gains the witness
(task identity + event sequence). Grounding verifies: event exists →
class matches the kind's witnessing class → body names this
ObjectID → bytes match. Any broken link → refusal, never a weaker
package. Grounding requires event-plane read access.

**Cold reconstruction (Q-G2-5):** the package records each fact's
witness; reconstruction re-walks the chain from the named task's
event stream. Stream unavailable → missing-inputs; witness
absent/mismatched → discrepancy. Three-result discipline unchanged.

**Prohibited shapes (Q-G2-6):** establishment by storage (bare
ObjectID); by assertion (caller source labels); by content shape;
per-object booleans/classification fields; L11-side object
allowlists; establishment by human memory at registration review.

**Constitutional sentence:** *Storage proves bytes; events prove
establishment. An object is a fact of kind F only when the
mechanism that mints F witnessed it in the committed event plane —
and L11 verifies the witness, never the assertion.*

**The cross-layer rule now complete (owner, at lock):** Governance
establishes what may govern (G1); the owning mechanism establishes
what is a fact (G2); L11 compares only mechanically established
facts.
