package ratchet

import (
	"encoding/json"
	"strings"
	"testing"
)

func validCandidateMap() map[string]any {
	return map[string]any{
		"artifact": "l11-candidate", "schema": 1,
		"family":          "skill-revision",
		"target":          "investigate-cve",
		"change_relation": "create-version",
		"content_sha256":  strings.Repeat("77", 32),
		"claimed_baseline": map[string]any{
			"door": "l9-catalog", "name": "investigate-cve", "version": 1,
			"artifact_sha256": strings.Repeat("22", 32),
		},
		"author":             "qwen2.5:7b",
		"authored_via":       "governed-task:t-0042",
		"advisory_rationale": "narrower tool grants in phase two",
	}
}

func TestCandidateParses(t *testing.T) {
	b, _ := json.Marshal(validCandidateMap())
	c, err := ParseCandidate(b, "test")
	if err != nil {
		t.Fatal(err)
	}
	if c.Author == "" {
		t.Fatal("authorship lost")
	}
}

func TestCandidateRefusals(t *testing.T) {
	cases := []struct {
		name   string
		mutate func(m map[string]any)
	}{
		{"unknown family", func(m map[string]any) { m["family"] = "model-swap" }},
		{"unknown change relation", func(m map[string]any) { m["change_relation"] = "hotfix" }},
		{"no content pin", func(m map[string]any) { m["content_sha256"] = "" }},
		{"unattributable author", func(m map[string]any) { m["author"] = "" }},
		{"unattributable channel", func(m map[string]any) { m["authored_via"] = "" }},
		{"malformed supersedes", func(m map[string]any) { m["supersedes"] = []any{"not-a-hash"} }},
		{"malformed evidence ref", func(m map[string]any) { m["evidence_refs"] = []any{"abc"} }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			m := validCandidateMap()
			tc.mutate(m)
			b, _ := json.Marshal(m)
			if _, err := ParseCandidate(b, "test"); err == nil {
				t.Fatal("doctored candidate accepted")
			}
		})
	}
}

// A candidate has NO lifecycle state: any status-like field is an
// unknown field under the closed schema and refused (D-L11-3/11).
func TestCandidateHasNoLifecycleState(t *testing.T) {
	for _, field := range []string{"status", "position", "disposition", "promoted", "evaluated", "state"} {
		m := validCandidateMap()
		m[field] = "pending"
		b, _ := json.Marshal(m)
		if _, err := ParseCandidate(b, "test"); err == nil {
			t.Fatalf("lifecycle field %q accepted — candidate registry in disguise", field)
		}
	}
}

func validPlanMap() map[string]any {
	return map[string]any{
		"artifact": "l11-evaluation-plan", "schema": 1,
		"candidate_hash": strings.Repeat("77", 32),
		"criteria":       []any{"bench-score-delta@1"},
		"required_runs": []any{
			map[string]any{"skill": "investigate-cve@1", "input_sha256": strings.Repeat("88", 32)},
		},
		"required_provenance": []any{"model_identity", "plan_reference"},
	}
}

func TestPlanParses(t *testing.T) {
	b, _ := json.Marshal(validPlanMap())
	if _, err := ParsePlan(b, "test"); err != nil {
		t.Fatal(err)
	}
}

func TestPlanRefusals(t *testing.T) {
	cases := []struct {
		name   string
		mutate func(m map[string]any)
	}{
		{"no pins", func(m map[string]any) { delete(m, "criteria") }},
		{"floating pin", func(m map[string]any) { m["criteria"] = []any{"bench-score-delta"} }},
		{"latest pin", func(m map[string]any) { m["criteria"] = []any{"bench-score-delta@latest"} }},
		{"no runs", func(m map[string]any) { m["required_runs"] = []any{} }},
		{"floating skill", func(m map[string]any) {
			m["required_runs"].([]any)[0].(map[string]any)["skill"] = "investigate-cve"
		}},
		{"no input commitment", func(m map[string]any) {
			m["required_runs"].([]any)[0].(map[string]any)["input_sha256"] = ""
		}},
		{"implicit provenance", func(m map[string]any) { m["required_provenance"] = []any{} }},
		{"model selection smuggled as provenance", func(m map[string]any) {
			m["required_provenance"] = []any{"choose_best_model"}
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			m := validPlanMap()
			tc.mutate(m)
			b, _ := json.Marshal(m)
			if _, err := ParsePlan(b, "test"); err == nil {
				t.Fatal("doctored plan accepted")
			}
		})
	}
}

// The plan schema is declarative-only: imperative/continuation
// vocabulary is structurally unrepresentable — closed schema refuses
// unknown fields (D-L11-10 owner lock).
func TestPlanImperativeUnrepresentable(t *testing.T) {
	for _, field := range []string{"retry_until", "on_failure", "then", "fallback_skill", "max_attempts", "schedule"} {
		m := validPlanMap()
		m[field] = "anything"
		b, _ := json.Marshal(m)
		if _, err := ParsePlan(b, "test"); err == nil {
			t.Fatalf("imperative field %q accepted", field)
		}
	}
}

func TestPlanConformance(t *testing.T) {
	pb, _ := json.Marshal(validPlanMap())
	plan, err := ParsePlan(pb, "test")
	if err != nil {
		t.Fatal(err)
	}
	planID := "sha256:" + strings.Repeat("99", 32)

	in := validInput(t, 0.9, 0.8)
	in.PlanRef = planID
	in.CandidateHash = plan.CandidateHash
	pkg, _, err := Compare(in)
	if err != nil {
		t.Fatal(err)
	}
	if got := CheckPlanConformance(plan, planID, pkg); len(got) != 0 {
		t.Fatalf("conforming package reported mismatches: %v", got)
	}

	other := *pkg
	other.CandidateHash = strings.Repeat("aa", 32)
	if got := CheckPlanConformance(plan, planID, &other); len(got) == 0 {
		t.Fatal("nonconforming package reported clean")
	}
}
