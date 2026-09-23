package context

// L8 amendment to L2 (openspec/changes/l8-subagents §5.2, C-L8-6/7):
// the lazy record-object source, several distinct sources per slot,
// and the tool:* slot kind.

import (
	"errors"
	"strings"
	"testing"
)

type memObjects map[string][]byte

func (m memObjects) GetObject(id string) ([]byte, error) {
	b, ok := m[id]
	if !ok {
		return nil, errors.New("durable-commit failed: missing")
	}
	return b, nil
}

func recordContract(t *testing.T) *Contract {
	t.Helper()
	return loadTestContract(t, `{"version":1,"workflow":"triage","slots":[
	 {"name":"brief","kind":"delegation-brief","requirement":"required","classes":["external-untrusted"]},
	 {"name":"tool-evidence","kind":"tool:*","requirement":"optional","classes":["external-untrusted","governed-record"]},
	 {"name":"prior","kind":"model-turn","requirement":"optional","classes":["external-untrusted"]}],
	 "sensitivity_ceiling":"internal"}`)
}

// absent declares an optional slot empty (the seam names every
// non-withheld slot; requirement decides).
func absent(name string, cls AuthorityClass) Source {
	return Source{Name: name, Kind: KindRecordObject, Authority: cls, Sensitivity: SensitivityPublic, Author: "seam"}
}

func objSource(name, id, kind string, cls AuthorityClass, objs ObjectReader) Source {
	return Source{Name: name, Kind: KindRecordObject, Authority: cls, Sensitivity: SensitivityPublic,
		Author: "read_file", ObjectID: id, Objects: objs, Items: []ContextItem{{Kind: kind}}}
}

func TestRecordObjectSourcesFillOneSlot(t *testing.T) {
	a, b := []byte("alpha\n"), []byte("beta\n")
	ida, idb := "sha256:"+evidenceHash(a), "sha256:"+evidenceHash(b)
	objs := memObjects{ida: a, idb: b}
	c := recordContract(t)
	brief := Source{Name: "brief", Kind: KindInline, Authority: AuthorityExternalUntrusted, Sensitivity: SensitivityPublic,
		Author: "delegating-model", Items: []ContextItem{{Kind: "delegation-brief", Evidence: []byte("compare")}}}
	// Two distinct sources, one slot, in either order → identical items.
	g1, err := Gather(c, []Assignment{{Slot: "brief", Source: brief}, {Slot: "prior", Source: absent("prior", AuthorityExternalUntrusted)},
		{Slot: "tool-evidence", Source: objSource("3:"+ida, ida, "tool:read_file", AuthorityExternalUntrusted, objs)},
		{Slot: "tool-evidence", Source: objSource("7:"+idb, idb, "tool:get_finding", AuthorityGovernedRecord, objs)}})
	if err != nil {
		t.Fatalf("two sources in one slot must gather: %v", err)
	}
	g2, err := Gather(c, []Assignment{{Slot: "brief", Source: brief}, {Slot: "prior", Source: absent("prior", AuthorityExternalUntrusted)},
		{Slot: "tool-evidence", Source: objSource("7:"+idb, idb, "tool:get_finding", AuthorityGovernedRecord, objs)},
		{Slot: "tool-evidence", Source: objSource("3:"+ida, ida, "tool:read_file", AuthorityExternalUntrusted, objs)}})
	if err != nil {
		t.Fatal(err)
	}
	if len(g1.items["tool-evidence"]) != 2 || len(g2.items["tool-evidence"]) != 2 {
		t.Fatalf("both items must be delivered: %d %d", len(g1.items["tool-evidence"]), len(g2.items["tool-evidence"]))
	}
	for i := range g1.items["tool-evidence"] {
		x, y := g1.items["tool-evidence"][i], g2.items["tool-evidence"][i]
		if x.Hash != y.Hash || x.Kind != y.Kind || x.Authority != y.Authority {
			t.Fatalf("assignment order changed presentation: %+v vs %+v", x, y)
		}
	}
	// The same source twice is still a malformed plan.
	if _, err := Gather(c, []Assignment{{Slot: "brief", Source: brief}, {Slot: "prior", Source: absent("prior", AuthorityExternalUntrusted)},
		{Slot: "tool-evidence", Source: objSource("3:"+ida, ida, "tool:read_file", AuthorityExternalUntrusted, objs)},
		{Slot: "tool-evidence", Source: objSource("3:"+ida, ida, "tool:read_file", AuthorityExternalUntrusted, objs)}}); err == nil || !strings.Contains(err.Error(), "assigned twice") {
		t.Fatalf("duplicate source: %v", err)
	}
	// Class is the slot's to permit — a derived class outside it refuses.
	if _, err := Gather(c, []Assignment{{Slot: "brief", Source: brief}, {Slot: "tool-evidence", Source: absent("tool-evidence", AuthorityExternalUntrusted)},
		{Slot: "prior", Source: objSource("9:"+ida, ida, "model-turn", AuthorityGovernedRecord, objs)}}); !errors.Is(err, ErrPlanOutsideContract) {
		t.Fatalf("class outside slot: %v", err)
	}
	// Kind routing: a tool item does not fill the model-turn slot.
	if _, err := Gather(c, []Assignment{{Slot: "brief", Source: brief}, {Slot: "tool-evidence", Source: absent("tool-evidence", AuthorityExternalUntrusted)},
		{Slot: "prior", Source: objSource("9:"+ida, ida, "tool:read_file", AuthorityExternalUntrusted, objs)}}); !errors.Is(err, ErrPlanOutsideContract) {
		t.Fatalf("kind outside slot: %v", err)
	}
	// A missing object is a hard error, never a silent absence.
	if _, err := Gather(c, []Assignment{{Slot: "brief", Source: brief}, {Slot: "prior", Source: absent("prior", AuthorityExternalUntrusted)},
		{Slot: "tool-evidence", Source: objSource("3:x", "sha256:"+strings.Repeat("0", 64), "tool:read_file", AuthorityExternalUntrusted, objs)}}); err == nil || !strings.Contains(err.Error(), "missing") {
		t.Fatalf("missing object: %v", err)
	}
	// Bytes that do not match their address refuse (self-verifying fetch).
	bad := memObjects{ida: []byte("tampered\n")}
	if _, err := Gather(c, []Assignment{{Slot: "brief", Source: brief}, {Slot: "prior", Source: absent("prior", AuthorityExternalUntrusted)},
		{Slot: "tool-evidence", Source: objSource("3:"+ida, ida, "tool:read_file", AuthorityExternalUntrusted, bad)}}); err == nil || !strings.Contains(err.Error(), "content address") {
		t.Fatalf("address mismatch: %v", err)
	}
}
