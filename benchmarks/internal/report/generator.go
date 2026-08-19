package report

// Generate creates a report from benchmark results.
func Generate(model, date string, benchmarks []Benchmark) Report {

	report := Report{
		Model:      model,
		Date:       date,
		Benchmarks: len(benchmarks),
		Results:    benchmarks,
	}

	var totalScore int

	for _, benchmark := range benchmarks {

		report.TotalPromptTokens += benchmark.PromptTokens
		report.TotalCompletionTokens += benchmark.CompletionTokens
		report.TotalTokens += benchmark.TotalTokens

		report.AverageTPS += benchmark.TPS
		report.AverageGenerationMS += benchmark.GenerationMS
		report.AverageTotalMS += benchmark.TotalMS

		if benchmark.Validated {
			report.Validated++
			report.TotalPassed += benchmark.Passed
			report.TotalFailed += benchmark.Failed
			totalScore += benchmark.Score
		}
	}

	if report.Benchmarks > 0 {

		count := float64(report.Benchmarks)

		report.AverageTPS /= count
		report.AverageGenerationMS /= count
		report.AverageTotalMS /= count
	}

	if report.Validated > 0 {
		report.AverageScore = totalScore / report.Validated
	}

	return report
}
