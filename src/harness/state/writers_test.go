package state

import (
	"errors"
	"strings"
	"testing"
)

// W-M1 (D-W-6 Wall A): the sink admits an event only under the writer
// identity that owns its class. Every caller-appendable class is
// admitted under its owner and refused under every other layer's
// identity, and the refusal is a constitution violation naming the
// pair. Primitive-only classes are covered by their own primitives
// (CreateTask, Transition, BindArtifact, recovery) and refused through
// AppendEvent regardless of writer.
func TestEventWriterInvariant(t *testing.T) {
	root, err := OpenRoot(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	tr, err := root.CreateTask("t-writers", TaskOptions{})
	if err != nil {
		t.Fatal(err)
	}
	defer tr.Close()
	if err := tr.Transition(StatusRunning, "test"); err != nil {
		t.Fatal(err)
	}
	layers := []string{"l1", "l2", "l3", "l4", "l5", "l6", "l6-recovery", "l7", "l8", "l10", "forger", ""}
	callerClasses := 0
	for class, owners := range eventWriters {
		if primitiveOnlyEvents[class] {
			for w := range owners {
				if _, err := tr.AppendEvent(class, w, []byte(`{}`)); !errors.Is(err, ErrConstitution) {
					t.Errorf("%s/%s: primitive-only class admitted through AppendEvent: %v", class, w, err)
				}
			}
			continue
		}
		callerClasses++
		for _, w := range layers {
			_, err := tr.AppendEvent(class, w, []byte(`{"w":"`+w+`"}`))
			switch {
			case owners[w] && err != nil:
				t.Errorf("%s written by its owner %q refused: %v", class, w, err)
			case !owners[w] && !errors.Is(err, ErrConstitution):
				t.Errorf("%s written by %q admitted (or refused for another reason): %v", class, w, err)
			case !owners[w] && !strings.Contains(err.Error(), "may not be written by"):
				t.Errorf("%s/%q: refusal must name the pair: %v", class, w, err)
			}
		}
	}
	if callerClasses == 0 {
		t.Fatal("no caller-appendable classes were exercised")
	}
	// The table covers the whole vocabulary and nothing else.
	for class := range eventClasses {
		if len(eventWriters[class]) == 0 {
			t.Errorf("class %q has no owning writer in eventWriters", class)
		}
	}
	for class := range eventWriters {
		if !eventClasses[class] {
			t.Errorf("eventWriters names %q, which is not in the event vocabulary", class)
		}
	}
	// Every event actually committed carries a writer its class owns.
	evs, err := root.ReadEvents("t-writers")
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range evs {
		if !eventWriters[e.Class][e.Writer] {
			t.Errorf("committed event %d %s has writer %q outside its owners", e.Seq, e.Class, e.Writer)
		}
	}
}

// The table is part of the constitution: every pair contributes a
// writer:<class>><writer> part to the hash (D-W-4). Asserted by
// recomputing over the same inputs the hash uses.
func TestEventWritersAreFoldedIntoTheHash(t *testing.T) {
	pairs := 0
	for _, ws := range eventWriters {
		pairs += len(ws)
	}
	if pairs < len(eventClasses) {
		t.Fatalf("fewer writer pairs (%d) than classes (%d)", pairs, len(eventClasses))
	}
	// A different table must produce a different hash: swap one owner
	// in a copy of the computation.
	base := ConstitutionHash()
	saved := eventWriters[EvL4Audit]
	eventWriters[EvL4Audit] = map[string]bool{"l7": true}
	moved := ConstitutionHash()
	eventWriters[EvL4Audit] = saved
	if moved == base {
		t.Fatal("changing the class→writer table did not change the constitution hash")
	}
	if ConstitutionHash() != base {
		t.Fatal("test did not restore the table")
	}
}
