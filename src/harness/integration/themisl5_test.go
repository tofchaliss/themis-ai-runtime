package integration

// L5 witnesses over a REAL walk (W-M2 Register B) and the walk helpers
// the Themis fixture generator uses. Moved from the dissolved
// src/themis stand-in (I-M4); the intake/admissibility tests that used
// the stand-in's Resolve now live in the Themis repository
// (internal/governance/adapters/harness) over the fixtures generated
// here.

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/tofchaliss/themis-ai-runtime/src/harness/runtime/model"
	"github.com/tofchaliss/themis-ai-runtime/src/harness/state"
)

type tuple struct {
	AnchorHash       string
	TaskID           string
	ArtifactBoundSeq int64
}

const reportB = `{"finding": "vulnerable-dep v1 in go.mod (ADV-2026-1)", "remediation": "bump to v2 (revised after verification)", "evidence": "go.mod updated; tests green"}`

func writeCall(id, path, content string) model.ToolCall {
	args, _ := json.Marshal(map[string]string{"path": path, "content": content})
	return model.ToolCall{ID: id, Name: "write_file", Arguments: args}
}

func verifyCall(id string) model.ToolCall {
	return model.ToolCall{ID: id, Name: "verify_report", Arguments: json.RawMessage(`{"path":"report.json","contract":"report-valid@2"}`)}
}

// walk runs remediate-dependency@3 under an anchored world with the
// given scripted parent and returns the world, the task result, and
// the tuple Themis would be handed (anchor from the deployment, task
// id, the artifact-bound seq found in the record).
func walk(t *testing.T, m *readingParent, task string) (*world, tuple) {
	t.Helper()
	w, err := newWorld(t, m, worldOpts{withDoor: true, serveFinding: true}, nil)
	if err != nil {
		t.Fatal(err)
	}
	res, err := w.o.SubmitTask(w.instantiate(t, task, demoFindingID))
	if err != nil {
		t.Fatalf("walk: %v", err)
	}
	tp := tuple{AnchorHash: w.anchorSHA, TaskID: task}
	if res.Status == state.StatusCompleted {
		tp.ArtifactBoundSeq = bindingSeq(t, w, task)
	}
	return w, tp
}

func bindingSeq(t *testing.T, w *world, task string) int64 {
	t.Helper()
	evs, err := w.sroot.ReadEvents(task)
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range evs {
		if e.Class == state.EvArtifact {
			return e.Seq
		}
	}
	t.Fatal("no artifact-bound in the record")
	return 0
}

func recordedOutcomes(t *testing.T, w *world, task string) []string {
	t.Helper()
	evs, _ := w.sroot.ReadEvents(task)
	var out []string
	for _, e := range evs {
		if e.Class == state.EvVerification {
			var b struct {
				Outcome string `json:"outcome"`
			}
			_ = json.Unmarshal(e.Body, &b)
			out = append(out, b.Outcome)
		}
	}
	return out
}

// ---- W-M2 Register B: a REAL walk's stream carries the full L5 machine
// and every governed op, written by l5 through the handle, in the
// order D-W-2/D-W-3 fix, with the egress acknowledgement naming the
// address the L6 binding then names (same digest, "sha256:" prefixed —
// the RawObjectID precedent) and every L5 witness preceding the
// binding and COMPLETED.
func TestRealWalkCarriesL5Witnesses(t *testing.T) {
	w, tuple := walk(t, &readingParent{finding: demoFindingID}, "t-l5-witness")
	evs, err := w.sroot.ReadEvents(tuple.TaskID)
	if err != nil {
		t.Fatal(err)
	}
	var edges []string
	var egressAck struct {
		seq  int64
		addr string
	}
	ops, provisionOps, activeOps, egressOps := 0, 0, 0, 0
	var boundSeq, completedSeq int64
	var boundAddr string
	for _, e := range evs {
		switch e.Class {
		case state.EvL5Transition, state.EvL5Op:
			if e.Writer != "l5" {
				t.Fatalf("seq %d %s writer %q", e.Seq, e.Class, e.Writer)
			}
		}
		switch e.Class {
		case state.EvL5Transition:
			var b struct{ From, To string }
			_ = json.Unmarshal(e.Body, &b)
			edges = append(edges, b.From+">"+b.To)
		case state.EvL5Op:
			var b struct {
				Op, Phase, Outcome, Address string `json:"-"`
			}
			var m map[string]any
			_ = json.Unmarshal(e.Body, &m)
			if m["op"] == "egress" {
				if m["outcome"] == "acknowledged" {
					egressAck.seq, egressAck.addr = e.Seq, m["artifact_address"].(string)
				}
				continue
			}
			ops++
			switch m["phase"] {
			case "provision":
				provisionOps++
			case "active":
				activeOps++
			case "egress":
				egressOps++
			}
			_ = b
		case state.EvArtifact:
			boundSeq, boundAddr = e.Seq, e.Refs[0].ID
		case state.EvLifecycle:
			var lb struct{ To string }
			_ = json.Unmarshal(e.Body, &lb)
			if lb.To == string(state.StatusCompleted) {
				completedSeq = e.Seq
			}
		}
	}
	// The full machine, in order (teardown edges follow the artifact
	// binding but precede COMPLETED — the walk tears down before it
	// transitions the task).
	want := []string{"PROVISIONING>ACTIVE", "ACTIVE>SEALED", "SEALED>EGRESSING", "EGRESSING>ACKNOWLEDGED", "ACKNOWLEDGED>TEARDOWN", "TEARDOWN>DESTROYED"}
	if strings.Join(edges, "|") != strings.Join(want, "|") {
		t.Fatalf("L5 edges: %v", edges)
	}
	// Provisioning and egress always run governed git ops; the ACTIVE
	// phase of this workflow uses in-process file tools only, so it may
	// legitimately witness zero subprocess ops — a fact about the
	// skill, not a gap (D-W-3 witnesses subprocesses, not tool calls).
	if provisionOps == 0 || egressOps == 0 {
		t.Fatalf("ops witnessed: provision=%d active=%d egress=%d (total %d)", provisionOps, activeOps, egressOps, ops)
	}
	if egressAck.seq == 0 {
		t.Fatal("no acknowledged egress witness")
	}
	if "sha256:"+egressAck.addr != boundAddr {
		t.Fatalf("egress acknowledged %s, binding names %s", egressAck.addr, boundAddr)
	}
	if !(egressAck.seq < boundSeq && boundSeq < completedSeq) {
		t.Fatalf("ordering: egress ack %d, bound %d, completed %d", egressAck.seq, boundSeq, completedSeq)
	}
	// The five links D-W-5 will replay all exist in this record.
	if !strings.Contains(strings.Join(edges, "|"), "ACTIVE>SEALED") {
		t.Fatal("seal edge missing")
	}
}
