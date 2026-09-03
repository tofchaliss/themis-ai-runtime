package benchmark

// Definition describes a benchmark.
type Definition struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Category    string `json:"category"`
	Prompt      string `json:"prompt"`
	Expected    string `json:"expected"`
	Description string `json:"description,omitempty"`
	Weight      int    `json:"weight"`
}
