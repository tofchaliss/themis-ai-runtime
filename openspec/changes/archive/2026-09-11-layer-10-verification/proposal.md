# Proposal: Layer 10 — Verification & Observability

Status: GRILL OPEN 2026-09-11. Implementation does not start before Gate 0.

## Why now

L9 shipped and archived (2026-09-11); the locked sequence continues L10 → L11.
Three forces converge on L10:

1. **OPEN-2 is blocked on it.** run_command (and the run_build/run_tests/
   run_scan capability class) has been deferred since L4 because no
   deterministic verification consumer exists. D-L9-14 fixed the ownership
   chain — Skill requires → L7 enforces gates → L4 authorizes → L5 executes →
   L2 classifies → **L10 evaluates verification semantics** → governance
   decides — and recorded the v1 honesty: no qualifying deterministic
   verification capability exists yet.
2. **The strongest subsystem is waiting to be extended.** The benchmark
   validators (keyword/regex/json), guardrails, and the F1/F2/F3 remediations
   are exactly the deterministic verification discipline L10 generalizes
   ("extend the strongest subsystem" — locked plan, Phase C).
3. **Observability substrate already exists.** L6's trace sink, event stream,
   and manifest projection plus L7's audit events already record most of the
   architecture's observability dimensions; L10 must define what is missing
   (metrics, timelines, verification results as typed records) without minting
   a second record plane.

## What L10 is (architecture baseline)

Verification answers *"did the agent actually succeed?"* — build, tests, lint,
git diff, security scan, static analysis, dependency scan. The model cannot
self-certify success. Observability answers *"what exactly happened?"* — the
recorded execution dimensions (00-p0-architecture-v2 §13).

## What this change delivers (subject to the grill)

- The deterministic verification vocabulary: typed gate outcomes, evaluators,
  and their authority classification (the D-L9-9/D-L9-14 deferred question).
- The verification-capability seam (run_build/run_tests/run_scan class)
  unblocking OPEN-2 — capabilities registered in L4, executed in L5, evaluated
  by L10.
- The observability surface over the existing L6/L7 record plane — no second
  record plane, no new security authority.
- Scaffold dirs `src/harness/verification/` and `src/harness/observability/`
  gain their first real code, or the scaffolds are corrected if the grill
  lands ownership elsewhere.

## Constraints inherited

Model output advisory; deterministic controls enforce; verification results
gain authority only through registered computation + deterministic execution +
governed environment + complete computational provenance (D-L9-14); L6 owns
durable truth; Themis owns security meaning; fail closed.
