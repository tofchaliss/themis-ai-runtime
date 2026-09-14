package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

func writeEnv(t *testing.T, dir string, v map[string]any) string {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	p := filepath.Join(dir, "request.json")
	if err := os.WriteFile(p, b, 0o600); err != nil {
		t.Fatal(err)
	}
	return p
}

// The submitted envelope carries the submitting process's OBSERVED
// origin, and the record therefore says which account on which host
// submitted the task.
func TestStampOriginObserves(t *testing.T) {
	dir := t.TempDir()
	req := writeEnv(t, dir, map[string]any{"version": 1, "task_id": "t-obs"})

	out, err := stampOrigin(req, filepath.Join(dir, "submissions"))
	if err != nil {
		t.Fatalf("stamp: %v", err)
	}
	var env map[string]any
	raw, err := os.ReadFile(out)
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(raw, &env); err != nil {
		t.Fatal(err)
	}
	origin, ok := env["origin"].(map[string]any)
	if !ok {
		t.Fatalf("submitted envelope carries no origin: %v", env)
	}
	if got := origin["submitter_uid"]; got != strconv.Itoa(os.Getuid()) {
		t.Errorf("submitter_uid = %v, want the invoking uid %d", got, os.Getuid())
	}
	if origin["submitter_host"] == "" {
		t.Error("submitter_host is empty — origin must name the host")
	}
	// The request must survive untouched apart from the stamp.
	if env["task_id"] != "t-obs" {
		t.Errorf("task_id was rewritten: %v", env["task_id"])
	}
}

// A request may NOT assert its own submitter identity. If it could, the
// record would carry a claim rather than an observation — and a claim
// about who submitted is exactly the thing this must never be.
func TestStampOriginRefusesForgedSubmitter(t *testing.T) {
	for _, key := range []string{"submitter_uid", "submitter_user", "submitter_host"} {
		dir := t.TempDir()
		req := writeEnv(t, dir, map[string]any{
			"version": 1, "task_id": "t-forge",
			"origin": map[string]any{key: "root"},
		})
		_, err := stampOrigin(req, filepath.Join(dir, "submissions"))
		if err == nil {
			t.Fatalf("%s: a request asserting its own submitter origin was ACCEPTED", key)
		}
		if !strings.Contains(err.Error(), "never asserted by the request") {
			t.Fatalf("%s: refused for the wrong reason: %v", key, err)
		}
	}
}

// L9's skill attribution rides the same map. Stamping must add keys,
// never rewrite or drop them — a submission that silently lost its
// skill attribution would be a different governed task.
func TestStampOriginPreservesSkillAttribution(t *testing.T) {
	dir := t.TempDir()
	req := writeEnv(t, dir, map[string]any{
		"version": 1, "task_id": "t-skill",
		"origin": map[string]any{
			"skill":         "remediate-dependency@1",
			"skill_version": "1",
		},
	})
	out, err := stampOrigin(req, filepath.Join(dir, "submissions"))
	if err != nil {
		t.Fatalf("stamp: %v", err)
	}
	raw, _ := os.ReadFile(out)
	var env map[string]any
	if err := json.Unmarshal(raw, &env); err != nil {
		t.Fatal(err)
	}
	origin := env["origin"].(map[string]any)
	if origin["skill"] != "remediate-dependency@1" {
		t.Errorf("skill attribution lost or rewritten: %v", origin)
	}
	if origin["skill_version"] != "1" {
		t.Errorf("skill_version lost or rewritten: %v", origin)
	}
	if origin["submitter_uid"] == nil {
		t.Error("stamp did not apply alongside the skill attribution")
	}
}

// Origin values are bounded by the envelope validator (64-byte keys,
// 256-byte values). The stamp must not be the thing that breaches them.
func TestStampOriginValuesAreBounded(t *testing.T) {
	o := SubmitterOrigin(strings.Repeat("h", 4096))
	for k, v := range o {
		if len(k) > 64 {
			t.Errorf("origin key %q exceeds the 64-byte bound", k)
		}
		if len(v) > 256 {
			t.Errorf("origin value for %q is %d bytes, over the 256-byte bound", k, len(v))
		}
	}
}

// A submission with no task_id cannot be named, stored, or later
// attributed — refuse rather than invent one.
func TestStampOriginRequiresTaskID(t *testing.T) {
	dir := t.TempDir()
	req := writeEnv(t, dir, map[string]any{"version": 1})
	if _, err := stampOrigin(req, filepath.Join(dir, "submissions")); err == nil {
		t.Fatal("an envelope with no task_id was accepted")
	}
}
