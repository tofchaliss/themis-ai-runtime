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
	"crypto/sha256"
	"encoding/hex"
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
	anchorOverride        string // row-minted anchor, when set
	shaOverride           string
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
		if r.want == "" {
			fmt.Printf("%-4s \033[36mn/a\033[0m       %s\n", r.id, r.what)
			skip++
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
		AnchorPath:          c.anchorPath(),
		AnchorSHA256:        c.anchorExpected(),
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

func (c *ctx) anchorPath() string {
	if c.anchorOverride != "" {
		return c.anchorOverride
	}
	return filepath.Join(c.rowDir, "policies/deployment", c.anchorFile)
}

func (c *ctx) anchorExpected() string {
	if c.shaOverride != "" {
		return c.shaOverride
	}
	return c.anchorSHA
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

// mintAnchor rewrites this row's anchor copy through `mutate`, registers
// the result as ACTIVE in the row's own anchors.json, and returns its
// path and hash. Scratch only — the deployment's anchor and registry are
// never touched.
func (c *ctx) mintAnchor(mutate func(map[string]any)) (string, string, error) {
	src := filepath.Join(c.rowDir, "policies/deployment", c.anchorFile)
	raw, err := os.ReadFile(src)
	if err != nil {
		return "", "", err
	}
	var a map[string]any
	if err := json.Unmarshal(raw, &a); err != nil {
		return "", "", err
	}
	mutate(a)
	out, err := json.MarshalIndent(a, "", " ")
	if err != nil {
		return "", "", err
	}
	out = append(out, '\n')
	p := filepath.Join(c.rowDir, "policies/deployment", "minted.json")
	if err := os.WriteFile(p, out, 0o600); err != nil {
		return "", "", err
	}
	sum := sha256.Sum256(out)
	sha := hex.EncodeToString(sum[:])
	name, _ := a["name"].(string)
	ver := 1
	if f, ok := a["deployment_version"].(float64); ok {
		ver = int(f)
	}
	reg := map[string]any{"version": 1, "kind": "deployment-anchors",
		"entries": []map[string]any{{"name": name, "version": ver,
			"artifact_sha256": sha, "state": "active", "steward": "phase-c"}}}
	rb, _ := json.MarshalIndent(reg, "", " ")
	if err := os.WriteFile(filepath.Join(c.rowDir, "policies/deployment/anchors.json"), rb, 0o600); err != nil {
		return "", "", err
	}
	return p, sha, nil
}

// openTwice opens once (which persists the observed registry state under
// the state root), mutates the registry, then opens again. The
// append-only wall spans restarts, so the SECOND open is the one that
// must refuse.
func (c *ctx) openTwice(mutateRegistry func(reg map[string]any)) error {
	if _, _, err := orchestration.Open(c.cfg()); err != nil {
		return fmt.Errorf("row setup: first Open refused: %w", err)
	}
	regPath := filepath.Join(c.rowDir, "policies/deployment/anchors.json")
	raw, err := os.ReadFile(regPath)
	if err != nil {
		return err
	}
	var reg map[string]any
	if err := json.Unmarshal(raw, &reg); err != nil {
		return err
	}
	mutateRegistry(reg)
	out, _ := json.MarshalIndent(reg, "", " ")
	if err := os.WriteFile(regPath, out, 0o600); err != nil {
		return err
	}
	_, _, err = orchestration.Open(c.cfg())
	return err
}

// sealComposition reproduces L7's composition seal (envelope.go
// canonical()): a versioned, ordered, length-prefixed field list. It is
// replicated rather than imported because the function is unexported —
// if L7's form changes, C17 will refuse for the wrong reason and the
// matrix will say so rather than silently pass.
func sealComposition(f map[string]string) string {
	order := []string{"workflow", "workflow_ceiling", "context_contract",
		"grant_template", "spec_template", "input_schema", "procedure", "grant", "spec"}
	var b strings.Builder
	b.WriteString("themis-skill-composition-v1")
	for _, k := range order {
		fmt.Fprintf(&b, "|%d:%s=%s", len(f[k]), k, f[k])
	}
	sum := sha256.Sum256([]byte(b.String()))
	return hex.EncodeToString(sum[:])
}

func fileSHA(p string) string {
	b, err := os.ReadFile(p)
	if err != nil {
		return ""
	}
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}

func skillHashes(skillDir string) map[string]string {
	raw, err := os.ReadFile(filepath.Join(skillDir, "skill.json"))
	if err != nil {
		return nil
	}
	var m map[string]any
	if json.Unmarshal(raw, &m) != nil {
		return nil
	}
	get := func(k string) string {
		if e, ok := m[k].(map[string]any); ok {
			if s, ok := e["sha256"].(string); ok {
				return s
			}
		}
		return ""
	}
	return map[string]string{
		"workflow": get("workflow"), "workflow_ceiling": get("workflow_ceiling"),
		"context_contract": get("context_contract"), "grant_template": get("grant_template"),
		"spec_template": get("spec_template"), "input_schema": get("input_schema"),
		"procedure": get("procedure"),
	}
}

func matrix() []row {
	return []row{
		{"C1", "anchor bytes != operator-supplied hash", "do not match the operator's expected hash",
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
		{"C4", "registration rebound to different bytes between Opens", "deployment identity is immutable",
			func(c *ctx) error {
				return c.openTwice(func(reg map[string]any) {
					// Rebind exactly the entry in force. Rebinding every
					// entry named rsys gave both versions the same hash
					// and tripped the duplicate-artifact check instead —
					// a correct refusal, for a different reason.
					for _, e := range reg["entries"].([]any) {
						m := e.(map[string]any)
						if m["artifact_sha256"] == c.anchorSHA {
							m["artifact_sha256"] = strings.Repeat("ab", 32)
						}
					}
				})
			}},
		{"C5", "registration deleted between Opens", "disappeared",
			func(c *ctx) error {
				return c.openTwice(func(reg map[string]any) {
					var kept []any
					for _, e := range reg["entries"].([]any) {
						if e.(map[string]any)["artifact_sha256"] != c.anchorSHA {
							kept = append(kept, e)
						}
					}
					reg["entries"] = kept
				})
			}},
		{"C9", "anchor pins a constitution this binary does not have", "constitution is not the anchored one",
			func(c *ctx) error {
				p, sha, err := c.mintAnchor(func(a map[string]any) {
					a["constitution"] = map[string]any{
						"state":         strings.Repeat("cd", 32),
						"orchestration": strings.Repeat("ef", 32)}
				})
				if err != nil {
					return fmt.Errorf("row setup: %w", err)
				}
				cfg := c.cfg()
				cfg.AnchorPath, cfg.AnchorSHA256 = p, sha
				_, _, err = orchestration.Open(cfg)
				return err
			}},
		{"C15", "anchored workflow paired with ANOTHER anchored bundle's ceiling", "not the artifact this anchored workflow bundles",
			func(c *ctx) error {
				// A second bundle is required for this row to mean
				// anything. Its ceiling is a byte-variant of the first
				// bundle's — semantically identical, so the workflow
				// loader accepts it and the ANCHOR check is what speaks.
				rem := filepath.Join(c.rowDir, "policies/skills/remediate-dependency")
				variant := filepath.Join(c.rowDir, "variant-ceiling.json")
				b, err := os.ReadFile(filepath.Join(rem, "ceiling.json"))
				if err != nil {
					return err
				}
				if err := os.WriteFile(variant, append(b, ' '), 0o600); err != nil {
					return err
				}
				inv := filepath.Join(c.rowDir, "policies/skills/investigate-cve")
				p, sha, err := c.mintAnchor(func(a map[string]any) {
					a["workflows"] = []map[string]any{
						{"workflow": fileSHA(filepath.Join(rem, "workflow.json")),
							"workflow_ceiling": fileSHA(filepath.Join(rem, "ceiling.json")),
							"context_contract": fileSHA(filepath.Join(rem, "contract.json"))},
						{"workflow": fileSHA(filepath.Join(inv, "workflow.json")),
							"workflow_ceiling": fileSHA(variant),
							"context_contract": fileSHA(filepath.Join(inv, "contract.json"))},
					}
				})
				if err != nil {
					return fmt.Errorf("row setup: %w", err)
				}
				c.anchorOverride, c.shaOverride = p, sha
				defer func() { c.anchorOverride, c.shaOverride = "", "" }()
				return c.openThenSubmit(func(e map[string]any) {
					e["workflow_ceiling_path"] = variant
				})
			}},
		{"C17", "submitter-chosen skill composition differing from the catalog's", "never its constituent hashes",
			func(c *ctx) error {
				skill := filepath.Join(c.rowDir, "policies/skills/remediate-dependency")
				h := skillHashes(skill)
				if h == nil || h["workflow"] == "" {
					return fmt.Errorf("row setup: skill.json unreadable")
				}
				return c.openThenSubmit(func(e map[string]any) {
					h["grant"] = fileSHA(e["grant_path"].(string))
					h["spec"] = fileSHA(e["spec_path"].(string))
					// Perturb one constituent L7 does NOT materialize,
					// then reseal: internally intact, so it reaches the
					// catalog comparison rather than failing seal
					// integrity first.
					h["input_schema"] = strings.Repeat("12", 32)
					comp := map[string]any{
						"workflow_sha256": h["workflow"], "workflow_ceiling_sha256": h["workflow_ceiling"],
						"context_contract_sha256": h["context_contract"], "grant_template_sha256": h["grant_template"],
						"spec_template_sha256": h["spec_template"], "input_schema_sha256": h["input_schema"],
						"grant_sha256": h["grant"], "spec_sha256": h["spec"],
						"composition_sha256": sealComposition(h),
					}
					// procedure_sha256 may only appear alongside
					// skill_procedure_path. Omitting both is legal and the
					// seal still matches: L7's sealed field list carries an
					// empty procedure identity in that case.
					h["procedure"] = ""
					comp["composition_sha256"] = sealComposition(h)
					e["origin"] = map[string]any{"skill": "remediate-dependency@1"}
					e["composition"] = comp
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
