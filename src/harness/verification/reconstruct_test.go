package verification

// Register P proofs (M5): reconstruction is pure, every single-element
// swap fails, missing inputs are typed, and views are reproducible.

import (
	"encoding/json"
	"strings"
	"testing"
)

func reconstructFixture(t *testing.T) (Evaluation, []byte, []byte, []byte, CanonicalizeFunc) {
	t.Helper()
	contractBytes := []byte(validContract())
	c, err := ParseContract(contractBytes, "fixture")
	if err != nil {
		t.Fatal(err)
	}
	raw := []byte(`build output: ok`)
	canonical := []byte("build_ok")
	canon := func(rawIn []byte, cfg json.RawMessage) (string, error) {
		if string(rawIn) == `build output: ok` {
			return "build_ok", nil
		}
		return "build_failed", nil
	}
	record := Evaluation{
		ContractName:      c.Name,
		ContractVersion:   c.Contract,
		ContractSHA256:    c.SHA256,
		Capability:        c.Verifier.Capability,
		RegistrySHA256:    c.Verifier.RegistrySHA256,
		Evidence:          []EvidenceRecord{{Slot: "build_target", ObjectID: hashBytes(raw)}},
		ExecutionRef:      "l4:9",
		RawObjectID:       hashBytes(raw),
		CanonicalObjectID: hashBytes(canonical),
		CanonicalResult:   "build_ok",
		Outcome:           OutcomePass,
		MatchedMapping:    "build_ok",
	}
	return record, contractBytes, raw, canonical, canon
}

func TestReconstructConsistent(t *testing.T) {
	record, cb, raw, canonical, canon := reconstructFixture(t)
	rep := Reconstruct(record, cb, raw, canonical, canon)
	if !rep.Consistent {
		t.Fatalf("consistent evaluation reported inconsistent: %+v", rep)
	}
	// Purity: identical inputs, identical report.
	rep2 := Reconstruct(record, cb, raw, canonical, canon)
	b1, _ := json.Marshal(rep)
	b2, _ := json.Marshal(rep2)
	if string(b1) != string(b2) {
		t.Error("reconstruction is not pure")
	}
}

func TestReconstructSingleElementSwapsFail(t *testing.T) {
	base, cb, raw, canonical, canon := reconstructFixture(t)

	t.Run("foreign contract bytes", func(t *testing.T) {
		foreign := []byte(strings.Replace(validContract(), `"-trimpath"`, `"-other"`, 1))
		if rep := Reconstruct(base, foreign, raw, canonical, canon); rep.Consistent {
			t.Error("swapped contract bytes reconstructed consistent")
		}
	})
	t.Run("claimed outcome", func(t *testing.T) {
		r := base
		r.Outcome = OutcomeFail // claim FAIL over a PASS-mapping result
		if rep := Reconstruct(r, cb, raw, canonical, canon); rep.Consistent {
			t.Error("outcome claim survived mapping recomputation")
		}
	})
	t.Run("swapped raw bytes", func(t *testing.T) {
		if rep := Reconstruct(base, cb, []byte("other raw"), canonical, canon); rep.Consistent {
			t.Error("swapped raw bytes reconstructed consistent")
		}
	})
	t.Run("swapped canonical bytes", func(t *testing.T) {
		if rep := Reconstruct(base, cb, raw, []byte("build_failed"), canon); rep.Consistent {
			t.Error("swapped canonical bytes reconstructed consistent")
		}
	})
	t.Run("claimed contract identity", func(t *testing.T) {
		r := base
		r.ContractSHA256 = strings.Repeat("aa", 32) // proposal-supplied claim
		if rep := Reconstruct(r, cb, raw, canonical, canon); rep.Consistent {
			t.Error("claimed contract identity survived the two-source check")
		}
	})
	t.Run("claimed capability", func(t *testing.T) {
		r := base
		r.Capability = "other_capability"
		if rep := Reconstruct(r, cb, raw, canonical, canon); rep.Consistent {
			t.Error("claimed capability survived the pin check")
		}
	})
}

func TestReconstructMissingInputsTyped(t *testing.T) {
	record, cb, raw, canonical, canon := reconstructFixture(t)

	cases := map[string]func() Report{
		"contract_bytes":   func() Report { return Reconstruct(record, nil, raw, canonical, canon) },
		"raw_bytes":        func() Report { return Reconstruct(record, cb, nil, canonical, canon) },
		"canonical_bytes":  func() Report { return Reconstruct(record, cb, raw, nil, canon) },
		"canonicalization": func() Report { return Reconstruct(record, cb, raw, canonical, nil) },
	}
	for name, run := range cases {
		t.Run(name, func(t *testing.T) {
			rep := run()
			if rep.Consistent {
				t.Fatal("missing input reported consistent")
			}
			found := false
			for _, m := range rep.MissingInputs {
				if m == name {
					found = true
				}
			}
			if !found {
				t.Errorf("missing input %q not typed: %v", name, rep.MissingInputs)
			}
		})
	}
}

func TestReconstructMachineryOutcomesSkipResultChain(t *testing.T) {
	// An UNAVAILABLE record has no canonical result; reconstruction
	// checks contract identity only and must not demand raw/canonical.
	record, cb, _, _, canon := reconstructFixture(t)
	record.Outcome = OutcomeUnavailable
	record.Reason = ReasonNoCanonicalResult
	record.RawObjectID, record.CanonicalObjectID, record.CanonicalResult = "", "", ""
	rep := Reconstruct(record, cb, nil, nil, canon)
	if !rep.Consistent {
		t.Fatalf("machinery-outcome record inconsistent: %+v", rep)
	}
}

func TestVerificationHistoryView(t *testing.T) {
	events := []VerificationEventView{
		{Seq: 4, Contract: "report-valid@1", Outcome: "FAIL"},
		{Seq: 9, Contract: "other@2", Outcome: "PASS"},
		{Seq: 12, Contract: "report-valid@1", Outcome: "PASS"},
	}
	v := VerificationHistory(events)
	if v.Latest["report-valid@1"] != "PASS" || v.Latest["other@2"] != "PASS" {
		t.Errorf("latest projection wrong: %v", v.Latest)
	}
	if v.Derived.FirstSeq != 4 || v.Derived.LastSeq != 12 || v.Derived.Events != 3 {
		t.Errorf("derived-from provenance wrong: %+v", v.Derived)
	}
	// Downgrade ordering: a later UNAVAILABLE displaces PASS.
	v2 := VerificationHistory(append(events, VerificationEventView{Seq: 20, Contract: "report-valid@1", Outcome: "UNAVAILABLE"}))
	if v2.Latest["report-valid@1"] != "UNAVAILABLE" {
		t.Error("downgrade did not displace PASS in the latest projection")
	}
	// Purity.
	a, _ := json.Marshal(VerificationHistory(events))
	b, _ := json.Marshal(VerificationHistory(events))
	if string(a) != string(b) {
		t.Error("view is not pure")
	}
}
