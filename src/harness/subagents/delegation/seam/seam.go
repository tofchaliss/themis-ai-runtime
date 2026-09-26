// Package seam is the L8 Delegator implementation
// (openspec/changes/archive/2026-09-23-layer-08-subagents §5.1, D-L8-2/3/4/8/15/16, C-L8-4..13,
// C-L8-17..20): L1 Resolve over the parent-subset sources + the
// template's one instruction → L2 Gather/Compose over the brief and
// the re-established evidence references → exactly one tool-less
// Model.Execute under the parent's governed model identity →
// StoreObject → AppendEvent(l8-delegation) → the result envelope.
//
// It is not a second L7: no workflow, cursor, or transition type; no
// L4 authorization (it imports neither tools nor execution); one
// Execute call site, not in a loop; one event literal; no goroutines;
// no model-name literal and no model-registry access — the request's
// Model is the L7-supplied identity. The wall tests pin each of these.
package seam

import (
	stdctx "context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	hctx "github.com/tofchaliss/themis-ai-runtime/src/harness/context"
	"github.com/tofchaliss/themis-ai-runtime/src/harness/instructions"
	"github.com/tofchaliss/themis-ai-runtime/src/harness/orchestration"
	"github.com/tofchaliss/themis-ai-runtime/src/harness/runtime/model"
	"github.com/tofchaliss/themis-ai-runtime/src/harness/state"
	"github.com/tofchaliss/themis-ai-runtime/src/harness/subagents/delegation"
)

// Seam holds the registry in force (loaded once at construction —
// C-L8-14 F: frozen for the process lifetime, verified against the
// anchor pin at Open by the wiring) and the parent's instruction
// policy. Nothing else: no model, no task, no conversation.
type Seam struct {
	registry *delegation.Registry
	policy   *instructions.Policy
}

// New loads the registry fail-closed. A registry that does not load
// means no delegation can be admitted — Open fails, by design.
func New(registryPath string, policy *instructions.Policy) (*Seam, error) {
	if policy == nil || policy.Hash == "" {
		return nil, fmt.Errorf("delegation seam: no validated instruction policy")
	}
	reg, err := delegation.LoadRegistry(registryPath)
	if err != nil {
		return nil, err
	}
	return &Seam{registry: reg, policy: policy}, nil
}

// RegistryHash is the identity of the registry in force (anchor pin).
func (s *Seam) RegistryHash() string { return s.registry.Hash }

// CheckDisjoint: the registry root must be disjoint from every
// task-writable root, or write_file could author a template the
// machinery accepts (the L9 catalog / L10 registry rule). Deployment
// wiring MUST check this before serving delegations.
func (s *Seam) CheckDisjoint(taskWritableRoots ...string) error {
	return state.CheckDisjointRoots(append([]string{s.registry.Root()}, taskWritableRoots...)...)
}

// Registered: assembly-time grant validation (C-L8-15 G) — existence
// in the governed registry, NOT current usability (C-L8-14 G, owner
// LOCK 2026-09-23): a withdrawn template remains assembly-admissible
// when a governed Skill references it; the delegate call refuses stage
// B with template-withdrawn, witnessed in the audit. Withdrawal blocks
// new delegations without invalidating the referencing Skill.
func (s *Seam) Registered(ref string) error {
	_, err := s.registry.Entry(ref)
	return err
}

func refuse(reason, detail string) error {
	return &orchestration.DelegationRefusal{Reason: reason, Detail: detail}
}

// composition is one instantiation's result: everything the capture
// and the witness name, plus the exact payload.
type composition struct {
	template *delegation.Template
	eis      *instructions.EffectiveSet
	payload  *hctx.Payload
	evidence []delegation.EvidenceRef
	capture  []byte
}

// capture is the deterministic instantiation capture (C-L8-13):
// identities only, never bytes, a pure function of the request and the
// record. The executor's audited evidence; the post-hook re-derives it
// and must agree.
type capture struct {
	TemplateRef  string                   `json:"template_ref"`
	TemplateHash string                   `json:"template_hash"`
	RegistryHash string                   `json:"registry_hash"`
	EISHash      string                   `json:"eis_hash"`
	ContractHash string                   `json:"contract_hash"`
	PayloadHash  string                   `json:"payload_hash"`
	RenderHash   string                   `json:"render_hash"`
	Evidence     []delegation.EvidenceRef `json:"evidence"`
}

// Instantiate is stage B (D-L8-15): pure reads, bounded by constants
// L8 does not own (C-L8-7), no model, no writes.
func (s *Seam) Instantiate(req orchestration.InstantiationRequest) ([]byte, error) {
	c, err := s.compose(req)
	if err != nil {
		return nil, err
	}
	return c.capture, nil
}

func (s *Seam) compose(req orchestration.InstantiationRequest) (*composition, error) {
	if req.Record == nil || req.TaskID == "" {
		return nil, fmt.Errorf("delegation seam: no record handle")
	}
	// 1. Template: the registry consulted exactly once, pre-instance
	// (C-L8-9 A); bytes re-read and verified against the frozen pin.
	entry, tpl, err := s.registry.Resolve(req.Template)
	if err != nil {
		switch {
		case errors.Is(err, delegation.ErrWithdrawn):
			return nil, refuse("template-withdrawn", err.Error())
		case errors.Is(err, delegation.ErrHashMismatch):
			return nil, refuse("template-hash-mismatch", err.Error())
		case errors.Is(err, delegation.ErrResolve), errors.Is(err, delegation.ErrTemplate):
			return nil, refuse("template-unresolvable", err.Error())
		default:
			return nil, refuse("registry-unreadable", err.Error())
		}
	}
	_ = entry

	// 2. Evidence: canonicalize (C-L8-6), then re-establish every
	// reference against the parent's own stream (C-L8-5): event at
	// seq exists, its Refs carry the object as evidence-payload, class
	// derives from the event, kind from the event. No bytes are read.
	events, err := req.Record.ReadEvents(req.TaskID)
	if err != nil {
		return nil, fmt.Errorf("delegation seam: read record: %w", err)
	}
	bySeq := map[int64]state.Event{}
	for _, ev := range events {
		bySeq[ev.Seq] = ev
	}
	raw := make([]delegation.EvidenceRef, 0, len(req.Evidence))
	for _, r := range req.Evidence {
		raw = append(raw, delegation.EvidenceRef{Seq: r.Seq, ObjectID: r.ObjectID})
	}
	refs, err := delegation.SortEvidence(raw)
	if err != nil {
		return nil, refuse("evidence-duplicate", err.Error())
	}
	type resolved struct {
		ref  delegation.EvidenceRef
		kind string
	}
	var items []resolved
	for i := range refs {
		ev, ok := bySeq[refs[i].Seq]
		if !ok {
			return nil, refuse("evidence-unreachable", fmt.Sprintf("no event at seq %d in this task's record", refs[i].Seq))
		}
		carried := false
		for _, rf := range ev.Refs {
			if rf.ID == refs[i].ObjectID && rf.Class == state.ObjEvidencePayload {
				carried = true
			}
		}
		if !carried {
			return nil, refuse("evidence-unreachable", fmt.Sprintf("event %d does not reference %s as evidence", ev.Seq, refs[i].ObjectID))
		}
		kind, class, err := deriveClass(ev, refs[i].ObjectID, req)
		if err != nil {
			return nil, err
		}
		refs[i].DerivedClass = string(class)
		// No per-item sensitivity is recorded on l4-audit or model-turn
		// events; every referenced item passed the PARENT contract's
		// ceiling, which is therefore the conservative sensitivity of
		// each reference (C-L8-7 §5). A template ceiling below it
		// refuses at Gather — the template narrows, never widens.
		if req.ParentSensitivityCeiling == "" {
			return nil, fmt.Errorf("delegation seam: no parent sensitivity ceiling")
		}
		refs[i].DerivedSensitivity = string(req.ParentSensitivityCeiling)
		items = append(items, resolved{ref: refs[i], kind: kind})
	}

	// 3. Slots (C-L8-14 C/D): brief in its slot; each reference to the
	// unique non-withheld slot whose kind matches; every other slot
	// declared absent so the contract's requirement decides.
	contract := tpl.Contract
	if len(req.Brief) > tpl.Brief.MaxBytes {
		return nil, refuse("brief-over-bound", fmt.Sprintf("brief is %d bytes; the template bounds it at %d", len(req.Brief), tpl.Brief.MaxBytes))
	}
	var briefKind string
	for _, sl := range contract.Slots {
		if sl.Name == tpl.Brief.Slot {
			briefKind = sl.Kind
		}
	}
	var briefItems []hctx.ContextItem
	if req.Brief != "" {
		briefItems = []hctx.ContextItem{{Kind: briefKind, Evidence: []byte(req.Brief)}}
	}
	assignments := []hctx.Assignment{{Slot: tpl.Brief.Slot, Source: hctx.Source{
		Name: "brief", Kind: hctx.KindInline, Authority: hctx.AuthorityExternalUntrusted,
		Sensitivity: hctx.SensitivityPublic, Author: "delegating-model", Items: briefItems,
	}}}
	filled := map[string]bool{tpl.Brief.Slot: true}
	objects := req.Record.Store()
	for _, it := range items {
		var target *hctx.Slot
		matches := 0
		for i := range contract.Slots {
			sl := &contract.Slots[i]
			if sl.Withhold || sl.Name == tpl.Brief.Slot {
				continue
			}
			if hctx.KindMatches(sl.Kind, it.kind) {
				matches++
				target = sl
			}
		}
		if matches != 1 {
			return nil, refuse("evidence-slot-ambiguous", fmt.Sprintf("%d slots of the template contract accept kind %q (reference %d:%s)", matches, it.kind, it.ref.Seq, it.ref.ObjectID))
		}
		filled[target.Name] = true
		assignments = append(assignments, hctx.Assignment{Slot: target.Name, Source: hctx.Source{
			Name: fmt.Sprintf("%d:%s", it.ref.Seq, it.ref.ObjectID), Kind: hctx.KindRecordObject,
			Authority: hctx.AuthorityClass(it.ref.DerivedClass), Sensitivity: hctx.Sensitivity(it.ref.DerivedSensitivity),
			Author: "parent-record", ObjectID: it.ref.ObjectID, Objects: objects,
			Items: []hctx.ContextItem{{Kind: it.kind, Version: fmt.Sprintf("seq:%d", it.ref.Seq)}},
		}})
	}
	for i := range contract.Slots {
		sl := &contract.Slots[i]
		if sl.Withhold || filled[sl.Name] {
			continue
		}
		assignments = append(assignments, hctx.Assignment{Slot: sl.Name, Source: hctx.AbsentSource(sl.Name)})
	}

	// 4. L1: the mandatory roots unconditionally, the optional scopes
	// only where the filter names them, the template's own file as the
	// one added source (C-L8-4). Same resolver, same policy.
	carry := map[string]bool{}
	for _, sc := range tpl.EISCarryScopes {
		carry[sc] = true
	}
	var sources []instructions.Source
	for _, src := range req.ParentSources {
		switch src.Kind {
		case instructions.ScopeHarnessSafety, instructions.ScopeHarnessSystem, instructions.ScopeThemisDomain:
			sources = append(sources, src)
		case instructions.ScopeTask:
			// task never carries
		default:
			if carry[src.Kind.String()] {
				sources = append(sources, src)
			}
		}
	}
	if tpl.Instruction != nil {
		src, err := instructions.ActivateDelegationSource(tpl.InstructionRaw, tpl.Instruction.SHA256)
		if err != nil {
			return nil, refuse("resolve-failed", err.Error())
		}
		sources = append(sources, src)
	}
	eis, err := instructions.Resolve(instructions.Config{Policy: s.policy, TaskID: req.TaskID}, sources...)
	if err != nil {
		return nil, refuse("resolve-failed", err.Error())
	}
	// D-L9-11 / C-L8-4: the delegated set is a SUBSET of the parent's
	// resolved set plus the template's own instruction. Resolve reads
	// the roots again; if any carried instruction is not byte-identical
	// to what the parent resolved at assembly, a governed root changed
	// under the running task — stage D (C-L8-14 F), never a silent
	// re-read (security review MED-1).
	if req.ParentEIS == nil {
		return nil, fmt.Errorf("delegation seam: no parent instruction set")
	}
	for id, h := range eis.SourceHashes {
		if id == "skill.delegation" {
			continue
		}
		if ph, ok := req.ParentEIS.SourceHashes[id]; !ok || ph != h {
			return nil, fmt.Errorf("delegation seam: carried instruction %q is not the parent's resolved instruction — a governed root changed under the running task", id)
		}
	}

	// 5. L2: Gather under L2's caps (bytes fetched lazily and
	// self-verified), Compose. A corruption verdict is machinery, not
	// a refusal (C-L8-5 A).
	g, err := hctx.Gather(contract, assignments)
	if err != nil {
		if errors.Is(err, state.ErrCorrupt) {
			return nil, fmt.Errorf("delegation seam: record integrity: %w", err)
		}
		return nil, refuse("compose-refused", err.Error())
	}
	payload, err := hctx.Compose(eis, s.policy, g)
	if err != nil {
		return nil, refuse("compose-refused", err.Error())
	}
	cap := capture{
		TemplateRef: req.Template, TemplateHash: tpl.Hash, RegistryHash: s.registry.Hash,
		EISHash: payload.EISHash, ContractHash: payload.ContractHash,
		PayloadHash: payload.PayloadHash, RenderHash: payload.RenderHash, Evidence: refs,
	}
	if cap.Evidence == nil {
		cap.Evidence = []delegation.EvidenceRef{}
	}
	cb, _ := json.Marshal(cap)
	return &composition{template: tpl, eis: eis, payload: payload, evidence: refs, capture: cb}, nil
}

// deriveClass is f(witnessing event) (C-L8-18): l4-audit → the
// recorded tool's registered trust under a still-matching registry
// hash; model-turn and l8-delegation → the floor; any other class is
// not a selectable evidence witness (C-L8-5 §4) — this switch IS the
// selectable set, stated once. It has no argument for the class of
// anything the event references.
func deriveClass(ev state.Event, objectID string, req orchestration.InstantiationRequest) (string, hctx.AuthorityClass, error) {
	switch ev.Class {
	case state.EvL4Audit:
		var body struct {
			Tool         string
			Decision     string
			RegistryHash string
		}
		if err := json.Unmarshal(ev.Body, &body); err != nil || body.Tool == "" {
			return "", "", refuse("evidence-unreachable", fmt.Sprintf("event %d audit body unreadable", ev.Seq))
		}
		if body.Decision != "authorized" {
			return "", "", refuse("evidence-unreachable", fmt.Sprintf("event %d is not an authorized call", ev.Seq))
		}
		if body.RegistryHash != req.RegistryHash {
			return "", "", refuse("evidence-registry-drift", fmt.Sprintf("event %d was authorized under registry %s, not the registry in force", ev.Seq, body.RegistryHash))
		}
		if req.ToolTrust == nil {
			return "", "", fmt.Errorf("delegation seam: no trust derivation wired")
		}
		trust, ok := req.ToolTrust(body.Tool)
		if !ok {
			return "", "", refuse("evidence-registry-drift", fmt.Sprintf("tool %q of event %d is not in the registry in force", body.Tool, ev.Seq))
		}
		return "tool:" + body.Tool, trust, nil
	case state.EvModelTurn:
		return "model-turn", hctx.AuthorityExternalUntrusted, nil
	case state.EvL8Delegation:
		d, err := delegation.Decode(ev.Body)
		if err != nil || d.OutputObjectRef != objectID {
			return "", "", refuse("evidence-unreachable", fmt.Sprintf("event %d does not name %s as its output", ev.Seq, objectID))
		}
		return "delegation-output", hctx.AuthorityExternalUntrusted, nil
	}
	return "", "", refuse("evidence-unreachable", "unselectable event class")
}

// Delegate is the post-hook (D-L8-8 steps 5-10).
func (s *Seam) Delegate(req orchestration.DelegationRequest) (*orchestration.DelegationResult, error) {
	if req.Task == nil || req.Runtime == nil || req.Model == "" || req.ParentCallSeq < 1 {
		return nil, fmt.Errorf("delegation seam: incomplete request")
	}
	// 5. Re-derive; the executor's capture must agree (stage D
	// otherwise — deterministic machinery disagreeing with itself).
	c, err := s.compose(req.InstantiationRequest)
	if err != nil {
		return nil, fmt.Errorf("delegation seam: re-derivation after an admitted instantiation failed: %w", err)
	}
	if string(c.capture) != string(req.Capture) {
		return nil, fmt.Errorf("delegation seam: re-derived composition disagrees with the instantiation capture")
	}
	for _, r := range c.evidence {
		if r.Seq >= req.ParentCallSeq {
			return nil, fmt.Errorf("delegation seam: evidence seq %d is not prior to the authorizing call %d", r.Seq, req.ParentCallSeq)
		}
	}

	// 6. Durable stores: template bytes (D-L8-17) and the composition
	// object = the exact model input (C-L8-9), before any model call.
	if err := orchestration.FaultAt("delegation.pre-composition-store"); err != nil {
		return nil, err
	}
	var templateRefs []string
	for _, b := range [][]byte{c.template.Raw, c.template.ContractRaw, c.template.InstructionRaw} {
		if len(b) == 0 {
			continue
		}
		id, err := req.Task.StoreObject(state.ObjEvidencePayload, b)
		if err != nil {
			return nil, err
		}
		templateRefs = append(templateRefs, id)
	}
	input, _ := json.Marshal(c.payload.Messages)
	compositionID, err := req.Task.StoreObject(state.ObjEvidencePayload, input)
	if err != nil {
		return nil, err
	}

	// 7. Exactly one tool-less model call under the parent's governed
	// identity, bounded by min(turn timeout, remaining wall budget)
	// after the pre-invocation floor check (D-L8-16 §3, C-L8-19 H).
	outcome := delegation.OutcomeCompleted
	termination := ""
	var output []byte
	var identity model.Identity
	endpoint := ""
	budget := req.TurnTimeout
	if !req.Deadline.IsZero() {
		rem := time.Until(req.Deadline)
		if rem <= 0 {
			outcome, termination = delegation.OutcomeProviderError, "deadline"
		} else if rem < budget {
			budget = rem
		}
	}
	if outcome == delegation.OutcomeCompleted {
		ctx, cancel := stdctx.WithTimeout(stdctx.Background(), budget)
		resp, mErr := req.Runtime.Execute(ctx, model.ExecutionRequest{
			Model: req.Model, Messages: c.payload.Messages, Options: model.DefaultOptions()})
		cancel()
		if mErr != nil {
			outcome = delegation.OutcomeProviderError
			termination = "error"
			if ctx.Err() != nil {
				termination = "deadline"
			}
		} else {
			output = []byte(resp.Content)
			identity = resp.Identity
			endpoint = model.RedactEndpoint(resp.Provenance.Endpoint)
			termination = string(resp.Termination)
		}
	}

	// 8. Output stored whole — never truncated (D-L8-16 §5); the bound
	// governs re-entry, not history.
	outputID := ""
	if outcome == delegation.OutcomeCompleted {
		if err := orchestration.FaultAt("delegation.pre-output-store"); err != nil {
			return nil, err
		}
		outputID, err = req.Task.StoreObject(state.ObjEvidencePayload, output)
		if err != nil {
			return nil, err
		}
		switch {
		case len(output) > c.template.MaxOutputBytes:
			outcome = delegation.OutcomeOutputOverBound
		case identity.Reported != "" && identity.Reported != identity.WireModel:
			outcome = delegation.OutcomeModelIdentityMismatch
		}
	}

	// 9. The witness.
	opts, _ := json.Marshal(model.DefaultOptions())
	var conflicts []json.RawMessage
	for _, cf := range c.eis.Conflicts {
		b, _ := json.Marshal(cf)
		conflicts = append(conflicts, b)
	}
	ev := &delegation.Event{
		ParentCallSeq: req.ParentCallSeq,
		Template:      delegation.TemplateIdentity{Ref: req.Template, RegistryHash: s.registry.Hash, TemplateHash: c.template.Hash},
		Composition: delegation.Composition{EISHash: c.payload.EISHash, ContractHash: c.payload.ContractHash,
			PayloadHash: c.payload.PayloadHash, RenderHash: c.payload.RenderHash, CompositionObjectRef: compositionID},
		TemplateObjectRefs: templateRefs,
		ModelIdentity: delegation.ModelIdentity{
			Governed:  delegation.GovernedModel{Name: req.Model, RegistryHash: req.ModelRegistryHash},
			Execution: delegation.ExecutionModel{WireModel: identity.WireModel, Runtime: identity.Runtime, Endpoint: endpoint, Reported: identity.Reported, OptionsHash: hex64(opts)},
		},
		EvidenceRefs:     c.evidence,
		ResolveConflicts: conflicts,
		OutputObjectRef:  outputID,
		Outcome:          outcome,
		Termination:      termination,
	}
	body, err := ev.Encode()
	if err != nil {
		return nil, err
	}
	refs := []state.Ref{{ID: compositionID, Class: state.ObjEvidencePayload}}
	for _, id := range templateRefs {
		refs = append(refs, state.Ref{ID: id, Class: state.ObjEvidencePayload})
	}
	if outputID != "" {
		refs = append(refs, state.Ref{ID: outputID, Class: state.ObjEvidencePayload})
	}
	if err := orchestration.FaultAt("delegation.pre-event-commit"); err != nil {
		return nil, err
	}
	committed, err := req.Task.AppendEvent(state.EvL8Delegation, "l8", body, refs...)
	if err != nil {
		return nil, err
	}
	res := &orchestration.DelegationResult{Outcome: string(outcome), Seq: committed.Seq, OutputObjectID: outputID}
	if outcome == delegation.OutcomeCompleted {
		res.Output = output
	}
	return res, nil
}

func hex64(b []byte) string {
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}
