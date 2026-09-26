package orchestration

// W-M2 (D-W-6 precision): the L5 writer identity cannot be introduced
// by another package's direct emission path. Architectural, not
// lexical: (1) `execution` imports nothing of the record plane — its
// only route to the record is the Witness interface L7 hands it;
// (2) no non-test file outside `state` passes the literal "l5" as an
// AppendEvent writer; (3) `.L5Sink()` is constructed only here, in the
// orchestrator, at the record boundary. Runtime refusal (the sink's
// class→writer table and the handle-only classes) is the independent
// second half.

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestL5WitnessWall(t *testing.T) {
	root := filepath.Join("..")
	fset := token.NewFileSet()
	scanned := 0
	err := filepath.WalkDir(root, func(p string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if p != root && (d.Name() == "testdata" || strings.HasPrefix(d.Name(), ".")) {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(p, ".go") || strings.HasSuffix(p, "_test.go") {
			return nil
		}
		f, err := parser.ParseFile(fset, p, nil, 0)
		if err != nil {
			return err
		}
		scanned++
		rel, _ := filepath.Rel(root, p)
		pkgDir := filepath.Dir(rel)
		// (1) execution imports no record-plane package.
		if pkgDir == "execution" {
			for _, imp := range f.Imports {
				if strings.HasSuffix(strings.Trim(imp.Path.Value, `"`), "/src/harness/state") {
					t.Errorf("%s imports state — L5 must reach the record only through the Witness handle", rel)
				}
			}
		}
		ast.Inspect(f, func(n ast.Node) bool {
			call, ok := n.(*ast.CallExpr)
			if !ok {
				return true
			}
			sel, ok := call.Fun.(*ast.SelectorExpr)
			if !ok {
				return true
			}
			switch sel.Sel.Name {
			case "AppendEvent":
				// (2) the writer argument is the second positional
				// argument; a literal "l5" outside state is a forgery path.
				if len(call.Args) >= 2 && pkgDir != "state" {
					if lit, ok := call.Args[1].(*ast.BasicLit); ok && lit.Value == `"l5"` {
						t.Errorf("%s: AppendEvent with writer \"l5\" outside state at %s", rel, fset.Position(lit.Pos()))
					}
				}
			case "L5Sink":
				// (3) the handle is constructed at the record boundary only.
				if pkgDir != "orchestration" && pkgDir != "state" {
					t.Errorf("%s: L5Sink() constructed outside the orchestrator at %s", rel, fset.Position(sel.Pos()))
				}
			}
			return true
		})
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if scanned < 50 {
		t.Fatalf("scanned only %d files — the wall is not covering the module", scanned)
	}
}
