// themis-ratchet is the L11 invocation surface (D-L11-14): every
// operation is a synchronous response to this explicit invocation —
// nothing watches, schedules, retries, or continues. Exact pins are
// required everywhere; nothing is defaulted or resolved to "latest".
// Once an invocation is accepted, its outcome is durably recorded —
// package or refusal — whatever it shows (no-discard). Exit code 0
// means "the invocation completed and its result was recorded"; a
// refusal fact is a completed invocation, not a process failure.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/tofchaliss/themis/ratchet"
	"github.com/tofchaliss/themis/state"
)

func main() {
	if len(os.Args) < 2 {
		fail("usage: themis-ratchet <compare|set|reconstruct|derive|candidate|plan> ...")
	}
	var err error
	switch os.Args[1] {
	case "compare":
		err = cmdCompare(os.Args[2:])
	case "set":
		err = cmdSet(os.Args[2:])
	case "reconstruct":
		err = cmdReconstruct(os.Args[2:])
	case "derive":
		err = cmdDerive(os.Args[2:])
	case "candidate":
		err = cmdCandidate(os.Args[2:])
	case "plan":
		err = cmdPlan(os.Args[2:])
	default:
		fail("unknown subcommand %q", os.Args[1])
	}
	if err != nil {
		fail("%v", err)
	}
}

func fail(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "themis-ratchet: "+format+"\n", args...)
	os.Exit(1)
}

func emit(v any) {
	b, _ := json.MarshalIndent(v, "", "  ")
	fmt.Println(string(b))
}

func openStore(root string) (*state.ObjectStore, error) {
	if root == "" {
		return nil, fmt.Errorf("--state-root is required")
	}
	r, err := state.OpenRoot(root)
	if err != nil {
		return nil, err
	}
	return r.Store(), nil
}

func readJSONFile(path string, v any) error {
	if path == "" {
		return fmt.Errorf("required file flag missing")
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return json.Unmarshal(b, v)
}

func cmdCompare(args []string) error {
	fs := flag.NewFlagSet("compare", flag.ExitOnError)
	regPath := fs.String("criteria-registry", "", "path to the governed criteria registry")
	criterionRef := fs.String("criterion", "", "exact criterion name@version")
	candidate := fs.String("candidate", "", "candidate content sha256")
	claimed := fs.String("claimed-baseline", "", "candidate-claimed baseline sha256 (optional)")
	admissionPath := fs.String("admission", "", "path to the admission observation JSON")
	candFactsPath := fs.String("candidate-facts", "", "path to candidate evidence refs JSON")
	baseFactsPath := fs.String("baseline-facts", "", "path to baseline evidence refs JSON")
	runs := fs.String("runs", "", "comma-separated run identities")
	planRef := fs.String("plan", "", "evaluation-plan object id (optional provenance)")
	stateRoot := fs.String("state-root", "", "L6 state root")
	if err := fs.Parse(args); err != nil {
		return err
	}
	store, err := openStore(*stateRoot)
	if err != nil {
		return err
	}
	reg, err := ratchet.LoadRegistry(*regPath)
	if err != nil {
		return err
	}
	_, criterion, err := reg.ResolveCriterion(*criterionRef)
	if err != nil {
		return err
	}
	var admission *ratchet.AdmissionObservation
	if *admissionPath != "" {
		admission = &ratchet.AdmissionObservation{}
		if err := readJSONFile(*admissionPath, admission); err != nil {
			return fmt.Errorf("admission observation: %v", err)
		}
	}
	var candFacts, baseFacts []ratchet.EvidenceRef
	if *candFactsPath != "" {
		if err := readJSONFile(*candFactsPath, &candFacts); err != nil {
			return fmt.Errorf("candidate facts: %v", err)
		}
	}
	if *baseFactsPath != "" {
		if err := readJSONFile(*baseFactsPath, &baseFacts); err != nil {
			return fmt.Errorf("baseline facts: %v", err)
		}
	}
	var runIDs []string
	for _, r := range strings.Split(*runs, ",") {
		if r = strings.TrimSpace(r); r != "" {
			runIDs = append(runIDs, r)
		}
	}
	pkg, refusal, err := ratchet.Compare(ratchet.CompareInput{
		CriterionRef:    *criterionRef,
		Criterion:       criterion,
		CandidateHash:   *candidate,
		ClaimedBaseline: *claimed,
		Admission:       admission,
		CandidateFacts:  candFacts,
		BaselineFacts:   baseFacts,
		RunIdentities:   runIDs,
		PlanRef:         *planRef,
	})
	if err != nil {
		return err // machinery failure mints nothing
	}
	// Accepted invocation → durable result, whatever it shows.
	if refusal != nil {
		id, _, serr := ratchet.StoreInstance(store, refusal)
		if serr != nil {
			return serr
		}
		emit(map[string]any{"kind": "refusal", "object": id, "reason": refusal.Reason, "detail": refusal.Detail})
		return nil
	}
	id, _, serr := ratchet.StoreInstance(store, pkg)
	if serr != nil {
		return serr
	}
	emit(map[string]any{"kind": "comparison", "object": id})
	return nil
}

func cmdSet(args []string) error {
	fs := flag.NewFlagSet("set", flag.ExitOnError)
	regPath := fs.String("sets-registry", "", "path to the governed regression-sets registry")
	setRef := fs.String("set", "", "exact set name@version — supplied by the consuming door, never resolved here")
	stateRoot := fs.String("state-root", "", "L6 state root")
	var members memberFlags
	fs.Var(&members, "member", "member=objectid constituent binding (repeatable)")
	if err := fs.Parse(args); err != nil {
		return err
	}
	store, err := openStore(*stateRoot)
	if err != nil {
		return err
	}
	reg, err := ratchet.LoadRegistry(*regPath)
	if err != nil {
		return err
	}
	_, set, err := reg.ResolveSet(*setRef)
	if err != nil {
		return err
	}
	constituents := map[string]*ratchet.ComparisonPackage{}
	for member, objID := range members.m {
		pkg, err := ratchet.LoadComparison(store, objID)
		if err != nil {
			return fmt.Errorf("member %s: %v", member, err)
		}
		constituents[member] = pkg
	}
	sp, err := ratchet.BuildRegressionPackage(*setRef, set, constituents)
	if err != nil {
		return err // incomplete = no set-level package; constituents stand alone
	}
	id, _, err := ratchet.StoreInstance(store, sp)
	if err != nil {
		return err
	}
	emit(map[string]any{"kind": "regression", "object": id})
	return nil
}

type memberFlags struct{ m map[string]string }

func (f *memberFlags) String() string { return "" }
func (f *memberFlags) Set(v string) error {
	member, obj, ok := strings.Cut(v, "=")
	if !ok || member == "" || obj == "" {
		return fmt.Errorf("member binding must be member@ver=objectid")
	}
	if f.m == nil {
		f.m = map[string]string{}
	}
	if _, dup := f.m[member]; dup {
		return fmt.Errorf("duplicate member %s", member)
	}
	f.m[member] = obj
	return nil
}

func cmdReconstruct(args []string) error {
	fs := flag.NewFlagSet("reconstruct", flag.ExitOnError)
	stateRoot := fs.String("state-root", "", "L6 state root")
	pkgID := fs.String("package", "", "comparison package object id")
	criterionFile := fs.String("criterion-file", "", "path to the conditioning criterion bytes (optional; absent = missing input)")
	if err := fs.Parse(args); err != nil {
		return err
	}
	store, err := openStore(*stateRoot)
	if err != nil {
		return err
	}
	pkgBytes, err := store.GetObject(*pkgID)
	if err != nil {
		return err
	}
	var criterionBytes []byte
	if *criterionFile != "" {
		criterionBytes, err = os.ReadFile(*criterionFile)
		if err != nil {
			return err
		}
	}
	rec, err := ratchet.Reconstruct(pkgBytes, criterionBytes)
	if err != nil {
		return err
	}
	// A discrepancy is durable evidence for Governance; confirmation
	// and missing-inputs are reported without minting (they change
	// nothing anyone must act on). Never repairs, never rewrites.
	if rec.Result == ratchet.ReconDiscrepancy {
		id, _, serr := ratchet.StoreInstance(store, rec)
		if serr != nil {
			return serr
		}
		emit(map[string]any{"kind": "reconstruction", "object": id, "result": rec.Result, "discrepancies": rec.Discrepancies})
		return nil
	}
	emit(rec)
	return nil
}

func cmdDerive(args []string) error {
	fs := flag.NewFlagSet("derive", flag.ExitOnError)
	stateRoot := fs.String("state-root", "", "L6 state root")
	pkgID := fs.String("package", "", "comparison package object id")
	regPath := fs.String("criteria-registry", "", "path to the governed criteria registry")
	if err := fs.Parse(args); err != nil {
		return err
	}
	store, err := openStore(*stateRoot)
	if err != nil {
		return err
	}
	pkg, err := ratchet.LoadComparison(store, *pkgID)
	if err != nil {
		return err
	}
	reg, err := ratchet.LoadRegistry(*regPath)
	if err != nil {
		return err
	}
	_, criterion, err := reg.ResolveCriterion(pkg.CriterionRef)
	if err != nil {
		return err
	}
	if criterion.SHA256 != pkg.CriterionSHA256 {
		return fmt.Errorf("registered criterion bytes do not match the package's conditioning tuple")
	}
	// Stateless derivations: computed, printed, never stored
	// (D-L11-4 claim 2). Absent ordering → the refusal explains why.
	out := map[string]any{"package": *pkgID, "criterion": pkg.CriterionRef, "delta": pkg.Delta}
	if perField, err := ratchet.DerivePerField(criterion, pkg.Delta); err == nil {
		out["per_field"] = perField
	} else {
		out["per_field_refusal"] = err.Error()
	}
	if rel, err := ratchet.DeriveRelation(criterion, pkg.Delta); err == nil {
		out["relation_under_k"] = rel
	} else {
		out["relation_refusal"] = err.Error()
	}
	if within, err := ratchet.DeriveNonRegression(criterion, pkg.Delta); err == nil {
		out["within_non_regression_region"] = within
	}
	emit(out)
	return nil
}

func cmdCandidate(args []string) error {
	return storeArtifact(args, "candidate file", func(raw []byte, origin string) (any, error) {
		return ratchet.ParseCandidate(raw, origin)
	})
}

func cmdPlan(args []string) error {
	return storeArtifact(args, "plan file", func(raw []byte, origin string) (any, error) {
		return ratchet.ParsePlan(raw, origin)
	})
}

// storeArtifact validates an authored artifact and stores its
// CANONICAL bytes. Authoring happened elsewhere, attributably — this
// surface only accepts, validates, and records (D-L11-14 §4).
func storeArtifact(args []string, what string, parse func([]byte, string) (any, error)) error {
	fs := flag.NewFlagSet("store", flag.ExitOnError)
	stateRoot := fs.String("state-root", "", "L6 state root")
	file := fs.String("file", "", "path to the authored artifact JSON")
	if err := fs.Parse(args); err != nil {
		return err
	}
	store, err := openStore(*stateRoot)
	if err != nil {
		return err
	}
	if *file == "" {
		return fmt.Errorf("--file is required (%s)", what)
	}
	raw, err := os.ReadFile(*file)
	if err != nil {
		return err
	}
	v, err := parse(raw, *file)
	if err != nil {
		return err
	}
	id, _, err := ratchet.StoreInstance(store, v)
	if err != nil {
		return err
	}
	emit(map[string]any{"kind": what, "object": id})
	return nil
}
