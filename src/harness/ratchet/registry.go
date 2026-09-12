package ratchet

// The L11 registries: Governance's append-only registration records
// for Comparison Criteria and Regression Sets (D-L11-6/9). L11 owns
// the FORMAT and this fail-closed read-only loader; Governance owns
// every registration act and the semantic review at registration.
// There is deliberately no write API anywhere in this package —
// "machinery must not self-register" holds structurally (the L9/L10
// wall, third application). Unregistered criterion- or set-shaped
// artifacts are data and measure nothing.

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

// Entry is one immutable name@version → artifact-hash binding.
type Entry struct {
	Name         string     `json:"name"`
	Version      int        `json:"version"`
	Artifact     string     `json:"artifact_sha256"`
	ArtifactPath string     `json:"artifact_path"` // relative to the registry file
	State        EntryState `json:"state"`
	Steward      string     `json:"steward,omitempty"` // accountability metadata, never authority
}

// Registry is one loaded registry state (criteria or sets — same
// governed format, distinct files, distinct artifact kinds). One
// load is one atomic state (the D-L9-10 TOCTOU rule).
type Registry struct {
	Version int     `json:"version"`
	Kind    string  `json:"kind"` // "criteria" | "regression-sets"
	Entries []Entry `json:"entries"`

	Hash string `json:"-"`
	dir  string
}

var registryKinds = map[string]bool{"criteria": true, "regression-sets": true}

// LoadRegistry reads a governed L11 registry fail-closed.
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
	if !registryKinds[r.Kind] {
		return nil, fmt.Errorf("%w: %s: unknown registry kind %q", ErrRegistry, path, r.Kind)
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
		if !shaSyntax.MatchString(e.Artifact) {
			return nil, fmt.Errorf("%w: %s: artifact_sha256 must be a sha256 hex digest", ErrRegistry, key)
		}
		if e.ArtifactPath == "" || filepath.IsAbs(e.ArtifactPath) || strings.Contains(e.ArtifactPath, "..") {
			return nil, fmt.Errorf("%w: %s: artifact_path must be relative within the registry root", ErrRegistry, key)
		}
		if e.State != StateActive && e.State != StateWithdrawn {
			return nil, fmt.Errorf("%w: %s: unknown state %q", ErrRegistry, key, e.State)
		}
	}
	r.Hash = hashBytes(raw)
	r.dir = filepath.Dir(path)
	return &r, nil
}

// Root returns the registry's directory for disjointness checks.
func (r *Registry) Root() string { return r.dir }

// CheckAppendOnly verifies this registry state against a previously
// observed one: bindings immutable, entries never disappear, state
// advances active→withdrawn only (Register T discipline).
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
		if cur.Artifact != p.Artifact {
			return fmt.Errorf("%w: %s rebound to a different artifact — bindings are immutable", ErrRegistry, key)
		}
		if p.State == StateWithdrawn && cur.State != StateWithdrawn {
			return fmt.Errorf("%w: %s un-withdrawn — state advances active→withdrawn only", ErrRegistry, key)
		}
	}
	return nil
}

// ParseRef parses an exact "name@version" reference. Floating
// references — name-only, @latest, ranges, non-canonical versions —
// are refused: L11 never resolves "which version applies" (D-L11-9
// Amendment 2 generalized).
func ParseRef(ref string) (string, int, error) {
	at := strings.LastIndex(ref, "@")
	if at <= 0 || at == len(ref)-1 {
		return "", 0, fmt.Errorf("%w: %q: an exact name@version is required — L11 selects no versions", ErrResolve, ref)
	}
	name, vs := ref[:at], ref[at+1:]
	if !nameSyntax.MatchString(name) {
		return "", 0, fmt.Errorf("%w: bad name %q", ErrResolve, name)
	}
	if !versionSyntax.MatchString(vs) {
		return "", 0, fmt.Errorf("%w: %q: version must be a canonical positive integer (no latest, no ranges, no leading zeros)", ErrResolve, ref)
	}
	v, err := strconv.Atoi(vs)
	if err != nil || v < 1 {
		return "", 0, fmt.Errorf("%w: %q: version must be an exact positive integer", ErrResolve, ref)
	}
	return name, v, nil
}

func (r *Registry) find(name string, version int) *Entry {
	for i := range r.Entries {
		if r.Entries[i].Name == name && r.Entries[i].Version == version {
			return &r.Entries[i]
		}
	}
	return nil
}

// resolveEntry performs the shared registry-side checks and returns
// the verified artifact bytes: exact entry, active state, confined
// regular file, hash match.
func (r *Registry) resolveEntry(ref string, maxBytes int64) (*Entry, []byte, error) {
	name, version, err := ParseRef(ref)
	if err != nil {
		return nil, nil, err
	}
	entry := r.find(name, version)
	if entry == nil {
		return nil, nil, fmt.Errorf("%w: %s@%d is not registered — unregistered artifacts are data and measure nothing", ErrResolve, name, version)
	}
	if entry.State == StateWithdrawn {
		return nil, nil, fmt.Errorf("%w: %s@%d is withdrawn — new comparison is refused; historical packages remain interpretable", ErrResolve, name, version)
	}
	p, err := confine.ResolvePath(r.dir, entry.ArtifactPath)
	if err != nil {
		return nil, nil, fmt.Errorf("%w: %s@%d artifact: %v", ErrResolve, name, version, err)
	}
	if info, serr := os.Lstat(p); serr != nil || !info.Mode().IsRegular() {
		return nil, nil, fmt.Errorf("%w: %s@%d artifact must be a regular file within the registry root", ErrResolve, name, version)
	}
	raw, err := readGoverned(p, maxBytes, ErrResolve)
	if err != nil {
		return nil, nil, err
	}
	if hashBytes(raw) != entry.Artifact {
		return nil, nil, fmt.Errorf("%w: %s@%d: artifact bytes do not match the registered hash", ErrResolve, name, version)
	}
	return entry, raw, nil
}

// ResolveCriterion returns the entry and loaded criterion for an
// exact reference from THIS loaded registry state, with two-way
// identity (registration ↔ self-declaration, the L9/L10 pattern).
func (r *Registry) ResolveCriterion(ref string) (*Entry, *Criterion, error) {
	if r.Kind != "criteria" {
		return nil, nil, fmt.Errorf("%w: %q is a %s registry, not a criteria registry", ErrResolve, r.dir, r.Kind)
	}
	entry, raw, err := r.resolveEntry(ref, maxCriterionBytes)
	if err != nil {
		return nil, nil, err
	}
	c, err := ParseCriterion(raw, ref)
	if err != nil {
		return nil, nil, err
	}
	if c.Name != entry.Name || c.Criterion != entry.Version {
		return nil, nil, fmt.Errorf("%w: %s: criterion self-declaration (%s@%d) disagrees with the registration — two-way identity check", ErrResolve, ref, c.Name, c.Criterion)
	}
	return entry, c, nil
}

// ResolveSet returns the entry and loaded regression set for an exact
// reference, two-way identity checked. Member criteria are NOT
// resolved here — a comparison request resolves each member against
// the criteria registry at evaluation time, so a missing member is a
// per-constituent refusal, never a silent gap (D-L11-9 §3).
func (r *Registry) ResolveSet(ref string) (*Entry, *RegressionSet, error) {
	if r.Kind != "regression-sets" {
		return nil, nil, fmt.Errorf("%w: %q is a %s registry, not a regression-sets registry", ErrResolve, r.dir, r.Kind)
	}
	entry, raw, err := r.resolveEntry(ref, maxSetBytes)
	if err != nil {
		return nil, nil, err
	}
	s, err := ParseRegressionSet(raw, ref)
	if err != nil {
		return nil, nil, err
	}
	if s.Name != entry.Name || s.Set != entry.Version {
		return nil, nil, fmt.Errorf("%w: %s: set self-declaration (%s@%d) disagrees with the registration — two-way identity check", ErrResolve, ref, s.Name, s.Set)
	}
	return entry, s, nil
}

// RegressionSet is the registered grouping of exact criterion pins
// (D-L11-9; D-L11-13 Amendment 1: a case is represented through K,
// grouping into S is a Governance-registered set of K identities).
type RegressionSet struct {
	Version int      `json:"version"`
	Name    string   `json:"name"`
	Set     int      `json:"set_version"`
	Members []string `json:"members"` // exact "name@version" pins

	SHA256 string `json:"-"`
	Raw    []byte `json:"-"`
}

// ParseRegressionSet validates set bytes fail-closed: exact member
// pins only — no floating members, or set identity means nothing.
func ParseRegressionSet(raw []byte, origin string) (*RegressionSet, error) {
	if err := checkNoDuplicateKeys(raw); err != nil {
		return nil, fmt.Errorf("%w: %s: %v", ErrSet, origin, err)
	}
	dec := jsonDecoder(raw)
	var s RegressionSet
	if err := dec.Decode(&s); err != nil {
		return nil, fmt.Errorf("%w: %s: %v", ErrSet, origin, err)
	}
	if dec.More() {
		return nil, fmt.Errorf("%w: %s: trailing content", ErrSet, origin)
	}
	if s.Version != 1 {
		return nil, fmt.Errorf("%w: %s: version must be 1", ErrSet, origin)
	}
	if !nameSyntax.MatchString(s.Name) || len(s.Name) > MaxNameLen {
		return nil, fmt.Errorf("%w: %s: bad set name %q", ErrSet, origin, s.Name)
	}
	if err := checkNameSemantics(s.Name, "regression set"); err != nil {
		return nil, fmt.Errorf("%w: %s: %v", ErrSet, origin, err)
	}
	if s.Set < 1 {
		return nil, fmt.Errorf("%w: %s: set_version must be a positive integer", ErrSet, origin)
	}
	if len(s.Members) == 0 {
		return nil, fmt.Errorf("%w: %s: a regression set enumerates at least one member", ErrSet, origin)
	}
	seen := map[string]bool{}
	for _, m := range s.Members {
		if _, _, err := ParseRef(m); err != nil {
			return nil, fmt.Errorf("%w: %s: member %q: %v", ErrSet, origin, m, err)
		}
		if seen[m] {
			return nil, fmt.Errorf("%w: %s: duplicate member %q", ErrSet, origin, m)
		}
		seen[m] = true
	}
	s.SHA256 = hashBytes(raw)
	s.Raw = raw
	return &s, nil
}
