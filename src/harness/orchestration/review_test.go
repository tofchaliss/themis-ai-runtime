package orchestration

// Regressions from the L7 security, test, and architecture reviews —
// each pins a remediated finding to its branch.

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/tofchaliss/themis/runtime/model"
	"github.com/tofchaliss/themis/state"
	"github.com/tofchaliss/themis/tools"
)

// Test review HIGH: phase capability sets narrow the grant at the
// gate — a granted control verb outside the phase cannot signal.
func TestPhaseCapabilityNarrowing(t *testing.T) {
	// ANALYZE exposes only read_file (no declare_done): the model's
	// declare_done must be not-available and produce NO signal; the
	// walk then fails via turns-exhausted (its governed edge).
	wf := strings.Replace(defaultWorkflow,
		`{"name":"ANALYZE","capabilities":["read_file","declare_done"]`,
		`{"name":"ANALYZE","capabilities":["read_file"]`, 1)
	m := &scriptedModel{steps: []model.ExecutionResponse{
		toolCall("declare_done", `{}`),
		toolCall("declare_done", `{}`),
		toolCall("declare_done", `{}`),
		toolCall("declare_done", `{}`),
	}}
	f := setup(t, m, wf)
	res, err := f.o.SubmitTask(f.envelope(t, "t-phase"))
	if err != nil {
		t.Fatalf("governed failure is not an error: %v", err)
	}
	if res.Status != state.StatusFailed {
		t.Fatalf("out-of-phase declare_done must not complete the walk: %+v", res)
	}
	evs, _ := f.o.root.ReadEvents("t-phase")
	sawNA, sawSignalTransition := false, false
	for _, ev := range evs {
		if ev.Class == state.EvL4Audit && strings.Contains(string(ev.Body), "not-available") {
			sawNA = true
		}
		if ev.Class == state.EvWorkflowTransition && strings.Contains(string(ev.Body), "phase-completion") {
			sawSignalTransition = true
		}
	}
	if !sawNA || sawSignalTransition {
		t.Fatalf("out-of-phase verb must be not-available and never signal: na=%v signaled=%v", sawNA, sawSignalTransition)
	}
}

// Test review HIGH: the turns-exhausted edge actually fires when the
// turn budget binds before the stay counter.
func TestTurnsExhaustedFires(t *testing.T) {
	wf := strings.Replace(defaultWorkflow,
		`{"on":"turn-no-action","to":"@stay","counter":2,"exhausted_to":"@fail"},
    {"on":"turn-provider-error","to":"@fail"},
    {"on":"tool-error","to":"@stay","counter":2,"exhausted_to":"@fail"},
    {"on":"turns-exhausted","to":"@fail"}]},
  {"name":"VERIFY"`,
		`{"on":"turn-no-action","to":"@stay","counter":9,"exhausted_to":"@fail"},
    {"on":"turn-provider-error","to":"@fail"},
    {"on":"tool-error","to":"@stay","counter":2,"exhausted_to":"@fail"},
    {"on":"turns-exhausted","to":"@fail"}]},
  {"name":"VERIFY"`, 1)
	m := &scriptedModel{steps: []model.ExecutionResponse{
		prose("a"), prose("b"), prose("c"), prose("d"), prose("e"),
	}}
	f := setup(t, m, wf)
	res, err := f.o.SubmitTask(f.envelope(t, "t-turns"))
	if err != nil || res.Status != state.StatusFailed {
		t.Fatalf("turn budget must bind: %+v %v", res, err)
	}
	evs, _ := f.o.root.ReadEvents("t-turns")
	fired := false
	for _, ev := range evs {
		if ev.Class == state.EvWorkflowTransition && strings.Contains(string(ev.Body), `"edge":"turns-exhausted"`) {
			fired = true
		}
	}
	if !fired {
		t.Fatal("the turns-exhausted edge itself must fire and be recorded")
	}
}

// Test review HIGH: declared tool-error drives its governed edge;
// undeclared tool-error stays model-visible and never reaches δ.
func TestToolErrorEdge(t *testing.T) {
	m := &scriptedModel{steps: []model.ExecutionResponse{
		toolCall("read_file", `{"path":"missing.go"}`),
		toolCall("read_file", `{"path":"missing.go"}`),
		toolCall("read_file", `{"path":"missing.go"}`), // counter 2 exhausted -> @fail
	}}
	f := setup(t, m, "")
	res, err := f.o.SubmitTask(f.envelope(t, "t-toolerr"))
	if err != nil || res.Status != state.StatusFailed {
		t.Fatalf("declared tool-error must reach its exhaustion edge: %+v %v", res, err)
	}
	evs, _ := f.o.root.ReadEvents("t-toolerr")
	fired := false
	for _, ev := range evs {
		if ev.Class == state.EvWorkflowTransition && strings.Contains(string(ev.Body), `"edge":"tool-error"`) {
			fired = true
		}
	}
	if !fired {
		t.Fatal("tool-error transition must be recorded")
	}

	// Undeclared variant: same failures, but the workflow does not
	// declare tool-error — the walk continues (model's problem) and
	// completes when declare_done arrives.
	wf := strings.ReplaceAll(defaultWorkflow, `"turns-exhausted","tool-error"`, `"turns-exhausted"`)
	wf = strings.ReplaceAll(wf, `
    {"on":"tool-error","to":"@stay","counter":2,"exhausted_to":"@fail"},`, "")
	wf = strings.ReplaceAll(wf, `
    {"on":"tool-error","to":"@fail"},`, "")
	m2 := &scriptedModel{steps: []model.ExecutionResponse{
		toolCall("read_file", `{"path":"missing.go"}`),
		toolCall("declare_done", `{}`),
		toolCall("declare_done", `{}`),
	}}
	f2 := setup(t, m2, wf)
	res2, err := f2.o.SubmitTask(f2.envelope(t, "t-toolerr2"))
	if err != nil || res2.Status != state.StatusCompleted {
		t.Fatalf("undeclared tool-error must stay the model's problem: %+v %v", res2, err)
	}
}

// Register B (test review MED): tool-call JSON as prose content — the
// qwen2.5-coder pattern — is a no-action turn, never a signal.
func TestJSONInProseIsNotAction(t *testing.T) {
	m := &scriptedModel{steps: []model.ExecutionResponse{
		prose(`{"tool_calls":[{"name":"declare_done","arguments":{}}]}`),
		toolCall("declare_done", `{}`),
		toolCall("declare_done", `{}`),
	}}
	f := setup(t, m, "")
	res, err := f.o.SubmitTask(f.envelope(t, "t-json"))
	if err != nil || res.Status != state.StatusCompleted {
		t.Fatal(err)
	}
	if tr := replayAndVerify(t, f, "t-json"); len(tr) != 2 {
		t.Fatalf("JSON-in-prose caused a transition: %v", tr)
	}
}

// Register B (test review MED): runtime quota exhaustion — the grant
// cap binds mid-walk as a live not-available denial.
func TestRuntimeQuotaExhaustion(t *testing.T) {
	f := setup(t, &scriptedModel{steps: []model.ExecutionResponse{
		toolCall("read_file", `{"path":"parser.go"}`),
		toolCall("read_file", `{"path":"parser.go"}`), // cap 1: denied
		toolCall("declare_done", `{}`),
		toolCall("declare_done", `{}`),
	}}, "")
	writeJSON(t, f.envDir, "grant.json",
		`{"version":1,"task_id":"T","total_max_calls":20,"entries":[
		  {"tool":"read_file","max_calls":1,"workspace":"@workspace"},
		  {"tool":"declare_done","max_calls":4}]}`)
	res, err := f.o.SubmitTask(f.envelope(t, "t-quota"))
	if err != nil || res.Status != state.StatusCompleted {
		t.Fatalf("quota denial is data, not death: %+v %v", res, err)
	}
	evs, _ := f.o.root.ReadEvents("t-quota")
	denied := 0
	for _, ev := range evs {
		if ev.Class == state.EvL4Audit && strings.Contains(string(ev.Body), "not-available") {
			denied++
		}
	}
	if denied != 1 {
		t.Fatalf("second read must be quota-denied exactly once: %d", denied)
	}
}

// Security MED-1: grant/spec task-identity binding at assembly.
func TestCrossArtifactIdentityBinding(t *testing.T) {
	f := setup(t, happyScript(), "")
	p := f.envelope(t, "t-bind")
	// Corrupt the per-task grant back to a foreign task id.
	gp := filepath.Join(f.envDir, "t-bind-grant.json")
	raw := readFile(t, gp)
	writeJSON(t, f.envDir, "t-bind-grant.json", strings.Replace(raw, `"task_id":"t-bind"`, `"task_id":"other"`, 1))
	if _, err := f.o.SubmitTask(p); !errors.Is(err, ErrAssembly) || !strings.Contains(err.Error(), "grant task_id") {
		t.Fatalf("foreign grant must refuse: %v", err)
	}
}

// Architecture 2d: literal workspace bindings refuse — only the
// @workspace placeholder may bind.
func TestLiteralWorkspaceRefused(t *testing.T) {
	f := setup(t, happyScript(), "")
	writeJSON(t, f.envDir, "grant.json",
		`{"version":1,"task_id":"T","total_max_calls":20,"entries":[
		  {"tool":"read_file","max_calls":8,"workspace":"`+f.stateDir+`"},
		  {"tool":"declare_done","max_calls":4}]}`)
	if _, err := f.o.SubmitTask(f.envelope(t, "t-lit")); !errors.Is(err, ErrAssembly) || !strings.Contains(err.Error(), "@workspace placeholder") {
		t.Fatalf("literal workspace binding must refuse: %v", err)
	}
}

// Test review HIGH: the startup sweep, hermetically — non-terminal
// recovered, terminal untouched, corrupt surfaced and preserved.
func TestStartupSweepHermetic(t *testing.T) {
	stateDir := filepath.Join(t.TempDir(), "state")
	root, err := state.OpenRoot(stateDir)
	if err != nil {
		t.Fatal(err)
	}
	// Terminal task.
	tr1, _ := root.CreateTask("t-done", state.TaskOptions{})
	_ = tr1.Transition(state.StatusRunning, "r")
	_ = tr1.Transition(state.StatusCompleted, "done")
	tr1.Close()
	// Mid-walk non-terminal task (a crash victim).
	tr2, _ := root.CreateTask("t-mid", state.TaskOptions{})
	_ = tr2.Transition(state.StatusRunning, "r")
	tr2.Close()
	// Corrupt task: mutate a committed entry.
	tr3, _ := root.CreateTask("t-bad", state.TaskOptions{})
	_ = tr3.Transition(state.StatusRunning, "r")
	tr3.Close()
	streamPath := filepath.Join(stateDir, "tasks", "t-bad", "events.log")
	raw, _ := os.ReadFile(streamPath)
	mutated := strings.Replace(string(raw), `"reason":"r"`, `"reason":"X"`, 1)
	if err := os.WriteFile(streamPath, []byte(mutated), 0o600); err != nil {
		t.Fatal(err)
	}

	_, rep, err := Open(Config{
		StateRoot: stateDir, ArtifactDir: filepath.Join(t.TempDir(), "a"),
		GitPath: gitBin(t), ProviderDir: t.TempDir(),
		SafetyRoot: filepath.Join(repoRoot, "instructions/global/safety"),
		SystemRoot: filepath.Join(repoRoot, "instructions/global/system"),
		PolicyPath: filepath.Join(repoRoot, "policies/security/instruction-directive-patterns.json"),
		Model:      happyScript(),
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(rep.Terminal) != 1 || rep.Terminal[0] != "t-done" {
		t.Fatalf("terminal bucket: %+v", rep)
	}
	if len(rep.Recovered) != 1 || rep.Recovered[0] != "t-mid" {
		t.Fatalf("recovered bucket: %+v", rep)
	}
	if len(rep.Corrupt) != 1 || rep.Corrupt[0] != "t-bad" {
		t.Fatalf("corrupt bucket: %+v", rep)
	}
	view, _ := root.ReadStatus("t-mid")
	if view.Status != state.StatusFailedPartial {
		t.Fatalf("non-terminal must be closed: %+v", view)
	}
	// Corrupt preserved untouched: still corrupt, bytes intact.
	if v := root.Verify("t-bad"); v.Verdict != state.VerdictCorrupt {
		t.Fatalf("corrupt must stay surfaced-and-preserved: %+v", v)
	}
}

// Architecture 4.3 / constitution floor: the wall-clock budget floor
// seals env-deadline and fails the task — never a workflow edge.
func TestWallClockBudgetFloor(t *testing.T) {
	m := &scriptedModel{steps: []model.ExecutionResponse{
		prose("thinking"), prose("thinking"), prose("thinking"),
	}, delay: 1200 * time.Millisecond} // pre-turn-3 floor check at 2.4s > the 2s budget, before the no-action counter exhausts
	f := setup(t, m, "")
	// Tight wall budget: provisioning fits, the walk does not.
	writeJSON(t, f.envDir, "spec.json",
		`{"version":1,"task_id":"T","repo":"demo","pinned_sha":"`+f.sha+`","limits":[{"dimension":"wall_deadline_s","value":2}]}`)
	res, err := f.o.SubmitTask(f.envelope(t, "t-wall"))
	if err != nil {
		t.Fatalf("a floor is typed, not an error: %v", err)
	}
	if res.Status != state.StatusFailed {
		t.Fatalf("wall floor must fail the task: %+v", res)
	}
	man, _ := f.o.root.ReadManifest("t-wall")
	_ = man
	evs, _ := f.o.root.ReadEvents("t-wall")
	floorSeen := false
	for _, ev := range evs {
		if ev.Class == state.EvLifecycle && strings.Contains(string(ev.Body), "wall-clock budget exhausted") {
			floorSeen = true
		}
	}
	if !floorSeen {
		t.Fatal("the floor's typed reason must be in the record")
	}
}

// Register A (test review MED): the real registry-v3 satisfies the
// two-way control vocabulary; a doctored control registry refuses at
// assembly.
func TestControlVocabularyTwoWay(t *testing.T) {
	reg, err := tools.LoadRegistry(mustAbs(t, filepath.Join(repoRoot, "policies/tools/registry-v3.json")))
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, td := range reg.Tools {
		if td.Name == "declare_done" {
			found = td.Control
		}
	}
	if !found {
		t.Fatal("registry-v3 must declare declare_done as control")
	}
	if err := checkControlVocabulary(reg); err != nil {
		t.Fatal(err)
	}
	// Doctored: a control tool outside the constitution.
	doctored := *reg
	doctored.Tools = append(doctored.Tools, tools.ToolDef{Name: "end_task", Control: true})
	if err := checkControlVocabulary(&doctored); !errors.Is(err, ErrAssembly) || !strings.Contains(err.Error(), "not in the constitution vocabulary") {
		t.Fatalf("non-constitution control tool must refuse: %v", err)
	}
}

// δ invariant branches, unit-level (test review LOW): undeclared
// event at δ and declared-without-edge are invariant violations.
func TestStepInvariantBranches(t *testing.T) {
	ceiling, _ := LoadWorkflowCeiling(writeJSON(t, t.TempDir(), "c.json", defaultCeiling))
	wf, err := LoadWorkflow(writeJSON(t, t.TempDir(), "w.json", defaultWorkflow), ceiling)
	if err != nil {
		t.Fatal(err)
	}
	w := &walk{wf: wf, phase: wf.Initial, edgeFires: map[string]int64{}}
	if _, err := w.step("weather-changed"); !errors.Is(err, ErrInvariant) || !strings.Contains(err.Error(), "without declaration") {
		t.Fatalf("undeclared event at δ must be invariant: %v", err)
	}
	// Declared-without-edge is unproducible through the loader; forge
	// the in-memory state to pin the runtime guard.
	w.wf.Phases[0].Edges = w.wf.Phases[0].Edges[:1]
	if _, err := w.step(EvTurnNoAction); !errors.Is(err, ErrInvariant) || !strings.Contains(err.Error(), "static totality violated") {
		t.Fatalf("declared event without edge must be invariant: %v", err)
	}
}

// grantWithinCeiling tool branch (test review MED).
func TestGrantToolAboveCeiling(t *testing.T) {
	f := setup(t, happyScript(), "")
	writeJSON(t, f.envDir, "grant.json",
		`{"version":1,"task_id":"T","total_max_calls":5,"entries":[
		  {"tool":"write_file","max_calls":1,"workspace":"@workspace","mutating":true},
		  {"tool":"declare_done","max_calls":4}]}`)
	if _, err := f.o.SubmitTask(f.envelope(t, "t-gtool")); !errors.Is(err, ErrAssembly) || !strings.Contains(err.Error(), `grant tool "write_file" exceeds`) {
		t.Fatalf("grant tool outside workflow ceiling must refuse: %v", err)
	}
}
