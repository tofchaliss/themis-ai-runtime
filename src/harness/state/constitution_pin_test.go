package state

import "testing"

// The L6 constitution hash is a governed identity (anchor pin
// constitution.state). This pin exists so that a change to the hash is
// always a DELIBERATE act with the new value written here.
//
// History:
//
//	33c6f6c54c2e7b56b33b27384488c03eb217aad0f504402b7ee750c27f22b8db
//	  — rsys@3..@5 (until 2026-09-26): classes, objects, edges, algo,
//	    retention, commit mode.
//	1df0e28548a48b373e779dce8f00ea906c7f98afc009d573afccdf1009d77685
//	  — W-M1 (D-W-4/D-W-6, 2026-09-26): the closed class→writer table
//	    folded in as writer:<class>><writer> parts. Anchor consequence:
//	    rsys@6 (one mint, with the Themis integration pins).
func TestConstitutionHashPinned(t *testing.T) {
	const pinned = "1df0e28548a48b373e779dce8f00ea906c7f98afc009d573afccdf1009d77685"
	const historical = "33c6f6c54c2e7b56b33b27384488c03eb217aad0f504402b7ee750c27f22b8db"
	got := ConstitutionHash()
	if got == historical {
		t.Fatalf("L6 constitution hash is the pre-W-M1 value: the class→writer table is not folded in (D-W-4)")
	}
	if got != pinned {
		t.Fatalf("L6 constitution hash moved: %s (pinned %s) — a constitution change requires a deliberate re-pin and an anchor consequence", got, pinned)
	}
}
