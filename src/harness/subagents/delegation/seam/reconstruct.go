package seam

// Reconstruction (Register C; D-L8-17, C-L8-9, C-L8-8): from the L6
// closure of one l8-delegation event ALONE — never the registry, never
// today's files or roots — re-derive what the delegated model saw and
// produced and compare every recorded identity with an independently
// derived value. Vocabulary (D-L10-12 / D-L11-17): CONFIRMED,
// UNREPRODUCIBLE-FOR-MISSING-INPUTS (a typed availability fact),
// DISCREPANCY (a typed self-consistency failure naming the pair —
// an architectural defect signal, zero authority). Nothing here
// establishes anything; a CONFIRMED reconstruction proves the record
// self-consistent, not the delegation's content true.

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"

	hctx "github.com/tofchaliss/themis-ai-runtime/src/harness/context"
	"github.com/tofchaliss/themis-ai-runtime/src/harness/orchestration"
	"github.com/tofchaliss/themis-ai-runtime/src/harness/runtime/model"
	"github.com/tofchaliss/themis-ai-runtime/src/harness/state"
	"github.com/tofchaliss/themis-ai-runtime/src/harness/subagents/delegation"
)

const (
	VerdictConfirmed      = "CONFIRMED"
	VerdictUnreproducible = "UNREPRODUCIBLE-FOR-MISSING-INPUTS"
	VerdictDiscrepancy    = "DISCREPANCY"
)

// Reconstruction is the typed result for one delegation.
type Reconstruction struct {
	TaskID        string   `json:"task_id"`
	Seq           int64    `json:"seq"`
	ParentCallSeq int64    `json:"parent_call_seq"`
	Verdict       string   `json:"verdict"`
	Outcome       string   `json:"outcome"`
	Template      string   `json:"template"`
	MissingInputs []string `json:"missing_inputs,omitempty"`
	Discrepancies []string `json:"discrepancies,omitempty"`
	Checks        []string `json:"checks"`
}

// ReconstructConfig carries the only non-L6 inputs a reconstruction
// admits: the trust derivation for l4-audit-witnessed evidence under
// the registry hash the audit recorded (the registry is anchor-pinned;
// a hash the caller cannot serve makes the class unreproducible, never
// assumed).
type ReconstructConfig struct {
	RegistryHash string
	ToolTrust    func(tool string) (hctx.AuthorityClass, bool)
}

// Verdict precedence: a DISCREPANCY (the record disagrees with
// itself — a defect signal, D-L11-17) outranks UNREPRODUCIBLE (an
// input is unavailable); a missing input never hides a discrepancy.
func (r *Reconstruction) missing(what string) {
	r.MissingInputs = append(r.MissingInputs, what)
	if r.Verdict != VerdictDiscrepancy {
		r.Verdict = VerdictUnreproducible
	}
}

func (r *Reconstruction) discrepancy(what string) {
	r.Discrepancies = append(r.Discrepancies, what)
	r.Verdict = VerdictDiscrepancy
}

func (r *Reconstruction) ok(what string) { r.Checks = append(r.Checks, what) }

// ReconstructTask reconstructs every delegation of a task.
func ReconstructTask(root *state.Root, taskID string, cfg ReconstructConfig) ([]Reconstruction, error) {
	events, err := root.ReadEvents(taskID)
	if err != nil {
		return nil, err
	}
	var out []Reconstruction
	for _, ev := range events {
		if ev.Class == state.EvL8Delegation {
			out = append(out, reconstructOne(root, taskID, ev, events, cfg))
		}
	}
	return out, nil
}

// ReconstructDelegation reconstructs the delegation witnessed at seq.
func ReconstructDelegation(root *state.Root, taskID string, seq int64, cfg ReconstructConfig) (Reconstruction, error) {
	events, err := root.ReadEvents(taskID)
	if err != nil {
		return Reconstruction{}, err
	}
	for _, ev := range events {
		if ev.Seq == seq && ev.Class == state.EvL8Delegation {
			return reconstructOne(root, taskID, ev, events, cfg), nil
		}
	}
	return Reconstruction{}, fmt.Errorf("no l8-delegation at seq %d in task %s", seq, taskID)
}

func reconstructOne(root *state.Root, taskID string, ev state.Event, events []state.Event, cfg ReconstructConfig) Reconstruction {
	r := Reconstruction{TaskID: taskID, Seq: ev.Seq, Verdict: VerdictConfirmed}
	d, err := delegation.Decode(ev.Body)
	if err != nil {
		r.discrepancy("witness body violates its own closure: " + err.Error())
		return r
	}
	r.ParentCallSeq, r.Outcome, r.Template = d.ParentCallSeq, string(d.Outcome), d.Template.Ref
	bySeq := map[int64]state.Event{}
	for _, e := range events {
		bySeq[e.Seq] = e
	}
	store := root.Store()
	fetch := func(id, what string) []byte {
		b, err := store.GetObject(id)
		if err != nil {
			r.missing(what + " " + id)
			return nil
		}
		return b
	}

	// 1. The authorizing audit, and the window between it and the
	// witness (C-L8-8): the audit exists, is an authorized delegate
	// call, and nothing else sits in the window.
	audit, ok := bySeq[d.ParentCallSeq]
	var ab struct {
		Tool, Decision, ArgsHash, RegistryHash string
	}
	if !ok || audit.Class != state.EvL4Audit {
		r.discrepancy(fmt.Sprintf("parent_call_seq %d is not an l4-audit", d.ParentCallSeq))
	} else if json.Unmarshal(audit.Body, &ab) != nil || ab.Decision != "authorized" {
		r.discrepancy(fmt.Sprintf("audit %d is not an authorized call", d.ParentCallSeq))
	} else {
		r.ok("authorizing l4-audit is an authorized call")
	}
	for _, e := range events {
		if e.Seq > d.ParentCallSeq && e.Seq < ev.Seq {
			r.discrepancy(fmt.Sprintf("event %d (%s) inside the l4-audit → l8-delegation window", e.Seq, e.Class))
		}
	}
	if len(r.Discrepancies) == 0 {
		r.ok("window between authorization and witness is empty")
	}

	// 2. Two-way template identity from stored bytes (D-L8-17).
	var manifestRaw, contractRaw, instructionRaw []byte
	for i, id := range d.TemplateObjectRefs {
		b := fetch(id, "template object")
		switch i {
		case 0:
			manifestRaw = b
		case 1:
			contractRaw = b
		case 2:
			instructionRaw = b
		}
	}
	var tpl *delegation.Template
	if manifestRaw != nil && contractRaw != nil {
		t, err := delegation.ParseTemplate(manifestRaw, contractRaw, instructionRaw)
		if err != nil {
			r.discrepancy("stored template bytes do not form a valid template: " + err.Error())
		} else {
			tpl = t
			if t.Hash != d.Template.TemplateHash {
				r.discrepancy("stored template hash ≠ recorded template_hash")
			} else if fmt.Sprintf("%s@%d", t.Name, t.TemplateVersion) != d.Template.Ref {
				r.discrepancy("stored template self-declaration ≠ recorded ref")
			} else {
				r.ok("template identity verified two-way from stored bytes")
			}
			if t.Contract.Hash != d.Composition.ContractHash {
				r.discrepancy("stored contract hash ≠ recorded contract_hash")
			}
		}
	}

	// 3. The composition object = the exact model input; its system
	// message hashes to render_hash.
	var msgs []model.Message
	if comp := fetch(d.Composition.CompositionObjectRef, "composition object"); comp != nil {
		if json.Unmarshal(comp, &msgs) != nil || len(msgs) != 2 || msgs[0].Role != model.RoleSystem || msgs[1].Role != model.RoleUser {
			r.discrepancy("composition object is not a [system, user] input")
			msgs = nil
		} else if hex64of(msgs[0].Content) != d.Composition.RenderHash {
			r.discrepancy("stored system message ≠ recorded render_hash")
		} else {
			r.ok("composition object holds the recorded EIS render")
		}
	}

	// 4. Evidence: re-establish each reference from the record and
	// re-derive its class; ordering and priority (C-L8-6, C-L8-8).
	var prev int64
	var assignments []hctx.Assignment
	evidenceOK := true
	for i, ref := range d.EvidenceRefs {
		if ref.Seq >= d.ParentCallSeq || (i > 0 && ref.Seq <= prev) {
			r.discrepancy(fmt.Sprintf("evidence_refs[%d] violates seq ordering", i))
			evidenceOK = false
		}
		prev = ref.Seq
		e, ok := bySeq[ref.Seq]
		if !ok {
			r.discrepancy(fmt.Sprintf("evidence_refs[%d]: no event at seq %d", i, ref.Seq))
			evidenceOK = false
			continue
		}
		carried := false
		for _, rf := range e.Refs {
			if rf.ID == ref.ObjectID && rf.Class == state.ObjEvidencePayload {
				carried = true
			}
		}
		if !carried {
			r.discrepancy(fmt.Sprintf("evidence_refs[%d]: event %d does not reference %s", i, ref.Seq, ref.ObjectID))
			evidenceOK = false
			continue
		}
		kind, class, derr := deriveClass(e, ref.ObjectID, orchestration.InstantiationRequest{RegistryHash: cfg.RegistryHash, ToolTrust: cfg.ToolTrust})
		if derr != nil {
			var refusal *orchestration.DelegationRefusal
			if e.Class == state.EvL4Audit && (cfg.ToolTrust == nil || asRefusal(derr, &refusal) && refusal.Reason == "evidence-registry-drift") {
				r.missing(fmt.Sprintf("registry for audit %d (class taken from the witness)", ref.Seq))
				class = hctx.AuthorityClass(ref.DerivedClass)
				var body struct{ Tool string }
				_ = json.Unmarshal(e.Body, &body)
				kind = "tool:" + body.Tool
			} else {
				r.discrepancy(fmt.Sprintf("evidence_refs[%d]: class not derivable: %v", i, derr))
				evidenceOK = false
				continue
			}
		}
		if string(class) != ref.DerivedClass {
			r.discrepancy(fmt.Sprintf("evidence_refs[%d]: recorded class %s ≠ derived %s", i, ref.DerivedClass, class))
			evidenceOK = false
		}
		if !store.HasObject(ref.ObjectID) {
			r.missing("evidence object " + ref.ObjectID)
			evidenceOK = false
			continue
		}
		if tpl != nil {
			var target *hctx.Slot
			matches := 0
			for j := range tpl.Contract.Slots {
				sl := &tpl.Contract.Slots[j]
				if sl.Withhold || sl.Name == tpl.Brief.Slot {
					continue
				}
				if hctx.KindMatches(sl.Kind, kind) {
					matches++
					target = sl
				}
			}
			if matches != 1 {
				r.discrepancy(fmt.Sprintf("evidence_refs[%d]: kind %q fits %d slots", i, kind, matches))
				evidenceOK = false
				continue
			}
			assignments = append(assignments, hctx.Assignment{Slot: target.Name, Source: hctx.Source{
				Name: fmt.Sprintf("%d:%s", ref.Seq, ref.ObjectID), Kind: hctx.KindRecordObject,
				Authority: class, Sensitivity: hctx.Sensitivity(ref.DerivedSensitivity),
				Author: "parent-record", ObjectID: ref.ObjectID, Objects: store,
				Items: []hctx.ContextItem{{Kind: kind, Version: fmt.Sprintf("seq:%d", ref.Seq)}},
			}})
		}
	}
	if evidenceOK && len(d.EvidenceRefs) > 0 {
		r.ok("every evidence reference re-established and re-classified from the record")
	}

	// 5. The brief: from the parent's own turn object (the tool call
	// whose arguments hash to the audit's ArgsHash).
	brief, briefFound := "", false
	if ab.ArgsHash != "" {
		for _, e := range events {
			if e.Class != state.EvModelTurn || e.Seq >= d.ParentCallSeq || len(e.Refs) == 0 {
				continue
			}
			turn, err := store.GetObject(e.Refs[0].ID)
			if err != nil {
				continue
			}
			var tb struct {
				ToolCalls []model.ToolCall `json:"tool_calls"`
			}
			if json.Unmarshal(turn, &tb) != nil {
				continue
			}
			for _, tc := range tb.ToolCalls {
				if hctx.EvidenceHash(tc.Arguments) == ab.ArgsHash {
					var args struct {
						Brief string `json:"brief"`
					}
					_ = json.Unmarshal(tc.Arguments, &args)
					brief, briefFound = args.Brief, true
				}
			}
		}
	}
	if !briefFound {
		r.missing("parent turn object carrying the delegate call")
	}

	// 6. Re-compose from stored bytes and compare byte-for-byte.
	if tpl != nil && msgs != nil && briefFound && evidenceOK {
		var briefKind string
		filled := map[string]bool{tpl.Brief.Slot: true}
		for _, sl := range tpl.Contract.Slots {
			if sl.Name == tpl.Brief.Slot {
				briefKind = sl.Kind
			}
		}
		for _, a := range assignments {
			filled[a.Slot] = true
		}
		var briefItems []hctx.ContextItem
		if brief != "" {
			briefItems = []hctx.ContextItem{{Kind: briefKind, Evidence: []byte(brief)}}
		}
		all := append([]hctx.Assignment{{Slot: tpl.Brief.Slot, Source: hctx.Source{
			Name: "brief", Kind: hctx.KindInline, Authority: hctx.AuthorityExternalUntrusted,
			Sensitivity: hctx.SensitivityPublic, Author: "delegating-model", Items: briefItems}}}, assignments...)
		for i := range tpl.Contract.Slots {
			sl := &tpl.Contract.Slots[i]
			if sl.Withhold || filled[sl.Name] {
				continue
			}
			all = append(all, hctx.Assignment{Slot: sl.Name, Source: hctx.AbsentSource(sl.Name)})
		}
		g, err := hctx.Gather(tpl.Contract, all)
		if err != nil {
			r.discrepancy("re-gather from stored bytes refused: " + err.Error())
		} else if p, err := hctx.ComposeWithSystem(msgs[0], d.Composition.EISHash, d.Composition.RenderHash, g); err != nil {
			r.discrepancy("re-compose from stored bytes refused: " + err.Error())
		} else {
			if p.Messages[1].Content != msgs[1].Content {
				r.discrepancy("re-derived user message ≠ stored composition object")
			}
			if p.PayloadHash != d.Composition.PayloadHash {
				r.discrepancy("re-derived payload_hash ≠ recorded")
			}
			if p.ContractHash != d.Composition.ContractHash {
				r.discrepancy("re-derived contract_hash ≠ recorded")
			}
			if len(r.Discrepancies) == 0 {
				r.ok("re-derived composition is byte-identical to the stored model input")
			}
		}
	}

	// 7. Output: present iff the outcome could produce one; bytes
	// self-verify at the store.
	if d.OutputObjectRef != "" {
		if fetch(d.OutputObjectRef, "output object") != nil {
			r.ok("output object present and self-consistent")
		}
	}
	return r
}

func asRefusal(err error, target **orchestration.DelegationRefusal) bool {
	r, ok := err.(*orchestration.DelegationRefusal)
	if ok {
		*target = r
	}
	return ok
}

func hex64of(s string) string {
	sum := sha256.Sum256([]byte(s))
	return hex.EncodeToString(sum[:])
}
