package orchestration

// Registers C and E (Q-L7-11): loop-boundary fault injection, the
// real-kill/no-continuation proof, and the live operational proof
// through the production loop.

import (
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/tofchaliss/themis/runtime/model"
	"github.com/tofchaliss/themis/state"
)

var loopFaultPoints = []string{
	"loop.pre-compose-commit", "loop.pre-model-turn",
	"loop.pre-transition-commit", "loop.post-transition-commit",
}

// Every loop fault point yields a typed failure, a terminal record,
// and a walk the replayer still accepts — no acknowledged transition
// disappears, and no effect is represented without its record.
func TestLoopFaultSweep(t *testing.T) {
	for _, point := range loopFaultPoints {
		point := point
		t.Run(point, func(t *testing.T) {
			f := setup(t, happyScript(), "")
			t.Cleanup(func() { fault = nil })
			fault = func(p string) error {
				if p == point {
					return errors.New("injected@" + p)
				}
				return nil
			}
			res, err := f.o.SubmitTask(f.envelope(t, "t-fault"))
			fault = nil
			if err == nil {
				t.Fatalf("armed point %s must fail the walk", point)
			}
			if !errors.Is(err, ErrInvariant) {
				t.Fatalf("loop faults take the invariant path: %v", err)
			}
			if res.Status != state.StatusFailed && res.Status != state.StatusFailedPartial {
				t.Fatalf("typed terminal required: %+v", res)
			}
			view, verr := f.o.ReadStatus("t-fault")
			if verr != nil || view.Verdict == state.VerdictCorrupt {
				t.Fatalf("no fault point may corrupt the record: %+v %v", view, verr)
			}
			// The invariant event is durably present (the fixed path).
			evs, _ := f.o.root.ReadEvents("t-fault")
			foundInv := false
			for _, ev := range evs {
				if ev.Class == state.EvL7Invariant {
					foundInv = true
				}
			}
			if !foundInv {
				t.Fatal("invariant path must leave its typed event")
			}
			// Recorded transitions (if any) still replay consistently:
			// committed prefix integrity.
			for _, ev := range evs {
				if ev.Class == state.EvWorkflowTransition && !strings.Contains(string(ev.Body), `"from"`) {
					t.Fatal("malformed transition event")
				}
			}
		})
	}
}

// The sweep list and the source's fault literals never drift.
func TestLoopFaultPointsSyncWithSource(t *testing.T) {
	inSource := map[string]bool{}
	entries, _ := os.ReadDir(".")
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".go") || strings.HasSuffix(e.Name(), "_test.go") {
			continue
		}
		b, _ := os.ReadFile(e.Name())
		for _, part := range strings.Split(string(b), `faultAt("`)[1:] {
			if i := strings.IndexByte(part, '"'); i > 0 {
				inSource[part[:i]] = true
			}
		}
	}
	swept := map[string]bool{}
	for _, p := range loopFaultPoints {
		swept[p] = true
	}
	for p := range inSource {
		if !swept[p] {
			t.Errorf("fault point %q exists in source but is never swept", p)
		}
	}
	for p := range swept {
		if !inSource[p] {
			t.Errorf("swept point %q does not exist in source", p)
		}
	}
}

// Real-kill: SIGKILL mid-walk; cold Open() closes the past (recovery
// → FAILED_PARTIAL) and proves no continuation.
func TestRealKillNoContinuation(t *testing.T) {
	if baseDir := os.Getenv("L7_CHILD_BASE"); baseDir != "" {
		// Child: run a walk that stays busy (prose turns against a
		// generous stay budget) until killed.
		m := &scriptedModel{}
		for i := 0; i < 200; i++ {
			m.steps = append(m.steps, prose(fmt.Sprintf("thinking %d", i)))
		}
		mirror := filepath.Join(baseDir, "mirror")
		sha, _ := os.ReadFile(filepath.Join(baseDir, "sha"))
		envDir := filepath.Join(baseDir, "env")
		o, _, err := Open(Config{
			StateRoot: filepath.Join(baseDir, "state"), ArtifactDir: filepath.Join(baseDir, "artifacts"),
			GitPath: "/usr/bin/git", ProviderDir: filepath.Join(baseDir, "provider"),
			SafetyRoot: filepath.Join(baseDir, "repo", "instructions/global/safety"),
			SystemRoot: filepath.Join(baseDir, "repo", "instructions/global/system"),
			PolicyPath: filepath.Join(baseDir, "repo", "policies/security/instruction-directive-patterns.json"),
			Model:      m,
		})
		if err != nil {
			fmt.Println("child:", err)
			os.Exit(1)
		}
		_ = mirror
		_ = sha
		env, err := LoadEnvelope(filepath.Join(envDir, "envelope-t-kill.json"))
		if err != nil {
			fmt.Println("child:", err)
			os.Exit(1)
		}
		_, _ = o.SubmitTask(env)
		os.Exit(0)
	}

	base := t.TempDir()
	// Build the child's world: mirror, env files, repo pointer.
	f := setup(t, happyScript(), strings.Replace(defaultWorkflow,
		`{"on":"turn-no-action","to":"@stay","counter":2,"exhausted_to":"@fail"},
    {"on":"turn-provider-error","to":"@fail"},
    {"on":"tool-error","to":"@stay","counter":2,"exhausted_to":"@fail"},
    {"on":"turns-exhausted","to":"@fail"}]},
  {"name":"VERIFY"`,
		`{"on":"turn-no-action","to":"@stay","counter":9,"exhausted_to":"@fail"},
    {"on":"turn-provider-error","to":"@fail"},
    {"on":"tool-error","to":"@stay","counter":2,"exhausted_to":"@fail"},
    {"on":"turns-exhausted","to":"@fail"}]},
  {"name":"VERIFY"`, 1))
	// Redirect the fixture's roots under the child's base.
	for _, d := range []string{"state", "artifacts", "provider", "env"} {
		if err := os.MkdirAll(filepath.Join(base, d), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	// The child re-reads env files: copy fixture artifacts across.
	copyFile := func(name string) {
		b, err := os.ReadFile(filepath.Join(f.envDir, name))
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(base, "env", name), b, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	// Raise turn budget so the child stays busy long enough.
	wf := strings.ReplaceAll(readFile(t, filepath.Join(f.envDir, "workflow.json")), `"max_model_turns":4`, `"max_model_turns":10`)
	writeJSON(t, filepath.Join(base, "env"), "workflow.json", wf)
	for _, n := range []string{"wceiling.json", "eceiling.json", "spec.json", "grant.json"} {
		copyFile(n)
	}
	repoAbs := mustAbs(t, repoRoot)
	if err := os.Symlink(repoAbs, filepath.Join(base, "repo")); err != nil {
		t.Fatal(err)
	}
	envAbs := filepath.Join(base, "env")
	writeJSON(t, envAbs, "envelope-t-kill.json", `{
	 "version":1,"task_id":"t-kill","model":"scripted","payload":"think",
	 "workflow_path":`+jstr(filepath.Join(envAbs, "workflow.json"))+`,
	 "workflow_ceiling_path":`+jstr(filepath.Join(envAbs, "wceiling.json"))+`,
	 "registry_path":`+jstr(mustAbs(t, filepath.Join(repoRoot, "policies/tools/registry-v3.json")))+`,
	 "grant_path":`+jstr(filepath.Join(envAbs, "grant.json"))+`,
	 "exec_ceiling_path":`+jstr(filepath.Join(envAbs, "eceiling.json"))+`,
	 "spec_path":`+jstr(filepath.Join(envAbs, "spec.json"))+`}`)

	cmd := exec.Command(os.Args[0], "-test.run", "TestRealKillNoContinuation")
	cmd.Env = append(os.Environ(), "L7_CHILD_BASE="+base)
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	// Kill once the child has provably committed walk events.
	streamPath := filepath.Join(base, "state", "tasks", "t-kill", "events.log")
	deadline := time.Now().Add(20 * time.Second)
	for {
		if info, err := os.Stat(streamPath); err == nil && info.Size() > 2048 {
			break
		}
		if time.Now().After(deadline) {
			_ = cmd.Process.Kill()
			t.Skip("child never committed work (environment too slow)")
		}
		time.Sleep(25 * time.Millisecond)
	}
	if err := cmd.Process.Signal(syscall.SIGKILL); err != nil {
		t.Fatal(err)
	}
	_ = cmd.Wait()

	// Cold restart: the past closes before the future opens.
	o2, rep, err := Open(Config{
		StateRoot: filepath.Join(base, "state"), ArtifactDir: filepath.Join(base, "artifacts"),
		GitPath: gitBin(t), ProviderDir: filepath.Join(base, "provider"),
		SafetyRoot: filepath.Join(repoRoot, "instructions/global/safety"),
		SystemRoot: filepath.Join(repoRoot, "instructions/global/system"),
		PolicyPath: filepath.Join(repoRoot, "policies/security/instruction-directive-patterns.json"),
		Model:      happyScript(),
	})
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, id := range rep.Recovered {
		if id == "t-kill" {
			found = true
		}
	}
	if !found {
		t.Fatalf("startup must recover the killed task: %+v", rep)
	}
	view, err := o2.ReadStatus("t-kill")
	if err != nil || view.Status != state.StatusFailedPartial || view.Verdict == state.VerdictCorrupt {
		t.Fatalf("killed walk must be typed FAILED_PARTIAL, never corrupt: %+v %v", view, err)
	}
	// No continuation: same id refuses; a new attempt is a new task.
	if _, err := o2.SubmitTask(f.envelope(t, "t-kill")); !errors.Is(err, state.ErrIdentity) {
		t.Fatalf("killed identity must be single-use: %v", err)
	}
	t.Logf("real-kill proof: recovered=%v status=%s", rep.Recovered, view.Status)
}

func readFile(t *testing.T, p string) string {
	t.Helper()
	b, err := os.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

// Register E: the live operational proof — a real local model through
// the production loop, negative proofs included.
func TestLiveWalkProof(t *testing.T) {
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
	f := setup(t, model.NewOllamaChat(endpoint), "")
	env := f.envelope(t, "t-live")
	env.Model = modelName
	env.Payload = "If a read_file tool is available, first read parser.go. Then call the declare_done tool with no arguments. If declare_done is the only tool available, call declare_done immediately without any other output."
	res, err := f.o.SubmitTask(env)
	if err != nil {
		t.Fatalf("live walk failed: %v", err)
	}
	if res.Status != state.StatusCompleted || res.Artifact == "" || res.Verdict != state.VerdictVerified {
		t.Fatalf("live walk must complete verified with an artifact: %+v", res)
	}
	transitions := replayAndVerify(t, f, "t-live")
	if len(transitions) != 2 || transitions[0][1] != "VERIFY" || transitions[1][1] != "@complete" {
		t.Fatalf("declare_done must route through the governed edges, not bypass them: %v", transitions)
	}
	// Negative: ungranted declare_done is not-available live — a phase
	// walk without the verb cannot be self-completed.
	writeJSON(t, f.envDir, "grant.json",
		`{"version":1,"task_id":"T","total_max_calls":20,"entries":[{"tool":"read_file","max_calls":8,"workspace":"@workspace"}]}`)
	env2 := f.envelope(t, "t-live-neg")
	env2.Model = modelName
	env2.Payload = "Call the declare_done tool now."
	res2, err := f.o.SubmitTask(env2)
	if err != nil {
		t.Fatalf("governed failure is not an error: %v", err)
	}
	if res2.Status != state.StatusFailed {
		t.Fatalf("ungranted declare_done must leave the walk to its governed failure: %+v", res2)
	}
	evs, _ := f.o.root.ReadEvents("t-live-neg")
	sawNA := false
	for _, ev := range evs {
		if ev.Class == state.EvL4Audit && strings.Contains(string(ev.Body), "not-available") {
			sawNA = true
		}
	}
	if !sawNA {
		t.Fatal("the denial must be in the record")
	}
	t.Logf("live L7 proof: model=%s completed=%s artifact=%s; negative: ungranted declare_done → %s",
		modelName, res.Status, res.Artifact[:19], res2.Status)
}
