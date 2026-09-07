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
	cfg   Config
	root  *state.Root
	store *execution.ArtifactStore
	prov  *execution.LocalProvider
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

	// Single-use identity + full governed attribution.
	task, err := o.root.CreateTask(env.TaskID, state.TaskOptions{
		RetryOf: env.RetryOf,
		GovernedHashes: map[string]string{
			"envelope": env.Hash, "workflow": wf.Hash, "workflow_ceiling": wfCeiling.Hash,
			"registry": reg.Hash, "grant_envelope": hashBytes(rawGrant), "grant_effective": grant.Hash,
			"exec_ceiling": execCeiling.Hash, "spec": spec.Hash, "context_contract": contract.Hash,
			"l1_eis": o.eis.Hash, "l1_policy": o.eis.PolicyHash,
			"l6_constitution": state.ConstitutionHash(), "l7_constitution": ConstitutionHash(),
		},
	})
	if err != nil {
		envn.Teardown()
		return res, err
	}
	defer task.Close()

	w := &walk{
		o: o, env: env, wf: wf, reg: reg, grant: grant, table: table,
		task: task, l5: envn, execCeiling: execCeiling, spec: spec,
		contract: contract,
	}
	return w.run()
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

func hashBytes(b []byte) string {
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}
