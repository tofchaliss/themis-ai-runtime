package report

import (
	"strings"
	"testing"
)

func TestGenerate(t *testing.T) {

	benchmarks := []Benchmark{
		{
			Benchmark: "B001",
			Validated: true,
			Score:     100,
			Passed:    6,

			PromptTokens:     100,
			CompletionTokens: 200,
			TotalTokens:      300,
			TPS:              10,
			GenerationMS:     1000,
			TotalMS:          1500,
		},
		{
			Benchmark: "B002",
			Validated: true,
			Score:     50,
			Passed:    3,
			Failed:    3,

			PromptTokens:     50,
			CompletionTokens: 100,
			TotalTokens:      150,
			TPS:              20,
			GenerationMS:     3000,
			TotalMS:          3500,
		},
		{
			// Not validated: metrics count, score does not.
			Benchmark:    "B003",
			TPS:          30,
			GenerationMS: 2000,
			TotalMS:      2500,
		},
	}

	report := Generate("test-model", "2026-08-19", benchmarks)

	if report.Benchmarks != 3 {
		t.Errorf("Benchmarks = %d, want 3", report.Benchmarks)
	}
	if report.Validated != 2 {
		t.Errorf("Validated = %d, want 2", report.Validated)
	}
	if report.AverageScore != 75 {
		t.Errorf("AverageScore = %d, want 75 (validated only)", report.AverageScore)
	}
	if report.TotalPassed != 9 || report.TotalFailed != 3 {
		t.Errorf("Passed/Failed = %d/%d, want 9/3", report.TotalPassed, report.TotalFailed)
	}
	if report.TotalTokens != 450 {
		t.Errorf("TotalTokens = %d, want 450", report.TotalTokens)
	}
	if report.AverageTPS != 20 {
		t.Errorf("AverageTPS = %f, want 20", report.AverageTPS)
	}
	if report.AverageGenerationMS != 2000 {
		t.Errorf("AverageGenerationMS = %f, want 2000", report.AverageGenerationMS)
	}
}

func TestGenerateEmpty(t *testing.T) {

	report := Generate("test-model", "2026-08-19", nil)

	if report.Benchmarks != 0 || report.AverageScore != 0 {
		t.Errorf("empty report: Benchmarks=%d AverageScore=%d, want 0/0",
			report.Benchmarks, report.AverageScore)
	}
}

func TestMarkdown(t *testing.T) {

	report := Generate("test-model", "2026-08-19", []Benchmark{
		{Benchmark: "B001", Validated: true, Score: 83},
		{Benchmark: "B002"},
	})

	md := Markdown(report)

	for _, want := range []string{
		"**Model:** test-model",
		"**Date:** 2026-08-19",
		"| B001",
		"83%",
		"Validated: **1/2**",
	} {
		if !strings.Contains(md, want) {
			t.Errorf("markdown missing %q", want)
		}
	}

	// Unvalidated benchmarks show "-" instead of a misleading 0%.
	if !strings.Contains(md, "| B002      |     - |") {
		t.Errorf("unvalidated benchmark should render '-' score:\n%s", md)
	}
}
