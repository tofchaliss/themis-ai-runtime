package orchestration

// The walk: deterministic mechanics around an advisory core
// (D-L7-2/3/5). δ consumes only declared typed events; model content
// never reaches control; every effect passes L4; every step obeys
// record-before-next-turn (D-L7-11): compose → commit → deliver;
// model output → commit → next turn; audit → commit → result
// delivery.

import (
	stdctx "context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	l2 "github.com/tofchaliss/themis/context"
	"github.com/tofchaliss/themis/execution"
	"github.com/tofchaliss/themis/instructions"
	"github.com/tofchaliss/themis/runtime/model"
	"github.com/tofchaliss/themis/state"
	"github.com/tofchaliss/themis/tools"
)

// errBudgetFloor marks the constitution's wall-clock budget floor.
var errBudgetFloor = fmt.Errorf("wall-clock budget floor")

// fault is the deterministic fault-injection seam for Register C.
var fault func(point string) error

func faultAt(point string) error {
	if fault != nil {
		return fault(point)
	}
	return nil
}

type walk struct {
	o           *Orchestrator
	env         *Envelope
	wf          *WorkflowDef
	reg         *tools.Registry
	grant       *tools.Grant
	table       map[string]tools.Executor
	task        *state.TaskRecord
	l5          *execution.Env
	execCeiling *execution.WorkspaceExecutionCeiling
	spec        *execution.ProvisionSpec
	contract    *l2.Contract
	// eis is the task's instruction set, resolved ONCE at assembly
	// (D-L9-11): the walk holds instruction bytes, never a path it
	// would re-read per phase.
	eis *instructions.EffectiveSet

	phase     string
	edgeFires map[string]int64 // "phase/event" -> fires
	callState tools.CallState
	turnSeq   int64
	deadline  time.Time // the spec's wall_deadline_s — the budget floor
	lastSeq   int64     // seq of the most recent causally-relevant event
	// verifState is the latest-per-contract verification walk state
	// (D-L10-9, the CallState pattern): opaque token -> latest
	// committed outcome. Updated only after the EvVerification record
	// commits (record-before-event); re-derived by the replayer from
	// the event prefix. Gate satisfaction is checked against it
	// statelessly at each transition evaluation.
	verifState map[string]string
	// storedContracts dedups the per-task contract-bytes store (the
	// object store is content-addressed, so this is an I/O nicety,
	// not a correctness mechanism).
	storedContracts map[string]bool
}

// run drives the walk to a typed terminal. Every exit path seals and
// tears down the environment and projects a terminal lifecycle state.
func (w *walk) run() (TaskResult, error) {
	res := TaskResult{TaskID: w.env.TaskID}
	w.phase = w.wf.Initial
	w.edgeFires = map[string]int64{}
	w.callState = tools.CallState{Calls: map[string]int{}}
	w.verifState = map[string]string{}
	w.storedContracts = map[string]bool{}
	if l, ok := w.spec.Limit(execution.DimWallDeadlineS); ok {
		w.deadline = time.Now().Add(time.Duration(l.Value) * time.Second)
	}

	if err := w.task.Transition(state.StatusRunning, "assembled"); err != nil {
		return w.invariant(res, err)
	}

	for {
		outcome, err := w.runPhase()
		if errors.Is(err, errBudgetFloor) {
			// Constitution floor: seal(env-deadline) → FAILED — never
			// a workflow edge (D-L7-3).
			_ = w.l5.Seal(execution.SealDeadline)
			w.l5.Teardown()
			if terr := w.task.Transition(state.StatusFailed, "floor: wall-clock budget exhausted"); terr != nil {
				return w.invariant(res, terr)
			}
			res.Status = state.StatusFailed
			return res, nil
		}
		if err != nil {
			return w.invariant(res, err)
		}
		switch outcome {
		case TargetComplete:
			return w.complete(res)
		case TargetFail:
			return w.fail(res, "workflow-declared failure")
		default:
			// transition to another phase: continue the walk.
			w.phase = outcome
		}
	}
}

// runPhase executes one phase visit: fresh composition, then paced
// model turns until an edge moves the walk. Returns the next phase
// name or a terminal target.
func (w *walk) runPhase() (string, error) {
	p := w.wf.phase(w.phase)
	if p == nil {
		return "", fmt.Errorf("%w: cursor at unknown phase %q", ErrInvariant, w.phase)
	}
	// One L2 composition per phase entry (D-L7-11): the composed
	// payload is durably recorded BEFORE model delivery.
	conversation, err := w.composePhase(p)
	if err != nil {
		return "", err
	}

	var turns int64
	for {
		if turns >= p.MaxModelTurns {
			return w.step(EvTurnsExhausted)
		}
		// The wall-clock budget floor (constitution: floor:budget →
		// seal(env-deadline) → FAILED) — constitution-owned, never a
		// workflow edge.
		if !w.deadline.IsZero() && time.Now().After(w.deadline) {
			return "", errBudgetFloor
		}
		if err := faultAt("loop.pre-model-turn"); err != nil {
			return "", err
		}
		ctx, cancel := stdctx.WithTimeout(stdctx.Background(), time.Duration(w.env.TurnTimeoutSec)*time.Second)
		resp, mErr := w.o.cfg.Model.Execute(ctx, model.ExecutionRequest{
			Model: w.env.Model, Messages: conversation,
			Tools: w.toolDefs(p), Options: model.DefaultOptions()})
		cancel()
		turns++

		if mErr != nil {
			// Structural turn fact: provider error (harness-observed).
			if err := w.recordTurn("provider-error", nil, ""); err != nil {
				return "", err
			}
			return w.step(EvTurnProviderError)
		}
		// Model output → durable object + model-turn event BEFORE the
		// next turn (D-L7-11): the model's own prior output is
		// model-visible later and must be reconstructable.
		outObj, err := w.task.StoreObject(state.ObjEvidencePayload, turnBytes(resp))
		if err != nil {
			return "", err
		}
		fact := "tool-calls"
		if resp.Termination != model.TerminationToolCalls || len(resp.ToolCalls) == 0 {
			fact = "no-action"
		}
		if err := w.recordTurn(fact, &outObj, resp.Content); err != nil {
			return "", err
		}

		if fact == "no-action" {
			// The model's own prose is model-visible later: append it
			// (architecture review 2e — the conversation is append-only
			// AND complete, matching the durable record).
			conversation = append(conversation, model.Message{Role: model.RoleAssistant, Content: resp.Content})
			next, err := w.step(EvTurnNoAction)
			if err != nil || next != TargetStay {
				return next, err
			}
			continue // explicit governed stay: pace another turn
		}

		conversation = append(conversation, model.Message{Role: model.RoleAssistant, ToolCalls: resp.ToolCalls, Content: resp.Content})
		// The phase capability set narrows the grant (D-L7-6 "granted
		// per phase by the definition"): a granted tool outside this
		// phase is not-available HERE, through the genuine gate —
		// narrowing by instantiation, never a second permission system
		// (test review HIGH: this was an implementation hole).
		phaseGrant := w.phaseGrant(p)
		for _, call := range resp.ToolCalls {
			msg, ev, audit := tools.Handle(w.reg, phaseGrant, w.table, call, w.callState)
			// Record-before-effect: evidence object + audit event
			// committed before the result re-enters the loop.
			var refs []state.Ref
			if ev != nil {
				id, oerr := w.task.StoreObject(state.ObjEvidencePayload, ev.Evidence)
				if oerr != nil {
					return "", oerr
				}
				refs = append(refs, state.Ref{ID: id, Class: state.ObjEvidencePayload})
			}
			ab, _ := json.Marshal(audit)
			aev, aerr := w.task.AppendEvent(state.EvL4Audit, "l4", ab, refs...)
			if aerr != nil {
				return "", aerr
			}
			w.lastSeq = aev.Seq
			// CallState increment between calls: the recorded L4
			// obligation, monotonic by construction.
			w.callState.Calls[call.Name]++
			w.callState.Total++

			// Every tool result — control verbs included — re-enters
			// the conversation, keeping the tool_call/result protocol
			// pairing intact (security review MED-2: a dropped result
			// would let model behavior steer provider-protocol errors).
			conversation = append(conversation, msg)

			// Verifier-eligible capability result → the injected L10
			// evaluator (D-L10-6 stages 4-5, mechanical composition at
			// the execution-result boundary — L7 calls no L10 API and
			// interprets nothing). Assembly guarantees the hook exists
			// and the five events are declared whenever such a
			// capability is granted.
			if audit.Decision == "authorized" && w.isVerifier(call.Name) {
				var evidence []byte
				if ev != nil {
					evidence = ev.Evidence
				}
				vmsg, next, verr := w.evaluateVerification(call, evidence)
				if verr != nil {
					return "", verr
				}
				if vmsg != "" {
					// The typed result reaches the model as data in the
					// conversation (aperture 2 is the record; this is
					// the in-walk tool-result view of the same fact).
					conversation = append(conversation, model.Message{
						Role: model.RoleTool, ToolCallID: call.ID, Content: vmsg})
				}
				if next != "" && next != TargetStay {
					return next, nil
				}
				continue
			}

			// Control signal? (a typed GATE OUTCOME, not model
			// content): only an authorized control verb produces one.
			if audit.Decision == "authorized" && w.isControl(call.Name) {
				next, err := w.step(controlVerbs[call.Name])
				if err != nil || (next != TargetStay && next != "") {
					return next, err
				}
				continue
			}
			if audit.Decision == "error" {
				// Declared execution failure? δ sees it only if the
				// definition declared it; otherwise it is the model's
				// problem (typed result in conversation) within budgets.
				if w.declared(EvToolError) {
					next, err := w.step(EvToolError)
					if err != nil || next != TargetStay {
						return next, err
					}
				}
			}
		}
	}
}

// step presents one declared typed event to δ and applies exactly
// one governed edge (totality is static; anything else is an
// invariant violation). It records the cause-carrying
// workflow-transition event BEFORE the walk moves.
func (w *walk) step(event string) (string, error) {
	if !w.declared(event) {
		return "", fmt.Errorf("%w: event %q reached δ without declaration", ErrInvariant, event)
	}
	p := w.wf.phase(w.phase)
	// Edge selection (L10 amendment, D-L10-13/9): gated edges for the
	// event are evaluated in definition order against the
	// latest-per-contract walk state — first satisfied gate wins; the
	// mandatory ungated fallback fires otherwise. Gate satisfaction is
	// derived at THIS transition evaluation, never cached; the
	// replayer re-derives the identical choice from the event prefix.
	var edge *Edge
	var gateIdx = -1
	idx := 0
	for i := range p.Edges {
		e := &p.Edges[i]
		if e.On != event {
			continue
		}
		if e.Gate != nil {
			if edge == nil && w.verifState[e.Gate.Contract] == e.Gate.Outcome {
				edge, gateIdx = e, idx
			}
			idx++
			continue
		}
		if edge == nil {
			edge = e
		}
	}
	if edge == nil {
		return "", fmt.Errorf("%w: declared event %q has no edge in phase %q — static totality violated", ErrInvariant, event, w.phase)
	}
	target := edge.To
	// Edge identity: ungated edges keep "<from>/<on>" (one per event,
	// loader-refused otherwise — pinned by TestEdgeIdentityUnique);
	// gated edges are "<from>/<on>#g<idx>" by definition order among
	// that event's gated edges, so the fired branch is recorded, not
	// derivable.
	key := w.phase + "/" + event
	if gateIdx >= 0 {
		key = fmt.Sprintf("%s/%s#g%d", w.phase, event, gateIdx)
	}
	exhausted := false
	if edge.Counter > 0 {
		if w.edgeFires[key] >= edge.Counter {
			target = edge.ExhaustedTo
			exhausted = true
		} else {
			w.edgeFires[key]++
		}
	}
	if target == TargetStay {
		return TargetStay, nil
	}
	// Record-before-effect: the transition event commits before the
	// cursor moves (D-L6-10; cause-carrying per Q-L7-11). A countered
	// edge has two governed targets, so which branch fired is recorded
	// (exhausted), not left derivable.
	if err := faultAt("loop.pre-transition-commit"); err != nil {
		return "", err
	}
	tb := map[string]any{
		"from": w.phase, "to": target, "edge_id": key,
		"exhausted": exhausted, "cause_seq": w.lastSeq,
	}
	if edge.Gate != nil {
		// The fired gate is recorded (opaque token + required outcome)
		// so the branch choice is record content, never re-derivation
		// guesswork (D-L10-13).
		tb["gate"] = map[string]string{"contract": edge.Gate.Contract, "outcome": edge.Gate.Outcome}
	}
	body, _ := json.Marshal(tb)
	if _, err := w.task.AppendEvent(state.EvWorkflowTransition, "l7", body); err != nil {
		return "", err
	}
	if err := faultAt("loop.post-transition-commit"); err != nil {
		return "", err
	}
	return target, nil
}

// isVerifier reports whether a granted tool is registered
// verifier-eligible — a registry classification, never an
// authorization branch (the call already passed the identical L4
// gate).
func (w *walk) isVerifier(name string) bool {
	for i := range w.reg.Tools {
		if w.reg.Tools[i].Name == name {
			return w.reg.Tools[i].VerifierEligible
		}
	}
	return false
}

// evaluateVerification runs the injected L10 evaluation for one
// executed verifier call and applies D-L10-6 stage 5: durable stores
// and the EvVerification record commit BEFORE the typed event enters
// δ, and BEFORE the walk state updates. Returns the conversation
// message for the model, and δ's outcome when a transition fired.
func (w *walk) evaluateVerification(call model.ToolCall, evidence []byte) (string, string, error) {
	if w.o.cfg.Verifier == nil {
		// Assembly refuses this configuration; reaching here is an
		// invariant, not a policy decision.
		return "", "", fmt.Errorf("%w: verifier-eligible call %q with no evaluator wired", ErrInvariant, call.Name)
	}
	// executionRef names the committed L4 audit event of this call —
	// the execution record the evaluation references (D-L10-10 #6).
	execRef := fmt.Sprintf("l4:%d", w.lastSeq)
	// The AUTHORIZING registry's hash crosses with the call so the
	// evaluator can refuse drift between the registry that authorized
	// and the registry the contract pins (close security review L-1).
	vo, err := w.o.cfg.Verifier.EvaluateCall(w.env.TaskID, call, evidence, execRef, w.reg.Hash)
	if err != nil {
		// Evaluator machinery failure mints NO outcome (D-L10-8): the
		// harness invariant path, fail closed.
		return "", "", fmt.Errorf("%w: L10 evaluator failure: %v", ErrInvariant, err)
	}
	if vo.Refused {
		// Pre-instance typed refusal (D-L10-8): surfaced to the model
		// as data, never an outcome, never an event. The L4 audit of
		// the call is already committed; no unanchored object is
		// written (close review L-1 — an object no event references
		// would be unreachable under retention-is-reachability).
		return "verification refused: " + vo.RefusalReason, "", nil
	}
	evName, ok := verificationEventFor[vo.Outcome]
	if !ok {
		// Outcomes are minted only by the L10 evaluator; anything else
		// is machinery breakage.
		return "", "", fmt.Errorf("%w: unknown verification outcome %q", ErrInvariant, vo.Outcome)
	}

	// Durable stores in the no-reopen window (D-L10-10/R-L9-2): the
	// bytes the evaluator actually held. Content addressing dedups.
	if err := faultAt("loop.pre-verification-store"); err != nil {
		return "", "", err
	}
	var refs []state.Ref
	if len(vo.ContractBytes) > 0 && !w.storedContracts[vo.ContractToken] {
		cid, oerr := w.task.StoreObject(state.ObjEvidencePayload, vo.ContractBytes)
		if oerr != nil {
			return "", "", oerr
		}
		w.storedContracts[vo.ContractToken] = true
		refs = append(refs, state.Ref{ID: cid, Class: state.ObjEvidencePayload})
	}
	var recordID string
	for _, b := range [][]byte{vo.RawBytes, vo.CanonicalBytes} {
		if len(b) == 0 {
			continue
		}
		id, oerr := w.task.StoreObject(state.ObjEvidencePayload, b)
		if oerr != nil {
			return "", "", oerr
		}
		refs = append(refs, state.Ref{ID: id, Class: state.ObjEvidencePayload})
	}
	if len(vo.Record) > 0 {
		id, oerr := w.task.StoreObject(state.ObjEvidencePayload, vo.Record)
		if oerr != nil {
			return "", "", oerr
		}
		recordID = id
		refs = append(refs, state.Ref{ID: id, Class: state.ObjEvidencePayload})
	}

	// Record-before-event (D-L10-6 stage 5): the evaluation fact
	// commits to durable history before workflow control can see it.
	// The body NAMES the evaluation-record object explicitly —
	// reconstruction must never content-sniff unlabeled refs, where a
	// record-shaped hostile report could shadow the genuine record
	// (close reviews: architecture H-1 / security H-1).
	body, _ := json.Marshal(map[string]string{
		"contract": vo.ContractToken, "outcome": vo.Outcome,
		"record": recordID})
	if err := faultAt("loop.pre-verification-commit"); err != nil {
		return "", "", err
	}
	aev, aerr := w.task.AppendEvent(state.EvVerification, "l10", body, refs...)
	if aerr != nil {
		return "", "", aerr
	}
	if err := faultAt("loop.post-verification-commit"); err != nil {
		return "", "", err
	}
	w.lastSeq = aev.Seq
	// Latest-per-contract walk state updates only after the commit
	// (D-L10-9); the replayer re-derives it from EvVerification
	// events in the prefix.
	w.verifState[vo.ContractToken] = vo.Outcome

	msg := fmt.Sprintf("verification %s: %s", vo.ContractToken, vo.Outcome)
	if !w.declared(evName) {
		// Declaration-gated exposure makes this unreachable when
		// assembly held; fail as the invariant it is.
		return "", "", fmt.Errorf("%w: verification event %q undeclared yet produced", ErrInvariant, evName)
	}
	next, err := w.step(evName)
	if err != nil {
		return "", "", err
	}
	return msg, next, nil
}

func (w *walk) declared(event string) bool {
	for _, e := range w.wf.DeclaredEvents {
		if e == event {
			return true
		}
	}
	return false
}

// phaseGrant is the per-phase narrowing of the task grant: entries
// filtered to the phase's declared capability set. Quotas (CallState,
// TotalMaxCalls) remain task-global.
func (w *walk) phaseGrant(p *Phase) *tools.Grant {
	inPhase := map[string]bool{}
	for _, c := range p.Capabilities {
		inPhase[c] = true
	}
	g := *w.grant
	g.Entries = nil
	for _, e := range w.grant.Entries {
		if inPhase[e.Tool] {
			g.Entries = append(g.Entries, e)
		}
	}
	return &g
}

func (w *walk) isControl(name string) bool {
	for _, t := range w.reg.Tools {
		if t.Name == name {
			return t.Control
		}
	}
	return false
}

// taskPayloadSlot / taskPayloadKind: the loop's fixed L2 plan — one
// inline assignment carrying the envelope's untrusted half. The
// contract must declare this slot; a contract that doesn't refuses at
// Gather (Plan ⊆ Contract, fail closed).
const (
	taskPayloadSlot = "task-payload"
	taskPayloadKind = "task-brief"
)

// composePhase builds the phase's fresh conversation through the full
// L2 pipeline and records the composed payload durably before
// delivery (architecture review 2a, owner decision 2026-09-07:
// implement, not narrow). The payload never reaches the conversation
// raw: it travels Gather→Compose — Plan ⊆ Contract, content-derived
// fencing, provenance labels, typed absence.
func (w *walk) composePhase(p *Phase) ([]model.Message, error) {
	src := l2.Source{
		Name: taskPayloadSlot, Kind: l2.KindInline,
		Authority: l2.AuthorityExternalUntrusted, Sensitivity: l2.SensitivityPublic,
		Author: "task-submitter",
		Items:  []l2.ContextItem{{Kind: taskPayloadKind, Evidence: []byte(w.env.Payload)}},
	}
	g, err := l2.Gather(w.contract, []l2.Assignment{{Slot: taskPayloadSlot, Source: src}})
	if err != nil {
		return nil, err
	}
	composed, err := l2.Compose(w.eis, w.o.policy, g)
	if err != nil {
		return nil, err
	}
	// The composed user message IS what the model sees — those exact
	// bytes are the stored object (D-L7-11: byte-exact reconstruction).
	payloadObj, err := w.task.StoreObject(state.ObjEvidencePayload, []byte(composed.Messages[1].Content))
	if err != nil {
		return nil, err
	}
	body, _ := json.Marshal(map[string]any{
		"phase": p.Name, "eis_hash": composed.EISHash, "contract_hash": composed.ContractHash,
		"render_hash": composed.RenderHash, "l2_payload_hash": composed.PayloadHash,
		"payload": payloadObj, "framing": "l2-composed-v1",
	})
	if err := faultAt("loop.pre-compose-commit"); err != nil {
		return nil, err
	}
	ev, err := w.task.AppendEvent(state.EvL2Delivery, "l7", body, state.Ref{ID: payloadObj, Class: state.ObjEvidencePayload})
	if err != nil {
		return nil, err
	}
	w.lastSeq = ev.Seq
	return composed.Messages, nil
}

// toolDefs exposes exactly the phase's granted capability subset to
// the model (declaration only — L4 re-checks everything anyway).
func (w *walk) toolDefs(p *Phase) []model.ToolDef {
	var defs []model.ToolDef
	for _, cap := range p.Capabilities {
		for _, t := range w.reg.Tools {
			if t.Name != cap {
				continue
			}
			props := map[string]any{}
			var required []string
			for _, prm := range t.Params {
				props[prm.Name] = map[string]string{"type": string(prm.Type), "description": prm.Description}
				if prm.Required {
					required = append(required, prm.Name)
				}
			}
			schema, _ := json.Marshal(map[string]any{"type": "object", "properties": props, "required": required})
			defs = append(defs, model.ToolDef{Name: t.Name, Description: t.Description, Parameters: schema})
		}
	}
	return defs
}

func (w *walk) recordTurn(fact string, outObj *string, content string) error {
	w.turnSeq++
	// Model provenance preserved per turn (architecture review 4.2).
	body, _ := json.Marshal(map[string]any{"fact": fact, "turn": w.turnSeq, "model": w.env.Model})
	var refs []state.Ref
	if outObj != nil {
		refs = append(refs, state.Ref{ID: *outObj, Class: state.ObjEvidencePayload})
	}
	ev, err := w.task.AppendEvent(state.EvModelTurn, "l7", body, refs...)
	if err == nil {
		w.lastSeq = ev.Seq
	}
	return err
}

func turnBytes(resp *model.ExecutionResponse) []byte {
	b, _ := json.Marshal(map[string]any{
		"content": resp.Content, "termination": resp.Termination, "tool_calls": resp.ToolCalls,
	})
	return b
}

// complete: @complete → seal(task-complete) → egress → bind → teardown
// → COMPLETED (the closed termination mapping).
func (w *walk) complete(res TaskResult) (TaskResult, error) {
	if err := w.l5.Seal(execution.SealTaskComplete); err != nil {
		return w.invariant(res, err)
	}
	addr, err := w.l5.Egress(w.execCeiling, w.spec, w.o.store)
	if err != nil {
		w.l5.Teardown()
		return w.failClosed(res, "egress: "+err.Error())
	}
	artifactBytes, err := w.o.store.Get(addr)
	if err != nil {
		w.l5.Teardown()
		return w.failClosed(res, "artifact readback: "+err.Error())
	}
	artObj, err := w.task.StoreObject(state.ObjEgressArtifact, artifactBytes)
	if err != nil {
		w.l5.Teardown()
		return w.failClosed(res, err.Error())
	}
	if err := w.task.BindArtifact(artObj); err != nil {
		w.l5.Teardown()
		return w.failClosed(res, err.Error())
	}
	if st := w.l5.Teardown(); st != execution.StateDestroyed {
		// Teardown anomaly is host-state news, not record news: the
		// task still completes; the anomaly is in the L5 trace.
		_ = st
	}
	if err := w.task.Transition(state.StatusCompleted, "workflow @complete"); err != nil {
		return w.invariant(res, err)
	}
	res.Status, res.Artifact = state.StatusCompleted, artObj
	if v, err := w.o.root.ReadStatus(w.env.TaskID); err == nil {
		res.Verdict = v.Verdict
	}
	return res, nil
}

// fail: @fail (workflow-chosen) → seal(caller-abort) → teardown → FAILED.
func (w *walk) fail(res TaskResult, reason string) (TaskResult, error) {
	_ = w.l5.Seal(execution.SealCallerAbort)
	w.l5.Teardown()
	if err := w.task.Transition(state.StatusFailed, reason); err != nil {
		return w.invariant(res, err)
	}
	res.Status = state.StatusFailed
	if v, err := w.o.root.ReadStatus(w.env.TaskID); err == nil {
		res.Verdict = v.Verdict
	}
	return res, nil
}

// failClosed: a floor fired (budget, record, egress failure) —
// constitution path, typed, never workflow-mappable.
func (w *walk) failClosed(res TaskResult, reason string) (TaskResult, error) {
	if err := w.task.Transition(state.StatusFailed, "floor: "+reason); err != nil {
		res.Status = state.StatusFailedPartial
		return res, nil
	}
	res.Status = state.StatusFailed
	return res, nil
}

// invariant: the fixed constitution-owned path (Q-L7-7): typed
// invariant event → fatal-breach seal → unconditional teardown →
// FAILED. Never reroutable by any workflow.
func (w *walk) invariant(res TaskResult, cause error) (TaskResult, error) {
	body, _ := json.Marshal(map[string]string{"cause": cause.Error()})
	if _, aerr := w.task.AppendEvent(state.EvL7Invariant, "l7", body); aerr != nil {
		// An unrecordable invariant still fails closed; the append
		// failure joins the returned cause so nothing is silent.
		cause = fmt.Errorf("%v (invariant event unrecordable: %v)", cause, aerr)
	}
	_ = w.l5.Seal(execution.SealFatalBreach)
	w.l5.Teardown()
	_ = w.task.Transition(state.StatusFailed, "invariant: "+cause.Error())
	res.Status = state.StatusFailed
	return res, fmt.Errorf("%w: %v", ErrInvariant, cause)
}
