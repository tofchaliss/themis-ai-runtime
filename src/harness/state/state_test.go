package state

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func testRoot(t *testing.T) *Root {
	t.Helper()
	r, err := OpenRoot(filepath.Join(t.TempDir(), "state"))
	if err != nil {
		t.Fatal(err)
	}
	return r
}

func body(s string) json.RawMessage { return json.RawMessage(`{"v":` + `"` + s + `"}`) }

// --- Register A: structural -----------------------------------------

// The state package exposes no generic durable write and no deletion
// path: source-scan proof (Q-L6-9 Register A; deletion-absence is the
// v1 GC-cannot-delete-reachable proof).
func TestNoDeletionPathStructural(t *testing.T) {
	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".go") || strings.HasSuffix(e.Name(), "_test.go") {
			continue
		}
		b, err := os.ReadFile(e.Name())
		if err != nil {
			t.Fatal(err)
		}
		src := string(b)
		if strings.Contains(src, "os.RemoveAll(") {
			t.Errorf("%s contains os.RemoveAll — no deletion path may exist in v1", e.Name())
		}
		// os.Remove is legal ONLY as staging-temp cleanup, pinned to
		// the exact line form; os.Truncate is legal ONLY at recovery's
		// frontier cut (test review: the scan must pin the one
		// primitive that can cut committed-stream bytes).
		for _, line := range strings.Split(src, "\n") {
			trimmed := strings.TrimSpace(line)
			if strings.Contains(trimmed, "os.Remove(") && trimmed != "defer os.Remove(tmpName) // no-op after successful publication cleanup" && trimmed != "defer os.Remove(tmpName)" {
				t.Errorf("%s: os.Remove outside pinned staging cleanup: %q", e.Name(), trimmed)
			}
			if strings.Contains(trimmed, "os.Truncate(") && e.Name() != "recovery.go" {
				t.Errorf("%s: os.Truncate outside recovery's frontier cut: %q", e.Name(), trimmed)
			}
		}
	}
}

// Closed vocabularies refuse the unknown; the lifecycle machine is
// proven over its full edge product.
func TestClosedVocabularies(t *testing.T) {
	r := testRoot(t)
	if _, err := r.Store().StoreObject("secrets", []byte("x")); !errors.Is(err, ErrConstitution) {
		t.Fatalf("unknown object class must refuse: %v", err)
	}
	tr, err := r.CreateTask("t-vocab", TaskOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := tr.AppendEvent("telemetry", "x", body("b")); !errors.Is(err, ErrConstitution) {
		t.Fatalf("unknown event class must refuse: %v", err)
	}
	for _, cls := range []string{EvRecovery, EvVerdict} {
		if _, err := tr.AppendEvent(cls, "x", body("b")); !errors.Is(err, ErrConstitution) || !strings.Contains(err.Error(), "recovery-owned") {
			t.Fatalf("%s must be recovery-owned: %v", cls, err)
		}
	}
	// Full edge product of the lifecycle machine.
	all := []TaskStatus{StatusCreated, StatusRunning, StatusCompleted, StatusFailed, StatusFailedPartial}
	for _, from := range all {
		for _, to := range all {
			legal := legalNext[from][to]
			want := map[TaskStatus]map[TaskStatus]bool{
				StatusCreated: {StatusRunning: true, StatusFailed: true, StatusFailedPartial: true},
				StatusRunning: {StatusCompleted: true, StatusFailed: true, StatusFailedPartial: true},
			}[from][to]
			if legal != want {
				t.Errorf("edge %s->%s: machine says %v, constitution says %v", from, to, legal, want)
			}
		}
	}
	// FAILED_PARTIAL is not caller-requestable.
	if err := tr.Transition(StatusFailedPartial, "forged"); !errors.Is(err, ErrLifecycle) || !strings.Contains(err.Error(), "not caller-requestable") {
		t.Fatalf("caller-requested FAILED_PARTIAL must refuse: %v", err)
	}
}

// Identity: single-use tasks, algorithm-prefixed objects, sink-owned
// sequence, relocatable root.
func TestIdentityStructure(t *testing.T) {
	r := testRoot(t)
	if _, err := r.CreateTask("t-id", TaskOptions{}); err != nil {
		t.Fatal(err)
	}
	if _, err := r.CreateTask("t-id", TaskOptions{}); !errors.Is(err, ErrIdentity) || !strings.Contains(err.Error(), "single-use") {
		t.Fatalf("task identity reuse must refuse: %v", err)
	}
	if _, err := r.CreateTask("../escape", TaskOptions{}); !errors.Is(err, ErrIdentity) {
		t.Fatal("path-shaped task id must refuse")
	}
	id, err := r.Store().StoreObject(ObjEvidencePayload, []byte("bytes"))
	if err != nil || !strings.HasPrefix(id, "sha256:") {
		t.Fatalf("object identity must be algorithm-prefixed: %q %v", id, err)
	}
	if _, err := r.Store().GetObject("md5:abcd"); !errors.Is(err, ErrIdentity) {
		t.Fatal("unknown algorithm prefix must refuse")
	}
	// Relocatability: move the state root; every identity and
	// verification must hold (Q-L6-3: location is never identity).
	tr, _ := r.CreateTask("t-move", TaskOptions{})
	if _, err := tr.AppendEvent(EvL4Audit, "l4", body("a"), Ref{ID: id, Class: ObjEvidencePayload}); err != nil {
		t.Fatal(err)
	}
	if err := tr.Transition(StatusRunning, "r"); err != nil {
		t.Fatal(err)
	}
	if err := tr.Transition(StatusCompleted, "done"); err != nil {
		t.Fatal(err)
	}
	tr.Close()
	newHome := filepath.Join(t.TempDir(), "relocated")
	if err := os.Rename(r.dir, newHome); err != nil {
		t.Fatal(err)
	}
	r2, err := OpenRoot(newHome)
	if err != nil {
		t.Fatal(err)
	}
	if v := r2.Verify("t-move"); v.Verdict != VerdictVerified {
		t.Fatalf("relocated root must verify: %+v", v)
	}
	if b, err := r2.Store().GetObject(id); err != nil || string(b) != "bytes" {
		t.Fatalf("relocated object must resolve by identity: %v", err)
	}
}

// The constitution hash is deterministic and recorded in every
// manifest.
func TestConstitutionHash(t *testing.T) {
	if ConstitutionHash() != ConstitutionHash() || len(ConstitutionHash()) != 64 {
		t.Fatal("constitution hash must be deterministic sha256")
	}
	r := testRoot(t)
	tr, _ := r.CreateTask("t-const", TaskOptions{})
	tr.Close()
	man, err := r.ReadManifest("t-const")
	if err != nil || man.ConstitutionHash != ConstitutionHash() {
		t.Fatalf("manifest must record the constitution hash: %v", err)
	}
}

// Root disjointness: pairwise, both directions, physical (symlinks
// resolved), case-folded, absolute-only, fail-closed on nonexistence
// (security review hardening).
func TestCheckDisjointRoots(t *testing.T) {
	base := t.TempDir()
	a := filepath.Join(base, "a")
	b := filepath.Join(base, "b")
	nested := filepath.Join(a, "state")
	for _, d := range []string{a, b, nested} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	if err := CheckDisjointRoots(a, b); err != nil {
		t.Fatalf("disjoint roots must pass: %v", err)
	}
	for _, pair := range [][]string{{a, nested}, {nested, a}, {a, a}} {
		if err := CheckDisjointRoots(pair...); !errors.Is(err, ErrIdentity) {
			t.Errorf("nested roots %v must refuse", pair)
		}
	}
	// Symlinked nesting is physical nesting.
	link := filepath.Join(base, "alias")
	if err := os.Symlink(a, link); err != nil {
		t.Fatal(err)
	}
	if err := CheckDisjointRoots(link, nested); !errors.Is(err, ErrIdentity) {
		t.Errorf("symlink-aliased nesting must refuse: %v", err)
	}
	// Relative, single, and nonexistent roots fail closed.
	if err := CheckDisjointRoots("relative", a); !errors.Is(err, ErrIdentity) {
		t.Error("relative root must refuse")
	}
	if err := CheckDisjointRoots(a); !errors.Is(err, ErrIdentity) {
		t.Error("single root must refuse")
	}
	if err := CheckDisjointRoots(filepath.Join(base, "ghost"), a); !errors.Is(err, ErrIdentity) {
		t.Error("nonexistent root must fail closed")
	}
}

// StatusView is structurally content-free: no field can carry event
// bodies or evidence bytes (compile-time shape pinned here).
func TestStatusViewContentFree(t *testing.T) {
	r := testRoot(t)
	tr, _ := r.CreateTask("t-view", TaskOptions{RetryOf: "t-old"})
	id, _ := tr.StoreObject(ObjEvidencePayload, []byte("evidence bytes"))
	if _, err := tr.AppendEvent(EvL2Delivery, "l2", body("payload"), Ref{ID: id, Class: ObjEvidencePayload}); err != nil {
		t.Fatal(err)
	}
	_ = tr.Transition(StatusRunning, "r")
	_ = tr.Transition(StatusCompleted, "done")
	tr.Close()
	view, err := r.ReadStatus("t-view")
	if err != nil {
		t.Fatal(err)
	}
	if view.Status != StatusCompleted || view.Verdict != VerdictVerified || view.RetryOf != "t-old" {
		t.Fatalf("view incomplete: %+v", view)
	}
	raw, _ := json.Marshal(view)
	if strings.Contains(string(raw), "evidence bytes") || strings.Contains(string(raw), "payload") {
		t.Fatal("StatusView must not carry contents")
	}
}

// --- Register B: behavioral -----------------------------------------

func TestObjectPlaneBehavior(t *testing.T) {
	r := testRoot(t)
	id, err := r.Store().StoreObject(ObjEvidencePayload, []byte("same"))
	if err != nil {
		t.Fatal(err)
	}
	// Idempotent duplicate.
	id2, err := r.Store().StoreObject(ObjEgressArtifact, []byte("same"))
	if err != nil || id2 != id {
		t.Fatalf("identical bytes must dedup to one identity: %v", err)
	}
	// Verify-on-read catches substitution.
	p, _ := r.Store().path(id)
	if err := os.Chmod(p, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte("tampered"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := r.Store().GetObject(id); !errors.Is(err, ErrCorrupt) {
		t.Fatalf("tampered object must fail verification: %v", err)
	}
	// Occupied-by-different-bytes on Put.
	if _, err := r.Store().StoreObject(ObjEvidencePayload, []byte("same")); !errors.Is(err, ErrCorrupt) || !strings.Contains(err.Error(), "occupied") {
		t.Fatalf("occupied address with different bytes must refuse: %v", err)
	}
}

func TestEventPlaneBehavior(t *testing.T) {
	r := testRoot(t)
	tr, err := r.CreateTask("t-ev", TaskOptions{})
	if err != nil {
		t.Fatal(err)
	}
	// Dangling reference dies at the door.
	ghost := objectID([]byte("never stored"))
	if _, err := tr.AppendEvent(EvL4Audit, "l4", body("a"), Ref{ID: ghost, Class: ObjEvidencePayload}); !errors.Is(err, ErrStream) || !strings.Contains(err.Error(), "already-durable") {
		t.Fatalf("dangling reference must refuse at append: %v", err)
	}
	// Sequence is sink-assigned and contiguous.
	e1, err := tr.AppendEvent(EvL4Audit, "l4", body("one"))
	if err != nil {
		t.Fatal(err)
	}
	e2, err := tr.AppendEvent(EvL4Audit, "l4", body("two"))
	if err != nil {
		t.Fatal(err)
	}
	if e2.Seq != e1.Seq+1 {
		t.Fatalf("sequence must be contiguous: %d %d", e1.Seq, e2.Seq)
	}
	// Invalid body JSON refused.
	if _, err := tr.AppendEvent(EvL4Audit, "l4", json.RawMessage("not-json")); !errors.Is(err, ErrStream) {
		t.Fatal("invalid body must refuse")
	}
	// Terminal task refuses appends.
	_ = tr.Transition(StatusRunning, "r")
	_ = tr.Transition(StatusFailed, "boom")
	if _, err := tr.AppendEvent(EvL4Audit, "l4", body("late")); !errors.Is(err, ErrLifecycle) {
		t.Fatalf("terminal task must refuse appends: %v", err)
	}
	tr.Close()
}

// A failed append is terminal for the stream — and stays terminal.
func TestFailedAppendTerminal(t *testing.T) {
	r := testRoot(t)
	tr, err := r.CreateTask("t-term", TaskOptions{})
	if err != nil {
		t.Fatal(err)
	}
	fault = func(p string) error {
		if p == "sink.pre-fsync" {
			return errors.New("injected")
		}
		return nil
	}
	_, err = tr.AppendEvent(EvL4Audit, "l4", body("doomed"))
	fault = nil
	if !errors.Is(err, ErrStream) || !strings.Contains(err.Error(), "terminal") {
		t.Fatalf("failed append must be typed terminal: %v", err)
	}
	if _, err := tr.AppendEvent(EvL4Audit, "l4", body("after")); !errors.Is(err, ErrStream) || !strings.Contains(err.Error(), "terminal") {
		t.Fatalf("stream must stay terminal: %v", err)
	}
	tr.Close()
}

// Manifest/stream disagreement and forged states are corruption.
func TestVerifierCatchesForgery(t *testing.T) {
	r := testRoot(t)
	tr, _ := r.CreateTask("t-forge", TaskOptions{})
	_ = tr.Transition(StatusRunning, "r")
	tr.Close()
	dir := r.taskDir("t-forge")
	man, _ := readManifest(dir)
	// Forge COMPLETED with no supporting lifecycle event.
	man.Status = StatusCompleted
	man.EventCount = 2
	man.StreamSummary = summaryOver(nil)
	if err := writeManifest(dir, man); err != nil {
		t.Fatal(err)
	}
	if v := r.Verify("t-forge"); v.Verdict != VerdictCorrupt || !strings.Contains(v.Detail, "no supporting lifecycle event") {
		t.Fatalf("unexplained manifest state must be corrupt: %+v", v)
	}
	// Forge FAILED_PARTIAL without a recovery event.
	tr2, _ := r.CreateTask("t-forge2", TaskOptions{})
	_ = tr2.Transition(StatusRunning, "r")
	tr2.Close()
	dir2 := r.taskDir("t-forge2")
	// Append a lifecycle event claiming FAILED_PARTIAL via raw stream
	// write is not possible through the API; simulate the projection
	// forgery only.
	man2, _ := readManifest(dir2)
	man2.Status = StatusFailedPartial
	if err := writeManifest(dir2, man2); err != nil {
		t.Fatal(err)
	}
	if v := r.Verify("t-forge2"); v.Verdict != VerdictCorrupt {
		t.Fatalf("forged FAILED_PARTIAL must be corrupt: %+v", v)
	}
}

// Recovery: projection, structural fact, idempotence.
func TestRecoveryAuthorities(t *testing.T) {
	r := testRoot(t)
	// Case 1: terminal event committed, projection lost (simulated by
	// rolling the manifest back to RUNNING after a real completion).
	tr, _ := r.CreateTask("t-rec1", TaskOptions{})
	_ = tr.Transition(StatusRunning, "r")
	_ = tr.Transition(StatusCompleted, "done")
	tr.Close()
	dir := r.taskDir("t-rec1")
	man, _ := readManifest(dir)
	man.Status = StatusRunning
	man.EventCount, man.StreamSummary = 0, ""
	if err := writeManifest(dir, man); err != nil {
		t.Fatal(err)
	}
	out, err := r.Recover("t-rec1")
	if err != nil || !out.Reconciled || out.Status != StatusCompleted {
		t.Fatalf("recovery must project the recorded decision: %+v %v", out, err)
	}
	if v := r.Verify("t-rec1"); v.Verdict != VerdictVerified {
		t.Fatalf("reconciled task must verify: %+v", v)
	}
	evs, _ := r.ReadEvents("t-rec1")
	foundRec := false
	for _, ev := range evs {
		if ev.Class == EvRecovery {
			foundRec = true
		}
	}
	if !foundRec {
		t.Fatal("projection must be evidenced by a reconciliation event")
	}

	// Case 2: record stops without a terminal — structural fact.
	tr2, _ := r.CreateTask("t-rec2", TaskOptions{})
	_ = tr2.Transition(StatusRunning, "r")
	if _, err := tr2.AppendEvent(EvL4Audit, "l4", body("mid")); err != nil {
		t.Fatal(err)
	}
	tr2.Close() // process "dies": no terminal transition
	out2, err := r.Recover("t-rec2")
	if err != nil || out2.Status != StatusFailedPartial || !out2.Acted {
		t.Fatalf("stopped record must become FAILED_PARTIAL: %+v %v", out2, err)
	}
	if v := r.Verify("t-rec2"); v.Verdict != VerdictVerified {
		t.Fatalf("recovered task must verify (recovery event present): %+v", v)
	}
	// Idempotence: a second pass appends nothing.
	before, _ := r.ReadEvents("t-rec2")
	out3, err := r.Recover("t-rec2")
	if err != nil || out3.Acted {
		t.Fatalf("second recovery must be a clean pass: %+v %v", out3, err)
	}
	after, _ := r.ReadEvents("t-rec2")
	if len(after) != len(before) {
		t.Fatal("idempotent recovery must append nothing")
	}
	// Recovery never originates: a task with NO terminal event never
	// recovers to COMPLETED (asserted across both cases above; the
	// only terminal recovery ever wrote without a recorded decision is
	// FAILED_PARTIAL).
}

// Torn tail: detected, preserved aside, typed TORN before recovery,
// VERIFIED-with-preserved-tail after.
func TestTornTail(t *testing.T) {
	r := testRoot(t)
	tr, _ := r.CreateTask("t-torn", TaskOptions{})
	_ = tr.Transition(StatusRunning, "r")
	tr.Close()
	stream := filepath.Join(r.taskDir("t-torn"), "events.log")
	f, err := os.OpenFile(stream, os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.WriteString("999\n{\"seq\":2,\"cl"); err != nil { // torn frame
		t.Fatal(err)
	}
	f.Close()
	if v := r.Verify("t-torn"); v.Verdict != VerdictTorn {
		t.Fatalf("torn tail must verdict TORN: %+v", v)
	}
	out, err := r.Recover("t-torn")
	if err != nil || out.TornSaved == "" || out.Status != StatusFailedPartial {
		t.Fatalf("recovery must preserve the tail and establish the structural fact: %+v %v", out, err)
	}
	saved, err := os.ReadFile(out.TornSaved)
	if err != nil || !strings.Contains(string(saved), "999\n") {
		t.Fatal("torn bytes must be preserved byte-exactly")
	}
	evs, _ := r.ReadEvents("t-torn")
	if len(evs) < 3 { // created, running, recovery, failed_partial lifecycle
		t.Fatalf("recovery events must be appended on sound framing: %d", len(evs))
	}
}

// Corrupt committed entry: verdict CORRUPT, recovery refuses to act.
func TestCorruptEntry(t *testing.T) {
	r := testRoot(t)
	tr, _ := r.CreateTask("t-corr", TaskOptions{})
	_ = tr.Transition(StatusRunning, "r")
	tr.Close()
	stream := filepath.Join(r.taskDir("t-corr"), "events.log")
	raw, _ := os.ReadFile(stream)
	mutated := strings.Replace(string(raw), `"reason":"r"`, `"reason":"X"`, 1)
	if mutated == string(raw) {
		t.Fatal("fixture failed to mutate")
	}
	if err := os.WriteFile(stream, []byte(mutated), 0o644); err != nil {
		t.Fatal(err)
	}
	if v := r.Verify("t-corr"); v.Verdict != VerdictCorrupt || !strings.Contains(v.Detail, "content verification") {
		t.Fatalf("mutated entry must be CORRUPT: %+v", v)
	}
	if _, err := r.Recover("t-corr"); !errors.Is(err, ErrCorrupt) {
		t.Fatalf("recovery must refuse a record it cannot trust: %v", err)
	}
}

// Reachability: shared objects, pinning, completeness verdict.
func TestScanReachable(t *testing.T) {
	r := testRoot(t)
	shared, _ := r.Store().StoreObject(ObjEvidencePayload, []byte("shared"))
	orphan, _ := r.Store().StoreObject(ObjEvidencePayload, []byte("orphan"))
	for _, id := range []string{"t-a", "t-b"} {
		tr, _ := r.CreateTask(id, TaskOptions{})
		if _, err := tr.AppendEvent(EvL2Delivery, "l2", body("d"), Ref{ID: shared, Class: ObjEvidencePayload}); err != nil {
			t.Fatal(err)
		}
		_ = tr.Transition(StatusRunning, "r")
		_ = tr.Transition(StatusCompleted, "done")
		tr.Close()
	}
	scan := r.ScanReachable()
	if !scan.Complete || !scan.Reachable[shared] || scan.Reachable[orphan] {
		t.Fatalf("scan must be complete, reach the shared object, and leave the orphan: %+v", scan)
	}
	// A torn task renders the scan incomplete — uncertainty pins.
	stream := filepath.Join(r.taskDir("t-b"), "events.log")
	f, _ := os.OpenFile(stream, os.O_APPEND|os.O_WRONLY, 0o644)
	f.WriteString("42\ntorn")
	f.Close()
	scan2 := r.ScanReachable()
	if scan2.Complete {
		t.Fatal("a torn record must render the scan INCOMPLETE")
	}
}

// The contamination flag is defense in depth: flag-only, never a
// refusal, never an edit.
func TestContaminationFlagOnly(t *testing.T) {
	r := testRoot(t)
	tr, _ := r.CreateTask("t-secret", TaskOptions{})
	secretish := json.RawMessage(`{"echo":"AKIA` + strings.Repeat("A", 16) + `"}`)
	if _, err := tr.AppendEvent(EvL4Audit, "l4", secretish); err != nil {
		t.Fatalf("suspect body must be recorded, not refused: %v", err)
	}
	evs, _ := r.ReadEvents("t-secret")
	var flagged, verbatim bool
	for _, ev := range evs {
		if ev.Class == EvContamination {
			flagged = true
		}
		if ev.Class == EvL4Audit && strings.Contains(string(ev.Body), "AKIA") {
			verbatim = true
		}
	}
	if !flagged || !verbatim {
		t.Fatalf("must flag AND preserve verbatim (flag-only): flagged=%v verbatim=%v", flagged, verbatim)
	}
	tr.Close()
}
