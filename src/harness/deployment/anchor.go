// Package deployment implements the G1 Deployment Anchor (D-G1-1,
// D-G1-1A — openspec/changes/g1-deployment-authority/design.md): the
// Governance-owned, content-addressed declaration of the exact
// artifact set under which a deployment may execute.
//
// The two-step boundary, never collapsed:
//  1. ANCHOR ADMISSION — a caller-supplied anchor path/hash
//     IDENTIFIES the requested deployment; only resolution against
//     the Governance-active anchors registry establishes that it is
//     a governed deployment definition (D-G1-1A: the C-1 lesson — a
//     caller may identify configuration; the governed owner must
//     establish its authority).
//  2. BUNDLE VALIDATION — a submission conforms to that admitted
//     definition (enforced at the L7 seam).
//
// This package READS; there is no write API (the standing wall).
package deployment

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

var (
	ErrAnchor    = errors.New("invalid deployment anchor")
	ErrAdmission = errors.New("deployment anchor admission refused")
)

var (
	shaSyntax  = regexp.MustCompile(`^[0-9a-f]{64}$`)
	nameSyntax = regexp.MustCompile(`^[a-z0-9]+(-[a-z0-9]+)*$`)
)

// Anchor is the closed deployment-definition schema (Q-G1-2). Every
// pin is an exact content hash; nothing is defaulted; unknown fields
// refuse. Pins L7 loads are enforced at Open/SubmitTask; pins for
// other planes (skill catalog, contract/criteria/set registries,
// model allowlist, door table) are the consumption pins those
// planes' consumers verify — admission enforced at consumption, the
// ratified pattern.
type Anchor struct {
	Version    int    `json:"version"`
	Name       string `json:"name"`
	Deployment int    `json:"deployment_version"`

	// Enforced by L7 at Open (dir-tree hashes; policy file hash):
	InstructionSafetyRoot string `json:"instruction_root_safety"`
	InstructionSystemRoot string `json:"instruction_root_system"`
	InstructionThemisRoot string `json:"instruction_root_themis"`
	InstructionPolicy     string `json:"instruction_policy"`

	// Enforced by L7 at SubmitTask (bundle artifact bytes):
	ToolRegistry    string   `json:"tool_registry"`
	WorkflowCeiling string   `json:"workflow_ceiling"`
	ExecCeiling     string   `json:"exec_ceiling"`
	ContextContract string   `json:"context_contract"`
	Workflows       []string `json:"workflows"` // the anchored workflow set
	Models          []string `json:"models"`    // the model allowlist (names)

	// Consumption pins for the other governed planes (verified where
	// those planes are consumed):
	SkillCatalog          string `json:"skill_catalog"`
	ContractRegistry      string `json:"contract_registry"`
	CriteriaRegistry      string `json:"criteria_registry"`
	RegressionSetRegistry string `json:"regression_set_registry"`

	SHA256 string `json:"-"` // of the exact anchor bytes — the deployment identity
	Raw    []byte `json:"-"`
}

const maxAnchorBytes = 1 << 20

// ParseAnchor validates anchor bytes fail-closed.
func ParseAnchor(raw []byte, origin string) (*Anchor, error) {
	if err := checkNoDuplicateKeys(raw); err != nil {
		return nil, fmt.Errorf("%w: %s: %v", ErrAnchor, origin, err)
	}
	dec := json.NewDecoder(strings.NewReader(string(raw)))
	dec.DisallowUnknownFields()
	var a Anchor
	if err := dec.Decode(&a); err != nil {
		return nil, fmt.Errorf("%w: %s: %v", ErrAnchor, origin, err)
	}
	if dec.More() {
		return nil, fmt.Errorf("%w: %s: trailing content", ErrAnchor, origin)
	}
	if a.Version != 1 {
		return nil, fmt.Errorf("%w: %s: version must be 1", ErrAnchor, origin)
	}
	if !nameSyntax.MatchString(a.Name) {
		return nil, fmt.Errorf("%w: %s: bad deployment name %q", ErrAnchor, origin, a.Name)
	}
	if a.Deployment < 1 {
		return nil, fmt.Errorf("%w: %s: deployment_version must be a positive integer", ErrAnchor, origin)
	}
	for field, v := range map[string]string{
		"instruction_root_safety": a.InstructionSafetyRoot,
		"instruction_root_system": a.InstructionSystemRoot,
		"instruction_root_themis": a.InstructionThemisRoot,
		"instruction_policy":      a.InstructionPolicy,
		"tool_registry":           a.ToolRegistry,
		"workflow_ceiling":        a.WorkflowCeiling,
		"exec_ceiling":            a.ExecCeiling,
		"context_contract":        a.ContextContract,
		"skill_catalog":           a.SkillCatalog,
		"contract_registry":       a.ContractRegistry,
		"criteria_registry":       a.CriteriaRegistry,
		"regression_set_registry": a.RegressionSetRegistry,
	} {
		if !shaSyntax.MatchString(v) {
			return nil, fmt.Errorf("%w: %s: %s must be a sha256 hex digest — nothing is defaulted", ErrAnchor, origin, field)
		}
	}
	if len(a.Workflows) == 0 {
		return nil, fmt.Errorf("%w: %s: the anchored workflow set must not be empty", ErrAnchor, origin)
	}
	seenW := map[string]bool{}
	for _, w := range a.Workflows {
		if !shaSyntax.MatchString(w) {
			return nil, fmt.Errorf("%w: %s: workflow pin %q must be a sha256 hex digest", ErrAnchor, origin, w)
		}
		if seenW[w] {
			return nil, fmt.Errorf("%w: %s: duplicate workflow pin", ErrAnchor, origin)
		}
		seenW[w] = true
	}
	if len(a.Models) == 0 {
		return nil, fmt.Errorf("%w: %s: the model allowlist must not be empty — an unlisted model is not a deployment default", ErrAnchor, origin)
	}
	seenM := map[string]bool{}
	for _, m := range a.Models {
		if m == "" || seenM[m] {
			return nil, fmt.Errorf("%w: %s: empty or duplicate model allowlist entry", ErrAnchor, origin)
		}
		seenM[m] = true
	}
	a.SHA256 = hashBytes(raw)
	a.Raw = raw
	return &a, nil
}

// registryEntry is one anchors-registry binding (the same governed
// registry shape as every other plane).
type registryEntry struct {
	Name     string `json:"name"`
	Version  int    `json:"version"`
	Artifact string `json:"artifact_sha256"`
	State    string `json:"state"`
	Steward  string `json:"steward,omitempty"`
}

// AdmitAnchor performs the D-G1-1A two-step for Open:
//
//	resolve exact anchor bytes (unavailable → refusal)
//	verify the operator's expected hash (mismatch → refusal)
//	establish Governance admission against the anchors registry
//	    (unregistered → refusal; withdrawn → refusal)
//	→ ACTIVE anchor, ready to freeze.
//
// The caller-supplied path and expectedSHA identify the REQUEST;
// only the registry resolution establishes governed status.
func AdmitAnchor(anchorPath, expectedSHA, anchorsRegistryPath string) (*Anchor, error) {
	raw, err := readGoverned(anchorPath, maxAnchorBytes)
	if err != nil {
		return nil, fmt.Errorf("%w: anchor bytes unavailable: %v", ErrAdmission, err)
	}
	if !shaSyntax.MatchString(expectedSHA) {
		return nil, fmt.Errorf("%w: the operator's expected anchor hash must be a sha256 hex digest", ErrAdmission)
	}
	if hashBytes(raw) != expectedSHA {
		return nil, fmt.Errorf("%w: anchor bytes do not match the operator's expected hash", ErrAdmission)
	}
	a, err := ParseAnchor(raw, anchorPath)
	if err != nil {
		return nil, err
	}
	regRaw, err := readGoverned(anchorsRegistryPath, maxAnchorBytes)
	if err != nil {
		return nil, fmt.Errorf("%w: anchors registry unavailable: %v", ErrAdmission, err)
	}
	if err := checkNoDuplicateKeys(regRaw); err != nil {
		return nil, fmt.Errorf("%w: anchors registry: %v", ErrAdmission, err)
	}
	var reg struct {
		Version int             `json:"version"`
		Kind    string          `json:"kind"`
		Entries []registryEntry `json:"entries"`
	}
	dec := json.NewDecoder(strings.NewReader(string(regRaw)))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&reg); err != nil || dec.More() {
		return nil, fmt.Errorf("%w: anchors registry unparseable", ErrAdmission)
	}
	if reg.Version < 1 || reg.Kind != "deployment-anchors" {
		return nil, fmt.Errorf("%w: not a deployment-anchors registry", ErrAdmission)
	}
	seen := map[string]bool{}
	var admitted *registryEntry
	for i := range reg.Entries {
		e := &reg.Entries[i]
		if !nameSyntax.MatchString(e.Name) || e.Version < 1 || !shaSyntax.MatchString(e.Artifact) {
			return nil, fmt.Errorf("%w: anchors registry: malformed entry", ErrAdmission)
		}
		if e.State != "active" && e.State != "withdrawn" {
			return nil, fmt.Errorf("%w: anchors registry: unknown state %q", ErrAdmission, e.State)
		}
		key := fmt.Sprintf("%s@%d", e.Name, e.Version)
		if seen[key] {
			return nil, fmt.Errorf("%w: anchors registry: duplicate registration %s", ErrAdmission, key)
		}
		seen[key] = true
		if e.Artifact == a.SHA256 {
			admitted = e
		}
	}
	if admitted == nil {
		return nil, fmt.Errorf("%w: anchor %s is not a Governance-registered deployment anchor — a matching hash is an identifier, never an admission claim", ErrAdmission, a.SHA256[:12])
	}
	if admitted.State == "withdrawn" {
		return nil, fmt.Errorf("%w: anchor %s@%d is withdrawn — a superseded deployment definition cannot open", ErrAdmission, admitted.Name, admitted.Version)
	}
	if admitted.Name != a.Name || admitted.Version != a.Deployment {
		return nil, fmt.Errorf("%w: anchor self-declaration (%s@%d) disagrees with the registration (%s@%d) — two-way identity", ErrAdmission, a.Name, a.Deployment, admitted.Name, admitted.Version)
	}
	return a, nil
}

// HashFile returns the sha256 of a file's exact bytes.
func HashFile(path string) (string, error) {
	b, err := readGoverned(path, 64<<20)
	if err != nil {
		return "", err
	}
	return hashBytes(b), nil
}

// HashDir fingerprints a directory tree deterministically: sha256
// over each regular file's slash-relative path and contents, sorted
// by path — the instruction-root content identity the anchor pins.
func HashDir(root string) (string, error) {
	var files []string
	err := filepath.WalkDir(root, func(p string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.Type().IsRegular() {
			files = append(files, p)
		}
		return nil
	})
	if err != nil {
		return "", err
	}
	sort.Strings(files)
	h := sha256.New()
	for _, f := range files {
		rel, err := filepath.Rel(root, f)
		if err != nil {
			return "", err
		}
		b, err := os.ReadFile(f)
		if err != nil {
			return "", err
		}
		h.Write([]byte(filepath.ToSlash(rel)))
		h.Write([]byte{0})
		h.Write(b)
		h.Write([]byte{0})
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

func hashBytes(b []byte) string {
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}

func readGoverned(path string, maxBytes int64) ([]byte, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil {
		return nil, err
	}
	if !info.Mode().IsRegular() {
		return nil, fmt.Errorf("%s: not a regular file", path)
	}
	if info.Size() > maxBytes {
		return nil, fmt.Errorf("%s: exceeds %d bytes", path, maxBytes)
	}
	raw, err := io.ReadAll(io.LimitReader(f, maxBytes+1))
	if err != nil {
		return nil, err
	}
	if int64(len(raw)) > maxBytes {
		return nil, fmt.Errorf("%s: exceeds %d bytes", path, maxBytes)
	}
	return raw, nil
}

func checkNoDuplicateKeys(raw []byte) error {
	dec := json.NewDecoder(strings.NewReader(string(raw)))
	type frame struct {
		object    bool
		keys      map[string]bool
		nextIsKey bool
	}
	var stack []*frame
	for {
		tok, err := dec.Token()
		if err != nil {
			if err.Error() == "EOF" {
				return nil
			}
			return err
		}
		if len(stack) > 0 {
			top := stack[len(stack)-1]
			if top.object && top.nextIsKey {
				if key, ok := tok.(string); ok {
					if top.keys[key] {
						return fmt.Errorf("duplicate key %q", key)
					}
					top.keys[key] = true
					top.nextIsKey = false
					continue
				}
			}
		}
		switch d := tok.(type) {
		case json.Delim:
			switch d {
			case '{':
				stack = append(stack, &frame{object: true, keys: map[string]bool{}, nextIsKey: true})
			case '[':
				stack = append(stack, &frame{})
			case '}', ']':
				stack = stack[:len(stack)-1]
				if len(stack) > 0 && stack[len(stack)-1].object {
					stack[len(stack)-1].nextIsKey = true
				}
			}
			continue
		}
		if len(stack) > 0 && stack[len(stack)-1].object {
			stack[len(stack)-1].nextIsKey = true
		}
	}
}
