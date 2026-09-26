package orchestration

// D-R-4 / D-R-2: the runtime governance doors are READ by loaders that
// never write, and L11 can reach none of them. (1) `skills`,
// `deployment`, `decisions`, and the Themis integration packages hold
// no filesystem writer and no subprocess; (2) `ratchet` and
// `cmd/themis-ratchet` import none of `skills`, `deployment`,
// `decisions` — the first half of an automation path cannot appear by
// convenience import. Runtime refusal is not the guard here; absence
// of the code path is.

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGovernanceDoorsAreReadOnlyAndUnreachableFromL11(t *testing.T) {
	root := ".."
	writers := map[string]bool{
		"WriteFile": true, "Create": true, "CreateTemp": true, "OpenFile": true, "Rename": true,
		"Remove": true, "RemoveAll": true, "Truncate": true, "Mkdir": true, "MkdirAll": true,
		"MkdirTemp": true, "Symlink": true, "Link": true, "Chmod": true, "Chown": true,
	}
	doorPkgs := []string{"skills", "deployment", "decisions", "integrations/themis/client", "integrations/themis/contracts"}
	// L9 instantiation writes the ENVELOPE and effective artifacts into
	// a caller-chosen OutDir (D-L9-13: an ordinary governed envelope);
	// it never writes the catalog. The door itself — catalog.go and the
	// manifest loader — is held to the no-writer rule like the others.
	writerAllowed := map[string]bool{"instantiate.go": true}
	forbiddenForL11 := []string{"/src/harness/skills", "/src/harness/deployment", "/src/harness/decisions"}
	fset := token.NewFileSet()
	scanned := 0
	for _, dir := range doorPkgs {
		pkgs, err := parser.ParseDir(fset, filepath.Join(root, dir), func(fi os.FileInfo) bool { return !strings.HasSuffix(fi.Name(), "_test.go") }, 0)
		if err != nil {
			t.Fatal(err)
		}
		for _, p := range pkgs {
			for name, f := range p.Files {
				scanned++
				if dir == "skills" && writerAllowed[filepath.Base(name)] {
					continue
				}
				osNames := map[string]bool{}
				for _, imp := range f.Imports {
					v := strings.Trim(imp.Path.Value, `"`)
					if v == "os/exec" || v == "net" || strings.HasSuffix(v, "/src/harness/ratchet") {
						t.Errorf("%s imports %s — a governance door loader neither executes nor reaches L11", name, v)
					}
					if v == "os" {
						if imp.Name != nil {
							osNames[imp.Name.Name] = true
						} else {
							osNames["os"] = true
						}
					}
				}
				ast.Inspect(f, func(n ast.Node) bool {
					sel, ok := n.(*ast.SelectorExpr)
					if !ok {
						return true
					}
					if id, ok := sel.X.(*ast.Ident); ok && osNames[id.Name] && writers[sel.Sel.Name] {
						t.Errorf("%s: %s.%s at %s — door loaders never write", name, id.Name, sel.Sel.Name, fset.Position(sel.Pos()))
					}
					return true
				})
			}
		}
	}
	for _, dir := range []string{"ratchet", "cmd/themis-ratchet"} {
		pkgs, err := parser.ParseDir(fset, filepath.Join(root, dir), func(fi os.FileInfo) bool { return !strings.HasSuffix(fi.Name(), "_test.go") }, parser.ImportsOnly)
		if err != nil {
			t.Fatal(err)
		}
		for _, p := range pkgs {
			for name, f := range p.Files {
				scanned++
				for _, imp := range f.Imports {
					v := strings.Trim(imp.Path.Value, `"`)
					for _, fb := range forbiddenForL11 {
						if strings.HasSuffix(v, fb) {
							t.Errorf("%s imports %s — L11 produces evidence only and can reach no door (D-R-4)", name, v)
						}
					}
				}
			}
		}
	}
	if scanned < 10 {
		t.Fatalf("scanned only %d files", scanned)
	}
}
