package context

import (
	stdctx "context"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/tofchaliss/themis/instructions"
	"github.com/tofchaliss/themis/runtime/model"
)

// Operational proof (gate-0 decision): one live local-model run of a
// full L1+L2 payload — shipped instruction roots, shipped pattern
// policy, a real contract, gathered evidence, composed and executed
// through the Model Interface. Conversation only (no tools). Skips
// when no local model is reachable so CI stays hermetic; the
// operationally-proven verdict cites a real execution of this test.
func TestLiveOperationalProof(t *testing.T) {
	endpoint := os.Getenv("THEMIS_LIVE_OLLAMA")
	if endpoint == "" {
		endpoint = "http://localhost:11434"
	}
	client := http.Client{Timeout: 2 * time.Second}
	if _, err := client.Get(endpoint + "/api/tags"); err != nil {
		t.Skipf("no local model endpoint at %s: %v", endpoint, err)
	}
	modelName := os.Getenv("THEMIS_LIVE_MODEL")
	if modelName == "" {
		modelName = "WhiteRabbitNeo/WHiteRabbitNeo-2.5-Qwen-2.5-Coder-7B:latest"
	}

	pol, err := instructions.LoadPolicy("../../../policies/security/instruction-directive-patterns.json")
	if err != nil {
		t.Fatal(err)
	}
	set, err := instructions.Resolve(instructions.Config{Policy: pol},
		instructions.Source{Kind: instructions.ScopeHarnessSafety, Root: "../../../instructions/global/safety"},
		instructions.Source{Kind: instructions.ScopeHarnessSystem, Root: "../../../instructions/global/system"},
		instructions.Source{Kind: instructions.ScopeThemisDomain, Root: "../../../instructions/themis"},
		instructions.Source{Kind: instructions.ScopeTask, Inline: []instructions.Instruction{{
			ID: "task.instructions", Scope: instructions.ScopeTask, Category: instructions.CategoryTask,
			Body: "State in one sentence whether the delivered finding indicates an open vulnerability, citing only the delivered evidence.\n"}}},
	)
	if err != nil {
		t.Fatal(err)
	}
	c, asg := stdAssignments(t)
	g, err := Gather(c, asg)
	if err != nil {
		t.Fatal(err)
	}
	p, err := Compose(set, pol, g)
	if err != nil {
		t.Fatal(err)
	}

	provider := model.NewOllamaChat(endpoint)
	ctx, cancel := stdctx.WithTimeout(stdctx.Background(), 120*time.Second)
	defer cancel()
	resp, err := provider.Execute(ctx, model.ExecutionRequest{
		Model: modelName, Messages: p.Messages, Options: model.DefaultOptions(),
	})
	if err != nil {
		t.Fatalf("live execution failed: %v", err)
	}
	if resp.Termination != model.TerminationStop || strings.TrimSpace(resp.Content) == "" {
		t.Fatalf("live run must terminate clean with content: %+v", resp.Termination)
	}
	// Payload-starvation guard: the task demands citing delivered
	// evidence, so the reply must echo at least one evidence token —
	// an empty or garbled user message would otherwise still pass.
	echoed := false
	for _, token := range []string{"CVE-2026-12345", "libXYZ", "1.4.2"} {
		if strings.Contains(resp.Content, token) {
			echoed = true
			break
		}
	}
	if !echoed {
		t.Fatalf("reply cites no delivered evidence token — payload starvation? %q", firstLine(resp.Content))
	}
	t.Logf("live proof: eis=%s payload=%s model=%s reply=%q",
		p.EISHash[:12], p.PayloadHash[:12], resp.Identity.WireModel, firstLine(resp.Content))
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

// L3 operational proof (Q-L3-9): live budget-pressure run — the model
// receives the deterministically reduced set plus the
// omitted_for_capacity marker and cites surviving evidence.
func TestLivePressureProof(t *testing.T) {
	endpoint := os.Getenv("THEMIS_LIVE_OLLAMA")
	if endpoint == "" {
		endpoint = "http://localhost:11434"
	}
	client := http.Client{Timeout: 2 * time.Second}
	if _, err := client.Get(endpoint + "/api/tags"); err != nil {
		t.Skipf("no local model endpoint at %s: %v", endpoint, err)
	}
	modelName := os.Getenv("THEMIS_LIVE_MODEL")
	if modelName == "" {
		modelName = "WhiteRabbitNeo/WHiteRabbitNeo-2.5-Qwen-2.5-Coder-7B:latest"
	}

	pol, err := instructions.LoadPolicy("../../../policies/security/instruction-directive-patterns.json")
	if err != nil {
		t.Fatal(err)
	}
	set, err := instructions.Resolve(instructions.Config{Policy: pol},
		instructions.Source{Kind: instructions.ScopeHarnessSafety, Root: "../../../instructions/global/safety"},
		instructions.Source{Kind: instructions.ScopeHarnessSystem, Root: "../../../instructions/global/system"},
		instructions.Source{Kind: instructions.ScopeThemisDomain, Root: "../../../instructions/themis"},
		instructions.Source{Kind: instructions.ScopeTask, Inline: []instructions.Instruction{{
			ID: "task.instructions", Scope: instructions.ScopeTask, Category: instructions.CategoryTask,
			Body: "In one sentence: what is the status of the delivered finding? Cite only delivered evidence.\n"}}},
	)
	if err != nil {
		t.Fatal(err)
	}
	c := loadTestContract(t, dropContract)
	g := pressureGathered(t, c, 4000, 400)
	mp := mgmtPolicy(t, `{"version":1,"name":"live-tight","budget":300,"drop_order":["source-files"],"rank_keys":["size_asc"],"dedup":"none"}`)
	managed, trace, err := Manage(mp, g)
	if err != nil {
		t.Fatal(err)
	}
	if trace.BudgetUsed > trace.BudgetLimit {
		t.Fatalf("managed set exceeds budget: %+v", trace)
	}
	p, err := Compose(set, pol, managed)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(p.Messages[1].Content, "omitted_for_capacity") {
		t.Fatal("pressure payload must carry the omission marker")
	}
	provider := model.NewOllamaChat(endpoint)
	ctx, cancel := stdctx.WithTimeout(stdctx.Background(), 120*time.Second)
	defer cancel()
	resp, err := provider.Execute(ctx, model.ExecutionRequest{
		Model: modelName, Messages: p.Messages, Options: model.DefaultOptions(),
	})
	if err != nil {
		t.Fatalf("live pressure execution failed: %v", err)
	}
	if resp.Termination != model.TerminationStop || strings.TrimSpace(resp.Content) == "" {
		t.Fatalf("live pressure run must terminate clean: %+v", resp.Termination)
	}
	echoed := false
	for _, token := range []string{"CVE-2026-12345", "OPEN", "libXYZ"} {
		if strings.Contains(resp.Content, token) {
			echoed = true
			break
		}
	}
	if !echoed {
		t.Fatalf("reply cites no surviving evidence: %q", firstLine(resp.Content))
	}
	t.Logf("live pressure proof: payload=%s used=%d/%d reply=%q",
		p.PayloadHash[:12], trace.BudgetUsed, trace.BudgetLimit, firstLine(resp.Content))
}
