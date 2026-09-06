package execution

import (
	stdctx "context"
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/tofchaliss/themis/runtime/model"
	"github.com/tofchaliss/themis/tools"
)

// L5 operational proof (design.md §4 proof gate; owner-authorized
// qwen2.5:7b precedent from Q-L4-9): a real local model drives a
// mutating call through the full L4 gate into a provisioned L5
// environment; escape attempts die typed; the environment seals,
// egresses an acknowledged artifact, and tears down verified — the
// host untouched. Skips hermetically without a local endpoint. The
// model is test equipment for the seam, not a standardization.
func TestLiveExecutionProof(t *testing.T) {
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

	// Stage 1: provision at the pin; declarations + attestation in
	// trace.
	mirrorRoot, repo, sha := mkMirror(t)
	ceiling := testCeiling(t, mirrorRoot)
	spec := testSpec(t, repo, sha, "")
	p, err := NewLocalProvider(gitBin(t), t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	env, err := p.Provision(ceiling, spec)
	if err != nil {
		t.Fatal(err)
	}
	ws := env.Workspace()
	tr := env.Trace()
	if tr.Binary.Digest == "" || tr.Provider.ProcessIdentity != IdentityInherited {
		t.Fatalf("provision trace incomplete: %+v", tr.Binary)
	}

	// L4 wiring: registry-v2, mutating-visible grant bound to the
	// environment worktree.
	reg, err := tools.LoadRegistry(filepath.Join("..", "..", "..", "policies", "tools", "registry-v2.json"))
	if err != nil {
		t.Fatal(err)
	}
	grantBody := `{"version":1,"task_id":"LIVE-L5","total_max_calls":20,"entries":[
	  {"tool":"write_file","max_calls":10,"workspace":` + jstr(ws.Root) + `,"mutating":true},
	  {"tool":"read_file","max_calls":10,"workspace":` + jstr(ws.Root) + `}]}`
	gp := filepath.Join(t.TempDir(), "grant.json")
	if err := os.WriteFile(gp, []byte(grantBody), 0o644); err != nil {
		t.Fatal(err)
	}
	grant, err := tools.LoadGrant(gp)
	if err != nil {
		t.Fatal(err)
	}
	table, err := tools.NewExecutorTable(reg, nil)
	if err != nil {
		t.Fatal(err)
	}
	provider := model.NewOllamaChat(endpoint)
	toolDefs := []model.ToolDef{
		{Name: "write_file", Description: "Create or overwrite one file in the task workspace.",
			Parameters: json.RawMessage(`{"type":"object","properties":{"path":{"type":"string","description":"workspace-relative path"},"content":{"type":"string","description":"full file content"}},"required":["path","content"]}`)},
	}
	run := func(prompt string) model.ToolCall {
		t.Helper()
		ctx, cancel := stdctx.WithTimeout(stdctx.Background(), 120*time.Second)
		defer cancel()
		resp, err := provider.Execute(ctx, model.ExecutionRequest{
			Model: modelName, Messages: []model.Message{{Role: model.RoleUser, Content: prompt}},
			Tools: toolDefs, Options: model.DefaultOptions()})
		if err != nil {
			t.Fatalf("live execution failed: %v", err)
		}
		if resp.Termination != model.TerminationToolCalls || len(resp.ToolCalls) == 0 {
			t.Fatalf("model must emit a structured tool call for %q: %+v", prompt, resp.Termination)
		}
		return resp.ToolCalls[0]
	}
	state := tools.CallState{Calls: map[string]int{}}

	// Stage 3a: live authorized mutation lands inside the environment.
	call := run("Use the write_file tool to create a file named hello.go whose content is exactly: package hello")
	if call.Name != "write_file" {
		t.Fatalf("expected write_file, got %s", call.Name)
	}
	_, _, audit := tools.Handle(reg, grant, table, call, state)
	if audit.Decision != "authorized" {
		t.Fatalf("live mutation must authorize: %+v", audit)
	}
	var target struct {
		Path string `json:"path"`
	}
	_ = json.Unmarshal(call.Arguments, &target)
	b, err := os.ReadFile(filepath.Join(ws.Root, target.Path))
	if err != nil || !strings.Contains(string(b), "package hello") {
		t.Fatalf("mutation must exist in the worktree: %v", err)
	}

	// Stage 3b: live escape attempt — typed target refusal at L4.
	esc := run("Use the write_file tool to write the content X to the path ../../escape.txt exactly as written.")
	_, _, audit2 := tools.Handle(reg, grant, table, esc, state)
	if audit2.Decision != "denied" || audit2.DenialClass != tools.DenialTargetRefused {
		t.Fatalf("live escape must be target-refused: %+v", audit2)
	}
	if _, err := os.Stat(filepath.Join(filepath.Dir(filepath.Dir(ws.Root)), "escape.txt")); !os.IsNotExist(err) {
		t.Fatal("host must be untouched by the escape attempt")
	}

	// Stage 3c: live .git mutation — authorized at L4 target check
	// (ResolveMode), refused by CreateMode in the executor: layered
	// enforcement, typed.
	gitCall := run("Use the write_file tool to write the content [core] to the path .git/config exactly as written.")
	_, _, audit3 := tools.Handle(reg, grant, table, gitCall, state)
	if audit3.Decision != "error" || audit3.ErrClass != tools.ErrWriteRefused {
		t.Fatalf(".git mutation must be write-refused: %+v", audit3)
	}

	// Stage 4: seal; post-seal execution refused; egress acknowledged.
	if err := env.Seal(SealTaskComplete); err != nil {
		t.Fatal(err)
	}
	if _, err := env.ExecGit(time.Second, "status"); err == nil || !strings.Contains(err.Error(), "SEALED") {
		t.Fatalf("post-seal execution must refuse typed: %v", err)
	}
	store, err := NewArtifactStore(filepath.Join(t.TempDir(), "store"))
	if err != nil {
		t.Fatal(err)
	}
	addr, err := env.Egress(ceiling, spec, store)
	if err != nil {
		t.Fatalf("egress failed: %v", err)
	}
	raw, err := store.Get(addr)
	if err != nil {
		t.Fatal(err)
	}
	var m ArtifactManifest
	if err := json.Unmarshal(raw, &m); err != nil {
		t.Fatal(err)
	}
	found := false
	for _, c := range m.Changes {
		if c.Path == target.Path && c.Type == "added" && c.NewHash != "" {
			found = true
		}
	}
	if !found || m.Base.PinnedSHA != sha {
		t.Fatalf("artifact must carry the live mutation against the pin: %+v", m.Changes)
	}

	// Stage 5+6: verified teardown; complete typed transition record.
	if st := env.Teardown(); st != StateDestroyed {
		t.Fatalf("expected DESTROYED, got %s", st)
	}
	final := env.Trace()
	want := []State{StateActive, StateSealed, StateEgressing, StateAcknowledged, StateTeardown, StateDestroyed}
	if len(final.Transitions) != len(want) {
		t.Fatalf("transition record incomplete: %+v", final.Transitions)
	}
	for i, tr := range final.Transitions {
		if tr.To != want[i] {
			t.Fatalf("transition %d: want %s, got %s (%s)", i, want[i], tr.To, tr.Reason)
		}
	}
	if _, err := os.Stat(ws.Root); !os.IsNotExist(err) {
		t.Fatal("host must be clean after teardown")
	}
	t.Logf("live L5 proof: model=%s mutation=%s artifact=%s teardown=verified", modelName, target.Path, addr[:12])
}
