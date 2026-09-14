// themis-phase-c runs the anchored negative-space matrix against the
// REAL admitted anchor: every row is an attempt to govern execution
// with something the anchor did not admit, and every one must REFUSE
// for the RIGHT reason. A refusal with the wrong reason is a finding,
// not a pass.
//
// ISOLATION. Several rows require deliberately corrupting pinned
// artifacts — mutating an instruction root, planting a symlink,
// swapping a registry. None of that may touch the live deployment: it
// is exactly what the AGENTS.md probe-isolation invariant forbids.
//
// The anchor pins CONTENT HASHES, not paths. So this tool copies the
// real instructions/ and policies/ trees into a scratch directory and
// mutates the copies. The copies are byte-identical, therefore admit
// identically, therefore the refusals are genuine — while the live
// deployment tree and its record plane are never written to. Each row
// also gets its own state root, so no row can see another's history.
package main

import (
	stdctx "context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/tofchaliss/themis/orchestration"
	"github.com/tofchaliss/themis/runtime/model"
	"github.com/tofchaliss/themis/tools"
	"github.com/tofchaliss/themis/verification/seam"
)

type ctx struct {
	repo, deploy, scratch string
	anchorFile, anchorSHA string
	pinnedSHA, mirrorRepo string
	gitPath               string
	rowDir                string // per-row scratch
}

type row struct {
	id, what, want string
	run            func(c *ctx) error
}

func main() {
	c := &ctx{}
	flag.StringVar(&c.repo, "repo", "", "themis-ai-runtime checkout (absolute)")
	flag.StringVar(&c.deploy, "deploy", "", "deployment root (absolute)")
	flag.StringVar(&c.anchorFile, "anchor-file", "rsys2.json", "anchor filename")
	flag.StringVar(&c.anchorSHA, "anchor-sha256", "", "operator's expected anchor hash")
	flag.StringVar(&c.pinnedSHA, "pinned-sha", "", "commit to provision at")
	flag.StringVar(&c.mirrorRepo, "mirror-repo", "demo-vuln-app", "repo under mirror_root")
	flag.StringVar(&c.gitPath, "git", "/usr/bin/git", "absolute git binary")
	only := flag.String("only", "", "run a single row by id (e.g. C7)")
	flag.Parse()
	for n, v := range map[string]string{
		"-repo": c.repo, "-deploy": c.deploy,
		"-anchor-sha256": c.anchorSHA, "-pinned-sha": c.pinnedSHA,
	} {
		if v == "" {
			bail("%s is required", n)
		}
	}
	c.scratch = filepath.Join(c.deploy, "ctest")
	must(os.RemoveAll(c.scratch), "clear scratch")
	must(os.MkdirAll(c.scratch, 0o700), "create scratch")

	rows := matrix()
	pass, fail, skip := 0, 0, 0
	for _, r := range rows {
		if *only != "" && r.id != *only {
			continue
		}
		c.rowDir = filepath.Join(c.scratch, r.id)
		must(os.MkdirAll(c.rowDir, 0o700), "row dir")
		if err := c.seed(); err != nil {
			fmt.Printf("%-4s SETUP FAILED: %v\n", r.id, err)
			fail++
			continue
		}
		err := r.run(c)
		switch {
		case err == nil:
			fmt.Printf("%-4s \033[31mADMITTED\033[0m  %s\n", r.id, r.what)
			fmt.Printf("      expected refusal containing %q — NOTHING REFUSED\n", r.want)
			fail++
		case strings.Contains(err.Error(), r.want):
			fmt.Printf("%-4s \033[32mrefused\033[0m   %s\n", r.id, r.what)
			fmt.Printf("      %s\n", firstLine(err.Error()))
			pass++
		default:
			fmt.Printf("%-4s \033[33mWRONG REASON\033[0m %s\n", r.id, r.what)
			fmt.Printf("      want %q\n      got  %s\n", r.want, firstLine(err.Error()))
			fail++
		}
	}
	fmt.Printf("\n%d refused correctly, %d findings, %d skipped\n", pass, fail, skip)
	fmt.Println("A refusal for the WRONG reason is a finding, not a pass.")
	if fail > 0 {
		os.Exit(1)
	}
}

// seed copies the real pinned trees into this row's scratch so the row
// can corrupt them without touching the deployment.
func (c *ctx) seed() error {
	for _, d := range []string{"instructions", "policies"} {
		if err := copyTree(filepath.Join(c.repo, d), filepath.Join(c.rowDir, d)); err != nil {
			return err
		}
	}
	return copyFile(filepath.Join(c.deploy, "execution-ceiling.json"),
		filepath.Join(c.rowDir, "execution-ceiling.json"))
}

func (c *ctx) cfg() orchestration.Config {
	r := c.rowDir
	return orchestration.Config{
		StateRoot: filepath.Join(r, "state"), ArtifactDir: filepath.Join(r, "artifacts"),
		GitPath: c.gitPath, ProviderDir: filepath.Join(r, "provider"),
		SafetyRoot:          filepath.Join(r, "instructions/global/safety"),
		SystemRoot:          filepath.Join(r, "instructions/global/system"),
		ThemisRoot:          filepath.Join(r, "instructions/themis"),
		PolicyPath:          filepath.Join(r, "policies/security/instruction-directive-patterns.json"),
		AnchorPath:          filepath.Join(r, "policies/deployment", c.anchorFile),
		AnchorSHA256:        c.anchorSHA,
		AnchorsRegistryPath: filepath.Join(r, "policies/deployment/anchors.json"),
		ExecCeilingPath:     filepath.Join(r, "execution-ceiling.json"),
		SkillCatalogPath:    filepath.Join(r, "policies/skills/catalog.json"),
		Model:               &inert{},
		Verifier:            c.verifier(),
	}
}

func (c *ctx) verifier() orchestration.VerificationEvaluator {
	l4, err := tools.LoadRegistry(filepath.Join(c.rowDir, "policies/tools/registry-v4.json"))
	if err != nil {
		return nil
	}
	return &seam.Evaluator{
		RegistryPath: filepath.Join(c.rowDir, "policies/verification/contracts.json"), L4: l4}
}

func (c *ctx) open() error { _, _, err := orchestration.Open(c.cfg()); return err }

// openThenSubmit opens successfully, then submits an envelope the row
// has deliberately made non-conforming.
func (c *ctx) openThenSubmit(mutate func(env map[string]any)) error {
	o, _, err := orchestration.Open(c.cfg())
	if err != nil {
		return fmt.Errorf("row setup: Open refused unexpectedly: %w", err)
	}
	r := c.rowDir
	skill := filepath.Join(r, "policies/skills/remediate-dependency")
	spec := writeJSON(r, "spec.json", map[string]any{
		"version": 1, "task_id": "c-row", "repo": c.mirrorRepo, "pinned_sha": c.pinnedSHA,
		"limits": []map[string]any{{"dimension": "wall_deadline_s", "value": 120}}})
	grant := writeJSON(r, "grant.json", map[string]any{
		"version": 1, "task_id": "c-row", "total_max_calls": 10,
		"entries": []map[string]any{{"tool": "declare_done", "max_calls": 2}}})
	env := map[string]any{
		"version": 1, "task_id": "c-row", "model": "qwen2.5:7b", "turn_timeout_sec": 60,
		"payload":               "phase-c negative space",
		"workflow_path":         filepath.Join(skill, "workflow.json"),
		"workflow_ceiling_path": filepath.Join(skill, "ceiling.json"),
		"context_contract_path": filepath.Join(skill, "contract.json"),
		"registry_path":         filepath.Join(r, "policies/tools/registry-v4.json"),
		"grant_path":            grant,
		"exec_ceiling_path":     filepath.Join(r, "execution-ceiling.json"),
		"spec_path":             spec,
	}
	mutate(env)
	_, err = o.SubmitTask(writeJSON(r, "envelope.json", env))
	return err
}

func matrix() []row {
	return []row{
		{"C1", "anchor bytes != operator-supplied hash", "expected anchor hash",
			func(c *ctx) error {
				cfg := c.cfg()
				cfg.AnchorSHA256 = strings.Repeat("11", 32)
				_, _, err := orchestration.Open(cfg)
				return err
			}},
		{"C6", "instruction root mutated after registration", "not the anchored artifact",
			func(c *ctx) error {
				p := filepath.Join(c.rowDir, "instructions/global/system")
				es, _ := os.ReadDir(p)
				for _, e := range es {
					if !e.IsDir() {
						f := filepath.Join(p, e.Name())
						b, _ := os.ReadFile(f)
						_ = os.WriteFile(f, append(b, '\n'), 0o644)
						break
					}
				}
				return c.open()
			}},
		{"C7", "symlink planted in a pinned instruction root", "non-regular entry",
			func(c *ctx) error {
				_ = os.Symlink("/etc/hostname",
					filepath.Join(c.rowDir, "instructions/global/safety/evil.md"))
				return c.open()
			}},
		{"C8", "instruction policy swapped", "instruction policy is not the anchored artifact",
			func(c *ctx) error {
				p := filepath.Join(c.rowDir, "policies/security/instruction-directive-patterns.json")
				b, _ := os.ReadFile(p)
				_ = os.WriteFile(p, append(b, ' '), 0o644)
				return c.open()
			}},
		{"C11", "model registry configured while the anchor declares absent", "declares no model registry",
			func(c *ctx) error {
				p := filepath.Join(c.rowDir, "models.json")
				_ = os.WriteFile(p, []byte(`{"models":{}}`), 0o600)
				cfg := c.cfg()
				cfg.ModelRegistryPath = p
				_, _, err := orchestration.Open(cfg)
				return err
			}},
		{"C12", "envelope names a model outside the allowlist", "not in the anchored allowlist",
			func(c *ctx) error {
				return c.openThenSubmit(func(e map[string]any) { e["model"] = "gpt-4o" })
			}},
		{"C13", "envelope supplies its own execution ceiling", "never chosen per task",
			func(c *ctx) error {
				return c.openThenSubmit(func(e map[string]any) {
					p := filepath.Join(c.rowDir, "other-ceiling.json")
					b, _ := os.ReadFile(filepath.Join(c.rowDir, "execution-ceiling.json"))
					_ = os.WriteFile(p, append(b, ' '), 0o600)
					e["exec_ceiling_path"] = p
				})
			}},
		{"C14", "workflow not in the anchored set", "not in the anchored workflow set",
			func(c *ctx) error {
				return c.openThenSubmit(func(e map[string]any) {
					e["workflow_path"] = filepath.Join(c.rowDir,
						"policies/skills/investigate-cve/workflow.json")
				})
			}},
		{"C15", "anchored workflow paired with another bundle's ceiling", "anchored",
			func(c *ctx) error {
				return c.openThenSubmit(func(e map[string]any) {
					e["workflow_ceiling_path"] = filepath.Join(c.rowDir,
						"policies/skills/investigate-cve/ceiling.json")
				})
			}},
		{"C16", "tool registry swapped", "tool registry is not the anchored artifact",
			func(c *ctx) error {
				return c.openThenSubmit(func(e map[string]any) {
					p := filepath.Join(c.rowDir, "other-registry.json")
					b, _ := os.ReadFile(filepath.Join(c.rowDir, "policies/tools/registry-v4.json"))
					_ = os.WriteFile(p, append(b, ' '), 0o600)
					e["registry_path"] = p
				})
			}},
		{"C18", "Open with no anchor and no explicit Unanchored", "governs by anchor or refuses to open",
			func(c *ctx) error {
				cfg := c.cfg()
				cfg.AnchorPath, cfg.AnchorSHA256, cfg.AnchorsRegistryPath = "", "", ""
				_, _, err := orchestration.Open(cfg)
				return err
			}},
		{"C19", "Unanchored declared alongside an anchor", "caller role is ambiguous",
			func(c *ctx) error {
				cfg := c.cfg()
				cfg.Unanchored = true
				_, _, err := orchestration.Open(cfg)
				return err
			}},
	}
}

// inert never gets called: every row must refuse before a model turn.
// If a row ever reaches it, that row failed to refuse and the error
// says so loudly rather than letting a walk proceed.
type inert struct{}

func (i *inert) Name() string { return "inert" }
func (i *inert) Execute(_ stdctx.Context, _ model.ExecutionRequest) (*model.ExecutionResponse, error) {
	return nil, fmt.Errorf("inert model reached — this row did NOT refuse")
}

func writeJSON(dir, name string, v map[string]any) string {
	b, _ := json.MarshalIndent(v, "", " ")
	p := filepath.Join(dir, name)
	_ = os.WriteFile(p, b, 0o600)
	return p
}

func copyTree(src, dst string) error {
	return filepath.Walk(src, func(p string, fi os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(src, p)
		target := filepath.Join(dst, rel)
		if fi.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		if !fi.Mode().IsRegular() {
			return nil
		}
		return copyFile(p, target)
	})
}

func copyFile(src, dst string) error {
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()
	_, err = io.Copy(out, in)
	return err
}

func firstLine(s string) string {
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		return s[:i]
	}
	return s
}

func must(err error, what string) {
	if err != nil {
		bail("%s: %v", what, err)
	}
}

func bail(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "\nFATAL: "+format+"\n", args...)
	os.Exit(2)
}
