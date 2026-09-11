package orchestration

// L10-M6 Register R proofs over gate-bearing walks through the
// production loop with a scripted model and a scripted evaluator:
// FAIL→PASS remediation, the no-verification non-completion
// invariant, the PASS→UNAVAILABLE downgrade, assembly refusals, and
// causal replay over every walk (the extended Register D).

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tofchaliss/themis/runtime/model"
	"github.com/tofchaliss/themis/state"
)

// scriptedEvaluator returns pre-scripted verification outcomes in
// call order — the L10 seam's shape without its machinery, so these
// proofs pin the L7 side deterministically.
type scriptedEvaluator struct {
	outcomes []string
	i        int
}

func (s *scriptedEvaluator) EvaluateCall(taskID string, call model.ToolCall, evidence []byte, execRef string) (*VerificationOutcome, error) {
	if s.i >= len(s.outcomes) {
		return &VerificationOutcome{Refused: true, RefusalReason: "script exhausted"}, nil
	}
	out := s.outcomes[s.i]
	s.i++
	record, _ := json.Marshal(map[string]string{"outcome": out, "scripted": "true"})
	return &VerificationOutcome{
		ContractToken:  "report-valid@1",
		Outcome:        out,
		Record:         record,
		ContractBytes:  []byte(`{"scripted-contract": true}`),
		RawBytes:       evidence,
		CanonicalBytes: []byte("scripted"),
	}, nil
}

const verifWalkWorkflow = `{
 "version":1,"name":"remediate-verify","initial":"WORK",
 "declared_events":["turn-no-action","turn-provider-error","turns-exhausted","tool-error",
   "signal:phase-completion-requested",
   "verification-pass","verification-fail","verification-inconclusive",
   "verification-unavailable","verification-invalid"],
 "phases":[
  {"name":"WORK","capabilities":["read_file","verify_report","declare_done"],"max_model_turns":8,"edges":[
    {"on":"signal:phase-completion-requested",
     "gate":{"contract":"report-valid@1","outcome":"PASS"},"to":"@complete"},
    {"on":"signal:phase-completion-requested","to":"@stay","counter":3,"exhausted_to":"@fail"},
    {"on":"verification-pass","to":"@stay","counter":3,"exhausted_to":"@fail"},
    {"on":"verification-fail","to":"@stay","counter":3,"exhausted_to":"@fail"},
    {"on":"verification-inconclusive","to":"@stay","counter":3,"exhausted_to":"@fail"},
    {"on":"verification-unavailable","to":"@stay","counter":3,"exhausted_to":"@fail"},
    {"on":"verification-invalid","to":"@fail"},
    {"on":"turn-no-action","to":"@stay","counter":3,"exhausted_to":"@fail"},
    {"on":"turn-provider-error","to":"@fail"},
    {"on":"tool-error","to":"@stay","counter":3,"exhausted_to":"@fail"},
    {"on":"turns-exhausted","to":"@fail"}]}
 ]}`

const verifWalkCeiling = `{"version":1,"allowed_tools":["read_file","verify_report","declare_done"],"max_total_calls":30,"max_walk_length":200,"max_turns_per_phase":10}`

func setupVerif(t *testing.T, m model.Interface, ev VerificationEvaluator) *fixture {
	t.Helper()
	f := &fixture{envDir: t.TempDir(), stateDir: filepath.Join(t.TempDir(), "state")}
	f.mirror, f.sha = mkMirror(t)
	o, rep, err := Open(Config{
		StateRoot: f.stateDir, ArtifactDir: filepath.Join(t.TempDir(), "artifacts"),
		GitPath: gitBin(t), ProviderDir: t.TempDir(),
		SafetyRoot: filepath.Join(repoRoot, "instructions/global/safety"),
		SystemRoot: filepath.Join(repoRoot, "instructions/global/system"),
		PolicyPath: filepath.Join(repoRoot, "policies/security/instruction-directive-patterns.json"),
		Model:      m,
		Verifier:   ev,
	})
	if err != nil {
		t.Fatal(err)
	}
	f.o, f.rep = o, rep
	writeJSON(t, f.envDir, "workflow.json", verifWalkWorkflow)
	writeJSON(t, f.envDir, "wceiling.json", verifWalkCeiling)
	writeJSON(t, f.envDir, "eceiling.json",
		`{"version":1,"mirror_root":"`+f.mirror+`","max_wall_deadline_sec":600,"max_file_bytes":1048576,"max_total_bytes":10485760,"max_file_count":500,"max_mem_bytes":1073741824,"max_cpu_time_sec":600,"max_proc_count":64}`)
	writeJSON(t, f.envDir, "spec.json",
		`{"version":1,"task_id":"T","repo":"demo","pinned_sha":"`+f.sha+`","limits":[{"dimension":"wall_deadline_s","value":90}]}`)
	writeJSON(t, f.envDir, "grant.json",
		`{"version":1,"task_id":"T","total_max_calls":30,"entries":[
		  {"tool":"read_file","max_calls":8,"workspace":"@workspace"},
		  {"tool":"verify_report","max_calls":6,"workspace":"@workspace"},
		  {"tool":"declare_done","max_calls":6}]}`)
	writeJSON(t, f.envDir, "context-contract.json",
		`{"version":1,"workflow":"remediate-verify","slots":[
		  {"name":"task-payload","kind":"task-brief","requirement":"required","classes":["external-untrusted"]}],
		  "sensitivity_ceiling":"public"}`)
	return f
}

func (f *fixture) verifEnvelope(t *testing.T, taskID string) string {
	t.Helper()
	abs := func(n string) string { return filepath.Join(f.envDir, n) }
	for _, tmpl := range []string{"grant.json", "spec.json"} {
		body := readFile(t, abs(tmpl))
		writeJSON(t, f.envDir, taskID+"-"+tmpl, strings.Replace(body, `"task_id":"T"`, `"task_id":"`+taskID+`"`, 1))
	}
	return writeJSON(t, f.envDir, "envelope-"+taskID+".json", `{
	 "version":1,"task_id":"`+taskID+`","model":"scripted","turn_timeout_sec":180,
	 "payload":"Remediate, verify the report, then declare done.",
	 "workflow_path":`+jstr(abs("workflow.json"))+`,"workflow_ceiling_path":`+jstr(abs("wceiling.json"))+`,
	 "registry_path":`+jstr(mustAbs(t, filepath.Join(repoRoot, "policies/tools/registry-v4.proposed.json")))+`,
	 "grant_path":`+jstr(abs(taskID+"-grant.json"))+`,"exec_ceiling_path":`+jstr(abs("eceiling.json"))+`,
	 "spec_path":`+jstr(abs(taskID+"-spec.json"))+`,
	 "context_contract_path":`+jstr(abs("context-contract.json"))+`}`)
}

func verifyCallStep() model.ExecutionResponse {
	return toolCall("verify_report", `{"path":"parser.go","contract":"report-valid@1"}`)
}

// TestGateWalkRemediation: FAIL closes the gate, PASS opens it — the
// remediation loop through the unmodified production loop, causally
// replayed.
func TestGateWalkRemediation(t *testing.T) {
	m := &scriptedModel{steps: []model.ExecutionResponse{
		verifyCallStep(),               // -> FAIL
		toolCall("declare_done", `{}`), // gate closed -> fallback @stay
		verifyCallStep(),               // -> PASS
		toolCall("declare_done", `{}`), // gate open -> @complete
	}}
	ev := &scriptedEvaluator{outcomes: []string{"FAIL", "PASS"}}
	f := setupVerif(t, m, ev)

	res, err := f.o.SubmitTask(f.verifEnvelope(t, "verif-remediate"))
	if err != nil {
		t.Fatal(err)
	}
	if res.Status != state.StatusCompleted {
		t.Fatalf("status = %s, want COMPLETED", res.Status)
	}

	// The two evaluations are durable history in commit order.
	evs, err := f.o.root.ReadEvents("verif-remediate")
	if err != nil {
		t.Fatal(err)
	}
	var outcomes []string
	for _, e := range evs {
		if e.Class == state.EvVerification {
			var b struct {
				Contract string `json:"contract"`
				Outcome  string `json:"outcome"`
			}
			_ = json.Unmarshal(e.Body, &b)
			if b.Contract != "report-valid@1" {
				t.Errorf("event carries wrong token %q", b.Contract)
			}
			outcomes = append(outcomes, b.Outcome)
		}
	}
	if fmt.Sprint(outcomes) != "[FAIL PASS]" {
		t.Errorf("recorded outcomes %v, want [FAIL PASS] — no overwrite, both instances durable", outcomes)
	}

	// Register D/R: the causal replay re-derives the gated transition.
	trs := replayAndVerify(t, f, "verif-remediate")
	final := trs[len(trs)-1]
	if final.To != TargetComplete || !strings.Contains(final.EdgeID, "#g0") {
		t.Errorf("final transition %+v must fire the gated edge", final)
	}
}

// TestNoVerificationNoCompletion: the model declares done without ever
// verifying — the gate cannot open; the reviewed fallback/exhaustion
// path governs; no bypass exists (D-L10-6 standing invariant, live).
func TestNoVerificationNoCompletion(t *testing.T) {
	m := &scriptedModel{steps: []model.ExecutionResponse{
		toolCall("declare_done", `{}`),
		toolCall("declare_done", `{}`),
		toolCall("declare_done", `{}`),
		toolCall("declare_done", `{}`),
		toolCall("declare_done", `{}`),
	}}
	f := setupVerif(t, m, &scriptedEvaluator{})

	res, err := f.o.SubmitTask(f.verifEnvelope(t, "verif-none"))
	if err != nil {
		t.Fatal(err)
	}
	if res.Status != state.StatusFailed {
		t.Fatalf("status = %s, want FAILED — completion without verification must be unreachable", res.Status)
	}
	replayAndVerify(t, f, "verif-none")
}

// TestDowngradeClosesGate: PASS then UNAVAILABLE — the latest
// committed evaluation governs; stale assurance does not hold the
// gate open (D-L10-9).
func TestDowngradeClosesGate(t *testing.T) {
	m := &scriptedModel{steps: []model.ExecutionResponse{
		verifyCallStep(),               // -> PASS
		verifyCallStep(),               // -> UNAVAILABLE (downgrade)
		toolCall("declare_done", `{}`), // gate closed again -> fallback
		toolCall("declare_done", `{}`),
		toolCall("declare_done", `{}`),
		toolCall("declare_done", `{}`),
	}}
	ev := &scriptedEvaluator{outcomes: []string{"PASS", "UNAVAILABLE"}}
	f := setupVerif(t, m, ev)

	res, err := f.o.SubmitTask(f.verifEnvelope(t, "verif-downgrade"))
	if err != nil {
		t.Fatal(err)
	}
	if res.Status != state.StatusFailed {
		t.Fatalf("status = %s, want FAILED — a downgraded gate must not reopen on stale PASS", res.Status)
	}
	replayAndVerify(t, f, "verif-downgrade")
}

// TestVerificationRefusalIsNotAnOutcome: a pre-instance refusal
// produces no verification event and no gate movement; the walk
// continues under its budgets.
func TestVerificationRefusalIsNotAnOutcome(t *testing.T) {
	m := &scriptedModel{steps: []model.ExecutionResponse{
		toolCall("verify_report", `{"path":"parser.go"}`), // no contract -> refusal
		toolCall("declare_done", `{}`),
		toolCall("declare_done", `{}`),
		toolCall("declare_done", `{}`),
		toolCall("declare_done", `{}`),
	}}
	f := setupVerif(t, m, &realRefusalEvaluator{})

	res, err := f.o.SubmitTask(f.verifEnvelope(t, "verif-refusal"))
	if err != nil {
		t.Fatal(err)
	}
	if res.Status != state.StatusFailed {
		t.Fatalf("status = %s, want FAILED via fallback exhaustion", res.Status)
	}
	evs, _ := f.o.root.ReadEvents("verif-refusal")
	for _, e := range evs {
		if e.Class == state.EvVerification {
			t.Fatal("a refusal must never produce a verification event")
		}
	}
	replayAndVerify(t, f, "verif-refusal")
}

type realRefusalEvaluator struct{}

func (realRefusalEvaluator) EvaluateCall(taskID string, call model.ToolCall, evidence []byte, execRef string) (*VerificationOutcome, error) {
	var args map[string]any
	_ = json.Unmarshal(call.Arguments, &args)
	if c, _ := args["contract"].(string); c == "" {
		return &VerificationOutcome{Refused: true, RefusalReason: "no contract named"}, nil
	}
	return &VerificationOutcome{Refused: true, RefusalReason: "unregistered"}, nil
}

// TestVerificationAssemblyRefusals pins the declaration-gated
// exposure at assembly: verifier capability without the verification
// vocabulary, and without a wired evaluator, both refuse before any
// execution.
func TestVerificationAssemblyRefusals(t *testing.T) {
	t.Run("no verification vocabulary declared", func(t *testing.T) {
		m := &scriptedModel{}
		f := setupVerif(t, m, &scriptedEvaluator{})
		// Doctor the workflow: strip the verification events but keep
		// the verifier capability exposed.
		doctored := strings.Replace(verifWalkWorkflow,
			`"signal:phase-completion-requested",
   "verification-pass","verification-fail","verification-inconclusive",
   "verification-unavailable","verification-invalid"`,
			`"signal:phase-completion-requested"`, 1)
		doctored = strings.Replace(doctored, `{"on":"signal:phase-completion-requested",
     "gate":{"contract":"report-valid@1","outcome":"PASS"},"to":"@complete"},`, ``, 1)
		doctored = strings.Replace(doctored, `{"on":"signal:phase-completion-requested","to":"@stay","counter":3,"exhausted_to":"@fail"},`,
			`{"on":"signal:phase-completion-requested","to":"@complete"},`, 1)
		doctored = strings.Replace(doctored, `{"on":"verification-pass","to":"@stay","counter":3,"exhausted_to":"@fail"},
    {"on":"verification-fail","to":"@stay","counter":3,"exhausted_to":"@fail"},
    {"on":"verification-inconclusive","to":"@stay","counter":3,"exhausted_to":"@fail"},
    {"on":"verification-unavailable","to":"@stay","counter":3,"exhausted_to":"@fail"},
    {"on":"verification-invalid","to":"@fail"},`, ``, 1)
		writeJSON(t, f.envDir, "workflow.json", doctored)
		if _, err := f.o.SubmitTask(f.verifEnvelope(t, "verif-undeclared")); err == nil {
			t.Fatal("verifier capability without declared verification events must refuse at assembly")
		}
	})

	t.Run("no evaluator wired", func(t *testing.T) {
		m := &scriptedModel{}
		f := setupVerif(t, m, nil)
		if _, err := f.o.SubmitTask(f.verifEnvelope(t, "verif-nohook")); err == nil {
			t.Fatal("verifier capability with no wired evaluator must refuse at assembly")
		}
	})
}
