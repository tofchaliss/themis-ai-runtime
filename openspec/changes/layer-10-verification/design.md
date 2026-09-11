# Design: Layer 10 — Verification & Observability

Grill OPEN 2026-09-11. §2 collects folded decisions as questions close; §3 is
the question list.

## 0. Position in the flow

L10 sits after execution and before governance meaning: it evaluates what
deterministic computations established about an execution, and exposes what
happened. It owns verification *semantics* (what a gate outcome means
mechanically), never security *meaning* (what Themis accepts). Verification
computations themselves are L4-registered capabilities executed under L5;
their records live in L6. L10 adds no runtime authority to any layer below.

## 1. Hard invariants (inherited, not grillable)

- Model output advisory; the model cannot self-certify success (arch §13).
- Deterministic controls enforce authorization, policy, state, verification,
  and security invariants (Day-0).
- Verification ownership chain (D-L9-14, locked): Skill requires → L7 enforces
  gate structure → L4 authorizes → L5 executes → L2 classifies/transports →
  L10 evaluates → Governance decides. No layer does another's job.
- Derived-authority preconditions (D-L9-14): registered computation +
  deterministic execution + governed environment + complete computational
  provenance — eligibility is capacity, not promotion; model-relayed claims
  ("I ran the tests; PASS") stay external-untrusted.
- L6 is the sole durable record plane; record truth ≠ security truth; no
  second evaluation subsystem (locked plan: L11 generalizes, never duplicates).
- Governed artifacts: versioned, DisallowUnknownFields, SHA-256 into the
  record, fail-closed loaders; nothing defaulted; absences typed and named.
- COMPLETED remains a structural claim about the walk and its required gate
  outcomes (D-L9-14); acceptance is Themis governance.

## 2. Locked decisions (fold target)

### D-L10-1 — Boundary and authority (LOCKED 2026-09-11, Q-L10-1)

L10 is the deterministic evidence-verification and execution-observability
layer.

**Verification** is deterministic evaluation of governed evidence against a
registered verification contract, producing a typed outcome with complete
computational provenance. L10 owns the verification mechanism and evaluates
the semantics of registered verification contracts; it does not author those
contracts unilaterally.

**Observability** is exposure of what governed execution actually did,
derived from the authoritative record — expository, never evaluative. L10 may
consume and expose structural-consistency results established by their owning
mechanisms, including L6 durable-record verification (Verify/ScanReachable)
and L7 workflow replay; L10 does not absorb or supersede those authorities.
(Owner amendment closing the implementer's challenge: trace-consistency
ownership stays single-homed — L6 owns "is the durable record structurally
sound/reachable?", L7 owns "does this record replay as the governed workflow
walk?"; L10 consumes/exposes those facts and may use registered contracts
whose inputs include them, but cannot reimplement either proposition and
declare it authoritative.)

**L10 may establish:** a mechanical fact — evidence E satisfies contract C@v
under verifier V@h in governed environment X; a typed contract outcome — the
registered verification algorithm produced a defined outcome over the
governed evidence; eligibility for L2 derived classification when the D-L9-14
preconditions actually hold; a bounded typed gate signal into L7's existing
event taxonomy. These facts never establish security meaning.

**L10 can never establish or perform:** security meaning, Enterprise
Position, acceptance/adequacy/safety judgments, promotion of model claims,
authorization of anything, mutation or reinterpretation of history, or
execution of capabilities outside the existing L4→L5 path.

**Structure:** one layer with two distinct mechanisms — verification and
observability. They share the constraints of no security authority and no
second history, but their functions do not blend: verification computes new
typed facts; observability exposes recorded facts. An observability view can
never become a verdict, and a verification outcome enters durable history
only through L6.

**Three standing prohibitions:** L10 is not a second Security Governance
engine. L10 is not a second durable-record plane; L6 remains the sole
authoritative durable history. L10 is not a second orchestrator; verifier
computations execute only through L4-authorized capabilities under L5, and
L10 cannot create or alter workflow control.

**Foundational three-proposition model (carried through the whole grill):**
mechanical validity ("did verification execute correctly?") → contract
outcome ("what did the registered verifier establish?") → security meaning
("what does this mean for the CVE / product / release?"). The first two
belong to L10; the third never does.

### D-L10-1a — The verification ladder (PROPOSED 2026-09-11, closing Q-L10-2; awaiting explicit owner lock)

Four positions, none skippable, each upward transition requiring its own
mechanism: (1) **model says X** — advisory, external-untrusted, zero
verification standing; includes model-relayed claims *about* verifier runs (a
claim about verification is not verification). (2) **evidence shows X** —
governed recorded bytes with provenance; an *input* to verification, never a
conclusion; its L2 class is whatever its source earned. (3) **deterministic
verifier establishes X′** — typed contract outcome + mechanical validity from
registered computation under the D-L9-14 preconditions. The prime is
load-bearing: the verifier establishes the *contract proposition* ("test
suite exited 0 on commit H"), always narrower than the security proposition X
("the fix works"); the X′→X gap is permanent and belongs to Governance.
(4) **Governance accepts X as security truth** — outside the harness, never
automatic, never inferable from any L10/L7 state. Transition mechanisms:
1→2 recording through governed channels (L2/L5/L6); 2→3 a registered contract
executed via L4→L5 with complete provenance; 3→4 a governance act. No
mechanism exists — or may be built — that performs 1→3 or 2→4 directly.

### D-L10-2 — Verification contracts (LOCKED 2026-09-11, Q-L10-3; registry fork resolved (a))

A Verification Contract is a governed, versioned, hash-pinned declarative
specification that defines how a registered verifier capability evaluates
specified governed evidence and maps its deterministic result into L10's
closed outcome vocabulary.

Contract authorship, governance registration, and Harness machinery ownership
are distinct. Anyone permitted to author may produce a contract proposal;
authorship confers no trust or authority. A contract becomes executable only
through a Governance registration act into the append-only **L10 Contract
Registry**. The registry has no model- or Harness-accessible write
capability; registration is an external governed act. Registry admission is
Governance-owned; registry machinery is Harness-owned; registry content is
not Harness authority. An unregistered contract-shaped artifact is DATA ONLY
and cannot be evaluated — no fallback to closest-match, latest, name-only
resolution, or contract-by-value.

A contract contains only the closed-schema fields defined by the L10 contract
schema, unknown fields refused, loading fail-closed: exact name@version
identity; exact L4-registered verifier capability identity/version/hash;
typed evidence input specification — the contract specifies what evidence
**kinds** it requires, and the *evaluation instance* resolves that
specification to concrete governed evidence objects (task state stays outside
contract identity, preserving reusability); pinned and hashed verifier
configuration; mapping of registered verifier results into the fixed L10
outcome vocabulary; failure-semantics mapping; required computational
provenance; contract SHA-256 commitment over its canonical representation.

A contract cannot contain executable logic, arbitrary expressions, scripts,
custom interpreters, or contract-by-value verifier definitions. Its verifier
binding is a reference into the closed L4 capability registry; it cannot
create, redefine, or authorize a capability. L10 owns verification-contract
identity and semantics; L4 owns capability identity and authorization; the
contract references the capability, never absorbs or redefines it (the L9
pattern).

A Skill may require a registered Verification Contract only by exact
immutable pin. A Skill cannot define, inline, parameterize, or alter
verification semantics.

**Outcome anti-smuggling (standing L10 invariant):** the L10 outcome
vocabulary is constitution-owned and closed. A contract maps the registered
verifier's defined results into that vocabulary but cannot introduce new
outcome names or import Security Governance vocabulary. Security-semantic
propositions (NOT_AFFECTED, REMEDIATED, SECURE) are structurally not L10
outcomes. Contract names, descriptions, and descriptive metadata are
uninterpreted and carry no authority. *A Verification Contract can determine
an L10 outcome, but cannot manufacture a security proposition.*

A Verification Contract establishes the deterministic relationship between
governed evidence, a registered verifier, configuration, and an L10 outcome.
It does not establish the security meaning of that outcome.

### D-L10-3 — Determinism boundary (LOCKED 2026-09-11, Q-L10-4)

Determinism is a registration property of the verifier capability, verified
by definition and enforced by refusal; it is never an observed property
claimed after the fact.

A capability is eligible for registration as a verifier only if, for the same
contract, capability identity/version/hash, pinned configuration, and
governed evidence objects, it produces the same **canonical** verification
result, independent of time, host, and uncontrolled environment noise. Any
declared residual nondeterminism must be bounded by a deterministic,
registered, hashed **canonicalization mechanism that is part of the verifier
capability** — never supplied or altered by the Verification Contract or the
task. The registration claim is not "identical raw bytes" but "same canonical
result for the same governed inputs and registered configuration." Ordering,
whitespace/serialization, and equivalent-representation differences are
potentially canonicalizable; uncontrolled randomness, time-dependence, or
network/environment-dependence affecting the substantive result are not
acceptable in v1; retry-until-the-desired-answer is never acceptable.

A Verification Contract may bind only a capability registered as
deterministic-verifier-eligible. Binding an ineligible capability is
structurally refused and fails closed.

Deterministic verification is the only producer of L10 contract outcomes.
Model assessment is advisory content and may be recorded as evidence, but a
model is not a verifier in v1 — the exclusion rests on the requirement for a
registered, reviewable computation, not merely replayability or temperature.
Model-relayed claims remain untrusted claims and never become verifier
outcomes. Human verification remains outside the Harness in v1 and does not
become an in-walk L10 outcome. Governance determination consumes L10
outcomes and is never produced by L10.

If a registered verifier produces a result outside the contract's declared
result mapping, evaluation fails mechanically with the applicable L10
invalid/failure outcome; no guessing, nearest-match, or semantic
interpretation is permitted.

Runtime detection of nondeterminism is evidence that the verifier's
registration premise has failed — surfaced as a typed fact; never averaging,
result selection, or retry-until-success. A rerun is a new evaluation
instance with its own complete record; no evaluation overwrites, supersedes,
or erases another.

**Registration establishes eligibility; execution establishes the actual
result.** Runtime evidence contradicting the registration premise is a
governance/registration defect, never permission for L10 to dynamically
redefine the verifier's trust level (the L4/L9 principle: trust properties
are established at the registration boundary, not invented opportunistically
during execution).

### D-L10-4 — Outcome vocabulary and semantics (LOCKED 2026-09-11, Q-L10-5; substantively pre-closes Q-L10-10)

L10 has a closed outcome vocabulary consisting of PASS, FAIL, INCONCLUSIVE,
UNAVAILABLE, and INVALID. Two conceptual classes:

ESTABLISHED — PASS (contract established the positive condition), FAIL
(contract established the negative condition). NOT-ESTABLISHED —
INCONCLUSIVE (verifier completed but could not determine), UNAVAILABLE
(verification could not be performed), INVALID (verification / evidence /
result was mechanically invalid). FAIL means the verifier established the
negative condition defined by the contract; the other three mean the
contract did not establish its proposition.

PASS and FAIL are contract outcomes: they establish respectively the
positive or negative condition defined by the registered verification
contract. INCONCLUSIVE is a contract outcome indicating that the registered
verifier completed but could not establish either condition. UNAVAILABLE and
INVALID are machinery-reserved outcomes indicating that verification was
unavailable or mechanically invalid; they cannot be produced through
contract result mapping — otherwise a contract could launder machinery
failure into a decision (or a decision into failure), destroying the
mechanical-validity boundary.

A Verification Contract may map registered verifier results only to PASS,
FAIL, or INCONCLUSIVE. It cannot introduce additional outcome values or map
machinery failure into a contract outcome.

L10 preserves all five outcomes as distinct typed facts. When a verification
result participates in workflow control, L7 receives the corresponding typed
event (verification-pass / -fail / -inconclusive / -unavailable / -invalid,
subject to the final event vocabulary) and the reviewed workflow lattice
determines the transition — L10 says FAIL, never "FAIL → remediation"; one
reviewed workflow may route FAIL to remediation and INCONCLUSIVE to
evidence-gathering, another may legitimately choose differently. A
PASS-requiring gate fails closed for every outcome other than PASS, but L7
must not collapse the non-PASS outcomes into one event — that would discard
deterministic information the reviewed walk legitimately uses. Gate safety
does not require outcome collapse.

None of the five outcomes carries security meaning. In particular, FAIL does
not mean "vulnerability exists," "CVE is open," or any equivalent security
proposition — only that the registered contract established its defined
negative condition. The vocabulary is constitution-owned: any change is a
constitutional amendment (owner act, version bump), never contract-supplied,
and the amendment process must refuse any value that names or implies a
security state. No ordering, severity scale, or partial credit exists among
outcomes; scores a verifier produces are raw evidence collapsed by the
contract mapping, never vocabulary members.

### D-L10-5 — Observability is derivation, never recording (LOCKED 2026-09-11, Q-L10-6 + Q-L10-7)

L6 is the sole authoritative durable execution record. L10 observability is a
read-only deterministic derivation layer over: (a) the authoritative L6
durable record, including its event stream, content-addressed objects,
manifests, and task records; (b) registered governed artifacts used as
interpretation keys; and (c) results of owning structural mechanisms,
including L6 structural verification and L7 workflow replay, consumed
through their owning APIs and never reimplemented by L10. **Direction lock:
L6 records → L10 derives. Never L10 records → L6.** The second-history
problem is thereby structurally impossible, not merely prohibited.

The observability mechanism has no L6 write capability. L10 views cannot
create, mutate, delete, or amend durable history.

L10 produces deterministic, versioned, reproducible views. A view is a pure
function over its declared record slice and registered interpretation
artifacts and carries provenance identifying the consumed event/object range
and view-function version. Views are recomputable and discardable. A
discrepancy between a view and its source record is a view defect; the view
never supersedes the record.

L10 does not collect execution information through agents, hooks,
interceptors, or parallel observation paths. If an execution dimension is
absent from the authoritative record, L10 reports the absence rather than
reconstructing it from a side channel. Closing such a recording gap belongs
to the layer that owns the underlying fact.

Observability output never controls workflow, authorization, or verification
outcomes. Any computation whose result is intended to act as a verification
gate is a Verification Contract and must enter through the D-L10-2 contract
mechanism rather than the observability mechanism.

**Precision (owner amendment):** verification evaluation records are NOT an
exception to L10's no-write rule. The verification mechanism produces
verification facts; those facts become durable through the existing L6 write
path; L10 observability then derives views over them. L10 does not write
verification evaluation records either — L6 owns their durable recording.
(The exact boundary sits in Q-L10-8.)

v1 scope: observability consists only of record-derived views. Live
operational telemetry is deferred as a recorded residual (Q-L10-15 becomes a
residual, not a v1 design question) and, if introduced later, must be
grilled separately (evidentiary status, retention, privacy/secrets, clock
semantics, correlation, availability, second-history risk) and explicitly
segregated from the authoritative/evidentiary record plane.

### D-L10-6 — Verification execution and record boundary (LOCKED 2026-09-11, Q-L10-8; trigger fork LOCKED: model-proposed only in v1)

Verification consists of five owned stages:

1. **Contract resolution — L10.** L10 resolves the exact registered C@v from
   the append-only Contract Registry in one atomic snapshot, verifies its
   active status, and obtains its immutable verifier binding, configuration,
   evidence specification, and result mapping. The evaluation instance
   resolves the contract's evidence requirements to concrete governed
   evidence object references.
2. **Authorization — L4.** The bound verifier capability is authorized
   through the existing grant/ceiling chain. Verification receives no
   special authority. There is no verification-specific authorization path.
3. **Execution — L5.** The registered verifier capability executes under the
   governed execution environment and pinned configuration. Raw output and
   canonical verifier result are captured as evidence with complete
   execution provenance.
4. **Evaluation — L10.** L10 performs only bounded, deterministic evaluation
   of the returned data: mechanical-validity checks and the contract's
   closed declarative result mapping. Mechanical failure produces the
   applicable machinery-reserved outcome; successful mapping produces PASS,
   FAIL, or INCONCLUSIVE. The evaluator cannot invoke capabilities, re-run
   verification, retry execution, or otherwise initiate execution.
5. **Durable recording — L7 → L6.** The verification evaluation fact is
   incorporated into the existing governed execution record through the
   established L7/L6 recording discipline. The durable record includes the
   contract identity, verifier capability identity/hash, configuration
   identity, input object references, canonical-result reference, outcome,
   and required provenance. The verification result is committed to durable
   history before its typed event is made available to workflow control.
   L7 records through existing machinery but does not own verification
   semantics; L6 remains the durable-history owner.

**Trigger: model-proposed only in v1.** A Skill or workflow may require a
verification outcome for a transition, but the requirement does not itself
cause execution. Verification is proposed as an ordinary capability call and
passes through the existing L7 → L4 → L5 path. If the model does not propose
a required verification, the gate cannot be satisfied and the governed
workflow eventually follows its reviewed exhaustion/failure path. No new L7
execution-initiation semantics are introduced. Standing invariant: *failure
to request required verification can prevent completion, but can never
produce successful verification or bypass its gate* — verification
availability is not verification authorization.

**Ownership:** L10 owns mechanical verification validity and contract
evaluation; L4 owns authorization; L5 owns execution environment
enforcement; L7 owns workflow control and recording sequencing; L6 owns
durable history. The verifier itself has no authority to declare an L10
outcome.

**Hostile-verifier rule (anti-laundering, carried from D-L10-4):** verifier
output — including strings like "PASS", "FAIL", "NOT_AFFECTED", "SECURE",
garbage, malformed, unexpected, or contradictory data — is untrusted
verifier-domain data until L10 validates result-domain membership and
applies the registered mapping. Verifier output "PASS" is NOT automatically
PASS; only the L10 evaluator creates PASS. The evaluator must remain safe
against a completely hostile verifier (zero trust discount, L10 form).

### D-L10-7 — Model ↔ L10 trust boundary (LOCKED 2026-09-11, Q-L10-9)

The model interacts with verification through exactly two apertures:
proposing verifier calls and reading recorded verification outcomes.
Everything else is unreachable.

The model may propose a verifier call. The proposal is advisory data and is
authorized through the existing L4 path.

The model may propose concrete evidence object references. L10 mechanically
validates those references against the Verification Contract's closed input
specification, including declared evidence kind, authority class, and
task-binding constraints. Within those constraints, evidence selection
remains model discretion. The selected and evaluated evidence is durably
recorded, making selection observable to Governance — the model can
cherry-pick within the permitted set, but never silently. If completeness is
required, that requirement must be declared by the registered Verification
Contract; L10 does not invent completeness requirements. (L10 constrains
what evidence MAY be evaluated; it does not decide which permitted evidence
the model must choose.)

The model may propose a registered Verification Contract, but only an
evaluation of the exact contract identity required by the reviewed workflow
can satisfy that workflow's verification gate. Invoking another registered
contract cannot substitute for the required contract.

The model may read recorded L10 outcomes through L2 as fenced, classified
data and may interpret them in prose. Such interpretation remains advisory
and cannot modify, reclassify, upgrade, or annotate the recorded
verification fact.

Model claims that verification succeeded are inert prose. L7 control
semantics consume only recorded typed verification events.

Model output cannot substitute for an L10 result. L10 alone produces
verification outcomes, and verification-typed L7 events originate only from
committed evaluation records.

The model may request another evaluation after a prior result. Each
evaluation is a new instance with its own complete record; no evaluation
overwrites, supersedes, or erases another. Repeated evaluation cannot be
used to select a preferred result. Any divergence under identical registered
inputs is surfaced as evidence of a violated determinism premise and does
not become a new trusted outcome. (Gate-selection semantics over multiple
evaluations are deliberately NOT decided here — carved out to Q-L10-10a;
"most recent" must not silently become an assumption.)

The model has no authority to waive, skip, defer, or substitute a
verification required by the reviewed workflow. Failure to propose a
required verification can only prevent progression and eventually follow
the workflow's reviewed failure/exhaustion path.

Model-authored contract-shaped artifacts remain untrusted data. They cannot
become evaluable without the external Governance registration act.

### D-L10-8 — Failure semantics (LOCKED 2026-09-11, Q-L10-10)

Verification failure semantics are stage-indexed across the D-L10-6
execution sequence. The boundary between refusal and verification outcome is
successful atomic creation of the evaluation instance.

**Before an evaluation instance exists: refusals are not verification
outcomes.** A proposal naming an unresolvable contract — including an
unregistered or withdrawn contract, unreadable registry, or contract
pin/hash mismatch at resolution — produces a typed refusal through the
existing refusal taxonomy and creates no evaluation record or verification
outcome. An L4 authorization refusal remains an L4 authorization fact
through the existing machinery and does not become an L10 outcome.

**After an evaluation instance exists: outcomes use the closed five-value
vocabulary.**

- **INVALID:** the evaluation's governed inputs or execution result violate
  a registered mechanical contract — evidence failing the contract's input
  specification (wrong kind, wrong authority class, task-binding violation,
  structurally invalid reference); mechanically unacceptable verifier
  results; result outside the registered result domain;
  canonicalization-integrity failure; incomplete required provenance;
  configuration/pin mismatch discovered during evaluation; detected
  nondeterminism. Evidence failing admissibility is recorded as INVALID and
  execution is skipped. **Owner amendment:** a structurally valid evidence
  reference whose required object cannot be resolved or read is UNAVAILABLE
  rather than INVALID — INVALID is a proposition about validity;
  UNAVAILABLE means the required thing could not be obtained.
- **UNAVAILABLE:** the evaluation cannot obtain a canonical verifier
  result — inability to provision the verifier environment, verifier crash,
  timeout, forced termination, or unresolvable/read-unavailable governed
  evidence. An execution that produces an in-domain result does not become
  UNAVAILABLE merely because its process exit status is nonzero.
- **PASS / FAIL / INCONCLUSIVE:** when execution produces an in-domain
  canonical result, L10 applies the registered contract mapping.

**The registered result domain is authoritative** for distinguishing an
established result from execution failure: exit 1 + valid test report
(in-domain) may map to FAIL; segmentation fault → UNAVAILABLE; malformed
report → INVALID. Never generic "nonzero = failure." Consequence carried to
the registration grill: the verifier's result domain must be explicit,
bounded, and honestly defined — a registration-review obligation.

**Evaluator failure:** a machinery defect in the L10 evaluator mints no
verification outcome. Evaluation aborts through the existing harness
invariant-failure handling; the gate remains unsatisfied. INVALID is a
result produced by a functioning evaluator about the evaluation; it cannot
classify failure of the evaluator itself. **Visible consequence: an
evaluation instance does not always produce one of five outcomes — the
sixth terminal is evaluator invariant failure with NO outcome.** This
distinction must remain visible in the implementation.

**Durable-record failure:** if the required L6 commit fails,
record-before-effect applies — no committed evaluation, no verification
event to workflow control.

**Reason codes:** INVALID and UNAVAILABLE carry a closed typed reason
vocabulary appropriate to their class. Reasons are diagnostic durable
record content and do not participate in L7 workflow control — otherwise
the diagnostic taxonomy becomes a hidden workflow language. Only the five
typed verification outcomes can produce verification workflow events;
timeout-specific routing, if ever genuinely needed, is a deliberate
vocabulary/workflow amendment. Refusals use the existing refusal taxonomy;
evaluator failures use the existing invariant-failure taxonomy; neither is
represented as a verification outcome.

### D-L10-9 — Evaluation multiplicity and gate consumption (LOCKED 2026-09-11, Q-L10-10a)

Multiple evaluation instances of the same C@v may exist within a task. Every
evaluation instance is durable; no evaluation overwrites, supersedes, or
erases another.

An evaluation instance is identified by its committed L6 evaluation record
and its position in the task's authoritative event sequence. No independent
L10 evaluation identity scheme is introduced.

Evaluation recency is determined exclusively by L6 commit sequence within
the task. Wall-clock timestamps have no ordering authority.

A verification gate is identified by exactly (contract identity, required
outcome) in v1. At transition evaluation, the gate is satisfied only if the
latest committed evaluation instance of that contract identity within the
task has the required outcome. An older successful evaluation cannot satisfy
a gate after a newer evaluation of the same contract has produced a
different outcome: PASS→FAIL, PASS→INCONCLUSIVE, PASS→UNAVAILABLE,
PASS→INVALID all render the gate unsatisfied; FAIL→PASS satisfies it. The
downgrade asymmetry is deliberate and fail-closed — the model can only lose
by gratuitous re-runs, never gain.

Gate satisfaction is derived deterministically from the applicable record
prefix at each transition evaluation and is not cached as independent
authority. A transition already taken on the basis of then-current facts
remains immutable historical execution; later evaluation results do not
retroactively alter that transition. (Current gate satisfaction ≠ historical
transition validity — otherwise a later verification could rewrite a
workflow that legitimately progressed earlier.)

L7 maintains the latest-per-contract state as walk state and its replay
mechanism independently derives the same state from the event prefix (the
CallState pattern). No L10 query channel is introduced and no
workflow-defined evaluation-selection policy exists in v1.

Evidence scope is not part of gate identity. The gate does not aggregate,
compare, or select evaluation evidence. Evidence admissibility, task
binding, completeness, and any required freshness/current-state property
belong to the registered Verification Contract and its reviewed
verification semantics. **Owner precision: gate machinery does not
establish freshness, and a contract establishes it only where freshness is
explicitly part of its reviewed input/verification semantics — "object
belongs to this task" does not imply "object represents the current
post-remediation state." L10 evaluates what the contract says; it does not
invent freshness semantics.** Verify-last is a review obligation, not a
runtime guarantee: a lattice that verifies before remediating can hold a
perfectly valid PASS that says nothing about the post-remediation state,
and L10 will not silently infer staleness.

Multiple evaluations using different evidence are legitimate and remain
independently durable; only the latest for the gate's contract identity
participates in gate satisfaction. A workflow requiring multiple
independently established propositions must express them through distinct
reviewed contracts or a contract whose registered input semantics establish
the required aggregate; v1 introduces no evaluation-set algebra.

**Registration/Skill-review obligations (accumulating list):** (1) verifier
result domain honestly defined (D-L10-8); (2) contract completeness
requirements explicit (D-L10-7); (3) freshness/current-state requirements
explicit where needed; (4) workflow sequencing places verification
appropriately — typically after the state-changing phase; (5)
latest-per-contract semantics understood when reviewing retry/remediation
workflows.

## 3. Grill — question list (OPEN; owner's sequence 2026-09-11, implementer's 12 merged in)

Owner's proposed starting boundary (working text, pending Q-L10-1 lock):
*L10 is the deterministic evidence-verification and execution-observability
layer. It establishes whether governed execution produced evidence satisfying
a registered verification contract and whether the execution trace is
structurally consistent. It does not decide security meaning, Enterprise
Position, remediation acceptance, or business truth.* Standing prohibition:
**L10 must not become a hidden second Security Governance engine** — it never
independently decides "CVE fixed", "false positive", "not affected",
"remediation acceptable", "release secure". It may establish deterministic
facts Governance subsequently uses.

| Q | Question |
|---|---|
| Q-L10-1 | **CLOSED → D-L10-1.** Boundary and authority: two mechanisms under one layer; trace-consistency stays single-homed with L6/L7; three-proposition model foundational. |
| Q-L10-2 | **Closure PROPOSED → D-L10-1a (ladder, X vs X′ narrowing); awaiting explicit owner lock.** |
| Q-L10-3 | **CLOSED → D-L10-2.** Contracts: L10-owned registry (fork (a)), governance-only registration, closed schema, exact pins, anti-smuggling invariant. |
| Q-L10-4 | **CLOSED → D-L10-3.** Determinism = registration property; canonical-result standard w/ registered canonicalization; model categorically not a verifier in v1. |
| Q-L10-5 | **CLOSED → D-L10-4.** Five-value closed vocabulary, ESTABLISHED/NOT-ESTABLISHED partition, machinery-reserved statuses, distinct typed L7 events, fail-closed gates without collapse. |
| Q-L10-6 | **CLOSED → D-L10-5.** Observability = read-only deterministic derivation; views are pure functions w/ provenance; absent dimensions reported, never side-channel collected. |
| Q-L10-7 | **CLOSED → D-L10-5.** Direction locked: L6 records → L10 derives, never the reverse; second history structurally impossible. |
| Q-L10-8 | **CLOSED → D-L10-6.** Five owned stages; L10 evaluation = bounded declarative mapping + validity checks; verifier never mints outcomes; record-before-event; trigger = model-proposed only in v1. |
| Q-L10-9 | **CLOSED → D-L10-7.** Two apertures (propose, read); bounded recorded evidence discretion; contract-identity-bound gates; claims inert; substitution unrepresentable. |
| Q-L10-10a | **CLOSED → D-L10-9.** Latest-per-contract gate consumption; identity = committed record + event position; stateless-at-δ satisfaction; evidence scope out of gate identity; freshness is contract semantics. |
| Q-L10-10 | **CLOSED → D-L10-8.** Stage-indexed failure taxonomy; instance-creation boundary; result-domain authoritative; evaluator failure mints no outcome; reasons never reach δ. |
| Q-L10-11 | Evidence provenance: which fields are NECESSARY (verification_id, verifier identity/version/hash, input object hashes, config hash, environment identity, timestamp/sequence, raw output ref, derived result, result hash) — establish, don't assume. |
| Q-L10-12 | Tamper resistance (adversarial register): modify verifier / config / input evidence / raw output / derived result / execution record — can an apparently valid verification survive? |
| Q-L10-13 | Replay and reproducibility: can L10 reproduce a verification from durable state; if replay ≠ original, what does that MEAN? Expectation: no automatic semantic rewrite of history — L6 records what happened, Governance decides what the discrepancy means. |
| Q-L10-14 | Verification gates in L7: what exactly crosses from L10 back into L7 — likely a tightly bounded typed outcome, never arbitrary verifier output. |
| Q-L10-15 | **RESIDUAL per D-L10-5** (owner, 2026-09-11): live operational telemetry deferred out of v1; future dedicated grill (evidentiary status, retention, privacy/secrets, clocks, correlation, availability, second-history risk). |
| Q-L10-16 | Security-sensitive observability: can observability leak secrets, credentials, sensitive context, protected evidence, prompts, tool arguments? Connect to L5 secret-contamination + L6 durability rules. |
| Q-L10-17 | Completion semantics: workflow COMPLETED ≠ verification PASSED ≠ security condition established ≠ Enterprise Position accepted — formally separated propositions. |
| Q-L10-18 | Proof gate: structural, adversarial, provenance, replay/reconstruction registers + live operational proof + independent architecture/security/test reviews; close as L7/L9 were closed. |
| Q-L10-19 | (merged from implementer) OPEN-2 proper: the deterministic policy artifact gating run_command / run_build / run_tests / run_scan as verification capabilities — allowlisted or manifest-pinned invocations, where the artifact lives, who reviews it. |
| Q-L10-20 | (merged) Extend-the-strongest-subsystem: precise relationship to benchmarks validators/gate/guardrails — generalize, wrap, or leave in place; no second validator family (and no second evaluation subsystem — that line belongs to L11). |
| Q-L10-21 | (merged) Expected-outputs contract (L9 Gate-0 residual, owner unassigned) and verification-evidence egress to Themis: does L10 own either; what stops a verification summary from becoming an Enterprise Position? |

## 4. Assets inventory (for the grill, factual)

- `benchmarks/internal/validator` — keyword/regex/json deterministic
  validators; violations score-affecting since F3.
- `benchmarks/internal/gate` — regression gate + verdict artifacts (admission
  enforced at routing since 753011c); L11 raw material but the verdict/digest
  pattern is L10-relevant precedent.
- `internal/service` guardrails — SuspectInjection, CheckStance.
- L6 — trace sink, append-only events, manifest projection, StatusView,
  content-addressed object store.
- L7 — audit events, typed gate outcomes as workflow transition conditions,
  Register D replayer.
- Scaffolds: `verification/{build,evidence,lint,security,test}`,
  `observability/{events,logging,metrics,tracing}` — empty (.gitkeep only).
- L4 — registry with result-trust-fixed-at-registration; execution ceilings.
- L5 — isolation contract, artifact egress, spec loading.
