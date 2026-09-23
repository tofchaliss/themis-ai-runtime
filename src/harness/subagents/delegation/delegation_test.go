package delegation

// Register A — admission / authority (design.md §5.5): loader refusal,
// registry integrity, closed world, disjointness, exact resolution, no
// write capability. The positive twin comes first (the C17 lesson): the
// registered dependency-triage@1 template LOADS through the real
// registry before any refusal is trusted.

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"

	hctx "github.com/tofchaliss/themis/context"
	"github.com/tofchaliss/themis/instructions"
)

var repoRoot = filepath.Join("..", "..", "..", "..")

func sha(b []byte) string {
	s := sha256.Sum256(b)
	return hex.EncodeToString(s[:])
}

// THE POSITIVE TWIN: the PROPOSED registration resolves end to end —
// registry → entry → manifest → pins → L2 contract → cross-checks.
func TestRegisteredTemplateLoads(t *testing.T) {
	regPath := filepath.Join(repoRoot, "policies/delegation/registry.json")
	r, err := LoadRegistry(regPath)
	if err != nil {
		t.Fatalf("the governed registry must load: %v", err)
	}
	raw, _ := os.ReadFile(regPath)
	if r.Hash != sha(raw) || string(r.Raw) != string(raw) {
		t.Fatal("registry identity must be the hash of the exact bytes")
	}
	entry, tpl, err := r.Resolve("dependency-triage@1")
	if err != nil {
		t.Fatalf("dependency-triage@1 must resolve: %v", err)
	}
	if entry.Template != tpl.Hash || tpl.Name != "dependency-triage" || tpl.TemplateVersion != 1 {
		t.Fatalf("identity mismatch: entry=%s tpl=%s %s@%d", entry.Template, tpl.Hash, tpl.Name, tpl.TemplateVersion)
	}
	if sha(tpl.ContractRaw) != tpl.ContextContract.SHA256 || tpl.Contract == nil || tpl.Contract.Hash != tpl.ContextContract.SHA256 {
		t.Fatal("the retained contract bytes must be the pinned bytes, loaded by L2")
	}
	if tpl.Instruction == nil || sha(tpl.InstructionRaw) != tpl.Instruction.SHA256 {
		t.Fatal("the retained instruction bytes must be the pinned bytes")
	}
	// The instruction file is a legitimate skill-scope source under
	// L1's own activation rule — the only source a template may add.
	if _, err := instructions.ActivateSkillSource(tpl.InstructionRaw, tpl.Instruction.SHA256); err != nil {
		t.Fatalf("the pinned instruction must activate as a skill-scope source: %v", err)
	}
	if strings.Join(tpl.EISCarryScopes, ",") != "repository,skill" {
		t.Fatalf("carry filter: %v", tpl.EISCarryScopes)
	}
	if tpl.Brief.Slot != "brief" || tpl.Brief.MaxBytes != 4096 || tpl.MaxOutputBytes != 65536 {
		t.Fatalf("literals: %+v %d", tpl.Brief, tpl.MaxOutputBytes)
	}
}

// The carry-filter vocabulary is pinned to L1's own scope names, so a
// rename in L1 fails here rather than silently emptying the filter.
func TestCarryVocabularyIsL1Vocabulary(t *testing.T) {
	for _, s := range []instructions.Scope{instructions.ScopeRepository, instructions.ScopeDirectory, instructions.ScopeSkill} {
		if !carryScopes[s.String()] {
			t.Fatalf("L1 scope %q is not in the filter's domain", s.String())
		}
	}
	for _, s := range []instructions.Scope{instructions.ScopeHarnessSafety, instructions.ScopeHarnessSystem, instructions.ScopeThemisDomain, instructions.ScopeTask} {
		if _, bad := nonCarryScopes[s.String()]; !bad {
			t.Fatalf("L1 scope %q must be a named non-carry refusal", s.String())
		}
	}
}

// bundle writes a valid template directory and returns the manifest
// path plus a mutator that rewrites the manifest from a map — so each
// refusal case changes exactly one thing against a passing baseline.
type bundle struct {
	dir, manifest string
	t             *testing.T
}

func newBundle(t *testing.T) *bundle {
	t.Helper()
	dir := t.TempDir()
	contract := `{"version":1,"workflow":"x","slots":[
	 {"name":"brief","kind":"delegation-brief","requirement":"required","classes":["external-untrusted"]},
	 {"name":"evidence","kind":"evidence-payload","requirement":"optional","classes":["derived"]},
	 {"name":"held","kind":"other","requirement":"optional","classes":["external-untrusted"],"withhold":true},
	 {"name":"wide","kind":"w","requirement":"optional","classes":["external-untrusted","derived"]},
	 {"name":"gov","kind":"g","requirement":"optional","classes":["governed-record"]}],
	 "sensitivity_ceiling":"public"}`
	inst := "# rules\n- reason over evidence only\n"
	write(t, filepath.Join(dir, "contract.json"), contract)
	write(t, filepath.Join(dir, "instruction.md"), inst)
	b := &bundle{dir: dir, manifest: filepath.Join(dir, "template.json"), t: t}
	b.set(nil)
	return b
}

func (b *bundle) base() map[string]any {
	c, _ := os.ReadFile(filepath.Join(b.dir, "contract.json"))
	i, _ := os.ReadFile(filepath.Join(b.dir, "instruction.md"))
	return map[string]any{
		"version": 1, "name": "triage", "template_version": 1,
		"context_contract": map[string]any{"path": "contract.json", "sha256": sha(c)},
		"instruction":      map[string]any{"path": "instruction.md", "sha256": sha(i)},
		"eis_carry_scopes": []any{"repository", "skill"},
		"brief":            map[string]any{"slot": "brief", "max_bytes": 1024},
		"max_output_bytes": 4096,
	}
}

// set rewrites the manifest from the baseline after mutate; a nil
// mutate writes the baseline.
func (b *bundle) set(mutate func(m map[string]any)) {
	m := b.base()
	if mutate != nil {
		mutate(m)
	}
	raw, _ := json.Marshal(m)
	write(b.t, b.manifest, string(raw))
}

func (b *bundle) setRaw(raw string) { write(b.t, b.manifest, raw) }

func write(t *testing.T, path, body string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestTemplateBaselineLoads(t *testing.T) {
	b := newBundle(t)
	tpl, err := LoadTemplate(b.manifest)
	if err != nil {
		t.Fatalf("baseline must load: %v", err)
	}
	if tpl.Dir != b.dir || len(tpl.Contract.Slots) != 5 {
		t.Fatalf("loaded shape: dir=%q slots=%d", tpl.Dir, len(tpl.Contract.Slots))
	}
	// No instruction is a legitimate template too.
	b.set(func(m map[string]any) { delete(m, "instruction") })
	tpl, err = LoadTemplate(b.manifest)
	if err != nil || tpl.Instruction != nil || tpl.InstructionRaw != nil {
		t.Fatalf("instruction is optional: %v %+v", err, tpl.Instruction)
	}
	// An absent carry filter carries nothing beyond the roots.
	b.set(func(m map[string]any) { delete(m, "eis_carry_scopes") })
	if tpl, err = LoadTemplate(b.manifest); err != nil || len(tpl.EISCarryScopes) != 0 {
		t.Fatalf("absent filter = empty filter: %v %v", err, tpl.EISCarryScopes)
	}
}

// Each refusal names ITS gate; a suppressed clause lets exactly one
// mutation through.
func TestTemplateRefusals(t *testing.T) {
	cases := []struct {
		name   string
		mutate func(m map[string]any)
		want   string
	}{
		// Disjointness (D-L8-5/6), by name.
		{"pins a workflow", func(m map[string]any) {
			m["workflow"] = map[string]any{"path": "w.json", "sha256": strings.Repeat("0", 64)}
		}, `may not pin "workflow"`},
		{"pins a grant", func(m map[string]any) { m["grant"] = "x" }, `may not pin "grant"`},
		{"pins a grant template", func(m map[string]any) { m["grant_template"] = "x" }, `may not pin "grant_template"`},
		{"pins a spec", func(m map[string]any) { m["spec_template"] = "x" }, `may not pin "spec_template"`},
		{"pins an input schema", func(m map[string]any) { m["input_schema"] = "x" }, `may not pin "input_schema"`},
		{"pins another template", func(m map[string]any) { m["template"] = "other@1" }, `may not pin "template"`},
		{"names tools", func(m map[string]any) { m["tools"] = []any{"read_file"} }, `may not pin "tools"`},
		{"names a model", func(m map[string]any) { m["model"] = "qwen" }, `may not pin "model"`},
		// Closed schema.
		{"unknown field", func(m map[string]any) { m["evidence_required"] = true }, "unknown field"},
		{"requiredness restated", func(m map[string]any) {
			m["brief"] = map[string]any{"slot": "brief", "max_bytes": 1, "requirement": "required"}
		}, "unknown field"},
		{"version missing", func(m map[string]any) { m["version"] = 0 }, "version required"},
		{"no identity", func(m map[string]any) { m["template_version"] = 0 }, "exact identity"},
		{"bad name", func(m map[string]any) { m["name"] = "Triage" }, "exact identity"},
		// Carry filter (C-L8-4).
		{"carry names a root", func(m map[string]any) { m["eis_carry_scopes"] = []any{"themis-domain", "skill"} }, "mandatory root"},
		{"carry names harness-safety", func(m map[string]any) { m["eis_carry_scopes"] = []any{"harness-safety"} }, "mandatory root"},
		{"carry names task", func(m map[string]any) { m["eis_carry_scopes"] = []any{"task"} }, "never carry"},
		{"carry names unknown", func(m map[string]any) { m["eis_carry_scopes"] = []any{"workspace"} }, "unknown scope"},
		{"carry duplicate", func(m map[string]any) { m["eis_carry_scopes"] = []any{"skill", "skill"} }, "duplicate eis_carry_scopes"},
		// Brief and output bounds.
		{"brief slot unnamed", func(m map[string]any) { m["brief"] = map[string]any{"slot": "", "max_bytes": 1} }, "brief.slot must name"},
		{"brief max_bytes zero", func(m map[string]any) { m["brief"] = map[string]any{"slot": "brief", "max_bytes": 0} }, "brief.max_bytes"},
		{"brief max_bytes over L2 cap", func(m map[string]any) {
			m["brief"] = map[string]any{"slot": "brief", "max_bytes": hctx.MaxItemBytes + 1}
		}, "brief.max_bytes"},
		{"output bound zero", func(m map[string]any) { m["max_output_bytes"] = 0 }, "max_output_bytes"},
		{"output bound over L2 cap", func(m map[string]any) { m["max_output_bytes"] = hctx.MaxItemBytes + 1 }, "max_output_bytes"},
		// Pins.
		{"contract hash mismatch", func(m map[string]any) {
			m["context_contract"] = map[string]any{"path": "contract.json", "sha256": strings.Repeat("a", 64)}
		}, "context_contract bytes do not match"},
		{"instruction hash mismatch", func(m map[string]any) {
			m["instruction"] = map[string]any{"path": "instruction.md", "sha256": strings.Repeat("a", 64)}
		}, "instruction bytes do not match"},
		{"pin sha not a digest", func(m map[string]any) {
			m["instruction"] = map[string]any{"path": "instruction.md", "sha256": "latest"}
		}, "sha256 hex digest"},
		{"pin absolute path", func(m map[string]any) {
			m["instruction"] = map[string]any{"path": "/etc/passwd", "sha256": strings.Repeat("a", 64)}
		}, "instruction pin"},
		{"pin traversal", func(m map[string]any) {
			m["instruction"] = map[string]any{"path": "../x.md", "sha256": strings.Repeat("a", 64)}
		}, "instruction pin"},
		// Cross-checks against the pinned contract (C-L8-14 D).
		{"brief slot not in contract", func(m map[string]any) { m["brief"] = map[string]any{"slot": "nope", "max_bytes": 1} }, "is not a slot of the pinned contract"},
		{"brief slot withheld", func(m map[string]any) { m["brief"] = map[string]any{"slot": "held", "max_bytes": 1} }, "withheld"},
		{"brief slot governed class", func(m map[string]any) { m["brief"] = map[string]any{"slot": "gov", "max_bytes": 1} }, "exactly [external-untrusted]"},
		{"brief slot permits a second class", func(m map[string]any) { m["brief"] = map[string]any{"slot": "wide", "max_bytes": 1} }, "exactly [external-untrusted]"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			b := newBundle(t)
			b.set(c.mutate)
			_, err := LoadTemplate(b.manifest)
			if err == nil || !errors.Is(err, ErrTemplate) || !strings.Contains(err.Error(), c.want) {
				t.Fatalf("refused for the wrong reason (want %q): %v", c.want, err)
			}
		})
	}
	// Byte-level walls.
	b := newBundle(t)
	base, _ := os.ReadFile(b.manifest)
	for _, c := range []struct{ name, raw, want string }{
		{"case-variant key", strings.Replace(string(base), `"max_output_bytes"`, `"Max_Output_Bytes"`, 1), "not an exact lowercase key"},
		{"duplicate key", strings.Replace(string(base), `"max_output_bytes":4096`, `"max_output_bytes":4096,"max_output_bytes":1`, 1), "duplicate key"},
		{"trailing content", string(base) + " {}", "after top-level value"},
	} {
		t.Run(c.name, func(t *testing.T) {
			b.setRaw(c.raw)
			_, err := LoadTemplate(b.manifest)
			if err == nil || !strings.Contains(err.Error(), c.want) {
				t.Fatalf("want %q: %v", c.want, err)
			}
		})
	}
	// Empty instruction file, and a symlinked pin escaping the bundle.
	b = newBundle(t)
	write(t, filepath.Join(b.dir, "instruction.md"), "")
	b.set(nil)
	if _, err := LoadTemplate(b.manifest); err == nil || !strings.Contains(err.Error(), "empty file") {
		t.Fatalf("empty instruction: %v", err)
	}
	b = newBundle(t)
	outside := filepath.Join(t.TempDir(), "outside.md")
	write(t, outside, "# outside\n")
	os.Remove(filepath.Join(b.dir, "instruction.md"))
	if err := os.Symlink(outside, filepath.Join(b.dir, "instruction.md")); err != nil {
		t.Skip("no symlinks")
	}
	b.set(nil) // hashes the symlink target's bytes, so only confinement can refuse
	if _, err := LoadTemplate(b.manifest); err == nil || !strings.Contains(err.Error(), "instruction pin") {
		t.Fatalf("symlink escape must refuse at confinement: %v", err)
	}
}

// registryWorld: a private registry root with one bundle registered.
func registryWorld(t *testing.T) (string, *bundle) {
	t.Helper()
	root := t.TempDir()
	b := &bundle{dir: filepath.Join(root, "triage"), t: t}
	if err := os.Mkdir(b.dir, 0o755); err != nil {
		t.Fatal(err)
	}
	b.manifest = filepath.Join(b.dir, "template.json")
	nb := newBundle(t)
	for _, f := range []string{"contract.json", "instruction.md"} {
		raw, _ := os.ReadFile(filepath.Join(nb.dir, f))
		write(t, filepath.Join(b.dir, f), string(raw))
	}
	b.set(nil)
	writeRegistry(t, root, b, "active")
	return root, b
}

func writeRegistry(t *testing.T, root string, b *bundle, state string) {
	t.Helper()
	raw, _ := os.ReadFile(b.manifest)
	reg := map[string]any{"version": 1, "entries": []any{map[string]any{
		"name": "triage", "version": 1, "template_sha256": sha(raw),
		"manifest_path": "triage/template.json", "state": state, "steward": "s"}}}
	rb, _ := json.Marshal(reg)
	write(t, filepath.Join(root, "registry.json"), string(rb))
}

func TestRegistryRefusals(t *testing.T) {
	root, _ := registryWorld(t)
	reg := filepath.Join(root, "registry.json")
	base, _ := os.ReadFile(reg)
	entry := func(fields string) string {
		return `{"version":1,"entries":[{"name":"triage","version":1,"template_sha256":"` + strings.Repeat("a", 64) + `","manifest_path":"triage/template.json","state":"active"` + fields + `}]}`
	}
	cases := []struct{ name, raw, want string }{
		{"bad identity", strings.Replace(string(base), `"name":"triage"`, `"name":"Triage"`, 1), "bad entry identity"},
		{"version zero", strings.Replace(entry(""), `"version":1,"template_sha256"`, `"version":0,"template_sha256"`, 1), "bad entry identity"},
		{"bad sha", strings.Replace(entry(""), strings.Repeat("a", 64), "latest", 1), "sha256 hex digest"},
		{"absolute manifest path", strings.Replace(entry(""), `"triage/template.json"`, `"/etc/passwd"`, 1), "relative within the registry root"},
		{"traversal manifest path", strings.Replace(entry(""), `"triage/template.json"`, `"../x/template.json"`, 1), "relative within the registry root"},
		{"unknown state", strings.Replace(entry(""), `"state":"active"`, `"state":"proposed"`, 1), "unknown state"},
		{"unknown field", entry(`,"latest":true`), "unknown field"},
		{"duplicate registration", strings.Replace(entry(""), `}]}`, `},{"name":"triage","version":1,"template_sha256":"`+strings.Repeat("b", 64)+`","manifest_path":"triage/template.json","state":"active"}]}`, 1), "bindings are immutable"},
		{"registry version missing", strings.Replace(entry(""), `{"version":1,"entries"`, `{"version":0,"entries"`, 1), "version required"},
		{"trailing content", string(base) + "{}", "trailing content"},
		{"case-variant key", strings.Replace(string(base), `"state"`, `"State"`, 1), "not an exact lowercase key"},
		{"duplicate key", strings.Replace(string(base), `"state":"active"`, `"state":"active","state":"withdrawn"`, 1), "duplicate key"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			write(t, reg, c.raw)
			_, err := LoadRegistry(reg)
			if err == nil || !errors.Is(err, ErrRegistry) || !strings.Contains(err.Error(), c.want) {
				t.Fatalf("want %q: %v", c.want, err)
			}
		})
	}
}

func TestParseRefExactOnly(t *testing.T) {
	for _, bad := range []string{"triage", "triage@", "@1", "triage@latest", "triage@1.2", "triage@0", "triage@01", "triage@>=1", "Triage@1", "triage@1 "} {
		if _, _, err := ParseRef(bad); err == nil || !errors.Is(err, ErrResolve) {
			t.Fatalf("%q must refuse: %v", bad, err)
		}
	}
	if n, v, err := ParseRef("dependency-triage@12"); err != nil || n != "dependency-triage" || v != 12 {
		t.Fatalf("exact ref: %s %d %v", n, v, err)
	}
}

func TestResolveRefusals(t *testing.T) {
	root, b := registryWorld(t)
	reg := filepath.Join(root, "registry.json")
	r, err := LoadRegistry(reg)
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := r.Resolve("triage@1"); err != nil {
		t.Fatalf("positive: %v", err)
	}
	if _, _, err := r.Resolve("triage@2"); err == nil || !strings.Contains(err.Error(), "is not registered") {
		t.Fatalf("unregistered: %v", err)
	}
	if _, _, err := r.Resolve("other@1"); err == nil || !strings.Contains(err.Error(), "is not registered") {
		t.Fatalf("unregistered name: %v", err)
	}
	// Withdrawn: the binding stays; new delegation refuses.
	writeRegistry(t, root, b, "withdrawn")
	rw, _ := LoadRegistry(reg)
	if _, _, err := rw.Resolve("triage@1"); !errors.Is(err, ErrWithdrawn) || !errors.Is(err, ErrResolve) {
		t.Fatalf("withdrawn: %v", err)
	}
	// Bytes drift under a fixed pin: unavailable, not "changed".
	writeRegistry(t, root, b, "active")
	r, _ = LoadRegistry(reg)
	b.set(func(m map[string]any) { m["max_output_bytes"] = 8192 })
	if _, _, err := r.Resolve("triage@1"); !errors.Is(err, ErrHashMismatch) || !errors.Is(err, ErrResolve) {
		t.Fatalf("drift: %v", err)
	}
	// Self-declaration disagreement: registry pins bytes that call
	// themselves something else.
	b.set(func(m map[string]any) { m["name"] = "other" })
	writeRegistry(t, root, b, "active")
	r, _ = LoadRegistry(reg)
	if _, _, err := r.Resolve("triage@1"); err == nil || !strings.Contains(err.Error(), "two-way identity") {
		t.Fatalf("self-declaration: %v", err)
	}
	b.set(nil)
	// Manifest path resolving through a symlink out of the root.
	outsideDir := t.TempDir()
	for _, f := range []string{"template.json", "contract.json", "instruction.md"} {
		raw, _ := os.ReadFile(filepath.Join(b.dir, f))
		write(t, filepath.Join(outsideDir, f), string(raw))
	}
	os.RemoveAll(b.dir)
	if err := os.Symlink(outsideDir, b.dir); err != nil {
		t.Skip("no symlinks")
	}
	writeRegistry(t, root, &bundle{dir: b.dir, manifest: filepath.Join(outsideDir, "template.json"), t: t}, "active")
	r, _ = LoadRegistry(reg)
	if _, _, err := r.Resolve("triage@1"); err == nil || !strings.Contains(err.Error(), "manifest") {
		t.Fatalf("symlinked manifest dir must refuse at confinement: %v", err)
	}
}

func TestAppendOnly(t *testing.T) {
	root, b := registryWorld(t)
	reg := filepath.Join(root, "registry.json")
	prior, _ := LoadRegistry(reg)
	mk := func(entries string) *Registry {
		write(t, reg, `{"version":1,"entries":[`+entries+`]}`)
		r, err := LoadRegistry(reg)
		if err != nil {
			t.Fatal(err)
		}
		return r
	}
	raw, _ := os.ReadFile(b.manifest)
	e1 := `{"name":"triage","version":1,"template_sha256":"` + sha(raw) + `","manifest_path":"triage/template.json","state":"%s"}`
	if err := mk(strings.Replace(e1, "%s", "active", 1) + `,{"name":"triage","version":2,"template_sha256":"` + strings.Repeat("c", 64) + `","manifest_path":"triage/template.json","state":"active"}`).CheckAppendOnly(prior); err != nil {
		t.Fatalf("append is fine: %v", err)
	}
	if err := mk(strings.Replace(e1, "%s", "withdrawn", 1)).CheckAppendOnly(prior); err != nil {
		t.Fatalf("withdrawal advances: %v", err)
	}
	if err := mk(``).CheckAppendOnly(prior); err == nil || !strings.Contains(err.Error(), "disappeared") {
		t.Fatalf("deletion: %v", err)
	}
	if err := mk(strings.Replace(strings.Replace(e1, "%s", "active", 1), sha(raw), strings.Repeat("d", 64), 1)).CheckAppendOnly(prior); err == nil || !strings.Contains(err.Error(), "rebound") {
		t.Fatalf("rebind: %v", err)
	}
	withdrawn := mk(strings.Replace(e1, "%s", "withdrawn", 1))
	if err := mk(strings.Replace(e1, "%s", "active", 1)).CheckAppendOnly(withdrawn); err == nil || !strings.Contains(err.Error(), "un-withdrawn") {
		t.Fatalf("un-withdrawal: %v", err)
	}
}

// --- the write wall -------------------------------------------------

var writers = map[string]bool{
	"WriteFile": true, "Create": true, "CreateTemp": true, "OpenFile": true,
	"Rename": true, "Remove": true, "RemoveAll": true, "Truncate": true,
	"Mkdir": true, "MkdirAll": true, "MkdirTemp": true,
	"Symlink": true, "Link": true, "Chmod": true, "Chown": true, "Lchown": true,
}

var forbiddenImports = map[string]bool{`"io/ioutil"`: true, `"bufio"`: true}

// scanWriters is the audit itself, shared by the package scan and the
// doctored-source self-check so the two cannot drift apart.
func scanWriters(fset *token.FileSet, name string, file *ast.File, report func(string)) {
	osNames := map[string]bool{}
	for _, imp := range file.Imports {
		if imp.Path == nil {
			continue
		}
		if forbiddenImports[imp.Path.Value] {
			report(name + " imports " + imp.Path.Value + " — indirect write path refused")
		}
		if imp.Path.Value != `"os"` {
			continue
		}
		if imp.Name != nil {
			osNames[imp.Name.Name] = true
		} else {
			osNames["os"] = true
		}
	}
	if len(osNames) == 0 {
		return
	}
	ast.Inspect(file, func(n ast.Node) bool {
		sel, ok := n.(*ast.SelectorExpr)
		if !ok {
			return true
		}
		if ident, ok := sel.X.(*ast.Ident); ok && osNames[ident.Name] && writers[sel.Sel.Name] {
			report(name + " references " + ident.Name + "." + sel.Sel.Name + " at " + fset.Position(sel.Pos()).String() + " — the registry machinery must not write")
		}
		return true
	})
}

func TestNoRegistryWriteCapability(t *testing.T) {
	fset := token.NewFileSet()
	pkgs, err := parser.ParseDir(fset, ".", func(fi os.FileInfo) bool {
		return !strings.HasSuffix(fi.Name(), "_test.go")
	}, 0)
	if err != nil {
		t.Fatal(err)
	}
	scanned := 0
	for _, pkg := range pkgs {
		for name, file := range pkg.Files {
			scanned++
			scanWriters(fset, name, file, func(msg string) { t.Error(msg) })
		}
	}
	if scanned == 0 {
		t.Fatal("audit scanned no files — the wall is not being checked")
	}
}

func TestWallCatchesFunctionValueBinding(t *testing.T) {
	src := "package delegation\n\nimport o \"os\"\n\nvar scribe = o.\n\tWriteFile\n"
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "doctored.go", src, 0)
	if err != nil {
		t.Fatal(err)
	}
	caught := false
	scanWriters(fset, "doctored.go", file, func(string) { caught = true })
	if !caught {
		t.Fatal("the AST walk missed an aliased, newline-split function-value binding — the wall does not hold")
	}
}
