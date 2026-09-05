package instructions

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

// The secret scanner needs must-pass evidence as well as must-reject
// (the Q-L1-2 dual-corpus discipline applied to the secret family):
// benign hyphenated prose must never abort a load.
func TestSecretScanMustPass(t *testing.T) {
	benign := []string{
		"Follow the risk-assessment-procedures checklist.\n",
		"Apply the task-prioritization-guidelines here.\n",
		"Record desk-side-review-notes-for-later in the trace.\n",
		"AKIAB is a prefix, not a key.\n",
		"api_key: reference it by environment-variable name only.\n",
	}
	for _, body := range benign {
		src := safetySource(t, map[string]string{
			"x.md": file("harness.safety.x", "harness-safety", "safety", body),
		})
		if _, err := Load(testConfig(t), src); err != nil {
			t.Errorf("benign body aborted load: %q: %v", body, err)
		}
	}
}

// Mirror of TestLoadTrustedFailuresAbort over the untrusted task
// inline path: everything except a foreign-namespace claim aborts
// (the Q-L1-6 dividing line — structural invalidity of the payload is
// an intake failure, never silent degradation).
func TestLoadUntrustedFailuresAbort(t *testing.T) {
	inline := func(inst Instruction) Source {
		return Source{Kind: ScopeTask, Inline: []Instruction{inst}}
	}
	ok := Instruction{ID: "task.instructions", Scope: ScopeTask, Category: CategoryTask, Body: "Analyze.\n"}
	cases := []struct {
		name    string
		src     Source
		wantErr error
	}{
		{"secret in body", inline(Instruction{ID: "task.instructions", Scope: ScopeTask, Category: CategoryTask,
			Body: "token AKIAABCDEFGHIJKLMNOP\n"}), ErrSecretDetected},
		{"empty body", inline(Instruction{ID: "task.instructions", Scope: ScopeTask, Category: CategoryTask,
			Body: " \n"}), ErrEmptyBody},
		{"oversized body", inline(Instruction{ID: "task.instructions", Scope: ScopeTask, Category: CategoryTask,
			Body: strings.Repeat("a", MaxInstructionBytes+1)}), ErrTooLarge},
		{"bad category", inline(Instruction{ID: "task.instructions", Scope: ScopeTask, Category: "tsk",
			Body: "b\n"}), ErrMalformed},
		{"scope mismatch", inline(Instruction{ID: "task.instructions", Scope: ScopeHarnessSafety, Category: CategoryTask,
			Body: "b\n"}), ErrScopeMismatch},
		{"unknown namespace", inline(Instruction{ID: "plugin.foo.x", Scope: ScopeTask, Category: CategoryTask,
			Body: "b\n"}), ErrUnknownNamespace},
		{"bad id syntax", inline(Instruction{ID: "Task.Instructions", Scope: ScopeTask, Category: CategoryTask,
			Body: "b\n"}), ErrBadID},
		{"protected declaration", inline(Instruction{ID: "task.instructions", Scope: ScopeTask, Category: CategoryTask,
			Protected: true, Body: "b\n"}), ErrProtectedDeclaration},
		{"duplicate ids in payload", Source{Kind: ScopeTask, Inline: []Instruction{ok, ok}}, ErrDuplicateID},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			res, err := Load(testConfig(t), tc.src)
			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("want %v, got %v", tc.wantErr, err)
			}
			if res != nil {
				t.Fatalf("abort must yield no result, got %+v", res)
			}
		})
	}
}

// Partial source availability never yields a result: a healthy trusted
// source followed by a failing one produces nil, not the healthy half.
func TestLoadMultiSourceAtomicity(t *testing.T) {
	good := safetySource(t, map[string]string{
		"a.md": file("harness.safety.a", "harness-safety", "safety", "A.\n"),
	})
	cases := []struct {
		name    string
		second  Source
		wantErr error
	}{
		{"missing root", Source{Kind: ScopeThemisDomain, Root: filepath.Join(t.TempDir(), "absent")}, ErrSourceUnavailable},
		{"empty root", Source{Kind: ScopeThemisDomain, Root: t.TempDir()}, ErrSourceUnavailable},
		{"malformed file", func() Source {
			dir := t.TempDir()
			writeFile(t, dir, "x.md", "not frontmatter")
			return Source{Kind: ScopeThemisDomain, Root: dir}
		}(), ErrMalformed},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			res, err := Load(testConfig(t), good, tc.second)
			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("want %v, got %v", tc.wantErr, err)
			}
			if res != nil {
				t.Fatalf("partial availability must yield no result, got %+v", res)
			}
		})
	}
}

// Caller-supplied provenance on inline instructions is overwritten:
// BodyHash and SourceRef are computed by the loader, never trusted
// from the payload.
func TestInlineProvenanceOverwritten(t *testing.T) {
	task := Source{Kind: ScopeTask, Inline: []Instruction{{
		ID: "task.instructions", Scope: ScopeTask, Category: CategoryTask,
		Body: "Analyze.\n", BodyHash: "forged", SourceRef: "forged",
	}}}
	res, err := Load(testConfig(t), task)
	if err != nil {
		t.Fatal(err)
	}
	got := res.Instructions[0]
	if got.BodyHash != bodyHash("Analyze.\n") || got.SourceRef != "inline:task[0]" {
		t.Fatalf("forged provenance survived: %+v", got)
	}
}

// The conflict record's hash is the real content hash: in the
// byte-identical shadow case it equals the surviving trusted
// instruction's BodyHash — content confers no ownership, and the
// record proves which bytes were rejected.
func TestConflictRecordsRealBodyHash(t *testing.T) {
	body := "Verification is mandatory.\n"
	src := safetySource(t, map[string]string{
		"verify.md": file("harness.safety.verify", "harness-safety", "safety", body),
	})
	task := Source{Kind: ScopeTask, Inline: []Instruction{
		{ID: "harness.safety.verify", Scope: ScopeTask, Category: CategoryTask, Body: body},
		{ID: "task.instructions", Scope: ScopeTask, Category: CategoryTask, Body: "Analyze.\n"},
	}}
	res, err := Load(testConfig(t), src, task)
	if err != nil {
		t.Fatal(err)
	}
	if res.Conflicts[0].BodyHash != bodyHash(body) || res.Conflicts[0].BodyHash != res.Instructions[0].BodyHash {
		t.Fatalf("conflict hash must be the real content hash: %+v", res.Conflicts[0])
	}
}

// The seven-scope precedence order is fixed forever (design D-L1-2);
// reordering the enum is a breaking change this test makes loud.
func TestScopeOrderFixedForever(t *testing.T) {
	order := []struct {
		scope Scope
		name  string
	}{
		{ScopeHarnessSafety, "harness-safety"},
		{ScopeHarnessSystem, "harness-system"},
		{ScopeThemisDomain, "themis-domain"},
		{ScopeRepository, "repository"},
		{ScopeDirectory, "directory"},
		{ScopeSkill, "skill"},
		{ScopeTask, "task"},
	}
	for i, entry := range order {
		if int(entry.scope) != i {
			t.Fatalf("scope %s must rank %d, got %d", entry.name, i, int(entry.scope))
		}
		if entry.scope.String() != entry.name {
			t.Fatalf("String() drift: %s vs %s", entry.scope, entry.name)
		}
		parsed, err := ParseScope(entry.name)
		if err != nil || parsed != entry.scope {
			t.Fatalf("ParseScope round-trip failed for %s: %v", entry.name, err)
		}
	}
}

// Full-result determinism: identical inputs give deeply equal results
// across runs, not merely the same ID order.
func TestLoadFullResultDeterminism(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "b.md", file("harness.safety.bravo", "harness-safety", "safety", "B body.\n"))
	if err := os.MkdirAll(filepath.Join(dir, "sub"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "sub", "a.md"),
		[]byte(file("harness.safety.alpha", "harness-safety", "safety", "A body.\n")), 0o644); err != nil {
		t.Fatal(err)
	}
	writeFile(t, dir, "notes.txt", "ignored — not an instruction file")
	src := Source{Kind: ScopeHarnessSafety, Root: dir}
	task := Source{Kind: ScopeTask, Inline: []Instruction{
		{ID: "harness.safety.bravo", Scope: ScopeTask, Category: CategoryTask, Body: "shadow attempt\n"},
		{ID: "task.instructions", Scope: ScopeTask, Category: CategoryTask, Body: "Analyze.\n"},
	}}
	first, err := Load(testConfig(t), src, task)
	if err != nil {
		t.Fatal(err)
	}
	// Recursion into sub/ is intended (every *.md under each root),
	// non-.md files are ignored, and order is full-path sorted:
	// b.md < sub/a.md.
	if len(first.Instructions) != 3 ||
		first.Instructions[0].ID != "harness.safety.bravo" ||
		first.Instructions[1].ID != "harness.safety.alpha" {
		t.Fatalf("walk/order wrong: %+v", first.Instructions)
	}
	for run := 0; run < 3; run++ {
		again, err := Load(testConfig(t), src, task)
		if err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(first, again) {
			t.Fatalf("results differ across runs:\n%+v\n%+v", first, again)
		}
	}
}

func TestFrontmatterLineMalformations(t *testing.T) {
	cases := []struct {
		name    string
		content string
		wantErr error
	}{
		{"no separator", "---\nidharness\n---\nb\n", ErrMalformed},
		{"colon without space", "---\nid:harness.safety.x\n---\nb\n", ErrMalformed},
		{"empty value", "---\nid: \n---\nb\n", ErrMalformed},
		{"leading space in key", "---\n id: harness.safety.x\n---\nb\n", ErrMalformed},
		{"empty frontmatter block", "---\n---\nb\n", ErrMalformed},
		{"crlf file", "---\r\nid: harness.safety.x\r\n---\r\nb\r\n", ErrMalformed},
		{"closing delimiter at EOF", "---\nid: harness.safety.x\nscope: harness-safety\ncategory: safety\n---", ErrMalformed},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			src := safetySource(t, map[string]string{"x.md": tc.content})
			if _, err := Load(testConfig(t), src); !errors.Is(err, tc.wantErr) {
				t.Fatalf("want %v, got %v", tc.wantErr, err)
			}
		})
	}
}

func TestLoadEdgeCasesPinned(t *testing.T) {
	// protected: false is legal and explicit.
	src := safetySource(t, map[string]string{
		"x.md": file("harness.safety.x", "harness-safety", "safety", "b\n", "protected: false"),
	})
	res, err := Load(testConfig(t), src)
	if err != nil || res.Instructions[0].Protected {
		t.Fatalf("protected: false must load unprotected: %v %+v", err, res)
	}
	// Exactly the cap passes; the cap is inclusive.
	atCap := safetySource(t, map[string]string{
		"y.md": file("harness.safety.y", "harness-safety", "safety", strings.Repeat("a", MaxInstructionBytes)),
	})
	if _, err := Load(testConfig(t), atCap); err != nil {
		t.Fatalf("body at exactly MaxInstructionBytes must pass: %v", err)
	}
	// A body containing "---" lines stays byte-exact.
	body := "keep\n---\nthis verbatim\n"
	verbatim := safetySource(t, map[string]string{
		"z.md": file("harness.safety.z", "harness-safety", "safety", body),
	})
	res, err = Load(testConfig(t), verbatim)
	if err != nil || res.Instructions[0].Body != body {
		t.Fatalf("body with --- lines not byte-exact: %v %q", err, res.Instructions[0].Body)
	}
	// Unreadable file aborts as unavailable.
	dir := t.TempDir()
	writeFile(t, dir, "x.md", file("harness.safety.q", "harness-safety", "safety", "b\n"))
	if err := os.Chmod(filepath.Join(dir, "x.md"), 0o000); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(filepath.Join(dir, "x.md"), 0o644) })
	if _, err := Load(testConfig(t), Source{Kind: ScopeHarnessSafety, Root: dir}); !errors.Is(err, ErrSourceUnavailable) {
		t.Fatalf("unreadable file must abort as unavailable, got %v", err)
	}
	// Inline on a non-task source without Root: rejected for the right reason.
	bad := Source{Kind: ScopeHarnessSafety, Inline: []Instruction{{ID: "harness.safety.i", Scope: ScopeHarnessSafety, Category: CategorySafety, Body: "b\n"}}}
	if _, err := Load(testConfig(t), bad); !errors.Is(err, ErrUnrecognizedSource) {
		t.Fatalf("inline non-task source must be rejected: %v", err)
	}
}
