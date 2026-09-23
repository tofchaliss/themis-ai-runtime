# Proposal: Layer 8 — Subagents

Status: **ARCHITECTURE LOCKED 2026-09-22 — D-L8-1..21 LOCKED, C-L8-1..21
complete, Gate 0 PASS (owner closure: `design.md` §7). Implementation
NOT STARTED; production NOT authorized.**
The decisions live in `design.md`; the milestone plan in `tasks.md`.
Outcome in one line: L8 v1 is a deterministic delegation mechanism for
isolated, tool-less, single-call model inference, requested through the
existing L4 gate, composed by L7 from a Governance-registered delegation
template, recorded as `l8-delegation` in the parent's L6 stream, and
returned to the parent as untrusted data. The scaffold collapses to one
package (`subagents/delegation/`); `roles/`, `runtime/`, `isolation/`
are deleted at implementation. The text below is the pre-grill agenda,
retained as history.

L8 is the only unimplemented layer of the eleven. L1–L7, L9, L10, L11
are closed and archived; G1 and G2 are closed.

## Why a grill before any code

The word "subagent" is the risk. It carries, by ordinary usage,
implications this architecture has spent eleven layers refusing. A
casual implementation would introduce one or more of:

- a second orchestration authority
- another runtime
- another tool-selection mechanism
- independent durable state
- independent instructions
- role-based authority
- uncontrolled delegation
- an approval mechanism that bypasses L7

Each of those is an architecture decision, and none of them would
announce itself as one. They would arrive as code that looked
reasonable.

The exposure is structural, not incidental. L8 sits **between the
model's reasoning and the governed execution machinery** — precisely the
seam every other layer exists to keep narrow. A second orchestrator
built there does not sit beside L7; it sits in front of it.

## The governing constraint (owner, top of proposal)

> **A subagent is a delegated reasoning execution, not a delegated
> authority.**

Everything below follows from that sentence, and the grill's job is to
either confirm it or replace it explicitly — never to erode it by
implementation.

## Prohibitions in force unless the grill explicitly decides otherwise

These are the default. Any of them may be lifted only by a locked
decision that says so in words, with its reasoning recorded:

1. L8 cannot own workflow semantics.
2. L8 cannot authorize tools.
3. L8 cannot execute outside L7.
4. L8 cannot create security truth.
5. L8 cannot own durable truth.
6. L8 cannot modify Governance state.
7. L8 cannot independently invoke another subagent.
8. L8 cannot establish verification.
9. L8 cannot create a second Skill/Procedure runtime.
10. L8 cannot turn model output into authority.
11. L8 cannot bypass L1–L7.

## The chain the grill must resolve

Delegation goes down and results come back up. Execution never leaves
the governed path:

```
L7 Orchestration
       │  delegates a reasoning task
       ▼
L8 Subagent Delegation
       │  invokes
       ▼
   Model / Submodel
       │  returns advisory output
       ▼
L8 result
       │  returned, not acted on
       ▼
L7 Orchestration
       │  decides; only L7 proceeds
       ▼
L4 → L5 → L6
```

The shape to preserve: **L8 returns to L7; L8 never reaches L4.** A
design in which a subagent's output travels to tool authorization
without passing back through L7's decision is a design in which L8 has
become another L7.

## Scope boundary — what this change does NOT absorb

The approval channel and `run_command` / OPEN-2 are **separate
decisions**. They are currently recorded together with L8 ("the approval
channel + subagents + run_command (L7 residuals / L8 / OPEN-2)") because
all three were deferred, not because they are one problem.

They may **constrain** the L8 design — an approval channel, if one
exists, would bound what delegation may request — but they must not be
silently absorbed into it. Absorbing them would let L8 acquire an
approval mechanism as a side effect of being implemented, which is
prohibition 11 arriving through the back door.

If the grill finds a genuine dependency, it records the dependency and
the separate decision it waits on. It does not decide the other
question.

## Grill questions (to be owner-set and owner-led)

Proposed starting set, in the order the L9–L11 pattern uses: problem →
boundary → ownership → authority → trust model → invariants → decision
matrix. The owner sets the final list; these are a draft for that
purpose, not a locked agenda.

### Problem and boundary

| Q | Question |
|---|---|
| Q-L8-1 | What problem does delegation solve that L7 phases and L9 skills do not already solve? If none, L8 is not needed and that is a valid outcome. |
| Q-L8-2 | What exactly is delegated — a prompt, a sub-task, a phase, a procedure? What is the unit? |
| Q-L8-3 | What is the boundary between "a second model call inside one L7 phase" and "a subagent"? Is there a difference that matters architecturally? |
| Q-L8-4 | Does a subagent exist as a durable identity, or only as an act within a task? |

### Ownership

| Q | Question |
|---|---|
| Q-L8-5 | Who owns a subagent's definition — Governance, the skill author, the submitter, or L7 at assembly? |
| Q-L8-6 | Are `roles/` a governed registry, a skill-side artifact, or an implementation detail with no authority? The directory name presumes an answer; the grill must supply one. |
| Q-L8-7 | Does a subagent appear in the deployment anchor (G1)? If it can change what an anchored deployment does, owner-finding-4 says it must be pinned. |
| Q-L8-8 | Who owns the record of a delegation — L6 as ordinary events within the parent task, or something new? (Default: L6, no new plane.) |

### Authority and trust

| Q | Question |
|---|---|
| Q-L8-9 | What authority, if any, does a subagent hold that its parent task does not? (Default: none.) |
| Q-L8-10 | Can a subagent's tool access be narrower than the parent grant? Can it ever be wider? (Default: narrower only, never wider — the L9 narrowing discipline.) |
| Q-L8-11 | Is subagent output data or instruction, on return to L7? (Constitution says data. Confirm it survives contact with delegation.) |
| Q-L8-12 | What prevents delegation depth from being unbounded — and is boundedness static like Q-L7-4, or dynamic? |
| Q-L8-13 | Prohibition 7 forbids a subagent invoking another. Is that the right default, or does the real use case require nesting? If nesting, what bounds it and who decides? |
| Q-L8-14 | What does a subagent receive as instructions — the parent's resolved EIS, a subset, or something composed? Who composes it, and does composition create a second L1? |

### Invariants and failure

| Q | Question |
|---|---|
| Q-L8-15 | What is the governed terminal of a delegation that fails, stalls, or returns nothing? |
| Q-L8-16 | Does delegation consume the parent's turn/call budget, or hold its own? A separate budget is a separate ceiling and therefore a separate authority. |
| Q-L8-17 | What must the record establish about a delegated execution for it to be reconstructible — and is that the same standard L10 applies to the parent? |
| Q-L8-18 | How does verification (L10) treat work done inside a delegation? Prohibition 8 says L8 cannot establish verification; what happens when a subagent's output would be verifier-eligible? |
| Q-L8-19 | What is the G2 answer — what event witnesses that a delegation occurred, and what kind of fact is it? |

### Termination of the recursion

| Q | Question |
|---|---|
| Q-L8-20 | L11 ended with "recursion terminates in Governance — no L12." What is L8's equivalent statement, and what prevents delegation from becoming an open-ended agent hierarchy? |

## Constraints inherited (not grillable)

Model output advisory, never authority. Deterministic controls enforce
authorization, policy, state, verification. External content is data,
not instructions. Fail closed; nothing defaulted. Author ≠ governance ≠
machinery. Themis owns security truth; the Harness must not become a
second security-governance system. L6 owns the record. Registration
gates precede execution. Day-0 prohibitions. The G1 anchor governs what
a deployment may execute; G2 governs when a fact is established.

## Gate 0 criteria (proposed)

Gate 0 is the owner's judgment that the architecture is settled enough
that implementation is conformance work rather than design work. For
L8 specifically, PASS should require all of:

- [ ] Every Q-L8-* answered and locked as a D-L8-* decision, or
      explicitly recorded as a residual with its own gate.
- [ ] The governing constraint either confirmed verbatim or replaced by
      a locked decision that states what replaced it and why.
- [ ] Each of the eleven prohibitions either still in force or lifted by
      a named decision. No prohibition may lapse by silence.
- [ ] An artifact inventory: the complete list of components L8 may
      introduce. As in L11, this is the implementation whitelist — a
      component whose architectural role is absent from it stops
      implementation until classified.
- [ ] A stated answer to "what makes this not a second L7?" that a
      reviewer can check against the implementation, not a claim that
      it isn't.
- [ ] The relationship to the approval channel and `run_command` /
      OPEN-2 recorded as dependency-or-independence, with neither
      decided here.
- [ ] Failure behaviour specified for every delegation outcome, each
      landing on a governed terminal.
- [ ] The execution constraint carried into every milestone: if a
      component appears necessary that is absent from the inventory,
      implementation STOPS and the item is classified (implementation
      detail / residual / genuine gap) before proceeding.

## What this proposal does not do

It does not answer any grill question. The questions above are a draft
agenda for an owner-led grill, in the pattern that closed L9, L10, and
L11 — the owner sets and closes the decisions; this document exists so
the session starts from a stated problem and an explicit prohibition
list rather than from a directory name.

It also does not schedule the work. L8 is the next major **architecture**
activity; deployment validation is the next major **empirical** activity.
They are separate tracks and neither blocks the other.
