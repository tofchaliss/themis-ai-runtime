package ratchet

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// writeCriterionFixture writes a valid criterion under dir and
// returns its bytes.
func writeCriterionFixture(t *testing.T, dir, rel string) []byte {
	t.Helper()
	raw := marshalCriterion(t, validCriterionMap())
	p := filepath.Join(dir, rel)
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, raw, 0o644); err != nil {
		t.Fatal(err)
	}
	return raw
}

func writeRegistry(t *testing.T, dir string, reg map[string]any) string {
	t.Helper()
	b, err := json.Marshal(reg)
	if err != nil {
		t.Fatal(err)
	}
	p := filepath.Join(dir, "registry.json")
	if err := os.WriteFile(p, b, 0o644); err != nil {
		t.Fatal(err)
	}
	return p
}

func criteriaRegistryMap(entries ...map[string]any) map[string]any {
	es := make([]any, 0, len(entries))
	for _, e := range entries {
		es = append(es, e)
	}
	return map[string]any{"version": 1, "kind": "criteria", "entries": es}
}

func TestRegistryResolveCriterion(t *testing.T) {
	dir := t.TempDir()
	raw := writeCriterionFixture(t, dir, "bench-score-delta/criterion.json")
	p := writeRegistry(t, dir, criteriaRegistryMap(map[string]any{
		"name": "bench-score-delta", "version": 1,
		"artifact_sha256": hashBytes(raw),
		"artifact_path":   "bench-score-delta/criterion.json",
		"state":           "active",
	}))
	r, err := LoadRegistry(p)
	if err != nil {
		t.Fatal(err)
	}
	entry, c, err := r.ResolveCriterion("bench-score-delta@1")
	if err != nil {
		t.Fatal(err)
	}
	if entry.Name != c.Name || c.SHA256 != entry.Artifact {
		t.Fatal("identity chain broken")
	}
}

func TestRegistryRefusals(t *testing.T) {
	dir := t.TempDir()
	raw := writeCriterionFixture(t, dir, "k/criterion.json")
	entry := func() map[string]any {
		return map[string]any{
			"name": "bench-score-delta", "version": 1,
			"artifact_sha256": hashBytes(raw),
			"artifact_path":   "k/criterion.json",
			"state":           "active",
		}
	}

	t.Run("unregistered is data", func(t *testing.T) {
		p := writeRegistry(t, dir, criteriaRegistryMap(entry()))
		r, err := LoadRegistry(p)
		if err != nil {
			t.Fatal(err)
		}
		if _, _, err := r.ResolveCriterion("other-criterion@1"); err == nil || !strings.Contains(err.Error(), "not registered") {
			t.Fatalf("unregistered resolution not refused: %v", err)
		}
	})
	t.Run("floating refs refused", func(t *testing.T) {
		p := writeRegistry(t, dir, criteriaRegistryMap(entry()))
		r, _ := LoadRegistry(p)
		for _, ref := range []string{"bench-score-delta", "bench-score-delta@latest", "bench-score-delta@01", "bench-score-delta@1..2", "@1"} {
			if _, _, err := r.ResolveCriterion(ref); err == nil {
				t.Fatalf("floating ref %q accepted", ref)
			}
		}
	})
	t.Run("withdrawn refuses new comparison", func(t *testing.T) {
		e := entry()
		e["state"] = "withdrawn"
		p := writeRegistry(t, dir, criteriaRegistryMap(e))
		r, _ := LoadRegistry(p)
		if _, _, err := r.ResolveCriterion("bench-score-delta@1"); err == nil || !strings.Contains(err.Error(), "withdrawn") {
			t.Fatalf("withdrawn resolution not refused: %v", err)
		}
	})
	t.Run("hash mismatch refused", func(t *testing.T) {
		e := entry()
		e["artifact_sha256"] = strings.Repeat("ab", 32)
		p := writeRegistry(t, dir, criteriaRegistryMap(e))
		r, _ := LoadRegistry(p)
		if _, _, err := r.ResolveCriterion("bench-score-delta@1"); err == nil || !strings.Contains(err.Error(), "registered hash") {
			t.Fatalf("hash mismatch not refused: %v", err)
		}
	})
	t.Run("two-way identity", func(t *testing.T) {
		e := entry()
		e["name"] = "other-name"
		p := writeRegistry(t, dir, criteriaRegistryMap(e))
		r, _ := LoadRegistry(p)
		if _, _, err := r.ResolveCriterion("other-name@1"); err == nil || !strings.Contains(err.Error(), "two-way identity") {
			t.Fatalf("self-declaration disagreement not refused: %v", err)
		}
	})
	t.Run("absolute artifact path refused", func(t *testing.T) {
		e := entry()
		e["artifact_path"] = "/etc/passwd"
		if _, err := LoadRegistry(writeRegistry(t, dir, criteriaRegistryMap(e))); err == nil {
			t.Fatal("absolute path accepted")
		}
	})
	t.Run("path traversal refused", func(t *testing.T) {
		e := entry()
		e["artifact_path"] = "../outside.json"
		if _, err := LoadRegistry(writeRegistry(t, dir, criteriaRegistryMap(e))); err == nil {
			t.Fatal("traversal accepted")
		}
	})
	t.Run("duplicate registration refused", func(t *testing.T) {
		if _, err := LoadRegistry(writeRegistry(t, dir, criteriaRegistryMap(entry(), entry()))); err == nil {
			t.Fatal("duplicate registration accepted")
		}
	})
	t.Run("unknown kind refused", func(t *testing.T) {
		m := criteriaRegistryMap(entry())
		m["kind"] = "candidates" // a candidate registry must not exist (D-L11-11)
		if _, err := LoadRegistry(writeRegistry(t, dir, m)); err == nil {
			t.Fatal("candidate registry kind accepted")
		}
	})
	t.Run("kind mismatch on resolve", func(t *testing.T) {
		p := writeRegistry(t, dir, criteriaRegistryMap(entry()))
		r, _ := LoadRegistry(p)
		if _, _, err := r.ResolveSet("bench-score-delta@1"); err == nil {
			t.Fatal("set resolution against criteria registry accepted")
		}
	})
}

func TestRegistryAppendOnly(t *testing.T) {
	dir := t.TempDir()
	raw := writeCriterionFixture(t, dir, "k/criterion.json")
	e := map[string]any{
		"name": "bench-score-delta", "version": 1,
		"artifact_sha256": hashBytes(raw),
		"artifact_path":   "k/criterion.json",
		"state":           "active",
	}
	prior, err := LoadRegistry(writeRegistry(t, dir, criteriaRegistryMap(e)))
	if err != nil {
		t.Fatal(err)
	}

	t.Run("disappearance refused", func(t *testing.T) {
		cur, _ := LoadRegistry(writeRegistry(t, dir, criteriaRegistryMap()))
		if err := cur.CheckAppendOnly(prior); err == nil {
			t.Fatal("disappeared registration accepted")
		}
	})
	t.Run("rebinding refused", func(t *testing.T) {
		e2 := map[string]any{}
		for k, v := range e {
			e2[k] = v
		}
		e2["artifact_sha256"] = strings.Repeat("cd", 32)
		cur, _ := LoadRegistry(writeRegistry(t, dir, criteriaRegistryMap(e2)))
		if err := cur.CheckAppendOnly(prior); err == nil {
			t.Fatal("rebinding accepted")
		}
	})
	t.Run("un-withdrawal refused", func(t *testing.T) {
		ew := map[string]any{}
		for k, v := range e {
			ew[k] = v
		}
		ew["state"] = "withdrawn"
		withdrawnPrior, _ := LoadRegistry(writeRegistry(t, dir, criteriaRegistryMap(ew)))
		cur, _ := LoadRegistry(writeRegistry(t, dir, criteriaRegistryMap(e)))
		if err := cur.CheckAppendOnly(withdrawnPrior); err == nil {
			t.Fatal("un-withdrawal accepted")
		}
	})
	t.Run("append is legal", func(t *testing.T) {
		e2 := map[string]any{
			"name": "second-criterion", "version": 1,
			"artifact_sha256": strings.Repeat("ef", 32),
			"artifact_path":   "k2/criterion.json",
			"state":           "active",
		}
		cur, err := LoadRegistry(writeRegistry(t, dir, criteriaRegistryMap(e, e2)))
		if err != nil {
			t.Fatal(err)
		}
		if err := cur.CheckAppendOnly(prior); err != nil {
			t.Fatalf("legal append refused: %v", err)
		}
	})
}

func validSetMap() map[string]any {
	return map[string]any{
		"version": 1, "name": "core-regression", "set_version": 1,
		"members": []any{"bench-score-delta@1"},
	}
}

func TestRegressionSetParse(t *testing.T) {
	b, _ := json.Marshal(validSetMap())
	s, err := ParseRegressionSet(b, "test")
	if err != nil {
		t.Fatal(err)
	}
	if s.Members[0] != "bench-score-delta@1" {
		t.Fatal("member lost")
	}

	cases := []struct {
		name   string
		mutate func(m map[string]any)
	}{
		{"empty members", func(m map[string]any) { m["members"] = []any{} }},
		{"floating member", func(m map[string]any) { m["members"] = []any{"bench-score-delta"} }},
		{"latest member", func(m map[string]any) { m["members"] = []any{"bench-score-delta@latest"} }},
		{"duplicate member", func(m map[string]any) { m["members"] = []any{"bench-score-delta@1", "bench-score-delta@1"} }},
		{"normative name", func(m map[string]any) { m["name"] = "safe-set" }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			m := validSetMap()
			tc.mutate(m)
			b, _ := json.Marshal(m)
			if _, err := ParseRegressionSet(b, "test"); err == nil {
				t.Fatal("doctored set accepted")
			}
		})
	}
}

func TestSetRegistryResolve(t *testing.T) {
	dir := t.TempDir()
	sb, _ := json.Marshal(validSetMap())
	if err := os.WriteFile(filepath.Join(dir, "set.json"), sb, 0o644); err != nil {
		t.Fatal(err)
	}
	p := writeRegistry(t, dir, map[string]any{
		"version": 1, "kind": "regression-sets",
		"entries": []any{map[string]any{
			"name": "core-regression", "version": 1,
			"artifact_sha256": hashBytes(sb),
			"artifact_path":   "set.json",
			"state":           "active",
		}},
	})
	r, err := LoadRegistry(p)
	if err != nil {
		t.Fatal(err)
	}
	_, s, err := r.ResolveSet("core-regression@1")
	if err != nil {
		t.Fatal(err)
	}
	if len(s.Members) != 1 {
		t.Fatal("set members lost")
	}
}
