package orchestration

// M0b supporting-layer prerequisites (openspec/changes/l8-subagents
// §5.3): F-L8-2 execution identity on model-turn, F-L8-4 the parent
// turn deadline is min(turn timeout, remaining wall budget). F-L8-3 is
// asserted in TestVerificationRefusalIsNotAnOutcome.

import (
	stdctx "context"
	"encoding/json"
	"testing"
	"time"

	"github.com/tofchaliss/themis/runtime/model"
	"github.com/tofchaliss/themis/state"
)

// identityModel decorates a scripted model with a provider-shaped
// identity and endpoint, as a real adapter reports them.
type identityModel struct{ inner model.Interface }

func (m identityModel) Name() string { return "scripted" }
func (m identityModel) Execute(ctx stdctx.Context, req model.ExecutionRequest) (*model.ExecutionResponse, error) {
	resp, err := m.inner.Execute(ctx, req)
	if err != nil {
		return nil, err
	}
	resp.Identity = model.Identity{WireModel: req.Model, Runtime: "scripted", Reported: "scripted-build-7"}
	resp.Provenance.Endpoint = "scripted://local"
	return resp, nil
}

func TestModelTurnRecordsExecutionIdentity(t *testing.T) {
	f := setup(t, identityModel{happyScript()}, "")
	res, err := f.o.SubmitTask(f.envelope(t, "t-ident"))
	if err != nil || res.Status != state.StatusCompleted {
		t.Fatalf("%+v %v", res, err)
	}
	evs, _ := f.o.root.ReadEvents("t-ident")
	turns := 0
	for _, e := range evs {
		if e.Class != state.EvModelTurn {
			continue
		}
		var body struct {
			Fact     string         `json:"fact"`
			Model    string         `json:"model"`
			Identity model.Identity `json:"identity"`
			Endpoint string         `json:"endpoint"`
		}
		if err := json.Unmarshal(e.Body, &body); err != nil {
			t.Fatal(err)
		}
		if body.Fact == "provider-error" {
			continue
		}
		turns++
		if body.Identity.WireModel != body.Model || body.Identity.Runtime != "scripted" || body.Identity.Reported != "scripted-build-7" || body.Endpoint != "scripted://local" {
			t.Fatalf("execution identity must be recorded beside the governed name: %s", e.Body)
		}
	}
	if turns == 0 {
		t.Fatal("no model turns recorded")
	}
}

// blockingModel never answers: it returns only when its context ends.
type blockingModel struct{}

func (blockingModel) Name() string { return "blocking" }
func (blockingModel) Execute(ctx stdctx.Context, req model.ExecutionRequest) (*model.ExecutionResponse, error) {
	<-ctx.Done()
	return nil, ctx.Err()
}

func TestParentTurnDeadlineIsMinOfTurnAndWall(t *testing.T) {
	f := setup(t, blockingModel{}, "")
	// A 2-second wall budget under a 180-second turn timeout: the turn
	// must be cut by the wall budget, and the walk must reach the
	// constitution floor in seconds, not minutes.
	writeJSON(t, f.envDir, "spec.json",
		`{"version":1,"task_id":"T","repo":"demo","pinned_sha":"`+f.sha+`","limits":[{"dimension":"wall_deadline_s","value":2}]}`)
	start := time.Now()
	res, err := f.o.SubmitTask(f.envelope(t, "t-deadline"))
	elapsed := time.Since(start)
	if err != nil {
		t.Fatal(err)
	}
	// Whether δ's turn-provider-error edge or the constitution floor
	// terminates the walk is a race the property does not depend on:
	// the property is that the blocked turn was CUT by the 2-second
	// wall budget, so the walk terminated in seconds, not after the
	// 180-second turn timeout.
	if res.Status != state.StatusFailed {
		t.Fatalf("expected a failed walk: %+v", res)
	}
	if elapsed > 20*time.Second {
		t.Fatalf("the turn outlived the wall budget: %s", elapsed)
	}
}
