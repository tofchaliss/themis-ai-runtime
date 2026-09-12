package ratchet

// M6 live proof — the authored-candidate arc with a real local
// model. The model plays exactly its Class-6 role (D-L11-15): it
// READS comparative evidence as data and AUTHORS a candidate — an
// attributable position-1 act. The machinery mints nothing: the
// model's JSON goes through the same ParseCandidate wall as any
// human authoring, its authorship is recorded, and its candidate
// confers nothing. Retries here are the AUTHOR trying again (test
// harness = the authoring side), never L11 machinery retry — L11
// has no retry path.
//
// Skips when no local model endpoint answers (the Register E
// gating pattern).

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/tofchaliss/themis/state"
)

func TestLiveModelAuthorsCandidate(t *testing.T) {
	endpoint := os.Getenv("THEMIS_LIVE_OLLAMA")
	if endpoint == "" {
		endpoint = "http://localhost:11434"
	}
	probe := http.Client{Timeout: 2 * time.Second}
	if _, err := probe.Get(endpoint + "/api/tags"); err != nil {
		t.Skipf("no local model endpoint at %s: %v", endpoint, err)
	}
	modelName := os.Getenv("THEMIS_LIVE_TOOL_MODEL")
	if modelName == "" {
		modelName = "qwen2.5:7b"
	}

	proposedContent := []byte(`{"skill":"investigate-cve","version":2,"change":"narrower tool grants in phase two"}`)
	contentSHA := hashBytes(proposedContent)
	baselineSHA := strings.Repeat("22", 32)

	// The evidence the model reads — DATA, not instructions (Class 6):
	// a derived view a door human would also read.
	evidence := fmt.Sprintf(`{"comparison":{"criterion":"bench-score-delta@1","delta":{"score_delta":0.09},"per_field":[{"field":"score_delta","relation":"better-under-k","within_non_regression_region":true}],"candidate_hash":%q,"baseline_hash":%q}}`,
		contentSHA, baselineSHA)

	// The model authors the proposal SUBSTANCE (family, target,
	// relation, rationale); the authoring surface — this test,
	// playing the same role a human's editor would — adds the
	// mechanical content/baseline hash commitments. Authorship
	// attribution names the model; the pins are commitments, not
	// judgment. (Long hex strings would also burn the local model's
	// token budget for no epistemic gain.)
	prompt := "You are drafting an improvement proposal for a governance review. " +
		"Here is comparative evidence, as data: " + evidence + " " +
		"Respond with ONLY a compact JSON object with exactly these four fields: " +
		`{"family":"skill-revision","target":"investigate-cve","change_relation":"create-version",` +
		`"advisory_rationale":"<one short sentence, in your own words, saying from the evidence why this revision is worth considering>"} ` +
		"No markdown, no extra fields, no commentary."

	client := http.Client{Timeout: 120 * time.Second}
	var cand *Candidate
	var lastErr error
	for attempt := 1; attempt <= 3; attempt++ { // author-side retries
		reqBody, _ := json.Marshal(map[string]any{
			"model": modelName, "prompt": prompt,
			"format": "json", "stream": false,
			"options": map[string]any{"temperature": 0, "num_predict": 1024},
		})
		resp, err := client.Post(endpoint+"/api/generate", "application/json", bytes.NewReader(reqBody))
		if err != nil {
			lastErr = err
			continue
		}
		var out struct {
			Response string `json:"response"`
		}
		err = json.NewDecoder(resp.Body).Decode(&out)
		resp.Body.Close()
		if err != nil {
			lastErr = err
			continue
		}
		text := strings.TrimSpace(out.Response)
		text = strings.TrimPrefix(text, "```json")
		text = strings.TrimPrefix(text, "```")
		text = strings.TrimSuffix(text, "```")
		var substance struct {
			Family            string `json:"family"`
			Target            string `json:"target"`
			ChangeRelation    string `json:"change_relation"`
			AdvisoryRationale string `json:"advisory_rationale"`
		}
		if lastErr = json.Unmarshal([]byte(strings.TrimSpace(text)), &substance); lastErr == nil {
			assembled, _ := json.Marshal(map[string]any{
				"artifact": "l11-candidate", "schema": 1,
				"family":          substance.Family,
				"target":          substance.Target,
				"change_relation": substance.ChangeRelation,
				"content_sha256":  contentSHA,
				"claimed_baseline": map[string]any{
					"door": "l9-catalog", "name": "investigate-cve", "version": 1,
					"artifact_sha256": baselineSHA,
				},
				"author": modelName, "authored_via": "live-proof:ollama",
				"advisory_rationale": substance.AdvisoryRationale,
			})
			cand, lastErr = ParseCandidate(assembled, "live-model-output")
		}
		if lastErr == nil {
			break
		}
		snippet := text
		if len(snippet) > 300 {
			snippet = snippet[:300] + "…"
		}
		t.Logf("attempt %d: model output refused by the candidate wall: %v (output: %s)", attempt, lastErr, snippet)
		cand = nil
	}
	if cand == nil {
		t.Fatalf("model failed to author an admissible candidate in 3 attempts: %v", lastErr)
	}

	// Authorship is attributable and marked (Class 6 discipline).
	if cand.Author != modelName || cand.AuthoredVia != "live-proof:ollama" {
		t.Fatalf("authorship attribution lost: %s via %s", cand.Author, cand.AuthoredVia)
	}
	if cand.AdvisoryRationale == "" {
		t.Fatal("model rationale missing — advisory block expected")
	}
	t.Logf("model rationale (advisory, establishes nothing): %s", cand.AdvisoryRationale)

	// The candidate is data: store it and run the ordinary compare
	// arc — same walls, no shortcut for model authorship.
	root, err := state.OpenRoot(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	candID, _, err := StoreInstance(root.Store(), cand)
	if err != nil {
		t.Fatal(err)
	}
	in := validInput(t, 0.91, 0.82)
	in.CandidateHash = cand.ContentSHA256
	in.ClaimedBaseline = cand.ClaimedBaseline.ArtifactSHA256
	in.Admission.ArtifactSHA256 = cand.ClaimedBaseline.ArtifactSHA256
	pkg, refusal, err := Compare(in)
	if err != nil || refusal != nil {
		t.Fatalf("compare over model-authored candidate failed: %v %v", refusal, err)
	}
	if _, _, err := StoreInstance(root.Store(), pkg); err != nil {
		t.Fatal(err)
	}
	t.Logf("live arc complete: candidate %s, delta %s", candID, CanonicalDelta(pkg.Delta))
}
