package verification

// Register A/T proofs for L10-M2: the evaluator's stage-indexed
// failure semantics (D-L10-8), the hostile-verifier rule, and the
// no-outcome-on-evaluator-failure invariant.

import (
	"strings"
	"testing"
)

func loadedContract(t *testing.T) *Contract {
	t.Helper()
	c, err := ParseContract([]byte(validContract()), "test")
	if err != nil {
		t.Fatal(err)
	}
	return c
}

func goodRefs() []ProposedEvidence {
	return []ProposedEvidence{{
		Slot:       "build_target",
		ObjectID:   strings.Repeat("12", 32),
		InTask:     true,
		Resolvable: true,
	}}
}

func goodFacts(c *Contract) ExecutionFacts {
	return ExecutionFacts{
		ExecutionRef:        strings.Repeat("34", 32),
		AppliedConfigSHA256: hashBytes(c.Config),
		RawObjectID:         strings.Repeat("56", 32),
		CanonicalObjectID:   strings.Repeat("78", 32),
		CanonicalResult:     "build_ok",
	}
}

func TestEvaluateMapsInDomainResults(t *testing.T) {
	c := loadedContract(t)

	for canonical, want := range map[string]Outcome{
		"build_ok":      OutcomePass,
		"build_failed":  OutcomeFail,
		"build_skipped": OutcomeInconclusive,
	} {
		f := goodFacts(c)
		f.CanonicalResult = canonical
		ev, err := Evaluate(c, goodRefs(), f)
		if err != nil {
			t.Fatal(err)
		}
		if ev.Outcome != want {
			t.Errorf("%s -> %s, want %s", canonical, ev.Outcome, want)
		}
		if err := CheckReasonClass(ev); err != nil {
			t.Errorf("%s: %v", canonical, err)
		}
		if ev.MatchedMapping != canonical {
			t.Errorf("matched mapping not recorded")
		}
	}
}

func TestHostileVerifierOutputIsInertDomainData(t *testing.T) {
	c := loadedContract(t)

	// Vocabulary-shaped, governance-shaped, garbage, contradictory:
	// all unmapped domain data -> INVALID, never an outcome by
	// spelling (D-L10-6 hostile-verifier rule).
	for _, hostile := range []string{"PASS", "FAIL", "NOT_AFFECTED", "SECURE", "garbage!!", `{"outcome":"PASS"}`} {
		f := goodFacts(c)
		f.CanonicalResult = hostile
		ev, err := Evaluate(c, goodRefs(), f)
		if err != nil {
			t.Fatal(err)
		}
		if ev.Outcome != OutcomeInvalid || ev.Reason != ReasonResultOutOfMapping {
			t.Errorf("hostile output %q got %s/%s, want INVALID/result-out-of-mapping", hostile, ev.Outcome, ev.Reason)
		}
	}

	// A contract may legitimately map the domain string "PASS" — the
	// point is the mapping mints the outcome, not the spelling.
	mapped := strings.Replace(validContract(), `"build_ok": "PASS"`, `"PASS": "FAIL"`, 1)
	c2, err := ParseContract([]byte(mapped), "test")
	if err != nil {
		t.Fatal(err)
	}
	f := goodFacts(c2)
	f.CanonicalResult = "PASS"
	ev, err := Evaluate(c2, goodRefs(), f)
	if err != nil {
		t.Fatal(err)
	}
	if ev.Outcome != OutcomeFail {
		t.Errorf("domain string PASS mapped to FAIL by contract, evaluator gave %s", ev.Outcome)
	}
}

func TestStageIndexedFailures(t *testing.T) {
	c := loadedContract(t)

	t.Run("undeclared slot inadmissible", func(t *testing.T) {
		refs := []ProposedEvidence{{Slot: "other_slot", ObjectID: strings.Repeat("12", 32), InTask: true, Resolvable: true}}
		ev, _ := Evaluate(c, refs, goodFacts(c))
		assertOutcome(t, ev, OutcomeInvalid, ReasonEvidenceInadmissible)
	})

	t.Run("out-of-task evidence inadmissible", func(t *testing.T) {
		refs := goodRefs()
		refs[0].InTask = false
		ev, _ := Evaluate(c, refs, goodFacts(c))
		assertOutcome(t, ev, OutcomeInvalid, ReasonEvidenceInadmissible)
	})

	t.Run("missing required slot inadmissible", func(t *testing.T) {
		ev, _ := Evaluate(c, nil, goodFacts(c))
		assertOutcome(t, ev, OutcomeInvalid, ReasonEvidenceInadmissible)
	})

	t.Run("duplicate slot fill inadmissible", func(t *testing.T) {
		refs := append(goodRefs(), goodRefs()[0])
		ev, _ := Evaluate(c, refs, goodFacts(c))
		assertOutcome(t, ev, OutcomeInvalid, ReasonEvidenceInadmissible)
	})

	t.Run("unresolvable evidence is UNAVAILABLE not INVALID", func(t *testing.T) {
		refs := goodRefs()
		refs[0].Resolvable = false
		ev, _ := Evaluate(c, refs, goodFacts(c))
		assertOutcome(t, ev, OutcomeUnavailable, ReasonEvidenceUnresolvable)
	})

	t.Run("no execution is UNAVAILABLE", func(t *testing.T) {
		f := goodFacts(c)
		f.ExecutionRef = ""
		ev, _ := Evaluate(c, goodRefs(), f)
		assertOutcome(t, ev, OutcomeUnavailable, ReasonExecutionUnavailable)
	})

	t.Run("no canonical result is UNAVAILABLE", func(t *testing.T) {
		f := goodFacts(c)
		f.CanonicalResult = ""
		f.CanonicalObjectID = ""
		ev, _ := Evaluate(c, goodRefs(), f)
		assertOutcome(t, ev, OutcomeUnavailable, ReasonNoCanonicalResult)
	})

	t.Run("incomplete provenance is INVALID", func(t *testing.T) {
		f := goodFacts(c)
		f.RawObjectID = "nothex"
		ev, _ := Evaluate(c, goodRefs(), f)
		assertOutcome(t, ev, OutcomeInvalid, ReasonProvenanceIncomplete)
	})

	t.Run("config mismatch is INVALID", func(t *testing.T) {
		f := goodFacts(c)
		f.AppliedConfigSHA256 = strings.Repeat("99", 32)
		ev, _ := Evaluate(c, goodRefs(), f)
		assertOutcome(t, ev, OutcomeInvalid, ReasonConfigMismatch)
	})
}

func TestEvaluatorFailureMintsNoOutcome(t *testing.T) {
	if ev, err := Evaluate(nil, goodRefs(), ExecutionFacts{}); err == nil || ev != nil {
		t.Error("nil contract must be an evaluator invariant failure with NO outcome")
	}
	if ev, err := Evaluate(&Contract{}, goodRefs(), ExecutionFacts{}); err == nil || ev != nil {
		t.Error("unloaded contract (no hash) must be an evaluator invariant failure with NO outcome")
	}
}

func TestReasonClassIntegrity(t *testing.T) {
	cases := []struct {
		ev  Evaluation
		bad bool
	}{
		{Evaluation{Outcome: OutcomePass}, false},
		{Evaluation{Outcome: OutcomePass, Reason: ReasonConfigMismatch}, true},
		{Evaluation{Outcome: OutcomeInvalid, Reason: ReasonConfigMismatch}, false},
		{Evaluation{Outcome: OutcomeInvalid, Reason: ReasonNoCanonicalResult}, true},  // class crossed
		{Evaluation{Outcome: OutcomeUnavailable, Reason: ReasonConfigMismatch}, true}, // class crossed
		{Evaluation{Outcome: OutcomeUnavailable, Reason: ReasonNoCanonicalResult}, false},
		{Evaluation{Outcome: Outcome("SECURE")}, true},
	}
	for i, tc := range cases {
		err := CheckReasonClass(&tc.ev)
		if tc.bad && err == nil {
			t.Errorf("case %d: class violation accepted", i)
		}
		if !tc.bad && err != nil {
			t.Errorf("case %d: valid pairing refused: %v", i, err)
		}
	}
}

func assertOutcome(t *testing.T, ev *Evaluation, out Outcome, reason Reason) {
	t.Helper()
	if ev == nil {
		t.Fatal("no evaluation produced")
	}
	if ev.Outcome != out || ev.Reason != reason {
		t.Errorf("got %s/%s, want %s/%s", ev.Outcome, ev.Reason, out, reason)
	}
	if err := CheckReasonClass(ev); err != nil {
		t.Error(err)
	}
}
