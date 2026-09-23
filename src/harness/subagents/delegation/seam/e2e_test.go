package seam

// Register B, positive path FIRST (design.md §5.5, the C17 lesson): a
// genuine delegate call from a governed walk is admitted, instantiated,
// executed by the injected adapter, witnessed by l8-delegation, and
// re-enters the parent framed under the registered trust. Then each
// refusal and each post-instance outcome, every one asserting its own
// gate. Unanchored test-harness caller role; the anchored proof is M6.

import (
	stdctx "context"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	hctx "github.com/tofchaliss/themis/context"
	"github.com/tofchaliss/themis/instructions"
	"github.com/tofchaliss/themis/orchestration"
	"github.com/tofchaliss/themis/runtime/model"
	"github.com/tofchaliss/themis/state"
	"github.com/tofchaliss/themis/subagents/delegation"
	vseam "github.com/tofchaliss/themis/verification/seam"
)

func repoRoot(t *testing.T) string {
	t.Helper()
	_, file, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(file), "..", "..", "..", "..", "..")
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
	repo := filepath.Join(root, "demo")
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

// dynModel is a parent model driven by the test: parent turns see the
// conversation and may read the record (a test convenience — the real
// model derives references from framed results it saw); the delegated
// call is recognized by its shape — no tools offered.
type dynModel struct {
	parent    func(turn int, conv []model.Message) model.ExecutionResponse
	delegated func(req model.ExecutionRequest) (*model.ExecutionResponse, error)
	turn      int
	delegCall int
	lastConv  []model.Message
	delegReq  *model.ExecutionRequest
}

func (m *dynModel) Name() string { return "dyn" }
func (m *dynModel) Execute(ctx stdctx.Context, req model.ExecutionRequest) (*model.ExecutionResponse, error) {
	if len(req.Tools) == 0 {
		m.delegCall++
		r := req
		m.delegReq = &r
		return m.delegated(req)
	}
	m.turn++
	m.lastConv = append([]model.Message(nil), req.Messages...)
	r := m.parent(m.turn, req.Messages)
	return &r, nil
}

func call(id, name, args string) model.ExecutionResponse {
	return model.ExecutionResponse{Termination: model.TerminationToolCalls,
		ToolCalls: []model.ToolCall{{ID: id, Name: name, Arguments: json.RawMessage(args)}}}
}

type world struct {
	o        *orchestration.Orchestrator
	sroot    *state.Root
	envDir   string
	stateDir string
	root     string
	sha      string
	scope    string // template_scope JSON array
	seam     *Seam
	// safety is this world's PRIVATE copy of the safety root, so tests
	// that change a root file never touch the repository.
	safety string
	// procedure, when set, is an activated parent skill-scope source
	// (envelope skill_procedure_path/sha256) — for the carry tests.
	procedure string
}

// delegationRegistry copies the governed delegation policies into a
// private root and lets a test mutate the template before re-pinning
// — the registration act, simulated (state and bounds only; the
// composition members stay the governed bytes).
func delegationRegistry(t *testing.T, mutate func(tpl map[string]any, entry map[string]any)) string {
	t.Helper()
	root := repoRoot(t)
	dst := t.TempDir()
	src := filepath.Join(root, "policies/delegation")
	if err := os.MkdirAll(filepath.Join(dst, "dependency-triage"), 0o755); err != nil {
		t.Fatal(err)
	}
	for _, f := range []string{"contract.json", "instruction.md", "template.json"} {
		b, err := os.ReadFile(filepath.Join(src, "dependency-triage", f))
		if err != nil {
			t.Fatal(err)
		}
		wj(t, filepath.Join(dst, "dependency-triage"), f, string(b))
	}
	rb, _ := os.ReadFile(filepath.Join(src, "registry.json"))
	var reg map[string]any
	_ = json.Unmarshal(rb, &reg)
	tb, _ := os.ReadFile(filepath.Join(dst, "dependency-triage/template.json"))
	var tpl map[string]any
	_ = json.Unmarshal(tb, &tpl)
	entry := reg["entries"].([]any)[0].(map[string]any)
	if mutate != nil {
		mutate(tpl, entry)
	}
	tb, _ = json.Marshal(tpl)
	wj(t, filepath.Join(dst, "dependency-triage"), "template.json", string(tb))
	entry["template_sha256"] = sha256hex(tb)
	rb, _ = json.Marshal(reg)
	return wj(t, dst, "registry.json", string(rb))
}

func sha256hex(b []byte) string { return hex64(b) }

func newWorld(t *testing.T, m model.Interface, registryPath string, wire bool) *world {
	t.Helper()
	root := repoRoot(t)
	w := &world{envDir: t.TempDir(), stateDir: filepath.Join(t.TempDir(), "state"), root: root, scope: `["dependency-triage@1"]`}
	mirror, sha := mkMirror(t)
	w.sha = sha
	policyPath := filepath.Join(root, "policies/security/instruction-directive-patterns.json")
	w.safety = filepath.Join(t.TempDir(), "safety")
	if err := os.MkdirAll(w.safety, 0o755); err != nil {
		t.Fatal(err)
	}
	ents, err := os.ReadDir(filepath.Join(root, "instructions/global/safety"))
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range ents {
		b, _ := os.ReadFile(filepath.Join(root, "instructions/global/safety", e.Name()))
		wj(t, w.safety, e.Name(), string(b))
	}
	var d orchestration.Delegator
	if wire {
		policy, err := instructions.LoadPolicy(policyPath)
		if err != nil {
			t.Fatal(err)
		}
		if registryPath == "" {
			registryPath = filepath.Join(root, "policies/delegation/registry.json")
		}
		s, err := New(registryPath, policy)
		if err != nil {
			t.Fatal(err)
		}
		w.seam = s
		d = s
	}
	o, _, err := orchestration.Open(orchestration.Config{
		Unanchored: true,
		StateRoot:  w.stateDir, ArtifactDir: filepath.Join(t.TempDir(), "artifacts"),
		GitPath: gitBin(t), ProviderDir: t.TempDir(),
		SafetyRoot: w.safety,
		SystemRoot: filepath.Join(root, "instructions/global/system"),
		ThemisRoot: filepath.Join(root, "instructions/themis"),
		PolicyPath: policyPath,
		Model:      m,
		Delegator:  d,
	})
	if err != nil {
		t.Fatal(err)
	}
	w.o = o
	wj(t, w.envDir, "workflow.json", `{
 "version":1,"name":"triage-walk","initial":"ANALYZE",
 "declared_events":["turn-no-action","turn-provider-error","turns-exhausted","tool-error","signal:phase-completion-requested"],
 "phases":[
  {"name":"ANALYZE","capabilities":["read_file","delegate","declare_done"],"max_model_turns":8,"edges":[
    {"on":"signal:phase-completion-requested","to":"@complete"},
    {"on":"turn-no-action","to":"@stay","counter":3,"exhausted_to":"@fail"},
    {"on":"turn-provider-error","to":"@fail"},
    {"on":"tool-error","to":"@stay","counter":5,"exhausted_to":"@fail"},
    {"on":"turns-exhausted","to":"@fail"}]}
 ]}`)
	wj(t, w.envDir, "wceiling.json", `{"version":1,"allowed_tools":["read_file","delegate","declare_done"],"max_total_calls":30,"max_walk_length":200,"max_turns_per_phase":10}`)
	wj(t, w.envDir, "eceiling.json",
		`{"version":1,"mirror_root":"`+mirror+`","max_wall_deadline_sec":600,"max_file_bytes":1048576,"max_total_bytes":10485760,"max_file_count":500,"max_mem_bytes":1073741824,"max_cpu_time_sec":600,"max_proc_count":64}`)
	wj(t, w.envDir, "context-contract.json",
		`{"version":1,"workflow":"triage-walk","slots":[
		  {"name":"task-payload","kind":"task-brief","requirement":"required","classes":["external-untrusted"]}],
		  "sensitivity_ceiling":"public"}`)
	sroot, err := state.OpenRoot(w.stateDir)
	if err != nil {
		t.Fatal(err)
	}
	w.sroot = sroot
	return w
}

func (w *world) envelope(t *testing.T, taskID string) string {
	t.Helper()
	wj(t, w.envDir, taskID+"-spec.json",
		`{"version":1,"task_id":"`+taskID+`","repo":"demo","pinned_sha":"`+w.sha+`","limits":[{"dimension":"wall_deadline_s","value":120}]}`)
	wj(t, w.envDir, taskID+"-grant.json",
		`{"version":1,"task_id":"`+taskID+`","total_max_calls":30,"entries":[
		  {"tool":"read_file","max_calls":8,"workspace":"@workspace"},
		  {"tool":"delegate","max_calls":3,"template_scope":`+w.scope+`},
		  {"tool":"declare_done","max_calls":6}]}`)
	js := func(s string) string { b, _ := json.Marshal(s); return string(b) }
	proc := ""
	if w.procedure != "" {
		b, err := os.ReadFile(w.procedure)
		if err != nil {
			t.Fatal(err)
		}
		proc = `"skill_procedure_path":` + js(w.procedure) + `,"skill_procedure_sha256":` + js(sha256hex(b)) + `,`
	}
	return wj(t, w.envDir, "envelope-"+taskID+".json", `{
	 "version":1,"task_id":"`+taskID+`","model":"dyn","turn_timeout_sec":180,`+proc+`
	 "payload":"Read go.mod, delegate a triage of what you read, then declare done.",
	 "workflow_path":`+js(filepath.Join(w.envDir, "workflow.json"))+`,
	 "workflow_ceiling_path":`+js(filepath.Join(w.envDir, "wceiling.json"))+`,
	 "registry_path":`+js(filepath.Join(w.root, "policies/tools/registry-v5.json"))+`,
	 "grant_path":`+js(filepath.Join(w.envDir, taskID+"-grant.json"))+`,
	 "exec_ceiling_path":`+js(filepath.Join(w.envDir, "eceiling.json"))+`,
	 "spec_path":`+js(filepath.Join(w.envDir, taskID+"-spec.json"))+`,
	 "context_contract_path":`+js(filepath.Join(w.envDir, "context-contract.json"))+`}`)
}

// readRef finds the authorized read_file audit in the record and
// returns its "<seq>:<objectID>" reference.
func (w *world) readRef(t *testing.T, taskID string) (int64, string) {
	if t != nil {
		t.Helper()
	}
	evs, err := w.sroot.ReadEvents(taskID)
	if err != nil {
		if t != nil {
			t.Fatal(err)
		}
		return 0, ""
	}
	for _, e := range evs {
		if e.Class != state.EvL4Audit {
			continue
		}
		var b struct{ Tool, Decision string }
		_ = json.Unmarshal(e.Body, &b)
		if b.Tool == "read_file" && b.Decision == "authorized" && len(e.Refs) == 1 {
			return e.Seq, e.Refs[0].ID
		}
	}
	if t != nil {
		t.Fatal("no authorized read_file audit in the record")
	}
	return 0, ""
}

func ref(seq int64, id string) string {
	return strings.TrimSpace(strings.Join([]string{itoa(seq), id}, ":"))
}

func itoa(n int64) string { b, _ := json.Marshal(n); return string(b) }

// triageParent is the standard parent script: read go.mod, delegate
// over that result with the given evidence/brief, then declare done.
func triageParent(w *world, taskID string, evidenceFn func(seq int64, id string) string, brief string) func(int, []model.Message) model.ExecutionResponse {
	return func(turn int, conv []model.Message) model.ExecutionResponse {
		switch turn {
		case 1:
			return call("c1", "read_file", `{"path":"go.mod"}`)
		case 2:
			seq, id := w.readRef(nil, taskID)
			ev := evidenceFn(seq, id)
			if ev == "L2DELIVERY" {
				evs, _ := w.sroot.ReadEvents(taskID)
				for _, e := range evs {
					if e.Class == state.EvL2Delivery && len(e.Refs) == 1 {
						ev = ref(e.Seq, e.Refs[0].ID)
					}
				}
			}
			args, _ := json.Marshal(map[string]string{"template": "dependency-triage@1", "evidence": ev, "brief": brief})
			return call("c2", "delegate", string(args))
		default:
			return call("c3", "declare_done", `{}`)
		}
	}
}

const triageNote = "Triage: go.mod pins vulnerable-dep v1; confidence: evidence-backed."

func okDelegated(req model.ExecutionRequest) (*model.ExecutionResponse, error) {
	return &model.ExecutionResponse{Content: triageNote, Termination: model.TerminationStop,
		Identity:   model.Identity{WireModel: req.Model, Runtime: "dyn", Reported: req.Model},
		Provenance: model.Provenance{Endpoint: "dyn://local"}}, nil
}

func events(t *testing.T, w *world, taskID string) []state.Event {
	t.Helper()
	evs, err := w.sroot.ReadEvents(taskID)
	if err != nil {
		t.Fatal(err)
	}
	return evs
}

func delegationEvent(t *testing.T, evs []state.Event) (*state.Event, *delegation.Event) {
	t.Helper()
	for i := range evs {
		if evs[i].Class == state.EvL8Delegation {
			d, err := delegation.Decode(evs[i].Body)
			if err != nil {
				t.Fatal(err)
			}
			return &evs[i], d
		}
	}
	return nil, nil
}

func delegateAudit(t *testing.T, evs []state.Event) (state.Event, map[string]any) {
	t.Helper()
	for _, e := range evs {
		if e.Class != state.EvL4Audit {
			continue
		}
		var b map[string]any
		_ = json.Unmarshal(e.Body, &b)
		if b["Tool"] == "delegate" {
			return e, b
		}
	}
	t.Fatal("no delegate audit")
	return state.Event{}, nil
}

// THE POSITIVE TWIN.
func TestDelegationPositivePath(t *testing.T) {
	m := &dynModel{delegated: okDelegated}
	w := newWorld(t, m, "", true)
	const task = "t-deleg-ok"
	m.parent = triageParent(w, task, func(seq int64, id string) string { return ref(seq, id) }, "Which dependency is vulnerable and why?")
	res, err := w.o.SubmitTask(w.envelope(t, task))
	if err != nil {
		t.Fatalf("the genuine delegate walk must be admitted and complete: %v", err)
	}
	if res.Status != state.StatusCompleted {
		t.Fatalf("status %s, want COMPLETED", res.Status)
	}
	if m.delegCall != 1 {
		t.Fatalf("exactly one delegated model call, got %d", m.delegCall)
	}
	evs := events(t, w, task)
	audit, ab := delegateAudit(t, evs)
	if ab["Decision"] != "authorized" || len(audit.Refs) != 1 {
		t.Fatalf("the delegate call must be authorized with its capture referenced: %v %v", ab, audit.Refs)
	}
	l8, d := delegationEvent(t, evs)
	if l8 == nil {
		t.Fatal("no l8-delegation witness")
	}
	if d.ParentCallSeq != audit.Seq || d.Outcome != delegation.OutcomeCompleted || d.Template.Ref != "dependency-triage@1" {
		t.Fatalf("witness: %+v", d)
	}
	readSeq, readID := w.readRef(t, task)
	if len(d.EvidenceRefs) != 1 || d.EvidenceRefs[0].Seq != readSeq || d.EvidenceRefs[0].ObjectID != readID || d.EvidenceRefs[0].DerivedClass != "external-untrusted" {
		t.Fatalf("evidence_refs must be the re-established reference with its derived class: %+v", d.EvidenceRefs)
	}
	if readSeq >= d.ParentCallSeq || d.ParentCallSeq >= l8.Seq {
		t.Fatalf("ordering evidence.seq < parent_call_seq < l8 seq violated: %d %d %d", readSeq, d.ParentCallSeq, l8.Seq)
	}
	// Window purity (C-L8-8): nothing between the authorizing audit
	// and the witness.
	for _, e := range evs {
		if e.Seq > audit.Seq && e.Seq < l8.Seq {
			t.Fatalf("event %d (%s) inside the l4-audit → l8-delegation window", e.Seq, e.Class)
		}
	}
	// The capture and the witness agree on every identity (two
	// independent invocations of deterministic machinery, C-L8-13).
	capBytes, err := w.sroot.Resolve(audit, 0)
	if err != nil {
		t.Fatal(err)
	}
	var cap capture
	if err := json.Unmarshal(capBytes, &cap); err != nil {
		t.Fatal(err)
	}
	if cap.EISHash != d.Composition.EISHash || cap.ContractHash != d.Composition.ContractHash || cap.PayloadHash != d.Composition.PayloadHash || cap.RenderHash != d.Composition.RenderHash || cap.TemplateHash != d.Template.TemplateHash {
		t.Fatalf("capture ≠ witness: %+v vs %+v", cap, d.Composition)
	}
	// Durable closure: composition = the exact model input; output =
	// the exact model output; template bytes referenced.
	comp, err := w.sroot.Store().GetObject(d.Composition.CompositionObjectRef)
	if err != nil {
		t.Fatal(err)
	}
	var msgs []model.Message
	if err := json.Unmarshal(comp, &msgs); err != nil || len(msgs) != 2 || msgs[0].Role != model.RoleSystem || msgs[1].Role != model.RoleUser {
		t.Fatalf("composition object must be the [system, user] input: %v %d", err, len(msgs))
	}
	if m.delegReq == nil || len(m.delegReq.Messages) != 2 || m.delegReq.Messages[1].Content != msgs[1].Content || m.delegReq.Messages[0].Content != msgs[0].Content || m.delegReq.Model != "dyn" {
		t.Fatal("the delegated model must have been called with exactly the stored composition under the parent's model identity")
	}
	if !strings.Contains(msgs[1].Content, "vulnerable-dep v1") || !strings.Contains(msgs[1].Content, "Which dependency is vulnerable") {
		t.Fatalf("the delegated input must carry the referenced evidence and the brief: %s", msgs[1].Content)
	}
	if !strings.Contains(msgs[0].Content, "Delegation instruction: dependency-triage") {
		t.Fatalf("the template's instruction must be in the delegated EIS: %.200s", msgs[0].Content)
	}
	out, err := w.sroot.Store().GetObject(d.OutputObjectRef)
	if err != nil || string(out) != triageNote {
		t.Fatalf("output object: %v %q", err, out)
	}
	if len(d.TemplateObjectRefs) != 3 {
		t.Fatalf("manifest, contract, instruction bytes must be referenced: %v", d.TemplateObjectRefs)
	}
	if d.ModelIdentity.Governed.Name != "dyn" || d.ModelIdentity.Execution.Reported != "dyn" || d.ModelIdentity.Execution.Endpoint != "dyn://local" {
		t.Fatalf("model identity: %+v", d.ModelIdentity)
	}
	// Re-entry (C-L8-11 A): the parent's next turn saw one tool
	// message paired to c2, framed under external-untrusted, whose hash
	// is the output object identity, carrying the model-authored bytes.
	var paired *model.Message
	for i := range m.lastConv {
		if m.lastConv[i].Role == model.RoleTool && m.lastConv[i].ToolCallID == "c2" {
			paired = &m.lastConv[i]
		}
	}
	if paired == nil {
		t.Fatal("the delegation result must re-enter as the paired tool message")
	}
	for _, want := range []string{"kind: tool-result:delegate", "authority: external-untrusted", "hash: " + strings.TrimPrefix(d.OutputObjectRef, "sha256:"), triageNote} {
		if !strings.Contains(paired.Content, want) {
			t.Fatalf("re-entry frame missing %q:\n%s", want, paired.Content)
		}
	}
	if strings.Contains(paired.Content, "template_hash") {
		t.Fatal("the instantiation capture must never reach the model")
	}
}

// Stage A: the L4 gate, not the seam.
func TestDelegationOutsideScopeIsAnL4Denial(t *testing.T) {
	m := &dynModel{delegated: okDelegated}
	w := newWorld(t, m, "", true)
	const task = "t-deleg-scope"
	m.parent = func(turn int, conv []model.Message) model.ExecutionResponse {
		if turn == 1 {
			return call("c1", "delegate", `{"template":"cve-analysis@1","brief":"x"}`)
		}
		return call("c2", "declare_done", `{}`)
	}
	res, err := w.o.SubmitTask(w.envelope(t, task))
	if err != nil || res.Status != state.StatusCompleted {
		t.Fatalf("%+v %v", res, err)
	}
	evs := events(t, w, task)
	_, ab := delegateAudit(t, evs)
	if ab["Decision"] != "denied" || !strings.Contains(ab["TracePredicate"].(string), "template-outside-grant-scope") {
		t.Fatalf("out-of-scope template must be an L4 target refusal: %v", ab)
	}
	if l8, _ := delegationEvent(t, evs); l8 != nil || m.delegCall != 0 {
		t.Fatal("no delegation may exist after an L4 denial")
	}
}

// Stage B refusals: typed in the l4-audit, no witness, no objects.
func TestDelegationStageBRefusals(t *testing.T) {
	cases := []struct {
		name     string
		evidence func(seq int64, id string) string
		brief    string
		want     string
	}{
		{"evidence beyond the stream", func(seq int64, id string) string { return ref(999, id) }, "b", "delegation-refused:evidence-unreachable"},
		{"evidence names an unreferenced object", func(seq int64, id string) string {
			return ref(seq, "sha256:"+strings.Repeat("0", 64))
		}, "b", "delegation-refused:evidence-unreachable"},
		{"evidence names a non-evidence event", func(seq int64, id string) string { return ref(1, id) }, "b", "delegation-refused:evidence-unreachable"},
		{"evidence names an unselectable witness that does reference the object", func(seq int64, id string) string {
			return "L2DELIVERY"
		}, "b", "delegation-refused:evidence-unreachable"},
		{"duplicate reference", func(seq int64, id string) string { return ref(seq, id) + "," + ref(seq, id) }, "b", "delegation-refused:evidence-duplicate"},
		{"brief over the template bound", func(seq int64, id string) string { return ref(seq, id) }, strings.Repeat("b", 4097), "delegation-refused:brief-over-bound"},
		{"brief empty but required by the contract", func(seq int64, id string) string { return ref(seq, id) }, "", "delegation-refused:compose-refused"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			m := &dynModel{delegated: okDelegated}
			w := newWorld(t, m, "", true)
			task := "t-b-" + strings.NewReplacer(" ", "-", "'", "").Replace(c.name)
			m.parent = triageParent(w, task, c.evidence, c.brief)
			res, err := w.o.SubmitTask(w.envelope(t, task))
			if err != nil || res.Status != state.StatusCompleted {
				t.Fatalf("a refused delegation is data; the walk continues: %+v %v", res, err)
			}
			evs := events(t, w, task)
			_, ab := delegateAudit(t, evs)
			if ab["Decision"] != "error" || ab["ErrClass"] != c.want {
				t.Fatalf("want %s in the audit, got %v", c.want, ab)
			}
			if l8, _ := delegationEvent(t, evs); l8 != nil || m.delegCall != 0 {
				t.Fatal("a stage-B refusal mints no witness and calls no model")
			}
			// The model saw the closed class, unframed.
			for _, msg := range m.lastConv {
				if msg.Role == model.RoleTool && msg.ToolCallID == "c2" && msg.Content != `{"error":"`+c.want+`"}` {
					t.Fatalf("typed error to the model: %s", msg.Content)
				}
			}
		})
	}
}

// Assembly refusals: the seam must exist, the scope must resolve.
func TestDelegationAssemblyRefusals(t *testing.T) {
	t.Run("no delegator wired", func(t *testing.T) {
		m := &dynModel{delegated: okDelegated}
		w := newWorld(t, m, "", false)
		w.scope = `[]`
		if _, err := w.o.SubmitTask(w.envelope(t, "t-nil")); !errors.Is(err, orchestration.ErrAssembly) || !strings.Contains(err.Error(), "no L8 delegator") {
			t.Fatalf("phase exposing delegate without a seam must refuse at assembly: %v", err)
		}
	})
	t.Run("scope entry unregistered", func(t *testing.T) {
		m := &dynModel{delegated: okDelegated}
		w := newWorld(t, m, "", true)
		w.scope = `["dependency-triage@1","ghost@1"]`
		if _, err := w.o.SubmitTask(w.envelope(t, "t-ghost")); !errors.Is(err, orchestration.ErrAssembly) || !strings.Contains(err.Error(), "does not resolve") {
			t.Fatalf("unregistered scope entry must refuse at assembly: %v", err)
		}
	})
	t.Run("scope entry withdrawn", func(t *testing.T) {
		reg := delegationRegistry(t, func(tpl, entry map[string]any) { entry["state"] = "withdrawn" })
		m := &dynModel{delegated: okDelegated}
		w := newWorld(t, m, reg, true)
		if _, err := w.o.SubmitTask(w.envelope(t, "t-withdrawn")); !errors.Is(err, orchestration.ErrAssembly) || !strings.Contains(err.Error(), "withdrawn") {
			t.Fatalf("withdrawn template in scope must refuse: %v", err)
		}
	})
	t.Run("scope named without a delegator", func(t *testing.T) {
		m := &dynModel{delegated: okDelegated}
		w := newWorld(t, m, "", false)
		wj(t, w.envDir, "workflow.json", strings.Replace(mustRead(t, filepath.Join(w.envDir, "workflow.json")), `"read_file","delegate","declare_done"`, `"read_file","declare_done"`, 1))
		if _, err := w.o.SubmitTask(w.envelope(t, "t-scope-nil")); !errors.Is(err, orchestration.ErrAssembly) || !strings.Contains(err.Error(), "no L8 delegator is wired to validate") {
			t.Fatalf("%v", err)
		}
	})
}

func mustRead(t *testing.T, p string) string {
	t.Helper()
	b, err := os.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

// Post-instance outcomes: witnessed, typed, never content.
func TestDelegationPostInstanceOutcomes(t *testing.T) {
	run := func(t *testing.T, reg string, delegated func(model.ExecutionRequest) (*model.ExecutionResponse, error), task string) (*dynModel, *world, *delegation.Event) {
		t.Helper()
		m := &dynModel{delegated: delegated}
		w := newWorld(t, m, reg, true)
		m.parent = triageParent(w, task, func(seq int64, id string) string { return ref(seq, id) }, "triage")
		res, err := w.o.SubmitTask(w.envelope(t, task))
		if err != nil || res.Status != state.StatusCompleted {
			t.Fatalf("a typed delegation failure is data; the walk continues: %+v %v", res, err)
		}
		_, d := delegationEvent(t, events(t, w, task))
		if d == nil {
			t.Fatal("a post-instance outcome must be witnessed")
		}
		return m, w, d
	}
	pairedError := func(t *testing.T, m *dynModel, want string) {
		t.Helper()
		for _, msg := range m.lastConv {
			if msg.Role == model.RoleTool && msg.ToolCallID == "c2" {
				if msg.Content != `{"error":"`+want+`"}` {
					t.Fatalf("want unframed %s, got %s", want, msg.Content)
				}
				return
			}
		}
		t.Fatal("no paired message")
	}
	t.Run("provider error", func(t *testing.T) {
		m, _, d := run(t, "", func(model.ExecutionRequest) (*model.ExecutionResponse, error) { return nil, errors.New("boom") }, "t-perr")
		if d.Outcome != delegation.OutcomeProviderError || d.OutputObjectRef != "" || d.Composition.CompositionObjectRef == "" {
			t.Fatalf("%+v", d)
		}
		pairedError(t, m, "delegation-provider-error")
	})
	t.Run("output over bound", func(t *testing.T) {
		reg := delegationRegistry(t, func(tpl, entry map[string]any) { tpl["max_output_bytes"] = 8 })
		m, w, d := run(t, reg, okDelegated, "t-over")
		if d.Outcome != delegation.OutcomeOutputOverBound || d.OutputObjectRef == "" {
			t.Fatalf("%+v", d)
		}
		// Captured whole, never truncated (D-L8-16 §5).
		if out, err := w.sroot.Store().GetObject(d.OutputObjectRef); err != nil || string(out) != triageNote {
			t.Fatalf("%v %q", err, out)
		}
		pairedError(t, m, "delegation-output-over-bound")
	})
	t.Run("model identity mismatch", func(t *testing.T) {
		m, _, d := run(t, "", func(req model.ExecutionRequest) (*model.ExecutionResponse, error) {
			r, _ := okDelegated(req)
			r.Identity.Reported = "something-else:7b"
			return r, nil
		}, "t-ident")
		if d.Outcome != delegation.OutcomeModelIdentityMismatch || d.ModelIdentity.Execution.Reported != "something-else:7b" {
			t.Fatalf("%+v", d)
		}
		pairedError(t, m, "delegation-model-identity-mismatch")
	})
}

// Stage D: machinery failure at each new fault point mints no outcome
// and fails the task; a pre-commit crash leaves orphans, never a fact.
func TestDelegationFaultPoints(t *testing.T) {
	for _, point := range []string{"delegation.pre-composition-store", "delegation.pre-output-store", "delegation.pre-event-commit"} {
		t.Run(point, func(t *testing.T) {
			m := &dynModel{delegated: okDelegated}
			w := newWorld(t, m, "", true)
			task := "t-fault-" + strings.TrimPrefix(point, "delegation.")
			m.parent = triageParent(w, task, func(seq int64, id string) string { return ref(seq, id) }, "triage")
			orchestration.SetFault(func(p string) error {
				if p == point {
					return errors.New("injected: " + p)
				}
				return nil
			})
			defer orchestration.SetFault(nil)
			res, err := w.o.SubmitTask(w.envelope(t, task))
			if err == nil || res.Status != state.StatusFailed {
				t.Fatalf("machinery failure must fail the task through the invariant path: %+v %v", res, err)
			}
			evs := events(t, w, task)
			if l8, _ := delegationEvent(t, evs); l8 != nil {
				t.Fatal("no witness may exist after a fault before commit")
			}
			if point == "delegation.pre-event-commit" && m.delegCall != 1 {
				t.Fatal("the model ran; without the witness the system establishes no delegation (stage E)")
			}
		})
	}
}

// C-L8-13: the post-hook re-derives the composition and must agree
// with the executor's capture; disagreement is stage D — no outcome,
// no witness — reachable directly against the seam.
func TestDelegateRefusesDoctoredCapture(t *testing.T) {
	m := &dynModel{delegated: okDelegated}
	w := newWorld(t, m, "", true)
	const task = "t-doctored"
	// Run a real walk to have a record with a read_file evidence item.
	m.parent = func(turn int, conv []model.Message) model.ExecutionResponse {
		if turn == 1 {
			return call("c1", "read_file", `{"path":"go.mod"}`)
		}
		return call("c2", "declare_done", `{}`)
	}
	if res, err := w.o.SubmitTask(w.envelope(t, task)); err != nil || res.Status != state.StatusCompleted {
		t.Fatalf("%+v %v", res, err)
	}
	seq, id := w.readRef(t, task)
	base := orchestration.InstantiationRequest{
		TaskID: task, Record: w.sroot, Template: "dependency-triage@1",
		Evidence: []orchestration.EvidenceRef{{Seq: seq, ObjectID: id}}, Brief: "triage",
		ParentSources: []instructions.Source{{Kind: instructions.ScopeHarnessSafety, Root: w.safety},
			{Kind: instructions.ScopeHarnessSystem, Root: filepath.Join(w.root, "instructions/global/system")}},
		ParentEIS: parentEIS(t, w), ParentSensitivityCeiling: hctx.SensitivityPublic,
		RegistryHash: registryHashOf(t, w), ToolTrust: func(string) (hctx.AuthorityClass, bool) { return hctx.AuthorityExternalUntrusted, true },
	}
	good, err := w.seam.Instantiate(base)
	if err != nil {
		t.Fatalf("instantiate: %v", err)
	}
	tr, err := w.sroot.CreateTask("t-doctored-2", state.TaskOptions{})
	if err != nil {
		t.Fatal(err)
	}
	defer tr.Close()
	req := orchestration.DelegationRequest{InstantiationRequest: base, Task: tr, ParentCallSeq: seq + 1, CallID: "x",
		Capture: []byte(strings.Replace(string(good), `"eis_hash":"`, `"eis_hash":"0`, 1)), Model: "dyn", Runtime: m, ModelRegistryHash: "unanchored"}
	if _, err := w.seam.Delegate(req); err == nil || !strings.Contains(err.Error(), "disagrees with the instantiation capture") {
		t.Fatalf("a doctored capture must be stage D: %v", err)
	}
	if m.delegCall != 0 {
		t.Fatal("no model call may happen before the capture agrees")
	}
	// And the genuine capture proceeds to a witness on this record.
	req.Capture = good
	res, err := w.seam.Delegate(req)
	if err != nil || res.Outcome != "completed" {
		t.Fatalf("%+v %v", res, err)
	}
}

func registryHashOf(t *testing.T, w *world) string {
	t.Helper()
	evs := events(t, w, "t-doctored")
	for _, e := range evs {
		if e.Class == state.EvL4Audit {
			var b struct{ RegistryHash string }
			_ = json.Unmarshal(e.Body, &b)
			return b.RegistryHash
		}
	}
	t.Fatal("no audit")
	return ""
}

// C-L8-4: the mandatory roots carry unconditionally; an optional scope
// carries only where the template's filter names it; the template's
// own instruction is the one source it adds.
func TestDelegatedEISCarryFilter(t *testing.T) {
	const marker = "PARENT-PROCEDURE-MARKER: prefer the smallest version bump that closes the advisory."
	for _, tc := range []struct {
		name    string
		carry   []any
		wantPar bool
	}{
		{"skill carried", []any{"repository", "skill"}, true},
		{"skill not carried", []any{"repository"}, false},
		{"nothing carried", []any{}, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			reg := delegationRegistry(t, func(tpl, entry map[string]any) { tpl["eis_carry_scopes"] = tc.carry })
			m := &dynModel{delegated: okDelegated}
			w := newWorld(t, m, reg, true)
			w.procedure = wj(t, w.envDir, "procedure.md", "# Procedure\n\n- "+marker+"\n")
			task := "t-carry-" + strings.ReplaceAll(tc.name, " ", "-")
			m.parent = triageParent(w, task, func(seq int64, id string) string { return ref(seq, id) }, "triage")
			if res, err := w.o.SubmitTask(w.envelope(t, task)); err != nil || res.Status != state.StatusCompleted {
				t.Fatalf("%+v %v", res, err)
			}
			if m.delegReq == nil {
				t.Fatal("no delegated call")
			}
			sys := m.delegReq.Messages[0].Content
			if strings.Contains(sys, marker) != tc.wantPar {
				t.Fatalf("parent skill procedure carried=%v, want %v", !tc.wantPar, tc.wantPar)
			}
			// Roots are unconditional: a protected safety instruction is
			// always in the delegated EIS.
			if !strings.Contains(sys, "advisory input to human and governed decisions") {
				t.Fatal("the mandatory safety root must carry regardless of the filter")
			}
			// The template's own instruction is always present.
			if !strings.Contains(sys, "Delegation instruction: dependency-triage") {
				t.Fatal("the template instruction must be present")
			}
			// And the parent's own EIS did carry the procedure.
			if !strings.Contains(m.lastConv[0].Content, marker) {
				t.Fatal("fixture: the parent EIS must contain the procedure")
			}
		})
	}
}

// D-L8-18: L10's history view observes the delegation read-only — no
// evidence kind, no contract, no outcome vocabulary of its own.
func TestL10HistoryObservesDelegation(t *testing.T) {
	m := &dynModel{delegated: okDelegated}
	w := newWorld(t, m, "", true)
	const task = "t-l10-view"
	m.parent = triageParent(w, task, func(seq int64, id string) string { return ref(seq, id) }, "triage")
	if res, err := w.o.SubmitTask(w.envelope(t, task)); err != nil || res.Status != state.StatusCompleted {
		t.Fatalf("%+v %v", res, err)
	}
	view, err := vseam.TaskVerificationHistory(w.sroot, task)
	if err != nil {
		t.Fatal(err)
	}
	l8, _ := delegationEvent(t, events(t, w, task))
	if len(view.Delegations) != 1 || view.Delegations[0].Seq != l8.Seq || view.Delegations[0].Template != "dependency-triage@1" || view.Delegations[0].Outcome != "completed" {
		t.Fatalf("%+v", view.Delegations)
	}
	if len(view.History) != 0 || len(view.Latest) != 0 {
		t.Fatal("a delegation is not a verification instance")
	}
}

// Stage D inside instantiation (D-L8-15, C-L8-5 A; architecture review
// HIGH-1): record corruption discovered while the delegate executor
// instantiates is an invariant, never a tool error the walk continues
// past — no audit claiming "seam-unavailable" commits, the task FAILS.
func TestCorruptionDuringInstantiationIsStageD(t *testing.T) {
	m := &dynModel{delegated: okDelegated}
	w := newWorld(t, m, "", true)
	const task = "t-corrupt"
	m.parent = func(turn int, conv []model.Message) model.ExecutionResponse {
		switch turn {
		case 1:
			return call("c1", "read_file", `{"path":"go.mod"}`)
		case 2:
			seq, id := w.readRef(nil, task)
			// Corrupt the referenced object on disk between the read and
			// the delegation: the store's re-hash must catch it.
			hex := strings.TrimPrefix(id, "sha256:")
			_ = filepath.WalkDir(w.stateDir, func(p string, de os.DirEntry, err error) error {
				if err == nil && !de.IsDir() && strings.Contains(p, hex) {
					_ = os.WriteFile(p, []byte("tampered\n"), 0o644)
				}
				return nil
			})
			args, _ := json.Marshal(map[string]string{"template": "dependency-triage@1", "evidence": ref(seq, id), "brief": "triage"})
			return call("c2", "delegate", string(args))
		}
		return call("c3", "declare_done", `{}`)
	}
	res, err := w.o.SubmitTask(w.envelope(t, task))
	if err == nil || !errors.Is(err, orchestration.ErrInvariant) || res.Status != state.StatusFailed {
		t.Fatalf("corruption at instantiation must be stage D: %+v %v", res, err)
	}
	for _, e := range events(t, w, task) {
		if e.Class == state.EvL4Audit && strings.Contains(string(e.Body), `"Tool":"delegate"`) {
			t.Fatalf("no audit may claim a tool error for a machinery failure: %s", e.Body)
		}
		if e.Class == state.EvL8Delegation {
			t.Fatal("no witness")
		}
	}
	if m.delegCall != 0 {
		t.Fatal("no model call")
	}
}

// parentEIS resolves the world's roots exactly as the parent does, for
// direct seam calls.
func parentEIS(t *testing.T, w *world) *instructions.EffectiveSet {
	t.Helper()
	policy, err := instructions.LoadPolicy(filepath.Join(w.root, "policies/security/instruction-directive-patterns.json"))
	if err != nil {
		t.Fatal(err)
	}
	eis, err := instructions.Resolve(instructions.Config{Policy: policy, TaskID: "x"},
		instructions.Source{Kind: instructions.ScopeHarnessSafety, Root: w.safety},
		instructions.Source{Kind: instructions.ScopeHarnessSystem, Root: filepath.Join(w.root, "instructions/global/system")})
	if err != nil {
		t.Fatal(err)
	}
	return eis
}

// Security review MED-1 (D-L9-11, C-L8-14 F): the delegated resolution
// re-reads the roots; a root file that changed under the running task
// is stage D, never a silent re-read into the delegated context.
func TestRootChangeUnderRunningTaskIsStageD(t *testing.T) {
	m := &dynModel{delegated: okDelegated}
	w := newWorld(t, m, "", true)
	const task = "t-root-change"
	m.parent = func(turn int, conv []model.Message) model.ExecutionResponse {
		switch turn {
		case 1:
			return call("c1", "read_file", `{"path":"go.mod"}`)
		case 2:
			p := filepath.Join(w.safety, "advisory-only.md")
			b, _ := os.ReadFile(p)
			_ = os.WriteFile(p, append(b, []byte("\nChanged under the running task.\n")...), 0o644)
			seq, id := w.readRef(nil, task)
			args, _ := json.Marshal(map[string]string{"template": "dependency-triage@1", "evidence": ref(seq, id), "brief": "triage"})
			return call("c2", "delegate", string(args))
		}
		return call("c3", "declare_done", `{}`)
	}
	res, err := w.o.SubmitTask(w.envelope(t, task))
	if err == nil || !errors.Is(err, orchestration.ErrInvariant) || res.Status != state.StatusFailed || !strings.Contains(err.Error(), "governed root changed") {
		t.Fatalf("a changed root must be stage D: %+v %v", res, err)
	}
	if m.delegCall != 0 {
		t.Fatal("no model call")
	}
}

// Security review MED-2 (C-L8-7 §5): every reference carries the parent
// contract's ceiling; a template whose contract ceiling is lower
// refuses at Gather — the template narrows, it never widens.
func TestTemplateCeilingNarrowsEvidence(t *testing.T) {
	m := &dynModel{delegated: okDelegated}
	w := newWorld(t, m, "", true)
	// The parent contract ceiling is "internal" for this world.
	wj(t, w.envDir, "context-contract.json",
		`{"version":1,"workflow":"triage-walk","slots":[
		  {"name":"task-payload","kind":"task-brief","requirement":"required","classes":["external-untrusted"]}],
		  "sensitivity_ceiling":"internal"}`)
	const task = "t-ceiling"
	m.parent = triageParent(w, task, func(seq int64, id string) string { return ref(seq, id) }, "triage")
	res, err := w.o.SubmitTask(w.envelope(t, task))
	if err != nil || res.Status != state.StatusCompleted {
		t.Fatalf("%+v %v", res, err)
	}
	_, ab := delegateAudit(t, events(t, w, task))
	if ab["Decision"] != "error" || ab["ErrClass"] != "delegation-refused:compose-refused" {
		t.Fatalf("an internal-ceiling reference must not enter a public-ceiling template: %v", ab)
	}
	if m.delegCall != 0 {
		t.Fatal("no model call")
	}
}
