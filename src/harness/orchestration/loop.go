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

	phase     string
	edgeFires map[string]int64 // "phase/event" -> fires
	callState tools.CallState
	turnSeq   int64
	deadline  time.Time // the spec's wall_deadline_s — the budget floor
	lastSeq   int64     // seq of the most recent causally-relevant event
}

// run drives the walk to a typed terminal. Every exit path seals and
// tears down the environment and projects a terminal lifecycle state.
func (w *walk) run() (TaskResult, error) {
	res := TaskResult{TaskID: w.env.TaskID}
	w.phase = w.wf.Initial
	w.edgeFires = map[string]int64{}
	w.callState = tools.CallState{Calls: map[string]int{}}
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
	var edge *Edge
	for i := range p.Edges {
		if p.Edges[i].On == event {
			edge = &p.Edges[i]
			break
		}
	}
	if edge == nil {
		return "", fmt.Errorf("%w: declared event %q has no edge in phase %q — static totality violated", ErrInvariant, event, w.phase)
	}
	target := edge.To
	key := w.phase + "/" + event
	if edge.Counter > 0 {
		if w.edgeFires[key] >= edge.Counter {
			target = edge.ExhaustedTo
		} else {
			w.edgeFires[key]++
		}
	}
	if target == TargetStay {
		return TargetStay, nil
	}
	// Record-before-effect: the transition event commits before the
	// cursor moves (D-L6-10; cause-carrying per Q-L7-11).
	if err := faultAt("loop.pre-transition-commit"); err != nil {
		return "", err
	}
	body, _ := json.Marshal(map[string]any{
		"from": w.phase, "to": target, "edge": event, "cause_seq": w.lastSeq,
	})
	if _, err := w.task.AppendEvent(state.EvWorkflowTransition, "l7", body); err != nil {
		return "", err
	}
	if err := faultAt("loop.post-transition-commit"); err != nil {
		return "", err
	}
	return target, nil
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
	composed, err := l2.Compose(w.o.eis, w.o.policy, g)
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
