package llm

import (
	"encoding/json"
	"fmt"
	"strings"
)

// ExtractJSON parses a model answer as a JSON object, tolerating
// surrounding prose or Markdown code fences by falling back to the
// outermost {...} block.
func ExtractJSON(answer string) (map[string]any, error) {

	var obj map[string]any

	if err := json.Unmarshal([]byte(answer), &obj); err == nil {
		return obj, nil
	}

	start := strings.Index(answer, "{")
	end := strings.LastIndex(answer, "}")

	if start == -1 || end <= start {
		return nil, fmt.Errorf("no JSON object found in answer")
	}

	if err := json.Unmarshal(
		[]byte(answer[start:end+1]),
		&obj,
	); err != nil {
		return nil, fmt.Errorf("answer contains malformed JSON: %w", err)
	}

	return obj, nil
}
