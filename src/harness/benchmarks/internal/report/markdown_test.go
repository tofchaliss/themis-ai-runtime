package report

import (
	"strings"
	"testing"
)

// Zero passed AND zero failed means no check ran — not that the model
// answered everything wrongly. Those are opposite conclusions about the
// same model, and "Average Score: 0%" states the second while the
// evidence supports neither.
//
// Observed 2026-09-15: a gpt-oss:20b report rendered "Average Score:
// 0%" across 19 benchmarks because `themis-bench evaluate` had exited
// non-zero and `validate` never ran. The ratio "Validated: 0/19" was
// right there and the number still misled.
func TestUnvalidatedReportStatesNoScore(t *testing.T) {
	out := Markdown(Report{Model: "m", Date: "2026-09-15", Benchmarks: 19, Validated: 0})
	if strings.Contains(out, "Average Score: **0%**") {
		t.Error("an unvalidated report rendered a 0% score — indistinguishable from a model that answered everything wrongly")
	}
	if !strings.Contains(out, "not validated") {
		t.Errorf("an unvalidated report must say so where the score would be:\n%s", out)
	}
	// A validated report still prints its number, including a real zero.
	scored := Markdown(Report{Model: "m", Date: "2026-09-15", Benchmarks: 2, Validated: 2, AverageScore: 0, TotalFailed: 7})
	if !strings.Contains(scored, "Average Score: **0%**") {
		t.Errorf("a validated zero is a real score and must be printed:\n%s", scored)
	}
}
