package skills

// Regressions from the L9 Class-3 security and architecture reviews.
// Each test pins one remediated finding to its branch.

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Security HIGH-1: an unvalidated task id reached filepath.Join and
// could overwrite sibling governed artifacts. The id is now held to
// L6's own predicate, and every write resolves through the canonical
// mutation predicate.
func TestTraversingTaskIDRefused(t *testing.T) {
	b := newBundle(t)
	victimDir := t.TempDir()
	victim := filepath.Join(victimDir, "victim-grant.json")
	if err := os.WriteFile(victim, []byte(`{"governed":"artifact"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	for _, bad := range []string{
		"../victim", "../../etc/passwd", "a/b", ".hidden", "",
		strings.Repeat("x", 129), "task id with spaces", "task\x00id",
	} {
		req := b.request(t)
		req.TaskID = bad
		req.OutDir = victimDir
		if _, err := Instantiate(b.catalogPath, "investigate-cve@1", req); !errors.Is(err, ErrResolve) {
			t.Fatalf("task_id %q must refuse before any write: %v", bad, err)
		}
	}
	// The victim artifact is untouched: the refusal precedes the write.
	got, err := os.ReadFile(victim)
	if err != nil || string(got) != `{"governed":"artifact"}` {
		t.Fatalf("a refused instantiation must not have written anything: %q %v", got, err)
	}
}

// Security HIGH-1 companion: retry_of also names a task identity and
// is held to the same predicate.
func TestInvalidRetryOfRefused(t *testing.T) {
	b := newBundle(t)
	req := b.request(t)
	req.RetryOf = "../elsewhere"
	if _, err := Instantiate(b.catalogPath, "investigate-cve@1", req); !errors.Is(err, ErrResolve) ||
		!strings.Contains(err.Error(), "retry_of") {
		t.Fatalf("an invalid retry_of must refuse: %v", err)
	}
}

// Security MED-1 / architecture HIGH: wall 2 was satisfiable by
// supplying only one root. Every task-writable root is now required —
// the workspace root especially, since that is the one a task holds
// write_file on.
func TestDisjointnessRequiresEveryWritableRoot(t *testing.T) {
	b := newBundle(t)
	for _, missing := range []string{"state", "artifacts", "workspace"} {
		req := b.request(t)
		switch missing {
		case "state":
			req.Deployment.StateRoot = ""
		case "artifacts":
			req.Deployment.ArtifactDir = ""
		case "workspace":
			req.Deployment.WorkspaceRoot = ""
		}
		if _, err := Instantiate(b.catalogPath, "investigate-cve@1", req); !errors.Is(err, ErrCatalog) ||
			!strings.Contains(err.Error(), "required to prove catalog disjointness") {
			t.Fatalf("omitting the %s root must refuse — the wall must not pass by luck: %v", missing, err)
		}
	}
	// And the workspace-collides case specifically: the catalog inside
	// the root a task can write to.
	req := b.request(t)
	req.Deployment.WorkspaceRoot = b.catalogDir
	if _, err := Instantiate(b.catalogPath, "investigate-cve@1", req); !errors.Is(err, ErrCatalog) ||
		!strings.Contains(err.Error(), "disjoint") {
		t.Fatalf("a catalog inside the workspace must refuse: %v", err)
	}
}

// Test review MEDIUM: the enum, integer, and boolean branches of
// Validate were entirely uncovered — D-L9-4 locks them as part of the
// closed v1 vocabulary, so they were dead code with respect to the
// suite. Each branch is exercised both ways here.
func TestValidateClosedVocabularyBranches(t *testing.T) {
	schema, err := parseSchema([]byte(`{"version":1,"fields":[
	 {"name":"sev","type":"string","required":true,"max_length":16,"enum":["low","high"]},
	 {"name":"depth","type":"integer"},
	 {"name":"deep","type":"boolean"}]}`))
	if err != nil {
		t.Fatal(err)
	}
	if err := schema.Validate(map[string]any{"sev": "high", "depth": 3, "deep": true}); err != nil {
		t.Fatalf("a conforming input must validate: %v", err)
	}
	// A Go int and a JSON-decoded float64 are held to the same rule.
	if err := schema.Validate(map[string]any{"sev": "low", "depth": float64(4)}); err != nil {
		t.Fatalf("an integral float64 must validate: %v", err)
	}
	for name, bad := range map[string]map[string]any{
		"enum non-member":      {"sev": "critical"},
		"enum wrong type":      {"sev": 3},
		"integer non-integral": {"sev": "low", "depth": 1.5},
		"integer wrong type":   {"sev": "low", "depth": "3"},
		"integer unbounded":    {"sev": "low", "depth": float64(1 << 54)},
		"boolean wrong type":   {"sev": "low", "deep": "yes"},
		"string too long":      {"sev": strings.Repeat("x", 17)},
	} {
		if err := schema.Validate(bad); err == nil {
			t.Errorf("%s must refuse: %v", name, bad)
		}
	}
}

// Test review MEDIUM: parseSchema's remaining closed-vocabulary
// refusals had no coverage.
func TestSchemaShapeRefusals(t *testing.T) {
	for name, body := range map[string]string{
		"max_length on integer": `{"version":1,"fields":[{"name":"n","type":"integer","max_length":8}]}`,
		"max_length too large":  `{"version":1,"fields":[{"name":"s","type":"string","max_length":70000}]}`,
		"enum on non-string":    `{"version":1,"fields":[{"name":"n","type":"boolean","enum":["a"]}]}`,
		"duplicate field":       `{"version":1,"fields":[{"name":"s","type":"string","max_length":4},{"name":"s","type":"string","max_length":4}]}`,
		"bad field name":        `{"version":1,"fields":[{"name":"Bad Name","type":"string","max_length":4}]}`,
		"no fields":             `{"version":1,"fields":[]}`,
	} {
		if _, err := parseSchema([]byte(body)); err == nil {
			t.Errorf("%s must refuse", name)
		}
	}
}

// Test review MEDIUM: checkGrantShape's two fail-closed branches — the
// "L9 never mints a scope" refusal and the complete-narrowing rule —
// had no coverage.
func TestGrantShapeRefusals(t *testing.T) {
	if err := checkGrantShape([]byte(`{"version":1,"task_id":"t-1","total_max_calls":4,
	 "entries":[{"tool":"read_file","max_calls":4,"workspace":"/tmp/minted"}]}`), "t-1"); err == nil ||
		!strings.Contains(err.Error(), "never mints a scope") {
		t.Fatalf("a literal workspace in the effective grant must refuse: %v", err)
	}
	if err := checkGrantShape([]byte(`{"version":1,"task_id":"t-1","total_max_calls":99,
	 "entries":[{"tool":"read_file","max_calls":4,"workspace":"@workspace"}]}`), "t-1"); err == nil ||
		!strings.Contains(err.Error(), "narrowing must be complete") {
		t.Fatalf("an aggregate above the sum of per-tool caps must refuse: %v", err)
	}
	if err := checkGrantShape([]byte(`{"version":1,"task_id":"other","total_max_calls":4,
	 "entries":[{"tool":"read_file","max_calls":4,"workspace":"@workspace"}]}`), "t-1"); err == nil {
		t.Fatal("a grant bound to another task must refuse")
	}
}

// Test review MEDIUM: manifest pin-path traversal and bad SHA syntax —
// the cheapest attacks — were uncovered.
func TestManifestPinPathRefusals(t *testing.T) {
	b := newBundle(t)
	raw, err := os.ReadFile(filepath.Join(b.dir, "skill.json"))
	if err != nil {
		t.Fatal(err)
	}
	for name, mutate := range map[string]func(map[string]any){
		"absolute pin path": func(d map[string]any) {
			d["procedure"] = map[string]string{"path": "/etc/passwd", "sha256": strings.Repeat("a", 64)}
		},
		"traversing pin path": func(d map[string]any) {
			d["procedure"] = map[string]string{"path": "../../../etc/passwd", "sha256": strings.Repeat("a", 64)}
		},
		"short sha": func(d map[string]any) { d["procedure"] = map[string]string{"path": "procedure.md", "sha256": "abc"} },
		"non-hex sha": func(d map[string]any) {
			d["procedure"] = map[string]string{"path": "procedure.md", "sha256": strings.Repeat("z", 64)}
		},
	} {
		var doc map[string]any
		if err := json.Unmarshal(raw, &doc); err != nil {
			t.Fatal(err)
		}
		mutate(doc)
		body, _ := json.Marshal(doc)
		p := write(t, t.TempDir(), "skill.json", string(body))
		if _, err := LoadManifest(p); !errors.Is(err, ErrManifest) {
			t.Errorf("%s must refuse at load: %v", name, err)
		}
	}
}
