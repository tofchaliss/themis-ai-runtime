package instructions

// Repository-instruction activation (L5-M4, D-L5-8 / Q-L5-8, closing
// the archived Q-L1-1 IOU). Provisioning establishes availability and
// identity; registration establishes authority. Four controls, L5
// supplying exactly one:
//
//	registration (this artifact, Themis-owned)
//	+ verified pinned checkout (L5's provenance — the caller passes
//	  identity from a provisioned, post-condition-verified workspace)
//	+ L1 scope cap (ScopeRepository ranks below every governed scope
//	  in the fixed precedence order)
//	+ L1 pattern gate (repository sources are untrusted: directive
//	  patterns and the secret scan run per load)
//
// An unregistered repository's instruction file is ordinary workspace
// data — invisible to the instruction plane, without error.
// Registration binds repository identity, never a filesystem path,
// and never pins content hashes (content-governed instructions belong
// in the governed tree; the scope cap and pattern gate are what make
// repo-mutable content safe).

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/tofchaliss/themis/confine"
)

var ErrRegistrationInvalid = errors.New("invalid repository registration")

// RegisteredRepo is one repository allowed to speak at repository
// scope, limited to an explicit instruction-path allowlist.
type RegisteredRepo struct {
	Identity string   `json:"identity"`
	Paths    []string `json:"paths"`
}

// RepoRegistration is the governed registration artifact.
type RepoRegistration struct {
	Version      int              `json:"version"`
	Repositories []RegisteredRepo `json:"repositories"`

	Hash string `json:"-"`
}

var repoIdentity = regexp.MustCompile(`^[a-zA-Z0-9._-]+(/[a-zA-Z0-9._-]+)*$`)

// LoadRepoRegistration: fail-closed governed-artifact posture.
func LoadRepoRegistration(path string) (*RepoRegistration, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("%w: %s: %v", ErrRegistrationInvalid, path, err)
	}
	dec := json.NewDecoder(strings.NewReader(string(raw)))
	dec.DisallowUnknownFields()
	var r RepoRegistration
	if err := dec.Decode(&r); err != nil {
		return nil, fmt.Errorf("%w: %s: %v", ErrRegistrationInvalid, path, err)
	}
	if dec.More() {
		return nil, fmt.Errorf("%w: %s: trailing content", ErrRegistrationInvalid, path)
	}
	if r.Version < 1 || len(r.Repositories) == 0 {
		return nil, fmt.Errorf("%w: %s: version and at least one repository required", ErrRegistrationInvalid, path)
	}
	seen := map[string]bool{}
	for _, repo := range r.Repositories {
		if !repoIdentity.MatchString(repo.Identity) || seen[repo.Identity] {
			return nil, fmt.Errorf("%w: bad or duplicate identity %q", ErrRegistrationInvalid, repo.Identity)
		}
		for _, seg := range strings.Split(repo.Identity, "/") {
			if seg == "." || seg == ".." {
				return nil, fmt.Errorf("%w: bad identity %q", ErrRegistrationInvalid, repo.Identity)
			}
		}
		seen[repo.Identity] = true
		if len(repo.Paths) == 0 {
			return nil, fmt.Errorf("%w: %q registers no instruction paths", ErrRegistrationInvalid, repo.Identity)
		}
		seenPath := map[string]bool{}
		for _, p := range repo.Paths {
			// Relative, traversal-free, non-VCS — via the canonical
			// deny-list predicate (per-segment, case-folded), never a
			// weaker string-prefix twin (M4 security review LOW).
			// Confinement proper runs again at activation against the
			// live worktree.
			if p == "" || strings.HasPrefix(p, "/") || strings.Contains(p, "..") || confine.DeniedVCSPath(p) {
				return nil, fmt.Errorf("%w: %q: bad instruction path %q", ErrRegistrationInvalid, repo.Identity, p)
			}
			clean := filepath.Clean(p)
			if seenPath[clean] {
				return nil, fmt.Errorf("%w: %q: duplicate instruction path %q", ErrRegistrationInvalid, repo.Identity, p)
			}
			seenPath[clean] = true
		}
	}
	sum := sha256.Sum256(raw)
	r.Hash = hex.EncodeToString(sum[:])
	return &r, nil
}

func (r *RepoRegistration) lookup(identity string) *RegisteredRepo {
	for i := range r.Repositories {
		if r.Repositories[i].Identity == identity {
			return &r.Repositories[i]
		}
	}
	return nil
}

var pinnedSHASyntax = regexp.MustCompile(`^[0-9a-f]{40}$`)

// ActivationRecord is the provenance L5 contributed, recorded per
// activation: which repository, which bytes (pin), which registered
// files actually resolved, and the content hash of each at
// activation time — the full Q-L5-8.5 tuple {repo, SHA, path,
// content hash} in one artifact (M4 security review LOW). Load
// re-hashes per instruction (BodyHash covers the body after
// frontmatter); ContentHashes cover the whole file as activated.
type ActivationRecord struct {
	Repo             string
	PinnedSHA        string
	RegistrationHash string
	Paths            []string
	ContentHashes    []string // sha256 hex, parallel to Paths
}

// ActivateRepositorySource computes eligibility for one provisioned
// repository:
//
//	eligible = registered(identity) ∧ provisioned(identity, pinned
//	SHA — caller passes both from a verified workspace) ∧ each
//	registered path resolves to a regular file with no symlink
//	traversal (the shared CreateMode-strength predicate — no
//	instruction-specific filesystem rule).
//
// Unregistered repository, or a registered path simply absent from
// the checkout: (zero, false, nil) — data, not instructions, no
// error. A registered path that exists but violates the resolution
// rules (symlink anywhere): typed error — a registered instruction
// channel in a refusable state is a real conflict, never a quiet
// skip.
func ActivateRepositorySource(reg *RepoRegistration, identity, pinnedSHA, worktreeRoot string) (Source, ActivationRecord, bool, error) {
	if reg == nil {
		return Source{}, ActivationRecord{}, false, fmt.Errorf("%w: nil registration", ErrRegistrationInvalid)
	}
	if !pinnedSHASyntax.MatchString(pinnedSHA) {
		return Source{}, ActivationRecord{}, false, fmt.Errorf("%w: activation requires a full pinned SHA, got %q", ErrRegistrationInvalid, pinnedSHA)
	}
	entry := reg.lookup(identity)
	if entry == nil {
		// Unregistered: invisible to the instruction plane, no error
		// (closed vocabulary — absent = nonexistent).
		return Source{}, ActivationRecord{}, false, nil
	}
	rec := ActivationRecord{Repo: identity, PinnedSHA: pinnedSHA, RegistrationHash: reg.Hash}
	var files []string
	for _, rel := range entry.Paths {
		// CreateMode-strength resolution: refuses symlinks anywhere in
		// the path, .git* components, and escapes; allows a
		// nonexistent final (checked next).
		abs, err := confine.CreatePath(worktreeRoot, rel)
		if err != nil {
			return Source{}, ActivationRecord{}, false, fmt.Errorf("%w: %s: registered path %q: %v", ErrRegistrationInvalid, identity, rel, err)
		}
		info, err := os.Lstat(abs)
		if err != nil {
			if os.IsNotExist(err) {
				continue // the repo carries no instructions at this pin — quiet
			}
			return Source{}, ActivationRecord{}, false, fmt.Errorf("%w: %s: %v", ErrRegistrationInvalid, identity, err)
		}
		if !info.Mode().IsRegular() {
			return Source{}, ActivationRecord{}, false, fmt.Errorf("%w: %s: registered path %q must be a regular file", ErrRegistrationInvalid, identity, rel)
		}
		b, err := os.ReadFile(abs)
		if err != nil {
			return Source{}, ActivationRecord{}, false, fmt.Errorf("%w: %s: %v", ErrRegistrationInvalid, identity, err)
		}
		sum := sha256.Sum256(b)
		files = append(files, abs)
		rec.Paths = append(rec.Paths, rel)
		rec.ContentHashes = append(rec.ContentHashes, hex.EncodeToString(sum[:]))
	}
	if len(files) == 0 {
		return Source{}, ActivationRecord{}, false, nil
	}
	return Source{Kind: ScopeRepository, files: files, activated: true}, rec, true, nil
}
