# Design: Layer 11 — Ratchet

Grill OPEN 2026-09-11. §2 collects folded decisions as questions close;
§3 is the question list.

## 0. Position in the flow

The Ratchet sits AFTER outcomes and BEFORE the next task: it turns
governed execution history and evaluation evidence into candidates for
improvement, carries comparative evidence to Governance, and enforces
regression-resistance at the points where improvements are consumed. It
owns no promotion authority: every promotion is an existing governance
door (catalog registration, contract registration, instruction
revision, Themis knowledge ingestion) exercised by humans.

## 1. Hard invariants (inherited, not grillable)

- Two gates, never collapsed: registration review (safety/architecture)
  precedes ANY execution including ratchet evaluation; ratchet
  evaluation (quality) is ordinary governed tasks under registered
  artifacts — no evaluation execution mode, no special authority in
  either direction (D-L9-16).
- No auto-promotion: no harness code path may register, promote,
  withdraw, or select an artifact version from evaluation results; a
  poor score is evidence supporting withdrawal, never a trigger
  (D-L9-16). The AI can suggest how Themis improves itself but cannot
  make the suggestion become Themis behavior (D-L9-8).
- No reduced review for AI authorship; provenance may raise attention,
  never lower the bar (D-L9-16 / Q-L9-7).
- No second evaluation subsystem: the benchmark plane's
  compare/gate/variants machinery is generalized, never forked
  (locked plan; D-L10-17 extension discipline — consume through owning
  APIs or bind by registration, never house another layer's invariant).
- Scores are evidence, not vocabulary: no numeric outcome enters any
  control vocabulary (D-L10-4); COMPLETED ≠ PASS ≠ better ≠ promoted
  (D-L10-15 extended).
- Governed artifacts fail closed; nothing defaulted; append-only
  registries with immutable bindings; admission enforced at consumption
  (the verdict/digest precedent).
- L6 is the sole record plane; Themis governance owns meaning and
  acceptance; the Knowledge Ratchet reaches enterprise knowledge only
  through the appropriate Themis owner.

## 2. Locked decisions (fold target)

### D-L11-1 — Boundary and authority (LOCKED 2026-09-11, Q-L11-1; owner amendment)

The Ratchet turns governed execution history, evaluation evidence, and
approved comparison criteria into improvement candidates and
comparative evidence for Governance, and **provides
regression-resistance evidence at the existing consumption/admission
doors** (owner amendment — never "enforces": if the Ratchet were the
final gate, it would have acquired promotion authority). It owns
candidate/evidence lifecycle machinery, but owns no promotion authority
and no security meaning. Every promotion remains an existing Governance
act — skill catalog, contract registry, instruction revision, or Themis
knowledge ingestion — exercised through its existing governed
mechanism. The Ratchet may make a suggestion evaluable and establish
comparative/regression evidence for that suggestion; it can never make
the suggestion become behavior. The door remains owned by the layer
that already owns the artifact; the Ratchet can make the door
refuseable/evaluable on regression evidence, never replace it.

**Ownership split (foundational, prevents a meta-governance layer):**
Ratchet owns — comparative machinery over L6-supplied history;
candidate construction and lifecycle; baseline-vs-candidate comparison;
evaluation orchestration/consumption of governed evidence (subject to
existing mechanisms); comparative and regression evidence; provenance
for reproducible comparison. Ratchet never owns — security meaning
(Governance); Skill promotion (L9 catalog governance); Contract
promotion (L10 registry governance); Instruction revision (L1
governance); enterprise knowledge ingestion (Knowledge Builder/Themis);
candidate acceptance (the existing door); any direct runtime behavior
change.

**No second evaluation subsystem, sharpened:** L10 answers "what did
this governed verification establish?"; L11 answers "is this proposed
improvement demonstrably better and regression-resistant relative to
the approved baseline?" — a separately defined improvement/comparison
semantics that may CAUSE ordinary governed evaluation to happen and
COMPARE its resulting evidence, but introduces no new PASS/FAIL,
verifier contracts, or verification semantics, and never redefines
L10's.

**Hard prohibitions:** the Ratchet cannot activate a candidate; modify
a Skill, Contract, Instruction, or Knowledge artifact directly; create
security meaning; create an Enterprise Position; bypass an existing
governance door; weaken an existing review requirement; substitute its
own evaluation semantics for L10; create a second evaluation subsystem;
convert comparative evidence into runtime behavior by itself.

**Positive authority:** observe governed history; construct candidates;
compare baseline and candidate behavior; orchestrate/consume governed
evaluation evidence; establish comparative evidence; establish
regression evidence; expose that evidence to the existing owner of the
relevant promotion door; retain the provenance needed to make the
comparison reproducible.

**Constitutional sentence:** *Ratchet can establish that an improvement
is worth considering; only the existing owner of the affected artifact
can make it real.*

## 3. Grill — question list (OPEN, implementer draft — owner may restructure)

| Q | Question |
|---|---|
| Q-L11-1 | **CLOSED → D-L11-1.** Evidence-not-authority boundary; provides-not-enforces at existing doors; ownership split; L10/L11 question separation; hard prohibitions + positive authority. |
| Q-L11-2 | The improvement ladder: observed-better (anecdote) ≠ evaluated-better (governed comparative evidence) ≠ promoted (governance act) ≠ relied-upon (deployment). What mechanism does each transition require? |
| Q-L11-3 | The candidate: which artifact families may be ratchet candidates (skill revisions, instruction revisions, verification contracts, benchmarks/regression tests, knowledge summaries); candidate identity, lifecycle states, and the D-L9-16 proposal-lifecycle residual — is a formal proposal artifact needed now, and who owns it? |
| Q-L11-4 | Comparative evaluation semantics: baseline-vs-candidate comparison as a governed, registered computation — relation to L10 verification contracts (is a comparison a contract? a distinct registered artifact?); who defines the metric; how variants (A/B) generalize beyond prompts. |
| Q-L11-5 | Promotion mechanics: enumerate each promotion act as an EXISTING governance door; what evidence package accompanies a promotion request; is there any new registry at all (candidate registry?) and its write-wall discipline. |
| Q-L11-6 | Regression prevention: failures → regression tests (who authors, who registers, where they run); the ratchet-up rule generalized from the benchmark gate (baseline must itself be admitted — the stepwise-bypass lesson); admission-at-consumption beyond the router. |
| Q-L11-7 | Purpose attribution (assigned residual): how evaluation-vs-operational purpose is recorded — existing Class-4 deployment context, a task-attribution field, or an L6 gap needing a narrow amendment; never Skill identity or execution semantics (D-L9-16). |
| Q-L11-8 | Model involvement and the selection tension: the model may author candidates and produce evaluation walks — but the ROUTER already selects models by validated score at startup. Is benchmark-driven routing "selection from evaluation results" requiring a human-in-the-loop reading, or deployment policy below the no-auto-promotion line? Draw the line explicitly. |
| Q-L11-9 | The benchmark plane reconciliation: what generalizes into the harness ratchet, what remains themis-bench-owned; single-home for compare/gate/variant semantics; the verdict/digest admission pattern as the shared discipline. |
| Q-L11-10 | Knowledge Ratchet boundary: validated discoveries → Themis owner via governed egress only; no harness-side knowledge base, no second enterprise-knowledge store; what the hand-off artifact looks like. |
| Q-L11-11 | Adversarial register: metric gaming, benchmark overfitting, self-serving evidence (the model improving its own scores), baseline manipulation, candidate spam; which walls are structural vs review obligations. |
| Q-L11-12 | Proof gate: registers and the live slice — candidate → governed comparative evaluation → evidence package → (human) promotion via an existing door → regression admission; closed as L7/L9/L10 were closed. |

## 4. Assets inventory (for the grill, factual)

- benchmarks/internal/gate — regression gate + digest-bound verdict
  artifacts; CheckBaseline (admitted-baseline chain, this session);
  admission enforced at the router (gatePassed).
- benchmarks/internal/report — Compare (cross-model, score history).
- Prompt partials + variants (A/B) with manifest attribution.
- internal/service Router — benchmark-driven model selection at
  startup (the Q-L11-8 tension).
- L9 catalog + L10 contract registry — the append-only,
  owner-promoted registration pattern (PROPOSED → Governance act →
  ACTIVE, twice proven in-session).
- L10 verification contracts + evaluation records — governed
  deterministic evidence with complete provenance.
- D-L9-16's recorded ratchet constitution; the L9 methodological
  lesson (mutation testing) as candidate regression material.
- Scaffold: src/harness/ratchet/{candidates,evaluations,feedback,
  promotion,regression} — .gitkeep only, pre-grill guesses; reconcile
  or remove at close (the L10 scaffold lesson).
