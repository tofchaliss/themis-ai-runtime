package tools

// The Q-L5-5.2 mechanization (integration-audit D9): the executor
// ambient-environment prohibition, enforced as a checkable wall
// rather than discipline. Executors receive exactly (entry, args,
// target); host environment must be unreachable — no os.Getenv,
// os.Environ, or os.LookupEnv reference anywhere in the tools or
// execution packages' non-test files. References are flagged, not
// just calls (function-value bindings included).

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

var forbiddenEnvRefs = map[string]bool{
	"Getenv": true, "Environ": true, "LookupEnv": true,
	"Setenv": true, "Unsetenv": true, "ExpandEnv": true,
}

func TestNoAmbientEnvironmentInExecutors(t *testing.T) {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("caller unavailable")
	}
	base := filepath.Dir(filepath.Dir(file)) // src/harness
	fset := token.NewFileSet()
	for _, dir := range []string{"tools", "execution"} {
		entries, err := os.ReadDir(filepath.Join(base, dir))
		if err != nil {
			t.Fatal(err)
		}
		for _, e := range entries {
			if !strings.HasSuffix(e.Name(), ".go") || strings.HasSuffix(e.Name(), "_test.go") {
				continue
			}
			path := filepath.Join(base, dir, e.Name())
			f, err := parser.ParseFile(fset, path, nil, 0)
			if err != nil {
				t.Fatal(err)
			}
			osNames := map[string]bool{}
			for _, imp := range f.Imports {
				if strings.Trim(imp.Path.Value, `"`) == "os" {
					name := "os"
					if imp.Name != nil {
						name = imp.Name.Name
					}
					if name == "." {
						t.Errorf("%s: dot-import of os defeats the env wall", path)
						continue
					}
					osNames[name] = true
				}
			}
			ast.Inspect(f, func(n ast.Node) bool {
				sel, ok := n.(*ast.SelectorExpr)
				if !ok {
					return true
				}
				if id, ok := sel.X.(*ast.Ident); ok && osNames[id.Name] && forbiddenEnvRefs[sel.Sel.Name] {
					t.Errorf("%s: reference to os.%s — executors must not reach the ambient environment (Q-L5-5.2)", fset.Position(sel.Pos()), sel.Sel.Name)
				}
				return true
			})
		}
	}
}
