package skills

// Regressions from the L9 Class-3 security and architecture reviews.
// Each test pins one remediated finding to its branch.

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Security HIGH-1: an unvalidated task id reached filepath.Join and
// could overwrite sibling governed artifacts. The id is now held to
// L6's own predicate, and every write resolves through the canonical
// mutation predicate.
func TestTraversingTaskIDRefused(t *testing.T) {
	b := newBundle(t)
	victimDir := t.TempDir()
	victim := filepath.Join(victimDir, "victim-grant.json")
	if err := os.WriteFile(victim, []byte(`{"governed":"artifact"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	for _, bad := range []string{
		"../victim", "../../etc/passwd", "a/b", ".hidden", "",
		strings.Repeat("x", 129), "task id with spaces", "task\x00id",
	} {
		req := b.request(t)
		req.TaskID = bad
		req.OutDir = victimDir
		if _, err := Instantiate(b.catalogPath, "investigate-cve@1", req); !errors.Is(err, ErrResolve) {
			t.Fatalf("task_id %q must refuse before any write: %v", bad, err)
		}
	}
	// The victim artifact is untouched: the refusal precedes the write.
	got, err := os.ReadFile(victim)
	if err != nil || string(got) != `{"governed":"artifact"}` {
		t.Fatalf("a refused instantiation must not have written anything: %q %v", got, err)
	}
}

// Security HIGH-1 companion: retry_of also names a task identity and
// is held to the same predicate.
func TestInvalidRetryOfRefused(t *testing.T) {
	b := newBundle(t)
	req := b.request(t)
	req.RetryOf = "../elsewhere"
	if _, err := Instantiate(b.catalogPath, "investigate-cve@1", req); !errors.Is(err, ErrResolve) ||
		!strings.Contains(err.Error(), "retry_of") {
		t.Fatalf("an invalid retry_of must refuse: %v", err)
	}
}

// Security MED-1 / architecture HIGH: wall 2 was satisfiable by
// supplying only one root. Every task-writable root is now required —
// the workspace root especially, since that is the one a task holds
// write_file on.
func TestDisjointnessRequiresEveryWritableRoot(t *testing.T) {
	b := newBundle(t)
	for _, missing := range []string{"state", "artifacts", "workspace"} {
		req := b.request(t)
		switch missing {
		case "state":
			req.Deployment.StateRoot = ""
		case "artifacts":
			req.Deployment.ArtifactDir = ""
		case "workspace":
			req.Deployment.WorkspaceRoot = ""
		}
		if _, err := Instantiate(b.catalogPath, "investigate-cve@1", req); !errors.Is(err, ErrCatalog) ||
			!strings.Contains(err.Error(), "required to prove catalog disjointness") {
			t.Fatalf("omitting the %s root must refuse — the wall must not pass by luck: %v", missing, err)
		}
	}
	// And the workspace-collides case specifically: the catalog inside
	// the root a task can write to.
	req := b.request(t)
	req.Deployment.WorkspaceRoot = b.catalogDir
	if _, err := Instantiate(b.catalogPath, "investigate-cve@1", req); !errors.Is(err, ErrCatalog) ||
		!strings.Contains(err.Error(), "disjoint") {
		t.Fatalf("a catalog inside the workspace must refuse: %v", err)
	}
}
