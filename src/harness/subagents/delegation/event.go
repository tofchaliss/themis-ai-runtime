package delegation

// The l8-delegation event body (D-L8-8/17/19, C-L8-6, C-L8-8, C-L8-10,
// C-L8-20): the one witness that a delegated reasoning execution was
// ESTABLISHED within the parent task. Identity is (task_id, seq) — the
// event carries no delegation id. Every semantic input needed to
// rebuild what the delegated model saw and produced is reachable from
// this body through content-addressed references; no new object class
// exists (composition and output are evidence-payload bytes). L6
// validates the envelope, never this content; this package owns the
// shape and its canonical encoding so the body is reproducible from
// its inputs — no clock, no random field, no map iteration order.

import (
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
)

// Outcome is the closed post-instance vocabulary (D-L8-15 stage C and
// D-L8-16 §5). "completed" is the only outcome whose output re-enters
// the parent as content.
type Outcome string

const (
	OutcomeCompleted             Outcome = "completed"
	OutcomeProviderError         Outcome = "provider-error"
	OutcomeOutputOverBound       Outcome = "output-over-bound"
	OutcomeModelIdentityMismatch Outcome = "model-identity-mismatch"
)

var outcomes = map[Outcome]bool{
	OutcomeCompleted: true, OutcomeProviderError: true,
	OutcomeOutputOverBound: true, OutcomeModelIdentityMismatch: true,
}

var objectIDSyntax = regexp.MustCompile(`^sha256:[0-9a-f]{64}$`)

// TemplateIdentity names the governed template under which the
// delegation ran: governance identity (name@version) plus the two
// artifact identities that make it re-verifiable two-way (D-L8-17).
type TemplateIdentity struct {
	Ref          string `json:"ref"`
	RegistryHash string `json:"registry_hash"`
	TemplateHash string `json:"template_hash"`
}

// Composition records what the delegated model was given, by identity,
// and where the exact bytes live (C-L8-9): the composition object is
// the exact model input (system + user messages, canonical).
type Composition struct {
	EISHash              string `json:"eis_hash"`
	ContractHash         string `json:"contract_hash"`
	PayloadHash          string `json:"payload_hash"`
	RenderHash           string `json:"render_hash"`
	CompositionObjectRef string `json:"composition_object_ref"`
}

// ModelIdentity separates the governed name from the execution
// identity the provider actually reported (C-L8-10).
type ModelIdentity struct {
	Governed  GovernedModel  `json:"governed"`
	Execution ExecutionModel `json:"execution"`
}

type GovernedModel struct {
	Name         string `json:"name"`
	RegistryHash string `json:"registry_hash"`
}

type ExecutionModel struct {
	WireModel   string `json:"wire_model"`
	Runtime     string `json:"runtime"`
	Endpoint    string `json:"endpoint"`
	Reported    string `json:"reported"`
	OptionsHash string `json:"options_hash"`
}

// EvidenceRef is one resolved evidence reference as the seam
// re-established it from the parent's record (C-L8-5): the declaring
// event's seq, the object, and the class and sensitivity DERIVED from
// that event — recorded so reconstruction re-derives them without the
// argument.
type EvidenceRef struct {
	Seq                int64  `json:"seq"`
	ObjectID           string `json:"object_id"`
	DerivedClass       string `json:"derived_class"`
	DerivedSensitivity string `json:"derived_sensitivity"`
}

// Event is the l8-delegation body.
type Event struct {
	ParentCallSeq      int64            `json:"parent_call_seq"`
	Template           TemplateIdentity `json:"template"`
	Composition        Composition      `json:"composition"`
	TemplateObjectRefs []string         `json:"template_object_refs"`
	ModelIdentity      ModelIdentity    `json:"model_identity"`
	EvidenceRefs       []EvidenceRef    `json:"evidence_refs"`
	// ResolveConflicts carries the delegated L1 resolution's conflict
	// records inside the witness (C-L8-12 Am. 1) so the l4-audit →
	// l8-delegation window stays empty. Opaque L1 content.
	ResolveConflicts []json.RawMessage `json:"resolve_conflicts,omitempty"`
	OutputObjectRef  string            `json:"output_object_ref,omitempty"`
	Outcome          Outcome           `json:"outcome"`
	Termination      string            `json:"termination,omitempty"`
}

// Validate is the body's own closure: closed outcome, well-formed
// references, evidence canonical (strictly ascending seq, no
// duplicates — C-L8-6) and prior to the authorizing call (C-L8-8),
// output present iff the outcome could have produced one.
func (e *Event) Validate() error {
	if e.ParentCallSeq < 1 {
		return fmt.Errorf("%w: parent_call_seq must name the authorizing l4-audit", ErrEvent)
	}
	if !outcomes[e.Outcome] {
		return fmt.Errorf("%w: unknown outcome %q", ErrEvent, e.Outcome)
	}
	if e.Template.Ref == "" || !shaSyntax.MatchString(e.Template.RegistryHash) || !shaSyntax.MatchString(e.Template.TemplateHash) {
		return fmt.Errorf("%w: template identity incomplete", ErrEvent)
	}
	for _, h := range []string{e.Composition.EISHash, e.Composition.ContractHash, e.Composition.PayloadHash, e.Composition.RenderHash} {
		if !shaSyntax.MatchString(h) {
			return fmt.Errorf("%w: composition hashes incomplete", ErrEvent)
		}
	}
	if !objectIDSyntax.MatchString(e.Composition.CompositionObjectRef) {
		return fmt.Errorf("%w: composition_object_ref must be a content address", ErrEvent)
	}
	if len(e.TemplateObjectRefs) == 0 {
		return fmt.Errorf("%w: template bytes must be referenced (D-L8-17)", ErrEvent)
	}
	for _, r := range e.TemplateObjectRefs {
		if !objectIDSyntax.MatchString(r) {
			return fmt.Errorf("%w: template_object_refs entry %q is not a content address", ErrEvent, r)
		}
	}
	var prev int64
	for i, r := range e.EvidenceRefs {
		if r.Seq < 1 || !objectIDSyntax.MatchString(r.ObjectID) || r.DerivedClass == "" {
			return fmt.Errorf("%w: evidence_refs[%d] incomplete", ErrEvent, i)
		}
		if r.Seq >= e.ParentCallSeq {
			return fmt.Errorf("%w: evidence_refs[%d] seq %d is not prior to the authorizing call %d (C-L8-8)", ErrEvent, i, r.Seq, e.ParentCallSeq)
		}
		if i > 0 && r.Seq <= prev {
			return fmt.Errorf("%w: evidence_refs must be strictly ascending by seq (canonical order, no duplicates — C-L8-6)", ErrEvent)
		}
		prev = r.Seq
	}
	switch e.Outcome {
	case OutcomeCompleted, OutcomeOutputOverBound, OutcomeModelIdentityMismatch:
		if !objectIDSyntax.MatchString(e.OutputObjectRef) {
			return fmt.Errorf("%w: outcome %s requires output_object_ref", ErrEvent, e.Outcome)
		}
	case OutcomeProviderError:
		if e.OutputObjectRef != "" {
			return fmt.Errorf("%w: provider-error has no output", ErrEvent)
		}
	}
	return nil
}

// Encode produces the canonical body bytes: validated, evidence in
// canonical order (the seam sorts before source construction; a body
// arriving unsorted is a machinery defect and refuses rather than
// being silently reordered), struct field order fixed by the type.
func (e *Event) Encode() ([]byte, error) {
	if err := e.Validate(); err != nil {
		return nil, err
	}
	return json.Marshal(e)
}

// Decode parses a body and re-validates it: a body that would not
// encode does not decode either.
func Decode(body []byte) (*Event, error) {
	var e Event
	if err := json.Unmarshal(body, &e); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrEvent, err)
	}
	if err := e.Validate(); err != nil {
		return nil, err
	}
	return &e, nil
}

// SortEvidence is the seam's canonicalization (C-L8-6): ascending seq.
// It reports a duplicate seq as an error rather than dedup'ing — L8
// performs no evidence dedup.
func SortEvidence(refs []EvidenceRef) ([]EvidenceRef, error) {
	out := append([]EvidenceRef(nil), refs...)
	sort.SliceStable(out, func(i, j int) bool { return out[i].Seq < out[j].Seq })
	for i := 1; i < len(out); i++ {
		if out[i].Seq == out[i-1].Seq {
			return nil, fmt.Errorf("%w: duplicate evidence reference at seq %d", ErrEvent, out[i].Seq)
		}
	}
	return out, nil
}
