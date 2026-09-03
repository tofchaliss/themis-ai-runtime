package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// DefaultTimeout bounds a single generation request. Local models can be
// slow to load and generate, so this is intentionally generous.
const DefaultTimeout = 10 * time.Minute

type Ollama struct {
	Endpoint string
	Client   *http.Client
}

func NewOllama(endpoint string) *Ollama {
	return &Ollama{
		Endpoint: endpoint,
		Client: &http.Client{
			Timeout: DefaultTimeout,
		},
	}
}

func (o *Ollama) Name() string {
	return "ollama"
}

func (o *Ollama) Run(
	ctx context.Context,
	req Request,
) (*Response, error) {

	body := map[string]any{
		"model":  req.Model,
		"prompt": req.Prompt,
		"stream": false,
		"options": map[string]any{
			"temperature": req.Options.Temperature,
			"seed":        req.Options.Seed,
		},
	}

	b, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}

	httpReq, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		o.Endpoint+"/api/generate",
		bytes.NewReader(b),
	)
	if err != nil {
		return nil, err
	}

	httpReq.Header.Set("Content-Type", "application/json")

	client := o.Client
	if client == nil {
		client = &http.Client{Timeout: DefaultTimeout}
	}

	resp, err := client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("ollama request: %w", err)
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read ollama response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf(
			"ollama returned %s: %s",
			resp.Status,
			truncate(raw, 512),
		)
	}

	var r OllamaResponse

	if err := json.Unmarshal(raw, &r); err != nil {
		return nil, fmt.Errorf("parse ollama response: %w", err)
	}

	if r.Error != "" {
		return nil, fmt.Errorf("ollama error: %s", r.Error)
	}

	return &Response{
		Runtime: o.Name(),
		Model:   req.Model,
		Answer:  r.Response,
		// Preserve the complete raw response.
		Raw:     raw,
		Metrics: r.Metrics(),
	}, nil
}

func truncate(b []byte, n int) string {
	if len(b) <= n {
		return string(b)
	}
	return string(b[:n]) + "..."
}
