package delegation

import (
	"errors"
	"strings"
	"testing"
)

func sha64(c byte) string { return strings.Repeat(string(c), 64) }

func goodEvent() *Event {
	return &Event{
		ParentCallSeq: 9,
		Template:      TemplateIdentity{Ref: "dependency-triage@1", RegistryHash: sha64('1'), TemplateHash: sha64('2')},
		Composition: Composition{EISHash: sha64('3'), ContractHash: sha64('4'), PayloadHash: sha64('5'),
			RenderHash: sha64('6'), CompositionObjectRef: "sha256:" + sha64('7')},
		TemplateObjectRefs: []string{"sha256:" + sha64('8'), "sha256:" + sha64('9')},
		ModelIdentity: ModelIdentity{Governed: GovernedModel{Name: "qwen2.5:7b", RegistryHash: sha64('a')},
			Execution: ExecutionModel{WireModel: "qwen2.5:7b", Runtime: "ollama", Endpoint: "http://localhost:11434", Reported: "qwen2.5:7b", OptionsHash: sha64('b')}},
		EvidenceRefs: []EvidenceRef{
			{Seq: 3, ObjectID: "sha256:" + sha64('c'), DerivedClass: "external-untrusted", DerivedSensitivity: "public"},
			{Seq: 7, ObjectID: "sha256:" + sha64('d'), DerivedClass: "governed-record", DerivedSensitivity: "internal"},
		},
		OutputObjectRef: "sha256:" + sha64('e'),
		Outcome:         OutcomeCompleted,
		Termination:     "stop",
	}
}

// Register C: the body is a pure function of its inputs.
func TestEventBodyReproducible(t *testing.T) {
	a, err := goodEvent().Encode()
	if err != nil {
		t.Fatal(err)
	}
	b, _ := goodEvent().Encode()
	if string(a) != string(b) {
		t.Fatal("two encodings of the same inputs differ")
	}
	if strings.Contains(string(a), "delegation_id") || strings.Contains(string(a), `"timestamp"`) {
		t.Fatalf("no identity field, no clock: %s", a)
	}
	back, err := Decode(a)
	if err != nil {
		t.Fatal(err)
	}
	c, _ := back.Encode()
	if string(c) != string(a) {
		t.Fatal("round trip changed the body")
	}
}

func TestEventClosure(t *testing.T) {
	cases := []struct {
		name   string
		mutate func(e *Event)
		want   string
	}{
		{"unknown outcome", func(e *Event) { e.Outcome = "succeeded" }, "unknown outcome"},
		{"evidence not prior to the call (C-L8-8)", func(e *Event) { e.EvidenceRefs[1].Seq = 9 }, "not prior"},
		{"evidence after the call", func(e *Event) { e.EvidenceRefs[1].Seq = 12 }, "not prior"},
		{"evidence unsorted (C-L8-6)", func(e *Event) { e.EvidenceRefs[0], e.EvidenceRefs[1] = e.EvidenceRefs[1], e.EvidenceRefs[0] }, "strictly ascending"},
		{"evidence duplicate", func(e *Event) { e.EvidenceRefs[1].Seq = 3 }, "strictly ascending"},
		{"evidence without derived class", func(e *Event) { e.EvidenceRefs[0].DerivedClass = "" }, "incomplete"},
		{"bare object id as evidence", func(e *Event) { e.EvidenceRefs[0].ObjectID = sha64('c') }, "incomplete"},
		{"no template bytes referenced (D-L8-17)", func(e *Event) { e.TemplateObjectRefs = nil }, "template bytes"},
		{"composition ref not an address", func(e *Event) { e.Composition.CompositionObjectRef = "x" }, "content address"},
		{"completed without output", func(e *Event) { e.OutputObjectRef = "" }, "requires output_object_ref"},
		{"provider-error with output", func(e *Event) { e.Outcome = OutcomeProviderError }, "no output"},
		{"no parent call", func(e *Event) { e.ParentCallSeq = 0 }, "parent_call_seq"},
		{"template identity incomplete", func(e *Event) { e.Template.TemplateHash = "latest" }, "template identity"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			e := goodEvent()
			c.mutate(e)
			_, err := e.Encode()
			if err == nil || !errors.Is(err, ErrEvent) || !strings.Contains(err.Error(), c.want) {
				t.Fatalf("want %q: %v", c.want, err)
			}
		})
	}
	// provider-error without output is legitimate.
	e := goodEvent()
	e.Outcome, e.OutputObjectRef = OutcomeProviderError, ""
	if _, err := e.Encode(); err != nil {
		t.Fatalf("provider-error: %v", err)
	}
	// SortEvidence canonicalizes and refuses duplicates without dedup.
	sorted, err := SortEvidence([]EvidenceRef{{Seq: 7, ObjectID: "x", DerivedClass: "c"}, {Seq: 3, ObjectID: "y", DerivedClass: "c"}})
	if err != nil || sorted[0].Seq != 3 || sorted[1].Seq != 7 {
		t.Fatalf("sort: %v %v", err, sorted)
	}
	if _, err := SortEvidence([]EvidenceRef{{Seq: 3}, {Seq: 3}}); err == nil {
		t.Fatal("duplicate seq must refuse, never dedup")
	}
}
