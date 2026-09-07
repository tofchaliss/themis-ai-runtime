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
	sum := sha256.Sum256(raw)
	e.Hash = hex.EncodeToString(sum[:])
	return &e, nil
}
