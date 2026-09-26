package orchestration

import "testing"

// The L7 constitution hash is a governed identity (anchor pin
// constitution.orchestration). Pinned so that any movement is a
// deliberate act; D-W-4 requires this value to stay UNCHANGED by the
// L5 witness amendment, which this test proves.
func TestConstitutionHashPinned(t *testing.T) {
	const pinned = "008be050c29740ce8d863b3f7ae858fa6b63eb29ad853d563b358e2156662370"
	if got := ConstitutionHash(); got != pinned {
		t.Fatalf("L7 constitution hash moved: %s (pinned %s)", got, pinned)
	}
}
