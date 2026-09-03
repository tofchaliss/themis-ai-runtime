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

// OpenAI talks to any OpenAI-compatible chat-completions endpoint:
// hosted OpenAI, vLLM, llama.cpp server, LM Studio, OpenRouter, etc.
type OpenAI struct {
	// Endpoint is the API base, e.g. "https://api.openai.com/v1".
	Endpoint string

	// APIKey is sent as a Bearer token when non-empty.
	APIKey string

	Client *http.Client
}

func NewOpenAI(endpoint, apiKey string) *OpenAI {
	return &OpenAI{
		Endpoint: endpoint,
		APIKey:   apiKey,
		Client: &http.Client{
			Timeout: DefaultTimeout,
		},
	}
}

func (o *OpenAI) Name() string {
	return "openai"
}

type openAIResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`

	Usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage"`

	Error *struct {
		Message string `json:"message"`
	} `json:"error"`
}

func (o *OpenAI) Run(
	ctx context.Context,
	req Request,
) (*Response, error) {

	body := map[string]any{
		"model": req.Model,
		"messages": []map[string]string{
			{"role": "user", "content": req.Prompt},
		},
		"temperature": req.Options.Temperature,
		"seed":        req.Options.Seed,
		"stream":      false,
	}

	b, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}

	httpReq, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		o.Endpoint+"/chat/completions",
		bytes.NewReader(b),
	)
	if err != nil {
		return nil, err
	}

	httpReq.Header.Set("Content-Type", "application/json")
	if o.APIKey != "" {
		httpReq.Header.Set("Authorization", "Bearer "+o.APIKey)
	}

	client := o.Client
	if client == nil {
		client = &http.Client{Timeout: DefaultTimeout}
	}

	start := time.Now()

	resp, err := client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("openai request: %w", err)
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read openai response: %w", err)
	}

	elapsed := time.Since(start)

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf(
			"openai endpoint returned %s: %s",
			resp.Status,
			truncate(raw, 512),
		)
	}

	var r openAIResponse

	if err := json.Unmarshal(raw, &r); err != nil {
		return nil, fmt.Errorf("parse openai response: %w", err)
	}

	if r.Error != nil {
		return nil, fmt.Errorf("openai error: %s", r.Error.Message)
	}

	if len(r.Choices) == 0 {
		return nil, fmt.Errorf("openai response contains no choices")
	}

	// The API does not report server-side timing, so generation time is
	// the observed wall-clock time of the request.
	elapsedMS := float64(elapsed.Milliseconds())

	metrics := Metrics{
		PromptTokens:     r.Usage.PromptTokens,
		CompletionTokens: r.Usage.CompletionTokens,
		TotalTokens:      r.Usage.TotalTokens,

		GenerationTimeMS: elapsedMS,
		TotalTimeMS:      elapsedMS,
	}

	if elapsedMS > 0 {
		metrics.TokensPerSecond =
			float64(metrics.CompletionTokens) / (elapsedMS / 1000)
	}

	return &Response{
		Runtime: o.Name(),
		Model:   req.Model,
		Answer:  r.Choices[0].Message.Content,
		Raw:     raw,
		Metrics: metrics,
	}, nil
}
