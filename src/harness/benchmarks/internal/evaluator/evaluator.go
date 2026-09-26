package evaluator

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/tofchaliss/themis-ai-runtime/src/harness/internal/llm"
)

// EvaluateRun normalizes a single run file. Run files are runtime-
// agnostic envelopes (llm.RunRecord); raw Ollama payloads written by
// older versions of the tool are still accepted.
func EvaluateRun(
	benchmark string,
	filename string,
) (*Result, error) {

	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	var record llm.RunRecord

	if err := json.Unmarshal(data, &record); err != nil {
		return nil, fmt.Errorf("parse run file %s: %w", filename, err)
	}

	// A record carrying a runtime IS an envelope. Deciding that by the
	// ANSWER instead sent every empty-answer envelope down the legacy
	// path, where the file is re-parsed as a bare Ollama payload — and
	// since an envelope keeps the runtime's fields under "raw", the
	// legacy parser found no top-level "done" and reported
	// "done=false; re-run the benchmark".
	//
	// Every part of that was wrong. The record's own done is TRUE; the
	// cause is in done_reason; and with temperature 0 and a pinned seed
	// a re-run reproduces the same output exactly, so the advice sent
	// the operator to repeat a deterministic result.
	if record.Runtime != "" {
		if record.Answer == "" {
			return nil, fmt.Errorf(
				"%s: the model produced no answer (%s) — this is a result about the model, not a transport fault, and re-running with a pinned seed reproduces it",
				benchmark, describeEmptyAnswer(record.Raw),
			)
		}
		return &Result{
			Benchmark: benchmark,
			Model:     record.Model,
			Runtime:   record.Runtime,
			Answer:    record.Answer,
			Metrics:   Metrics(record.Metrics),
		}, nil
	}

	return evaluateLegacyRun(benchmark, filename, data)
}

// describeEmptyAnswer reports WHY the answer is empty, from the
// runtime's own record. Saying only "empty" would leave the operator to
// guess between a transport fault, a refusal, and a model that spent
// its whole budget without emitting anything — three different facts
// calling for three different responses.
func describeEmptyAnswer(raw json.RawMessage) string {
	if len(raw) == 0 {
		return "no runtime record retained"
	}
	var r struct {
		Done       bool   `json:"done"`
		DoneReason string `json:"done_reason"`
		Thinking   string `json:"thinking"`
		EvalCount  int    `json:"eval_count"`
		Error      string `json:"error"`
	}
	if err := json.Unmarshal(raw, &r); err != nil {
		return "runtime record unreadable"
	}
	switch {
	case r.Error != "":
		return "runtime error: " + r.Error
	case !r.Done:
		return "generation did not finish"
	case r.DoneReason == "length" && len(r.Thinking) > 0:
		// Reasoning models emit intermediate text separately. Budget
		// spent there and never converted into an answer is a model
		// result — commonly a repetition loop.
		return fmt.Sprintf(
			"done_reason=length after %d tokens, all of it in the reasoning channel (%d bytes) and none in the answer",
			r.EvalCount, len(r.Thinking))
	case r.DoneReason == "length":
		return fmt.Sprintf("done_reason=length after %d tokens with an empty answer", r.EvalCount)
	case r.DoneReason != "":
		return "done_reason=" + r.DoneReason + " with an empty answer"
	}
	return "the runtime reported completion with an empty answer"
}

// evaluateLegacyRun handles pre-envelope run files, which are raw
// Ollama /api/generate payloads.
func evaluateLegacyRun(
	benchmark string,
	filename string,
	data []byte,
) (*Result, error) {

	var raw llm.OllamaResponse

	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("parse run file %s: %w", filename, err)
	}

	if raw.Error != "" {
		return nil, fmt.Errorf(
			"run file %s contains a runtime error: %s",
			filename,
			raw.Error,
		)
	}

	if raw.Model == "" {
		return nil, fmt.Errorf("invalid run file %s: missing model", filename)
	}

	if !raw.Done {
		return nil, fmt.Errorf(
			"%s: run is incomplete (done=false); re-run the benchmark",
			benchmark,
		)
	}

	return &Result{
		Benchmark: benchmark,
		Model:     raw.Model,
		Runtime:   "ollama",
		Answer:    raw.Response,
		Metrics:   Metrics(raw.Metrics()),
	}, nil
}

// EvaluateAll normalizes every run for the given date and model. It
// keeps going when a single run fails and reports all failures at the
// end.
func EvaluateAll(
	root string,
	date string,
	model string,
) error {

	files, err := FindRuns(root, date, model)
	if err != nil {
		return err
	}

	if len(files) == 0 {
		return fmt.Errorf(
			"no runs found in %s",
			filepath.Join(root, "runs", date, model),
		)
	}

	failed := 0

	for _, file := range files {

		benchmark := strings.TrimSuffix(
			filepath.Base(file),
			filepath.Ext(file),
		)

		result, err := EvaluateRun(benchmark, file)
		if err != nil {
			fmt.Fprintf(os.Stderr, "✗ %s: %v\n", benchmark, err)
			failed++
			continue
		}

		if err := WriteResult(
			root,
			date,
			model,
			benchmark,
			result,
		); err != nil {
			fmt.Fprintf(os.Stderr, "✗ %s: %v\n", benchmark, err)
			failed++
			continue
		}

		fmt.Printf("✓ %s evaluated\n", benchmark)
	}

	if failed > 0 {
		return fmt.Errorf(
			"%d of %d runs failed to evaluate",
			failed,
			len(files),
		)
	}

	return nil
}
