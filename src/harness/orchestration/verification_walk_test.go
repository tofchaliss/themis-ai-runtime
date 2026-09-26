package orchestration

// L10-M6 Register R proofs over gate-bearing walks through the
// production loop with a scripted model and a scripted evaluator:
// FAIL→PASS remediation, the no-verification non-completion
// invariant, the PASS→UNAVAILABLE downgrade, assembly refusals, and
// causal replay over every walk (the extended Register D).

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tofchaliss/themis-ai-runtime/src/harness/runtime/model"
	"github.com/tofchaliss/themis-ai-runtime/src/harness/state"
)

// scriptedEvaluator returns pre-scripted verification outcomes in
// call order — the L10 seam's shape without its machinery, so these
// proofs pin the L7 side deterministically.
type scriptedEvaluator struct {
	outcomes []string
	i        int
}

func (s *scriptedEvaluator) PreResolve(taskID string, call model.ToolCall, authRegistrySHA256 string) (string, error) {
	return "", nil
}

func (s *scriptedEvaluator) EvaluateCall(taskID string, call model.ToolCall, evidence []byte, execRef, authRegistrySHA256 string) (*VerificationOutcome, error) {
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
		Unanchored: true,
		StateRoot:  f.stateDir, ArtifactDir: filepath.Join(t.TempDir(), "artifacts"),
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
	 "registry_path":`+jstr(mustAbs(t, filepath.Join(repoRoot, "policies/tools/registry-v4.json")))+`,
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
		// An AUTHORIZED call (contract named, so L4 admits it) whose
		// contract the seam cannot resolve: the pre-instance refusal.
		toolCall("verify_report", `{"path":"parser.go","contract":"nope@1"}`),
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
	recorded := ""
	for _, e := range evs {
		if e.Class == state.EvVerification {
			t.Fatal("a refusal must never produce a verification event")
		}
		// F-L8-3: the pre-instance refusal text the model saw rides in
		// the call's own l4-audit, committed before the result
		// re-entered — reconstructable, no event of its own.
		if e.Class == state.EvL4Audit {
			var body struct {
				Tool                string
				Decision            string
				VerificationRefusal string
			}
			if json.Unmarshal(e.Body, &body) == nil && body.Tool == "verify_report" {
				if body.Decision != "authorized" {
					t.Fatalf("the verifier call must have been authorized: %s", e.Body)
				}
				recorded = body.VerificationRefusal
			}
		}
	}
	if recorded != "unregistered" {
		t.Fatalf("the refusal text must be recorded verbatim in the l4-audit body: %q", recorded)
	}
	replayAndVerify(t, f, "verif-refusal")
}

type realRefusalEvaluator struct{}

func (realRefusalEvaluator) PreResolve(taskID string, call model.ToolCall, authRegistrySHA256 string) (string, error) {
	var args map[string]any
	_ = json.Unmarshal(call.Arguments, &args)
	if c, _ := args["contract"].(string); c == "" {
		return "no contract named", nil
	}
	return "unregistered", nil
}

func (r realRefusalEvaluator) EvaluateCall(taskID string, call model.ToolCall, evidence []byte, execRef, authRegistrySHA256 string) (*VerificationOutcome, error) {
	refusal, _ := r.PreResolve(taskID, call, authRegistrySHA256)
	return &VerificationOutcome{Refused: true, RefusalReason: refusal}, nil
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

	// The subtest above strips the declarations AND the edges together,
	// so either of the two assembly gates alone refuses it and neither
	// is the control under test. Both survived the 2026-09-14 mutation
	// pass because of that mutual cover — and loop.go:474's "unreachable
	// when assembly held" rests on exactly these two.
	//
	// Each of the two below moves ONE dimension for ONE event.
	t.Run("declared but no edge in the exposing phase", func(t *testing.T) {
		f := setupVerif(t, &scriptedModel{}, &scriptedEvaluator{})
		// Every verification event still DECLARED; one has no edge, so
		// only the reachability gate can refuse.
		doctored := strings.Replace(verifWalkWorkflow,
			`{"on":"verification-inconclusive","to":"@stay","counter":3,"exhausted_to":"@fail"},`, ``, 1)
		if doctored == verifWalkWorkflow {
			t.Fatal("the edge to remove was not found — the fixture changed shape")
		}
		writeJSON(t, f.envDir, "workflow.json", doctored)
		_, err := f.o.SubmitTask(f.verifEnvelope(t, "verif-noedge"))
		if err == nil {
			t.Fatal("a phase exposing a verifier capability with an unreachable verification event was ACCEPTED")
		}
		if !strings.Contains(err.Error(), "no edge for reachable event") {
			t.Fatalf("refused for the wrong reason — declaration must still hold: %v", err)
		}
	})

	// The assembly gate's companion — "exposes a verifier-eligible
	// capability but the workflow does not declare <ev>" — cannot be
	// reached with its edge intact, because the LOADER already refuses
	// an edge on an undeclared event. That ordering is what makes the
	// assembly declaration gate redundant, and it is also what makes
	// loop.go:474 ("verification event undeclared yet produced")
	// unreachable. Pinned here so a future change that let such a
	// workflow load would surface as a changed refusal rather than
	// silently promote two dead guards into live ones.
	t.Run("an edge on an undeclared event never reaches assembly", func(t *testing.T) {
		f := setupVerif(t, &scriptedModel{}, &scriptedEvaluator{})
		doctored := strings.Replace(verifWalkWorkflow, `"verification-inconclusive",`, ``, 1)
		if doctored == verifWalkWorkflow {
			t.Fatal("the declaration to remove was not found — the fixture changed shape")
		}
		writeJSON(t, f.envDir, "workflow.json", doctored)
		_, err := f.o.SubmitTask(f.verifEnvelope(t, "verif-undeclared-one"))
		if err == nil {
			t.Fatal("a workflow with an edge on an undeclared event was ACCEPTED")
		}
		if !errors.Is(err, ErrWorkflow) {
			t.Fatalf("this must be a workflow-definition refusal, not an assembly one: %v", err)
		}
		if !strings.Contains(err.Error(), "edge on undeclared event") {
			t.Fatalf("refused for the wrong reason: %v", err)
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

// --- fault sweep over the verification store/commit window ----------

var verificationFaultPoints = []string{
	"loop.pre-verification-store",
	"loop.pre-verification-commit",
	"loop.post-verification-commit",
}

// TestVerificationFaultSweep: a crash at any point in the
// store→commit→state window takes the invariant path with a typed
// terminal, an uncorrupted record, and no verification event without
// its durable record (record-before-event proven under fault, closing
// the deferral recorded in the L7 amendment).
func TestVerificationFaultSweep(t *testing.T) {
	for _, point := range verificationFaultPoints {
		point := point
		t.Run(point, func(t *testing.T) {
			m := &scriptedModel{steps: []model.ExecutionResponse{verifyCallStep()}}
			ev := &scriptedEvaluator{outcomes: []string{"PASS"}}
			f := setupVerif(t, m, ev)
			t.Cleanup(func() { fault = nil })
			fault = func(p string) error {
				if p == point {
					return fmt.Errorf("injected@%s", p)
				}
				return nil
			}
			res, err := f.o.SubmitTask(f.verifEnvelope(t, "verif-fault"))
			fault = nil
			if err == nil {
				t.Fatalf("armed point %s must fail the walk", point)
			}
			if res.Status != state.StatusFailed && res.Status != state.StatusFailedPartial {
				t.Fatalf("typed terminal required: %+v", res)
			}
			view, verr := f.o.ReadStatus("verif-fault")
			if verr != nil || view.Verdict == state.VerdictCorrupt {
				t.Fatalf("no fault point may corrupt the record: %+v %v", view, verr)
			}
			evs, _ := f.o.root.ReadEvents("verif-fault")
			for _, e := range evs {
				if e.Class == state.EvVerification {
					// An event exists only past the commit point; its
					// named record object must be retrievable.
					var b struct {
						Record string `json:"record"`
					}
					_ = json.Unmarshal(e.Body, &b)
					if b.Record == "" {
						t.Fatal("verification event without a named record")
					}
					if _, gerr := f.o.root.Store().GetObject(b.Record); gerr != nil {
						t.Fatal("verification event references an unretrievable record — record-before-event violated")
					}
				}
			}
		})
	}
}

// TestTwoGateLadderInProduction: two satisfiable gated edges on one
// event — production δ must fire the FIRST in definition order (#g0),
// pinned against the real loop, not a mirror (close test review:
// the ladder mutation survived the mirror-only test).
func TestTwoGateLadderInProduction(t *testing.T) {
	twoGate := strings.Replace(verifWalkWorkflow,
		`{"on":"signal:phase-completion-requested",
     "gate":{"contract":"report-valid@1","outcome":"PASS"},"to":"@complete"},`,
		`{"on":"signal:phase-completion-requested",
     "gate":{"contract":"report-valid@1","outcome":"PASS"},"to":"@complete"},
    {"on":"signal:phase-completion-requested",
     "gate":{"contract":"report-valid@1","outcome":"PASS"},"to":"@fail"},`, 1)
	m := &scriptedModel{steps: []model.ExecutionResponse{
		verifyCallStep(),
		toolCall("declare_done", `{}`),
	}}
	ev := &scriptedEvaluator{outcomes: []string{"PASS"}}
	f := setupVerif(t, m, ev)
	writeJSON(t, f.envDir, "workflow.json", twoGate)

	res, err := f.o.SubmitTask(f.verifEnvelope(t, "verif-twogate"))
	if err != nil {
		t.Fatal(err)
	}
	// Both gates satisfied; the first (to @complete) must win. A
	// last-satisfied-wins mutation would land on @fail.
	if res.Status != state.StatusCompleted {
		t.Fatalf("status = %s — first-satisfied gate must win in definition order", res.Status)
	}
	evs, _ := f.o.root.ReadEvents("verif-twogate")
	for _, e := range evs {
		if e.Class == state.EvWorkflowTransition && strings.Contains(string(e.Body), "@complete") {
			if !strings.Contains(string(e.Body), "#g0") {
				t.Fatalf("completing transition must record gate #g0: %s", string(e.Body))
			}
		}
	}
}

// TestCrossTaskReplayCannotSatisfyGate (Register T #8): a PASS in one
// task must be invisible to another task's gate — the second walk,
// with no verification of its own, must fail closed.
func TestCrossTaskReplayCannotSatisfyGate(t *testing.T) {
	// Task A: verified PASS, completes.
	mA := &scriptedModel{steps: []model.ExecutionResponse{
		verifyCallStep(),
		toolCall("declare_done", `{}`),
	}}
	f := setupVerif(t, mA, &scriptedEvaluator{outcomes: []string{"PASS"}})
	resA, err := f.o.SubmitTask(f.verifEnvelope(t, "verif-task-a"))
	if err != nil || resA.Status != state.StatusCompleted {
		t.Fatalf("task A: %v %+v", err, resA)
	}

	// Task B on the SAME orchestrator/state root: never verifies.
	// Task A's committed PASS must not leak into B's gate state.
	f.o.cfg.Model = &scriptedModel{steps: []model.ExecutionResponse{
		toolCall("declare_done", `{}`),
		toolCall("declare_done", `{}`),
		toolCall("declare_done", `{}`),
		toolCall("declare_done", `{}`),
	}}
	resB, err := f.o.SubmitTask(f.verifEnvelope(t, "verif-task-b"))
	if err != nil {
		t.Fatal(err)
	}
	if resB.Status != state.StatusFailed {
		t.Fatalf("task B completed on task A's verification — cross-task leak: %+v", resB)
	}
	evsB, _ := f.o.root.ReadEvents("verif-task-b")
	for _, e := range evsB {
		if e.Class == state.EvVerification {
			t.Fatal("task B has a verification event it never earned")
		}
	}
}

// TestInvalidAndInconclusiveWalks drives the remaining outcome events
// through the production loop: INVALID takes its @fail edge;
// INCONCLUSIVE stays and the gate never opens.
func TestInvalidAndInconclusiveWalks(t *testing.T) {
	t.Run("INVALID routes to @fail", func(t *testing.T) {
		m := &scriptedModel{steps: []model.ExecutionResponse{verifyCallStep()}}
		f := setupVerif(t, m, &scriptedEvaluator{outcomes: []string{"INVALID"}})
		res, err := f.o.SubmitTask(f.verifEnvelope(t, "verif-invalid"))
		if err != nil {
			t.Fatal(err)
		}
		if res.Status != state.StatusFailed {
			t.Fatalf("INVALID must take the reviewed @fail edge: %+v", res)
		}
		replayAndVerify(t, f, "verif-invalid")
	})
	t.Run("INCONCLUSIVE never opens the gate", func(t *testing.T) {
		m := &scriptedModel{steps: []model.ExecutionResponse{
			verifyCallStep(),
			toolCall("declare_done", `{}`),
			toolCall("declare_done", `{}`),
			toolCall("declare_done", `{}`),
			toolCall("declare_done", `{}`),
		}}
		f := setupVerif(t, m, &scriptedEvaluator{outcomes: []string{"INCONCLUSIVE"}})
		res, err := f.o.SubmitTask(f.verifEnvelope(t, "verif-inconclusive"))
		if err != nil {
			t.Fatal(err)
		}
		if res.Status != state.StatusFailed {
			t.Fatalf("INCONCLUSIVE must not satisfy a PASS gate: %+v", res)
		}
		replayAndVerify(t, f, "verif-inconclusive")
	})
}

// TestEventTamperDetectedAtReadBoundary (Register T #5): an
// EvVerification event body altered in storage is detected by the L6
// integrity mechanisms at the authoritative read boundary.
func TestEventTamperDetectedAtReadBoundary(t *testing.T) {
	m := &scriptedModel{steps: []model.ExecutionResponse{
		verifyCallStep(),
		toolCall("declare_done", `{}`),
	}}
	f := setupVerif(t, m, &scriptedEvaluator{outcomes: []string{"PASS"}})
	res, err := f.o.SubmitTask(f.verifEnvelope(t, "verif-tamper"))
	if err != nil || res.Status != state.StatusCompleted {
		t.Fatalf("%v %+v", err, res)
	}

	// Post-hoc storage tamper: flip the recorded outcome in the raw
	// event stream file.
	//
	// Neither condition below may skip. As written at the L10 close
	// this read named "events.jsonl" while L6 writes "events.log", so
	// the missing file skipped and the test reported green without
	// ever running — Register T #5 was recorded CLOSED on evidence
	// that never executed. A skip here means the stream's storage
	// shape moved out from under this test, which is precisely when
	// the tamper evidence must fail loudly rather than evaporate.
	streamPath := filepath.Join(f.stateDir, "tasks", "verif-tamper", "events.log")
	raw, rerr := os.ReadFile(streamPath)
	if rerr != nil {
		t.Fatalf("event stream not at the expected path — this test cannot prove tamper detection: %v", rerr)
	}
	tampered := strings.Replace(string(raw), `\"outcome\":\"PASS\"`, `\"outcome\":\"FAIL\"`, 1)
	if tampered == string(raw) {
		tampered = strings.Replace(string(raw), `"outcome":"PASS"`, `"outcome":"FAIL"`, 1)
	}
	if tampered == string(raw) {
		t.Fatalf("no PASS outcome bytes in the stream representation — nothing was tampered, so a clean read back would prove nothing (stream %d bytes)", len(raw))
	}
	if err := os.WriteFile(streamPath, []byte(tampered), 0o644); err != nil {
		t.Fatal(err)
	}
	// The authoritative read boundary must refuse or verdict-corrupt.
	if _, err := f.o.root.ReadEvents("verif-tamper"); err == nil {
		view, verr := f.o.ReadStatus("verif-tamper")
		if verr == nil && view.Verdict != state.VerdictCorrupt {
			t.Fatal("tampered event stream read back clean — L6 integrity did not detect")
		}
	}
}
