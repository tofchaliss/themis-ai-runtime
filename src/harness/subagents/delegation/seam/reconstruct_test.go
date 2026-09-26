package seam

// Register C (M5): byte-exact reconstruction from L6 alone; typed
// verdicts; window purity; permutation invariance; quota-attempt
// semantics; the class-derivation function's totality; the
// conversation projection.

import (
	"encoding/json"
	"errors"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"

	hctx "github.com/tofchaliss/themis-ai-runtime/src/harness/context"
	"github.com/tofchaliss/themis-ai-runtime/src/harness/orchestration"
	"github.com/tofchaliss/themis-ai-runtime/src/harness/runtime/model"
	"github.com/tofchaliss/themis-ai-runtime/src/harness/state"
	"github.com/tofchaliss/themis-ai-runtime/src/harness/subagents/delegation"
	"github.com/tofchaliss/themis-ai-runtime/src/harness/tools"
)

func trustFor(t *testing.T, w *world) ReconstructConfig {
	t.Helper()
	reg, err := tools.LoadRegistry(filepath.Join(w.root, "policies/tools/registry-v5.json"))
	if err != nil {
		t.Fatal(err)
	}
	return ReconstructConfig{RegistryHash: reg.Hash, ToolTrust: func(name string) (hctx.AuthorityClass, bool) {
		for _, tl := range reg.Tools {
			if tl.Name == name {
				return tl.Trust, true
			}
		}
		return "", false
	}}
}

// completedWalk runs the positive walk under a private registry copy
// and returns the world and the witness seq.
func completedWalk(t *testing.T, task string) (*world, string, int64) {
	t.Helper()
	reg := delegationRegistry(t, nil)
	m := &dynModel{delegated: okDelegated}
	w := newWorld(t, m, reg, true)
	m.parent = triageParent(w, task, func(seq int64, id string) string { return ref(seq, id) }, "Which dependency is vulnerable?")
	if res, err := w.o.SubmitTask(w.envelope(t, task)); err != nil || res.Status != state.StatusCompleted {
		t.Fatalf("%+v %v", res, err)
	}
	l8, _ := delegationEvent(t, events(t, w, task))
	return w, reg, l8.Seq
}

func TestReconstructionConfirmedFromRecordAlone(t *testing.T) {
	const task = "t-recon"
	w, reg, seq := completedWalk(t, task)
	cfg := trustFor(t, w)
	r, err := ReconstructDelegation(w.sroot, task, seq, cfg)
	if err != nil || r.Verdict != VerdictConfirmed {
		t.Fatalf("fresh record must reconstruct CONFIRMED: %v %+v", err, r)
	}
	if len(r.Checks) < 6 {
		t.Fatalf("expected the full check list: %v", r.Checks)
	}
	// (i) template withdrawn, (ii) template and contract files deleted,
	// (iii) an instruction root file changed — the record is the only
	// input; every one of these leaves the reconstruction CONFIRMED.
	if err := os.RemoveAll(filepath.Dir(reg)); err != nil {
		t.Fatal(err)
	}
	safety := filepath.Join(w.safety, "advisory-only.md")
	orig, _ := os.ReadFile(safety)
	if err := os.WriteFile(safety, append(orig, []byte("\nAdded line for the reconstruction test.\n")...), 0o644); err != nil {
		t.Fatal(err)
	}
	r, err = ReconstructDelegation(w.sroot, task, seq, cfg)
	if err != nil || r.Verdict != VerdictConfirmed {
		t.Fatalf("after withdrawal, deletion, and root change the record alone must still CONFIRM: %v %+v", err, r)
	}
	all, err := ReconstructTask(w.sroot, task, cfg)
	if err != nil || len(all) != 1 || all[0].Seq != seq {
		t.Fatalf("%v %+v", err, all)
	}
}

func TestReconstructionTypedFailures(t *testing.T) {
	t.Run("missing object is UNREPRODUCIBLE", func(t *testing.T) {
		const task = "t-recon-missing"
		w, _, seq := completedWalk(t, task)
		l8, d := delegationEvent(t, events(t, w, task))
		_ = l8
		// Remove the output object from the store: an availability
		// fact, typed and neutral — never a discrepancy.
		id := strings.TrimPrefix(d.OutputObjectRef, "sha256:")
		removed := false
		_ = filepath.WalkDir(filepath.Join(w.stateDir), func(p string, de os.DirEntry, err error) error {
			if err == nil && !de.IsDir() && strings.Contains(p, id) {
				removed = os.Remove(p) == nil
			}
			return nil
		})
		if !removed {
			t.Skip("object layout not found on disk")
		}
		r, err := ReconstructDelegation(w.sroot, task, seq, trustFor(t, w))
		if err != nil || r.Verdict != VerdictUnreproducible || len(r.MissingInputs) == 0 {
			t.Fatalf("%v %+v", err, r)
		}
		if len(r.Discrepancies) != 0 {
			t.Fatalf("a missing input is not a discrepancy: %v", r.Discrepancies)
		}
	})
	t.Run("doctored identity is DISCREPANCY", func(t *testing.T) {
		const task = "t-recon-doctored"
		w, _, seq := completedWalk(t, task)
		// Reconstruct a doctored witness body through the same function
		// the task path uses, by feeding a modified event.
		evs := events(t, w, task)
		var ev state.Event
		for _, e := range evs {
			if e.Seq == seq {
				ev = e
			}
		}
		d, err := delegation.Decode(ev.Body)
		if err != nil {
			t.Fatal(err)
		}
		d.Composition.PayloadHash = strings.Repeat("0", 64)
		doctored, _ := json.Marshal(d)
		ev.Body = doctored
		r := reconstructOne(w.sroot, task, ev, evs, trustFor(t, w))
		if r.Verdict != VerdictDiscrepancy {
			t.Fatalf("doctored payload_hash must be a DISCREPANCY: %+v", r)
		}
		found := false
		for _, dsc := range r.Discrepancies {
			if strings.Contains(dsc, "payload_hash") {
				found = true
			}
		}
		if !found {
			t.Fatalf("the discrepancy must name the disagreeing pair: %v", r.Discrepancies)
		}
	})
	t.Run("event inside the window is DISCREPANCY", func(t *testing.T) {
		const task = "t-recon-window"
		w, _, seq := completedWalk(t, task)
		evs := events(t, w, task)
		var ev state.Event
		for _, e := range evs {
			if e.Seq == seq {
				ev = e
			}
		}
		d, _ := delegation.Decode(ev.Body)
		d.ParentCallSeq = d.ParentCallSeq - 1 // now a foreign event sits in the window
		for i := range d.EvidenceRefs {
			if d.EvidenceRefs[i].Seq >= d.ParentCallSeq {
				d.EvidenceRefs[i].Seq = d.ParentCallSeq - 1
			}
		}
		doctored, _ := json.Marshal(d)
		ev.Body = doctored
		r := reconstructOne(w.sroot, task, ev, evs, trustFor(t, w))
		if r.Verdict != VerdictDiscrepancy || !anyContains(r.Discrepancies, "window") {
			t.Fatalf("the window violation must be named: %+v", r)
		}
	})
	t.Run("doctored template identity is DISCREPANCY", func(t *testing.T) {
		const task = "t-recon-tpl"
		w, _, seq := completedWalk(t, task)
		evs := events(t, w, task)
		var ev state.Event
		for _, e := range evs {
			if e.Seq == seq {
				ev = e
			}
		}
		d, _ := delegation.Decode(ev.Body)
		d.Template.TemplateHash = strings.Repeat("f", 64)
		doctored, _ := json.Marshal(d)
		ev.Body = doctored
		r := reconstructOne(w.sroot, task, ev, evs, trustFor(t, w))
		if r.Verdict != VerdictDiscrepancy || !anyContains(r.Discrepancies, "template_hash") {
			t.Fatalf("stored template bytes ≠ recorded identity must be named: %+v", r)
		}
	})
}

func anyContains(list []string, sub string) bool {
	for _, s := range list {
		if strings.Contains(s, sub) {
			return true
		}
	}
	return false
}

// C-L8-6 §7: the evidence argument denotes a set — permuting it
// changes ArgsHash only; composition hashes and evidence_refs are
// identical. Two items with EQUAL bytes (equal hashes) are the case
// that exposes a missing canonical sort.
func TestEvidencePermutationIsInvariant(t *testing.T) {
	run := func(t *testing.T, task string, order func(a, b string) string) (*delegation.Event, map[string]any) {
		m := &dynModel{delegated: okDelegated}
		w := newWorld(t, m, "", true)
		m.parent = func(turn int, conv []model.Message) model.ExecutionResponse {
			switch turn {
			case 1:
				return call("c1", "read_file", `{"path":"go.mod"}`)
			case 2:
				return call("c1b", "read_file", `{"path":"go.mod"}`) // identical bytes, second event
			case 3:
				evs, _ := w.sroot.ReadEvents(task)
				var refs []string
				for _, e := range evs {
					if e.Class == state.EvL4Audit && len(e.Refs) == 1 {
						refs = append(refs, ref(e.Seq, e.Refs[0].ID))
					}
				}
				args, _ := json.Marshal(map[string]string{"template": "dependency-triage@1", "evidence": order(refs[0], refs[1]), "brief": "compare"})
				return call("c2", "delegate", string(args))
			}
			return call("c3", "declare_done", `{}`)
		}
		if res, err := w.o.SubmitTask(w.envelope(t, task)); err != nil || res.Status != state.StatusCompleted {
			t.Fatalf("%+v %v", res, err)
		}
		evs := events(t, w, task)
		_, d := delegationEvent(t, evs)
		if d == nil {
			t.Fatal("no witness")
		}
		_, ab := delegateAudit(t, evs)
		return d, ab
	}
	fwd, abF := run(t, "t-perm-a", func(a, b string) string { return a + "," + b })
	rev, abR := run(t, "t-perm-b", func(a, b string) string { return b + "," + a })
	if abF["ArgsHash"] == abR["ArgsHash"] {
		t.Fatal("fixture: the two argument orders must differ")
	}
	if fwd.Composition.PayloadHash != rev.Composition.PayloadHash || fwd.Composition.RenderHash != rev.Composition.RenderHash || fwd.Composition.ContractHash != rev.Composition.ContractHash {
		t.Fatalf("argument order changed the composition: %+v vs %+v", fwd.Composition, rev.Composition)
	}
	if len(fwd.EvidenceRefs) != 2 || fwd.EvidenceRefs[0].Seq != rev.EvidenceRefs[0].Seq || fwd.EvidenceRefs[1].Seq != rev.EvidenceRefs[1].Seq {
		t.Fatalf("evidence_refs must be canonical: %+v vs %+v", fwd.EvidenceRefs, rev.EvidenceRefs)
	}
	if fwd.EvidenceRefs[0].ObjectID != fwd.EvidenceRefs[1].ObjectID {
		t.Fatal("fixture: the two references must name one object (equal bytes)")
	}
}

// C-L8-19 A/B: every proposed delegate call consumes the quota at
// proposal — denied, refused, and completed alike.
func TestDelegateQuotaCountsEveryAttempt(t *testing.T) {
	m := &dynModel{delegated: okDelegated}
	w := newWorld(t, m, "", true)
	const task = "t-quota"
	m.parent = func(turn int, conv []model.Message) model.ExecutionResponse {
		switch turn {
		case 1:
			return call("c1", "read_file", `{"path":"go.mod"}`)
		case 2:
			return call("c2", "delegate", `{"template":"cve-analysis@1","brief":"x"}`) // denied: scope
		case 3:
			return call("c3", "delegate", `{"template":"dependency-triage@1","evidence":"999:sha256:`+strings.Repeat("0", 64)+`","brief":"x"}`) // refused: stage B
		case 4:
			seq, id := w.readRef(nil, task)
			return call("c4", "delegate", `{"template":"dependency-triage@1","evidence":"`+ref(seq, id)+`","brief":"x"}`) // completed
		case 5:
			return call("c5", "delegate", `{"template":"dependency-triage@1","brief":"x"}`) // 4th attempt: quota (max_calls 3)
		}
		return call("c6", "declare_done", `{}`)
	}
	if res, err := w.o.SubmitTask(w.envelope(t, task)); err != nil || res.Status != state.StatusCompleted {
		t.Fatalf("%+v %v", res, err)
	}
	var decisions []string
	for _, e := range events(t, w, task) {
		if e.Class != state.EvL4Audit {
			continue
		}
		var b struct{ Tool, Decision, TracePredicate string }
		_ = json.Unmarshal(e.Body, &b)
		if b.Tool == "delegate" {
			decisions = append(decisions, b.Decision+":"+b.TracePredicate)
		}
	}
	if len(decisions) != 4 || !strings.HasPrefix(decisions[0], "denied:") || !strings.HasPrefix(decisions[1], "error:") || !strings.HasPrefix(decisions[2], "authorized") || !strings.Contains(decisions[3], "quota-exhausted") {
		t.Fatalf("denied + refused + completed must each consume the quota: %v", decisions)
	}
	if m.delegCall != 1 {
		t.Fatalf("exactly one model execution, got %d", m.delegCall)
	}
}

// C-L8-18: f is total over the event classes L7/L8 write and maps each
// to the floor or a registry-derived value; no branch reads a
// referenced object's prior class.
func TestClassDerivationIsTotalAndFloorBound(t *testing.T) {
	req := orchestration.InstantiationRequest{RegistryHash: "R", ToolTrust: func(name string) (hctx.AuthorityClass, bool) {
		if name == "get_finding" {
			return hctx.AuthorityGovernedRecord, true
		}
		if name == "read_file" {
			return hctx.AuthorityExternalUntrusted, true
		}
		return "", false
	}}
	obj := "sha256:" + strings.Repeat("a", 64)
	audit := func(tool, decision, reg string) state.Event {
		b, _ := json.Marshal(map[string]string{"Tool": tool, "Decision": decision, "RegistryHash": reg})
		return state.Event{Class: state.EvL4Audit, Seq: 5, Body: b}
	}
	l8 := func(output string) state.Event {
		e := &delegation.Event{ParentCallSeq: 3, Outcome: delegation.OutcomeCompleted, OutputObjectRef: output,
			Template:           delegation.TemplateIdentity{Ref: "x@1", RegistryHash: strings.Repeat("1", 64), TemplateHash: strings.Repeat("2", 64)},
			Composition:        delegation.Composition{EISHash: strings.Repeat("3", 64), ContractHash: strings.Repeat("4", 64), PayloadHash: strings.Repeat("5", 64), RenderHash: strings.Repeat("6", 64), CompositionObjectRef: obj},
			TemplateObjectRefs: []string{obj}}
		b, _ := e.Encode()
		return state.Event{Class: state.EvL8Delegation, Seq: 9, Body: b}
	}
	cases := []struct {
		name  string
		ev    state.Event
		class hctx.AuthorityClass
		kind  string
		want  string // refusal reason when not derivable
	}{
		{"l4-audit governed tool", audit("get_finding", "authorized", "R"), hctx.AuthorityGovernedRecord, "tool:get_finding", ""},
		{"l4-audit untrusted tool", audit("read_file", "authorized", "R"), hctx.AuthorityExternalUntrusted, "tool:read_file", ""},
		{"l4-audit denied", audit("read_file", "denied", "R"), "", "", "evidence-unreachable"},
		{"l4-audit registry drift", audit("read_file", "authorized", "OLD"), "", "", "evidence-registry-drift"},
		{"l4-audit unknown tool", audit("ghost", "authorized", "R"), "", "", "evidence-registry-drift"},
		{"model-turn", state.Event{Class: state.EvModelTurn, Seq: 4, Body: []byte(`{"fact":"tool-calls"}`)}, hctx.AuthorityExternalUntrusted, "model-turn", ""},
		{"l8-delegation output", l8(obj), hctx.AuthorityExternalUntrusted, "delegation-output", ""},
		{"l8-delegation other object", l8("sha256:" + strings.Repeat("b", 64)), "", "", "evidence-unreachable"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			kind, class, err := deriveClass(c.ev, obj, req)
			if c.want == "" {
				if err != nil || class != c.class || kind != c.kind {
					t.Fatalf("%v %s %s", err, class, kind)
				}
				return
			}
			var r *orchestration.DelegationRefusal
			if err == nil || !errors.As(err, &r) || r.Reason != c.want {
				t.Fatalf("want %s: %v", c.want, err)
			}
		})
	}
	// Totality: every other class in the closed vocabulary refuses.
	for _, cls := range []string{state.EvLifecycle, state.EvRecovery, state.EvVerdict, state.EvContamination, state.EvL1Conflict,
		state.EvL2Delivery, state.EvL3Selection, state.EvL5Transition, state.EvL5Op, state.EvArtifact,
		state.EvWorkflowTransition, state.EvL7Invariant, state.EvVerification} {
		if _, _, err := deriveClass(state.Event{Class: cls, Seq: 2, Body: []byte(`{}`)}, obj, req); err == nil {
			t.Fatalf("%s must not witness evidence", cls)
		}
	}
	// Structural: f reads no object and no prior class — it takes the
	// event, the object id, and the request, and calls no store.
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "seam.go", nil, 0)
	if err != nil {
		t.Fatal(err)
	}
	for _, decl := range f.Decls {
		fd, ok := decl.(*ast.FuncDecl)
		if !ok || fd.Name.Name != "deriveClass" {
			continue
		}
		ast.Inspect(fd.Body, func(n ast.Node) bool {
			if c, ok := n.(*ast.CallExpr); ok {
				if sel, ok := c.Fun.(*ast.SelectorExpr); ok {
					switch sel.Sel.Name {
					case "GetObject", "ReadEvents", "Resolve", "HasObject":
						t.Errorf("deriveClass reads the store at %s — class is a function of the witnessing event only", fset.Position(c.Pos()))
					}
				}
			}
			if sel, ok := n.(*ast.SelectorExpr); ok && sel.Sel.Name == "Refs" {
				t.Errorf("deriveClass reads Refs at %s — object class is never authority class", fset.Position(sel.Pos()))
			}
			return true
		})
	}
}

// C-L8-18 A: the conversation after a delegation is a deterministic
// projection of the record — the framed result and the refusal error
// re-derive byte-exactly.
func TestConversationProjectionAfterDelegation(t *testing.T) {
	m := &dynModel{delegated: okDelegated}
	w := newWorld(t, m, "", true)
	const task = "t-proj"
	m.parent = func(turn int, conv []model.Message) model.ExecutionResponse {
		switch turn {
		case 1:
			return call("c1", "read_file", `{"path":"go.mod"}`)
		case 2:
			return call("c2", "delegate", `{"template":"dependency-triage@1","evidence":"999:sha256:`+strings.Repeat("0", 64)+`","brief":"x"}`)
		case 3:
			seq, id := w.readRef(nil, task)
			return call("c3", "delegate", `{"template":"dependency-triage@1","evidence":"`+ref(seq, id)+`","brief":"x"}`)
		}
		return call("c4", "declare_done", `{}`)
	}
	if res, err := w.o.SubmitTask(w.envelope(t, task)); err != nil || res.Status != state.StatusCompleted {
		t.Fatalf("%+v %v", res, err)
	}
	evs := events(t, w, task)
	l8, _ := delegationEvent(t, evs)
	seen := map[string]string{}
	for _, msg := range m.lastConv {
		if msg.Role == model.RoleTool {
			seen[msg.ToolCallID] = msg.Content
		}
	}
	got, err := orchestration.ProjectDelegationMessage(w.sroot, task, l8.Seq, "external-untrusted")
	if err != nil || got != seen["c3"] {
		t.Fatalf("framed result must re-derive byte-exactly:\n%v\n%q\n%q", err, got, seen["c3"])
	}
	for _, e := range evs {
		if e.Class != state.EvL4Audit {
			continue
		}
		var b struct{ Tool, Decision string }
		_ = json.Unmarshal(e.Body, &b)
		if b.Tool == "delegate" && b.Decision == "error" {
			msg, err := orchestration.ProjectRefusalMessage(e)
			if err != nil || msg != seen["c2"] {
				t.Fatalf("refusal must re-derive byte-exactly: %v %q %q", err, msg, seen["c2"])
			}
		}
	}
}

// Register C: without the parent turn object that carried the delegate
// call, the brief cannot be recovered and the re-compose cannot run —
// UNREPRODUCIBLE, never a silent CONFIRMED.
func TestReconstructionWithoutParentTurnIsUnreproducible(t *testing.T) {
	const task = "t-recon-noturn"
	w, _, seq := completedWalk(t, task)
	evs := events(t, w, task)
	var d *delegation.Event
	for _, e := range evs {
		if e.Seq == seq {
			d, _ = delegation.Decode(e.Body)
		}
	}
	var turnObj string
	for _, e := range evs {
		if e.Class == state.EvModelTurn && e.Seq < d.ParentCallSeq && len(e.Refs) == 1 {
			turnObj = e.Refs[0].ID
		}
	}
	removeObject(t, w.stateDir, turnObj)
	r, err := ReconstructDelegation(w.sroot, task, seq, trustFor(t, w))
	if err != nil || r.Verdict != VerdictUnreproducible || !anyContains(r.MissingInputs, "parent turn object") {
		t.Fatalf("%v %+v", err, r)
	}
}

func removeObject(t *testing.T, stateDir, id string) {
	t.Helper()
	hex := strings.TrimPrefix(id, "sha256:")
	removed := false
	_ = filepath.WalkDir(stateDir, func(p string, de os.DirEntry, err error) error {
		if err == nil && !de.IsDir() && strings.Contains(p, hex) {
			removed = os.Remove(p) == nil
		}
		return nil
	})
	if !removed {
		t.Skip("object layout not found on disk")
	}
}

// C-L8-18 A: only an execution error of the closed vocabulary projects
// as a refusal message.
func TestProjectRefusalMessageIsClosed(t *testing.T) {
	ok := state.Event{Seq: 1, Body: []byte(`{"Tool":"delegate","Decision":"error","ErrClass":"delegation-refused:brief-over-bound"}`)}
	if msg, err := orchestration.ProjectRefusalMessage(ok); err != nil || msg != `{"error":"delegation-refused:brief-over-bound"}` {
		t.Fatalf("%v %q", err, msg)
	}
	for _, bad := range []string{
		`{"Tool":"delegate","Decision":"authorized"}`,
		`{"Tool":"delegate","Decision":"error","ErrClass":"delegation-refused:made-up"}`,
		`{"Tool":"delegate","Decision":"denied","DenialClass":"not-available"}`,
	} {
		if _, err := orchestration.ProjectRefusalMessage(state.Event{Seq: 2, Body: []byte(bad)}); err == nil {
			t.Fatalf("must refuse to project %s", bad)
		}
	}
}
