// themis-evidence-harness produces Phase D evidence for one concrete
// Deployment Instance: it opens an orchestrator under the REAL admitted
// anchor, submits one governed task, and re-establishes the deployment
// from the durable record alone.
//
// This is EVIDENCE TOOLING, not production wiring. It lives in the
// deployment root, not in the repository, and deliberately does not
// decide what a production caller looks like (runbook Step 12 remains
// open). It is the smallest thing that can exercise Open/SubmitTask
// against a real anchor.
//
// It has no authority of its own: every refusal below comes from the
// harness under test. This program only reports what happened.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/tofchaliss/themis/deployment"
	"github.com/tofchaliss/themis/orchestration"
	"github.com/tofchaliss/themis/ratchet"
	"github.com/tofchaliss/themis/runtime/model"
	"github.com/tofchaliss/themis/state"
	"github.com/tofchaliss/themis/tools"
	"github.com/tofchaliss/themis/verification/seam"
)

func main() {
	repo := flag.String("repo", "", "themis-ai-runtime checkout (absolute)")
	deploy := flag.String("deploy", "", "deployment root (absolute)")
	anchorSHA := flag.String("anchor-sha256", "", "operator's expected anchor hash")
	anchorName := flag.String("anchor-name", "rsys", "expected deployment name")
	anchorVer := flag.Int("anchor-version", 1, "expected deployment version")
	taskID := flag.String("task", "rsys-d1", "task id")
	repoName := flag.String("mirror-repo", "demo-vuln-app", "repo dir under mirror_root")
	pinnedSHA := flag.String("pinned-sha", "", "40-hex commit to provision at")
	endpoint := flag.String("endpoint", "http://localhost:11434", "model endpoint")
	modelName := flag.String("model", "qwen2.5:7b", "model name (must be in the anchored allowlist)")
	gitPath := flag.String("git", "/usr/bin/git", "absolute git binary the deployment pins")
	turnTimeout := flag.Int("turn-timeout-sec", 180, "per-turn model timeout")
	wallSec := flag.Int("wall-sec", 300, "spec wall_deadline_s (must be <= ceiling)")
	payloadFile := flag.String("payload-file", "", "task brief file; empty uses the built-in default")
	flag.Parse()

	for name, v := range map[string]string{
		"-repo": *repo, "-deploy": *deploy,
		"-anchor-sha256": *anchorSHA, "-pinned-sha": *pinnedSHA,
	} {
		if v == "" {
			fail("%s is required", name)
		}
	}

	envDir := filepath.Join(*deploy, "harness", "env")
	must(os.MkdirAll(envDir, 0o700), "create env dir")

	stateRoot := filepath.Join(*deploy, "state")
	artifactDir := filepath.Join(*deploy, "artifacts")
	providerDir := filepath.Join(*deploy, "provider")
	mirrorRoot := filepath.Join(*deploy, "mirror")
	anchorPath := filepath.Join(*repo, "policies", "deployment", "rsys.json")
	anchorsReg := filepath.Join(*repo, "policies", "deployment", "anchors.json")
	ceilingPath := filepath.Join(*deploy, "execution-ceiling.json")

	section("D1 — Open under the admitted anchor")

	l4, err := tools.LoadRegistry(filepath.Join(*repo, "policies/tools/registry-v4.json"))
	must(err, "load L4 registry")
	ev := &seam.Evaluator{
		RegistryPath: filepath.Join(*repo, "policies/verification/contracts.json"),
		L4:           l4,
	}
	// Required before serving evaluations: the contract registry root
	// must be disjoint from every task-writable root, or write_file
	// could author registrations the machinery accepts.
	must(ev.CheckDisjoint(stateRoot, artifactDir, providerDir, mirrorRoot, envDir),
		"registry/workspace disjointness")

	o, report, err := orchestration.Open(orchestration.Config{
		StateRoot: stateRoot, ArtifactDir: artifactDir,
		GitPath: *gitPath, ProviderDir: providerDir,
		SafetyRoot: filepath.Join(*repo, "instructions/global/safety"),
		SystemRoot: filepath.Join(*repo, "instructions/global/system"),
		ThemisRoot: filepath.Join(*repo, "instructions/themis"),
		PolicyPath: filepath.Join(*repo, "policies/security/instruction-directive-patterns.json"),

		AnchorPath:          anchorPath,
		AnchorSHA256:        *anchorSHA,
		AnchorsRegistryPath: anchorsReg,
		ExecCeilingPath:     ceilingPath,
		SkillCatalogPath:    filepath.Join(*repo, "policies/skills/catalog.json"),
		// ModelRegistryPath stays empty: the anchor declares "absent".

		Model:    model.NewOllamaChat(*endpoint),
		Verifier: ev,
	})
	must(err, "Open")
	fmt.Printf("  opened; startup sweep: recovered=%v terminal=%v corrupt=%v\n",
		report.Recovered, report.Terminal, report.Corrupt)

	section("D2 — Assemble a task whose bundle is fully anchored")

	skill := filepath.Join(*repo, "policies/skills/remediate-dependency")
	specPath := writeJSON(envDir, *taskID+"-spec.json", map[string]any{
		"version": 1, "task_id": *taskID,
		"repo": *repoName, "pinned_sha": *pinnedSHA,
		"limits": []map[string]any{{"dimension": "wall_deadline_s", "value": *wallSec}},
	})
	grantPath := writeJSON(envDir, *taskID+"-grant.json", map[string]any{
		"version": 1, "task_id": *taskID, "total_max_calls": 30,
		"entries": []map[string]any{
			{"tool": "read_file", "max_calls": 10, "workspace": "@workspace"},
			{"tool": "write_file", "max_calls": 4, "workspace": "@workspace", "mutating": true},
			{"tool": "verify_report", "max_calls": 6, "workspace": "@workspace"},
			{"tool": "declare_done", "max_calls": 6},
		},
	})
	payload := defaultPayload
	if *payloadFile != "" {
		b, err := os.ReadFile(*payloadFile)
		must(err, "read payload file")
		payload = string(b)
	}
	envPath := writeJSON(envDir, *taskID+"-envelope.json", map[string]any{
		"version": 1, "task_id": *taskID, "model": *modelName,
		"turn_timeout_sec":      *turnTimeout,
		"payload":               payload,
		"workflow_path":         filepath.Join(skill, "workflow.json"),
		"workflow_ceiling_path": filepath.Join(skill, "ceiling.json"),
		"context_contract_path": filepath.Join(skill, "contract.json"),
		"registry_path":         filepath.Join(*repo, "policies/tools/registry-v4.json"),
		"grant_path":            grantPath,
		"exec_ceiling_path":     ceilingPath,
		"spec_path":             specPath,
	})
	fmt.Printf("  envelope: %s\n", envPath)

	section("D3 — Walk to a typed terminal")

	res, err := o.SubmitTask(envPath)
	if err != nil {
		fmt.Printf("  SubmitTask returned: %v\n", err)
	}
	fmt.Printf("  task=%s status=%s verdict=%s artifact=%q\n",
		res.TaskID, res.Status, res.Verdict, res.Artifact)
	if res.Status == "" {
		fail("no typed terminal reached — nothing to evidence")
	}

	section("D4/D5/D6 — Re-establish the deployment from the record alone")

	root, err := state.OpenRoot(stateRoot)
	must(err, "open state root")
	events, err := root.ReadEvents(*taskID)
	must(err, "read events")

	var recorded string
	var anchorBytes []byte
	for _, e := range events {
		var body struct {
			GovernedHashes map[string]string `json:"governed_hashes"`
		}
		if json.Unmarshal(e.Body, &body) == nil && body.GovernedHashes != nil {
			if h, ok := body.GovernedHashes["deployment_anchor"]; ok {
				recorded = h
			}
		}
	}
	if recorded == "" {
		fail("D4: task record carries no deployment identity")
	}
	if recorded == "unanchored" {
		fail("D4: record says UNANCHORED — this deployment must be governed by anchor")
	}
	fmt.Printf("  D4 deployment_anchor = %s\n", recorded)
	if recorded != *anchorSHA {
		fail("D4: recorded identity %s != operator anchor %s", recorded, *anchorSHA)
	}

	for _, e := range events {
		for _, ref := range e.Refs {
			b, gerr := root.Store().GetObject(ref.ID)
			if gerr == nil && ratchet.InstanceID(b) == recorded {
				anchorBytes = b
			}
		}
	}
	if anchorBytes == nil {
		fail("D5: anchor bytes are not durable in the record")
	}
	fmt.Printf("  D5 anchor bytes recovered from the record: %d bytes\n", len(anchorBytes))

	a, err := deployment.VerifyAnchorRecord(recorded, anchorBytes, anchorsReg)
	must(err, "D6: re-establish deployment from record + registry")
	if a.Name != *anchorName || a.Deployment != *anchorVer {
		fail("D6: re-established the wrong deployment: %s@%d", a.Name, a.Deployment)
	}
	fmt.Printf("  D6 re-established %s@%d from record + registry alone\n", a.Name, a.Deployment)

	section("RESULT")
	fmt.Printf("  status   : %s\n", res.Status)
	fmt.Printf("  verdict  : %s\n", res.Verdict)
	fmt.Printf("  anchor   : %s@%d (%s)\n", a.Name, a.Deployment, short(recorded))
	fmt.Printf("  events   : %d\n", len(events))
	fmt.Println("\nD1-D6 evidence produced. A non-COMPLETED status is still")
	fmt.Println("evidence: what Phase D requires is a TYPED terminal, not success.")
}

// defaultPayload states the ORDER of work explicitly. The first Phase D
// run (rsys-d1) showed why: qwen2.5:7b called declare_done on turn 1,
// never called write_file at all, then tried three times to verify a
// report.json it had never written — each attempt failing closed with
// file-unreadable until the tool-error counter exhausted. The model's
// belief that it had done the work is not evidence that it had; the
// brief therefore spells out that the file must EXIST before it can be
// verified.
//
// This is task framing, not a control. Nothing here can make an invalid
// report pass report-valid@1 — the gate is deterministic and lives in
// L10. A clearer brief only removes an avoidable reason to fail.
const defaultPayload = `Remediate one vulnerable dependency in this workspace.

Dependency: github.com/dgrijalva/jwt-go
Advisory:   CVE-2020-26160
Fixed by:   github.com/golang-jwt/jwt/v4 at v4.5.0 (the maintained successor)

Do the work in this order. Each step depends on the previous one.

1. ANALYZE. Read go.mod and the files that import the dependency.
   Establish what is vulnerable strictly from what you read here.
   When the picture is clear, request phase completion.

2. REMEDIATE. Update go.mod to the fixed module and version using
   write_file.

3. Then WRITE report.json at the workspace root using write_file. It
   must be a JSON object with exactly three non-empty string fields:
     "finding"     - what was vulnerable, citing workspace evidence
     "remediation" - what you changed
     "evidence"    - where the change is visible
   State plain facts. Do not draw security conclusions.

4. ONLY AFTER report.json exists, call verify_report on it under the
   contract report-valid@1. Verifying a file you have not written will
   fail: the file must be on disk first.

5. When verification reports PASS, declare done.`

func section(s string) { fmt.Printf("\n=== %s ===\n", s) }

func short(h string) string {
	if len(h) > 12 {
		return h[:12]
	}
	return h
}

func writeJSON(dir, name string, v any) string {
	b, err := json.MarshalIndent(v, "", " ")
	must(err, "marshal "+name)
	p := filepath.Join(dir, name)
	must(os.WriteFile(p, b, 0o600), "write "+name)
	return p
}

func must(err error, what string) {
	if err != nil {
		fail("%s: %v", what, err)
	}
}

func fail(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "\nFAIL: "+format+"\n", args...)
	os.Exit(1)
}
