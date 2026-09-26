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
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/tofchaliss/themis-ai-runtime/src/harness/deployment"
	"github.com/tofchaliss/themis-ai-runtime/src/harness/instructions"
	themisclient "github.com/tofchaliss/themis-ai-runtime/src/harness/integrations/themis/client"
	"github.com/tofchaliss/themis-ai-runtime/src/harness/integrations/themis/contracts"
	"github.com/tofchaliss/themis-ai-runtime/src/harness/orchestration"
	"github.com/tofchaliss/themis-ai-runtime/src/harness/runtime/model"
	"github.com/tofchaliss/themis-ai-runtime/src/harness/skills"
	"github.com/tofchaliss/themis-ai-runtime/src/harness/state"
	dseam "github.com/tofchaliss/themis-ai-runtime/src/harness/subagents/delegation/seam"
	"github.com/tofchaliss/themis-ai-runtime/src/harness/tools"
	vseam "github.com/tofchaliss/themis-ai-runtime/src/harness/verification/seam"
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

// readingParent: get_finding, read_file, declare, write the report,
// verify, declare. It records every tool message it saw.
type readingParent struct {
	turn     int
	seen     []model.Message
	finding  string
	delegReq *model.ExecutionRequest
	// report overrides the turn-6 report content; tail supplies the
	// calls for turns 8.. in order (then declare_done). Intake tests
	// use them to verify one thing and egress another.
	report string
	tail   []model.ToolCall
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
	report := p.report
	if report == "" {
		report = `{"finding": "` + p.finding + `: vulnerable-dep v1 in go.mod", "remediation": "bump to v2", "evidence": "go.mod updated"}`
	}
	rb, _ := json.Marshal(report)
	switch p.turn {
	case 1:
		return call("c1", "get_finding", `{"id":"`+p.finding+`"}`), nil
	case 2:
		return call("c3", "read_file", `{"path":"go.mod"}`), nil
	case 3:
		return &model.ExecutionResponse{Content: "noted", Termination: model.TerminationStop}, nil
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
	if i := p.turn - 8; i < len(p.tail) {
		return &model.ExecutionResponse{Termination: model.TerminationToolCalls, ToolCalls: []model.ToolCall{p.tail[i]}}, nil
	}
	return call("c8", "declare_done", `{}`), nil
}

type world struct {
	o          *orchestration.Orchestrator
	sroot      *state.Root
	base, root string
	sha        string
	anchorSHA  string
	regs       string // the anchors registry the deployment opened under
	contract   string // the Themis interface contract file the anchor pins
	gov, reg   *httptest.Server
}

// The demo Finding as the live authority serves it (Governance
// FindingView shape), with positions and proposals PRESENT so the
// projection is exercised on every read. UUID identities (D-I-4).
const (
	demoFindingID = "b1be6f86-2ecd-451f-9411-95f1f32fd501"
	demoProductID = "0f8fad5b-d9cb-469f-a165-70867728950e"
	demoReadKey   = "tk_read_test_0123456789"
)

func findingView(id string) string {
	return `{"id":"` + id + `","release_id":"3f0c1a2e-0000-4000-8000-000000000001","faultline_id":"3f0c1a2e-0000-4000-8000-000000000002","cve":"ADV-2026-1","stage":"identified",
 "components":[{"purl":"pkg:golang/demo/vulnerable-dep@v1","name":"vulnerable-dep","version":"v1","ecosystem":"golang","claim_class":"carrier"}],
 "current_position":{"version":2,"stance":"accepted_risk","rationale":"PRIOR-DECISION-MUST-NOT-REACH-THE-MODEL"},
 "positions":[{"version":1,"stance":"affected"}],"proposals":[{"proposal_id":"p-1","stance":"mitigated"}]}`
}

// themisStandIn serves the demo Finding and Product over HTTP as the
// Themis estate would: the harness-side tests prove the seam, never a
// running Themis (D-I-7). keySeen records the credential presented.
func themisStandIn(t *testing.T, keySeen *string, serveFinding bool) (*httptest.Server, *httptest.Server) {
	t.Helper()
	gov := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		*keySeen = r.Header.Get("X-API-Key")
		if !serveFinding || r.URL.Path != "/api/v1/findings/"+demoFindingID {
			http.Error(w, `{"title":"not found"}`, 404)
			return
		}
		_, _ = w.Write([]byte(findingView(demoFindingID)))
	}))
	reg := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/products/"+demoProductID {
			http.Error(w, `{"title":"not found"}`, 404)
			return
		}
		_, _ = w.Write([]byte(`{"id":"` + demoProductID + `","name":"demo-vuln-app"}`))
	}))
	t.Cleanup(gov.Close)
	t.Cleanup(reg.Close)
	return gov, reg
}

func writeContract(t *testing.T, dir string, gov, reg *httptest.Server) string {
	t.Helper()
	return wj(t, dir, "contract.json", `{"version":1,"governance_base_url":"`+gov.URL+`","registry_base_url":"`+reg.URL+`","governance_spec_sha256":"`+strings.Repeat("d", 64)+`","registry_spec_sha256":"`+strings.Repeat("e", 64)+`","themis_commit":"`+strings.Repeat("f", 40)+`"}`)
}

// worldOpts shapes the read door a test world opens with: withDoor
// wires a contract + HTTP seam (an anchor pinning it, unless mutate
// says otherwise); seamContract lets a test build the seam over a
// DIFFERENT contract than the anchor pins.
type worldOpts struct {
	withDoor     bool
	serveFinding bool
	seamContract string
	keySeen      *string
}

// newWorld opens an anchored orchestrator; mutate edits the anchor map.
func newWorld(t *testing.T, m model.Interface, opts worldOpts, mutate func(map[string]any)) (*world, error) {
	t.Helper()
	root := repoRoot(t)
	base := t.TempDir()
	mirror, sha := mkMirror(t)
	w := &world{base: base, root: root, sha: sha}
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
	bundle := filepath.Join(root, "policies/skills/remediate-dependency-4")
	themisPin := "absent"
	var seam tools.ThemisSeam
	if opts.withDoor {
		keySeen := opts.keySeen
		if keySeen == nil {
			keySeen = new(string)
		}
		w.gov, w.reg = themisStandIn(t, keySeen, opts.serveFinding)
		w.contract = writeContract(t, base, w.gov, w.reg)
		h, err := deployment.HashFile(w.contract)
		if err != nil {
			t.Fatal(err)
		}
		themisPin = h
		seamPath := w.contract
		if opts.seamContract != "" {
			seamPath = opts.seamContract
		}
		c, err := contracts.Load(seamPath)
		if err != nil {
			t.Fatal(err)
		}
		cl, err := themisclient.New(c, demoReadKey, nil)
		if err != nil {
			t.Fatal(err)
		}
		seam = cl
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
		"skills":                       []any{"remediate-dependency@4"},
		"contract_registry":            hashFile(filepath.Join(root, "policies/verification/contracts.json")),
		"criteria_registry":            hashFile(filepath.Join(root, "policies/ratchet/criteria.json")),
		"regression_set_registry":      hashFile(filepath.Join(root, "policies/ratchet/regression-sets.json")),
		"delegation_template_registry": hashFile(filepath.Join(root, "policies/delegation/registry.json")),
		"themis_contract":              themisPin,
	}
	if mutate != nil {
		mutate(a)
	}
	ab, _ := json.Marshal(a)
	anchorPath := wj(t, base, "anchor.json", string(ab))
	sum := sha256.Sum256(ab)
	w.anchorSHA = hex.EncodeToString(sum[:])
	regs := wj(t, base, "anchors.json", `{"version":1,"kind":"deployment-anchors","entries":[{"name":"test-rsys","version":6,"artifact_sha256":"`+w.anchorSHA+`","state":"active","steward":"t"}]}`)
	w.regs = regs

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
	if seam != nil {
		cfg.ThemisSeam = seam
		cfg.ThemisContractPath = w.contract
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
	path, err := skills.Instantiate(filepath.Join(w.root, "policies/skills/catalog.json"), "remediate-dependency@4", skills.Request{
		TaskID: task, Repo: "demo-vuln-app", PinnedSHA: w.sha,
		Inputs:        map[string]any{"finding": finding, "dependency": "vulnerable-dep", "advisory": "ADV-2026-1"},
		WallDeadlineS: 300,
		Deployment: skills.Deployment{Model: "scripted", TurnTimeoutSec: 180,
			RegistryPath: filepath.Join(w.root, "policies/tools/registry-v5.json"), ExecCeilingPath: filepath.Join(w.base, "execution-ceiling.json"),
			StateRoot: filepath.Join(w.base, "state"), ArtifactDir: filepath.Join(w.base, "artifacts"), WorkspaceRoot: filepath.Join(w.base, "provider")},
		OutDir: filepath.Join(w.base, "envelopes"),
	})
	if err != nil {
		t.Fatalf("L9 instantiation of remediate-dependency@4: %v", err)
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

// THE POSITIVE CHAIN: anchor pin → contract → HTTP seam → L4 →
// governed-record → model. The bytes the model sees are the PROJECTED
// Finding (no positions, no proposals); the credential is presented to
// the authority and appears in no record.
func TestReadDoorPositivePath(t *testing.T) {
	m := &readingParent{finding: demoFindingID}
	var keySeen string
	w, err := newWorld(t, m, worldOpts{withDoor: true, serveFinding: true, keySeen: &keySeen}, nil)
	if err != nil {
		t.Fatalf("anchored Open with the Themis contract: %v", err)
	}
	const task = "t-read-ok"
	res, err := w.o.SubmitTask(w.instantiate(t, task, demoFindingID))
	if err != nil || res.Status != state.StatusCompleted {
		t.Fatalf("%+v %v", res, err)
	}
	a := audits(t, w, task)
	if a["get_finding"]["Decision"] != "authorized" {
		t.Fatalf("the read must be authorized: %v", a["get_finding"])
	}
	if _, ok := a["get_product"]; ok {
		t.Fatal("remediate-dependency@4 holds no get_product (D-I-4)")
	}
	if keySeen != demoReadKey {
		t.Fatalf("the read key must reach the authority: %q", keySeen)
	}
	var framed string
	for _, msg := range m.seen {
		if msg.Role == model.RoleTool && msg.ToolCallID == "c1" {
			framed = msg.Content
		}
	}
	if !strings.Contains(framed, "kind: tool-result:get_finding") || !strings.Contains(framed, "authority: governed-record") || !strings.Contains(framed, `"cve":"ADV-2026-1"`) {
		t.Fatalf("the Finding must re-enter framed as governed-record with the projected bytes:\n%s", framed)
	}
	for _, forbidden := range []string{"current_position", "positions", "proposals", "PRIOR-DECISION", demoReadKey} {
		if strings.Contains(framed, forbidden) {
			t.Fatalf("%q reached the model", forbidden)
		}
	}
	// The evidence object in the record IS the projected bytes, hashed
	// by L4 — and the credential is in no object and no event.
	evs, _ := w.sroot.ReadEvents(task)
	restated := false
	for _, e := range evs {
		if strings.Contains(string(e.Body), demoReadKey) {
			t.Fatalf("credential in event %d", e.Seq)
		}
		for _, r := range e.Refs {
			b, _ := w.sroot.Store().GetObject(r.ID)
			if strings.Contains(string(b), demoReadKey) || strings.Contains(string(b), "PRIOR-DECISION") {
				t.Fatalf("credential or unprojected Finding content in object %s", r.ID)
			}
		}
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
	if man.GovernedHashes["skill"] != "remediate-dependency@4" || man.GovernedHashes["deployment_anchor"] != w.anchorSHA {
		t.Fatalf("record: %v", man.GovernedHashes)
	}
}

func hex64(b []byte) string {
	s := sha256.Sum256(b)
	return hex.EncodeToString(s[:])
}

func TestReadDoorRefusals(t *testing.T) {
	t.Run("anchor pins a contract, no door configured", func(t *testing.T) {
		m := &readingParent{finding: demoFindingID}
		_, err := newWorld(t, m, worldOpts{}, func(a map[string]any) { a["themis_contract"] = strings.Repeat("ab", 32) })
		if !errors.Is(err, orchestration.ErrAssembly) || !strings.Contains(err.Error(), "pins a Themis contract but no read door is configured") {
			t.Fatalf("%v", err)
		}
	})
	t.Run("anchor declares absent, door configured", func(t *testing.T) {
		m := &readingParent{finding: demoFindingID}
		_, err := newWorld(t, m, worldOpts{withDoor: true, serveFinding: true}, func(a map[string]any) { a["themis_contract"] = "absent" })
		if !errors.Is(err, orchestration.ErrAssembly) || !strings.Contains(err.Error(), "declares no Themis contract") {
			t.Fatalf("%v", err)
		}
	})
	t.Run("contract bytes are not the pinned bytes", func(t *testing.T) {
		m := &readingParent{finding: demoFindingID}
		_, err := newWorld(t, m, worldOpts{withDoor: true, serveFinding: true}, func(a map[string]any) { a["themis_contract"] = strings.Repeat("cd", 32) })
		if !errors.Is(err, orchestration.ErrAssembly) || !strings.Contains(err.Error(), "Themis contract is not the anchored artifact") {
			t.Fatalf("%v", err)
		}
	})
	t.Run("contract modified after pinning", func(t *testing.T) {
		m := &readingParent{finding: demoFindingID}
		w, err := newWorld(t, m, worldOpts{withDoor: true, serveFinding: true}, nil)
		if err != nil {
			t.Fatal(err)
		}
		// An endpoint change after the anchor: the per-task
		// re-verification refuses the next SubmitTask.
		b, _ := os.ReadFile(w.contract)
		wj(t, filepath.Dir(w.contract), "contract.json", string(b)+"\n")
		_, err = w.o.SubmitTask(w.instantiate(t, "t-mod", demoFindingID))
		if !errors.Is(err, orchestration.ErrAssembly) || !strings.Contains(err.Error(), "Themis contract is not the anchored artifact") {
			t.Fatalf("an interface enters a deployment only by Governance act: %v", err)
		}
	})
	t.Run("door built over another contract than the pin", func(t *testing.T) {
		m := &readingParent{finding: demoFindingID}
		var seen string
		gov2, reg2 := themisStandIn(t, &seen, true)
		other := writeContract(t, t.TempDir(), gov2, reg2)
		_, err := newWorld(t, m, worldOpts{withDoor: true, serveFinding: true, seamContract: other}, nil)
		if !errors.Is(err, orchestration.ErrAssembly) || !strings.Contains(err.Error(), "not built over the anchored contract") {
			t.Fatalf("%v", err)
		}
	})
	readCase := func(t *testing.T, task, id string, serve bool, want map[string]string) {
		t.Helper()
		m := &readingParent{finding: id}
		w, err := newWorld(t, m, worldOpts{withDoor: true, serveFinding: serve}, nil)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := w.o.SubmitTask(w.instantiate(t, task, id)); err != nil {
			t.Fatal(err)
		}
		a := audits(t, w, task)["get_finding"]
		for k, v := range want {
			if a[k] != v {
				t.Fatalf("get_finding audit %s = %v, want %s (%v)", k, a[k], v, a)
			}
		}
	}
	t.Run("finding the authority does not serve is unavailable", func(t *testing.T) {
		// 404 from Governance (unknown, withdrawn, or not this estate's):
		// the seam fails closed and L4 records seam-unavailable.
		readCase(t, "t-unknown", "b1be6f86-2ecd-451f-9411-95f1f32fd509", true, map[string]string{"Decision": "error", "ErrClass": "seam-unavailable"})
	})
	t.Run("authority down is unavailable", func(t *testing.T) {
		readCase(t, "t-down", demoFindingID, false, map[string]string{"Decision": "error", "ErrClass": "seam-unavailable"})
	})
	t.Run("id outside the uuid scope is an L4 refusal", func(t *testing.T) {
		// The grant scopes get_finding to uuid; a prefixed id through it
		// is refused at L4 before the seam is ever consulted.
		readCase(t, "t-scope", "FIND-2026-0001", true, map[string]string{"Decision": "denied", "TracePredicate": "themis-id-outside-grant-scope"})
	})
	t.Run("get_finding cannot mint a class other than governed-record", func(t *testing.T) {
		// Structural: the registry loader refuses a Themis read
		// registered at any other trust; the projected record carries no
		// class field of its own.
		body := `{"version":5,"tools":[{"name":"get_finding","description":"d","target":"themis-id","timeout_sec":5,"trust":"derived","params":[{"name":"id","type":"string","required":true,"description":"i","target":true}]}]}`
		p := wj(t, t.TempDir(), "r.json", body)
		if _, err := tools.LoadRegistry(p); err == nil || !strings.Contains(err.Error(), "derived") {
			t.Fatalf("derived must be refused at load: %v", err)
		}
		var seen string
		gov, reg := themisStandIn(t, &seen, true)
		c, _ := contracts.Load(writeContract(t, t.TempDir(), gov, reg))
		cl, _ := themisclient.New(c, demoReadKey, nil)
		b, err := cl.Read("finding", demoFindingID)
		if err != nil {
			t.Fatal(err)
		}
		var rec map[string]any
		_ = json.Unmarshal(b, &rec)
		for _, k := range []string{"authority", "trust", "class", "current_position", "positions", "proposals"} {
			if _, ok := rec[k]; ok {
				t.Fatalf("the projected record must carry no %s", k)
			}
		}
	})
	t.Run("L2 ThemisReader stays unwired", func(t *testing.T) {
		// The loop constructs no KindThemis source; the only Themis
		// bytes in any record enter through l4-audit under get_finding.
		m := &readingParent{finding: demoFindingID}
		w, err := newWorld(t, m, worldOpts{withDoor: true, serveFinding: true}, nil)
		if err != nil {
			t.Fatal(err)
		}
		const task = "t-l2"
		if _, err := w.o.SubmitTask(w.instantiate(t, task, demoFindingID)); err != nil {
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
