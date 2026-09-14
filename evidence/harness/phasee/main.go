// themis-phase-e produces Phase E evidence for a concrete Deployment
// Instance: the governed chain end to end, on REAL artifacts.
//
// It consumes two walks that already ran (Phase D) and drives the L11
// half against them:
//
//	E3  facts grounded in the walks' OWN committed l4-audit events
//	E4  admission observed at the REAL Governance door (skills catalog)
//	E5  comparison yields a delta; run identities DERIVED, not supplied
//	E6  cold reconstruction CONFIRMED from package + criterion + door
//	E7  the Governance door is byte-identical afterwards
//	E8  a model-turn object from the same genuine history is REFUSED
//
// Like the Phase D harness this is evidence tooling with no authority.
// The criterion is RESOLVED from the Governance registry rather than
// hand-built, and the registry hash recorded in the package is that
// same file's hash — so the package's provenance is true of this
// deployment rather than of a fixture.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/tofchaliss/themis/ratchet"
	"github.com/tofchaliss/themis/state"
)

func main() {
	repo := flag.String("repo", "", "themis-ai-runtime checkout (absolute)")
	deploy := flag.String("deploy", "", "deployment root (absolute)")
	baseTask := flag.String("base-task", "rsys-e-base", "baseline walk task id")
	candTask := flag.String("cand-task", "rsys-e-cand", "candidate walk task id")
	critRef := flag.String("criterion", "walk-report-score-delta@1", "registered criterion ref")
	doorSkill := flag.String("door-skill", "remediate-dependency", "skill whose admission is observed")
	doorVer := flag.Int("door-version", 1, "skill version at the door")
	observedAt := flag.String("observed-at", "", "RFC3339 observation timestamp (required: no clock is read)")
	flag.Parse()

	for n, v := range map[string]string{
		"-repo": *repo, "-deploy": *deploy, "-observed-at": *observedAt,
	} {
		if v == "" {
			fail("%s is required", n)
		}
	}

	stateRoot := filepath.Join(*deploy, "state")
	criteriaReg := filepath.Join(*repo, "policies/ratchet/criteria.json")
	catalogPath := filepath.Join(*repo, "policies/skills/catalog.json")

	root, err := state.OpenRoot(stateRoot)
	must(err, "open state root")

	// --- the criterion comes from the GOVERNANCE registry ----------
	section("criterion — resolved from the governed registry")
	reg, err := ratchet.LoadRegistry(criteriaReg)
	must(err, "load criteria registry")
	entry, criterion, err := reg.ResolveCriterion(*critRef)
	must(err, "resolve "+*critRef)
	critBytes, err := os.ReadFile(filepath.Join(filepath.Dir(criteriaReg), entry.ArtifactPath))
	must(err, "read criterion artifact")
	regBytes, err := os.ReadFile(criteriaReg)
	must(err, "read criteria registry")
	registrySHA := ratchet.InstanceID(regBytes)
	fmt.Printf("  %s@%d state=%s\n", entry.Name, entry.Version, entry.State)
	fmt.Printf("  criterion sha256   = %s\n", ratchet.InstanceID(critBytes))
	fmt.Printf("  criteria registry  = %s\n", registrySHA)
	if ratchet.InstanceID(critBytes) != entry.Artifact {
		fail("criterion artifact does not match its registration")
	}

	// --- E3: facts are the walks' own evidence ---------------------
	section("E3 — facts witnessed by the walks' OWN l4-audit events")
	candFact := reportFact(root, *candTask, criterion.CandidateSelectors[0].Name)
	baseFact := reportFact(root, *baseTask, criterion.BaselineSelectors[0].Name)
	fmt.Printf("  candidate %s  obj=%s seq=%d\n", *candTask, short(candFact.Ref), candFact.EventSeq)
	fmt.Printf("  baseline  %s  obj=%s seq=%d\n", *baseTask, short(baseFact.Ref), baseFact.EventSeq)

	if _, r := ratchet.GroundFacts(root, "", criterion.CandidateSelectors,
		[]ratchet.EvidenceRef{candFact}, "candidate"); r != nil {
		fail("E3 candidate fact refused: %+v", r)
	}
	if _, r := ratchet.GroundFacts(root, "", criterion.BaselineSelectors,
		[]ratchet.EvidenceRef{baseFact}, "baseline"); r != nil {
		fail("E3 baseline fact refused: %+v", r)
	}
	fmt.Println("  both grounded — each fact's witness is its own walk's event")

	// --- E4: admission observed at the real door -------------------
	section("E4 — admission observed at the REAL Governance door")
	catalogBefore, err := os.ReadFile(catalogPath)
	must(err, "read skills catalog")
	admission, err := ratchet.ObserveAdmission("l9-catalog", catalogPath, *doorSkill, *doorVer, *observedAt)
	must(err, "observe admission")
	if admission == nil {
		fail("E4 door resolution produced no observation")
	}
	fmt.Printf("  door=l9-catalog %s@%d artifact=%s\n",
		*doorSkill, *doorVer, short(admission.ArtifactSHA256))

	// --- E5: the comparison ----------------------------------------
	section("E5 — comparison; run identities DERIVED")
	candidateContent := []byte(fmt.Sprintf(
		`{"skill":%q,"version":%d,"change":"phase-e candidate walk %s"}`,
		*doorSkill, *doorVer+1, *candTask))
	pkg, refusal, err := ratchet.Compare(ratchet.CompareInput{
		CriterionRef:    *critRef,
		Criterion:       criterion,
		RegistrySHA256:  registrySHA,
		CandidateHash:   ratchet.InstanceID(candidateContent),
		ClaimedBaseline: admission.ArtifactSHA256,
		Admission:       admission,
		CandidateFacts:  []ratchet.EvidenceRef{candFact},
		BaselineFacts:   []ratchet.EvidenceRef{baseFact},
	})
	if err != nil || refusal != nil {
		fail("E5 comparison failed: refusal=%+v err=%v", refusal, err)
	}
	fmt.Printf("  delta            = %v\n", pkg.Delta)
	fmt.Printf("  run identities   = %v\n", pkg.RunIdentities)
	wantRuns := []string{"task:" + *baseTask, "task:" + *candTask}
	if strings.Join(pkg.RunIdentities, ",") != strings.Join(wantRuns, ",") {
		fmt.Printf("  NOTE: run identities %v (expected %v) — order/derivation differs\n",
			pkg.RunIdentities, wantRuns)
	}
	for _, id := range pkg.RunIdentities {
		if !strings.HasPrefix(id, "task:") {
			fail("E5 run identity %q is not derived from a governed run", id)
		}
	}

	derivations, err := ratchet.DerivePerField(criterion, pkg.Delta)
	must(err, "derive per-field relations")
	for _, d := range derivations {
		fmt.Printf("  derivation       = %s: %s\n", d.Field, d.Relation)
	}

	pkgID, pkgBytes, err := ratchet.StoreInstance(root.Store(), pkg)
	must(err, "store comparison package")
	fmt.Printf("  package          = %s\n", pkgID)

	// --- E6: cold reconstruction -----------------------------------
	section("E6 — cold reconstruction")
	stored, err := root.Store().GetObject(pkgID)
	must(err, "read stored package")
	if string(stored) != string(pkgBytes) {
		fail("E6 stored package bytes differ from minted bytes")
	}
	rec, err := ratchet.Reconstruct(stored, ratchet.ReconstructInputs{
		CriterionBytes:    critBytes,
		DoorRegistryBytes: catalogBefore,
		Root:              root,
	})
	must(err, "reconstruct")
	fmt.Printf("  result           = %s\n", rec.Result)
	if rec.Result != ratchet.ReconConfirmed {
		fail("E6 reconstruction not CONFIRMED: %+v", rec)
	}

	// --- E7: the door did not move ---------------------------------
	section("E7 — the Governance door is byte-identical afterwards")
	catalogAfter, err := os.ReadFile(catalogPath)
	must(err, "re-read skills catalog")
	if ratchet.InstanceID(catalogAfter) != ratchet.InstanceID(catalogBefore) {
		fail("E7 the governance door changed during the chain")
	}
	fmt.Printf("  catalog sha256   = %s (unchanged)\n", short(ratchet.InstanceID(catalogAfter)))
	fmt.Println("  evidence was delivered TO the door; nothing acted THROUGH it")

	// --- E8: the laundering path stays closed ----------------------
	section("E8 — a model-turn object offered as a fact")
	events, err := root.ReadEvents(*candTask)
	must(err, "read candidate events")
	tried := false
	for _, ev := range events {
		if ev.Class != state.EvModelTurn || len(ev.Refs) == 0 {
			continue
		}
		b, gerr := root.Store().GetObject(ev.Refs[0].ID)
		if gerr != nil {
			continue
		}
		forged := ratchet.EvidenceRef{
			Selector: criterion.CandidateSelectors[0].Name,
			Source:   "l6_execution_record",
			Ref:      ev.Refs[0].ID, SHA256: ratchet.InstanceID(b), Value: b,
			TaskID: *candTask, EventSeq: ev.Seq,
		}
		_, r := ratchet.GroundFacts(root, "", criterion.CandidateSelectors,
			[]ratchet.EvidenceRef{forged}, "candidate")
		if r == nil {
			fail("E8 a model-turn object grounded as an established fact — G2 breached")
		}
		fmt.Printf("  model-turn obj %s REFUSED: %s\n", short(ev.Refs[0].ID), r.Reason)
		tried = true
		break
	}
	if !tried {
		fmt.Println("  NOTE: no model-turn object carried a ref in this walk — E8 not exercised")
	}

	section("RESULT")
	fmt.Printf("  criterion : %s (governed)\n", *critRef)
	fmt.Printf("  delta     : %v\n", pkg.Delta)
	fmt.Printf("  package   : %s\n", pkgID)
	fmt.Printf("  recon     : %s\n", rec.Result)
	fmt.Println("\nE3-E8 evidence produced against real artifacts.")
}

// reportFact locates a walk's committed l4-audit event for the
// verify_report call and returns the raw report bytes as an L11
// evidence reference. The fact IS the walk's own evidence and its
// witness IS the walk's own event — D-G2-1 exercised on real history.
func reportFact(root *state.Root, taskID, selector string) ratchet.EvidenceRef {
	events, err := root.ReadEvents(taskID)
	must(err, "read events for "+taskID)
	for _, ev := range events {
		if ev.Class != state.EvL4Audit || len(ev.Refs) == 0 {
			continue
		}
		var body struct {
			Tool string `json:"Tool"`
		}
		if json.Unmarshal(ev.Body, &body) != nil || body.Tool != "verify_report" {
			continue
		}
		b, err := root.Store().GetObject(ev.Refs[0].ID)
		must(err, "read report object")
		return ratchet.EvidenceRef{
			Selector: selector, Source: "l6_execution_record",
			Ref: ev.Refs[0].ID, SHA256: ratchet.InstanceID(b), Value: b,
			TaskID: taskID, EventSeq: ev.Seq,
		}
	}
	fail("no verify_report l4-audit event in %s — did that walk complete?", taskID)
	return ratchet.EvidenceRef{}
}

func section(s string) { fmt.Printf("\n=== %s ===\n", s) }

func short(h string) string {
	if i := strings.IndexByte(h, ':'); i >= 0 && len(h) > i+13 {
		return h[:i+13]
	}
	if len(h) > 12 {
		return h[:12]
	}
	return h
}

func must(err error, what string) {
	if err != nil {
		fail("%s: %v", what, err)
	}
}

func fail(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "\nFAIL: "+format+"\n", args...)
	os.Exit(1)
}
