// Package intake is Themis's decision door (D-T-1..8): it resolves a
// referencable execution tuple against the harness record plane,
// re-establishes production (D-T-4) and verification (D-T-5) from the
// record, renders the evidence view, and (T-M4) appends a Position
// (D-T-7) with an observed decision witness (D-T-8).
//
// It depends on the harness module's READ-ONLY contracts (state,
// deployment, verification, verification/seam) and on nothing that
// executes: no tools, orchestration, execution, or runtime/model
// import, ever (D-T-10 writer wall). No harness package may reach this
// package, directly or transitively (D-T-10 dependency wall).
//
// T-M3 lands admissibility ONLY: Resolve and EvidenceView. Nothing in
// this file writes; Append/Current (the Position) are T-M4.
//
// Record facts this package replays, as the harness writes them today
// (verified against the harness sources 2026-09-25, T-M3 Gate 1):
//
//	class                 writer  body / refs
//	lifecycle             l6      {"to","reason"} (+ governed_hashes on CREATED)
//	l2-delivery           l7      {"kind":"materialized-governed-artifacts",
//	                               "objects":{"deployment_anchor": <object id>, ...}}
//	l4-audit              l4      {Tool, Decision, ResultHash (hex), RegistryHash, ...}
//	l10-verification      l10     {"contract","outcome","record"}; refs: evidence
//	artifact-bound        l6      {"artifact": <object id>}; refs [{id, egress-artifact}]
//	model-turn            l7      refs: the exact output object
//
// The constitution names `l5-transition` and `l5-op` classes but no
// harness layer writes them (T-M3 Gate 1, gap 1): D-T-4's chain is
// replayed over the links that EXIST — binding, object re-hash,
// terminal COMPLETED after the binding, no competing binding, layer-
// owned writers — and the L5 witness is recorded as an owed harness
// amendment, never simulated here.
//
// The egress artifact is the L5 egress MANIFEST (`changes[]` of path,
// type, new_hash, content), not the report file itself (T-M3 Gate 1,
// gap 2): D-T-5 check (3) is therefore "the reconstruction's raw bytes
// are a member of the bound manifest, by content hash AND by bytes",
// which is the property D-T-5 states (the PASS is about THIS artifact).
package intake

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"

	"github.com/tofchaliss/themis-ai-runtime/src/harness/deployment"
	"github.com/tofchaliss/themis-ai-runtime/src/harness/state"
	"github.com/tofchaliss/themis-ai-runtime/src/harness/verification"
	vseam "github.com/tofchaliss/themis-ai-runtime/src/harness/verification/seam"
)

// Tuple is the referencable execution tuple (D-T-1): the ADMISSIBILITY
// HANDLE a caller supplies. Nothing else — no object id, no report
// path — is ever an input.
type Tuple struct {
	AnchorHash       string
	TaskID           string
	ArtifactBoundSeq int64
}

// Checkout names Themis's OWN governed checkout: the registries that
// establish whether a recorded anchor and a recorded contract were
// governed (D-T-2, D-T-5(1)). Never caller-supplied per call; a
// deployment property of Themis.
type Checkout struct {
	AnchorsRegistryPath   string
	ContractsRegistryPath string
}

// WitnessL6Only names what the production replay establishes TODAY:
// the L6-written links (binding, object, COMPLETED) — not the L5-owned
// witnesses D-T-4 claims. Owner classification 2026-09-25: L5 witness
// events are a REQUIRED pre-T-M4 harness amendment
// (openspec/changes/l5-witness-events/); a Resolution carrying this
// value is valid as-recorded evidence and MUST NOT be used to create a
// Position. The value becomes WitnessL5Owned only when the five-link
// replay exists.
const WitnessL6Only = "l6-record-only"

// Typed refusal classes. Every refusal wraps exactly one of these and
// names the failed link in its message (D-T-4: link-named, never a
// generic "invalid artifact").
var (
	// ErrNotReferencable: the tuple does not name a completed, verified,
	// anchored execution (D-T-1, D-T-6 rows 1–2).
	ErrNotReferencable = errors.New("execution not referencable")
	// ErrDeployment: the recorded anchor cannot be re-established as
	// governed on Themis's checkout (D-T-2).
	ErrDeployment = errors.New("deployment-refused")
	// ErrProvenance: the production chain does not replay (D-T-4).
	ErrProvenance = errors.New("artifact-provenance-refused")
	// ErrVerification: no reproducible PASS over the bound artifact
	// (D-T-5).
	ErrVerification = errors.New("verification-refused")
)

// Resolution is everything Themis DERIVED from the tuple. A Position
// (T-M4) references these identities; it copies none of the bytes.
type Resolution struct {
	Tuple Tuple

	// D-T-2: the anchor as recorded and as registered on Themis's
	// checkout, with its lifecycle state AT INTAKE (D-T-6: withdrawal
	// after the execution is recorded, not a refusal).
	AnchorName    string
	AnchorVersion int
	AnchorState   string

	// D-T-4: the egress artifact the binding names, re-hashed.
	ArtifactObjectID string
	ArtifactBytes    []byte // the egress manifest, exact bytes
	BindingSeq       int64
	CompletedSeq     int64
	// ProductionWitness states which layer's witnesses the replay ran
	// over. WitnessL6Only until the L5 witness amendment lands.
	ProductionWitness string

	// D-T-5: the verification fact re-established over the artifact.
	VerificationSeq  int64
	ContractName     string
	ContractVersion  int
	ContractSHA256   string
	ContractState    string // registry state at intake (D-T-6)
	VerifierAuditSeq int64
	Reconstruction   verification.Report
	VerifiedPath     string // the manifest member the PASS is about
	VerifiedHash     string // hex sha256 of that member == raw bytes

	// Evidence view inputs (D-T-7: three facts, rendered separately).
	ModelTurns []ModelTurn
	Events     []state.Event
}

// ModelTurn is one recorded model output: an identity, never content
// served as authority (the bytes are reachable only through the
// record, at the floor).
type ModelTurn struct {
	Seq      int64
	ObjectID string
}

var (
	shaHex = regexp.MustCompile(`^[0-9a-f]{64}$`)
)

func hexOf(b []byte) string {
	s := sha256.Sum256(b)
	return hex.EncodeToString(s[:])
}

// Resolve performs D-T-1 → D-T-2 → D-T-4 → D-T-5 under the D-T-6 table
// and returns the derived Resolution, or a link-named refusal. It
// reads only; it never consults the L7 gate, the recorded outcome as
// a claim, or any caller-supplied identity beyond the tuple.
func Resolve(root *state.Root, checkout Checkout, tuple Tuple) (*Resolution, error) {
	if root == nil {
		return nil, fmt.Errorf("%w: no record plane", ErrNotReferencable)
	}
	// ---- D-T-1: the tuple names a completed, verified, anchored execution
	if !state.ValidTaskID(tuple.TaskID) {
		return nil, fmt.Errorf("%w: task id %q is not well-formed", ErrNotReferencable, tuple.TaskID)
	}
	if !shaHex.MatchString(tuple.AnchorHash) {
		return nil, fmt.Errorf("%w: anchor hash is not a well-formed sha256 identity", ErrNotReferencable)
	}
	if tuple.ArtifactBoundSeq <= 0 {
		return nil, fmt.Errorf("%w: artifact-bound seq must be positive", ErrNotReferencable)
	}
	sv, err := root.ReadStatus(tuple.TaskID)
	if err != nil {
		return nil, fmt.Errorf("%w: record for task %s unavailable: %v", ErrNotReferencable, tuple.TaskID, err)
	}
	if sv.Verdict != state.VerdictVerified {
		return nil, fmt.Errorf("%w: record verdict %s — no partial acceptance", ErrNotReferencable, sv.Verdict)
	}
	if sv.Status != state.StatusCompleted {
		return nil, fmt.Errorf("%w: not a completed execution (status %s)", ErrNotReferencable, sv.Status)
	}
	man, err := root.ReadManifest(tuple.TaskID)
	if err != nil {
		return nil, fmt.Errorf("%w: manifest unavailable: %v", ErrNotReferencable, err)
	}
	recorded := man.GovernedHashes["deployment_anchor"]
	switch {
	case recorded == "" || recorded == "unanchored":
		return nil, fmt.Errorf("%w: unanchored execution", ErrNotReferencable)
	case recorded != tuple.AnchorHash:
		return nil, fmt.Errorf("%w: the record identifies deployment anchor %s…, the tuple names %s…", ErrNotReferencable, recorded[:12], tuple.AnchorHash[:12])
	}
	events, err := root.ReadEvents(tuple.TaskID)
	if err != nil {
		return nil, fmt.Errorf("%w: event stream unavailable: %v", ErrNotReferencable, err)
	}
	res := &Resolution{Tuple: tuple, Events: events, BindingSeq: tuple.ArtifactBoundSeq}

	// ---- D-T-2: the anchor BYTES come from the record; registration
	// from Themis's checkout, any lifecycle state, two-way identity.
	anchorObj := materializedAnchorObject(events)
	if anchorObj == "" {
		return nil, fmt.Errorf("%w: the record carries no materialized deployment_anchor object", ErrDeployment)
	}
	anchorBytes, err := root.Store().GetObject(anchorObj)
	if err != nil {
		return nil, fmt.Errorf("%w: recorded anchor bytes unavailable: %v", ErrDeployment, err)
	}
	anchor, err := deployment.VerifyAnchorRecord(recorded, anchorBytes, checkout.AnchorsRegistryPath)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrDeployment, err)
	}
	res.AnchorName, res.AnchorVersion = anchor.Name, anchor.Deployment
	res.AnchorState, err = anchorStateAtIntake(checkout.AnchorsRegistryPath, anchor.SHA256)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrDeployment, err)
	}

	// ---- D-T-4: causal replay of the production chain.
	if err := replayProduction(root, man, events, res); err != nil {
		return nil, err
	}

	// ---- D-T-5: the L10 fact re-established over the bound artifact.
	if err := reestablishVerification(root, checkout, events, res); err != nil {
		return nil, err
	}

	for _, ev := range events {
		if ev.Class == state.EvModelTurn && ev.Writer == "l7" && len(ev.Refs) == 1 {
			res.ModelTurns = append(res.ModelTurns, ModelTurn{Seq: ev.Seq, ObjectID: ev.Refs[0].ID})
		}
	}
	return res, nil
}

// materializedAnchorObject finds the L7 materialization record and
// returns the deployment_anchor object id it names, or "".
func materializedAnchorObject(events []state.Event) string {
	for _, ev := range events {
		if ev.Class != state.EvL2Delivery || ev.Writer != "l7" {
			continue
		}
		var body struct {
			Kind    string            `json:"kind"`
			Objects map[string]string `json:"objects"`
		}
		if json.Unmarshal(ev.Body, &body) != nil || body.Kind != "materialized-governed-artifacts" {
			continue
		}
		id := body.Objects["deployment_anchor"]
		if id == "" {
			continue
		}
		// The named object must be one this event actually references
		// (never a bare id in a body).
		for _, r := range ev.Refs {
			if r.ID == id {
				return id
			}
		}
	}
	return ""
}

func anchorStateAtIntake(registryPath, artifactSHA string) (string, error) {
	reg, err := deployment.LoadRegistry(registryPath)
	if err != nil {
		return "", err
	}
	for _, e := range reg.Entries {
		if e.Artifact == artifactSHA {
			return e.State, nil
		}
	}
	return "", fmt.Errorf("anchor %s… not registered on Themis's checkout", artifactSHA[:12])
}

// egressManifest is the closed subset of the L5 egress manifest Themis
// reads. Unknown fields are the harness's; Themis interprets none.
type egressManifest struct {
	TaskID  string `json:"task_id"`
	Changes []struct {
		Path    string `json:"path"`
		Type    string `json:"type"`
		NewHash string `json:"new_hash"`
		Content string `json:"content"`
	} `json:"changes"`
}

func replayProduction(root *state.Root, man *state.Manifest, events []state.Event, res *Resolution) error {
	seq := res.BindingSeq
	var binding *state.Event
	for i := range events {
		if events[i].Seq == seq {
			binding = &events[i]
		}
	}
	if binding == nil {
		return fmt.Errorf("%w: no event at seq %d", ErrProvenance, seq)
	}
	if binding.Class != state.EvArtifact {
		return fmt.Errorf("%w: event at seq %d is %s, not artifact-bound", ErrProvenance, seq, binding.Class)
	}
	for _, ev := range events {
		if ev.Class == state.EvArtifact && ev.Seq != seq {
			return fmt.Errorf("%w: competing artifact-bound at seq %d (the tuple names seq %d)", ErrProvenance, ev.Seq, seq)
		}
	}
	if binding.Writer != "l6" {
		return fmt.Errorf("%w: artifact-bound at seq %d written by %q, not l6", ErrProvenance, seq, binding.Writer)
	}
	if len(binding.Refs) != 1 || binding.Refs[0].Class != state.ObjEgressArtifact {
		return fmt.Errorf("%w: artifact-bound at seq %d does not reference exactly one egress-artifact", ErrProvenance, seq)
	}
	var body struct {
		Artifact string `json:"artifact"`
	}
	if json.Unmarshal(binding.Body, &body) != nil || body.Artifact != binding.Refs[0].ID {
		return fmt.Errorf("%w: artifact-bound body names a different address than its reference", ErrProvenance)
	}
	id := binding.Refs[0].ID
	// Terminal COMPLETED must FOLLOW the binding, written by l6.
	var completed int64
	for _, ev := range events {
		if ev.Class != state.EvLifecycle || ev.Writer != "l6" {
			continue
		}
		var lb struct {
			To string `json:"to"`
		}
		if json.Unmarshal(ev.Body, &lb) == nil && lb.To == string(state.StatusCompleted) {
			completed = ev.Seq
		}
	}
	switch {
	case completed == 0:
		return fmt.Errorf("%w: no lifecycle COMPLETED written by l6", ErrProvenance)
	case completed < seq:
		return fmt.Errorf("%w: lifecycle COMPLETED (seq %d) precedes the binding (seq %d)", ErrProvenance, completed, seq)
	}
	// The manifest projection must agree with the stream.
	inManifest := false
	for _, a := range man.ArtifactAddrs {
		if a == id {
			inManifest = true
		}
	}
	if !inManifest {
		return fmt.Errorf("%w: manifest does not project artifact %s", ErrProvenance, id)
	}
	// The object re-hashes to the address (the store verifies content);
	// corruption or absence refuses (D-T-6).
	b, err := root.Store().GetObject(id)
	if err != nil {
		return fmt.Errorf("%w: artifact object %s: %v", ErrProvenance, id, err)
	}
	var em egressManifest
	if err := json.Unmarshal(b, &em); err != nil {
		return fmt.Errorf("%w: artifact %s is not an egress manifest: %v", ErrProvenance, id, err)
	}
	if em.TaskID != res.Tuple.TaskID {
		return fmt.Errorf("%w: egress manifest names task %q, the tuple names %q", ErrProvenance, em.TaskID, res.Tuple.TaskID)
	}
	res.ArtifactObjectID, res.ArtifactBytes, res.CompletedSeq = id, b, completed
	res.ProductionWitness = WitnessL6Only
	return nil
}

func reestablishVerification(root *state.Root, checkout Checkout, events []state.Event, res *Resolution) error {
	// The verification the harness progressed on is the LAST committed
	// l10-verification before the binding; one after the binding
	// cannot be about the production (D-T-5 (4)).
	var vev *state.Event
	for i := range events {
		ev := &events[i]
		if ev.Class != state.EvVerification {
			continue
		}
		if ev.Seq < res.BindingSeq {
			vev = ev
		} else if vev == nil {
			return fmt.Errorf("%w: verification does not correspond to artifact production (verification seq %d after binding seq %d)", ErrVerification, ev.Seq, res.BindingSeq)
		}
	}
	if vev == nil {
		return fmt.Errorf("%w: no reproducible PASS (no l10-verification precedes the binding)", ErrVerification)
	}
	if vev.Writer != "l10" {
		return fmt.Errorf("%w: l10-verification at seq %d written by %q, not l10", ErrVerification, vev.Seq, vev.Writer)
	}
	rep := vseam.ReconstructEvent(root, *vev, events)
	res.VerificationSeq, res.Reconstruction = vev.Seq, rep
	rec := rep.Evaluation
	if len(rep.MissingInputs) > 0 {
		return fmt.Errorf("%w: verification evidence unavailable (%v)", ErrVerification, rep.MissingInputs)
	}
	if !rep.Consistent {
		return fmt.Errorf("%w: no reproducible PASS (reconstruction inconsistent: %s)", ErrVerification, failedChecks(rep))
	}
	if rec.Outcome != verification.OutcomePass {
		return fmt.Errorf("%w: no reproducible PASS (reconstructed outcome %s)", ErrVerification, rec.Outcome)
	}
	// (1) contract registered on Themis's checkout, any lifecycle,
	// two-way identity: name@version ↔ contract hash. The STORED bytes
	// were what the reconstruction used; the registry establishes only
	// that they were governed (D-T-6).
	creg, err := verification.LoadRegistry(checkout.ContractsRegistryPath)
	if err != nil {
		return fmt.Errorf("%w: contracts registry unreadable: %v", ErrVerification, err)
	}
	res.ContractName, res.ContractVersion, res.ContractSHA256 = rec.ContractName, rec.ContractVersion, rec.ContractSHA256
	registered := false
	for _, e := range creg.Entries {
		if e.Contract != rec.ContractSHA256 {
			continue
		}
		if e.Name != rec.ContractName || e.Version != rec.ContractVersion {
			return fmt.Errorf("%w: contract not registered (%s@%d's bytes are registered as %s@%d)", ErrVerification, rec.ContractName, rec.ContractVersion, e.Name, e.Version)
		}
		registered, res.ContractState = true, string(e.State)
		break
	}
	if !registered {
		return fmt.Errorf("%w: contract not registered (%s@%d, %s…)", ErrVerification, rec.ContractName, rec.ContractVersion, short(rec.ContractSHA256))
	}
	// (2) execution_ref → an AUTHORIZED l4-audit of the recorded
	// capability whose ResultHash is the hash of the raw bytes.
	var auditSeq int64 = -1
	if _, err := fmt.Sscanf(rec.ExecutionRef, "l4:%d", &auditSeq); err != nil {
		return fmt.Errorf("%w: verifier execution not authorized (execution_ref %q)", ErrVerification, rec.ExecutionRef)
	}
	rawBytes, err := root.Store().GetObject("sha256:" + rec.RawObjectID)
	if err != nil || rec.RawObjectID == "" {
		return fmt.Errorf("%w: verification evidence unavailable (raw bytes %s)", ErrVerification, short(rec.RawObjectID))
	}
	rawHex := hexOf(rawBytes)
	var audit *state.Event
	for i := range events {
		if events[i].Seq == auditSeq && events[i].Class == state.EvL4Audit {
			audit = &events[i]
		}
	}
	if audit == nil {
		return fmt.Errorf("%w: verifier execution not authorized (no l4-audit at seq %d)", ErrVerification, auditSeq)
	}
	var ab struct {
		Tool         string `json:"Tool"`
		Decision     string `json:"Decision"`
		ResultHash   string `json:"ResultHash"`
		RegistryHash string `json:"RegistryHash"`
	}
	_ = json.Unmarshal(audit.Body, &ab)
	switch {
	case audit.Writer != "l4":
		return fmt.Errorf("%w: verifier execution not authorized (audit at seq %d written by %q)", ErrVerification, auditSeq, audit.Writer)
	case ab.Decision != "authorized":
		return fmt.Errorf("%w: verifier execution not authorized (decision %q)", ErrVerification, ab.Decision)
	case ab.Tool != rec.Capability:
		return fmt.Errorf("%w: verifier execution not authorized (audit executed %q, record names %q)", ErrVerification, ab.Tool, rec.Capability)
	case ab.RegistryHash != rec.RegistrySHA256:
		return fmt.Errorf("%w: verifier execution not authorized (authorizing registry differs)", ErrVerification)
	case ab.ResultHash != rawHex:
		return fmt.Errorf("%w: verifier execution not authorized (audit ResultHash %s… ≠ raw bytes %s…)", ErrVerification, short(ab.ResultHash), short(rawHex))
	}
	res.VerifierAuditSeq = auditSeq
	// (3) the raw bytes ARE a member of the bound artifact: by content
	// hash AND by bytes. verify(A) → egress(B) refuses here.
	var em egressManifest
	_ = json.Unmarshal(res.ArtifactBytes, &em)
	for _, c := range em.Changes {
		if c.Type == "deleted" || c.NewHash != rawHex {
			continue
		}
		if c.Content != string(rawBytes) {
			return fmt.Errorf("%w: verified bytes are not the bound artifact (member %s hashes as the verified bytes but differs)", ErrVerification, c.Path)
		}
		res.VerifiedPath, res.VerifiedHash = c.Path, rawHex
		return nil
	}
	return fmt.Errorf("%w: verified bytes are not the bound artifact (no member of egress %s has hash %s…)", ErrVerification, short(res.ArtifactObjectID), short(rawHex))
}

func failedChecks(rep verification.Report) string {
	out := ""
	for _, c := range rep.Checks {
		if !c.OK {
			if out != "" {
				out += ", "
			}
			out += c.Name
		}
	}
	if out == "" {
		out = "unnamed"
	}
	return out
}

func short(s string) string {
	if len(s) > 12 {
		return s[:12]
	}
	return s
}

// EvidenceView renders the three facts SEPARATELY (proposal: harness
// fact, verification fact, and — not yet — governance fact), each by
// identity. It contains no bytes of the report and no conclusion: what
// the artifact means is the human's decision (B-T-2).
type EvidenceView struct {
	Execution struct {
		AnchorHash    string `json:"anchor_hash"`
		AnchorName    string `json:"anchor_name"`
		AnchorVersion int    `json:"anchor_version"`
		AnchorState   string `json:"anchor_state_at_intake"`
		TaskID        string `json:"task_id"`
	} `json:"execution"`
	ModelTurns []ModelTurn `json:"model_turns"`
	Artifact   struct {
		ObjectID     string `json:"object_id"`
		BindingSeq   int64  `json:"artifact_bound_seq"`
		CompletedSeq int64  `json:"completed_seq"`
		VerifiedPath string `json:"verified_member_path"`
		VerifiedHash string `json:"verified_member_sha256"`
		// ProductionWitness is rendered so the view never claims more
		// than the record supports (owner, 2026-09-25).
		ProductionWitness string `json:"production_witness"`
	} `json:"artifact"`
	Verification struct {
		Seq            int64  `json:"seq"`
		Contract       string `json:"contract"`
		ContractSHA256 string `json:"contract_sha256"`
		ContractState  string `json:"contract_state_at_intake"`
		AuditSeq       int64  `json:"verifier_audit_seq"`
		Outcome        string `json:"reconstructed_outcome"`
		Consistent     bool   `json:"reconstruction_consistent"`
		Checks         int    `json:"checks"`
	} `json:"verification"`
}

// View derives the evidence view from a Resolution. Pure.
func (r *Resolution) View() EvidenceView {
	var v EvidenceView
	v.Execution.AnchorHash, v.Execution.AnchorName, v.Execution.AnchorVersion = r.Tuple.AnchorHash, r.AnchorName, r.AnchorVersion
	v.Execution.AnchorState, v.Execution.TaskID = r.AnchorState, r.Tuple.TaskID
	v.ModelTurns = append([]ModelTurn(nil), r.ModelTurns...)
	v.Artifact.ObjectID, v.Artifact.BindingSeq, v.Artifact.CompletedSeq = r.ArtifactObjectID, r.BindingSeq, r.CompletedSeq
	v.Artifact.VerifiedPath, v.Artifact.VerifiedHash = r.VerifiedPath, r.VerifiedHash
	v.Artifact.ProductionWitness = r.ProductionWitness
	v.Verification.Seq = r.VerificationSeq
	v.Verification.Contract = fmt.Sprintf("%s@%d", r.ContractName, r.ContractVersion)
	v.Verification.ContractSHA256, v.Verification.ContractState = r.ContractSHA256, r.ContractState
	v.Verification.AuditSeq = r.VerifierAuditSeq
	v.Verification.Outcome = string(r.Reconstruction.Evaluation.Outcome)
	v.Verification.Consistent = r.Reconstruction.Consistent
	v.Verification.Checks = len(r.Reconstruction.Checks)
	return v
}
