package state

// Recovery (Q-L6-6): projection authority (it may finish projecting
// decisions the record already contains, saying so in an appended
// reconciliation event), structural-fact authority (FAILED_PARTIAL,
// committed as recovery's own event before projection), and zero
// origination authority (it never derives a semantic outcome from
// activity). Recovery only appends; it never rewrites the record.
// Torn tails are preserved aside, never truncated.

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// RecoveryOutcome reports what recovery established.
type RecoveryOutcome struct {
	TaskID     string
	Status     TaskStatus
	Acted      bool   // false = clean pass, nothing appended (idempotence)
	TornSaved  string // path of the preserved tail, if any
	Reconciled bool   // a durably recorded decision was projected
}

// Recover examines one task cold and drives it to a consistent,
// typed state. Idempotent in state effect: a second pass over a
// recovered task appends nothing.
func (r *Root) Recover(taskID string) (RecoveryOutcome, error) {
	out := RecoveryOutcome{TaskID: taskID}
	dir := r.taskDir(taskID)
	streamPath := filepath.Join(dir, "events.log")
	parsed, err := parseStream(streamPath)
	if err != nil {
		return out, err
	}
	if parsed.Corrupt != "" {
		// Committed-entry corruption is a verdict, never a lifecycle
		// edit (Q-L6-6): recovery does not touch a record it cannot
		// trust; Verify reports it.
		return out, fmt.Errorf("%w: %s: %s", ErrCorrupt, taskID, parsed.Corrupt)
	}
	// Refuse a record that races a live writer (security review MED):
	// two appenders would manufacture durable sequence corruption. v1
	// is single-process; the in-process registry is the guard, and
	// the cross-process case is a recorded residual.
	if r.isLive(taskID) {
		return out, fmt.Errorf("%w: task %s has a live writer — recovery is a cold operation", ErrStream, taskID)
	}

	// Preserve a torn tail aside — identity-less crash bytes, kept
	// for audit, removed from the framing path (Q-L6-2).
	if len(parsed.TornTail) > 0 {
		saved := filepath.Join(dir, fmt.Sprintf("events.torn.%d", len(parsed.Events)))
		if _, err := os.Stat(saved); os.IsNotExist(err) {
			if err := os.WriteFile(saved, parsed.TornTail, 0o400); err != nil {
				return out, fmt.Errorf("%w: preserving torn tail: %v", ErrPersist, err)
			}
			if err := fsyncDir(dir); err != nil {
				return out, fmt.Errorf("%w: %v", ErrPersist, err)
			}
		}
		// Relocate, don't destroy: with the bytes preserved aside
		// (above, fsync'd first), the stream file is cut back to the
		// committed frontier so appends resume on sound framing. The
		// committed record — entries [0, n) — survives byte-identical;
		// the torn bytes' only home moves, their content never
		// changes. This is the resolution of the append-after-torn
		// invariant conflict: live-process appends after an ambiguous
		// tail stay terminal (Q-L6-3); cold recovery knows the exact
		// frontier and may resume behind it.
		if err := faultAt("recovery.pre-truncate"); err != nil {
			return out, fmt.Errorf("%w: %v", ErrPersist, err)
		}
		if err := os.Truncate(streamPath, parsed.CommittedLen); err != nil {
			return out, fmt.Errorf("%w: %v", ErrPersist, err)
		}
		if err := faultAt("recovery.post-truncate"); err != nil {
			return out, fmt.Errorf("%w: %v", ErrPersist, err)
		}
		out.TornSaved = saved
	}

	// The manifest is a projection; derive the record's own view.
	lastLifecycle, lifecycleFound := lastLifecycleEvent(parsed.Events)
	var man *Manifest
	if _, statErr := os.Stat(filepath.Join(dir, "manifest.json")); os.IsNotExist(statErr) {
		// Creation crashed between the CREATED event and its
		// projection: project what the record contains — but only if
		// the record actually contains the creation (recovery never
		// creates a stream CreateTask didn't).
		if len(parsed.Events) == 0 {
			return out, fmt.Errorf("%w: %s: no manifest and no committed events — nothing to recover", ErrCorrupt, taskID)
		}
		// Project attribution from the CREATED event — recovery never
		// invents what the record already carries (architecture review
		// 6b: zero origination extends to attribution).
		created := createdEventBody(parsed.Events)
		if created == nil {
			return out, fmt.Errorf("%w: %s: record does not begin at CREATED", ErrCorrupt, taskID)
		}
		man = &Manifest{TaskID: taskID, Status: StatusCreated,
			ConstitutionHash: created.ConstitutionHash, RetryOf: created.RetryOf,
			GovernedHashes: created.GovernedHashes}
	} else {
		var manErr error
		man, manErr = readManifest(dir)
		if manErr != nil {
			return out, manErr
		}
		// Projection-legality gate (security review HIGH): recovery
		// must never launder a record Verify would call corrupt. An
		// existing manifest state must be explained by a prior durable
		// lifecycle event — a manifest ahead of its stream (stream
		// loss, replacement with a shorter prefix) is corruption, and
		// recovery refuses to touch it.
		if !statusExplained(man.Status, parsed.Events) {
			return out, fmt.Errorf("%w: %s: manifest state %s has no supporting lifecycle event — recovery refuses", ErrCorrupt, taskID, man.Status)
		}
	}

	st, err := resumeStream(streamPath, parsed)
	if err != nil {
		return out, err
	}
	defer st.close()
	appendRecovery := func(kind string, detail map[string]any) error {
		body := map[string]any{"kind": kind}
		for k, v := range detail {
			body[k] = v
		}
		raw, _ := json.Marshal(body)
		_, err := st.append(EvRecovery, "l6-recovery", raw, nil, r.store)
		return err
	}

	// Case 1: the record holds a terminal decision the manifest never
	// projected — projection authority: finish it, say so.
	if lifecycleFound && terminal(lastLifecycle) && man.Status != lastLifecycle {
		if err := appendRecovery("projection-reconciled", map[string]any{"projected": string(lastLifecycle)}); err != nil {
			return out, err
		}
		man.Status = lastLifecycle
		man.EventCount = int64(len(parsed.Events))
		man.StreamSummary = parsed.Summary
		if err := writeManifest(dir, man); err != nil {
			return out, err
		}
		out.Status, out.Acted, out.Reconciled = lastLifecycle, true, true
		return out, nil
	}

	// Case 2: already consistent and terminal — clean idempotent pass.
	if terminal(man.Status) {
		out.Status = man.Status
		out.Acted = len(parsed.TornTail) > 0
		return out, nil
	}

	// Case 3: the record stopped without a terminal decision —
	// structural fact, recovery-owned: FAILED_PARTIAL, event first.
	if err := appendRecovery("failed-partial", map[string]any{"committed_events": len(parsed.Events)}); err != nil {
		return out, err
	}
	if _, err := st.append(EvLifecycle, "l6-recovery", lifecycleBody(StatusFailedPartial, "recovery: record not cleanly completed"), nil, r.store); err != nil {
		return out, err
	}
	man.Status = StatusFailedPartial
	man.EventCount = st.count
	man.StreamSummary = st.summaryHex()
	if err := writeManifest(dir, man); err != nil {
		return out, err
	}
	out.Status, out.Acted = StatusFailedPartial, true
	return out, nil
}

// statusExplained reports whether a manifest state is supported by a
// prior durable lifecycle event — the same rule the verifier applies
// (Q-L6-6: a manifest state unexplainable by the stream is itself
// evidence of corruption).
func statusExplained(status TaskStatus, events []Event) bool {
	for _, ev := range events {
		if ev.Class != EvLifecycle {
			continue
		}
		var b struct {
			To string `json:"to"`
		}
		if json.Unmarshal(ev.Body, &b) == nil && TaskStatus(b.To) == status {
			return true
		}
	}
	return false
}

func lastLifecycleEvent(events []Event) (TaskStatus, bool) {
	for i := len(events) - 1; i >= 0; i-- {
		if events[i].Class != EvLifecycle {
			continue
		}
		var body struct {
			To string `json:"to"`
		}
		if json.Unmarshal(events[i].Body, &body) == nil && body.To != "" {
			return TaskStatus(body.To), true
		}
	}
	return "", false
}

