package skills

// Instantiation: the envelope compiler (D-L9-13). {skill@version,
// caller inputs} → an ordinary governed envelope plus effective
// grant/spec artifacts. Everything here is pre-submission machinery:
// the closed four-class surface (D-L9-7), downward-only narrowing
// (D-L9-5), placeholder-only field-scoped substitution (D-L9-4), and
// zero authority — L7 gives the output no trust discount and
// re-validates everything.

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/tofchaliss/themis/confine"
	"github.com/tofchaliss/themis/execution"
	"github.com/tofchaliss/themis/state"
)

func hashBytes(b []byte) string {
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}

// Deployment carries the Class-4 inputs (D-L9-7): infrastructure
// identity from the operator's governed configuration — never the
// skill's business, never the caller's authority.
type Deployment struct {
	Model           string
	TurnTimeoutSec  int
	RegistryPath    string
	ExecCeilingPath string
	// StateRoot/ArtifactDir/WorkspaceRoot: the task-writable roots the
	// catalog must be disjoint from (D-L9-8 wall 2). Operator-supplied
	// like the rest of Class 4; the workspace root may be empty when
	// the provider assigns it per task, in which case its parent
	// (ProviderDir) is the meaningful root to compare.
	StateRoot     string
	ArtifactDir   string
	WorkspaceRoot string
}

// checkCatalogDisjoint enforces wall 2 with the same physical,
// case-folded predicate L7 uses for its own roots — one confinement
// judgment shared, never a second implementation.
func (d Deployment) checkCatalogDisjoint(catalogRoot string) error {
	abs, err := filepath.Abs(catalogRoot)
	if err != nil {
		return fmt.Errorf("%w: catalog root: %v", ErrCatalog, err)
	}
	// Every task-writable root is REQUIRED: the workspace is the root a
	// task actually holds write_file on, so omitting it would leave the
	// wall satisfied by luck (security MED-1 / architecture HIGH). The
	// package's own posture is that nothing is defaulted.
	roots := []string{abs}
	for name, r := range map[string]string{
		"state root": d.StateRoot, "artifact dir": d.ArtifactDir, "workspace root": d.WorkspaceRoot,
	} {
		if r == "" {
			return fmt.Errorf("%w: %s is required to prove catalog disjointness — nothing is defaulted", ErrCatalog, name)
		}
		ra, err := filepath.Abs(r)
		if err != nil {
			return fmt.Errorf("%w: %v", ErrCatalog, err)
		}
		roots = append(roots, ra)
	}
	if err := state.CheckDisjointRoots(roots...); err != nil {
		return fmt.Errorf("%w: the skill catalog must be disjoint from every task-writable root: %v", ErrCatalog, err)
	}
	return nil
}

// Request is the closed instantiation surface. Class 3 (task state):
// TaskID, RetryOf, Repo, PinnedSHA, Inputs. Class 2 (downward-only
// bounds): QuotaOverrides, WallDeadlineS. Anything else a caller
// might want is not a field — refused by unrepresentability.
type Request struct {
	TaskID    string
	RetryOf   string
	Repo      string
	PinnedSHA string
	Inputs    map[string]any

	QuotaOverrides map[string]int // tool → narrowed max_calls (≤ skill bound)
	WallDeadlineS  int            // 0 = the skill's bound; >0 must be ≤ it

	Deployment Deployment
	OutDir     string
}

// pinnedSHASyntax matches L5's provisioning contract exactly (a full
// 40-hex commit SHA — abbreviations and branch names are not pins).
// L9 must never emit an envelope L7/L5 will refuse, so the check is
// the same one, not a looser cousin.
var pinnedSHASyntax = regexp.MustCompile(`^[0-9a-f]{40}$`)

// repoSyntax is the L5 spec's repository-name rule verbatim; dot
// segments are refused separately below, exactly as L5 does.
// Validating here keeps the fail-closed posture at every layer rather
// than relying on the downstream loader to catch it.
var repoSyntax = regexp.MustCompile(`^[a-zA-Z0-9._-]+(/[a-zA-Z0-9._-]+)*$`)

// Instantiate resolves one exact skill reference against one atomic
// catalog state and compiles the ordinary governed envelope. The
// walk-facing artifacts (workflow, ceiling, contract) are referenced
// at their resolved catalog paths — byte-verified here against their
// pins, and hashed again independently by L7 at assembly.
func Instantiate(catalogPath, ref string, req Request) (string, error) {
	cat, err := LoadCatalog(catalogPath)
	if err != nil {
		return "", err
	}
	entry, m, err := cat.Resolve(ref)
	if err != nil {
		return "", err
	}
	// The task id becomes part of the artifact filenames written below,
	// so it is validated against L6's OWN predicate before it can steer
	// a path (security review HIGH-1: an unvalidated id reached
	// filepath.Join and could overwrite sibling governed artifacts).
	if !state.ValidTaskID(req.TaskID) {
		return "", fmt.Errorf("%w: task_id %q is not a valid task identity", ErrResolve, req.TaskID)
	}
	if req.RetryOf != "" && !state.ValidTaskID(req.RetryOf) {
		return "", fmt.Errorf("%w: retry_of %q is not a valid task identity", ErrResolve, req.RetryOf)
	}
	if !repoSyntax.MatchString(req.Repo) {
		return "", fmt.Errorf("%w: bad repository name %q", ErrResolve, req.Repo)
	}
	for _, seg := range strings.Split(req.Repo, "/") {
		if seg == "." || seg == ".." {
			return "", fmt.Errorf("%w: traversal segment in repository name %q", ErrResolve, req.Repo)
		}
	}
	if !pinnedSHASyntax.MatchString(req.PinnedSHA) {
		return "", fmt.Errorf("%w: pinned_sha must be a full 40-hex commit SHA, got %q — branch names are not pins", ErrResolve, req.PinnedSHA)
	}
	d := req.Deployment
	if d.Model == "" || d.TurnTimeoutSec <= 0 || d.RegistryPath == "" || d.ExecCeilingPath == "" {
		return "", fmt.Errorf("%w: deployment inputs (model, turn timeout, registry, exec ceiling) are operator-governed and required", ErrResolve)
	}
	if req.OutDir == "" {
		return "", fmt.Errorf("%w: output directory required", ErrResolve)
	}
	// D-L9-8 wall 2: the governed catalog root is disjoint from every
	// task-writable root. A catalog reachable from a workspace (or
	// from the state/artifact roots) would make registration
	// indirectly writable by a task holding write_file — the wall must
	// hold structurally, at the designed call site, using the same
	// predicate L7 uses for its own roots.
	if err := req.Deployment.checkCatalogDisjoint(cat.Root()); err != nil {
		return "", err
	}

	// Resolve every pin: bytes verified against the reviewed hashes
	// BEFORE anything becomes executable (D-L9-11).
	wfPath, _, err := m.resolvePin("workflow", m.Workflow)
	if err != nil {
		return "", err
	}
	wcPath, _, err := m.resolvePin("workflow_ceiling", m.WorkflowCeiling)
	if err != nil {
		return "", err
	}
	ccPath, _, err := m.resolvePin("context_contract", m.ContextContract)
	if err != nil {
		return "", err
	}
	_, grantRaw, err := m.resolvePin("grant_template", m.GrantTemplate)
	if err != nil {
		return "", err
	}
	_, specRaw, err := m.resolvePin("spec_template", m.SpecTemplate)
	if err != nil {
		return "", err
	}
	_, schemaRaw, err := m.resolvePin("input_schema", m.InputSchema)
	if err != nil {
		return "", err
	}
	procPath, _, err := m.resolvePin("procedure", m.Procedure)
	if err != nil {
		return "", err
	}

	schema, err := parseSchema(schemaRaw)
	if err != nil {
		return "", err
	}
	if err := schema.Validate(req.Inputs); err != nil {
		return "", err
	}
	// The payload is the caller's typed inputs, serialized
	// deterministically — external-untrusted data selecting the
	// subject, fenced by L2 at delivery. Caller input NEVER touches
	// procedure text, workflow, contract, or ceiling (D-L9-4).
	payloadBytes, err := json.Marshal(req.Inputs)
	if err != nil {
		return "", fmt.Errorf("%w: %v", ErrInputs, err)
	}

	effGrant, err := instantiateGrantTemplate(grantRaw, req.TaskID, req.QuotaOverrides)
	if err != nil {
		return "", err
	}
	effSpec, err := instantiateSpecTemplate(specRaw, req.TaskID, req.Repo, req.PinnedSHA, req.WallDeadlineS)
	if err != nil {
		return "", err
	}

	if err := os.MkdirAll(req.OutDir, 0o755); err != nil {
		return "", fmt.Errorf("%w: %v", ErrResolve, err)
	}
	// Every write resolves through the canonical mutation predicate: it
	// refuses traversal, symlinked parents, and non-regular targets.
	// filepath.Join refuses nothing (security review HIGH-1).
	grantPath, err := confine.CreatePath(req.OutDir, req.TaskID+"-grant.json")
	if err != nil {
		return "", fmt.Errorf("%w: effective grant path: %v", ErrResolve, err)
	}
	specPath, err := confine.CreatePath(req.OutDir, req.TaskID+"-spec.json")
	if err != nil {
		return "", fmt.Errorf("%w: effective spec path: %v", ErrResolve, err)
	}
	if err := os.WriteFile(grantPath, effGrant, 0o644); err != nil {
		return "", fmt.Errorf("%w: %v", ErrResolve, err)
	}
	if err := os.WriteFile(specPath, effSpec, 0o644); err != nil {
		return "", fmt.Errorf("%w: %v", ErrResolve, err)
	}
	// Re-validate the substituted SPEC through its own governed loader
	// (the L7 instantiateGrant precedent): a substitution producing a
	// structurally invalid artifact must fail here, at the
	// substitution site, not later inside SubmitTask.
	effSpecLoaded, err := execution.LoadSpec(specPath)
	if err != nil {
		return "", fmt.Errorf("%w: effective spec is invalid after substitution: %v", ErrResolve, err)
	}
	if effSpecLoaded.TaskID != req.TaskID {
		return "", fmt.Errorf("%w: effective spec does not bind to task %q", ErrResolve, req.TaskID)
	}
	// The grant deliberately CANNOT be loaded here: it still carries
	// the "@workspace" placeholder, and a workspace does not exist
	// until L7 provisions one. Binding it is L7's designed act
	// (orchestrator.instantiateGrant), so L9 validates what it can —
	// structure, placeholders, downward-only quotas — and records the
	// substituted bytes' identity rather than a loaded grant's hash.
	// This is the honest split, not a skipped check.
	if err := checkGrantShape(effGrant, req.TaskID); err != nil {
		return "", err
	}

	abs := func(p string) string {
		a, aerr := filepath.Abs(p)
		if aerr != nil {
			return p
		}
		return a
	}
	env := map[string]any{
		"version":          1,
		"task_id":          req.TaskID,
		"model":            d.Model,
		"turn_timeout_sec": d.TurnTimeoutSec,
		"payload":          string(payloadBytes),

		"workflow_path":         abs(wfPath),
		"workflow_ceiling_path": abs(wcPath),
		"registry_path":         abs(d.RegistryPath),
		"grant_path":            abs(grantPath),
		"exec_ceiling_path":     abs(d.ExecCeilingPath),
		"spec_path":             abs(specPath),
		"context_contract_path": abs(ccPath),

		// The activated skill-instruction artifact (Q-L1-1 IOU closed
		// via ActivateSkillSource): pinned bytes, verified again by L7.
		"skill_procedure_path":   abs(procPath),
		"skill_procedure_sha256": m.Procedure.SHA256,

		// The composition commitment (D-L9-11a / C2): the identities L9
		// resolved and verified, carried across the seam so L7's own
		// materialization is checked against them rather than against a
		// pair the envelope author chose freely. Identities only — the
		// artifacts themselves stay on disk.
		"composition": map[string]string{
			"workflow_sha256":         m.Workflow.SHA256,
			"workflow_ceiling_sha256": m.WorkflowCeiling.SHA256,
			"context_contract_sha256": m.ContextContract.SHA256,
			"procedure_sha256":        m.Procedure.SHA256,
		},

		// Opaque provenance (D-L9-13): mandatory for skill-produced
		// envelopes, preserved verbatim by L7, interpreted by nobody.
		// Template AND effective identities travel together (D-L9-7,
		// extending the L7 grant_envelope/grant_effective pattern), so
		// the record shows both what was reviewed and what executed.
		"origin": map[string]string{
			"skill":             fmt.Sprintf("%s@%d", entry.Name, entry.Version),
			"skill_composition": m.CompositionHash,
			"skill_catalog":     cat.Hash,
			// Template identity is the reviewed pin; effective identity
			// is the substituted bytes L9 actually emitted. For the
			// grant, L7 records its own grant_effective after binding
			// the workspace — the two together span the whole chain.
			"grant_template":         m.GrantTemplate.SHA256,
			"grant_instantiated":     hashBytes(effGrant),
			"spec_template":          m.SpecTemplate.SHA256,
			"spec_effective":         effSpecLoaded.Hash,
			"skill_procedure":        m.Procedure.SHA256,
			"skill_workflow":         m.Workflow.SHA256,
			"skill_workflow_ceiling": m.WorkflowCeiling.SHA256,
			"skill_context_contract": m.ContextContract.SHA256,
			"skill_input_schema":     m.InputSchema.SHA256,
		},
	}
	if req.RetryOf != "" {
		env["retry_of"] = req.RetryOf
	}
	envBytes, err := json.MarshalIndent(env, "", " ")
	if err != nil {
		return "", fmt.Errorf("%w: %v", ErrResolve, err)
	}
	envPath, err := confine.CreatePath(req.OutDir, "envelope-"+req.TaskID+".json")
	if err != nil {
		return "", fmt.Errorf("%w: envelope path: %v", ErrResolve, err)
	}
	if err := os.WriteFile(envPath, envBytes, 0o644); err != nil {
		return "", fmt.Errorf("%w: %v", ErrResolve, err)
	}
	return envPath, nil
}

// instantiateGrantTemplate binds @task_id and applies downward-only
// quota narrowing. Placeholder-only, field-scoped (the L7 @workspace
// precedent): a template carrying a literal task_id is refused — the
// substitutable fields are exactly the declared ones.
func instantiateGrantTemplate(raw []byte, taskID string, overrides map[string]int) ([]byte, error) {
	var doc map[string]any
	if err := json.Unmarshal(raw, &doc); err != nil {
		return nil, fmt.Errorf("%w: grant template: %v", ErrResolve, err)
	}
	if doc["task_id"] != "@task_id" {
		return nil, fmt.Errorf("%w: grant template task_id must be the @task_id placeholder", ErrResolve)
	}
	doc["task_id"] = taskID
	applied := map[string]bool{}
	if entries, ok := doc["entries"].([]any); ok {
		for _, e := range entries {
			entry, ok := e.(map[string]any)
			if !ok {
				return nil, fmt.Errorf("%w: grant template has a malformed entry", ErrResolve)
			}
			tool, _ := entry["tool"].(string)
			if narrowed, want := overrides[tool]; want {
				bound, ok := entry["max_calls"].(float64)
				if !ok {
					return nil, fmt.Errorf("%w: grant template entry %q has no numeric max_calls to narrow", ErrResolve, tool)
				}
				if narrowed < 1 || float64(narrowed) > bound {
					return nil, fmt.Errorf("%w: quota for %q may only narrow: %d must be within [1, %d]", ErrResolve, tool, narrowed, int(bound))
				}
				entry["max_calls"] = narrowed
				applied[tool] = true
			}
		}
	}
	for tool := range overrides {
		if !applied[tool] {
			return nil, fmt.Errorf("%w: quota override for %q matches no grant entry — the surface is closed", ErrResolve, tool)
		}
	}
	// Narrowing is complete, not partial (D-L9-5): the aggregate cap
	// comes down with the per-tool caps so a narrowed grant cannot
	// retain a stale, unreachable total. It only ever decreases.
	if entries, ok := doc["entries"].([]any); ok {
		sum := 0
		for _, e := range entries {
			if entry, ok := e.(map[string]any); ok {
				switch v := entry["max_calls"].(type) {
				case float64:
					sum += int(v)
				case int:
					sum += v
				}
			}
		}
		if total, ok := doc["total_max_calls"].(float64); ok && float64(sum) < total {
			doc["total_max_calls"] = sum
		}
	}
	return json.Marshal(doc)
}

// checkGrantShape validates everything about the substituted grant
// that does not require a provisioned workspace: identity binding,
// positive caps, an aggregate cap, and the invariant that every
// workspace binding is still the placeholder L7 will bind (a literal
// path here would be an L9-authored scope, which L9 may never mint).
func checkGrantShape(effGrant []byte, taskID string) error {
	var doc struct {
		Version       int    `json:"version"`
		TaskID        string `json:"task_id"`
		TotalMaxCalls int    `json:"total_max_calls"`
		Entries       []struct {
			Tool      string `json:"tool"`
			MaxCalls  int    `json:"max_calls"`
			Workspace string `json:"workspace"`
		} `json:"entries"`
	}
	if err := json.Unmarshal(effGrant, &doc); err != nil {
		return fmt.Errorf("%w: effective grant is invalid after substitution: %v", ErrResolve, err)
	}
	if doc.Version < 1 || doc.TaskID != taskID {
		return fmt.Errorf("%w: effective grant is invalid after substitution: task binding", ErrResolve)
	}
	if doc.TotalMaxCalls <= 0 {
		return fmt.Errorf("%w: effective grant is invalid after substitution: total_max_calls must be positive", ErrResolve)
	}
	if len(doc.Entries) == 0 {
		return fmt.Errorf("%w: effective grant is invalid after substitution: no entries", ErrResolve)
	}
	sum := 0
	for _, e := range doc.Entries {
		if e.Tool == "" || e.MaxCalls <= 0 {
			return fmt.Errorf("%w: effective grant is invalid after substitution: entry %q needs a positive cap", ErrResolve, e.Tool)
		}
		if e.Workspace != "" && e.Workspace != "@workspace" {
			return fmt.Errorf("%w: effective grant is invalid after substitution: workspace binding must remain the @workspace placeholder — L9 never mints a scope", ErrResolve)
		}
		sum += e.MaxCalls
	}
	// The aggregate can never exceed what the per-tool caps allow: an
	// unreachable total is a stale bound, and after narrowing it must
	// have come down with them (D-L9-5 narrowing is complete).
	if doc.TotalMaxCalls > sum {
		return fmt.Errorf("%w: effective grant total_max_calls %d exceeds the sum of its per-tool caps %d — narrowing must be complete", ErrResolve, doc.TotalMaxCalls, sum)
	}
	return nil
}

// instantiateSpecTemplate binds the Class-3 subject fields and
// applies the downward-only wall-deadline narrowing.
func instantiateSpecTemplate(raw []byte, taskID, repo, pinnedSHA string, wallS int) ([]byte, error) {
	var doc map[string]any
	if err := json.Unmarshal(raw, &doc); err != nil {
		return nil, fmt.Errorf("%w: spec template: %v", ErrResolve, err)
	}
	for field, val := range map[string]string{"task_id": "@task_id", "repo": "@repo", "pinned_sha": "@pinned_sha"} {
		if doc[field] != val {
			return nil, fmt.Errorf("%w: spec template %s must be the %s placeholder", ErrResolve, field, val)
		}
	}
	doc["task_id"] = taskID
	doc["repo"] = repo
	doc["pinned_sha"] = pinnedSHA
	if wallS > 0 {
		limits, ok := doc["limits"].([]any)
		if !ok {
			return nil, fmt.Errorf("%w: spec template declares no limits to narrow", ErrResolve)
		}
		narrowed := false
		for _, l := range limits {
			limit, ok := l.(map[string]any)
			if !ok {
				// A malformed limit entry is a refusal, never a skip:
				// silently ignoring it could leave the dimension the
				// caller asked to narrow un-narrowed.
				return nil, fmt.Errorf("%w: spec template has a malformed limit entry", ErrResolve)
			}
			if limit["dimension"] == "wall_deadline_s" {
				if narrowed {
					// The template declares the dimension twice; L5
					// refuses duplicates, so refuse here rather than
					// narrow one and leave the other.
					return nil, fmt.Errorf("%w: spec template declares wall_deadline_s more than once", ErrResolve)
				}
				bound, ok := limit["value"].(float64)
				if !ok {
					return nil, fmt.Errorf("%w: wall_deadline_s bound is not numeric", ErrResolve)
				}
				if float64(wallS) > bound {
					return nil, fmt.Errorf("%w: wall deadline may only narrow: %d exceeds the skill bound %d", ErrResolve, wallS, int(bound))
				}
				limit["value"] = wallS
				narrowed = true
			}
		}
		if !narrowed {
			return nil, fmt.Errorf("%w: the skill declares no wall_deadline_s bound to narrow", ErrResolve)
		}
	}
	return json.Marshal(doc)
}
