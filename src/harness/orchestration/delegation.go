package orchestration

// The L8 delegation seam boundary (openspec/changes/l8-subagents §5.1,
// D-L8-3, C-L8-12/13/17). L7 defines only the boundary types; the
// implementation lives in subagents/delegation/seam and is injected at
// service wiring, so L7 keeps zero dependency on the delegation
// package (the verification-seam pattern). The interface carries no
// conversation: no []model.Message, no ExecutionResponse, no system
// message — the parent conversation cannot reach the seam (C-L8-17;
// AST wall in the tests).

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	hctx "github.com/tofchaliss/themis/context"
	"github.com/tofchaliss/themis/instructions"
	"github.com/tofchaliss/themis/runtime/model"
	"github.com/tofchaliss/themis/state"
	"github.com/tofchaliss/themis/tools"
)

// Delegator is the one-way L8 seam. Three entry points, each a pure
// function of its request and the record: Registered (assembly-time
// grant validation), Instantiate (stage B, invoked by the `delegate`
// executor through the adapter below), Delegate (the post-hook:
// re-derive, store, one model call, store, witness).
type Delegator interface {
	// Registered reports whether an exact template reference resolves
	// in the registry in force (C-L8-15 G: assembly validates the
	// grant's template_scope entries; it is not call authorization).
	Registered(ref string) error
	// Instantiate performs the deterministic instantiation and returns
	// the capture (identities only). A *DelegationRefusal is the
	// stage-B refusal; any other error is machinery failure.
	Instantiate(req InstantiationRequest) ([]byte, error)
	// Delegate re-derives the composition, executes exactly one
	// tool-less model call, stores, and commits the l8-delegation
	// witness. An error is machinery failure (stage D, invariant path);
	// every post-instance outcome is a result, never an error.
	Delegate(req DelegationRequest) (*DelegationResult, error)
}

// EvidenceRef is one shape-checked reference into the parent's record.
type EvidenceRef struct {
	Seq      int64
	ObjectID string
}

// InstantiationRequest is what the executor's compose half receives:
// the authorized selection plus a READ handle on the record and the
// parent's already-activated instruction sources. Nothing of the
// parent's conversation.
type InstantiationRequest struct {
	TaskID   string
	Root     *state.Root // read handle: events + objects
	Template string
	Evidence []EvidenceRef
	Brief    string
	// ParentSources are the parent's activated L1 sources; the seam
	// carries the mandatory roots unconditionally and the optional
	// scopes only where the template's filter names them (C-L8-4).
	ParentSources []instructions.Source
	// RegistryHash and ToolTrust derive an l4-audit item's class: the
	// registered trust of the recorded tool under a still-matching
	// registry hash (C-L8-5 §4). A function value, not the registry.
	RegistryHash string
	ToolTrust    func(tool string) (hctx.AuthorityClass, bool)
}

// DelegationRequest is the post-hook's input.
type DelegationRequest struct {
	InstantiationRequest
	Task          *state.TaskRecord // write handle for stores + the witness
	ParentCallSeq int64             // the authorizing l4-audit's seq
	CallID        string            // protocol pairing only; not identity
	Capture       []byte            // the executor's instantiation capture
	// Model is the parent's governed model identity (env.Model) and
	// Runtime the same injected adapter (C-L8-10). ModelRegistryHash is
	// the anchor-pinned registry ("absent" when the deployment declares
	// none).
	Model             string
	Runtime           model.Interface
	ModelRegistryHash string
	TurnTimeout       time.Duration
	Deadline          time.Time // zero = no wall budget
}

// DelegationResult is the seam→L7 value (D-L8-2 result wrapper):
// deterministic metadata plus the output bytes; L7 unwraps it, the
// model never sees it.
type DelegationResult struct {
	Outcome        string // completed | provider-error | output-over-bound | model-identity-mismatch
	Seq            int64  // the l8-delegation event's seq
	OutputObjectID string
	Output         []byte // present only when Outcome == completed
}

// DelegationRefusal is the typed stage-B refusal; Reason is a member of
// the closed L4 vocabulary (tools.DelegationRefusalReason).
type DelegationRefusal struct {
	Reason string
	Detail string
}

func (r *DelegationRefusal) Error() string { return "delegation-refused:" + r.Reason + ": " + r.Detail }

// SetFault installs (or clears, with nil) the Register C fault
// injector for tests in other packages; production never sets one.
func SetFault(f func(point string) error) { fault = f }

// recordRefLine renders the record-reference furniture appended after
// a framed result: the witnessing event's seq and the object's content
// address — exactly the `<seq>:<id>` an evidence argument names.
func recordRefLine(seq int64, objectID string) string {
	return fmt.Sprintf("record-ref: %d:%s\n", seq, objectID)
}

// FaultAt exposes the Register C fault-injection seam to the delegation
// implementation for its three new points (delegation.pre-composition-
// store, delegation.pre-output-store, delegation.pre-event-commit).
func FaultAt(point string) error { return faultAt(point) }

// delegationInstantiator adapts the seam's compose half to L4's
// executor injection (tools.DelegationInstantiator), bound per task at
// assembly. It maps the seam's refusal onto the closed L4 class and
// nothing else.
type delegationInstantiator struct {
	d    Delegator
	base InstantiationRequest
}

func (a delegationInstantiator) Instantiate(template string, evidence []tools.EvidenceRef, brief string) ([]byte, error) {
	req := a.base
	req.Template, req.Brief = template, brief
	for _, e := range evidence {
		req.Evidence = append(req.Evidence, EvidenceRef{Seq: e.Seq, ObjectID: e.ObjectID})
	}
	capture, err := a.d.Instantiate(req)
	if err != nil {
		var r *DelegationRefusal
		if errors.As(err, &r) {
			return nil, &tools.ErrDelegationRefusal{Reason: tools.DelegationRefusalReason(r.Reason), Detail: r.Detail}
		}
		return nil, fmt.Errorf("delegation instantiation: %w", err)
	}
	return capture, nil
}

// isDelegation is a registry classification, never an authorization
// branch: the call already passed the identical L4 gate.
func (w *walk) isDelegation(name string) bool {
	for i := range w.reg.Tools {
		if w.reg.Tools[i].Name == name {
			return w.reg.Tools[i].Target == tools.TargetDelegationTemplate
		}
	}
	return false
}

// delegate runs the post-hook for one AUTHORIZED, INSTANTIATED delegate
// call whose l4-audit (at parentCallSeq) has committed, and returns the
// paired tool-role message. It appends exactly one message, calls no
// δ step, and interprets nothing: a completed outcome re-enters framed
// under the capability's registered trust with the output object's
// identity; every other outcome re-enters as the closed typed error,
// unframed (C-L8-11).
func (w *walk) delegate(call model.ToolCall, parentCallSeq int64, capture []byte) (model.Message, error) {
	if w.o.cfg.Delegator == nil {
		return model.Message{}, fmt.Errorf("%w: delegation call %q with no delegator wired", ErrInvariant, call.Name)
	}
	var args struct {
		Template string `json:"template"`
		Evidence string `json:"evidence"`
		Brief    string `json:"brief"`
	}
	if err := json.Unmarshal(call.Arguments, &args); err != nil {
		return model.Message{}, fmt.Errorf("%w: authorized delegate arguments unreadable: %v", ErrInvariant, err)
	}
	refs, err := tools.ParseEvidenceRefs(args.Evidence)
	if err != nil {
		return model.Message{}, fmt.Errorf("%w: authorized delegate evidence unreadable: %v", ErrInvariant, err)
	}
	req := DelegationRequest{
		InstantiationRequest: w.instBase,
		Task:                 w.task, ParentCallSeq: parentCallSeq, CallID: call.ID, Capture: capture,
		Model: w.env.Model, Runtime: w.o.cfg.Model, ModelRegistryHash: w.o.modelRegistryHash(),
		TurnTimeout: time.Duration(w.env.TurnTimeoutSec) * time.Second, Deadline: w.deadline,
	}
	req.Template, req.Brief = args.Template, args.Brief
	for _, r := range refs {
		req.Evidence = append(req.Evidence, EvidenceRef{Seq: r.Seq, ObjectID: r.ObjectID})
	}
	res, err := w.o.cfg.Delegator.Delegate(req)
	if err != nil {
		// Stage D: no outcome minted, the invariant path.
		return model.Message{}, fmt.Errorf("%w: delegation machinery: %v", ErrInvariant, err)
	}
	if res.Seq > w.lastSeq {
		w.lastSeq = res.Seq
	}
	var content string
	switch res.Outcome {
	case "completed":
		trust := ""
		for i := range w.reg.Tools {
			if w.reg.Tools[i].Name == call.Name {
				trust = string(w.reg.Tools[i].Trust)
			}
		}
		hash := res.OutputObjectID
		if len(hash) > len("sha256:") && hash[:7] == "sha256:" {
			hash = hash[7:]
		}
		content = tools.FrameToolResult(call.Name, trust, hash, res.Output) + recordRefLine(res.Seq, res.OutputObjectID)
	case "provider-error":
		content = `{"error":"` + string(tools.ErrDelegationProviderError) + `"}`
	case "output-over-bound":
		content = `{"error":"` + string(tools.ErrDelegationOutputOverBound) + `"}`
	case "model-identity-mismatch":
		content = `{"error":"` + string(tools.ErrDelegationModelIdentityMismatch) + `"}`
	default:
		return model.Message{}, fmt.Errorf("%w: unknown delegation outcome %q", ErrInvariant, res.Outcome)
	}
	return model.Message{Role: model.RoleTool, ToolCallID: call.ID, Content: content}, nil
}
