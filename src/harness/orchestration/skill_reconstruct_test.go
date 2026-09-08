package orchestration

// Register D (L9): cold reconstruction. Given only the durable record,
// the catalog, and the pinned artifact bytes, the exact reviewed
// composition that executed must be recoverable — and any
// inconsistency between the declared composition and the independently
// recorded artifact identities must be deterministically DETECTABLE
// (D-L9-11 attribution honesty; detection of inconsistency, never an
// attribution of intent).

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	l2 "github.com/tofchaliss/themis/context"
	"github.com/tofchaliss/themis/skills"
	"github.com/tofchaliss/themis/state"
)

// From record + catalog + bytes alone: which reviewed composition ran?
func TestColdReconstructionOfExecutedComposition(t *testing.T) {
	f := setup(t, happyScript(), "")
	env := instantiateP0(t, f, "t-cold", skills.Request{})
	if _, err := f.o.SubmitTask(env); err != nil {
		t.Fatal(err)
	}

	// COLD: everything below reads only the durable record and the
	// governed catalog — no in-memory state from the run above.
	man, err := f.o.root.ReadManifest("t-cold")
	if err != nil {
		t.Fatal(err)
	}
	ref := man.GovernedHashes["origin:skill"]
	declared := man.GovernedHashes["origin:skill_composition"]
	if ref == "" || declared == "" {
		t.Fatal("the record must name the skill and its composition")
	}

	catPath := mustAbs(t, filepath.Join(repoRoot, "policies/skills/catalog.proposed.json"))
	cat, err := skills.LoadCatalog(catPath)
	if err != nil {
		t.Fatal(err)
	}
	entry, m, err := cat.Resolve(ref)
	if err != nil {
		t.Fatalf("the executed skill must still resolve from the catalog: %v", err)
	}
	// The catalog's registered composition must equal what the record
	// says executed.
	if entry.Composition != declared {
		t.Fatalf("declared composition %s does not match the registered %s", declared, entry.Composition)
	}

	// The substitution gap closes only when the catalog's pins are
	// compared against what L7 recorded INDEPENDENTLY — L7 hashes the
	// bytes it actually loaded, with no knowledge of the skill. An
	// earlier version of this test compared origin:skill_* (written by
	// L9 from the manifest pin) against the same manifest pin: x == x,
	// which could never fail. These comparisons can.
	for label, pair := range map[string][2]string{
		"workflow":         {m.Workflow.SHA256, man.GovernedHashes["workflow"]},
		"workflow_ceiling": {m.WorkflowCeiling.SHA256, man.GovernedHashes["workflow_ceiling"]},
		"context_contract": {m.ContextContract.SHA256, man.GovernedHashes["context_contract"]},
	} {
		if pair[0] == "" || pair[1] == "" {
			t.Fatalf("%s: both identities must be present to compare (%q vs %q)", label, pair[0], pair[1])
		}
		if pair[0] != pair[1] {
			t.Errorf("%s drifted between review and execution: catalog pin %s vs L7-recorded %s", label, pair[0], pair[1])
		}
	}

	// The pins must also match the bytes on disk right now, so the
	// three-way agreement (catalog pin = L7's independent hash = the
	// reviewed bytes) is what actually holds.
	for label, pin := range map[string]struct{ path, sha string }{
		"workflow":         {m.Workflow.Path, m.Workflow.SHA256},
		"workflow_ceiling": {m.WorkflowCeiling.Path, m.WorkflowCeiling.SHA256},
		"context_contract": {m.ContextContract.Path, m.ContextContract.SHA256},
		"procedure":        {m.Procedure.Path, m.Procedure.SHA256},
		"input_schema":     {m.InputSchema.Path, m.InputSchema.SHA256},
	} {
		body, err := os.ReadFile(filepath.Join(m.Dir, pin.path))
		if err != nil {
			t.Fatalf("%s: %v", label, err)
		}
		if hashBytes(body) != pin.sha {
			t.Errorf("%s bytes on disk do not match the reviewed pin", label)
		}
	}
}

// D-L9-11a claim 1 — EXECUTION INTEGRITY, REFUSED at assembly.
// An envelope whose composition commits to an identity the materialized
// artifact does not have is refused before the task runs. This is the
// property C2 closes: L7's filesystem read is materialization of an
// already-selected artifact, not selection of one.
func TestCompositionArtifactMismatchIsRefused(t *testing.T) {
	f := setup(t, happyScript(), "")
	base := f.envelope(t, "t-cmis")
	// Commit to a workflow identity that is not the workflow named.
	bad := withEnvelopeFields(t, base, map[string]any{
		"composition": map[string]string{
			"workflow_sha256":         strings.Repeat("d", 64),
			"workflow_ceiling_sha256": strings.Repeat("e", 64),
			"context_contract_sha256": strings.Repeat("f", 64),
		},
	}, "envelope-t-cmis-2.json")
	res, err := f.o.SubmitTask(bad)
	if err == nil {
		t.Fatal("an artifact that does not match its committed identity must be refused, not executed")
	}
	if !errors.Is(err, ErrInvariant) {
		t.Fatalf("the mismatch must be an invariant failure: %v", err)
	}
	if res.Status == state.StatusCompleted {
		t.Fatal("a refused composition must never reach a completed terminal")
	}
	// A path named without its committed identity is not a reference.
	partial := withEnvelopeFields(t, base, map[string]any{
		"composition": map[string]string{"workflow_sha256": strings.Repeat("d", 64)},
	}, "envelope-t-cmis-3.json")
	if _, err := f.o.SubmitTask(partial); err == nil ||
		!strings.Contains(err.Error(), "sha256 identity") {
		t.Fatalf("an incomplete composition must refuse at load: %v", err)
	}
}

// D-L9-11a claim 2 — GOVERNANCE IDENTITY, NOT established by L7.
// An internally consistent composition is executed: every artifact
// matches what the submission committed to. L7 holds no governance
// attestation in v1 (C3 is a residual), so it makes NO claim that the
// submitted composition is the registered one. That mismatch is
// recoverable post-hoc from record + catalog — detection, not refusal.
func TestGovernanceIdentityIsNotEstablishedBySelfDeclaredAttribution(t *testing.T) {
	f := setup(t, happyScript(), "")
	base := f.envelope(t, "t-selfdecl")
	// Consistent by construction: commit to the identities of the
	// artifacts this envelope actually names, while CLAIMING to be the
	// governed investigate-cve@1 composition.
	ceiling, _ := LoadWorkflowCeiling(filepath.Join(f.envDir, "wceiling.json"))
	wf, err := LoadWorkflow(filepath.Join(f.envDir, "workflow.json"), ceiling)
	if err != nil {
		t.Fatal(err)
	}
	probe, err := LoadEnvelope(f.envelope(t, "t-selfdecl-probe"))
	if err != nil {
		t.Fatal(err)
	}
	contract, err := l2.LoadContract(probe.ContextContractPath)
	if err != nil {
		t.Fatal(err)
	}
	catPath := mustAbs(t, filepath.Join(repoRoot, "policies/skills/catalog.proposed.json"))
	cat, err := skills.LoadCatalog(catPath)
	if err != nil {
		t.Fatal(err)
	}
	entry, _, err := cat.Resolve("investigate-cve@1")
	if err != nil {
		t.Fatal(err)
	}
	forged := withEnvelopeFields(t, base, map[string]any{
		"composition": map[string]string{
			"workflow_sha256":         wf.Hash,
			"workflow_ceiling_sha256": ceiling.Hash,
			"context_contract_sha256": contract.Hash,
		},
		"origin": map[string]string{
			"skill":             "investigate-cve@1",
			"skill_composition": entry.Composition,
		},
	}, "envelope-t-selfdecl-2.json")

	// v1 posture: this EXECUTES. L7 verified integrity, which holds.
	res, err := f.o.SubmitTask(forged)
	if err != nil || res.Status != state.StatusCompleted {
		t.Fatalf("an internally consistent composition executes — L7 checks integrity, not admission: %v %+v", err, res)
	}
	// But the governance claim is false, and that is discoverable from
	// record + catalog: the registered composition's workflow identity
	// is not the one this task executed.
	man, _ := f.o.root.ReadManifest("t-selfdecl")
	_, registered, err := cat.Resolve(man.GovernedHashes["origin:skill"])
	if err != nil {
		t.Fatal(err)
	}
	if registered.Workflow.SHA256 == man.GovernedHashes["workflow"] {
		t.Fatal("this fixture should NOT have executed the registered skill's workflow")
	}
	// The honest statement of what v1 provides.
	if man.GovernedHashes["origin:skill_composition"] != entry.Composition {
		t.Fatal("the self-declared claim is recorded verbatim, so an auditor can compare it")
	}
}

// Inconsistency is detectable from the record alone: an envelope
// declaring a composition whose artifact identities do not match what
// L7 independently recorded can be caught after the fact.
func TestAttributionInconsistencyIsDetectable(t *testing.T) {
	f := setup(t, happyScript(), "")
	base := f.envelope(t, "t-incons")
	// Claim the real skill's identity on an envelope that executes the
	// FIXTURE's workflow instead.
	catPath := mustAbs(t, filepath.Join(repoRoot, "policies/skills/catalog.proposed.json"))
	cat, err := skills.LoadCatalog(catPath)
	if err != nil {
		t.Fatal(err)
	}
	entry, m, err := cat.Resolve("investigate-cve@1")
	if err != nil {
		t.Fatal(err)
	}
	lying := withEnvelopeFields(t, base, map[string]any{
		"origin": map[string]string{
			"skill":             "investigate-cve@1",
			"skill_composition": entry.Composition,
			"skill_workflow":    m.Workflow.SHA256,
		},
	}, "envelope-t-incons-2.json")
	res, err := f.o.SubmitTask(lying)
	if err != nil || res.Status != state.StatusCompleted {
		t.Fatalf("L7 executes the envelope regardless of its claim: %v %+v", err, res)
	}
	// After the fact: the claimed skill workflow and the workflow L7
	// actually recorded disagree. Detection, not prevention — L7 stays
	// skill-blind by design.
	man, _ := f.o.root.ReadManifest("t-incons")
	if man.GovernedHashes["origin:skill_workflow"] == man.GovernedHashes["workflow"] {
		t.Fatal("this fixture should NOT have executed the skill's workflow")
	}
	// The inconsistency is exactly what a verifier would flag.
	if man.GovernedHashes["origin:skill_workflow"] == "" {
		t.Fatal("the claim must be recorded for the comparison to be possible")
	}
}
