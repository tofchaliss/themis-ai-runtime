// Package decisions loads the governed door-decision records (D-R-2):
// one immutable, content-addressed record per registration,
// withdrawal, or reliance act at a runtime governance door (the skill
// catalog, the anchors registry). A record explains WHY a door was
// exercised — the actor, the rationale, and the L11 evidence considered
// — and is two-way bound to the door entry that cites it. It never
// determines whether a method is executable or relied upon: the
// catalog and the anchor do (D-R-1). This package is read-only and
// reaches no L11 package (D-R-4).
package decisions

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

	"github.com/tofchaliss/themis-ai-runtime/src/harness/internal/strictjson"
)

// ErrDecision: a record is absent, malformed, or not the one the door
// entry cites. Fail closed: the door refuses to load.
var ErrDecision = errors.New("invalid door decision record")

const maxRecordBytes = 64 << 10

var (
	idSyntax     = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{2,79}$`)
	shaSyntax    = regexp.MustCompile(`^[0-9a-f]{64}$`)
	objectSyntax = regexp.MustCompile(`^sha256:[0-9a-f]{64}$`)
	nameSyntax   = regexp.MustCompile(`^[a-z][a-z0-9-]{0,63}$`)
	kinds        = map[string]bool{"registration": true, "withdrawal": true, "reliance": true}
	actorKinds   = map[string]bool{"commit": true, "key": true}
)

// Target is the exact governed identity the decision was about: a
// skill (composition hash) or an anchor (artifact hash).
type Target struct {
	Name    string `json:"name"`
	Version int    `json:"version"`
	Hash    string `json:"hash"`
}

// Evidence is one L11 comparison package the decider considered,
// cited by reference only (D-R-2 §4): criterion identity, the package
// object id, and the record root that holds it. Never interpreted here.
type Evidence struct {
	Criterion       string `json:"criterion"`
	PackageObjectID string `json:"package_object_id"`
	RecordRoot      string `json:"record_root"`
}

// Actor identifies who exercised the door with no stronger
// authentication claim than the door possesses (D-R-2 §3):
// kind "commit" = asserted git identity; kind "key" = authenticated
// Themis principal.
type Actor struct {
	Kind string `json:"kind"`
	ID   string `json:"id"`
}

// Record is the closed schema of policies/decisions/<id>.json.
type Record struct {
	ID        string     `json:"id"`
	Kind      string     `json:"kind"`
	Target    Target     `json:"target"`
	Evidence  []Evidence `json:"evidence"`
	Rationale string     `json:"rationale"`
	DecidedAt string     `json:"decided_at"`
	Actor     Actor      `json:"actor"`

	Hash string `json:"-"` // sha256 of the exact record bytes
}

// Load reads one record fail-closed from dir/<id>.json.
func Load(dir, id string) (*Record, error) {
	if !idSyntax.MatchString(id) {
		return nil, fmt.Errorf("%w: id %q is not well-formed", ErrDecision, id)
	}
	path := filepath.Join(dir, id+".json")
	fi, err := os.Lstat(path)
	if err != nil {
		return nil, fmt.Errorf("%w: %s: %v", ErrDecision, path, err)
	}
	if !fi.Mode().IsRegular() || fi.Size() > maxRecordBytes {
		return nil, fmt.Errorf("%w: %s: not a regular bounded file", ErrDecision, path)
	}
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("%w: %s: %v", ErrDecision, path, err)
	}
	defer f.Close()
	raw, err := io.ReadAll(io.LimitReader(f, maxRecordBytes+1))
	if err != nil {
		return nil, fmt.Errorf("%w: %s: %v", ErrDecision, path, err)
	}
	r, err := Parse(raw, path)
	if err != nil {
		return nil, err
	}
	if r.ID != id {
		return nil, fmt.Errorf("%w: %s: record id %q is not the file's id %q", ErrDecision, path, r.ID, id)
	}
	return r, nil
}

// Parse validates exact bytes against the closed schema.
func Parse(raw []byte, origin string) (*Record, error) {
	if err := strictjson.Check(raw); err != nil {
		return nil, fmt.Errorf("%w: %s: %v", ErrDecision, origin, err)
	}
	dec := json.NewDecoder(strings.NewReader(string(raw)))
	dec.DisallowUnknownFields()
	var r Record
	if err := dec.Decode(&r); err != nil {
		return nil, fmt.Errorf("%w: %s: %v", ErrDecision, origin, err)
	}
	if dec.More() {
		return nil, fmt.Errorf("%w: %s: trailing content", ErrDecision, origin)
	}
	switch {
	case !idSyntax.MatchString(r.ID):
		return nil, fmt.Errorf("%w: %s: id", ErrDecision, origin)
	case !kinds[r.Kind]:
		return nil, fmt.Errorf("%w: %s: kind must be registration, withdrawal, or reliance", ErrDecision, origin)
	case !nameSyntax.MatchString(r.Target.Name) || r.Target.Version < 1 || !shaSyntax.MatchString(r.Target.Hash):
		return nil, fmt.Errorf("%w: %s: target must name an exact governed identity with its hash", ErrDecision, origin)
	case r.Evidence == nil:
		return nil, fmt.Errorf("%w: %s: evidence must be present (empty is an explicit fact)", ErrDecision, origin)
	case strings.TrimSpace(r.Rationale) == "":
		return nil, fmt.Errorf("%w: %s: rationale required", ErrDecision, origin)
	case len(r.DecidedAt) < 10:
		return nil, fmt.Errorf("%w: %s: decided_at required", ErrDecision, origin)
	case !actorKinds[r.Actor.Kind] || strings.TrimSpace(r.Actor.ID) == "":
		return nil, fmt.Errorf("%w: %s: actor must be commit:<author> or key:<KeyID>", ErrDecision, origin)
	}
	for i, e := range r.Evidence {
		if e.Criterion == "" || !objectSyntax.MatchString(e.PackageObjectID) || e.RecordRoot == "" {
			return nil, fmt.Errorf("%w: %s: evidence[%d] must cite criterion, package object id, and record root", ErrDecision, origin, i)
		}
	}
	sum := sha256.Sum256(raw)
	r.Hash = hex.EncodeToString(sum[:])
	return &r, nil
}

// Bind establishes the two-way binding between a door entry and the
// record it cites (D-R-2 §2): the record exists, its bytes hash to
// what the entry pinned, and its target is exactly the entry.
func Bind(dir, ref, refHash, name string, version int, hash string) (*Record, error) {
	r, err := Load(dir, ref)
	if err != nil {
		return nil, err
	}
	if r.Hash != refHash {
		return nil, fmt.Errorf("%w: %s: record bytes hash to %s…, the entry pinned %s…", ErrDecision, ref, r.Hash[:12], short(refHash))
	}
	if r.Target.Name != name || r.Target.Version != version || r.Target.Hash != hash {
		return nil, fmt.Errorf("%w: %s: record targets %s@%d %s…, the entry is %s@%d %s…", ErrDecision, ref, r.Target.Name, r.Target.Version, short(r.Target.Hash), name, version, short(hash))
	}
	return r, nil
}

func short(s string) string {
	if len(s) > 12 {
		return s[:12]
	}
	return s
}

// Dir resolves the decisions directory beside a governed registry file:
// policies/<door>/<file> → policies/decisions.
func Dir(registryPath string) string {
	return filepath.Join(filepath.Dir(filepath.Dir(registryPath)), "decisions")
}
