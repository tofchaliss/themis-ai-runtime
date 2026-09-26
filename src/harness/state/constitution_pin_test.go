package state

import "testing"

// The L6 constitution hash is a governed identity (anchor pin
// constitution.state). This pin exists so that a change to the hash is
// always a DELIBERATE act with the new value written here — and so
// that housekeeping (the 2026-09-26 module rename, D-I-2) can prove it
// moved nothing. W-M1 (eventWriters folded in, D-W-4) will change this
// value on purpose and record the old one as historical.
func TestConstitutionHashPinned(t *testing.T) {
	const pinned = "33c6f6c54c2e7b56b33b27384488c03eb217aad0f504402b7ee750c27f22b8db"
	if got := ConstitutionHash(); got != pinned {
		t.Fatalf("L6 constitution hash moved: %s (pinned %s) — a constitution change requires a deliberate re-pin and an anchor consequence", got, pinned)
	}
}
