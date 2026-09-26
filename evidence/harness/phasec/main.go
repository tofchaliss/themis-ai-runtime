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
	"regexp"
	"strings"

	hctx "github.com/tofchaliss/themis-ai-runtime/src/harness/context"
	"github.com/tofchaliss/themis-ai-runtime/src/harness/instructions"
	"github.com/tofchaliss/themis-ai-runtime/src/harness/orchestration"
	"github.com/tofchaliss/themis-ai-runtime/src/harness/runtime/model"
	"github.com/tofchaliss/themis-ai-runtime/src/harness/skills"
	"github.com/tofchaliss/themis-ai-runtime/src/harness/state"
	dseam "github.com/tofchaliss/themis-ai-runtime/src/harness/subagents/delegation/seam"
	"github.com/tofchaliss/themis-ai-runtime/src/harness/tools"
	"github.com/tofchaliss/themis-ai-runtime/src/harness/verification/seam"
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

// positiveRow marks a C-twin: it must ADMIT — run returns nil on the
// expected outcome and a typed error otherwise. Without twins,
// "refuse everything" would satisfy every negative row (the C17
// lesson). Tracked by id so the existing unkeyed row literals stay
// as they are.
var positiveRows = map[string]bool{}

func positive(r row) row { positiveRows[r.id] = true; return r }

func main() {
	c := &ctx{}
	flag.StringVar(&c.repo, "repo", "", "themis-ai-runtime checkout (absolute)")
	flag.StringVar(&c.deploy, "deploy", "", "deployment root (absolute)")
	flag.StringVar(&c.anchorFile, "anchor-file", "rsys4.proposed.json", "anchor filename")
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
		if positiveRows[r.id] {
			if err == nil {
				fmt.Printf("%-4s \033[32madmitted\033[0m  %s\n", r.id, r.what)
				pass++
			} else {
				fmt.Printf("%-4s \033[31mREFUSED\033[0m   %s\n      %s\n", r.id, r.what, firstLine(err.Error()))
				fail++
			}
			continue
		}
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
		// L8: the anchor pins the delegation-template registry; the
		// seam is built over the row's copy of it.
		DelegationRegistryPath: filepath.Join(r, "policies/delegation/registry.json"),
		Model:                  &inert{},
		Verifier:               c.verifier(),
		Delegator:              c.delegator(),
	}
}

func (c *ctx) delegator() orchestration.Delegator {
	policy, err := instructions.LoadPolicy(filepath.Join(c.rowDir, "policies/security/instruction-directive-patterns.json"))
	if err != nil {
		return nil
	}
	s, err := dseam.New(filepath.Join(c.rowDir, "policies/delegation/registry.json"), policy)
	if err != nil {
		return nil
	}
	return s
}

func (c *ctx) verifier() orchestration.VerificationEvaluator {
	l4, err := tools.LoadRegistry(filepath.Join(c.rowDir, "policies/tools/registry-v5.json"))
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
		"registry_path":         filepath.Join(r, "policies/tools/registry-v5.json"),
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
		{"C17", "submitter-chosen skill composition differing from the catalog's", "composition's input_schema is not the input_schema",
			func(c *ctx) error {
				skill := filepath.Join(c.rowDir, "policies/skills/remediate-dependency")
				h := skillHashes(skill)
				if h == nil || h["workflow"] == "" {
					return fmt.Errorf("row setup: skill.json unreadable")
				}
				// D-SA-9: the allowlist gate precedes correspondence, so
				// the row's anchor must admit the skill for the
				// correspondence gate to be the one that refuses.
				p, sha, err := c.mintAnchor(func(a map[string]any) {
					a["skills"] = []string{"remediate-dependency@1"}
				})
				if err != nil {
					return fmt.Errorf("row setup: %w", err)
				}
				c.anchorOverride, c.shaOverride = p, sha
				defer func() { c.anchorOverride, c.shaOverride = "", "" }()
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
					// D-SA-5: the load-bearing selector; origin is
					// attribution and must agree with it.
					e["skill"] = "remediate-dependency@1"
					e["origin"] = map[string]any{"skill": "remediate-dependency@1"}
					e["composition"] = comp
				})
			}},
		{"C16", "tool registry swapped", "tool registry is not the anchored artifact",
			func(c *ctx) error {
				return c.openThenSubmit(func(e map[string]any) {
					p := filepath.Join(c.rowDir, "other-registry.json")
					b, _ := os.ReadFile(filepath.Join(c.rowDir, "policies/tools/registry-v5.json"))
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

		// ---- Layer 8 (delegation) rows: the anchor pins the delegation-
		// template registry; every row is an attempt to govern a
		// delegation with something the anchor did not admit. The
		// positive twin C20+ comes FIRST in evidence order (the C17
		// lesson): a genuine delegate call from remediate-dependency@2
		// under the real anchor is admitted, executed by a scripted
		// delegated model, witnessed, and reconstructs CONFIRMED.
		positive(row{"C20+", "POSITIVE: remediate-dependency@2 delegates under the real anchor (witness + CONFIRMED reconstruction)", "admitted",
			func(c *ctx) error { return c.delegatingWalk("c20-pos", nil, checkDelegated) }}),
		{"C20", "delegation-template registry rewritten after pinning", "delegation-template registry is not the anchored artifact",
			func(c *ctx) error {
				p := filepath.Join(c.rowDir, "policies/delegation/registry.json")
				b, _ := os.ReadFile(p)
				_ = os.WriteFile(p, append(b, ' '), 0o644)
				return c.open()
			}},
		{"C21", "delegation seam configured while the anchor declares absent", "declares no delegation-template registry",
			func(c *ctx) error {
				p, sha, err := c.mintAnchor(func(a map[string]any) { a["delegation_template_registry"] = "absent" })
				if err != nil {
					return fmt.Errorf("row setup: %w", err)
				}
				c.anchorOverride, c.shaOverride = p, sha
				defer func() { c.anchorOverride, c.shaOverride = "", "" }()
				return c.open()
			}},
		{"C22", "wired seam holds a registry that is not the pinned bytes", "wired delegation seam holds a registry",
			func(c *ctx) error {
				// A byte-different copy (reformatted) of the same registry:
				// the configured PATH still hashes to the pin; the seam was
				// built over the copy.
				src := filepath.Join(c.rowDir, "policies/delegation/registry.json")
				raw, _ := os.ReadFile(src)
				var reg map[string]any
				_ = json.Unmarshal(raw, &reg)
				other := filepath.Join(c.rowDir, "policies/delegation/other-registry.json")
				ob, _ := json.Marshal(reg)
				_ = os.WriteFile(other, ob, 0o600)
				policy, err := instructions.LoadPolicy(filepath.Join(c.rowDir, "policies/security/instruction-directive-patterns.json"))
				if err != nil {
					return fmt.Errorf("row setup: %w", err)
				}
				sm, err := dseam.New(other, policy)
				if err != nil {
					return fmt.Errorf("row setup: %w", err)
				}
				cfg := c.cfg()
				cfg.Delegator = sm
				_, _, err = orchestration.Open(cfg)
				return err
			}},
		{"C23", "Skill's template_scope names a template the pinned registry never registered", "does not resolve",
			func(c *ctx) error {
				// Under a Skill the scope is fixed-by-skill (D-SA-4: a
				// caller-widened scope trips Instantiates first), so the
				// unregistered entry must come from the GOVERNED side: the
				// pinned registry is one that never registered
				// dependency-triage@1, while the Skill's grant template
				// still names it. Assembly refuses at scope resolution.
				regPath := filepath.Join(c.rowDir, "policies/delegation/registry.json")
				_ = os.WriteFile(regPath, []byte("{\"version\":1,\"entries\":[]}\n"), 0o600)
				p, sha, err := c.mintAnchor(func(a map[string]any) { a["delegation_template_registry"] = fileSHA(regPath) })
				if err != nil {
					return fmt.Errorf("row setup: %w", err)
				}
				c.anchorOverride, c.shaOverride = p, sha
				defer func() { c.anchorOverride, c.shaOverride = "", "" }()
				return c.delegatingWalk("c23", nil, nil)
			}},
		{"C24", "phase exposes delegate but no seam is wired", "no L8 delegator is wired",
			func(c *ctx) error {
				// The anchor pins a registry, so Open needs the path; the
				// seam itself is absent → assembly refuses the phase.
				cfg := c.cfg()
				cfg.Delegator = nil
				_, _, err := orchestration.Open(cfg)
				if err != nil {
					return err
				}
				return c.delegatingWalkWith(cfg, "c24", nil, nil)
			}},
		positive(row{"C25", "template withdrawn under the pinned registry: Skill still assembles, delegate refuses stage B (C-L8-14 G)", "admitted",
			func(c *ctx) error {
				// Withdraw in the row's copy, re-pin the anchor to the
				// withdrawn registry (a Governance act in miniature), then
				// walk: assembly admits, the delegate call refuses
				// template-withdrawn in the audit, no witness.
				regPath := filepath.Join(c.rowDir, "policies/delegation/registry.json")
				raw, _ := os.ReadFile(regPath)
				_ = os.WriteFile(regPath, []byte(strings.Replace(string(raw), `"state": "active"`, `"state": "withdrawn"`, 1)), 0o600)
				p, sha, err := c.mintAnchor(func(a map[string]any) { a["delegation_template_registry"] = fileSHA(regPath) })
				if err != nil {
					return fmt.Errorf("row setup: %w", err)
				}
				c.anchorOverride, c.shaOverride = p, sha
				defer func() { c.anchorOverride, c.shaOverride = "", "" }()
				return c.delegatingWalk("c25", nil, checkWithdrawnRefusal)
			}}),
		positive(row{"C26", "delegate names a template outside the Skill's template_scope", "admitted",
			func(c *ctx) error {
				// Stage A: an L4 target refusal witnessed in the audit; the
				// walk continues and completes without a delegation.
				return c.delegatingWalk("c26", func(env map[string]any) {
					env["_delegate_template"] = "cve-analysis@1"
				}, checkScopeDenial)
			}}),
		positive(row{"C27", "delegate references evidence beyond the task's record", "admitted",
			func(c *ctx) error {
				return c.delegatingWalk("c27", func(env map[string]any) {
					env["_delegate_evidence"] = "999:sha256:" + strings.Repeat("0", 64)
				}, checkUnreachableRefusal)
			}}),
	}
}

// inert never gets called: every row must refuse before a model turn.
// If a row ever reaches it, that row failed to refuse and the error
// says so loudly rather than letting a walk proceed.
// delegatingWalk runs remediate-dependency@2 (L9-instantiated for this
// row's deployment copy) under the row's anchored orchestrator with a
// scripted parent model: read go.mod, delegate over that read, declare
// done, write the report, verify it, declare done. mutateEnv edits the
// L9 envelope before submission (a "_delegate_*" key steers the
// scripted delegate call instead); check inspects the record.
func (c *ctx) delegatingWalk(task string, mutateEnv func(map[string]any), check func(root *state.Root, task string, m *scriptedParent) error) error {
	return c.delegatingWalkWith(c.cfg(), task, mutateEnv, check)
}

func (c *ctx) delegatingWalkWith(cfg orchestration.Config, task string, mutateEnv func(map[string]any), check func(root *state.Root, task string, m *scriptedParent) error) error {
	m := &scriptedParent{template: "dependency-triage@1"}
	cfg.Model = m
	o, _, err := orchestration.Open(cfg)
	if err != nil {
		return fmt.Errorf("row setup: Open refused unexpectedly: %w", err)
	}
	r := c.rowDir
	_ = os.MkdirAll(filepath.Join(r, "state"), 0o755)
	envPath, err := skills.Instantiate(filepath.Join(r, "policies/skills/catalog.json"), "remediate-dependency@2", skills.Request{
		TaskID: task, Repo: c.mirrorRepo, PinnedSHA: c.pinnedSHA,
		Inputs:        map[string]any{"dependency": "vulnerable-dep", "advisory": "ADV-2026-1"},
		WallDeadlineS: 300,
		Deployment: skills.Deployment{Model: "qwen2.5:7b", TurnTimeoutSec: 60,
			RegistryPath: filepath.Join(r, "policies/tools/registry-v5.json"), ExecCeilingPath: filepath.Join(r, "execution-ceiling.json"),
			StateRoot: filepath.Join(r, "state"), ArtifactDir: filepath.Join(r, "artifacts"), WorkspaceRoot: filepath.Join(r, "provider")},
		OutDir: filepath.Join(r, "envelopes"),
	})
	if err != nil {
		return fmt.Errorf("row setup: L9 instantiation: %w", err)
	}
	if mutateEnv != nil {
		raw, _ := os.ReadFile(envPath)
		var env map[string]any
		_ = json.Unmarshal(raw, &env)
		mutateEnv(env)
		if t, ok := env["_delegate_template"].(string); ok {
			m.template = t
			delete(env, "_delegate_template")
		}
		if e, ok := env["_delegate_evidence"].(string); ok {
			m.evidence = e
			delete(env, "_delegate_evidence")
		}
		b, _ := json.MarshalIndent(env, "", " ")
		envPath = filepath.Join(r, "envelope-mutated.json")
		_ = os.WriteFile(envPath, b, 0o600)
	}
	res, err := o.SubmitTask(envPath)
	if err != nil {
		return err
	}
	if res.Status != state.StatusCompleted {
		return fmt.Errorf("walk ended %s, want COMPLETED", res.Status)
	}
	if check == nil {
		return nil
	}
	root, err := state.OpenRoot(filepath.Join(r, "state"))
	if err != nil {
		return err
	}
	rowDirForRoot[root] = filepath.Join(r, "state")
	return check(root, task, m)
}

// scriptedParent is the parent model for the delegating walk. The
// delegated call (no tools offered) answers with a fixed triage note;
// parent turns follow the @2 lattice and form the evidence reference
// from the record-ref furniture they SAW.
type scriptedParent struct {
	turn       int
	template   string
	evidence   string // "" = derive from the read_file furniture
	delegCalls int
	delegReq   *model.ExecutionRequest
}

var recordRef = regexp.MustCompile(`record-ref: ([0-9]+:sha256:[0-9a-f]{64})`)

func (p *scriptedParent) Name() string { return "scripted" }
func (p *scriptedParent) Execute(_ stdctx.Context, req model.ExecutionRequest) (*model.ExecutionResponse, error) {
	if len(req.Tools) == 0 {
		p.delegCalls++
		r := req
		p.delegReq = &r
		return &model.ExecutionResponse{Content: "Triage: go.mod pins vulnerable-dep v1; evidence-backed.", Termination: model.TerminationStop,
			Identity: model.Identity{WireModel: req.Model, Runtime: "scripted", Reported: req.Model}, Provenance: model.Provenance{Endpoint: "scripted://phase-c"}}, nil
	}
	p.turn++
	call := func(id, name, args string) *model.ExecutionResponse {
		return &model.ExecutionResponse{Termination: model.TerminationToolCalls,
			ToolCalls: []model.ToolCall{{ID: id, Name: name, Arguments: json.RawMessage(args)}}}
	}
	switch p.turn {
	case 1:
		return call("c1", "read_file", `{"path":"go.mod"}`), nil
	case 2:
		ev := p.evidence
		if ev == "" {
			for _, msg := range req.Messages {
				if msg.Role == model.RoleTool && msg.ToolCallID == "c1" {
					if mm := recordRef.FindStringSubmatch(msg.Content); mm != nil {
						ev = mm[1]
					}
				}
			}
		}
		args, _ := json.Marshal(map[string]string{"template": p.template, "evidence": ev, "brief": "Is vulnerable-dep the affected dependency?"})
		return call("c2", "delegate", string(args)), nil
	case 3:
		return call("c3", "declare_done", `{}`), nil
	case 4:
		rb, _ := json.Marshal(`{"finding": "vulnerable-dep v1 in go.mod", "remediation": "bump to v2", "evidence": "go.mod updated"}`)
		return call("c4", "write_file", `{"path":"report.json","content":`+string(rb)+`}`), nil
	case 5:
		return call("c5", "verify_report", `{"path":"report.json","contract":"report-valid@2"}`), nil
	}
	return call("c6", "declare_done", `{}`), nil
}

func delegateAudit(root *state.Root, task string) (map[string]any, bool, error) {
	evs, err := root.ReadEvents(task)
	if err != nil {
		return nil, false, err
	}
	var audit map[string]any
	witnessed := false
	for _, e := range evs {
		if e.Class == state.EvL8Delegation {
			witnessed = true
		}
		if e.Class == state.EvL4Audit {
			var b map[string]any
			_ = json.Unmarshal(e.Body, &b)
			if b["Tool"] == "delegate" {
				audit = b
			}
		}
	}
	if audit == nil {
		return nil, false, fmt.Errorf("no delegate call in the record")
	}
	return audit, witnessed, nil
}

// checkDelegated: the positive twin's record properties — authorized
// call, witness present, exactly one delegated model execution, and a
// reconstruction from the record that is CONFIRMED.
func checkDelegated(root *state.Root, task string, m *scriptedParent) error {
	rowRoot := filepath.Dir(rootDirOf(root))
	audit, witnessed, err := delegateAudit(root, task)
	if err != nil {
		return err
	}
	if audit["Decision"] != "authorized" || !witnessed || m.delegCalls != 1 {
		return fmt.Errorf("decision=%v witnessed=%v delegated calls=%d", audit["Decision"], witnessed, m.delegCalls)
	}
	l4, _ := tools.LoadRegistry(filepath.Join(rowRoot, "policies/tools/registry-v5.json"))
	cfg := dseam.ReconstructConfig{}
	if l4 != nil {
		cfg.RegistryHash = l4.Hash
		cfg.ToolTrust = func(name string) (hctx.AuthorityClass, bool) {
			for _, t := range l4.Tools {
				if t.Name == name {
					return t.Trust, true
				}
			}
			return "", false
		}
	}
	recs, err := dseam.ReconstructTask(root, task, cfg)
	if err != nil {
		return err
	}
	if len(recs) != 1 || recs[0].Verdict != dseam.VerdictConfirmed {
		return fmt.Errorf("reconstruction: %+v", recs)
	}
	return nil
}

func checkWithdrawnRefusal(root *state.Root, task string, m *scriptedParent) error {
	audit, witnessed, err := delegateAudit(root, task)
	if err != nil {
		return err
	}
	if audit["Decision"] != "error" || audit["ErrClass"] != "delegation-refused:template-withdrawn" || witnessed || m.delegCalls != 0 {
		return fmt.Errorf("want stage-B template-withdrawn with no witness: %v witnessed=%v calls=%d", audit, witnessed, m.delegCalls)
	}
	return nil
}

func checkScopeDenial(root *state.Root, task string, m *scriptedParent) error {
	audit, witnessed, err := delegateAudit(root, task)
	if err != nil {
		return err
	}
	if audit["Decision"] != "denied" || !strings.Contains(fmt.Sprint(audit["TracePredicate"]), "template-outside-grant-scope") || witnessed || m.delegCalls != 0 {
		return fmt.Errorf("want an L4 scope denial with no witness: %v witnessed=%v", audit, witnessed)
	}
	return nil
}

func checkUnreachableRefusal(root *state.Root, task string, m *scriptedParent) error {
	audit, witnessed, err := delegateAudit(root, task)
	if err != nil {
		return err
	}
	if audit["Decision"] != "error" || audit["ErrClass"] != "delegation-refused:evidence-unreachable" || witnessed || m.delegCalls != 0 {
		return fmt.Errorf("want stage-B evidence-unreachable with no witness: %v witnessed=%v", audit, witnessed)
	}
	return nil
}

// rootDirOf: the state root's directory (its parent is the row dir).
var rowDirForRoot = map[*state.Root]string{}

func rootDirOf(r *state.Root) string { return rowDirForRoot[r] }

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
