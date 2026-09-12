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

### D-L11-5 — Baseline authority (PROPOSED 2026-09-12, closing Q-L11-5; AWAITING OWNER LOCK)

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
reconciliation semantics D-L11-3 forbids.

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
Withdrawn = not admitted for NEW comparisons; historical packages
remain interpretable (L10 withdrawal semantics reapplied).

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
observation; L11 itself never owns the policy.

**5. One baseline per package.** The comparative atom is pairwise:
exactly one candidate, one baseline, one criterion. Multiple
baselines = multiple packages; combining them is aggregation by
enumeration (D-L11-4 Amendment 3). No multi-baseline Δ semantics.

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

**Constitutional sentence:** *A baseline is whatever the owning
door's authoritative record says it is; L11 may observe that record,
condition comparisons on the observation, and refuse without it — it
may never determine, correct, or currency-judge a baseline itself.*

Open item for owner: the closed vocabulary of criterion-declared
baseline constraints (§4) is defined at Q-L11-6 with the rest of
what a criterion can say.

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
| Q-L11-5 | **Closure PROPOSED → D-L11-5; AWAITING OWNER LOCK.** Baseline authority: doors own baselines, L11 observes; admission as production precondition; currency observed + criterion-constrained; one baseline per package; candidate claims never choose. |
| Q-L11-6 | Comparison criterion ownership: who owns the criterion registry (L11 / Governance / L10 / existing Themis mechanism); what exactly a criterion can say. Carries the comparator-binding open decision (registered code identity vs declarative Δ language). |
| Q-L11-7 | What "better" actually means: who resolves multi-metric tradeoffs when K yields metrics without an ordering; keeping tradeoff resolution out of L11. |
| Q-L11-8 | Model-router/evaluation tension: evaluation result → runtime model selection — prohibited automatic promotion/reliance, or deployment policy consuming evidence? |
| Q-L11-9 | Regression resistance: what "provides regression-resistance evidence" establishes — enumerative report vs "no observed regression"; what constitutes the regression set; keeping L11 from becoming a quality gate. |
| Q-L11-10 | Evaluation execution: can L11 create tasks, choose Skill/model/evidence/verifier, repeat, A/B, schedule — or is it evidence-consumer only? (L7/L10/L11 seam.) |
| Q-L11-11 | Candidate/evidence lifecycle: durable machinery for identity, evidence refs, comparison identity, lineage, reproducibility, supersession — what L6 stores vs what L11 derives. |
| Q-L11-12 | Feedback: is ratchet/feedback evidence, observation, candidate input, governance commentary, evaluation result — or a dangerous catch-all to eliminate? |
| Q-L11-13 | Regression corpus / reference set: who owns regression cases, reference inputs, expected outputs, golden examples, acceptance thresholds; L11 artifacts or existing Governance/Themis knowledge? |
| Q-L11-14 | Automation boundary: may the Ratchet automatically generate candidates, run evaluations, compare, open proposals, request review — where automation stops, mechanically enforceable. |
| Q-L11-15 | Router and deployment policy: formally distinguish evidence→Governance-promotion from validated-evidence→deterministic-deployment-selection, if the latter is legitimate (consumption-of-evidence rule). |
| Q-L11-16 | Failure and incompleteness: failed/differing/incomplete/incomparable/unavailable inputs — NO COMPARISON vs INCONCLUSIVE vs new vocabulary; resist a second copy of L10's outcomes. |
| Q-L11-17 | Reproducibility: reconstruct "why did Ratchet say C compared favorably with B" — L10-strength provenance; must comparison itself be deterministic? |
| Q-L11-18 | Security meaning boundary: can Ratchet ever conclude "safer"/"reduces risk"/"acceptable"? (Preliminary: no — Governance propositions.) |
| Q-L11-19 | Ratchet's own improvement: candidates targeting L11's own machinery/criteria/corpus — the recursion, answered without creating an L12. |
| Q-L11-20 | Constitutional closure: freeze ownership, seams, artifact types, registries, schemas, persistence, failure semantics, automation limits, residuals, proof obligations; scaffold disposition. |

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
