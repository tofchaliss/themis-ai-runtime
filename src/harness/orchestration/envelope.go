package orchestration

// The task envelope: the ONLY task input (D-L7-1). The governed half
// references governed artifacts by absolute path (relative paths
// would make identity depend on process cwd); the payload half is
// and stays external-untrusted. Nothing is defaulted — a missing
// reference is a typed refusal, never a helpful fill (Q-L7-10).

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type Envelope struct {
	Version int    `json:"version"`
	TaskID  string `json:"task_id"`
	RetryOf string `json:"retry_of,omitempty"`
	Model   string `json:"model"`
	// TurnTimeoutSec bounds one model call — governed input, not an
	// L7-authored constant (architecture review 1.2).
	TurnTimeoutSec int `json:"turn_timeout_sec"`
	// Payload: the untrusted task half. External-untrusted forever.
	Payload string `json:"payload"`
	// Governed artifact references (absolute paths).
	WorkflowPath        string `json:"workflow_path"`
	WorkflowCeilingPath string `json:"workflow_ceiling_path"`
	RegistryPath        string `json:"registry_path"`
	GrantPath           string `json:"grant_path"`
	ExecCeilingPath     string `json:"exec_ceiling_path"`
	SpecPath            string `json:"spec_path"`
	// ContextContractPath: the L2 workflow context contract the loop
	// composes against (architecture review 2a, owner decision
	// 2026-09-07: the payload travels through the full L2 pipeline).
	ContextContractPath string `json:"context_contract_path"`

	// SkillProcedurePath/SHA: OPTIONAL activated skill-instruction
	// artifact (L9). When present the loop activates it as the L1
	// ScopeSkill source with byte verification against the pin; when
	// absent nothing changes. L7 never resolves a skill — it verifies
	// bytes against a hash the envelope states (D-L9-11).
	SkillProcedurePath   string `json:"skill_procedure_path,omitempty"`
	SkillProcedureSHA256 string `json:"skill_procedure_sha256,omitempty"`

	// Origin: OPAQUE governed attribution (D-L9-13). L7 records it
	// verbatim into the task's governed hashes and exercises ZERO
	// semantics on it: it never selects a workflow, derives a grant,
	// authorizes anything, or influences δ. Skill-blindness means the
	// loop cannot tell a skill-instantiated envelope from a
	// hand-assembled one except by the presence of these strings.
	Origin map[string]string `json:"origin,omitempty"`

	Hash string `json:"-"`
}

func LoadEnvelope(path string) (*Envelope, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("%w: %s: %v", ErrEnvelope, path, err)
	}
	dec := json.NewDecoder(strings.NewReader(string(raw)))
	dec.DisallowUnknownFields()
	var e Envelope
	if err := dec.Decode(&e); err != nil {
		return nil, fmt.Errorf("%w: %s: %v", ErrEnvelope, path, err)
	}
	if dec.More() {
		return nil, fmt.Errorf("%w: %s: trailing content", ErrEnvelope, path)
	}
	if e.Version < 1 || e.TaskID == "" {
		return nil, fmt.Errorf("%w: version and task_id required", ErrEnvelope)
	}
	if e.TurnTimeoutSec <= 0 {
		return nil, fmt.Errorf("%w: missing turn_timeout_sec — nothing is defaulted", ErrEnvelope)
	}
	// No defaulting: every reference is required and named in the
	// refusal, so an adapter that "forgot" one is caught here, typed.
	refs := map[string]string{
		"model":                 e.Model,
		"payload":               e.Payload,
		"workflow_path":         e.WorkflowPath,
		"workflow_ceiling_path": e.WorkflowCeilingPath,
		"registry_path":         e.RegistryPath,
		"grant_path":            e.GrantPath,
		"exec_ceiling_path":     e.ExecCeilingPath,
		"spec_path":             e.SpecPath,
		"context_contract_path": e.ContextContractPath,
	}
	for name, v := range refs {
		if v == "" {
			return nil, fmt.Errorf("%w: missing %s — nothing is defaulted", ErrEnvelope, name)
		}
	}
	for name, p := range refs {
		if strings.HasSuffix(name, "_path") && !filepath.IsAbs(p) {
			return nil, fmt.Errorf("%w: %s must be absolute", ErrEnvelope, name)
		}
	}
	// The payload is the one unbounded caller string: cap it (every
	// tool evidence path already has its cap; security review LOW).
	if len(e.Payload) > 64<<10 {
		return nil, fmt.Errorf("%w: payload exceeds the 64KiB envelope cap", ErrEnvelope)
	}
	// The optional skill-procedure pair travels together or not at
	// all, and its path is absolute like every other reference. The
	// pin is mandatory when the path is present: unverifiable
	// instruction bytes are never delivered.
	if (e.SkillProcedurePath == "") != (e.SkillProcedureSHA256 == "") {
		return nil, fmt.Errorf("%w: skill_procedure_path and skill_procedure_sha256 are set together or not at all", ErrEnvelope)
	}
	if e.SkillProcedurePath != "" && !filepath.IsAbs(e.SkillProcedurePath) {
		return nil, fmt.Errorf("%w: skill_procedure_path must be absolute", ErrEnvelope)
	}
	// Attribution is bounded strings — opaque, but not unbounded.
	for k, v := range e.Origin {
		if len(k) > 64 || len(v) > 256 {
			return nil, fmt.Errorf("%w: origin attribution field %q exceeds its bound", ErrEnvelope, k)
		}
	}
	sum := sha256.Sum256(raw)
	e.Hash = hex.EncodeToString(sum[:])
	return &e, nil
}
