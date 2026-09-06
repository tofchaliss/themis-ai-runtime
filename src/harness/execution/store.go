package execution

// ArtifactStore (Q-L5-11): write-once, content-addressed, addressable
// retrieval. Immutability is structural, never procedural: the write
// path has no overwrite branch, and Get verifies content against the
// address — tampering is DETECTED at read (a local operator can still
// rewrite bytes on disk; the trusted-local-operator model applies).
// The store sits OUTSIDE the execution provider boundary — teardown
// code structurally cannot reach it. Durable lifecycle (retention,
// GC) is L6's; v1 retains everything.

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

var ErrPersist = errors.New("artifact-persistence-failed")

type ArtifactStore struct {
	Dir string
}

func NewArtifactStore(dir string) (*ArtifactStore, error) {
	if !filepath.IsAbs(dir) {
		return nil, fmt.Errorf("%w: store dir must be absolute", ErrPersist)
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrPersist, err)
	}
	return &ArtifactStore{Dir: dir}, nil
}

// Put persists one artifact and returns its content address. The
// manifest hash IS the address: a modified artifact is a different
// address, so tampering is unhideable by construction. Write path:
// O_CREAT|O_EXCL (no overwrite path exists in the code), write,
// fsync, then the caller may treat the address as acknowledged.
// An existing file at the address is verified byte-identical
// (content-addressing makes Put idempotent) — anything else is
// corruption, typed.
func (s *ArtifactStore) Put(manifest []byte) (string, error) {
	sum := sha256.Sum256(manifest)
	addr := hex.EncodeToString(sum[:])
	final := filepath.Join(s.Dir, addr+".json")
	f, err := os.OpenFile(final, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o444)
	if err != nil {
		if os.IsExist(err) {
			existing, rerr := os.ReadFile(final)
			if rerr == nil && sha256.Sum256(existing) == sum {
				return addr, nil // same bytes at the same address: already acknowledged
			}
			return "", fmt.Errorf("%w: address %s occupied by different bytes", ErrPersist, addr)
		}
		return "", fmt.Errorf("%w: %v", ErrPersist, err)
	}
	if _, err := f.Write(manifest); err != nil {
		f.Close()
		return "", fmt.Errorf("%w: %v", ErrPersist, err)
	}
	if err := f.Sync(); err != nil {
		f.Close()
		return "", fmt.Errorf("%w: %v", ErrPersist, err)
	}
	if err := f.Close(); err != nil {
		return "", fmt.Errorf("%w: %v", ErrPersist, err)
	}
	return addr, nil
}

// Get retrieves an artifact by address, verifying the content
// against it — a reader can never receive bytes that do not hash to
// the address they asked for.
func (s *ArtifactStore) Get(addr string) ([]byte, error) {
	b, err := os.ReadFile(filepath.Join(s.Dir, addr+".json"))
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrPersist, err)
	}
	sum := sha256.Sum256(b)
	if hex.EncodeToString(sum[:]) != addr {
		return nil, fmt.Errorf("%w: stored bytes do not match address %s", ErrPersist, addr)
	}
	return b, nil
}
