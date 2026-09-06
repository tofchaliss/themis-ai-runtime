package context

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

// The workflow context contract is the ceiling (design §3, Q-L2-1):
// Themis defines it as a versioned, reviewed artifact; the plan must
// be a subset of it or composition fails closed. Its hash lands in
// the delivery trace. Closure rules and the expansion ceiling join
// the schema at the L4/L7 era with a version bump — v1 has no
// multi-hop connector and no capability fetch, so the fields would be
// dead configuration.

type SlotRequirement string

const (
	SlotRequired SlotRequirement = "required"
	SlotOptional SlotRequirement = "optional"
)

// Slot declares one expected context class for the workflow.
type Slot struct {
	Name        string           `json:"name"`
	Kind        string           `json:"kind"` // item kind that fills it ("file:*" matches filesystem items)
	Requirement SlotRequirement  `json:"requirement"`
	Classes     []AuthorityClass `json:"classes"`            // permitted authority classes
	Withhold    bool             `json:"withhold,omitempty"` // this contract deliberately excludes the slot
}

// Contract is the loaded, validated workflow context contract.
type Contract struct {
	Version            int         `json:"version"`
	Workflow           string      `json:"workflow"`
	Slots              []Slot      `json:"slots"`
	SensitivityCeiling Sensitivity `json:"sensitivity_ceiling"`

	Hash string `json:"-"` // SHA-256 of the contract file bytes, hex
}

// LoadContract reads and validates the contract artifact. Fail
// closed: unknown fields, duplicate slots, unknown classes or
// requirements, empty slot sets, and trailing bytes are all errors.
func LoadContract(path string) (*Contract, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("%w: %s: %v", ErrContractInvalid, path, err)
	}
	dec := json.NewDecoder(strings.NewReader(string(raw)))
	dec.DisallowUnknownFields()
	var c Contract
	if err := dec.Decode(&c); err != nil {
		return nil, fmt.Errorf("%w: %s: %v", ErrContractInvalid, path, err)
	}
	if dec.More() {
		return nil, fmt.Errorf("%w: %s: trailing content", ErrContractInvalid, path)
	}
	sum := sha256.Sum256(raw)
	c.Hash = hex.EncodeToString(sum[:])
	if err := c.validate(); err != nil {
		return nil, fmt.Errorf("%w: %s: %v", ErrContractInvalid, path, err)
	}
	return &c, nil
}

func (c *Contract) validate() error {
	if c.Version < 1 || c.Workflow == "" || len(c.Slots) == 0 {
		return fmt.Errorf("version, workflow, and at least one slot are required")
	}
	if _, ok := sensitivityRank[c.SensitivityCeiling]; !ok {
		return fmt.Errorf("unknown sensitivity ceiling %q", c.SensitivityCeiling)
	}
	seen := map[string]bool{}
	valid := map[AuthorityClass]bool{
		AuthorityGovernedRecord: true, AuthorityGovernedExternal: true,
		AuthorityDerived: true, AuthorityExternalUntrusted: true,
	}
	for _, s := range c.Slots {
		if s.Name == "" || s.Kind == "" {
			return fmt.Errorf("slot missing name or kind")
		}
		if seen[s.Name] {
			return fmt.Errorf("duplicate slot %q", s.Name)
		}
		seen[s.Name] = true
		if s.Requirement != SlotRequired && s.Requirement != SlotOptional {
			return fmt.Errorf("slot %q has unknown requirement %q", s.Name, s.Requirement)
		}
		if len(s.Classes) == 0 {
			return fmt.Errorf("slot %q permits no authority classes", s.Name)
		}
		for _, cl := range s.Classes {
			if !valid[cl] {
				return fmt.Errorf("slot %q permits unknown class %q", s.Name, cl)
			}
		}
		if s.Withhold && s.Requirement == SlotRequired {
			return fmt.Errorf("slot %q cannot be both required and withheld", s.Name)
		}
	}
	return nil
}

func (c *Contract) slot(name string) *Slot {
	for i := range c.Slots {
		if c.Slots[i].Name == name {
			return &c.Slots[i]
		}
	}
	return nil
}

// kindMatches: a slot with kind "file:*" accepts any filesystem item;
// otherwise exact match. Mechanical, never semantic.
func kindMatches(slotKind, itemKind string) bool {
	if slotKind == "file:*" {
		return strings.HasPrefix(itemKind, "file:")
	}
	return slotKind == itemKind
}

func (s *Slot) classPermitted(c AuthorityClass) bool {
	for _, cl := range s.Classes {
		if cl == c {
			return true
		}
	}
	return false
}
