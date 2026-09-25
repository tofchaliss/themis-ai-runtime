// Package intake is Themis's decision door (D-T-1..8): it resolves a
// referencable execution tuple against the harness record plane,
// re-establishes production (D-T-4) and verification (D-T-5) from the
// record, renders the evidence view, and appends a Position (D-T-7)
// with an observed decision witness (D-T-8).
//
// It depends on the harness module's READ-ONLY contracts (state,
// deployment, verification, verification/seam) and on nothing that
// executes: no tools, orchestration, execution, or runtime/model
// import, ever (D-T-10 writer wall). No harness package may reach this
// package, directly or transitively (D-T-10 dependency wall).
//
// T-M1 establishes the package and its boundary only; Resolve,
// EvidenceView, Append, and Current land in T-M3.
package intake

import (
	"github.com/tofchaliss/themis/deployment"
	"github.com/tofchaliss/themis/state"
	"github.com/tofchaliss/themis/verification"
)

// Tuple is the referencable execution tuple (D-T-1): the ADMISSIBILITY
// HANDLE a caller supplies. Nothing else — no object id, no report
// path — is ever an input.
type Tuple struct {
	AnchorHash       string
	TaskID           string
	ArtifactBoundSeq int64
}

// The read-only harness contracts this package consumes, named so the
// dependency is explicit and the wall test can assert it is the whole
// of it.
var (
	_ = state.OpenRoot
	_ = deployment.VerifyAnchorRecord
	_ = verification.Reconstruct
)
