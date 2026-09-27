package integration

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/tofchaliss/themis-ai-runtime/src/harness/orchestration"
	"github.com/tofchaliss/themis-ai-runtime/src/harness/state"
)

// D-P-1/D-P-2 over a real walk of remediate-dependency@5: every phase
// entry's recorded l2-delivery payload contains, byte-exact, the
// phase-state re-derived from the record as it stood at that seq; the
// capabilities equal the narrowed grant; `completed` is the ordered
// chain. Reconstruction never needs the loop's memory.
func TestPhaseStateIsDerivedFromTheRecordByteExact(t *testing.T) {
	w, _ := walk(t, &readingParent{finding: demoFindingID}, "t-phase-state")
	evs, err := w.sroot.ReadEvents("t-phase-state")
	if err != nil {
		t.Fatal(err)
	}
	ceiling, err := orchestration.LoadWorkflowCeiling(filepath.Join(w.root, "policies/skills/remediate-dependency-5/ceiling.json"))
	if err != nil {
		t.Fatal(err)
	}
	wf, err := orchestration.LoadWorkflow(filepath.Join(w.root, "policies/skills/remediate-dependency-5/workflow.json"), ceiling)
	if err != nil {
		t.Fatal(err)
	}
	gb, err := os.ReadFile(filepath.Join(w.base, "envelopes", "t-phase-state-grant.json"))
	if err != nil {
		t.Fatal(err)
	}
	var grant struct {
		Entries []struct {
			Tool string `json:"tool"`
		} `json:"entries"`
	}
	if err := json.Unmarshal(gb, &grant); err != nil {
		t.Fatal(err)
	}
	deliveries := 0
	var phases []string
	for i, ev := range evs {
		if ev.Class != state.EvL2Delivery {
			continue
		}
		var body struct {
			Phase   string `json:"phase"`
			Payload string `json:"payload"`
		}
		if json.Unmarshal(ev.Body, &body) != nil || body.Phase == "" || body.Payload == "" {
			continue
		}
		deliveries++
		phases = append(phases, body.Phase)
		var caps []string
		for _, p := range wf.Phases {
			if p.Name != body.Phase {
				continue
			}
			in := map[string]bool{}
			for _, c := range p.Capabilities {
				in[c] = true
			}
			for _, e := range grant.Entries {
				if in[e.Tool] {
					caps = append(caps, e.Tool)
				}
			}
		}
		_, want, entrySeq, err := orchestration.DerivePhaseState(wf, evs[:i], body.Phase, caps)
		if err != nil {
			t.Fatalf("seq %d (%s): %v", ev.Seq, body.Phase, err)
		}
		payload, err := w.sroot.Store().GetObject(body.Payload)
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Contains(payload, want) {
			t.Fatalf("seq %d (%s): delivered payload lacks the record-derived phase-state %s", ev.Seq, body.Phase, want)
		}
		if !bytes.Contains(payload, []byte("authority: derived")) || !bytes.Contains(payload, []byte("kind: phase-state")) {
			t.Fatalf("seq %d: phase-state must be fenced with its derived provenance labels", ev.Seq)
		}
		if entrySeq >= ev.Seq {
			t.Fatalf("seq %d: provenance %d must precede the delivery", ev.Seq, entrySeq)
		}
	}
	if deliveries < 2 || phases[0] != "ANALYZE" || phases[1] != "REMEDIATE" {
		t.Fatalf("expected ANALYZE then REMEDIATE deliveries, got %v", phases)
	}
}
