package state

// L6 operational proof, run 1 (Q-L6-9 Register C): a full live
// L1→L5 task with L6 wired — instruction resolution recorded, a real
// local model driving an authorized mutation through L4 inside a
// provisioned L5 environment, every floor event synchronously
// persisted with record-before-effect ordering — then BYTE-EXACT cold
// reconstruction from durable state alone. Skips hermetically without
// a local model endpoint. (Run 2, the kill proof, is
// TestRealKillRecovery.)

import (
	stdctx "context"
	"encoding/json"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/tofchaliss/themis/execution"
	"github.com/tofchaliss/themis/instructions"
	"github.com/tofchaliss/themis/runtime/model"
	"github.com/tofchaliss/themis/tools"
)

func TestLiveTaskReconstruction(t *testing.T) {
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
	gitBin := "/usr/bin/git"
	if _, err := os.Stat(gitBin); err != nil {
		t.Skip("no pinned git")
	}
	repoRoot := "../../.."

	// --- L1: resolve shipped instructions; record the governed load.
	policy, err := instructions.LoadPolicy(filepath.Join(repoRoot, "policies/security/instruction-directive-patterns.json"))
	if err != nil {
		t.Fatal(err)
	}
	l1res, err := instructions.Load(instructions.Config{Policy: policy},
		instructions.Source{Kind: instructions.ScopeHarnessSafety, Root: filepath.Join(repoRoot, "instructions/global/safety")},
		instructions.Source{Kind: instructions.ScopeHarnessSystem, Root: filepath.Join(repoRoot, "instructions/global/system")},
	)
	if err != nil {
		t.Fatal(err)
	}

	// --- L5: provision at a pin from a local mirror.
	mirrorRoot := t.TempDir()
	mkRepo := func() string {
		repo := filepath.Join(mirrorRoot, "demo")
		run := func(args ...string) string {
			cmd := exec.Command(gitBin, append([]string{"-c", "user.name=t", "-c", "user.email=t@t"}, args...)...)
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
		if err := os.WriteFile(filepath.Join(repo, "parser.go"), []byte("package parser // L6-SENTINEL\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		run("add", ".")
		run("commit", "-q", "-m", "seed")
		return strings.TrimSpace(run("rev-parse", "HEAD"))
	}
	sha := mkRepo()
	writeJSON := func(name, body string) string {
		p := filepath.Join(t.TempDir(), name)
		if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
		return p
	}
	ceiling, err := execution.LoadCeiling(writeJSON("ceiling.json",
		`{"version":1,"mirror_root":"`+mirrorRoot+`","max_wall_deadline_sec":120,"max_file_bytes":1048576,"max_total_bytes":10485760,"max_file_count":500,"max_mem_bytes":1073741824,"max_cpu_time_sec":600,"max_proc_count":64}`))
	if err != nil {
		t.Fatal(err)
	}
	spec, err := execution.LoadSpec(writeJSON("spec.json",
		`{"version":1,"task_id":"L6-LIVE","repo":"demo","pinned_sha":"`+sha+`","limits":[{"dimension":"wall_deadline_s","value":60}]}`))
	if err != nil {
		t.Fatal(err)
	}
	provider, err := execution.NewLocalProvider(gitBin, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	env, err := provider.Provision(ceiling, spec)
	if err != nil {
		t.Fatal(err)
	}
	ws := env.Workspace()

	// --- L4 wiring.
	reg, err := tools.LoadRegistry(filepath.Join(repoRoot, "policies/tools/registry-v2.json"))
	if err != nil {
		t.Fatal(err)
	}
	grant, err := tools.LoadGrant(writeJSON("grant.json",
		`{"version":1,"task_id":"L6-LIVE","total_max_calls":20,"entries":[{"tool":"write_file","max_calls":5,"workspace":`+jstr(ws.Root)+`,"mutating":true}]}`))
	if err != nil {
		t.Fatal(err)
	}
	table, err := tools.NewExecutorTable(reg, nil)
	if err != nil {
		t.Fatal(err)
	}

	// --- L6: state root, disjointness checked at assembly (Q-L6-4).
	stateRoot := filepath.Join(t.TempDir(), "state")
	artifactStoreDir := filepath.Join(t.TempDir(), "artifacts")
	for _, d := range []string{stateRoot, artifactStoreDir} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	if err := CheckDisjointRoots(stateRoot, artifactStoreDir, mirrorRoot, ws.Root); err != nil {
		t.Fatal(err)
	}
	root, err := OpenRoot(stateRoot)
	if err != nil {
		t.Fatal(err)
	}
	task, err := root.CreateTask("L6-LIVE", TaskOptions{GovernedHashes: map[string]string{
		"registry": reg.Hash, "grant": grant.Hash, "ceiling": ceiling.Hash,
		"spec": spec.Hash, "l1_policy": l1res.PolicyHash,
	}})
	if err != nil {
		t.Fatal(err)
	}
	if err := task.Transition(StatusRunning, "provisioned"); err != nil {
		t.Fatal(err)
	}
	var emitted []Event
	must := func(ev Event, err error) Event {
		t.Helper()
		if err != nil {
			t.Fatal(err)
		}
		emitted = append(emitted, ev)
		return ev
	}
	l1body, _ := json.Marshal(map[string]any{"instructions": len(l1res.Instructions), "conflicts": len(l1res.Conflicts), "policy": l1res.PolicyHash})
	must(task.AppendEvent(EvL1Conflict, "l1", l1body))

	// Record-before-effect for the payload leg (D-L6-10; security
	// review): the delivery record commits BEFORE the prompt reaches
	// the model.
	prompt := "Use the write_file tool to create a file named notes.md whose content is exactly: L6 proof"
	promptObj, err := task.StoreObject(ObjEvidencePayload, []byte(prompt))
	if err != nil {
		t.Fatal(err)
	}
	delivery, _ := json.Marshal(map[string]string{"payload": promptObj})
	must(task.AppendEvent(EvL2Delivery, "l2", delivery, Ref{ID: promptObj, Class: ObjEvidencePayload}))

	// --- Live model call driving the mutation.
	mp := model.NewOllamaChat(endpoint)
	ctx, cancel := stdctx.WithTimeout(stdctx.Background(), 120*time.Second)
	defer cancel()
	resp, err := mp.Execute(ctx, model.ExecutionRequest{
		Model: modelName,
		Messages: []model.Message{{Role: model.RoleUser, Content: prompt}},
		Tools: []model.ToolDef{{Name: "write_file", Description: "Create one file in the workspace.",
			Parameters: json.RawMessage(`{"type":"object","properties":{"path":{"type":"string"},"content":{"type":"string"}},"required":["path","content"]}`)}},
		Options: model.DefaultOptions()})
	if err != nil || resp.Termination != model.TerminationToolCalls || len(resp.ToolCalls) == 0 {
		t.Fatalf("live model must emit a tool call: %v %+v", err, resp.Termination)
	}
	call := resp.ToolCalls[0]
	msg, ev4, audit := tools.Handle(reg, grant, table, call, tools.CallState{Calls: map[string]int{}})
	if audit.Decision != "authorized" || ev4 == nil {
		t.Fatalf("live mutation must authorize: %+v", audit)
	}
	// Record-before-effect (D-L6-10): evidence object + audit event
	// are durably committed BEFORE the result message would re-enter
	// the model loop.
	evObj, err := task.StoreObject(ObjEvidencePayload, ev4.Evidence)
	if err != nil {
		t.Fatal(err)
	}
	auditBody, _ := json.Marshal(audit)
	must(task.AppendEvent(EvL4Audit, "l4", auditBody, Ref{ID: evObj, Class: ObjEvidencePayload}))
	_ = msg // the result may now proceed to the model loop

	// --- Seal, egress, artifact into BOTH stores; bind the address.
	if err := env.Seal(execution.SealTaskComplete); err != nil {
		t.Fatal(err)
	}
	l5store, err := execution.NewArtifactStore(artifactStoreDir)
	if err != nil {
		t.Fatal(err)
	}
	addr, err := env.Egress(ceiling, spec, l5store)
	if err != nil {
		t.Fatalf("egress: %v", err)
	}
	artifactBytes, err := l5store.Get(addr)
	if err != nil {
		t.Fatal(err)
	}
	artObj, err := task.StoreObject(ObjEgressArtifact, artifactBytes)
	if err != nil {
		t.Fatal(err)
	}
	if err := task.BindArtifact(artObj); err != nil {
		t.Fatal(err)
	}
	for _, trn := range env.Trace().Transitions {
		tb, _ := json.Marshal(trn)
		must(task.AppendEvent(EvL5Transition, "l5", tb))
	}
	if st := env.Teardown(); st != execution.StateDestroyed {
		t.Fatalf("teardown: %s", st)
	}
	if err := task.Transition(StatusCompleted, "task complete"); err != nil {
		t.Fatal(err)
	}
	liveEvents, err := root.ReadEvents("L6-LIVE")
	if err != nil {
		t.Fatal(err)
	}
	task.Close()
	// The in-memory events captured at emit must appear cold
	// byte-identically (test review: live-to-durable fidelity, not
	// just parse determinism).
	byseq := map[int64]Event{}
	for _, ev := range liveEvents {
		byseq[ev.Seq] = ev
	}
	for _, em := range emitted {
		cold, ok := byseq[em.Seq]
		if !ok {
			t.Fatalf("emitted event %d missing cold", em.Seq)
		}
		lb, _ := json.Marshal(em)
		cb, _ := json.Marshal(cold)
		if string(lb) != string(cb) {
			t.Fatalf("event %d differs: live-emitted vs cold", em.Seq)
		}
	}

	// --- BYTE-EXACT cold reconstruction from durable state alone.
	cold, err := OpenRoot(stateRoot)
	if err != nil {
		t.Fatal(err)
	}
	v := cold.Verify("L6-LIVE")
	if v.Verdict != VerdictVerified {
		t.Fatalf("cold verification: %+v", v)
	}
	coldEvents, err := cold.ReadEvents("L6-LIVE")
	if err != nil || len(coldEvents) != len(liveEvents) {
		t.Fatalf("cold event count mismatch: %d vs %d", len(coldEvents), len(liveEvents))
	}
	for i := range coldEvents {
		lb, _ := json.Marshal(liveEvents[i])
		cb, _ := json.Marshal(coldEvents[i])
		if string(lb) != string(cb) {
			t.Fatalf("event %d differs cold vs live", i)
		}
	}
	// The evidence bytes round-trip through the object plane exactly,
	// reached only through the event that carries their provenance.
	var coldAudit *Event
	for i := range coldEvents {
		if coldEvents[i].Class == EvL4Audit {
			coldAudit = &coldEvents[i]
		}
	}
	if coldAudit == nil {
		t.Fatal("audit event missing cold")
	}
	bytesBack, err := cold.Resolve(*coldAudit, 0)
	if err != nil || string(bytesBack) != string(ev4.Evidence) {
		t.Fatalf("evidence must reconstruct byte-exactly: %v", err)
	}
	// The artifact reconstructs and re-verifies against its identity.
	view, err := cold.ReadStatus("L6-LIVE")
	if err != nil || view.Status != StatusCompleted || view.Verdict != VerdictVerified || len(view.ArtifactAddrs) != 1 {
		t.Fatalf("status view: %+v %v", view, err)
	}
	artBack, err := cold.Store().GetObject(view.ArtifactAddrs[0])
	if err != nil || string(artBack) != string(artifactBytes) {
		t.Fatalf("artifact must reconstruct byte-exactly: %v", err)
	}
	man, _ := cold.ReadManifest("L6-LIVE")
	if man.GovernedHashes["registry"] != reg.Hash || man.ConstitutionHash != ConstitutionHash() {
		t.Fatal("governed attribution must survive cold")
	}
	t.Logf("live L6 proof: model=%s events=%d artifact=%s verdict=%s — byte-exact cold reconstruction",
		modelName, len(coldEvents), artObj[:19], view.Verdict)
}

func jstr(s string) string {
	b, _ := json.Marshal(s)
	return string(b)
}
