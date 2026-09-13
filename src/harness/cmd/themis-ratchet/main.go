// themis-ratchet is the L11 invocation surface (D-L11-14): every
// operation is a synchronous response to this explicit invocation —
// nothing watches, schedules, retries, or continues. Exact pins are
// required everywhere; nothing is defaulted or resolved to "latest".
//
// Acceptance boundary (close-review remediation): flags are
// SYNTAX-validated first — a malformed request is a usage error
// (exit 1, nothing minted; the request was never accepted). Once
// syntax passes and the store is open, the invocation is ACCEPTED:
// from that point every outcome is durably recorded — comparison
// package, or refusal fact (including unregistered/withdrawn
// criterion refusals) — whatever it shows. Exit 0 means "the
// invocation completed and its result was recorded"; a refusal is a
// completed invocation. Machinery failure (store unwritable,
// registry unreadable) exits 1 and mints nothing.
//
// The no-discard property this surface can enforce is the
// machinery's: no result is emitted before it is durably stored,
// and no code path drops an unfavorable result that a favorable one
// would keep. Operator-level suppression (throwaway state roots,
// re-running elsewhere) is request-level cherry-picking — visible,
// not prevented (D-L11-4 §6), and defended at doors via
// complete-under-S requirements.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/tofchaliss/themis/confine"
	"github.com/tofchaliss/themis/ratchet"
	"github.com/tofchaliss/themis/state"
)

var (
	shaHex   = regexp.MustCompile(`^[0-9a-f]{64}$`)
	runIdent = regexp.MustCompile(`^[a-zA-Z0-9:_\-./]{1,256}$`)
	objectID = regexp.MustCompile(`^sha256:[0-9a-f]{64}$`)
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

// readBounded reads a CLI input file with the same size discipline
// the package applies to governed artifacts (close-review M-2).
func readBounded(path string) ([]byte, error) {
	if path == "" {
		return nil, fmt.Errorf("required file flag missing")
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	if len(b) > 4<<20 {
		return nil, fmt.Errorf("%s exceeds 4MiB", path)
	}
	return b, nil
}

func cmdCompare(args []string) error {
	fs := flag.NewFlagSet("compare", flag.ExitOnError)
	regPath := fs.String("criteria-registry", "", "path to the governed criteria registry")
	regPin := fs.String("registry-sha256", "", "expected criteria-registry hash (admission-at-consumption pin; optional)")
	criterionRef := fs.String("criterion", "", "exact criterion name@version")
	candidate := fs.String("candidate", "", "candidate content sha256")
	claimed := fs.String("claimed-baseline", "", "candidate-claimed baseline sha256 (optional)")
	door := fs.String("door", "", "owning door of the baseline (closed table: l9-catalog, l10-contract-registry, l11-criteria, l11-regression-sets)")
	doorRegistry := fs.String("door-registry", "", "path to the door's registry file — L11 resolves the observation from its bytes")
	baseline := fs.String("baseline", "", "baseline identity at the door, exact name@version")
	observedAt := fs.String("observed-at", "", "observation instant recorded in the package")
	candFactsPath := fs.String("candidate-facts", "", "path to candidate evidence refs JSON")
	baseFactsPath := fs.String("baseline-facts", "", "path to baseline evidence refs JSON")
	evidenceRoot := fs.String("evidence-root", "", "root for external-plane evidence refs (benchmark/gate files), confined")
	runs := fs.String("runs", "", "comma-separated run identities")
	planRef := fs.String("plan", "", "evaluation-plan object id (optional provenance)")
	stateRoot := fs.String("state-root", "", "L6 state root")
	if err := fs.Parse(args); err != nil {
		return err
	}

	// --- Syntax stage: a malformed request is never accepted.
	if !shaHex.MatchString(*candidate) {
		return fmt.Errorf("--candidate must be a sha256 hex digest")
	}
	if *claimed != "" && !shaHex.MatchString(*claimed) {
		return fmt.Errorf("--claimed-baseline must be a sha256 hex digest")
	}
	if _, _, err := ratchet.ParseRef(*criterionRef); err != nil {
		return err
	}
	baseName, baseVer, err := ratchet.ParseRef(*baseline)
	if err != nil {
		return fmt.Errorf("--baseline: %v", err)
	}
	if *door == "" || *doorRegistry == "" || *observedAt == "" {
		return fmt.Errorf("--door, --door-registry, and --observed-at are required — L11 resolves the admission observation itself")
	}
	if *regPin != "" && !shaHex.MatchString(*regPin) {
		return fmt.Errorf("--registry-sha256 must be a sha256 hex digest")
	}
	if *planRef != "" && !objectID.MatchString(*planRef) {
		return fmt.Errorf("--plan must be an object id (sha256:<hex>)")
	}
	var runIDs []string
	for _, r := range strings.Split(*runs, ",") {
		if r = strings.TrimSpace(r); r != "" {
			if !runIdent.MatchString(r) {
				return fmt.Errorf("run identity %q is malformed", r)
			}
			runIDs = append(runIDs, r)
		}
	}
	if len(runIDs) == 0 {
		return fmt.Errorf("--runs must name at least one run identity")
	}
	candRaw, err := readBounded(*candFactsPath)
	if err != nil {
		return fmt.Errorf("candidate facts: %v", err)
	}
	candFacts, err := ratchet.ParseEvidenceRefs(candRaw, *candFactsPath)
	if err != nil {
		return err
	}
	baseRaw, err := readBounded(*baseFactsPath)
	if err != nil {
		return fmt.Errorf("baseline facts: %v", err)
	}
	baseFacts, err := ratchet.ParseEvidenceRefs(baseRaw, *baseFactsPath)
	if err != nil {
		return err
	}

	// --- Acceptance: store open. From here, outcomes are durable.
	store, err := openStore(*stateRoot)
	if err != nil {
		return err
	}
	recordRefusal := func(r *ratchet.RefusalFact) error {
		r.CriterionRef = *criterionRef
		r.Candidate = *candidate
		r.Claimed = *claimed
		id, _, serr := ratchet.StoreInstance(store, r)
		if serr != nil {
			return serr
		}
		emit(map[string]any{"kind": "refusal", "object": id, "reason": r.Reason, "detail": r.Detail})
		return nil
	}

	reg, err := ratchet.LoadRegistry(*regPath)
	if err != nil {
		return err // machinery: the governed registry itself is unreadable
	}
	if *regPin != "" && reg.Hash != *regPin {
		// Admission-at-consumption (the verdict/digest pattern): the
		// registry in force is not the pinned state — refuse durably.
		return recordRefusal(&ratchet.RefusalFact{Artifact: "l11-refusal", Schema: 1,
			Reason: ratchet.ReasonIntegrityFailure,
			Detail: "criteria registry bytes do not match the supplied pin"})
	}
	_, criterion, err := reg.ResolveCriterion(*criterionRef)
	if err != nil {
		// Accepted invocation naming an unresolvable criterion → a
		// durable refusal, never silence (close-review M-1).
		reason := ratchet.ReasonUnregisteredArtifact
		if strings.Contains(err.Error(), "withdrawn") {
			reason = ratchet.ReasonWithdrawnArtifact
		}
		return recordRefusal(&ratchet.RefusalFact{Artifact: "l11-refusal", Schema: 1,
			Reason: reason, Detail: err.Error()})
	}

	// --- L11 resolves the door observation itself (D-L11-5 §2).
	admission, err := ratchet.ObserveAdmission(*door, *doorRegistry, baseName, baseVer, *observedAt)
	if err != nil {
		return err // machinery: door registry unreadable/unparseable
	}
	// admission == nil → Compare refuses with unadmitted-baseline.

	// --- Ground the supplied facts against their claimed planes.
	external := func(ref string) ([]byte, error) {
		if *evidenceRoot == "" {
			return nil, fmt.Errorf("no --evidence-root supplied for external-plane refs")
		}
		p, err := confine.ResolvePath(*evidenceRoot, ref)
		if err != nil {
			return nil, err
		}
		return readBounded(p)
	}
	if r := ratchet.GroundFacts(store, external, criterion.CandidateSelectors, candFacts, "candidate"); r != nil {
		return recordRefusal(r)
	}
	if r := ratchet.GroundFacts(store, external, criterion.BaselineSelectors, baseFacts, "baseline"); r != nil {
		return recordRefusal(r)
	}

	pkg, refusal, err := ratchet.Compare(ratchet.CompareInput{
		CriterionRef:    *criterionRef,
		Criterion:       criterion,
		RegistrySHA256:  reg.Hash,
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
	if refusal != nil {
		return recordRefusal(refusal)
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
	setsRegPath := fs.String("sets-registry", "", "path to the governed regression-sets registry")
	critRegPath := fs.String("criteria-registry", "", "path to the governed criteria registry (member hash binding)")
	setRef := fs.String("set", "", "exact set name@version — supplied by the consuming door, never resolved here")
	stateRoot := fs.String("state-root", "", "L6 state root")
	var members memberFlags
	fs.Var(&members, "member", "member@ver=sha256:objectid constituent binding (repeatable)")
	if err := fs.Parse(args); err != nil {
		return err
	}
	store, err := openStore(*stateRoot)
	if err != nil {
		return err
	}
	setsReg, err := ratchet.LoadRegistry(*setsRegPath)
	if err != nil {
		return err
	}
	_, set, err := setsReg.ResolveSet(*setRef)
	if err != nil {
		return err
	}
	critReg, err := ratchet.LoadRegistry(*critRegPath)
	if err != nil {
		return err
	}
	inSet := map[string]bool{}
	for _, m := range set.Members {
		inSet[m] = true
	}
	constituents := map[string]*ratchet.ComparisonPackage{}
	criteria := map[string]*ratchet.Criterion{}
	for member, objID := range members.m {
		if !inSet[member] {
			return fmt.Errorf("--member %s is not a member of %s — bindings outside the set are refused, not ignored", member, *setRef)
		}
		pkg, err := ratchet.LoadComparison(store, objID)
		if err != nil {
			return fmt.Errorf("member %s: %v", member, err)
		}
		constituents[member] = pkg
		_, mc, err := critReg.ResolveCriterion(member)
		if err != nil {
			return fmt.Errorf("member %s: %v", member, err)
		}
		criteria[member] = mc
	}
	sp, err := ratchet.BuildRegressionPackage(*setRef, set, constituents, criteria)
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
		return fmt.Errorf("member binding must be member@ver=sha256:objectid")
	}
	if _, _, err := ratchet.ParseRef(member); err != nil {
		return err
	}
	if !objectID.MatchString(obj) {
		return fmt.Errorf("constituent %q is not an object id (sha256:<hex>)", obj)
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
	doorRegistry := fs.String("door-registry", "", "path to the observed door registry bytes (optional; absent = missing input)")
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
	var criterionBytes, doorBytes []byte
	if *criterionFile != "" {
		if criterionBytes, err = readBounded(*criterionFile); err != nil {
			return err
		}
	}
	if *doorRegistry != "" {
		if doorBytes, err = readBounded(*doorRegistry); err != nil {
			return err
		}
	}
	rec, err := ratchet.Reconstruct(pkgBytes, criterionBytes, doorBytes)
	if err != nil {
		return err
	}
	// Discrepancy AND missing-inputs facts are durable (D-L11-17 §4,
	// D-L11-20 §2 inventory); confirmation changes nothing anyone
	// must act on and is reported without minting. Never repairs.
	if rec.Result != ratchet.ReconConfirmed {
		id, _, serr := ratchet.StoreInstance(store, rec)
		if serr != nil {
			return serr
		}
		emit(map[string]any{"kind": "reconstruction", "object": id, "result": rec.Result,
			"missing_inputs": rec.MissingInputs, "discrepancies": rec.Discrepancies})
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
	// (D-L11-4 claim 2). Every refusal is explicit — absence is
	// never ambiguous.
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
	} else {
		out["non_regression_refusal"] = err.Error()
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
	raw, err := readBounded(*file)
	if err != nil {
		return fmt.Errorf("%s: %v", what, err)
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
