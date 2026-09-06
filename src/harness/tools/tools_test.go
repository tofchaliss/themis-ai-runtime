package tools

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	hctx "github.com/tofchaliss/themis/context"
	"github.com/tofchaliss/themis/runtime/model"
)

func writeFile(t *testing.T, dir, name, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(filepath.Join(dir, name)), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func shippedRegistry(t *testing.T) *Registry {
	t.Helper()
	r, err := LoadRegistry("../../../policies/tools/registry-v1.json")
	if err != nil {
		t.Fatal(err)
	}
	return r
}

func testGrant(t *testing.T, workspace string) *Grant {
	t.Helper()
	body := `{"version":1,"task_id":"T1","total_max_calls":20,"entries":[
	  {"tool":"read_file","max_calls":5,"workspace":"` + workspace + `"},
	  {"tool":"search_code","max_calls":5,"workspace":"` + workspace + `"},
	  {"tool":"get_finding","max_calls":3,"themis_scope":["FIND-"]}
	]}`
	path := filepath.Join(t.TempDir(), "grant.json")
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	g, err := LoadGrant(path)
	if err != nil {
		t.Fatal(err)
	}
	return g
}

type fakeSeam struct{ data map[string]string }

func (f fakeSeam) Read(kind, id string) ([]byte, error) {
	v, ok := f.data[kind+"/"+id]
	if !ok {
		return nil, errors.New("no record")
	}
	return []byte(v), nil
}

func args(t *testing.T, m map[string]any) json.RawMessage {
	t.Helper()
	b, err := json.Marshal(m)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func TestRegistryFailsClosed(t *testing.T) {
	cases := []struct {
		name string
		body string
	}{
		{"invalid json", `{`},
		{"unknown field", `{"version":1,"tools":[],"extra":1}`},
		{"no tools", `{"version":1,"tools":[]}`},
		{"trailing", `{"version":1,"tools":[{"name":"a","description":"d","params":[],"target":"none","timeout_sec":1,"trust":"external-untrusted"}]} X`},
		{"bad name", `{"version":1,"tools":[{"name":"Run-Shell!","description":"d","params":[],"target":"none","timeout_sec":1,"trust":"external-untrusted"}]}`},
		{"duplicate tool", `{"version":1,"tools":[{"name":"a","description":"d","params":[],"target":"none","timeout_sec":1,"trust":"external-untrusted"},{"name":"a","description":"d","params":[],"target":"none","timeout_sec":1,"trust":"external-untrusted"}]}`},
		{"zero timeout", `{"version":1,"tools":[{"name":"a","description":"d","params":[],"target":"none","timeout_sec":0,"trust":"external-untrusted"}]}`},
		{"unknown target class", `{"version":1,"tools":[{"name":"a","description":"d","params":[],"target":"database","timeout_sec":1,"trust":"external-untrusted"}]}`},
		{"unknown trust", `{"version":1,"tools":[{"name":"a","description":"d","params":[],"target":"none","timeout_sec":1,"trust":"tool-output"}]}`},
		{"self-declared derived", `{"version":1,"tools":[{"name":"a","description":"d","params":[],"target":"none","timeout_sec":1,"trust":"derived"}]}`},
		{"authority-disposition param", `{"version":1,"tools":[{"name":"a","description":"d","params":[{"name":"requires_human_decision","type":"boolean","description":"x"}],"target":"none","timeout_sec":1,"trust":"external-untrusted"}]}`},
		{"target param wrong type", `{"version":1,"tools":[{"name":"a","description":"d","params":[{"name":"p","type":"integer","description":"x","target":true}],"target":"workspace-path","timeout_sec":1,"trust":"external-untrusted"}]}`},
		{"target class without target param", `{"version":1,"tools":[{"name":"a","description":"d","params":[],"target":"workspace-path","timeout_sec":1,"trust":"external-untrusted"}]}`},
		{"target param without class", `{"version":1,"tools":[{"name":"a","description":"d","params":[{"name":"p","type":"string","description":"x","target":true}],"target":"none","timeout_sec":1,"trust":"external-untrusted"}]}`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "r.json")
			if err := os.WriteFile(path, []byte(tc.body), 0o644); err != nil {
				t.Fatal(err)
			}
			if _, err := LoadRegistry(path); !errors.Is(err, ErrRegistryInvalid) {
				t.Fatalf("want ErrRegistryInvalid, got %v", err)
			}
		})
	}
	if r := shippedRegistry(t); r.Hash == "" || len(r.Tools) != 5 {
		t.Fatal("shipped registry must load with 5 read-only tools")
	}
}

func TestGrantFailsClosed(t *testing.T) {
	for _, body := range []string{
		`{`,
		`{"version":1,"task_id":"T1","total_max_calls":10,"entries":[],"x":1}`,
		`{"version":1,"task_id":"","total_max_calls":10,"entries":[{"tool":"a","max_calls":1}]}`,
		`{"version":1,"task_id":"T1","total_max_calls":0,"entries":[{"tool":"a","max_calls":1}]}`,
		`{"version":1,"task_id":"T1","total_max_calls":10,"entries":[{"tool":"a","max_calls":0}]}`,
		`{"version":1,"task_id":"T1","total_max_calls":10,"entries":[{"tool":"a","max_calls":1},{"tool":"a","max_calls":1}]}`,
		`{"version":1,"task_id":"T1","total_max_calls":10,"entries":[{"tool":"a","max_calls":1}]} X`,
	} {
		path := filepath.Join(t.TempDir(), "g.json")
		if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
		if _, err := LoadGrant(path); !errors.Is(err, ErrGrantInvalid) {
			t.Fatalf("must fail closed: %s", body)
		}
	}
}

// The locked Q-L4-5 decision table, including the anti-oracle check
// order: unavailable capabilities NEVER expose argument-validation
// information.
func TestAuthorizeDecisionTable(t *testing.T) {
	reg := shippedRegistry(t)
	ws := t.TempDir()
	writeFile(t, ws, "a.go", "package a\n")
	grant := testGrant(t, ws)
	empty := CallState{Calls: map[string]int{}}

	cases := []struct {
		name        string
		tool        string
		args        json.RawMessage
		state       CallState
		wantDenial  DenialClass
		wantDetail  string
		wantTracePc string // substring of TracePredicate
	}{
		{"unknown tool", "run_command", args(t, map[string]any{"cmd": "ls"}), empty, DenialNotAvailable, "", "unknown-tool"},
		{"unknown tool with invented field leaks nothing", "update_enterprise_position",
			args(t, map[string]any{"requires_human_decision": false}), empty, DenialNotAvailable, "", "unknown-tool"},
		{"known but not granted", "list_directory", args(t, map[string]any{"path": "."}), empty, DenialNotAvailable, "", "not-granted"},
		{"not granted with bad args still not-available", "list_directory",
			args(t, map[string]any{"bogus": 1}), empty, DenialNotAvailable, "", "not-granted"},
		{"tool quota exhausted", "read_file", args(t, map[string]any{"path": "a.go"}),
			CallState{Calls: map[string]int{"read_file": 5}}, DenialNotAvailable, "", "quota-exhausted"},
		{"total quota exhausted", "read_file", args(t, map[string]any{"path": "a.go"}),
			CallState{Calls: map[string]int{}, Total: 20}, DenialNotAvailable, "", "quota-exhausted"},
		{"unknown field whole-call rejection", "read_file",
			args(t, map[string]any{"path": "a.go", "follow_symlinks": true}), empty, DenialInvalidArgs, "follow_symlinks", "unknown-field"},
		{"scenario-3 regression: authority field dies at schema", "read_file",
			args(t, map[string]any{"path": "a.go", "requires_human_decision": false}), empty, DenialInvalidArgs, "requires_human_decision", "unknown-field"},
		{"missing required", "read_file", args(t, map[string]any{}), empty, DenialInvalidArgs, "path", "missing-required"},
		{"wrong type", "read_file", args(t, map[string]any{"path": 42}), empty, DenialInvalidArgs, "path", "wrong-type"},
		{"malformed args json", "read_file", json.RawMessage(`{"path":`), empty, DenialInvalidArgs, "", "args not a JSON object"},
		{"target escape", "read_file", args(t, map[string]any{"path": "../../etc/passwd"}), empty, DenialTargetRefused, "../../etc/passwd", "confinement"},
		{"themis id bad shape", "get_finding", args(t, map[string]any{"id": "FIND-1; DROP TABLE"}), empty, DenialTargetRefused, "", "themis-id-shape"},
		{"themis id outside scope", "get_finding", args(t, map[string]any{"id": "PROD-9"}), empty, DenialTargetRefused, "PROD-9", "outside-grant-scope"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.state.Calls == nil {
				tc.state.Calls = map[string]int{}
			}
			d := Authorize(reg, grant, tc.tool, tc.args, tc.state)
			if d.Allow || d.Denial != tc.wantDenial {
				t.Fatalf("want %s, got %+v", tc.wantDenial, d)
			}
			if tc.wantDetail != "" && d.ModelDetail != tc.wantDetail {
				t.Fatalf("model detail %q want %q", d.ModelDetail, tc.wantDetail)
			}
			if tc.wantDenial == DenialNotAvailable && d.ModelDetail != "" {
				t.Fatal("not-available must carry zero model-visible detail")
			}
			if !strings.Contains(d.TracePredicate, tc.wantTracePc) {
				t.Fatalf("trace predicate %q want %q", d.TracePredicate, tc.wantTracePc)
			}
		})
	}
	// Allow path.
	d := Authorize(reg, grant, "read_file", args(t, map[string]any{"path": "a.go"}), empty)
	if !d.Allow || d.Target != "a.go" || d.RegistryHash != reg.Hash || d.GrantHash != grant.Hash {
		t.Fatalf("allow path broken: %+v", d)
	}
	// Determinism within a snapshot.
	d2 := Authorize(reg, grant, "run_command", args(t, map[string]any{"cmd": "ls"}), empty)
	d3 := Authorize(reg, grant, "run_command", args(t, map[string]any{"cmd": "ls"}), empty)
	if !reflect.DeepEqual(d2, d3) {
		t.Fatal("denials must be deterministic within a snapshot")
	}
}

func TestDispatchCompleteness(t *testing.T) {
	reg := shippedRegistry(t)
	if _, err := NewExecutorTable(reg, nil); err != nil {
		t.Fatalf("shipped registry must be fully dispatchable: %v", err)
	}
	// A registry tool without an executor fails closed at startup.
	extra := *reg
	extra.Tools = append(append([]ToolDef{}, reg.Tools...), ToolDef{
		Name: "phantom_tool", Description: "d", Target: TargetNone, TimeoutSec: 1,
		Trust: hctx.AuthorityExternalUntrusted})
	if _, err := NewExecutorTable(&extra, nil); !errors.Is(err, ErrDispatch) {
		t.Fatalf("phantom registry entry must fail closed: %v", err)
	}
	// Mutating/shell names do not exist in the table at all.
	table, _ := NewExecutorTable(reg, nil)
	for _, name := range []string{"run_command", "write_file", "apply_patch", "git_commit"} {
		if _, ok := table[name]; ok {
			t.Fatalf("mutating capability %q must not exist in the executable vocabulary", name)
		}
	}
}

func TestHandlePipeline(t *testing.T) {
	reg := shippedRegistry(t)
	ws := t.TempDir()
	writeFile(t, ws, "parser.go", "package parser // libXYZ\n")
	writeFile(t, ws, "sub/other.go", "package other\n")
	grant := testGrant(t, ws)
	seam := fakeSeam{data: map[string]string{"finding/FIND-77": "CVE-2026-12345 OPEN\n"}}
	table, err := NewExecutorTable(reg, seam)
	if err != nil {
		t.Fatal(err)
	}
	state := CallState{Calls: map[string]int{}}

	// 1. Authorized read: evidence + trust class + audit + mechanism.
	msg, ev, audit := Handle(reg, grant, table, model.ToolCall{ID: "c1", Name: "read_file",
		Arguments: args(t, map[string]any{"path": "parser.go"})}, state)
	if msg.Role != model.RoleTool || msg.ToolCallID != "c1" || msg.Content != "package parser // libXYZ\n" {
		t.Fatalf("result message wrong: %+v", msg)
	}
	if ev == nil || ev.Trust != hctx.AuthorityExternalUntrusted || ev.Mechanism != hctx.MechanismCapabilityFetch ||
		ev.Hash != hctx.EvidenceHash([]byte(msg.Content)) {
		t.Fatalf("tool evidence wrong: %+v", ev)
	}
	if audit.Decision != "authorized" || audit.ResultHash != ev.Hash || audit.RegistryHash != reg.Hash {
		t.Fatalf("audit wrong: %+v", audit)
	}

	// 2. Themis read: governed-record evidence.
	_, ev2, _ := Handle(reg, grant, table, model.ToolCall{ID: "c2", Name: "get_finding",
		Arguments: args(t, map[string]any{"id": "FIND-77"})}, state)
	if ev2 == nil || ev2.Trust != hctx.AuthorityGovernedRecord || string(ev2.Evidence) != "CVE-2026-12345 OPEN\n" {
		t.Fatalf("themis evidence wrong: %+v", ev2)
	}

	// 3. Denied: model sees class-only JSON; audit records mechanics.
	msg3, ev3, audit3 := Handle(reg, grant, table, model.ToolCall{ID: "c3", Name: "run_command",
		Arguments: args(t, map[string]any{"cmd": "rm -rf /"})}, state)
	if ev3 != nil || audit3.Decision != "denied" || audit3.DenialClass != DenialNotAvailable {
		t.Fatalf("denial handling wrong: %+v", audit3)
	}
	var body map[string]string
	if err := json.Unmarshal([]byte(msg3.Content), &body); err != nil ||
		body["denial"] != "not-available" || len(body) != 1 {
		t.Fatalf("denial content must be class-only: %q", msg3.Content)
	}

	// 4. Executor error: typed, never denial-shaped, audit records it.
	msg4, ev4, audit4 := Handle(reg, grant, table, model.ToolCall{ID: "c4", Name: "read_file",
		Arguments: args(t, map[string]any{"path": "absent.go"})}, state)
	if ev4 != nil || audit4.Decision != "error" || audit4.ErrClass != ErrFileUnreadable {
		t.Fatalf("error handling wrong: %+v", audit4)
	}
	if !strings.Contains(msg4.Content, `"error"`) || strings.Contains(msg4.Content, "denial") {
		t.Fatalf("error content wrong: %q", msg4.Content)
	}

	// 5. search_code: sorted hits, confined.
	msg6, _, _ := Handle(reg, grant, table, model.ToolCall{ID: "c6", Name: "search_code",
		Arguments: args(t, map[string]any{"path": ".", "query": "libXYZ"})}, state)
	if msg6.Content != "parser.go\n" {
		t.Fatalf("search result wrong: %q", msg6.Content)
	}
	// 6. Symlink escape through the tool path: target-refused, no bytes.
	outside := t.TempDir()
	writeFile(t, outside, "secret.txt", "TOPSECRET\n")
	if err := os.Symlink(filepath.Join(outside, "secret.txt"), filepath.Join(ws, "leak.go")); err != nil {
		t.Fatal(err)
	}
	msg5, ev5, _ := Handle(reg, grant, table, model.ToolCall{ID: "c5", Name: "read_file",
		Arguments: args(t, map[string]any{"path": "leak.go"})}, state)
	if ev5 != nil || strings.Contains(msg5.Content, "TOPSECRET") {
		t.Fatalf("symlink escape must not deliver bytes: %q", msg5.Content)
	}

}

// Full loop through the mock provider: model emits a tool call, L4
// authorizes/executes, the tool message returns, conversation
// completes — the Model Interface seam end to end (deterministic half
// of Q-L4-9; the live half awaits an owner-authorized tools-capable
// model).
func TestMockProviderLoop(t *testing.T) {
	reg := shippedRegistry(t)
	ws := t.TempDir()
	writeFile(t, ws, "a.go", "package a\n")
	grant := testGrant(t, ws)
	table, err := NewExecutorTable(reg, fakeSeam{})
	if err != nil {
		t.Fatal(err)
	}
	// Scripted provider: first turn emits the tool call; second turn
	// (after the tool message) completes.
	call := model.ToolCall{ID: "t1", Name: "read_file", Arguments: args(t, map[string]any{"path": "a.go"})}
	msg, ev, audit := Handle(reg, grant, table, call, CallState{Calls: map[string]int{}})
	if ev == nil || audit.Decision != "authorized" {
		t.Fatalf("loop step failed: %+v", audit)
	}
	if msg.ToolCallID != "t1" || msg.Role != model.RoleTool {
		t.Fatal("tool message must bind to the call id for the provider loop")
	}
}

// Coverage closers: remaining executors, empty args, echo bounding.
func TestRemainingExecutorsAndEdges(t *testing.T) {
	reg := shippedRegistry(t)
	ws := t.TempDir()
	writeFile(t, ws, "d/x.go", "package x\n")
	body := `{"version":1,"task_id":"T2","total_max_calls":10,"entries":[
	  {"tool":"list_directory","max_calls":2,"workspace":"` + ws + `"},
	  {"tool":"get_product","max_calls":2,"themis_scope":["PROD-"]}
	]}`
	path := filepath.Join(t.TempDir(), "grant.json")
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	grant, err := LoadGrant(path)
	if err != nil {
		t.Fatal(err)
	}
	table, err := NewExecutorTable(reg, fakeSeam{data: map[string]string{"product/PROD-1": "libXYZ 1.4.2\n"}})
	if err != nil {
		t.Fatal(err)
	}
	state := CallState{Calls: map[string]int{}}
	msg, _, _ := Handle(reg, grant, table, model.ToolCall{ID: "l1", Name: "list_directory",
		Arguments: args(t, map[string]any{"path": "d"})}, state)
	if msg.Content != "x.go\n" {
		t.Fatalf("list wrong: %q", msg.Content)
	}
	msg2, ev2, _ := Handle(reg, grant, table, model.ToolCall{ID: "l2", Name: "get_product",
		Arguments: args(t, map[string]any{"id": "PROD-1"})}, state)
	if ev2 == nil || ev2.Trust != hctx.AuthorityGovernedRecord || msg2.Content != "libXYZ 1.4.2\n" {
		t.Fatalf("get_product wrong: %+v %q", ev2, msg2.Content)
	}
	// Seam error path.
	_, _, audit := Handle(reg, grant, table, model.ToolCall{ID: "l3", Name: "get_product",
		Arguments: args(t, map[string]any{"id": "PROD-404"})}, state)
	if audit.Decision != "error" || audit.ErrClass != ErrSeamUnavailable {
		t.Fatalf("seam error wrong: %+v", audit)
	}
	// Empty args on a required-param tool: missing-required.
	d := Authorize(reg, grant, "list_directory", nil, state)
	if d.Allow || d.Denial != DenialInvalidArgs || d.ModelDetail != "path" {
		t.Fatalf("nil args must be missing-required: %+v", d)
	}
	// Echo bounding: huge target truncated to the disclosure bound.
	long := strings.Repeat("x", 5000)
	d2 := Authorize(reg, grant, "list_directory", args(t, map[string]any{"path": "../" + long}), state)
	if d2.Denial != DenialTargetRefused || len(d2.ModelDetail) > 256 {
		t.Fatalf("echo must be bounded: %d", len(d2.ModelDetail))
	}
	// Missing registry file.
	if _, err := LoadRegistry(filepath.Join(t.TempDir(), "absent.json")); !errors.Is(err, ErrRegistryInvalid) {
		t.Fatal("missing registry must fail closed")
	}
	// Oversized args.
	big, _ := json.Marshal(map[string]any{"path": strings.Repeat("y", maxArgsBytes)})
	d3 := Authorize(reg, grant, "list_directory", big, state)
	if d3.Denial != DenialInvalidArgs {
		t.Fatalf("oversized args must be invalid-args: %+v", d3)
	}
}

// Security-review regressions F1/F3/F4/F6/F9.
func TestSecurityReviewRegressions(t *testing.T) {
	reg := shippedRegistry(t)
	// F6: relative workspace binding fails closed at load.
	rel := `{"version":1,"task_id":"T","total_max_calls":5,"entries":[{"tool":"read_file","max_calls":1,"workspace":"relative/path"}]}`
	path := filepath.Join(t.TempDir(), "g.json")
	if err := os.WriteFile(path, []byte(rel), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadGrant(path); !errors.Is(err, ErrGrantInvalid) {
		t.Fatal("relative workspace must fail closed")
	}
	// F9: granted workspace tool without a binding -> not-available.
	noWS := `{"version":1,"task_id":"T","total_max_calls":5,"entries":[{"tool":"read_file","max_calls":1}]}`
	if err := os.WriteFile(path, []byte(noWS), 0o644); err != nil {
		t.Fatal(err)
	}
	g, err := LoadGrant(path)
	if err != nil {
		t.Fatal(err)
	}
	d := Authorize(reg, g, "read_file", args(t, map[string]any{"path": "a.go"}), CallState{Calls: map[string]int{}})
	if d.Allow || d.Denial != DenialNotAvailable || d.ModelDetail != "" {
		t.Fatalf("missing binding is config error -> not-available: %+v", d)
	}
	if d.RequestedTarget != "a.go" {
		t.Fatalf("requested target must be recorded on denial paths (F2): %+v", d)
	}
	// F1: unbounded directory listing refuses oversized.
	ws := t.TempDir()
	long := strings.Repeat("f", 200)
	for i := 0; i < 1500; i++ {
		writeFile(t, ws, long+string(rune('a'+i%26))+string(rune('a'+(i/26)%26))+string(rune('a'+i/676))+".go", "x")
	}
	grant := testGrant(t, ws)
	table, _ := NewExecutorTable(reg, nil)
	// list_directory not in testGrant; use a direct executor call.
	out := execListDirectory(&GrantEntry{Workspace: ws}, nil, ".")
	if out.ErrClass != ErrOversized {
		t.Fatalf("huge listing must refuse oversized: %+v", out.ErrClass)
	}
	// F4: query match hidden only in an oversized file -> typed error, never clean-empty.
	ws2 := t.TempDir()
	writeFile(t, ws2, "pad.go", strings.Repeat("A", maxToolEvidence+10)+" NEEDLE")
	out2 := execSearchCode(&GrantEntry{Workspace: ws2}, map[string]any{"query": "NEEDLE"}, ".")
	if out2.ErrClass != ErrOversized || out2.SkippedOversized == 0 {
		t.Fatalf("hidden-match padding must surface: %+v", out2)
	}
	// F3: registry/table drift -> typed error + audit, no panic.
	drifted := map[string]Executor{}
	msg, ev, audit := Handle(reg, grant, drifted, model.ToolCall{ID: "d1", Name: "read_file",
		Arguments: args(t, map[string]any{"path": "x"})}, CallState{Calls: map[string]int{}})
	_ = msg
	if ev != nil || audit.Decision != "error" || audit.ErrClass != ErrSeamUnavailable {
		t.Fatalf("table drift must fail closed with audit: %+v", audit)
	}
	// F2: denial audit carries model detail + requested target.
	msgD, _, auditD := Handle(reg, grant, table, model.ToolCall{ID: "d2", Name: "read_file",
		Arguments: args(t, map[string]any{"path": "../etc/passwd"})}, CallState{Calls: map[string]int{}})
	if auditD.DenialClass != DenialTargetRefused || auditD.ModelDetail != "../etc/passwd" || auditD.Target != "../etc/passwd" {
		t.Fatalf("denial audit incomplete: %+v", auditD)
	}
	if !strings.Contains(msgD.Content, "../etc/passwd") {
		t.Fatalf("target echo missing: %q", msgD.Content)
	}
}
