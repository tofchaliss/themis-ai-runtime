package themis

// T-M3 Register A — the decision door's ADMISSIBILITY machinery
// (D-T-1..6), against real harness records produced by real walks.
// Positive path first; then the key negative the owner named: L10
// PASS on report A, egress of report B → `verification-refused:
// verified bytes are not the bound artifact`, with the harness itself
// having COMPLETED the task (the L7 gate is not Themis's evidence);
// then the inverse positive (re-verified after mutation → admitted on
// the second fact); then every other refusal, link-named, and the two
// D-T-6 "proceed and record" rows.

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/tofchaliss/themis-ai-runtime/src/harness/runtime/model"
	"github.com/tofchaliss/themis-ai-runtime/src/harness/state"
	"github.com/tofchaliss/themis-app/intake"
)

const reportB = `{"finding": "FIND-2026-0001: vulnerable-dep v1 in go.mod", "remediation": "bump to v2 (revised after verification)", "evidence": "go.mod updated; tests green"}`

func writeCall(id, path, content string) model.ToolCall {
	args, _ := json.Marshal(map[string]string{"path": path, "content": content})
	return model.ToolCall{ID: id, Name: "write_file", Arguments: args}
}

func verifyCall(id string) model.ToolCall {
	return model.ToolCall{ID: id, Name: "verify_report", Arguments: json.RawMessage(`{"path":"report.json","contract":"report-valid@2"}`)}
}

// walk runs remediate-dependency@3 under an anchored world with the
// given scripted parent and returns the world, the task result, and
// the tuple Themis would be handed (anchor from the deployment, task
// id, the artifact-bound seq found in the record).
func walk(t *testing.T, m *readingParent, task string) (*world, intake.Tuple) {
	t.Helper()
	w, err := newWorld(t, m, storeCopy(t, nil), "", nil)
	if err != nil {
		t.Fatal(err)
	}
	res, err := w.o.SubmitTask(w.instantiate(t, task, "FIND-2026-0001"))
	if err != nil {
		t.Fatalf("walk: %v", err)
	}
	tuple := intake.Tuple{AnchorHash: w.anchorSHA, TaskID: task}
	if res.Status == state.StatusCompleted {
		tuple.ArtifactBoundSeq = bindingSeq(t, w, task)
	}
	return w, tuple
}

func bindingSeq(t *testing.T, w *world, task string) int64 {
	t.Helper()
	evs, err := w.sroot.ReadEvents(task)
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range evs {
		if e.Class == state.EvArtifact {
			return e.Seq
		}
	}
	t.Fatal("no artifact-bound in the record")
	return 0
}

func (w *world) checkout() intake.Checkout {
	return intake.Checkout{AnchorsRegistryPath: w.regs, ContractsRegistryPath: filepath.Join(w.root, "policies/verification/contracts.json")}
}

func recordedOutcomes(t *testing.T, w *world, task string) []string {
	t.Helper()
	evs, _ := w.sroot.ReadEvents(task)
	var out []string
	for _, e := range evs {
		if e.Class == state.EvVerification {
			var b struct {
				Outcome string `json:"outcome"`
			}
			_ = json.Unmarshal(e.Body, &b)
			out = append(out, b.Outcome)
		}
	}
	return out
}

func mustRefuse(t *testing.T, err error, class error, link string) {
	t.Helper()
	if err == nil {
		t.Fatalf("admitted; wanted %v: %s", class, link)
	}
	if !errors.Is(err, class) || !strings.Contains(err.Error(), link) {
		t.Fatalf("wrong refusal: %v\nwanted %v … %q", err, class, link)
	}
}

// THE POSITIVE PATH: a real walk → tuple → admitted, with every
// identity derived from the record, none from the caller.
func TestIntakePositivePath(t *testing.T) {
	m := &readingParent{finding: "FIND-2026-0001"}
	w, tuple := walk(t, m, "t-intake-ok")
	r, err := intake.Resolve(w.sroot, w.checkout(), tuple)
	if err != nil {
		t.Fatalf("admissible execution refused: %v", err)
	}
	man, _ := w.sroot.ReadManifest(tuple.TaskID)
	if r.ArtifactObjectID != man.ArtifactAddrs[0] {
		t.Fatalf("artifact identity derived %s, record projects %v", r.ArtifactObjectID, man.ArtifactAddrs)
	}
	if r.AnchorName != "test-rsys" || r.AnchorVersion != 6 || r.AnchorState != "active" {
		t.Fatalf("anchor: %s@%d %s", r.AnchorName, r.AnchorVersion, r.AnchorState)
	}
	if r.ContractName != "report-valid" || r.ContractVersion != 2 || r.ContractState != "active" {
		t.Fatalf("contract: %s@%d %s", r.ContractName, r.ContractVersion, r.ContractState)
	}
	wantA := `{"finding": "FIND-2026-0001: vulnerable-dep v1 in go.mod", "remediation": "bump to v2", "evidence": "go.mod updated"}`
	if r.VerifiedPath != "report.json" || r.VerifiedHash != hex64([]byte(wantA)) {
		t.Fatalf("verified member: %s %s", r.VerifiedPath, r.VerifiedHash)
	}
	if !(r.VerificationSeq < r.BindingSeq && r.BindingSeq < r.CompletedSeq) || r.VerifierAuditSeq <= 0 {
		t.Fatalf("ordering: verification %d audit %d binding %d completed %d", r.VerificationSeq, r.VerifierAuditSeq, r.BindingSeq, r.CompletedSeq)
	}
	if len(r.ModelTurns) == 0 {
		t.Fatal("evidence view: no model turns derived")
	}
	// The verifier's audit ResultHash is the hash of the verified member
	// (D-T-5 (2)) — the same identity three ways.
	if a := audits(t, w, tuple.TaskID); a["verify_report"]["ResultHash"] != r.VerifiedHash {
		t.Fatalf("audit ResultHash %v ≠ verified member %s", a["verify_report"]["ResultHash"], r.VerifiedHash)
	}
	v := r.View()
	vb, _ := json.Marshal(v)
	if strings.Contains(string(vb), "bump to v2") {
		t.Fatalf("the evidence view must carry identities, never report content:\n%s", vb)
	}
	if v.Verification.Outcome != "PASS" || !v.Verification.Consistent || v.Artifact.ObjectID != r.ArtifactObjectID {
		t.Fatalf("view: %s", vb)
	}
	// The replay names what it witnessed: L6-written links only, until
	// the L5 witness amendment lands (owner classification 2026-09-25).
	if r.ProductionWitness != intake.WitnessL6Only || v.Artifact.ProductionWitness != intake.WitnessL6Only {
		t.Fatalf("production witness claimed: %q", r.ProductionWitness)
	}
}

// THE KEY NEGATIVE (owner, T-M3): verify report A → PASS, then write
// report B over it, then declare done. The harness COMPLETES the task
// (its gate consumed the recorded PASS) and egresses B. Themis refuses:
// the PASS is not about the bound bytes.
func TestIntakeVerifiedBytesAreNotTheBoundArtifact(t *testing.T) {
	m := &readingParent{finding: "FIND-2026-0001", tail: []model.ToolCall{writeCall("c8", "report.json", reportB)}}
	w, tuple := walk(t, m, "t-verify-a-egress-b")
	if tuple.ArtifactBoundSeq == 0 {
		t.Fatal("fixture: the harness did not complete the walk")
	}
	if got := recordedOutcomes(t, w, tuple.TaskID); len(got) != 1 || got[0] != "PASS" {
		t.Fatalf("fixture: recorded outcomes %v", got)
	}
	// The egressed member is B, and it is not what was verified.
	man, _ := w.sroot.ReadManifest(tuple.TaskID)
	art, _ := w.sroot.Store().GetObject(man.ArtifactAddrs[0])
	if !strings.Contains(string(art), "revised after verification") {
		t.Fatal("fixture: the egress does not carry report B")
	}
	_, err := intake.Resolve(w.sroot, w.checkout(), tuple)
	mustRefuse(t, err, intake.ErrVerification, "verified bytes are not the bound artifact")
}

// THE INVERSE POSITIVE: verify A, write B, verify B → PASS, declare.
// Admitted on the SECOND fact, which is about the bound bytes.
func TestIntakeReverifiedAfterMutation(t *testing.T) {
	m := &readingParent{finding: "FIND-2026-0001", tail: []model.ToolCall{writeCall("c8", "report.json", reportB), verifyCall("c9")}}
	w, tuple := walk(t, m, "t-verify-a-b-egress-b")
	if got := recordedOutcomes(t, w, tuple.TaskID); len(got) != 2 {
		t.Fatalf("fixture: recorded outcomes %v", got)
	}
	r, err := intake.Resolve(w.sroot, w.checkout(), tuple)
	if err != nil {
		t.Fatalf("re-verified artifact refused: %v", err)
	}
	if r.VerifiedHash != hex64([]byte(reportB)) {
		t.Fatalf("admitted on the wrong fact: %s", r.VerifiedHash)
	}
	evs, _ := w.sroot.ReadEvents(tuple.TaskID)
	var vseqs []int64
	for _, e := range evs {
		if e.Class == state.EvVerification {
			vseqs = append(vseqs, e.Seq)
		}
	}
	if r.VerificationSeq != vseqs[1] {
		t.Fatalf("admitted on verification seq %d, the fact about B is seq %d", r.VerificationSeq, vseqs[1])
	}
}

func TestIntakeRefusals(t *testing.T) {
	m := &readingParent{finding: "FIND-2026-0001"}
	w, tuple := walk(t, m, "t-intake-neg")
	ck := w.checkout()
	if _, err := intake.Resolve(w.sroot, ck, tuple); err != nil {
		t.Fatalf("positive twin: %v", err)
	}
	evs, _ := w.sroot.ReadEvents(tuple.TaskID)
	seqOf := func(class string) int64 {
		for _, e := range evs {
			if e.Class == class {
				return e.Seq
			}
		}
		t.Fatalf("no %s event", class)
		return 0
	}
	copyWith := func(t *testing.T, src string, edit func(string) string) string {
		t.Helper()
		b, err := os.ReadFile(src)
		if err != nil {
			t.Fatal(err)
		}
		return wj(t, t.TempDir(), filepath.Base(src), edit(string(b)))
	}

	// ---- D-T-1
	t.Run("tuple names a different anchor", func(t *testing.T) {
		tp := tuple
		tp.AnchorHash = strings.Repeat("ab", 32)
		_, err := intake.Resolve(w.sroot, ck, tp)
		mustRefuse(t, err, intake.ErrNotReferencable, "the record identifies deployment anchor")
	})
	t.Run("unknown task", func(t *testing.T) {
		tp := tuple
		tp.TaskID = "t-never-ran"
		_, err := intake.Resolve(w.sroot, ck, tp)
		mustRefuse(t, err, intake.ErrNotReferencable, "unavailable")
	})
	t.Run("malformed handle", func(t *testing.T) {
		_, err := intake.Resolve(w.sroot, ck, intake.Tuple{AnchorHash: "deadbeef", TaskID: tuple.TaskID, ArtifactBoundSeq: 1})
		mustRefuse(t, err, intake.ErrNotReferencable, "well-formed")
		_, err = intake.Resolve(w.sroot, ck, intake.Tuple{AnchorHash: tuple.AnchorHash, TaskID: tuple.TaskID})
		mustRefuse(t, err, intake.ErrNotReferencable, "seq must be positive")
	})
	// ---- D-T-4
	t.Run("seq names a model turn, not the binding", func(t *testing.T) {
		tp := tuple
		tp.ArtifactBoundSeq = seqOf(state.EvModelTurn)
		_, err := intake.Resolve(w.sroot, ck, tp)
		mustRefuse(t, err, intake.ErrProvenance, "not artifact-bound")
	})
	t.Run("seq names the verification, not the binding", func(t *testing.T) {
		tp := tuple
		tp.ArtifactBoundSeq = seqOf(state.EvVerification)
		_, err := intake.Resolve(w.sroot, ck, tp)
		mustRefuse(t, err, intake.ErrProvenance, "is l10-verification, not artifact-bound")
	})
	t.Run("seq beyond the record", func(t *testing.T) {
		tp := tuple
		tp.ArtifactBoundSeq = 100000
		_, err := intake.Resolve(w.sroot, ck, tp)
		mustRefuse(t, err, intake.ErrProvenance, "no event at seq")
	})
	// ---- D-T-2 / D-T-6 anchor rows
	t.Run("anchor never registered on Themis's checkout", func(t *testing.T) {
		regs := copyWith(t, w.regs, func(s string) string { return strings.Replace(s, w.anchorSHA, strings.Repeat("cd", 32), 1) })
		_, err := intake.Resolve(w.sroot, intake.Checkout{AnchorsRegistryPath: regs, ContractsRegistryPath: ck.ContractsRegistryPath}, tuple)
		mustRefuse(t, err, intake.ErrDeployment, "no longer registered")
	})
	t.Run("anchor registered under another identity", func(t *testing.T) {
		regs := copyWith(t, w.regs, func(s string) string { return strings.Replace(s, `"version":6`, `"version":7`, 1) })
		_, err := intake.Resolve(w.sroot, intake.Checkout{AnchorsRegistryPath: regs, ContractsRegistryPath: ck.ContractsRegistryPath}, tuple)
		mustRefuse(t, err, intake.ErrDeployment, "disagrees with its registration")
	})
	t.Run("anchors registry unreadable", func(t *testing.T) {
		_, err := intake.Resolve(w.sroot, intake.Checkout{AnchorsRegistryPath: filepath.Join(t.TempDir(), "none.json"), ContractsRegistryPath: ck.ContractsRegistryPath}, tuple)
		mustRefuse(t, err, intake.ErrDeployment, "")
	})
	t.Run("anchor withdrawn after the execution: proceeds, state recorded", func(t *testing.T) {
		regs := copyWith(t, w.regs, func(s string) string { return strings.Replace(s, `"state":"active"`, `"state":"withdrawn"`, 1) })
		r, err := intake.Resolve(w.sroot, intake.Checkout{AnchorsRegistryPath: regs, ContractsRegistryPath: ck.ContractsRegistryPath}, tuple)
		if err != nil {
			t.Fatalf("withdrawal must not unmake history: %v", err)
		}
		if r.AnchorState != "withdrawn" || r.View().Execution.AnchorState != "withdrawn" {
			t.Fatalf("state at intake: %s", r.AnchorState)
		}
	})
	// ---- D-T-5 / D-T-6 contract rows
	t.Run("contract withdrawn after the execution: proceeds, state recorded", func(t *testing.T) {
		cp := copyWith(t, ck.ContractsRegistryPath, func(s string) string {
			i := strings.LastIndex(s, `"state": "active"`)
			return s[:i] + `"state": "withdrawn"` + s[i+len(`"state": "active"`):]
		})
		r, err := intake.Resolve(w.sroot, intake.Checkout{AnchorsRegistryPath: ck.AnchorsRegistryPath, ContractsRegistryPath: cp}, tuple)
		if err != nil {
			t.Fatalf("reconstruction uses the stored contract bytes, never today's entry: %v", err)
		}
		if r.ContractState != "withdrawn" {
			t.Fatalf("state at intake: %s", r.ContractState)
		}
	})
	t.Run("contract not registered on Themis's checkout", func(t *testing.T) {
		cp := copyWith(t, ck.ContractsRegistryPath, func(s string) string {
			i := strings.Index(s, `,
    {
      "name": "report-valid",
      "version": 2`)
			j := strings.LastIndex(s, "}")
			return s[:i] + "\n  ]\n" + s[j:]
		})
		_, err := intake.Resolve(w.sroot, intake.Checkout{AnchorsRegistryPath: ck.AnchorsRegistryPath, ContractsRegistryPath: cp}, tuple)
		mustRefuse(t, err, intake.ErrVerification, "contract not registered")
	})
	t.Run("contract bytes registered under another identity", func(t *testing.T) {
		cp := copyWith(t, ck.ContractsRegistryPath, func(s string) string {
			i := strings.LastIndex(s, `"version": 2`)
			return s[:i] + `"version": 3` + s[i+len(`"version": 2`):]
		})
		_, err := intake.Resolve(w.sroot, intake.Checkout{AnchorsRegistryPath: ck.AnchorsRegistryPath, ContractsRegistryPath: cp}, tuple)
		mustRefuse(t, err, intake.ErrVerification, "registered as report-valid@3")
	})
	t.Run("contracts registry unreadable", func(t *testing.T) {
		_, err := intake.Resolve(w.sroot, intake.Checkout{AnchorsRegistryPath: ck.AnchorsRegistryPath, ContractsRegistryPath: filepath.Join(t.TempDir(), "none.json")}, tuple)
		mustRefuse(t, err, intake.ErrVerification, "contracts registry unreadable")
	})
}

// D-T-6 rows that need their own record: not completed; evidence
// missing; record corrupt. Each mutates its OWN world.
func TestIntakeUnavailabilityFailsClosed(t *testing.T) {
	t.Run("task did not complete (report invalid, gate never satisfied)", func(t *testing.T) {
		m := &readingParent{finding: "FIND-2026-0001", report: `{"finding": "x", "remediation": "y"}`}
		w, err := newWorld(t, m, storeCopy(t, nil), "", nil)
		if err != nil {
			t.Fatal(err)
		}
		res, _ := w.o.SubmitTask(w.instantiate(t, "t-failed", "FIND-2026-0001"))
		if res.Status == state.StatusCompleted {
			t.Fatal("fixture: an invalid report must not complete")
		}
		_, rerr := intake.Resolve(w.sroot, w.checkout(), intake.Tuple{AnchorHash: w.anchorSHA, TaskID: "t-failed", ArtifactBoundSeq: 1})
		mustRefuse(t, rerr, intake.ErrNotReferencable, "not a completed execution")
	})
	t.Run("artifact object corrupted on disk", func(t *testing.T) {
		w, tuple := walk(t, &readingParent{finding: "FIND-2026-0001"}, "t-art-corrupt")
		man, _ := w.sroot.ReadManifest(tuple.TaskID)
		corruptObject(t, w, man.ArtifactAddrs[0])
		_, err := intake.Resolve(w.sroot, w.checkout(), tuple)
		if err == nil {
			t.Fatal("corrupt artifact admitted")
		}
		if !errors.Is(err, intake.ErrProvenance) && !errors.Is(err, intake.ErrNotReferencable) {
			t.Fatalf("wrong class: %v", err)
		}
	})
	t.Run("verification raw evidence removed", func(t *testing.T) {
		w, tuple := walk(t, &readingParent{finding: "FIND-2026-0001"}, "t-ev-missing")
		a := audits(t, w, tuple.TaskID)
		removeObject(t, w, "sha256:"+a["verify_report"]["ResultHash"].(string))
		_, err := intake.Resolve(w.sroot, w.checkout(), tuple)
		if err == nil {
			t.Fatal("missing evidence admitted")
		}
		if !errors.Is(err, intake.ErrVerification) && !errors.Is(err, intake.ErrNotReferencable) {
			t.Fatalf("wrong class: %v", err)
		}
	})
	t.Run("event stream torn", func(t *testing.T) {
		w, tuple := walk(t, &readingParent{finding: "FIND-2026-0001"}, "t-torn")
		p := filepath.Join(w.base, "state", "tasks", tuple.TaskID, "events.log")
		b, _ := os.ReadFile(p)
		if err := os.WriteFile(p, b[:len(b)-7], 0o644); err != nil {
			t.Fatal(err)
		}
		_, err := intake.Resolve(w.sroot, w.checkout(), tuple)
		mustRefuse(t, err, intake.ErrNotReferencable, "")
	})
}

func objectPath(w *world, id string) string {
	h := strings.TrimPrefix(id, "sha256:")
	return filepath.Join(w.base, "state", "objects", "sha256", h[:2], h)
}

func corruptObject(t *testing.T, w *world, id string) {
	t.Helper()
	p := objectPath(w, id)
	if err := os.Chmod(p, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(`{"changes":[]}`), 0o644); err != nil {
		t.Fatal(err)
	}
}

func removeObject(t *testing.T, w *world, id string) {
	t.Helper()
	if err := os.Remove(objectPath(w, id)); err != nil {
		t.Fatal(err)
	}
}

// ---- Forged records: Themis must not trust a CLAIM in the record.
//
// A real walk produces task A. A second task F is built from A's own
// durable objects through the L6 primitives (the only way a record
// comes to exist), with ONE alteration. The un-altered forge is the
// positive twin: it proves the fixture builds an admissible record, so
// each refusal is the alteration's, not the fixture's.

type forge struct {
	invalidRaw      bool // PASS claimed over an invalid report; audit and egress agree with the bytes
	auditResultHash string
	recordOutcome   string
}

func forgeRecord(t *testing.T, w *world, from string, task string, f forge) intake.Tuple {
	t.Helper()
	evs, err := w.sroot.ReadEvents(from)
	if err != nil {
		t.Fatal(err)
	}
	man, _ := w.sroot.ReadManifest(from)
	st := w.sroot.Store()
	get := func(id string) []byte {
		b, err := st.GetObject(id)
		if err != nil {
			t.Fatal(err)
		}
		return b
	}
	put := func(b []byte) string {
		id, err := st.StoreObject(state.ObjEvidencePayload, b)
		if err != nil {
			t.Fatal(err)
		}
		return id
	}
	var materialized, audit, verif state.Event
	for _, e := range evs {
		switch {
		case e.Class == state.EvL2Delivery && strings.Contains(string(e.Body), "materialized-governed-artifacts"):
			materialized = e
		case e.Class == state.EvL4Audit && strings.Contains(string(e.Body), `"Tool":"verify_report"`):
			audit = e
		case e.Class == state.EvVerification:
			verif = e
		}
	}
	var vbody struct {
		Contract, Outcome, Record string
	}
	_ = json.Unmarshal(verif.Body, &vbody)
	var record map[string]any
	_ = json.Unmarshal(get(vbody.Record), &record)
	raw := get("sha256:" + record["raw_object_id"].(string))
	var manifest map[string]any
	_ = json.Unmarshal(get(man.ArtifactAddrs[0]), &manifest)

	// The alteration.
	if f.invalidRaw {
		raw = []byte(`{"finding": "FIND-2026-0001: forged", "remediation": "none"}`) // no "evidence"
		record["raw_object_id"] = hex64(raw)
		ch := manifest["changes"].([]any)[0].(map[string]any)
		ch["content"], ch["new_hash"], ch["size"] = string(raw), hex64(raw), len(raw)
	}
	if f.recordOutcome != "" {
		record["outcome"] = f.recordOutcome
	}
	var ab map[string]any
	_ = json.Unmarshal(audit.Body, &ab)
	ab["ResultHash"] = hex64(raw)
	if f.auditResultHash != "" {
		ab["ResultHash"] = f.auditResultHash
	}

	tr, err := w.sroot.CreateTask(task, state.TaskOptions{GovernedHashes: man.GovernedHashes})
	if err != nil {
		t.Fatal(err)
	}
	defer tr.Close()
	if err := tr.Transition(state.StatusRunning, "forged"); err != nil {
		t.Fatal(err)
	}
	if _, err := tr.AppendEvent(materialized.Class, materialized.Writer, materialized.Body, materialized.Refs...); err != nil {
		t.Fatal(err)
	}
	abody, _ := json.Marshal(ab)
	aev, err := tr.AppendEvent(state.EvL4Audit, "l4", abody, audit.Refs...)
	if err != nil {
		t.Fatal(err)
	}
	record["execution_ref"] = "l4:" + itoa(aev.Seq)
	rb, _ := json.Marshal(record)
	rawID := put(raw)
	recID := put(rb)
	refs := []state.Ref{{ID: "sha256:" + record["contract_sha256"].(string), Class: state.ObjEvidencePayload}, {ID: rawID, Class: state.ObjEvidencePayload},
		{ID: "sha256:" + record["canonical_object_id"].(string), Class: state.ObjEvidencePayload}, {ID: recID, Class: state.ObjEvidencePayload}}
	vb, _ := json.Marshal(map[string]string{"contract": vbody.Contract, "outcome": vbody.Outcome, "record": recID})
	if _, err := tr.AppendEvent(state.EvVerification, "l10", vb, refs...); err != nil {
		t.Fatal(err)
	}
	manifest["task_id"] = task
	mb, _ := json.Marshal(manifest)
	artID, err := tr.StoreObject(state.ObjEgressArtifact, mb)
	if err != nil {
		t.Fatal(err)
	}
	if err := tr.BindArtifact(artID); err != nil {
		t.Fatal(err)
	}
	if err := tr.Transition(state.StatusCompleted, "forged"); err != nil {
		t.Fatal(err)
	}
	return intake.Tuple{AnchorHash: w.anchorSHA, TaskID: task, ArtifactBoundSeq: bindingSeq(t, w, task)}
}

func itoa(n int64) string { return strconv.FormatInt(n, 10) }

func TestIntakeRefusesClaims(t *testing.T) {
	w, _ := walk(t, &readingParent{finding: "FIND-2026-0001"}, "t-real")
	ck := w.checkout()
	t.Run("positive twin: the un-altered forge is admissible", func(t *testing.T) {
		tuple := forgeRecord(t, w, "t-real", "t-forge-ok", forge{})
		r, err := intake.Resolve(w.sroot, ck, tuple)
		if err != nil {
			t.Fatalf("fixture: %v", err)
		}
		if r.VerifiedPath != "report.json" {
			t.Fatalf("%+v", r.View())
		}
	})
	t.Run("PASS claimed over an invalid report: the reconstruction, not the claim, decides", func(t *testing.T) {
		// Event says PASS, record says PASS, audit and egress agree with
		// the bytes — only re-running the canonicalization over the raw
		// bytes shows the claim is false.
		tuple := forgeRecord(t, w, "t-real", "t-forge-claim", forge{invalidRaw: true})
		_, err := intake.Resolve(w.sroot, ck, tuple)
		mustRefuse(t, err, intake.ErrVerification, "no reproducible PASS (reconstruction inconsistent: canonicalization")
	})
	t.Run("record outcome disagrees with the event", func(t *testing.T) {
		tuple := forgeRecord(t, w, "t-real", "t-forge-outcome", forge{recordOutcome: "FAIL"})
		_, err := intake.Resolve(w.sroot, ck, tuple)
		mustRefuse(t, err, intake.ErrVerification, "event_outcome")
	})
	t.Run("verifier audit did not produce the raw bytes", func(t *testing.T) {
		tuple := forgeRecord(t, w, "t-real", "t-forge-audit", forge{auditResultHash: strings.Repeat("ef", 32)})
		_, err := intake.Resolve(w.sroot, ck, tuple)
		mustRefuse(t, err, intake.ErrVerification, "verifier execution not authorized (audit ResultHash")
	})
}

// ---- W-M2 Register B: a REAL walk's stream carries the full L5 machine
// and every governed op, written by l5 through the handle, in the
// order D-W-2/D-W-3 fix, with the egress acknowledgement naming the
// address the L6 binding then names (same digest, "sha256:" prefixed —
// the RawObjectID precedent) and every L5 witness preceding the
// binding and COMPLETED.
func TestRealWalkCarriesL5Witnesses(t *testing.T) {
	w, tuple := walk(t, &readingParent{finding: "FIND-2026-0001"}, "t-l5-witness")
	evs, err := w.sroot.ReadEvents(tuple.TaskID)
	if err != nil {
		t.Fatal(err)
	}
	var edges []string
	var egressAck struct {
		seq  int64
		addr string
	}
	ops, provisionOps, activeOps, egressOps := 0, 0, 0, 0
	var boundSeq, completedSeq int64
	var boundAddr string
	for _, e := range evs {
		switch e.Class {
		case state.EvL5Transition, state.EvL5Op:
			if e.Writer != "l5" {
				t.Fatalf("seq %d %s writer %q", e.Seq, e.Class, e.Writer)
			}
		}
		switch e.Class {
		case state.EvL5Transition:
			var b struct{ From, To string }
			_ = json.Unmarshal(e.Body, &b)
			edges = append(edges, b.From+">"+b.To)
		case state.EvL5Op:
			var b struct {
				Op, Phase, Outcome, Address string `json:"-"`
			}
			var m map[string]any
			_ = json.Unmarshal(e.Body, &m)
			if m["op"] == "egress" {
				if m["outcome"] == "acknowledged" {
					egressAck.seq, egressAck.addr = e.Seq, m["artifact_address"].(string)
				}
				continue
			}
			ops++
			switch m["phase"] {
			case "provision":
				provisionOps++
			case "active":
				activeOps++
			case "egress":
				egressOps++
			}
			_ = b
		case state.EvArtifact:
			boundSeq, boundAddr = e.Seq, e.Refs[0].ID
		case state.EvLifecycle:
			var lb struct{ To string }
			_ = json.Unmarshal(e.Body, &lb)
			if lb.To == string(state.StatusCompleted) {
				completedSeq = e.Seq
			}
		}
	}
	// The full machine, in order (teardown edges follow the artifact
	// binding but precede COMPLETED — the walk tears down before it
	// transitions the task).
	want := []string{"PROVISIONING>ACTIVE", "ACTIVE>SEALED", "SEALED>EGRESSING", "EGRESSING>ACKNOWLEDGED", "ACKNOWLEDGED>TEARDOWN", "TEARDOWN>DESTROYED"}
	if strings.Join(edges, "|") != strings.Join(want, "|") {
		t.Fatalf("L5 edges: %v", edges)
	}
	// Provisioning and egress always run governed git ops; the ACTIVE
	// phase of this workflow uses in-process file tools only, so it may
	// legitimately witness zero subprocess ops — a fact about the
	// skill, not a gap (D-W-3 witnesses subprocesses, not tool calls).
	if provisionOps == 0 || egressOps == 0 {
		t.Fatalf("ops witnessed: provision=%d active=%d egress=%d (total %d)", provisionOps, activeOps, egressOps, ops)
	}
	if egressAck.seq == 0 {
		t.Fatal("no acknowledged egress witness")
	}
	if "sha256:"+egressAck.addr != boundAddr {
		t.Fatalf("egress acknowledged %s, binding names %s", egressAck.addr, boundAddr)
	}
	if !(egressAck.seq < boundSeq && boundSeq < completedSeq) {
		t.Fatalf("ordering: egress ack %d, bound %d, completed %d", egressAck.seq, boundSeq, completedSeq)
	}
	// The five links D-W-5 will replay all exist in this record.
	if !strings.Contains(strings.Join(edges, "|"), "ACTIVE>SEALED") {
		t.Fatal("seal edge missing")
	}
}
