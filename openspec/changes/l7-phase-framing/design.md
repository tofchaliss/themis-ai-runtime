# Design record — L7 phase framing (PROPOSED until each Q-P is locked)

Facts from code (2026-09-27): the system message is the L1 EIS render,
resolved ONCE per task (`resolveTaskEIS`) from harness-safety,
harness-system, themis-domain and the skill procedure; sections are
renderer furniture and "any delivered text carrying a rule lives in an
instruction file" (D-L1-6, Q-L1-8). The user message is L2 `Compose`
over `Gather(contract, plan)` with Plan ⊆ Contract; the loop's plan is
fixed: one inline slot `task-payload` (kind `task-brief`, authority
`external-untrusted`). L2 authority classes: `governed-record`,
`governed-external`, `derived`, `external-untrusted`. Per-phase
composition records `phase`, `eis_hash`, `contract_hash`,
`render_hash`, `l2_payload_hash` in `l2-delivery` (D-L7-11).

## D-P-1 — Where the phase fact lives (LOCKED 2026-09-27, owner: option 1)

> The current phase is never an L1 instruction and never task-supplied
> data. It is a derived L2 context fact computed from the recorded
> workflow state and delivered only when the skill's context contract
> explicitly declares the `phase-state` slot.

Split rule from fact. L1: one static harness-system instruction
("work proceeds in phases; each phase starts a fresh conversation; the
Context names the current phase, the completed phases and this phase's
capabilities; a tool not listed there is unavailable"). L2: a second
fixed loop slot `phase-state` (kind `phase-state`, authority `derived`,
author `harness`, sensitivity public), composed by the loop at every
phase entry from the recorded `workflow-transition` chain; Plan ⊆
Contract, so a contract that does not declare the slot refuses at
Gather — the runtime never injects it silently. The EIS stays stable
across phases; the Context changes because workflow state changed
(D-L7-11 preserved: fresh composition, now complete).

Rejected: per-phase EIS re-resolution under `task.*` (couples L1
identity to L7 state, puts a harness fact on the untrusted shelf);
renderer heading (furniture carrying semantics, Q-L1-8).

Guard carried into Q-P-2: do not solve phase framing by adding phase
semantics to L1; design how the derived fact is computed, bounded and
represented.

## D-P-2 — What the phase fact says (LOCKED 2026-09-27, owner: option 1)

> `phase-state` describes workflow position and current capability
> only. It contains no execution counters, verification results,
> prior-phase outputs, tool history, or semantic summaries. Every field
> is deterministically derived from the governed workflow and its
> recorded transition state.

Closed vocabulary: `{workflow: name@version, phase: <name>, completed:
[<names in order>], capabilities: [<tool names of the NARROWED phase
grant, Phase.capabilities ∩ task grant>]}`. Provenance: the `cause_seq`
of the `workflow-transition` that entered the phase; the initial phase
cites the RUNNING lifecycle seq. What the model reads as capabilities
equals what L4 will authorize; L2 does not become an authorizer.

Rejected: counters and verification outcome (L7/L10 facts under other
authorities; "turns remaining" would read as instruction); prior-phase
summaries and tool history (evidence, Q-P-3's subject).

## D-P-3 — Evidence continuity across phases (LOCKED 2026-09-27, owner: option 1)

> Fresh conversation does not imply loss of durable evidence. It means
> evidence is not automatically reintroduced into the next context. A
> phase must explicitly declare continuity when the workflow requires it.

Current: no cross-phase evidence carry. Current continuity: the task
inputs (brief) plus the workspace and context the phase can itself
read. The I-M5 failure was orientation, not missing evidence: both
models held the Finding's facts in the brief and never reached a step
that needed more.

Future design (recorded, NOT current architecture): skill-selected,
contract-declared `prior-evidence` — the workflow phase declares
`carry: [{from, tool, …}]`; the loop resolves the matching audited
objects through the existing `record-object` source kind (C-L8-7),
authority class derived from the witnessing event (C-L8-18),
provenance by seq; Plan ⊆ Contract; fail closed. Trigger to reopen: a
real workflow demonstrates a later phase needs evidence it cannot
re-derive. Not a trigger: model orientation or demo convenience.

Rejected: automatic carry of every governed-record read in completed
phases — the harness would be selecting context for the model, unbounded
by the skill.

## D-P-4 — remediate-dependency@5 and rsys@7 (LOCKED 2026-09-27, owner: option 1)

> The context contract and phase framing are part of the skill
> composition identity. A skill revision cannot silently acquire or omit
> a required harness fact under the same composition identity.

`@5` = `@4` with exactly: `contract.json` declaring `phase-state`
(kind `phase-state`, requirement `required`, classes `[derived]`); a
phase-scoped procedure (each section opens with the phase it belongs
to; REMEDIATE works from its declared inputs and never assumes a prior
phase's `get_finding` result is present); workflow, ceiling and grant
unchanged. Registration record `reg-skill-remediate-dependency-5`
cites this grill and the two host records (`demo-remediate-0001/0002`
under `rsys@6`). `@4` stays `active` in the catalog: recognized history,
not admitted by the next anchor.

`rsys@7` = `rsys@6` + new `instruction_root_system` (the L1 phase
rule), new `skill_catalog`, the `@5` workflow bundle (new context
contract hash), `skills: [@1, @2, @5]`. It is the first anchor whose
allowlist drops `remediate-dependency@4` because the composition no
longer satisfies the phase-framing contract — an anchor admission
decision, NOT a withdrawal. No L7 constitution change: `constitution.
orchestration` and `constitution.state` are untouched (the fixed slot
plan is code, not control vocabulary).

Rejected: `phase-state` optional (one composition identity under two
framings); mutating `@4` (immutable binding).

## D-P-5 — Model allowlist and the demo's model (LOCKED 2026-09-27, owner: option 1)

> `rsys@7` preserves the existing model allowlist. Model selection
> remains a deployment fact. Before `rsys@7` is minted, the intended
> model gpt-oss:20b must complete the `@5` governed workflow under the
> test anchor; qwen2.5:7b and cyberpal20b-v3 are captured as
> compatibility evidence without changing their deployment status.

The `TestLiveCapture` gate proves execution compatibility, not model
quality: COMPLETED means the model participated in the governed
workflow under that test — not that its remediation is correct, that
it is safe for every skill, that it is superior, or that it should stay
deployed. qwen2.5:7b's observed behavior (does not complete the
two-phase skill under the tested conditions) is a capability
observation, not a removal; no model admission/removal policy exists
and this grill creates none. No `models.json` now (a separate
architecture decision when model governance is a real requirement).

## D-P-6 — The `search_code` denial at ANALYZE (LOCKED 2026-09-27, owner: option 1; classification pending the host line)

Classify from the surviving raw call (`/tmp/cap-gptoss/turn-04-raw.json`)
and close. Expected: a model-authored `search_code` call missing `path`,
naming `pattern` instead of `query` (L4 `invalid-args`), or targeting
outside the workspace (L4 `target-refused`) — L4 enforcing the governed
tool contract on model-authored arguments; the walk continued and
declared done, so no boundary failed. Closure text once confirmed:
"Expected L4 refusal of a malformed model call; classified as model
argument error / L4 invalid-args (or target-refused). No architecture,
registry, procedure, or tool-contract change required."

Rejected: rewording `search_code` to suit one model's call (a registry
amendment on no evidence of a defective contract); ignoring the denial
(every denial in a record is classified).

## Grill state (2026-09-27)
Q-P-1..5 LOCKED; Q-P-6 LOCKED contingent on the host classification
line. No architecture is opened from the denial. Implementation is
P-M1..P-M3 in `tasks.md`.

