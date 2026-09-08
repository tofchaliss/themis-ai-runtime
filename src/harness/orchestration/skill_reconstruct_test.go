package orchestration

// Register D (L9): cold reconstruction. Given only the durable record,
// the catalog, and the pinned artifact bytes, the exact reviewed
// composition that executed must be recoverable — and any
// inconsistency between the declared composition and the independently
// recorded artifact identities must be deterministically DETECTABLE
// (D-L9-11 attribution honesty; detection of inconsistency, never an
// attribution of intent).

import (
	"os"
	"path/filepath"
	"testing"

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

	// And the composition's own pins must equal the artifact identities
	// L7 recorded independently at assembly — this is the check that
	// closes the substitution gap between catalog resolution and
	// execution.
	for label, pair := range map[string][2]string{
		"workflow":         {m.Workflow.SHA256, man.GovernedHashes["origin:skill_workflow"]},
		"workflow_ceiling": {m.WorkflowCeiling.SHA256, man.GovernedHashes["origin:skill_workflow_ceiling"]},
		"context_contract": {m.ContextContract.SHA256, man.GovernedHashes["origin:skill_context_contract"]},
		"input_schema":     {m.InputSchema.SHA256, man.GovernedHashes["origin:skill_input_schema"]},
		"procedure":        {m.Procedure.SHA256, man.GovernedHashes["origin:skill_procedure"]},
	} {
		if pair[0] != pair[1] {
			t.Errorf("%s identity drifted: catalog pin %s vs recorded %s", label, pair[0], pair[1])
		}
	}

	// The record's own governed workflow hash must equal the reviewed
	// workflow's bytes: what was reviewed is what L7 actually executed.
	wfBytes, err := os.ReadFile(filepath.Join(m.Dir, m.Workflow.Path))
	if err != nil {
		t.Fatal(err)
	}
	if man.GovernedHashes["workflow"] != hashBytes(wfBytes) {
		t.Fatal("the executed workflow is not the reviewed workflow")
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
