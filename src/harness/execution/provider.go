package execution

import (
	"errors"
	"fmt"
)

var ErrAdmission = errors.New("environment-provision-refused")

// ProcessIdentity is the declared identity property (Q-L5-6). v1's
// local provider declares "inherited" — honestly. "least-privileged"
// is not in the vocabulary.
type ProcessIdentity string

const (
	IdentityInherited     ProcessIdentity = "inherited"
	IdentityDedicatedUser ProcessIdentity = "dedicated-user"
	IdentityNamespaced    ProcessIdentity = "namespaced"
)

// IsolationStrength is the declared strength of the network and
// host-service isolation properties (Q-L5-7). They are separate
// properties because they fail independently.
type IsolationStrength string

const (
	DeniedByConstruction IsolationStrength = "denied-by-construction"
	DeniedByEnforcement  IsolationStrength = "denied-by-enforcement"
)

// TerminationClass is the declared bounded-termination strength
// (Q-L5-2 amendment: cooperative cancellation is not "unconditional
// kill" and must not be declared as it).
type TerminationClass string

const (
	TerminationGroupKill   TerminationClass = "group-kill"
	TerminationCooperative TerminationClass = "cooperative"
)

// ProviderDeclaration is what a provider guarantees, at delivered
// strength. Every claim here is checked against spec requirements at
// admission; a dimension absent from Limits is a dimension the
// provider makes no claim about (Q-L5-9: enforced | observed |
// absent; never "supported").
type ProviderDeclaration struct {
	Name            string
	ProcessIdentity ProcessIdentity
	Network         IsolationStrength
	HostServices    IsolationStrength
	Termination     TerminationClass
	Limits          map[string]Strength
}

// Admit checks that the provider declares every dimension the spec
// requires, at sufficient strength. enforced satisfies observed;
// observed never satisfies enforced; absent satisfies nothing.
// Fail closed: refusal is environment-provision-refused, typed.
func Admit(decl ProviderDeclaration, spec *ProvisionSpec) error {
	for _, l := range spec.Limits {
		have, ok := decl.Limits[l.Dimension]
		if !ok {
			return fmt.Errorf("%w: provider %q makes no claim for required dimension %q", ErrAdmission, decl.Name, l.Dimension)
		}
		if l.Strength == StrengthEnforced && have != StrengthEnforced {
			return fmt.Errorf("%w: provider %q delivers %q for %q; spec requires enforced — observation is not enforcement", ErrAdmission, decl.Name, have, l.Dimension)
		}
	}
	return nil
}

// BinaryAttestation records the identity of the pinned executable at
// provision time (Q-L5-6: digest + mode; attestation is evidence of
// what was executed, never proof of trustworthy behavior).
type BinaryAttestation struct {
	Path   string
	Digest string // sha256 of the binary as resolved at provision
	Mode   string
}

// Workspace is the provisioned identity recorded in the trace.
type Workspace struct {
	Repo      string
	PinnedSHA string
	Root      string // absolute worktree path — the confinement root
}

// Transition is one typed lifecycle edge. Every transition that
// changes what L5 may claim is durable in the trace (Q-L5-12).
type Transition struct {
	From   State
	To     State
	Reason string
}

// OpRecord is one audited subprocess invocation — provisioning ops
// use the same record as active-phase ops: provisioning is execution
// (Q-L5-3), there is no trusted-by-position invocation.
type OpRecord struct {
	Phase      string // "provision" | "active" | "teardown"
	Argv       []string
	Exit       int
	Outcome    string // "ok" | typed failure
	MaxRSSByte int64  // observed (rusage), never represented as a bound
}

// Trace is the durable environment record (Q-L5-12 minimum: provision
// identity, provider declaration, effective envelope, seal reason,
// egress outcome, artifact address if acknowledged, teardown
// verification, final terminal state). Timing joins at the L6
// trace-sink era — explicit deferral, matching L4.
type Trace struct {
	CeilingHash string
	SpecHash    string
	Provider    ProviderDeclaration
	Binary      BinaryAttestation
	Workspace   Workspace
	Transitions []Transition
	Ops         []OpRecord
	SealReason  SealReason
	// EgressOutcome: "acknowledged" or the typed refusal/failure —
	// recorded for the durable minimum (Q-L5-12).
	EgressOutcome string
	// ArtifactAddress: the content address the store acknowledged;
	// empty when no artifact was produced.
	ArtifactAddress string
	// TeardownVerified: true only when post-teardown host-cleanliness
	// assertions passed. DESTROYED is never declared without it.
	TeardownVerified bool
}
