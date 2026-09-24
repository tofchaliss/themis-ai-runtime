// themis-instantiate is the operator's L9 entry point: it instantiates
// a REGISTERED Skill for one deployment and writes the ordinary
// governed envelope themis-run submits. It resolves through the
// governed catalog only, narrows within the Skill's bounds, and mints
// no authority — exactly skills.Instantiate, exposed as a command so a
// deployment host never hand-writes an envelope (the D8 step of the
// deployment test plan). Nothing here opens a deployment or runs a
// task.
package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/tofchaliss/themis/skills"
)

type kv map[string]any

func (m kv) String() string { return fmt.Sprint(map[string]any(m)) }
func (m kv) Set(s string) error {
	k, v, ok := strings.Cut(s, "=")
	if !ok || k == "" {
		return fmt.Errorf("-input wants key=value, got %q", s)
	}
	m[k] = v
	return nil
}

func main() {
	inputs := kv{}
	var (
		catalog     = flag.String("catalog", "", "governed skill catalog (absolute)")
		skill       = flag.String("skill", "", "exact name@version to instantiate")
		task        = flag.String("task", "", "task id")
		retryOf     = flag.String("retry-of", "", "prior task id this retries (optional)")
		repo        = flag.String("repo", "", "repository under the ceiling's mirror_root")
		pinned      = flag.String("pinned-sha", "", "full 40-hex commit to provision at")
		modelName   = flag.String("model", "", "governed model name (must be in the anchor allowlist)")
		turnTimeout = flag.Int("turn-timeout", 180, "turn_timeout_sec")
		wall        = flag.Int("wall-deadline", 0, "wall_deadline_s narrowing (0 = the Skill's bound)")
		registry    = flag.String("registry", "", "tool registry the deployment anchors (absolute)")
		ceiling     = flag.String("exec-ceiling", "", "the deployment's execution ceiling (absolute)")
		stateRoot   = flag.String("state", "", "deployment state root (absolute)")
		artifacts   = flag.String("artifacts", "", "deployment artifact dir (absolute)")
		workspaces  = flag.String("workspaces", "", "deployment workspace/provider root (absolute)")
		out         = flag.String("out", "", "directory to write the envelope and its effective grant/spec (absolute)")
	)
	flag.Var(inputs, "input", "key=value task input (repeatable; validated against the Skill's input schema)")
	flag.Parse()
	for n, v := range map[string]string{
		"-catalog": *catalog, "-skill": *skill, "-task": *task, "-repo": *repo, "-pinned-sha": *pinned,
		"-model": *modelName, "-registry": *registry, "-exec-ceiling": *ceiling,
		"-state": *stateRoot, "-artifacts": *artifacts, "-workspaces": *workspaces, "-out": *out,
	} {
		if v == "" {
			fail("%s is required — nothing is defaulted", n)
		}
	}
	path, err := skills.Instantiate(*catalog, *skill, skills.Request{
		TaskID: *task, RetryOf: *retryOf, Repo: *repo, PinnedSHA: *pinned, Inputs: inputs,
		WallDeadlineS: *wall,
		Deployment: skills.Deployment{
			Model: *modelName, TurnTimeoutSec: *turnTimeout, RegistryPath: *registry, ExecCeilingPath: *ceiling,
			StateRoot: *stateRoot, ArtifactDir: *artifacts, WorkspaceRoot: *workspaces,
		},
		OutDir: *out,
	})
	if err != nil {
		fail("%v", err)
	}
	fmt.Println(path)
}

func fail(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "themis-instantiate: "+format+"\n", args...)
	os.Exit(2)
}
