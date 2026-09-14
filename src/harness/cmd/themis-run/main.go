// themis-run is the production invocation surface: it opens a governed
// deployment under its admitted Deployment Anchor and submits ONE task.
//
// Shape (Q-PW-1, owner-locked 2026-09-14): a CLI invoked per task, not a
// service. Every operation is a synchronous response to this explicit
// invocation — nothing here watches, schedules, retries, or continues,
// the same discipline D-L11-14 fixed for themis-ratchet. A service would
// add a lifecycle, a listening surface and a scheduler; the constitution
// has refused all three elsewhere and nothing about production changes
// that.
//
// Authority (Q-PW-2): this binary has NONE. It is a submitter, and a
// submitter chooses a task WITHIN a deployment, never the deployment.
// Every artifact it names must BE the anchored one or L7 refuses it.
// Access control is the operating system's: whoever can execute this
// binary and read the deployment root can submit, and the deployment
// root is mode 700. Submitter origin is RECORDED — never authenticated,
// never authority — through the envelope's opaque `origin` attribution
// (D-L9-13), which L7 copies verbatim into the task's governed hashes
// and exercises zero semantics on.
//
// What crosses back (Q-PW-4): a receipt, not a judgement. Typed terminal
// status, record verdict, artifact address, deployment identity. Nothing
// interpretive; the evidence stays in the record plane where it was
// witnessed.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/tofchaliss/themis/orchestration"
	"github.com/tofchaliss/themis/runtime/model"
	"github.com/tofchaliss/themis/tools"
	"github.com/tofchaliss/themis/verification/seam"
)

func main() {
	var (
		deploy    = flag.String("deploy", "", "deployment root (absolute)")
		repo      = flag.String("governed-root", "", "root holding the governed artifact trees (absolute)")
		anchor    = flag.String("anchor", "", "deployment anchor file (absolute)")
		anchorSHA = flag.String("anchor-sha256", "", "the operator's expected anchor hash")
		registry  = flag.String("anchors-registry", "", "Governance-active anchors registry (absolute)")
		envelope  = flag.String("envelope", "", "task envelope to submit (absolute)")
		endpoint  = flag.String("model-endpoint", "http://localhost:11434", "model endpoint")
		modelReg  = flag.String("model-registry", "", "models.json, or empty when the anchor declares absent")
		gitPath   = flag.String("git", "/usr/bin/git", "absolute git binary this deployment pins")
		jsonOut   = flag.Bool("json", false, "emit the receipt as JSON")
	)
	flag.Parse()

	for name, v := range map[string]string{
		"-deploy": *deploy, "-governed-root": *repo, "-anchor": *anchor,
		"-anchor-sha256": *anchorSHA, "-anchors-registry": *registry,
		"-envelope": *envelope,
	} {
		if v == "" {
			fail("%s is required — nothing is defaulted", name)
		}
	}

	l4, err := tools.LoadRegistry(filepath.Join(*repo, "policies/tools/registry-v4.json"))
	if err != nil {
		fail("tool registry: %v", err)
	}
	ev := &seam.Evaluator{
		RegistryPath: filepath.Join(*repo, "policies/verification/contracts.json"),
		L4:           l4,
	}
	// The contract registry root must be disjoint from every
	// task-writable root, or write_file could author registrations the
	// machinery accepts. Deployment wiring MUST check this before
	// serving evaluations — so it is checked here, not assumed.
	if err := ev.CheckDisjoint(
		filepath.Join(*deploy, "state"),
		filepath.Join(*deploy, "artifacts"),
		filepath.Join(*deploy, "provider"),
		filepath.Join(*deploy, "mirror"),
	); err != nil {
		fail("registry/workspace disjointness: %v", err)
	}

	o, report, err := orchestration.Open(orchestration.Config{
		StateRoot:   filepath.Join(*deploy, "state"),
		ArtifactDir: filepath.Join(*deploy, "artifacts"),
		ProviderDir: filepath.Join(*deploy, "provider"),
		GitPath:     *gitPath,

		SafetyRoot: filepath.Join(*repo, "instructions/global/safety"),
		SystemRoot: filepath.Join(*repo, "instructions/global/system"),
		ThemisRoot: filepath.Join(*repo, "instructions/themis"),
		PolicyPath: filepath.Join(*repo, "policies/security/instruction-directive-patterns.json"),

		AnchorPath:          *anchor,
		AnchorSHA256:        *anchorSHA,
		AnchorsRegistryPath: *registry,
		ExecCeilingPath:     filepath.Join(*deploy, "execution-ceiling.json"),
		SkillCatalogPath:    filepath.Join(*repo, "policies/skills/catalog.json"),
		ModelRegistryPath:   *modelReg,

		Model:    model.NewOllamaChat(*endpoint),
		Verifier: ev,
		// Unanchored stays false. Production governs by anchor or
		// refuses to open; there is no flag here to opt out, so this
		// binary cannot fall into the test-harness caller role even by
		// mistake.
	})
	if err != nil {
		fail("open: %v", err)
	}

	// themis-run IS the submitter, so it authors the submission: the
	// operator's envelope is a request, and this stamps its own
	// observed origin onto it before submitting. SubmitTask loads the
	// path it is given and hashes what it loads, so the recorded
	// configuration is always the one that actually ran.
	stamped, err := stampOrigin(*envelope, filepath.Join(*deploy, "submissions"))
	if err != nil {
		fail("submission: %v", err)
	}
	fmt.Fprintf(os.Stderr, "themis-run: submitting %s\n", stamped)

	res, subErr := o.SubmitTask(stamped)

	rec := receipt{
		TaskID:    res.TaskID,
		Status:    string(res.Status),
		Verdict:   string(res.Verdict),
		Artifact:  res.Artifact,
		Anchor:    *anchorSHA,
		Recovered: append([]string{}, report.Recovered...),
	}
	if subErr != nil {
		rec.Refusal = subErr.Error()
	}

	if *jsonOut {
		b, _ := json.MarshalIndent(rec, "", " ")
		fmt.Println(string(b))
	} else {
		fmt.Printf("task      : %s\n", rec.TaskID)
		fmt.Printf("status    : %s\n", rec.Status)
		fmt.Printf("verdict   : %s\n", rec.Verdict)
		fmt.Printf("artifact  : %s\n", rec.Artifact)
		fmt.Printf("deployment: %s\n", short(rec.Anchor))
		if len(rec.Recovered) > 0 {
			fmt.Printf("recovered : %v\n", rec.Recovered)
		}
		if rec.Refusal != "" {
			fmt.Printf("refusal   : %s\n", rec.Refusal)
		}
	}

	// Exit code separates "the deployment refused this submission" from
	// "the task ran and reached a governed terminal". A FAILED task is
	// NOT an error here: it walked to a typed terminal through a
	// declared edge, which is the system working. Conflating the two
	// would teach operators to retry governed failures.
	if subErr != nil {
		os.Exit(1)
	}
}

type receipt struct {
	TaskID    string   `json:"task_id"`
	Status    string   `json:"status"`
	Verdict   string   `json:"verdict"`
	Artifact  string   `json:"artifact,omitempty"`
	Anchor    string   `json:"deployment_anchor"`
	Recovered []string `json:"recovered_at_open,omitempty"`
	Refusal   string   `json:"refusal,omitempty"`
}

// stampOrigin writes the envelope that will actually be submitted:
// the operator's request plus this process's observed origin.
//
// The request may NOT carry submitter_* keys itself. A requester that
// could set them would be asserting who submitted a task, and an
// assertion is exactly what this is not: the value is that the record
// carries an OBSERVATION by the submitting process. Refusing is the
// only way the distinction survives contact with a hostile request.
//
// L9's skill attribution rides the same map and is preserved untouched
// — this adds keys, it never rewrites them.
func stampOrigin(requestPath, outDir string) (string, error) {
	raw, err := os.ReadFile(requestPath)
	if err != nil {
		return "", err
	}
	var env map[string]any
	if err := json.Unmarshal(raw, &env); err != nil {
		return "", fmt.Errorf("envelope is not a JSON object: %v", err)
	}
	origin := map[string]string{}
	if existing, ok := env["origin"].(map[string]any); ok {
		for k, v := range existing {
			if strings.HasPrefix(k, "submitter_") {
				return "", fmt.Errorf("the request sets origin key %q — submitter origin is OBSERVED by the submitting process, never asserted by the request", k)
			}
			s, ok := v.(string)
			if !ok {
				return "", fmt.Errorf("origin value for %q is not a string", k)
			}
			origin[k] = s
		}
	}
	host, _ := os.Hostname()
	for k, v := range SubmitterOrigin(host) {
		origin[k] = v
	}
	env["origin"] = origin

	out, err := json.MarshalIndent(env, "", " ")
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(outDir, 0o700); err != nil {
		return "", err
	}
	taskID, _ := env["task_id"].(string)
	if taskID == "" {
		return "", fmt.Errorf("envelope carries no task_id")
	}
	p := filepath.Join(outDir, taskID+"-submitted.json")
	if err := os.WriteFile(p, out, 0o600); err != nil {
		return "", err
	}
	return p, nil
}

// SubmitterOrigin reports the invoking process's own OS context as
// opaque attribution for the envelope's `origin` map.
//
// It is an OBSERVATION by the submitting process, not an authenticated
// identity and not a claim carried in the task payload. L7 records it
// verbatim and exercises zero semantics on it — it selects nothing,
// derives nothing, authorizes nothing. Its worth is accountability:
// the durable record can say which account on which host submitted a
// task, and can never be read as saying that account had authority.
//
// Keys are bounded at 64 bytes and values at 256 by the envelope
// validator. The `skill` prefix is reserved for L9 attribution and is
// deliberately not used here.
func SubmitterOrigin(hostname string) map[string]string {
	o := map[string]string{
		"submitter_uid":  strconv.Itoa(os.Getuid()),
		"submitter_host": bound(hostname, 256),
	}
	if u, err := user.Current(); err == nil {
		o["submitter_user"] = bound(u.Username, 256)
	}
	return o
}

func bound(s string, n int) string {
	if len(s) > n {
		return s[:n]
	}
	return s
}

func short(h string) string {
	if len(h) > 12 {
		return h[:12]
	}
	return h
}

func fail(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "themis-run: "+format+"\n", args...)
	os.Exit(2)
}
