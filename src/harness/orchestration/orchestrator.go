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

	l2 "github.com/tofchaliss/themis/context"
	"github.com/tofchaliss/themis/execution"
	"github.com/tofchaliss/themis/instructions"
	"github.com/tofchaliss/themis/runtime/model"
	"github.com/tofchaliss/themis/state"
	"github.com/tofchaliss/themis/tools"
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
	PolicyPath string
	// Model is the model.Interface provider (injected so the loop is
	// provider-agnostic and Register-testable with a scripted model).
	Model model.Interface
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
	eis, err := instructions.Resolve(instructions.Config{Policy: policy},
		instructions.Source{Kind: instructions.ScopeHarnessSafety, Root: cfg.SafetyRoot},
		instructions.Source{Kind: instructions.ScopeHarnessSystem, Root: cfg.SystemRoot},
	)
	if err != nil {
		return nil, nil, err
	}
	// Render once at Open: an unrenderable L1 configuration fails the
	// orchestrator here, not mid-walk (delivery itself re-renders
	// through the L2 composer's SystemMessage seam per composition).
	if _, _, err := eis.Render(policy); err != nil {
		return nil, nil, err
	}
	o := &Orchestrator{cfg: cfg, root: root, store: store, prov: prov, eis: eis, policy: policy}

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
	// The EXACT bytes each loader accepted, captured once for durable
	// storage after the task exists (ADG-L9/L6-1 option (i)). Read here
	// and carried — never re-read after verification, which would
	// reintroduce the second-materialization problem this closes.
	materialized := map[string][]byte{}
	for label, path := range map[string]string{
		"workflow": env.WorkflowPath, "workflow_ceiling": env.WorkflowCeilingPath,
		"context_contract": env.ContextContractPath, "spec": env.SpecPath,
	} {
		body, rerr := os.ReadFile(path)
		if rerr != nil {
			return res, fmt.Errorf("%w: %s: %v", ErrAssembly, label, rerr)
		}
		materialized[label] = body
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
	for _, t := range reg.Tools {
		registered[t.Name] = true
	}
	for _, p := range wf.Phases {
		for _, c := range p.Capabilities {
			if !registered[c] {
				return res, fmt.Errorf("%w: workflow capability %q is not in the registry", ErrAssembly, c)
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
	grant, effPath, err := instantiateGrant(rawGrant, ws.Root)
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
	table, err := tools.NewExecutorTable(reg, nil)
	if err != nil {
		envn.Teardown()
		return res, err
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
	eis, err := o.resolveTaskEIS(env)
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
	}
	// Opaque attribution: recorded verbatim, interpreted by nobody
	// (D-L9-13). L7 does not know what a skill is; it only preserves
	// what the envelope stated, so the record can be checked against
	// the catalog after the fact.
	for k, v := range env.Origin {
		governed["origin:"+k] = v
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
		return res, fmt.Errorf("%w: the grant changed between attribution and execution", ErrInvariant)
	}
	// Reconstruction provenance (R-L9-1 / ADG-L9/L6-1): store the EXACT
	// bytes verified at assembly as L6 content-addressed objects, and
	// reference them from the record. L6 derives each object's address
	// by hashing the bytes itself, so the recorded identity cannot have
	// come from an envelope claim — which is the independent source an
	// equality assertion could never supply. The bytes are the ones
	// already captured and verified; nothing is re-read.
	if err := recordMaterializedArtifacts(task, materialized); err != nil {
		return res, err
	}

	w := &walk{
		o: o, env: env, wf: wf, reg: reg, grant: grant, table: table,
		task: task, l5: envn, execCeiling: execCeiling, spec: spec,
		contract: contract, eis: eis,
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
func (o *Orchestrator) resolveTaskEIS(env *Envelope) (*instructions.EffectiveSet, error) {
	sources := []instructions.Source{
		{Kind: instructions.ScopeHarnessSafety, Root: o.cfg.SafetyRoot},
		{Kind: instructions.ScopeHarnessSystem, Root: o.cfg.SystemRoot},
	}
	declaredProcedure := env.SkillProcedurePath != ""
	if declaredProcedure {
		// The no-symlink/regular-file property must hold at READ time,
		// not merely when the path was written — the same re-check the
		// repository activation path performs before reading.
		info, err := os.Lstat(env.SkillProcedurePath)
		if err != nil || !info.Mode().IsRegular() {
			return nil, fmt.Errorf("%w: skill procedure must be a regular file", ErrAssembly)
		}
		procedure, err := os.ReadFile(env.SkillProcedurePath)
		if err != nil {
			return nil, fmt.Errorf("%w: skill procedure: %v", ErrAssembly, err)
		}
		// The procedure becomes model INSTRUCTION text, so its identity is
		// verified against bytes L7 hashed itself — never by comparing two
		// envelope-supplied strings, which is integrity without binding
		// (the option-A state D-L9-11a rejected; M5 review HIGH-1).
		if c := env.Composition; c != nil {
			if err := c.verify("procedure", c.Procedure, hashBytes(procedure)); err != nil {
				return nil, err
			}
		}
		src, err := instructions.ActivateSkillSource(procedure, env.SkillProcedureSHA256)
		if err != nil {
			return nil, fmt.Errorf("%w: %v", ErrAssembly, err)
		}
		sources = append(sources, src)
	}
	eis, err := instructions.Resolve(instructions.Config{Policy: o.policy, TaskID: env.TaskID}, sources...)
	if err != nil {
		return nil, err
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
			return nil, fmt.Errorf("%w: the declared skill procedure was not admitted into the instruction set (%d conflict(s)) — the reviewed composition cannot be affirmed", ErrAssembly, len(eis.Conflicts))
		}
	}
	// Render at assembly, mirroring Open's rule: an unrenderable
	// instruction configuration fails the task here, not mid-walk.
	if _, _, err := eis.Render(o.policy); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrAssembly, err)
	}
	return eis, nil
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
func instantiateGrant(raw []byte, wsRoot string) (*tools.Grant, string, error) {
	var doc map[string]any
	if err := json.Unmarshal(raw, &doc); err != nil {
		return nil, "", fmt.Errorf("%w: envelope grant: %v", ErrAssembly, err)
	}
	if entries, ok := doc["entries"].([]any); ok {
		for _, e := range entries {
			if entry, ok := e.(map[string]any); ok {
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
	f, err := os.CreateTemp("", "effective-grant-")
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
	return g, path, nil
}

// grantAuthorityDigest hashes exactly what a grant PERMITS — tools,
// per-tool caps, workspace bindings, mutating flags, and the aggregate
// — excluding the task identity that legitimately differs between two
// otherwise identical tasks. Recording it makes "same authority?" a
// question the record can answer directly.
func grantAuthorityDigest(g *tools.Grant) string {
	entries := append([]tools.GrantEntry(nil), g.Entries...)
	sort.Slice(entries, func(i, j int) bool { return entries[i].Tool < entries[j].Tool })
	var b strings.Builder
	fmt.Fprintf(&b, "v%d;total=%d;", g.Version, g.TotalMaxCalls)
	for _, e := range entries {
		// The workspace path is per-task by construction (L7 binds it to
		// the provisioned root), so the digest records only WHETHER an
		// entry is workspace-scoped, not which instance.
		fmt.Fprintf(&b, "%s:%d:%t:%t;", e.Tool, e.MaxCalls, e.Workspace != "", e.Mutating)
	}
	return hashBytes([]byte(b.String()))
}

func hashBytes(b []byte) string {
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}
