package state

// The read surface (Q-L6-10): typed and structural. Audit/governance
// reads the complete verified record with bytes inseparable from the
// event that carries their provenance and classification; L7 receives
// a StatusView that structurally cannot carry contents (the ItemRef
// move, third use); the model has no L6 read path at all — there is
// no state-plane verb in any registry, and historical evidence
// re-enters model context only through L2 as a future registered,
// classified source. L6 enforces no semantic read authorization:
// persistence, identity, and integrity are L6's; eligibility is the
// consumer boundary's.

import (
	"fmt"
	"path/filepath"
)

// StatusView is the ONLY L7-facing read: lifecycle status, record
// verdict, identity and links, opaque artifact addresses, counts.
// No field can carry event contents or evidence bytes — orchestration
// may consume structural facts for decisions a governed workflow
// explicitly owns, and nothing more.
type StatusView struct {
	TaskID        string
	Status        TaskStatus
	Verdict       Verdict
	RetryOf       string
	ArtifactAddrs []string
	EventCount    int64
	StreamSummary string
}

// ReadStatus derives the structural view cold, verdict included.
func (r *Root) ReadStatus(taskID string) (StatusView, error) {
	man, err := readManifest(r.taskDir(taskID))
	if err != nil {
		return StatusView{}, err
	}
	v := r.Verify(taskID)
	return StatusView{
		TaskID: man.TaskID, Status: man.Status, Verdict: v.Verdict,
		RetryOf: man.RetryOf, ArtifactAddrs: append([]string(nil), man.ArtifactAddrs...),
		EventCount: man.EventCount, StreamSummary: man.StreamSummary,
	}, nil
}

// ReadEvents is the audit read: the complete committed stream, cold,
// integrity-checked. Event bodies carry the emitting layer's
// provenance and classification — the label travels with the record.
func (r *Root) ReadEvents(taskID string) ([]Event, error) {
	parsed, err := parseStream(filepath.Join(r.taskDir(taskID), "events.log"))
	if err != nil {
		return nil, err
	}
	if parsed.Corrupt != "" {
		return nil, fmt.Errorf("%w: %s", ErrCorrupt, parsed.Corrupt)
	}
	return parsed.Events, nil
}

// Resolve returns the bytes one event's reference names, verified
// against the address — reachable only THROUGH an event, so the
// bytes arrive with the event that carries their provenance and
// classification (Q-L6-10: never served naked).
func (r *Root) Resolve(ev Event, refIndex int) ([]byte, error) {
	if refIndex < 0 || refIndex >= len(ev.Refs) {
		return nil, fmt.Errorf("%w: event %d has no reference %d", ErrIdentity, ev.Seq, refIndex)
	}
	return r.store.GetObject(ev.Refs[refIndex])
}

// ReadManifest is the audit read of the projection (a set of claims
// the verifier re-derives — never trust it without Verify).
func (r *Root) ReadManifest(taskID string) (*Manifest, error) {
	return readManifest(r.taskDir(taskID))
}
