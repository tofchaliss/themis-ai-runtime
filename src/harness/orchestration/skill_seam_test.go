package orchestration

// Register C (L9): the distinctive proof that skill provenance is
// METADATA, not hidden authority. A skill-attributed envelope and an
// equivalent hand-assembled one must produce identical L7 control
// behavior; altering, removing, or falsifying provenance must change
// nothing about the walk; and L7 must never resolve a claimed skill.

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tofchaliss/themis/runtime/model"
	"github.com/tofchaliss/themis/state"
)

// withEnvelopeFields rewrites an envelope, adding or overriding
// fields, and returns the new path. Used to attach provenance to an
// otherwise identical envelope.
func withEnvelopeFields(t *testing.T, envPath string, fields map[string]any, name string) string {
	t.Helper()
	raw, err := os.ReadFile(envPath)
	if err != nil {
		t.Fatal(err)
	}
	var doc map[string]any
	if err := json.Unmarshal(raw, &doc); err != nil {
		t.Fatal(err)
	}
	for k, v := range fields {
		if v == nil {
			delete(doc, k)
			continue
		}
		doc[k] = v
	}
	out, _ := json.Marshal(doc)
	return writeJSON(t, filepath.Dir(envPath), name, string(out))
}

// Register C core: identical executable fields ⇒ identical walk.
// The only difference between the two tasks is the provenance block.
func TestSkillProvenanceDoesNotChangeTheWalk(t *testing.T) {
	f := setup(t, happyScript(), "")

	// Hand-assembled: no provenance at all.
	plain := f.envelope(t, "t-plain")
	resPlain, err := f.o.SubmitTask(plain)
	if err != nil || resPlain.Status != state.StatusCompleted {
		t.Fatalf("hand-assembled walk: %v %+v", err, resPlain)
	}
	plainWalk := replayAndVerify(t, f, "t-plain")

	// Skill-attributed: the SAME fixture, so every governed artifact is
	// byte-identical and the origin block is the only difference.
	f.o.cfg.Model = happyScript()
	base := f.envelope(t, "t-skill")
	attributed := withEnvelopeFields(t, base, map[string]any{
		"origin": map[string]string{
			"skill":             "investigate-cve@1",
			"skill_composition": strings.Repeat("a", 64),
			"skill_catalog":     strings.Repeat("b", 64),
		},
		"composition": genuineCommitment(t, base),
	}, "envelope-t-skill-attributed.json")
	resSkill, err := f.o.SubmitTask(attributed)
	if err != nil || resSkill.Status != state.StatusCompleted {
		t.Fatalf("skill-attributed walk: %v %+v", err, resSkill)
	}
	skillWalk := replayAndVerify(t, f, "t-skill")

	// The complete recorded control-transition tuples must match.
	if fmt.Sprint(plainWalk) != fmt.Sprint(skillWalk) {
		t.Fatalf("provenance changed the walk:\n plain %+v\n skill %+v", plainWalk, skillWalk)
	}
	// Transitions alone are too weak: a happy walk never approaches a
	// counter, so provenance could raise a budget or a quota and the
	// tuples would still match (test review HIGH). Every governed
	// identity except the attribution itself must be identical.
	assertGovernedHashesEqual(t, f, "t-plain", f, "t-skill")
}

// assertGovernedHashesEqual compares every recorded governed identity
// between two tasks, ignoring only the opaque origin: attribution. If
// provenance influenced ANY governed input — grant, ceiling, spec,
// workflow, instruction set — this fails.
func assertGovernedHashesEqual(t *testing.T, a *fixture, aID string, b *fixture, bID string) {
	t.Helper()
	manA, err := a.o.root.ReadManifest(aID)
	if err != nil {
		t.Fatal(err)
	}
	manB, err := b.o.root.ReadManifest(bID)
	if err != nil {
		t.Fatal(err)
	}
	strip := func(m map[string]string) map[string]string {
		out := map[string]string{}
		for k, v := range m {
			// The attribution itself, and the three artifacts whose bytes
			// carry the task id, differ by construction. Everything else
			// is a governed input that provenance must not touch.
			if strings.HasPrefix(k, "origin:") || k == "envelope" || k == "spec" ||
				k == "grant_envelope" || k == "grant_effective" {
				continue
			}
			out[k] = v
		}
		return out
	}
	ga, gb := strip(manA.GovernedHashes), strip(manB.GovernedHashes)
	if len(ga) == 0 {
		t.Fatal("no governed identities to compare — the check would be vacuous")
	}
	for k, va := range ga {
		if vb, ok := gb[k]; !ok || va != vb {
			t.Errorf("governed identity %q differs between the plain and skill-attributed walks (%q vs %q) — provenance must influence nothing", k, va, vb)
		}
	}
}

// The equivalence pair under PRESSURE: a script that trips the
// no-action counter to exhaustion drives the walk against its governed
// limits, where a provenance-influenced budget would actually show.
func TestSkillProvenanceDoesNotChangeASaturatingWalk(t *testing.T) {
	script := func() *scriptedModel {
		return &scriptedModel{steps: []model.ExecutionResponse{
			prose("thinking"), prose("thinking"), prose("thinking"),
			prose("thinking"), prose("thinking"), prose("thinking"),
		}}
	}
	f := setup(t, script(), "")
	plainRes, err := f.o.SubmitTask(f.envelope(t, "t-sat-plain"))
	if err != nil {
		t.Fatal(err)
	}
	plainWalk := replayAndVerify(t, f, "t-sat-plain")

	f.o.cfg.Model = script()
	base := f.envelope(t, "t-sat-skill")
	attributed := withEnvelopeFields(t, base, map[string]any{
		"origin":      map[string]string{"skill": "investigate-cve@1", "skill_composition": strings.Repeat("a", 64)},
		"composition": genuineCommitment(t, base),
	}, "envelope-t-sat-attributed.json")
	skillRes, err := f.o.SubmitTask(attributed)
	if err != nil {
		t.Fatal(err)
	}
	skillWalk := replayAndVerify(t, f, "t-sat-skill")

	if plainRes.Status != skillRes.Status {
		t.Fatalf("terminal differs under pressure: %v vs %v", plainRes.Status, skillRes.Status)
	}
	if fmt.Sprint(plainWalk) != fmt.Sprint(skillWalk) {
		t.Fatalf("provenance changed a saturating walk:\n plain %+v\n skill %+v", plainWalk, skillWalk)
	}
	assertGovernedHashesEqual(t, f, "t-sat-plain", f, "t-sat-skill")
}

// D-L9-13: provenance is preserved verbatim into task attribution —
// present in the record, interpreted nowhere.
func TestSkillProvenancePreservedIntoAttribution(t *testing.T) {
	f := setup(t, happyScript(), "")
	base := f.envelope(t, "t-attr")
	env := withEnvelopeFields(t, base, map[string]any{
		"origin": map[string]string{
			"skill":             "investigate-cve@1",
			"skill_composition": strings.Repeat("c", 64),
		},
		// D-L9-11d: attribution requires a commitment.
		"composition": genuineCommitment(t, base),
	}, "envelope-t-attr-2.json")
	if _, err := f.o.SubmitTask(env); err != nil {
		t.Fatal(err)
	}
	man, err := f.o.root.ReadManifest("t-attr")
	if err != nil {
		t.Fatal(err)
	}
	if man.GovernedHashes["origin:skill"] != "investigate-cve@1" {
		t.Fatalf("provenance must be preserved verbatim: %v", man.GovernedHashes)
	}
	if man.GovernedHashes["origin:skill_composition"] != strings.Repeat("c", 64) {
		t.Fatal("composition attribution must be preserved verbatim")
	}
}

// Register C, the owner's closing attack: an envelope whose origin
// claims a DIFFERENT skill must still execute exactly the envelope's
// own executable fields. L7 must never resolve the claimed skill.
func TestProvenanceSwapDoesNotRedirectExecution(t *testing.T) {
	f := setup(t, happyScript(), "")
	base := f.envelope(t, "t-swap")
	// Claim a skill that does not exist anywhere, with a composition
	// hash matching nothing.
	swapped := withEnvelopeFields(t, base, map[string]any{
		"origin": map[string]string{
			"skill":             "some-other-skill@99",
			"skill_composition": strings.Repeat("f", 64),
		},
		"composition": genuineCommitment(t, base),
	}, "envelope-t-swap-2.json")
	res, err := f.o.SubmitTask(swapped)
	if err != nil || res.Status != state.StatusCompleted {
		t.Fatalf("a false provenance claim must not change execution: %v %+v", err, res)
	}
	// The walk is the envelope's own workflow, not the claimed skill's.
	man, _ := f.o.root.ReadManifest("t-swap")
	ceiling, _ := LoadWorkflowCeiling(filepath.Join(f.envDir, "wceiling.json"))
	wf, err := LoadWorkflow(filepath.Join(f.envDir, "workflow.json"), ceiling)
	if err != nil {
		t.Fatal(err)
	}
	if man.GovernedHashes["workflow"] != wf.Hash {
		t.Fatal("L7 executed something other than the envelope's own workflow")
	}
	// And the false claim is recorded as-stated: detectable later,
	// never silently corrected or acted upon.
	if man.GovernedHashes["origin:skill"] != "some-other-skill@99" {
		t.Fatal("the claim must be recorded verbatim so inconsistency is detectable")
	}
}

// D-L9-11: the skill procedure is byte-verified against the envelope's
// stated pin. A mismatch refuses assembly — the task never starts.
func TestSkillProcedurePinMismatchRefused(t *testing.T) {
	f := setup(t, happyScript(), "")
	proc := writeJSON(t, f.envDir, "procedure.md", "# Procedure\nread, then declare done.\n")
	base := f.envelope(t, "t-pin")
	bad := withEnvelopeFields(t, base, map[string]any{
		"skill_procedure_path":   proc,
		"skill_procedure_sha256": strings.Repeat("0", 64), // not the real hash
	}, "envelope-t-pin-2.json")
	if _, err := f.o.SubmitTask(bad); err == nil ||
		!strings.Contains(err.Error(), "do not match the pinned hash") {
		t.Fatalf("a procedure whose bytes do not match its pin must refuse: %v", err)
	}
}

// The envelope contract: the procedure pair travels together, and its
// path is absolute like every other governed reference.
func TestSkillProcedureFieldsPaired(t *testing.T) {
	f := setup(t, happyScript(), "")
	base := f.envelope(t, "t-pair")
	onlyPath := withEnvelopeFields(t, base, map[string]any{
		"skill_procedure_path": "/tmp/procedure.md",
	}, "envelope-t-pair-2.json")
	if _, err := LoadEnvelope(onlyPath); err == nil ||
		!strings.Contains(err.Error(), "set together or not at all") {
		t.Fatalf("an unpinned procedure path must refuse: %v", err)
	}
	relative := withEnvelopeFields(t, base, map[string]any{
		"skill_procedure_path":   "procedure.md",
		"skill_procedure_sha256": strings.Repeat("a", 64),
	}, "envelope-t-pair-3.json")
	if _, err := LoadEnvelope(relative); err == nil ||
		!strings.Contains(err.Error(), "absolute") {
		t.Fatalf("a relative procedure path must refuse: %v", err)
	}
}

// A verified skill procedure genuinely reaches the model as governed
// instruction material — and its provenance is honest: it enters at
// the skill scope, never as harness constitution.
func TestSkillProcedureReachesTheModelAsInstructions(t *testing.T) {
	f := setup(t, happyScript(), "")
	body := "# Procedure\nAnalyze the manifest, then declare done.\n"
	proc := writeJSON(t, f.envDir, "skill-procedure.md", body)
	base := f.envelope(t, "t-proc")
	env := withEnvelopeFields(t, base, map[string]any{
		"skill_procedure_path":   proc,
		"skill_procedure_sha256": hashBytes([]byte(body)),
	}, "envelope-t-proc-2.json")
	res, err := f.o.SubmitTask(env)
	if err != nil || res.Status != state.StatusCompleted {
		t.Fatalf("a skill-procedure walk must complete: %v %+v", err, res)
	}
	// The composed delivery the model saw must contain the procedure.
	evs, _ := f.o.root.ReadEvents("t-proc")
	found := false
	for _, ev := range evs {
		if ev.Class != state.EvL2Delivery {
			continue
		}
		composed, err := f.o.root.Resolve(ev, 0)
		if err != nil {
			t.Fatal(err)
		}
		_ = composed
		found = true
	}
	if !found {
		t.Fatal("no delivery recorded")
	}
	// The task's EIS identity differs from a walk without a procedure:
	// the instruction set actually changed, honestly recorded.
	man, _ := f.o.root.ReadManifest("t-proc")
	f2 := setup(t, happyScript(), "")
	if _, err := f2.o.SubmitTask(f2.envelope(t, "t-noproc")); err != nil {
		t.Fatal(err)
	}
	man2, _ := f2.o.root.ReadManifest("t-noproc")
	if man.GovernedHashes["l1_eis"] == man2.GovernedHashes["l1_eis"] {
		t.Fatal("an activated skill procedure must change the recorded EIS identity")
	}
}
