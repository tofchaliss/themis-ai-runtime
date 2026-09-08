package skills

// The Skill manifest: a governed, hash-pinned atomic composition
// (D-L9-1). It pins by SHA-256 one complete set of the artifact kinds
// L1–L7 already execute, plus a typed input schema and a procedure
// artifact reference. It is inert data — nothing here executes, and
// there is deliberately no skill-reference field of any kind, making
// recursion/nesting unrepresentable (D-L9-12).

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

	"github.com/tofchaliss/themis/confine"
)

var (
	ErrManifest = errors.New("invalid skill manifest")
	ErrCatalog  = errors.New("invalid skill catalog")
	ErrResolve  = errors.New("skill resolution refused")
	ErrInputs   = errors.New("skill inputs refused")
)

var (
	shaSyntax  = regexp.MustCompile(`^[0-9a-f]{64}$`)
	nameSyntax = regexp.MustCompile(`^[a-z0-9]+(-[a-z0-9]+)*$`)
)

// MaxSkillNameLen bounds the skill name so "name@version" fits the
// envelope's origin-value cap with room to spare (envelope bound is
// 256; a version is at most a few digits).
const MaxSkillNameLen = 128

// Pin references one composed artifact: a path relative to the
// manifest's directory plus the SHA-256 of its exact reviewed bytes.
type Pin struct {
	Path   string `json:"path"`
	SHA256 string `json:"sha256"`
}

// Manifest is the atomic composition unit. Every field is a fixed pin
// (Class 1, D-L9-7); the composition hash — SHA-256 of the manifest
// bytes — is the identity of the reviewed combination.
type Manifest struct {
	Version int    `json:"version"`
	Name    string `json:"name"`
	Skill   int    `json:"skill_version"`

	Workflow        Pin `json:"workflow"`
	WorkflowCeiling Pin `json:"workflow_ceiling"`
	ContextContract Pin `json:"context_contract"`
	GrantTemplate   Pin `json:"grant_template"`
	SpecTemplate    Pin `json:"spec_template"`
	InputSchema     Pin `json:"input_schema"`
	Procedure       Pin `json:"procedure"`

	CompositionHash string `json:"-"`
	Dir             string `json:"-"` // manifest directory, for pin resolution
}

// LoadManifest reads and validates a skill manifest fail-closed. It
// validates structure and pin syntax; pin BYTES are verified at
// resolution (Resolve), where the composition becomes executable.
func LoadManifest(path string) (*Manifest, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("%w: %s: %v", ErrManifest, path, err)
	}
	dec := json.NewDecoder(strings.NewReader(string(raw)))
	dec.DisallowUnknownFields()
	var m Manifest
	if err := dec.Decode(&m); err != nil {
		return nil, fmt.Errorf("%w: %s: %v", ErrManifest, path, err)
	}
	if dec.More() {
		return nil, fmt.Errorf("%w: %s: trailing content", ErrManifest, path)
	}
	if m.Version < 1 {
		return nil, fmt.Errorf("%w: %s: version required", ErrManifest, path)
	}
	if !nameSyntax.MatchString(m.Name) {
		return nil, fmt.Errorf("%w: %s: bad skill name %q", ErrManifest, path, m.Name)
	}
	// Name length is bounded here so the "name@version" attribution
	// L9 emits can never exceed the envelope's own origin-value bound
	// — L9 must not be able to write an envelope L7 will refuse.
	if len(m.Name) > MaxSkillNameLen {
		return nil, fmt.Errorf("%w: %s: skill name exceeds %d bytes", ErrManifest, path, MaxSkillNameLen)
	}
	if m.Skill < 1 {
		return nil, fmt.Errorf("%w: %s: skill_version must be positive", ErrManifest, path)
	}
	pins := map[string]Pin{
		"workflow": m.Workflow, "workflow_ceiling": m.WorkflowCeiling,
		"context_contract": m.ContextContract, "grant_template": m.GrantTemplate,
		"spec_template": m.SpecTemplate, "input_schema": m.InputSchema,
		"procedure": m.Procedure,
	}
	for name, p := range pins {
		if p.Path == "" {
			return nil, fmt.Errorf("%w: %s: missing %s pin — a skill is one COMPLETE composition, nothing is defaulted", ErrManifest, path, name)
		}
		if filepath.IsAbs(p.Path) || strings.Contains(p.Path, "..") {
			return nil, fmt.Errorf("%w: %s: %s pin path must be relative within the skill bundle", ErrManifest, path, name)
		}
		if !shaSyntax.MatchString(p.SHA256) {
			return nil, fmt.Errorf("%w: %s: %s pin needs a sha256 hex digest", ErrManifest, path, name)
		}
	}
	sum := sha256.Sum256(raw)
	m.CompositionHash = hex.EncodeToString(sum[:])
	m.Dir = filepath.Dir(path)
	return &m, nil
}

// resolvePin reads a pinned artifact and verifies its bytes against
// the pin — the D-L9-11 admission rule: bytes are verified BEFORE the
// composition becomes executable, and the recorded hashes correspond
// to the bytes actually admitted.
func (m *Manifest) resolvePin(name string, p Pin) (string, []byte, error) {
	// Confined resolution through the single canonical predicate
	// (the L1 repository-activation precedent): a pin pointing at a
	// symlink must not read outside the reviewed bundle. Reuse, never
	// a second confinement implementation.
	abs, err := confine.ResolvePath(m.Dir, p.Path)
	if err != nil {
		return "", nil, fmt.Errorf("%w: %s artifact: %v", ErrResolve, name, err)
	}
	info, err := os.Lstat(abs)
	if err != nil || !info.Mode().IsRegular() {
		return "", nil, fmt.Errorf("%w: %s artifact must be a regular file within the skill bundle", ErrResolve, name)
	}
	b, err := os.ReadFile(abs)
	if err != nil {
		return "", nil, fmt.Errorf("%w: %s artifact: %v", ErrResolve, name, err)
	}
	sum := sha256.Sum256(b)
	if hex.EncodeToString(sum[:]) != p.SHA256 {
		return "", nil, fmt.Errorf("%w: %s artifact bytes do not match the reviewed pin — the composition cannot be affirmed", ErrResolve, name)
	}
	return abs, b, nil
}
