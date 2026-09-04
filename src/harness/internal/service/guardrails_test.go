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
