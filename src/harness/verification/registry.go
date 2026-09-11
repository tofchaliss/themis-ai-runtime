package verification

// The L10 Contract Registry: Governance's append-only registration
// record for Verification Contracts (D-L10-2, registry fork (a)).
// Registration is an external governed act — this package only READS;
// there is deliberately no write API anywhere in it, so "machinery
// must not self-register" holds structurally (the L9 catalog wall,
// reapplied). An unregistered contract-shaped artifact is data and
// cannot be evaluated: no closest-match, no latest, no name-only
// resolution, no contract-by-value.

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/tofchaliss/themis/confine"
)

type EntryState string

const (
	StateActive    EntryState = "active"
	StateWithdrawn EntryState = "withdrawn"
)

// Entry is one immutable name@version → contract-hash binding.
type Entry struct {
	Name         string     `json:"name"`
	Version      int        `json:"version"`
	Contract     string     `json:"contract_sha256"`
	ContractPath string     `json:"contract_path"` // relative to the registry file
	State        EntryState `json:"state"`
	Steward      string     `json:"steward,omitempty"` // accountability metadata, never authority
}

type Registry struct {
	Version int     `json:"version"`
	Entries []Entry `json:"entries"`

	Hash string `json:"-"`
	dir  string
}

// EligibilityChecker reports whether a capability is registered
// deterministic-verifier-eligible in the L4 registry version the
// contract pins (D-L10-3: eligibility is a registration property,
// enforced by refusal at bind time). Implementations live with the L4
// wiring; this package fails closed without one.
type EligibilityChecker interface {
	VerifierEligible(capability, registrySHA256 string) (bool, error)
}

// LoadRegistry reads the governed contract registry fail-closed. One
// load is one atomic registry state (the D-L9-10 TOCTOU rule):
// resolution works from a single Load result.
func LoadRegistry(path string) (*Registry, error) {
	raw, err := readGoverned(path, maxRegistryBytes, ErrRegistry)
	if err != nil {
		return nil, err
	}
	if err := checkNoDuplicateKeys(raw); err != nil {
		return nil, fmt.Errorf("%w: %s: %v", ErrRegistry, path, err)
	}
	dec := jsonDecoder(raw)
	var r Registry
	if err := dec.Decode(&r); err != nil {
		return nil, fmt.Errorf("%w: %s: %v", ErrRegistry, path, err)
	}
	if dec.More() {
		return nil, fmt.Errorf("%w: %s: trailing content", ErrRegistry, path)
	}
	if r.Version < 1 {
		return nil, fmt.Errorf("%w: %s: version required", ErrRegistry, path)
	}
	seen := map[string]bool{}
	for _, e := range r.Entries {
		if !nameSyntax.MatchString(e.Name) || e.Version < 1 {
			return nil, fmt.Errorf("%w: bad entry identity %q@%d", ErrRegistry, e.Name, e.Version)
		}
		key := fmt.Sprintf("%s@%d", e.Name, e.Version)
		if seen[key] {
			return nil, fmt.Errorf("%w: duplicate registration %s — bindings are immutable", ErrRegistry, key)
		}
		seen[key] = true
		if !shaSyntax.MatchString(e.Contract) {
			return nil, fmt.Errorf("%w: %s: contract_sha256 must be a sha256 hex digest", ErrRegistry, key)
		}
		if e.ContractPath == "" || filepath.IsAbs(e.ContractPath) || strings.Contains(e.ContractPath, "..") {
			return nil, fmt.Errorf("%w: %s: contract_path must be relative within the registry root", ErrRegistry, key)
		}
		if e.State != StateActive && e.State != StateWithdrawn {
			return nil, fmt.Errorf("%w: %s: unknown state %q", ErrRegistry, key, e.State)
		}
	}
	r.Hash = hashBytes(raw)
	r.dir = filepath.Dir(path)
	return &r, nil
}

// Root returns the registry's directory — the governed registry root,
// for disjointness checks at consuming call sites.
func (r *Registry) Root() string { return r.dir }

// CheckAppendOnly verifies this registry state against a previously
// observed one: every prior binding must still be present with the
// SAME contract hash, and state may only advance active→withdrawn.
// Deletion, rebinding, and un-withdrawal are refusals — out-of-band
// registry mutation is detected rather than trusted (Register T #7).
func (r *Registry) CheckAppendOnly(prior *Registry) error {
	if prior == nil {
		return nil
	}
	current := map[string]Entry{}
	for _, e := range r.Entries {
		current[fmt.Sprintf("%s@%d", e.Name, e.Version)] = e
	}
	for _, p := range prior.Entries {
		key := fmt.Sprintf("%s@%d", p.Name, p.Version)
		cur, ok := current[key]
		if !ok {
			return fmt.Errorf("%w: %s disappeared — the registry is append-only, registrations remain permanently interpretable", ErrRegistry, key)
		}
		if cur.Contract != p.Contract {
			return fmt.Errorf("%w: %s rebound to a different contract — bindings are immutable", ErrRegistry, key)
		}
		if p.State == StateWithdrawn && cur.State != StateWithdrawn {
			return fmt.Errorf("%w: %s un-withdrawn — state advances active→withdrawn only", ErrRegistry, key)
		}
	}
	return nil
}

// ParseRef parses an exact "name@version" contract reference.
// Floating references — name-only, @latest, ranges — are refused
// (D-L10-2: no fallback resolution of any kind).
func ParseRef(ref string) (string, int, error) {
	at := strings.LastIndex(ref, "@")
	if at <= 0 || at == len(ref)-1 {
		return "", 0, fmt.Errorf("%w: %q: an exact name@version is required — nothing is defaulted", ErrResolve, ref)
	}
	name, vs := ref[:at], ref[at+1:]
	if !nameSyntax.MatchString(name) {
		return "", 0, fmt.Errorf("%w: bad contract name %q", ErrResolve, name)
	}
	// Canonical form only: "+1" and "01" would give one identity many
	// textual spellings — record/audit ambiguity (security review L-2).
	if !versionSyntax.MatchString(vs) {
		return "", 0, fmt.Errorf("%w: %q: version must be a canonical positive integer (no latest, no ranges, no leading zeros)", ErrResolve, ref)
	}
	v, err := strconv.Atoi(vs)
	if err != nil || v < 1 {
		return "", 0, fmt.Errorf("%w: %q: version must be an exact positive integer (no latest, no ranges)", ErrResolve, ref)
	}
	return name, v, nil
}

// Resolve returns the entry and loaded contract for an exact reference
// from THIS loaded registry state. Withdrawn entries refuse typed. The
// contract bytes are hash-verified against the registered binding and
// the contract's self-declared identity must agree with the
// registration (two-way identity, the L9 pattern). The verifier
// binding's eligibility is checked fail-closed: without a checker, or
// on any checker error, or on ineligibility, resolution is refused —
// binding a non-eligible capability is structurally unreachable
// (D-L10-3).
func (r *Registry) Resolve(ref string, elig EligibilityChecker) (*Entry, *Contract, error) {
	name, version, err := ParseRef(ref)
	if err != nil {
		return nil, nil, err
	}
	var entry *Entry
	for i := range r.Entries {
		if r.Entries[i].Name == name && r.Entries[i].Version == version {
			entry = &r.Entries[i]
			break
		}
	}
	if entry == nil {
		return nil, nil, fmt.Errorf("%w: %s@%d is not registered — unregistered contract-shaped artifacts are data", ErrResolve, name, version)
	}
	if entry.State == StateWithdrawn {
		return nil, nil, fmt.Errorf("%w: %s@%d is withdrawn — new evaluation is refused; historical records remain interpretable", ErrResolve, name, version)
	}
	contractPath, err := confine.ResolvePath(r.dir, entry.ContractPath)
	if err != nil {
		return nil, nil, fmt.Errorf("%w: %s@%d contract: %v", ErrResolve, name, version, err)
	}
	if info, serr := os.Lstat(contractPath); serr != nil || !info.Mode().IsRegular() {
		return nil, nil, fmt.Errorf("%w: %s@%d contract must be a regular file within the registry root", ErrResolve, name, version)
	}
	c, err := LoadContract(contractPath)
	if err != nil {
		return nil, nil, err
	}
	if c.SHA256 != entry.Contract {
		return nil, nil, fmt.Errorf("%w: %s@%d: contract bytes do not match the registered hash", ErrResolve, name, version)
	}
	if c.Name != name || c.Contract != version {
		return nil, nil, fmt.Errorf("%w: %s@%d: contract self-declaration (%s@%d) disagrees with the registration — two-way identity check", ErrResolve, name, version, c.Name, c.Contract)
	}
	if elig == nil {
		return nil, nil, fmt.Errorf("%w: %s@%d: no eligibility checker wired — verifier eligibility fails closed", ErrResolve, name, version)
	}
	ok, err := elig.VerifierEligible(c.Verifier.Capability, c.Verifier.RegistrySHA256)
	if err != nil {
		return nil, nil, fmt.Errorf("%w: %s@%d: eligibility check failed: %v", ErrResolve, name, version, err)
	}
	if !ok {
		return nil, nil, fmt.Errorf("%w: %s@%d: capability %q is not registered deterministic-verifier-eligible", ErrResolve, name, version, c.Verifier.Capability)
	}
	return entry, c, nil
}
