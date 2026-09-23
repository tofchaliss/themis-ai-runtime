package orchestration

// Conversation projection after a delegation (C-L8-18 A, D-L7-11): the
// model-visible tool message is a deterministic function of the record
// — the l8-delegation witness plus its output object, or the l4-audit
// for a refusal — never a stored transcript. Read-only.

import (
	"encoding/json"
	"fmt"

	"github.com/tofchaliss/themis/state"
	"github.com/tofchaliss/themis/tools"
)

// ProjectDelegationMessage re-derives the paired tool message for the
// delegation witnessed at seq. trust is the delegate capability's
// registered trust in the registry that authorized the call.
func ProjectDelegationMessage(root *state.Root, taskID string, seq int64, trust string) (string, error) {
	events, err := root.ReadEvents(taskID)
	if err != nil {
		return "", err
	}
	for _, ev := range events {
		if ev.Seq != seq || ev.Class != state.EvL8Delegation {
			continue
		}
		var body struct {
			Outcome         string `json:"outcome"`
			OutputObjectRef string `json:"output_object_ref"`
		}
		if err := json.Unmarshal(ev.Body, &body); err != nil {
			return "", err
		}
		switch body.Outcome {
		case "completed":
			out, err := root.Store().GetObject(body.OutputObjectRef)
			if err != nil {
				return "", err
			}
			hash := body.OutputObjectRef
			if len(hash) > 7 && hash[:7] == "sha256:" {
				hash = hash[7:]
			}
			return tools.FrameToolResult("delegate", trust, hash, out) + recordRefLine(seq, body.OutputObjectRef), nil
		case "provider-error":
			return `{"error":"` + string(tools.ErrDelegationProviderError) + `"}`, nil
		case "output-over-bound":
			return `{"error":"` + string(tools.ErrDelegationOutputOverBound) + `"}`, nil
		case "model-identity-mismatch":
			return `{"error":"` + string(tools.ErrDelegationModelIdentityMismatch) + `"}`, nil
		}
		return "", fmt.Errorf("unknown outcome %q", body.Outcome)
	}
	return "", fmt.Errorf("no l8-delegation at seq %d", seq)
}

// ProjectRefusalMessage re-derives the tool message for an l4-audit
// that recorded an execution error (a stage-B delegation refusal, or
// any tool error): the closed class, unframed.
func ProjectRefusalMessage(audit state.Event) (string, error) {
	var body struct {
		Decision string
		ErrClass string
	}
	if err := json.Unmarshal(audit.Body, &body); err != nil {
		return "", err
	}
	if body.Decision != "error" || !tools.KnownErrorClass(tools.ErrorClass(body.ErrClass)) {
		return "", fmt.Errorf("audit %d is not an execution error of the closed vocabulary", audit.Seq)
	}
	return `{"error":"` + body.ErrClass + `"}`, nil
}

// ExecutionBound is the static boundedness statement (Register D,
// C-L8-19 I) computed at assembly from loaded artifacts: parent model
// turns are bounded by the workflow's worst-case walk (≤ ceiling
// max_walk_length); delegations by min(delegate max_calls, grant
// total) (≤ ceiling max_total_calls); every model execution is one or
// the other. L8 subdivides the capability budget, never multiplies it.
type ExecutionBound struct {
	Turns       int64 // ≤ W
	Delegations int64 // min(D, G) ≤ M
	Executions  int64 // Turns + Delegations ≤ W + M
	WalkCeiling int64 // W
	CallCeiling int64 // M
}

func executionBound(wf *WorkflowDef, c *WorkflowCeiling, g *tools.Grant, reg *tools.Registry) ExecutionBound {
	b := ExecutionBound{Turns: wf.WorstCaseLen, WalkCeiling: c.MaxWalkLength, CallCeiling: int64(c.MaxTotalCalls)}
	var d int64
	for _, e := range g.Entries {
		for _, t := range reg.Tools {
			if t.Name == e.Tool && t.Target == tools.TargetDelegationTemplate {
				d += int64(e.MaxCalls)
			}
		}
	}
	if int64(g.TotalMaxCalls) < d {
		d = int64(g.TotalMaxCalls)
	}
	b.Delegations = d
	b.Executions = b.Turns + b.Delegations
	return b
}
