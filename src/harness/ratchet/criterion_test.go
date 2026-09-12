package ratchet

import (
	"encoding/json"
	"strings"
	"testing"
)

// validCriterionMap returns a fully valid criterion as a mutable map
// — each refusal test doctors exactly one aspect (the L10 doctored-
// definition pattern).
func validCriterionMap() map[string]any {
	return map[string]any{
		"version":           1,
		"name":              "bench-score-delta",
		"criterion_version": 1,
		"families":          []any{"skill-revision"},
		"candidate_selectors": []any{
			map[string]any{"name": "candidate_score", "source": "benchmark_validated_score", "params": map[string]any{"benchmark": "themis-bench-core"}},
		},
		"baseline_selectors": []any{
			map[string]any{"name": "baseline_score", "source": "benchmark_validated_score", "params": map[string]any{"benchmark": "themis-bench-core"}},
		},
		"comparator": map[string]any{"name": "numeric-score-delta", "version": 1},
		"config":     map[string]any{},
		"delta_shape": []any{
			map[string]any{"name": "score_delta", "type": "number"},
		},
		"ordering": map[string]any{
			"kind": "per-metric",
			"fields": []any{
				map[string]any{"name": "score_delta", "direction": "maximize", "equal_tolerance": 0.0, "non_regression_min": -0.05},
			},
		},
		"baseline_constraints": []any{"requires-current-active"},
		"provenance":           []any{"evidence_records", "run_identities", "admission_observation"},
	}
}

func marshalCriterion(t *testing.T, m map[string]any) []byte {
	t.Helper()
	b, err := json.Marshal(m)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func TestCriterionValidLoads(t *testing.T) {
	c, err := ParseCriterion(marshalCriterion(t, validCriterionMap()), "test")
	if err != nil {
		t.Fatalf("valid criterion refused: %v", err)
	}
	if c.Name != "bench-score-delta" || c.Criterion != 1 {
		t.Fatalf("identity mismatch: %s@%d", c.Name, c.Criterion)
	}
	if c.SHA256 == "" || len(c.Raw) == 0 {
		t.Fatal("identity bytes not captured")
	}
}

func TestCriterionRefusals(t *testing.T) {
	cases := []struct {
		name    string
		mutate  func(m map[string]any)
		wantSub string
	}{
		{"unknown family", func(m map[string]any) { m["families"] = []any{"model-swap"} }, "unknown candidate family"},
		{"no families", func(m map[string]any) { m["families"] = []any{} }, "at least one applicable family"},
		{"selector source is L11 output", func(m map[string]any) {
			m["candidate_selectors"].([]any)[0].(map[string]any)["source"] = "l11_comparative_package"
		}, "L11 output is terminal"},
		{"selector params not object", func(m map[string]any) {
			m["candidate_selectors"].([]any)[0].(map[string]any)["params"] = []any{}
		}, "params must be a JSON object"},
		{"no baseline selectors", func(m map[string]any) { m["baseline_selectors"] = []any{} }, "at least one baseline selector"},
		{"bad comparator version", func(m map[string]any) {
			m["comparator"].(map[string]any)["version"] = 0
		}, "comparator.version"},
		{"config not object", func(m map[string]any) { m["config"] = "auto" }, "config must be a JSON object"},
		{"empty delta shape", func(m map[string]any) { m["delta_shape"] = []any{} }, "delta_shape required"},
		{"unknown delta type", func(m map[string]any) {
			m["delta_shape"].([]any)[0].(map[string]any)["type"] = "expression"
		}, "unknown type"},
		{"normative delta field name", func(m map[string]any) {
			m["delta_shape"].([]any)[0].(map[string]any)["name"] = "security_improved"
			m["ordering"] = nil
		}, "asserts a proposition"},
		{"normative criterion name", func(m map[string]any) { m["name"] = "safer-skill" }, "asserts a proposition"},
		{"unknown ordering kind", func(m map[string]any) {
			m["ordering"].(map[string]any)["kind"] = "ranked-choice"
		}, "unknown ordering kind"},
		{"ordering field outside delta shape", func(m map[string]any) {
			m["ordering"].(map[string]any)["fields"].([]any)[0].(map[string]any)["name"] = "latency_delta"
		}, "not a delta_shape field"},
		{"missing equal_tolerance", func(m map[string]any) {
			delete(m["ordering"].(map[string]any)["fields"].([]any)[0].(map[string]any), "equal_tolerance")
		}, "equal_tolerance"},
		{"missing non_regression_min", func(m map[string]any) {
			delete(m["ordering"].(map[string]any)["fields"].([]any)[0].(map[string]any), "non_regression_min")
		}, "non_regression_min"},
		{"weight outside scalarization", func(m map[string]any) {
			m["ordering"].(map[string]any)["fields"].([]any)[0].(map[string]any)["weight"] = 0.5
		}, "only meaningful under scalarization"},
		{"scalarization without weight", func(m map[string]any) {
			m["ordering"].(map[string]any)["kind"] = "scalarization"
		}, "positive weight"},
		{"unknown baseline constraint", func(m map[string]any) {
			m["baseline_constraints"] = []any{"requires-recent"}
		}, "unknown baseline constraint"},
		{"relaxed provenance", func(m map[string]any) {
			m["provenance"] = []any{"evidence_records"}
		}, "not criterion-relaxable"},
		{"bad version", func(m map[string]any) { m["version"] = 2 }, "version must be 1"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			m := validCriterionMap()
			tc.mutate(m)
			_, err := ParseCriterion(marshalCriterion(t, m), "test")
			if err == nil {
				t.Fatalf("doctored criterion accepted")
			}
			if !strings.Contains(err.Error(), tc.wantSub) {
				t.Fatalf("wrong refusal: %v (want %q)", err, tc.wantSub)
			}
		})
	}
}

func TestCriterionUnknownFieldRefused(t *testing.T) {
	m := validCriterionMap()
	// A continuation field: the schema must have no room for
	// consequences (D-L11-6 §5) — closed schema refuses it as unknown.
	m["then"] = map[string]any{"promote": true}
	_, err := ParseCriterion(marshalCriterion(t, m), "test")
	if err == nil {
		t.Fatal("continuation field accepted — the closed schema wall is gone")
	}
}

func TestCriterionDuplicateKeyRefused(t *testing.T) {
	raw := marshalCriterion(t, validCriterionMap())
	doctored := strings.Replace(string(raw), `"version":1`, `"version":1,"version":1`, 1)
	if _, err := ParseCriterion([]byte(doctored), "test"); err == nil {
		t.Fatal("duplicate key accepted")
	}
}

func TestOrderingNoneDeclaresNoFields(t *testing.T) {
	m := validCriterionMap()
	m["ordering"] = map[string]any{"kind": "none", "fields": []any{}}
	if _, err := ParseCriterion(marshalCriterion(t, m), "test"); err != nil {
		t.Fatalf("kind none refused: %v", err)
	}
	m["ordering"].(map[string]any)["fields"] = []any{map[string]any{"name": "score_delta", "direction": "maximize"}}
	if _, err := ParseCriterion(marshalCriterion(t, m), "test"); err == nil {
		t.Fatal("kind none with fields accepted")
	}
}

func TestOrderingAbsentIsLegal(t *testing.T) {
	m := validCriterionMap()
	delete(m, "ordering")
	c, err := ParseCriterion(marshalCriterion(t, m), "test")
	if err != nil {
		t.Fatalf("descriptive criterion refused: %v", err)
	}
	if c.Ordering != nil {
		t.Fatal("absent ordering decoded as present")
	}
}
