package seam

// L10-M6: the closure demonstration with the REAL seam — the model
// (scripted) writes a report, proposes verification under the
// PROPOSED contract, the real evaluator resolves/canonicalizes/maps,
// the gate opens on PASS through the unmodified production loop, and
// cold reconstruction re-derives the same historical outcome.
// (The Register E live-model variant runs the same lattice against a
// local model when one is present.)

import (
	stdctx "context"
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/tofchaliss/themis/orchestration"
	"github.com/tofchaliss/themis/runtime/model"
	"github.com/tofchaliss/themis/state"
	"github.com/tofchaliss/themis/tools"
)

type scripted struct {
	steps []model.ExecutionResponse
	i     int
}

func (s *scripted) Name() string { return "scripted" }
func (s *scripted) Execute(ctx stdctx.Context, req model.ExecutionRequest) (*model.ExecutionResponse, error) {
	if s.i >= len(s.steps) {
		return &model.ExecutionResponse{Content: "…", Termination: model.TerminationStop}, nil
	}
	r := s.steps[s.i]
	s.i++
	return &r, nil
}

func call(name, args string) model.ExecutionResponse {
	return model.ExecutionResponse{Termination: model.TerminationToolCalls,
		ToolCalls: []model.ToolCall{{ID: "c", Name: name, Arguments: json.RawMessage(args)}}}
}

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
	if err := os.WriteFile(filepath.Join(repo, "go.mod"), []byte("module demo // vulnerable-dep v1\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	run("add", ".")
	run("commit", "-q", "-m", "seed")
	return root, strings.TrimSpace(run("rev-parse", "HEAD"))
}

func wj(t *testing.T, dir, name, body string) string {
	t.Helper()
	p := filepath.Join(dir, name)
	if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	return p
}

func e2eFixture(t *testing.T, m model.Interface) (*orchestration.Orchestrator, string, *state.Root, string) {
	t.Helper()
	root := repoRoot(t)
	envDir := t.TempDir()
	stateDir := filepath.Join(t.TempDir(), "state")
	mirror, sha := mkMirror(t)

	l4, err := tools.LoadRegistry(filepath.Join(root, "policies/tools/registry-v4.proposed.json"))
	if err != nil {
		t.Fatal(err)
	}
	ev := &Evaluator{
		RegistryPath: filepath.Join(root, "policies/verification/contracts.proposed.json"),
		L4:           l4,
	}

	o, _, err := orchestration.Open(orchestration.Config{
		StateRoot: stateDir, ArtifactDir: filepath.Join(t.TempDir(), "artifacts"),
		GitPath: gitBin(t), ProviderDir: t.TempDir(),
		SafetyRoot: filepath.Join(root, "instructions/global/safety"),
		SystemRoot: filepath.Join(root, "instructions/global/system"),
		PolicyPath: filepath.Join(root, "policies/security/instruction-directive-patterns.json"),
		Model:      m,
		Verifier:   ev,
	})
	if err != nil {
		t.Fatal(err)
	}

	// The remediate-dependency lattice: REMEDIATE (write) → gate on
	// report-valid@1 PASS at declare_done.
	wj(t, envDir, "workflow.json", `{
 "version":1,"name":"remediate-dependency","initial":"REMEDIATE",
 "declared_events":["turn-no-action","turn-provider-error","turns-exhausted","tool-error",
   "signal:phase-completion-requested",
   "verification-pass","verification-fail","verification-inconclusive",
   "verification-unavailable","verification-invalid"],
 "phases":[
  {"name":"REMEDIATE","capabilities":["read_file","write_file","verify_report","declare_done"],"max_model_turns":8,"edges":[
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
 ]}`)
	wj(t, envDir, "wceiling.json", `{"version":1,"allowed_tools":["read_file","write_file","verify_report","declare_done"],"max_total_calls":30,"max_walk_length":200,"max_turns_per_phase":10}`)
	wj(t, envDir, "eceiling.json",
		`{"version":1,"mirror_root":"`+mirror+`","max_wall_deadline_sec":600,"max_file_bytes":1048576,"max_total_bytes":10485760,"max_file_count":500,"max_mem_bytes":1073741824,"max_cpu_time_sec":600,"max_proc_count":64}`)
	wj(t, envDir, "spec.json",
		`{"version":1,"task_id":"remediate-1","repo":"demo","pinned_sha":"`+sha+`","limits":[{"dimension":"wall_deadline_s","value":120}]}`)
	wj(t, envDir, "grant.json",
		`{"version":1,"task_id":"remediate-1","total_max_calls":30,"entries":[
		  {"tool":"read_file","max_calls":8,"workspace":"@workspace"},
		  {"tool":"write_file","max_calls":4,"workspace":"@workspace","mutating":true},
		  {"tool":"verify_report","max_calls":6,"workspace":"@workspace"},
		  {"tool":"declare_done","max_calls":6}]}`)
	wj(t, envDir, "context-contract.json",
		`{"version":1,"workflow":"remediate-dependency","slots":[
		  {"name":"task-payload","kind":"task-brief","requirement":"required","classes":["external-untrusted"]}],
		  "sensitivity_ceiling":"public"}`)
	js := func(s string) string { b, _ := json.Marshal(s); return string(b) }
	env := wj(t, envDir, "envelope.json", `{
	 "version":1,"task_id":"remediate-1","model":"scripted","turn_timeout_sec":180,
	 "payload":"Remediate the vulnerable dependency, write report.json, verify it, then declare done.",
	 "workflow_path":`+js(filepath.Join(envDir, "workflow.json"))+`,
	 "workflow_ceiling_path":`+js(filepath.Join(envDir, "wceiling.json"))+`,
	 "registry_path":`+js(filepath.Join(root, "policies/tools/registry-v4.proposed.json"))+`,
	 "grant_path":`+js(filepath.Join(envDir, "grant.json"))+`,
	 "exec_ceiling_path":`+js(filepath.Join(envDir, "eceiling.json"))+`,
	 "spec_path":`+js(filepath.Join(envDir, "spec.json"))+`,
	 "context_contract_path":`+js(filepath.Join(envDir, "context-contract.json"))+`}`)

	sroot, err := state.OpenRoot(stateDir)
	if err != nil {
		t.Fatal(err)
	}
	return o, env, sroot, stateDir
}

func TestRemediateDependencyE2E(t *testing.T) {
	report := `{"finding": "vulnerable-dep v1 in go.mod", "remediation": "bumped to v2 (workspace edit)", "evidence": "go.mod updated"}`
	m := &scripted{steps: []model.ExecutionResponse{
		call("write_file", `{"path":"report.json","content":`+string(mustJSON(report))+`}`),
		call("verify_report", `{"path":"report.json","contract":"report-valid@1"}`),
		call("declare_done", `{}`),
	}}
	o, env, sroot, _ := e2eFixture(t, m)

	res, err := o.SubmitTask(env)
	if err != nil {
		t.Fatal(err)
	}
	if res.Status != state.StatusCompleted {
		t.Fatalf("status = %s, want COMPLETED", res.Status)
	}

	// The real evaluation is durable history with the PROPOSED
	// contract's token.
	view, err := TaskVerificationHistory(sroot, "remediate-1")
	if err != nil {
		t.Fatal(err)
	}
	if view.Latest["report-valid@1"] != "PASS" {
		t.Fatalf("latest projection %v", view.Latest)
	}

	// Cold reconstruction: same historical outcome, no discrepancy,
	// nothing rewritten.
	reports, artifacts, err := ReconstructTask(sroot, "remediate-1")
	if err != nil {
		t.Fatal(err)
	}
	if len(reports) != 1 || !reports[0].Consistent {
		t.Fatalf("reconstruction: %+v", reports)
	}
	if len(artifacts) != 0 {
		t.Error("consistent walk must yield no discrepancy artifacts")
	}
	if reports[0].Evaluation.Outcome != "PASS" {
		t.Errorf("reconstructed outcome %s", reports[0].Evaluation.Outcome)
	}
}

func TestRemediateDependencyFailClosesGate(t *testing.T) {
	// The report is missing a required field: the REAL contract maps
	// report_invalid -> FAIL; the gate never opens; the reviewed
	// exhaustion path governs.
	bad := `{"finding": "x", "remediation": "y"}`
	m := &scripted{steps: []model.ExecutionResponse{
		call("write_file", `{"path":"report.json","content":`+string(mustJSON(bad))+`}`),
		call("verify_report", `{"path":"report.json","contract":"report-valid@1"}`),
		call("declare_done", `{}`),
		call("declare_done", `{}`),
		call("declare_done", `{}`),
		call("declare_done", `{}`),
	}}
	o, env, sroot, _ := e2eFixture(t, m)

	res, err := o.SubmitTask(env)
	if err != nil {
		t.Fatal(err)
	}
	if res.Status != state.StatusFailed {
		t.Fatalf("status = %s, want FAILED — an invalid report must not complete", res.Status)
	}
	view, err := TaskVerificationHistory(sroot, "remediate-1")
	if err != nil {
		t.Fatal(err)
	}
	if view.Latest["report-valid@1"] != "FAIL" {
		t.Fatalf("latest projection %v", view.Latest)
	}
}

func mustJSON(s string) []byte {
	b, err := json.Marshal(s)
	if err != nil {
		panic(err)
	}
	return b
}

var _ = errors.New // keep imports tidy under edits

// TestLiveRemediateWalk — Register E proper: the remediate-dependency
// lattice, the REAL seam, and a real local model through the
// unmodified production loop. Deliberately boring: the model writes
// the report, requests verification under the registered contract,
// and the gate opens only on the L10 PASS. Skips when no local model
// endpoint answers.
func TestLiveRemediateWalk(t *testing.T) {
	endpoint := os.Getenv("THEMIS_LIVE_OLLAMA")
	if endpoint == "" {
		endpoint = "http://localhost:11434"
	}
	client := http.Client{Timeout: 2 * time.Second}
	if _, err := client.Get(endpoint + "/api/tags"); err != nil {
		t.Skipf("no local model endpoint at %s: %v", endpoint, err)
	}
	modelName := os.Getenv("THEMIS_LIVE_TOOL_MODEL")
	if modelName == "" {
		modelName = "qwen2.5:7b"
	}

	o, env, sroot, _ := e2eFixture(t, model.NewOllamaChat(endpoint))
	// Rewrite the envelope for the live model with a payload that
	// spells out the method (the scripted fixture's payload is terse).
	raw, err := os.ReadFile(env)
	if err != nil {
		t.Fatal(err)
	}
	reportContent := `{"finding": "vulnerable dependency v1 pinned in go.mod", "remediation": "updated the dependency to the fixed version", "evidence": "go.mod change in this workspace"}`
	payload := "Do these three tool calls in order, one per turn, and never answer in plain text. " +
		"TURN 1: call write_file with exactly two arguments: path = report.json and content = " + reportContent +
		" (pass that JSON object as the content string; write_file accepts ONLY path and content). " +
		"TURN 2: call verify_report with exactly two arguments: path = report.json and contract = report-valid@1. " +
		"TURN 3: after the verification message says PASS, call declare_done with no arguments at all."
	body := strings.Replace(string(raw), `"model":"scripted"`, `"model":`+string(mustJSON(modelName)), 1)
	body = strings.Replace(body,
		`"payload":"Remediate the vulnerable dependency, write report.json, verify it, then declare done."`,
		`"payload":`+string(mustJSON(payload)), 1)
	if err := os.WriteFile(env, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}

	res, err := o.SubmitTask(env)
	if err != nil {
		t.Fatal(err)
	}
	if res.Status != state.StatusCompleted {
		view, _ := TaskVerificationHistory(sroot, "remediate-1")
		evs, _ := sroot.ReadEvents("remediate-1")
		for _, e := range evs {
			switch e.Class {
			case state.EvModelTurn, state.EvL4Audit, state.EvVerification, state.EvWorkflowTransition:
				t.Logf("seq=%d class=%s body=%s", e.Seq, e.Class, string(e.Body))
			}
		}
		t.Fatalf("live walk status = %s (verifications: %v)", res.Status, view.Latest)
	}

	view, err := TaskVerificationHistory(sroot, "remediate-1")
	if err != nil {
		t.Fatal(err)
	}
	if view.Latest["report-valid@1"] != "PASS" {
		t.Fatalf("live walk completed without a PASS: %v", view.Latest)
	}
	reports, artifacts, err := ReconstructTask(sroot, "remediate-1")
	if err != nil {
		t.Fatal(err)
	}
	for _, r := range reports {
		if !r.Consistent {
			t.Fatalf("live evaluation failed cold reconstruction: %+v", r)
		}
	}
	_ = artifacts
	t.Logf("LIVE PASS: %d evaluation(s), all reconstruct consistent", len(reports))
}
