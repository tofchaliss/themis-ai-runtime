package integration

// The two walls that survive the dissolution of the src/themis stand-in
// (I-M4, D-I-7): (1) no harness package, whole graph, imports the Themis
// repository — the Themis-facing seam is HTTP, never a Go dependency;
// (2) the capability wall — no anchored registry declares a capability
// that could write, decide, or classify a Position, and only the two
// Themis read capabilities mint governed-record.

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

const (
	harnessModule = "github.com/tofchaliss/themis-ai-runtime/src/harness"
	themisRepo    = "github.com/themis-project/themis"
)

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

func TestWall1HarnessGraphReachesNoThemis(t *testing.T) {
	deps := goList(t, "..", "-deps", "./...")
	if len(deps) < 50 {
		t.Fatalf("suspiciously few harness deps (%d)", len(deps))
	}
	for _, d := range deps {
		if strings.HasPrefix(d, themisRepo) || strings.Contains(d, "themis-app") {
			t.Errorf("harness dependency graph reaches %s — the harness never imports Themis (D-I-7)", d)
		}
	}
}

// Wall 4 — capability wall: no registry the deployment could anchor
// declares a capability whose NAME, target class, or trust could
// write, decide, or classify a Position; and the harness executor
// vocabulary has no such verb. Checked over the decoded registrations,
// never over prose (a description saying "decides what it means" is
// not a capability).
func TestWall4NoPositionCapability(t *testing.T) {
	entries, err := filepath.Glob(filepath.Join(repoRoot, "policies", "tools", "registry-v*.json"))
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
	b, err := os.ReadFile(filepath.Join("..", "tools", "execute.go"))
	if err != nil {
		t.Fatal(err)
	}
	for _, bad := range []string{`"write_position"`, `"set_position"`, `"decide"`, `"accept_finding"`, `"write_finding"`} {
		if strings.Contains(string(b), bad) {
			t.Errorf("harness executor table carries %s", bad)
		}
	}
}
