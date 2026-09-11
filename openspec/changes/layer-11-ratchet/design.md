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

### D-L11-2 — The improvement ladder (LOCKED 2026-09-11, Q-L11-2; two owner amendments)

Four epistemic positions for an improvement candidate — Observed,
Evaluated, Promoted, Relied-upon. No position implies the next; each
upward transition is performed only by its owning mechanism; no
mechanism may perform a skip (1→3, 2→4, or otherwise). The Ratchet may
establish comparative and regression evidence; it never promotes,
activates, or makes an artifact relied-upon.

**1. Observed** — a claim or observation that something may be better
(AI suggestion, human observation, single execution outcome, informal
metric, unexpected behavior). Authority: none. A governed task's
COMPLETED or L10 PASS is still only position 1 with respect to
improvement — success is not betterness; the leak "PASS → the current
thing works → the new thing is better" is closed here. AI authorship →
Observed, nothing more.

**2. Evaluated (owner tightening)** — *a reproducible comparative
result showing a candidate/baseline difference under a registered
comparison criterion, with both artifacts/runs admitted through their
normal governance mechanisms.* The Ratchet establishes "C compared
with B under K → Δ"; it does not establish "C is better" unless
"better" is itself precisely defined by the registered criterion.
Required conditions, all of: candidate identity; baseline identity;
both registered/admitted; both executions governed; registered
criterion + version/hash; comparable input/evidence; run identities;
deterministic Δ computation; complete provenance sufficient to
reproduce. **Baseline rule, constitutional (the CheckBaseline lesson
as ladder law): an unadmitted baseline cannot establish position 2** —
otherwise an apparently excellent improvement can be manufactured
against an artifact never legitimately allowed to govern.

**3. Promoted** — solely an act of the existing owning Governance door
(Skill → catalog; Instruction → L1 governance; Contract → registry;
Knowledge → Themis ingestion). The Ratchet supplies the evidence
package; the door decides; evidence never obligates. **Promotion
without position-2 evidence is permitted** — Governance authority does
not derive from Ratchet evaluation. **Amendment A: where no position-2
evidence exists, the promotion record must not imply that comparative
evaluation occurred; evidence absence is explicit** (e.g.
comparative_evidence = absent), closing the backdoor where a bare
"promoted" is later read as "the Ratchet found it better."

**4. Relied-upon** — the normal consumption machinery actually causes
the artifact to govern work (skill resolved and instantiated; revision
becomes the active instruction set; contract actually selected;
regression test in the effective evaluation set). Promotion
establishes admissibility/executability, never reliance; a promoted
artifact may never be used. Position 4 is an established fact about
consumption, not another Ratchet decision.

**Non-implication matrix (constitutional):** 1⇏2 (observation isn't
comparison) · 2⇏3 (no auto-promotion) · 3⇏4 (registration isn't
reliance) · 4⇏2 (incumbency is not evidence) · 3⇏2 (promotion doesn't
manufacture evaluation) · 2-under-K ⇏ generally-better (Δ is
criterion-relative) · 4 ⇏ good (usage isn't validation). An artifact
can be relied upon for years with no valid comparative evidence.

**Downward movement (Amendment B):** *withdrawal, demotion,
retirement, or replacement are acts of the owning
governance/consumption mechanism. Ratchet evidence may support such an
act but never triggers it.* The ladder records positions established
by their owning mechanisms; it is not an L11 state machine and must
not become one.

**Uniformity:** the ladder is identical in shape across candidate
families (Skill, Instruction, Contract, Regression test, Knowledge);
the mechanism at each arrow changes by family, the proposition being
established does not.

**Vocabulary rule (owner precision):** in governed artifacts, *better*
is not a free-standing proposition — it is shorthand only for a
position-2 comparative result whose criterion and baseline are
explicitly identified ("C produced Δ under K against admitted baseline
B"; "C is better than B under K" only where K defines the ordering).
Prohibited semantic upgrades: PASS→better, COMPLETED→better,
PROMOTED→better, RELIED-UPON→better.

**L10/L11 relationship:** L10's PASS can be an INPUT to an L11
comparison; L11 cannot reinterpret PASS as "better."

**Constitutional invariant (carried):** *the Ratchet may establish
that an improvement is comparatively supported under a declared
criterion; only the existing owner of the affected artifact can make
it governed behavior.*

### D-L11-3 — The Candidate (LOCKED 2026-09-11, Q-L11-3; owner amendments)

A Candidate is a content-addressed proposal package representing a
possible change to a governed artifact or governed knowledge. It is
data, not a runtime artifact and not a source of governance truth. It
identifies its family, target, proposed content commitment, admitted
baseline, authorship/provenance, rationale, and any attached
comparative evidence.

Candidate existence confers no authority and creates no evaluation or
execution path. **Executable candidates must pass the affected
artifact's ordinary registration mechanism before they may participate
in governed evaluation; there is no trial-registration, sandbox, or
evaluation-only admission path** — that would be a second admission
channel, exactly what D-L9-16 forbids. Registration-for-evaluation is
ordinary registration under the ordinary rules: "admissible for
governed execution," never "endorsed as better."

**The ladder is epistemic, not temporal (owner: 1→3→2 PROVES the
ladder is epistemic, not an exception to it).** Executable candidates
legitimately proceed 1 → 3 (registered) → 2 (evaluated by ordinary
governed tasks vs the admitted baseline); non-executable candidates
may proceed 1 → 2 → 3, subject to actual family semantics. All
non-implications hold in every direction regardless of temporal order.
**No implication between 2 and 4 (owner tightening):** Governance/
deployment may decide 3+2→4, 3+2→not-relied-upon, or 3→4 without
position-2 evidence — discretion retained; evidence informs, never
obligates; a candidate may never reach 4 at all.

**Single-home promotion truth:** the Candidate does NOT record
promoted, active, relied-upon, or withdrawn governance state —
promotion truth lives solely at the affected door (two representations
of one governance fact would demand reconciliation semantics L11 must
not own). **A Candidate has no authoritative lifecycle state at all
(owner amendment):** proposal disposition, if recorded, is owned by
the mechanism that receives it, and any practical "withdrawn" marker
is proposal disposition, never governance state, never confusable with
withdrawal of the promoted artifact. **Lineage is not state:**
candidates may reference candidates (supersedes) — answering "where
did this proposal come from?", never "is this proposal governed?"

**Candidate contents (closed, minimal):** family (skill-revision |
instruction-revision | contract-revision | regression-test |
knowledge); target descriptor — target identity + a closed declarative
change relation (create-version | revise | replace), introduced only
as the existing doors actually need it and never interpreted by L11
into authority; proposed content commitment (SHA-256 pins); admitted
baseline reference; authorship/provenance (AI provenance raises
attention, never lowers the bar); advisory rationale; optional
position-2 evidence references; optional lineage references.

**No candidate machinery:** no candidate executor, no candidate
registry with execution authority, no candidate runtime, no
candidate-specific authorization, no candidate-specific evaluation
engine — the existing walls do the work (unregistered = data,
D-L9-8). **Proposal handoff (owner amendment):** Governance receives
proposals through the existing governed proposal/registration
mechanisms; L11 creates no parallel admission channel. The *.proposed.*
convention is the v1 REPRESENTATION of proposal handoff where
applicable — never an independent authority mechanism, and not
constitutionalized as a filename pattern.

**Knowledge:** the authoritative door is outside the Harness —
candidate → L11 evidence package → governed egress → Themis knowledge
ingestion; successful handoff never means "knowledge promoted"; the
Themis-side mechanism owns that fact.

**D-L9-16 residual disposition:** the "no proposal artifact in v1"
deferral is matured here — the formal Candidate exists from L11 v1;
v1 families: regression-test, skill-revision, contract-revision,
instruction-revision (in-repo doors), knowledge as egress package.

**Architectural principle:** *the Candidate describes what might
change; the existing door decides whether it may govern; the Ratchet
establishes what comparative evidence exists; the existing consumption
mechanism determines whether it is relied upon.*

### D-L11-4 — Comparison criteria and comparative evidence (PROPOSED 2026-09-11, closing Q-L11-4; AWAITING OWNER LOCK)

1. **The Comparison Criterion K** is a governed, registered, versioned,
hash-pinned declarative artifact — the L11 sibling of the L10
Verification Contract, for a different question. Closed schema: exact
identity (name@version + SHA-256); evidence selectors for candidate
and baseline sides (typed, declarative, closed vocabulary); a
**registered comparator binding** — the deterministic Δ computation,
registered per criterion family as in-harness code identity (the
canonicalizer pattern from D-L10-12: pure, environment-free, reviewed
harness code — never an expression language, never criterion-carried
logic; D-L9-4 zero-interpreters a third time); pinned configuration;
the Δ output shape; provenance requirements. Governance-registered;
unregistered criterion-shaped artifacts are data; append-only, exact
pins, the same walls.

2. **Comparisons consume only established governed facts** — L10
evaluation records, benchmark validation scores, L6 execution records,
gate verdicts. A comparison never executes anything, never re-judges
evidence, never mints semantic judgment: arithmetic and aggregation
over facts whose meaning was fixed where they were made. Runs are
ordinary governed tasks (D-L9-16 gate 2) or benchmark-plane runs,
identified by their records. This is the mechanical content of "no
second evaluation subsystem": L11 adds a comparator OVER evidence,
never an evaluator OF executions.

3. **Δ is typed comparative evidence, never vocabulary.** Output =
content-addressed comparative-evidence package: K@v + hash; candidate
identity + evidence record refs; admitted-baseline identity + an
**admission attestation** (door state observed at comparison time —
the ladder's baseline law made mechanical: no attestation, no position
2); run identities; Δ in the declared shape; provenance sufficient to
re-derive Δ cold (D-L10-10 discipline). Δ never enters any control
vocabulary, L7 event, L10 outcome, or routing decision — it exists to
be carried to a door by a human.

4. **Reproducibility as registration bar:** same criterion + same
referenced records ⇒ same Δ (D-L10-3 reapplied). Cold re-derivation
disagreeing with a recorded Δ = discrepancy fact for Governance, never
a rewrite (D-L10-12 inherited).

5. **Variants generalize:** prompt A/B = "instruction-family candidate
vs baseline under the benchmark-score criterion"; a variant is a
candidate; no separate variant mechanism in L11 (physical moves settle
in Q-L11-9).

6. **A comparison is NOT:** a gate, a verification, a selection input
(Q-L11-8 quarantine), or automatic (produced on request, no daemon
watching history).

Flagged edge: evidence selectors are the cherry-picking surface —
declarative + reviewed at registration; the package records exactly
which records were consumed; selection within the declared frame is
visible, not prevented (adversarial hardening in Q-L11-11).

Open decision for owner: comparator as registered in-harness code
identity (proposed, the canonicalizer precedent) vs a declarative Δ
language inside the criterion (rejected by implementer as an
interpreter by another name; the benchmark gate's Compare/AverageDrop
shows the proposed shape).

## 3. Grill — question list (OPEN, implementer draft — owner may restructure)

| Q | Question |
|---|---|
| Q-L11-1 | **CLOSED → D-L11-1.** Evidence-not-authority boundary; provides-not-enforces at existing doors; ownership split; L10/L11 question separation; hard prohibitions + positive authority. |
| Q-L11-2 | **CLOSED → D-L11-2.** Four positions w/ owning mechanisms; baseline rule as ladder law; evidence-absence explicit at promotion; downward moves are acts; criterion-relative vocabulary. |
| Q-L11-3 | **CLOSED → D-L11-3.** Candidate = content-addressed proposal data; no lifecycle state; single-home promotion truth; 1→3→2 for executable families (epistemic ladder proven); no trial admission; *.proposed.* as representation only. |
| Q-L11-4 | **Closure PROPOSED → D-L11-4 (criteria + comparator bindings + evidence packages); AWAITING OWNER LOCK** — one open decision: comparator as registered code identity vs declarative Δ language. |
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
