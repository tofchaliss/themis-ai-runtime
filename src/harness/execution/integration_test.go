package execution

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tofchaliss/themis/runtime/model"
	"github.com/tofchaliss/themis/tools"
)

// L5-M2: the L4 decision table re-rooted inside a provisioned
// environment. The environment's worktree IS the confinement root —
// identical L4 verdicts, no second permission system (hard
// invariant: same L4 gate).
func TestL4DecisionTableInsideEnvironment(t *testing.T) {
	mirrorRoot, repo, sha := mkMirror(t)
	ceiling := testCeiling(t, mirrorRoot)
	p, err := NewLocalProvider(gitBin(t), t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	env, err := p.Provision(ceiling, testSpec(t, repo, sha, ""))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = env.Seal(SealCallerAbort); env.Teardown() }()
	ws := env.Workspace()

	reg, err := tools.LoadRegistry(filepath.Join("..", "..", "..", "policies", "tools", "registry-v1.json"))
	if err != nil {
		t.Fatal(err)
	}
	grantBody := `{"version":1,"task_id":"L5M2","total_max_calls":20,"entries":[
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
	state := tools.CallState{Calls: map[string]int{}}
	call := func(name, args string) model.ToolCall {
		return model.ToolCall{ID: "c", Name: name, Arguments: json.RawMessage(args)}
	}

	// Authorized read inside the environment delivers the pinned
	// content as classified evidence.
	_, ev, audit := tools.Handle(reg, grant, table, call("read_file", `{"path":"parser.go"}`), state)
	if ev == nil || audit.Decision != "authorized" {
		t.Fatalf("authorized read inside env: %+v", audit)
	}
	if !strings.Contains(string(ev.Evidence), "SENTINEL-L5") {
		t.Fatal("evidence must be the pinned checkout content")
	}
	// Escape attempts get the identical L4 verdicts inside.
	_, ev2, audit2 := tools.Handle(reg, grant, table, call("read_file", `{"path":"../../etc/passwd"}`), state)
	if ev2 != nil || audit2.DenialClass != tools.DenialTargetRefused {
		t.Fatalf("escape must be target-refused inside env: %+v", audit2)
	}
	// Ungranted tool: not-available, zero model detail.
	_, ev3, audit3 := tools.Handle(reg, grant, table, call("get_finding", `{"id":"F-1"}`), state)
	if ev3 != nil || audit3.DenialClass != tools.DenialNotAvailable {
		t.Fatalf("ungranted must be not-available inside env: %+v", audit3)
	}
	// The workspace binding is the environment worktree, recorded.
	if audit.Target != "parser.go" {
		t.Fatalf("target must be recorded: %+v", audit)
	}
}

func jstr(s string) string {
	b, _ := json.Marshal(s)
	return string(b)
}

// L5-M2 mechanical rule (Q-L5-5.2): executor code never touches the
// ambient process environment. The harness process retains the host
// env, so for in-process execution the boundary is this API-surface
// rule — checked here, not promised.
func TestNoAmbientEnvInExecutorSources(t *testing.T) {
	forbidden := []string{"os.Getenv", "os.Environ", "os.Setenv", "os.LookupEnv", "os.ExpandEnv", "syscall.Environ"}
	for _, root := range []string{".", "../tools", "../confine"} {
		// Recursive walk: a future executor subpackage must not be
		// silently skipped (test review).
		err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
			if err != nil || d.IsDir() || !strings.HasSuffix(d.Name(), ".go") || strings.HasSuffix(d.Name(), "_test.go") {
				return err
			}
			b, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			for _, f := range forbidden {
				if strings.Contains(string(b), f) {
					t.Errorf("%s uses %s — executors receive (entry, args, target), never ambient environment", path, f)
				}
			}
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
	}
}
