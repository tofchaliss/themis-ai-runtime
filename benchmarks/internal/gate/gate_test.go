package gate

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeScore(t *testing.T, root, date, model, benchmark string, score int) {
	t.Helper()

	dir := filepath.Join(root, "validation", date, model)
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}

	content := fmt.Sprintf(
		`{"benchmark": %q, "score": %d, "missing": [], "violations": []}`,
		benchmark,
		score,
	)

	if err := os.WriteFile(
		filepath.Join(dir, benchmark+".json"),
		[]byte(content),
		0644,
	); err != nil {
		t.Fatal(err)
	}
}

func TestGate(t *testing.T) {

	t.Run("improvement passes", func(t *testing.T) {
		root := t.TempDir()
		writeScore(t, root, "2026-08-01", "m", "B001", 50)
		writeScore(t, root, "2026-08-02", "m", "B001", 80)

		r, err := Compare(root, "m", "2026-08-01", "2026-08-02")
		if err != nil {
			t.Fatal(err)
		}

		if !r.Pass(0) {
			t.Errorf("expected pass, got %+v", r)
		}
		if r.AverageDrop() != -30 {
			t.Errorf("AverageDrop = %d, want -30", r.AverageDrop())
		}
	})

	t.Run("drop within tolerance passes", func(t *testing.T) {
		root := t.TempDir()
		writeScore(t, root, "2026-08-01", "m", "B001", 80)
		writeScore(t, root, "2026-08-02", "m", "B001", 76)

		r, err := Compare(root, "m", "2026-08-01", "2026-08-02")
		if err != nil {
			t.Fatal(err)
		}

		if !r.Pass(5) {
			t.Error("drop of 4 should pass with max-drop 5")
		}
		if r.Pass(3) {
			t.Error("drop of 4 should fail with max-drop 3")
		}
	})

	t.Run("missing benchmark always fails", func(t *testing.T) {
		root := t.TempDir()
		writeScore(t, root, "2026-08-01", "m", "B001", 50)
		writeScore(t, root, "2026-08-01", "m", "B002", 50)
		writeScore(t, root, "2026-08-02", "m", "B001", 100)

		r, err := Compare(root, "m", "2026-08-01", "2026-08-02")
		if err != nil {
			t.Fatal(err)
		}

		if r.Pass(100) {
			t.Error("missing benchmark must fail regardless of tolerance")
		}
		if len(r.Missing) != 1 || r.Missing[0] != "B002" {
			t.Errorf("Missing = %v, want [B002]", r.Missing)
		}
	})

	t.Run("new benchmarks are informational", func(t *testing.T) {
		root := t.TempDir()
		writeScore(t, root, "2026-08-01", "m", "B001", 80)
		writeScore(t, root, "2026-08-02", "m", "B001", 80)
		writeScore(t, root, "2026-08-02", "m", "B002", 0)

		r, err := Compare(root, "m", "2026-08-01", "2026-08-02")
		if err != nil {
			t.Fatal(err)
		}

		// Average fell 80 -> 40 because of the new benchmark; that is a
		// real regression signal and still gated.
		if r.Pass(5) {
			t.Error("expected fail: average dropped 40 points")
		}
		if len(r.Added) != 1 || r.Added[0] != "B002" {
			t.Errorf("Added = %v, want [B002]", r.Added)
		}
	})

	t.Run("missing validation data is an error", func(t *testing.T) {
		root := t.TempDir()
		writeScore(t, root, "2026-08-02", "m", "B001", 50)

		if _, err := Compare(root, "m", "2026-08-01", "2026-08-02"); err == nil {
			t.Error("expected error for missing baseline data")
		}
	})

	t.Run("report renders pass and fail", func(t *testing.T) {
		root := t.TempDir()
		writeScore(t, root, "2026-08-01", "m", "B001", 80)
		writeScore(t, root, "2026-08-02", "m", "B001", 20)

		r, err := Compare(root, "m", "2026-08-01", "2026-08-02")
		if err != nil {
			t.Fatal(err)
		}

		text := Report(r, 5)
		if !strings.Contains(text, "FAIL") || !strings.Contains(text, "▼ B001: 80% -> 20%") {
			t.Errorf("report:\n%s", text)
		}
	})
}
