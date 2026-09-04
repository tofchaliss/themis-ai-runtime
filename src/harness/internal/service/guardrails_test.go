package service

import (
	"encoding/json"
	"strings"
	"testing"
)

func factsFromJSON(t *testing.T, raw string) map[string]any {
	t.Helper()
	var m map[string]any
	if err := json.Unmarshal([]byte(raw), &m); err != nil {
		t.Fatal(err)
	}
	return m
}

func TestValidateRecommendation(t *testing.T) {

	valid := `{
		"finding_id": "F-1", "recommended_stance": "open",
		"confidence": "low", "rationale": "insufficient evidence",
		"evidence_basis": []}`

	tests := []struct {
		name    string
		rec     string
		wantErr string
	}{
		{name: "valid recommendation passes", rec: valid},
		{
			name:    "missing confidence fails",
			rec:     `{"finding_id": "F-1", "recommended_stance": "open", "rationale": "r", "evidence_basis": []}`,
			wantErr: `missing required field "confidence"`,
		},
		{
			name:    "null field fails",
			rec:     strings.Replace(valid, `"F-1"`, `null`, 1),
			wantErr: `field "finding_id" must not be null`,
		},
		{
			name:    "bad stance fails",
			rec:     strings.Replace(valid, `"open"`, `"maybe"`, 1),
			wantErr: "invalid stance",
		},
		{
			name:    "bad confidence fails",
			rec:     strings.Replace(valid, `"low"`, `"certain"`, 1),
			wantErr: "invalid confidence",
		},
		{
			name:    "non-string stance fails on kind",
			rec:     strings.Replace(valid, `"open"`, `true`, 1),
			wantErr: `field "recommended_stance" must be a string`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateRecommendation(factsFromJSON(t, tt.rec))
			if tt.wantErr == "" {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("error = %v, want containing %q", err, tt.wantErr)
			}
		})
	}
}

func TestValidateExtractFacts(t *testing.T) {

	tests := []struct {
		name    string
		facts   string
		wantErr string // empty = valid
	}{
		{
			name:  "complete facts pass",
			facts: completeFacts,
		},
		{
			name:  "all-null facts pass",
			facts: nullFacts,
		},
		{
			name:    "missing field fails naming it",
			facts:   `{"cwe": null}`,
			wantErr: `missing required field "affected_component"`,
		},
		{
			name:    "string field with number fails",
			facts:   strings.Replace(nullFacts, `"cve": null`, `"cve": 2024`, 1),
			wantErr: `field "cve" must be a string`,
		},
		{
			name:    "array field with string fails",
			facts:   strings.Replace(nullFacts, `"references": null`, `"references": "see advisory"`, 1),
			wantErr: `field "references" must be a array`,
		},
		{
			name:    "object field with array fails",
			facts:   strings.Replace(nullFacts, `"cvss": null`, `"cvss": [9.8]`, 1),
			wantErr: `field "cvss" must be a object`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateExtractFacts(factsFromJSON(t, tt.facts))
			if tt.wantErr == "" {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				return
			}
			if err == nil {
				t.Fatalf("expected error containing %q, got nil", tt.wantErr)
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("error = %q, want containing %q", err, tt.wantErr)
			}
		})
	}
}
