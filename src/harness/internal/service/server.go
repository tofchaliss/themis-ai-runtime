// Package service is the Themis AI runtime: an HTTP service exposing
// evidence-based vulnerability-analysis operations backed by LLMs.
//
// It shares its model layer (internal/llm) with the benchmark harness,
// routes each operation to the model that scores best on the matching
// benchmark category, and enforces deterministic guardrails on both
// input (prompt-injection flagging) and output (JSON contracts).
package service

import (
	"context"
	"embed"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"text/template"
	"time"

	"github.com/tofchaliss/themis/internal/llm"
)

//go:embed prompts/*.md
var promptFS embed.FS

var prompts = template.Must(
	template.ParseFS(promptFS, "prompts/*.md"),
)

// Categories the operations route on. They match the benchmark
// definitions' category field.
const (
	categoryExtraction = "Extraction"
	categoryReasoning  = "Reasoning"
)

// Server is the Themis runtime service.
type Server struct {
	Registry *llm.Registry
	Router   *Router

	// DefaultModel is used when routing has no data for a category and
	// the request does not name a model.
	DefaultModel string

	// Timeout bounds one model invocation.
	Timeout time.Duration
}

// Handler returns the service's HTTP handler.
func (s *Server) Handler() http.Handler {

	mux := http.NewServeMux()

	mux.HandleFunc("GET /healthz", s.handleHealth)
	mux.HandleFunc("POST /v1/extract", s.handleExtract)
	mux.HandleFunc("POST /v1/recommend-position", s.handleRecommend)

	return mux
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"status": "ok",
		"routes": s.Router.Table(),
	})
}

type extractRequest struct {
	Evidence string `json:"evidence"`
	Model    string `json:"model"`
}

func (s *Server) handleExtract(w http.ResponseWriter, r *http.Request) {

	var req extractRequest

	if err := decodeRequest(w, r, &req); err != nil {
		return
	}

	if strings.TrimSpace(req.Evidence) == "" {
		writeError(w, http.StatusBadRequest, "evidence is required")
		return
	}

	answer, meta, err := s.invoke(
		r.Context(),
		req.Model,
		categoryExtraction,
		"extract.md",
		map[string]any{"Evidence": req.Evidence},
	)
	if err != nil {
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}

	facts, err := llm.ExtractJSON(answer)
	if err != nil {
		writeError(w, http.StatusBadGateway, fmt.Sprintf(
			"model did not return valid JSON: %v", err,
		))
		return
	}

	if err := ValidateExtractFacts(facts); err != nil {
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}

	meta["injection_suspected"] = SuspectInjection(req.Evidence)

	writeJSON(w, http.StatusOK, map[string]any{
		"facts": facts,
		"meta":  meta,
	})
}

type recommendRequest struct {
	FindingID string `json:"finding_id"`
	Evidence  string `json:"evidence"`
	Model     string `json:"model"`
}

func (s *Server) handleRecommend(w http.ResponseWriter, r *http.Request) {

	var req recommendRequest

	if err := decodeRequest(w, r, &req); err != nil {
		return
	}

	if strings.TrimSpace(req.Evidence) == "" {
		writeError(w, http.StatusBadRequest, "evidence is required")
		return
	}
	if strings.TrimSpace(req.FindingID) == "" {
		writeError(w, http.StatusBadRequest, "finding_id is required")
		return
	}

	answer, meta, err := s.invoke(
		r.Context(),
		req.Model,
		categoryReasoning,
		"recommend.md",
		map[string]any{
			"FindingID": req.FindingID,
			"Evidence":  req.Evidence,
		},
	)
	if err != nil {
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}

	recommendation, err := llm.ExtractJSON(answer)
	if err != nil {
		writeError(w, http.StatusBadGateway, fmt.Sprintf(
			"model did not return valid JSON: %v", err,
		))
		return
	}

	if err := RequireFields(
		recommendation,
		"finding_id",
		"recommended_stance",
		"confidence",
		"rationale",
		"evidence_basis",
	); err != nil {
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}

	stance, _ := recommendation["recommended_stance"].(string)
	if err := CheckStance(stance); err != nil {
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}

	// The service is advisory only; no model output may claim
	// otherwise.
	recommendation["requires_human_decision"] = true

	meta["injection_suspected"] = SuspectInjection(req.Evidence)

	writeJSON(w, http.StatusOK, map[string]any{
		"recommendation": recommendation,
		"meta":           meta,
	})
}

// invoke renders the operation prompt, resolves the model (explicit
// request override, then benchmark-driven routing, then the default),
// and runs it.
func (s *Server) invoke(
	ctx context.Context,
	requestedModel string,
	category string,
	promptName string,
	data map[string]any,
) (string, map[string]any, error) {

	modelName := requestedModel
	routed := false

	if modelName == "" {
		if best, ok := s.Router.Route(category); ok {
			modelName, routed = best, true
		} else {
			modelName = s.DefaultModel
		}
	}

	if modelName == "" {
		return "", nil, fmt.Errorf(
			"no model available for category %s: no benchmark data, "+
				"no default model, and none requested",
			category,
		)
	}

	var prompt strings.Builder

	if err := prompts.ExecuteTemplate(&prompt, promptName, data); err != nil {
		return "", nil, fmt.Errorf("render prompt: %w", err)
	}

	rt, resolvedModel, options, err := s.Registry.Resolve(modelName)
	if err != nil {
		return "", nil, err
	}

	if s.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, s.Timeout)
		defer cancel()
	}

	start := time.Now()

	resp, err := rt.Run(ctx, llm.Request{
		Model:   resolvedModel,
		Prompt:  prompt.String(),
		Options: options,
	})
	if err != nil {
		return "", nil, fmt.Errorf("model %s: %w", modelName, err)
	}

	meta := map[string]any{
		"model":      modelName,
		"runtime":    resp.Runtime,
		"routed":     routed,
		"category":   category,
		"latency_ms": time.Since(start).Milliseconds(),
		"tokens":     resp.Metrics.TotalTokens,
	}

	return resp.Answer, meta, nil
}

func decodeRequest(w http.ResponseWriter, r *http.Request, v any) error {

	dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20))
	dec.DisallowUnknownFields()

	if err := dec.Decode(v); err != nil {
		writeError(w, http.StatusBadRequest, fmt.Sprintf(
			"invalid request body: %v", err,
		))
		return err
	}

	return nil
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	if err := json.NewEncoder(w).Encode(v); err != nil {
		log.Printf("write response: %v", err)
	}
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}
