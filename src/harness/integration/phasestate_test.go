package integration

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tofchaliss/themis-ai-runtime/src/harness/deployment"
	"github.com/tofchaliss/themis-ai-runtime/src/harness/orchestration"
	"github.com/tofchaliss/themis-ai-runtime/src/harness/skills"
	"github.com/tofchaliss/themis-ai-runtime/src/harness/state"
)

// D-P-1/D-P-2 over a real walk of remediate-dependency@6: every phase
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
	ceiling, err := orchestration.LoadWorkflowCeiling(filepath.Join(w.root, "policies/skills/remediate-dependency-6/ceiling.json"))
	if err != nil {
		t.Fatal(err)
	}
	wf, err := orchestration.LoadWorkflow(filepath.Join(w.root, "policies/skills/remediate-dependency-6/workflow.json"), ceiling)
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
		if !bytes.Contains(payload, []byte(fmt.Sprintf("version: seq:%d", entrySeq))) {
			t.Fatalf("seq %d: the delivered fact must cite the record-derived provenance seq %d", ev.Seq, entrySeq)
		}
		if entrySeq >= ev.Seq {
			t.Fatalf("seq %d: provenance %d must precede the delivery", ev.Seq, entrySeq)
		}
		// L4 twin (D-P-2): every tool L4 authorized in this phase is one
		// the delivered fact listed — what the model read equals what L4
		// let through.
		listed := map[string]bool{}
		for _, c := range caps {
			listed[c] = true
		}
		for _, later := range evs[i+1:] {
			if later.Class == state.EvL2Delivery {
				break
			}
			if later.Class != state.EvL4Audit {
				continue
			}
			var ab struct {
				Tool, Decision string
			}
			_ = json.Unmarshal(later.Body, &ab)
			if ab.Decision == "authorized" && !listed[ab.Tool] {
				t.Fatalf("seq %d: L4 authorized %q in %s but the phase fact listed %v", later.Seq, ab.Tool, body.Phase, caps)
			}
		}
	}
	if deliveries < 2 || phases[0] != "ANALYZE" || phases[1] != "REMEDIATE" {
		t.Fatalf("expected ANALYZE then REMEDIATE deliveries, got %v", phases)
	}
}

// D-P-4: "registry = history, allowlist = only what composes" has a
// deterministic witness — a REAL pre-D-P-1 catalog skill
// (remediate-dependency@4, still active in the catalog) admitted by an
// anchor still refuses at composition because its contract lacks the
// slot; the record is sealed FAILED with no model turn.
func TestPreFramingCatalogSkillRefusesAtComposition(t *testing.T) {
	root := tdRepoRoot(t)
	hf := func(p string) string {
		h, err := deployment.HashFile(filepath.Join(root, "policies/skills/remediate-dependency-4", p))
		if err != nil {
			t.Fatal(err)
		}
		return h
	}
	w, err := newWorld(t, &readingParent{finding: demoFindingID}, worldOpts{withDoor: true, serveFinding: true}, func(a map[string]any) {
		// @4's bundle replaces @6's: the workflow bytes are identical and an
		// anchor admits one bundle per workflow.
		a["skills"] = []any{"remediate-dependency@4"}
		a["workflows"] = []any{map[string]any{
			"workflow": hf("workflow.json"), "workflow_ceiling": hf("ceiling.json"), "context_contract": hf("contract.json")}}
	})
	if err != nil {
		t.Fatal(err)
	}
	_ = os.MkdirAll(filepath.Join(w.base, "state"), 0o755)
	path, err := skills.Instantiate(filepath.Join(w.root, "policies/skills/catalog.json"), "remediate-dependency@4", skills.Request{
		TaskID: "t-rd4", Repo: "demo-vuln-app", PinnedSHA: w.sha,
		Inputs:        map[string]any{"finding": demoFindingID, "dependency": "vulnerable-dep", "advisory": "ADV-2026-1"},
		WallDeadlineS: 300,
		Deployment: skills.Deployment{Model: "scripted", TurnTimeoutSec: 180,
			RegistryPath: filepath.Join(w.root, "policies/tools/registry-v5.json"), ExecCeilingPath: filepath.Join(w.base, "execution-ceiling.json"),
			StateRoot: filepath.Join(w.base, "state"), ArtifactDir: filepath.Join(w.base, "artifacts"), WorkspaceRoot: filepath.Join(w.base, "provider")},
		OutDir: filepath.Join(w.base, "envelopes"),
	})
	if err != nil {
		t.Fatalf("L9 still instantiates @4 (it is a registered composition): %v", err)
	}
	_, err = w.o.SubmitTask(path)
	if err == nil || !strings.Contains(err.Error(), `no slot "phase-state"`) {
		t.Fatalf("@4 must refuse at composition by name: %v", err)
	}
	evs, _ := w.sroot.ReadEvents("t-rd4")
	for _, ev := range evs {
		if ev.Class == state.EvModelTurn {
			t.Fatal("a refused composition must not reach the model")
		}
	}
}
