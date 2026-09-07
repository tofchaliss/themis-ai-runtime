package state

// Cold verification (Q-L6-3/6/9): the verifier re-derives every
// manifest claim from the record — the manifest certifies nothing
// itself. Verdicts are orthogonal to lifecycle and never rewrite
// history. The reachability scan carries a typed completeness
// verdict: anything short of COMPLETE deletes nothing (Q-L6-7 —
// enforced today by having no deletion path at all).

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// VerifyResult is the cold verdict for one task.
type VerifyResult struct {
	TaskID  string
	Verdict Verdict
	Detail  string
}

// Verify re-derives one task's record cold. It writes nothing.
func (r *Root) Verify(taskID string) VerifyResult {
	res := VerifyResult{TaskID: taskID, Verdict: VerdictVerified}
	dir := r.taskDir(taskID)
	parsed, err := parseStream(filepath.Join(dir, "events.log"))
	if err != nil {
		return corrupt(res, err.Error())
	}
	if parsed.Corrupt != "" {
		return corrupt(res, parsed.Corrupt)
	}
	man, err := readManifest(dir)
	if err != nil {
		return corrupt(res, "manifest unreadable: "+err.Error())
	}
	if man.TaskID != taskID {
		return corrupt(res, "manifest identity mismatch")
	}
	if man.ConstitutionHash == "" {
		return corrupt(res, "manifest lacks constitution hash")
	}
	// Lifecycle-sequence legality (test review HIGH): the recorded
	// decisions themselves must walk the closed machine — the first
	// lifecycle event is CREATED and every successor follows legalNext.
	var prev TaskStatus
	var seen []TaskStatus
	for _, ev := range parsed.Events {
		if ev.Class != EvLifecycle {
			continue
		}
		var body struct {
			To string `json:"to"`
		}
		if json.Unmarshal(ev.Body, &body) != nil || body.To == "" {
			return corrupt(res, fmt.Sprintf("entry %d: malformed lifecycle event", ev.Seq))
		}
		to := TaskStatus(body.To)
		if prev == "" {
			if to != StatusCreated {
				return corrupt(res, "illegal lifecycle sequence: record does not begin at CREATED")
			}
		} else if !legalNext[prev][to] {
			return corrupt(res, fmt.Sprintf("illegal lifecycle sequence: %s -> %s", prev, to))
		}
		prev = to
		seen = append(seen, to)
	}
	// Projection legality (Q-L6-6): a manifest state unexplainable by
	// a prior durable lifecycle event is itself evidence of
	// corruption. A manifest BEHIND the stream (stale projection,
	// recovery pending) is legal; a manifest AHEAD of it is not.
	explained := false
	for _, st := range seen {
		if st == man.Status {
			explained = true
		}
	}
	if !explained {
		return corrupt(res, fmt.Sprintf("manifest state %s has no supporting lifecycle event", man.Status))
	}
	// Attribution is a projection of the CREATED event (architecture
	// review 6b): re-derive governed hashes, retry_of, and the
	// constitution hash from the record.
	if created := createdEventBody(parsed.Events); created != nil {
		if created.ConstitutionHash != man.ConstitutionHash {
			return corrupt(res, "manifest constitution hash diverges from the CREATED event")
		}
		if created.RetryOf != man.RetryOf {
			return corrupt(res, "manifest retry_of diverges from the CREATED event")
		}
		if !mapsEqual(created.GovernedHashes, man.GovernedHashes) {
			return corrupt(res, "manifest governed hashes diverge from the CREATED event")
		}
	}
	// FAILED_PARTIAL requires recovery's own committed evidence.
	if man.Status == StatusFailedPartial {
		found := false
		for _, ev := range parsed.Events {
			if ev.Class == EvRecovery {
				found = true
			}
		}
		if !found {
			return corrupt(res, "FAILED_PARTIAL without a recovery event")
		}
	}
	// Terminal binding: count + summary over entries [0, EventCount).
	// Bounds first — a forged count must yield a verdict, never a
	// verifier panic (security review MED).
	if terminal(man.Status) {
		if man.EventCount < 0 || man.EventCount > int64(len(parsed.Events)) {
			return corrupt(res, "manifest claims more events than the record holds")
		}
		if summaryOver(parsed.Events[:man.EventCount]) != man.StreamSummary {
			return corrupt(res, "stream summary mismatch")
		}
	}
	// Every reference resolves and verifies: dangles are lies.
	var boundArtifacts []string
	for _, ev := range parsed.Events {
		for _, ref := range ev.Refs {
			if _, err := r.store.GetObject(ref.ID); err != nil {
				return corrupt(res, fmt.Sprintf("event %d: reference %s: %v", ev.Seq, ref.ID, err))
			}
		}
		if ev.Class == EvArtifact && len(ev.Refs) == 1 {
			boundArtifacts = append(boundArtifacts, ev.Refs[0].ID)
		}
	}
	// The manifest's artifact list is a pure projection of the
	// artifact-bound events — re-derived, never trusted (architecture
	// review 3A; BindArtifact enforces present state-store objects at
	// the door, so every address must resolve here).
	if len(man.ArtifactAddrs) != len(boundArtifacts) {
		return corrupt(res, "manifest artifact projection diverges from the stream")
	}
	for i, addr := range man.ArtifactAddrs {
		if addr != boundArtifacts[i] {
			return corrupt(res, "manifest artifact projection diverges from the stream")
		}
		if _, err := r.store.GetObject(addr); err != nil {
			return corrupt(res, "manifest artifact address unresolvable: "+addr)
		}
	}
	if hasTornArtifacts(dir) || len(parsed.TornTail) > 0 {
		res.Verdict = VerdictTorn
		res.Detail = "torn tail present (preserved)"
	}
	return res
}

func corrupt(res VerifyResult, detail string) VerifyResult {
	res.Verdict = VerdictCorrupt
	res.Detail = detail
	return res
}

// createdAttribution is the CREATED event's projected attribution.
type createdAttribution struct {
	RetryOf          string            `json:"retry_of"`
	GovernedHashes   map[string]string `json:"governed_hashes"`
	ConstitutionHash string            `json:"constitution_hash"`
}

func createdEventBody(events []Event) *createdAttribution {
	for _, ev := range events {
		if ev.Class != EvLifecycle {
			continue
		}
		var b struct {
			To string `json:"to"`
			createdAttribution
		}
		if json.Unmarshal(ev.Body, &b) == nil && TaskStatus(b.To) == StatusCreated {
			return &b.createdAttribution
		}
	}
	return nil
}

func mapsEqual(a, b map[string]string) bool {
	if len(a) != len(b) {
		return false
	}
	for k, v := range a {
		if b[k] != v {
			return false
		}
	}
	return true
}

func summaryOver(events []Event) string {
	sum := [32]byte{}
	for _, ev := range events {
		sum = sum256Chain(sum, ev.BodyHash)
	}
	return hex.EncodeToString(sum[:])
}

func hasTornArtifacts(dir string) bool {
	matches, _ := filepath.Glob(filepath.Join(dir, "events.torn.*"))
	return len(matches) > 0
}

// ScanResult is the reachability scan with its typed completeness
// verdict (Q-L6-7): COMPLETE permits candidate evaluation; anything
// else deletes nothing. No probabilistic GC.
type ScanResult struct {
	Complete  bool
	Reason    string // why incomplete, when it is
	Reachable map[string]bool
	Tasks     []string
}

// ScanReachable traverses every task's manifest and stream, following
// object references. Any unreadable, torn, or corrupt record renders
// the scan INCOMPLETE — and pins everything by doing so.
func (r *Root) ScanReachable() ScanResult {
	res := ScanResult{Complete: true, Reachable: map[string]bool{}}
	entries, err := os.ReadDir(filepath.Join(r.dir, "tasks"))
	if err != nil {
		return ScanResult{Complete: false, Reason: "task root unreadable: " + err.Error()}
	}
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		id := e.Name()
		res.Tasks = append(res.Tasks, id)
		v := r.Verify(id)
		if v.Verdict != VerdictVerified {
			// Torn or corrupt: uncertainty renders the scan
			// INCOMPLETE and thereby pins everything (Q-L6-7). We
			// still collect what we CAN see, but the verdict stands.
			res.Complete = false
			res.Reason = fmt.Sprintf("task %s: %s: %s", id, v.Verdict, v.Detail)
		}
		parsed, perr := parseStream(filepath.Join(r.taskDir(id), "events.log"))
		if perr != nil {
			res.Complete = false
			res.Reason = fmt.Sprintf("task %s: %v", id, perr)
			continue
		}
		for _, ev := range parsed.Events {
			for _, ref := range ev.Refs {
				res.Reachable[ref.ID] = true
			}
		}
		if man, merr := readManifest(r.taskDir(id)); merr == nil {
			for _, a := range man.ArtifactAddrs {
				res.Reachable[a] = true
			}
		}
	}
	return res
}
