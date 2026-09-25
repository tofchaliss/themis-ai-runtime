// Package store is Themis's governed, anchor-pinned, READ-ONLY store:
// the Findings and Products registries (D-T-9) and the L4 read seam
// that serves them to the harness under the capability's registered
// class. This package has no filesystem writer of any kind — a Themis
// read-store immutability wall, pinned by test (D-T-10 wall 3). It
// carries no authority class field: the L4 registry's `trust` on
// get_finding / get_product mints `governed-record`; the store only
// supplies the governed bytes.
package store

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
	"strings"
)

var (
	// ErrStore: a registry that is not a valid governed store.
	ErrStore = errors.New("invalid themis store")
	// ErrUnavailable: a record the read door cannot serve — unknown,
	// or withdrawn (history, not servable current context).
	ErrUnavailable = errors.New("themis record unavailable")
	// ErrPin: the loaded bytes are not the anchored bytes.
	ErrPin = errors.New("themis store is not the anchored artifact")
)

const maxRegistryBytes = 4 << 20

var (
	findingID  = regexp.MustCompile(`^FIND-[A-Za-z0-9][A-Za-z0-9._:-]{0,120}$`)
	productID  = regexp.MustCompile(`^PROD-[A-Za-z0-9][A-Za-z0-9._:-]{0,120}$`)
	severities = map[string]bool{"low": true, "medium": true, "high": true, "critical": true}
)

// Finding is one immutable registered Finding. `summary` is a fact
// statement; dispositions are Positions (D-T-7), never fields here.
type Finding struct {
	ID              string `json:"id"`
	Product         string `json:"product"`
	Advisory        string `json:"advisory"`
	Component       string `json:"component"`
	AffectedVersion string `json:"affected_version"`
	FixedVersion    string `json:"fixed_version"`
	Severity        string `json:"severity"`
	Summary         string `json:"summary"`
	State           string `json:"state"`
	Steward         string `json:"steward"`
}

// Product is a minimal name-and-version referential record (D-T-9,
// locked minimal): no disposition, no hierarchy, no authorization.
type Product struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Version string `json:"version"`
	State   string `json:"state"`
	Steward string `json:"steward"`
}

type findingsFile struct {
	Version int       `json:"version"`
	Entries []Finding `json:"entries"`
}

type productsFile struct {
	Version int       `json:"version"`
	Entries []Product `json:"entries"`
}

// Store is one loaded, pin-verifiable state of both registries. One
// load is one atomic state.
type Store struct {
	findings map[string]json.RawMessage
	products map[string]json.RawMessage
	state    map[string]string // id → state
	// Hash is the anchor pin: SHA-256 over the exact findings bytes
	// followed by the exact products bytes.
	Hash string
	dir  string
}

// Load reads both registries fail-closed from dir (the governed
// `policies/themis`). Records are retained as their EXACT bytes, so the
// read door serves what Governance registered, never a re-encoding.
func Load(dir string) (*Store, error) {
	fb, err := readGoverned(filepath.Join(dir, "findings.json"))
	if err != nil {
		return nil, err
	}
	pb, err := readGoverned(filepath.Join(dir, "products.json"))
	if err != nil {
		return nil, err
	}
	s := &Store{findings: map[string]json.RawMessage{}, products: map[string]json.RawMessage{}, state: map[string]string{}, dir: dir}

	if err := checkKeys(pb); err != nil {
		return nil, fmt.Errorf("%w: products.json: %v", ErrStore, err)
	}
	var pf productsFile
	var praw struct {
		Entries []json.RawMessage `json:"entries"`
	}
	if err := decodeStrict(pb, &pf); err != nil {
		return nil, fmt.Errorf("%w: products.json: %v", ErrStore, err)
	}
	_ = json.Unmarshal(pb, &praw)
	if pf.Version < 1 {
		return nil, fmt.Errorf("%w: products.json: version required", ErrStore)
	}
	for i, p := range pf.Entries {
		if !productID.MatchString(p.ID) || p.Name == "" || p.Version == "" || p.Steward == "" {
			return nil, fmt.Errorf("%w: products.json: entry %d incomplete or bad id %q", ErrStore, i, p.ID)
		}
		if p.State != "active" && p.State != "withdrawn" {
			return nil, fmt.Errorf("%w: products.json: %s: unknown state %q", ErrStore, p.ID, p.State)
		}
		if _, dup := s.products[p.ID]; dup {
			return nil, fmt.Errorf("%w: products.json: duplicate registration %s — records are immutable", ErrStore, p.ID)
		}
		s.products[p.ID] = praw.Entries[i]
		s.state[p.ID] = p.State
	}

	if err := checkKeys(fb); err != nil {
		return nil, fmt.Errorf("%w: findings.json: %v", ErrStore, err)
	}
	var ff findingsFile
	var fraw struct {
		Entries []json.RawMessage `json:"entries"`
	}
	if err := decodeStrict(fb, &ff); err != nil {
		return nil, fmt.Errorf("%w: findings.json: %v", ErrStore, err)
	}
	_ = json.Unmarshal(fb, &fraw)
	if ff.Version < 1 {
		return nil, fmt.Errorf("%w: findings.json: version required", ErrStore)
	}
	for i, f := range ff.Entries {
		if !findingID.MatchString(f.ID) || f.Advisory == "" || f.Component == "" || f.Summary == "" || f.Steward == "" {
			return nil, fmt.Errorf("%w: findings.json: entry %d incomplete or bad id %q", ErrStore, i, f.ID)
		}
		if !severities[f.Severity] {
			return nil, fmt.Errorf("%w: findings.json: %s: unknown severity %q", ErrStore, f.ID, f.Severity)
		}
		if f.State != "active" && f.State != "withdrawn" {
			return nil, fmt.Errorf("%w: findings.json: %s: unknown state %q", ErrStore, f.ID, f.State)
		}
		if _, ok := s.products[f.Product]; !ok {
			return nil, fmt.Errorf("%w: findings.json: %s references unregistered product %q", ErrStore, f.ID, f.Product)
		}
		if _, dup := s.findings[f.ID]; dup {
			return nil, fmt.Errorf("%w: findings.json: duplicate registration %s — records are immutable", ErrStore, f.ID)
		}
		s.findings[f.ID] = fraw.Entries[i]
		s.state[f.ID] = f.State
	}
	sum := sha256.Sum256(append(append([]byte{}, fb...), pb...))
	s.Hash = hex.EncodeToString(sum[:])
	return s, nil
}

// Root is the governed store directory.
func (s *Store) Root() string { return s.dir }

// StoreHash is the pin of the bytes this state was loaded from — what
// the harness compares to the anchor at Open (D-T-9).
func (s *Store) StoreHash() string { return s.Hash }

// Read is the L4 read seam (tools.ThemisSeam): the exact registered
// bytes of an ACTIVE record, by kind and id. Withdrawn and unknown
// records are unavailable — typed, never a best-effort. The store
// mints no class.
func (s *Store) Read(kind, id string) ([]byte, error) {
	var recs map[string]json.RawMessage
	switch kind {
	case "finding":
		recs = s.findings
	case "product":
		recs = s.products
	default:
		return nil, fmt.Errorf("%w: unknown record kind %q", ErrUnavailable, kind)
	}
	raw, ok := recs[id]
	if !ok {
		return nil, fmt.Errorf("%w: no %s %q", ErrUnavailable, kind, id)
	}
	if s.state[id] != "active" {
		return nil, fmt.Errorf("%w: %s %q is withdrawn — history, not servable current context", ErrUnavailable, kind, id)
	}
	return append([]byte(nil), raw...), nil
}

// Exists reports whether a Finding is registered (any state) — the
// D-T-7 existence gate resolves through the store, never through the
// id's shape.
func (s *Store) Exists(kind, id string) (state string, ok bool) {
	switch kind {
	case "finding":
		_, ok = s.findings[id]
	case "product":
		_, ok = s.products[id]
	}
	return s.state[id], ok
}

// HashOf computes the pin for a store directory without loading it —
// what themis-status prints and an anchor pins.
func HashOf(dir string) (string, error) {
	fb, err := os.ReadFile(filepath.Join(dir, "findings.json"))
	if err != nil {
		return "", err
	}
	pb, err := os.ReadFile(filepath.Join(dir, "products.json"))
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(append(append([]byte{}, fb...), pb...))
	return hex.EncodeToString(sum[:]), nil
}

func decodeStrict(raw []byte, v any) error {
	dec := json.NewDecoder(strings.NewReader(string(raw)))
	dec.DisallowUnknownFields()
	if err := dec.Decode(v); err != nil {
		return err
	}
	if dec.More() {
		return errors.New("trailing content")
	}
	return nil
}

// checkKeys: exact lowercase keys, no duplicates (the strictjson wall,
// restated here so this package imports no harness internal).
func checkKeys(raw []byte) error {
	dec := json.NewDecoder(strings.NewReader(string(raw)))
	type frame struct {
		object    bool
		keys      map[string]bool
		nextIsKey bool
	}
	var stack []*frame
	exact := regexp.MustCompile(`^[a-z0-9_]+$`)
	for {
		tok, err := dec.Token()
		if err != nil {
			if err == io.EOF {
				return nil
			}
			return err
		}
		if len(stack) > 0 {
			top := stack[len(stack)-1]
			if top.object && top.nextIsKey {
				if key, ok := tok.(string); ok {
					if !exact.MatchString(key) {
						return fmt.Errorf("key %q is not an exact lowercase key", key)
					}
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

// readGoverned: refuse a non-regular PATH before opening (Lstat — an
// open follows symlinks, so a handle stat alone would admit a registry
// reached through a link), then one open, stat on the handle, bounded
// read.
func readGoverned(path string) ([]byte, error) {
	if li, err := os.Lstat(path); err != nil || !li.Mode().IsRegular() {
		return nil, fmt.Errorf("%w: %s: not a regular file", ErrStore, path)
	}
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("%w: %s: %v", ErrStore, path, err)
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil {
		return nil, fmt.Errorf("%w: %s: %v", ErrStore, path, err)
	}
	if !info.Mode().IsRegular() {
		return nil, fmt.Errorf("%w: %s: not a regular file", ErrStore, path)
	}
	if info.Size() > maxRegistryBytes {
		return nil, fmt.Errorf("%w: %s: exceeds %d bytes", ErrStore, path, maxRegistryBytes)
	}
	buf := make([]byte, info.Size())
	if _, err := io.ReadFull(f, buf); err != nil {
		return nil, fmt.Errorf("%w: %s: %v", ErrStore, path, err)
	}
	return buf, nil
}
