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
		"tool_registry":     h("registry"),
		"constitution":      map[string]any{"state": h("l6c"), "orchestration": h("l7c")},
		"execution_ceiling": h("eceiling"),
		"workflows": []any{map[string]any{
			"workflow": h("wf1"), "workflow_ceiling": h("wceiling"),
			"context_contract": h("contract")}},
		"models":         []any{"scripted"},
		"model_registry": "absent",
		"skill_catalog":  h("catalog"), "contract_registry": h("l10reg"),
		"criteria_registry": h("l11reg"), "regression_set_registry": h("setreg"),
		"delegation_template_registry": "absent",
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
			w := m["workflows"].([]any)[0]
			m["workflows"] = []any{w, w}
		}},
		{"bundle missing a pin", func(m map[string]any) {
			m["workflows"].([]any)[0].(map[string]any)["context_contract"] = ""
		}},
		{"missing execution ceiling pin", func(m map[string]any) { m["execution_ceiling"] = "" }},
		{"missing constitution pin", func(m map[string]any) {
			m["constitution"] = map[string]any{"state": "", "orchestration": ""}
		}},
		{"empty models", func(m map[string]any) { m["models"] = []any{} }},
		{"empty model entry", func(m map[string]any) { m["models"] = []any{""} }},
		{"model registry neither hash nor absent", func(m map[string]any) { m["model_registry"] = "local" }},
		{"delegation registry neither hash nor absent", func(m map[string]any) { m["delegation_template_registry"] = "latest" }},
		{"delegation registry missing", func(m map[string]any) { delete(m, "delegation_template_registry") }},
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

// The CRITICAL-1 wall: a symlink inside a pinned tree must REFUSE,
// never be silently skipped — consumers (the instruction loader)
// follow symlinks, so skipping would leave unpinned content inside a
// pinned tree.
func TestHashDirRefusesNonRegularEntries(t *testing.T) {
	outside := filepath.Join(t.TempDir(), "evil.md")
	if err := os.WriteFile(outside, []byte("unpinned directive\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "a.md"), []byte("alpha\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	before, err := HashDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(dir, "evil.md")); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	after, err := HashDir(dir)
	if err == nil {
		t.Fatalf("symlink accepted into a pinned tree (hash %s vs %s)", after, before)
	}
	if !strings.Contains(err.Error(), "non-regular entry") {
		t.Fatalf("wrong refusal: %v", err)
	}
}

// Duplicate artifact hashes across registrations make admission
// order-dependent (close-review LOW-1).
func TestAdmitAnchorRefusesDuplicateArtifact(t *testing.T) {
	m := validAnchorMap()
	ab, _ := json.Marshal(m)
	dir := t.TempDir()
	p := filepath.Join(dir, "anchor.json")
	os.WriteFile(p, ab, 0o644)
	rb, _ := json.Marshal(map[string]any{
		"version": 1, "kind": "deployment-anchors",
		"entries": []any{
			map[string]any{"name": "local-dev", "version": 1, "artifact_sha256": hashBytes(ab), "state": "withdrawn"},
			map[string]any{"name": "local-dev", "version": 2, "artifact_sha256": hashBytes(ab), "state": "active"},
		},
	})
	reg := filepath.Join(dir, "anchors.json")
	os.WriteFile(reg, rb, 0o644)
	if _, err := AdmitAnchor(p, hashBytes(ab), reg); err == nil || !strings.Contains(err.Error(), "more than once") {
		t.Fatalf("order-dependent admission accepted: %v", err)
	}
}

// M-4: the tree encoding must be injective — a file set cannot be
// impersonated by writing one file and deleting a sibling.
func TestHashDirFramingIsInjective(t *testing.T) {
	mk := func(files map[string]string) string {
		t.Helper()
		dir := t.TempDir()
		for name, body := range files {
			p := filepath.Join(dir, name)
			if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
				t.Fatal(err)
			}
		}
		h, err := HashDir(dir)
		if err != nil {
			t.Fatal(err)
		}
		return h
	}
	// The classic separator-collision pair: two files vs one file
	// whose content replays the separators.
	two := mk(map[string]string{"a": "", "b": "c"})
	one := mk(map[string]string{"a": "\x00b\x00c"})
	if two == one {
		t.Fatal("distinct trees share a pin — the framing is not injective")
	}
	// Sibling deletion must change the pin.
	full := mk(map[string]string{"x": "1", "y": "2"})
	partial := mk(map[string]string{"x": "1"})
	if full == partial {
		t.Fatal("deleting a sibling left the pin unchanged")
	}
}

// Negative space the close review found untested (M-8 tail): the
// loader's own bounds and shapes.
func TestAnchorLoaderBounds(t *testing.T) {
	dir := t.TempDir()
	t.Run("oversize anchor refused", func(t *testing.T) {
		p := filepath.Join(dir, "big.json")
		pad := strings.Repeat("x", maxAnchorBytes+1)
		os.WriteFile(p, []byte(`{"version":1,"pad":"`+pad+`"}`), 0o644)
		if _, err := AdmitAnchor(p, strings.Repeat("ab", 32), filepath.Join(dir, "nope.json")); err == nil {
			t.Fatal("oversize anchor read")
		}
	})
	t.Run("directory as anchor refused", func(t *testing.T) {
		sub := filepath.Join(dir, "adir")
		os.MkdirAll(sub, 0o755)
		if _, err := AdmitAnchor(sub, strings.Repeat("ab", 32), filepath.Join(dir, "nope.json")); err == nil {
			t.Fatal("directory accepted as anchor")
		}
	})
	t.Run("missing registry refuses closed", func(t *testing.T) {
		m := validAnchorMap()
		ab, _ := json.Marshal(m)
		p := filepath.Join(dir, "a.json")
		os.WriteFile(p, ab, 0o644)
		if _, err := AdmitAnchor(p, hashBytes(ab), filepath.Join(dir, "absent-registry.json")); err == nil {
			t.Fatal("missing anchors registry admitted")
		}
	})
	t.Run("registry version zero refused", func(t *testing.T) {
		m := validAnchorMap()
		ab, _ := json.Marshal(m)
		p := filepath.Join(dir, "a2.json")
		os.WriteFile(p, ab, 0o644)
		rb, _ := json.Marshal(map[string]any{"version": 0, "kind": "deployment-anchors", "entries": []any{}})
		reg := filepath.Join(dir, "reg0.json")
		os.WriteFile(reg, rb, 0o644)
		if _, err := AdmitAnchor(p, hashBytes(ab), reg); err == nil {
			t.Fatal("version-zero registry accepted")
		}
	})
	t.Run("malformed registry entry refused", func(t *testing.T) {
		m := validAnchorMap()
		ab, _ := json.Marshal(m)
		p := filepath.Join(dir, "a3.json")
		os.WriteFile(p, ab, 0o644)
		rb, _ := json.Marshal(map[string]any{"version": 1, "kind": "deployment-anchors",
			"entries": []any{map[string]any{"name": "local-dev", "version": 0, "artifact_sha256": hashBytes(ab), "state": "active"}}})
		reg := filepath.Join(dir, "reg1.json")
		os.WriteFile(reg, rb, 0o644)
		if _, err := AdmitAnchor(p, hashBytes(ab), reg); err == nil {
			t.Fatal("malformed entry accepted")
		}
	})
}

// Owner finding 3: the read path re-establishes what a recorded
// deployment identity MEANT — from the registry and the bytes, never
// from the runtime assertion that was accepted at submission.
func TestVerifyAnchorRecord(t *testing.T) {
	m := validAnchorMap()
	ab, _ := json.Marshal(m)
	dir := t.TempDir()
	rb, _ := json.Marshal(map[string]any{
		"version": 1, "kind": "deployment-anchors",
		"entries": []any{map[string]any{
			"name": "local-dev", "version": 1,
			"artifact_sha256": hashBytes(ab), "state": "active"}},
	})
	reg := filepath.Join(dir, "anchors.json")
	os.WriteFile(reg, rb, 0o644)

	t.Run("recorded identity + bytes + registry re-establish", func(t *testing.T) {
		a, err := VerifyAnchorRecord(hashBytes(ab), ab, reg)
		if err != nil || a.Name != "local-dev" {
			t.Fatalf("%+v %v", a, err)
		}
	})
	t.Run("bytes not matching the recorded identity refuse", func(t *testing.T) {
		other, _ := json.Marshal(validAnchorMap())
		if _, err := VerifyAnchorRecord(strings.Repeat("ee", 32), other, reg); err == nil {
			t.Fatal("mismatched bytes accepted")
		}
	})
	// The three entry guards below are adjacent and each catches what
	// the others do, so asserting only "refused" leaves all three
	// unevidenced — the 2026-09-14 mutation pass found every one of them
	// surviving despite these subtests existing. They must therefore
	// assert WHICH refusal, and the distinction is not cosmetic: "this
	// run declared itself unanchored", "this record is malformed", and
	// "the bytes are missing" are three different facts about a record,
	// and an auditor acts differently on each.
	t.Run("unanchored record has nothing to verify", func(t *testing.T) {
		_, err := VerifyAnchorRecord("unanchored", nil, reg)
		if err == nil {
			t.Fatal("unanchored record verified as governed")
		}
		if !strings.Contains(err.Error(), "declares an unanchored run") {
			t.Fatalf("an unanchored record must be reported as unanchored, not as malformed: %v", err)
		}
	})
	t.Run("malformed identity is malformed, not unanchored", func(t *testing.T) {
		// Well-formed bytes, ill-formed recorded identity: only the
		// syntax guard can fire.
		_, err := VerifyAnchorRecord("not-a-sha256", ab, reg)
		if err == nil {
			t.Fatal("a record carrying an ill-formed anchor identity verified")
		}
		if !strings.Contains(err.Error(), "no well-formed deployment-anchor identity") {
			t.Fatalf("refused for the wrong reason: %v", err)
		}
	})
	t.Run("missing bytes refuse", func(t *testing.T) {
		_, err := VerifyAnchorRecord(hashBytes(ab), nil, reg)
		if err == nil {
			t.Fatal("absent anchor bytes accepted")
		}
		if !strings.Contains(err.Error(), "not available in the record") {
			t.Fatalf("absent bytes must be reported as absent, not as a hash mismatch: %v", err)
		}
	})
	// G1 two-way identity, the same shape as L9's manifest-vs-
	// registration check. The registry says which anchor it admitted by
	// ARTIFACT HASH; the anchor says what it is. A registration whose
	// name or version disagrees with the bytes it admits means the
	// record cannot say which deployment governed the task — the
	// registry would name one deployment and the bytes another.
	t.Run("registration disagreeing with the anchor's self-declaration refuses", func(t *testing.T) {
		for what, entry := range map[string]map[string]any{
			"name disagrees": {
				"name": "some-other-deployment", "version": 1,
				"artifact_sha256": hashBytes(ab), "state": "active"},
			"version disagrees": {
				"name": "local-dev", "version": 7,
				"artifact_sha256": hashBytes(ab), "state": "active"},
		} {
			b, _ := json.Marshal(map[string]any{
				"version": 1, "kind": "deployment-anchors", "entries": []any{entry}})
			p := filepath.Join(dir, "disagree-"+strings.ReplaceAll(what, " ", "-")+".json")
			if err := os.WriteFile(p, b, 0o644); err != nil {
				t.Fatal(err)
			}
			_, err := VerifyAnchorRecord(hashBytes(ab), ab, p)
			if err == nil {
				t.Errorf("%s: a registration naming a different deployment than the bytes it admits VERIFIED", what)
				continue
			}
			if !strings.Contains(err.Error(), "self-declaration disagrees with its registration") {
				t.Errorf("%s: refused for the wrong reason: %v", what, err)
			}
		}
	})
	t.Run("deregistered anchor makes the deployment uninterpretable", func(t *testing.T) {
		empty, _ := json.Marshal(map[string]any{"version": 1, "kind": "deployment-anchors", "entries": []any{}})
		p2 := filepath.Join(dir, "empty.json")
		os.WriteFile(p2, empty, 0o644)
		if _, err := VerifyAnchorRecord(hashBytes(ab), ab, p2); err == nil || !strings.Contains(err.Error(), "no longer registered") {
			t.Fatalf("deregistered anchor verified: %v", err)
		}
	})
	t.Run("withdrawn anchor still explains past execution", func(t *testing.T) {
		wb, _ := json.Marshal(map[string]any{
			"version": 1, "kind": "deployment-anchors",
			"entries": []any{map[string]any{
				"name": "local-dev", "version": 1,
				"artifact_sha256": hashBytes(ab), "state": "withdrawn"}},
		})
		p3 := filepath.Join(dir, "withdrawn.json")
		os.WriteFile(p3, wb, 0o644)
		if _, err := VerifyAnchorRecord(hashBytes(ab), ab, p3); err != nil {
			t.Fatalf("withdrawal rewrote history: %v", err)
		}
	})
}

func TestRegistryAppendOnly(t *testing.T) {
	mk := func(entries ...map[string]any) *Registry {
		es := make([]any, 0, len(entries))
		for _, e := range entries {
			es = append(es, e)
		}
		b, _ := json.Marshal(map[string]any{"version": 1, "kind": "deployment-anchors", "entries": es})
		r, err := parseRegistry(b)
		if err != nil {
			t.Fatal(err)
		}
		return r
	}
	e1 := map[string]any{"name": "d", "version": 1, "artifact_sha256": strings.Repeat("11", 32), "state": "active"}
	prior := mk(e1)
	t.Run("append is legal", func(t *testing.T) {
		e2 := map[string]any{"name": "d", "version": 2, "artifact_sha256": strings.Repeat("22", 32), "state": "active"}
		if err := mk(e1, e2).CheckAppendOnly(prior); err != nil {
			t.Fatal(err)
		}
	})
	t.Run("withdrawal is legal", func(t *testing.T) {
		w := map[string]any{"name": "d", "version": 1, "artifact_sha256": strings.Repeat("11", 32), "state": "withdrawn"}
		if err := mk(w).CheckAppendOnly(prior); err != nil {
			t.Fatal(err)
		}
	})
	t.Run("un-withdrawal refused", func(t *testing.T) {
		w := map[string]any{"name": "d", "version": 1, "artifact_sha256": strings.Repeat("11", 32), "state": "withdrawn"}
		if err := mk(e1).CheckAppendOnly(mk(w)); err == nil {
			t.Fatal("un-withdrawal accepted")
		}
	})
	t.Run("rebinding refused", func(t *testing.T) {
		r := map[string]any{"name": "d", "version": 1, "artifact_sha256": strings.Repeat("33", 32), "state": "active"}
		if err := mk(r).CheckAppendOnly(prior); err == nil {
			t.Fatal("rebinding accepted")
		}
	})
	t.Run("deletion refused", func(t *testing.T) {
		if err := mk().CheckAppendOnly(prior); err == nil {
			t.Fatal("deletion accepted")
		}
	})
}

// D-SA-9: the Skill allowlist is a closed, exact set. Absent is legal
// (it admits no skill-attributed task — enforced at L7); a floating
// reference or a duplicate is an anchor defect.
func TestSkillAllowlistRules(t *testing.T) {
	cases := []struct {
		name   string
		skills any
		want   string
	}{
		{"absent is legal", nil, ""},
		{"exact refs", []any{"investigate-cve@1", "remediate-dependency@1"}, ""},
		{"floating ref refused", []any{"investigate-cve@latest"}, "exact name@version"},
		{"name-only refused", []any{"investigate-cve"}, "exact name@version"},
		{"duplicate refused", []any{"investigate-cve@1", "investigate-cve@1"}, "duplicate skill allowlist entry"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			m := validAnchorMap()
			if c.skills != nil {
				m["skills"] = c.skills
			}
			raw, _ := json.Marshal(m)
			a, err := ParseAnchor(raw, "test")
			if c.want == "" {
				if err != nil {
					t.Fatalf("must parse: %v", err)
				}
				if c.skills != nil && len(a.Skills) != len(c.skills.([]any)) {
					t.Fatalf("skills not carried: %v", a.Skills)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), c.want) {
				t.Fatalf("refused for the wrong reason (want %q): %v", c.want, err)
			}
		})
	}
}
