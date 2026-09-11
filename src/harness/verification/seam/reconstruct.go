package seam

// Reconstruction and view tooling over a live state root (D-L10-12):
// read-only derivation, outside any walk, model-unreachable. The
// discrepancy artifact is stored through the EXISTING L6 object-store
// primitive as an evidence-payload — content-addressed, outside every
// task event sequence (audit scope). L6 vocabulary check (D-L10-12
// OPEN item), result recorded: the evidence-payload class and the
// store primitive carry the artifact without any authority-semantics
// change and no task-stream append; the retention-anchoring of
// audit-scope objects under a FUTURE reachability GC is a recorded
// ADG follow-up (no deleter exists in L6 today — "no probabilistic
// GC", store has no Delete).

import (
	"encoding/json"
	"fmt"

	"github.com/tofchaliss/themis/state"
	"github.com/tofchaliss/themis/verification"
)

// ReconstructTask reconstructs every committed evaluation of a task
// from the durable record alone. For each inconsistent (or
// input-missing) reconstruction, the typed discrepancy artifact is
// stored via the root object store and its content address returned.
// The task record is never modified; the examined records are never
// rewritten (no-retroactive-mutation, proven by the caller re-reading
// events).
func ReconstructTask(root *state.Root, taskID string) ([]verification.Report, []string, error) {
	events, err := root.ReadEvents(taskID)
	if err != nil {
		return nil, nil, fmt.Errorf("reconstruction inputs unavailable: %v", err)
	}

	var reports []verification.Report
	var artifacts []string
	for _, ev := range events {
		if ev.Class != state.EvVerification {
			continue
		}
		rep := reconstructOne(root, ev, events)
		reports = append(reports, rep)
		if !rep.Consistent {
			body, merr := json.Marshal(rep)
			if merr != nil {
				return nil, nil, merr
			}
			id, serr := root.Store().StoreObject(state.ObjEvidencePayload, body)
			if serr != nil {
				return nil, nil, serr
			}
			artifacts = append(artifacts, id)
		}
	}
	return reports, artifacts, nil
}

func reconstructOne(root *state.Root, ev state.Event, events []state.Event) verification.Report {
	fail := func(missing string) verification.Report {
		return verification.Report{
			ViewVersion:   "reconstruct-v1",
			Consistent:    false,
			MissingInputs: []string{missing},
		}
	}

	// The evaluation record is named EXPLICITLY by the event body's
	// "record" object id — never selected by content-sniffing the
	// unlabeled refs, where a record-shaped hostile report (raw bytes
	// are model-authored) could mask the genuine record (close
	// architecture review H-1).
	var body struct {
		Contract string `json:"contract"`
		Outcome  string `json:"outcome"`
		Record   string `json:"record"`
	}
	if err := json.Unmarshal(ev.Body, &body); err != nil || body.Record == "" {
		return fail("evaluation_record")
	}
	rb, err := root.Store().GetObject(body.Record)
	if err != nil {
		return fail("evaluation_record")
	}
	var cand verification.Evaluation
	if json.Unmarshal(rb, &cand) != nil || cand.Outcome == "" || cand.ContractSHA256 == "" {
		return fail("evaluation_record")
	}
	record := &cand

	// Durable bytes by content address. The record carries bare
	// sha256 hex (L10 identity); the L6 store address is the same
	// digest under its algo-prefixed representation — same bytes, two
	// purposes, no second identity mechanism (the BodyHash/ObjectID
	// precedent, literally).
	addr := func(hexDigest string) string { return "sha256:" + hexDigest }
	contractBytes, _ := root.Store().GetObject(addr(record.ContractSHA256))
	var rawBytes, canonicalBytes []byte
	if record.RawObjectID != "" {
		rawBytes, _ = root.Store().GetObject(addr(record.RawObjectID))
	}
	if record.CanonicalObjectID != "" {
		canonicalBytes, _ = root.Store().GetObject(addr(record.CanonicalObjectID))
	}

	var canon verification.CanonicalizeFunc
	if c, ok := canonicalizers[record.Capability]; ok {
		canon = verification.CanonicalizeFunc(c)
	}
	rep := verification.Reconstruct(*record, contractBytes, rawBytes, canonicalBytes, canon)

	// Cross-checks the pure core cannot perform (close security
	// review M-1): (a) the event body δ actually consumed must agree
	// with the evaluation-record content — a record/event divergence
	// is an inconsistency, not a nuance; (b) the execution reference
	// must resolve to the committed L4 audit whose executed tool and
	// authorizing registry agree with the record — the recorded-pin
	// vs executed two-source leg (D-L10-10 #2).
	addCheck := func(name, recorded, recomputed string) {
		ok := recorded == recomputed && recorded != ""
		rep.Checks = append(rep.Checks, verification.Check{Name: name, OK: ok, Recorded: recorded, Recomputed: recomputed})
		if !ok {
			rep.Consistent = false
		}
	}
	addCheck("event_contract", body.Contract, fmt.Sprintf("%s@%d", record.ContractName, record.ContractVersion))
	addCheck("event_outcome", body.Outcome, string(record.Outcome))

	var auditSeq int64 = -1
	if _, err := fmt.Sscanf(record.ExecutionRef, "l4:%d", &auditSeq); err == nil {
		found := false
		for _, ae := range events {
			if ae.Seq != auditSeq || ae.Class != state.EvL4Audit {
				continue
			}
			found = true
			var audit struct {
				Tool         string `json:"Tool"`
				RegistryHash string `json:"RegistryHash"`
			}
			_ = json.Unmarshal(ae.Body, &audit)
			addCheck("executed_capability", record.Capability, audit.Tool)
			addCheck("authorizing_registry", record.RegistrySHA256, audit.RegistryHash)
			break
		}
		if !found {
			rep.MissingInputs = append(rep.MissingInputs, "execution_record")
			rep.Consistent = false
		}
	} else {
		rep.MissingInputs = append(rep.MissingInputs, "execution_record")
		rep.Consistent = false
	}
	return rep
}

// TaskVerificationHistory derives the pure history view from a task's
// committed verification events. Read-only; no walk effect.
func TaskVerificationHistory(root *state.Root, taskID string) (verification.HistoryView, error) {
	events, err := root.ReadEvents(taskID)
	if err != nil {
		return verification.HistoryView{}, err
	}
	var views []verification.VerificationEventView
	for _, ev := range events {
		if ev.Class != state.EvVerification {
			continue
		}
		var body struct {
			Contract string `json:"contract"`
			Outcome  string `json:"outcome"`
		}
		if err := json.Unmarshal(ev.Body, &body); err != nil {
			continue
		}
		views = append(views, verification.VerificationEventView{
			Seq: ev.Seq, Contract: body.Contract, Outcome: body.Outcome})
	}
	return verification.VerificationHistory(views), nil
}
