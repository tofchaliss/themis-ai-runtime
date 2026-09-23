# Tasks: Layer 8 — Subagents (Delegation)

Grill closed 2026-09-22 (Q-L8-1..20 all disposed; design.md §2 holds
D-L8-1..21, §5 the Gate 0 inventory and five proof registers). Gate 0
PASS 2026-09-22 (D-L8-21). Architecture CLOSED 2026-09-22 (owner LOCK at
C-L8-21); challenge record C-L8-1..21 in design.md §6. Three-state verdicts per milestone:
architecture-conformant · test-evidenced · operationally-proven. An
editing command or commit message is never evidence. The C17 lesson is
part of L8 proof discipline: the positive path is proven before any
negative is trusted.

**Standing rule for every milestone (D-L8-21):** if implementation
requires an artifact, package, registry, runtime path, event class,
authority field, or control mechanism not present in design.md §5,
implementation STOPS and the surface is classified (implementation
detail / residual / architectural decision) before work continues.

## 0. Gate

- [x] Grill held; Q-L8-1..20 disposed; D-L8-1..21 locked (owner-led,
      2026-09-22)
- [x] Gate 0 PASS (owner, 2026-09-22): inventory = implementation
      whitelist; five registers judged sufficient
- [ ] Gate 1: implementation design reviewed against §5 before each
      milestone's code — M1 portion recorded 2026-09-23 in
      `RESUME-HERE.md` (registry schema, template schema, preflight);
      grant digest, L4 target validation, L7 post-hook, L2 composition,
      L6 event construction to be recorded there before M2/M3 code

## 1. L8-M0 — P-L8-1 prerequisite: provider-response hard ceiling (Class 3, NOT L8)

Parent runtime hardening; blocks M4+. Closes the unbounded `io.ReadAll`
at `runtime/model/ollama.go:138` and `openai.go:155`.

- [x] Governed hard byte ceiling on the provider response body at both
      adapters (`io.LimitReader` + typed over-limit termination, never a
      silent truncation; a truncated body is a failed turn, not a turn)
      — `runtime/model.readBounded`, `ErrResponseOverCeiling` (2026-09-23)
- [x] Ceiling is a deployment-supplied governed input — Gate 1 decision:
      compiled hard bound `DefaultMaxResponseBytes` (256 KiB) that the
      anchor-pinned model registry (`max_response_bytes`) may NARROW,
      never widen; the `absent`-registry path keeps the compiled bound
      (`themis-run` builds the adapter directly); ≤ L2 `MaxItemBytes`
      pinned by test (C-L8-7)
- [x] Tests: over-limit body → typed termination, nothing stored as a
      turn; at-limit body → ordinary turn; mutation: remove the limiter
      → test fails (`runtime/model/ceiling_test.go`, probed 2026-09-23)
- [x] Recorded as an amendment to the model-interface seam (not an L8
      artifact): L4 archive `amendments/p-l8-model-seam/`
- [x] **P-L8-2:** adapters populate provider-neutral `Identity.Reported`
      from the provider payload (inside the adapter); test: reported ≠
      requested is observable (C-L8-10) — both adapters, 2026-09-23

## 1b. L8-M0b — Supporting-layer prerequisites (owner closure §35; NOT L8)

Must be complete before L8 implementation is declared safe. None
reopens D-L8-1..21.

- [x] **F-L8-2 (L7):** `model-turn` body gains `Identity{WireModel,
      Runtime, Reported}` + `Endpoint` (additive; no constitution change)
      — `TestModelTurnRecordsExecutionIdentity`; L7 archive
      `amendments/l8-prerequisites/`
- [x] **F-L8-3 (L10):** verifier-seam pre-instance refusal text
      reconstructable from the record — `VerificationEvaluator.PreResolve`
      before the `l4-audit` commit, refusal in the audit body
      (`VerificationRefusal`); L10 archive
      `amendments/f-l8-3-refusal-record/` (residual: second-load window)
- [x] **F-L8-4 (L7):** parent turn context deadline =
      `min(turn_timeout_sec, remaining deadline)` —
      `TestParentTurnDeadlineIsMinOfTurnAndWall`, mutation-probed
- [x] **Q-SA-6 (L9/L7, issue #1):** DISCHARGED 2026-09-23 — D-SA-4
      `template_scope` equality built (`tools.Instantiates`), anchored
      positive twin + live anchored walk green; record archived at
      `openspec/changes/archive/2026-09-23-l9-l7-skill-admission-identity/`
      (production `rsys@4` act still the owner's)

## 2. L8-M1 — Governance artifacts + loaders (Class 3)

- [x] `policies/delegation/registry.json` (append-only; `name, version,
      template_sha256, manifest_path, state, steward`) — mirrors
      `policies/verification/contracts.json` (2026-09-23)
- [x] `policies/delegation/<name>/template.json` loader: closed schema,
      DisallowUnknownFields, duplicate-key refusal, trailing-content
      refusal, sha256 pins for `context_contract` and optional
      `instruction`, `eis_carry_scopes[]` ⊆ {repository, directory,
      skill} (mandatory roots carried unconditionally by L7 code, never
      filterable; `task` never carries — C-L8-4), `brief {slot, max_bytes}`,
      `max_output_bytes`; **disjointness enforced at load**: any
      workflow / ceiling / grant / spec / input_schema / template
      reference → refusal (D-L8-5/6)
- [x] Registry loader: `Resolve(name@version)` exact only, no
      `latest`/ranges; `CheckAppendOnly` (deletion / rebind /
      un-withdrawal detected); two-way identity agreement; confined
      paths; no write API — AST write-wall (the L9/L10 wall pattern)
- [ ] First registered template (PROPOSED → Governance act): one
      template for `remediate-dependency@1`'s triage need; contract is
      an ordinary L2 contract with a `brief` slot of class
      `external-untrusted` — `dependency-triage@1` written and listed
      `active` so the positive twin loads (2026-09-23); the
      registration ACT is the owner's commit keeping or changing it
- [x] Loader cross-checks: brief slot exists in the pinned contract and
      permits only `external-untrusted`; permitted classes ∈ L2's closed
      vocabulary; no requiredness/slot semantics in `template.json`
      (C-L8-14 D)
- [x] `policies/delegation/README.md` with the registration-review
      checklist (C-L8-14): no directive to disregard a higher scope; no
      factual claims in instruction files; classes/sensitivity/bounds
      justified
- [x] `themis-preflight` verifies every registered template's bytes
      against its registry pin (C-L8-14 F) — "Delegation templates"
      section, manifest + every pin; PASS on the P0 bundle
- [x] Register A proofs + mutation probes (wall, closed world,
      disjointness, exact resolution) — `subagents/delegation/
      delegation_test.go`: positive twin through the real registry
      first; 13 probes (root-naming carry, disjointness, pin hash,
      brief class, withheld, output bound, withdrawn, registry hash,
      two-way identity, rebind, un-withdrawal, exact version, key wall)
      all killed 2026-09-23
- [x] Package `src/harness/subagents/delegation/`; **delete `roles/`,
      `runtime/`, `isolation/`** in the same change (2026-09-23)

## 3. L8-M2 — L4 amendment: `delegate` capability (Class 3; L4 archived-layer amendment)

- [x] Tool registry v5: `delegate`, target class `delegation-template`,
      `verifier_eligible: false`, `trust: external-untrusted`, params
      `template` / `evidence` / `brief` (string; `evidence` = list of
      `<seq>:<objectID>` references into the parent's stream,
      shape-checked by regex at L4; existence, task reachability, hash
      integrity, and class derivation re-established in the seam from
      the record — C-L8-5; D-L8-15 stage A/B)
- [x] `template_scope` is a fixed member of the Skill's `grant_template`,
      copied verbatim at instantiation; `instantiateGrantTemplate`
      accepts no override for it (C-L8-16 F/G) —
      `skills.TestInstantiationSurfaceHasNoTemplateScope` (2026-09-23)
- [x] `GrantEntry.TemplateScope []string` — LANDED with skill-admission
      SA-M4 (commit dc8034c; exact-ref validation at `LoadGrant`,
      set-equality in `tools.Instantiates`, digested in
      `grantAuthorityDigest`); inert at `Authorize` until `delegate`
      exists. Do not re-add.
- [x] `Authorize` checks target ∈ scope by **exact `name@version` equality** (no prefix matching;
      loader refuses non-exact entries) — the single authoritative
      substitution gate (C-L8-15 G)
- [x] **`grantAuthorityDigest` includes `TemplateScope`** — LANDED with
      SA-M4 (dc8034c); differing-digest cases (gained / swapped /
      emptied) added 2026-09-23
- [x] `execDelegate` executor = **instantiation** (C-L8-12 Am. 1,
      LOCKED at C-L8-21): resolve template, validate evidence against
      the record, `Resolve`, `Gather`/`Compose` in memory, bound checks —
      pure reads under registry `timeout_sec`; failure → `ErrClass:
      delegation-refused:<class>` (reconstructable via `l4-audit`);
      success → deterministic **instantiation capture** (identities only,
      fact kind `l6_execution_record`) as the audited evidence;
      composition/model call/output/witness stay in the post-hook,
      which **re-derives** the composition without reading the capture;
      runtime re-derivation ≠ capture → stage D invariant; at
      reconstruction any pairwise mismatch → typed DISCREPANCY (C-L8-13)
- [x] Delegation error classes added to the closed `ErrorClass`
      vocabulary (C-L8-11) — `tools/delegate.go`, `KnownErrorClass`
- [x] Amendment record under
      `openspec/changes/archive/2026-09-06-layer-04-tool-interface/amendments/l8-delegate/`
      (L4 side landed 2026-09-23; the seam's compose half lands at M4)

## 4. L8-M3 — L6 constitution amendment: `l8-delegation` (Class 3/4 — constitution hash changes)

- [x] `EvL8Delegation = "l8-delegation"` in `eventClasses` (caller
      class, not primitive-only); closed-vocabulary tests updated
      (`state.TestL8DelegationEventClass` pins count 16 and
      membership; 2026-09-23)
- [x] Event body per D-L8-8 (parent_call_seq / template / composition /
      evidence_refs / model_identity / output_object_ref / outcome /
      termination — **no `delegation_id`**; identity = `(task_id,
      seq)`, C-L8-20); L6 validates the envelope, never the content;
      Register C: event body reproducible from inputs (no random field)
      — `subagents/delegation/event.go` (`Event`, closure, canonical
      encoding, `SortEvidence`); `TestEventBodyReproducible`,
      `TestEventClosure` (2026-09-23)
- [x] Amendment record under
      `openspec/changes/archive/2026-09-07-layer-06-durable-state/amendments/l8-delegation-event/`
- [x] Anchor consequence recorded: `constitution.state` re-pin → M6
      (`rsys2.json` proposed pin is stale as of 2026-09-23)

## 5. L8-M4 — Delegator seam + L7 wiring (Class 3; L7 archived-layer amendment)

- [x] `orchestration.Config.Delegator` one-way interface (three entry
      points after C-L8-12 Am. 1/C-L8-13 — Registered / Instantiate /
      Delegate; one post-hook call site; 2026-09-23):
      inputs `{task id, model identity, call id, args, parent_call_seq}`
      + a read handle on the task record; **no conversation, response,
      or system-message type in the signature** (C-L8-17; wall test by
      AST)
- [x] Assembly: a phase exposing `delegate` with nil delegator →
      `ErrAssembly` (mirror of the verifier check); `delegate` must be
      in ceiling `allowed_tools`; every grant `template_scope` entry
      must resolve in the template registry (grant validation, not
      call authorization — C-L8-15 G)
- [x] Loop post-hook `isDelegation` (registry classification, never an
      authorization branch) → seam → paired tool-result re-entry via
      the existing `frameToolResult` path with the `delegate` entry's
      `trust` (`authority: external-untrusted`, `hash:` = output object
      id); non-completed outcomes as unframed typed errors from the
      closed class set; the branch never calls `w.step` (C-L8-11);
      sequential in request order (D-L8-16 §6)
- [x] Seam: resolve template (stage B refusals typed, no event, no
      objects) → L1 `Resolve` over parent-subset sources + optional
      template instruction (a new resolution epoch, hashed) → L2
      `Gather`/`Compose` — the brief as its own `external-untrusted`
      source in its slot (author = delegating model, origin = task +
      call seq; fenced; not pattern-scanned, like the task payload —
      C-L8-17 B/C) — with evidence refs resolved per C-L8-5 (event
      at seq in the parent stream → `Refs` contains id →
      `GetObject` re-hash → class from the event under a matching
      registry hash; selectable event classes `{l4-audit, model-turn,
      l8-delegation}`);
      every reference's seq strictly < `parent_call_seq` (the
      authorizing `l4-audit`) else stage B refusal — C-L8-8;
      each reference assigned to the unique non-withheld contract slot
      whose `kind` matches the item's event-derived kind (kinds:
      `tool:<name>` for `l4-audit`, `model-turn`, `delegation-output`
      for `l8-delegation` — C-L8-17 F); zero or >1
      → `delegation-refused:evidence-slot-ambiguous` (C-L8-14 C);
      references canonicalized by ascending seq before source
      construction, exact duplicates refused (stage B), one `Source`
      per reference, no L8 dedup — C-L8-6; sources are **lazy L6-object
      sources** (new L2 source kind, `collect()` = `GetObject` + hash
      verify) so L2 Gather pulls bytes under its own caps and the seam
      reads no evidence bytes — C-L8-7 and the brief in its slot →
      store the composition object = the exact model input bytes
      (system + user messages, canonical) + template manifest/contract/
      instruction bytes (R-L9-2, C-L8-9) → deadline/floor
      check → one `Model.Execute` under parent `turn_timeout_sec` →
      store output → `l8-delegation` → envelope; the model call's context
      deadline = `min(turn_timeout_sec, remaining to parent deadline)`
      after the pre-invocation floor check (C-L8-19 H); `output-over-bound`
      path per D-L8-16 §5 (full storage, typed failure re-entry, no
      truncation)
- [x] `faultAt` points: `delegation.pre-composition-store`,
      `delegation.pre-output-store`, `delegation.pre-event-commit`
- [x] Not-a-second-L7 wall test (D-L8-21 §3): imports, single
      `Execute` whose `Model:` is the L7-supplied identity (mutation:
      constant → fails), no model-name literal, no registry access,
      single call site, no goroutines, single event literal,
      `controlVerbs`/`verificationEvents` unchanged
- [x] **F-L8-2:** parent `model-turn` body gains `Identity` + `Endpoint`
      (additive; no constitution change); `l8-delegation.model_identity`
      = `{governed{name, registry_hash}, execution{...}}`; stage C
      outcome `model-identity-mismatch` when reported ≠ requested
      (C-L8-10)
- [x] Amendment record under
      `openspec/changes/archive/2026-09-07-layer-07-orchestration/amendments/l8-delegation-seam/`
- [x] Landed 2026-09-23 — see the L7 archive amendment
      `amendments/l8-delegation-seam/` (seam, assembly, post-hook,
      walls, 17/17 mutation probes) and the L2/L1 amendments
      `l8-record-object-source/`, `l8-delegation-source/`. Residual:
      per-item `derived_sensitivity` is the floor rank in v1.

## 6. L8-M5 — Record, reconstruction, observation (Class 2/3)

- [x] Register C: byte-exact reconstruction of what the delegated model
      saw and produced; reconstruction from L6 alone after (i) template
      withdrawal, (ii) template/contract file deletion, (iii) root file
      change → CONFIRMED; missing object → typed UNREPRODUCIBLE; doctored
      identity → typed DISCREPANCY; registry rebind → load refusal
      (C-L8-9); window purity — every event between
      `parent_call_seq` and the `l8-delegation` seq is delegation-owned,
      and `evidence.seq < parent_call_seq` holds for every ref (C-L8-8); evidence-order permutation → identical
      composition hashes (mutation: remove the seam sort → fails on two
      equal-hash items — C-L8-6); two-way template identity (registry hash vs
      stored bytes); fault sweep at every new point (stage E orphan
      object retained, reported by reachability, **not a fact**;
      recovery infers no execution)
- [x] L10 history/reconstruction views include `l8-delegation`
      (read-only; no evidence kind, contract, or outcome)
- [x] Register D: static boundedness statement computed at assembly
      and recorded (`l8_execution_bound`): executions ≤ V·T + min(D,G)
      ≤ W + M; output captured ≤ min(D,G)·P; wall ≤ Δ (C-L8-19 I) —
      `TestStaticExecutionBoundWithDelegate`; the admitted-to-parent
      output bound (·B) is a per-delegation template bound enforced at
      re-entry (`TestDelegationPostInstanceOutcomes/output over
      bound`), not a static term; quota-attempt semantics pinned
      (`TestDelegateQuotaCountsEveryAttempt`)
- [x] Register C: the class derivation function `f` is total over
      event classes and maps every L7/L8-writable class to the floor or
      a registry-derived value; mutation: a branch reading a referenced
      object's prior class → fails (C-L8-18); conversation projection
      after a delegation (framed result, refusal error) re-derives
      byte-exactly from `l8-delegation`/`l4-audit` + objects (C-L8-18 A)
- [x] Landed 2026-09-23 — `subagents/delegation/seam/reconstruct.go`
      (`ReconstructDelegation`/`ReconstructTask`: CONFIRMED /
      UNREPRODUCIBLE-FOR-MISSING-INPUTS / DISCREPANCY naming the pair;
      discrepancy outranks missing input), `orchestration/projection.go`
      (`ProjectDelegationMessage`, `ProjectRefusalMessage`,
      `executionBound` recorded as `l8_execution_bound`), L10
      `HistoryView.Delegations`; tests in `reconstruct_test.go`
      (record-alone CONFIRMED after withdrawal/deletion/root change,
      typed failures, permutation invariance on equal-hash items,
      quota-attempt semantics, f totality + AST wall, byte-exact
      projection); probes: seam sort, window purity, payload compare,
      template two-way identity — all killed. Residuals: registry-less
      reconstruction takes an l4-audit class from the witness and
      reports the registry as a missing input; fault sweep at the
      three points is in `TestDelegationFaultPoints` (M4), orphan
      reachability reporting not separately asserted.

## 7. L8-M6 — Anchor, positive path, live proof, close (Class 3/4)

- [x] Deployment anchor: `delegation_template_registry` pin +
      `constitution.state` re-pin → `rsys@4` PROPOSED
      (`policies/deployment/rsys4.proposed.json`, 2026-09-23; with it
      PROPOSED: registry-v5, `report-valid@2`, `remediate-dependency@2`,
      `dependency-triage@1`) — the Governance act is the owner's;
      `themis-status` / `themis-preflight` print the pin
- [x] **Register B, positive path FIRST:** a TEST-HARNESS anchor with
      the `rsys@4` pin set accepts and executes a genuine `delegate`
      call from `remediate-dependency@2` (L9-instantiated), the model's
      reference formed from the record-ref furniture, witness present,
      reconstruction CONFIRMED (`TestAnchoredDelegationPositivePath`);
      negatives: scope, closed world, withdrawn, unreachable evidence,
      over-bound brief, over-bound output, nil delegator, pin/path
      disagreement (unanchored suite + `TestAnchoredDelegationRefusals`);
      both-ways mutation on 21 controls (M4/M5 probes). NOT done:
      production `rsys@4` via `themis-run` (owner act)
- [~] Register E: `TestLiveDelegationWalk` (qwen2.5:7b, anchored @2,
      unmodified loop) — admitted, typed terminal (FAILED via
      turn-no-action exhaustion in REMEDIATE); the model declared done
      in ANALYZE without delegating, so "result consumed by the next
      turn" is NOT evidenced live (model behaviour, the live-register
      discipline); reconstruction of any live delegation is asserted
      CONFIRMED when one exists
- [~] Three Class-3 close reviews in isolated worktrees against
      `71a7188` (2026-09-23): architecture and security complete; test
      review interrupted (session rate limit) — relaunched against the
      remediated commit. Dispositions:
      - architecture HIGH-1 (machinery failure in instantiation
        downgraded to a tool error) → invariant channel in the loop,
        `TestCorruptionDuringInstantiationIsStageD`, probe killed;
        MED-2 (withdrawn template in `template_scope` refused the whole
        Skill at assembly, stricter than C-L8-14 G) → **owner LOCK
        2026-09-23: amend the code, C-L8-14 G governs.** `Registered`
        distinguishes existence from lifecycle usability
        (`Registry.Entry`): a withdrawn template remains
        assembly-admissible when referenced by a governed Skill; the
        delegate call refuses stage B with `template-withdrawn`,
        witnessed in the audit; unregistered still refuses at assembly.
        Proofs: unregistered → assembly refuses; withdrawn → assembly
        admits, stage-B refusal witnessed, walk completes (unanchored +
        anchored); withdrawn and never requested → walk completes;
        active → positive path. Mutant (usability at assembly) killed; MED-3 (`Delegator` has
        three entry points vs D-L8-21 §3 "one method") → **owner LOCK
        2026-09-23: §3 text amended** (three one-way entry points; no
        method carries the parent conversation), code unchanged, no
        architecture reopened; MED-4 (record-ref
        furniture changes every task's tool-result bytes) → recorded in
        the L7 amendment addendum; MED-5 (@2 procedure named
        `report-valid@1`) → fixed, re-pinned; MED-6 (seam registry not
        bound to the pin) → Open compares `RegistryHash()`; LOW-7/8/9 →
        fixed
      - security: no CRITICAL/HIGH; MED-1 (delegated EIS re-read from
        disk) → parent-set subset check, stage D on divergence, probe
        killed; MED-2 (sensitivity floor) → parent contract ceiling,
        probe killed; LOW-1..5 → fixed (LOW-1 = arch MED-6); note:
        providers reporting dated model ids will mint
        `model-identity-mismatch` on every delegation — residual
      - test (relaunched against `ad020a7`): HIGH-1 (the "prefix"
        scope case exercised the ref-shape gate, not C-L8-15 G) → case
        `dependency-triage@10` vs scope `@1` refuses
        `template-outside-grant-scope`, prefix mutant killed; HIGH-2
        (seam-hash-vs-pin check untested) → anchored negative, killed;
        MED-3 (missing parent turn object silently CONFIRMED) → typed
        UNREPRODUCIBLE, killed; MED-4 (`evidence-slot-ambiguous`,
        model-turn / delegation-output evidence untested) →
        `TestEvidenceSlotAmbiguousRefuses`,
        `TestModelTurnAndDelegationOutputAsEvidence`, `tool:*` mutant
        killed; MED-5 (Register D overclaim) → `ExecutionBound` gains
        `OutputCaptured = min(D,G)·P` and `WallDeadlineS = Δ`; the
        admitted-to-parent term (·B) is per template and stays a
        per-delegation bound, not a static one — wording narrowed
        below; the unreachable `Executions > W+M` runtime check
        removed (recorded statement only); MED-6 (deadline / empty
        output / unreported identity paths) →
        `TestDelegateDeadlineAndEdgeOutcomes`, floor and reported-empty
        mutants killed; MED-7 (stage-E orphan not asserted) → the
        pre-event-commit fault test asserts the composition object is
        retained and unreachable; LOW-8/9/10/11/12/13 → tests added,
        mutants killed; LOW-14 (post-hook seq guard is mutual cover
        with `Event.Validate`) → recorded, left as defence in depth.
        12/12 named survivors now killed
- [ ] `traceability.md`; archive under
      `openspec/changes/archive/<date>-layer-08-subagents/`;
      `docs/harness-layer-status.md` and `execution-chain.md` updated
      (L8 row, G2 fact table row `l8_delegation_record`)

## Residuals carried (each with its own gate; none implemented here — owner closure §36)

- Tool-capable L8 (reopens the depth-1 proof; a new architecture
  decision)
- δ-declarable delegation event (new workflow vocabulary + proofs)
- Parallel fan-out (explicit deterministic ordering key required)
- Per-target quotas in the grant vocabulary (per-template `max_calls`)
  — generic L4 residual (C-L8-16 D)
- Caller-narrowed `template_scope` — L9 instantiation-surface
  extension, safe in principle (C-L8-16 F)
- Phase-level capability parameters (per-phase template binding);
  grants are task-level for every tool in v1 (C-L8-15 F)
- Parent-loop handling of provider-reported model mismatch (L7
  decision; P-L8-2 makes it observable)
- Provider version/digest governance beyond the registry's
  runtime+endpoint pin
- Approval channel; anything touching the dissolved OPEN-2
