// Package verification is Layer 10's verification mechanism: the
// governed Verification Contract artifact and its registry (D-L10-2).
// L10 owns exactly one domain identity — Verification Contract identity
// (D-L10-10) — and evaluates registered contracts; it never authors
// them, never authorizes, never executes, and never records history
// (D-L10-1: not a Governance engine, not a record plane, not an
// orchestrator).
package verification

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"
)

var (
	ErrContract = errors.New("invalid verification contract")
	ErrRegistry = errors.New("invalid contract registry")
	ErrResolve  = errors.New("contract resolution refused")
)

var (
	shaSyntax  = regexp.MustCompile(`^[0-9a-f]{64}$`)
	nameSyntax = regexp.MustCompile(`^[a-z0-9]+(-[a-z0-9]+)*$`)
	// capSyntax mirrors the L4 registry's tool-name syntax — the
	// binding references a capability under L4's naming rules, not
	// L10's contract-name rules.
	capSyntax     = regexp.MustCompile(`^[a-z][a-z0-9_]{1,63}$`)
	slotSyntax    = regexp.MustCompile(`^[a-z0-9]+(_[a-z0-9]+)*$`)
	versionSyntax = regexp.MustCompile(`^[1-9][0-9]*$`)
)

// Outcome is the closed five-value L10 vocabulary (D-L10-4). PASS,
// FAIL, and INCONCLUSIVE are contract outcomes; UNAVAILABLE and
// INVALID are machinery-reserved and structurally unreachable from any
// contract mapping — a contract cannot launder machinery failure into
// a decision, or a decision into failure.
type Outcome string

const (
	OutcomePass         Outcome = "PASS"
	OutcomeFail         Outcome = "FAIL"
	OutcomeInconclusive Outcome = "INCONCLUSIVE"
	OutcomeUnavailable  Outcome = "UNAVAILABLE" // machinery-reserved
	OutcomeInvalid      Outcome = "INVALID"     // machinery-reserved
)

// contractMappable is the subset a contract's result mapping may
// target (D-L10-4). Everything else is refused at load.
var contractMappable = map[Outcome]bool{
	OutcomePass:         true,
	OutcomeFail:         true,
	OutcomeInconclusive: true,
}

// EvidenceSlot declares one evidence input the contract requires: the
// contract specifies evidence KINDS; the evaluation instance resolves
// them to concrete governed objects (D-L10-2 — task state stays
// outside contract identity). Kind is a closed vocabulary; TaskBound
// is mandatory true in v1 (D-L10-7 task-binding: an evaluation may
// consume only objects recorded within its own task).
type EvidenceSlot struct {
	Name      string `json:"name"`
	Kind      string `json:"kind"` // closed: "artifact"
	Required  bool   `json:"required"`
	TaskBound bool   `json:"task_bound"`
}

// evidenceKinds is the closed v1 kind vocabulary.
var evidenceKinds = map[string]bool{"artifact": true}

// VerifierBinding pins the exact L4-registered capability that
// executes under this contract. Capability identity in L4 is the tool
// name within a registry version; the registry hash pins that version
// (D-L10-10 #2). The binding references the capability — it cannot
// create, redefine, or authorize one (D-L10-2).
type VerifierBinding struct {
	Capability     string `json:"capability"`
	RegistrySHA256 string `json:"registry_sha256"`
}

// Contract is the governed, versioned, hash-pinned declarative
// specification of D-L10-2. It contains no executable logic, no
// expressions, no interpreters, and no contract-by-value verifier
// definition. Config is carried by value inside the canonical
// representation, so the contract hash covers it and no external
// config store is needed in v1 (D-L10-10 #3: no hash without
// recoverable bytes — trivially satisfied by inclusion).
//
// The D-L10-2 "failure-semantics mapping" is realized as the
// INCONCLUSIVE entries of ResultMapping plus the machinery-reserved
// statuses of D-L10-8; a separate free-form failure field would open
// semantics the design closed.
type Contract struct {
	Version  int    `json:"version"`
	Name     string `json:"name"`
	Contract int    `json:"contract_version"`

	Verifier VerifierBinding `json:"verifier"`
	Evidence []EvidenceSlot  `json:"evidence"`

	Config json.RawMessage `json:"config"`

	// ResultMapping maps canonical verifier-domain results to contract
	// outcomes. Keys are verifier-domain strings; values may target
	// only PASS, FAIL, or INCONCLUSIVE.
	ResultMapping map[string]Outcome `json:"result_mapping"`

	// Provenance lists the required computational-provenance elements.
	// Closed vocabulary; v1 requires exactly the full set — provenance
	// completeness is not contract-relaxable (D-L10-10).
	Provenance []string `json:"provenance"`

	SHA256 string `json:"-"` // of the exact contract bytes
	// Raw carries the exact verified bytes so downstream durable
	// storage never re-reads the file after verification — the
	// R-L9-2 no-reopen window, structurally.
	Raw []byte `json:"-"`
}

// provenanceRequired is the closed, complete v1 provenance element
// set. A contract must declare exactly these; declaring fewer would
// relax D-L10-10, declaring others is unrepresentable.
var provenanceRequired = []string{
	"execution_record",
	"raw_output",
	"canonical_result",
}

// maxConfigBytes bounds the inline configuration so a contract stays a
// compact governed artifact rather than an artifact transport.
const maxConfigBytes = 64 * 1024

// MaxContractNameLen bounds the contract name so "name@version" stays
// within downstream identity bounds (the L9 precedent; contract
// identity flows into D-L10-13 gate tokens).
const MaxContractNameLen = 128

// maxContractBytes bounds a contract file; maxRegistryBytes bounds the
// registry file (L-3: no unbounded read of attacker-swappable bytes).
const (
	maxContractBytes = 1 << 20 // 1 MiB
	maxRegistryBytes = 4 << 20 // 4 MiB
)

// LoadContract reads and validates a Verification Contract
// fail-closed: closed schema, unknown fields refused, trailing content
// refused, nothing defaulted.
func LoadContract(path string) (*Contract, error) {
	raw, err := readGoverned(path, maxContractBytes, ErrContract)
	if err != nil {
		return nil, err
	}
	return ParseContract(raw, path)
}

// ParseContract validates contract bytes fail-closed. The SHA-256 of
// the exact bytes is the contract identity component (D-L10-2).
func ParseContract(raw []byte, origin string) (*Contract, error) {
	if err := checkNoDuplicateKeys(raw); err != nil {
		return nil, fmt.Errorf("%w: %s: %v", ErrContract, origin, err)
	}
	dec := jsonDecoder(raw)
	var c Contract
	if err := dec.Decode(&c); err != nil {
		return nil, fmt.Errorf("%w: %s: %v", ErrContract, origin, err)
	}
	if dec.More() {
		return nil, fmt.Errorf("%w: %s: trailing content", ErrContract, origin)
	}
	if c.Version != 1 {
		return nil, fmt.Errorf("%w: %s: version must be 1", ErrContract, origin)
	}
	if !nameSyntax.MatchString(c.Name) || len(c.Name) > MaxContractNameLen {
		return nil, fmt.Errorf("%w: %s: bad contract name %q", ErrContract, origin, c.Name)
	}
	if c.Contract < 1 {
		return nil, fmt.Errorf("%w: %s: contract_version must be a positive integer", ErrContract, origin)
	}
	if !capSyntax.MatchString(c.Verifier.Capability) {
		return nil, fmt.Errorf("%w: %s: verifier.capability must be a valid L4 tool name", ErrContract, origin)
	}
	if !shaSyntax.MatchString(c.Verifier.RegistrySHA256) {
		return nil, fmt.Errorf("%w: %s: verifier.registry_sha256 must be a sha256 hex digest", ErrContract, origin)
	}
	if len(c.Evidence) == 0 {
		return nil, fmt.Errorf("%w: %s: at least one evidence slot required", ErrContract, origin)
	}
	slots := map[string]bool{}
	for _, s := range c.Evidence {
		if !slotSyntax.MatchString(s.Name) {
			return nil, fmt.Errorf("%w: %s: bad evidence slot name %q", ErrContract, origin, s.Name)
		}
		if slots[s.Name] {
			return nil, fmt.Errorf("%w: %s: duplicate evidence slot %q", ErrContract, origin, s.Name)
		}
		slots[s.Name] = true
		if !evidenceKinds[s.Kind] {
			return nil, fmt.Errorf("%w: %s: slot %q: unknown evidence kind %q", ErrContract, origin, s.Name, s.Kind)
		}
		if !s.TaskBound {
			// Task binding is not contract-relaxable in v1: evidence
			// outside the evaluating task's record is inadmissible.
			return nil, fmt.Errorf("%w: %s: slot %q: task_bound must be true in v1", ErrContract, origin, s.Name)
		}
		if !s.Required {
			// Optional evidence is a hidden default under a closed
			// schema; deferred until a real requirement (L-5).
			return nil, fmt.Errorf("%w: %s: slot %q: required must be true in v1", ErrContract, origin, s.Name)
		}
	}
	if len(c.Config) > maxConfigBytes {
		return nil, fmt.Errorf("%w: %s: config exceeds %d bytes", ErrContract, origin, maxConfigBytes)
	}
	// Config must be a JSON object: null/scalar/array configs are
	// hidden defaults (L-1). "{}" declares none explicitly.
	trimmed := strings.TrimSpace(string(c.Config))
	if !strings.HasPrefix(trimmed, "{") || !json.Valid(c.Config) {
		return nil, fmt.Errorf("%w: %s: config must be a JSON object (use {} for none)", ErrContract, origin)
	}
	if len(c.ResultMapping) == 0 {
		return nil, fmt.Errorf("%w: %s: result_mapping required", ErrContract, origin)
	}
	for domain, out := range c.ResultMapping {
		if domain == "" {
			return nil, fmt.Errorf("%w: %s: empty result-domain member", ErrContract, origin)
		}
		if !contractMappable[out] {
			// UNAVAILABLE/INVALID (and anything unknown) are
			// unreachable from a contract mapping (D-L10-4).
			return nil, fmt.Errorf("%w: %s: result %q maps to %q — contracts may map only to PASS, FAIL, or INCONCLUSIVE", ErrContract, origin, domain, out)
		}
	}
	if err := checkProvenance(c.Provenance); err != nil {
		return nil, fmt.Errorf("%w: %s: %v", ErrContract, origin, err)
	}
	c.SHA256 = hashBytes(raw)
	c.Raw = raw
	return &c, nil
}

func checkProvenance(got []string) error {
	if len(got) != len(provenanceRequired) {
		return fmt.Errorf("provenance must declare exactly %v — completeness is not contract-relaxable", provenanceRequired)
	}
	seen := map[string]bool{}
	for _, p := range got {
		seen[p] = true
	}
	for _, req := range provenanceRequired {
		if !seen[req] {
			return fmt.Errorf("provenance missing required element %q", req)
		}
	}
	return nil
}

func hashBytes(b []byte) string {
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}

// jsonDecoder returns a strict decoder over the exact bytes: unknown
// fields refused; callers must also check More() for trailing content.
func jsonDecoder(raw []byte) *json.Decoder {
	dec := json.NewDecoder(strings.NewReader(string(raw)))
	dec.DisallowUnknownFields()
	return dec
}

// checkNoDuplicateKeys refuses JSON whose objects repeat a key at any
// depth. encoding/json is last-value-wins on duplicates, which would
// let reviewed text and decoded semantics disagree — fatal for the
// result-mapping wall (security review M-1).
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

// readGoverned reads a governed artifact with a size bound: the bytes
// are attacker-swappable before any hash check, so unbounded reads are
// a memory-DoS surface (security review L-3).
func readGoverned(path string, maxBytes int64, class error) ([]byte, error) {
	// One open, stat on the HANDLE, bounded read on the same handle —
	// no stat-then-reopen window a symlink swap could widen (close
	// security review L-2).
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("%w: %s: %v", class, path, err)
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil {
		return nil, fmt.Errorf("%w: %s: %v", class, path, err)
	}
	if !info.Mode().IsRegular() {
		return nil, fmt.Errorf("%w: %s: not a regular file", class, path)
	}
	if info.Size() > maxBytes {
		return nil, fmt.Errorf("%w: %s: exceeds %d bytes", class, path, maxBytes)
	}
	raw, err := io.ReadAll(io.LimitReader(f, maxBytes+1))
	if err != nil {
		return nil, fmt.Errorf("%w: %s: %v", class, path, err)
	}
	if int64(len(raw)) > maxBytes {
		return nil, fmt.Errorf("%w: %s: exceeds %d bytes", class, path, maxBytes)
	}
	return raw, nil
}
