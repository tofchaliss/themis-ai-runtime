package ratchet

// M5 structural walls (D-L11-14 §5: the boundary is checkable code
// shape, not intent). The wall flags REFERENCES — including
// function-value bindings — not just calls (the L10 audit-of-audit
// lesson).

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

// forbiddenImports: no clocks/timers (nothing in L11 may wake up),
// no process execution, no network — L11 consumes committed records
// and produces objects, full stop.
var forbiddenImports = map[string]string{
	"time":      "no clocks or timers — L11 never initiates (D-L11-14)",
	"os/exec":   "no process execution — L11 has no execution plane (D-L11-10)",
	"net":       "no network — L11 reads local governed artifacts only",
	"net/http":  "no network — L11 reads local governed artifacts only",
	"math/rand": "no randomness — comparators are deterministic (D-L11-17)",
	// close-review additions: the deprecated and low-level write
	// routes around the os wall.
	"io/ioutil": "deprecated write surface — the ratchet package writes nothing itself",
	"syscall":   "no low-level filesystem/process access",
}

// forbiddenOSRefs: the no-write wall. This package READS registries
// and hands bytes to the L6 store; it never writes files itself —
// a write API here is the seed of a self-registering mechanism.
var forbiddenOSRefs = map[string]bool{
	"WriteFile": true, "Create": true, "CreateTemp": true,
	"OpenFile": true, "Rename": true, "Remove": true,
	"RemoveAll": true, "MkdirAll": true, "Mkdir": true,
	"Chmod": true, "Chtimes": true, "Truncate": true, "Link": true,
	"Symlink": true, "WriteString": true,
}

type wallViolation struct {
	pos  token.Position
	what string
}

// auditFile inspects one parsed file for wall violations.
func auditFile(fset *token.FileSet, f *ast.File) []wallViolation {
	var out []wallViolation
	osNames := map[string]bool{} // local names binding the os package
	for _, imp := range f.Imports {
		path := strings.Trim(imp.Path.Value, `"`)
		if why, bad := forbiddenImports[path]; bad {
			out = append(out, wallViolation{fset.Position(imp.Pos()), "import " + path + ": " + why})
		}
		if path == "os" {
			name := "os"
			if imp.Name != nil {
				name = imp.Name.Name
			}
			if name == "." {
				// A dot-import would defeat selector matching below.
				out = append(out, wallViolation{fset.Position(imp.Pos()), "dot-import of os defeats the write wall"})
				continue
			}
			osNames[name] = true
		}
	}
	ast.Inspect(f, func(n ast.Node) bool {
		switch v := n.(type) {
		case *ast.SelectorExpr:
			if id, ok := v.X.(*ast.Ident); ok && osNames[id.Name] && forbiddenOSRefs[v.Sel.Name] {
				out = append(out, wallViolation{fset.Position(v.Pos()), "reference to os." + v.Sel.Name + " — the ratchet package writes nothing itself"})
			}
		case *ast.GoStmt:
			out = append(out, wallViolation{fset.Position(v.Pos()), "go statement — no work outlives an invocation (D-L11-14)"})
		}
		return true
	})
	return out
}

func packageDir(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("caller unavailable")
	}
	return filepath.Dir(file)
}

func TestASTWall(t *testing.T) {
	dir := packageDir(t)
	fset := token.NewFileSet()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		if !strings.HasSuffix(e.Name(), ".go") || strings.HasSuffix(e.Name(), "_test.go") {
			continue
		}
		f, err := parser.ParseFile(fset, filepath.Join(dir, e.Name()), nil, 0)
		if err != nil {
			t.Fatal(err)
		}
		for _, v := range auditFile(fset, f) {
			t.Errorf("%s: %s", v.pos, v.what)
		}
	}
}

// Audit-of-audit: the wall itself must catch references, not merely
// calls — including function-value bindings (the L10 lesson).
func TestWallCatchesFunctionValueBinding(t *testing.T) {
	src := `package x
import "os"
var w = os.WriteFile
func f() { _ = w }
`
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "synthetic.go", src, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(auditFile(fset, f)) == 0 {
		t.Fatal("wall missed a function-value binding of os.WriteFile")
	}
	src2 := `package x
import "time"
func f() { _ = time.Now }
`
	f2, _ := parser.ParseFile(fset, "synthetic2.go", src2, 0)
	if len(auditFile(fset, f2)) == 0 {
		t.Fatal("wall missed a time import")
	}
	src3 := `package x
func f() { go func() {}() }
`
	f3, _ := parser.ParseFile(fset, "synthetic3.go", src3, 0)
	if len(auditFile(fset, f3)) == 0 {
		t.Fatal("wall missed a go statement")
	}
}

// No mutable package state: every package-level var is one of the
// closed declaration tables, and no function assigns to any of them
// (D-L11-11: no caches, no status, no counters — an L11 enumeration
// plus mutable status is a registry in disguise).
func TestNoMutablePackageState(t *testing.T) {
	allowedVars := map[string]bool{
		// closed vocabularies / tables (written only at declaration)
		"Families": true, "selectorSources": true,
		"baselineConstraints": true, "orderingKinds": true,
		"deltaFieldTypes": true, "normativeTokens": true,
		"provenanceRequired": true, "comparators": true,
		"reasonClasses": true, "registryKinds": true,
		"changeRelations": true, "requiredProvenance": true,
		// M7 remediation tables: the closed door table (C-1) and the
		// L6-plane source set (H-1) — declaration-only, like the rest.
		"doorHashFields": true, "l6Sources": true,
		// error values + syntax patterns
		"ErrCriterion": true, "ErrSet": true, "ErrRegistry": true,
		"ErrResolve": true, "shaSyntax": true, "nameSyntax": true,
		"fieldSyntax": true, "versionSyntax": true,
	}
	dir := packageDir(t)
	fset := token.NewFileSet()
	entries, _ := os.ReadDir(dir)
	pkgVars := map[string]bool{}
	var files []*ast.File
	for _, e := range entries {
		if !strings.HasSuffix(e.Name(), ".go") || strings.HasSuffix(e.Name(), "_test.go") {
			continue
		}
		f, err := parser.ParseFile(fset, filepath.Join(dir, e.Name()), nil, 0)
		if err != nil {
			t.Fatal(err)
		}
		files = append(files, f)
		for _, d := range f.Decls {
			gd, ok := d.(*ast.GenDecl)
			if !ok || gd.Tok != token.VAR {
				continue
			}
			for _, spec := range gd.Specs {
				vs := spec.(*ast.ValueSpec)
				for _, name := range vs.Names {
					pkgVars[name.Name] = true
					if !allowedVars[name.Name] {
						t.Errorf("%s: package-level var %q is not in the closed allowlist — extend deliberately or remove", fset.Position(name.Pos()), name.Name)
					}
				}
			}
		}
	}
	// No function may write to package-level state.
	for _, f := range files {
		for _, d := range f.Decls {
			fd, ok := d.(*ast.FuncDecl)
			if !ok || fd.Body == nil {
				continue
			}
			ast.Inspect(fd.Body, func(n ast.Node) bool {
				switch v := n.(type) {
				case *ast.AssignStmt:
					for _, lhs := range v.Lhs {
						checkWrite(t, fset, lhs, pkgVars)
					}
					for _, rhs := range v.Rhs {
						checkAlias(t, fset, rhs, pkgVars)
					}
				case *ast.IncDecStmt:
					checkWrite(t, fset, v.X, pkgVars)
				}
				return true
			})
		}
	}
}

func checkWrite(t *testing.T, fset *token.FileSet, lhs ast.Expr, pkgVars map[string]bool) {
	t.Helper()
	switch v := lhs.(type) {
	case *ast.Ident:
		if pkgVars[v.Name] {
			t.Errorf("%s: function writes package-level var %q — L11 has no mutable state", fset.Position(v.Pos()), v.Name)
		}
	case *ast.IndexExpr:
		if id, ok := v.X.(*ast.Ident); ok && pkgVars[id.Name] {
			t.Errorf("%s: function writes into package-level table %q", fset.Position(v.Pos()), id.Name)
		}
	case *ast.StarExpr:
		// *p = ... where p aliases package state (close-review gap).
		if id, ok := v.X.(*ast.UnaryExpr); ok {
			if inner, ok := id.X.(*ast.Ident); ok && pkgVars[inner.Name] {
				t.Errorf("%s: pointer write through package-level var %q", fset.Position(v.Pos()), inner.Name)
			}
		}
	}
}

// checkAlias flags `x := pkgTable` bindings — the aliasing route to
// mutating a package-level map (close-review gap). Reading a table
// entry is fine; taking the whole table as a value is not needed
// anywhere in this package and is refused wholesale.
func checkAlias(t *testing.T, fset *token.FileSet, rhs ast.Expr, pkgVars map[string]bool) {
	t.Helper()
	if id, ok := rhs.(*ast.Ident); ok && pkgVars[id.Name] {
		t.Errorf("%s: package-level table %q aliased into a local — mutation route", fset.Position(id.Pos()), id.Name)
	}
}

// L11 output is terminal (D-L11-15 Class 4/5): the only importers of
// this package in the whole harness are the package itself and its
// invocation surface. In particular the router / internal/service
// must NOT import it — Δ is not a selection input.
func TestRatchetImportersAreClosed(t *testing.T) {
	// Full module-relative paths (close-review gap: base-name
	// matching would silently allow a future internal/ratchet).
	root := filepath.Join(packageDir(t), "..")
	allowedImporters := map[string]bool{
		"ratchet":            true,
		"cmd/themis-ratchet": true,
	}
	fset := token.NewFileSet()
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if !strings.HasSuffix(path, ".go") {
			return nil
		}
		f, perr := parser.ParseFile(fset, path, nil, parser.ImportsOnly)
		if perr != nil {
			return nil // non-building fixtures are not importers
		}
		for _, imp := range f.Imports {
			if strings.Trim(imp.Path.Value, `"`) == "github.com/tofchaliss/themis/ratchet" {
				rel, rerr := filepath.Rel(root, filepath.Dir(path))
				if rerr != nil || !allowedImporters[filepath.ToSlash(rel)] {
					t.Errorf("%s imports the ratchet package — L11 output is terminal and enters no other plane", path)
				}
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

// API closure: the exported surface is deliberate. Any new export is
// a review decision, not an accident (the L9/L10 guard pattern).
func TestAPIClosure(t *testing.T) {
	allowed := map[string]bool{
		// registries + artifacts
		"Entry": true, "EntryState": true, "Registry": true,
		"StateActive": true, "StateWithdrawn": true,
		"LoadRegistry": true, "ParseRef": true,
		"Criterion": true, "LoadCriterion": true, "ParseCriterion": true,
		"Selector": true, "ComparatorBinding": true, "DeltaField": true,
		"Ordering": true, "OrderingField": true, "Families": true,
		"RegressionSet": true, "ParseRegressionSet": true,
		"MaxNameLen": true,
		// comparison core
		"Comparator": true, "LookupComparator": true, "CanonicalDelta": true,
		"Compare": true, "CompareInput": true, "ComparisonPackage": true,
		"RefusalFact": true, "ReasonClass": true, "CheckReasonClass": true,
		"EvidenceRef": true, "AdmissionObservation": true,
		"ReasonUnregisteredArtifact": true, "ReasonWithdrawnArtifact": true,
		"ReasonUnadmittedBaseline": true, "ReasonEvidenceUnavailable": true,
		"ReasonComparabilityViolation": true, "ReasonProvenanceViolation": true,
		"ReasonIntegrityFailure": true,
		// derivations (stateless)
		"Relation": true, "RelBetter": true, "RelWorse": true,
		"RelEqual": true, "RelIncomparable": true,
		"DerivePerField": true, "DeriveRelation": true,
		"DeriveNonRegression": true, "FieldDerivation": true,
		// packages, persistence, reconstruction
		"CanonicalBytes": true, "InstanceID": true,
		"SetConstituent": true, "RegressionPackage": true,
		"BuildRegressionPackage": true, "DeriveResistantUnderSet": true,
		"StoreInstance": true, "LoadComparison": true,
		"Reconstruct": true, "Reconstruction": true, "ReconstructionResult": true,
		"ReconConfirmed": true, "ReconMissingInputs": true, "ReconDiscrepancy": true,
		// candidate + plan
		"Candidate": true, "BaselineRef": true, "ParseCandidate": true,
		"EvaluationPlan": true, "RequiredRun": true, "ParsePlan": true,
		"CheckPlanConformance": true,
		// M7 remediation surface: door resolution (C-1), evidence
		// grounding + strict input parsing (H-1/M-2) — deliberate.
		"ObserveAdmission": true, "ReverifyAdmission": true,
		"GroundFacts": true, "ExternalResolver": true,
		"ParseEvidenceRefs": true,
		// errors
		"ErrCriterion": true, "ErrSet": true, "ErrRegistry": true, "ErrResolve": true,
	}
	// Exported METHODS are closed too (close-review gap: a future
	// exported method escaped the regime entirely).
	allowedMethods := map[string]bool{
		"Registry.Root": true, "Registry.CheckAppendOnly": true,
		"Registry.ResolveCriterion": true, "Registry.ResolveSet": true,
	}
	dir := packageDir(t)
	fset := token.NewFileSet()
	entries, _ := os.ReadDir(dir)
	for _, e := range entries {
		if !strings.HasSuffix(e.Name(), ".go") || strings.HasSuffix(e.Name(), "_test.go") {
			continue
		}
		f, err := parser.ParseFile(fset, filepath.Join(dir, e.Name()), nil, 0)
		if err != nil {
			t.Fatal(err)
		}
		for _, d := range f.Decls {
			switch v := d.(type) {
			case *ast.FuncDecl:
				if v.Recv == nil && v.Name.IsExported() && !allowed[v.Name.Name] {
					t.Errorf("%s: undeclared exported func %s", fset.Position(v.Pos()), v.Name.Name)
				}
				if v.Recv != nil && v.Name.IsExported() {
					recv := receiverTypeName(v.Recv)
					if recv != "" && ast.IsExported(recv) && !allowedMethods[recv+"."+v.Name.Name] {
						t.Errorf("%s: undeclared exported method %s.%s", fset.Position(v.Pos()), recv, v.Name.Name)
					}
				}
			case *ast.GenDecl:
				for _, spec := range v.Specs {
					switch s := spec.(type) {
					case *ast.TypeSpec:
						if s.Name.IsExported() && !allowed[s.Name.Name] {
							t.Errorf("%s: undeclared exported type %s", fset.Position(s.Pos()), s.Name.Name)
						}
					case *ast.ValueSpec:
						for _, name := range s.Names {
							if name.IsExported() && !allowed[name.Name] {
								t.Errorf("%s: undeclared exported value %s", fset.Position(name.Pos()), name.Name)
							}
						}
					}
				}
			}
		}
	}
}

func receiverTypeName(fl *ast.FieldList) string {
	if fl == nil || len(fl.List) == 0 {
		return ""
	}
	switch e := fl.List[0].Type.(type) {
	case *ast.Ident:
		return e.Name
	case *ast.StarExpr:
		if id, ok := e.X.(*ast.Ident); ok {
			return id.Name
		}
	}
	return ""
}

// The forbidden-component tripwires (D-L11-20 §6): no identifier in
// the package may name a forbidden subsystem.
func TestForbiddenComponentNames(t *testing.T) {
	forbidden := []string{"feedback", "champion", "scheduler", "queue", "optimizer", "watcher", "poller"}
	dir := packageDir(t)
	fset := token.NewFileSet()
	entries, _ := os.ReadDir(dir)
	for _, e := range entries {
		if !strings.HasSuffix(e.Name(), ".go") || strings.HasSuffix(e.Name(), "_test.go") {
			continue
		}
		f, err := parser.ParseFile(fset, filepath.Join(dir, e.Name()), nil, 0)
		if err != nil {
			t.Fatal(err)
		}
		ast.Inspect(f, func(n ast.Node) bool {
			id, ok := n.(*ast.Ident)
			if !ok {
				return true
			}
			lower := strings.ToLower(id.Name)
			for _, bad := range forbidden {
				if strings.Contains(lower, bad) {
					t.Errorf("%s: identifier %q names a forbidden subsystem (%s)", fset.Position(id.Pos()), id.Name, bad)
				}
			}
			return true
		})
	}
}
