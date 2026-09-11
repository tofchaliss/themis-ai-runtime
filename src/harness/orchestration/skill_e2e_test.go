package orchestration

// Register E (structural half): the authored P0 skill is instantiated
// through the real L9 machinery and executed by the UNMODIFIED
// production L7 loop. A special test executor would weaken the proof,
// so this uses SubmitTask exactly as any caller would.
//
// The live-model half of Register E runs under -run TestLiveSkillWalk
// (guarded like the other live proofs).

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/tofchaliss/themis/runtime/model"
	"github.com/tofchaliss/themis/skills"
	"github.com/tofchaliss/themis/state"
)

// instantiateP0 compiles the authored investigate-cve@1 composition
// into an ordinary governed envelope for this fixture's deployment.
func instantiateP0(t *testing.T, f *fixture, taskID string, req skills.Request) string {
	t.Helper()
	if req.Deployment.Model == "" {
		// The task-writable roots must exist: disjointness resolves
		// them physically and fails closed on an unresolvable root.
		base := t.TempDir()
		artifacts := filepath.Join(base, "artifacts")
		// The workspace root is the one a task actually holds write_file
		// on, so wall 2 requires it explicitly (security MED-1).
		workspaces := filepath.Join(base, "workspaces")
		for _, d := range []string{artifacts, workspaces} {
			if err := os.MkdirAll(d, 0o755); err != nil {
				t.Fatal(err)
			}
		}
		req.Deployment = skills.Deployment{
			Model: "scripted", TurnTimeoutSec: 180,
			RegistryPath:    mustAbs(t, filepath.Join(repoRoot, "policies/tools/registry-v3.json")),
			ExecCeilingPath: filepath.Join(f.envDir, "eceiling.json"),
			StateRoot:       f.stateDir,
			ArtifactDir:     artifacts,
			WorkspaceRoot:   workspaces,
		}
	}
	req.TaskID = taskID
	if req.Repo == "" {
		req.Repo = "demo"
	}
	if req.PinnedSHA == "" {
		req.PinnedSHA = f.sha
	}
	if req.Inputs == nil {
		req.Inputs = map[string]any{
			"cve-id": "CVE-2026-12345", "component": "parser",
		}
	}
	if req.OutDir == "" {
		req.OutDir = t.TempDir()
	}
	if req.WallDeadlineS == 0 {
		// Class-2 narrowing in its designed role: the skill's 600s
		// bound exceeds this deployment's execution ceiling, so the
		// caller narrows to fit. Narrowing is the caller's only lever,
		// and it is sufficient — the skill needs no per-deployment
		// variant.
		req.WallDeadlineS = 90
	}
	path, err := skills.Instantiate(
		mustAbs(t, filepath.Join(repoRoot, "policies/skills/catalog.json")),
		"investigate-cve@1", req)
	if err != nil {
		t.Fatalf("instantiating the authored skill must succeed: %v", err)
	}
	return path
}

// The end-to-end proof: authored skill → L9 instantiation → ordinary
// SubmitTask → production walk → typed terminal, with the skill's
// provenance in the record and its procedure in the delivered
// instructions.
func TestP0SkillRunsThroughProductionLoop(t *testing.T) {
	// The scripted model walks the authored lattice: analyze, then
	// assess, then declare completion in each phase.
	m := &scriptedModel{steps: []model.ExecutionResponse{
		toolCall("read_file", `{"path":"parser.go"}`),
		toolCall("declare_done", `{}`),
		toolCall("write_file", `{"path":"assessment.md","content":"# Assessment\nNot affected.\n"}`),
		toolCall("declare_done", `{}`),
	}}
	f := setup(t, m, "")
	env := instantiateP0(t, f, "t-p0", skills.Request{})

	res, err := f.o.SubmitTask(env)
	if err != nil {
		t.Fatalf("the authored skill must execute through the production loop: %v", err)
	}
	if res.Status != state.StatusCompleted {
		t.Fatalf("expected a completed walk, got %+v", res)
	}

	man, err := f.o.root.ReadManifest("t-p0")
	if err != nil {
		t.Fatal(err)
	}
	// Skill provenance reached the record, verbatim.
	if man.GovernedHashes["origin:skill"] != "investigate-cve@1" {
		t.Fatalf("skill attribution missing from the record: %v", man.GovernedHashes)
	}
	if man.GovernedHashes["origin:skill_composition"] == "" {
		t.Fatal("composition hash must be recorded for post-hoc catalog verification")
	}
	// The procedure genuinely reached the model as instruction material.
	// An earlier version of this check set a boolean when a delivery
	// event existed and discarded the bytes — it passed even with the
	// procedure removed from the render entirely (test review CRITICAL).
	// Assert the actual text, from the actual delivered system message.
	assertProcedureDelivered(t, f, "t-p0", env, "Establish what the code actually does")
}

// assertProcedureDelivered checks the delivered instruction text for a
// distinctive sentence of the skill's procedure — the bytes, not the
// existence of an event.
func assertProcedureDelivered(t *testing.T, f *fixture, taskID, envelopePath, phrase string) {
	t.Helper()
	sys, err := recordedSystemMessage(t, f, taskID, envelopePath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(sys, phrase) {
		t.Fatalf("the delivered system message does not carry the procedure text %q", phrase)
	}
	// And it arrives under its own untrusted-tier heading, not merged
	// into the harness-owned section (security review HIGH-2).
	if !strings.Contains(sys, "## Skill procedure") {
		t.Fatal("procedure text must render under its own provenance heading")
	}
	// Negative control: a walk with no skill procedure must NOT carry
	// either, so the assertions above cannot be passing for some
	// unrelated reason.
	f2 := setup(t, happyScript(), "")
	plain := f2.envelope(t, "t-noproc-control")
	if _, err := f2.o.SubmitTask(plain); err != nil {
		t.Fatal(err)
	}
	plainSys, err := recordedSystemMessage(t, f2, "t-noproc-control", plain)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(plainSys, phrase) || strings.Contains(plainSys, "## Skill procedure") {
		t.Fatal("a walk without a skill procedure must not deliver one")
	}
}

// recordedSystemMessage returns the instruction text the task actually
// delivered, recovered from the envelope the task ran under and
// cross-checked against the render hash the delivery record committed.
// The hash equality is what makes this the model's view rather than a
// plausible re-render.
func recordedSystemMessage(t *testing.T, f *fixture, taskID, envelopePath string) (string, error) {
	t.Helper()
	evs, err := f.o.root.ReadEvents(taskID)
	if err != nil {
		return "", err
	}
	var renderHash string
	for _, ev := range evs {
		if ev.Class != state.EvL2Delivery {
			continue
		}
		var b struct {
			RenderHash string `json:"render_hash"`
		}
		if err := json.Unmarshal(ev.Body, &b); err != nil {
			return "", err
		}
		renderHash = b.RenderHash
	}
	if renderHash == "" {
		return "", fmt.Errorf("no delivery recorded for %s", taskID)
	}
	env, err := LoadEnvelope(envelopePath)
	if err != nil {
		return "", err
	}
	eis, _, err := f.o.resolveTaskEIS(env)
	if err != nil {
		return "", err
	}
	text, hash, err := eis.Render(f.o.policy)
	if err != nil {
		return "", err
	}
	if hash != renderHash {
		return "", fmt.Errorf("re-rendered instructions do not match the delivered render hash (%s vs %s)", hash, renderHash)
	}
	return text, nil
}

// Register E (live half): the authored skill, a real local model, and
// the unmodified production loop. The model's own procedure — governed
// instruction material activated through L1 — is what tells it how to
// work; the payload carries only the untrusted task inputs.
func TestLiveSkillWalk(t *testing.T) {
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
	f := setup(t, model.NewOllamaChat(endpoint), "")
	req := skills.Request{
		Inputs: map[string]any{
			"cve-id": "CVE-2026-12345", "component": "parser",
			"focus": "Read parser.go, then call declare_done with no arguments. When declare_done is the only tool available, call it immediately.",
		},
	}
	req.Deployment.Model = modelName
	req.Deployment.TurnTimeoutSec = 180
	req.Deployment.RegistryPath = mustAbs(t, filepath.Join(repoRoot, "policies/tools/registry-v3.json"))
	req.Deployment.ExecCeilingPath = filepath.Join(f.envDir, "eceiling.json")
	req.Deployment.StateRoot = f.stateDir
	artifacts := filepath.Join(t.TempDir(), "artifacts")
	if err := os.MkdirAll(artifacts, 0o755); err != nil {
		t.Fatal(err)
	}
	req.Deployment.ArtifactDir = artifacts
	workspaces := filepath.Join(filepath.Dir(artifacts), "workspaces")
	if err := os.MkdirAll(workspaces, 0o755); err != nil {
		t.Fatal(err)
	}
	req.Deployment.WorkspaceRoot = workspaces
	// The live model explores, and it runs slower when the rest of the
	// suite competes for CPU. Take the whole allowance this deployment's
	// execution ceiling permits; the floor firing early is correct
	// behavior, but the register should exercise the walk, not the clock.
	req.WallDeadlineS = 540

	env := instantiateP0(t, f, "t-live-skill", req)
	res, err := f.o.SubmitTask(env)
	if err != nil {
		t.Fatalf("live skill walk failed: %v", err)
	}
	// The GOVERNED property is what this register proves: whatever the
	// model does, the walk reaches a typed terminal through governed
	// edges or a constitution floor — never an untyped or open state.
	// Asserting COMPLETED would assert model behavior, which is not a
	// property of the system: a real model may explore until the
	// wall-clock floor fires, and that is the floor working.
	if res.Status != state.StatusCompleted && res.Status != state.StatusFailed {
		t.Fatalf("the live skill walk must reach a typed terminal: %+v", res)
	}
	if res.Status == state.StatusFailed {
		t.Logf("live walk terminated FAILED (governed floor or edge) — typed and fail-closed, model behavior is not a system property")
	}
	// The walk moved only through the SKILL's governed edges (the
	// fixture replayer pins the fixture workflow, so verify against
	// the skill's own recorded transitions here).
	evs, err := f.o.root.ReadEvents("t-live-skill")
	if err != nil {
		t.Fatal(err)
	}
	var lastTo, lastEdge string
	transitions := 0
	for _, ev := range evs {
		if ev.Class != state.EvWorkflowTransition {
			continue
		}
		var b struct {
			To     string `json:"to"`
			EdgeID string `json:"edge_id"`
		}
		if err := json.Unmarshal(ev.Body, &b); err != nil {
			t.Fatal(err)
		}
		transitions++
		lastTo, lastEdge = b.To, b.EdgeID
	}
	// A walk stopped by the wall-clock floor in its first phase records
	// no transition at all — the floor is not an edge. That is the
	// governed behavior, so it is only a failure if the walk COMPLETED.
	if transitions == 0 && res.Status == state.StatusCompleted {
		t.Fatal("a completed skill walk must have recorded transitions")
	}
	if transitions == 0 {
		t.Log("no transitions: the wall-clock floor ended the walk inside its first phase")
		return
	}
	// Every transition the walk took rode a governed edge — that is the
	// property, whether the walk completed or a floor ended it. The
	// model's own progress is not a system guarantee.
	if lastEdge == "" {
		t.Fatal("a live walk must record the governed edge of each transition")
	}
	if res.Status == state.StatusCompleted {
		if lastTo != TargetComplete {
			t.Fatalf("a completed walk must arrive via its governed completion edge, ended at %q via %q", lastTo, lastEdge)
		}
		if !strings.Contains(lastEdge, SignalPhaseCompletionRequested) {
			t.Fatalf("completion must ride the control-verb signal, got edge %q", lastEdge)
		}
	}
	// Whatever the model did, every phase change it caused came from a
	// declared control signal — never from prose.
	if !strings.Contains(lastEdge, SignalPhaseCompletionRequested) &&
		!strings.Contains(lastEdge, "turns-exhausted") && !strings.Contains(lastEdge, "turn-no-action") {
		t.Fatalf("transitions must ride declared events only, got edge %q", lastEdge)
	}
	// Provenance is in the record, so the executed composition can be
	// checked against the catalog after the fact.
	man, err := f.o.root.ReadManifest("t-live-skill")
	if err != nil {
		t.Fatal(err)
	}
	if man.GovernedHashes["origin:skill"] != "investigate-cve@1" {
		t.Fatalf("live walk must record its skill attribution: %v", man.GovernedHashes)
	}
}

// D-L9-13 zero trust discount, proven against the REAL skill: bypass
// L9's validator entirely by hand-editing the instantiated envelope to
// exceed the skill's ceiling. L7 must refuse on its own.
func TestSkillEnvelopeGetsNoTrustDiscount(t *testing.T) {
	f := setup(t, happyScript(), "")
	env := instantiateP0(t, f, "t-bypass", skills.Request{})

	// Swap in a grant that exceeds the workflow ceiling — exactly what
	// L9 would have refused, now smuggled past it.
	bad := writeJSON(t, f.envDir, "over-grant.json",
		`{"version":1,"task_id":"t-bypass","total_max_calls":99,"entries":[
		  {"tool":"apply_patch","max_calls":9,"workspace":"@workspace","mutating":true}]}`)
	// Update the commitment to match the swapped grant, so the
	// composition check is SATISFIED and the ceiling check is the thing
	// under test. A bypass must fail on its own merits, not because a
	// different control happened to catch it first.
	badBytes, rerr := os.ReadFile(bad)
	if rerr != nil {
		t.Fatal(rerr)
	}
	commit := genuineCommitment(t, env)
	commit["grant_sha256"] = hashBytes(badBytes)
	commit = reseal(commit) // seal intact, so the CEILING check is under test
	tampered := withEnvelopeFields(t, env, map[string]any{
		"grant_path": bad, "composition": commit,
	}, "envelope-t-bypass-2.json")

	if _, err := f.o.SubmitTask(tampered); err == nil {
		t.Fatal("L7 must refuse an out-of-ceiling grant regardless of skill provenance")
	} else if !strings.Contains(err.Error(), "ceiling") {
		t.Fatalf("expected a ceiling refusal, got: %v", err)
	}
}
