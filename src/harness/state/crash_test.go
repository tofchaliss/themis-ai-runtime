package state

// Register C (Q-L6-9): exhaustive deterministic fault-point sweep +
// real process-kill proof. Universal assertions: no crash point
// yields an acknowledged durable record containing a dangling
// reference, and no crash point lets recovery originate a semantic
// outcome.

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"testing"
	"time"
)

// faultPoints enumerates EVERY seam in the commit paths — a sampled
// sweep would be the silent-cap antipattern.
var faultPoints = []string{
	"object.write", "object.pre-fsync", "object.pre-link", "object.pre-dirsync",
	"sink.pre-write", "sink.pre-fsync",
	"manifest.pre-write", "manifest.pre-rename", "manifest.pre-dirsync",
}

func TestFaultPointSweepExhaustive(t *testing.T) {
	for _, point := range faultPoints {
		point := point
		t.Run(point, func(t *testing.T) {
			rootDir := filepath.Join(t.TempDir(), "state")
			r, err := OpenRoot(rootDir)
			if err != nil {
				t.Fatal(err)
			}
			tr, err := r.CreateTask("t-sweep", TaskOptions{})
			if err != nil {
				t.Fatal(err)
			}
			if err := tr.Transition(StatusRunning, "r"); err != nil {
				t.Fatal(err)
			}
			// Arm the fault, then drive the full commit chain; the
			// armed point errors, simulating death at that boundary.
			t.Cleanup(func() { fault = nil })
			fault = func(p string) error {
				if p == point {
					return errors.New("injected@" + p)
				}
				return nil
			}
			var faulted bool
			id, err := tr.StoreObject(ObjEvidencePayload, []byte("payload-"+point))
			if err != nil {
				faulted = true
			} else {
				if _, err := tr.AppendEvent(EvL2Delivery, "l2", body("d"), Ref{ID: id, Class: ObjEvidencePayload}); err != nil {
					faulted = true
				} else if err := tr.Transition(StatusCompleted, "done"); err != nil {
					faulted = true
				}
			}
			fault = nil
			if !faulted {
				t.Fatalf("fault point %s was never exercised by the chain", point)
			}
			tr.Close() // process dies here

			// Cold restart: recover, verify, assert the universals.
			r2, err := OpenRoot(rootDir)
			if err != nil {
				t.Fatal(err)
			}
			out, err := r2.Recover("t-sweep")
			if err != nil {
				t.Fatalf("recovery must handle every crash point: %v", err)
			}
			v := r2.Verify("t-sweep")
			if v.Verdict == VerdictCorrupt {
				t.Fatalf("no crash point may yield corruption: %+v", v)
			}
			// Universal 1: no dangling reference anywhere durable —
			// Verify resolves every ref; additionally the object
			// address is ABSENT or COMPLETE, never partial.
			if id != "" && r2.Store().HasObject(id) {
				if _, gerr := r2.Store().GetObject(id); gerr != nil {
					t.Fatalf("occupied address must be complete: %v", gerr)
				}
			}
			// Universal 2: recovery never originates a semantic
			// outcome — COMPLETED only if the record holds the
			// committed terminal decision.
			evs, _ := r2.ReadEvents("t-sweep")
			hasTerminal := false
			for _, ev := range evs {
				if ev.Class == EvLifecycle {
					var b struct {
						To string `json:"to"`
					}
					_ = json.Unmarshal(ev.Body, &b)
					if terminal(TaskStatus(b.To)) && TaskStatus(b.To) != StatusFailedPartial {
						hasTerminal = true
					}
				}
			}
			switch out.Status {
			case StatusCompleted:
				if !hasTerminal {
					t.Fatal("recovery originated COMPLETED without a recorded decision")
				}
			case StatusFailedPartial:
				if hasTerminal {
					t.Fatal("recovery ignored a recorded terminal decision")
				}
			default:
				t.Fatalf("recovery must land a typed terminal: %s", out.Status)
			}
			// Object-plane retry property (D-L6-11): after the crash,
			// a retry of the same bytes succeeds or the address is
			// already complete — never permanently poisoned.
			retryID, rerr := r2.Store().StoreObject(ObjEvidencePayload, []byte("payload-"+point))
			if rerr != nil {
				t.Fatalf("object write must be retryable after any crash point: %v", rerr)
			}
			if b, err := r2.Store().GetObject(retryID); err != nil || string(b) != "payload-"+point {
				t.Fatalf("retried object must verify: %v", err)
			}
		})
	}
}

// Real process-kill proof: a child process runs a task loop; SIGKILL
// mid-write; cold recovery yields typed FAILED_PARTIAL with recovery
// evidence, nothing dangling, nothing silently repaired.
func TestRealKillRecovery(t *testing.T) {
	if rootDir := os.Getenv("L6_CHILD_ROOT"); rootDir != "" {
		// Child mode: append events as fast as possible until killed.
		r, err := OpenRoot(rootDir)
		if err != nil {
			fmt.Println("child:", err)
			os.Exit(1)
		}
		tr, err := r.CreateTask("t-kill", TaskOptions{})
		if err != nil {
			fmt.Println("child:", err)
			os.Exit(1)
		}
		_ = tr.Transition(StatusRunning, "r")
		for i := 0; ; i++ {
			id, err := tr.StoreObject(ObjEvidencePayload, []byte(fmt.Sprintf("payload-%d", i)))
			if err != nil {
				os.Exit(1)
			}
			if _, err := tr.AppendEvent(EvL4Audit, "l4", body(fmt.Sprintf("call-%d", i)), Ref{ID: id, Class: ObjEvidencePayload}); err != nil {
				os.Exit(1)
			}
		}
	}

	rootDir := filepath.Join(t.TempDir(), "state")
	cmd := exec.Command(os.Args[0], "-test.run", "TestRealKillRecovery")
	cmd.Env = append(os.Environ(), "L6_CHILD_ROOT="+rootDir)
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	// Poll for committed work before killing (test review: a fixed
	// sleep flakes on loaded machines — kill only once the child has
	// provably committed events).
	streamPath := filepath.Join(rootDir, "tasks", "t-kill", "events.log")
	deadline := time.Now().Add(10 * time.Second)
	for {
		if info, err := os.Stat(streamPath); err == nil && info.Size() > 1024 {
			break
		}
		if time.Now().After(deadline) {
			_ = cmd.Process.Kill()
			t.Fatal("child never committed work")
		}
		time.Sleep(20 * time.Millisecond)
	}
	if err := cmd.Process.Signal(syscall.SIGKILL); err != nil {
		t.Fatal(err)
	}
	_ = cmd.Wait()

	r, err := OpenRoot(rootDir)
	if err != nil {
		t.Fatal(err)
	}
	out, err := r.Recover("t-kill")
	if err != nil {
		t.Fatalf("recovery after SIGKILL: %v", err)
	}
	if out.Status != StatusFailedPartial {
		t.Fatalf("killed task must be FAILED_PARTIAL, got %s", out.Status)
	}
	v := r.Verify("t-kill")
	if v.Verdict == VerdictCorrupt {
		t.Fatalf("killed task must never verify corrupt: %+v", v)
	}
	evs, err := r.ReadEvents("t-kill")
	if err != nil || len(evs) < 3 {
		t.Fatalf("committed events must survive the kill: %d %v", len(evs), err)
	}
	// Every surviving reference resolves (no dangles at any kill
	// point) — Verify checked it; double-check explicitly.
	for _, ev := range evs {
		for _, ref := range ev.Refs {
			if _, err := r.Store().GetObject(ref.ID); err != nil {
				t.Fatalf("dangling reference after kill: %v", err)
			}
		}
	}
	// Recovery evidence present; second pass idempotent.
	foundRec := false
	for _, ev := range evs {
		if ev.Class == EvRecovery {
			foundRec = true
		}
	}
	if !foundRec {
		t.Fatal("FAILED_PARTIAL must carry recovery evidence")
	}
	out2, err := r.Recover("t-kill")
	if err != nil || out2.Acted {
		t.Fatalf("second recovery must be a clean pass: %+v %v", out2, err)
	}
	t.Logf("real-kill proof: %d committed events, torn=%v, status=%s", len(evs), out.TornSaved != "", out.Status)
}
