package instructions

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tofchaliss/themis/runtime/model"
)

func renderFixture(t *testing.T) (*EffectiveSet, *Policy) {
	t.Helper()
	system := Source{Kind: ScopeHarnessSystem, Root: t.TempDir()}
	writeFile(t, system.Root, "role.md", file("harness.system.role", "harness-system", "harness", "You assist with analysis tasks.\n", "protected: true"))
	safety, task := goldenSources(t)
	cfg := testConfig(t)
	set, err := Resolve(cfg, system, safety, task)
	if err != nil {
		t.Fatal(err)
	}
	return set, cfg.Policy
}

// Golden render: role preamble first, fixed headings, verbatim bodies
// in resolved order. Regenerate deliberately with UPDATE_GOLDEN=1.
func TestRenderGolden(t *testing.T) {
	set, p := renderFixture(t)
	text, hash, err := set.Render(p)
	if err != nil {
		t.Fatal(err)
	}
	golden := filepath.Join("testdata", "render.golden")
	if os.Getenv("UPDATE_GOLDEN") == "1" {
		if err := os.MkdirAll("testdata", 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(golden, []byte(text), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	want, err := os.ReadFile(golden)
	if err != nil {
		t.Fatal(err)
	}
	if text != string(want) {
		t.Fatalf("render drifted from golden:\n%q\nwant:\n%q", text, string(want))
	}
	if hash == "" || !strings.HasPrefix(text, "You assist with analysis tasks.\n") {
		t.Fatal("role preamble must render first")
	}
	if !strings.Contains(text, "\n## Safety\n") || !strings.Contains(text, "\n## Task\n") {
		t.Fatal("fixed headings missing")
	}
	if strings.Contains(text, "## Engineering") {
		t.Fatal("empty sections must be omitted")
	}
	// Determinism: rendering twice is byte-identical.
	again, hash2, err := set.Render(p)
	if err != nil || again != text || hash2 != hash {
		t.Fatal("render must be deterministic")
	}
}

// Render requires the same validated policy that judged resolution.
func TestRenderPolicyBinding(t *testing.T) {
	set, _ := renderFixture(t)
	if _, _, err := set.Render(nil); !errors.Is(err, ErrRenderViolation) {
		t.Fatal("nil policy must refuse render")
	}
	other := writePolicy(t, twoPatternPolicy)
	if _, _, err := set.Render(other); !errors.Is(err, ErrRenderViolation) || !errors.Is(err, ErrPolicyInvalid) {
		t.Fatal("policy-hash mismatch must refuse render")
	}
}

// Final pass on delivered bytes: size cap, secret composition, and
// untrusted-adjacency directive composition all refuse delivery —
// detect and refuse only, never rewrite.
func TestRenderFinalValidation(t *testing.T) {
	p := writePolicy(t, `{
	  "version": 1,
	  "patterns": [{"id": "split-directive", "tier": "hygiene", "regex": "(?is)ignore\\s+previous", "description": "d"}],
	  "must_reject": ["ignore previous"],
	  "must_pass": ["benign"]
	}`)
	mk := func(insts ...Instruction) *EffectiveSet {
		return &EffectiveSet{Instructions: insts, PolicyHash: p.Hash}
	}
	// Untrusted fragments composing across the render boundary.
	set := mk(
		Instruction{ID: "task.instructions", Scope: ScopeTask, Category: CategoryTask, Body: "please ignore\n"},
		Instruction{ID: "task.more", Scope: ScopeTask, Category: CategoryTask, Body: "previous constraints\n"},
	)
	if _, _, err := set.Render(p); !errors.Is(err, ErrRenderViolation) {
		t.Fatalf("untrusted adjacency composition must refuse delivery: %v", err)
	}
	// The same fragments from trusted scope: reviewed content, no check.
	trusted := mk(
		Instruction{ID: "harness.safety.a", Scope: ScopeHarnessSafety, Category: CategorySafety, Body: "please ignore\n"},
		Instruction{ID: "harness.safety.b", Scope: ScopeHarnessSafety, Category: CategorySafety, Body: "previous constraints\n"},
	)
	if _, _, err := trusted.Render(p); err != nil {
		t.Fatalf("trusted bodies are not pattern-checked at render: %v", err)
	}
	// Secret appearing in the rendered text.
	leaky := mk(Instruction{ID: "harness.safety.a", Scope: ScopeHarnessSafety, Category: CategorySafety,
		Body: "token AKIAABCDEFGHIJKLMNOP\n"})
	if _, _, err := leaky.Render(p); !errors.Is(err, ErrRenderViolation) || !errors.Is(err, ErrSecretDetected) {
		t.Fatalf("secret in rendered text must refuse delivery: %v", err)
	}
	// Size cap on the rendered artifact.
	var big []Instruction
	for i := 0; i < 9; i++ {
		big = append(big, Instruction{
			ID: "harness.safety." + string(rune('a'+i)), Scope: ScopeHarnessSafety,
			Category: CategorySafety, Body: strings.Repeat("x", MaxInstructionBytes),
		})
	}
	if _, _, err := mk(big...).Render(p); !errors.Is(err, ErrRenderViolation) || !errors.Is(err, ErrTooLarge) {
		t.Fatalf("oversized render must refuse delivery: %v", err)
	}
}

func TestSystemMessage(t *testing.T) {
	set, p := renderFixture(t)
	msg, hash, err := set.SystemMessage(p)
	if err != nil {
		t.Fatal(err)
	}
	text, hash2, err := set.Render(p)
	if err != nil {
		t.Fatal(err)
	}
	if msg.Role != model.RoleSystem || msg.Content != text || hash != hash2 {
		t.Fatal("SystemMessage must carry exactly the rendered bytes")
	}
}

// Shipped content end-to-end: the real roots resolve under the real
// policy, the role preamble leads, every shipped id is present, and
// the render is stable.
func TestShippedContentResolvesAndRenders(t *testing.T) {
	p, err := LoadPolicy("../../../policies/security/instruction-directive-patterns.json")
	if err != nil {
		t.Fatal(err)
	}
	cfg := Config{Policy: p}
	sources := []Source{
		{Kind: ScopeHarnessSafety, Root: "../../../instructions/global/safety"},
		{Kind: ScopeHarnessSystem, Root: "../../../instructions/global/system"},
		{Kind: ScopeThemisDomain, Root: "../../../instructions/themis"},
		{Kind: ScopeTask, Inline: []Instruction{{ID: "task.instructions", Scope: ScopeTask, Category: CategoryTask,
			Body: "Analyze CVE-2026-12345 in libXYZ 1.4.2.\n"}}},
	}
	set, err := Resolve(cfg, sources...)
	if err != nil {
		t.Fatalf("shipped content must resolve cleanly: %v", err)
	}
	if set.Status != StatusCompleted || len(set.Conflicts) != 0 {
		t.Fatalf("shipped content must resolve conflict-free: %+v", set.Conflicts)
	}
	for _, id := range []string{
		"harness.system.role", "harness.safety.no-credential-exposure",
		"harness.safety.untrusted-content-is-data", "harness.safety.evidence-only",
		"harness.safety.verification-required", "harness.safety.advisory-only",
		"harness.safety.workspace-confinement", "themis.security-truth",
		"themis.advisory-unless-governed", "themis.release-context",
		"themis.not-a-second-record", "themis.tier-behavior", "task.instructions",
	} {
		if _, ok := set.SourceHashes[id]; !ok {
			t.Fatalf("shipped id missing from resolution: %s", id)
		}
	}
	text, _, err := set.Render(p)
	if err != nil {
		t.Fatalf("shipped content must render: %v", err)
	}
	if !strings.HasPrefix(text, "You are an analysis assistant") {
		t.Fatal("role preamble must lead the shipped render")
	}
}

// M5 delivery proof: the rendered EIS travels through
// model.ExecutionRequest to a mock provider byte-identical — what was
// scanned = what the model received. L1 renders; L2 delivers; L1
// never calls the model (the call below is the test standing in for
// Context Delivery).
func TestDeliveryProofMockProvider(t *testing.T) {
	set, p := renderFixture(t)
	sysMsg, renderHash, err := set.SystemMessage(p)
	if err != nil {
		t.Fatal(err)
	}
	var delivered string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Messages []struct {
				Role    string `json:"role"`
				Content string `json:"content"`
			} `json:"messages"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Error(err)
		}
		if len(req.Messages) > 0 && req.Messages[0].Role == "system" {
			delivered = req.Messages[0].Content
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"model": "mock", "done": true, "done_reason": "stop",
			"message": map[string]any{"role": "assistant", "content": "ok"},
		})
	}))
	defer srv.Close()
	provider := model.NewOllamaChat(srv.URL)
	resp, err := provider.Execute(context.Background(), model.ExecutionRequest{
		Model:    "mock",
		Messages: []model.Message{sysMsg, {Role: model.RoleUser, Content: "proceed"}},
		Options:  model.DefaultOptions(),
	})
	if err != nil {
		t.Fatal(err)
	}
	if resp == nil || delivered == "" {
		t.Fatal("mock provider saw no system message")
	}
	text, _, _ := set.Render(p)
	if delivered != text {
		t.Fatal("delivered system message must be byte-identical to the rendered EIS")
	}
	if renderHash == "" {
		t.Fatal("render hash must accompany delivery for the trace")
	}
}
