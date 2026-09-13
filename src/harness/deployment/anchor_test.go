package deployment

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func validAnchorMap() map[string]any {
	h := func(seed string) string { return hashBytes([]byte(seed)) }
	return map[string]any{
		"version": 1, "name": "local-dev", "deployment_version": 1,
		"instruction_root_safety": h("safety"), "instruction_root_system": h("system"),
		"instruction_root_themis": h("themis"), "instruction_policy": h("policy"),
		"tool_registry": h("registry"), "workflow_ceiling": h("wceiling"),
		"exec_ceiling": h("eceiling"), "context_contract": h("contract"),
		"workflows": []any{h("wf1")}, "models": []any{"scripted"},
		"skill_catalog": h("catalog"), "contract_registry": h("l10reg"),
		"criteria_registry": h("l11reg"), "regression_set_registry": h("setreg"),
	}
}

func writeAnchorWorld(t *testing.T, anchorMap map[string]any, entryState string, registerHash string) (string, string, string) {
	t.Helper()
	dir := t.TempDir()
	ab, _ := json.Marshal(anchorMap)
	anchorPath := filepath.Join(dir, "anchor.json")
	os.WriteFile(anchorPath, ab, 0o644)
	if registerHash == "" {
		registerHash = hashBytes(ab)
	}
	rb, _ := json.Marshal(map[string]any{
		"version": 1, "kind": "deployment-anchors",
		"entries": []any{map[string]any{
			"name": anchorMap["name"], "version": anchorMap["deployment_version"],
			"artifact_sha256": registerHash, "state": entryState}},
	})
	regPath := filepath.Join(dir, "anchors.json")
	os.WriteFile(regPath, rb, 0o644)
	return anchorPath, hashBytes(ab), regPath
}

func TestAdmitAnchor(t *testing.T) {
	t.Run("admitted active anchor opens", func(t *testing.T) {
		p, h, reg := writeAnchorWorld(t, validAnchorMap(), "active", "")
		a, err := AdmitAnchor(p, h, reg)
		if err != nil {
			t.Fatal(err)
		}
		if a.SHA256 != h || a.Name != "local-dev" {
			t.Fatalf("identity wrong: %+v", a)
		}
	})
	t.Run("operator hash mismatch refused", func(t *testing.T) {
		p, _, reg := writeAnchorWorld(t, validAnchorMap(), "active", "")
		if _, err := AdmitAnchor(p, strings.Repeat("ab", 32), reg); err == nil {
			t.Fatal("hash mismatch admitted")
		}
	})
	t.Run("unregistered anchor refused — the D-G1-1A core", func(t *testing.T) {
		// The forged-anchor attack: self-consistent bytes, correct
		// self-computed hash, NO Governance registration.
		p, h, reg := writeAnchorWorld(t, validAnchorMap(), "active", strings.Repeat("cd", 32))
		_, err := AdmitAnchor(p, h, reg)
		if err == nil || !strings.Contains(err.Error(), "identifier, never an admission claim") {
			t.Fatalf("forged anchor admitted: %v", err)
		}
	})
	t.Run("withdrawn anchor refused", func(t *testing.T) {
		p, h, reg := writeAnchorWorld(t, validAnchorMap(), "withdrawn", "")
		if _, err := AdmitAnchor(p, h, reg); err == nil || !strings.Contains(err.Error(), "withdrawn") {
			t.Fatalf("withdrawn anchor admitted: %v", err)
		}
	})
	t.Run("two-way identity refused", func(t *testing.T) {
		m := validAnchorMap()
		m["deployment_version"] = 2 // registry registers version 1
		ab, _ := json.Marshal(m)
		dir := t.TempDir()
		p := filepath.Join(dir, "anchor.json")
		os.WriteFile(p, ab, 0o644)
		rb, _ := json.Marshal(map[string]any{
			"version": 1, "kind": "deployment-anchors",
			"entries": []any{map[string]any{
				"name": "local-dev", "version": 1,
				"artifact_sha256": hashBytes(ab), "state": "active"}},
		})
		reg := filepath.Join(dir, "anchors.json")
		os.WriteFile(reg, rb, 0o644)
		if _, err := AdmitAnchor(p, hashBytes(ab), reg); err == nil || !strings.Contains(err.Error(), "two-way identity") {
			t.Fatalf("self-declaration disagreement admitted: %v", err)
		}
	})
	t.Run("duplicate registry identity refused", func(t *testing.T) {
		m := validAnchorMap()
		ab, _ := json.Marshal(m)
		dir := t.TempDir()
		p := filepath.Join(dir, "anchor.json")
		os.WriteFile(p, ab, 0o644)
		e := map[string]any{"name": "local-dev", "version": 1,
			"artifact_sha256": hashBytes(ab), "state": "active"}
		rb, _ := json.Marshal(map[string]any{"version": 1, "kind": "deployment-anchors", "entries": []any{e, e}})
		reg := filepath.Join(dir, "anchors.json")
		os.WriteFile(reg, rb, 0o644)
		if _, err := AdmitAnchor(p, hashBytes(ab), reg); err == nil {
			t.Fatal("duplicate registration admitted")
		}
	})
	t.Run("wrong registry kind refused", func(t *testing.T) {
		p, h, _ := writeAnchorWorld(t, validAnchorMap(), "active", "")
		dir := t.TempDir()
		rb, _ := json.Marshal(map[string]any{"version": 1, "kind": "criteria", "entries": []any{}})
		reg := filepath.Join(dir, "anchors.json")
		os.WriteFile(reg, rb, 0o644)
		if _, err := AdmitAnchor(p, h, reg); err == nil {
			t.Fatal("wrong-kind registry accepted")
		}
	})
}

func TestParseAnchorRefusals(t *testing.T) {
	cases := []struct {
		name   string
		mutate func(m map[string]any)
	}{
		{"missing pin", func(m map[string]any) { m["tool_registry"] = "" }},
		{"malformed pin", func(m map[string]any) { m["skill_catalog"] = "zz" }},
		{"empty workflows", func(m map[string]any) { m["workflows"] = []any{} }},
		{"duplicate workflow", func(m map[string]any) {
			h := m["workflows"].([]any)[0]
			m["workflows"] = []any{h, h}
		}},
		{"empty models", func(m map[string]any) { m["models"] = []any{} }},
		{"empty model entry", func(m map[string]any) { m["models"] = []any{""} }},
		{"bad name", func(m map[string]any) { m["name"] = "Local Dev" }},
		{"version zero", func(m map[string]any) { m["deployment_version"] = 0 }},
		{"unknown field", func(m map[string]any) { m["auto_approve"] = true }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			m := validAnchorMap()
			tc.mutate(m)
			b, _ := json.Marshal(m)
			if _, err := ParseAnchor(b, "test"); err == nil {
				t.Fatal("doctored anchor accepted")
			}
		})
	}
	t.Run("duplicate key refused", func(t *testing.T) {
		b, _ := json.Marshal(validAnchorMap())
		doctored := strings.Replace(string(b), `"version":1`, `"version":1,"version":1`, 1)
		if _, err := ParseAnchor([]byte(doctored), "test"); err == nil {
			t.Fatal("duplicate key accepted")
		}
	})
}

func TestHashDirDeterministic(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, "sub"), 0o755)
	os.WriteFile(filepath.Join(dir, "a.md"), []byte("alpha\n"), 0o644)
	os.WriteFile(filepath.Join(dir, "sub", "b.md"), []byte("beta\n"), 0o644)
	h1, err := HashDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	h2, _ := HashDir(dir)
	if h1 != h2 {
		t.Fatal("dir hash unstable")
	}
	os.WriteFile(filepath.Join(dir, "sub", "b.md"), []byte("beta!\n"), 0o644)
	h3, _ := HashDir(dir)
	if h3 == h1 {
		t.Fatal("content change invisible to dir hash")
	}
}
