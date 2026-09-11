package seam

// M5 end-to-end: reconstruction over a real L6 root — a committed
// evaluation reconstructs consistent; a tampered-outcome record
// yields a discrepancy artifact stored OUTSIDE the task stream with
// the original record byte-identical (no retroactive mutation).

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/tofchaliss/themis/state"
)

func seedTask(t *testing.T, tamperOutcome bool) (*state.Root, string) {
	t.Helper()
	root, err := state.OpenRoot(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	task, err := root.CreateTask("t-recon", state.TaskOptions{})
	if err != nil {
		t.Fatal(err)
	}

	e := proposedEvaluator(t)
	// Seed the committed L4 audit the evaluation references, as the
	// loop would have (the M-1 cross-check resolves it).
	auditBody, _ := json.Marshal(map[string]string{
		"Tool": "verify_report", "RegistryHash": e.L4.Hash})
	aev, err := task.AppendEvent(state.EvL4Audit, "l4", auditBody)
	if err != nil {
		t.Fatal(err)
	}
	vo, err := e.EvaluateCall("t-recon", verifyCall("report-valid@1", "r.json"), []byte(goodReport), fmt.Sprintf("l4:%d", aev.Seq), e.L4.Hash)
	if err != nil || vo.Refused {
		t.Fatalf("%v %+v", err, vo)
	}
	if tamperOutcome {
		// A record whose outcome field was forged after evaluation:
		// reconstruction must expose it via mapping recomputation.
		var ev map[string]any
		_ = json.Unmarshal(vo.Record, &ev)
		ev["outcome"] = "FAIL"
		vo.Record, _ = json.Marshal(ev)
	}

	// The loop's stage-5 discipline, reproduced: stores, then the
	// event whose body NAMES the record object (H-1 discipline).
	var refs []state.Ref
	var recordID string
	for _, b := range [][]byte{vo.ContractBytes, vo.RawBytes, vo.CanonicalBytes, vo.Record} {
		id, oerr := task.StoreObject(state.ObjEvidencePayload, b)
		if oerr != nil {
			t.Fatal(oerr)
		}
		refs = append(refs, state.Ref{ID: id, Class: state.ObjEvidencePayload})
		recordID = id // last stored = the record
	}
	body, _ := json.Marshal(map[string]string{"contract": vo.ContractToken, "outcome": vo.Outcome, "record": recordID})
	if _, err := task.AppendEvent(state.EvVerification, "l10", body, refs...); err != nil {
		t.Fatal(err)
	}
	return root, "t-recon"
}

func TestReconstructTaskConsistent(t *testing.T) {
	root, id := seedTask(t, false)
	reports, artifacts, err := ReconstructTask(root, id)
	if err != nil {
		t.Fatal(err)
	}
	if len(reports) != 1 || !reports[0].Consistent {
		t.Fatalf("expected one consistent report: %+v", reports)
	}
	if len(artifacts) != 0 {
		t.Error("consistent reconstruction must store no discrepancy artifact")
	}

	view, err := TaskVerificationHistory(root, id)
	if err != nil {
		t.Fatal(err)
	}
	if view.Latest["report-valid@1"] != "PASS" {
		t.Errorf("history view: %v", view.Latest)
	}
}

func TestReconstructTaskDiscrepancy(t *testing.T) {
	root, id := seedTask(t, true)

	before, err := root.ReadEvents(id)
	if err != nil {
		t.Fatal(err)
	}

	reports, artifacts, err := ReconstructTask(root, id)
	if err != nil {
		t.Fatal(err)
	}
	if len(reports) != 1 || reports[0].Consistent {
		t.Fatal("forged outcome reconstructed consistent")
	}
	if len(artifacts) != 1 {
		t.Fatalf("expected one discrepancy artifact, got %d", len(artifacts))
	}
	// The artifact is durable, content-addressed, and retrievable.
	b, err := root.Store().GetObject(artifacts[0])
	if err != nil {
		t.Fatal(err)
	}
	if !json.Valid(b) {
		t.Error("discrepancy artifact must be typed JSON")
	}

	// No retroactive mutation: the task event stream is byte-identical
	// after reconstruction — same count, same hashes.
	after, err := root.ReadEvents(id)
	if err != nil {
		t.Fatal(err)
	}
	if len(after) != len(before) {
		t.Fatalf("reconstruction changed the task stream: %d -> %d events", len(before), len(after))
	}
	for i := range after {
		if after[i].BodyHash != before[i].BodyHash || after[i].Seq != before[i].Seq {
			t.Fatal("reconstruction mutated historical events")
		}
	}
}
