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
| Q-L10-8 | Verification execution: L10 defines semantics · L4 authorizes · L5 executes · L6 records — precisely what does L10 itself do vs L4/L5? |
| Q-L10-9 | Model involvement: may the model request verification, select a verifier, interpret the result, declare success, override failure, manufacture evidence? Expected shape: model proposes → L4 authorizes → L5 executes → L10 verifies → typed result; the model cannot manufacture the result. |
| Q-L10-10 | **Substantively pre-closed by D-L10-4** (taxonomy mapped: verified-false=FAIL, not-verified=INCONCLUSIVE, unavailable/verifier-error=UNAVAILABLE, invalid/nondeterminism=INVALID, evidence-insufficient splits by resolvability); formal confirmation at its turn. |
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
