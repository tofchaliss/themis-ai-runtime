package verification

// Record-derived views (D-L10-5): pure, versioned, reproducible
// functions over declared record slices, carrying derived-from
// provenance. Views expose recorded facts; they never mutate, never
// verdict, and never feed workflow control. Sensitivity inheritance
// (D-L10-14): a view is as protected as its most sensitive input;
// these views carry only contract tokens, outcomes, and sequence
// numbers — no evidence content.

// VerificationEventView is one committed EvVerification event's
// semantic content as a view consumes it.
type VerificationEventView struct {
	Seq      int64  `json:"seq"`
	Contract string `json:"contract"`
	Outcome  string `json:"outcome"`
}

// DerivedFrom is the view's provenance: the event range consumed and
// the view function's version — sufficient for any consumer to
// re-derive the view (D-L10-5).
type DerivedFrom struct {
	FirstSeq    int64  `json:"first_seq"`
	LastSeq     int64  `json:"last_seq"`
	Events      int    `json:"events"`
	ViewVersion string `json:"view_version"`
}

// HistoryView is the verification history of one task: every
// evaluation instance in commit order plus the latest-per-contract
// projection — the same projection δ maintains as walk state, derived
// here independently from the record (the replayer-equivalence
// property, exposed).
type HistoryView struct {
	Derived DerivedFrom             `json:"derived_from"`
	History []VerificationEventView `json:"history"`
	Latest  map[string]string       `json:"latest_per_contract"`
}

const historyViewVersion = "verification-history-v1"

// VerificationHistory derives the history view from a task's
// verification events, supplied in commit order. Pure: same events →
// same view, always.
func VerificationHistory(events []VerificationEventView) HistoryView {
	v := HistoryView{
		Derived: DerivedFrom{ViewVersion: historyViewVersion, Events: len(events)},
		Latest:  map[string]string{},
	}
	for i, e := range events {
		if i == 0 {
			v.Derived.FirstSeq = e.Seq
		}
		v.Derived.LastSeq = e.Seq
		v.History = append(v.History, e)
		v.Latest[e.Contract] = e.Outcome
	}
	return v
}
