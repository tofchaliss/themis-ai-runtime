package skills

// The Skill catalog: governance's append-only registration record
// (D-L9-3/10). Registration in this governed file IS the v1 review
// evidence; the machinery here only READS — there is deliberately no
// write API anywhere in this package, so "machinery must not
// self-register" holds structurally (Gate 0 constraint), and no tool
// verb can reach it (the model-facing wall is the registry, which has
// no catalog capability at all).

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

type EntryState string

const (
	StateActive    EntryState = "active"
	StateWithdrawn EntryState = "withdrawn"
)

// Entry is one immutable name@version → composition-hash binding.
type Entry struct {
	Name         string     `json:"name"`
	Version      int        `json:"version"`
	Composition  string     `json:"composition_sha256"`
	ManifestPath string     `json:"manifest_path"` // relative to the catalog file
	State        EntryState `json:"state"`
	Steward      string     `json:"steward,omitempty"` // accountability metadata, never authority
}

type Catalog struct {
	Version int     `json:"version"`
	Entries []Entry `json:"entries"`

	Hash string `json:"-"`
	dir  string
}

// LoadCatalog reads the governed catalog fail-closed. One load is one
// atomic catalog state: resolution and instantiation work from a
// single Load result (D-L9-10 TOCTOU rule).
func LoadCatalog(path string) (*Catalog, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("%w: %s: %v", ErrCatalog, path, err)
	}
	dec := json.NewDecoder(strings.NewReader(string(raw)))
	dec.DisallowUnknownFields()
	var c Catalog
	if err := dec.Decode(&c); err != nil {
		return nil, fmt.Errorf("%w: %s: %v", ErrCatalog, path, err)
	}
	if dec.More() {
		return nil, fmt.Errorf("%w: %s: trailing content", ErrCatalog, path)
	}
	if c.Version < 1 {
		return nil, fmt.Errorf("%w: %s: version required", ErrCatalog, path)
	}
	seen := map[string]bool{}
	for _, e := range c.Entries {
		if !nameSyntax.MatchString(e.Name) || e.Version < 1 {
			return nil, fmt.Errorf("%w: bad entry identity %q@%d", ErrCatalog, e.Name, e.Version)
		}
		key := fmt.Sprintf("%s@%d", e.Name, e.Version)
		if seen[key] {
			// An immutable binding can never be restated: a duplicate
			// is either a rebind attempt or a corrupted catalog.
			return nil, fmt.Errorf("%w: duplicate registration %s — bindings are immutable", ErrCatalog, key)
		}
		seen[key] = true
		if !shaSyntax.MatchString(e.Composition) {
			return nil, fmt.Errorf("%w: %s: composition_sha256 must be a sha256 hex digest", ErrCatalog, key)
		}
		if e.ManifestPath == "" || filepath.IsAbs(e.ManifestPath) || strings.Contains(e.ManifestPath, "..") {
			return nil, fmt.Errorf("%w: %s: manifest_path must be relative within the catalog root", ErrCatalog, key)
		}
		if e.State != StateActive && e.State != StateWithdrawn {
			return nil, fmt.Errorf("%w: %s: unknown state %q", ErrCatalog, key, e.State)
		}
	}
	c.Hash = hashBytes(raw)
	c.dir = filepath.Dir(path)
	return &c, nil
}

// Root returns the catalog's directory — the governed catalog root,
// for the disjointness check at the instantiation call site.
func (c *Catalog) Root() string { return c.dir }

// CheckAppendOnly verifies this catalog state against a previously
// observed one: every prior binding must still be present with the
// SAME composition hash, and state may only advance active→withdrawn
// (D-L9-10). Deletion, rebinding, and un-withdrawal are refusals —
// out-of-band catalog mutation is detected rather than trusted.
// Governance holds the prior state; the harness only verifies.
func (c *Catalog) CheckAppendOnly(prior *Catalog) error {
	if prior == nil {
		return nil
	}
	current := map[string]Entry{}
	for _, e := range c.Entries {
		current[fmt.Sprintf("%s@%d", e.Name, e.Version)] = e
	}
	for _, p := range prior.Entries {
		key := fmt.Sprintf("%s@%d", p.Name, p.Version)
		cur, ok := current[key]
		if !ok {
			return fmt.Errorf("%w: %s disappeared — the catalog is append-only, registrations remain permanently interpretable", ErrCatalog, key)
		}
		if cur.Composition != p.Composition {
			return fmt.Errorf("%w: %s rebound to a different composition — bindings are immutable", ErrCatalog, key)
		}
		if p.State == StateWithdrawn && cur.State != StateWithdrawn {
			return fmt.Errorf("%w: %s un-withdrawn — state advances active→withdrawn only", ErrCatalog, key)
		}
	}
	return nil
}

// ParseRef parses an exact "name@version" reference. Floating
// references — name-only, @latest, ranges — are refused: "latest" is
// a mutable lookup policy, not a version (D-L9-10).
func ParseRef(ref string) (string, int, error) {
	at := strings.LastIndex(ref, "@")
	if at <= 0 || at == len(ref)-1 {
		return "", 0, fmt.Errorf("%w: %q: an exact name@version is required — nothing is defaulted", ErrResolve, ref)
	}
	name, vs := ref[:at], ref[at+1:]
	if !nameSyntax.MatchString(name) {
		return "", 0, fmt.Errorf("%w: bad skill name %q", ErrResolve, name)
	}
	v, err := strconv.Atoi(vs)
	if err != nil || v < 1 {
		return "", 0, fmt.Errorf("%w: %q: version must be an exact positive integer (no latest, no ranges)", ErrResolve, ref)
	}
	return name, v, nil
}

// Resolve returns the entry for an exact reference from THIS loaded
// catalog state, and verifies executability: withdrawn entries refuse
// typed. The returned manifest is loaded and its composition hash is
// verified against the registered binding — identity is registration,
// not content resemblance (D-L9-8 wall 4).
func (c *Catalog) Resolve(ref string) (*Entry, *Manifest, error) {
	name, version, err := ParseRef(ref)
	if err != nil {
		return nil, nil, err
	}
	var entry *Entry
	for i := range c.Entries {
		if c.Entries[i].Name == name && c.Entries[i].Version == version {
			entry = &c.Entries[i]
			break
		}
	}
	if entry == nil {
		return nil, nil, fmt.Errorf("%w: %s@%d is not registered — unregistered skill-shaped artifacts are data", ErrResolve, name, version)
	}
	if entry.State == StateWithdrawn {
		return nil, nil, fmt.Errorf("%w: %s@%d is withdrawn — new instantiation is refused; historical records remain interpretable", ErrResolve, name, version)
	}
	m, err := LoadManifest(filepath.Join(c.dir, entry.ManifestPath))
	if err != nil {
		return nil, nil, err
	}
	if m.CompositionHash != entry.Composition {
		return nil, nil, fmt.Errorf("%w: %s@%d: manifest bytes do not match the registered composition hash", ErrResolve, name, version)
	}
	if m.Name != name || m.Skill != version {
		return nil, nil, fmt.Errorf("%w: %s@%d: manifest self-declaration (%s@%d) disagrees with the registration — two-way identity check", ErrResolve, name, version, m.Name, m.Skill)
	}
	return entry, m, nil
}
