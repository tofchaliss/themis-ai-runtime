// Package context implements Layer 2 of the Themis AI Harness:
// Context Delivery. It retrieves, validates, normalizes,
// provenance-tags, and packages evidence for one model payload — and
// decides nothing else. Governing design:
// openspec/changes/layer-02-context-delivery/design.md (§3 grill
// record governs). L2 may transform representation, never security
// meaning; it re-encodes existing data and never creates
// propositions. Its failure can starve or garble model input, never
// bypass authorization, verification, or governance.
package context

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
)

// AuthorityClass is one of the four mutually exclusive
// authority/provenance classes, determined by who authored an item's
// propositions plus the registered governance treatment of that
// content — never by storage, retrieval, or transport (design §3,
// Q-L2-10). Transport is envelope metadata with zero authority
// weight.
type AuthorityClass string

const (
	// AuthorityGovernedRecord: authored by a Themis governed process —
	// Themis-originated governed truth.
	AuthorityGovernedRecord AuthorityClass = "governed-record"
	// AuthorityGovernedExternal: externally authored content accepted,
	// stored, and managed through a governed Themis process — never
	// promoted to Themis-originated truth.
	AuthorityGovernedExternal AuthorityClass = "governed-external"
	// AuthorityDerived: output of a registered computation executed by
	// Themis, with bounded proposition vocabulary and complete
	// computational provenance.
	AuthorityDerived AuthorityClass = "derived"
	// AuthorityExternalUntrusted: externally authored, no qualifying
	// governance treatment. The fail-closed default: unproven
	// provenance lands here; higher classes are earned through
	// registered provenance, never asserted.
	AuthorityExternalUntrusted AuthorityClass = "external-untrusted"
)

// Sensitivity levels, ordered. Sensitivity is orthogonal to authority
// (Stage-3 §3.12): a governed record can be restricted; external
// prose can be public.
type Sensitivity string

const (
	SensitivityPublic     Sensitivity = "public"
	SensitivityInternal   Sensitivity = "internal"
	SensitivityRestricted Sensitivity = "restricted"
)

var sensitivityRank = map[Sensitivity]int{
	SensitivityPublic:     0,
	SensitivityInternal:   1,
	SensitivityRestricted: 2,
}

// Availability is the typed-absence vocabulary (design §3, Q-L2-4):
// silence is a statement, so it is typed. The four states never
// collapse into bare absence.
type Availability string

const (
	AvailabilityDelivered     Availability = "delivered"
	AvailabilityUnavailable   Availability = "unavailable"
	AvailabilityNotApplicable Availability = "not_applicable"
	AvailabilityWithheld      Availability = "withheld_by_contract"
	// AvailabilityOmittedCapacity is the fifth state (Q-L3-3, owner
	// amendment to the locked Q-L2-4 vocabulary): the source
	// delivered, the contract permitted dropping, and the
	// deterministic capacity policy selected it for omission.
	AvailabilityOmittedCapacity Availability = "omitted_for_capacity"
)

// SourceStatus and DeliveryStatus are distinguished so a delivery
// refusal never falsely implies source unavailability (Q-L2-5).
type SourceStatus string

const (
	SourceAvailable   SourceStatus = "available"
	SourceUnavailable SourceStatus = "unavailable"
)

type DeliveryStatus string

const (
	DeliveryDelivered DeliveryStatus = "delivered"
	DeliveryRefused   DeliveryStatus = "refused"
	DeliveryWithheld  DeliveryStatus = "withheld"
	DeliveryNone      DeliveryStatus = "none"
	// DeliveryOmittedCapacity: capacity omission — distinct from
	// unavailable (source delivered fine) and from withheld
	// (deliberate contract exclusion). The distinction survives into
	// the trace.
	DeliveryOmittedCapacity DeliveryStatus = "omitted_for_capacity"
)

// Mechanism is transport metadata. It carries zero authority weight.
type Mechanism string

const (
	MechanismPlannedConnector Mechanism = "planned-connector"
	MechanismCapabilityFetch  Mechanism = "capability-fetch" // L4-era; reserved
)

// Provenance records authorship: origin is immutable across storage,
// retrieval, re-serialization, and transport — only an explicit
// governed transition changes an item's class, and external
// authorship remains recorded even then.
type Provenance struct {
	Origin string // "themis" | "external" | "computation"
	Source string // registered source name
	Author string // authoring party/process/computation
}

// ContextItem is the only unit of context: typed evidence with
// provenance. Evidence is byte-exact — L2 never sanitizes,
// paraphrases, or rewrites it (L2-5.1); imperative text in evidence
// is data the analyst must see.
type ContextItem struct {
	Kind        string
	Provenance  Provenance
	Authority   AuthorityClass
	Producer    string // who made it available (connector/ingestion service)
	Sensitivity Sensitivity
	Version     string // as-of / version pin
	Evidence    []byte
	Hash        string // SHA-256 of Evidence, hex — set by the gatherer
}

func evidenceHash(b []byte) string {
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}

// Caps (gate-0 decision): v1 constants, fail closed; revisited when
// Layer 3 owns token budgets.
const (
	MaxItemBytes    = 256 * 1024
	MaxContextBytes = 1024 * 1024
	// MaxContextItems bounds framing-overhead amplification: many tiny
	// items pass byte caps while the rendered view explodes (security
	// review HIGH-3).
	MaxContextItems = 512
	// MaxComposedBytes caps the rendered user message including
	// framing overhead. A Gather-passing plan with maximal metadata on
	// every item can still exceed this and refuse at Compose — that
	// direction is fail-closed by design, not an accident.
	MaxComposedBytes = MaxContextBytes + 256*1024
)

// Sentinel errors. ErrIntake-classified failures are re-exported from
// the instructions package so StatusOf works uniformly across L1+L2.
var (
	ErrContractInvalid     = errors.New("invalid context contract")
	ErrPlanOutsideContract = errors.New("plan exceeds the workflow contract")
	ErrUnrecognizedSource  = errors.New("unrecognized context source")
	ErrConfinement         = errors.New("path escapes the confinement root")
	ErrRequiredMissing     = errors.New("required context slot not deliverable")
	ErrItemTooLarge        = errors.New("context item exceeds size cap")
	ErrContextTooLarge     = errors.New("total context exceeds size cap")
	ErrSensitivityCeiling  = errors.New("item sensitivity exceeds the contract ceiling")
	ErrFramingCollision    = errors.New("no collision-free frame delimiter available")
	ErrMetadataInvalid     = errors.New("metadata field unsafe for frame rendering")
	ErrCompose             = errors.New("composition refused")
)
