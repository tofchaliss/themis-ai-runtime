package state

// TaskRecord is the live handle for one task's durable state — the
// three mutation primitives (Q-L6-4) behind one ordering discipline:
// decision → lifecycle event → DurableCommit → manifest projection →
// DurableCommit (Q-L6-6). There is no other write path.

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

// Root is one durable state root: a global content-addressed object
// store plus per-task streams and manifests.
type Root struct {
	dir   string
	store *ObjectStore

	// live tracks in-process writers so recovery can refuse to race
	// one (security review MED). Cross-process exclusion is a
	// recorded residual of the single-process v1 model.
	liveMu sync.Mutex
	live   map[string]bool
}

// OpenRoot prepares a state root. The root must be absolute; its
// disjointness from workspace/mirror/provider/store roots is the task
// assembler's obligation (CheckDisjointRoots), enforced fail-closed
// there — the one place all roots are known.
func OpenRoot(dir string) (*Root, error) {
	if !filepath.IsAbs(dir) {
		return nil, fmt.Errorf("%w: state root must be absolute", ErrPersist)
	}
	if err := os.MkdirAll(filepath.Join(dir, "tasks"), 0o755); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrPersist, err)
	}
	store, err := newObjectStore(dir)
	if err != nil {
		return nil, err
	}
	return &Root{dir: dir, store: store, live: map[string]bool{}}, nil
}

// Store exposes the object store (its only mutation is StoreObject).
func (r *Root) Store() *ObjectStore { return r.store }

func (r *Root) taskDir(id string) string { return filepath.Join(r.dir, "tasks", id) }

func (r *Root) isLive(id string) bool {
	r.liveMu.Lock()
	defer r.liveMu.Unlock()
	return r.live[id]
}

func (r *Root) setLive(id string, v bool) {
	r.liveMu.Lock()
	defer r.liveMu.Unlock()
	if v {
		r.live[id] = true
	} else {
		delete(r.live, id)
	}
}

// TaskOptions carries the only caller-suppliable manifest facts.
type TaskOptions struct {
	RetryOf        string
	GovernedHashes map[string]string
}

// TaskRecord is the single writer for one task's stream + manifest.
type TaskRecord struct {
	mu     sync.Mutex
	root   *Root
	dir    string
	stream *stream
	man    *Manifest
}

// CreateTask mints a single-use task identity: the task directory is
// created exclusively — an existing identity refuses, forever
// (Q-L6-3: no reuse, no resurrection). Re-runs are new identities,
// linked via retry_of.
func (r *Root) CreateTask(id string, opts TaskOptions) (*TaskRecord, error) {
	if !taskIDSyntax(id) {
		return nil, fmt.Errorf("%w: bad task id %q", ErrIdentity, id)
	}
	dir := r.taskDir(id)
	if err := os.Mkdir(dir, 0o755); err != nil {
		if os.IsExist(err) {
			return nil, fmt.Errorf("%w: task id %q already exists — identities are single-use", ErrIdentity, id)
		}
		return nil, fmt.Errorf("%w: %v", ErrPersist, err)
	}
	st, err := openStream(filepath.Join(dir, "events.log"))
	if err != nil {
		return nil, err
	}
	t := &TaskRecord{root: r, dir: dir, stream: st, man: &Manifest{
		TaskID: id, Status: StatusCreated, ConstitutionHash: ConstitutionHash(),
		RetryOf: opts.RetryOf, GovernedHashes: opts.GovernedHashes,
	}}
	// Manifest-first: a crashed task keeps attribution (D-L6-2). The
	// CREATED projection is preceded by its lifecycle event like every
	// other transition.
	if _, err := t.appendLocked(EvLifecycle, "l6", createdBody(opts), nil); err != nil {
		return nil, err
	}
	if err := writeManifest(dir, t.man); err != nil {
		return nil, err
	}
	r.setLive(id, true)
	return t, nil
}

func lifecycleBody(to TaskStatus, reason string) json.RawMessage {
	b, _ := json.Marshal(map[string]string{"to": string(to), "reason": reason})
	return b
}

// createdBody carries the caller-supplied manifest facts INTO the
// record, so recovery projects them instead of inventing (architecture
// review 6b: zero origination extends to attribution).
func createdBody(opts TaskOptions) json.RawMessage {
	b, _ := json.Marshal(map[string]any{
		"to": string(StatusCreated), "reason": "created",
		"retry_of": opts.RetryOf, "governed_hashes": opts.GovernedHashes,
		"constitution_hash": ConstitutionHash(),
	})
	return b
}

// AppendEvent is the caller-facing sink primitive: envelope validated
// (class in the closed vocabulary, refs durable), content owned by
// the emitting layer, sequence assigned by the sink, synchronously
// committed (D-L6-10). Recovery-only classes refuse.
func (t *TaskRecord) AppendEvent(class, writer string, body json.RawMessage, refs ...Ref) (Event, error) {
	if primitiveOnlyEvents[class] {
		return Event{}, fmt.Errorf("%w: event class %q is recovery-owned or primitive-owned", ErrConstitution, class)
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	if terminal(t.man.Status) {
		return Event{}, fmt.Errorf("%w: task %s is terminal", ErrLifecycle, t.man.TaskID)
	}
	return t.appendLocked(class, writer, body, refs)
}

func (t *TaskRecord) appendLocked(class, writer string, body json.RawMessage, refs []Ref) (Event, error) {
	ev, err := t.stream.append(class, writer, body, refs, t.root.store)
	if err != nil {
		return Event{}, err
	}
	// Flag-only defense in depth (D-L6-1): a secret-shaped body is
	// recorded as suspected contamination — never edited, never
	// refused here; the authoritative removal boundary is upstream.
	if class != EvContamination && suspectSecret(body) {
		cb, _ := json.Marshal(map[string]int64{"suspect_seq": ev.Seq})
		if _, cerr := t.stream.append(EvContamination, "l6", cb, nil, t.root.store); cerr != nil {
			return Event{}, cerr
		}
	}
	return ev, nil
}

// StoreObject is the task-scoped door to the object plane; the
// {class, provenance} declaration becomes durable in the event that
// references the returned identity.
func (t *TaskRecord) StoreObject(class string, bytes []byte) (string, error) {
	return t.root.store.StoreObject(class, bytes)
}

// Transition drives the closed lifecycle machine: caller-requestable
// states only (FAILED_PARTIAL is recovery-owned), lifecycle event
// durably committed BEFORE the manifest projection (Q-L6-6), terminal
// transitions bind event count + stream summary (Q-L6-3).
func (t *TaskRecord) Transition(to TaskStatus, reason string) error {
	if !callerRequestable[to] {
		return fmt.Errorf("%w: status %q is not caller-requestable", ErrLifecycle, to)
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.transitionLocked(to, reason)
}

func (t *TaskRecord) transitionLocked(to TaskStatus, reason string) error {
	if !legalNext[t.man.Status][to] {
		return fmt.Errorf("%w: %s -> %s (%s)", ErrLifecycle, t.man.Status, to, reason)
	}
	if _, err := t.appendLocked(EvLifecycle, "l6", lifecycleBody(to, reason), nil); err != nil {
		return err
	}
	t.man.Status = to
	if terminal(to) {
		t.man.EventCount = t.stream.count
		t.man.StreamSummary = t.stream.summaryHex()
	}
	return writeManifest(t.dir, t.man)
}

// BindArtifact records an acknowledged artifact on the manifest.
// The reference rule applies at THIS door too (security review): the
// address must be a present state-store object, and the binding is
// recorded as an event BEFORE the manifest projection — the manifest
// stays a pure projection of the stream (architecture review 3A).
func (t *TaskRecord) BindArtifact(addr string) error {
	if !t.root.store.HasObject(addr) {
		return fmt.Errorf("%w: artifact %q is not a durable state-store object — a durable record may reference only already-durable material", ErrStream, addr)
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	if terminal(t.man.Status) {
		return fmt.Errorf("%w: task %s is terminal", ErrLifecycle, t.man.TaskID)
	}
	body, _ := json.Marshal(map[string]string{"artifact": addr})
	if _, err := t.appendLocked(EvArtifact, "l6", body, []Ref{{ID: addr, Class: ObjEgressArtifact}}); err != nil {
		return err
	}
	t.man.ArtifactAddrs = append(t.man.ArtifactAddrs, addr)
	return writeManifest(t.dir, t.man)
}

// Close releases the stream handle (no durability implication: every
// append already committed synchronously) and deregisters the live
// writer.
func (t *TaskRecord) Close() {
	t.stream.close()
	t.root.setLive(t.man.TaskID, false)
}

// suspectSecret is the flag-only pattern family (DAY-0 §3 shape —
// deliberately mirrored from the L1 loader's family; drift between
// the two lists affects flagging only, never authority).
func suspectSecret(b []byte) bool {
	s := string(b)
	for _, marker := range []string{"-----BEGIN ", "AKIA", "ghp_", "gho_", "github_pat_", "xoxb-", "xoxp-"} {
		if strings.Contains(s, marker) {
			return true
		}
	}
	return false
}

// CheckDisjointRoots refuses nested roots — the task assembler calls
// it with every root it wires (state, artifact store, mirrors,
// provider base, every grant workspace); pairwise non-nesting, both
// directions (Q-L6-4).
func CheckDisjointRoots(roots ...string) error {
	if len(roots) < 2 {
		return fmt.Errorf("%w: disjointness needs at least two roots", ErrIdentity)
	}
	sep := string(filepath.Separator)
	resolved := make([]string, len(roots))
	for i, root := range roots {
		if !filepath.IsAbs(root) {
			return fmt.Errorf("%w: root %q must be absolute", ErrIdentity, root)
		}
		// Physical, case-folded comparison (security review): lexical
		// checks miss symlinked and case-aliased nesting; resolution
		// failure fails closed.
		rp, err := filepath.EvalSymlinks(root)
		if err != nil {
			return fmt.Errorf("%w: root %q: %v", ErrIdentity, root, err)
		}
		resolved[i] = strings.ToLower(filepath.Clean(rp))
	}
	for i := 0; i < len(resolved); i++ {
		for j := i + 1; j < len(resolved); j++ {
			a, b := resolved[i], resolved[j]
			if a == b || strings.HasPrefix(a+sep, b+sep) || strings.HasPrefix(b+sep, a+sep) {
				return fmt.Errorf("%w: roots %q and %q are nested", ErrIdentity, roots[i], roots[j])
			}
		}
	}
	return nil
}
