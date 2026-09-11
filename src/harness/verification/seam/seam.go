// Package seam wires the L10 verification mechanism into the L7
// execution-result boundary (D-L10-13): it implements
// orchestration.VerificationEvaluator by composing the contract
// registry, the registered canonicalization, and the pure L10
// evaluator. It is deliberately a separate package: orchestration
// keeps zero dependency on verification (contract-blind, deps-proven)
// and verification keeps zero dependency on orchestration; only this
// bridge imports both.
package seam

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/tofchaliss/themis/orchestration"
	"github.com/tofchaliss/themis/runtime/model"
	"github.com/tofchaliss/themis/tools"
	"github.com/tofchaliss/themis/verification"
)

// Canonicalizer is one registered canonicalization: a pure,
// environment-independent transform from raw captured bytes (+ pinned
// config) to the canonical verifier-domain result (D-L10-3/12 — its
// execution is provided by this already-governed deterministic
// mechanism, never by verifier runtime, network, filesystem, time,
// or host identity). Registered per capability in code: its identity
// is the harness build, reviewed like all harness code.
type Canonicalizer func(raw []byte, config json.RawMessage) (string, error)

// canonicalizers is the closed registration: capability name → its
// canonicalization. Adding an entry is part of registering a verifier
// capability (a reviewed change), never runtime configuration.
var canonicalizers = map[string]Canonicalizer{
	"verify_report": canonReport,
}

// canonReport canonicalizes the verify_report capture: the raw bytes
// must parse as a JSON object whose closed required fields are
// non-empty strings. Domain: report_valid | report_invalid.
// (A missing artifact never reaches canonicalization — the executor
// fails the call and the tool-error path governs; see the recorded
// v1 mapping note in the L10 tasks.)
func canonReport(raw []byte, config json.RawMessage) (string, error) {
	// Unknown fields in the REPORT are fine — the report is model
	// output; only the closed required core is checked.
	var report map[string]any
	if err := json.Unmarshal(raw, &report); err != nil {
		return "report_invalid", nil
	}
	for _, field := range []string{"finding", "remediation", "evidence"} {
		v, ok := report[field].(string)
		if !ok || strings.TrimSpace(v) == "" {
			return "report_invalid", nil
		}
	}
	return "report_valid", nil
}

// Evaluator implements orchestration.VerificationEvaluator over a
// governed contract registry path and the decoded L4 registry the
// deployment actually runs.
type Evaluator struct {
	RegistryPath string          // L10 contract registry (governed file)
	L4           *tools.Registry // the decoded L4 registry in force
}

// VerifierEligible implements verification.EligibilityChecker against
// the deployment's L4 registry: the capability must carry the
// verifier_eligible registration property AND the contract's pinned
// registry hash must be the registry actually in force — a contract
// bound to a different registry version is not eligible here.
func (e *Evaluator) VerifierEligible(capability, registrySHA string) (bool, error) {
	if e.L4 == nil {
		return false, fmt.Errorf("no L4 registry wired")
	}
	if registrySHA != e.L4.Hash {
		return false, nil
	}
	for i := range e.L4.Tools {
		if e.L4.Tools[i].Name == capability {
			return e.L4.Tools[i].VerifierEligible, nil
		}
	}
	return false, nil
}

// EvaluateCall runs the D-L10-6 pipeline stages 1 and 4 around the
// already-completed stage 2-3 (the L4-authorized, executor-captured
// call): atomic contract resolution, canonicalization through the
// registered mechanism, and the pure L10 evaluation. Refusals are
// typed and pre-instance (D-L10-8); an error return is evaluator
// machinery failure and mints nothing.
func (e *Evaluator) EvaluateCall(taskID string, call model.ToolCall, resultEvidence []byte, executionRef string) (*orchestration.VerificationOutcome, error) {
	refuse := func(reason string) *orchestration.VerificationOutcome {
		return &orchestration.VerificationOutcome{Refused: true, RefusalReason: reason}
	}

	var args map[string]any
	if err := json.Unmarshal(call.Arguments, &args); err != nil {
		return refuse("unparseable verification arguments"), nil
	}
	ref, _ := args["contract"].(string)
	if ref == "" {
		return refuse("no contract named — a verification proposal must reference an exact registered contract"), nil
	}

	// Stage 1 — atomic resolution from ONE registry load (D-L10-2 /
	// D-L9-10 TOCTOU rule). Every resolution failure is a typed
	// refusal: no evaluation instance exists yet.
	reg, err := verification.LoadRegistry(e.RegistryPath)
	if err != nil {
		return refuse("contract registry unreadable: " + err.Error()), nil
	}
	entry, contract, err := reg.Resolve(ref, e)
	if err != nil {
		return refuse(err.Error()), nil
	}
	_ = entry

	// The proposal's capability must BE the contract's bound verifier:
	// invoking capability X under a contract that binds Y is a
	// proposal defect, refused pre-instance.
	if contract.Verifier.Capability != call.Name {
		return refuse(fmt.Sprintf("contract %s binds capability %q, not %q", ref, contract.Verifier.Capability, call.Name)), nil
	}

	// The evaluation instance now exists. Canonicalization through the
	// registered mechanism; absence of a canonicalizer for an eligible
	// capability is machinery breakage, not a graded outcome.
	canon, ok := canonicalizers[call.Name]
	if !ok {
		return nil, fmt.Errorf("no registered canonicalization for eligible capability %q", call.Name)
	}

	facts := verification.ExecutionFacts{
		ExecutionRef:        executionRef,
		AppliedConfigSHA256: hashBytes(contract.Config),
	}
	var canonicalBytes []byte
	if len(resultEvidence) > 0 {
		canonical, cerr := canon(resultEvidence, contract.Config)
		if cerr != nil {
			// Canonicalization could not produce a canonical result:
			// UNAVAILABLE territory — leave canonical facts empty and
			// let the evaluator grade it (D-L10-8).
			canonical = ""
		}
		if canonical != "" {
			canonicalBytes = []byte(canonical)
			facts.RawObjectID = hashBytes(resultEvidence)
			facts.CanonicalObjectID = hashBytes(canonicalBytes)
			facts.CanonicalResult = canonical
		}
	}

	// Evidence: the artifact the verifier consumed IS the captured
	// raw bytes — workspace-confined (in-task) and read successfully
	// (resolvable). Its object identity is the content address the L6
	// store will independently derive from the same bytes (the
	// two-source property stays real).
	slot := ""
	if len(contract.Evidence) == 1 {
		slot = contract.Evidence[0].Name
	}
	refs := []verification.ProposedEvidence{{
		Slot:       slot,
		ObjectID:   hashBytes(resultEvidence),
		InTask:     true,
		Resolvable: len(resultEvidence) > 0,
	}}

	ev, err := verification.Evaluate(contract, refs, facts)
	if err != nil {
		return nil, err // evaluator invariant: mints nothing
	}
	record, err := json.Marshal(ev)
	if err != nil {
		return nil, err
	}

	// No-reopen window (R-L9-2): the durable contract bytes are the
	// exact bytes the resolution verified — carried on the Contract,
	// never re-read from the file after verification.
	return &orchestration.VerificationOutcome{
		ContractToken:  fmt.Sprintf("%s@%d", contract.Name, contract.Contract),
		Outcome:        string(ev.Outcome),
		Record:         record,
		ContractBytes:  contract.Raw,
		RawBytes:       resultEvidence,
		CanonicalBytes: canonicalBytes,
	}, nil
}

func hashBytes(b []byte) string {
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}
