package verification

// Register A (structural) proofs for L10-M1: invalid contract and
// registry objects cannot exist; the registry has no write capability;
// eligibility fails closed. Each refusal is pinned by a doctored
// artifact (the L7 TestLoaderRefusals pattern).

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const validContractBody = `{
  "version": 1,
  "name": "go-build-clean",
  "contract_version": 1,
  "verifier": {
    "capability": "run-go-build",
    "registry_sha256": "%s"
  },
  "evidence": [
    {"name": "build_target", "kind": "artifact", "required": true, "task_bound": true}
  ],
  "config": {"goflags": "-trimpath"},
  "result_mapping": {
    "build_ok": "PASS",
    "build_failed": "FAIL",
    "build_skipped": "INCONCLUSIVE"
  },
  "provenance": ["execution_record", "raw_output", "canonical_result"]
}`

var fakeRegistrySHA = strings.Repeat("ab", 32)

func validContract() string {
	return fmt.Sprintf(validContractBody, fakeRegistrySHA)
}

func write(t *testing.T, dir, name, content string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	return path
}

// registryFor writes a registry binding one contract file, computing
// the real hash so the happy path is honest.
func registryFor(t *testing.T, dir, contractJSON string) string {
	t.Helper()
	cpath := write(t, dir, "go-build-clean/contract.json", contractJSON)
	raw, err := os.ReadFile(cpath)
	if err != nil {
		t.Fatal(err)
	}
	reg := fmt.Sprintf(`{
  "version": 1,
  "entries": [
    {"name": "go-build-clean", "version": 1, "contract_sha256": %q,
     "contract_path": "go-build-clean/contract.json", "state": "active",
     "steward": "security-engineering"}
  ]
}`, hashBytes(raw))
	return write(t, dir, "contracts.json", reg)
}

type allowAll struct{}

func (allowAll) VerifierEligible(cap, sha string) (bool, error) { return true, nil }

type denyAll struct{}

func (denyAll) VerifierEligible(cap, sha string) (bool, error) { return false, nil }

type errChecker struct{}

func (errChecker) VerifierEligible(cap, sha string) (bool, error) {
	return true, fmt.Errorf("registry unreadable")
}

func TestContractHappyPath(t *testing.T) {
	dir := t.TempDir()
	regPath := registryFor(t, dir, validContract())

	r, err := LoadRegistry(regPath)
	if err != nil {
		t.Fatal(err)
	}
	entry, c, err := r.Resolve("go-build-clean@1", allowAll{})
	if err != nil {
		t.Fatal(err)
	}
	if entry.State != StateActive || c.Name != "go-build-clean" || c.Contract != 1 {
		t.Errorf("resolved identity wrong: %+v / %+v", entry, c)
	}
	if c.SHA256 != entry.Contract {
		t.Error("contract hash must equal registered binding")
	}
}

func TestContractClosedSchema(t *testing.T) {
	dir := t.TempDir()

	cases := map[string]string{
		"unknown field": strings.Replace(validContract(),
			`"version": 1,`, `"version": 1, "extra": true,`, 1),
		"trailing content": validContract() + `{"more": 1}`,
		"bad version": strings.Replace(validContract(),
			`"version": 1,`, `"version": 2,`, 1),
		"machinery outcome in mapping": strings.Replace(validContract(),
			`"build_failed": "FAIL"`, `"build_failed": "INVALID"`, 1),
		"unavailable in mapping": strings.Replace(validContract(),
			`"build_failed": "FAIL"`, `"build_failed": "UNAVAILABLE"`, 1),
		"governance vocabulary as outcome": strings.Replace(validContract(),
			`"build_ok": "PASS"`, `"build_ok": "NOT_AFFECTED"`, 1),
		"unknown evidence kind": strings.Replace(validContract(),
			`"kind": "artifact"`, `"kind": "opinion"`, 1),
		"task binding relaxed": strings.Replace(validContract(),
			`"task_bound": true`, `"task_bound": false`, 1),
		"provenance relaxed": strings.Replace(validContract(),
			`"provenance": ["execution_record", "raw_output", "canonical_result"]`,
			`"provenance": ["raw_output"]`, 1),
		"empty result mapping": strings.Replace(validContract(),
			`"result_mapping": {
    "build_ok": "PASS",
    "build_failed": "FAIL",
    "build_skipped": "INCONCLUSIVE"
  }`, `"result_mapping": {}`, 1),
		"bad registry sha": strings.Replace(validContract(),
			fakeRegistrySHA, "nothex", 1),
	}

	for name, body := range cases {
		t.Run(name, func(t *testing.T) {
			path := write(t, dir, "c-"+strings.ReplaceAll(name, " ", "-")+".json", body)
			if _, err := LoadContract(path); err == nil {
				t.Errorf("doctored contract loaded: %s", name)
			}
		})
	}
}

func TestRegistryRefusals(t *testing.T) {
	dir := t.TempDir()
	regPath := registryFor(t, dir, validContract())
	r, err := LoadRegistry(regPath)
	if err != nil {
		t.Fatal(err)
	}

	t.Run("floating references refused", func(t *testing.T) {
		for _, ref := range []string{"go-build-clean", "go-build-clean@latest", "go-build-clean@", "@1", "go-build-clean@0", "go-build-clean@1.2"} {
			if _, _, err := r.Resolve(ref, allowAll{}); err == nil {
				t.Errorf("floating reference resolved: %q", ref)
			}
		}
	})

	t.Run("unregistered contract is data", func(t *testing.T) {
		if _, _, err := r.Resolve("other-contract@1", allowAll{}); err == nil {
			t.Error("unregistered contract resolved")
		}
	})

	t.Run("duplicate registration refused at load", func(t *testing.T) {
		dup := strings.Replace(readFile(t, regPath), `"entries": [`, `"entries": [
    {"name": "go-build-clean", "version": 1, "contract_sha256": "`+strings.Repeat("cd", 32)+`",
     "contract_path": "go-build-clean/contract.json", "state": "active"},`, 1)
		p := write(t, dir, "dup.json", dup)
		if _, err := LoadRegistry(p); err == nil {
			t.Error("duplicate binding loaded")
		}
	})

	t.Run("tampered contract bytes refused", func(t *testing.T) {
		cpath := filepath.Join(dir, "go-build-clean/contract.json")
		tampered := strings.Replace(validContract(), `"-trimpath"`, `"-tampered"`, 1)
		if err := os.WriteFile(cpath, []byte(tampered), 0644); err != nil {
			t.Fatal(err)
		}
		if _, _, err := r.Resolve("go-build-clean@1", allowAll{}); err == nil {
			t.Error("tampered contract bytes resolved against registered hash")
		}
		// restore for later subtests
		if err := os.WriteFile(cpath, []byte(validContract()), 0644); err != nil {
			t.Fatal(err)
		}
	})

	t.Run("two-way identity refused", func(t *testing.T) {
		d2 := t.TempDir()
		liar := strings.Replace(validContract(), `"name": "go-build-clean"`, `"name": "other-name"`, 1)
		cpath := write(t, d2, "go-build-clean/contract.json", liar)
		raw := readFile(t, cpath)
		reg := fmt.Sprintf(`{"version": 1, "entries": [
      {"name": "go-build-clean", "version": 1, "contract_sha256": %q,
       "contract_path": "go-build-clean/contract.json", "state": "active"}]}`,
			hashBytes([]byte(raw)))
		r2, err := LoadRegistry(write(t, d2, "contracts.json", reg))
		if err != nil {
			t.Fatal(err)
		}
		if _, _, err := r2.Resolve("go-build-clean@1", allowAll{}); err == nil {
			t.Error("self-declaration disagreeing with registration resolved")
		}
	})

	t.Run("withdrawn refuses evaluation", func(t *testing.T) {
		withdrawn := strings.Replace(readFile(t, regPath), `"state": "active"`, `"state": "withdrawn"`, 1)
		r2, err := LoadRegistry(write(t, dir, "withdrawn.json", withdrawn))
		if err != nil {
			t.Fatal(err)
		}
		if _, _, err := r2.Resolve("go-build-clean@1", allowAll{}); err == nil {
			t.Error("withdrawn contract resolved")
		}
	})
}

func TestAppendOnly(t *testing.T) {
	dir := t.TempDir()
	regPath := registryFor(t, dir, validContract())
	prior, err := LoadRegistry(regPath)
	if err != nil {
		t.Fatal(err)
	}

	t.Run("deletion detected", func(t *testing.T) {
		empty := `{"version": 1, "entries": []}`
		cur, err := LoadRegistry(write(t, dir, "deleted.json", empty))
		if err != nil {
			t.Fatal(err)
		}
		if err := cur.CheckAppendOnly(prior); err == nil {
			t.Error("deleted binding passed append-only check")
		}
	})

	t.Run("rebind detected", func(t *testing.T) {
		rebound := strings.Replace(readFile(t, regPath), prior.Entries[0].Contract, strings.Repeat("ef", 32), 1)
		cur, err := LoadRegistry(write(t, dir, "rebound.json", rebound))
		if err != nil {
			t.Fatal(err)
		}
		if err := cur.CheckAppendOnly(prior); err == nil {
			t.Error("rebound binding passed append-only check")
		}
	})

	t.Run("un-withdrawal detected", func(t *testing.T) {
		withdrawn := strings.Replace(readFile(t, regPath), `"state": "active"`, `"state": "withdrawn"`, 1)
		w, err := LoadRegistry(write(t, dir, "w.json", withdrawn))
		if err != nil {
			t.Fatal(err)
		}
		cur, err := LoadRegistry(regPath) // active again
		if err != nil {
			t.Fatal(err)
		}
		if err := cur.CheckAppendOnly(w); err == nil {
			t.Error("un-withdrawal passed append-only check")
		}
	})
}

func TestEligibilityFailsClosed(t *testing.T) {
	dir := t.TempDir()
	regPath := registryFor(t, dir, validContract())
	r, err := LoadRegistry(regPath)
	if err != nil {
		t.Fatal(err)
	}

	if _, _, err := r.Resolve("go-build-clean@1", nil); err == nil {
		t.Error("resolution without an eligibility checker must refuse")
	}
	if _, _, err := r.Resolve("go-build-clean@1", denyAll{}); err == nil {
		t.Error("ineligible capability binding must refuse")
	}
	if _, _, err := r.Resolve("go-build-clean@1", errChecker{}); err == nil {
		t.Error("eligibility check error must refuse (fail closed)")
	}
}

func TestPathEscapeRefused(t *testing.T) {
	dir := t.TempDir()
	for _, p := range []string{"../outside.json", "/abs.json"} {
		reg := fmt.Sprintf(`{"version": 1, "entries": [
      {"name": "x", "version": 1, "contract_sha256": %q,
       "contract_path": %q, "state": "active"}]}`, strings.Repeat("ab", 32), p)
		if _, err := LoadRegistry(write(t, dir, "esc.json", reg)); err == nil {
			t.Errorf("path-escaping contract_path loaded: %q", p)
		}
	}

	t.Run("symlinked contract_path refused", func(t *testing.T) {
		// Lexically clean path, symlinked outside the registry root
		// (security review L-6: pin the refusal, not just the lexical
		// cases — the L9 symlink lesson).
		outside := t.TempDir()
		foreign := write(t, outside, "foreign.json", validContract())
		d := t.TempDir()
		if err := os.MkdirAll(filepath.Join(d, "x"), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.Symlink(foreign, filepath.Join(d, "x", "contract.json")); err != nil {
			t.Fatal(err)
		}
		raw := readFile(t, foreign)
		reg := fmt.Sprintf(`{"version": 1, "entries": [
      {"name": "go-build-clean", "version": 1, "contract_sha256": %q,
       "contract_path": "x/contract.json", "state": "active"}]}`, hashBytes([]byte(raw)))
		r, err := LoadRegistry(write(t, d, "contracts.json", reg))
		if err != nil {
			t.Fatal(err)
		}
		if _, _, err := r.Resolve("go-build-clean@1", allowAll{}); err == nil {
			t.Error("symlinked contract_path resolved outside the registry root")
		}
	})
}

func TestHardeningRefusals(t *testing.T) {
	dir := t.TempDir()

	t.Run("duplicate JSON keys refused", func(t *testing.T) {
		dup := strings.Replace(validContract(),
			`"build_ok": "PASS",`, `"build_ok": "PASS", "build_ok": "FAIL",`, 1)
		if _, err := LoadContract(write(t, dir, "dup.json", dup)); err == nil {
			t.Error("duplicate result_mapping key loaded last-wins")
		}
		dupTop := strings.Replace(validContract(),
			`"version": 1,`, `"version": 1, "version": 1,`, 1)
		if _, err := LoadContract(write(t, dir, "duptop.json", dupTop)); err == nil {
			t.Error("duplicate top-level key loaded")
		}
	})

	t.Run("non-object config refused", func(t *testing.T) {
		for i, cfg := range []string{`null`, `"str"`, `7`, `[1]`} {
			bad := strings.Replace(validContract(),
				`"config": {"goflags": "-trimpath"}`, `"config": `+cfg, 1)
			if _, err := LoadContract(write(t, dir, fmt.Sprintf("cfg%d.json", i), bad)); err == nil {
				t.Errorf("non-object config loaded: %s", cfg)
			}
		}
	})

	t.Run("non-canonical versions refused", func(t *testing.T) {
		for _, ref := range []string{"x@+1", "x@01", "x@ 1", "x@1e0"} {
			if _, _, err := ParseRef(ref); err == nil {
				t.Errorf("non-canonical version parsed: %q", ref)
			}
		}
	})

	t.Run("optional evidence refused", func(t *testing.T) {
		bad := strings.Replace(validContract(), `"required": true`, `"required": false`, 1)
		if _, err := LoadContract(write(t, dir, "opt.json", bad)); err == nil {
			t.Error("optional evidence slot loaded — hidden default")
		}
	})

	t.Run("oversized contract refused", func(t *testing.T) {
		big := strings.Replace(validContract(),
			`"-trimpath"`, `"`+strings.Repeat("x", maxContractBytes)+`"`, 1)
		if _, err := LoadContract(write(t, dir, "big.json", big)); err == nil {
			t.Error("oversized contract loaded")
		}
	})

	t.Run("overlong name refused", func(t *testing.T) {
		long := strings.Repeat("a", MaxContractNameLen+1)
		bad := strings.Replace(validContract(), `"name": "go-build-clean"`, `"name": "`+long+`"`, 1)
		if _, err := LoadContract(write(t, dir, "long.json", bad)); err == nil {
			t.Error("overlong contract name loaded")
		}
	})
}

// TestNoRegistryWriteCapability proves structurally that this package
// cannot write: no file in the package (tests excluded) REFERENCES an
// os-package mutation function at all — not merely "calls" it, so a
// function-value binding (var scribe = os.WriteFile) is caught too.
// The walk is a real AST inspection over selector expressions, immune
// to import aliases, newline-after-dot formatting, and comments
// (security review H-1: the string-matching version was PoC-bypassed
// by exactly those routes). No exempt function exists — this package
// writes nothing. Mutable-writer set extended past the L9 list.
func TestNoRegistryWriteCapability(t *testing.T) {
	fset := token.NewFileSet()
	pkgs, err := parser.ParseDir(fset, ".", func(fi os.FileInfo) bool {
		return !strings.HasSuffix(fi.Name(), "_test.go")
	}, 0)
	if err != nil {
		t.Fatal(err)
	}
	writers := map[string]bool{
		"WriteFile": true, "Create": true, "CreateTemp": true, "OpenFile": true,
		"Rename": true, "Remove": true, "RemoveAll": true, "Truncate": true,
		"Mkdir": true, "MkdirAll": true, "MkdirTemp": true,
		"Symlink": true, "Link": true, "Chmod": true, "Chown": true, "Lchown": true,
	}
	// Packages that provide indirect write paths; importing them at
	// all is refused in this package (nothing here needs them).
	forbiddenImports := map[string]bool{
		`"io/ioutil"`: true, `"bufio"`: true,
	}
	scanned := 0
	flagged := 0
	for _, pkg := range pkgs {
		for name, file := range pkg.Files {
			scanned++
			osNames := map[string]bool{}
			for _, imp := range file.Imports {
				if imp.Path == nil {
					continue
				}
				if forbiddenImports[imp.Path.Value] {
					t.Errorf("%s imports %s — indirect write path refused in the registry package", name, imp.Path.Value)
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
				continue
			}
			ast.Inspect(file, func(n ast.Node) bool {
				sel, ok := n.(*ast.SelectorExpr)
				if !ok {
					return true
				}
				ident, ok := sel.X.(*ast.Ident)
				if !ok {
					return true
				}
				if osNames[ident.Name] && writers[sel.Sel.Name] {
					flagged++
					t.Errorf("%s references %s.%s at %s — the registry machinery must not write",
						name, ident.Name, sel.Sel.Name, fset.Position(sel.Pos()))
				}
				return true
			})
		}
	}
	if scanned == 0 {
		t.Fatal("audit scanned no files — the wall is not being checked")
	}
	_ = flagged
}

// TestWallCatchesFunctionValueBinding proves the audit itself is
// coupled to its invariant: a doctored source file that binds
// os.WriteFile to a variable (the H-1 PoC) must be flagged by the same
// AST walk. The audit-of-the-audit the L9 mutation lesson demands.
func TestWallCatchesFunctionValueBinding(t *testing.T) {
	src := `package verification

import o "os"

var scribe = o.
	WriteFile
`
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "doctored.go", src, 0)
	if err != nil {
		t.Fatal(err)
	}
	osNames := map[string]bool{}
	for _, imp := range file.Imports {
		if imp.Path.Value == `"os"` {
			if imp.Name != nil {
				osNames[imp.Name.Name] = true
			} else {
				osNames["os"] = true
			}
		}
	}
	caught := false
	ast.Inspect(file, func(n ast.Node) bool {
		sel, ok := n.(*ast.SelectorExpr)
		if !ok {
			return true
		}
		if ident, ok := sel.X.(*ast.Ident); ok && osNames[ident.Name] && sel.Sel.Name == "WriteFile" {
			caught = true
		}
		return true
	})
	if !caught {
		t.Fatal("the AST walk missed an aliased, newline-split function-value binding — the wall does not hold")
	}
}

func readFile(t *testing.T, path string) string {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}
