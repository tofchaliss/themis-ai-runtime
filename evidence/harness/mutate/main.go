// themis-mutate is a systematic mutation pass over the harness's
// REFUSALS.
//
// The operator is deliberately narrow: for every `if <cond> { ... return
// <error> ... }`, replace `<cond>` with `false` and run that package's
// tests. A control whose guard can be disabled with no test failing is
// an UNTESTED control — and that is the one class the 2026-09-14
// evidence sweep structurally could not reach, because it finds controls
// that are untested AND correctly cited.
//
// Why this operator and not a general mutation suite: this system's
// entire thesis is that deterministic controls refuse. A guard that
// never refuses in any test is precisely the defect worth finding, and
// the same mutation done by hand has already exposed real gaps today.
//
// ISOLATION IS ENFORCED, not requested. This program mutates source
// files in place, so it REFUSES to run anywhere but a linked git
// worktree — detected by `.git` being a regular file rather than a
// directory. The AGENTS.md probe-isolation invariant is not advice here;
// running this against a live tree would corrupt it.
//
// A surviving mutant is a FINDING, not a failure: it says the control at
// that line is not covered by its package's tests. Some survivors will
// be equivalent mutants (the condition is unreachable, or redundant with
// another guard). Each needs judgement; none should be dismissed
// unexamined.
package main

import (
	"flag"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

type guard struct {
	pkg      string
	file     string
	line     int
	condFrom int // byte offset of condition start
	condTo   int
	condText string
	snippet  string
}

func main() {
	repo := flag.String("worktree", "", "path to a LINKED git worktree (never the live tree)")
	only := flag.String("package", "", "limit to one package, e.g. orchestration")
	limit := flag.Int("limit", 0, "stop after N mutants (0 = all)")
	timeout := flag.String("timeout", "90s", "per-package go test timeout")
	flag.Parse()
	if *repo == "" {
		bail("-worktree is required")
	}

	// Isolation, enforced. A linked worktree has .git as a FILE; the
	// main checkout has it as a directory.
	gi, err := os.Stat(filepath.Join(*repo, ".git"))
	if err != nil {
		bail("%s is not a git checkout: %v", *repo, err)
	}
	if gi.IsDir() {
		bail("%s is the MAIN checkout (.git is a directory).\n"+
			"This program rewrites source in place. Create a linked worktree:\n"+
			"  git worktree add /tmp/themis-mutate HEAD", *repo)
	}
	mod := filepath.Join(*repo, "src", "harness")
	if _, err := os.Stat(filepath.Join(mod, "go.mod")); err != nil {
		bail("no module at %s: %v", mod, err)
	}

	guards, err := collect(mod, *only)
	if err != nil {
		bail("collect: %v", err)
	}
	sort.Slice(guards, func(i, j int) bool {
		if guards[i].pkg != guards[j].pkg {
			return guards[i].pkg < guards[j].pkg
		}
		if guards[i].file != guards[j].file {
			return guards[i].file < guards[j].file
		}
		return guards[i].line < guards[j].line
	})
	if *limit > 0 && len(guards) > *limit {
		guards = guards[:*limit]
	}
	fmt.Printf("refusal guards found: %d\n", len(guards))
	fmt.Printf("worktree: %s\n\n", *repo)

	// Baseline: the suite must be green before any mutant means anything.
	fmt.Print("baseline (unmutated) ... ")
	pkgs := map[string]bool{}
	for _, g := range guards {
		pkgs[g.pkg] = true
	}
	for p := range pkgs {
		if ok, out := runTests(mod, p, *timeout); !ok {
			fmt.Printf("FAILED for ./%s\n%s\n", p, tail(out, 12))
			bail("baseline is not green — mutation results would be meaningless")
		}
	}
	fmt.Printf("green across %d package(s)\n\n", len(pkgs))

	var survivors []guard
	killed, compileErr := 0, 0
	start := time.Now()
	for i, g := range guards {
		orig, err := os.ReadFile(g.file)
		if err != nil {
			bail("read %s: %v", g.file, err)
		}
		mutated := append([]byte{}, orig[:g.condFrom]...)
		mutated = append(mutated, []byte("false")...)
		mutated = append(mutated, orig[g.condTo:]...)
		if err := os.WriteFile(g.file, mutated, 0o644); err != nil {
			bail("write %s: %v", g.file, err)
		}
		ok, out := runTests(mod, g.pkg, *timeout)
		if werr := os.WriteFile(g.file, orig, 0o644); werr != nil {
			bail("RESTORE FAILED for %s: %v — worktree is now dirty", g.file, werr)
		}
		switch {
		case strings.Contains(out, "[build failed]") || strings.Contains(out, "declared and not used"):
			compileErr++
			fmt.Printf("  [%d/%d] %s:%d does-not-compile\n", i+1, len(guards), rel(mod, g.file), g.line)
		case ok:
			survivors = append(survivors, g)
			fmt.Printf("  [%d/%d] \033[31mSURVIVED\033[0m %s:%d  if %s\n",
				i+1, len(guards), rel(mod, g.file), g.line, oneLine(g.condText))
		default:
			killed++
			if (i+1)%25 == 0 {
				fmt.Printf("  [%d/%d] %d killed, %d survived, %s elapsed\n",
					i+1, len(guards), killed, len(survivors), time.Since(start).Round(time.Second))
			}
		}
	}

	fmt.Printf("\n=== RESULT (%s) ===\n", time.Since(start).Round(time.Second))
	fmt.Printf("  guards mutated : %d\n", len(guards))
	fmt.Printf("  killed         : %d\n", killed)
	fmt.Printf("  did not compile: %d  (cannot ship — not a gap)\n", compileErr)
	fmt.Printf("  SURVIVED       : %d\n", len(survivors))
	if len(survivors) == 0 {
		fmt.Println("\nEvery refusal guard is covered by its package's tests.")
		return
	}
	fmt.Println("\nSurvivors — each is a control whose guard can be disabled with no")
	fmt.Println("test failing. Some will be equivalent mutants; none are self-evidently")
	fmt.Println("safe. Each needs a judgement recorded.")
	fmt.Println()
	for _, g := range survivors {
		fmt.Printf("  %s:%d\n      if %s\n      %s\n",
			rel(mod, g.file), g.line, oneLine(g.condText), oneLine(g.snippet))
	}
}

// collect finds every `if <cond> { ... return ...Errorf/errors.New... }`
// in non-test sources. The body test is textual on purpose: it selects
// the refusal sites that carry a message, which are the ones the
// architecture treats as controls.
func collect(mod, only string) ([]guard, error) {
	var out []guard
	err := filepath.Walk(mod, func(p string, fi os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if fi.IsDir() {
			if fi.Name() == "testdata" || fi.Name() == "bin" {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(p, ".go") || strings.HasSuffix(p, "_test.go") {
			return nil
		}
		pkg := filepath.ToSlash(rel(mod, filepath.Dir(p)))
		if pkg == "." {
			pkg = ""
		}
		if only != "" && pkg != only {
			return nil
		}
		src, rerr := os.ReadFile(p)
		if rerr != nil {
			return rerr
		}
		fset := token.NewFileSet()
		f, perr := parser.ParseFile(fset, p, src, 0)
		if perr != nil {
			return nil // unparseable: skip rather than guess
		}
		ast.Inspect(f, func(n ast.Node) bool {
			ifs, ok := n.(*ast.IfStmt)
			if !ok || ifs.Cond == nil {
				return true
			}
			// Already a literal? Nothing to suppress.
			if id, ok := ifs.Cond.(*ast.Ident); ok && (id.Name == "false" || id.Name == "true") {
				return true
			}
			bodyFrom := fset.Position(ifs.Body.Pos()).Offset
			bodyTo := fset.Position(ifs.Body.End()).Offset
			if bodyFrom < 0 || bodyTo > len(src) {
				return true
			}
			body := string(src[bodyFrom:bodyTo])
			if !strings.Contains(body, "return") {
				return true
			}
			if !strings.Contains(body, "fmt.Errorf") && !strings.Contains(body, "errors.New") {
				return true
			}
			cf := fset.Position(ifs.Cond.Pos()).Offset
			ct := fset.Position(ifs.Cond.End()).Offset
			out = append(out, guard{
				pkg: pkg, file: p, line: fset.Position(ifs.Cond.Pos()).Line,
				condFrom: cf, condTo: ct,
				condText: string(src[cf:ct]),
				snippet:  firstReturn(body),
			})
			return true
		})
		return nil
	})
	return out, err
}

func runTests(mod, pkg, timeout string) (bool, string) {
	target := "./" + pkg + "/"
	if pkg == "" {
		target = "."
	}
	cmd := exec.Command("go", "test", target, "-count=1", "-timeout", timeout)
	cmd.Dir = mod
	// Hermetic: live proofs must skip, or a model call would dominate
	// every run and make timings meaningless.
	cmd.Env = append(os.Environ(), "THEMIS_LIVE_OLLAMA=http://127.0.0.1:9")
	b, err := cmd.CombinedOutput()
	return err == nil, string(b)
}

func firstReturn(body string) string {
	for _, l := range strings.Split(body, "\n") {
		t := strings.TrimSpace(l)
		if strings.HasPrefix(t, "return") {
			return t
		}
	}
	return strings.TrimSpace(body)
}

func oneLine(s string) string {
	s = strings.Join(strings.Fields(s), " ")
	if len(s) > 120 {
		return s[:120] + "…"
	}
	return s
}

func rel(base, p string) string {
	r, err := filepath.Rel(base, p)
	if err != nil {
		return p
	}
	return r
}

func tail(s string, n int) string {
	lines := strings.Split(strings.TrimRight(s, "\n"), "\n")
	if len(lines) > n {
		lines = lines[len(lines)-n:]
	}
	return strings.Join(lines, "\n")
}

func bail(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "\nthemis-mutate: "+format+"\n", args...)
	os.Exit(2)
}
