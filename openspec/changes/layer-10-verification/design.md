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
| Q-L10-2 | What counts as verification? The ladder: model says X ≠ evidence shows X ≠ deterministic verifier establishes X ≠ Governance accepts X as security truth. |
| Q-L10-3 | Verification contracts: who defines one; contents (inputs, expected evidence, algorithm, result vocabulary, failure semantics, provenance requirements, version, hash); can a Skill define verification semantics or only require execution of a registered verifier (L9 precedent says the latter)? |
| Q-L10-4 | Deterministic vs probabilistic verification: explicit boundary between deterministic verification, model assessment, model-relayed claims, human verification, Governance determination — incl. the derived-evidence eligibility rule (D-L9-14). |
| Q-L10-5 | Verification authority: what does a result (PASS/FAIL/INCONCLUSIVE/UNAVAILABLE/INVALID) actually mean — fact, claim, evidence classification, workflow gate signal — and who may interpret it? |
| Q-L10-6 | Observability truth: what does L10 observe (transitions, tool calls, args, authz decisions, environment, files changed, artifacts, verification executions, model turns, errors, timeouts, termination) without creating a competing event/history system with L6? |
| Q-L10-7 | L6 relationship: does L10 produce observations → L6 records them, or does L6 record → L10 derives views, or a combination? Never two authoritative histories. |
| Q-L10-8 | Verification execution: L10 defines semantics · L4 authorizes · L5 executes · L6 records — precisely what does L10 itself do vs L4/L5? |
| Q-L10-9 | Model involvement: may the model request verification, select a verifier, interpret the result, declare success, override failure, manufacture evidence? Expected shape: model proposes → L4 authorizes → L5 executes → L10 verifies → typed result; the model cannot manufacture the result. |
| Q-L10-10 | Failure semantics: verified-false vs not-verified vs unavailable vs invalid vs failed vs verifier-error vs evidence-insufficient — distinctions that materially affect L7 behavior. |
| Q-L10-11 | Evidence provenance: which fields are NECESSARY (verification_id, verifier identity/version/hash, input object hashes, config hash, environment identity, timestamp/sequence, raw output ref, derived result, result hash) — establish, don't assume. |
| Q-L10-12 | Tamper resistance (adversarial register): modify verifier / config / input evidence / raw output / derived result / execution record — can an apparently valid verification survive? |
| Q-L10-13 | Replay and reproducibility: can L10 reproduce a verification from durable state; if replay ≠ original, what does that MEAN? Expectation: no automatic semantic rewrite of history — L6 records what happened, Governance decides what the discrepancy means. |
| Q-L10-14 | Verification gates in L7: what exactly crosses from L10 back into L7 — likely a tightly bounded typed outcome, never arbitrary verifier output. |
| Q-L10-15 | Observability vs audit: operational telemetry ≠ execution trace ≠ security audit evidence ≠ durable historical record — else L10 absorbs L6. |
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
