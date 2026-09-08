package skills

// Register A (structural) + Register B (adversarial) for L9-M1/M2:
// "can the invalid object even exist?" and "do the boundaries survive
// hostile input?". Every test pins a locked decision by name.

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const repoRoot = "../../.."

func sha(b []byte) string {
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}

func write(t *testing.T, dir, name, body string) string {
	t.Helper()
	p := filepath.Join(dir, name)
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	return p
}

// --- fixture: a complete, valid skill bundle + catalog -------------

type bundle struct {
	dir         string // skill bundle dir
	catalogDir  string // catalog root (contains catalog.json + bundle/)
	catalogPath string
	composition string
}

const (
	fxWorkflow = `{
 "version":1,"name":"analyze-verify","initial":"ANALYZE",
 "declared_events":["turn-no-action","turn-provider-error","turns-exhausted","tool-error","signal:phase-completion-requested"],
 "phases":[
  {"name":"ANALYZE","capabilities":["read_file","declare_done"],"max_model_turns":4,"edges":[
    {"on":"signal:phase-completion-requested","to":"VERIFY"},
    {"on":"turn-no-action","to":"@stay","counter":2,"exhausted_to":"@fail"},
    {"on":"turn-provider-error","to":"@fail"},
    {"on":"tool-error","to":"@stay","counter":2,"exhausted_to":"@fail"},
    {"on":"turns-exhausted","to":"@fail"}]},
  {"name":"VERIFY","capabilities":["declare_done"],"max_model_turns":3,"edges":[
    {"on":"signal:phase-completion-requested","to":"@complete"},
    {"on":"turn-no-action","to":"@stay","counter":2,"exhausted_to":"@fail"},
    {"on":"turn-provider-error","to":"@fail"},
    {"on":"tool-error","to":"@fail"},
    {"on":"turns-exhausted","to":"@fail"}]}
 ]}`
	fxCeiling  = `{"version":1,"allowed_tools":["read_file","declare_done"],"max_total_calls":20,"max_walk_length":100,"max_turns_per_phase":10}`
	fxContract = `{"version":1,"workflow":"analyze-verify","slots":[{"name":"task-payload","kind":"task-brief","requirement":"required","classes":["external-untrusted"]}],"sensitivity_ceiling":"public"}`
	fxGrant    = `{"version":1,"task_id":"@task_id","total_max_calls":20,"entries":[{"tool":"read_file","max_calls":8,"workspace":"@workspace"},{"tool":"declare_done","max_calls":4}]}`
	fxSpec     = `{"version":1,"task_id":"@task_id","repo":"@repo","pinned_sha":"@pinned_sha","limits":[{"dimension":"wall_deadline_s","value":90}]}`
	fxSchema   = `{"version":1,"fields":[{"name":"cve-id","type":"string","required":true,"max_length":64},{"name":"depth","type":"integer"}]}`
	fxProc     = "# Procedure\n\nRead the manifest, assess, then declare done.\n"
)

// newBundle writes a valid skill bundle and a catalog registering it.
func newBundle(t *testing.T) *bundle {
	t.Helper()
	root := t.TempDir()
	bdir := filepath.Join(root, "bundle")

	write(t, bdir, "workflow.json", fxWorkflow)
	write(t, bdir, "ceiling.json", fxCeiling)
	write(t, bdir, "contract.json", fxContract)
	write(t, bdir, "grant.json", fxGrant)
	write(t, bdir, "spec.json", fxSpec)
	write(t, bdir, "schema.json", fxSchema)
	write(t, bdir, "procedure.md", fxProc)

	man := fmt.Sprintf(`{
 "version":1,"name":"investigate-cve","skill_version":1,
 "workflow":{"path":"workflow.json","sha256":%q},
 "workflow_ceiling":{"path":"ceiling.json","sha256":%q},
 "context_contract":{"path":"contract.json","sha256":%q},
 "grant_template":{"path":"grant.json","sha256":%q},
 "spec_template":{"path":"spec.json","sha256":%q},
 "input_schema":{"path":"schema.json","sha256":%q},
 "procedure":{"path":"procedure.md","sha256":%q}}`,
		sha([]byte(fxWorkflow)), sha([]byte(fxCeiling)), sha([]byte(fxContract)),
		sha([]byte(fxGrant)), sha([]byte(fxSpec)), sha([]byte(fxSchema)), sha([]byte(fxProc)))
	write(t, bdir, "skill.json", man)

	cat := fmt.Sprintf(`{"version":1,"entries":[
	 {"name":"investigate-cve","version":1,"composition_sha256":%q,"manifest_path":"bundle/skill.json","state":"active"}]}`,
		sha([]byte(man)))
	catPath := write(t, root, "catalog.json", cat)
	return &bundle{dir: bdir, catalogDir: root, catalogPath: catPath, composition: sha([]byte(man))}
}

// deployment returns Class-4 inputs with roots disjoint from the catalog.
func (b *bundle) deployment(t *testing.T) Deployment {
	t.Helper()
	other := t.TempDir()
	return Deployment{
		Model: "scripted", TurnTimeoutSec: 180,
		RegistryPath:    mustAbs(t, filepath.Join(repoRoot, "policies/tools/registry-v3.json")),
		ExecCeilingPath: write(t, other, "eceiling.json", `{"version":1,"mirror_root":"`+other+`","max_wall_deadline_sec":120,"max_file_bytes":1048576,"max_total_bytes":10485760,"max_file_count":500,"max_mem_bytes":1073741824,"max_cpu_time_sec":600,"max_proc_count":64}`),
		StateRoot:       filepath.Join(other, "state"),
		ArtifactDir:     filepath.Join(other, "artifacts"),
		WorkspaceRoot:   filepath.Join(other, "workspaces"),
	}
}

func mustAbs(t *testing.T, p string) string {
	t.Helper()
	a, err := filepath.Abs(p)
	if err != nil {
		t.Fatal(err)
	}
	return a
}

func (b *bundle) request(t *testing.T) Request {
	t.Helper()
	d := b.deployment(t)
	for _, root := range []string{d.StateRoot, d.ArtifactDir, d.WorkspaceRoot} {
		if err := os.MkdirAll(root, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	return Request{
		TaskID: "t-1", Repo: "demo", PinnedSHA: strings.Repeat("a", 40),
		Inputs:     map[string]any{"cve-id": "CVE-2026-12345"},
		Deployment: d, OutDir: t.TempDir(),
	}
}

// --- Register A: structural -----------------------------------------

// D-L9-1: a manifest is a COMPLETE composition — every pin mandatory.
func TestManifestPinsMandatory(t *testing.T) {
	b := newBundle(t)
	raw, err := os.ReadFile(filepath.Join(b.dir, "skill.json"))
	if err != nil {
		t.Fatal(err)
	}
	for _, pin := range []string{"workflow", "workflow_ceiling", "context_contract",
		"grant_template", "spec_template", "input_schema", "procedure"} {
		var doc map[string]any
		if err := json.Unmarshal(raw, &doc); err != nil {
			t.Fatal(err)
		}
		delete(doc, pin)
		body, _ := json.Marshal(doc)
		p := write(t, t.TempDir(), "skill.json", string(body))
		_, err := LoadManifest(p)
		if !errors.Is(err, ErrManifest) || !strings.Contains(err.Error(), pin) {
			t.Fatalf("missing %s must refuse BY NAME: %v", pin, err)
		}
	}
}

// D-L9-1/D-L9-4: unknown manifest fields refuse — the schema is closed,
// so a skill cannot smuggle configuration past review.
func TestManifestClosedSchema(t *testing.T) {
	b := newBundle(t)
	raw, _ := os.ReadFile(filepath.Join(b.dir, "skill.json"))
	var doc map[string]any
	_ = json.Unmarshal(raw, &doc)
	doc["requires_model"] = "qwen2.5:7b" // D-L9-7: skills are model-agnostic
	body, _ := json.Marshal(doc)
	p := write(t, t.TempDir(), "skill.json", string(body))
	if _, err := LoadManifest(p); !errors.Is(err, ErrManifest) {
		t.Fatalf("unknown manifest field must refuse: %v", err)
	}
	// Trailing content refuses too (fail-closed loader convention).
	p2 := write(t, t.TempDir(), "skill.json", string(raw)+"{}")
	if _, err := LoadManifest(p2); !errors.Is(err, ErrManifest) {
		t.Fatalf("trailing content must refuse: %v", err)
	}
}

// D-L9-12: recursion is UNREPRESENTABLE — the manifest struct has no
// skill-reference field of any name, so nesting cannot be expressed.
func TestNoSkillReferenceField(t *testing.T) {
	b := newBundle(t)
	raw, _ := os.ReadFile(filepath.Join(b.dir, "skill.json"))
	for _, field := range []string{"skill_ref", "skills", "includes", "extends", "parent"} {
		var doc map[string]any
		_ = json.Unmarshal(raw, &doc)
		doc[field] = "investigate-cve@1"
		body, _ := json.Marshal(doc)
		p := write(t, t.TempDir(), "skill.json", string(body))
		if _, err := LoadManifest(p); !errors.Is(err, ErrManifest) {
			t.Fatalf("skill-reference field %q must be unrepresentable: %v", field, err)
		}
	}
}

// D-L9-4: the input-schema vocabulary is closed — rich features are
// unrepresentable, not merely unhandled.
func TestSchemaVocabularyClosed(t *testing.T) {
	for _, bad := range []string{
		`{"version":1,"fields":[{"name":"x","type":"string","max_length":8,"pattern":"^a+$"}]}`,
		`{"version":1,"fields":[{"name":"x","type":"string","max_length":8,"$ref":"#/defs/x"}]}`,
		`{"version":1,"fields":[{"name":"x","type":"string","max_length":8,"default":"a"}]}`,
		`{"version":1,"fields":[{"name":"x","type":"string","max_length":8,"if":{}}]}`,
		`{"version":1,"fields":[{"name":"x","type":"object"}]}`,
		`{"version":1,"fields":[{"name":"x","type":"array"}]}`,
	} {
		if _, err := parseSchema([]byte(bad)); !errors.Is(err, ErrInputs) {
			t.Fatalf("rich schema feature must refuse: %s -> %v", bad, err)
		}
	}
	// Nothing is defaulted: a string field must declare its bound.
	if _, err := parseSchema([]byte(`{"version":1,"fields":[{"name":"x","type":"string"}]}`)); !errors.Is(err, ErrInputs) {
		t.Fatalf("string field without max_length must refuse: %v", err)
	}
}

// D-L9-10: only exact name@version resolves — floating references are
// refused ("latest" is a lookup policy, not a version).
func TestNoFloatingReferences(t *testing.T) {
	for _, ref := range []string{"investigate-cve", "investigate-cve@latest", "investigate-cve@>=1",
		"investigate-cve@", "@1", "investigate-cve@1.0", "investigate-cve@-1", "investigate-cve@0"} {
		if _, _, err := ParseRef(ref); err == nil {
			t.Fatalf("floating/invalid reference %q must refuse", ref)
		}
	}
	name, v, err := ParseRef("investigate-cve@3")
	if err != nil || name != "investigate-cve" || v != 3 {
		t.Fatalf("exact reference must parse: %v %v %v", name, v, err)
	}
}

// D-L9-10: bindings are immutable; a catalog restating name@version
// with a different hash is a corrupted/rebound catalog.
func TestCatalogRebindRefused(t *testing.T) {
	dup := fmt.Sprintf(`{"version":1,"entries":[
	 {"name":"a","version":1,"composition_sha256":%q,"manifest_path":"x/skill.json","state":"active"},
	 {"name":"a","version":1,"composition_sha256":%q,"manifest_path":"y/skill.json","state":"active"}]}`,
		strings.Repeat("a", 64), strings.Repeat("b", 64))
	p := write(t, t.TempDir(), "catalog.json", dup)
	if _, err := LoadCatalog(p); !errors.Is(err, ErrCatalog) || !strings.Contains(err.Error(), "immutable") {
		t.Fatalf("duplicate binding must refuse as immutable: %v", err)
	}
}

// D-L9-10: append-only — deletion, rebinding, and un-withdrawal are
// detected against a prior observed state.
func TestCatalogAppendOnly(t *testing.T) {
	mk := func(entries string) *Catalog {
		p := write(t, t.TempDir(), "catalog.json", `{"version":1,"entries":[`+entries+`]}`)
		c, err := LoadCatalog(p)
		if err != nil {
			t.Fatal(err)
		}
		return c
	}
	h1, h2 := strings.Repeat("a", 64), strings.Repeat("b", 64)
	prior := mk(fmt.Sprintf(`{"name":"a","version":1,"composition_sha256":%q,"manifest_path":"a/s.json","state":"active"}`, h1))

	// Adding a new version is legal.
	grown := mk(fmt.Sprintf(`{"name":"a","version":1,"composition_sha256":%q,"manifest_path":"a/s.json","state":"active"},
	 {"name":"a","version":2,"composition_sha256":%q,"manifest_path":"b/s.json","state":"active"}`, h1, h2))
	if err := grown.CheckAppendOnly(prior); err != nil {
		t.Fatalf("adding a version must be legal: %v", err)
	}
	// Withdrawal is legal (active -> withdrawn).
	withdrawn := mk(fmt.Sprintf(`{"name":"a","version":1,"composition_sha256":%q,"manifest_path":"a/s.json","state":"withdrawn"}`, h1))
	if err := withdrawn.CheckAppendOnly(prior); err != nil {
		t.Fatalf("withdrawal must be legal: %v", err)
	}
	// Deletion is refused.
	empty := mk(fmt.Sprintf(`{"name":"z","version":9,"composition_sha256":%q,"manifest_path":"z/s.json","state":"active"}`, h2))
	if err := empty.CheckAppendOnly(prior); !errors.Is(err, ErrCatalog) || !strings.Contains(err.Error(), "disappeared") {
		t.Fatalf("deletion must refuse: %v", err)
	}
	// Rebinding is refused.
	rebound := mk(fmt.Sprintf(`{"name":"a","version":1,"composition_sha256":%q,"manifest_path":"a/s.json","state":"active"}`, h2))
	if err := rebound.CheckAppendOnly(prior); !errors.Is(err, ErrCatalog) || !strings.Contains(err.Error(), "rebound") {
		t.Fatalf("rebinding must refuse: %v", err)
	}
	// Un-withdrawal is refused.
	if err := prior.CheckAppendOnly(withdrawn); !errors.Is(err, ErrCatalog) || !strings.Contains(err.Error(), "un-withdrawn") {
		t.Fatalf("un-withdrawal must refuse: %v", err)
	}
}

// D-L9-3 / D-L9-16: the skills package can never write catalog storage
// — machinery cannot self-register or self-promote. An earlier version
// of this test grepped for five function NAMES and was defeated by a
// function called PromoteSkill (test review CRITICAL). This audits the
// CAPABILITY instead: the package legitimately writes only the three
// effective artifacts inside the caller's OutDir, so every filesystem
// write in non-test sources must occur in Instantiate and nowhere else.
func TestNoCatalogWriteCapability(t *testing.T) {
	fset := token.NewFileSet()
	pkgs, err := parser.ParseDir(fset, ".", func(fi os.FileInfo) bool {
		return !strings.HasSuffix(fi.Name(), "_test.go")
	}, 0)
	if err != nil {
		t.Fatal(err)
	}
	writers := map[string]bool{
		"WriteFile": true, "Create": true, "OpenFile": true,
		"Rename": true, "Remove": true, "RemoveAll": true, "Truncate": true,
	}
	scanned := 0
	for _, pkg := range pkgs {
		for name, file := range pkg.Files {
			scanned++
			var fn string
			ast.Inspect(file, func(n ast.Node) bool {
				switch node := n.(type) {
				case *ast.FuncDecl:
					fn = node.Name.Name
				case *ast.SelectorExpr:
					pkgIdent, ok := node.X.(*ast.Ident)
					if !ok || pkgIdent.Name != "os" {
						return true
					}
					if !writers[node.Sel.Name] {
						return true
					}
					// Instantiate emits the effective grant/spec/envelope
					// into the caller's OutDir; that is its whole job.
					// Any OTHER function gaining a write is the capability
					// this test exists to refuse.
					if fn != "Instantiate" {
						t.Errorf("%s: %s writes the filesystem via os.%s — only Instantiate may write, and never catalog storage (D-L9-3/D-L9-16)",
							filepath.Base(name), fn, node.Sel.Name)
					}
				}
				return true
			})
		}
	}
	if scanned == 0 {
		t.Fatal("scanned no sources — the audit would be vacuously true")
	}
}

// D-L9-8 wall 1: no registry anywhere carries a catalog/skill verb —
// the model has no capability that could reach registration. Decoded
// scan, following the L6 no-state-verb precedent.
func TestNoCatalogVerbInRegistries(t *testing.T) {
	dir := filepath.Join(repoRoot, "policies/tools")
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	forbidden := []string{"register_skill", "withdraw_skill", "publish_skill", "write_catalog", "catalog"}
	scanned := 0
	for _, e := range entries {
		if !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		raw, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			t.Fatal(err)
		}
		var doc struct {
			Tools []struct {
				Name string `json:"name"`
			} `json:"tools"`
		}
		if err := json.Unmarshal(raw, &doc); err != nil {
			t.Fatalf("%s: %v", e.Name(), err)
		}
		scanned++
		for _, tool := range doc.Tools {
			for _, bad := range forbidden {
				if strings.Contains(strings.ToLower(tool.Name), bad) {
					t.Fatalf("%s: registry exposes %q — no catalog capability may exist (D-L9-8 wall 1)", e.Name(), tool.Name)
				}
			}
		}
	}
	if scanned == 0 {
		t.Fatal("scanned no registries — the wall would be vacuously true")
	}
}
