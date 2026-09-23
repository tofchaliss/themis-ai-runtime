# L7 Amendment: the L8 delegation seam (L8 M4, 2026-09-23)

Authorized by: L8 design §5.1/§5.2 (D-L8-3/4/8/15/16, C-L8-5..13,
C-L8-17..20; owner LOCK 2026-09-22). Additive; L7 gains no semantic
dependency on the delegation package (the verification-seam pattern).

## Boundary (orchestration/delegation.go)

- `Config.Delegator` — the one-way injected seam. Three entry points,
  each a pure function of its request and the record: `Registered`
  (assembly-time grant validation), `Instantiate` (stage B, invoked by
  the `delegate` executor through a per-task adapter), `Delegate`
  (post-hook). C-L8-12 Am. 1 / C-L8-13 split instantiation from
  execution; D-L8-21 §3's "one method" therefore reads as "one
  post-hook call site". No request or result type reaches
  `model.Message`, `ExecutionResponse`, or a system message
  (`TestDelegatorInterfaceCarriesNoConversation`).
- `InstantiationRequest`: task id, READ handle on the record, template
  ref, evidence refs, brief, the parent's activated L1 sources, and the
  registry-in-force hash + a trust function for class derivation.
  `DelegationRequest` adds the task write handle, `parent_call_seq`,
  call id, the executor's capture, the parent's governed model name
  and the SAME injected adapter, the model-registry pin, turn timeout,
  and wall deadline. `DelegationResult`: outcome, witness seq, output
  object id, output bytes (completed only).

## Assembly

- A phase exposing a delegation-class capability with no delegator
  wired → `ErrAssembly` (mirror of the verifier check).
- Every grant `template_scope` entry must resolve through
  `Delegator.Registered` (C-L8-15 G, grant validation) — including
  withdrawn entries; entries with no delegator wired refuse.
- The executor table is built after `CreateTask` so the `delegate`
  executor's compose half is bound to THIS record.
- `resolveTaskEIS` now also returns the parent's activated sources.

## Loop post-hook (loop.go, delegation.go)

For an AUTHORIZED delegation-class call: the executor's evidence (the
capture) is NOT appended to the conversation; after the `l4-audit`
commits and counters advance, `walk.delegate` builds the request,
calls the seam once, and appends exactly one tool-role message paired
by `ToolCallID`: completed → `tools.FrameToolResult("delegate",
external-untrusted, <output object hash>, bytes)`; every other outcome →
the closed typed error, unframed. The branch calls no `w.step`
(`TestDelegationBranchNeverSteps`); `controlVerbs` and
`verificationEvents` are byte-unchanged. A seam error is `ErrInvariant`
(stage D). `FaultAt`/`SetFault` expose the Register C injector for the
seam's three points.

## Seam implementation (subagents/delegation/seam)

L1 `Resolve` over the mandatory roots (unconditional) + optional parent
scopes named by the template's carry filter + the template instruction
(`instructions.ActivateDelegationSource`); L2 `Gather`/`Compose` over
the brief (inline, external-untrusted, its slot) and one lazy
record-object source per re-established reference (class =
f(witnessing event): l4-audit → registered trust under a matching
registry hash, model-turn / l8-delegation → floor; kind `tool:<name>`
/ `model-turn` / `delegation-output`; unique non-withheld slot by
kind else `evidence-slot-ambiguous`); every other non-withheld slot
declared absent. Capture = identities only. `Delegate` re-derives and
refuses a disagreeing capture (stage D), stores template bytes + the
composition object (exact `[system, user]` input), one tool-less
`Execute` under `min(turn timeout, remaining wall budget)` with the
request's `Model`, stores the output whole, classifies
completed / provider-error / output-over-bound /
model-identity-mismatch, and commits `l8-delegation` with refs to
every object. Walls: no tools/execution imports, one Execute (not in a
loop, `Model:` = request), one AppendEvent literal, no goroutines, no
model-name literal (`TestSeamIsNotASecondL7`, `TestDelegationPackageWall`).

Evidence: `subagents/delegation/seam/e2e_test.go` — positive path
first (admitted, witnessed, re-entered, window pure, capture = witness,
composition = exact model input, output = exact model output), stage A
denial, six stage-B refusals, four assembly refusals, three
post-instance outcomes, three fault points, doctored capture, carry
filter ×3. Mutation probes 2026-09-23: 17 controls, all killed after
folding the selectable-witness set into the single class-derivation
switch (two survivors were mutual cover, now one predicate).

Residual (recorded): per-item sensitivity derivation — l4-audit and
model-turn record no sensitivity, so `derived_sensitivity` is the
floor rank for every reference (every referenced item already passed
the parent's ceiling, C-L8-7 §5).

## Addendum (2026-09-23, after the Class-3 reviews)

- **Record-ref furniture (M6; architecture review MED-4):** after the
  frame of every AUTHORIZED evidence-bearing tool result — in every
  task, not only delegating ones — L7 appends
  `record-ref: <seq>:<objectID>` (the committed audit's seq and the
  evidence object's address), and after a completed delegation's frame
  the witness seq + output id. Deterministic, record-derived, outside
  the content-derived fence; part of the D-L7-11 projection
  (`ProjectDelegationMessage` renders it). It exists so a model can
  form a `<seq>:<id>` evidence reference from what it saw (C-L8-5).
- **Stage D inside instantiation (architecture review HIGH-1):** a
  non-refusal error from the delegate executor's compose half (record
  corruption, seam defect) is stashed by the per-task adapter and read
  by the loop immediately after `Handle`, BEFORE any audit commits →
  `ErrInvariant`, task FAILED, CORRUPT preserved. L4's `Outcome` has no
  invariant channel; this is the channel. `TestCorruptionDuringInstantiationIsStageD`.
- **Delegated EIS ⊆ parent's resolved set (security review MED-1):**
  the seam's `Resolve` re-reads the roots; every carried instruction
  must be byte-identical (`SourceHashes`) to the parent's set resolved
  at assembly, else stage D ("a governed root changed under the
  running task", C-L8-14 F). `TestRootChangeUnderRunningTaskIsStageD`.
- **Derived sensitivity = parent contract ceiling (security review
  MED-2):** a template whose contract ceiling is lower refuses at
  Gather (`compose-refused`); the template narrows, never widens.
  `TestTemplateCeilingNarrowsEvidence`.
- **Seam ≠ anchor (architecture MED-6 / security LOW-1):** Open
  requires the wired seam's `RegistryHash()` to equal the pin.
- Endpoints persist redacted to scheme://host (security LOW-4); the
  L10 seam carries a `PreResolve`'d contract into `EvaluateCall`
  (security LOW-5); typed registry sentinels (LOW-2); `RecordReader`
  read handle and a wider forbidden-call wall (LOW-9); declared-absence
  sources at the floor with no class check (LOW-7); `delegate`
  registrations are floor-trust by loader rule (LOW-3).
- **C-L8-14 G conformance (owner LOCK 2026-09-23):** assembly's
  template_scope validation is EXISTENCE (`Registry.Entry`), not
  usability; a withdrawn template stays assembly-admissible and the
  delegate call refuses stage B with `template-withdrawn`. "Registered
  establishes historical registration, not current usability; current
  usability is decided at the delegate admission boundary."
