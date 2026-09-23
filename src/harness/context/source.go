package context

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/tofchaliss/themis/confine"
)

// SourceKind is a registered connector kind. Recognition never
// discovery: only registered kinds produce items, and each kind
// constrains which authority classes its registration may declare —
// that is how fail-closed classification becomes mechanical.
type SourceKind string

const (
	// KindInline: the task envelope's data half. Always
	// external-untrusted: caller-supplied facts earn nothing.
	KindInline SourceKind = "inline"
	// KindFilesystem: confined workspace reads. External-untrusted in
	// v1 — file authorship is unprovable until L5 provenance exists.
	KindFilesystem SourceKind = "filesystem"
	// KindSearch: deterministic lexical retrieval over the confined
	// workspace (M3): walk + substring match, sorted results, capped.
	// External-untrusted like all repository content.
	KindSearch SourceKind = "search"
	// KindThemis: reads through the Themis-owned data-access boundary.
	// The only v1 kind that may declare governed classes. v1 ships the
	// typed contract as a stub (gate-0 decision); real wiring waits
	// for integrations/themis.
	KindThemis SourceKind = "themis"
	// KindRecordObject (L8 amendment, C-L8-7): a lazy read of ONE
	// content-addressed L6 object the L8 seam already resolved and
	// classified from its witnessing event. Metadata-only
	// construction: the seam supplies the object id, the item kind,
	// and the DERIVED class/sensitivity (class(item) = f(witnessing
	// event), C-L8-18); collect() fetches the bytes through the
	// reader and verifies them against the address, so L2 pulls
	// evidence under its own caps and the seam reads no bytes.
	KindRecordObject SourceKind = "record-object"
)

// AbsentSource declares a slot empty (L8 seam usage): no reader, no
// object, no items, floor class. Gather treats it as unavailable and
// lets the slot's requirement decide; it never carries authority, so
// the slot's permitted-class check does not apply to it (C-L8-2: a
// permitted class is never assigned by naming it).
func AbsentSource(slot string) Source {
	return Source{Name: "absent:" + slot, Kind: KindRecordObject, Authority: AuthorityExternalUntrusted,
		Sensitivity: SensitivityPublic, Author: "seam"}
}

// IsAbsence reports a declared-absence source.
func (s Source) IsAbsence() bool {
	return s.Kind == KindRecordObject && s.ObjectID == "" && len(s.Items) == 0
}

// ObjectReader is the typed read seam to the content-addressed store
// (L6). L2 never learns where objects live; a read failure is a hard
// error (a named reference that cannot be read is a refusal, never a
// silent absence), and the reader's own corruption verdict propagates.
type ObjectReader interface {
	GetObject(id string) ([]byte, error)
}

// allowedClasses is the registration constraint table: a source kind
// can only mint the classes its provenance can actually prove.
var allowedClasses = map[SourceKind][]AuthorityClass{
	KindInline:     {AuthorityExternalUntrusted},
	KindFilesystem: {AuthorityExternalUntrusted},
	KindSearch:     {AuthorityExternalUntrusted},
	KindThemis:     {AuthorityGovernedRecord, AuthorityGovernedExternal},
	// The record-object kind carries whatever class the witnessing
	// event derived (C-L8-5): the registration is the event, and the
	// seam that read it is deterministic machinery, not a caller.
	KindRecordObject: {AuthorityGovernedRecord, AuthorityGovernedExternal, AuthorityDerived, AuthorityExternalUntrusted},
}

// Source is a registered context source. Authority and Sensitivity
// are fixed at registration — never caller- or item-supplied.
type Source struct {
	Name        string
	Kind        SourceKind
	Authority   AuthorityClass
	Sensitivity Sensitivity
	Author      string // authoring party recorded in provenance

	// KindInline: the task facts. Evidence/kind/version are honored;
	// every classification field is overwritten from the registration.
	Items []ContextItem
	// KindFilesystem: confinement root + relative paths.
	// KindSearch: confinement root + Query (+ optional Suffix filter).
	Root   string
	Paths  []string
	Query  string
	Suffix string
	// KindThemis: the typed read contract (stub in v1).
	Reader ThemisReader
	// KindRecordObject: the content address to fetch and the reader;
	// Items[0].Kind / Version carry the item metadata (exactly one).
	ObjectID string
	Objects  ObjectReader
}

// ThemisReader is the typed seam to the Themis-owned data-access
// boundary. L2 never sees storage implementation details; a DB
// migration is never a model-facing change (Q-L2-8).
type ThemisReader interface {
	// Read returns evidence bytes plus the record's version/as-of.
	Read(kind string) (evidence []byte, version string, err error)
}

// Metadata fields render into the model view's frame headers, so
// they are constrained: single-line, no control characters, no
// brackets (marker spoofing), bounded length (framing-overhead
// amplification). Evidence bytes are NOT constrained — only metadata
// (security review HIGH-1/3).
var metaSafe = regexp.MustCompile(`^[^\x00-\x1f\x7f\[\]]*$`)

const (
	maxKindLen    = 256
	maxVersionLen = 256
	maxNameLen    = 128
)

func checkMeta(field, value string, max int) error {
	if len(value) > max {
		return fmt.Errorf("%w: %s exceeds %d bytes", ErrMetadataInvalid, field, max)
	}
	if !metaSafe.MatchString(value) {
		return fmt.Errorf("%w: %s contains control or bracket characters", ErrMetadataInvalid, field)
	}
	return nil
}

func checkSource(s Source) error {
	if s.Name == "" {
		return fmt.Errorf("%w: source has no registered name", ErrUnrecognizedSource)
	}
	if err := checkMeta("source name", s.Name, maxNameLen); err != nil {
		return err
	}
	classes, ok := allowedClasses[s.Kind]
	if !ok {
		return fmt.Errorf("%w: kind %q", ErrUnrecognizedSource, s.Kind)
	}
	if _, ok := sensitivityRank[s.Sensitivity]; !ok {
		return fmt.Errorf("%w: source %s has unknown sensitivity %q", ErrUnrecognizedSource, s.Name, s.Sensitivity)
	}
	for _, c := range classes {
		if s.Authority == c {
			return nil
		}
	}
	return fmt.Errorf("%w: kind %q cannot declare authority class %q — higher classes are earned through registered provenance",
		ErrUnrecognizedSource, s.Kind, s.Authority)
}

// collect gathers the source's items. Classification, provenance,
// producer, and hash are stamped from the registration — anything the
// caller pre-filled in those fields is overwritten (no field
// determines its own treatment). A retrieval failure returns
// (nil, false, nil): source unavailable, which the slot's requirement
// classifies. Confinement violations are hard errors, never
// "unavailable".
func (s Source) collect() (items []ContextItem, available bool, err error) {
	origin := "external"
	if s.Kind == KindThemis {
		origin = "themis"
	}
	stamp := func(it ContextItem, evidence []byte, version string) (ContextItem, error) {
		if len(evidence) > MaxItemBytes {
			return it, fmt.Errorf("%w: %s/%s is %d bytes (cap %d)", ErrItemTooLarge, s.Name, it.Kind, len(evidence), MaxItemBytes)
		}
		if err := checkMeta("item kind", it.Kind, maxKindLen); err != nil {
			return it, err
		}
		if err := checkMeta("item version", it.Version, maxVersionLen); err != nil {
			return it, err
		}
		if version != "" {
			if err := checkMeta("item version", version, maxVersionLen); err != nil {
				return it, err
			}
		}
		it.Provenance = Provenance{Origin: origin, Source: s.Name, Author: s.Author}
		it.Authority = s.Authority
		it.Sensitivity = s.Sensitivity
		it.Producer = s.Name
		it.Evidence = evidence
		if version != "" {
			it.Version = version
		}
		it.Hash = evidenceHash(evidence)
		return it, nil
	}

	switch s.Kind {
	case KindRecordObject:
		if s.ObjectID == "" && len(s.Items) == 0 {
			// Declared absence: the seam names every non-withheld slot
			// and this one has no reference — the slot's requirement
			// decides (optional → typed absence; required → refusal).
			return nil, false, nil
		}
		if s.Objects == nil || s.ObjectID == "" || len(s.Items) != 1 {
			return nil, false, fmt.Errorf("%w: record-object source %s needs a reader, an object id, and exactly one item declaration", ErrUnrecognizedSource, s.Name)
		}
		b, err := s.Objects.GetObject(s.ObjectID)
		if err != nil {
			// Hard error, never "unavailable": the reference was named
			// and re-established; a read failure refuses the whole
			// gather (C-L8-8 §5), and a corruption verdict propagates.
			return nil, false, fmt.Errorf("%w: record-object source %s: %w", ErrUnrecognizedSource, s.Name, err)
		}
		if "sha256:"+evidenceHash(b) != s.ObjectID {
			return nil, false, fmt.Errorf("%w: record-object source %s: bytes do not match their content address", ErrUnrecognizedSource, s.Name)
		}
		it, err := stamp(ContextItem{Kind: s.Items[0].Kind, Version: s.Items[0].Version}, b, "")
		if err != nil {
			return nil, false, err
		}
		return []ContextItem{it}, true, nil

	case KindInline:
		for _, raw := range s.Items {
			it, err := stamp(ContextItem{Kind: raw.Kind, Version: raw.Version}, raw.Evidence, "")
			if err != nil {
				return nil, false, err
			}
			items = append(items, it)
		}
		return items, true, nil

	case KindFilesystem:
		paths := append([]string(nil), s.Paths...)
		sort.Strings(paths)
		for _, rel := range paths {
			abs, err := confinedPath(s.Root, rel)
			if err != nil {
				return nil, false, err
			}
			b, err := os.ReadFile(abs)
			if err != nil {
				return nil, false, nil // source unavailable; requirement decides
			}
			it, err := stamp(ContextItem{Kind: "file:" + filepath.ToSlash(rel), Version: rel}, b, "")
			if err != nil {
				return nil, false, err
			}
			items = append(items, it)
		}
		return items, true, nil

	case KindSearch:
		if s.Query == "" {
			return nil, false, fmt.Errorf("%w: search source %s has no query", ErrUnrecognizedSource, s.Name)
		}
		rootAbs, err := confinedPath(s.Root, ".")
		if err != nil {
			// "." is always lexically inside; only a missing root errors.
			return nil, false, err
		}
		const maxMatches = 64
		var matches []string
		var confineErr error
		walkErr := filepath.WalkDir(rootAbs, func(path string, d os.DirEntry, err error) error {
			if err != nil || d.IsDir() {
				return err
			}
			// Only regular files: a file symlink would be followed by
			// ReadFile and could point outside the root (security
			// review HIGH-2) — refuse hard, matching the filesystem
			// kind's confinement contract.
			if !d.Type().IsRegular() {
				rel, _ := filepath.Rel(rootAbs, path)
				confineErr = fmt.Errorf("%w: %q is not a regular file", ErrConfinement, rel)
				return confineErr
			}
			if s.Suffix != "" && !strings.HasSuffix(path, s.Suffix) {
				return nil
			}
			if info, err := d.Info(); err != nil || info.Size() > MaxItemBytes {
				// Oversized files cannot become items; skipping the scan
				// read bounds memory. Deterministic: size is the guard.
				return nil
			}
			b, err := os.ReadFile(path)
			if err != nil || !strings.Contains(string(b), s.Query) {
				return nil
			}
			rel, err := filepath.Rel(rootAbs, path)
			if err != nil {
				return err
			}
			matches = append(matches, filepath.ToSlash(rel))
			return nil
		})
		if confineErr != nil {
			return nil, false, confineErr
		}
		if walkErr != nil {
			return nil, false, nil
		}
		sort.Strings(matches)
		// No silent caps: over-cap match sets refuse deterministically
		// rather than truncating into a complete-looking result
		// (security review MEDIUM-1) — narrow the query or suffix.
		if len(matches) > maxMatches {
			return nil, false, fmt.Errorf("%w: search %s matched %d files (cap %d) — narrow the query",
				ErrContextTooLarge, s.Name, len(matches), maxMatches)
		}
		for _, rel := range matches {
			abs, err := confinedPath(s.Root, rel)
			if err != nil {
				return nil, false, err
			}
			b, err := os.ReadFile(abs)
			if err != nil {
				return nil, false, nil
			}
			it, err := stamp(ContextItem{Kind: "file:" + rel, Version: rel}, b, "")
			if err != nil {
				return nil, false, err
			}
			items = append(items, it)
		}
		return items, len(items) > 0, nil

	case KindThemis:
		if s.Reader == nil {
			return nil, false, nil // stub not wired: unavailable, typed
		}
		for _, raw := range s.Items {
			evidence, version, err := s.Reader.Read(raw.Kind)
			if err != nil {
				return nil, false, nil
			}
			it, err := stamp(ContextItem{Kind: raw.Kind}, evidence, version)
			if err != nil {
				return nil, false, err
			}
			items = append(items, it)
		}
		return items, true, nil
	}
	return nil, false, fmt.Errorf("%w: kind %q", ErrUnrecognizedSource, s.Kind)
}

// confinedPath resolves rel under root and refuses any escape —
// absolute paths, "..", and symlinks resolving outside the root
// (gate-0 decision). A confinement violation is a hard refusal, never
// a quiet skip.
// confinedPath delegates to the canonical leaf implementation
// (confine.ResolvePath) — semantics unchanged from the shipped L2
// contract; ErrConfinement is the same error value.
func confinedPath(root, rel string) (string, error) {
	return confine.ResolvePath(root, rel)
}

// ConfinePath is the exported confinement check for other harness
// layers (L4 executors): resolves rel under root, refusing absolute
// paths, "..", and symlink escapes. Single implementation — tool
// target validation and context gathering must never drift apart.
func ConfinePath(root, rel string) (string, error) {
	return confinedPath(root, rel)
}
