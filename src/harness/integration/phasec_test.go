// Package integration holds Phase C of the L1–L11 integration audit:
// ONE end-to-end Themis workflow across the whole chain — task
// initiation → instructions (incl. the themis root) → context →
// tools → execution → durable evidence → orchestration → L10
// verification gate → L11 comparison whose facts are WITNESSED BY
// THE WALK'S OWN COMMITTED EVENTS → evidence at the Governance door
// (the real catalog) → cold reconstruction. Test-only package: it
// exercises seams, it is not a shipping component.
package integration

import (
	stdctx "context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tofchaliss/themis-ai-runtime/src/harness/deployment"
	"github.com/tofchaliss/themis-ai-runtime/src/harness/orchestration"
	"github.com/tofchaliss/themis-ai-runtime/src/harness/ratchet"
	"github.com/tofchaliss/themis-ai-runtime/src/harness/runtime/model"
	"github.com/tofchaliss/themis-ai-runtime/src/harness/state"
	"github.com/tofchaliss/themis-ai-runtime/src/harness/tools"
	seam "github.com/tofchaliss/themis-ai-runtime/src/harness/verification/seam"
)

const repoRoot = "../../.."

func mustAbs(t *testing.T, p string) string {
	t.Helper()
	a, err := filepath.Abs(p)
	if err != nil {
		t.Fatal(err)
	}
	return a
}

type scripted struct {
	steps []model.ExecutionResponse
	i     int
}

func (s *scripted) Name() string { return "scripted" }
func (s *scripted) Execute(ctx stdctx.Context, req model.ExecutionRequest) (*model.ExecutionResponse, error) {
	if s.i >= len(s.steps) {
		return &model.ExecutionResponse{Content: "…", Termination: model.TerminationStop}, nil
	}
	r := s.steps[s.i]
	s.i++
	return &r, nil
}

func call(name, args string) model.ExecutionResponse {
	return model.ExecutionResponse{Termination: model.TerminationToolCalls,
		ToolCalls: []model.ToolCall{{ID: "c", Name: name, Arguments: json.RawMessage(args)}}}
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

// writeFixtureArtifacts materializes the governed bundle artifacts
// BEFORE Open, so the deployment anchor can pin them (G1).
func writeFixtureArtifacts(t *testing.T, envDir, mirror string) {
	t.Helper()
	wj(t, envDir, "workflow.json", `{
 "version":1,"name":"remediate-dependency","initial":"REMEDIATE",
 "declared_events":["turn-no-action","turn-provider-error","turns-exhausted","tool-error",
   "signal:phase-completion-requested",
   "verification-pass","verification-fail","verification-inconclusive",
   "verification-unavailable","verification-invalid"],
 "phases":[
  {"name":"REMEDIATE","capabilities":["read_file","write_file","verify_report","declare_done"],"max_model_turns":8,"edges":[
    {"on":"signal:phase-completion-requested",
     "gate":{"contract":"report-valid@1","outcome":"PASS"},"to":"@complete"},
    {"on":"signal:phase-completion-requested","to":"@stay","counter":3,"exhausted_to":"@fail"},
    {"on":"verification-pass","to":"@stay","counter":3,"exhausted_to":"@fail"},
    {"on":"verification-fail","to":"@stay","counter":3,"exhausted_to":"@fail"},
    {"on":"verification-inconclusive","to":"@stay","counter":3,"exhausted_to":"@fail"},
    {"on":"verification-unavailable","to":"@stay","counter":3,"exhausted_to":"@fail"},
    {"on":"verification-invalid","to":"@fail"},
    {"on":"turn-no-action","to":"@stay","counter":3,"exhausted_to":"@fail"},
    {"on":"turn-provider-error","to":"@fail"},
    {"on":"tool-error","to":"@stay","counter":3,"exhausted_to":"@fail"},
    {"on":"turns-exhausted","to":"@fail"}]}
 ]}`)
	wj(t, envDir, "wceiling.json", `{"version":1,"allowed_tools":["read_file","write_file","verify_report","declare_done"],"max_total_calls":30,"max_walk_length":200,"max_turns_per_phase":10}`)
	wj(t, envDir, "eceiling.json",
		`{"version":1,"mirror_root":"`+mirror+`","max_wall_deadline_sec":600,"max_file_bytes":1048576,"max_total_bytes":10485760,"max_file_count":500,"max_mem_bytes":1073741824,"max_cpu_time_sec":600,"max_proc_count":64}`)
	wj(t, envDir, "context-contract.json",
		`{"version":1,"workflow":"remediate-dependency","slots":[
		  {"name":"task-payload","kind":"task-brief","requirement":"required","classes":["external-untrusted"]}],
		  "sensitivity_ceiling":"public"}`)

}

// buildAnchor pins the fixture's artifacts and registers the anchor
// as a Governance act would — proposed bytes become the governed
// registry; admission is then established from it, never asserted.
func buildAnchor(t *testing.T, envDir string) (anchorPath, anchorSHA, anchorsReg string) {
	t.Helper()
	hd := func(p string) string {
		h, err := deployment.HashDir(p)
		if err != nil {
			t.Fatal(err)
		}
		return h
	}
	hf := func(p string) string {
		h, err := deployment.HashFile(p)
		if err != nil {
			t.Fatal(err)
		}
		return h
	}
	env := func(n string) string { return hf(filepath.Join(envDir, n)) }
	anchor := map[string]any{
		"version": 1, "name": "phase-c", "deployment_version": 1,
		"instruction_root_safety": hd(filepath.Join(repoRoot, "instructions/global/safety")),
		"instruction_root_system": hd(filepath.Join(repoRoot, "instructions/global/system")),
		"instruction_root_themis": hd(filepath.Join(repoRoot, "instructions/themis")),
		"instruction_policy":      hf(filepath.Join(repoRoot, "policies/security/instruction-directive-patterns.json")),
		"tool_registry":           hf(filepath.Join(repoRoot, "policies/tools/registry-v4.json")),
		"constitution": map[string]any{
			"state": state.ConstitutionHash(), "orchestration": orchestration.ConstitutionHash()},
		// The DEPLOYMENT's execution ceiling: this test IS a concrete
		// deployment instance, so it supplies its own ceiling bytes
		// (real mirror_root) and the anchor pins them.
		"execution_ceiling": env("eceiling.json"),
		"workflows": []any{map[string]any{
			"workflow":         env("workflow.json"),
			"workflow_ceiling": env("wceiling.json"),
			"context_contract": env("context-contract.json"),
		}},
		"models":                       []any{"scripted"},
		"model_registry":               "absent",
		"skill_catalog":                hf(filepath.Join(repoRoot, "policies/skills/catalog.json")),
		"contract_registry":            hf(filepath.Join(repoRoot, "policies/verification/contracts.json")),
		"criteria_registry":            hf(filepath.Join(repoRoot, "policies/ratchet/criteria.json")),
		"regression_set_registry":      hf(filepath.Join(repoRoot, "policies/ratchet/regression-sets.json")),
		"delegation_template_registry": "absent",
		"themis_contract":              "absent",
	}
	ab, _ := json.Marshal(anchor)
	dir := t.TempDir()
	anchorPath = filepath.Join(dir, "anchor.json")
	if err := os.WriteFile(anchorPath, ab, 0o644); err != nil {
		t.Fatal(err)
	}
	parsed, err := deployment.ParseAnchor(ab, "phase-c")
	if err != nil {
		t.Fatal(err)
	}
	anchorSHA = parsed.SHA256
	rb, _ := json.Marshal(map[string]any{
		"version": 1, "kind": "deployment-anchors",
		"entries": []any{map[string]any{
			"name": "phase-c", "version": 1,
			"artifact_sha256": anchorSHA, "state": "active", "steward": "owner"}},
	})
	anchorsReg = filepath.Join(dir, "anchors.json")
	if err := os.WriteFile(anchorsReg, rb, 0o644); err != nil {
		t.Fatal(err)
	}
	return anchorPath, anchorSHA, anchorsReg
}

// verifyDeployment re-establishes, from the durable record alone,
// what deployment governed a task: the CREATED event's governed
// hashes carry the anchor identity, the record's stored objects
// carry the anchor bytes, and the registry proves what that identity
// meant (owner finding 3 — the G2 principle applied to G1).
func verifyDeployment(t *testing.T, root *state.Root, taskID, anchorsReg string) {
	t.Helper()
	events, err := root.ReadEvents(taskID)
	if err != nil {
		t.Fatal(err)
	}
	var recordedHash string
	var anchorBytes []byte
	for _, ev := range events {
		var body struct {
			To             string            `json:"to"`
			GovernedHashes map[string]string `json:"governed_hashes"`
		}
		if json.Unmarshal(ev.Body, &body) == nil && body.GovernedHashes != nil {
			if h, ok := body.GovernedHashes["deployment_anchor"]; ok {
				recordedHash = h
			}
		}
		// The anchor bytes ride the record as an ordinary governed
		// artifact object; find the one that hashes to the identity.
		for _, ref := range ev.Refs {
			b, gerr := root.Store().GetObject(ref.ID)
			if gerr != nil {
				continue
			}
			if recordedHash != "" && ratchet.InstanceID(b) == recordedHash {
				anchorBytes = b
			}
		}
	}
	if recordedHash == "" {
		t.Fatal("task record carries no deployment identity")
	}
	if recordedHash == "unanchored" {
		t.Fatal("phase C must run anchored")
	}
	if anchorBytes == nil {
		// Objects may be bound before the identity is seen; re-scan.
		for _, ev := range events {
			for _, ref := range ev.Refs {
				b, gerr := root.Store().GetObject(ref.ID)
				if gerr == nil && ratchet.InstanceID(b) == recordedHash {
					anchorBytes = b
				}
			}
		}
	}
	a, verr := deployment.VerifyAnchorRecord(recordedHash, anchorBytes, anchorsReg)
	if verr != nil {
		t.Fatalf("deployment not re-establishable from the record: %v", verr)
	}
	if a.Name != "phase-c" || a.Deployment != 1 {
		t.Fatalf("re-established the wrong deployment: %s@%d", a.Name, a.Deployment)
	}
}

// chainFixture opens ONE orchestrator (with the themis root wired —
// audit D7) able to run multiple remediate walks in one state root.
func chainFixture(t *testing.T, m model.Interface) (*orchestration.Orchestrator, *state.Root, string, func(taskID string) string) {
	t.Helper()
	envDir := t.TempDir()
	stateDir := filepath.Join(t.TempDir(), "state")
	mirror, sha := mkMirror(t)

	l4, err := tools.LoadRegistry(filepath.Join(repoRoot, "policies/tools/registry-v4.json"))
	if err != nil {
		t.Fatal(err)
	}
	ev := &seam.Evaluator{
		RegistryPath: filepath.Join(repoRoot, "policies/verification/contracts.json"),
		L4:           l4,
	}
	// Phase C runs ANCHORED — the chain's closest-to-production form
	// (G1 close-review M-1: the flagship walk must not be the one
	// place the deployment anchor is absent). The anchor is built
	// over the fixture's own artifacts and registered as a Governance
	// act would register it.
	writeFixtureArtifacts(t, envDir, mirror)
	anchorPath, anchorSHA, anchorsReg := buildAnchor(t, envDir)
	o, _, err := orchestration.Open(orchestration.Config{
		StateRoot: stateDir, ArtifactDir: filepath.Join(t.TempDir(), "artifacts"),
		GitPath: gitBin(t), ProviderDir: t.TempDir(),
		SafetyRoot: filepath.Join(repoRoot, "instructions/global/safety"),
		SystemRoot: filepath.Join(repoRoot, "instructions/global/system"),
		ThemisRoot: filepath.Join(repoRoot, "instructions/themis"),
		PolicyPath: filepath.Join(repoRoot, "policies/security/instruction-directive-patterns.json"),
		Model:      m,
		Verifier:   ev,
		AnchorPath: anchorPath, AnchorSHA256: anchorSHA, AnchorsRegistryPath: anchorsReg,
		ExecCeilingPath:  filepath.Join(envDir, "eceiling.json"),
		SkillCatalogPath: filepath.Join(repoRoot, "policies/skills/catalog.json"),
	})
	if err != nil {
		t.Fatal(err)
	}
	if derr := ev.CheckDisjoint(stateDir, mirror, envDir); derr != nil {
		t.Fatal(derr)
	}

	js := func(s string) string { b, _ := json.Marshal(s); return string(b) }
	mkEnvelope := func(taskID string) string {
		wj(t, envDir, taskID+"-spec.json",
			`{"version":1,"task_id":"`+taskID+`","repo":"demo","pinned_sha":"`+sha+`","limits":[{"dimension":"wall_deadline_s","value":120}]}`)
		wj(t, envDir, taskID+"-grant.json",
			`{"version":1,"task_id":"`+taskID+`","total_max_calls":30,"entries":[
			  {"tool":"read_file","max_calls":8,"workspace":"@workspace"},
			  {"tool":"write_file","max_calls":4,"workspace":"@workspace","mutating":true},
			  {"tool":"verify_report","max_calls":6,"workspace":"@workspace"},
			  {"tool":"declare_done","max_calls":6}]}`)
		return wj(t, envDir, taskID+"-envelope.json", `{
		 "version":1,"task_id":"`+taskID+`","model":"scripted","turn_timeout_sec":180,
		 "payload":"Remediate the vulnerable dependency, write report.json, verify it, then declare done.",
		 "workflow_path":`+js(filepath.Join(envDir, "workflow.json"))+`,
		 "workflow_ceiling_path":`+js(filepath.Join(envDir, "wceiling.json"))+`,
		 "registry_path":`+js(mustAbs(t, filepath.Join(repoRoot, "policies/tools/registry-v4.json")))+`,
		 "grant_path":`+js(filepath.Join(envDir, taskID+"-grant.json"))+`,
		 "exec_ceiling_path":`+js(filepath.Join(envDir, taskID+"-eceiling.json"))+`,
		 "spec_path":`+js(filepath.Join(envDir, taskID+"-spec.json"))+`,
		 "context_contract_path":`+js(filepath.Join(envDir, "context-contract.json"))+`}`)
	}
	// per-task exec ceilings (same content, distinct files so nothing
	// couples the two walks)
	mkEnvelopeFull := func(taskID string) string {
		wj(t, envDir, taskID+"-eceiling.json",
			`{"version":1,"mirror_root":"`+mirror+`","max_wall_deadline_sec":600,"max_file_bytes":1048576,"max_total_bytes":10485760,"max_file_count":500,"max_mem_bytes":1073741824,"max_cpu_time_sec":600,"max_proc_count":64}`)
		return mkEnvelope(taskID)
	}

	sroot, err := state.OpenRoot(stateDir)
	if err != nil {
		t.Fatal(err)
	}
	return o, sroot, stateDir, mkEnvelopeFull
}

// reportWithScore is the walk's report content: canonReport's three
// required strings plus a numeric score the L11 comparison consumes.
func reportWithScore(score float64) string {
	return fmt.Sprintf(`{"finding": "vulnerable-dep v1 in go.mod", "remediation": "bumped to v2", "evidence": "go.mod updated", "score": %g}`, score)
}

func scriptFor(report string) *scripted {
	rb, _ := json.Marshal(report)
	return &scripted{steps: []model.ExecutionResponse{
		call("write_file", `{"path":"report.json","content":`+string(rb)+`}`),
		call("verify_report", `{"path":"report.json","contract":"report-valid@1"}`),
		call("declare_done", `{}`),
	}}
}

// verifyReportFact locates the walk's committed l4-audit event for
// the verify_report call and returns the raw report bytes as a
// witnessed L11 fact — the fact IS the walk's own evidence, and its
// witness IS the walk's own event (D-G2-1 exercised end-to-end).
func verifyReportFact(t *testing.T, root *state.Root, taskID, selector string) ratchet.EvidenceRef {
	t.Helper()
	events, err := root.ReadEvents(taskID)
	if err != nil {
		t.Fatal(err)
	}
	for _, ev := range events {
		if ev.Class != state.EvL4Audit {
			continue
		}
		var body struct {
			Tool string `json:"Tool"`
		}
		if json.Unmarshal(ev.Body, &body) != nil || body.Tool != "verify_report" {
			continue
		}
		if len(ev.Refs) == 0 {
			continue
		}
		objID := ev.Refs[0].ID
		bytes, err := root.Store().GetObject(objID)
		if err != nil {
			t.Fatal(err)
		}
		return ratchet.EvidenceRef{
			Selector: selector, Source: "l6_execution_record",
			Ref: objID, SHA256: ratchet.InstanceID(bytes), Value: bytes,
			TaskID: taskID, EventSeq: ev.Seq,
		}
	}
	t.Fatalf("no verify_report audit event in %s", taskID)
	return ratchet.EvidenceRef{}
}

func phaseCCriterion(t *testing.T) (*ratchet.Criterion, []byte) {
	t.Helper()
	raw, err := json.Marshal(map[string]any{
		"version": 1, "name": "walk-report-score-delta", "criterion_version": 1,
		"families": []any{"skill-revision"},
		"candidate_selectors": []any{map[string]any{
			"name": "candidate_report", "source": "l6_execution_record",
			"params": map[string]any{"witness_tool": "verify_report"}}},
		"baseline_selectors": []any{map[string]any{
			"name": "baseline_report", "source": "l6_execution_record",
			"params": map[string]any{"witness_tool": "verify_report"}}},
		"comparator": map[string]any{"name": "numeric-score-delta", "version": 1},
		"config": map[string]any{"fields": []any{map[string]any{
			"delta": "score_delta", "candidate": "candidate_report", "baseline": "baseline_report"}}},
		"delta_shape": []any{map[string]any{"name": "score_delta", "type": "number"}},
		"ordering": map[string]any{"kind": "per-metric", "fields": []any{map[string]any{
			"name": "score_delta", "direction": "maximize",
			"equal_tolerance": 0.0, "non_regression_min": -0.05}}},
		"baseline_constraints": []any{"requires-current-active"},
		"provenance":           []any{"evidence_records", "run_identities", "admission_observation"},
	})
	if err != nil {
		t.Fatal(err)
	}
	c, err := ratchet.ParseCriterion(raw, "phase-c")
	if err != nil {
		t.Fatal(err)
	}
	return c, raw
}

// TestPhaseCEndToEndChain — the audit's Phase C register.
func TestPhaseCEndToEndChain(t *testing.T) {
	// ---- Two governed walks through the FULL chain (baseline, then
	// candidate) in one state root, gated on the real registered
	// report-valid@1 contract, themis instruction root wired.
	baseModel := scriptFor(reportWithScore(0.82))
	o, sroot, _, mkEnv := chainFixture(t, baseModel)
	res, err := o.SubmitTask(mkEnv("remediate-base"))
	if err != nil || res.Status != state.StatusCompleted {
		t.Fatalf("baseline walk: %v %v", res, err)
	}
	// swap the scripted model for the candidate walk
	*baseModel = *scriptFor(reportWithScore(0.91))
	res, err = o.SubmitTask(mkEnv("remediate-cand"))
	if err != nil || res.Status != state.StatusCompleted {
		t.Fatalf("candidate walk: %v %v", res, err)
	}

	// ---- L11 facts = the walks' OWN evidence, witnessed by the
	// walks' OWN committed events.
	candFact := verifyReportFact(t, sroot, "remediate-cand", "candidate_report")
	baseFact := verifyReportFact(t, sroot, "remediate-base", "baseline_report")

	// ---- Grounding verifies the witnesses against the real event
	// plane (D-G2-1, live against genuine walk history).
	criterion, criterionRaw := phaseCCriterion(t)
	if _, r := ratchet.GroundFacts(sroot, "", criterion.CandidateSelectors, []ratchet.EvidenceRef{candFact}, "candidate"); r != nil {
		t.Fatalf("candidate fact refused: %+v", r)
	}
	if _, r := ratchet.GroundFacts(sroot, "", criterion.BaselineSelectors, []ratchet.EvidenceRef{baseFact}, "baseline"); r != nil {
		t.Fatalf("baseline fact refused: %+v", r)
	}

	// ---- Admission observation against the REAL Governance door:
	// the live skills catalog, byte-hash-grounded.
	catalogPath := filepath.Join(repoRoot, "policies/skills/catalog.json")
	catalogBytes, err := os.ReadFile(catalogPath)
	if err != nil {
		t.Fatal(err)
	}
	admission, err := ratchet.ObserveAdmission("l9-catalog", catalogPath, "investigate-cve", 1, "2026-09-13T00:00:00Z")
	if err != nil || admission == nil {
		t.Fatalf("door resolution failed: %+v %v", admission, err)
	}

	// ---- The comparison: candidate walk vs baseline walk under the
	// registered criterion (registry = test-fixture Governance act).
	proposedContent := []byte(`{"skill":"investigate-cve","version":2,"change":"phase-c candidate"}`)
	pkg, refusal, err := ratchet.Compare(ratchet.CompareInput{
		CriterionRef:    "walk-report-score-delta@1",
		Criterion:       criterion,
		RegistrySHA256:  ratchet.InstanceID([]byte("phase-c-registry")),
		CandidateHash:   ratchet.InstanceID(proposedContent),
		ClaimedBaseline: admission.ArtifactSHA256,
		Admission:       admission,
		CandidateFacts:  []ratchet.EvidenceRef{candFact},
		BaselineFacts:   []ratchet.EvidenceRef{baseFact},
	})
	if err != nil || refusal != nil {
		t.Fatalf("comparison failed: %+v %v", refusal, err)
	}
	if pkg.Delta["score_delta"] < 0.089 || pkg.Delta["score_delta"] > 0.091 {
		t.Fatalf("delta wrong: %v", pkg.Delta)
	}
	// Run identities derive from the two walks — never caller text.
	want := "task:remediate-base,task:remediate-cand"
	if strings.Join(pkg.RunIdentities, ",") != want {
		t.Fatalf("run identities %v, want %s", pkg.RunIdentities, want)
	}
	pkgID, pkgBytes, err := ratchet.StoreInstance(sroot.Store(), pkg)
	if err != nil {
		t.Fatal(err)
	}

	// ---- Derivations for the door human.
	perField, err := ratchet.DerivePerField(criterion, pkg.Delta)
	if err != nil || perField[0].Relation != ratchet.RelBetter {
		t.Fatalf("derivation: %+v %v", perField, err)
	}

	// ---- Cold reconstruction: package bytes + criterion bytes +
	// the real door bytes + the same record plane. CONFIRMED means
	// the whole chain — walk events included — re-derives.
	stored, err := sroot.Store().GetObject(pkgID)
	if err != nil {
		t.Fatal(err)
	}
	if string(stored) != string(pkgBytes) {
		t.Fatal("stored bytes differ")
	}
	rec, err := ratchet.Reconstruct(stored, ratchet.ReconstructInputs{
		CriterionBytes:    criterionRaw,
		DoorRegistryBytes: catalogBytes,
		Root:              sroot,
	})
	if err != nil {
		t.Fatal(err)
	}
	if rec.Result != ratchet.ReconConfirmed {
		t.Fatalf("phase-c reconstruction: %+v", rec)
	}

	// ---- The Governance door is byte-identical after the entire
	// chain: evidence was delivered TO it; nothing acted THROUGH it.
	after, err := os.ReadFile(catalogPath)
	if err != nil {
		t.Fatal(err)
	}
	if ratchet.InstanceID(after) != ratchet.InstanceID(catalogBytes) {
		t.Fatal("the governance door changed during the chain")
	}

	// ---- Negative arc: a model-turn object cannot serve as a fact
	// even inside this genuine history (the laundering path stays
	// closed under real events).
	events, err := sroot.ReadEvents("remediate-cand")
	if err != nil {
		t.Fatal(err)
	}
	for _, ev := range events {
		if ev.Class == state.EvModelTurn && len(ev.Refs) > 0 {
			objID := ev.Refs[0].ID
			bytes, gerr := sroot.Store().GetObject(objID)
			if gerr != nil {
				continue
			}
			forged := ratchet.EvidenceRef{
				Selector: "candidate_report", Source: "l6_execution_record",
				Ref: objID, SHA256: ratchet.InstanceID(bytes), Value: bytes,
				TaskID: "remediate-cand", EventSeq: ev.Seq,
			}
			if _, r := ratchet.GroundFacts(sroot, "", criterion.CandidateSelectors, []ratchet.EvidenceRef{forged}, "candidate"); r == nil {
				t.Fatal("model-turn object grounded as an established fact")
			}
			break
		}
	}
}
