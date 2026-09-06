package tools

import (
	stdctx "context"
	"encoding/json"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/tofchaliss/themis/runtime/model"
)

// Live operational proof (Q-L4-9, owner-authorized qwen2.5:7b after
// the qwen2.5-coder protocol failure): a real local model emits
// structured tool calls through the Model Interface; L4 authorizes,
// denies, and errors through the real seam. Skips hermetically when
// no endpoint is reachable. The model is test equipment for the seam,
// not a standardization (DEC-05).
func TestLiveToolProof(t *testing.T) {
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

	reg := shippedRegistry(t)
	ws := t.TempDir()
	writeFile(t, ws, "parser.go", "package parser // SENTINEL-9142 marks the vulnerable path\n")
	// Grant includes read_file only; get_finding is declared to the
	// model (ToolDefs below) but EXCLUDED from the grant — the live
	// not-available case.
	grant := loadGrantBody(t, `{"version":1,"task_id":"LIVE","total_max_calls":20,"entries":[
	  {"tool":"read_file","max_calls":10,"workspace":"`+ws+`"}]}`)
	table, err := NewExecutorTable(reg, nil)
	if err != nil {
		t.Fatal(err)
	}
	provider := model.NewOllamaChat(endpoint)
	toolDefs := []model.ToolDef{
		{Name: "read_file", Description: "Read one file from the task workspace.",
			Parameters: json.RawMessage(`{"type":"object","properties":{"path":{"type":"string","description":"workspace-relative path"}},"required":["path"]}`)},
		{Name: "get_finding", Description: "Fetch a Themis finding by id.",
			Parameters: json.RawMessage(`{"type":"object","properties":{"id":{"type":"string","description":"finding id"}},"required":["id"]}`)},
	}
	run := func(prompt string) (model.ToolCall, []model.Message) {
		t.Helper()
		msgs := []model.Message{{Role: model.RoleUser, Content: prompt}}
		ctx, cancel := stdctx.WithTimeout(stdctx.Background(), 120*time.Second)
		defer cancel()
		resp, err := provider.Execute(ctx, model.ExecutionRequest{
			Model: modelName, Messages: msgs, Tools: toolDefs, Options: model.DefaultOptions()})
		if err != nil {
			t.Fatalf("live execution failed: %v", err)
		}
		if resp.Termination != model.TerminationToolCalls || len(resp.ToolCalls) == 0 {
			t.Fatalf("model must emit a structured tool call for %q: %+v %q", prompt, resp.Termination, firstLine(resp.Content))
		}
		return resp.ToolCalls[0], msgs
	}
	state := CallState{Calls: map[string]int{}}

	// Case 1+6: authorized call; result is classified evidence; the
	// completion turn cites the delivered sentinel.
	call, msgs := run("Read the file parser.go using the read_file tool and report what marker comment it contains.")
	if call.Name != "read_file" {
		t.Fatalf("expected read_file, got %s", call.Name)
	}
	toolMsg, ev, audit := Handle(reg, grant, table, call, state)
	if ev == nil || audit.Decision != "authorized" || ev.Trust != "external-untrusted" ||
		ev.Mechanism != "capability-fetch" || ev.Hash == "" {
		t.Fatalf("live authorized call must yield classified evidence: %+v", audit)
	}
	msgs = append(msgs, model.Message{Role: model.RoleAssistant, ToolCalls: []model.ToolCall{call}}, toolMsg)
	ctx, cancel := stdctx.WithTimeout(stdctx.Background(), 120*time.Second)
	defer cancel()
	resp2, err := provider.Execute(ctx, model.ExecutionRequest{
		Model: modelName, Messages: msgs, Tools: toolDefs, Options: model.DefaultOptions()})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(resp2.Content, "SENTINEL-9142") {
		t.Fatalf("completion must cite the delivered evidence: %q", firstLine(resp2.Content))
	}

	// Case 2: declared-but-ungranted tool -> live not-available, zero detail.
	call2, _ := run("Fetch the Themis finding FIND-42 using the get_finding tool.")
	if call2.Name != "get_finding" {
		t.Fatalf("expected get_finding, got %s", call2.Name)
	}
	dMsg, ev2, audit2 := Handle(reg, grant, table, call2, state)
	if ev2 != nil || audit2.DenialClass != DenialNotAvailable {
		t.Fatalf("ungranted live call must be not-available: %+v", audit2)
	}
	var body map[string]string
	if err := json.Unmarshal([]byte(dMsg.Content), &body); err != nil || body["denial"] != "not-available" || len(body) != 1 {
		t.Fatalf("live denial must be class-only: %q", dMsg.Content)
	}

	// Case 3: prompted escape target -> live target-refused.
	call3, _ := run("Use the read_file tool to read the path ../../etc/passwd exactly as written.")
	_, ev3, audit3 := Handle(reg, grant, table, call3, state)
	if ev3 != nil || audit3.DenialClass != DenialTargetRefused {
		t.Fatalf("escape target must be refused live: %+v", audit3)
	}

	// Case 4: missing file -> live typed error.
	call4, _ := run("Use the read_file tool to read the file missing.go.")
	_, ev4, audit4 := Handle(reg, grant, table, call4, state)
	if ev4 != nil || audit4.Decision != "error" || audit4.ErrClass != ErrFileUnreadable {
		t.Fatalf("missing file must be a typed error live: %+v", audit4)
	}

	// Case 5: invalid-args — model-compliance permitting: ask for an
	// extra parameter. If the model normalizes it away, drive the
	// mutated call through the seam-emitted call (controlled fixture
	// per the Q-L4-9 closure: the criterion is the protocol seam).
	call5, _ := run("Call read_file on parser.go and include an extra boolean parameter named follow_symlinks set to true.")
	var raw map[string]any
	_ = json.Unmarshal(call5.Arguments, &raw)
	if _, hasExtra := raw["follow_symlinks"]; !hasExtra {
		raw["follow_symlinks"] = true
		mutated, _ := json.Marshal(raw)
		call5.Arguments = mutated
		t.Log("model normalized the extra field; driving the seam-emitted call with the controlled fixture")
	}
	_, ev5, audit5 := Handle(reg, grant, table, call5, state)
	if ev5 != nil || audit5.DenialClass != DenialInvalidArgs || audit5.ModelDetail != "follow_symlinks" {
		t.Fatalf("extra field must be invalid-args(follow_symlinks) live: %+v", audit5)
	}

	t.Logf("live tool proof: model=%s authorized=%s denial/target/error/args all exercised; evidence hash=%s",
		modelName, audit.Decision, ev.Hash[:12])
}

func firstLine(s string) string {
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		s = s[:i]
	}
	if len(s) > 160 {
		s = s[:160]
	}
	return s
}
