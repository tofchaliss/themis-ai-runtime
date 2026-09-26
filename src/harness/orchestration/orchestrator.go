package orchestration

// The public L7 seam is exactly: Open · SubmitTask · ReadStatus
// (Q-L7-10). Open closes the past before the future opens (Q-L7-8);
// SubmitTask is the single typed assembly boundary — the
// ⊆-checkpoint that trusts the submitter for nothing (D-L7-2 draft /
// D-L7-10); ReadStatus serves the structurally content-free view.

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	l2 "github.com/tofchaliss/themis-ai-runtime/src/harness/context"
	"github.com/tofchaliss/themis-ai-runtime/src/harness/deployment"
	"github.com/tofchaliss/themis-ai-runtime/src/harness/execution"
	"github.com/tofchaliss/themis-ai-runtime/src/harness/instructions"
	"github.com/tofchaliss/themis-ai-runtime/src/harness/internal/strictjson"
	"github.com/tofchaliss/themis-ai-runtime/src/harness/runtime/model"
	"github.com/tofchaliss/themis-ai-runtime/src/harness/skills"
	"github.com/tofchaliss/themis-ai-runtime/src/harness/state"
	"github.com/tofchaliss/themis-ai-runtime/src/harness/tools"
)

// Config is the governance-side wiring an operator supplies once per
// orchestrator — infrastructure identity, not per-task policy.
type Config struct {
	StateRoot   string
	ArtifactDir string
	GitPath     string
	ProviderDir string // L5 provider base
	// InstructionRoots + PolicyPath: the L1 governed configuration.
	SafetyRoot string
	SystemRoot string
	// ThemisRoot is the themis-domain instruction root (the L2
	// authority-vocabulary's interpretive half — integration-audit
	// D7). Optional until the deployment anchor pins it; when set it
	// resolves with the same discipline as the other roots.
	ThemisRoot string
	PolicyPath string
	// G1 Deployment Anchor (D-G1-1/D-G1-1A). When AnchorPath is set,
	// Open ADMITS the anchor against the Governance-active anchors
	// registry (never trusting the operator hash as an admission
	// claim), verifies the instruction roots/policy against its pins,
	// and freezes it; SubmitTask then refuses any bundle artifact
	// that is not the anchored one. Unanchored mode remains legal for
	// the recorded test-harness caller role only — production wiring
	// REQUIRES the anchor (G1 lock).
	AnchorPath          string
	AnchorSHA256        string
	AnchorsRegistryPath string
	// ModelRegistryPath is the model registry (models.json) the
	// anchor pins — what allowlisted NAMES resolve to (runtime,
	// endpoint, credentials). Anchored deployments must configure it
	// unless the anchor declares "absent" (close-review HIGH-3).
	ModelRegistryPath string
	// SkillCatalogPath is the governed skill catalog. Under an
	// anchored deployment it is AUTHORITATIVE for skill composition
	// resolution (owner finding 1): the submitter selects an anchored
	// skill identity; the catalog supplies its composition hash.
	SkillCatalogPath string
	// ExecCeilingPath is the DEPLOYMENT's execution ceiling — exact
	// bytes supplied at Open, hash-bound by the admitted anchor
	// (owner decision 2026-09-13). Deployment-supplied never means
	// submitter-selected: an anchored task's ceiling must BE these
	// bytes, whatever path its envelope names.
	ExecCeilingPath string
	// DelegationRegistryPath is the L8 delegation-template registry
	// the anchor pins (`delegation_template_registry`); the seam in
	// Config.Delegator must have been built from these bytes (the
	// wiring's obligation). Empty iff the anchor declares "absent".
	DelegationRegistryPath string
	// ThemisStorePath is the Themis v0 read store directory the anchor
	// pins (`themis_store`); Config.ThemisSeam must have been built
	// from those bytes (the wiring's obligation, checked at Open
	// through ThemisSeam's StoreHash when it offers one). Empty iff the
	// anchor declares "absent".
	ThemisStorePath string
	// ThemisSeam is the injected L4 read door to Themis records
	// (tools.ThemisSeam): get_finding / get_product serve its bytes
	// under the registry's governed-record trust. Nil = no Themis
	// records in this deployment; the executors fail closed typed. L7
	// never reads through it and holds no Themis type — the seam is
	// handed to the executor table and nothing else.
	ThemisSeam tools.ThemisSeam
	// Unanchored is the EXPLICIT opt-in to running without a
	// deployment anchor — the recorded test-harness caller role
	// (close-review MEDIUM-1). Without it an anchorless Open refuses,
	// so a production deployment cannot fall into the bypass
	// silently; unanchored records carry an explicit sentinel.
	Unanchored bool
	// Model is the model.Interface provider (injected so the loop is
	// provider-agnostic and Register-testable with a scripted model).
	Model model.Interface
	// Verifier is the injected L10 evaluation seam (D-L10-13): the
	// evaluator sits in the governed result-processing path as
	// mechanical composition — L7 gains no semantic dependency (no
	// verification import, no "evaluate contract C" query channel; the
	// hook fires only on verifier-eligible capability results).
	// Assembly refuses a task granting a verifier-eligible capability
	// when no evaluator is wired (fail closed).
	Verifier VerificationEvaluator
	// Delegator is the injected L8 delegation seam (D-L8-3): the same
	// mechanical-composition posture as the verifier — L7 calls no L8
	// API and interprets nothing; the hook fires only on an authorized
	// delegation-class capability result. Assembly refuses a phase
	// exposing such a capability when no delegator is wired, and a
	// grant template_scope entry the registry does not know.
	Delegator Delegator
}

// VerificationEvaluator is the one-way L10 seam. The implementation
// lives with the service wiring; orchestration defines only the
// boundary types so the L7 package keeps zero dependency on the
// verification package (the skill-blind pattern, applied to
// contracts).
type VerificationEvaluator interface {
	// PreResolve runs the pre-instance stage of the L10 pipeline
	// (D-L10-8 "before an evaluation instance exists") for one
	// AUTHORIZED verifier-eligible call, BEFORE L7 commits the call's
	// l4-audit: a non-empty refusal is recorded in that audit body so
	// the text the model sees is reconstructable from the record
	// (F-L8-3 — the C-L8-12 mechanism applied to the L10 seam). No
	// evidence, no instance, no outcome, no event. An error return is
	// machinery failure (invariant path).
	PreResolve(taskID string, call model.ToolCall, authRegistrySHA256 string) (refusal string, err error)
	// EvaluateCall runs the L10 pipeline for one executed
	// verifier-eligible capability call. An error return is an
	// evaluator machinery failure and follows the harness
	// invariant-failure path — no outcome is minted (D-L10-8).
	EvaluateCall(taskID string, call model.ToolCall, resultEvidence []byte, executionRef, authRegistrySHA256 string) (*VerificationOutcome, error)
}

// VerificationOutcome is what crosses back: an opaque contract token,
// one of the five outcome values, the durable payloads, or a typed
// pre-instance refusal (D-L10-8 — refusals are not outcomes and never
// reach δ).
type VerificationOutcome struct {
	ContractToken  string
	Outcome        string
	Record         []byte // evaluation record content (opaque to L7)
	ContractBytes  []byte // resolved contract bytes (stored, R-L9-2 pattern)
	RawBytes       []byte
	CanonicalBytes []byte

	Refused       bool
	RefusalReason string
}

// StartupReport records what Open found and closed (Q-L7-8).
type StartupReport struct {
	Recovered []string // driven to a typed terminal
	Corrupt   []string // surfaced, preserved, untouched
	Terminal  []string // already closed
}

// Orchestrator is the L7 instance: one per state root (recorded v1
// deployment invariant).
type Orchestrator struct {
	cfg    Config
	root   *state.Root
	store  *execution.ArtifactStore
	prov   *execution.LocalProvider
	eis    *instructions.EffectiveSet
	policy *instructions.Policy
	anchor *deployment.Anchor // nil = unanchored (test-harness caller role)
	// execCeiling is the frozen hash of the deployment-supplied
	// execution ceiling verified against the anchor at Open.
	execCeiling string
}

// Open prepares the orchestrator and drives every discovered
// non-terminal record to a typed terminal BEFORE any new work is
// accepted — after Open, resume has no object, by reachability.
func Open(cfg Config) (*Orchestrator, *StartupReport, error) {
	if cfg.Model == nil {
		return nil, nil, fmt.Errorf("%w: no model interface configured — nothing is defaulted", ErrAssembly)
	}
	root, err := state.OpenRoot(cfg.StateRoot)
	if err != nil {
		return nil, nil, err
	}
	store, err := execution.NewArtifactStore(cfg.ArtifactDir)
	if err != nil {
		return nil, nil, err
	}
	prov, err := execution.NewLocalProvider(cfg.GitPath, cfg.ProviderDir)
	if err != nil {
		return nil, nil, err
	}
	// L1: resolve + render once per orchestrator lifetime (the
	// governed instruction configuration).
	policy, err := instructions.LoadPolicy(cfg.PolicyPath)
	if err != nil {
		return nil, nil, err
	}
	if cfg.AnchorPath != "" && cfg.Unanchored {
		return nil, nil, fmt.Errorf("%w: Unanchored declared alongside a deployment anchor — the caller role is ambiguous", ErrAssembly)
	}
	if cfg.AnchorPath == "" && !cfg.Unanchored {
		return nil, nil, fmt.Errorf("%w: no deployment anchor configured and Unanchored not explicitly set — a deployment governs by anchor or refuses to open (G1)", ErrAssembly)
	}
	// G1 admission runs BEFORE the instruction plane is consumed: the
	// bytes that become the EIS must already be the anchored bytes
	// (close-review CRITICAL-2 — verify-then-use, never
	// verify-after-use).
	var admitted *deployment.Anchor
	frozenCeiling := ""
	if cfg.AnchorPath != "" {
		// Append-only across restarts (owner finding 2): the last
		// observed registry state is held under this record root;
		// deletion, rebinding, or un-withdrawal between Opens is
		// TAMPER, detected at the read boundary.
		prior, perr := deployment.LoadObserved(cfg.StateRoot)
		if perr != nil {
			return nil, nil, fmt.Errorf("%w: observed anchors state unreadable: %v", ErrAssembly, perr)
		}
		current, cerr := deployment.LoadRegistry(cfg.AnchorsRegistryPath)
		if cerr != nil {
			return nil, nil, fmt.Errorf("%w: %v", ErrAssembly, cerr)
		}
		if aoErr := current.CheckAppendOnly(prior); aoErr != nil {
			return nil, nil, fmt.Errorf("%w: %v", ErrAssembly, aoErr)
		}
		a, aerr := deployment.AdmitAnchor(cfg.AnchorPath, cfg.AnchorSHA256, cfg.AnchorsRegistryPath)
		if aerr != nil {
			return nil, nil, fmt.Errorf("%w: %v", ErrAssembly, aerr)
		}
		if cfg.ThemisRoot == "" {
			return nil, nil, fmt.Errorf("%w: anchored deployments pin the themis instruction root — it must be configured", ErrAssembly)
		}
		if verr := verifyAnchoredInstructionPlane(cfg, a); verr != nil {
			return nil, nil, verr
		}
		// The deployment's execution ceiling: exact bytes supplied
		// here, accepted only when they hash to the anchor's pin.
		if cfg.ExecCeilingPath == "" {
			return nil, nil, fmt.Errorf("%w: an anchored deployment supplies its execution ceiling at Open — none is configured", ErrAssembly)
		}
		ecHash, eerr := deployment.HashFile(cfg.ExecCeilingPath)
		if eerr != nil {
			return nil, nil, fmt.Errorf("%w: deployment execution ceiling unreadable: %v", ErrAssembly, eerr)
		}
		if ecHash != a.ExecutionCeiling {
			return nil, nil, fmt.Errorf("%w: the supplied execution ceiling is not the one deployment %s@%d pins", ErrAssembly, a.Name, a.Deployment)
		}
		if _, lerr := execution.LoadCeiling(cfg.ExecCeilingPath); lerr != nil {
			// A pinned ceiling that cannot instantiate is a governance
			// artifact defect, caught here rather than mid-walk.
			return nil, nil, fmt.Errorf("%w: the anchored execution ceiling does not load: %v", ErrAssembly, lerr)
		}
		frozenCeiling = ecHash
		if serr := recordObservedRegistry(cfg.StateRoot, cfg.AnchorsRegistryPath); serr != nil {
			return nil, nil, fmt.Errorf("%w: %v", ErrAssembly, serr)
		}
		admitted = a
	}
	openSources := []instructions.Source{
		{Kind: instructions.ScopeHarnessSafety, Root: cfg.SafetyRoot},
		{Kind: instructions.ScopeHarnessSystem, Root: cfg.SystemRoot},
	}
	if cfg.ThemisRoot != "" {
		openSources = append(openSources, instructions.Source{Kind: instructions.ScopeThemisDomain, Root: cfg.ThemisRoot})
	}
	eis, err := instructions.Resolve(instructions.Config{Policy: policy}, openSources...)
	if err != nil {
		return nil, nil, err
	}
	// Render once at Open: an unrenderable L1 configuration fails the
	// orchestrator here, not mid-walk (delivery itself re-renders
	// through the L2 composer's SystemMessage seam per composition).
	if _, _, err := eis.Render(policy); err != nil {
		return nil, nil, err
	}
	o := &Orchestrator{cfg: cfg, root: root, store: store, prov: prov, eis: eis, policy: policy, anchor: admitted, execCeiling: frozenCeiling}

	// Startup sweep: close the past before opening the future.
	rep := &StartupReport{}
	tasksDir := filepath.Join(cfg.StateRoot, "tasks")
	entries, rdErr := os.ReadDir(tasksDir)
	if rdErr != nil && !os.IsNotExist(rdErr) {
		// Fail closed: an unreadable task root means unknowable
		// non-terminal state — the past cannot be closed, so the
		// future does not open (security review MED-5).
		return nil, nil, fmt.Errorf("%w: startup sweep cannot enumerate tasks: %v", ErrAssembly, rdErr)
	}
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		id := e.Name()
		view, verr := root.ReadStatus(id)
		if verr != nil {
			// A record CORRUPT verdict is surfaced-and-preserved; a
			// transient IO failure is NOT corruption — the past cannot
			// be examined, so the future does not open (architecture
			// review 2c: verdicts and IO errors must not conflate).
			if errors.Is(verr, state.ErrCorrupt) {
				rep.Corrupt = append(rep.Corrupt, id)
				continue
			}
			return nil, nil, fmt.Errorf("%w: startup sweep cannot read task %s: %v", ErrAssembly, id, verr)
		}
		if view.Verdict == state.VerdictCorrupt {
			rep.Corrupt = append(rep.Corrupt, id)
			continue // surfaced, preserved, untouched
		}
		switch view.Status {
		case state.StatusCompleted, state.StatusFailed, state.StatusFailedPartial:
			rep.Terminal = append(rep.Terminal, id)
		default:
			if _, rerr := root.Recover(id); rerr != nil {
				if errors.Is(rerr, state.ErrCorrupt) {
					rep.Corrupt = append(rep.Corrupt, id)
					continue
				}
				return nil, nil, fmt.Errorf("%w: startup sweep cannot recover task %s: %v", ErrAssembly, id, rerr)
			}
			rep.Recovered = append(rep.Recovered, id)
		}
	}
	return o, rep, nil
}

// verifyAnchoredSkill resolves a skill-attributed envelope's
// composition from the ANCHORED catalog (owner finding 1). An
// envelope claiming a skill under an anchored deployment must name a
// skill the catalog registers ACTIVE, and its sealed composition
// must be the composition that registration binds. L7 stays
// hash-comparing — it learns nothing about skills — but the identity
// it compares against now comes from governed bytes rather than from
// the submitter.
func (o *Orchestrator) verifyAnchoredSkill(a *deployment.Anchor, env *Envelope) (*skills.Catalog, *skills.Manifest, error) {
	// The selector is the load-bearing skill field (D-SA-5) — never
	// origin, which is attribution the validator has already required
	// to agree with it. Origin is read by nobody here.
	ref := env.Skill
	if ref == "" {
		// No Skill claim: the envelope validator already refuses
		// skill-attributing origin keys without the field, so there
		// is no skill identity to resolve.
		return nil, nil, nil
	}
	if o.cfg.SkillCatalogPath == "" {
		return nil, nil, fmt.Errorf("%w: skill-attributed task under an anchored deployment but no governed catalog is configured", ErrAssembly)
	}
	// ONE read: the bytes the anchor pin is verified against are the
	// bytes resolution consumes and the record retains (arch review
	// MED-1 — the captureVerified discipline; a second read would
	// verify A and use B).
	cat, cerr := skills.LoadCatalog(o.cfg.SkillCatalogPath)
	if cerr != nil {
		return nil, nil, fmt.Errorf("%w: anchored skill catalog unreadable: %v", ErrAssembly, cerr)
	}
	if cat.Hash != a.SkillCatalog {
		return nil, nil, fmt.Errorf("%w: skill catalog is not the anchored artifact (deployment %s@%d)", ErrAssembly, a.Name, a.Deployment)
	}
	// Resolution establishes: registered, ACTIVE (withdrawn refuses
	// typed — D-SA-8), manifest bytes two-way against the registered
	// composition hash. What it does NOT establish is that the
	// SUBMITTED composition is that one — that is the correspondence
	// below (D-SA-2).
	_, m, rerr := cat.Resolve(ref)
	if rerr != nil {
		return nil, nil, fmt.Errorf("%w: %v", ErrAssembly, rerr)
	}
	c := env.Composition
	if c == nil {
		return nil, nil, fmt.Errorf("%w: skill attribution without a composition commitment", ErrInvariant)
	}
	// D-SA-2 (v): every Skill-fixed member of the submitted commitment
	// must EQUAL the manifest's pin for that member — per member, with
	// its own refusal, never one derived hash (D-SA-2/Q-SA-5). The
	// manifest's pins are Governance's; the commitment's fields are the
	// submitter's claims; the seal is consulted by nothing here (D-SA-6:
	// two consumers only, never a Governance comparison). Effective
	// grant and spec are per-task and are judged by the instantiation
	// relation (D-SA-4), not by equality.
	for _, member := range []struct{ label, claimed, pinned string }{
		{"workflow", c.Workflow, m.Workflow.SHA256},
		{"workflow_ceiling", c.WorkflowCeiling, m.WorkflowCeiling.SHA256},
		{"context_contract", c.ContextContract, m.ContextContract.SHA256},
		{"grant_template", c.GrantTemplate, m.GrantTemplate.SHA256},
		{"spec_template", c.SpecTemplate, m.SpecTemplate.SHA256},
		{"input_schema", c.InputSchema, m.InputSchema.SHA256},
		{"procedure", c.Procedure, m.Procedure.SHA256},
	} {
		if member.claimed != member.pinned {
			return nil, nil, fmt.Errorf("%w: the submitted composition's %s is not the %s that %s registers — a submitter selects an anchored skill, never its constituent hashes", ErrAssembly, member.label, member.label, ref)
		}
	}
	return cat, m, nil
}

// verifyAnchoredInstructionPlane checks the instruction roots, the
// policy, and the model registry against the ADMITTED anchor's pins.
// Called at Open BEFORE resolution and again per task before
// re-resolution: resolveTaskEIS re-walks the roots on every
// SubmitTask, so a one-shot startup assertion would leave the
// anchor's instruction claim unbacked for every later task
// (close-review HIGH-1/CRITICAL-2).
func verifyAnchoredInstructionPlane(cfg Config, a *deployment.Anchor) error {
	for _, check := range []struct{ label, root, want string }{
		{"safety instruction root", cfg.SafetyRoot, a.InstructionSafetyRoot},
		{"system instruction root", cfg.SystemRoot, a.InstructionSystemRoot},
		{"themis instruction root", cfg.ThemisRoot, a.InstructionThemisRoot},
	} {
		got, herr := deployment.HashDir(check.root)
		if herr != nil {
			return fmt.Errorf("%w: %s unreadable for anchor verification: %v", ErrAssembly, check.label, herr)
		}
		if got != check.want {
			return fmt.Errorf("%w: %s is not the anchored artifact (deployment %s@%d)", ErrAssembly, check.label, a.Name, a.Deployment)
		}
	}
	if got, herr := deployment.HashFile(cfg.PolicyPath); herr != nil || got != a.InstructionPolicy {
		return fmt.Errorf("%w: instruction policy is not the anchored artifact (deployment %s@%d)", ErrAssembly, a.Name, a.Deployment)
	}
	// The compiled control vocabularies: a rebuilt binary whose
	// constitution differs cannot open under an anchor that pinned
	// the old one (owner finding 4 — "can changing this artifact
	// change the behavior or authority of an anchored deployment?").
	if a.Constitution.State != state.ConstitutionHash() {
		return fmt.Errorf("%w: L6 constitution is not the anchored one (deployment %s@%d)", ErrAssembly, a.Name, a.Deployment)
	}
	if a.Constitution.Orchestration != ConstitutionHash() {
		return fmt.Errorf("%w: L7 constitution is not the anchored one (deployment %s@%d)", ErrAssembly, a.Name, a.Deployment)
	}
	// The model registry governs what allowlisted NAMES resolve to —
	// runtime, endpoint, credential env (close-review HIGH-3; Q-G1-2's
	// endpoint clause). "absent" is the anchor's explicit declaration
	// that the deployment ships no registry, never a default.
	if a.ModelRegistry == "absent" {
		if cfg.ModelRegistryPath != "" {
			return fmt.Errorf("%w: the anchor declares no model registry but one is configured (deployment %s@%d)", ErrAssembly, a.Name, a.Deployment)
		}
	} else {
		if cfg.ModelRegistryPath == "" {
			return fmt.Errorf("%w: the anchor pins a model registry but none is configured (deployment %s@%d)", ErrAssembly, a.Name, a.Deployment)
		}
		got, herr := deployment.HashFile(cfg.ModelRegistryPath)
		if herr != nil || got != a.ModelRegistry {
			return fmt.Errorf("%w: model registry is not the anchored artifact (deployment %s@%d) — an endpoint enters a deployment only by Governance act", ErrAssembly, a.Name, a.Deployment)
		}
	}
	// The delegation-template registry (L8, Q-L8-7): the same posture
	// as the model registry — "absent" is a declaration, a pin is a
	// hash-bound artifact, and a template enters a deployment only by
	// Governance act.
	if a.DelegationTemplateRegistry == "absent" {
		if cfg.DelegationRegistryPath != "" {
			return fmt.Errorf("%w: the anchor declares no delegation-template registry but one is configured (deployment %s@%d)", ErrAssembly, a.Name, a.Deployment)
		}
	} else {
		if cfg.DelegationRegistryPath == "" {
			return fmt.Errorf("%w: the anchor pins a delegation-template registry but none is configured (deployment %s@%d)", ErrAssembly, a.Name, a.Deployment)
		}
		got, herr := deployment.HashFile(cfg.DelegationRegistryPath)
		if herr != nil || got != a.DelegationTemplateRegistry {
			return fmt.Errorf("%w: delegation-template registry is not the anchored artifact (deployment %s@%d) — a template enters a deployment only by Governance act", ErrAssembly, a.Name, a.Deployment)
		}
		// The seam that will serve delegations must hold THOSE bytes,
		// not a registry it loaded earlier from the same path
		// (architecture review MED-6).
		if h, ok := cfg.Delegator.(interface{ RegistryHash() string }); ok && h.RegistryHash() != a.DelegationTemplateRegistry {
			return fmt.Errorf("%w: the wired delegation seam holds a registry that is not the anchored artifact (deployment %s@%d)", ErrAssembly, a.Name, a.Deployment)
		}
	}
	// The Themis read store (D-T-9): the same posture — "absent" is a
	// declaration, a pin is a hash-bound artifact, and a Finding
	// enters a deployment only by Governance act. The pin is over the
	// two registries' exact bytes (findings then products), computed
	// here without any Themis import.
	if a.ThemisStore == "absent" {
		if cfg.ThemisStorePath != "" || cfg.ThemisSeam != nil {
			return fmt.Errorf("%w: the anchor declares no Themis store but one is configured (deployment %s@%d)", ErrAssembly, a.Name, a.Deployment)
		}
	} else {
		if cfg.ThemisStorePath == "" {
			return fmt.Errorf("%w: the anchor pins a Themis store but none is configured (deployment %s@%d)", ErrAssembly, a.Name, a.Deployment)
		}
		got, herr := themisStoreHash(cfg.ThemisStorePath)
		if herr != nil || got != a.ThemisStore {
			return fmt.Errorf("%w: Themis store is not the anchored artifact (deployment %s@%d) — a Finding enters a deployment only by Governance act", ErrAssembly, a.Name, a.Deployment)
		}
		if h, ok := cfg.ThemisSeam.(interface{ StoreHash() string }); ok && h.StoreHash() != a.ThemisStore {
			return fmt.Errorf("%w: the wired Themis seam holds a store that is not the anchored artifact (deployment %s@%d)", ErrAssembly, a.Name, a.Deployment)
		}
	}
	return nil
}

// themisStoreHash is the `themis_store` pin: SHA-256 over the exact
// findings.json bytes followed by the exact products.json bytes. L7
// computes it from the files so it imports nothing of Themis.
func themisStoreHash(dir string) (string, error) {
	fb, err := os.ReadFile(filepath.Join(dir, "findings.json"))
	if err != nil {
		return "", err
	}
	pb, err := os.ReadFile(filepath.Join(dir, "products.json"))
	if err != nil {
		return "", err
	}
	return hashBytes(append(append([]byte{}, fb...), pb...)), nil
}

// recordObservedRegistry persists the anchors-registry state this
// Open admitted against, so the next Open can prove the registry only
// grew (owner finding 2).
func recordObservedRegistry(stateRoot, registryPath string) error {
	raw, err := os.ReadFile(registryPath)
	if err != nil {
		return err
	}
	dst := deployment.ObservedRegistryPath(stateRoot)
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	return os.WriteFile(dst, raw, 0o644)
}

// ReadStatus serves the structurally content-free view (Q-L6-10).
func (o *Orchestrator) ReadStatus(taskID string) (state.StatusView, error) {
	return o.root.ReadStatus(taskID)
}

// TaskResult is SubmitTask's typed outcome.
type TaskResult struct {
	TaskID   string
	Status   state.TaskStatus
	Verdict  state.Verdict
	Artifact string // state-store address, when completed with egress
}

// SubmitTask is the single typed assembly + execution boundary. It
// takes the envelope PATH and loads it here — an in-process caller
// can therefore never execute a mutated envelope under a stale hash
// (architecture review 1.3: the recorded attribution is always the
// executed configuration). It loads every governed artifact
// fail-closed, performs the ⊆-checkpoint, mints the single-use
// identity, and runs the walk to a typed terminal. It trusts the
// submitter for nothing.
func (o *Orchestrator) SubmitTask(envelopePath string) (TaskResult, error) {
	env, err := LoadEnvelope(envelopePath)
	if err != nil {
		return TaskResult{}, err
	}
	res := TaskResult{TaskID: env.TaskID}
	// Under an anchored, skill-attributed submission these hold the
	// catalog state and manifest admission resolved (D-SA-2) — the
	// reference for instantiation (D-SA-4) and the exact bytes the
	// record retains (Claim 2 evidence). Nil otherwise.
	var skillCatalog *skills.Catalog
	var skillManifest *skills.Manifest

	// Governed artifacts, fail closed.
	wfCeiling, err := LoadWorkflowCeiling(env.WorkflowCeilingPath)
	if err != nil {
		return res, err
	}
	wf, err := LoadWorkflow(env.WorkflowPath, wfCeiling)
	if err != nil {
		return res, err
	}
	reg, err := tools.LoadRegistry(env.RegistryPath)
	if err != nil {
		return res, err
	}
	execCeiling, err := execution.LoadCeiling(env.ExecCeilingPath)
	if err != nil {
		return res, err
	}
	spec, err := execution.LoadSpec(env.SpecPath)
	if err != nil {
		return res, err
	}
	contract, err := l2.LoadContract(env.ContextContractPath)
	if err != nil {
		return res, err
	}
	// G1 bundle validation: every governing artifact must BE the
	// anchored one — the submitter chooses a task WITHIN the
	// deployment, never the deployment (Q-G1-5). Fail closed, no
	// closest match, no partial bundle (Q-G1-9).
	if a := o.anchor; a != nil {
		// Re-verify the instruction plane per task: resolveTaskEIS
		// re-walks the roots, so the anchor's claim must be re-checked
		// against the bytes THIS task will consume (HIGH-1).
		if verr := verifyAnchoredInstructionPlane(o.cfg, a); verr != nil {
			return res, verr
		}
		// D-SA-9: deployment-level Skill admissibility — the anchor's
		// exact allowlist, checked BEFORE any catalog resolution and
		// independently of the bundle gate below. Catalog membership
		// says the identity exists; only this pin says this deployment
		// may run it. Mirrors the model allowlist: a skill enters a
		// deployment only by Governance act.
		if env.Skill != "" {
			admitted := false
			for _, s := range a.Skills {
				if s == env.Skill {
					admitted = true
					break
				}
			}
			if !admitted {
				return res, fmt.Errorf("%w: skill %q is not in the anchored skill allowlist (deployment %s@%d) — a skill enters a deployment only by Governance act", ErrAssembly, env.Skill, a.Name, a.Deployment)
			}
		}
		if reg.Hash != a.ToolRegistry {
			return res, fmt.Errorf("%w: tool registry is not the anchored artifact (deployment %s@%d) — a mutually consistent bundle is not a governed bundle", ErrAssembly, a.Name, a.Deployment)
		}
		// The workflow bundle is INDIVISIBLE: the submitter selects an
		// anchored workflow, and that selection fixes its ceilings and
		// contract — no pairing an anchored workflow with another
		// bundle's ceiling (close-review M-6 / owner finding 4).
		var bundle *deployment.WorkflowBundle
		for i := range a.Workflows {
			if a.Workflows[i].Workflow == wf.Hash {
				bundle = &a.Workflows[i]
				break
			}
		}
		if bundle == nil {
			return res, fmt.Errorf("%w: workflow is not in the anchored workflow set (deployment %s@%d)", ErrAssembly, a.Name, a.Deployment)
		}
		// The execution ceiling is the DEPLOYMENT's, verified against
		// the anchor at Open and frozen: an envelope may name any
		// path, but its bytes must be the deployment's ceiling —
		// deployment-supplied never means submitter-selected.
		if execCeiling.Hash != o.execCeiling {
			return res, fmt.Errorf("%w: the task's execution ceiling is not the deployment's ceiling (deployment %s@%d) — the ceiling is supplied at Open, never chosen per task", ErrAssembly, a.Name, a.Deployment)
		}
		for _, check := range []struct{ label, got, want string }{
			{"workflow ceiling", wfCeiling.Hash, bundle.WorkflowCeiling},
			{"context contract", contract.Hash, bundle.ContextContract},
		} {
			if check.got != check.want {
				return res, fmt.Errorf("%w: %s is not the artifact this anchored workflow bundles (deployment %s@%d)", ErrAssembly, check.label, a.Name, a.Deployment)
			}
		}
		// D-SA-3: under an anchored deployment a skill-scope instruction
		// artifact enters only through a Skill-attributed envelope whose
		// composition corresponds. An unattributed procedure would be a
		// submitter-authored trusted-scope instruction source verified
		// only against the submitter's own hash — hash integrity is not
		// Governance provenance (finding F-SA-1).
		if env.Skill == "" && env.SkillProcedurePath != "" {
			return res, fmt.Errorf("%w: a procedure is a Skill artifact — an anchored deployment (%s@%d) delivers skill-scope instructions only through a corresponding Skill composition, never from an unattributed envelope", ErrAssembly, a.Name, a.Deployment)
		}
		// Skill composition resolves from the ANCHORED CATALOG, never
		// from the submitter's own hash (owner finding 1): the
		// submitter selects an anchored skill identity; the catalog
		// says what that identity is composed of.
		cat, man, serr := o.verifyAnchoredSkill(a, env)
		if serr != nil {
			return res, serr
		}
		skillCatalog, skillManifest = cat, man
		allowedModel := false
		for _, m := range a.Models {
			if env.Model == m {
				allowedModel = true
				break
			}
		}
		if !allowedModel {
			return res, fmt.Errorf("%w: model %q is not in the anchored allowlist (deployment %s@%d) — a model enters a deployment only by Governance act", ErrAssembly, env.Model, a.Name, a.Deployment)
		}
	}
	// The EXACT bytes each loader accepted, for durable storage after
	// the task exists (ADG-L9/L6-1). These are read once here and then
	// PROVEN to be the same bytes the loaders hashed — a second
	// independent read would produce identity over A with durability
	// containing B, which is the TOCTOU shape R-L9-2 forbids and which
	// this code previously had (final security review H-2).
	materialized := map[string][]byte{}
	for _, m := range []struct {
		label, path, loaderHash string
	}{
		{"workflow", env.WorkflowPath, wf.Hash},
		{"workflow_ceiling", env.WorkflowCeilingPath, wfCeiling.Hash},
		{"context_contract", env.ContextContractPath, contract.Hash},
		{"spec", env.SpecPath, spec.Hash},
	} {
		body, rerr := captureVerified(m.label, m.path, m.loaderHash)
		if rerr != nil {
			return res, rerr
		}
		materialized[m.label] = body
	}
	if o.anchor != nil {
		// Q-G1-7 evidence, not merely the identifier: the anchor
		// BYTES are durable, so cold reconstruction can interpret
		// what the deployment pinned (close-review MEDIUM-2).
		materialized["deployment_anchor"] = o.anchor.Raw
	}

	// C2 (D-L9-11a/b): the commitment's seal was verified at load, so
	// these identities are one sealed unit rather than independently
	// chosen strings. Each artifact L7 materialized must match the one
	// the sealed composition names.
	//
	// The templates and input schema are sealed but not materialized by
	// L7 (they are L9-side inputs); the seal binds them so the executed
	// composition cannot silently differ from the reviewed one in those
	// members either — detectable post-hoc against the catalog.
	// (was: D-L9-11a) when the envelope carries a composition
	// commitment, every artifact L7 just materialized must be the one
	// that submission committed to. L7 learns nothing about skills from
	// this — it compares hashes it computed against identities it was
	// given. A mismatch means the executed artifact is not the named
	// one: an integrity violation, not a governance judgment.
	if c := env.Composition; c != nil {
		for _, check := range []struct{ label, want, got string }{
			{"workflow", c.Workflow, wf.Hash},
			{"workflow_ceiling", c.WorkflowCeiling, wfCeiling.Hash},
			{"context_contract", c.ContextContract, contract.Hash},
			{"spec", c.Spec, spec.Hash},
		} {
			if err := c.verify(check.label, check.want, check.got); err != nil {
				return res, err
			}
		}
	}
	// Cross-artifact binding: the context contract is minted for one
	// workflow; a contract for another lattice is refused, not adapted.
	if contract.Workflow != wf.Name {
		return res, fmt.Errorf("%w: context contract is for workflow %q, envelope workflow is %q", ErrAssembly, contract.Workflow, wf.Name)
	}

	// ⊆-checkpoint part 1 (pre-provision): control verbs exist in the
	// constitution both ways; workflow capabilities ⊆ registry; grant
	// (raw) ⊆ workflow ceiling.
	if err := checkControlVocabulary(reg); err != nil {
		return res, err
	}
	registered := map[string]bool{}
	verifierEligible := map[string]bool{}
	delegationClass := map[string]bool{}
	for _, t := range reg.Tools {
		registered[t.Name] = true
		if t.VerifierEligible {
			verifierEligible[t.Name] = true
		}
		if t.Target == tools.TargetDelegationTemplate {
			delegationClass[t.Name] = true
		}
	}
	declaredEv := map[string]bool{}
	for _, e := range wf.DeclaredEvents {
		declaredEv[e] = true
	}
	for _, p := range wf.Phases {
		phaseEdges := map[string]bool{}
		for _, e := range p.Edges {
			phaseEdges[e.On] = true
		}
		exposesVerifier := false
		exposesDelegation := false
		for _, c := range p.Capabilities {
			if !registered[c] {
				return res, fmt.Errorf("%w: workflow capability %q is not in the registry", ErrAssembly, c)
			}
			if verifierEligible[c] {
				exposesVerifier = true
			}
			if delegationClass[c] {
				exposesDelegation = true
			}
		}
		if exposesDelegation && o.cfg.Delegator == nil {
			// Mirror of the verifier check (L8 §5.1): the seam that
			// would execute the delegation must exist before a phase
			// may offer the capability — a refusal at assembly, never
			// a runtime surprise.
			return res, fmt.Errorf("%w: phase %q exposes delegation capability but no L8 delegator is wired", ErrAssembly, p.Name)
		}
		if exposesVerifier {
			// Declaration-gated verification exposure + reachability
			// totality (D-L10-13, L10 amendment): a phase exposing a
			// verifier-eligible capability makes all five verification
			// events reachable, so the workflow must declare them and
			// THIS phase must map each — a load/assembly refusal, never
			// a runtime surprise. No evaluator wired = the events could
			// never be produced correctly = refusal (fail closed).
			if o.cfg.Verifier == nil {
				return res, fmt.Errorf("%w: phase %q exposes verifier-eligible capability but no L10 evaluator is wired", ErrAssembly, p.Name)
			}
			for ev := range verificationEvents {
				if !declaredEv[ev] {
					return res, fmt.Errorf("%w: phase %q exposes a verifier-eligible capability but the workflow does not declare %q", ErrAssembly, p.Name, ev)
				}
				if !phaseEdges[ev] {
					return res, fmt.Errorf("%w: phase %q exposes a verifier-eligible capability but has no edge for reachable event %q", ErrAssembly, p.Name, ev)
				}
			}
		}
	}
	rawGrant, err := os.ReadFile(env.GrantPath)
	if err != nil {
		return res, fmt.Errorf("%w: %v", ErrAssembly, err)
	}
	// The grant carries actual tool authority, so its identity is
	// committed like the rest — against the ENVELOPE bytes, since L7
	// rewrites the grant to bind @workspace and the executed bytes are
	// deliberately not the submitted ones.
	if c := env.Composition; c != nil {
		if err := c.verify("grant", c.Grant, hashBytes(rawGrant)); err != nil {
			return res, err
		}
	}
	// D-SA-4: under an anchored, skill-attributed submission the
	// effective grant and spec must be legitimate INSTANTIATIONS of the
	// registered templates. The reference templates are resolved from
	// the manifest's pins — never from the commitment's claimed hashes
	// (reference-source rule): the claim was consumed only by the seal
	// and by D-SA-2's equality. The relation is L4/L5 vocabulary
	// applied mechanically; L7 decides no grant policy.
	if skillManifest != nil {
		_, grantTpl, terr := skillManifest.ResolvePin("grant_template")
		if terr != nil {
			return res, fmt.Errorf("%w: %v", ErrAssembly, terr)
		}
		if err := tools.Instantiates(rawGrant, grantTpl, env.TaskID); err != nil {
			return res, fmt.Errorf("%w: %v", ErrAssembly, err)
		}
		_, specTpl, terr := skillManifest.ResolvePin("spec_template")
		if terr != nil {
			return res, fmt.Errorf("%w: %v", ErrAssembly, terr)
		}
		if err := execution.SpecInstantiates(spec, specTpl); err != nil {
			return res, fmt.Errorf("%w: %v", ErrAssembly, err)
		}
	}
	// Cross-artifact identity binding (security review MED-1): a
	// grant or spec minted for another task is refused, not accepted
	// on faith.
	if spec.TaskID != env.TaskID {
		return res, fmt.Errorf("%w: spec task_id %q does not bind to envelope task %q", ErrAssembly, spec.TaskID, env.TaskID)
	}

	// L5 provisioning (the loop owns the environment lifecycle).
	envn, err := o.prov.Provision(execCeiling, spec)
	if err != nil {
		return res, err
	}
	ws := envn.Workspace()

	// Grant instantiation: the envelope grant carries the
	// "@workspace" placeholder; assembly binds it to the provisioned
	// root. Binding an instance scope is instantiation (narrowing),
	// never authority creation; both the envelope grant hash and the
	// effective grant hash are recorded.
	grant, effPath, err := instantiateGrant(rawGrant, ws.Root, o.cfg.StateRoot)
	if err != nil {
		envn.Teardown()
		return res, err
	}
	defer os.Remove(effPath)
	if grant.TaskID != env.TaskID {
		envn.Teardown()
		return res, fmt.Errorf("%w: grant task_id %q does not bind to envelope task %q", ErrAssembly, grant.TaskID, env.TaskID)
	}
	if err := grantWithinCeiling(grant, wfCeiling); err != nil {
		envn.Teardown()
		return res, err
	}
	// Register D (C-L8-19 I): the static execution bound, computed
	// from the loaded artifacts and recorded. Executions ≤ W + M holds
	// by construction of the two checks above (worst-case walk ≤ W at
	// load; grant total ≤ M just now), so it is a recorded statement,
	// not a runtime check that could fire.
	bound := executionBound(wf, wfCeiling, grant, reg, spec)
	// Grant validation, not call authorization (C-L8-15 G): every
	// template_scope entry must EXIST in the delegation registry in
	// force, as unregistered phase capabilities are refused. Existence,
	// not usability: a withdrawn template stays admissible here and is
	// refused at the delegate boundary (C-L8-14 G). Without a delegator
	// no entry can be validated, so entries refuse.
	for _, e := range grant.Entries {
		for _, ref := range e.TemplateScope {
			if o.cfg.Delegator == nil {
				envn.Teardown()
				return res, fmt.Errorf("%w: grant %q names template_scope %q but no L8 delegator is wired to validate it", ErrAssembly, e.Tool, ref)
			}
			if err := o.cfg.Delegator.Registered(ref); err != nil {
				envn.Teardown()
				return res, fmt.Errorf("%w: grant %q template_scope entry %q does not resolve: %v", ErrAssembly, e.Tool, ref, err)
			}
		}
	}

	// Root disjointness at its designed call site (Q-L6-4/D-L7-10).
	if err := state.CheckDisjointRoots(o.cfg.StateRoot, o.cfg.ArtifactDir, ws.Root); err != nil {
		envn.Teardown()
		return res, err
	}

	// Per-task instruction resolution, ONCE, before the walk begins.
	// Without a skill procedure this is the orchestrator's base EIS;
	// with one, the activated skill source joins it — resolved and
	// byte-verified here so the walk never re-reads instruction bytes
	// from mutable storage (D-L9-11).
	eis, procedureBytes, parentSources, err := o.resolveTaskEIS(env)
	if err != nil {
		envn.Teardown()
		return res, err
	}

	// Single-use identity + full governed attribution.
	governed := map[string]string{
		"envelope": env.Hash, "workflow": wf.Hash, "workflow_ceiling": wfCeiling.Hash,
		"registry": reg.Hash, "grant_envelope": hashBytes(rawGrant), "grant_effective": grant.Hash,
		// The authority the grant confers, hashed independently of the
		// task-bound bytes: two tasks differing only in identity have
		// the same authority digest, so any influence on caps, tools,
		// or the aggregate is visible in the record even though the
		// grant hashes legitimately differ (L9 test review HIGH).
		"grant_authority": grantAuthorityDigest(grant),
		"exec_ceiling":    execCeiling.Hash, "spec": spec.Hash, "context_contract": contract.Hash,
		"l1_eis": eis.Hash, "l1_policy": eis.PolicyHash,
		// The EIS hash covers delivered bodies only, so two resolutions
		// with different conflict outcomes can share it. Record the
		// status so the record distinguishes a clean resolution from one
		// that dropped material (L9 security MED-2).
		"l1_status":       string(eis.Status),
		"l6_constitution": state.ConstitutionHash(), "l7_constitution": ConstitutionHash(),
		// Register D: the static bound this task ran under.
		"l8_execution_bound": fmt.Sprintf("turns<=%d;delegations<=%d;executions<=%d;output_captured<=%d;wall_s<=%d", bound.Turns, bound.Delegations, bound.Executions, bound.OutputCaptured, bound.WallDeadlineS),
	}
	// Opaque attribution: recorded verbatim, interpreted by nobody
	// (D-L9-13). L7 does not know what a skill is; it only preserves
	// what the envelope stated, so the record can be checked against
	// the catalog after the fact.
	for k, v := range env.Origin {
		governed["origin:"+k] = v
	}
	// The load-bearing selector admission actually used (D-SA-5) —
	// recorded beside the attribution so the two independently derived
	// values stay comparable in the record.
	if skillManifest != nil {
		governed["skill"] = env.Skill
	} else if env.Skill != "" {
		// Unanchored: the selector is a Claim-1 attribution only, and
		// the record must not read as if a registered composition was
		// verified (D-SA-7; security review LOW-1).
		governed["skill_claimed"] = env.Skill
	}
	if o.anchor != nil {
		// Q-G1-7: the assembly record proves "executed under this
		// ADMITTED deployment anchor", re-verifiable at replay.
		governed["deployment_anchor"] = o.anchor.SHA256
	} else {
		// An unanchored record must be distinguishable from an
		// anchored one with a dropped key (close-review MEDIUM-1).
		governed["deployment_anchor"] = "unanchored"
	}
	task, err := o.root.CreateTask(env.TaskID, state.TaskOptions{
		RetryOf:        env.RetryOf,
		GovernedHashes: governed,
	})
	if err != nil {
		envn.Teardown()
		return res, err
	}
	defer task.Close()

	// Record-before-effect extends to authority: the grant handed to the
	// walk must be the one whose authority was just recorded. Any
	// divergence between recording and use — however introduced — is an
	// invariant violation, not a difference to tolerate.
	if grantAuthorityDigest(grant) != governed["grant_authority"] {
		envn.Teardown()
		return res, fmt.Errorf("%w: the grant changed between attribution and execution", ErrInvariant)
	}
	// Reconstruction provenance (R-L9-1 / ADG-L9/L6-1): store the EXACT
	// bytes verified at assembly as L6 content-addressed objects, and
	// reference them from the record. L6 derives each object's address
	// by hashing the bytes itself, so the recorded identity cannot have
	// come from an envelope claim — which is the independent source an
	// equality assertion could never supply. The bytes are the ones
	// already captured and verified; nothing is re-read.
	// R-L9-2: the submitted grant bytes and the verified procedure bytes
	// join the durable set — the exact bytes already held, never a
	// re-read (a re-read yields identity over A with durability over B).
	// L1 remains the sole authority for procedure IDENTITY; L6 owns only
	// the durable bytes, so no second identity mechanism is created.
	materialized["grant_submitted"] = rawGrant
	if len(procedureBytes) > 0 {
		materialized["procedure"] = procedureBytes
	}
	// Claim 2 evidence (A-SA-11): the exact Governance bytes an anchored
	// skill admission consumed — catalog and manifest — join the durable
	// closure, content-addressed and deduplicated across tasks, so
	// "this task ran the registered composition" reconstructs from the
	// record alone and never from today's catalog (D-SA-8).
	if skillCatalog != nil && skillManifest != nil {
		materialized["skill_catalog"] = skillCatalog.Raw
		materialized["skill_manifest"] = skillManifest.Raw
	}
	if err := recordMaterializedArtifacts(task, materialized); err != nil {
		// Post-CreateTask failures must still tear the environment down;
		// the record stays non-terminal and the startup sweep closes it
		// (final security review M-1: these were the only paths that
		// leaked a provisioned workspace).
		envn.Teardown()
		return res, err
	}

	// The executor table is built with the task in hand so the
	// delegate executor's compose half is bound to THIS record (L8
	// M2/M4): a read handle, the parent's activated sources, and the
	// registry-in-force for class derivation. Nil delegator = the
	// executor fails closed at the call; assembly already refused any
	// phase that could reach it.
	instBase := InstantiationRequest{
		TaskID: env.TaskID, Record: o.root, ParentSources: parentSources, RegistryHash: reg.Hash,
		ParentEIS: eis, ParentSensitivityCeiling: contract.SensitivityCeiling,
		ToolTrust: func(tool string) (l2.AuthorityClass, bool) {
			for i := range reg.Tools {
				if reg.Tools[i].Name == tool {
					return reg.Tools[i].Trust, true
				}
			}
			return "", false
		},
	}
	var inst *delegationInstantiator
	var instIface tools.DelegationInstantiator
	if o.cfg.Delegator != nil {
		inst = &delegationInstantiator{d: o.cfg.Delegator, base: instBase}
		instIface = inst
	}
	table, err := tools.NewExecutorTableWith(reg, o.cfg.ThemisSeam, instIface)
	if err != nil {
		envn.Teardown()
		return res, err
	}
	w := &walk{
		o: o, env: env, wf: wf, reg: reg, grant: grant, table: table,
		task: task, l5: envn, execCeiling: execCeiling, spec: spec,
		contract: contract, eis: eis, instBase: instBase, inst: inst,
	}
	return w.run()
}

// recordMaterializedArtifacts binds the governed artifact bytes L7
// actually executed into the durable record through L6's existing
// primitives: one content-addressed object per artifact, referenced
// from a single event. No new object class and no new event class —
// evidence-payload already means "durable bytes referenced by an
// event", and the reference carries its own classification.
func recordMaterializedArtifacts(task *state.TaskRecord, materialized map[string][]byte) error {
	labels := make([]string, 0, len(materialized))
	for label := range materialized {
		labels = append(labels, label)
	}
	sort.Strings(labels)
	ids := map[string]string{}
	var refs []state.Ref
	for _, label := range labels {
		id, err := task.StoreObject(state.ObjEvidencePayload, materialized[label])
		if err != nil {
			return fmt.Errorf("%w: recording materialized %s: %v", ErrAssembly, label, err)
		}
		ids[label] = id
		refs = append(refs, state.Ref{ID: id, Class: state.ObjEvidencePayload})
	}
	body, _ := json.Marshal(map[string]any{
		"kind": "materialized-governed-artifacts", "objects": ids,
	})
	if _, err := task.AppendEvent(state.EvL2Delivery, "l7", body, refs...); err != nil {
		return fmt.Errorf("%w: recording materialized artifacts: %v", ErrAssembly, err)
	}
	return nil
}

// resolveTaskEIS resolves the instruction set for ONE task. The
// orchestrator's governed roots are always present; an envelope
// carrying a verified skill procedure adds it as the ScopeSkill
// source. Byte verification against the envelope's stated pin happens
// inside the activation seam: L7 resolves no skill and consults no
// catalog — it verifies bytes against a hash it was given.
func (o *Orchestrator) resolveTaskEIS(env *Envelope) (*instructions.EffectiveSet, []byte, []instructions.Source, error) {
	sources := []instructions.Source{
		{Kind: instructions.ScopeHarnessSafety, Root: o.cfg.SafetyRoot},
		{Kind: instructions.ScopeHarnessSystem, Root: o.cfg.SystemRoot},
	}
	if o.cfg.ThemisRoot != "" {
		// The themis-domain root: the interpretive half of the L2
		// authority vocabulary (integration-audit D7).
		sources = append(sources, instructions.Source{Kind: instructions.ScopeThemisDomain, Root: o.cfg.ThemisRoot})
	}
	declaredProcedure := env.SkillProcedurePath != ""
	var procedureBytes []byte
	if declaredProcedure {
		// The no-symlink/regular-file property must hold at READ time,
		// not merely when the path was written — the same re-check the
		// repository activation path performs before reading.
		info, err := os.Lstat(env.SkillProcedurePath)
		if err != nil || !info.Mode().IsRegular() {
			return nil, nil, nil, fmt.Errorf("%w: skill procedure must be a regular file", ErrAssembly)
		}
		procedure, err := os.ReadFile(env.SkillProcedurePath)
		if err != nil {
			return nil, nil, nil, fmt.Errorf("%w: skill procedure: %v", ErrAssembly, err)
		}
		// The procedure becomes model INSTRUCTION text, so its identity is
		// verified against bytes L7 hashed itself — never by comparing two
		// envelope-supplied strings, which is integrity without binding
		// (the option-A state D-L9-11a rejected; M5 review HIGH-1).
		if c := env.Composition; c != nil {
			if err := c.verify("procedure", c.Procedure, hashBytes(procedure)); err != nil {
				return nil, nil, nil, err
			}
		}
		src, err := instructions.ActivateSkillSource(procedure, env.SkillProcedureSHA256)
		if err != nil {
			return nil, nil, nil, fmt.Errorf("%w: %v", ErrAssembly, err)
		}
		procedureBytes = procedure
		sources = append(sources, src)
	}
	eis, err := instructions.Resolve(instructions.Config{Policy: o.policy, TaskID: env.TaskID}, sources...)
	if err != nil {
		return nil, nil, nil, err
	}
	if declaredProcedure {
		// A declared procedure is a MANDATORY member of the composition,
		// unlike an optional repository instruction: if resolution
		// dropped it (directive-pattern rejection), the task would run
		// under an attribution asserting a composition whose instruction
		// half never reached the model. That is loss of attribution
		// integrity, so it fails closed here rather than completing
		// cleanly with an unfalsifiable record (L9 security MED-2).
		delivered := false
		for _, inst := range eis.Instructions {
			if inst.Scope == instructions.ScopeSkill {
				delivered = true
				break
			}
		}
		if !delivered {
			return nil, nil, nil, fmt.Errorf("%w: the declared skill procedure was not admitted into the instruction set (%d conflict(s)) — the submitted composition cannot be affirmed", ErrAssembly, len(eis.Conflicts))
		}
	}
	// Render at assembly, mirroring Open's rule: an unrenderable
	// instruction configuration fails the task here, not mid-walk.
	if _, _, err := eis.Render(o.policy); err != nil {
		return nil, nil, nil, fmt.Errorf("%w: %v", ErrAssembly, err)
	}
	return eis, procedureBytes, sources, nil
}

func checkControlVocabulary(reg *tools.Registry) error {
	// Two-way ⊆ (Q-L7-6): control-flagged registry entries must exist
	// in the constitution vocabulary; the executor-completeness half
	// is already enforced by NewExecutorTable.
	for _, t := range reg.Tools {
		if t.Control {
			if _, ok := controlVerbs[t.Name]; !ok {
				return fmt.Errorf("%w: registry control tool %q is not in the constitution vocabulary", ErrAssembly, t.Name)
			}
		}
	}
	return nil
}

func grantWithinCeiling(g *tools.Grant, c *WorkflowCeiling) error {
	allowed := map[string]bool{}
	for _, t := range c.AllowedTools {
		allowed[t] = true
	}
	for _, e := range g.Entries {
		if !allowed[e.Tool] {
			return fmt.Errorf("%w: grant tool %q exceeds the workflow ceiling", ErrAssembly, e.Tool)
		}
	}
	if g.TotalMaxCalls > c.MaxTotalCalls {
		return fmt.Errorf("%w: grant total %d exceeds workflow ceiling %d", ErrAssembly, g.TotalMaxCalls, c.MaxTotalCalls)
	}
	return nil
}

// instantiateGrant substitutes the @workspace placeholder with the
// provisioned root — ONLY in workspace fields, post-parse (security
// review LOW: a token anywhere else stays literal and fails its own
// validation) — and loads the effective grant through the normal
// fail-closed loader.
func instantiateGrant(raw []byte, wsRoot, stagingRoot string) (*tools.Grant, string, error) {
	// The placeholder guard below inspects the raw document by EXACT
	// key, while the loader matches keys case-insensitively: without
	// this wall a "Workspace" entry would pass the guard unseen and
	// bind a literal host path (security review CRITICAL-1).
	if err := strictjson.Check(raw); err != nil {
		return nil, "", fmt.Errorf("%w: envelope grant: %v", ErrAssembly, err)
	}
	var doc map[string]any
	if err := json.Unmarshal(raw, &doc); err != nil {
		return nil, "", fmt.Errorf("%w: envelope grant: %v", ErrAssembly, err)
	}
	if entries, ok := doc["entries"].([]any); ok {
		for _, e := range entries {
			if entry, ok := e.(map[string]any); ok {
				for k := range entry {
					switch k {
					case "tool", "max_calls", "workspace", "themis_scope", "mutating", "template_scope":
					default:
						return nil, "", fmt.Errorf("%w: envelope grant: entry key %q is not a grant field", ErrAssembly, k)
					}
				}
				if ws, ok := entry["workspace"].(string); ok {
					// Placeholder-only workspace bindings (architecture
					// review 2d): a literal path could bind a read tool
					// into the state root or any host directory — the
					// workspace is the one value only assembly may bind.
					if ws != "@workspace" {
						return nil, "", fmt.Errorf("%w: grant workspace bindings must use the @workspace placeholder, got %q", ErrAssembly, ws)
					}
					entry["workspace"] = wsRoot
				}
			}
		}
	}
	sub, err := json.Marshal(doc)
	if err != nil {
		return nil, "", fmt.Errorf("%w: %v", ErrAssembly, err)
	}
	// Staged under the state root, not shared host tmp: live
	// authority bytes stay inside the disjoint-root discipline every
	// other authority artifact obeys (integration-audit D10).
	f, err := os.CreateTemp(stagingRoot, "effective-grant-")
	if err != nil {
		return nil, "", fmt.Errorf("%w: %v", ErrAssembly, err)
	}
	path := f.Name()
	if _, err := f.Write(sub); err != nil {
		f.Close()
		os.Remove(path)
		return nil, "", fmt.Errorf("%w: %v", ErrAssembly, err)
	}
	f.Close()
	g, err := tools.LoadGrant(path)
	if err != nil {
		os.Remove(path)
		return nil, "", err
	}
	// Post-bind assertion: the only workspace a loaded effective grant
	// may carry is the one assembly bound. Whatever route a literal
	// took past the guard above, it does not reach a tool.
	for _, e := range g.Entries {
		if e.Workspace != "" && e.Workspace != wsRoot {
			os.Remove(path)
			return nil, "", fmt.Errorf("%w: grant %q carries workspace %q, not the assembly-bound root — a workspace is bound by assembly, never submitted", ErrInvariant, e.Tool, e.Workspace)
		}
	}
	return g, path, nil
}

// captureVerified reads the bytes to store as reconstruction evidence
// and PROVES they are the bytes the loader hashed. If the artifact
// changed between the loader's read and this one, the record would
// claim durability over bytes that never executed — identity over A
// with durability over B, the TOCTOU shape R-L9-2 forbids (final
// security review H-2).
//
// It is a function rather than inline code so the refusal is reachable
// on its own: the mismatch cannot be produced end-to-end without a
// second process writing between two reads inside one call, and a
// production seam admitting that write would be a worse trade than
// testing the control directly. The positive half — that the stored
// bytes really are the loaded ones on the wired paths — is established
// end-to-end by TestDurableBytesMustMatchWhatWasLoaded.
func captureVerified(label, path, loaderHash string) ([]byte, error) {
	body, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("%w: %s: %v", ErrAssembly, label, err)
	}
	if hashBytes(body) != loaderHash {
		return nil, fmt.Errorf("%w: %s changed between loading and durable capture", ErrInvariant, label)
	}
	return body, nil
}

// grantAuthorityDigest hashes exactly what a grant PERMITS — tools,
// per-tool caps, workspace bindings, themis-id scopes, mutating flags,
// and the aggregate — excluding the task identity that legitimately
// differs between two otherwise identical tasks. Recording it makes
// "same authority?" a question the record can answer directly.
//
// Every field that gates a tool call at authorization time belongs
// here. ThemisScope decides which Themis identifiers a themis-id tool
// may reach (tools/authorize.go), so two grants differing only in
// their scope prefixes confer materially different authority; omitting
// it would let a widened scope record an unchanged digest, which is
// precisely the substitution this digest exists to expose.
func grantAuthorityDigest(g *tools.Grant) string {
	entries := append([]tools.GrantEntry(nil), g.Entries...)
	sort.Slice(entries, func(i, j int) bool { return entries[i].Tool < entries[j].Tool })
	var b strings.Builder
	fmt.Fprintf(&b, "v%d;total=%d;", g.Version, g.TotalMaxCalls)
	for _, e := range entries {
		// The workspace path is per-task by construction (L7 binds it to
		// the provisioned root), so the digest records only WHETHER an
		// entry is workspace-scoped, not which instance.
		// TemplateScope is authority over WHICH delegation templates a
		// delegate entry may name (D-SA-4 / L8 M2): the 2026-09-15 class
		// of defect was a digest blind to a scope field, so it joins the
		// digest the day the field exists.
		fmt.Fprintf(&b, "%s:%d:%t:%t:%s:%s;", e.Tool, e.MaxCalls, e.Workspace != "", e.Mutating, scopeDigest(e.ThemisScope), scopeDigest(e.TemplateScope))
	}
	return hashBytes([]byte(b.String()))
}

// scopeDigest encodes a themis-id scope as the SET of prefixes it
// permits. Order and repetition do not change what a scope authorizes,
// so they must not change the digest — otherwise two authority-identical
// grants would read as a change. Each prefix is length-prefixed because
// prefixes are author-supplied text: without it, ["a;b"] and ["a","b"]
// would encode alike and a scope could be widened without moving the
// digest.
func scopeDigest(scope []string) string {
	if len(scope) == 0 {
		return "-"
	}
	uniq := append([]string(nil), scope...)
	sort.Strings(uniq)
	var b strings.Builder
	var prev string
	for i, p := range uniq {
		if i > 0 && p == prev {
			continue
		}
		prev = p
		fmt.Fprintf(&b, "%d|%s,", len(p), p)
	}
	return hashBytes([]byte(b.String()))
}

func hashBytes(b []byte) string {
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}

// modelRegistryHash is the anchor-pinned model registry identity the
// record names for a delegation's governed model identity (C-L8-10):
// the pin, "absent" when the anchor declares none, "unanchored" in the
// test-harness caller role.
func (o *Orchestrator) modelRegistryHash() string {
	if o.anchor != nil {
		return o.anchor.ModelRegistry
	}
	return "unanchored"
}
