// Package tools implements Layer 4 of the Themis AI Harness: the
// Tool Interface — the first enforcement layer. It answers exactly
// one question: is this capability authorized for this execution?
// Governing design: openspec/changes/layer-04-tool-interface/design.md
// (§3 grill record). Locked: ExecutionGrant ⊆ WorkflowCeiling ⊆
// Registry; authorization is computed from capability/policy state
// independent of model output and task instructions; authorization
// precedes argument validation; L4 exposes actionable capability
// state, never authorization-policy state; absent from the
// vocabulary = does not exist.
package tools

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	hctx "github.com/tofchaliss/themis/context"
)

var (
	ErrRegistryInvalid = errors.New("invalid tool registry")
	ErrGrantInvalid    = errors.New("invalid task grant")
	ErrDispatch        = errors.New("dispatch table incomplete")
)

// TargetClass names the deterministic target-validation rule a tool
// uses (registry declares the class; the grant binds the instance
// scope — Q-L4-2).
type TargetClass string

const (
	TargetWorkspacePath TargetClass = "workspace-path"
	TargetThemisID      TargetClass = "themis-id"
	TargetNone          TargetClass = "none"
)

// ParamType is the closed argument type vocabulary.
type ParamType string

const (
	ParamString  ParamType = "string"
	ParamInteger ParamType = "integer"
	ParamBoolean ParamType = "boolean"
)

// ParamDef declares one argument of a capability. Schemas carry the
// operation's subject, never its authority disposition — the loader
// rejects governance-attribute parameter names outright (L1 Scenario
// 3 made structural).
type ParamDef struct {
	Name        string    `json:"name"`
	Type        ParamType `json:"type"`
	Required    bool      `json:"required,omitempty"`
	Description string    `json:"description"`
	Target      bool      `json:"target,omitempty"` // this param carries the target value
}

// ToolDef is one registered capability.
type ToolDef struct {
	Name        string      `json:"name"`
	Description string      `json:"description"`
	Params      []ParamDef  `json:"params"`
	Target      TargetClass `json:"target"`
	TimeoutSec  int         `json:"timeout_sec"`
	// Trust is the result trust class fixed at registration
	// (Q-L4-3). "derived" is structurally rejected in v1: no tool may
	// self-declare computational provenance it does not carry.
	Trust hctx.AuthorityClass `json:"trust"`
}

// Registry is the governed capability artifact. Absent = does not
// exist as an executable capability.
type Registry struct {
	Version int       `json:"version"`
	Tools   []ToolDef `json:"tools"`

	Hash string `json:"-"`
}

var nameSyntax = regexp.MustCompile(`^[a-z][a-z0-9_]{1,63}$`)

// forbiddenParams are authority-disposition names no schema may carry
// (defense in depth on top of review — the vocabulary check the L1
// grill demanded).
var forbiddenParams = map[string]bool{
	"requires_human_decision": true, "authority": true, "authority_class": true,
	"approved": true, "approve": true, "approval": true, "authorization": true,
	"authorized": true, "verified": true, "trust": true, "trust_class": true,
	"trusted": true, "sensitivity": true, "grant": true, "grants": true,
	"permission": true, "permissions": true, "human_approved": true,
}

// LoadRegistry: fail-closed artifact posture.
func LoadRegistry(path string) (*Registry, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("%w: %s: %v", ErrRegistryInvalid, path, err)
	}
	dec := json.NewDecoder(strings.NewReader(string(raw)))
	dec.DisallowUnknownFields()
	var r Registry
	if err := dec.Decode(&r); err != nil {
		return nil, fmt.Errorf("%w: %s: %v", ErrRegistryInvalid, path, err)
	}
	if dec.More() {
		return nil, fmt.Errorf("%w: %s: trailing content", ErrRegistryInvalid, path)
	}
	if r.Version < 1 || len(r.Tools) == 0 {
		return nil, fmt.Errorf("%w: %s: version and at least one tool required", ErrRegistryInvalid, path)
	}
	seen := map[string]bool{}
	for i := range r.Tools {
		t := &r.Tools[i]
		if !nameSyntax.MatchString(t.Name) {
			return nil, fmt.Errorf("%w: bad tool name %q", ErrRegistryInvalid, t.Name)
		}
		if seen[t.Name] {
			return nil, fmt.Errorf("%w: duplicate tool %q", ErrRegistryInvalid, t.Name)
		}
		seen[t.Name] = true
		if t.Description == "" || t.TimeoutSec <= 0 {
			return nil, fmt.Errorf("%w: tool %q needs description and positive timeout", ErrRegistryInvalid, t.Name)
		}
		switch t.Target {
		case TargetWorkspacePath, TargetThemisID, TargetNone:
		default:
			return nil, fmt.Errorf("%w: tool %q has unknown target class %q", ErrRegistryInvalid, t.Name, t.Target)
		}
		switch t.Trust {
		case hctx.AuthorityExternalUntrusted, hctx.AuthorityGovernedRecord, hctx.AuthorityGovernedExternal:
		case hctx.AuthorityDerived:
			return nil, fmt.Errorf("%w: tool %q declares derived trust — v1 registers no computational provenance; derived is earned, not asserted", ErrRegistryInvalid, t.Name)
		default:
			return nil, fmt.Errorf("%w: tool %q has unknown trust class %q", ErrRegistryInvalid, t.Name, t.Trust)
		}
		targets := 0
		pnames := map[string]bool{}
		for _, p := range t.Params {
			if !nameSyntax.MatchString(p.Name) || pnames[p.Name] {
				return nil, fmt.Errorf("%w: tool %q has bad or duplicate param %q", ErrRegistryInvalid, t.Name, p.Name)
			}
			pnames[p.Name] = true
			if forbiddenParams[p.Name] {
				return nil, fmt.Errorf("%w: tool %q param %q is an authority-disposition name — schemas carry subject, never disposition", ErrRegistryInvalid, t.Name, p.Name)
			}
			switch p.Type {
			case ParamString, ParamInteger, ParamBoolean:
			default:
				return nil, fmt.Errorf("%w: tool %q param %q has unknown type %q", ErrRegistryInvalid, t.Name, p.Name, p.Type)
			}
			if p.Target {
				targets++
				if p.Type != ParamString {
					return nil, fmt.Errorf("%w: tool %q target param must be a string", ErrRegistryInvalid, t.Name)
				}
			}
		}
		if t.Target != TargetNone && targets != 1 {
			return nil, fmt.Errorf("%w: tool %q needs exactly one target param for class %s", ErrRegistryInvalid, t.Name, t.Target)
		}
		if t.Target == TargetNone && targets != 0 {
			return nil, fmt.Errorf("%w: tool %q declares a target param but no target class", ErrRegistryInvalid, t.Name)
		}
	}
	sum := sha256.Sum256(raw)
	r.Hash = hex.EncodeToString(sum[:])
	return &r, nil
}

func (r *Registry) tool(name string) *ToolDef {
	for i := range r.Tools {
		if r.Tools[i].Name == name {
			return &r.Tools[i]
		}
	}
	return nil
}

// GrantEntry binds one capability instance scope to this execution.
type GrantEntry struct {
	Tool     string `json:"tool"`
	MaxCalls int    `json:"max_calls"`
	// Workspace: confinement root for workspace-path targets.
	Workspace string `json:"workspace,omitempty"`
	// ThemisScope: permitted id prefixes for themis-id targets.
	// Prefix matching is coarse ("FIND-1" authorizes "FIND-123");
	// grant authors use delimiter-terminated prefixes ("FIND-1:") for
	// exact families (F8, documented convention).
	ThemisScope []string `json:"themis_scope,omitempty"`
}

// Grant is the execution-scoped allowlist: ExecutionGrant ⊆
// WorkflowCeiling ⊆ Registry (Q-L4-1; the ceiling⊆ check is L7-era —
// v1 validates grant ⊆ registry at use).
type Grant struct {
	Version       int          `json:"version"`
	TaskID        string       `json:"task_id"`
	Entries       []GrantEntry `json:"entries"`
	TotalMaxCalls int          `json:"total_max_calls"`

	Hash string `json:"-"`
}

// LoadGrant: fail-closed artifact posture.
func LoadGrant(path string) (*Grant, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("%w: %s: %v", ErrGrantInvalid, path, err)
	}
	dec := json.NewDecoder(strings.NewReader(string(raw)))
	dec.DisallowUnknownFields()
	var g Grant
	if err := dec.Decode(&g); err != nil {
		return nil, fmt.Errorf("%w: %s: %v", ErrGrantInvalid, path, err)
	}
	if dec.More() {
		return nil, fmt.Errorf("%w: %s: trailing content", ErrGrantInvalid, path)
	}
	if g.Version < 1 || g.TaskID == "" || len(g.Entries) == 0 || g.TotalMaxCalls <= 0 {
		return nil, fmt.Errorf("%w: %s: version, task, entries, and positive total cap required", ErrGrantInvalid, path)
	}
	seen := map[string]bool{}
	for _, e := range g.Entries {
		if e.Tool == "" || seen[e.Tool] {
			return nil, fmt.Errorf("%w: empty or duplicate grant entry %q", ErrGrantInvalid, e.Tool)
		}
		seen[e.Tool] = true
		if e.MaxCalls <= 0 {
			return nil, fmt.Errorf("%w: grant %q needs a positive call cap", ErrGrantInvalid, e.Tool)
		}
		// Absolute workspace bindings only: a relative binding makes
		// confinement depend on process cwd — non-reproducible across
		// invocations (security review F6).
		if e.Workspace != "" && !filepath.IsAbs(e.Workspace) {
			return nil, fmt.Errorf("%w: grant %q workspace must be absolute", ErrGrantInvalid, e.Tool)
		}
	}
	sum := sha256.Sum256(raw)
	g.Hash = hex.EncodeToString(sum[:])
	return &g, nil
}

func (g *Grant) entry(tool string) *GrantEntry {
	for i := range g.Entries {
		if g.Entries[i].Tool == tool {
			return &g.Entries[i]
		}
	}
	return nil
}
