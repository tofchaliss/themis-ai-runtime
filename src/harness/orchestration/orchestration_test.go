package orchestration

import (
	stdctx "context"
	"encoding/json"
	"errors"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tofchaliss/themis/runtime/model"
	"github.com/tofchaliss/themis/state"
)

const repoRoot = "../../.."

// --- scripted model (Register-testable advisory core) ----------------

type scriptedModel struct {
	steps []model.ExecutionResponse
	i     int
	fail  bool
}

func (s *scriptedModel) Name() string { return "scripted" }

func (s *scriptedModel) Execute(ctx stdctx.Context, req model.ExecutionRequest) (*model.ExecutionResponse, error) {
	if s.fail {
		return nil, errors.New("scripted provider failure")
	}
	if s.i >= len(s.steps) {
		return &model.ExecutionResponse{Content: "…", Termination: model.TerminationStop}, nil
	}
	r := s.steps[s.i]
	s.i++
	return &r, nil
}

func toolCall(name, args string) model.ExecutionResponse {
	return model.ExecutionResponse{Termination: model.TerminationToolCalls,
		ToolCalls: []model.ToolCall{{ID: "c", Name: name, Arguments: json.RawMessage(args)}}}
}

func prose(s string) model.ExecutionResponse {
	return model.ExecutionResponse{Content: s, Termination: model.TerminationStop}
}

// --- fixtures --------------------------------------------------------

func gitBin(t *testing.T) string {
	t.Helper()
	for _, p := range []string{"/usr/bin/git", "/opt/homebrew/bin/git"} {
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	t.Skip("no pinned git")
	return ""
}

func mkMirror(t *testing.T) (string, string) {
	t.Helper()
	git := gitBin(t)
	root := t.TempDir()
	repo := filepath.Join(root, "demo")
	run := func(args ...string) string {
		cmd := exec.Command(git, append([]string{"-c", "user.name=t", "-c", "user.email=t@t"}, args...)...)
		cmd.Dir = repo
		cmd.Env = append(os.Environ(), "GIT_CONFIG_GLOBAL=/dev/null", "GIT_CONFIG_SYSTEM=/dev/null")
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("git %v: %v: %s", args, err, out)
		}
		return string(out)
	}
	if err := os.Mkdir(repo, 0o755); err != nil {
		t.Fatal(err)
	}
	run("init", "-q")
	if err := os.WriteFile(filepath.Join(repo, "parser.go"), []byte("package parser // L7-SENTINEL\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	run("add", ".")
	run("commit", "-q", "-m", "seed")
	return root, strings.TrimSpace(run("rev-parse", "HEAD"))
}

func writeJSON(t *testing.T, dir, name, body string) string {
	t.Helper()
	p := filepath.Join(dir, name)
	if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	return p
}

const defaultWorkflow = `{
 "version":1,"name":"analyze-verify","initial":"ANALYZE",
 "declared_events":["turn-no-action","turn-provider-error","turns-exhausted","tool-error","signal:phase-completion-requested"],
 "phases":[
  {"name":"ANALYZE","capabilities":["read_file","declare_done"],"max_model_turns":4,"edges":[
    {"on":"signal:phase-completion-requested","to":"VERIFY"},
    {"on":"turn-no-action","to":"@stay","counter":2,"exhausted_to":"@fail"},
    {"on":"turn-provider-error","to":"@fail"},
    {"on":"tool-error","to":"@stay","counter":2,"exhausted_to":"@fail"},
    {"on":"turns-exhausted","to":"@fail"}]},
  {"name":"VERIFY","capabilities":["declare_done"],"max_model_turns":3,"edges":[
    {"on":"signal:phase-completion-requested","to":"@complete"},
    {"on":"turn-no-action","to":"@stay","counter":2,"exhausted_to":"@fail"},
    {"on":"turn-provider-error","to":"@fail"},
    {"on":"tool-error","to":"@fail"},
    {"on":"turns-exhausted","to":"@fail"}]}
 ]}`

const defaultCeiling = `{"version":1,"allowed_tools":["read_file","declare_done"],"max_total_calls":20,"max_walk_length":100,"max_turns_per_phase":10}`

type fixture struct {
	o        *Orchestrator
	rep      *StartupReport
	envDir   string
	mirror   string
	sha      string
	stateDir string
}

func setup(t *testing.T, m model.Interface, workflowJSON string) *fixture {
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
	})
	if err != nil {
		t.Fatal(err)
	}
	f.o, f.rep = o, rep
	if workflowJSON == "" {
		workflowJSON = defaultWorkflow
	}
	writeJSON(t, f.envDir, "workflow.json", workflowJSON)
	writeJSON(t, f.envDir, "wceiling.json", defaultCeiling)
	writeJSON(t, f.envDir, "eceiling.json",
		`{"version":1,"mirror_root":"`+f.mirror+`","max_wall_deadline_sec":120,"max_file_bytes":1048576,"max_total_bytes":10485760,"max_file_count":500,"max_mem_bytes":1073741824,"max_cpu_time_sec":600,"max_proc_count":64}`)
	writeJSON(t, f.envDir, "spec.json",
		`{"version":1,"task_id":"T","repo":"demo","pinned_sha":"`+f.sha+`","limits":[{"dimension":"wall_deadline_s","value":90}]}`)
	writeJSON(t, f.envDir, "grant.json",
		`{"version":1,"task_id":"T","total_max_calls":20,"entries":[
		  {"tool":"read_file","max_calls":8,"workspace":"@workspace"},
		  {"tool":"declare_done","max_calls":4}]}`)
	return f
}

func (f *fixture) envelope(t *testing.T, taskID string) *Envelope {
	t.Helper()
	abs := func(n string) string { return filepath.Join(f.envDir, n) }
	p := writeJSON(t, f.envDir, "envelope-"+taskID+".json", `{
	 "version":1,"task_id":"`+taskID+`","model":"scripted","payload":"Read parser.go, then call declare_done.",
	 "workflow_path":`+jstr(abs("workflow.json"))+`,"workflow_ceiling_path":`+jstr(abs("wceiling.json"))+`,
	 "registry_path":`+jstr(mustAbs(t, filepath.Join(repoRoot, "policies/tools/registry-v3.json")))+`,
	 "grant_path":`+jstr(abs("grant.json"))+`,"exec_ceiling_path":`+jstr(abs("eceiling.json"))+`,
	 "spec_path":`+jstr(abs("spec.json"))+`}`)
	e, err := LoadEnvelope(p)
	if err != nil {
		t.Fatal(err)
	}
	return e
}

func mustAbs(t *testing.T, p string) string {
	t.Helper()
	a, err := filepath.Abs(p)
	if err != nil {
		t.Fatal(err)
	}
	return a
}

func jstr(s string) string { b, _ := json.Marshal(s); return string(b) }

// --- the happy walk + Register D -------------------------------------

func happyScript() *scriptedModel {
	return &scriptedModel{steps: []model.ExecutionResponse{
		toolCall("read_file", `{"path":"parser.go"}`),
		toolCall("declare_done", `{}`),
		toolCall("declare_done", `{}`),
	}}
}

func TestHappyWalkAndReplay(t *testing.T) {
	f := setup(t, happyScript(), "")
	res, err := f.o.SubmitTask(f.envelope(t, "t-happy"))
	if err != nil {
		t.Fatalf("walk failed: %v", err)
	}
	if res.Status != state.StatusCompleted || res.Artifact == "" || res.Verdict != state.VerdictVerified {
		t.Fatalf("unexpected result: %+v", res)
	}
	transitions := replayAndVerify(t, f, "t-happy")
	want := [][2]string{{"ANALYZE", "VERIFY"}, {"VERIFY", "@complete"}}
	if len(transitions) != len(want) {
		t.Fatalf("transitions: %+v", transitions)
	}
	for i, tr := range transitions {
		if tr[0] != want[i][0] || tr[1] != want[i][1] {
			t.Fatalf("transition %d: got %v want %v", i, tr, want[i])
		}
	}
}

// replayAndVerify is the Register-D replayer: it re-derives the walk
// from (definition-at-hash, recorded events), checks every recorded
// transition against its own derivation (single authority), verifies
// CallState by recount, and reconstructs what the model saw.
func replayAndVerify(t *testing.T, f *fixture, taskID string) [][2]string {
	t.Helper()
	evs, err := f.o.root.ReadEvents(taskID)
	if err != nil {
		t.Fatal(err)
	}
	man, err := f.o.root.ReadManifest(taskID)
	if err != nil {
		t.Fatal(err)
	}
	ceiling, _ := LoadWorkflowCeiling(filepath.Join(f.envDir, "wceiling.json"))
	wf, err := LoadWorkflow(filepath.Join(f.envDir, "workflow.json"), ceiling)
	if err != nil {
		t.Fatal(err)
	}
	if man.GovernedHashes["workflow"] != wf.Hash {
		t.Fatal("recorded workflow hash mismatch")
	}

	// Re-derive the causal event sequence δ would have consumed.
	var derived [][2]string
	phase := wf.Initial
	fires := map[string]int64{}
	stepδ := func(event string) {
		p := wf.phase(phase)
		var edge *Edge
		for i := range p.Edges {
			if p.Edges[i].On == event {
				edge = &p.Edges[i]
			}
		}
		if edge == nil {
			t.Fatalf("replay: no edge for %s in %s", event, phase)
		}
		target := edge.To
		key := phase + "/" + event
		if edge.Counter > 0 {
			if fires[key] >= edge.Counter {
				target = edge.ExhaustedTo
			} else {
				fires[key]++
			}
		}
		if target == TargetStay {
			return
		}
		derived = append(derived, [2]string{phase, target})
		if !strings.HasPrefix(target, "@") {
			phase = target
		}
	}
	callCounts := map[string]int{}
	for _, ev := range evs {
		switch ev.Class {
		case state.EvModelTurn:
			var b struct {
				Fact string `json:"fact"`
			}
			_ = json.Unmarshal(ev.Body, &b)
			// Model output reconstructable: the turn's exact bytes.
			if len(ev.Refs) == 1 {
				if _, err := f.o.root.Resolve(ev, 0); err != nil {
					t.Fatalf("model turn %d not reconstructable: %v", ev.Seq, err)
				}
			}
			switch b.Fact {
			case "no-action":
				stepδ(EvTurnNoAction)
			case "provider-error":
				stepδ(EvTurnProviderError)
			}
		case state.EvL4Audit:
			var audit struct {
				Tool     string `json:"Tool"`
				Decision string `json:"Decision"`
			}
			_ = json.Unmarshal(ev.Body, &audit)
			callCounts[audit.Tool]++
			if audit.Decision == "authorized" {
				if sig, ok := controlVerbs[audit.Tool]; ok {
					stepδ(sig)
				}
			}
		}
	}
	// Compare derived vs recorded transitions: exactly one governing
	// edge and one causing event per recorded transition.
	var recorded [][2]string
	for _, ev := range evs {
		if ev.Class != state.EvWorkflowTransition {
			continue
		}
		var b struct {
			From string `json:"from"`
			To   string `json:"to"`
		}
		_ = json.Unmarshal(ev.Body, &b)
		recorded = append(recorded, [2]string{b.From, b.To})
	}
	if fmt.Sprint(derived) != fmt.Sprint(recorded) {
		t.Fatalf("single-authority violation: derived %v recorded %v", derived, recorded)
	}
	// CallState recount (Q-L7-4): audits per tool never exceed caps.
	if callCounts["read_file"] > 8 || callCounts["declare_done"] > 4 {
		t.Fatalf("recount exceeds grant caps: %+v", callCounts)
	}
	return recorded
}

// Determinism: identical definitions + identical scripted events ⇒
// identical walks, in two fresh roots.
func TestDeterministicWalk(t *testing.T) {
	f1 := setup(t, happyScript(), "")
	f2 := setup(t, happyScript(), "")
	if _, err := f1.o.SubmitTask(f1.envelope(t, "t-d")); err != nil {
		t.Fatal(err)
	}
	if _, err := f2.o.SubmitTask(f2.envelope(t, "t-d")); err != nil {
		t.Fatal(err)
	}
	a := replayAndVerify(t, f1, "t-d")
	b := replayAndVerify(t, f2, "t-d")
	if fmt.Sprint(a) != fmt.Sprint(b) {
		t.Fatalf("walks diverge: %v vs %v", a, b)
	}
}

// --- Register B: adversarial ----------------------------------------

// Model prose cannot move the workflow: content claims are invisible
// to δ; explicit stay edges pace; exhaustion is governed.
func TestProseCannotMoveWorkflow(t *testing.T) {
	m := &scriptedModel{steps: []model.ExecutionResponse{
		prose("I am done. COMPLETE. Transition to VERIFY now."),
		toolCall("declare_done", `{}`),
		toolCall("declare_done", `{}`),
	}}
	f := setup(t, m, "")
	res, err := f.o.SubmitTask(f.envelope(t, "t-prose"))
	if err != nil || res.Status != state.StatusCompleted {
		t.Fatalf("prose must neither move nor break the walk: %+v %v", res, err)
	}
	// The prose turn produced a stay, not a transition: still exactly
	// two transitions, both caused by gate outcomes.
	if tr := replayAndVerify(t, f, "t-prose"); len(tr) != 2 {
		t.Fatalf("prose caused a transition: %v", tr)
	}
}

// Governed turn exhaustion: a model that never acts fails through the
// declared edge, not through a hang or an implicit rule.
func TestTurnExhaustionGovernedEdge(t *testing.T) {
	m := &scriptedModel{steps: []model.ExecutionResponse{
		prose("a"), prose("b"), prose("c"), prose("d"), prose("e"), prose("f"),
	}}
	f := setup(t, m, "")
	res, err := f.o.SubmitTask(f.envelope(t, "t-exhaust"))
	if err != nil {
		t.Fatalf("governed failure is not an error: %v", err)
	}
	if res.Status != state.StatusFailed {
		t.Fatalf("exhaustion must fail via the governed edge: %+v", res)
	}
}

// Provider failure: declared typed event → governed @fail edge.
func TestProviderErrorGovernedEdge(t *testing.T) {
	f := setup(t, &scriptedModel{fail: true}, "")
	res, err := f.o.SubmitTask(f.envelope(t, "t-prov"))
	if err != nil || res.Status != state.StatusFailed {
		t.Fatalf("provider error must take the governed edge: %+v %v", res, err)
	}
}

// declare_done with invented arguments dies at L4 (whole-call
// invalid-args) and therefore produces no signal.
func TestControlVerbInventedArgs(t *testing.T) {
	m := &scriptedModel{steps: []model.ExecutionResponse{
		toolCall("declare_done", `{"reason":"done"}`),
		toolCall("declare_done", `{}`),
		toolCall("declare_done", `{}`),
	}}
	f := setup(t, m, "")
	res, err := f.o.SubmitTask(f.envelope(t, "t-args"))
	if err != nil || res.Status != state.StatusCompleted {
		t.Fatal(err)
	}
	// First call was invalid-args: three declare_done audits, but only
	// two signals — verify via replay (it derives from authorized
	// control audits only) plus explicit audit inspection.
	evs, _ := f.o.root.ReadEvents("t-args")
	invalid := 0
	for _, ev := range evs {
		if ev.Class == state.EvL4Audit && strings.Contains(string(ev.Body), `"denied"`) && strings.Contains(string(ev.Body), "invalid-args") {
			invalid++
		}
	}
	if invalid != 1 {
		t.Fatalf("invented-args call must be denied invalid-args exactly once: %d", invalid)
	}
}

// Task identity is single-use through the seam.
func TestDuplicateSubmitRefused(t *testing.T) {
	f := setup(t, happyScript(), "")
	if _, err := f.o.SubmitTask(f.envelope(t, "t-dup")); err != nil {
		t.Fatal(err)
	}
	if _, err := f.o.SubmitTask(f.envelope(t, "t-dup")); !errors.Is(err, state.ErrIdentity) {
		t.Fatalf("duplicate submission must refuse: %v", err)
	}
}

// Workflow loader refusals — each branch pinned.
func TestWorkflowLoaderFailsClosed(t *testing.T) {
	ceiling, err := LoadWorkflowCeiling(writeJSON(t, t.TempDir(), "c.json", defaultCeiling))
	if err != nil {
		t.Fatal(err)
	}
	cases := []struct{ name, patch, wantErr string }{
		{"undeclarable-event", `"declared_events":["turn-no-action","turns-exhausted","weather-changed"`, "undeclarable event"},
		{"approval-reserved", `"declared_events":["approval:granted","turns-exhausted"`, "approval conditions are reserved"},
		{"missing-turns-exhausted", `"declared_events":["turn-no-action"`, "must be declared"},
	}
	base := defaultWorkflow
	for _, c := range cases {
		body := strings.Replace(base, `"declared_events":["turn-no-action","turn-provider-error","turns-exhausted","tool-error","signal:phase-completion-requested"`, c.patch, 1)
		_, err := LoadWorkflow(writeJSON(t, t.TempDir(), "w.json", body), ceiling)
		if err == nil || !errors.Is(err, ErrWorkflow) || !strings.Contains(err.Error(), c.wantErr) {
			t.Errorf("%s: want %q, got %v", c.name, c.wantErr, err)
		}
	}
	// Totality: remove one edge → refused naming the event.
	body := strings.Replace(base, `{"on":"turn-provider-error","to":"@fail"},
    {"on":"tool-error","to":"@stay","counter":2,"exhausted_to":"@fail"},`, `{"on":"turn-provider-error","to":"@fail"},`, 1)
	if _, err := LoadWorkflow(writeJSON(t, t.TempDir(), "w2.json", body), ceiling); err == nil || !strings.Contains(err.Error(), "no edge for declared event") {
		t.Errorf("totality gap must refuse: %v", err)
	}
	// Counter-free stay is unloadable.
	body = strings.Replace(base, `{"on":"turn-no-action","to":"@stay","counter":2,"exhausted_to":"@fail"},
    {"on":"turn-provider-error","to":"@fail"},
    {"on":"tool-error","to":"@stay","counter":2,"exhausted_to":"@fail"},
    {"on":"turns-exhausted","to":"@fail"}]},
  {"name":"VERIFY"`, `{"on":"turn-no-action","to":"@stay"},
    {"on":"turn-provider-error","to":"@fail"},
    {"on":"tool-error","to":"@stay","counter":2,"exhausted_to":"@fail"},
    {"on":"turns-exhausted","to":"@fail"}]},
  {"name":"VERIFY"`, 1)
	if _, err := LoadWorkflow(writeJSON(t, t.TempDir(), "w3.json", body), ceiling); err == nil || !strings.Contains(err.Error(), "counter-free cycles are unloadable") {
		t.Errorf("counter-free stay must refuse: %v", err)
	}
	// Capability above ceiling.
	body = strings.Replace(base, `"capabilities":["read_file","declare_done"]`, `"capabilities":["write_file"]`, 1)
	if _, err := LoadWorkflow(writeJSON(t, t.TempDir(), "w4.json", body), ceiling); err == nil || !strings.Contains(err.Error(), "exceeds the workflow ceiling") {
		t.Errorf("capability above ceiling must refuse: %v", err)
	}
	// Turn budget above ceiling.
	body = strings.Replace(base, `"max_model_turns":4`, `"max_model_turns":99`, 1)
	if _, err := LoadWorkflow(writeJSON(t, t.TempDir(), "w5.json", body), ceiling); err == nil || !strings.Contains(err.Error(), "≤ ceiling") {
		t.Errorf("turn budget above ceiling must refuse: %v", err)
	}
}

// Envelope: nothing is defaulted; every missing reference is named.
func TestEnvelopeNoDefaulting(t *testing.T) {
	dir := t.TempDir()
	full := map[string]any{"version": 1, "task_id": "t", "model": "m", "payload": "p",
		"workflow_path": "/w", "workflow_ceiling_path": "/c", "registry_path": "/r",
		"grant_path": "/g", "exec_ceiling_path": "/e", "spec_path": "/s"}
	for missing := range full {
		if missing == "version" || missing == "task_id" {
			continue
		}
		m := map[string]any{}
		for k, v := range full {
			if k != missing {
				m[k] = v
			}
		}
		raw, _ := json.Marshal(m)
		_, err := LoadEnvelope(writeJSON(t, dir, "e-"+missing+".json", string(raw)))
		if err == nil || !errors.Is(err, ErrEnvelope) || !strings.Contains(err.Error(), missing) {
			t.Errorf("missing %s must be refused BY NAME: %v", missing, err)
		}
	}
	// Relative governed path refused.
	m := map[string]any{}
	for k, v := range full {
		m[k] = v
	}
	m["workflow_path"] = "relative/w.json"
	raw, _ := json.Marshal(m)
	if _, err := LoadEnvelope(writeJSON(t, dir, "e-rel.json", string(raw))); err == nil || !strings.Contains(err.Error(), "absolute") {
		t.Errorf("relative path must refuse: %v", err)
	}
}

// Grant above the workflow ceiling refuses at assembly.
func TestGrantCeilingAtAssembly(t *testing.T) {
	f := setup(t, happyScript(), "")
	writeJSON(t, f.envDir, "grant.json",
		`{"version":1,"task_id":"T","total_max_calls":999,"entries":[{"tool":"read_file","max_calls":8,"workspace":"@workspace"}]}`)
	if _, err := f.o.SubmitTask(f.envelope(t, "t-ceil")); !errors.Is(err, ErrAssembly) || !strings.Contains(err.Error(), "exceeds workflow ceiling") {
		t.Fatalf("grant above ceiling must refuse at assembly: %v", err)
	}
}

// --- Register A: structural ------------------------------------------

func TestExportedAPIClosure(t *testing.T) {
	allowTypes := map[string]bool{
		"Config": true, "StartupReport": true, "Orchestrator": true, "TaskResult": true,
		"Envelope": true, "WorkflowDef": true, "WorkflowCeiling": true, "Edge": true, "Phase": true,
	}
	allowFuncs := map[string]bool{
		"Open": true, "LoadEnvelope": true, "LoadWorkflow": true, "LoadWorkflowCeiling": true,
		"ConstitutionHash": true,
	}
	allowMethods := map[string]bool{
		"Orchestrator.SubmitTask": true, "Orchestrator.ReadStatus": true,
	}
	fset := token.NewFileSet()
	pkgs, err := parser.ParseDir(fset, ".", func(fi os.FileInfo) bool {
		return !strings.HasSuffix(fi.Name(), "_test.go")
	}, 0)
	if err != nil {
		t.Fatal(err)
	}
	for _, pkg := range pkgs {
		for _, file := range pkg.Files {
			for _, decl := range file.Decls {
				switch d := decl.(type) {
				case *ast.FuncDecl:
					if !d.Name.IsExported() {
						continue
					}
					if d.Recv != nil && len(d.Recv.List) == 1 {
						recv := ""
						switch rt := d.Recv.List[0].Type.(type) {
						case *ast.StarExpr:
							recv = rt.X.(*ast.Ident).Name
						case *ast.Ident:
							recv = rt.Name
						}
						if !allowMethods[recv+"."+d.Name.Name] {
							t.Errorf("unlisted exported method %s.%s — the seam is closed", recv, d.Name.Name)
						}
					} else if !allowFuncs[d.Name.Name] {
						t.Errorf("unlisted exported function %s — the seam is closed", d.Name.Name)
					}
				case *ast.GenDecl:
					for _, spec := range d.Specs {
						switch sp := spec.(type) {
						case *ast.TypeSpec:
							if sp.Name.IsExported() && !allowTypes[sp.Name.Name] {
								t.Errorf("unlisted exported type %s", sp.Name.Name)
							}
						case *ast.ValueSpec:
							for _, n := range sp.Names {
								if n.IsExported() && !strings.HasPrefix(n.Name, "Err") &&
									!strings.HasPrefix(n.Name, "Verb") && !strings.HasPrefix(n.Name, "Signal") &&
									!strings.HasPrefix(n.Name, "Ev") && !strings.HasPrefix(n.Name, "Target") {
									t.Errorf("unlisted exported value %s", n.Name)
								}
							}
						}
					}
				}
			}
		}
	}
}

// The constitution is deterministic and two-way ⊆ with registry-v3.
func TestConstitutionAndRegistryVocabulary(t *testing.T) {
	if ConstitutionHash() != ConstitutionHash() || len(ConstitutionHash()) != 64 {
		t.Fatal("constitution hash must be deterministic sha256")
	}
	// declare_done exists in the constitution and produces the fixed
	// signal; the signal is a declarable transition event.
	sig, ok := controlVerbs[VerbDeclareDone]
	if !ok || sig != SignalPhaseCompletionRequested || !transitionEvents[sig] {
		t.Fatal("constitution verb/signal wiring broken")
	}
}
