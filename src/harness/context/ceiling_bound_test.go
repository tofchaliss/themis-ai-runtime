package context

import (
	"testing"

	"github.com/tofchaliss/themis-ai-runtime/src/harness/runtime/model"
)

// C-L8-7: the provider-response ceiling (P-L8-1) never exceeds L2's
// per-item cap, so every object a model turn or delegation can produce
// is referenceable under what L2 accepts per item.
func TestProviderCeilingWithinItemCap(t *testing.T) {
	if model.DefaultMaxResponseBytes > int64(MaxItemBytes) {
		t.Fatalf("provider ceiling %d exceeds L2 MaxItemBytes %d", model.DefaultMaxResponseBytes, MaxItemBytes)
	}
}
