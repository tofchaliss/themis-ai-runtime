package seam

// Not-a-second-L7 (D-L8-21 §3), reviewer-checkable and mutation-
// testable: each check is a wall a relaxation must break.

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"regexp"
	"strings"
	"testing"
)

func parsePkg(t *testing.T, dir string) (*token.FileSet, map[string]*ast.File) {
	t.Helper()
	fset := token.NewFileSet()
	pkgs, err := parser.ParseDir(fset, dir, func(fi os.FileInfo) bool { return !strings.HasSuffix(fi.Name(), "_test.go") }, 0)
	if err != nil {
		t.Fatal(err)
	}
	files := map[string]*ast.File{}
	for _, p := range pkgs {
		for n, f := range p.Files {
			files[n] = f
		}
	}
	if len(files) == 0 {
		t.Fatal("no files scanned")
	}
	return fset, files
}

func TestSeamIsNotASecondL7(t *testing.T) {
	fset, files := parsePkg(t, ".")
	forbidden := map[string]bool{
		`"github.com/tofchaliss/themis/tools"`:     true,
		`"github.com/tofchaliss/themis/execution"`: true,
		`"os/exec"`: true, `"net"`: true, `"net/http"`: true, `"math/rand"`: true,
	}
	modelLiteral := regexp.MustCompile(`(?i)(qwen|gpt|llama|mistral|deepseek|claude)`)
	executeSites, appendSites, goStmts := 0, 0, 0
	for name, f := range files {
		for _, imp := range f.Imports {
			if forbidden[imp.Path.Value] {
				t.Errorf("%s imports %s — the seam authorizes nothing and executes no process", name, imp.Path.Value)
			}
		}
		ast.Inspect(f, func(n ast.Node) bool {
			switch x := n.(type) {
			case *ast.GoStmt:
				goStmts++
				t.Errorf("%s: goroutine at %s — the seam is one control flow", name, fset.Position(x.Pos()))
			case *ast.CallExpr:
				if sel, ok := x.Fun.(*ast.SelectorExpr); ok {
					switch sel.Sel.Name {
					case "Execute":
						executeSites++
						// The one Execute's Model: is the L7-supplied identity.
						if len(x.Args) == 2 {
							if lit, ok := x.Args[1].(*ast.CompositeLit); ok {
								for _, el := range lit.Elts {
									kv, ok := el.(*ast.KeyValueExpr)
									if !ok {
										continue
									}
									if k, ok := kv.Key.(*ast.Ident); ok && k.Name == "Model" {
										if _, isLit := kv.Value.(*ast.BasicLit); isLit {
											t.Errorf("%s: Execute names a model literal at %s", name, fset.Position(kv.Pos()))
										}
										if s, ok := kv.Value.(*ast.SelectorExpr); !ok || s.Sel.Name != "Model" {
											t.Errorf("%s: Execute's Model must be the request's Model at %s", name, fset.Position(kv.Pos()))
										}
									}
								}
							}
						}
					case "AppendEvent":
						appendSites++
						if len(x.Args) > 0 {
							if s, ok := x.Args[0].(*ast.SelectorExpr); !ok || s.Sel.Name != "EvL8Delegation" {
								t.Errorf("%s: AppendEvent with a class other than l8-delegation at %s", name, fset.Position(x.Pos()))
							}
						}
					case "Authorize", "Handle", "Transition", "Seal", "Teardown", "SubmitTask", "Open", "CreateTask", "Recover", "BindArtifact":
						t.Errorf("%s: %s at %s — no authorization, lifecycle, or environment act in the seam", name, sel.Sel.Name, fset.Position(x.Pos()))
					}
				}
			case *ast.BasicLit:
				if x.Kind == token.STRING && modelLiteral.MatchString(x.Value) {
					t.Errorf("%s: model-name literal %s at %s", name, x.Value, fset.Position(x.Pos()))
				}
			case *ast.ForStmt, *ast.RangeStmt:
				// Execute must not sit inside any loop.
				ast.Inspect(n, func(m ast.Node) bool {
					if c, ok := m.(*ast.CallExpr); ok {
						if sel, ok := c.Fun.(*ast.SelectorExpr); ok && sel.Sel.Name == "Execute" {
							t.Errorf("%s: Execute inside a loop at %s", name, fset.Position(c.Pos()))
						}
					}
					return true
				})
			}
			return true
		})
	}
	if executeSites != 1 {
		t.Errorf("exactly one Model.Execute call site, found %d", executeSites)
	}
	if appendSites != 1 {
		t.Errorf("exactly one AppendEvent site, found %d", appendSites)
	}
	_ = goStmts
}

// The core delegation package imports none of orchestration, tools,
// execution — it holds no state machine, cursor, or transition type.
func TestDelegationPackageWall(t *testing.T) {
	_, files := parsePkg(t, "..")
	forbidden := map[string]bool{
		`"github.com/tofchaliss/themis/orchestration"`: true,
		`"github.com/tofchaliss/themis/tools"`:         true,
		`"github.com/tofchaliss/themis/execution"`:     true,
		`"github.com/tofchaliss/themis/runtime/model"`: true,
	}
	for name, f := range files {
		for _, imp := range f.Imports {
			if forbidden[imp.Path.Value] {
				t.Errorf("%s imports %s", name, imp.Path.Value)
			}
		}
		ast.Inspect(f, func(n ast.Node) bool {
			if ts, ok := n.(*ast.TypeSpec); ok {
				lc := strings.ToLower(ts.Name.Name)
				for _, bad := range []string{"machine", "cursor", "transition", "phase", "workflow", "scheduler"} {
					if strings.Contains(lc, bad) {
						t.Errorf("%s declares %s — no workflow vocabulary in L8", name, ts.Name.Name)
					}
				}
			}
			return true
		})
	}
}
