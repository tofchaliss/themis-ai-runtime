package context

import (
	"errors"
	"testing"
)

// D-P-1: the harness-state kind mints the derived class only — a fact
// the loop computes from the record — and no other kind may claim it.
func TestHarnessStateKindMintsDerivedOnly(t *testing.T) {
	ok := Source{Name: "phase-state", Kind: KindHarnessState, Authority: AuthorityDerived, Sensitivity: SensitivityPublic, Author: "harness",
		Items: []ContextItem{{Kind: "phase-state", Evidence: []byte(`{"phase":"ANALYZE"}`)}}}
	if err := checkSource(ok); err != nil {
		t.Fatal(err)
	}
	items, avail, err := ok.collect()
	if err != nil || !avail || len(items) != 1 || items[0].Provenance.Origin != "harness" || items[0].Authority != AuthorityDerived {
		t.Fatalf("collect: %+v %v %v", items, avail, err)
	}
	for _, bad := range []Source{
		{Name: "phase-state", Kind: KindHarnessState, Authority: AuthorityGovernedRecord, Sensitivity: SensitivityPublic},
		{Name: "phase-state", Kind: KindHarnessState, Authority: AuthorityExternalUntrusted, Sensitivity: SensitivityPublic},
		{Name: "task-payload", Kind: KindInline, Authority: AuthorityDerived, Sensitivity: SensitivityPublic},
	} {
		if err := checkSource(bad); !errors.Is(err, ErrUnrecognizedSource) {
			t.Fatalf("%s/%s must be refused: %v", bad.Kind, bad.Authority, err)
		}
	}
}
