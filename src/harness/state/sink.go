package state

// The trace sink: one append-only, length-framed, per-entry-hashed
// event stream per task (Q-L6-2/3). The sink — never the caller —
// assigns sequence numbers, at successful commit. Every floor append
// is synchronous: DurableCommit before the emitting layer proceeds
// (D-L6-10; v1 has no write-behind path). A failed append is
// terminal for the stream: the tail state is ambiguous and appending
// past it would corrupt the framing of everything after (Q-L6-3).

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"time"
)

// Event is one durable sink entry. Body is the emitting layer's typed
// event, opaque to L6 (the envelope is L6's; the meaning is the
// layer's). Refs name object identities this event depends on — all
// must be durable at append time. TS is annotation: the host's UTC
// claim at append, never ordering authority, never model-visible.
type Event struct {
	Seq      int64           `json:"seq"`
	Class    string          `json:"class"`
	Writer   string          `json:"writer"`
	TS       string          `json:"ts"`
	Refs     []Ref           `json:"refs,omitempty"`
	Body     json.RawMessage `json:"body"`
	BodyHash string          `json:"body_hash"`
}

// Ref durably binds an object reference to its declared class — the
// {class, provenance} declaration rides the referencing event
// (architecture review 2a): the class is recorded where the reference
// is, and the provenance is the event's writer and body.
type Ref struct {
	ID    string `json:"id"`
	Class string `json:"class"`
}

// maxEventBytes caps a framed entry well under the parser's 9-digit
// frame limit (security review HIGH): an entry that committed but
// could not be re-parsed would be an acknowledged record expelled as
// torn. Evidence payloads belong in the object plane, not in bodies.
const maxEventBytes = 8 << 20

// stream is the single writer for one task's event log.
type stream struct {
	f        *os.File
	next     int64
	summary  [32]byte // running hash over entry hashes
	terminal bool     // failed append: no identity will ever be assigned again
	count    int64
}

func openStream(path string) (*stream, error) {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrStream, err)
	}
	return &stream{f: f}, nil
}

// entryHash is the per-entry content hash (over the canonical entry
// bytes minus the frame), giving corruption localization (Q-L6-3).
func entryHash(b []byte) string {
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}

// sum256Chain is the single stream-summary step: one running hash
// over entry hashes — cross-shape binding without a Merkle/chain
// construction (Q-L6-3).
func sum256Chain(prev [32]byte, bodyHash string) [32]byte {
	return sha256.Sum256(append(prev[:], bodyHash...))
}

// append frames and durably commits one event, assigning its
// sequence. On any failure the stream is terminal, typed.
func (s *stream) append(class, writer string, body json.RawMessage, refs []Ref, store *ObjectStore) (Event, error) {
	if s.terminal {
		return Event{}, fmt.Errorf("%w: stream is terminal after a failed append", ErrStream)
	}
	if !eventClasses[class] {
		return Event{}, fmt.Errorf("%w: unknown event class %q", ErrConstitution, class)
	}
	if !json.Valid(body) {
		return Event{}, fmt.Errorf("%w: event body must be valid JSON", ErrStream)
	}
	// The reference rule enforced at the door: dangles die here, not
	// at the verifier (Q-L6-2).
	for _, r := range refs {
		if !objectClasses[r.Class] {
			return Event{}, fmt.Errorf("%w: reference %q declares unknown class %q", ErrConstitution, r.ID, r.Class)
		}
		if !store.HasObject(r.ID) {
			return Event{}, fmt.Errorf("%w: reference %q is not durable — a durable record may reference only already-durable material", ErrStream, r.ID)
		}
	}
	ev := Event{
		Seq:    s.next,
		Class:  class,
		Writer: writer,
		TS:     time.Now().UTC().Format(time.RFC3339Nano),
		Refs:   refs,
		Body:   body,
	}
	payload, err := json.Marshal(ev)
	if err != nil {
		return Event{}, fmt.Errorf("%w: %v", ErrStream, err)
	}
	ev.BodyHash = entryHash(payload)
	full, err := json.Marshal(ev)
	if err != nil {
		return Event{}, fmt.Errorf("%w: %v", ErrStream, err)
	}
	if len(full) > maxEventBytes {
		// Pre-write refusal: nothing landed, the stream state is
		// unambiguous, so this is typed and NON-terminal.
		return Event{}, fmt.Errorf("%w: event of %d bytes exceeds the %d-byte frame cap — large payloads belong in the object plane", ErrStream, len(full), maxEventBytes)
	}
	frame := []byte(strconv.Itoa(len(full)) + "\n")
	frame = append(frame, full...)
	frame = append(frame, '\n')

	fail := func(cause error) (Event, error) {
		s.terminal = true
		return Event{}, fmt.Errorf("%w: append failed (stream now terminal): %v", ErrStream, cause)
	}
	if err := faultAt("sink.pre-write"); err != nil {
		return fail(err)
	}
	if _, err := s.f.Write(frame); err != nil {
		return fail(err)
	}
	if err := faultAt("sink.pre-fsync"); err != nil {
		return fail(err)
	}
	if err := s.f.Sync(); err != nil {
		return fail(err)
	}
	// DurableCommit succeeded: the identity exists now, not before
	// (Q-L6-3: seq assigned by the sink at successful commit).
	s.next++
	s.count++
	s.summary = sum256Chain(s.summary, ev.BodyHash)
	return ev, nil
}

func (s *stream) close() { _ = s.f.Close() }

// summaryHex is the running stream summary over committed entries.
func (s *stream) summaryHex() string { return hex.EncodeToString(s.summary[:]) }

// parseResult is a cold read of a stream file.
type parseResult struct {
	Events  []Event
	Summary string
	// TornTail holds trailing bytes that do not parse as a complete
	// framed entry — detectable, identity-less, preserved by callers
	// (recovery), never truncated (Q-L6-2).
	TornTail []byte
	// Corrupt names the first integrity violation found in COMMITTED
	// entries (hash mismatch, sequence gap) — distinct from a torn
	// tail, which is a crash artifact.
	Corrupt string
	// CommittedLen is the byte offset of the committed frontier —
	// exactly where sound framing ends.
	CommittedLen int64
}

var framePrefix = regexp.MustCompile(`^\d{1,9}\n`)

// parseStream reads a stream cold, verifying framing, per-entry
// hashes, and sequence contiguity. It never modifies the file.
func parseStream(path string) (parseResult, error) {
	res := parseResult{}
	raw, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return res, nil // no events yet: an empty, valid stream
		}
		return res, fmt.Errorf("%w: %v", ErrStream, err)
	}
	sum := [32]byte{}
	offset := 0
	var wantSeq int64
	for {
		m := framePrefix.Find(raw[offset:])
		if m == nil {
			if offset < len(raw) {
				res.TornTail = raw[offset:]
			}
			break
		}
		n, _ := strconv.Atoi(string(m[:len(m)-1]))
		start := offset + len(m)
		if start+n+1 > len(raw) || raw[start+n] != '\n' {
			res.TornTail = raw[offset:]
			break
		}
		entry := raw[start : start+n]
		var ev Event
		dec := json.NewDecoder(bytes.NewReader(entry))
		dec.DisallowUnknownFields()
		if err := dec.Decode(&ev); err != nil {
			res.TornTail = raw[offset:]
			break
		}
		if dec.More() {
			// Junk between the JSON value and the frame terminator:
			// unproducible by the writer, invisible to every view —
			// corruption, not a crash artifact (security review INFO).
			res.Corrupt = fmt.Sprintf("entry %d carries trailing bytes inside its frame", ev.Seq)
			return res, nil
		}
		// Verify the per-entry hash over the canonical pre-hash form.
		check := ev
		check.BodyHash = ""
		canonical, _ := json.Marshal(check)
		if entryHash(canonical) != ev.BodyHash {
			res.Corrupt = fmt.Sprintf("entry %d fails content verification", ev.Seq)
			return res, nil
		}
		if ev.Seq != wantSeq {
			res.Corrupt = fmt.Sprintf("sequence discontinuity: want %d got %d", wantSeq, ev.Seq)
			return res, nil
		}
		wantSeq++
		sum = sum256Chain(sum, ev.BodyHash)
		res.Events = append(res.Events, ev)
		offset = start + n + 1
		res.CommittedLen = int64(offset)
	}
	res.Summary = hex.EncodeToString(sum[:])
	return res, nil
}

// resumeStream opens a stream for appending after a cold parse,
// restoring the writer position from the committed entries (used by
// recovery, which only ever appends).
func resumeStream(path string, parsed parseResult) (*stream, error) {
	s, err := openStream(path)
	if err != nil {
		return nil, err
	}
	s.next = int64(len(parsed.Events))
	s.count = int64(len(parsed.Events))
	sum := [32]byte{}
	for _, ev := range parsed.Events {
		sum = sum256Chain(sum, ev.BodyHash)
	}
	s.summary = sum
	return s, nil
}
