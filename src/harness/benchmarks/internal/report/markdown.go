package report

import (
	"fmt"
	"strings"
)

// Markdown renders a report as Markdown.
func Markdown(report Report) string {

	var b strings.Builder

	fmt.Fprintf(&b, "# Themis Benchmark Report\n\n")

	fmt.Fprintf(&b, "**Model:** %s\n\n", report.Model)
	fmt.Fprintf(&b, "**Date:** %s\n\n", report.Date)

	fmt.Fprintf(&b, "## Summary\n\n")

	fmt.Fprintf(&b, "- Benchmarks: **%d**\n", report.Benchmarks)
	fmt.Fprintf(&b, "- Prompt Tokens: **%d**\n", report.TotalPromptTokens)
	fmt.Fprintf(&b, "- Completion Tokens: **%d**\n", report.TotalCompletionTokens)
	fmt.Fprintf(&b, "- Total Tokens: **%d**\n", report.TotalTokens)
	fmt.Fprintf(&b, "- Average TPS: **%.2f**\n", report.AverageTPS)
	fmt.Fprintf(&b, "- Average Generation Time: **%.2f ms**\n", report.AverageGenerationMS)
	fmt.Fprintf(&b, "- Average Total Time: **%.2f ms**\n", report.AverageTotalMS)

	// Validation Summary
	fmt.Fprintf(&b, "- Validated: **%d/%d**\n", report.Validated, report.Benchmarks)
	fmt.Fprintf(&b, "- Average Score: **%d%%**\n", report.AverageScore)
	fmt.Fprintf(&b, "- Passed Checks: **%d**\n", report.TotalPassed)
	fmt.Fprintf(&b, "- Failed Checks: **%d**\n", report.TotalFailed)

	fmt.Fprintf(&b, "\n")

	fmt.Fprintf(&b, "## Benchmark Results\n\n")

	fmt.Fprintf(
		&b,
		"| Benchmark | Score | Passed | Failed | Violations | Prompt | Completion | Total | TPS | Load (ms) | Prompt (ms) | Generation (ms) | Total (ms) |\n",
	)

	fmt.Fprintf(
		&b,
		"|-----------|------:|-------:|-------:|-----------:|-------:|-----------:|------:|----:|----------:|------------:|----------------:|-----------:|\n",
	)

	for _, benchmark := range report.Results {

		score := "-"
		if benchmark.Validated {
			score = fmt.Sprintf("%d%%", benchmark.Score)
		}

		fmt.Fprintf(&b, "| %-9s | %5s | %6d | %6d | %10d | %6d | %10d | %5d | %5.2f | %10.2f | %11.2f | %15.2f | %10.2f |\n",
			benchmark.Benchmark,
			score,
			benchmark.Passed,
			benchmark.Failed,
			benchmark.Violations,
			benchmark.PromptTokens,
			benchmark.CompletionTokens,
			benchmark.TotalTokens,
			benchmark.TPS,
			benchmark.LoadMS,
			benchmark.PromptMS,
			benchmark.GenerationMS,
			benchmark.TotalMS,
		)
	}

	return b.String()
}
