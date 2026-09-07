package state

// The task manifest: a derived projection of the authoritative event
// stream, never the record itself (Q-L6-6). Single-use identity
// (Q-L6-3), closed monotonic machine, atomic-replace writes,
// manifest-first so a crashed task keeps attribution.

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Manifest is the durable task envelope. Every field is a claim the
// cold verifier re-derives from the stream — the manifest certifies
// nothing itself (Q-L6-3).
type Manifest struct {
	TaskID           string            `json:"task_id"`
	Status           TaskStatus        `json:"status"`
	ConstitutionHash string            `json:"constitution_hash"`
	RetryOf          string            `json:"retry_of,omitempty"`
	GovernedHashes   map[string]string `json:"governed_hashes,omitempty"`
	ArtifactAddrs    []string          `json:"artifact_addresses,omitempty"`
	// Bound once at each terminal transition (Q-L6-3): counts and
	// stream summary over entries [0, EventCount).
	EventCount    int64  `json:"event_count"`
	StreamSummary string `json:"stream_summary,omitempty"`
}

var taskIDSyntax = func(id string) bool {
	if id == "" || len(id) > 128 {
		return false
	}
	for _, r := range id {
		if !(r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == '-' || r == '_' || r == '.') {
			return false
		}
	}
	return !strings.HasPrefix(id, ".")
}

// writeManifest is the single atomic-replace write path:
// write temp → fsync → rename → fsync dir. Rename-replace is
// deliberate here (unlike objects): the manifest is the one mutable
// shape, and atomic replacement is its commit discipline.
func writeManifest(dir string, m *Manifest) error {
	raw, err := json.Marshal(m)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrPersist, err)
	}
	tmp, err := os.CreateTemp(dir, "manifest-")
	if err != nil {
		return fmt.Errorf("%w: %v", ErrPersist, err)
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)
	if err := faultAt("manifest.pre-write"); err != nil {
		tmp.Close()
		return fmt.Errorf("%w: %v", ErrPersist, err)
	}
	if _, err := tmp.Write(raw); err != nil {
		tmp.Close()
		return fmt.Errorf("%w: %v", ErrPersist, err)
	}
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		return fmt.Errorf("%w: %v", ErrPersist, err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("%w: %v", ErrPersist, err)
	}
	if err := faultAt("manifest.pre-rename"); err != nil {
		return fmt.Errorf("%w: %v", ErrPersist, err)
	}
	if err := os.Rename(tmpName, filepath.Join(dir, "manifest.json")); err != nil {
		return fmt.Errorf("%w: %v", ErrPersist, err)
	}
	if err := fsyncDir(dir); err != nil {
		return fmt.Errorf("%w: %v", ErrPersist, err)
	}
	return nil
}

func readManifest(dir string) (*Manifest, error) {
	raw, err := os.ReadFile(filepath.Join(dir, "manifest.json"))
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrPersist, err)
	}
	dec := json.NewDecoder(strings.NewReader(string(raw)))
	dec.DisallowUnknownFields()
	var m Manifest
	if err := dec.Decode(&m); err != nil {
		return nil, fmt.Errorf("%w: manifest: %v", ErrCorrupt, err)
	}
	if dec.More() {
		return nil, fmt.Errorf("%w: manifest: trailing content", ErrCorrupt)
	}
	return &m, nil
}

func terminal(s TaskStatus) bool {
	return s == StatusCompleted || s == StatusFailed || s == StatusFailedPartial
}
