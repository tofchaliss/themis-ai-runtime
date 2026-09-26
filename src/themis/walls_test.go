package themis

// The four walls of D-T-10, as tests. Each is a mutation target: relax
// the boundary and the corresponding wall must fail.

import (
	"encoding/json"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

const (
	harnessModule = "github.com/tofchaliss/themis-ai-runtime/src/harness"
	themisModule  = "github.com/tofchaliss/themis-app"
	intakePkg     = themisModule + "/intake"
)

var harnessDir = filepath.Join("..", "harness")

// goList runs `go list` in dir and returns stdout lines.
func goList(t *testing.T, dir string, args ...string) []string {
	t.Helper()
	cmd := exec.Command("go", append([]string{"list"}, args...)...)
	cmd.Dir = dir
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("go list %v in %s: %v", args, dir, err)
	}
	var lines []string
	for _, l := range strings.Split(string(out), "\n") {
		if l = strings.TrimSpace(l); l != "" {
			lines = append(lines, l)
		}
	}
	return lines
}

// Wall 1 — dependency wall: the ENTIRE dependency graph of every
// harness LAYER package reaches no Themis package at all (an indirect
// path harness → A → B → themis-app/... is a violation too). The one
// binary that legitimately wires the read seam, cmd/themis-run, is
// governed by wall 2; every other package, including every other
// binary, is under this wall.
func TestWall1HarnessGraphReachesNoThemis(t *testing.T) {
	pkgs := goList(t, harnessDir, "./...")
	var layer []string
	for _, p := range pkgs {
		if p == harnessModule+"/cmd/themis-run" {
			continue
		}
		layer = append(layer, p)
	}
	if len(layer) < 20 {
		t.Fatalf("suspiciously few harness packages (%d) — the wall is not being checked", len(layer))
	}
	deps := goList(t, harnessDir, append([]string{"-deps"}, layer...)...)
	if len(deps) < 50 {
		t.Fatalf("suspiciously few harness deps (%d)", len(deps))
	}
	for _, d := range deps {
		if strings.HasPrefix(d, themisModule) {
			t.Errorf("harness dependency graph reaches %s — no harness layer package may depend on Themis (D-T-10)", d)
		}
	}
}

// Wall 2 — binary wall: cmd/themis-run imports NOTHING of the Themis
// module. Since D-I-3 the read door is an HTTP client inside the
// harness (integrations/themis/client); the stand-in store is gone.
func TestWall2ThemisRunImportsNoThemisPackage(t *testing.T) {
	deps := goList(t, harnessDir, "-deps", "./cmd/themis-run")
	for _, d := range deps {
		if strings.HasPrefix(d, themisModule) {
			t.Fatalf("cmd/themis-run reaches %s — the harness binary links no Themis code (D-I-7)", d)
		}
	}
}

// Wall 3 — writer wall: intake imports no execution or model
// machinery and has at most one filesystem write site (the O_EXCL
// Position append, T-M3).
func TestWall3WriterBoundaries(t *testing.T) {
	forbidden := map[string]bool{
		`"` + harnessModule + `/tools"`:         true,
		`"` + harnessModule + `/orchestration"`: true,
		`"` + harnessModule + `/execution"`:     true,
		`"` + harnessModule + `/runtime/model"`: true,
		`"` + harnessModule + `/instructions"`:  true,
		`"` + harnessModule + `/context"`:       true,
		`"os/exec"`:                             true, `"net"`: true, `"net/http"`: true, `"math/rand"`: true, `"io/ioutil"`: true,
	}
	writers := map[string]bool{
		"WriteFile": true, "Create": true, "CreateTemp": true, "OpenFile": true,
		"Rename": true, "Remove": true, "RemoveAll": true, "Truncate": true,
		"Mkdir": true, "MkdirAll": true, "MkdirTemp": true,
		"Symlink": true, "Link": true, "Chmod": true, "Chown": true, "Lchown": true,
	}
	check := func(dir string, maxWriteSites int) {
		fset := token.NewFileSet()
		pkgs, err := parser.ParseDir(fset, dir, func(fi os.FileInfo) bool { return !strings.HasSuffix(fi.Name(), "_test.go") }, 0)
		if err != nil {
			t.Fatal(err)
		}
		sites := 0
		scanned := 0
		for _, p := range pkgs {
			for name, f := range p.Files {
				scanned++
				osNames := map[string]bool{}
				for _, imp := range f.Imports {
					if forbidden[imp.Path.Value] {
						t.Errorf("%s imports %s — outside the Themis boundary (D-T-10)", name, imp.Path.Value)
					}
					if imp.Path.Value == `"os"` {
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
						sites++
						if sel.Sel.Name != "OpenFile" {
							t.Errorf("%s: %s.%s at %s — the only permitted writer is the O_EXCL Position append", name, id.Name, sel.Sel.Name, fset.Position(sel.Pos()))
						}
					}
					return true
				})
			}
		}
		if scanned == 0 {
			t.Fatalf("%s: no files scanned — the wall is not being checked", dir)
		}
		if sites > maxWriteSites {
			t.Errorf("%s: %d write sites, at most %d permitted", dir, sites, maxWriteSites)
		}
	}
	check("intake", 1)
}

// Wall 4 — capability wall: no registry the deployment could anchor
// declares a capability whose NAME, target class, or trust could
// write, decide, or classify a Position; and the harness executor
// vocabulary has no such verb. Checked over the decoded registrations,
// never over prose (a description saying "decides what it means" is
// not a capability).
func TestWall4NoPositionCapability(t *testing.T) {
	entries, err := filepath.Glob(filepath.Join("..", "..", "policies", "tools", "registry-v*.json"))
	if err != nil || len(entries) == 0 {
		t.Fatal("no tool registries found")
	}
	type tool struct {
		Name   string `json:"name"`
		Target string `json:"target"`
		Trust  string `json:"trust"`
	}
	type registry struct {
		Tools []tool `json:"tools"`
	}
	allowedTargets := map[string]bool{"workspace-path": true, "themis-id": true, "none": true, "delegation-template": true}
	for _, p := range entries {
		b, err := os.ReadFile(p)
		if err != nil {
			t.Fatal(err)
		}
		var r registry
		if err := json.Unmarshal(b, &r); err != nil {
			t.Fatalf("%s: %v", p, err)
		}
		if len(r.Tools) == 0 {
			t.Fatalf("%s: no tools decoded — the wall is not being checked", p)
		}
		for _, tl := range r.Tools {
			lc := strings.ToLower(tl.Name)
			for _, bad := range []string{"position", "decide", "accept", "disposition", "set_finding", "write_finding", "put_", "update_"} {
				if strings.Contains(lc, bad) {
					t.Errorf("%s registers %q — no Position-writing capability may exist in the harness registry (D-T-10)", p, tl.Name)
				}
			}
			if !allowedTargets[tl.Target] {
				t.Errorf("%s: %q has target class %q outside the closed set", p, tl.Name, tl.Target)
			}
			// The two Themis-id readers are the only governed-record
			// capabilities and they are READS by name.
			if tl.Trust == "governed-record" && tl.Name != "get_finding" && tl.Name != "get_product" {
				t.Errorf("%s: %q mints governed-record — only the two Themis read capabilities may", p, tl.Name)
			}
		}
	}
	b, err := os.ReadFile(filepath.Join(harnessDir, "tools", "execute.go"))
	if err != nil {
		t.Fatal(err)
	}
	for _, bad := range []string{`"write_position"`, `"set_position"`, `"decide"`, `"accept_finding"`, `"write_finding"`} {
		if strings.Contains(string(b), bad) {
			t.Errorf("harness executor table carries %s", bad)
		}
	}
}
