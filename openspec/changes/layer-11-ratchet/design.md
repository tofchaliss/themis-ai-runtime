# Design: Layer 11 — Ratchet

Grill OPEN 2026-09-11 → **CLOSED 2026-09-12: D-L11-1..20 all LOCKED
(20/20)**. §2 holds the locked constitution; §3 the closed question
table. Next: Gate 0 → implementation per tasks.md.

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

### D-L11-4 — The comparative proposition boundary (LOCKED 2026-09-12, Q-L11-4; owner amendments: (1) "authoritative L11 comparative fact" naming, (2) admission as observation/reference never L11 fact, (3) aggregation may increase enumeration never semantic strength; plus constitutional promotion-eligibility prohibition)

The L11 equivalent of the D-L10-1a X→X′ discipline: every L11
artifact establishes exactly one named proposition, each deliberately
weaker than what a consumer might wish to read into it, and no
combination of L11 artifacts upgrades the proposition class.

**1. The single authoritative L11 comparative fact (Amendment 1:
the qualifier is constitutional — L11's authority is local to the
comparison it performed, never over the underlying evidence,
admission, security meaning, or governance truth. The plane
division: L6 owns the durable historical record; L10 establishes
verification facts; L11 establishes comparative facts; Governance
establishes meaning and decisions).** The strongest proposition L11
machinery may establish:

> Under registered criterion K@v (hash), the registered comparator
> bound to K deterministically mapped exactly the enumerated evidence
> records E1..En (ObjectIDs, from governed runs R1..Rm) for candidate
> C (content hash) and baseline B (content hash) — together with an
> OBSERVATION/REFERENCE to the owning door's authoritative admission
> record for B as resolved at comparison time — to Δ in K's declared
> shape.

Every clause is identity, enumeration, attestation-of-observation, or
deterministic computation. Nothing in it is judgment. Δ does not
exist as a free-standing fact: "Δ = X" detached from (K, C, B, E,
comparator identity, attestation) is not an L11 proposition — bare
deltas are the laundering surface, so the package is the atom and Δ
is a field of it. No fact without grounding: a proposition
referencing evidence outside its enumerated record set is invalid
(D-L10-10 reapplied). The actual atom is the conditioning tuple
(C, B, K@v, comparator@v, E-set, R-set, admission observations) → Δ;
this forecloses the later "optimization" of storing bare deltas —
the delta's meaning depends on what was compared and under which
criterion.

**Amendment 2 (admission is an observed fact, never an L11
assertion):** the comparison records an observation/reference to the
owning door's authoritative admission record as resolved at
comparison time; L11 does not establish or infer admission itself.
The package establishes "at comparison time, L11 observed admission
record X for B at owning door D" — never "B is legitimately
admitted"; the latter remains the door's proposition. No bare
`baseline_admitted = true` boolean may exist in any L11 schema to be
read as authoritative. The pattern: L11 can establish that it
observed an authoritative fact; it cannot become the owner of that
fact.

**2. The six claims, disposed:**

1. `Δ = X` — YES, as the conditioned fact above; never bare.
2. `C performed better than B under K` — DERIVED ONLY, and only when
   K's registration formally declares an ordering/improvement region
   over its Δ shape. Then "better-under-K" is definitionally
   equivalent to "Δ ∈ K's declared region" — a rewriting of claim 1,
   not a new proposition. It is computed as a stateless derivation
   (the views.go / D-L10-9 pattern) and NEVER stored in the package —
   a stored ordering could drift from K; single-home for ordering
   semantics is K itself. If K declares no ordering (multi-metric Δ),
   no better-claim exists anywhere in L11: the tradeoff is unresolved
   and REMAINS unresolved inside L11 — resolution is the Q-L11-7
   question and is not an L11 computation. The token "better" appears
   in L11 output only as machine-derived "better-under-K@v".
   (Owner confirmation: better-under-K is a named projection of the
   comparison fact, not a new epistemic fact; and better-under-K ≠
   better ≠ improvement ≠ safe ≠ recommended ≠ promotable — a
   distinction that must survive implementation.)
3. `C is an improvement` — OUTSIDE. Criterion-free betterness does
   not exist (D-L11-2 vocabulary rule made structural: no L11 schema
   has a field able to carry it).
4. `C is recommended` — OUTSIDE as an established fact. Advisory
   rationale (model- or human-authored) may be CARRIED inside a
   Candidate, explicitly marked advisory — never
   machinery-established, never derived, never aggregated into a
   ranking. Cross-criterion ranking is recommendation by sort order
   and equally outside; ordering a view by ONE K's registered
   ordering is a claim-2 rewriting and allowed.
5. `C is safer` — OUTSIDE, categorically. Security propositions are
   Governance meaning (detail deferred to Q-L11-18). Even where K's
   metric is security-relevant, the L11 fact is Δ over that metric;
   "safer" is its Governance interpretation.
6. `C should be promoted` — OUTSIDE, categorically. No deontic
   vocabulary in any L11 schema: no should/must/ready/eligible.
   **Constitutional prohibition (owner): L11 MUST NOT mint, store,
   or derive promotion eligibility.** Even where K declares
   `Δ >= threshold`, L11 may say "C produced Δ satisfying K's
   declared comparison region" — never "C is eligible for
   promotion". Eligibility is not purely comparative: the owning
   door may add governance review, artifact-specific constraints,
   security policy, compatibility, scope, operational conditions,
   review status. No shortcut: L11 delivers comparative evidence to
   Governance; the door combines it with its own requirements to
   make the promotion decision.

**3. Artifact → proposition table (the X→X′ ladder):**

| L11 artifact | Establishes exactly | Never establishes |
|---|---|---|
| Candidate | this proposal exists with this content commitment, family, target, claimed baseline, authorship | any evaluative fact; that its claimed baseline is correct or current |
| Comparative-evidence package (THE comparison record — one artifact, not two) | the §1 conditioned fact | betterness beyond K-derivation; improvement; recommendation |
| Regression-evidence package | enumerative: each criterion in declared registered set S@v produced Δ within its declared non-regression region | the universal "introduces no regression"; anything about criteria outside S |
| Stateless derivations (views) | rewritings of stored facts (better-under-K; latest-per-(C,B,K)) | any proposition not derivable from stored packages |
| Discrepancy fact (cold re-derivation disagreement) | the two computations disagree | which one is right (Governance interprets; D-L10-12 inherited) |
| Advisory blocks | nothing — carried data, marked advisory | — |

NOT L11 artifacts at all: **baselines** (L11 consumes baseline
identity and attests observed admission; it never establishes "B is
the right/current baseline" — baseline authority is Q-L11-5,
untouched here); **metrics** (consumed facts minted where they were
made — L10 records, benchmark scores, L6 records; L11 mints only Δ
over them); **recommendations** (not an artifact type).

**4. Aggregation rule (Amendment 3, constitutional phrase):
*Aggregation may increase enumeration; it may not increase semantic
strength.*** L11 may legitimately aggregate into a larger ENUMERATED
proposition — "the declared comparison set S contains three completed
comparisons with results Δ1, Δ2, Δ3" is still mechanical. What it may
never produce is Δ1+Δ2+Δ3 → "therefore a good improvement". N
packages under K1..Kn never sum to "improvement"; a clean regression
set plus a favorable comparison never sums to "should be promoted";
synthesis across criteria is Governance judgment at the door.

**5. Enumerative, never universal.** L11 states what WAS compared,
never what is unaffected. Every negative/absence claim carries the
identity of the declared set it is relative to (D-L11-2 Amendment A
generalized).

**6. Direction symmetry (owner: LOCK as stated).** A Δ showing the
candidate better, worse, equal, or incomparable is the same class of
fact — comparative results under K — packaged under the same rules,
equally representable and durable. **The comparison mechanism has no
"success-only" result path** — otherwise the Ratchet becomes a
confirmation engine rather than an evaluation mechanism. Deliberately
NOT solved here (owner concurrence): selective criterion choice (ten
registered criteria available, the model picks the three favorable
ones — can that be called "evaluated"?) and selective package
production/suppression belong to Q-L11-14; the proposition schema
must not be bent to pre-solve them, and their resolution cannot alter
the proposition semantics.

**Locked constitutional form (owner):** The authoritative comparative
fact produced by L11 is a conditioned, reproducible relation
identifying candidate C, admitted baseline B, registered criterion
K@v, registered comparator identity/version, the enumerated governed
evidence and run records, the observed admission references, and the
deterministically computed result Δ. Δ has no standalone L11 meaning
outside this conditioning package. A criterion may define a
deterministic ordering or improvement region over its declared result
shape; better-under-K may therefore exist only as a stateless
derivation of the comparison fact and is never stored as independent
truth. Criterion-relative comparative results do not imply
criterion-free improvement, safety, recommendation, promotion
eligibility, acceptance, or security meaning. L11 may aggregate
comparisons only by increasing the explicit enumeration of
comparative evidence; aggregation may not increase the semantic
strength of the proposition. Regression evidence is therefore
necessarily relative to an explicitly declared comparison set and
never establishes an unrestricted universal absence of regression.
Comparative results are direction-symmetric: favorable, unfavorable,
equal, and incomparable outcomes are all the same class of L11 fact.
Selective production or suppression of comparisons is an independent
governance/automation concern (Q-L11-14) and cannot alter the
proposition semantics.

(Owner note at lock: this also resolves the apparent
"regression-resistance" ambiguity from D-L11-1 — L11 can establish
bounded, enumerated regression evidence; it cannot establish that "no
regression exists.")

### D-L11-5 — Baseline authority (LOCKED 2026-09-12, Q-L11-5; owner precision amendment on the constitutional formulation)

**Core: L11 owns no baseline authority.** "Baseline" is not an
L11-determined fact. For every candidate family the baseline
universe is the owning door's authoritative record, and every
baseline question decomposes into a door proposition (owned there)
plus an L11 observation (owned here, per D-L11-4 Amendment 2).

**1. Per-family baseline home (the D-L11-1 ownership split
extended):** skill-revision → the L9 catalog's admitted
registrations (current = active per catalog governance);
contract-revision → the L10 contract registry (active entries);
instruction-revision → the active instruction set under L1
governance; regression-test → the effective evaluation set as
governed at its home (Q-L11-13); knowledge → the Themis-side owner
(harness observes only through governed reference). L11 maintains no
baseline registry, no baseline list, no "current pointer" of its own
— a second home for "what governs now" would demand exactly the
reconciliation semantics D-L11-3 forbids. (Owner enumeration at
lock: no baseline registry, no "current baseline" pointer, no
canonical baseline list, no baseline SELECTION authority, no
baseline-CORRECTION mechanism. Key invariant: L11 observes what the
owning authority says; it does not decide what the owning authority
should say — preventing a future "smart baseline manager" from
quietly becoming Governance.)

**2. The baseline observation proposition.** At comparison time L11
resolves the owning door's record and establishes only: "at
comparison time, against door-registry bytes with hash H, L11
observed admission record X for B at door D, in state s,
current-active: yes/no." Grounded in the door-registry hash so cold
reconstruction can re-verify the observation (Q-L11-17).
Correctness ("the right baseline") and currency ("the baseline
now") are DOOR propositions; L11 records observations of them and
never improves on them.

**3. Admission is a production precondition (the D-L11-2 ladder law
made mechanical).** No resolvable admission record at the owning
door → NO comparative-evidence package is minted at all. The
failure is a recorded refusal (the D-L10-8 pattern: refusal
evidence, never a weaker package) — otherwise unadmitted-baseline
packages would circulate and be misread as position-2 evidence.
(Owner sharpening at lock: NOT "baseline unavailable → INCONCLUSIVE
comparison" — that would leave something superficially resembling
position-2 evidence in circulation. The distinction: cannot
establish baseline admission → the comparison DOES NOT EXIST;
baseline admitted → the comparison executes and may have an
unfavorable/incomplete result. Refusal boundary, D-L10-8 style.)
Withdrawn = not admitted for NEW comparisons; historical packages
against a subsequently withdrawn baseline remain historical
evidence — withdrawal does not rewrite history (L10 semantics
reapplied).

**4. Currency is observed by default, constrained by registration.**
A comparison against an admitted-but-superseded baseline is
producible; the package visibly records the observed non-currency.
Default refusal would make L11 a policy owner; silent production
would enable the stepwise-drift attack (the CheckBaseline lesson:
each step compared against a slightly-worse predecessor). The knob
sits in governed registration instead: a criterion K may DECLARE
baseline constraints (closed declarative vocabulary — e.g.
requires-current-active; exact vocabulary settled in Q-L11-6),
reviewed at registration and enforced fail-closed at comparison
time. Consumption doors additionally enforce their own currency/
chain requirements at consumption (the gate/router precedent —
admission enforced at consumption). Drift prevention is thus
Governance-declared and door-enforced, fed by honest L11
observation; L11 itself never owns the policy. (Owner precision at
lock — not a "currency judgment": L11 does not decide whether a
baseline is sufficiently current; it EVALUATES whether the observed
baseline satisfies constraints already declared by the registered
criterion. The door asserts "B is currently active" → L11 observes;
K declares requires-current-active → L11 checks the observed fact.
Semantic authority stays with the door and the criterion
registration; L11 only executes the declared condition. This
distinction is load-bearing for Q-L11-6.)

**5. One baseline per package.** The comparative atom is pairwise:
exactly one candidate, one baseline, one criterion. Multiple
baselines = multiple packages; combining them is aggregation by
enumeration (D-L11-4 Amendment 3). No multi-baseline Δ semantics —
C vs {B1,B2,B3} → Δ does not exist unless a multi-baseline criterion
is deliberately introduced later through registration; reconstruction
stays clean.

**6. The candidate claims; it never chooses authoritatively.** The
candidate's baseline reference (D-L11-3) is a CLAIM — proposer data
directing comparison, establishing nothing. The comparison is
conditioned on the door observation, never on the claim; a
claim/observation mismatch is recorded visibly in the package.
Choosing an unadmitted baseline yields refusal (§3); choosing a
stale admitted baseline yields a package that says so (§4); choosing
flatteringly WITHIN the admitted set is the baseline face of the
cherry-picking surface — visible in the package, hardened at
Q-L11-14, never solved by proposition schema.

**Mismatch rule (owner, fourth case):** candidate claims B, door
resolves B′ — this must NOT silently resolve to B′. The comparison
either explicitly compares against the observed B′ (mismatch
recorded) or refuses; it never silently substitutes the door's
baseline and leaves the candidate's claim looking satisfied.

**Admission vs selection (owner refinement):** two distinct
questions. (A) "Is B an admitted governed artifact?" — the door
answers. (B) "Is B the APPROPRIATE comparison baseline for this
candidate?" — a door may hold ten admitted historical versions and
does not know which one a criterion requires; appropriateness comes
from a registered criterion or existing family-specific comparison
policy, never from L11. D-L11-5 establishes only that the baseline
used is visible and provenance-bound; which baseline a candidate may
REQUEST is selection policy → Q-L11-14.

**Constitutional formulation (owner amendment, replacing the
implementer sentence):** *L11 owns no baseline authority. Baseline
identity and admission are authoritative only according to the
affected domain's existing owning mechanism; any additional
baseline-selection or currency constraint must come from an already
governed mechanism, such as the registered comparison criterion.
L11 may observe and mechanically apply those declarations, but may
not create, correct, or interpret them.* Retained core invariant:
*a candidate claims a baseline; it never chooses an authoritative
baseline.*

Carried to Q-L11-6: the closed vocabulary of criterion-declared
baseline constraints (§4), with the rest of what a criterion can
say.

### D-L11-6 — Criterion ownership and registration (LOCKED 2026-09-12, Q-L11-6; owner amendments: (1) applicability ≠ selection, (2) ordering region is criterion-relative derivation, never Governance meaning. Comparator option A LOCKED.)

The owner's sharpened question: who is allowed to define the
comparison semantics that L11 is subsequently permitted to execute?

**1. Comparison semantics have exactly three homes, each already
governed.** (a) The CRITERION DECLARATION — what to compare, over
which evidence, under which constraints, with which Δ shape and
optional ordering: a Governance-registered declarative artifact.
(b) The COMPARATOR IMPLEMENTATION — the deterministic Δ computation:
reviewed in-harness code shipped through the ordinary Class-2/3
pipeline. (c) The REGISTRATION BINDING them — a Governance act in
the criterion registry. L11 machinery EXECUTES registered semantics
and defines none: an unregistered criterion-shaped artifact is data
(the L9/L10 wall, third application).

**2. Registry ownership: Governance, in a NEW L11-format criterion
registry.** Not the L10 contract registry — criteria answer a
different question (comparison, not verification); housing them
together would let comparison semantics masquerade as verification
semantics, exactly the blur D-L11-1 prohibits. Not a Themis-side
mechanism — criteria are harness-plane comparative semantics with no
security meaning; Themis remains the security system of record and
gains no new obligation. Division as with L9/L10: **L11 owns the
registry FORMAT and the fail-closed read-only loader; Governance
owns every registration act and the semantic review at
registration.** Same walls, verbatim: append-only, immutable
name@version→hash bindings, active→withdrawn only, two-way identity
(registration ↔ self-declaration), no write API anywhere in harness
code (AST-audit wall), *.proposed.* → owner act → ACTIVE.

**3. Authorship vs registration.** Anyone — the model included — may
AUTHOR a proposed criterion; authorship raises attention, never
lowers the bar (D-L9-16). Only Governance registration makes it
executable semantics. Registration review is the semantic gate: the
reviewer answers "should this comparison frame exist?" — the
cherry-picking-by-selector surface (D-L11-4 flag) is examined HERE,
once, at the only point where semantics enter. (Owner rationale at
lock: a criterion controls how improvement evidence is INTERPRETED —
if the model could register its own criterion it could define its
own success measure, a serious Ratchet escape. The model may propose
the frame of evaluation; Governance decides whether that frame
becomes a governed comparison semantic.)

**4. What a criterion can say (closed schema, nothing defaulted):**
identity (name@version + SHA-256, two-way); applicable candidate
family/families; candidate-side and baseline-side EVIDENCE SELECTORS
— typed, declarative, closed vocabulary, selecting only established
governed facts (L10 evaluation records by contract token + outcome,
validated benchmark scores, L6 execution records, gate verdicts;
D-L11-4 §2 carried); the COMPARATOR BINDING (registered code
identity, §6); pinned comparator CONFIGURATION by value, hashed (the
L10 config pattern); the declared Δ OUTPUT SHAPE (closed type
vocabulary); optionally an ORDERING/IMPROVEMENT REGION over that
shape — declarative data with closed semantics (per-field direction
and thresholds), interpreted only by the registered derivation code,
enabling better-under-K (D-L11-4 claim 2); BASELINE CONSTRAINTS from
the closed vocabulary — v1 proposes exactly one token,
`requires-current-active` (D-L11-5 §4); extension is a schema
revision, not free text; PROVENANCE REQUIREMENTS.

(Owner confirmation on selectors at lock: selectors over established
governed facts, never arbitrary queries — criterion → arbitrary
query → arbitrary evidence would create a second context/evidence
authority. The selector identifies what evidence the criterion
requires; L2/L10/L6 remain owners of the underlying evidence.)

**Amendment 1 (owner) — applicability ≠ selection:** *A criterion
may declare where it is applicable and what evidence/constraints it
requires, but it cannot cause itself to be selected or require its
own execution. Selection remains governed by the existing evaluation
workflow/consumption mechanism.* "K applies to skill revisions" is
criterion metadata; "therefore K must be selected for this
candidate" is selection policy, owned elsewhere (Q-L11-14). Without
this wall the criterion becomes partly self-authorizing — a hidden
L11 policy engine.

**5. What a criterion can NEVER say:** no proposition beyond Δ and
its declared region (no "improvement"/"safe"/"recommended"/
"eligible" — D-L11-4); no security meaning; no promotion
consequence, no door obligation ("if Δ ≥ x then …" has no legal
continuation — the criterion has NO continuation language: no
then/should/must/promote/accept/activate/eligible/deploy, the
anti-smuggling principle that protected L9 and L10); no executable
logic, expressions, or templates (D-L9-4 zero-interpreters); no
evidence source outside the established-facts vocabulary; no
redefinition or reinterpretation of L10 outcomes (owner
confirmation: a criterion may incorporate PASS as an established
input fact into its comparison; it cannot redefine PASS → good or
PASS → improvement — L10 owns the meaning of its result, L11
consumes it as evidence); no reference to unregistered artifacts.

**6. The comparator binding — the carried open decision, proposed
as (A) registered in-harness code identity.** The comparator set is
closed, reviewed Go (the canonicalizer precedent, D-L10-12:
canonReport shape); a criterion binds comparator name@version; a new
comparator is a code change through the full pipeline with review,
never data. Option (B), a declarative Δ language inside the
criterion, is rejected as an interpreter by another name: its
evaluator would become an unreviewed semantic engine, and criterion
review would silently become program review. The narrow declarative
remnant that IS allowed — ordering regions, §4 — is data with closed
fixed semantics, not composable logic; the line is "parameters yes,
programs never." **(Owner: LOCKED as option A.** `threshold = 0.90,
direction = maximize` is legal because the interpreter already has
fixed semantics; `if metric_a > metric_b: ... else: ...` is not,
because the criterion language would become an executable semantic
language requiring another interpreter and another security review
surface.)

**7. Recursion flag.** A criterion is itself a governed artifact;
criterion-revision as a candidate FAMILY (the Ratchet proposing
improvements to its own criteria) is deliberately NOT decided here —
that is Q-L11-19's recursion question. v1 families remain as
D-L11-3 fixed them.

**Constitutional sentence:** *Comparison semantics enter L11 only
through Governance registration of a declarative criterion bound to
a reviewed comparator; L11 executes what is registered, and nothing
that is registered can say more than Δ.*

**Amendment 2 (owner) — precision on "nothing more than Δ":** *the
criterion may define the deterministic interpretation of Δ within
its declared shape, including an ordering/improvement region, but
that interpretation remains criterion-relative and cannot acquire
Governance meaning.* better-under-K does not violate "nothing more
than Δ" — it is a deterministic projection of Δ under K. The
escalation hierarchy: Δ → K-defined ordering → better-under-K → NO
FURTHER SEMANTIC ESCALATION (improvement, recommendation, safety,
promotion eligibility, acceptance all remain outside L11).

**Carried to Q-L11-7 (owner):** the ordering vocabulary is NOT
expanded here. A criterion can define a fixed, closed partial/total
ordering if its semantics are explicitly registered; whether any
multi-metric TRADEOFF resolution can legitimately reside in a
registered criterion without becoming hidden Governance judgment —
"decide whether the accuracy improvement is worth the cost
increase" — is exactly Q-L11-7's question, and L11 must not
improvise the answer.

### D-L11-7 — What "better" means: orderings and the tradeoff boundary (LOCKED 2026-09-12, Q-L11-7; owner amendments: (1) composite wall as v1-vocabulary exclusion not permanent law, (2) scalarization is projection never replacement; plus no-optimization-objective invariant)

Frame (already locked): "better" has no L11 meaning outside K
(D-L11-4 claim 2); the ordering region is criterion-relative
derivation, never Governance meaning (D-L11-6 Amendment 2). What
remains: which ordering STRUCTURES may K legitimately declare, and
where registrable ordering ends and human judgment begins.

**1. The closed ordering taxonomy** — a criterion declares exactly
one of four structures, weakest to strongest:

- **(i) No ordering.** Δ is descriptive; no better-claim is
  derivable, ever. Always legal; the default.
- **(ii) Per-metric orderings.** Each Δ field declares direction
  (maximize|minimize) and optional thresholds → per-metric
  derivations only ("improved-under-K on latency", "regressed-under-K
  on cost"), enumerated, never aggregated across metrics. The
  multi-metric tradeoff remains visibly unresolved.
- **(iii) Dominance (partial order).** better-under-K iff
  better-or-equal on every declared metric and strictly better on at
  least one; symmetrically worse-under-K; else equal or
  INCOMPARABLE. Nothing is traded off — dominance is the strongest
  claim derivable without any tradeoff commitment.
- **(iv) Registered scalarization (total order).** A fixed
  deterministic reduction of Δ to an ordered scalar — weights,
  lexicographic priority, or another closed form — supplied ONLY as
  pinned parameters to a reviewed comparator with fixed semantics
  (D-L11-6: parameters yes, programs never).

(Owner at lock: higher expressiveness is NOT higher authority — a
scalarized criterion is not "more authoritative" than a dominance
criterion; it simply contains more precommitted comparative
semantics.)

**2. The tradeoff boundary — the actual answer to Q-L11-7.**
Declaring scalarization weights IS a tradeoff judgment. It is
legitimate exactly because it is made AT REGISTRATION by Governance
— reviewed, versioned, hash-pinned, visible — not improvised at
comparison time by L11 or the model. Registrability test
(constitutional): *a tradeoff may be encoded in K only if Governance
is willing to commit to it in advance for every future comparison
under that K; anything Governance would want to weigh case-by-case
must remain outside K.* "Is +5% accuracy worth +30% cost for this
use?" fails the test — context-dependent judgment; the criterion
must then stay at (ii)/(iii) and the multi-metric Δ goes to the door
unresolved. L11 never improvises the resolution; there is no
resolution machinery to improvise WITH — orderings are declared
data, applied by fixed code.

**3. Ties and equality.** K's ordering declares its equality
semantics: exact equality, or a registered tolerance as a pinned
parameter. equal-under-K is a first-class, direction-symmetric
result. No tie-breaking by ANYTHING outside Δ — not recency, not
model preference, not authorship, not execution time, not candidate
origin; none may silently turn equal-under-K into "C wins". If
Governance wants a tie-break rule, it must be part of K before
comparison. A tie is an answer, not a problem.

**4. Incomparability is a result, not a failure.**
incomparable-under-K (dominance criteria where neither side
dominates) is a first-class package outcome — never coerced toward
worse or equal, never an error. Distinct from Q-L11-16 machinery
failure: incomparable = the comparison SUCCEEDED and the ordering is
undefined at that point of Δ-space; NO COMPARISON = no package
exists. (Direction symmetry, D-L11-4 §6, extended to the full
four-value derivation: better | worse | equal | incomparable, each
under-K.)

**5. Thresholds define regions, never consequences.** A threshold
places Δ inside or outside a declared region (improvement region,
non-regression region); region membership is one more derived,
direction-symmetric boolean. Any continuation is illegal (D-L11-6
§5: no continuation language).

**6. No cross-criterion "better", and no derivation chains.**
better-under-K1 ∧ better-under-K2 composes into nothing (aggregation
raises enumeration, never strength — D-L11-4 Amendment 3). A
composite judgment across criteria is Governance's at the door. If
Governance wants a registered composite, the composite is itself a
criterion computed from the UNDERLYING established facts.
**Amendment 1 (owner) — the wall, precisely bounded:** *a v1
criterion cannot consume an L11 comparative proposition as an input
to create a stronger comparative proposition. L11 comparative
evidence is excluded from the v1 criterion input vocabulary; any
future higher-order comparison (statistical/meta-analysis) requires
a separate architecture decision.* The prohibited laundering shape:
C vs B under K1 → better-K1 → K2 consumes better-K1 → C "better"
under K2. The v1 exclusion closes it without constitutionalizing a
permanent ban on higher-order mechanisms that would face their own
fresh Class-4 decision.

**7. Scalarization provenance (owner Amendment 2):**
*scalarization is a projection of the complete declared Δ and never
replaces or obscures the underlying Δ fields required for
reconstruction.* A package under a scalarized K retains the full
multi-metric Δ (accuracy +10%, latency −50%), not merely the scalar
— otherwise reconstruction loses the evidence the scalar was
produced from, and scalarization becomes a compression/laundering
surface. (D-L11-4's conditioning rule made explicit at its easiest
escape point.)

**8. The criterion is not an optimization engine (owner
invariant).** K may say maximize X / minimize Y / weights = w. It
may never say "find the candidate maximizing K" or "keep generating
candidates until the score improves" — that would turn L11 into an
autonomous improvement optimizer. The Ratchet compares a candidate
that EXISTS; it does not own the search strategy for producing the
next candidate (search/generation belongs to the automation grill,
Q-L11-14).

**Constitutional sentence:** *"Better" is always a deterministic
projection of Δ under a registered ordering; a tradeoff may be
encoded only as parameters Governance committed to at registration;
any tradeoff requiring case-by-case judgment remains unresolved
inside L11 and is delivered to the door as multi-metric Δ.*

**Carried principle (owner):** *a criterion may precommit
comparative semantics; it may never perform case-specific tradeoff
judgment.* The precommitment test is constitutional: it prevents
"candidate C exists → choose weights that favor C → C wins"; asking
"what weighting should we use for THIS candidate?" is no longer
criterion semantics but case-specific Governance judgment, outside
L11.

Flagged: adversarial weight-tuning of scalarizations (choosing
weights that flatter a known candidate) is a registration-review
obligation, carried to Q-L11-14/Q-L11-20's adversarial register.
The incomparable/NO-COMPARISON distinction carries directly into
Q-L11-16.

### D-L11-8 — Selection is not promotion: the router boundary (LOCKED 2026-09-12, Q-L11-8; owner amendments: (A) position-4 wording, (B) fallback must be predeclared in registered policy; plus no-side-effect invariant)

Resolving the quarantined tension: validated benchmark evidence →
router → model selected for runtime — is that evaluation→behavior
(prohibited shape) or legitimate consumption?

**1. The two acts, constitutionally distinguished.** PROMOTION
changes what is governed — it alters the set of admissible
alternatives (registration, activation, withdrawal: acts at doors).
SELECTION deterministically chooses AMONG the already-admitted set —
which admitted alternative governs work now, under fixed deployment
policy. In ladder terms (Amendment A wording): **selection is one
form of the existing 3→4 consumption mechanism; the router is an
instance of that mechanism, not the definition of position 4** —
future consumption mechanisms may establish reliance without being
a router; D-L11-2's generality is preserved. The prohibited shape is
machinery performing 2→3. The owner's two scenarios split precisely
here: admitted-A(82) vs admitted-B(91) → select B is 3→4 reliance;
unadmitted-B(91) → select B is 2→(skip 3)→4, already structurally
refused by the admission walls (gatePassed). And 4⇏2 holds: being
selected confers no epistemic upgrade — reliance is not endorsement.

**2. The five conditions for legitimate evidence-driven selection
(each mechanically checkable; violating any one makes it promotion
by another name):**

- **(a) Closed candidate set.** Selection operates only over
  alternatives each independently admitted through its own door.
  Evidence may ORDER the admitted set; it may never EXTEND it — an
  unadmitted alternative with an excellent score is invisible to
  selection.
- **(b) Admitted evidence.** The evidence consumed is itself
  governed and admission-checked at the point of consumption (the
  existing gatePassed discipline: verdict pass + model identity +
  currency + digest + baseline chain). Ungoverned scores select
  nothing; this is the verdict/digest precedent generalized.
- **(c) Fixed registered policy.** The selection rule is
  deterministic deployment policy — code/config through the
  ordinary review pipeline — evaluated at declared decision points
  (e.g. startup), reproducible from its inputs. Not model judgment,
  not L11 judgment, not tunable at runtime.
- **(d) No governance state is written.** Selection promotes,
  demotes, registers, and withdraws NOTHING. Deselection is not
  demotion; no "current champion" registry may exist (the
  second-home prohibition again — which artifact was relied upon is
  an L6 historical fact, never a governance state). The next policy
  evaluation may choose differently. (Owner emphasis at lock — the
  future implementation temptation: `selected_model_for_task = B` is
  execution state and fine; a persisted `current_champion = B` is a
  governance fact and forbidden — otherwise benchmark → router →
  champion registry → future router, and the router has quietly
  become an authority mechanism. Historical selection belongs in L6
  execution history; governed admissibility belongs at the owning
  door. deselected ≠ withdrawn; selected ≠ promoted.)
- **(e) Fail closed to admission, never to evidence.** Missing,
  invalid, or stale evidence → refusal or fallback — never "select
  the best unverified." **Amendment B (owner):** *any fallback
  selected when required evidence is unavailable must itself be
  explicitly defined by the fixed registered selection policy and
  must select only from the already-admitted set.* An undeclared
  "pick default" on missing evidence would silently become an
  unreviewed policy.

**3. Why this does not breach D-L9-16.** The no-auto-promotion
prohibition targets evaluation results changing WHAT IS GOVERNED.
Selection under (a)–(e) changes only which admitted artifact is
consumed — the same act-class as an L7 gate consuming a recorded
verification outcome to choose among declared edges (L10 precedent:
recorded evidence deterministically steering within a
Governance-declared space). The admissible SPACE is always
human-governed; evidence steers within it, never redraws it.

**4. The tests for "promotion by another name" (any one crossing
the line):** selection enlarges the admissible set; selection output
becomes durable governance state; the policy consults unadmitted
evidence; the policy itself is chosen or tuned at runtime by
evaluation results (meta-selection — the policy is fixed by
deployment governance, its revision is an ordinary reviewed change);
evidence-consumption bypasses the admission check at the consumption
point. Note recorded: a policy that leaves some admitted artifact
permanently unselected is legitimate preference, not de facto
withdrawal — admissibility is unchanged, and the pattern is visible
in L6 records for Governance to notice. **Completing invariant
(owner): a selection mechanism must not change the future
admissible set as a side effect of selection** — "select B → persist
B as preferred → future admissions automatically exclude A" starts
as selection and becomes governance at the second step; not
permitted.

**5. L11's role at runtime: none.** The router consumes the
BENCHMARK plane's admitted verdict artifacts under that plane's own
gate discipline — machinery that predates L11 and remains
deployment/bench-owned. **Standing wall (resolving the D-L11-4 §6
quarantine): L11 comparative-evidence packages are not selection
inputs.** Δ is addressed to doors — humans — and enters no routing
decision. If Governance ever wants runtime selection to consume an
L11-class artifact, that is a fresh architecture decision (the
D-L11-7 Amendment-1 pattern: excluded from the vocabulary now, not
banned by permanent law without a concrete requirement).

**6. Generalization handoff.** D-L11-8 defines the boundary;
Q-L11-15 formalizes it as the consumption-of-evidence rule
(evidence → Governance promotion vs validated evidence →
deterministic deployment selection), with the router as its existing
proven instance.

**The deeper principle (owner, at lock):** *evaluation evidence may
be consumed by runtime policy without becoming promotion, provided
the consuming mechanism operates strictly within an already-governed
choice set and cannot mutate that set.* This is the architectural
escape valve that keeps L11 from accidentally forbidding ordinary
deterministic systems that legitimately use evidence to choose
between already-approved alternatives. (And the quarantine
resolution is how an architecture ratchet should work: L11
comparison → runtime router cannot be achieved by adding an input
field; it requires a fresh architecture decision.)

**Constitutional sentence:** *Selection consumes evidence to choose
among what Governance has already admitted; promotion changes what
is admitted. Evidence may deterministically order the admitted set
under fixed registered policy; it may never extend that set, and no
selection outcome is ever a governance state.*

### D-L11-9 — Regression resistance: the bounded enumerative claim (LOCKED 2026-09-12, Q-L11-9; owner amendments: (1) bounded resistance derivation, (2) exact set resolution — L11 never selects the applicable set)

**1. Regression evidence introduces NO new fact type.** A
regression-evidence package is aggregation-by-enumeration (D-L11-4
Amendment 3) of ordinary comparative facts, specialized by exactly
two things: the criterion set carries its own registered identity,
and each member's declared region is designated a NON-REGRESSION
region. The proposition: *"for candidate C against admitted
baseline B, every criterion in registered regression set S@v
produced a comparative package, with each Δ's position relative to
its declared non-regression region recorded."* No new vocabulary,
no new outcome class, no second evaluation semantics.

**2. The regression set S is a registered artifact.** S@v =
versioned, hash-pinned, append-only enumeration of exact criterion
pins (K@v — no floating members, or set identity means nothing).
Registered through the same Governance-act mechanism as criteria
(L11 owns format + fail-closed loader; Governance owns
registration). WHO OWNS "THE" REGRESSION SET: the door. A door (or
Governance policy for that family) declares which S it requires —
"promotion requests at this door carry regression evidence under
S@v" is the door's requirement. L11 can evaluate any registered S
on request; the SIGNIFICANCE of a particular S — that it guards a
particular door — is the door's declaration, never L11's (the
D-L11-5 ownership pattern applied to sets). The CONTENT of the
underlying cases — reference inputs, golden outputs, acceptance
thresholds inside criteria — is Q-L11-13's question, untouched
here. (Owner formulation at lock: L11 never says "this is the
regression set that protects this artifact"; it can only say "I
evaluated registered set S@v".)

**3. Completeness under S is mechanical — no partial regression
packages.** A regression-evidence package under S@v REQUIRES one
constituent comparative package per member of S. Anything less is
not a regression-evidence package under S — it is merely N
individual comparison packages that cannot claim the set identity.
This closes the "passed 9 of 10, silently omit the 10th" laundering
shape structurally: partial coverage is representable only in a
form that visibly lacks set identity. A refused constituent
(D-L11-5 §3: baseline unadmitted, evidence unresolvable) means the
set-level package does not exist. (Owner: constitutional — safer
than a `coverage = 3/4` field misread downstream as a regression
result. The three cases, explicit: **A** — a constituent could not
be compared → no S-level package exists; **B** — all constituents
exist, one outside its region → complete regression evidence exists
and DEMONSTRATES an observed regression under that K; **C** — all
exist and all satisfy → complete evidence from which
resistant-under-S may be derived.)

**4. Completeness ≠ favorability — "resistant" is a derivation.**
A complete-under-S package whose constituents include out-of-region
Δs is equally producible and durable (direction symmetry, D-L11-4
§6 — a regression FOUND is first-class evidence, arguably the
Ratchet's most valuable output). **Amendment 1 (owner) — the
bounded derivation:** *resistant-under-S@v is a deterministic
derivation that every EXPLICITLY ENUMERATED constituent comparison
in the registered finite set S@v satisfies its declared
non-regression region. It never establishes unrestricted absence of
regression.* "All members of this declared finite set satisfied
their declared regions" — never "no regression exists." A stateless
derived boolean, the better-under-K pattern exactly: derived, never
stored, never free-standing. There is no stored "resistant" flag
anywhere.

**Incomparability rule (owner, explicit implication of D-L11-7):**
an incomparable-under-K constituent does NOT satisfy a
non-regression region unless K itself explicitly defines
incomparability as inside that region. K → incomparable means
neither "regression" nor "no regression" automatically; the
registered K decides what its region means — declared semantics,
never L11 improvisation.

**5. The universal claim is structurally unmakeable.** "Introduces
no regression" quantifies over an undefined universe; L11's
strongest emission is complete-under-S enumeration. Dimensions not
in S are simply not spoken about — absence of evidence stays
explicit (D-L11-2 Amendment A lineage). If Governance wants more
covered, it registers a larger S version; the claim grows by
enumeration, never by quantifier.

**6. Never a gate.** L11 produces the package; doors decide what it
permits (D-L11-1: provides, never enforces). A door requiring
regression evidence under S@v as an admission input is the door's
own rule, enforced by the door at consumption (the benchmark-gate
precedent: the gate enforces, the evidence plane establishes) — and
evidence never obligates approval. L11 has no blocking authority,
no veto path, no "regression check failed → refuse promotion" code
anywhere.

**7. Set evolution and the stepwise lesson.** New S versions are
new registrations (append-only). **Amendment 2 (owner) — exact set
resolution:** *L11 evaluates an exact registered S@v supplied by the
governed consuming mechanism. Any latest/current/currency
requirement is resolved by the owning door or an already-governed
policy; L11 does not select the applicable set.* Given S@1/S@2/S@3,
L11 never internally resolves "the latest active S is S@3" — even
that tiny version-selection authority is refused; the door supplies
`required_regression_set = S@2` (or a pre-registered policy resolves
it deterministically), and L11 mechanically verifies and applies the
resolved requirement — exactly the D-L11-5 pattern. Every constituent comparison obeys D-L11-5
unchanged: admitted baseline, observed admission, no drift — the
CheckBaseline chain lesson applies per-constituent with no
set-level exception.

**Constitutional sentence:** *Regression resistance is never a
property of a candidate; it is a complete, enumerated comparative
record under a registered set — and the door, not the Ratchet,
decides what that record permits.*

**Carried invariant (owner, → Q-L11-10):** *a complete regression
record can contain evidence of regression; completeness is
coverage, not favorability.*

### D-L11-10 — Evaluation execution: describe and consume, never run (LOCKED 2026-09-12, Q-L11-10; owner amendment: plan conformance is record-to-plan matching, never a new evaluation subsystem)

Answering the owner's dichotomy: L11 is an **evidence consumer plus
a describer of needed evaluations** — an "orchestrator" only in the
inert sense of emitting plans and checking conformance, never an
initiator or executor.

**1. No L11 execution plane.** L11 has no executor, no runtime, no
model access, no tool access, no verifier invocation, no task
machinery. Everything that RUNS runs as an ordinary governed L7
task (D-L9-16 gate 2) or a benchmark-plane run, under existing
initiation and execution authority. L11's machinery in full:
read-only registries, pure comparators over committed records,
package assembly, stateless derivations. A comparison executes
nothing (D-L11-4 §2).

**2. The eight verbs, disposed:**

- **Create evaluation tasks — describe yes, instantiate no.** L11
  may construct an EVALUATION PLAN (§4): data naming what needs to
  run. Task instantiation happens only through the existing task
  initiation door, same as any task. L11 possesses no task-creation
  API the ordinary plane lacks.
- **Choose the Skill — no.** The plan NAMES registered identities
  (an executable candidate is already registered through its door —
  D-L11-3, 1→3→2); ordinary L7 resolution applies unchanged. L11
  performs no runtime skill selection.
- **Choose the model — no.** Model selection is deployment policy
  (D-L11-8). A plan may require that model identity be RECORDED as
  run provenance (K selectors may condition on it); it never
  selects.
- **Choose evidence — no.** K's registered selectors choose
  declaratively (D-L11-6); L11 machinery applies them; no ad-hoc
  evidence picking exists.
- **Select the verifier — no.** Verification remains model-proposed
  within walks under registered contracts (D-L10-6); L11 consumes
  L10 outcomes as established facts and never touches the
  verification path.
- **Repeat evaluations — describe yes, run no.** Repetition = more
  ordinary tasks through the same initiation door. L11 may record
  that evidence is insufficient for a package (evidence absence,
  explicit); it cannot loop-run anything (no optimization engine,
  D-L11-7 §8).
- **A/B comparisons — compare yes, cause no.** Comparing the two
  arms' evidence is L11's core function; causing both arms to run
  is two ordinary governed runs through existing channels.
- **Schedule — no.** Scheduling is initiation authority, owned by
  existing operational mechanisms; whether ANY automation may act
  on plans is Q-L11-14's question, not granted here.

**3. The three-layer seam.** L7 owns how a task executes; L10 owns
what verification establishes within it; L11 owns how the committed
records of two admitted artifacts compare. L11 sits strictly
DOWNSTREAM of L6 records; its only upstream expression is a plan —
data addressed to doors and humans.

**4. The Evaluation Plan (new artifact, inert).** Content-addressed
data: candidate identity, baseline identity, criterion/set pins
(exact — D-L11-9 Amendment 2), required runs (registered skill@v,
input commitments), required provenance. Nothing imperative, no
continuation language (the D-L11-6 discipline). A plan is a REQUEST
representation with the same standing as a Candidate: existence
confers nothing, executes nothing. Post-hoc, L11 may mechanically
check that runs CONFORM to a plan (identity comparison against
recorded provenance) and record the observation in the package —
enumerated fact, not judgment.

**Owner amendment — conformance ≠ evaluation:** the post-hoc
conformance check is NOT itself a verification/evaluation
execution; the drift "L11 evaluates whether the run satisfies the
plan, therefore L11 is an evaluator" is foreclosed. The five-role
division: L7 executes the governed workflow; L10 evaluates
registered verification contracts and produces L10 outcomes; L11
mechanically determines whether already-committed records
correspond to the identities/requirements enumerated in the plan
and whether those records can participate in a comparison; the L11
comparator compares admitted evidence under the registered
criterion; Governance determines what any resulting evidence means
for promotion/admission.

**Plan is never authority (owner invariant):** *an Evaluation Plan
is never an authority-bearing prerequisite for execution.* A task
initiated pursuant to a plan is valid because the ordinary
initiation/execution mechanisms authorize it — never because the
plan does. Directionality: Evaluation Plan → ordinary task
initiation → L7 → L10 → L6 → L11; never Evaluation Plan → L11
executor → L7.

**Plan language stays declarative (owner):** allowed — candidate
identity, baseline identity, Skill identity/version, criterion/set
identity/version, required input commitments, required provenance,
required number/type of runs, expected artifact/evidence
identities. Not allowed — "try Skill A, then B"; "if result fails,
rerun"; "choose the best model"; "select the verifier"; "increase
sample size until confidence is sufficient"; "retry until PASS";
"pick whichever candidate performs better". Those are
orchestration/optimization semantics and live outside L11; because
L11 cannot select Skill/model/verifier/evidence at runtime, a plan
cannot become a disguised imperative program.

**5. Purpose attribution (assigned residual, minimal form).**
Evaluation tasks are ORDINARY tasks — no execution mode, no special
authority, no semantic flag consulted by any control path. Purpose
attribution is initiation-time recorded data: a run initiated from
a plan records the plan reference; K selectors may then select runs
by plan reference. Whether that reference lives in the existing
task envelope or needs a narrow L6 amendment is an implementation
decision for the milestones — constitutionally it is attribution
data, never execution semantics (D-L9-16 preserved).
(Owner form: durable provenance `run R17 → evaluation_plan P42`,
never an execution-mode flag `task.execution_mode = evaluation` —
the latter would create an implicit special execution plane.
Constitutional requirement: *evaluation-plan attribution is
provenance about why/under what declared evaluation context a run
was initiated; it must not alter L7, L10, L4, or L5 behavior.*)

**Constitutional sentence:** *L11 may describe the evaluation it
needs and consume the records that result; everything that
executes, executes as an ordinary governed task through existing
initiation and execution authority — L11 runs nothing, selects
nothing at runtime, and schedules nothing.*

**Recorded invariants (owner, at lock):** no L11 execution plane;
no L11 task-creation/instantiation authority; no
Skill/model/evidence/verifier runtime selection; no L11 repetition
loop or optimizer; no L11 scheduling authority; evaluation runs are
ordinary governed tasks; L7 owns execution; L10 owns verification
semantics; L6 owns the authoritative execution/evaluation history;
L11 consumes committed records and performs comparison/package
assembly; Evaluation Plans are inert declarative data; plan
conformance is mechanical record-to-plan matching, not a new
verification subsystem; plan attribution is provenance only and
cannot affect execution semantics; an L11 insufficiency result can
describe missing evidence but cannot cause its production; no L11
schedule, retry, or execution backdoor.

The chain: *describe → governed initiation → L7 execution → L10
verification → L6 durable record → L11 comparison/evidence.* And
the boundary that matters most (owner): *L11 can tell Governance
what evidence exists and what additional evidence is needed; it
cannot manufacture the evidence needed to prove its own
conclusion.* The innocent-looking loop "compare → insufficient →
run again → compare → optimize" is structurally unbuildable.

### D-L11-11 — Lifecycle and durable ownership: two planes, hash identity, derived everything else (LOCKED 2026-09-12, Q-L11-11; owner amendment: production recording via existing L6 mechanisms only — no L11 event subsystem)

**1. Exactly two durable planes, both existing — L11 introduces no
third.** (a) The GOVERNED REGISTRATION plane: criterion registry
and regression-set registry under policies/ — Governance-owned
acts, append-only, already decided (D-L11-6/9). (b) The L6 RECORD
plane: every L11-produced instance artifact. There is no L11
database, no L11 state directory, no L11-owned mutable store of any
kind.

**2. Identity rule — registered names vs instance hashes.**
Registered artifacts (criteria K, sets S) carry name@version +
SHA-256 with two-way identity. INSTANCE artifacts — Candidates,
Evaluation Plans, comparative-evidence packages, regression-level
packages, refusal records, discrepancy facts — carry CONTENT-HASH
IDENTITY ONLY. No name@version for instances: naming is the
registration plane's privilege, and hash-only identity structurally
prevents a "candidate registry by name" (a proto-catalog with
status columns) from ever emerging.

**3. What L6 stores (authoritative, append-only, content-addressed
objects — the StoreObject/discrepancy-artifact precedent):**
candidate bytes; evaluation-plan bytes; comparative-evidence
package bytes (constituent references by ObjectID); regression
set-level package bytes; refusal records (D-L11-5 §3); discrepancy
facts (D-L11-4 §4-inherited cold-rederivation disagreements) — on
top of the evidence records L6 already holds (execution records,
L10 records). "No identity without bytes" (D-L10-10): a reference
is valid only if the addressed bytes exist and hash-verify at read
(D-L10-11 read-boundary detection covers L11 artifacts for free;
missing/mismatched constituents fail closed and mint discrepancy
facts, never repaired packages). The *.proposed.* handoff files at
doors are REPRESENTATIONS derived from the authoritative
content-addressed bytes, never a second authority (D-L11-3).

**4. What L11 derives, statelessly, on demand — never stores:**
better-under-K; resistant-under-S; latest-per-(C,B,K) views;
lineage graphs (walking supersedes references); "evidence available
for candidate C" summaries; plan-conformance displays; candidate
enumerations. All recomputed from L6 bytes + registered artifacts
(the views.go/D-L10-9 pattern). v1 has NO derived-state cache at
all — if a future cache is ever justified, it is disposable by
definition and its loss may change no answer. (Owner absolute at
lock: *a derived L11 result is not made more authoritative merely
by caching it.*)

**5. Supersession is a claim, never a state change.** Instances are
immutable; nothing can be marked superseded. A newer candidate
CLAIMS to supersede an older one by hash reference (D-L11-3:
lineage answers "where did this come from", never "is this
governed"). "What claims to supersede X" is a derived view;
"X is superseded" is not an establishable L11 fact — the receiving
mechanism owns any disposition it cares to record, in its own
plane. (Owner: the claim "C2 claims to supersede C1" vs the fact
"C1 has been superseded" — only the owning mechanism establishes
the latter; `candidate-X.status = superseded` would be lifecycle
authority.)

**6. The negative list — what must never become L11-owned
authoritative state (consolidating every prior lock):** candidate
lifecycle/disposition (D-L11-3); promotion truth (doors); selection
or champion state (D-L11-8); resistant/better flags (D-L11-9/4);
baseline identity or currency pointers (D-L11-5); applicable-set or
applicable-criterion choices (D-L11-9 Am. 2, D-L11-6 Am. 1);
comparison queues, schedules, work lists, or retry state
(D-L11-10); any status column on any enumeration of instances. An
index that acquires a status column has become a registry; refused
structurally. (Owner, kept explicit as the implementation-review
tripwire for months from now: *an L11 enumeration plus mutable
status is a registry in disguise* — and the drift path
candidate → candidate_id → status → current_candidate silently
recreates a governance registry inside L11.)

**7. Production recording (owner amendment).** *Production of an
L11 instance artifact is durably represented by the L6
content-addressed object. If provenance requires recording the
production act, that record must use an existing L6 event mechanism
or a separately governed additive L6 amendment; it must not create
an L11 lifecycle/event stream. Such an event records that
production occurred and its provenance, never an L11 lifecycle
state.* Never `L11 → ratchet_events.log`, never
`L11 → candidate_state table` — no generic L11 event stream may
emerge as a future implementation convenience.

**Refusal is a fact about an attempt, not a state of a package
(owner subtlety):** "at time T, package construction refused
because constituent E was unavailable" is an authoritative
historical fact; it never becomes `package P.status = refused` —
refusal records an attempted operation, not a mutable lifecycle
state of anything.

**Constitutional sentence:** *L11's durable truth lives entirely in
the Governance registries and the L6 record plane; everything else
L11 knows is recomputed on demand — if deleting every L11-side
cache changed any answer, the architecture is broken.*

(Owner summary at lock: no L11 state machine, no candidate
registry, no "current" pointer, no authoritative cache, no
scheduler, no execution engine. *L11 has durable evidence, not
durable opinions.*)

### D-L11-12 — "Feedback" dissolves: no feedback subsystem exists (LOCKED 2026-09-12, Q-L11-12; owner strengthening: "feedback" is not an architectural concept in L11)

**1. The word names nothing that is not already owned.** Every
legitimate sense of "feedback" maps onto an artifact whose owner is
already locked:

| "Feedback" sense | What it actually is | Owner |
|---|---|---|
| "This skill performed badly" | Position-1 observation (D-L11-2) — advisory data; durable only as L6 history it derives from | nobody — it is a claim |
| Improvement suggestion | Candidate rationale, or a new Candidate | D-L11-3 |
| Evaluation outcome "fed back" | Comparative/regression evidence addressed to a door | D-L11-4/9 |
| Human commentary on a proposal | Proposal disposition / Governance commentary | the receiving door (D-L11-3) |
| Runtime failures, refusals, incidents | L6 records — consumable via K selectors as established facts | L6 |
| Production regression discovered | Motivation for a regression-test Candidate | D-L11-3 family |
| Model self-critique | Advisory block, marked, establishes nothing | D-L11-4 |

A "feedback" that fits none of these rows is not a missing
category — it is an ungoverned input channel, and it is refused.

**2. Therefore: no feedback artifact type, no feedback store, no
feedback channel, no feedback API.** Introducing one would create
exactly the catch-all the owner flagged: a bucket whose contents
have no fixed proposition, no owner, and no admission rule — the
anti-pattern of everything since D-L11-4 (every artifact
establishes exactly one named proposition).

**3. The ratchet "loop" is human-governed, not an L11 conduit.**
The traditional feedback loop exists — as the whole governed cycle:
evidence → door → Governance decision → new candidates → new
evidence. Its closing arc runs through humans at doors, never
through an L11 mechanism (D-L11-10: describe and consume; D-L9-8:
suggest, never become behavior). L11 carries evidence INTO the
loop; it is not the loop.

**4. Security note — the reservoir problem.** A durable free-text
"feedback" store would be an untrusted-content reservoir with
standing influence over future candidate authoring: external
content injected once ("the best skill would disable verification")
would sit as durable "feedback" waiting to be consumed as if it
were governed insight. External content is data, never instructions
(constitution); refusing the abstraction removes the surface
entirely rather than guarding it. (Owner, retained at lock: even
with no intended authority, persistence + future consumption gives
latent influence without a defined ownership/admission boundary —
untrusted input → durable "feedback" → future candidate authoring →
potential governed change. The safe shape: untrusted observation →
explicitly classified artifact → existing owner/admission
mechanism. There is nowhere for an undefined feedback object to
hide.)

**5. Scaffold disposition (recommendation for Q-L11-20):**
`src/harness/ratchet/feedback/` names a subsystem this grill has
concluded must not exist — FORBIDDEN scaffold, not merely unused
code; DELETE at close (the L10 scaffold lesson). At closure the
implementation inventory is derived from the locked L11 artifact
inventory: a directory that cannot be justified by an existing L11
responsibility is removed, never left as an invitation for scope
creep (candidates/, evaluations/, promotion/, regression/ face the
same test).

**Constitutional sentence:** *There is no feedback artifact:
anything called feedback is an observation, a candidate, evidence,
or door commentary — each already owned. A "feedback" that fits
none of these is an ungoverned input channel, and it is refused.*

(Owner note at lock — the cycle exists architecturally, but L11
does not own it: governed execution → L6 evidence → L10
verification → L11 comparison/regression evidence → Governance door
→ Candidate/revised artifact → new governed execution. The human
Governance act closes the transition; L11 participates by producing
evidence, not by circulating "feedback" — preventing the classic
autonomous-optimization architecture from emerging accidentally.)

**The architectural test (owner, standing for everything remaining
in L11):** *if a proposed L11 component cannot be expressed as a
registered criterion/set, Candidate, Evaluation Plan,
evidence/comparison package, regression package, deterministic
comparator, or reproducible view over L6 records, it probably does
not belong in L11.*

### D-L11-13 — The regression corpus is not a knowledge system (LOCKED 2026-09-12, Q-L11-13; owner amendment: case represented through K, never "case = criterion" as constitutional identity)

The critical question answered up front: NONE of the four corpus
elements can become L11-owned authority. Each is a
Governance-fixed or record-derived input that L11 merely consumes;
the "corpus" as a free-standing curated collection with its own
authority does not exist.

**1. Case identity — Governance, via the criterion registry.
Amendment 1 (owner):** *in v1, no independent regression-case
authority exists. A test case is REPRESENTED THROUGH the applicable
registered criterion and its pinned parameters/evidence selectors;
any grouping into S is a Governance-registered set of K
identities.* Not "case = criterion" as constitutional identity —
K stays comparison semantics, S stays a registered grouping
(D-L11-6 separation preserved); the case is representable without
becoming a third concept. "What is being tested" is fixed at
registration — no separate case object, no case database, no L11
case curation. A
new case enters as a regression-test-family Candidate (D-L11-3)
through the ordinary registration door: author-never-admit
(D-L9-8), production failures motivating new cases arrive exactly
this way (D-L11-12 table).

**2. Reference input — content-addressed bytes, standing from
registration.** Fixture bytes are data with hash identity, pinned
by K (and echoed in Evaluation Plans as input commitments,
D-L11-10). Anyone may AUTHOR a fixture (model included; attention
raised, bar never lowered); its STANDING comes solely from being
hash-pinned inside a Governance-reviewed registration. **Enterprise
wall:** inputs derived from enterprise/production content are
Themis-owned knowledge — they enter the harness-plane corpus only
through the appropriate Themis owner and governed channel; the
corpus must never become a shadow copy of enterprise knowledge (the
no-second-enterprise-knowledge-store hard invariant, applied to
fixtures). (Owner at lock: the drift shape to watch — production
fixtures accumulating under a `ratchet/corpus/` with its own
curation/versioning/retention/selection would quietly BE a second
enterprise-knowledge store. *L11 may reference Themis-owned
knowledge; it may not establish a parallel authoritative knowledge
collection merely because that knowledge is useful for regression
evaluation.* And physical storage through L6 does not transfer
ownership of semantic meaning to L11.)

**3. Reference/golden output — two legitimate sources, both
outside L11.** (a) RECORD-DERIVED: the expected result is the
admitted baseline's actually-recorded behavior, referenced by L6
record identity — provenance-complete, no one "declared" it, the
fact chain establishes it. (b) HUMAN-DECLARED: an expectation
authored and reviewed at registration — a Governance commitment,
exactly like scalarization weights (D-L11-7 precommitment test).
In both cases the golden is a pinned parameter of K; its
correctness is established at the door, never by L11. L11 never
decides an output is "right" — the comparator mechanically
compares against the pin. (The L10 lesson carried: expected-output
semantics were subsumed into governed contracts once already,
D-L10-18; the same move here, no third home.)

**4. Acceptance threshold — the declared non-regression region,
two-stage normativity.** The threshold is K's region declaration
(D-L11-9): its EXISTENCE is Governance's registration commitment
(precommitment test); its CONSEQUENCE — what satisfying or failing
it permits — is the door's (D-L11-9 §6: never a gate). L11
computes region membership, full stop. Normative significance is
split between registration and door; no residue lands in L11.

**5. Storage discipline.** Pinned fixture/golden bytes live as
registry-plane files (registry-relative, hash-verified at load —
the L10 contract-file pattern) or as L6 objects referenced by
hash for large artifacts; milestone implementation decision,
constitutionally constrained: *bytes are hash-pinned by the
registered artifact wherever housed; no unpinned corpus directory
with curation authority exists.*

**6. Growth and drift (owner strong lock):** *evaluation results
must never autonomously modify the population against which those
same results are evaluated.* Prohibited, enumerated: bad result →
remove difficult case; bad result → loosen threshold; bad result →
reweight metric; bad result → replace golden; bad result → retire
regression case. Each would create the Ratchet's most dangerous
failure mode — the system improving its measured score by changing
what counts as evidence. The corpus grows only by registration
acts (append-only), shrinks only by withdrawal; corpus evolution
remains an explicit Governance act. (Owner distinction retained:
L11 can determine whether an observed result lies inside K's
declared region; it cannot decide that the region is appropriate
or what satisfying it authorizes.)

**Constitutional sentence:** *The regression corpus is the set of
registered criteria and their hash-pinned parameters — what is
tested, with what inputs, against what expectation, at what
threshold, each fixed by Governance at registration or derived
from admitted-baseline records. L11 verifies hashes and computes
Δ; it owns no case, no fixture, no golden, and no threshold.*

### D-L11-14 — The automation boundary: complete, never initiate (LOCKED 2026-09-12, Q-L11-14; owner amendment: no-discard applies from ACCEPTED INVOCATION; plus the no-self-continuation invariant)

The governing principle, then the four powers separately:
**automation in L11 may deterministically COMPLETE what a governed
request began; it may never BEGIN.** Nothing in L11 watches,
schedules, polls, or triggers. This is where the deferred
cherry-picking questions (D-L11-4 §6, D-L11-5 §6, D-L11-7 flag)
come due.

**1. Criterion selection — the requester, with exact pins.** A
comparison request names exact K@v (a set request names exact S@v —
D-L11-9 Amendment 2, now generalized to all criterion references).
L11 refuses "compare C and B" without exact criterion identity;
there is no applicable-criteria discovery that auto-runs anything
(D-L11-6 Amendment 1: criteria never self-select). The model may
SUGGEST criteria — advisory. Who legitimately selects: the human
requester, or a pre-registered door policy resolving
deterministically BEFORE L11 is invoked. Selecting flatteringly
within the registered set remains possible — and visible: the
package names its K, and doors defend themselves by requiring
complete-under-S (D-L11-9 §3), where member choice was Governance's
at S registration, not the requester's.

**2. Baseline selection — the requester names B exactly; L11
never defaults.** No "compare against whatever is current" resolved
inside L11 — any currency resolution happens at the door or in
pre-registered policy before invocation (D-L11-5 machinery
unchanged: claim recorded, observation conditions, K constraints
checked, mismatch visible, no silent substitution). Flattering
baseline choice within the admitted set: visible in the package
(observed non-currency recorded), constrained by K's declared
baseline constraints, and defeated at doors requiring
current-active evidence.

**3. Package production — request-driven, and NO DISCARD AFTER
INITIATION.** No daemon watches history producing comparisons
(D-L11-4 §6 confirmed: produced on request). But the request-driven
model needs one hard rule to kill silent suppression: **once a
comparison is initiated, its outcome is durably recorded — package
or refusal record — regardless of direction.** "Run it quietly,
discard if unfavorable" is structurally impossible: initiation
commits to a durable result (the no-success-only-path rule of
D-L11-4 §6, extended from representation to production).
**Owner amendment — "initiated" defined:** *an L11
comparison/package operation is initiated when it has been ACCEPTED
FOR EXECUTION by the existing L11 invocation boundary — not merely
when an external user considered or drafted a request.* Accepted
invocation → comparison attempt → package OR refusal → durable L6
record. Legitimate pre-invocation selection is untouched: someone
may decide not to request a comparison, and L11 makes no epistemic
claim about what an unrequested comparison would have shown.
Cherry-picking then survives only at the request level — choosing
not to ask — which no proposition claims to prevent (D-L11-4:
"selective production is visible, not prevented") and which
complete-under-S requirements expose wherever a door cares.

**4. Candidate generation — REFUSED as machinery; authorship is
always an attributable act.** The owner's distinction is real: "AI
suggested a candidate" is position-1 authorship by an author (the
model, in an ordinary governed task, with provenance naming it —
informed by reading packages as data). "Ratchet automatically
generated a Candidate" would make the MACHINERY an author —
blurring the attention-raising provenance rule (D-L9-16: whose
authorship raises attention?) and assembling the autonomous
optimizer from innocent parts: results → auto-candidates → plans →
evaluations → results (each step separately harmless, the loop
prohibited by D-L11-7 §8). v1: L11 machinery constructs no
Candidates; every Candidate has an author acting through ordinary
governed means. Same disposition for the remaining verbs: L11 does
not auto-open registration proposals (handoff is an act by
someone), and does not "request human review" as a workflow act —
it makes evidence discoverable (derived views); existing mechanisms
carry requests.

**5. Mechanical enforcement (not "the AI shouldn't").** v1 L11 has
NO entry point that is not a synchronous response to an explicit
invocation from an existing governed surface. No timers, no
watchers, no pollers, no queues or work lists (D-L11-11 negative
list), no goroutines outliving an invocation. Reviewable
structurally (the AST-wall discipline: no time.Ticker/cron
machinery in the package) — the boundary is checkable code shape,
not intent.

**Constitutional sentence:** *Automation in L11 completes governed
requests deterministically; it never initiates. Nothing in L11
watches, schedules, generates, or selects — every comparison,
plan, and candidate begins with an attributable act outside the
machinery, and once begun, its outcome is recorded whatever it
shows.*

**Additional invariant (owner, at lock — stronger and more
implementation-testable than "no automation"):** *no L11 operation
may create another L11 operation as a consequence of its result.*
Enumerated: comparison cannot trigger comparison; refusal cannot
trigger retry; regression cannot trigger Candidate creation;
Candidate observation cannot trigger Evaluation Plan creation;
package production cannot trigger registration; evidence cannot
trigger scheduling. This closes the possibility of building an
autonomous loop out of individually synchronous calls. The
termination boundary: *L11 can consume evidence and produce
evidence, but it cannot consume its own output as an instruction
to continue.* (Owner distinction retained: this is the line
between AUTOMATION — completing an explicitly governed request —
and AUTONOMY — creating the condition that causes the request to
exist. "L11 consumes a baseline; it does not discover one":
policy → resolve B → L11(B), never L11 → find current B →
compare.)

### D-L11-15 — The consumption-of-evidence rule (LOCKED 2026-09-12, Q-L11-15; owner amendment: L11 structural incompleteness/refusal vocabulary kept explicitly separate from L10's outcome vocabulary)

**The general rule, formalizing D-L11-8's deeper principle:**
*consumption never confers authority. Who may consume which
evidence, for what purpose, under what fixed policy is determined
BEFORE the evidence exists; evidence informs the exercise of
authority a consumer already holds and never expands it.* The
consumption classes below are CLOSED: a consumption not in the
table is refused by default, and a new class is a fresh
architecture decision (the ratchet pattern, third application).

**The six consumption classes:**

**Class 1 — Governance promotion (humans at doors).** Consumes:
comparative/regression packages, advisory content (kept visibly
separate). Policy: the door's own requirements. Authority ceiling:
full discretion — evidence informs, never obligates (D-L11-2/3);
the only record constraint is D-L11-2 Amendment A: evidence
presence/absence explicit in the promotion record.

**Class 2 — Deterministic deployment selection (3→4 consumption
mechanisms).** The D-L11-8 five conditions, now stated as the
GENERAL rule for any runtime consumer, present or future — closed
admitted set, admitted evidence checked at consumption, fixed
registered policy, no governance state written, fail closed to
admission. The router is the instance; the rule is the law. A new
consumption mechanism inherits all five conditions or it is not a
legitimate Class-2 consumer. (Owner: this is the constitutional
test precisely so nobody can later argue a new "benchmark
selector" is not a router and therefore does not inherit D-L11-8.
It does.)

**Class 3 — L11 comparative/regression establishment.** Consumes:
established governed facts per K's registered selectors (v1
vocabulary: L2/L6/L10/benchmark facts), under exact externally
supplied K/S (D-L11-14). Produces packages; no continuation
(no-self-continuation invariant). Fully bounded by
D-L11-4/5/6/7/9/10/14.

**Owner amendment — Class 3 gets NO generic failure vocabulary
analogous to L10's PASS/FAIL/INCONCLUSIVE/UNAVAILABLE/INVALID
(those remain L10-owned).** L11's vocabulary is structural —
whether a proposition/package EXISTS, never whether an evaluation
"passed." The clean model (owner, at lock):
required input/criterion/baseline/admission unavailable before
comparison → no package, refusal/absence fact recorded; required
constituent exists and is unfavorable → complete package with
observed regression; all constituents satisfy their regions →
complete package, clean; comparison succeeds but K has no
ordering → package exists, Δ incomparable/descriptive;
evidence/reference integrity failure → no valid package,
discrepancy/refusal as appropriate; L10 verifier returned
INCONCLUSIVE → that is an L10 fact consumed by K — L11 does not
rename it. Preserved boundaries: no comparison = no package;
incomparable ≠ no comparison (an incomparable result is a
legitimate package if K permits the Δ shape). Full assembly at
Q-L11-16.

**Class 4 — L11 reading its own output: TERMINAL, with exactly two
licensed re-reads.** The dangerous class, given its precise
boundary. (a) UNDERLYING facts consumed by many comparisons — fine
and intended: the same L10 record may feed any number of K's;
facts are reusable. (b) L11 OUTPUT as comparator input — refused
(D-L11-7 Amendment 1, now completed into the general form):
**L11 output is terminal within L11.** Packages exit toward doors
and views; they never re-enter the comparator. Exactly two
licensed re-reads of own output exist: (i) re-derivation/
verification of an existing package — confirming or minting a
DISCREPANCY fact, never a new comparative fact (Q-L11-17); (ii)
stateless rewritings already licensed by D-L11-4 (better-under-K,
resistant-under-S, enumerations, lineage views). Nothing else.
This is the data-plane complement of D-L11-14's control-plane
invariant: no operation triggers an operation, and no output feeds
a comparison — the loop is closed on both planes. (Owner explicit
rule at lock: *an L11-produced package may be represented,
verified, or rewritten, but cannot serve as an evidence
constituent of another L11 comparison in v1.* The two licensed
re-reads are identity-preserving/representational, never
epistemically additive; the prohibited transition P1 → K2 → P2
would let L11 recursively manufacture evidence from its own
conclusions.)

**Class 5 — L11 packages as runtime selection evidence: EXCLUDED.**
Standing wall from D-L11-8 §5 restated as consumption law: Δ is
addressed to doors; it enters no routing or runtime selection
decision. Reversal requires a fresh architecture decision, not an
input field.

**Class 6 — the model reading evidence as data.** In ordinary
governed tasks the model may read packages/views as UNTRUSTED
CONTEXT DATA — informing candidates it authors (D-L11-14 §4),
establishing nothing, triggering nothing. Data-not-instructions
discipline applies: a package is never an instruction to the
model, and a model "acting on evidence" is just authorship with
provenance, subject to every existing wall.

**The matrix, compressed:**

| Class | Consumer | May do | May never do |
|---|---|---|---|
| 1 | Door humans | decide with discretion | be obligated by evidence |
| 2 | Fixed deployment policy | order the admitted set | extend it / write governance state |
| 3 | L11 comparator | produce packages from facts | continue, select, initiate |
| 4 | L11 itself | verify + rewrite own output | re-compare it |
| 5 | Runtime selection | — (excluded) | consume Δ at all |
| 6 | Model in tasks | read as data, author | treat as instruction, establish |

**Constitutional sentence:** *Consumption never confers authority:
who consumes which evidence under what policy is fixed before the
evidence exists. L11's own outputs are terminal — read again only
to be verified or rewritten, never to compare, select, or
continue.*

**Recorded invariants (owner, at lock):** *every L11 consumer has
a pre-existing authority boundary; evidence can narrow, inform, or
satisfy that boundary, but can never enlarge it.* And for L11
itself: *L11 output is terminal with respect to epistemic
production — it may be verified or rewritten, but it cannot become
new comparative evidence.* (Class 6 phrasing kept: *readable does
not mean authoritative, and evidence does not become instruction
merely because the model sees it.*)

### D-L11-16 — Failure and incompleteness: no outcome vocabulary at all (LOCKED 2026-09-12, Q-L11-16; owner refinement: discrepancy is a SUBSEQUENT fact about a produced package, never a third state of the comparison)

**1. The answer to "NO COMPARISON vs INCONCLUSIVE vs new
vocabulary" is: NEITHER — L11 has no outcome enum.** L10's five
outcomes are propositions about what a verification established;
L11's states are structural facts about EXISTENCE. The owner's
D-L11-15 boundary sentence is the law here: *L10 says what a
verification contract established; L11 says whether the evidence
required for a comparison exists and what the registered comparator
derived from it.* Exactly three artifact-level situations exist —
already individually locked, assembled here:

- **Refusal fact** — an accepted invocation could not produce a
  package (precondition failure). Carries a closed REASON CLASS
  (§3). No package exists (D-L11-5 §3).
- **Package** — the comparison executed; Δ recorded whatever it
  shows. Favorable / unfavorable / equal / incomparable are
  CONTENT of Δ under K (D-L11-7), never distinct states.
- **Discrepancy fact** — a re-derivation disagrees with a recorded
  package (D-L10-12 inherited; D-L11-15 Class 4).

**2. The owner's eight scenarios, mapped:** candidate evaluation
fails → the task failure is an L6/L10 fact; at comparison time K's
selectors find no usable evidence → refusal
(evidence-unavailable). Baseline evaluation fails → same.
Candidate/baseline inputs differ → K's declared comparability
requirements violated → refusal (comparability-violation) — NOTE:
this is NOT "incomparable"; incomparability is an ORDERING concept
after a successful comparison (D-L11-7 §4), input mismatch
PREVENTS the comparison. (Owner, preserved verbatim in the
architecture: *input incompatibility prevents comparison; metric
incomparability occurs after a successful comparison.* Fundamentally
different propositions.) Evidence incomplete → refusal per missing
constituent; set-level, no S-package (D-L11-9 case A). Metrics
incomparable → a PACKAGE with incomparable/descriptive Δ — not a
failure. One run unavailable → refusal (evidence-unavailable).
Regression corpus incomplete → no S-package; constituent packages
stand alone without set identity. Evaluator/registry versions
differ from K's provenance requirements → refusal
(provenance-violation); the comparator version cannot differ — it
is pinned by K.

**3. Closed refusal reason classes (the L10 CheckReasonClass
pattern):** unregistered-artifact; withdrawn-artifact;
unadmitted-baseline; evidence-unavailable; comparability-violation;
provenance-violation; integrity-failure. Reasons name the FAILED
PRECONDITION, never quality; the class list is closed and extended
only by schema revision.

**4. L10's tokens never appear as L11 states.**
PASS/FAIL/INCONCLUSIVE/UNAVAILABLE/INVALID occur in L11 artifacts
only INSIDE consumed evidence references, quoted as the L10 facts
they are. An L10 INCONCLUSIVE consumed by K stays an L10 fact — K's
selectors declare whether such evidence satisfies the criterion's
requirements; L11 performs no renaming, no mapping, no L11-side
"inconclusive."

**5. Refusals are terminal, neutral, and non-retriable by
machinery.** Terminal: refusal triggers nothing (D-L11-14
no-self-continuation — no retry). Neutral: a refusal says NOTHING
about candidate or baseline quality — it establishes only that a
named precondition failed; a pile of refusals is not a quality
signal about C, and no derivation may aggregate refusal counts
into anything evaluative (the aggregation rule applies: refusals
enumerate, never strengthen). (Owner, prohibited derivations
enumerated at lock: "many refusals for C → C is bad"; "many
refusals for B → B is unreliable"; "high refusal rate → candidate
regression". A refusal is epistemically neutral about the compared
artifacts. Also confirmed: L10 FAIL does not become an L11
"regression" automatically — whether a particular L10 fact
participates in K's comparison semantics is determined by K's
registered evidence requirements.) Non-retriable by machinery: a human
may re-invoke after fixing the precondition; the new invocation is
a new attributable act.

**6. Incompleteness is representational, never quantified.** No
coverage fields, no percentages, no "partial" status (D-L11-9 §3;
D-L11-11 no-status-column tripwire). Incompleteness is visible as
the ABSENCE of a set-level package plus the presence of whatever
constituents exist.

**Constitutional sentence (owner refinement folded — the three
situations are not mutually exclusive states of one comparison;
the discrepancy is subsequent and never replaces the original,
per D-L11-11 immutability):** *L11 has no outcome vocabulary: an
accepted comparison invocation either produces a package or
produces a refusal fact for a named precondition; a later licensed
re-derivation may produce a discrepancy fact. Everything else —
favorable, unfavorable, incomparable, inconclusive — is content
owned by K's declared semantics or by the plane that minted the
consumed fact.*

The closed model (owner): accepted invocation → precondition
failure → refusal fact; OR comparison succeeds → package → later
re-derivation → agrees (confirmation) or disagrees (discrepancy
fact). The resulting boundary: *L10 has outcome semantics; L11 has
package existence and comparison semantics* — L10 establishes
verification facts; L11 establishes comparison facts or refuses to
establish one.

### D-L11-17 — Reproducibility: same committed inputs, same proposition (LOCKED 2026-09-12, Q-L11-17; owner wording refinement: discrepancy sources are broader than registration — "architectural defect, never normalized")

**The centerpiece (owner):** *reproducibility means reconstruction
of the same PROPOSITION from the same committed inputs — not
merely obtaining the same numerical Δ.* Re-derivation re-verifies
the full conditioned relation: selector application, admission
observations against recorded door-registry hashes (D-L11-5 §2),
comparator computation, region derivations. A matching Δ atop a
broken conditioning chain is NOT a reproduction.

**1. The sufficient conditioning tuple (carried by every
package):** candidate content hash; baseline content hash; K
name@version + SHA-256 (two-way, D-L11-6); comparator registered
name@version; comparator configuration bytes hash (pinned by K);
enumerated evidence ObjectIDs; run identities; admission
observation references including the observed door-registry hash;
Δ in K's declared shape. Sufficiency test, mechanical: a cold
process holding ONLY the package bytes, read access to the L6
record plane, and the governed registries re-derives the
proposition with no other context — possible precisely because
L11 has no mutable state to depend on (D-L11-11). (Owner at lock:
*a Δ without its conditioning chain is not the L11 proposition* —
"same Δ" does not establish "same comparison" unless the entire
conditioning relation reconstructs. And cold reconstruction is the
ARCHITECTURAL PROOF: delete every L11-side cache and mutable
state, retain only package + L6 + registries, reconstruct — if the
proposition changes, something necessary was outside the tuple or
an authority boundary was violated. D-L11-11's cache rule and this
rule reinforce each other; no separate reproducibility subsystem
exists.)

**2. The nine grill points, answered:** (i) tuple as above; (ii)
every constituent referenced by immutable ObjectID — no identity
without bytes (D-L11-11 §3); (iii) K by exact name@version + hash;
(iv) comparator captured as registered name@version, whose
semantics are immutable per version — **constitutional rule: any
semantic change to a comparator is a NEW version; same version,
same semantics, forever** (the code identity behind a version is
governed by the ordinary review pipeline and repo history, the
registration binds the name); (v) config bytes pinned by K,
hash-checked; (vi) cold reconstruction requires no L11 state —
package + L6 + registries only; (vii)–(ix) below.

**3. Determinism requirement (D-L10-3 reapplied).** The comparator
is pure and environment-free: same tuple → bit-identical canonical
Δ serialization. No wall-clock, no randomness, no
platform-dependent arithmetic; canonical serialization is part of
the registered comparator semantics. Stateless derivations
(better-under-K, resistant-under-S) are reproducible for free —
they are functions of package + K.

**4. The three reconstruction results (answering vii/ix — the
discrepancy/unavailable distinction, precise):**

- **CONFIRMED** — all constituents resolve, hash-verify, and the
  re-derived proposition matches.
- **UNREPRODUCIBLE-FOR-MISSING-INPUTS** — a constituent's bytes
  are absent. An AVAILABILITY fact, epistemically neutral (the
  refusal-neutrality discipline): nothing disagrees; re-derivation
  simply cannot proceed. Recorded as a fact naming the missing
  ObjectIDs.
- **DISCREPANCY** — either (a) all inputs present and re-derivation
  yields a differing proposition/Δ, or (b) bytes present but
  hash-mismatched (an integrity/tamper fact at the read boundary,
  D-L10-11). Recorded as a discrepancy fact for Governance —
  which computation is right is Governance's question (D-L11-4).

**5. Reconstruction never repairs (answering viii).** No
reconstruction path writes, amends, or replaces the original
package (D-L11-11 immutability; D-L11-16: the discrepancy is
subsequent, the original stands). Reconstruction is licensed
re-read (i) of D-L11-15 Class 4 — a pure function, the L10
Reconstruct pattern — and it is NOT a new mechanism: no
reconstruction daemon, invoked like everything else (D-L11-14).

**Constitutional sentence (owner refinement folded — a
discrepancy can arise from corruption, implementation defect,
registry integrity failure, or another mechanism failure, not
necessarily the original registration; the response is identical
either way — record, never tolerate or repair):** *Every package
carries its complete conditioning tuple; a cold process with the
package, the record plane, and the registries re-derives the same
proposition — or records a missing-inputs fact or a discrepancy
fact, and never a repaired package. Same committed inputs, same
proposition; anything less is an architectural defect and is never
normalized as an acceptable reproducibility condition.*

(Owner property at lock: *the package is self-describing enough to
reconstruct its own epistemic provenance, but it is never
authoritative about anything beyond the proposition its
conditioning tuple establishes.*)

### D-L11-18 — The security meaning boundary: metric, never meaning (LOCKED 2026-09-12, Q-L11-18; owner addition: semantic proximity does not confer semantic authority)

**1. The categorical answer, formalized.** L11 can never conclude
"this candidate is secure/safer", "this reduces security risk",
"this eliminates the vulnerability", "this is acceptable", "this
satisfies the security requirement", "this should be promoted".
Those are Governance/Themis propositions — Themis is the security
system of record, and the Harness owns no security truth
(constitution). L11's ceiling with security-relevant inputs is
exactly its ceiling everywhere: the conditioned comparative fact —
Δ over metric M under K — where M's meaning was fixed by the plane
that minted it.

**2. What makes a security-relevant metric LEGITIMATE input to K —
four conditions, all required:**

- **(a) Established fact.** M comes from a governed plane — an L10
  outcome count, L6 record content, validated benchmark score,
  gate verdict — minted with its meaning fixed there (D-L11-6 §5).
  A quantity nobody governed is not a metric; it is a claim.
- **(b) Provenance-descriptive naming, never normative renaming.**
  The selector references M by its FACT identity: "count of L10
  FAIL outcomes under contract sast-scan@2" — never "number of
  security failures" as a fresh L11 name. Selector vocabulary
  describes provenance; it must not embed a judgment the
  referenced plane did not make. `vuln_finding_count_delta`
  describes a provenance-derived quantity; `security_improved`
  asserts meaning — refused at registration.
- **(c) Arithmetic only.** The comparator computes over M; it
  never interprets M's security significance. No threshold in K
  acquires the meaning "acceptable risk" — a region is a declared
  region; its consequence belongs to the door (D-L11-13 two-stage
  normativity).
- **(d) Sensitivity inheritance.** Security-relevant evidence may
  be sensitive; packages carrying it inherit the classification of
  their constituents, and existing redaction/egress rules apply
  unchanged (D-L10-14/D-L10-18 lineage). Comparison confers no
  declassification.

**3. The three mechanical walls against the normative slide (the
"interesting case" — metrics that SOUND normative):**

- **Schema wall.** No field in K, packages, or derivations can
  carry a security predicate. Registration review explicitly asks:
  does any metric name, region name, or description assert a
  security proposition? Assertion → refused; description of
  provenance → admissible.
- **Region-name wall.** Regions are named STRUCTURALLY
  (improvement region, non-regression region) — never semantically
  ("safe region", "acceptable-risk region": refused at
  registration).
- **Derivation wall.** better-under-K over a security-relevant
  metric derives exactly "better-under-K@v" and nothing warmer —
  no view, aggregation, or summary may rename it toward "safer".
  Criterion rationale/documentation is advisory text and cannot be
  quoted as established meaning.

**4. Who turns Δ into security meaning: the door, in its own
plane.** Governance reads "exploitable-finding count 3→1 under K
against admitted B" and concludes what it concludes; any recorded
security conclusion lives in Themis-owned/Governance artifacts,
never in an L11 package. L11 delivering evidence TO a security
decision is not L11 participating IN the security decision.

**5. The double-negative subtlety.** Even "this candidate is NOT
worse security-wise" is prohibited as an L11 proposition —
direction symmetry does not rescue it. The maximum is the bounded
enumerative form: "no observed regression under registered S@v"
(D-L11-9), whose security interpretation is the door's.

**Constitutional sentence:** *A security-relevant quantity enters
L11 as the fact some governed plane established, is compared as a
number, and leaves as Δ; at no point does the criterion, the
comparator, or any derivation say what that Δ means for security —
Themis and Governance alone turn comparative evidence into
security meaning.*

**Recorded invariant (owner, at lock — the reviewer's test for
future additions):** *semantic proximity does not confer semantic
authority.* A metric named "vulnerability count" inside an L11
package does not make L11 responsible for the meaning of
vulnerability; a threshold beside it does not make L11 responsible
for the meaning of acceptable risk; delivering the package to a
security workflow does not make L11 part of the security decision.
(Owner note: this closes the loophole where L11 acquires security
authority through vocabulary, thresholds, metric names, summaries,
or "obvious" interpretations. Also confirmed: L11 cannot promote
an arbitrary model assertion into a metric merely by putting a
number around it — condition (a) stands.)

### D-L11-19 — Recursion without an L12 (LOCKED 2026-09-12, Q-L11-19; no architectural amendment)

**1. The answer: the recursion was never open.** L11 was never
self-governing, so self-improvement was never self-authorized.
Every artifact that governs L11's behavior already lives at a door
L11 does not own: criteria and sets at the Governance registries
(L11 owns format/loader, never admission — D-L11-6 §2);
comparators and L11's own implementation in the ordinary code
pipeline (Class-2/3/4, human review). The owner's formulation is
the whole decision: *L11 can be the TARGET of a Candidate; L11
cannot be the authority that admits or activates that Candidate.*
There is no L12 because the fixed point of every layer's authority
is already Governance — humans — and recursion terminates there,
not in another layer.

**2. The nine cases, mapped to existing doors:**

| Change | Route | Door |
|---|---|---|
| Change a K | criterion-revision Candidate → new K version | Governance, criterion registry |
| Change an S | regression-set-revision Candidate → new S version | Governance, set registry |
| Comparator implementation | ordinary code change + registration of comparator@v+1 (semantic change = new version, D-L11-17) | code pipeline + Governance registration |
| Comparator configuration | pinned inside K → K revision | criterion registry |
| Pinned fixtures/goldens | parameters of K (D-L11-13) → K revision | criterion registry |
| L11 implementation itself | ordinary Class-2/3/4 change; never a Ratchet subject | code pipeline (+ Class-4 stop-and-ask where architectural) |

Family extension: criterion-revision and regression-set-revision
join the Candidate family list — exactly the D-L11-3 rule
"introduced only as the existing doors actually need it"; the
doors exist (the L11-format Governance registries), the walls are
identical.

**3. Evaluating a new L11 version against the old — NOT with L11
evidence.** The already-locked walls make L11-about-L11 comparison
structurally impossible in v1: the evidence-selector vocabulary
excludes L11 output (D-L11-7 Am. 1, D-L11-15 Class 4), so a
criterion whose subject matter is L11's own packages/behavior is
UNREGISTRABLE. A new comparator or L11 version is therefore
admitted on ordinary engineering evidence — tests, reviews,
registers, the same machinery that proved L7–L10 without L11's
help. The Ratchet does not measure itself; the pipeline that built
it measures it. (Owner at lock: no "Ratchet benchmark proving the
Ratchet is better" — not a missing capability, an INTENTIONAL
architectural exclusion; it prevents the Ratchet from
manufacturing the evidence needed to validate its own
modifications.)

**4. Anti-circularity: a criterion cannot define its own
success.** Only a REGISTERED K measures anything (unregistered =
data); a proposed K measures nothing pre-admission — so "evaluate
K-candidate under K-candidate" is structurally void. Post-
admission, comparing artifacts under K@2 vs K@1 yields two
separate enumerated packages (never a merged "K@2 is better"
claim — cross-criterion composition is banned, D-L11-7 §6). The
gate against a self-flattering criterion is where it always was:
Governance registration review plus the precommitment test.

**5. The closing invariant:** *no chain exists in which
L11-produced evidence is a sufficient condition for modifying
L11.* Evidence may inform the humans at the doors; it never gates,
authorizes, or triggers the change (consumption never confers
authority — D-L11-15; no self-continuation — D-L11-14; evidence
never obligates — D-L11-2). And the attention asymmetry applies at
its maximum here: a Candidate targeting L11's own machinery is the
peak self-serving-evidence risk, so provenance raises registration
review attention — never lowers the bar and never fast-tracks.
(Owner formulation at lock: *self-targeting provenance increases
scrutiny; it never increases authority* — more review attention,
no altered gate, no bypass, no privileged treatment. The legal
chain: L11 evidence → human attention → Candidate → ordinary
review → Governance decision → new governed artifact. Never: L11
evidence → automatic acceptance → new L11 behavior — that would be
L12 authority without the name.)

**Constitutional sentence:** *The Ratchet can be improved, but
never by its own authority: every artifact that governs L11 is
admitted at a door L11 does not own, on evidence L11 did not mint
about itself — the recursion terminates in humans, not in an L12.*

### D-L11-20 — Constitutional closure (LOCKED 2026-09-12, Q-L11-20; closure audit PASSED; Gate 0 knowledge rule recorded)

A closure audit, not a design. The owner's 18 points and the final
question, answered against the locked record.

**1. The constitution index (points 1–14 → locked decisions):**
boundary/purpose → D-L11-1; authority ownership → D-L11-1 split +
D-L11-5/6/9/13; seams (L6/L7/L9/L10/router) → D-L11-8/10/11;
artifact inventory → §2 below; registered-vs-instance identity →
D-L11-11 §2; registries → D-L11-6/9; Candidate + Evaluation Plan →
D-L11-3/10; package semantics → D-L11-4/9; persistence +
reconstruction → D-L11-11/17; failure/refusal/discrepancy →
D-L11-16; security prohibition → D-L11-18; consumption classes →
D-L11-15; automation/autonomy → D-L11-14; recursion → D-L11-19.
Every point has exactly one home; no point is homeless; no two
decisions contend for the same point.

**2. The complete, closed artifact inventory:**
- REGISTERED (name@version + hash, Governance-admitted): Criterion
  K; Regression set S; comparator version bindings.
- INSTANCE (content hash only, L6-stored): Candidate; Evaluation
  Plan; comparative-evidence package; regression-evidence package;
  refusal fact; discrepancy fact; missing-inputs (reconstruction)
  fact.
- DERIVED (stateless, never stored): better-under-K;
  resistant-under-S; latest-per-(C,B,K); enumerations; lineage
  views; conformance displays.
Nothing else exists. Anything not on this list fails the D-L11-12
architectural test.

**3. Implementation obligations (point 15) — every one an
instantiation of a proven pattern:** criterion + set registry
loaders (verification/registry.go pattern); criterion schema
loader (contract.go pattern: closed schema, duplicate-key,
readGoverned, two-way identity); comparator interface + v1
comparator(s) (canonReport / gate-Compare precedent; pure,
deterministic, canonical serialization); selector resolution over
L2/L6/L10/benchmark records; package assembly; Reconstruct (L10
pattern); derivations (views.go pattern); Candidate/Plan schemas +
L6 StoreObject persistence; a synchronous invocation surface;
attribution representation; AST walls (no registry write APIs, no
timers/tickers/watchers, terminal-output enforcement); API-closure
guards; registers + live proof slice (candidate → governed
evaluation → package → human promotion via existing door →
regression admission — the Q-L11-12-old proof gate, now point 15).

**4. The final question — what must implementation still invent?
NOTHING architecturally.** Four enumerated implementation CHOICES
remain, each inside locked constraints, each through ordinary
review, none an invention: (a) attribution representation —
existing task-envelope field vs narrow additive L6 amendment
(D-L11-10 §5); (b) fixture/golden physical housing —
registry-plane files vs L6 objects by size (D-L11-13 §5); (c) the
v1 comparator set — at least one concrete family for the proof
slice (e.g. numeric-score-delta over benchmark/L10 facts); (d) the
invocation surface — CLI subcommand vs service endpoint (Class-3
review either way). Anything else discovered mid-implementation
that does not fit the locked inventory is Class-4 stop-and-ask —
never a helper subsystem, never developer convenience.

**5. Residuals (point 16), recorded:** pre-L11 standing residuals
unchanged (L5 process-exec amendment; live telemetry grill; L6
GC-anchoring ADG; L9 loader hardening; C3 attestation; L8
delegation; equivalent-mutant record). L11-minted residuals:
higher-order/meta-comparison (fresh architecture decision if ever
wanted — D-L11-7 Am. 1); Δ-consuming runtime selection (fresh
decision — D-L11-8/15); multi-baseline criteria (D-L11-5 §5);
physical unification of benchmark-plane compare/variants with L11
comparators NOT performed in v1 — the benchmark plane stays
bench-owned, single-home holds because L11 consumes its facts
without forking its semantics; knowledge-family live proof may
lag the in-repo families (egress package specified in D-L11-3;
owner decides at Gate 0 whether the v1 proof slice includes it).

**Gate 0 rule (owner, DECIDED at lock):** *the v1 proof slice
includes one knowledge-family egress path where the existing
Themis ingestion door is available; otherwise the family remains
explicitly unproved/residual and no substitute L11 knowledge
authority is introduced.* The knowledge proof demonstrates ONLY
that the existing walls hold: L11 can construct the
knowledge-family Candidate/egress representation; its content is
hash/provenance bound; L11 does not ingest or mutate Themis
knowledge; the receiving Themis mechanism remains the sole
admission authority; L11 evidence does not obligate ingestion; the
handoff does not become a second knowledge store in L11. No new
knowledge ingestion mechanism inside the Harness; no fake local
substitute — the constitution is never weakened to make a proof
green.

**6. Forbidden components (point 17), consolidated:** feedback
subsystem; candidate registry/status store; champion registry; L11
event stream; authoritative cache; scheduler/timer/watcher/queue;
L11 executor; baseline manager; version-selection authority;
coverage quantification; outcome enum; security predicates;
meta-comparison; self-evaluation; optimization objective.
**Scaffold disposition:** DELETE ALL pre-grill scaffold dirs
(src/harness/ratchet/{candidates,evaluations,feedback,promotion,
regression}) — feedback/ and promotion/ name subsystems the grill
forbade (D-L11-12; promotion is door-owned); the rest are
pre-grill guesses. Implementation creates justified structure from
the locked design (the L10 lesson: structure follows decisions,
never precedes them).

**7. Genuine-future-gap conditions (point 18) — a NEW architecture
decision is required iff someone wants:** a seventh consumption
class; L11 output as evidence input; Δ in runtime selection;
multi-baseline Δ semantics; non-human initiation authority; a
third durable plane; corpus curation authority; an L11 outcome
vocabulary; an L11-about-L11 criterion. Anything on this list
appearing as an implementation "convenience" is an architecture
violation, not a feature.

**Constitutional sentence:** *L11 is architecturally frozen: every
component the implementation will build instantiates a locked
decision through an already-proven pattern; anything that cannot
is a recorded residual or a new architecture decision — never a
convenience.*

**Owner closure audit result (at lock):** constitutional coverage,
artifact inventory, authority ownership, seams, persistence,
failure semantics, security-meaning boundary, consumption model,
automation boundary, recursion boundary, implementation
architecture, forbidden subsystems, future-gap criteria — ALL
CLOSED. The artifact inventory is the implementation's
architectural WHITELIST: anything else needs explicit
architectural justification. Scaffold deletion confirmed
(retaining feedback/ or promotion/ merely because directories
exist would be architectural inertia — the opposite of this
grill's outcome).

**The implementation rule (owner, strict):** *if implementation
discovers something that appears necessary but is absent from the
locked constitution, do not solve it locally. Stop at the
boundary, classify it as implementation detail, residual, or
genuine architecture gap, and reopen the grill only for the last
category.* That is what makes the 20/20 closure meaningful rather
than merely documenting agreement.

**GRILL STATUS: D-L11-1 through D-L11-20 — 20/20 LOCKED
(2026-09-11..12). Architectural grilling STOPPED by owner
direction. Next phase: proposal.md, tasks.md, traceability
skeleton, then Gate 0, then implementation.**

### [Reallocated] Implementer material on criteria/comparator/packages (2026-09-11; formerly proposed as D-L11-4, superseded by owner restructure 2026-09-12)

Retained verbatim below as implementer input, not a closure.
Reallocation: criterion schema, registry, and the comparator-binding
open decision (registered code identity vs declarative Δ language) →
Q-L11-6; ordering/"better" → Q-L11-7; consume-only-established-facts →
folds into D-L11-4 §1-2 above; admission attestation → D-L11-4 §1 and
Q-L11-5; reproducibility/cold re-derivation → Q-L11-17; variants →
Q-L11-13/Q-L11-15; not-a-gate/not-automatic → Q-L11-14; failure
vocabulary → Q-L11-16.

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

## 3. Grill — question list (RESTRUCTURED BY OWNER 2026-09-12)

The owner replaced the implementer draft (old Q-L11-4..12) with the
list below. Old-draft items with no direct slot are CARRIED, not
dropped: benchmark-plane reconciliation → Q-L11-13/15; knowledge
boundary → Q-L11-11/18 + D-L11-3's knowledge paragraph; adversarial
register (gaming, overfitting, self-serving evidence, baseline
manipulation, candidate spam) → Q-L11-14 and closure at Q-L11-20;
proof gate/registers/live slice → Q-L11-20; purpose-attribution
residual → Q-L11-10.

| Q | Question |
|---|---|
| Q-L11-1 | **CLOSED → D-L11-1.** Evidence-not-authority boundary; provides-not-enforces; ownership split; hard prohibitions + positive authority. |
| Q-L11-2 | **CLOSED → D-L11-2.** Improvement ladder; baseline rule as ladder law; evidence-absence explicit; downward moves are acts; criterion-relative vocabulary. |
| Q-L11-3 | **CLOSED → D-L11-3.** Candidate = content-addressed proposal data; no lifecycle state; single-home promotion truth; 1→3→2 epistemic; no trial admission. |
| Q-L11-4 | **CLOSED → D-L11-4.** Conditioned comparative fact as the L11 atom; six claims disposed; X→X′ artifact ladder; admission as observation; enumeration-not-strength aggregation; direction symmetry; eligibility prohibition. |
| Q-L11-5 | **CLOSED → D-L11-5.** Doors own baselines, L11 observes and mechanically applies registered constraints; admission = production precondition (refusal, never INCONCLUSIVE); no silent baseline substitution; pairwise only; claims never choose. |
| Q-L11-6 | **CLOSED → D-L11-6.** Governance-owned criterion registry, L11 format+loader; three homes for semantics; closed schema; comparator = registered code identity (option A LOCKED); applicability ≠ selection; ordering is criterion-relative derivation. |
| Q-L11-7 | **CLOSED → D-L11-7.** Ordering taxonomy (none/per-metric/dominance/scalarization); precommitment test; ties/incomparability first-class; v1 no meta-comparison; scalarization = projection; no optimization objective. |
| Q-L11-8 | **CLOSED → D-L11-8.** Selection ≠ promotion; five conditions; selection = one form of 3→4 consumption; predeclared fallbacks; no side-effect on admissible set; Δ never a selection input (v1). |
| Q-L11-9 | **CLOSED → D-L11-9.** Enumeration under registered S@v; doors own set significance and exact resolution; no partial packages; bounded resistant-under-S derivation; incomparability per K; never a gate; completeness = coverage. |
| Q-L11-10 | **CLOSED → D-L11-10.** Describe-and-consume, never run; eight verbs disposed; inert declarative Evaluation Plan (never authority); conformance = record-to-plan matching; attribution = provenance only; 15 recorded invariants. |
| Q-L11-11 | **CLOSED → D-L11-11.** Two durable planes only; registered names vs instance hashes; L6 stores, L11 derives statelessly; supersession = claim; negative list; production via existing L6 mechanisms; refusal = fact about attempt. |
| Q-L11-12 | **CLOSED → D-L11-12.** "Feedback" is not an architectural concept in L11; dissolution table; no type/store/channel; loop is human-governed; reservoir surface removed; feedback/ scaffold forbidden; the architectural test. |
| Q-L11-13 | **CLOSED → D-L11-13.** Corpus = registered criteria + hash-pinned parameters (case represented through K, not a third concept); enterprise wall; goldens record-derived or declared; thresholds two-stage; results never modify their own test population. |
| Q-L11-14 | **CLOSED → D-L11-14.** Complete-never-initiate; exact pins; no-discard from accepted invocation; candidate generation refused as machinery; no timers/watchers/queues; no L11 operation creates another L11 operation. |
| Q-L11-15 | **CLOSED → D-L11-15.** Consumption never confers authority; six closed classes; L11 output terminal (two licensed re-reads, never a constituent); Class-2 five conditions as constitutional test; L11/L10 vocabulary separation. |
| Q-L11-16 | **CLOSED → D-L11-16.** No outcome enum: package or refusal fact, discrepancy subsequent; closed reason classes; refusals terminal/neutral/non-retriable; incompleteness representational; L10 tokens foreign to L11. |
| Q-L11-17 | **CLOSED → D-L11-17.** Same inputs → same PROPOSITION; sufficient tuple; comparator semantics immutable per version; determinism; three reconstruction results; never repairs; defects never normalized. |
| Q-L11-18 | **CLOSED → D-L11-18.** Metric never meaning; four legitimacy conditions; three walls; sensitivity inheritance; double-negative prohibited; semantic proximity confers no authority. |
| Q-L11-19 | **CLOSED → D-L11-19.** Recursion was never open: L11 targets admitted at doors L11 doesn't own; L11-about-L11 criteria unregistrable; proposed K measures nothing; no evidence chain modifies L11; scrutiny never authority. |
| Q-L11-20 | **CLOSED → D-L11-20.** Closure audit PASSED; artifact inventory = implementation whitelist; nothing to invent (4 flagged choices); scaffold deletion; Gate 0 knowledge rule; strict implementation rule. **GRILL 20/20 LOCKED.** |

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
