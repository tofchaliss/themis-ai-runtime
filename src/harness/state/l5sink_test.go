package state

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"
)

// W-M2 (D-W-6 Wall B, D-W-2/3): the L5 handle is the only path that
// produces an event whose class→writer pair identifies L5; it emits
// exactly the two L5 forms; the record refuses the classes through
// every other path; and the handle refuses after the task is terminal.
func TestL5SinkIsTheOnlyL5EmissionPath(t *testing.T) {
	root, err := OpenRoot(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	tr, err := root.CreateTask("t-l5", TaskOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if err := tr.Transition(StatusRunning, "test"); err != nil {
		t.Fatal(err)
	}
	sink := tr.L5Sink()
	if err := sink.Transition("PROVISIONING", "ACTIVE", "provisioned"); err != nil {
		t.Fatalf("handle transition: %v", err)
	}
	if err := sink.Op("provision", []string{"git", "clone"}, 0, "ok"); err != nil {
		t.Fatalf("handle op: %v", err)
	}
	if err := sink.Egress("refused: mem_bytes observed breach", "", 0, 0); err != nil {
		t.Fatalf("handle egress refusal: %v", err)
	}
	if err := sink.Egress("acknowledged", "sha256:abc", 12, 1); err != nil {
		t.Fatalf("handle egress ack: %v", err)
	}
	// Shape rules the handle enforces itself.
	if err := sink.Egress("acknowledged", "", 0, 0); err == nil {
		t.Fatal("acknowledged without an address must refuse")
	}
	if err := sink.Egress("refused: x", "sha256:abc", 0, 0); err == nil {
		t.Fatal("a refusal carrying an address must refuse")
	}
	if err := sink.Transition("", "ACTIVE", ""); err == nil {
		t.Fatal("transition without from must refuse")
	}
	// The same classes through the caller-facing door: refused under
	// every identity, l5 included.
	for _, w := range []string{"l5", "l7", "l6", ""} {
		if _, err := tr.AppendEvent(EvL5Transition, w, []byte(`{"from":"a","to":"b"}`)); !errors.Is(err, ErrConstitution) {
			t.Errorf("AppendEvent(l5-transition, %q) admitted: %v", w, err)
		}
		if _, err := tr.AppendEvent(EvL5Op, w, []byte(`{"phase":"active"}`)); !errors.Is(err, ErrConstitution) {
			t.Errorf("AppendEvent(l5-op, %q) admitted: %v", w, err)
		}
	}
	// What the record holds: writer l5 on exactly the four handle events.
	evs, _ := root.ReadEvents("t-l5")
	var got []string
	for _, e := range evs {
		if e.Class == EvL5Transition || e.Class == EvL5Op {
			if e.Writer != writerL5 {
				t.Errorf("seq %d writer %q", e.Seq, e.Writer)
			}
			var b map[string]any
			_ = json.Unmarshal(e.Body, &b)
			got = append(got, e.Class+":"+strings.TrimSpace(func() string {
				if o, ok := b["outcome"].(string); ok {
					return o
				}
				return b["to"].(string)
			}()))
		}
	}
	want := []string{"l5-transition:ACTIVE", "l5-op:ok", "l5-op:refused: mem_bytes observed breach", "l5-op:acknowledged"}
	if strings.Join(got, "|") != strings.Join(want, "|") {
		t.Fatalf("record: %v", got)
	}
	// Terminal task: the handle refuses like every other writer.
	if err := tr.Transition(StatusFailed, "done"); err != nil {
		t.Fatal(err)
	}
	if err := sink.Transition("ACTIVE", "SEALED", "task-complete"); !errors.Is(err, ErrLifecycle) {
		t.Fatalf("handle after terminal: %v", err)
	}
	tr.Close()
	var nilSink *L5Sink
	if err := nilSink.Op("active", nil, 0, "ok"); err == nil {
		t.Fatal("unbound handle must refuse")
	}
}
