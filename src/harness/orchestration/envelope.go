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
	"regexp"
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

	// Composition: OPTIONAL integrity commitment over the artifacts a
	// submitted composition names (D-L9-11a / C2). It carries
	// IDENTITIES ONLY — never bytes — so envelope size is fixed
	// regardless of artifact size. L7 uses it for exactly one thing:
	// verifying that each artifact it materializes is the one the
	// submission committed to. It is NOT a skill identity: nothing
	// resolves a skill from it, and it opens no admission path.
	//
	// What this establishes: the executed artifacts are cryptographically
	// consistent with the submitted composition. What it does NOT
	// establish: that the submitted composition is the one governance
	// registered — that remains a catalog/post-hoc property in v1, and
	// no field here may be described as authenticated or trusted.
	Composition *CompositionCommitment `json:"composition,omitempty"`

	// Origin: OPAQUE governed attribution (D-L9-13). L7 records it
	// verbatim into the task's governed hashes and exercises ZERO
	// semantics on it: it never selects a workflow, derives a grant,
	// authorizes anything, or influences δ. Skill-blindness means the
	// loop cannot tell a skill-instantiated envelope from a
	// hand-assembled one except by the presence of these strings.
	Origin map[string]string `json:"origin,omitempty"`

	Hash string `json:"-"`
}

// sha256Syntax: a committed identity is a full hex digest, nothing else.
var sha256Syntax = regexp.MustCompile(`^[0-9a-f]{64}$`)

// CompositionCommitment names the expected SHA-256 of each artifact a
// submitted composition covers. Every field is a plain identity; the
// struct deliberately has no name, version, or reference field, so it
// cannot be resolved back to a skill and cannot become a second
// identity system (owner constraint, M5).
type CompositionCommitment struct {
	// The Skill-fixed composition (D-L9-11c): every artifact whose
	// identity must remain stable during execution. Deployment-governed
	// inputs (registry, exec ceiling) are a different identity domain
	// and are deliberately NOT sealed here.
	Workflow        string `json:"workflow_sha256"`
	WorkflowCeiling string `json:"workflow_ceiling_sha256"`
	ContextContract string `json:"context_contract_sha256"`
	GrantTemplate   string `json:"grant_template_sha256"`
	SpecTemplate    string `json:"spec_template_sha256"`
	InputSchema     string `json:"input_schema_sha256"`
	Procedure       string `json:"procedure_sha256,omitempty"`

	// Grant/Spec: identities of the EFFECTIVE artifacts this envelope
	// references. They are instantiated from the sealed templates, so
	// they are covered by the seal transitively while still being
	// checkable against what L7 materializes.
	Grant string `json:"grant_sha256"`
	Spec  string `json:"spec_sha256"`

	// Seal: SHA-256 over the canonical serialization of every field
	// above (D-L9-11b). Without it the commitment is N independently
	// choosable claims — an adversary swaps a path and its identity
	// together — which is the option-A shape D-L9-11a rejected. The
	// seal makes the identities one indivisible unit.
	//
	// It seals the SUBMITTED composition. It does NOT authenticate a
	// Skill: that the submitted composition is the governance-
	// registered one is a separate fact, established only from catalog
	// evidence (D-L9-11a Claim 2).
	Seal string `json:"composition_sha256"`
}

// canonical serializes the sealed fields in a fixed order, length-
// framed so no value can forge a field boundary. The seal is a hash
// over exactly these bytes.
func (c *CompositionCommitment) canonical() []byte {
	var b strings.Builder
	b.WriteString("themis-skill-composition-v1")
	for _, f := range c.sealedFields() {
		fmt.Fprintf(&b, "|%d:%s=%s", len(f.sha), f.label, f.sha)
	}
	return []byte(b.String())
}

type sealedField struct{ label, sha string }

// sealedFields is the ordered, closed field list the seal covers.
func (c *CompositionCommitment) sealedFields() []sealedField {
	return []sealedField{
		{"workflow", c.Workflow},
		{"workflow_ceiling", c.WorkflowCeiling},
		{"context_contract", c.ContextContract},
		{"grant_template", c.GrantTemplate},
		{"spec_template", c.SpecTemplate},
		{"input_schema", c.InputSchema},
		{"procedure", c.Procedure},
		{"grant", c.Grant},
		{"spec", c.Spec},
	}
}

// sealed computes the seal over the current field values.
func (c *CompositionCommitment) sealed() string {
	sum := sha256.Sum256(c.canonical())
	return hex.EncodeToString(sum[:])
}

// checkSeal verifies the commitment is internally intact: the seal
// must be the hash of the identities it accompanies. A mismatch means
// the identity set was altered after it was sealed.
func (c *CompositionCommitment) checkSeal() error {
	if !sha256Syntax.MatchString(c.Seal) {
		return fmt.Errorf("%w: composition needs its sealing hash — unsealed identities are independently choosable claims, not a composition", ErrEnvelope)
	}
	if got := c.sealed(); got != c.Seal {
		return fmt.Errorf("%w: the composition's identities do not match its seal (sealed %s, identities hash to %s)", ErrInvariant, c.Seal, got)
	}
	return nil
}

// verify checks one materialized artifact against the identity the
// sealed composition commits to. A mismatch means the executed
// artifact is not the artifact the composition named — an integrity
// violation, not a governance judgment.
func (c *CompositionCommitment) verify(label, want, got string) error {
	if want == "" {
		return fmt.Errorf("%w: composition commits no identity for %s — a named artifact without its identity is not a reference", ErrEnvelope, label)
	}
	if want != got {
		return fmt.Errorf("%w: %s does not match the identity this composition commits to (want %s, materialized %s)", ErrInvariant, label, want, got)
	}
	return nil
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
	// A composition commitment must be complete for every artifact L7
	// materializes: a path without its committed identity is not a
	// reference (D-L9-11a). Absent entirely is fine — a hand-assembled
	// envelope carries no composition and executes exactly as before.
	// D-L9-11d: Skill attribution REQUIRES a commitment. Without this
	// an envelope could carry governance-looking attribution while
	// evading verification entirely, making C2 submitter-elective.
	// The reverse stays legal: no attribution and no commitment is an
	// ordinary hand-assembled envelope.
	// The gate must cover EVERY skill-attributing key, not one literal:
	// L9 emits nine, and gating only "skill" let a fabricated
	// attribution reach the durable record with zero verification
	// (final security review H-1). The namespace is L9-authored, so it
	// is treated as closed: "skill" or any "skill_" prefix attributes a
	// skill and therefore requires the commitment it refers to.
	for k := range e.Origin {
		if k != "skill" && !strings.HasPrefix(k, "skill_") {
			continue
		}
		if e.Composition == nil {
			return nil, fmt.Errorf("%w: origin key %q attributes a skill, so the envelope must carry the composition commitment that attribution refers to", ErrInvariant, k)
		}
	}
	if c := e.Composition; c != nil {
		// Every sealed identity is a full digest; the procedure's is
		// present exactly when a procedure is delivered.
		for _, f := range c.sealedFields() {
			if f.label == "procedure" {
				continue
			}
			if !sha256Syntax.MatchString(f.sha) {
				return nil, fmt.Errorf("%w: composition needs a sha256 identity for %s", ErrEnvelope, f.label)
			}
		}
		// Symmetric pairing: committing to an artifact L7 will never
		// materialize would leave the record asserting a composition
		// member that did not participate.
		if (c.Procedure == "") != (e.SkillProcedurePath == "") {
			return nil, fmt.Errorf("%w: the composition's procedure identity and skill_procedure_path are present together or not at all", ErrEnvelope)
		}
		if c.Procedure != "" && !sha256Syntax.MatchString(c.Procedure) {
			return nil, fmt.Errorf("%w: composition needs a sha256 identity for procedure", ErrEnvelope)
		}
		// The seal makes the identities one unit (D-L9-11b): check it
		// before any of them is used.
		if err := c.checkSeal(); err != nil {
			return nil, err
		}
	}
	sum := sha256.Sum256(raw)
	e.Hash = hex.EncodeToString(sum[:])
	return &e, nil
}
