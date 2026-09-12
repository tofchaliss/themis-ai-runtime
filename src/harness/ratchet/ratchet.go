// Package ratchet is Layer 11's comparison mechanism: the governed
// Comparison Criterion and Regression Set artifacts, their
// Governance-owned registries, and the deterministic comparative
// machinery over already-committed governed records.
//
// Constitution: openspec/changes/layer-11-ratchet/design.md §2
// (D-L11-1..20, LOCKED 2026-09-12). The load-bearing walls:
//   - L11 establishes comparative facts only; it owns no promotion
//     authority, no security meaning, no baseline authority, and no
//     outcome vocabulary (D-L11-1/4/5/16/18).
//   - This package READS registries; there is deliberately no write
//     API anywhere in it (the L9/L10 wall, third application).
//   - Automation completes governed requests; it never initiates:
//     no timers, watchers, pollers, queues, or background work
//     (D-L11-14), and no operation creates another operation.
//   - L11 output is terminal: packages are never comparator inputs
//     (D-L11-15 Class 4).
package ratchet

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"
)

var (
	ErrCriterion = errors.New("invalid comparison criterion")
	ErrSet       = errors.New("invalid regression set")
	ErrRegistry  = errors.New("invalid ratchet registry")
	ErrResolve   = errors.New("ratchet resolution refused")
)

var (
	shaSyntax  = regexp.MustCompile(`^[0-9a-f]{64}$`)
	nameSyntax = regexp.MustCompile(`^[a-z0-9]+(-[a-z0-9]+)*$`)
	// fieldSyntax names Δ fields and selector slots (snake_case, the
	// L10 slot precedent).
	fieldSyntax   = regexp.MustCompile(`^[a-z0-9]+(_[a-z0-9]+)*$`)
	versionSyntax = regexp.MustCompile(`^[1-9][0-9]*$`)
)

// MaxNameLen bounds registered names so "name@version" stays within
// downstream identity bounds (the L9/L10 precedent).
const MaxNameLen = 128

const (
	maxCriterionBytes = 1 << 20 // 1 MiB
	maxSetBytes       = 1 << 20 // 1 MiB
	maxRegistryBytes  = 4 << 20 // 4 MiB
)

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

// checkNoDuplicateKeys refuses JSON whose objects repeat a key at any
// depth: encoding/json is last-value-wins, which would let reviewed
// text and decoded semantics disagree (the L10 security-review M-1
// lesson, reapplied to criterion review).
func checkNoDuplicateKeys(raw []byte) error {
	dec := json.NewDecoder(strings.NewReader(string(raw)))
	type frame struct {
		object    bool
		keys      map[string]bool
		nextIsKey bool
	}
	var stack []*frame
	for {
		tok, err := dec.Token()
		if err != nil {
			if err.Error() == "EOF" {
				return nil
			}
			return err
		}
		if len(stack) > 0 {
			top := stack[len(stack)-1]
			if top.object && top.nextIsKey {
				if key, ok := tok.(string); ok {
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

// readGoverned reads a governed artifact with a size bound on a single
// handle — no stat-then-reopen window (the L10 close-review L-2/L-3
// lessons, reapplied).
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
	raw, err := io.ReadAll(io.LimitReader(f, maxBytes+1))
	if err != nil {
		return nil, fmt.Errorf("%w: %s: %v", class, path, err)
	}
	if int64(len(raw)) > maxBytes {
		return nil, fmt.Errorf("%w: %s: exceeds %d bytes", class, path, maxBytes)
	}
	return raw, nil
}
