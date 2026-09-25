package seam

// Register B under an ANCHORED deployment (design.md §5.5; M6): the
// anchor pins the delegation-template registry; a genuine delegate
// call from the proposed remediate-dependency@2 Skill, instantiated
// through L9, is admitted and executed; the model forms its evidence
// reference from the record-ref furniture it saw. Then the anchor-side
// negatives, each mutation-coupled both ways. Register E (live) at the
// end: the unmodified loop against a local model.

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"

	hctx "github.com/tofchaliss/themis/context"
	"github.com/tofchaliss/themis/deployment"
	"github.com/tofchaliss/themis/instructions"
	"github.com/tofchaliss/themis/orchestration"
	"github.com/tofchaliss/themis/runtime/model"
	"github.com/tofchaliss/themis/skills"
	"github.com/tofchaliss/themis/state"
	"github.com/tofchaliss/themis/tools"
	vseam "github.com/tofchaliss/themis/verification/seam"
)

type anchoredWorld struct {
	o          *orchestration.Orchestrator
	sroot      *state.Root
	base       string
	root       string
	mirror     string
	sha        string
	registry   string // delegation registry path pinned by the anchor
	ceiling    string
	catalog    string
	anchorPath string
}

// newAnchoredWorld builds a deployment anchor over the governed
// repository artifacts (registry-v5, the @2 bundle, the delegation
// registry at registryPath) and opens an anchored orchestrator with
// the verifier and delegation seams wired. mutate edits the anchor
// map; wire controls whether the delegation seam/path are configured.
func newAnchoredWorld(t *testing.T, m model.Interface, modelName, registryPath string, mutate func(map[string]any), wire bool, seamRegistry ...string) (*anchoredWorld, error) {
	t.Helper()
	root := repoRoot(t)
	base := t.TempDir()
	mirror, sha := mkMirror(t)
	w := &anchoredWorld{base: base, root: root, mirror: mirror, sha: sha, registry: registryPath}
	if w.registry == "" {
		w.registry = filepath.Join(root, "policies/delegation/registry.json")
	}
	w.catalog = filepath.Join(root, "policies/skills/catalog.json")
	w.ceiling = wj(t, base, "execution-ceiling.json",
		`{"version":1,"mirror_root":"`+mirror+`","max_wall_deadline_sec":600,"max_file_bytes":1048576,"max_total_bytes":10485760,"max_file_count":500,"max_mem_bytes":1073741824,"max_cpu_time_sec":600,"max_proc_count":64}`)
	hashDir := func(p string) string {
		h, err := deployment.HashDir(p)
		if err != nil {
			t.Fatal(err)
		}
		return h
	}
	hashFile := func(p string) string {
		h, err := deployment.HashFile(p)
		if err != nil {
			t.Fatal(err)
		}
		return h
	}
	bundle := filepath.Join(root, "policies/skills/remediate-dependency-2")
	a := map[string]any{
		"version": 1, "name": "test-rsys", "deployment_version": 4,
		"instruction_root_safety": hashDir(filepath.Join(root, "instructions/global/safety")),
		"instruction_root_system": hashDir(filepath.Join(root, "instructions/global/system")),
		"instruction_root_themis": hashDir(filepath.Join(root, "instructions/themis")),
		"instruction_policy":      hashFile(filepath.Join(root, "policies/security/instruction-directive-patterns.json")),
		"tool_registry":           hashFile(filepath.Join(root, "policies/tools/registry-v5.json")),
		"constitution":            map[string]any{"state": state.ConstitutionHash(), "orchestration": orchestration.ConstitutionHash()},
		"execution_ceiling":       hashFile(w.ceiling),
		"workflows": []any{map[string]any{
			"workflow":         hashFile(filepath.Join(bundle, "workflow.json")),
			"workflow_ceiling": hashFile(filepath.Join(bundle, "ceiling.json")),
			"context_contract": hashFile(filepath.Join(bundle, "contract.json")),
		}},
		"models":                       []any{modelName},
		"model_registry":               "absent",
		"skill_catalog":                hashFile(w.catalog),
		"skills":                       []any{"remediate-dependency@2"},
		"contract_registry":            hashFile(filepath.Join(root, "policies/verification/contracts.json")),
		"criteria_registry":            hashFile(filepath.Join(root, "policies/ratchet/criteria.json")),
		"regression_set_registry":      hashFile(filepath.Join(root, "policies/ratchet/regression-sets.json")),
		"delegation_template_registry": hashFile(w.registry),
		"themis_store":                 "absent",
	}
	if mutate != nil {
		mutate(a)
	}
	ab, _ := json.Marshal(a)
	w.anchorPath = wj(t, base, "anchor.json", string(ab))
	sum := sha256.Sum256(ab)
	anchorSHA := hex.EncodeToString(sum[:])
	regs := wj(t, base, "anchors.json", `{"version":1,"kind":"deployment-anchors","entries":[{"name":"test-rsys","version":4,"artifact_sha256":"`+anchorSHA+`","state":"active","steward":"t"}]}`)

	l4, err := tools.LoadRegistry(filepath.Join(root, "policies/tools/registry-v5.json"))
	if err != nil {
		t.Fatal(err)
	}
	ev := &vseam.Evaluator{RegistryPath: filepath.Join(root, "policies/verification/contracts.json"), L4: l4}
	policyPath := filepath.Join(root, "policies/security/instruction-directive-patterns.json")
	cfg := orchestration.Config{
		StateRoot: filepath.Join(base, "state"), ArtifactDir: filepath.Join(base, "artifacts"),
		GitPath: gitBin(t), ProviderDir: filepath.Join(base, "provider"),
		SafetyRoot: filepath.Join(root, "instructions/global/safety"),
		SystemRoot: filepath.Join(root, "instructions/global/system"),
		ThemisRoot: filepath.Join(root, "instructions/themis"),
		PolicyPath: policyPath,
		AnchorPath: w.anchorPath, AnchorSHA256: anchorSHA, AnchorsRegistryPath: regs,
		ExecCeilingPath: w.ceiling, SkillCatalogPath: w.catalog,
		Model: m, Verifier: ev,
	}
	if wire {
		policy, err := instructions.LoadPolicy(policyPath)
		if err != nil {
			t.Fatal(err)
		}
		seamPath := w.registry
		if len(seamRegistry) > 0 && seamRegistry[0] != "" {
			seamPath = seamRegistry[0]
		}
		s, err := New(seamPath, policy)
		if err != nil {
			t.Fatal(err)
		}
		cfg.Delegator = s
		cfg.DelegationRegistryPath = w.registry
	}
	o, _, err := orchestration.Open(cfg)
	if err != nil {
		return nil, err
	}
	w.o = o
	sroot, err := state.OpenRoot(filepath.Join(base, "state"))
	if err != nil {
		t.Fatal(err)
	}
	w.sroot = sroot
	return w, nil
}

// instantiate runs the proposed remediate-dependency@2 Skill through
// L9 for this deployment and returns the envelope path.
func (w *anchoredWorld) instantiate(t *testing.T, taskID, modelName string) string {
	t.Helper()
	// The state root must exist before L9's disjointness check.
	_ = os.MkdirAll(filepath.Join(w.base, "state"), 0o755)
	path, err := skills.Instantiate(w.catalog, "remediate-dependency@2", skills.Request{
		TaskID: taskID, Repo: "demo", PinnedSHA: w.sha,
		Inputs:        map[string]any{"dependency": "vulnerable-dep", "advisory": "ADV-2026-1"},
		WallDeadlineS: 300,
		Deployment: skills.Deployment{Model: modelName, TurnTimeoutSec: 180,
			RegistryPath: filepath.Join(w.root, "policies/tools/registry-v5.json"), ExecCeilingPath: w.ceiling,
			StateRoot: filepath.Join(w.base, "state"), ArtifactDir: filepath.Join(w.base, "artifacts"),
			WorkspaceRoot: filepath.Join(w.base, "provider")},
		OutDir: filepath.Join(w.base, "envelopes"),
	})
	if err != nil {
		t.Fatalf("L9 instantiation of remediate-dependency@2: %v", err)
	}
	return path
}

var recordRef = regexp.MustCompile(`record-ref: ([0-9]+:sha256:[0-9a-f]{64})`)

// remediateParent is the @2 walk: read, delegate over what it read
// (reference formed from the furniture it SAW), declare, write the
// report, verify it, declare.
func remediateParent(t *testing.T) (func(int, []model.Message) model.ExecutionResponse, *string) {
	var usedRef string
	report := `{"finding": "vulnerable-dep v1 in go.mod", "remediation": "bump to v2", "evidence": "go.mod updated"}`
	rb, _ := json.Marshal(report)
	return func(turn int, conv []model.Message) model.ExecutionResponse {
		switch turn {
		case 1:
			return call("c1", "read_file", `{"path":"go.mod"}`)
		case 2:
			for _, msg := range conv {
				if msg.Role == model.RoleTool && msg.ToolCallID == "c1" {
					if m := recordRef.FindStringSubmatch(msg.Content); m != nil {
						usedRef = m[1]
					}
				}
			}
			args, _ := json.Marshal(map[string]string{"template": "dependency-triage@1", "evidence": usedRef, "brief": "Is vulnerable-dep the affected dependency?"})
			return call("c2", "delegate", string(args))
		case 3:
			return call("c3", "declare_done", `{}`)
		case 4:
			return call("c4", "write_file", `{"path":"report.json","content":`+string(rb)+`}`)
		case 5:
			return call("c5", "verify_report", `{"path":"report.json","contract":"report-valid@2"}`)
		}
		return call("c6", "declare_done", `{}`)
	}, &usedRef
}

func TestAnchoredDelegationPositivePath(t *testing.T) {
	m := &dynModel{delegated: okDelegated}
	w, err := newAnchoredWorld(t, m, "dyn", "", nil, true)
	if err != nil {
		t.Fatalf("anchored Open: %v", err)
	}
	parent, usedRef := remediateParent(t)
	m.parent = parent
	const task = "rsys-deleg-1"
	res, err := w.o.SubmitTask(w.instantiate(t, task, "dyn"))
	if err != nil {
		t.Fatalf("the anchored delegate walk must be admitted: %v", err)
	}
	if res.Status != state.StatusCompleted {
		t.Fatalf("status %s, want COMPLETED\n%s", res.Status, summarize(w.sroot, task))
	}
	evs, _ := w.sroot.ReadEvents(task)
	var l8 *state.Event
	for i := range evs {
		if evs[i].Class == state.EvL8Delegation {
			l8 = &evs[i]
		}
	}
	if l8 == nil || m.delegCall != 1 {
		t.Fatal("exactly one witnessed delegation")
	}
	var body struct {
		Outcome      string `json:"outcome"`
		EvidenceRefs []struct {
			Seq      int64  `json:"seq"`
			ObjectID string `json:"object_id"`
		} `json:"evidence_refs"`
	}
	_ = json.Unmarshal(l8.Body, &body)
	if body.Outcome != "completed" || len(body.EvidenceRefs) != 1 || *usedRef == "" ||
		*usedRef != strings.TrimSpace(strings.Join([]string{itoa(body.EvidenceRefs[0].Seq), body.EvidenceRefs[0].ObjectID}, ":")) {
		t.Fatalf("the reference the model formed from the furniture must be the one witnessed: %q vs %+v", *usedRef, body.EvidenceRefs)
	}
	man, err := w.sroot.ReadManifest(task)
	if err != nil || man.GovernedHashes["deployment_anchor"] == "unanchored" || man.GovernedHashes["skill"] != "remediate-dependency@2" {
		t.Fatalf("anchored, skill-attributed record expected: %v %v", err, man.GovernedHashes)
	}
	r, err := ReconstructDelegation(w.sroot, task, l8.Seq, trustForRegistry(t, w.root))
	if err != nil || r.Verdict != VerdictConfirmed {
		t.Fatalf("reconstruction: %v %+v", err, r)
	}
}

func trustForRegistry(t *testing.T, root string) ReconstructConfig {
	t.Helper()
	reg, err := tools.LoadRegistry(filepath.Join(root, "policies/tools/registry-v5.json"))
	if err != nil {
		t.Fatal(err)
	}
	return ReconstructConfig{RegistryHash: reg.Hash, ToolTrust: func(name string) (hctx.AuthorityClass, bool) {
		for _, tl := range reg.Tools {
			if tl.Name == name {
				return tl.Trust, true
			}
		}
		return "", false
	}}
}

// Anchor-side negatives, each with its positive twin above.
func TestAnchoredDelegationRefusals(t *testing.T) {
	t.Run("anchor pins a registry, none configured", func(t *testing.T) {
		m := &dynModel{delegated: okDelegated}
		_, err := newAnchoredWorld(t, m, "dyn", "", nil, false)
		if !errors.Is(err, orchestration.ErrAssembly) || !strings.Contains(err.Error(), "none is configured") {
			t.Fatalf("%v", err)
		}
	})
	t.Run("anchor declares absent, seam configured", func(t *testing.T) {
		m := &dynModel{delegated: okDelegated}
		_, err := newAnchoredWorld(t, m, "dyn", "", func(a map[string]any) { a["delegation_template_registry"] = "absent" }, true)
		if !errors.Is(err, orchestration.ErrAssembly) || !strings.Contains(err.Error(), "declares no delegation-template registry") {
			t.Fatalf("%v", err)
		}
	})
	t.Run("registry bytes are not the pinned bytes", func(t *testing.T) {
		m := &dynModel{delegated: okDelegated}
		_, err := newAnchoredWorld(t, m, "dyn", "", func(a map[string]any) { a["delegation_template_registry"] = strings.Repeat("ab", 32) }, true)
		if !errors.Is(err, orchestration.ErrAssembly) || !strings.Contains(err.Error(), "delegation-template registry is not the anchored artifact") {
			t.Fatalf("%v", err)
		}
	})
	t.Run("seam holds a registry that is not the pinned bytes", func(t *testing.T) {
		// The configured path hashes to the pin; the wired seam was built
		// over other bytes (a copy with different formatting) — refused
		// at Open (architecture review MED-6).
		other := delegationRegistry(t, nil)
		m := &dynModel{delegated: okDelegated}
		_, err := newAnchoredWorld(t, m, "dyn", "", nil, true, other)
		if !errors.Is(err, orchestration.ErrAssembly) || !strings.Contains(err.Error(), "wired delegation seam holds a registry") {
			t.Fatalf("%v", err)
		}
	})
	t.Run("pinned registry withdraws the template: Skill still runs, delegate refuses stage B", func(t *testing.T) {
		reg := delegationRegistry(t, func(tpl, entry map[string]any) { entry["state"] = "withdrawn" })
		m := &dynModel{delegated: okDelegated}
		w, err := newAnchoredWorld(t, m, "dyn", reg, nil, true)
		if err != nil {
			t.Fatal(err)
		}
		parent, _ := remediateParent(t)
		m.parent = parent
		const task = "rsys-withdrawn"
		res, err := w.o.SubmitTask(w.instantiate(t, task, "dyn"))
		if err != nil || res.Status != state.StatusCompleted {
			t.Fatalf("withdrawal must not invalidate the referencing Skill (C-L8-14 G): %+v %v", res, err)
		}
		witnessed, refused := false, false
		for _, e := range w.events(t, task) {
			if e.Class == state.EvL8Delegation {
				witnessed = true
			}
			if e.Class == state.EvL4Audit && strings.Contains(string(e.Body), `"delegation-refused:template-withdrawn"`) {
				refused = true
			}
		}
		if witnessed || !refused || m.delegCall != 0 {
			t.Fatalf("expected a stage-B refusal in the audit and no witness: witnessed=%v refused=%v calls=%d", witnessed, refused, m.delegCall)
		}
	})
}

// Register E: the UNMODIFIED loop against a local model — the anchored
// @2 Skill with delegate exposed. Skipped without an endpoint. The
// governed property asserted is admission + a typed terminal; if the
// model chose to delegate, the witness reconstructs CONFIRMED and the
// result re-entered the parent's next turn.
func TestLiveDelegationWalk(t *testing.T) {
	endpoint := os.Getenv("THEMIS_LIVE_OLLAMA")
	if endpoint == "" {
		endpoint = "http://localhost:11434"
	}
	client := http.Client{Timeout: 2 * time.Second}
	if _, err := client.Get(endpoint + "/api/tags"); err != nil {
		t.Skipf("no local model endpoint at %s: %v", endpoint, err)
	}
	modelName := os.Getenv("THEMIS_LIVE_TOOL_MODEL")
	if modelName == "" {
		modelName = "qwen2.5:7b"
	}
	live := model.NewOllamaChat(endpoint)
	w, err := newAnchoredWorld(t, live, modelName, "", nil, true)
	if err != nil {
		t.Fatal(err)
	}
	const task = "rsys-live-deleg"
	start := time.Now()
	res, err := w.o.SubmitTask(w.instantiate(t, task, modelName))
	if err != nil {
		t.Fatalf("admission: %v", err)
	}
	if res.Status != state.StatusCompleted && res.Status != state.StatusFailed {
		t.Fatalf("typed terminal expected: %+v", res)
	}
	evs, _ := w.sroot.ReadEvents(task)
	delegations, calls := 0, 0
	for _, e := range evs {
		if e.Class == state.EvL8Delegation {
			delegations++
		}
		if e.Class == state.EvL4Audit && strings.Contains(string(e.Body), `"Tool":"delegate"`) {
			calls++
		}
	}
	t.Logf("live: %s in %s — status %s, delegate calls %d, witnessed delegations %d\n%s", modelName, time.Since(start).Round(time.Second), res.Status, calls, delegations, summarize(w.sroot, task))
	for _, e := range evs {
		if e.Class == state.EvL8Delegation {
			r, err := ReconstructDelegation(w.sroot, task, e.Seq, trustForRegistry(t, w.root))
			if err != nil || r.Verdict != VerdictConfirmed {
				t.Fatalf("live delegation must reconstruct CONFIRMED: %v %+v", err, r)
			}
		}
	}
}

// summarize renders the record compactly for a failure message.
func summarize(root *state.Root, task string) string {
	evs, _ := root.ReadEvents(task)
	var b strings.Builder
	for _, e := range evs {
		body := string(e.Body)
		if len(body) > 160 {
			body = body[:160] + "…"
		}
		b.WriteString(itoa(e.Seq) + " " + e.Class + " " + body + "\n")
	}
	return b.String()
}

func (w *anchoredWorld) events(t *testing.T, task string) []state.Event {
	t.Helper()
	evs, err := w.sroot.ReadEvents(task)
	if err != nil {
		t.Fatal(err)
	}
	return evs
}
