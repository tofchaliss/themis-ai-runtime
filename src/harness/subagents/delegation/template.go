package delegation

// The template manifest loader (D-L8-5/6, C-L8-4, C-L8-14, C-L8-15).
// A template pins, by hash: one ordinary L2 context contract and at
// most one skill-scope instruction file; and states, as literals: the
// EIS carry-over filter, which contract slot the brief fills and its
// byte bound, and the delegated output bound. The hash of the manifest
// bytes is the template's artifact identity and transitively covers
// every pinned byte (C-L8-15 E). Nothing here composes, executes, or
// authorizes: the loader establishes admissibility, identity, and
// accountability — never truth, never authority (C-L8-14 H).

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/tofchaliss/themis/confine"
	hctx "github.com/tofchaliss/themis/context"
	"github.com/tofchaliss/themis/internal/strictjson"
)

// Pin is a {path, sha256} reference to a governed byte artifact,
// relative to the manifest directory.
type Pin struct {
	Path   string `json:"path"`
	SHA256 string `json:"sha256"`
}

// Brief declares which slot of the pinned contract the caller's
// bounded untrusted brief fills, and its byte bound. The slot's
// requiredness and class live in the contract (C-L8-14 D).
type Brief struct {
	Slot     string `json:"slot"`
	MaxBytes int    `json:"max_bytes"`
}

// Template is the loaded, pin-verified manifest.
type Template struct {
	Version         int      `json:"version"`
	Name            string   `json:"name"`
	TemplateVersion int      `json:"template_version"`
	ContextContract Pin      `json:"context_contract"`
	Instruction     *Pin     `json:"instruction,omitempty"`
	EISCarryScopes  []string `json:"eis_carry_scopes,omitempty"`
	Brief           Brief    `json:"brief"`
	MaxOutputBytes  int      `json:"max_output_bytes"`

	Hash string `json:"-"` // of the exact manifest bytes — the artifact identity
	Raw  []byte `json:"-"`
	Dir  string `json:"-"`

	// The pinned artifacts as loaded and verified at this load, so a
	// consumer composes from the bytes the pins were checked against
	// and never re-reads unverified (C-L8-9).
	Contract       *hctx.Contract `json:"-"`
	ContractRaw    []byte         `json:"-"`
	InstructionRaw []byte         `json:"-"` // nil when no instruction is pinned
}

// carryScopes is the filter's entire domain (C-L8-4): the three
// mandatory roots are carried unconditionally by L7 code and are not
// filterable; `task` never carries. Naming any of those here is a
// load refusal, not a no-op.
var carryScopes = map[string]bool{"repository": true, "directory": true, "skill": true}

// nonCarryScopes are the L1 scope names a filter must not name, each
// with the rule it would contradict.
var nonCarryScopes = map[string]string{
	"harness-safety": "a mandatory root — carried unconditionally, outside the filter's domain",
	"harness-system": "a mandatory root — carried unconditionally, outside the filter's domain",
	"themis-domain":  "a mandatory root — carried unconditionally, outside the filter's domain",
	"task":           "task-scope instructions never carry into a delegation",
}

// disjointKeys are artifact families a delegation template may not
// pin (D-L8-5/6): naming one is refused by rule, before the closed
// schema's generic unknown-field refusal, so the refusal names the
// boundary being crossed.
var disjointKeys = map[string]bool{
	"workflow": true, "workflow_ceiling": true, "ceiling": true,
	"grant": true, "grant_template": true, "spec": true, "spec_template": true,
	"input_schema": true, "template": true, "templates": true,
	"tools": true, "model": true, "scope": true, "procedure": true,
}

// LoadTemplate reads and verifies one manifest fail-closed, resolving
// its pins under the manifest directory.
func LoadTemplate(path string) (*Template, error) {
	raw, err := readGoverned(path, maxTemplateBytes, ErrTemplate)
	if err != nil {
		return nil, err
	}
	t, err := parseManifest(raw, path)
	if err != nil {
		return nil, err
	}
	t.Dir = filepath.Dir(path)
	// Pins: confined, regular, bytes verified against the sha.
	_, contractRaw, err := t.resolvePin("context_contract", t.ContextContract)
	if err != nil {
		return nil, err
	}
	var instRaw []byte
	if t.Instruction != nil {
		_, instRaw, err = t.resolvePin("instruction", *t.Instruction)
		if err != nil {
			return nil, err
		}
	}
	if err := t.bind(path, contractRaw, instRaw); err != nil {
		return nil, err
	}
	return t, nil
}

// ParseTemplate verifies a manifest and its members from BYTES — the
// reconstruction path (D-L8-17, C-L8-9): the registry and the files
// are never the historical source of truth; the stored objects are.
// Identical rules to LoadTemplate minus the filesystem confinement,
// which does not apply to content-addressed bytes.
func ParseTemplate(manifestRaw, contractRaw, instructionRaw []byte) (*Template, error) {
	t, err := parseManifest(manifestRaw, "<bytes>")
	if err != nil {
		return nil, err
	}
	if !shaSyntax.MatchString(t.ContextContract.SHA256) || hashBytes(contractRaw) != t.ContextContract.SHA256 {
		return nil, fmt.Errorf("%w: context_contract bytes do not match the pinned hash", ErrTemplate)
	}
	if t.Instruction != nil {
		if !shaSyntax.MatchString(t.Instruction.SHA256) || hashBytes(instructionRaw) != t.Instruction.SHA256 {
			return nil, fmt.Errorf("%w: instruction bytes do not match the pinned hash", ErrTemplate)
		}
	} else {
		instructionRaw = nil
	}
	if err := t.bind("<bytes>", contractRaw, instructionRaw); err != nil {
		return nil, err
	}
	return t, nil
}

func parseManifest(raw []byte, path string) (*Template, error) {
	if err := strictjson.Check(raw); err != nil {
		return nil, fmt.Errorf("%w: %s: %v", ErrTemplate, path, err)
	}
	// Disjointness first, by name.
	var keys map[string]json.RawMessage
	if err := json.Unmarshal(raw, &keys); err != nil {
		return nil, fmt.Errorf("%w: %s: %v", ErrTemplate, path, err)
	}
	for k := range keys {
		if disjointKeys[k] {
			return nil, fmt.Errorf("%w: %s: a delegation template may not pin %q — it is not a Skill and defines no workflow, authority, tools, or nested template (D-L8-5/6)", ErrTemplate, path, k)
		}
	}
	dec := jsonDecoder(raw)
	var t Template
	if err := dec.Decode(&t); err != nil {
		return nil, fmt.Errorf("%w: %s: %v", ErrTemplate, path, err)
	}
	if dec.More() {
		return nil, fmt.Errorf("%w: %s: trailing content", ErrTemplate, path)
	}
	if t.Version < 1 {
		return nil, fmt.Errorf("%w: %s: version required", ErrTemplate, path)
	}
	if !nameSyntax.MatchString(t.Name) || t.TemplateVersion < 1 {
		return nil, fmt.Errorf("%w: %s: name and template_version must declare an exact identity", ErrTemplate, path)
	}
	seen := map[string]bool{}
	for _, s := range t.EISCarryScopes {
		if why, bad := nonCarryScopes[s]; bad {
			return nil, fmt.Errorf("%w: %s: eis_carry_scopes names %q: %s (C-L8-4)", ErrTemplate, path, s, why)
		}
		if !carryScopes[s] {
			return nil, fmt.Errorf("%w: %s: eis_carry_scopes names unknown scope %q — the filter's domain is {repository, directory, skill}", ErrTemplate, path, s)
		}
		if seen[s] {
			return nil, fmt.Errorf("%w: %s: duplicate eis_carry_scopes entry %q", ErrTemplate, path, s)
		}
		seen[s] = true
	}
	if t.Brief.Slot == "" {
		return nil, fmt.Errorf("%w: %s: brief.slot must name the contract slot the brief fills", ErrTemplate, path)
	}
	if t.Brief.MaxBytes < 1 || t.Brief.MaxBytes > hctx.MaxItemBytes {
		return nil, fmt.Errorf("%w: %s: brief.max_bytes must be within [1, %d]", ErrTemplate, path, hctx.MaxItemBytes)
	}
	if t.MaxOutputBytes < 1 || t.MaxOutputBytes > hctx.MaxItemBytes {
		return nil, fmt.Errorf("%w: %s: max_output_bytes must be within [1, %d] — every referenceable object is bounded by what L2 accepts per item (C-L8-7)", ErrTemplate, path, hctx.MaxItemBytes)
	}
	t.Hash = hashBytes(raw)
	t.Raw = raw
	return &t, nil
}

// bind attaches verified member bytes and runs the contract
// cross-checks (C-L8-14 D).
func (t *Template) bind(path string, contractRaw, instRaw []byte) error {
	c, err := hctx.ParseContract(contractRaw)
	if err != nil {
		return fmt.Errorf("%w: %s: context_contract: %v", ErrTemplate, path, err)
	}
	t.Contract, t.ContractRaw = c, contractRaw
	if t.Instruction != nil {
		if len(instRaw) == 0 {
			return fmt.Errorf("%w: %s: instruction pin resolves to an empty file", ErrTemplate, path)
		}
		t.InstructionRaw = instRaw
	}
	// Cross-checks against the pinned contract (C-L8-14 D): the brief
	// slot exists, is delivered, and permits exactly the untrusted
	// class — the template never restates contract semantics.
	var slot *hctx.Slot
	for i := range c.Slots {
		if c.Slots[i].Name == t.Brief.Slot {
			slot = &c.Slots[i]
		}
	}
	if slot == nil {
		return fmt.Errorf("%w: %s: brief.slot %q is not a slot of the pinned contract", ErrTemplate, path, t.Brief.Slot)
	}
	if slot.Withhold {
		return fmt.Errorf("%w: %s: brief.slot %q is withheld by the pinned contract — a withheld slot cannot carry the brief", ErrTemplate, path, t.Brief.Slot)
	}
	if len(slot.Classes) != 1 || slot.Classes[0] != hctx.AuthorityExternalUntrusted {
		return fmt.Errorf("%w: %s: brief.slot %q must permit exactly [external-untrusted] — the brief is untrusted content, never a governed class (D-L8-4)", ErrTemplate, path, t.Brief.Slot)
	}
	return nil
}

// resolvePin reads a pinned artifact confined to the manifest
// directory and verifies its bytes against the pin — the same
// admission rule as skill members (D-L9-11): bytes verified BEFORE the
// composition is usable, and the recorded hash is the hash of what
// was admitted.
func (t *Template) resolvePin(name string, p Pin) (string, []byte, error) {
	if !shaSyntax.MatchString(p.SHA256) {
		return "", nil, fmt.Errorf("%w: %s pin must carry a sha256 hex digest", ErrTemplate, name)
	}
	full, err := confine.ResolvePath(t.Dir, p.Path)
	if err != nil {
		return "", nil, fmt.Errorf("%w: %s pin: %v", ErrTemplate, name, err)
	}
	if info, serr := os.Lstat(full); serr != nil || !info.Mode().IsRegular() {
		return "", nil, fmt.Errorf("%w: %s pin must be a regular file within the template directory", ErrTemplate, name)
	}
	raw, err := readGoverned(full, maxTemplateBytes, ErrTemplate)
	if err != nil {
		return "", nil, err
	}
	if hashBytes(raw) != p.SHA256 {
		return "", nil, fmt.Errorf("%w: %s bytes do not match the pinned hash — the reviewed composition can no longer be affirmed", ErrTemplate, name)
	}
	return full, raw, nil
}
