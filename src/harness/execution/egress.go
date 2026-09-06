package execution

// Artifact Egress Contract (Q-L5-10 / D-L5-5): the artifact is a
// diff against the pinned base — structural, provenance-bound,
// content-neutral. The gate judges structure, provenance, and bounds,
// never meaning. Egress reads only a cleanly SEALED workspace
// (Q-L5-12: the object being copied is stable), materializes into
// the ArtifactStore, and L5 custody ends at acknowledgment
// (Q-L5-11). No partial artifacts: any failure produces nothing.

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func hashHex(b []byte) string {
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}

var ErrEgress = errors.New("artifact-egress-refused")

// ChangeEntry is one changed path in the diff-against-pinned-base.
// Symlinks are recorded, never followed: Target is data (bytes in
// the artifact), not a path anything dereferences.
type ChangeEntry struct {
	Path    string `json:"path"`
	Type    string `json:"type"` // added | modified | deleted | symlink
	OldHash string `json:"old_hash,omitempty"`
	NewHash string `json:"new_hash,omitempty"`
	Size    int64  `json:"size"`
	Mode    string `json:"mode,omitempty"`
	// Content carries the full new bytes for added/modified regular
	// files — the artifact is self-contained and reproducible without
	// touching the (destroyed) workspace.
	Content string `json:"content,omitempty"`
	Target  string `json:"target,omitempty"` // symlink target string, as data
}

// ArtifactManifest is the governed hand-off object. Its canonical
// JSON bytes are what the store hashes: the manifest hash IS the
// artifact address.
type ArtifactManifest struct {
	Base struct {
		Repo      string `json:"repo"`
		PinnedSHA string `json:"pinned_sha"`
	} `json:"base"`
	TaskID       string        `json:"task_id"`
	SpecHash     string        `json:"spec_hash"`
	CeilingHash  string        `json:"ceiling_hash"`
	BinaryDigest string        `json:"binary_digest"`
	Provider     string        `json:"provider"`
	Changes      []ChangeEntry `json:"changes"`
	// ExcludedVCS notes any .git*-component paths excluded from the
	// artifact — excluded-and-noted, never shipped (Q-L5-10).
	ExcludedVCS []string `json:"excluded_vcs,omitempty"`
	// Observed egress accounting: the deterministic consumers of the
	// observed limit dimensions (Q-L5-9.5).
	ObservedTotalBytes int64 `json:"observed_total_bytes"`
	ObservedFileCount  int64 `json:"observed_file_count"`
}

// Egress runs the Artifact Egress Contract against a cleanly sealed
// environment and persists the result. On success the environment is
// ACKNOWLEDGED and the address is recorded in the trace; on any
// failure the environment stays in EGRESSING for the caller to tear
// down, and nothing was persisted (or, for a persistence failure
// after a built manifest, nothing is acknowledged).
func (e *Env) Egress(ceiling *WorkspaceExecutionCeiling, spec *ProvisionSpec, store *ArtifactStore) (string, error) {
	if err := e.beginEgress(); err != nil {
		return "", err
	}
	manifest, err := e.buildManifest(ceiling, spec)
	if err != nil {
		e.setEgress("refused: " + err.Error())
		return "", err
	}
	raw, err := json.Marshal(manifest)
	if err != nil {
		e.setEgress("refused: manifest encoding")
		return "", fmt.Errorf("%w: %v", ErrEgress, err)
	}
	addr, err := store.Put(raw)
	if err != nil {
		// Persistence failure fails closed: no acknowledgment, no
		// evidence, and the workspace is NOT preserved for recovery —
		// the caller tears down; re-execution is an orchestration
		// decision (Q-L5-11).
		e.setEgress("persistence-failed")
		return "", err
	}
	e.mu.Lock()
	e.trace.ArtifactAddress = addr
	e.trace.EgressOutcome = "acknowledged"
	err = e.transitionLocked(StateAcknowledged, "artifact "+addr[:12])
	e.mu.Unlock()
	if err != nil {
		return "", err
	}
	return addr, nil
}

func (e *Env) setEgress(outcome string) {
	e.mu.Lock()
	e.trace.EgressOutcome = outcome
	e.mu.Unlock()
}

// effectiveBound returns the tightest bound for a dimension: the
// ceiling, narrowed by the spec where the spec requests less. Specs
// narrow, never widen (validated at admission).
func effectiveBound(ceiling int64, spec *ProvisionSpec, dim string) int64 {
	if l, ok := spec.Limit(dim); ok && l.Value < ceiling {
		return l.Value
	}
	return ceiling
}

func (e *Env) buildManifest(ceiling *WorkspaceExecutionCeiling, spec *ProvisionSpec) (*ArtifactManifest, error) {
	e.mu.Lock()
	ws := e.trace.Workspace
	m := &ArtifactManifest{
		TaskID:       spec.TaskID,
		SpecHash:     e.trace.SpecHash,
		CeilingHash:  e.trace.CeilingHash,
		BinaryDigest: e.trace.Binary.Digest,
		Provider:     e.trace.Provider.Name,
	}
	e.mu.Unlock()
	m.Base.Repo = ws.Repo
	m.Base.PinnedSHA = ws.PinnedSHA

	// The workspace is sealed: this listing is of a stable object.
	out, err := e.runGit("egress", e.budget(), ws.Root, "status", "--porcelain=v1", "-z", "-uall")
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrEgress, err)
	}
	maxFile := effectiveBound(ceiling.MaxFileBytes, spec, DimFileBytes)
	maxTotal := effectiveBound(ceiling.MaxTotalBytes, spec, DimDiskBytes)

	for _, entry := range strings.Split(out, "\x00") {
		if len(entry) < 4 {
			continue
		}
		code, rel := entry[:2], entry[3:]
		// Defense in depth: .git* components never ship (git does not
		// list them, but the exclusion is the contract's, not git's).
		if vcsComponent(rel) {
			m.ExcludedVCS = append(m.ExcludedVCS, rel)
			continue
		}
		ce := ChangeEntry{Path: rel}
		switch {
		case code == "??":
			ce.Type = "added"
		case strings.Contains(code, "D"):
			ce.Type = "deleted"
		default:
			ce.Type = "modified"
		}
		if ce.Type != "added" {
			oldHash, err := e.pinnedBlobHash(ws.Root, rel)
			if err != nil {
				return nil, err
			}
			ce.OldHash = oldHash
		}
		if ce.Type != "deleted" {
			info, err := os.Lstat(filepath.Join(ws.Root, rel))
			if err != nil {
				return nil, fmt.Errorf("%w: %s: %v", ErrEgress, rel, err)
			}
			switch {
			case info.Mode()&os.ModeSymlink != 0:
				// Recorded, never followed.
				target, err := os.Readlink(filepath.Join(ws.Root, rel))
				if err != nil {
					return nil, fmt.Errorf("%w: %s: %v", ErrEgress, rel, err)
				}
				ce.Type = "symlink"
				ce.Target = target
			case !info.Mode().IsRegular():
				return nil, fmt.Errorf("%w: %s: only regular files and typed symlink entries may egress", ErrEgress, rel)
			default:
				b, err := os.ReadFile(filepath.Join(ws.Root, rel))
				if err != nil {
					return nil, fmt.Errorf("%w: %s: %v", ErrEgress, rel, err)
				}
				ce.Size = int64(len(b))
				ce.Mode = info.Mode().Perm().String()
				ce.NewHash = hashHex(b)
				ce.Content = string(b)
				if ce.Size > maxFile {
					// The observed file_bytes dimension gating
					// acceptance deterministically (Q-L5-9.5).
					return nil, fmt.Errorf("%w: %s: %d bytes exceeds file_bytes bound %d", ErrEgress, rel, ce.Size, maxFile)
				}
			}
		}
		m.Changes = append(m.Changes, ce)
		m.ObservedFileCount++
		m.ObservedTotalBytes += ce.Size
	}
	if m.ObservedFileCount > ceiling.MaxFileCount {
		return nil, fmt.Errorf("%w: %d changed files exceeds file_count bound %d", ErrEgress, m.ObservedFileCount, ceiling.MaxFileCount)
	}
	if m.ObservedTotalBytes > maxTotal {
		return nil, fmt.Errorf("%w: %d total bytes exceeds disk_bytes bound %d", ErrEgress, m.ObservedTotalBytes, maxTotal)
	}
	return m, nil
}

// pinnedBlobHash returns the sha256 of a path's content at the
// pinned base, via ls-tree + cat-file (no colon-bearing revspecs —
// those are refused by the endpoint guard, deliberately).
func (e *Env) pinnedBlobHash(root, rel string) (string, error) {
	lt, err := e.runGit("egress", e.budget(), root, "ls-tree", "HEAD", "--", rel)
	if err != nil {
		return "", fmt.Errorf("%w: %v", ErrEgress, err)
	}
	fields := strings.Fields(strings.TrimSpace(strings.SplitN(lt, "\t", 2)[0]))
	if len(fields) < 3 || fields[1] != "blob" {
		return "", fmt.Errorf("%w: %s: no pinned blob", ErrEgress, rel)
	}
	blob, err := e.runGit("egress", e.budget(), root, "cat-file", "blob", fields[2])
	if err != nil {
		return "", fmt.Errorf("%w: %v", ErrEgress, err)
	}
	return hashHex([]byte(blob)), nil
}

func vcsComponent(rel string) bool {
	for _, seg := range strings.Split(filepath.ToSlash(rel), "/") {
		if strings.HasPrefix(seg, ".git") {
			return true
		}
	}
	return false
}

