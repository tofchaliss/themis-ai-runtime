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
// address, so tampering is unhideable by construction. The write
// path inherits the L6 crash-safe publication discipline (D-L6-11 —
// the direct-at-final-address write had a crash window that left a
// durable partial permanently poisoning its address):
//
//	write temp → fsync temp → atomic no-replace link → fsync
//	directory → remove temp
//
// so an address is only ever ABSENT (retryable) or COMPLETE
// (verifiable). An existing file at the address is verified
// byte-identical (content-addressing makes Put idempotent) —
// anything else is corruption, typed.
func (s *ArtifactStore) Put(manifest []byte) (string, error) {
	sum := sha256.Sum256(manifest)
	addr := hex.EncodeToString(sum[:])
	final := filepath.Join(s.Dir, addr+".json")
	ackExisting := func() (string, error) {
		existing, rerr := os.ReadFile(final)
		if rerr == nil && sha256.Sum256(existing) == sum {
			// The dirsync makes THIS caller's ack independently
			// durable even when another writer published.
			if derr := fsyncDir(s.Dir); derr != nil {
				return "", fmt.Errorf("%w: %v", ErrPersist, derr)
			}
			return addr, nil
		}
		return "", fmt.Errorf("%w: address %s occupied by different bytes", ErrPersist, addr)
	}
	if _, err := os.Stat(final); err == nil {
		return ackExisting()
	}
	tmp, err := os.CreateTemp(s.Dir, "staging-")
	if err != nil {
		return "", fmt.Errorf("%w: %v", ErrPersist, err)
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)
	if _, err := tmp.Write(manifest); err != nil {
		tmp.Close()
		return "", fmt.Errorf("%w: %v", ErrPersist, err)
	}
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		return "", fmt.Errorf("%w: %v", ErrPersist, err)
	}
	if err := tmp.Close(); err != nil {
		return "", fmt.Errorf("%w: %v", ErrPersist, err)
	}
	if err := os.Chmod(tmpName, 0o444); err != nil {
		return "", fmt.Errorf("%w: %v", ErrPersist, err)
	}
	if err := os.Link(tmpName, final); err != nil {
		if os.IsExist(err) {
			return ackExisting()
		}
		return "", fmt.Errorf("%w: %v", ErrPersist, err)
	}
	if err := fsyncDir(s.Dir); err != nil {
		return "", fmt.Errorf("%w: %v", ErrPersist, err)
	}
	return addr, nil
}

func fsyncDir(dir string) error {
	d, err := os.Open(dir)
	if err != nil {
		return err
	}
	defer d.Close()
	return d.Sync()
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
