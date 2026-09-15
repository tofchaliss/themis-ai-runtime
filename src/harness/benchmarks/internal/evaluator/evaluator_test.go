package evaluator

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeRunFile(t *testing.T, content string) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), "B001.json")

	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	return path
}

func TestEvaluateRun(t *testing.T) {

	t.Run("envelope run file", func(t *testing.T) {
		path := writeRunFile(t, `{
			"benchmark": "B001",
			"model": "test-model",
			"runtime": "openai",
			"answer": "hello",
			"options": {"temperature": 0, "seed": 42},
			"metrics": {
				"prompt_tokens": 10,
				"completion_tokens": 20,
				"total_tokens": 30,
				"generation_time_ms": 4000,
				"tokens_per_second": 5
			},
			"raw": {"anything": true}
		}`)

		r, err := EvaluateRun("B001", path)
		if err != nil {
			t.Fatal(err)
		}

		if r.Runtime != "openai" || r.Model != "test-model" || r.Answer != "hello" {
			t.Errorf("result = %+v", r)
		}
		if r.Metrics.CompletionTokens != 20 || r.Metrics.TokensPerSecond != 5 {
			t.Errorf("metrics = %+v", r.Metrics)
		}
	})

	t.Run("legacy ollama run file", func(t *testing.T) {
		path := writeRunFile(t, `{
			"model": "test-model",
			"response": "hello",
			"done": true,
			"prompt_eval_count": 10,
			"eval_count": 20,
			"eval_duration": 4000000000,
			"total_duration": 5000000000
		}`)

		r, err := EvaluateRun("B001", path)
		if err != nil {
			t.Fatal(err)
		}

		if r.Runtime != "ollama" || r.Answer != "hello" {
			t.Errorf("result = %+v", r)
		}
		if r.Metrics.GenerationTimeMS != 4000 {
			t.Errorf("GenerationTimeMS = %f, want 4000", r.Metrics.GenerationTimeMS)
		}
		// 20 tokens over 4 seconds.
		if r.Metrics.TokensPerSecond != 5 {
			t.Errorf("TokensPerSecond = %f, want 5", r.Metrics.TokensPerSecond)
		}
	})

	t.Run("legacy error payload is rejected", func(t *testing.T) {
		path := writeRunFile(t, `{"error": "model not found"}`)

		if _, err := EvaluateRun("B001", path); err == nil {
			t.Error("expected error for runtime-error payload")
		}
	})

	t.Run("legacy incomplete run is rejected", func(t *testing.T) {
		path := writeRunFile(t, `{"model": "m", "response": "partial", "done": false}`)

		if _, err := EvaluateRun("B001", path); err == nil {
			t.Error("expected error for incomplete run")
		}
	})

	t.Run("malformed JSON is rejected", func(t *testing.T) {
		path := writeRunFile(t, `not json`)

		if _, err := EvaluateRun("B001", path); err == nil {
			t.Error("expected error for malformed run file")
		}
	})
}

// A record carrying a runtime IS an envelope. Deciding that by the
// ANSWER instead sent every empty-answer envelope down the legacy path,
// where the file was re-parsed as a bare Ollama payload; an envelope
// keeps the runtime's fields under "raw", so the legacy parser found no
// top-level "done" and reported "done=false; re-run the benchmark".
//
// Three things wrong at once, all observed on a real gpt-oss:20b run of
// B005 (2026-09-15): the record's own done was TRUE, the cause was in
// done_reason, and with temperature 0 and a pinned seed a re-run
// reproduces the output exactly — so the advice sent the operator to
// repeat a deterministic result.
//
// The shapes below are the ones that actually occur. Each must be named
// for what it is, because a transport fault, a refusal, and a model that
// spends its whole budget without answering call for three different
// responses.
func TestEmptyAnswerIsDiagnosedNotMisreported(t *testing.T) {
	dir := t.TempDir()
	write := func(t *testing.T, name, body string) string {
		t.Helper()
		p := filepath.Join(dir, name)
		if err := os.WriteFile(p, []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
		return p
	}

	for what, tc := range map[string]struct{ raw, want string }{
		// The observed B005 case: a reasoning model that looped until
		// the token ceiling, emitting everything into `thinking`.
		"budget spent in the reasoning channel": {
			`{"done":true,"done_reason":"length","eval_count":3911,"thinking":"` +
				strings.Repeat("I'm not sure. ", 40) + `","response":""}`,
			"all of it in the reasoning channel",
		},
		"truncated with no reasoning channel": {
			`{"done":true,"done_reason":"length","eval_count":4096,"response":""}`,
			"done_reason=length after 4096 tokens with an empty answer",
		},
		"generation genuinely unfinished": {
			`{"done":false,"response":""}`,
			"generation did not finish",
		},
		"runtime refused": {
			`{"done":false,"error":"model not found","response":""}`,
			"runtime error: model not found",
		},
		"stopped cleanly but said nothing": {
			`{"done":true,"done_reason":"stop","eval_count":2,"response":""}`,
			"done_reason=stop with an empty answer",
		},
	} {
		body := `{"benchmark":"B005","model":"m","runtime":"ollama","answer":"",` +
			`"metrics":{},"raw":` + tc.raw + `}`
		_, err := EvaluateRun("B005", write(t, "e-"+strings.ReplaceAll(what, " ", "-")+".json", body))
		if err == nil {
			t.Errorf("%s: an empty answer evaluated as a result", what)
			continue
		}
		if strings.Contains(err.Error(), "done=false; re-run") {
			t.Errorf("%s: reported the legacy misdiagnosis: %v", what, err)
		}
		if !strings.Contains(err.Error(), tc.want) {
			t.Errorf("%s: cause not named — got %v, want it to mention %q", what, err, tc.want)
		}
	}

	// A populated envelope still evaluates, so the change narrows to
	// empty answers rather than refusing envelopes generally.
	ok := write(t, "ok.json", `{"benchmark":"B005","model":"m","runtime":"ollama",`+
		`"answer":"an answer","metrics":{},"raw":{"done":true,"done_reason":"stop"}}`)
	res, err := EvaluateRun("B005", ok)
	if err != nil || res.Answer != "an answer" {
		t.Fatalf("a populated envelope must evaluate: %+v %v", res, err)
	}

	// And a genuine legacy file — no runtime field — still takes the
	// legacy path, where done=false is the true diagnosis.
	legacy := write(t, "legacy.json", `{"model":"m","response":"x","done":false}`)
	if _, lerr := EvaluateRun("B005", legacy); lerr == nil ||
		!strings.Contains(lerr.Error(), "done=false") {
		t.Fatalf("a legacy record must still be judged on its own top-level done: %v", lerr)
	}
}
