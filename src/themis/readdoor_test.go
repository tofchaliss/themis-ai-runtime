package themis

// T-M2 Register B, positive path FIRST: under an anchor that pins the
// Themis store, remediate-dependency@3 (L9-instantiated) reads the
// governed Finding through get_finding, the result re-enters framed
// as governed-record, and the walk completes with L10 PASS. Then each
// negative in the owner's T-M2 Gate 1 note, asserting its own gate.
// Register D: the model's restatement of the Finding is a model-turn
// object at the floor; the Finding's bytes entered only under the
// registry's class.

import (
	stdctx "context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/tofchaliss/themis-app/store"
	"github.com/tofchaliss/themis/deployment"
	"github.com/tofchaliss/themis/instructions"
	"github.com/tofchaliss/themis/orchestration"
	"github.com/tofchaliss/themis/runtime/model"
	"github.com/tofchaliss/themis/skills"
	"github.com/tofchaliss/themis/state"
	dseam "github.com/tofchaliss/themis/subagents/delegation/seam"
	"github.com/tofchaliss/themis/tools"
	vseam "github.com/tofchaliss/themis/verification/seam"
)

func repoRoot(t *testing.T) string {
	t.Helper()
	_, file, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(file), "..", "..")
}

func gitBin(t *testing.T) string {
	t.Helper()
	for _, p := range []string{"/usr/bin/git", "/opt/homebrew/bin/git"} {
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	t.Skip("no pinned git")
	return ""
}

func mkMirror(t *testing.T) (string, string) {
	t.Helper()
	git := gitBin(t)
	root := t.TempDir()
	repo := filepath.Join(root, "demo-vuln-app")
	run := func(args ...string) string {
		cmd := exec.Command(git, append([]string{"-c", "user.name=t", "-c", "user.email=t@t"}, args...)...)
		cmd.Dir = repo
		cmd.Env = append(os.Environ(), "GIT_CONFIG_GLOBAL=/dev/null", "GIT_CONFIG_SYSTEM=/dev/null")
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("git %v: %v: %s", args, err, out)
		}
		return string(out)
	}
	if err := os.Mkdir(repo, 0o755); err != nil {
		t.Fatal(err)
	}
	run("init", "-q")
	if err := os.WriteFile(filepath.Join(repo, "go.mod"), []byte("module demo // vulnerable-dep v1\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	run("add", ".")
	run("commit", "-q", "-m", "seed")
	return root, strings.TrimSpace(run("rev-parse", "HEAD"))
}

func wj(t *testing.T, dir, name, body string) string {
	t.Helper()
	p := filepath.Join(dir, name)
	if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	return p
}

// readingParent: get_finding, get_product, read_file, declare, write
// the report, verify, declare. It records every tool message it saw.
type readingParent struct {
	turn     int
	seen     []model.Message
	finding  string
	delegReq *model.ExecutionRequest
}

func (p *readingParent) Name() string { return "scripted" }
func (p *readingParent) Execute(_ stdctx.Context, req model.ExecutionRequest) (*model.ExecutionResponse, error) {
	if len(req.Tools) == 0 {
		r := req
		p.delegReq = &r
		return &model.ExecutionResponse{Content: "triage", Termination: model.TerminationStop}, nil
	}
	p.turn++
	// Accumulate every tool message across phases: each phase composes
	// a fresh conversation, so the ANALYZE results are not in the
	// REMEDIATE request.
	for _, msg := range req.Messages {
		if msg.Role == model.RoleTool {
			p.seen = append(p.seen, msg)
		}
	}
	call := func(id, name, args string) *model.ExecutionResponse {
		return &model.ExecutionResponse{Termination: model.TerminationToolCalls,
			ToolCalls: []model.ToolCall{{ID: id, Name: name, Arguments: json.RawMessage(args)}}}
	}
	rb, _ := json.Marshal(`{"finding": "` + p.finding + `: vulnerable-dep v1 in go.mod", "remediation": "bump to v2", "evidence": "go.mod updated"}`)
	switch p.turn {
	case 1:
		return call("c1", "get_finding", `{"id":"`+p.finding+`"}`), nil
	case 2:
		return call("c2", "get_product", `{"id":"PROD-demo-vuln-app"}`), nil
	case 3:
		return call("c3", "read_file", `{"path":"go.mod"}`), nil
	case 4:
		// The model RESTATES the Finding in prose: a model-turn object.
		return &model.ExecutionResponse{Content: "The Finding says demo-vuln-app pins vulnerable-dep v1; ADV-2026-1 is fixed in v2. I will bump it.", Termination: model.TerminationStop}, nil
	case 5:
		return call("c5", "declare_done", `{}`), nil
	case 6:
		return call("c6", "write_file", `{"path":"report.json","content":`+string(rb)+`}`), nil
	case 7:
		return call("c7", "verify_report", `{"path":"report.json","contract":"report-valid@2"}`), nil
	}
	return call("c8", "declare_done", `{}`), nil
}

type world struct {
	o          *orchestration.Orchestrator
	sroot      *state.Root
	base, root string
	sha        string
	storeDir   string
	anchorSHA  string
}

// storeCopy copies the governed Themis store into a private dir and
// lets a test mutate it before pinning.
func storeCopy(t *testing.T, mutate func(dir string)) string {
	t.Helper()
	dir := t.TempDir()
	for _, f := range []string{"findings.json", "products.json"} {
		b, err := os.ReadFile(filepath.Join(repoRoot(t), "policies/themis", f))
		if err != nil {
			t.Fatal(err)
		}
		wj(t, dir, f, string(b))
	}
	if mutate != nil {
		mutate(dir)
	}
	return dir
}

// newWorld opens an anchored orchestrator whose anchor pins storeDir
// (or "absent" when storeDir is ""); seamDir is the store the seam is
// built over (defaults to storeDir); mutate edits the anchor map.
func newWorld(t *testing.T, m model.Interface, storeDir, seamDir string, mutate func(map[string]any)) (*world, error) {
	t.Helper()
	root := repoRoot(t)
	base := t.TempDir()
	mirror, sha := mkMirror(t)
	w := &world{base: base, root: root, sha: sha, storeDir: storeDir}
	ceiling := wj(t, base, "execution-ceiling.json",
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
	bundle := filepath.Join(root, "policies/skills/remediate-dependency-3")
	themisPin := "absent"
	if storeDir != "" {
		h, err := store.HashOf(storeDir)
		if err != nil {
			t.Fatal(err)
		}
		themisPin = h
	}
	a := map[string]any{
		"version": 1, "name": "test-rsys", "deployment_version": 6,
		"instruction_root_safety": hashDir(filepath.Join(root, "instructions/global/safety")),
		"instruction_root_system": hashDir(filepath.Join(root, "instructions/global/system")),
		"instruction_root_themis": hashDir(filepath.Join(root, "instructions/themis")),
		"instruction_policy":      hashFile(filepath.Join(root, "policies/security/instruction-directive-patterns.json")),
		"tool_registry":           hashFile(filepath.Join(root, "policies/tools/registry-v5.json")),
		"constitution":            map[string]any{"state": state.ConstitutionHash(), "orchestration": orchestration.ConstitutionHash()},
		"execution_ceiling":       hashFile(ceiling),
		"workflows": []any{map[string]any{
			"workflow":         hashFile(filepath.Join(bundle, "workflow.json")),
			"workflow_ceiling": hashFile(filepath.Join(bundle, "ceiling.json")),
			"context_contract": hashFile(filepath.Join(bundle, "contract.json")),
		}},
		"models":                       []any{"scripted"},
		"model_registry":               "absent",
		"skill_catalog":                hashFile(filepath.Join(root, "policies/skills/catalog.json")),
		"skills":                       []any{"remediate-dependency@3"},
		"contract_registry":            hashFile(filepath.Join(root, "policies/verification/contracts.json")),
		"criteria_registry":            hashFile(filepath.Join(root, "policies/ratchet/criteria.json")),
		"regression_set_registry":      hashFile(filepath.Join(root, "policies/ratchet/regression-sets.json")),
		"delegation_template_registry": hashFile(filepath.Join(root, "policies/delegation/registry.json")),
		"themis_store":                 themisPin,
	}
	if mutate != nil {
		mutate(a)
	}
	ab, _ := json.Marshal(a)
	anchorPath := wj(t, base, "anchor.json", string(ab))
	sum := sha256.Sum256(ab)
	w.anchorSHA = hex.EncodeToString(sum[:])
	regs := wj(t, base, "anchors.json", `{"version":1,"kind":"deployment-anchors","entries":[{"name":"test-rsys","version":6,"artifact_sha256":"`+w.anchorSHA+`","state":"active","steward":"t"}]}`)

	l4, err := tools.LoadRegistry(filepath.Join(root, "policies/tools/registry-v5.json"))
	if err != nil {
		t.Fatal(err)
	}
	policyPath := filepath.Join(root, "policies/security/instruction-directive-patterns.json")
	policy, err := instructions.LoadPolicy(policyPath)
	if err != nil {
		t.Fatal(err)
	}
	ds, err := dseam.New(filepath.Join(root, "policies/delegation/registry.json"), policy)
	if err != nil {
		t.Fatal(err)
	}
	cfg := orchestration.Config{
		StateRoot: filepath.Join(base, "state"), ArtifactDir: filepath.Join(base, "artifacts"),
		GitPath: gitBin(t), ProviderDir: filepath.Join(base, "provider"),
		SafetyRoot: filepath.Join(root, "instructions/global/safety"),
		SystemRoot: filepath.Join(root, "instructions/global/system"),
		ThemisRoot: filepath.Join(root, "instructions/themis"),
		PolicyPath: policyPath,
		AnchorPath: anchorPath, AnchorSHA256: w.anchorSHA, AnchorsRegistryPath: regs,
		ExecCeilingPath: ceiling, SkillCatalogPath: filepath.Join(root, "policies/skills/catalog.json"),
		DelegationRegistryPath: filepath.Join(root, "policies/delegation/registry.json"),
		Model:                  m, Verifier: &vseam.Evaluator{RegistryPath: filepath.Join(root, "policies/verification/contracts.json"), L4: l4},
		Delegator: ds,
	}
	if seamDir == "" {
		seamDir = storeDir
	}
	if seamDir != "" {
		st, err := store.Load(seamDir)
		if err != nil {
			t.Fatal(err)
		}
		cfg.ThemisSeam = st
		cfg.ThemisStorePath = storeDir
	}
	o, _, err := orchestration.Open(cfg)
	if err != nil {
		return nil, err
	}
	w.o = o
	w.sroot, err = state.OpenRoot(filepath.Join(base, "state"))
	if err != nil {
		t.Fatal(err)
	}
	return w, nil
}

func (w *world) instantiate(t *testing.T, task, finding string) string {
	t.Helper()
	_ = os.MkdirAll(filepath.Join(w.base, "state"), 0o755)
	path, err := skills.Instantiate(filepath.Join(w.root, "policies/skills/catalog.json"), "remediate-dependency@3", skills.Request{
		TaskID: task, Repo: "demo-vuln-app", PinnedSHA: w.sha,
		Inputs:        map[string]any{"finding": finding, "dependency": "vulnerable-dep", "advisory": "ADV-2026-1"},
		WallDeadlineS: 300,
		Deployment: skills.Deployment{Model: "scripted", TurnTimeoutSec: 180,
			RegistryPath: filepath.Join(w.root, "policies/tools/registry-v5.json"), ExecCeilingPath: filepath.Join(w.base, "execution-ceiling.json"),
			StateRoot: filepath.Join(w.base, "state"), ArtifactDir: filepath.Join(w.base, "artifacts"), WorkspaceRoot: filepath.Join(w.base, "provider")},
		OutDir: filepath.Join(w.base, "envelopes"),
	})
	if err != nil {
		t.Fatalf("L9 instantiation of remediate-dependency@3: %v", err)
	}
	return path
}

func audits(t *testing.T, w *world, task string) map[string]map[string]any {
	t.Helper()
	evs, err := w.sroot.ReadEvents(task)
	if err != nil {
		t.Fatal(err)
	}
	out := map[string]map[string]any{}
	for _, e := range evs {
		if e.Class != state.EvL4Audit {
			continue
		}
		var b map[string]any
		_ = json.Unmarshal(e.Body, &b)
		if tool, ok := b["Tool"].(string); ok {
			out[tool] = b
		}
	}
	return out
}

// THE POSITIVE CHAIN: anchor pin → store → seam → L4 → governed-record → model.
func TestReadDoorPositivePath(t *testing.T) {
	m := &readingParent{finding: "FIND-2026-0001"}
	w, err := newWorld(t, m, storeCopy(t, nil), "", nil)
	if err != nil {
		t.Fatalf("anchored Open with the Themis store: %v", err)
	}
	const task = "t-read-ok"
	res, err := w.o.SubmitTask(w.instantiate(t, task, "FIND-2026-0001"))
	if err != nil || res.Status != state.StatusCompleted {
		t.Fatalf("%+v %v", res, err)
	}
	a := audits(t, w, task)
	if a["get_finding"]["Decision"] != "authorized" || a["get_product"]["Decision"] != "authorized" {
		t.Fatalf("both reads must be authorized: %v %v", a["get_finding"], a["get_product"])
	}
	// The result the model saw: framed under governed-record, with the
	// registered bytes inside the fence.
	var framed string
	for _, msg := range m.seen {
		if msg.Role == model.RoleTool && msg.ToolCallID == "c1" {
			framed = msg.Content
		}
	}
	if !strings.Contains(framed, "kind: tool-result:get_finding") || !strings.Contains(framed, "authority: governed-record") || !strings.Contains(framed, `"advisory": "ADV-2026-1"`) {
		t.Fatalf("the Finding must re-enter framed as governed-record with the registered bytes:\n%s", framed)
	}
	// The evidence object in the record IS the registered bytes.
	st, _ := store.Load(w.storeDir)
	want, _ := st.Read("finding", "FIND-2026-0001")
	if h := a["get_finding"]["ResultHash"].(string); h != hex64(want) {
		t.Fatalf("audit ResultHash %s ≠ hash of the registered bytes %s", h, hex64(want))
	}
	// Register D — laundering: the model's restatement is a model-turn
	// object; the record's class derivation puts model-turn at the
	// floor, so the same sentence never carries governed-record.
	evs, _ := w.sroot.ReadEvents(task)
	restated := false
	for _, e := range evs {
		if e.Class == state.EvModelTurn && len(e.Refs) == 1 {
			b, _ := w.sroot.Store().GetObject(e.Refs[0].ID)
			if strings.Contains(string(b), "The Finding says demo-vuln-app") {
				restated = true
				if e.Refs[0].Class != state.ObjEvidencePayload {
					t.Fatalf("model-turn object class: %s", e.Refs[0].Class)
				}
			}
		}
	}
	if !restated {
		t.Fatal("fixture: the restatement turn was not recorded")
	}
	man, _ := w.sroot.ReadManifest(task)
	if man.GovernedHashes["skill"] != "remediate-dependency@3" || man.GovernedHashes["deployment_anchor"] != w.anchorSHA {
		t.Fatalf("record: %v", man.GovernedHashes)
	}
}

func hex64(b []byte) string {
	s := sha256.Sum256(b)
	return hex.EncodeToString(s[:])
}

func TestReadDoorRefusals(t *testing.T) {
	t.Run("anchor pins the store, none configured", func(t *testing.T) {
		m := &readingParent{finding: "FIND-2026-0001"}
		_, err := newWorld(t, m, "", "", func(a map[string]any) { a["themis_store"] = strings.Repeat("ab", 32) })
		if !errors.Is(err, orchestration.ErrAssembly) || !strings.Contains(err.Error(), "pins a Themis store but none is configured") {
			t.Fatalf("%v", err)
		}
	})
	t.Run("anchor declares absent, seam configured", func(t *testing.T) {
		m := &readingParent{finding: "FIND-2026-0001"}
		_, err := newWorld(t, m, storeCopy(t, nil), "", func(a map[string]any) { a["themis_store"] = "absent" })
		if !errors.Is(err, orchestration.ErrAssembly) || !strings.Contains(err.Error(), "declares no Themis store") {
			t.Fatalf("%v", err)
		}
	})
	t.Run("store bytes are not the pinned bytes", func(t *testing.T) {
		m := &readingParent{finding: "FIND-2026-0001"}
		dir := storeCopy(t, nil)
		w, err := newWorld(t, m, dir, "", func(a map[string]any) { a["themis_store"] = strings.Repeat("cd", 32) })
		if !errors.Is(err, orchestration.ErrAssembly) || !strings.Contains(err.Error(), "Themis store is not the anchored artifact") {
			t.Fatalf("%v %v", w, err)
		}
	})
	t.Run("registry modified after pinning", func(t *testing.T) {
		m := &readingParent{finding: "FIND-2026-0001"}
		dir := storeCopy(t, nil)
		w, err := newWorld(t, m, dir, "", nil)
		if err != nil {
			t.Fatal(err)
		}
		// A Finding added after the anchor: the per-task re-verification
		// refuses the next SubmitTask.
		b, _ := os.ReadFile(filepath.Join(dir, "findings.json"))
		wj(t, dir, "findings.json", string(b)+"\n")
		_, err = w.o.SubmitTask(w.instantiate(t, "t-mod", "FIND-2026-0001"))
		if !errors.Is(err, orchestration.ErrAssembly) || !strings.Contains(err.Error(), "Themis store is not the anchored artifact") {
			t.Fatalf("a Finding enters a deployment only by Governance act: %v", err)
		}
	})
	t.Run("seam built over other bytes than the pin", func(t *testing.T) {
		m := &readingParent{finding: "FIND-2026-0001"}
		pinned := storeCopy(t, nil)
		other := storeCopy(t, func(d string) {
			b, _ := os.ReadFile(filepath.Join(d, "products.json"))
			wj(t, d, "products.json", string(b)+"\n")
		})
		_, err := newWorld(t, m, pinned, other, nil)
		if !errors.Is(err, orchestration.ErrAssembly) || !strings.Contains(err.Error(), "wired Themis seam holds a store") {
			t.Fatalf("%v", err)
		}
	})
	// Read-time refusals: the walk continues; the audit names the gate;
	// no governed bytes reach the model.
	readCase := func(t *testing.T, name, finding string, mutateStore func(string), want map[string]string) {
		t.Helper()
		m := &readingParent{finding: finding}
		w, err := newWorld(t, m, storeCopy(t, mutateStore), "", nil)
		if err != nil {
			t.Fatal(err)
		}
		res, err := w.o.SubmitTask(w.instantiate(t, name, finding))
		if err != nil {
			t.Fatalf("a read refusal is data; the walk continues: %v", err)
		}
		_ = res
		a := audits(t, w, name)["get_finding"]
		for k, v := range want {
			if got, _ := a[k].(string); !strings.Contains(got, v) {
				t.Fatalf("audit %s: want %q in %q (%v)", k, v, got, a)
			}
		}
		for _, msg := range m.seen {
			if msg.Role == model.RoleTool && msg.ToolCallID == "c1" && strings.Contains(msg.Content, "authority: governed-record") {
				t.Fatal("no governed bytes may reach the model on a refused read")
			}
		}
	}
	t.Run("withdrawn finding is unavailable", func(t *testing.T) {
		readCase(t, "t-withdrawn", "FIND-2026-0001", func(d string) {
			b, _ := os.ReadFile(filepath.Join(d, "findings.json"))
			wj(t, d, "findings.json", strings.Replace(string(b), `"state": "active"`, `"state": "withdrawn"`, 1))
		}, map[string]string{"Decision": "error", "ErrClass": "seam-unavailable"})
	})
	t.Run("unknown finding is unavailable", func(t *testing.T) {
		readCase(t, "t-unknown", "FIND-2026-9999", nil, map[string]string{"Decision": "error", "ErrClass": "seam-unavailable"})
	})
	t.Run("id outside themis_scope is an L4 refusal", func(t *testing.T) {
		// The grant scopes get_finding to FIND-; a PROD- id through it is
		// refused at L4 before the seam is ever consulted.
		readCase(t, "t-scope", "PROD-demo-vuln-app", nil, map[string]string{"Decision": "denied", "TracePredicate": "themis-id-outside-grant-scope"})
	})
	t.Run("get_finding cannot mint a class other than governed-record", func(t *testing.T) {
		// Structural: the registry loader refuses a Themis read
		// registered at any other trust; the seam has no class field.
		body := `{"version":5,"tools":[{"name":"get_finding","description":"d","target":"themis-id","timeout_sec":5,"trust":"derived","params":[{"name":"id","type":"string","required":true,"description":"i","target":true}]}]}`
		p := wj(t, t.TempDir(), "r.json", body)
		if _, err := tools.LoadRegistry(p); err == nil || !strings.Contains(err.Error(), "derived") {
			t.Fatalf("derived must be refused at load: %v", err)
		}
		st, _ := store.Load(storeCopy(t, nil))
		b, _ := st.Read("finding", "FIND-2026-0001")
		var rec map[string]any
		_ = json.Unmarshal(b, &rec)
		for _, k := range []string{"authority", "trust", "class"} {
			if _, ok := rec[k]; ok {
				t.Fatalf("the store must carry no class field: %s", k)
			}
		}
	})
	t.Run("L2 ThemisReader stays unwired", func(t *testing.T) {
		// The loop constructs no KindThemis source; the only Themis
		// bytes in any record enter through l4-audit under get_finding.
		m := &readingParent{finding: "FIND-2026-0001"}
		w, err := newWorld(t, m, storeCopy(t, nil), "", nil)
		if err != nil {
			t.Fatal(err)
		}
		const task = "t-l2"
		if _, err := w.o.SubmitTask(w.instantiate(t, task, "FIND-2026-0001")); err != nil {
			t.Fatal(err)
		}
		evs, _ := w.sroot.ReadEvents(task)
		for _, e := range evs {
			if e.Class == state.EvL2Delivery && strings.Contains(string(e.Body), "ADV-2026-1") {
				t.Fatal("Themis bytes must not enter through L2 composition in v0")
			}
		}
	})
}
