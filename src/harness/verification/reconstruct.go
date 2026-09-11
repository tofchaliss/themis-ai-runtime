package verification

// Reconstruction (D-L10-12): pure recomputation over durable inputs —
// the observability-class half of L10. It consumes only supplied
// bytes and the registered canonicalization, creates no evaluation,
// invokes nothing through L4/L5, and never touches history. The
// recorded outcome remains the authoritative historical fact
// (D-L10-10 #8); a mismatch here is a new typed discrepancy fact
// whose meaning belongs to Governance.

import (
	"encoding/json"
	"fmt"
)

// CanonicalizeFunc is the registered canonicalization's signature as
// reconstruction consumes it (supplied by the seam's registration;
// pure and environment-free by the D-L10-12 eligibility constraint).
type CanonicalizeFunc func(raw []byte, config json.RawMessage) (string, error)

// Check is one reconstruction comparison: a recorded value against an
// independently recomputed one.
type Check struct {
	Name       string `json:"name"`
	OK         bool   `json:"ok"`
	Recorded   string `json:"recorded"`
	Recomputed string `json:"recomputed"`
}

// Report is the deterministic reconstruction-discrepancy artifact
// content (never a "finding" — Governance classifies meaning). A
// missing required input is a typed failure, never a silent skip.
type Report struct {
	ViewVersion string  `json:"view_version"`
	Consistent  bool    `json:"consistent"`
	Checks      []Check `json:"checks"`
	// MissingInputs lists required durable inputs that could not be
	// supplied — itself a Register T signal.
	MissingInputs []string `json:"missing_inputs,omitempty"`
	// Evaluation identifies the examined record by its content (the
	// committed record is the identity; D-L10-9).
	Evaluation Evaluation `json:"evaluation"`
}

const reconstructViewVersion = "reconstruct-v1"

// Reconstruct performs the D-L10-10 #9 chain over supplied durable
// inputs. It rewrites nothing and grades nothing — it reports
// consistency.
func Reconstruct(record Evaluation, contractBytes, rawBytes, canonicalBytes []byte, canon CanonicalizeFunc) Report {
	rep := Report{ViewVersion: reconstructViewVersion, Evaluation: record, Consistent: true}

	miss := func(name string, b []byte) bool {
		if len(b) == 0 {
			rep.MissingInputs = append(rep.MissingInputs, name)
			rep.Consistent = false
			return true
		}
		return false
	}
	check := func(name, recorded, recomputed string) {
		ok := recorded == recomputed && recorded != ""
		rep.Checks = append(rep.Checks, Check{Name: name, OK: ok, Recorded: recorded, Recomputed: recomputed})
		if !ok {
			rep.Consistent = false
		}
	}

	// Contract bytes ↔ recorded contract identity (L10-computed at
	// load vs H(stored bytes), the two-source rule).
	if !miss("contract_bytes", contractBytes) {
		check("contract_sha256", record.ContractSHA256, hashBytes(contractBytes))
		// The stored contract must still parse and self-declare the
		// recorded identity (two-way, cold).
		if c, err := ParseContract(contractBytes, "reconstruction"); err != nil {
			check("contract_parses", "parseable", "unparseable: "+err.Error())
		} else {
			check("contract_identity", fmt.Sprintf("%s@%d", record.ContractName, record.ContractVersion), fmt.Sprintf("%s@%d", c.Name, c.Contract))
			check("capability_pin", record.Capability, c.Verifier.Capability)

			// Machinery outcomes carry no canonical result; the chain
			// below applies only to contract outcomes.
			if record.Outcome == OutcomePass || record.Outcome == OutcomeFail || record.Outcome == OutcomeInconclusive {
				if !miss("raw_bytes", rawBytes) {
					check("raw_object_id", record.RawObjectID, hashBytes(rawBytes))
					if canon == nil {
						rep.MissingInputs = append(rep.MissingInputs, "canonicalization")
						rep.Consistent = false
					} else if !miss("canonical_bytes", canonicalBytes) {
						check("canonical_object_id", record.CanonicalObjectID, hashBytes(canonicalBytes))
						recanon, cerr := canon(rawBytes, c.Config)
						if cerr != nil {
							check("canonicalization", record.CanonicalResult, "error: "+cerr.Error())
						} else {
							check("canonicalization", record.CanonicalResult, recanon)
							check("canonical_bytes_match", record.CanonicalResult, string(canonicalBytes))
						}
						// Mapping recomputation: the authoritative
						// legitimacy check (matched-mapping is recorded
						// convenience, never a substitute).
						if out, ok := c.ResultMapping[record.CanonicalResult]; ok {
							check("outcome", string(record.Outcome), string(out))
						} else {
							check("outcome", string(record.Outcome), "unmapped:"+record.CanonicalResult)
						}
					}
				}
			}
		}
	}

	return rep
}
