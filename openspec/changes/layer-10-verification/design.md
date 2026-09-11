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

*(empty — populated as questions close)*

## 3. Grill — question list (OPEN)

| Q | Question |
|---|---|
| Q-L10-1 | What is a verification gate outcome? Closed vocabulary (PASS/FAIL/…, typed evidence)? Who may produce one, and what makes it *deterministic* rather than asserted? |
| Q-L10-2 | The verification-capability class (run_build/run_tests/run_scan, OPEN-2): what is its L4 registration shape — contracts, result trust fixed at registration, derived-eligibility as a registration property? |
| Q-L10-3 | Authority classification of verification outputs (the D-L9-9/D-L9-14 deferred decision): when does an individual result actually classify as derived, and who checks the preconditions? |
| Q-L10-4 | What does L10 itself *execute*, if anything? Is it a pure evaluator over L6 records + capability results, or does it own any runtime step? (Second-orchestrator prevention: reachability, not discipline.) |
| Q-L10-5 | Where does the gate-outcome → workflow-transition binding live? (L7 already enforces typed gate outcomes as transition conditions — what exactly does L10 add on top of the L7 event taxonomy?) |
| Q-L10-6 | Observability: which of the architecture's recorded dimensions are already satisfied by L6 events + L7 audit + task attribution, and what is genuinely missing (metrics, timelines, verification-result records)? No second record plane. |
| Q-L10-7 | Observability read surface: who consumes it (operator, Themis, governance), through what seam (StatusView extension? new read-only views over L6?), and with what authority (observation never mutates)? |
| Q-L10-8 | run_command policy (OPEN-2 proper): what deterministic policy gates arbitrary-command execution as a verification capability — allowlisted invocations? manifest-pinned commands? Where does the policy artifact live and who reviews it? |
| Q-L10-9 | Expected-outputs contract (Gate-0 residual from L9, owner deliberately unassigned): does L10 own it, and does v1 need it — or is it recorded again as a residual with a named owner condition? |
| Q-L10-10 | Verification evidence egress: how do verification results reach Themis (governed egress hand-off? typed evidence records?), and what stops a verification summary from becoming an Enterprise Position? |
| Q-L10-11 | Relationship to the benchmark validators and guardrails: generalize, wrap, or leave in place? ("Extend the strongest subsystem" — what does extension concretely mean without forking a second validator family?) |
| Q-L10-12 | Proof gate: what are the L10 proof registers, which slice proves it live (remediate-dependency was deferred until OPEN-2 — is it the L10 slice?), and what does the unmodified-production-loop rule mean here? |

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
