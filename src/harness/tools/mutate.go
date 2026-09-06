package tools

// Mutating executors (registry-v2, L5-M3 / D-L5-4). Both run behind
// the identical L4 gate — the environment adds containment, never a
// second permission system. All mutation targets pass CreateMode
// confinement (cfn.CreatePath): no symlink anywhere in the
// write path, broad .git* deny-list, parent chain must exist. Results
// record {path, old_hash, new_hash} so the trace can prove exactly
// what changed.

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"strings"

	cfn "github.com/tofchaliss/themis/confine"
)

// ErrWriteRefused joins the bounded executor error vocabulary for the
// mutation era (deliberate vocabulary extension, D-L5-4): the
// authorized mutation could not be applied — confinement refusal,
// missing source, unresolvable parent. Non-topological like its
// Q-L4-8 siblings.
const ErrWriteRefused ErrorClass = "write-refused"

// MutationRecord is the evidence a mutating executor returns: what
// changed, provably. OldHash is empty for creations; NewHash is empty
// for deletions.
type MutationRecord struct {
	Path    string `json:"path"`
	Op      string `json:"op"`
	OldHash string `json:"old_hash,omitempty"`
	NewHash string `json:"new_hash,omitempty"`
	Bytes   int    `json:"bytes"`
}

func hashBytes(b []byte) string {
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}

func execWriteFile(entry *GrantEntry, args map[string]any, target string) Outcome {
	content, _ := args["content"].(string)
	if len(content) > maxToolEvidence {
		return Outcome{ErrClass: ErrOversized}
	}
	abs, err := cfn.CreatePath(entry.Workspace, target)
	if err != nil {
		return Outcome{ErrClass: ErrWriteRefused}
	}
	rec := MutationRecord{Path: target, Op: "write", NewHash: hashBytes([]byte(content)), Bytes: len(content)}
	if old, err := os.ReadFile(abs); err == nil {
		rec.OldHash = hashBytes(old)
	}
	if err := os.WriteFile(abs, []byte(content), 0o644); err != nil {
		return Outcome{ErrClass: ErrWriteRefused}
	}
	ev, _ := json.Marshal(rec)
	return Outcome{Evidence: ev}
}

// patchDoc is the structured patch vocabulary: deterministic,
// strictly parsed, transactional. Unified-diff text is NOT the
// vocabulary — a diff applier's fuzzy matching is exactly the
// nondeterminism a mutation boundary must not have.
type patchDoc struct {
	Ops []patchOp `json:"ops"`
}

type patchOp struct {
	Op      string `json:"op"` // write | delete | rename
	Path    string `json:"path,omitempty"`
	From    string `json:"from,omitempty"`
	To      string `json:"to,omitempty"`
	Content string `json:"content,omitempty"`
}

// resolvedOp is a fully validated operation, ready to apply.
type resolvedOp struct {
	patchOp
	abs, absTo string
}

// execApplyPatch is a transaction (Q-L5-4.4): parse the whole patch →
// validate ALL operations (confinement, existence, deny-list — via
// CreateMode) → apply; a mid-apply failure rolls back from the undo
// journal, so any failure applies NOTHING.
func execApplyPatch(entry *GrantEntry, args map[string]any, target string) Outcome {
	raw, _ := args["patch"].(string)
	if len(raw) > maxToolEvidence {
		return Outcome{ErrClass: ErrOversized}
	}
	dec := json.NewDecoder(strings.NewReader(raw))
	dec.DisallowUnknownFields()
	var doc patchDoc
	if err := dec.Decode(&doc); err != nil || dec.More() || len(doc.Ops) == 0 {
		return Outcome{ErrClass: ErrWriteRefused}
	}

	// Phase 1: validate everything before touching anything.
	resolved := make([]resolvedOp, 0, len(doc.Ops))
	for _, op := range doc.Ops {
		r := resolvedOp{patchOp: op}
		switch op.Op {
		case "write":
			abs, err := cfn.CreatePath(entry.Workspace, op.Path)
			if err != nil {
				return Outcome{ErrClass: ErrWriteRefused}
			}
			r.abs = abs
		case "delete":
			abs, err := cfn.CreatePath(entry.Workspace, op.Path)
			if err != nil {
				return Outcome{ErrClass: ErrWriteRefused}
			}
			if _, err := os.Stat(abs); err != nil {
				return Outcome{ErrClass: ErrWriteRefused} // deleting the nonexistent: whole patch refused
			}
			r.abs = abs
		case "rename":
			from, err := cfn.CreatePath(entry.Workspace, op.From)
			if err != nil {
				return Outcome{ErrClass: ErrWriteRefused}
			}
			if _, err := os.Stat(from); err != nil {
				return Outcome{ErrClass: ErrWriteRefused}
			}
			// Destination is create-mode too: rename-onto-symlink and
			// rename into .git* fail before any mutation.
			to, err := cfn.CreatePath(entry.Workspace, op.To)
			if err != nil {
				return Outcome{ErrClass: ErrWriteRefused}
			}
			r.abs, r.absTo = from, to
		default:
			return Outcome{ErrClass: ErrWriteRefused}
		}
		resolved = append(resolved, r)
	}

	// Phase 2: apply with an undo journal.
	type undo struct {
		path    string
		content []byte // nil = path did not exist
	}
	var journal []undo
	snapshot := func(path string) {
		if b, err := os.ReadFile(path); err == nil {
			journal = append(journal, undo{path, b})
		} else {
			journal = append(journal, undo{path, nil})
		}
	}
	rollback := func() {
		for i := len(journal) - 1; i >= 0; i-- {
			u := journal[i]
			if u.content == nil {
				_ = os.Remove(u.path)
			} else {
				_ = os.WriteFile(u.path, u.content, 0o644)
			}
		}
	}

	var records []MutationRecord
	for _, r := range resolved {
		switch r.Op {
		case "write":
			snapshot(r.abs)
			rec := MutationRecord{Path: r.Path, Op: "write", NewHash: hashBytes([]byte(r.Content)), Bytes: len(r.Content)}
			if old, err := os.ReadFile(r.abs); err == nil {
				rec.OldHash = hashBytes(old)
			}
			if err := os.WriteFile(r.abs, []byte(r.Content), 0o644); err != nil {
				rollback()
				return Outcome{ErrClass: ErrWriteRefused}
			}
			records = append(records, rec)
		case "delete":
			snapshot(r.abs)
			old, err := os.ReadFile(r.abs)
			if err != nil {
				rollback()
				return Outcome{ErrClass: ErrWriteRefused}
			}
			if err := os.Remove(r.abs); err != nil {
				rollback()
				return Outcome{ErrClass: ErrWriteRefused}
			}
			records = append(records, MutationRecord{Path: r.Path, Op: "delete", OldHash: hashBytes(old), Bytes: 0})
		case "rename":
			snapshot(r.abs)
			snapshot(r.absTo)
			old, err := os.ReadFile(r.abs)
			if err != nil {
				rollback()
				return Outcome{ErrClass: ErrWriteRefused}
			}
			if err := os.Rename(r.abs, r.absTo); err != nil {
				rollback()
				return Outcome{ErrClass: ErrWriteRefused}
			}
			records = append(records,
				MutationRecord{Path: r.From, Op: "rename-from", OldHash: hashBytes(old)},
				MutationRecord{Path: r.To, Op: "rename-to", NewHash: hashBytes(old), Bytes: len(old)})
		}
	}
	ev, err := json.Marshal(records)
	if err != nil {
		rollback()
		return Outcome{ErrClass: ErrWriteRefused}
	}
	if len(ev) > maxToolEvidence {
		rollback()
		return Outcome{ErrClass: ErrOversized}
	}
	return Outcome{Evidence: ev}
}
