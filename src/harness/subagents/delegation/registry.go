package delegation

// The delegation-template registry: Governance's append-only
// registration record (D-L8-5/6; mirrors verification/registry.go and
// the L9 catalog). Registration is an external governed act. An
// unregistered template-shaped artifact is data: no closest match, no
// latest, no name-only resolution, no template-by-value.

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/tofchaliss/themis-ai-runtime/src/harness/confine"
	"github.com/tofchaliss/themis-ai-runtime/src/harness/internal/strictjson"
)

const (
	maxRegistryBytes = 4 << 20 // 4 MiB
	maxTemplateBytes = 1 << 20 // 1 MiB
)

var (
	shaSyntax  = regexp.MustCompile(`^[0-9a-f]{64}$`)
	nameSyntax = regexp.MustCompile(`^[a-z0-9]+(-[a-z0-9]+)*$`)
)

type EntryState string

const (
	StateActive    EntryState = "active"
	StateWithdrawn EntryState = "withdrawn"
)

// Entry is one immutable name@version → template-hash binding.
// Governance identity is name@version; artifact identity is the hash
// of the manifest bytes (C-L8-15 H). Two names may bind one hash
// (aliases); one name@version never rebinds.
type Entry struct {
	Name         string     `json:"name"`
	Version      int        `json:"version"`
	Template     string     `json:"template_sha256"`
	ManifestPath string     `json:"manifest_path"` // relative to the registry file
	State        EntryState `json:"state"`
	Steward      string     `json:"steward,omitempty"` // accountability metadata, never authority
}

type Registry struct {
	Version int     `json:"version"`
	Entries []Entry `json:"entries"`

	Hash string `json:"-"` // of the exact registry bytes — what the anchor pins
	Raw  []byte `json:"-"`
	dir  string
}

// LoadRegistry reads the governed registry fail-closed. One load is one
// atomic registry state (the D-L9-10 TOCTOU rule): resolution works
// from a single Load result.
func LoadRegistry(path string) (*Registry, error) {
	raw, err := readGoverned(path, maxRegistryBytes, ErrRegistry)
	if err != nil {
		return nil, err
	}
	if err := strictjson.Check(raw); err != nil {
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
		if !shaSyntax.MatchString(e.Template) {
			return nil, fmt.Errorf("%w: %s: template_sha256 must be a sha256 hex digest", ErrRegistry, key)
		}
		if e.ManifestPath == "" || filepath.IsAbs(e.ManifestPath) || strings.Contains(e.ManifestPath, "..") {
			return nil, fmt.Errorf("%w: %s: manifest_path must be relative within the registry root", ErrRegistry, key)
		}
		if e.State != StateActive && e.State != StateWithdrawn {
			return nil, fmt.Errorf("%w: %s: unknown state %q", ErrRegistry, key, e.State)
		}
	}
	r.Hash = hashBytes(raw)
	r.Raw = raw
	r.dir = filepath.Dir(path)
	return &r, nil
}

// Root returns the registry's directory — the governed registry root.
func (r *Registry) Root() string { return r.dir }

// CheckAppendOnly verifies this registry state against a previously
// observed one: every prior binding must still be present with the
// SAME template hash, and state may only advance active→withdrawn
// (C-L8-15 C). Deletion, rebinding, and un-withdrawal are refusals.
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
		if cur.Template != p.Template {
			return fmt.Errorf("%w: %s rebound to a different template — bindings are immutable; any change is a new version", ErrRegistry, key)
		}
		if p.State == StateWithdrawn && cur.State != StateWithdrawn {
			return fmt.Errorf("%w: %s un-withdrawn — state advances active→withdrawn only", ErrRegistry, key)
		}
	}
	return nil
}

// ParseRef parses an exact "name@version" reference. Floating
// references — name-only, @latest, ranges — are refused: "latest" is a
// mutable lookup policy, not a version (D-L8-4 closed world).
func ParseRef(ref string) (string, int, error) {
	at := strings.LastIndex(ref, "@")
	if at <= 0 || at == len(ref)-1 {
		return "", 0, fmt.Errorf("%w: %q: an exact name@version is required — nothing is defaulted", ErrResolve, ref)
	}
	name, vs := ref[:at], ref[at+1:]
	if !nameSyntax.MatchString(name) {
		return "", 0, fmt.Errorf("%w: bad template name %q", ErrResolve, name)
	}
	v, err := strconv.Atoi(vs)
	if err != nil || v < 1 || strconv.Itoa(v) != vs {
		return "", 0, fmt.Errorf("%w: %q: version must be an exact positive integer (no latest, no ranges)", ErrResolve, ref)
	}
	return name, v, nil
}

// Entry returns the registration for an exact reference regardless of
// its lifecycle state — historical registration, not current
// usability (C-L8-14 G, owner LOCK 2026-09-23). Assembly uses it to
// validate a Skill's template_scope: a reference to something that was
// never registered is refused; a withdrawn registration is admissible
// and stage B decides usability at the delegate boundary.
func (r *Registry) Entry(ref string) (*Entry, error) {
	name, version, err := ParseRef(ref)
	if err != nil {
		return nil, err
	}
	for i := range r.Entries {
		if r.Entries[i].Name == name && r.Entries[i].Version == version {
			return &r.Entries[i], nil
		}
	}
	return nil, fmt.Errorf("%w: %s@%d is not registered — unregistered template-shaped artifacts are data", ErrResolve, name, version)
}

// Resolve returns the entry and the loaded template for an exact
// reference from THIS loaded registry state. Withdrawn entries refuse
// typed (forward-only availability, C-L8-14 G). The manifest bytes
// are hash-verified against the registered binding, and the
// manifest's self-declared identity must agree with the registration
// (two-way identity, the L9 pattern).
func (r *Registry) Resolve(ref string) (*Entry, *Template, error) {
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
		return nil, nil, fmt.Errorf("%w: %s@%d is not registered — unregistered template-shaped artifacts are data", ErrResolve, name, version)
	}
	if entry.State == StateWithdrawn {
		return nil, nil, fmt.Errorf("%w: %w: %s@%d — new delegation is refused; historical records remain interpretable", ErrResolve, ErrWithdrawn, name, version)
	}
	manifestPath, err := confine.ResolvePath(r.dir, entry.ManifestPath)
	if err != nil {
		return nil, nil, fmt.Errorf("%w: %s@%d manifest: %v", ErrResolve, name, version, err)
	}
	if info, serr := os.Lstat(manifestPath); serr != nil || !info.Mode().IsRegular() {
		return nil, nil, fmt.Errorf("%w: %s@%d manifest must be a regular file within the registry root", ErrResolve, name, version)
	}
	t, err := LoadTemplate(manifestPath)
	if err != nil {
		return nil, nil, err
	}
	if t.Hash != entry.Template {
		return nil, nil, fmt.Errorf("%w: %w: %s@%d: template bytes do not match the registered hash", ErrResolve, ErrHashMismatch, name, version)
	}
	if t.Name != name || t.TemplateVersion != version {
		return nil, nil, fmt.Errorf("%w: %s@%d: template self-declaration (%s@%d) disagrees with the registration — two-way identity check", ErrResolve, name, version, t.Name, t.TemplateVersion)
	}
	return entry, t, nil
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

// readGoverned: one open, stat on the HANDLE, bounded read on the same
// handle — no stat-then-reopen window a symlink swap could widen (the
// L10 close-review posture).
func readGoverned(path string, maxBytes int64, class error) ([]byte, error) {
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
	buf := make([]byte, info.Size())
	if _, err := io.ReadFull(f, buf); err != nil {
		return nil, fmt.Errorf("%w: %s: %v", class, path, err)
	}
	return buf, nil
}
