package context

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// CreateMode confinement: branch-asserted against the Q-L5-4 locked
// rules, including the escape that motivated the mode split.
func TestConfineCreatePath(t *testing.T) {
	root := t.TempDir()
	outside := t.TempDir()
	mustMkdir := func(p string) {
		t.Helper()
		if err := os.MkdirAll(p, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	mustWrite := func(p string) {
		t.Helper()
		if err := os.WriteFile(p, []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	mustLink := func(target, link string) {
		t.Helper()
		if err := os.Symlink(target, link); err != nil {
			t.Fatal(err)
		}
	}
	mustMkdir(filepath.Join(root, "pkg"))
	mustMkdir(filepath.Join(root, "internal"))
	mustWrite(filepath.Join(root, "pkg", "main.go"))
	// The motivating escape: symlinked parent + nonexistent target.
	mustLink(outside, filepath.Join(root, "evil"))
	// A symlink resolving INSIDE the workspace — still refused.
	mustLink(filepath.Join(root, "internal"), filepath.Join(root, "inlink"))
	// A symlinked final target and a plain directory target.
	mustLink(filepath.Join(root, "pkg", "main.go"), filepath.Join(root, "pkg", "alias.go"))

	refuse := []struct{ name, rel, wantErr string }{
		{"absolute", "/etc/passwd", ""},
		{"empty", "", ""},
		{"dotdot-escape", "../x", ""},
		{"root-itself", ".", "cannot be the confinement root"},
		{"git-dir", ".git/config", ".git* is never a mutation target"},
		{"gitignore", ".gitignore", ".git*"},
		{"github-nested", "a/.github/w.yml", ".git*"},
		// Case-fold regression (M2/M3 security review HIGH): on
		// case-insensitive filesystems .GIT IS .git.
		{"git-upper", ".GIT/config", ".git*"},
		{"git-mixed", ".Git/hooks/pre-commit", ".git*"},
		{"github-upper-nested", "a/.GitHub/w.yml", ".git*"},
		{"symlink-parent-outside", "evil/payload.go", "symlink in write path"},
		{"symlink-parent-inside", "inlink/f.go", "symlink in write path"},
		{"missing-parent", "new/dir/f.go", "parent chain must exist"},
		{"parent-is-file", "pkg/main.go/x", "not a directory"},
		{"target-symlink", "pkg/alias.go", "target is a symlink"},
		{"target-directory", "pkg", "target is a directory"},
	}
	for _, c := range refuse {
		_, err := ConfineCreatePath(root, c.rel)
		if err == nil || !errors.Is(err, ErrConfinement) || !strings.Contains(err.Error(), c.wantErr) {
			t.Errorf("%s: want ErrConfinement with %q, got %v", c.name, c.wantErr, err)
		}
	}

	// Legitimate creation and overwrite. Paths come back under the
	// canonicalized root (macOS /var -> /private/var).
	rootResolved, err := filepath.EvalSymlinks(root)
	if err != nil {
		t.Fatal(err)
	}
	p, err := ConfineCreatePath(root, "pkg/new.go")
	if err != nil || p != filepath.Join(rootResolved, "pkg", "new.go") {
		t.Fatalf("new file under verified parents must pass: %q %v", p, err)
	}
	p, err = ConfineCreatePath(root, "pkg/main.go")
	if err != nil || p != filepath.Join(rootResolved, "pkg", "main.go") {
		t.Fatalf("overwrite of a regular file must pass: %q %v", p, err)
	}
	// Interior ".." that stays contained collapses lexically and then
	// obeys the same rules.
	if _, err := ConfineCreatePath(root, "pkg/../internal/ok.go"); err != nil {
		t.Fatalf("contained interior ..: %v", err)
	}
	// No confinement root: fail closed.
	if _, err := ConfineCreatePath("", "x"); !errors.Is(err, ErrConfinement) {
		t.Fatal("empty root must refuse")
	}
	// Phantom root: fail closed.
	if _, err := ConfineCreatePath(filepath.Join(root, "nope"), "x"); !errors.Is(err, ErrConfinement) {
		t.Fatal("nonexistent root must refuse")
	}
}

// ResolveMode and CreateMode must disagree exactly where their
// existence semantics differ — the documented split, not drift.
func TestModeSplitIsDeliberate(t *testing.T) {
	root := t.TempDir()
	outside := t.TempDir()
	if err := os.Symlink(outside, filepath.Join(root, "evil")); err != nil {
		t.Fatal(err)
	}
	// ResolveMode's lexical fallback admits the nonexistent path under
	// a symlinked parent (read semantics: the open will fail);
	// CreateMode refuses it (write semantics: the create would land
	// outside). This asymmetry is the reason CreateMode exists.
	if _, err := ConfinePath(root, "evil/ghost.go"); err != nil {
		t.Fatalf("resolve-mode lexical fallback changed — revisit the mode split: %v", err)
	}
	if _, err := ConfineCreatePath(root, "evil/ghost.go"); !errors.Is(err, ErrConfinement) {
		t.Fatal("create-mode must refuse the symlinked-parent escape")
	}
}
