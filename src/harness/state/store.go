package state

// The L6 object store: content-addressed, immutable, write-once, with
// the crash-safe publication discipline (D-L6-11): the final address
// is only ever ABSENT (retryable) or COMPLETE (verifiable) — never a
// durable partial. Objects are raw payload bytes; the {class,
// provenance} declaration is durably recorded in the event that
// references the object (an unreferenced object is an orphan by
// definition), so identical bytes produced by different tasks dedup
// to one object without a provenance conflict.

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// fault is the deterministic fault-injection seam for Register C
// (Q-L6-9). Nil in production; tests set it to fail at named points.
var fault func(point string) error

func faultAt(point string) error {
	if fault != nil {
		return fault(point)
	}
	return nil
}

func fsyncDir(dir string) error {
	d, err := os.Open(dir)
	if err != nil {
		return err
	}
	defer d.Close()
	return d.Sync()
}

// ObjectStore is rooted at <root>/objects with a staging area at
// <root>/objects/tmp.
// Recorded residual (security review LOW): crashed staging temps
// accumulate in objects/tmp — pure bounded-by-crashes waste, never
// re-read, never re-linked, never poisoning an address. The natural
// cleanup is a future quiescent startup sweep alongside GC.
type ObjectStore struct {
	dir string
	tmp string
}

func newObjectStore(root string) (*ObjectStore, error) {
	dir := filepath.Join(root, "objects")
	tmp := filepath.Join(dir, "tmp")
	if err := os.MkdirAll(tmp, 0o755); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrPersist, err)
	}
	return &ObjectStore{dir: dir, tmp: tmp}, nil
}

// path derives the storage location from the identity — never the
// reverse (Q-L6-3): objects/sha256/<first2>/<hex>.
func (s *ObjectStore) path(id string) (string, error) {
	algo, hex, ok := strings.Cut(id, ":")
	if !ok || algo != hashAlgo || len(hex) != 64 {
		return "", fmt.Errorf("%w: unknown or malformed object id %q", ErrIdentity, id)
	}
	return filepath.Join(s.dir, algo, hex[:2], hex), nil
}

// StoreObject persists one declarable object and returns its
// identity. The class must come from the closed vocabulary; there is
// no Update and no Delete — this is the only object mutation
// primitive (Q-L6-4). Publication sequence (D-L6-11):
//
//	write temp → fsync temp → atomic no-replace link → fsync
//	directory → remove temp
//
// Idempotent: same bytes at an occupied address acknowledge; an
// occupied address holding different bytes is corruption, typed.
func (s *ObjectStore) StoreObject(class string, bytes []byte) (string, error) {
	if !objectClasses[class] {
		return "", fmt.Errorf("%w: unknown object class %q", ErrConstitution, class)
	}
	id := objectID(bytes)
	final, err := s.path(id)
	if err != nil {
		return "", err
	}
	if existing, err := os.ReadFile(final); err == nil {
		// Occupied: complete by construction (D-L6-11) — verify. The
		// dirsync makes THIS caller's ack independently durable: an
		// ack must never precede durability, even when another writer
		// did the publication (security review MED).
		if objectID(existing) == id {
			if err := fsyncDir(filepath.Dir(final)); err != nil {
				return "", fmt.Errorf("%w: %v", ErrPersist, err)
			}
			return id, nil // idempotent re-write of identical bytes
		}
		return "", fmt.Errorf("%w: address %s occupied by different bytes", ErrCorrupt, id)
	}
	if err := os.MkdirAll(filepath.Dir(final), 0o755); err != nil {
		return "", fmt.Errorf("%w: %v", ErrPersist, err)
	}
	tmp, err := os.CreateTemp(s.tmp, "obj-")
	if err != nil {
		return "", fmt.Errorf("%w: %v", ErrPersist, err)
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName) // no-op after successful publication cleanup
	if err := faultAt("object.write"); err != nil {
		tmp.Close()
		return "", fmt.Errorf("%w: %v", ErrPersist, err)
	}
	if _, err := tmp.Write(bytes); err != nil {
		tmp.Close()
		return "", fmt.Errorf("%w: %v", ErrPersist, err)
	}
	if err := faultAt("object.pre-fsync"); err != nil {
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
	if err := faultAt("object.pre-link"); err != nil {
		return "", fmt.Errorf("%w: %v", ErrPersist, err)
	}
	// Atomic no-replace publication: link fails if the address is
	// occupied (a concurrent identical write — verify and accept).
	if err := os.Link(tmpName, final); err != nil {
		if os.IsExist(err) {
			if existing, rerr := os.ReadFile(final); rerr == nil && objectID(existing) == id {
				if serr := fsyncDir(filepath.Dir(final)); serr != nil {
					return "", fmt.Errorf("%w: %v", ErrPersist, serr)
				}
				return id, nil
			}
			return "", fmt.Errorf("%w: address %s occupied by different bytes", ErrCorrupt, id)
		}
		return "", fmt.Errorf("%w: %v", ErrPersist, err)
	}
	if err := faultAt("object.pre-dirsync"); err != nil {
		return "", fmt.Errorf("%w: %v", ErrPersist, err)
	}
	if err := fsyncDir(filepath.Dir(final)); err != nil {
		return "", fmt.Errorf("%w: %v", ErrPersist, err)
	}
	return id, nil
}

// GetObject retrieves bytes by identity, verifying content against
// the address — a reader can never receive bytes that do not hash to
// the identity it asked for (Q-L6-3 verify-on-read).
func (s *ObjectStore) GetObject(id string) ([]byte, error) {
	p, err := s.path(id)
	if err != nil {
		return nil, err
	}
	b, err := os.ReadFile(p)
	if err != nil {
		return nil, fmt.Errorf("%w: object %s: %v", ErrPersist, id, err)
	}
	if objectID(b) != id {
		return nil, fmt.Errorf("%w: object %s fails content verification", ErrCorrupt, id)
	}
	return b, nil
}

// HasObject reports durable presence of a published address — used
// by the sink's reference check (published = complete, D-L6-11).
func (s *ObjectStore) HasObject(id string) bool {
	p, err := s.path(id)
	if err != nil {
		return false
	}
	info, err := os.Stat(p)
	return err == nil && info.Mode().IsRegular()
}
