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

	t.Run("verdict records a pass", func(t *testing.T) {
		root := t.TempDir()
		writeScore(t, root, "2026-08-01", "org/m", "B001", 50)
		writeScore(t, root, "2026-08-02", "org/m", "B001", 80)

		r, err := Compare(root, "org/m", "2026-08-01", "2026-08-02")
		if err != nil {
			t.Fatal(err)
		}

		path, err := WriteVerdict(root, r, 0)
		if err != nil {
			t.Fatal(err)
		}

		// Literal layout, not VerdictPath: the location is a contract
		// with service.gatePassed, which cannot import this package.
		want := filepath.Join(root, "gate", "2026-08-02", "org/m", "verdict.json")
		if path != want {
			t.Errorf("verdict path = %s, want %s", path, want)
		}

		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		for _, s := range []string{`"pass": true`, `"model": "org/m"`, `"baseline": "2026-08-01"`, `"current": "2026-08-02"`} {
			if !strings.Contains(string(data), s) {
				t.Errorf("verdict missing %s:\n%s", s, data)
			}
		}

		digest, err := RunDigest(filepath.Join(root, "validation", "2026-08-02", "org/m"))
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(string(data), fmt.Sprintf(`"scores_digest": %q`, digest)) {
			t.Errorf("verdict must fingerprint the judged scores:\n%s", data)
		}
	})

	t.Run("verdict refuses path-escaping model names", func(t *testing.T) {
		root := t.TempDir()

		for _, model := range []string{"../evil", "a/../../b", "/abs", ""} {
			if _, err := WriteVerdict(root, Result{Model: model, Current: "2026-08-02"}, 0); err == nil {
				t.Errorf("model %q must be rejected", model)
			}
		}
	})

	t.Run("baseline must be an admitted run", func(t *testing.T) {
		root := t.TempDir()
		writeScore(t, root, "2026-08-01", "m", "B001", 90)
		writeScore(t, root, "2026-08-02", "m", "B001", 10)
		writeScore(t, root, "2026-08-03", "m", "B001", 12)

		// Bootstrap: no verdicts yet, self-baseline allowed...
		if err := CheckBaseline(root, "m", "2026-08-01", "2026-08-01"); err != nil {
			t.Errorf("bootstrap self-baseline refused: %v", err)
		}
		// ...but a non-self baseline is not, even with no verdicts.
		if err := CheckBaseline(root, "m", "2026-08-01", "2026-08-02"); err == nil {
			t.Error("ungated baseline accepted with no bootstrap verdict")
		}

		// Admit day 1, then record day 2's regression as a failure.
		r1, err := Compare(root, "m", "2026-08-01", "2026-08-01")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := WriteVerdict(root, r1, 0); err != nil {
			t.Fatal(err)
		}
		r2, err := Compare(root, "m", "2026-08-01", "2026-08-02")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := WriteVerdict(root, r2, 0); err != nil {
			t.Fatal(err)
		}

		// The failed day 2 must not baseline day 3 (stepwise regression).
		if err := CheckBaseline(root, "m", "2026-08-02", "2026-08-03"); err == nil {
			t.Error("failed baseline accepted: ratchet can be walked down")
		}
		// Self-baseline is bootstrap-only: refused once any verdict exists.
		if err := CheckBaseline(root, "m", "2026-08-03", "2026-08-03"); err == nil {
			t.Error("self-baseline accepted after bootstrap")
		}
		// The admitted day 1 is a valid baseline.
		if err := CheckBaseline(root, "m", "2026-08-01", "2026-08-03"); err != nil {
			t.Errorf("admitted baseline refused: %v", err)
		}
	})

	t.Run("verdict records a fail", func(t *testing.T) {
		root := t.TempDir()
		writeScore(t, root, "2026-08-01", "m", "B001", 80)
		writeScore(t, root, "2026-08-02", "m", "B001", 20)

		r, err := Compare(root, "m", "2026-08-01", "2026-08-02")
		if err != nil {
			t.Fatal(err)
		}

		path, err := WriteVerdict(root, r, 5)
		if err != nil {
			t.Fatal(err)
		}

		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(string(data), `"pass": false`) {
			t.Errorf("failing gate must record pass=false:\n%s", data)
		}
	})

	t.Run("self-baseline bootstrap passes", func(t *testing.T) {
		root := t.TempDir()
		writeScore(t, root, "2026-08-01", "m", "B001", 50)

		r, err := Compare(root, "m", "2026-08-01", "2026-08-01")
		if err != nil {
			t.Fatal(err)
		}

		if !r.Pass(0) {
			t.Error("a run compared against itself must pass")
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
