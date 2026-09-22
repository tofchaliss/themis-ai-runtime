package orchestration

// Skill admission (openspec/changes/l9-l7-skill-admission-identity):
// the twin suite. Every negative names the gate that refused it; the
// positive path is asserted first (the C17 lesson: "refuse
// everything" must fail some test here).

import (
	"encoding/json"
	"go/ast"
	"go/importer"
	"go/parser"
	"go/token"
	"go/types"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/tofchaliss/themis/deployment"
	"github.com/tofchaliss/themis/runtime/model"
	"github.com/tofchaliss/themis/skills"
	"github.com/tofchaliss/themis/state"
)

// D-SA-5 coherence matrix at LoadEnvelope: the load-bearing `skill`
// field and the attribution-only origin must agree, and a claim
// requires the commitment it claims.
func TestSkillFieldCoherence(t *testing.T) {
	f := setupVerif(t, happyScript(), &scriptedEvaluator{})
	base := f.verifEnvelope(t, "coh")
	commit := genuineCommitment(t, base)

	cases := []struct {
		name   string
		fields map[string]any
		want   string // "" = must load
	}{
		{"ordinary envelope", map[string]any{}, ""},
		{"commitment without claim is Claim 1 only", map[string]any{"composition": commit}, ""},
		{"claim + commitment + matching origin", map[string]any{
			"skill": "investigate-cve@1", "composition": commit,
			"origin": map[string]string{"skill": "investigate-cve@1", "skill_catalog": strings.Repeat("a", 64)}}, ""},
		{"claim + commitment, no origin", map[string]any{
			"skill": "investigate-cve@1", "composition": commit}, ""},
		{"claim without commitment", map[string]any{"skill": "investigate-cve@1"},
			"must carry the composition commitment that claim refers to"},
		{"origin attribution without the selector", map[string]any{
			"composition": commit, "origin": map[string]string{"skill": "investigate-cve@1"}},
			"load-bearing skill field"},
		{"origin skill_ key without the selector", map[string]any{
			"composition": commit, "origin": map[string]string{"skill_catalog": strings.Repeat("a", 64)}},
			"load-bearing skill field"},
		{"selector and attribution disagree", map[string]any{
			"skill": "investigate-cve@1", "composition": commit,
			"origin": map[string]string{"skill": "remediate-dependency@1"}},
			"disagree"},
		{"floating reference refused", map[string]any{
			"skill": "investigate-cve@latest", "composition": commit},
			"exact name@version"},
		{"name-only reference refused", map[string]any{
			"skill": "investigate-cve", "composition": commit},
			"exact name@version"},
	}
	for i, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			p := withEnvelopeFields(t, base, c.fields, "envelope-coh-"+string(rune('a'+i))+".json")
			env, err := LoadEnvelope(p)
			if c.want == "" {
				if err != nil {
					t.Fatalf("must load: %v", err)
				}
				if want, ok := c.fields["skill"]; ok && env.Skill != want {
					t.Fatalf("skill field not carried: %q", env.Skill)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), c.want) {
				t.Fatalf("refused for the wrong reason (want %q): %v", c.want, err)
			}
		})
	}
}

// D-SA-3: under an anchored deployment, a skill-scope procedure enters
// only through a corresponding Skill composition. The unattributed
// procedure (finding F-SA-1) is refused at assembly with its own
// message; the same envelope under the explicitly unanchored role
// keeps today's behaviour (D-SA-7); and an ATTRIBUTED envelope is
// never refused with the D-SA-3 message — whatever else admission
// says about it belongs to the correspondence gates.
func TestAnchoredUnattributedProcedureRefused(t *testing.T) {
	f := setupVerif(t, happyScript(), &scriptedEvaluator{})
	proc := writeJSON(t, f.envDir, "procedure.md", "# Procedure\nread, verify the report, then declare done.\n")
	procSHA := hashBytes([]byte(readFile(t, proc)))
	fileHash := func(p string) string {
		h, err := deployment.HashFile(p)
		if err != nil {
			t.Fatal(err)
		}
		return h
	}
	envHash := func(n string) string { return fileHash(filepath.Join(f.envDir, n)) }
	ap, sha, reg, apCeiling := anchorWorld(t, func(m map[string]any) {
		m["execution_ceiling"] = envHash("eceiling.json")
		m["workflows"] = []any{map[string]any{
			"workflow":         envHash("workflow.json"),
			"workflow_ceiling": envHash("wceiling.json"),
			"context_contract": envHash("context-contract.json"),
		}}
	}, filepath.Join(f.envDir, "eceiling.json"))
	cfg := anchoredConfig(t, t.TempDir(), ap, sha, reg, apCeiling)
	cfg.SkillCatalogPath = mustAbs(t, filepath.Join(repoRoot, "policies/skills/catalog.json"))
	cfg.Verifier = &scriptedEvaluator{}
	o, _, err := Open(cfg)
	if err != nil {
		t.Fatal(err)
	}

	base := f.verifEnvelope(t, "unattr")
	unattributed := withEnvelopeFields(t, base, map[string]any{
		"skill_procedure_path": proc, "skill_procedure_sha256": procSHA,
	}, "envelope-unattr-2.json")
	_, err = o.SubmitTask(unattributed)
	if err == nil || !strings.Contains(err.Error(), "never from an unattributed envelope") {
		t.Fatalf("D-SA-3: an unattributed procedure under an anchored deployment must refuse with its own message: %v", err)
	}

	// Twin 1: the unanchored role preserves today's behaviour — the
	// same envelope is NOT refused for that reason (D-SA-7).
	_, err = f.o.SubmitTask(unattributed)
	if err != nil && strings.Contains(err.Error(), "never from an unattributed envelope") {
		t.Fatalf("D-SA-7: the unanchored role must not apply the anchored procedure rule: %v", err)
	}

	// Twin 2: an ATTRIBUTED envelope never trips D-SA-3 — the
	// correspondence gates own everything after this point.
	c := genuineCommitment(t, unattributed)
	attributed := withEnvelopeFields(t, base, map[string]any{
		"skill_procedure_path": proc, "skill_procedure_sha256": procSHA,
		"skill": "investigate-cve@1", "composition": c,
		"origin": map[string]string{"skill": "investigate-cve@1"},
	}, "envelope-unattr-3.json")
	_, err = o.SubmitTask(attributed)
	if err != nil && strings.Contains(err.Error(), "never from an unattributed envelope") {
		t.Fatalf("D-SA-3 must not fire on an attributed envelope: %v", err)
	}
}

// L9 emits the load-bearing selector and the attribution as two
// independently derived values that agree.
func TestInstantiateEmitsSkillSelector(t *testing.T) {
	f := setup(t, happyScript(), "")
	p := instantiateP0(t, f, "sel-1", skills.Request{})
	env, err := LoadEnvelope(p)
	if err != nil {
		t.Fatal(err)
	}
	if env.Skill != "investigate-cve@1" {
		t.Fatalf("skill selector = %q", env.Skill)
	}
	if env.Origin["skill"] != env.Skill {
		t.Fatalf("attribution %q disagrees with selector %q", env.Origin["skill"], env.Skill)
	}
}

// Wall (D-SA-3 / D-SA-5): the ONLY call site of the L1 skill-source
// activation seam in the orchestration package is the one inside
// resolveTaskEIS, which SubmitTask reaches only past the anchored
// procedure gate. A second call site would be a second path for
// caller bytes into skill scope.
func TestActivateSkillSourceSingleCallSite(t *testing.T) {
	fset := token.NewFileSet()
	pkgs, err := parser.ParseDir(fset, ".", func(fi os.FileInfo) bool {
		return !strings.HasSuffix(fi.Name(), "_test.go")
	}, 0)
	if err != nil {
		t.Fatal(err)
	}
	var sites []string
	for _, pkg := range pkgs {
		for fname, file := range pkg.Files {
			var enclosing string
			ast.Inspect(file, func(n ast.Node) bool {
				if fd, ok := n.(*ast.FuncDecl); ok {
					enclosing = fd.Name.Name
				}
				if sel, ok := n.(*ast.SelectorExpr); ok && sel.Sel.Name == "ActivateSkillSource" {
					sites = append(sites, filepath.Base(fname)+":"+enclosing)
				}
				return true
			})
		}
	}
	if len(sites) != 1 || sites[0] != "orchestrator.go:resolveTaskEIS" {
		t.Fatalf("ActivateSkillSource call sites = %v, want exactly [orchestrator.go:resolveTaskEIS]", sites)
	}
}

// D-SA-9: deployment admissibility is the anchor's exact allowlist,
// checked before any catalog resolution and distinguishable from a
// correspondence failure. Absent allowlist ⇒ no skill-attributed task
// is admitted; a listed skill passes THIS gate (what the later gates
// say is theirs).
func TestAnchoredSkillAllowlist(t *testing.T) {
	f := setupVerif(t, happyScript(), &scriptedEvaluator{})
	fileHash := func(p string) string {
		h, err := deployment.HashFile(p)
		if err != nil {
			t.Fatal(err)
		}
		return h
	}
	envHash := func(n string) string { return fileHash(filepath.Join(f.envDir, n)) }
	bundle := func(m map[string]any) {
		m["execution_ceiling"] = envHash("eceiling.json")
		m["workflows"] = []any{map[string]any{
			"workflow":         envHash("workflow.json"),
			"workflow_ceiling": envHash("wceiling.json"),
			"context_contract": envHash("context-contract.json"),
		}}
	}
	base := f.verifEnvelope(t, "allow")
	attributed := withEnvelopeFields(t, base, map[string]any{
		"skill": "investigate-cve@1", "composition": genuineCommitment(t, base),
	}, "envelope-allow-2.json")

	open := func(mutate func(m map[string]any)) *Orchestrator {
		ap, sha, reg, apCeiling := anchorWorld(t, func(m map[string]any) {
			bundle(m)
			if mutate != nil {
				mutate(m)
			}
		}, filepath.Join(f.envDir, "eceiling.json"))
		cfg := anchoredConfig(t, t.TempDir(), ap, sha, reg, apCeiling)
		cfg.SkillCatalogPath = mustAbs(t, filepath.Join(repoRoot, "policies/skills/catalog.json"))
		cfg.Verifier = &scriptedEvaluator{}
		o, _, err := Open(cfg)
		if err != nil {
			t.Fatal(err)
		}
		return o
	}
	// Negative: no allowlist → refused by the allowlist gate, before
	// the catalog is consulted.
	_, err := open(nil).SubmitTask(attributed)
	if err == nil || !strings.Contains(err.Error(), "not in the anchored skill allowlist") {
		t.Fatalf("absent allowlist must refuse at the allowlist gate: %v", err)
	}
	// Negative: allowlist naming another skill → same gate.
	_, err = open(func(m map[string]any) { m["skills"] = []any{"remediate-dependency@1"} }).SubmitTask(attributed)
	if err == nil || !strings.Contains(err.Error(), "not in the anchored skill allowlist") {
		t.Fatalf("an unlisted skill must refuse at the allowlist gate: %v", err)
	}
	// Positive twin for THIS gate: listed → the allowlist message never
	// appears; the later gates decide the rest.
	_, err = open(func(m map[string]any) { m["skills"] = []any{"investigate-cve@1"} }).SubmitTask(attributed)
	if err != nil && strings.Contains(err.Error(), "not in the anchored skill allowlist") {
		t.Fatalf("a listed skill must pass the allowlist gate: %v", err)
	}
}

// anchoredSkillWorld opens an anchored deployment that pins the REAL
// governed catalog and investigate-cve@1's bundle, and returns the
// orchestrator plus a genuine L9-instantiated envelope for it. This is
// the positive twin's world; negatives mutate one thing from here.
func anchoredSkillWorld(t *testing.T, m model.Interface, mutateAnchor func(m map[string]any)) (*Orchestrator, *fixture, string) {
	t.Helper()
	f := setup(t, m, "")
	catalogPath := mustAbs(t, filepath.Join(repoRoot, "policies/skills/catalog.json"))
	cat, err := skills.LoadCatalog(catalogPath)
	if err != nil {
		t.Fatal(err)
	}
	_, man, err := cat.Resolve("investigate-cve@1")
	if err != nil {
		t.Fatal(err)
	}
	fileHash := func(p string) string {
		h, err := deployment.HashFile(p)
		if err != nil {
			t.Fatal(err)
		}
		return h
	}
	base := t.TempDir()
	artifacts, workspaces := filepath.Join(base, "artifacts"), filepath.Join(base, "workspaces")
	// The task-writable roots must exist before instantiation: L9's
	// disjointness wall resolves them physically (Open creates the
	// state root later, but L9 runs first).
	for _, d := range []string{artifacts, workspaces, filepath.Join(base, "state")} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	registry := mustAbs(t, filepath.Join(repoRoot, "policies/tools/registry-v4.json"))
	env := instantiateP0(t, f, "anch-"+strings.ToLower(t.Name()[len("Test"):]), skills.Request{Deployment: skills.Deployment{
		Model: "scripted", TurnTimeoutSec: 180, RegistryPath: registry,
		ExecCeilingPath: filepath.Join(f.envDir, "eceiling.json"),
		StateRoot:       filepath.Join(base, "state"), ArtifactDir: artifacts, WorkspaceRoot: workspaces,
	}})
	ap, sha, reg, apCeiling := anchorWorld(t, func(a map[string]any) {
		a["tool_registry"] = fileHash(registry)
		a["execution_ceiling"] = fileHash(filepath.Join(f.envDir, "eceiling.json"))
		a["workflows"] = []any{map[string]any{
			"workflow":         man.Workflow.SHA256,
			"workflow_ceiling": man.WorkflowCeiling.SHA256,
			"context_contract": man.ContextContract.SHA256,
		}}
		a["skills"] = []any{"investigate-cve@1"}
		a["skill_catalog"] = fileHash(catalogPath)
		if mutateAnchor != nil {
			mutateAnchor(a)
		}
	}, filepath.Join(f.envDir, "eceiling.json"))
	cfg := anchoredConfig(t, base, ap, sha, reg, apCeiling)
	cfg.Model = m
	cfg.SkillCatalogPath = catalogPath
	cfg.Verifier = &scriptedEvaluator{}
	o, _, err := Open(cfg)
	if err != nil {
		t.Fatal(err)
	}
	return o, f, env
}

func p0Script() model.Interface {
	return &scriptedModel{steps: []model.ExecutionResponse{
		toolCall("read_file", `{"path":"parser.go"}`),
		toolCall("declare_done", `{}`),
		toolCall("write_file", `{"path":"assessment.md","content":"# Assessment\nNot affected.\n"}`),
		toolCall("declare_done", `{}`),
	}}
}

// THE POSITIVE TWIN (Q-SA-12 / A-SA-11): a genuine L9-instantiated
// investigate-cve@1 envelope is ADMITTED and EXECUTED under an anchored
// deployment, with the correspondence evidence in the record. Before
// this test, "refuse everything" satisfied every skill-admission test
// in the repository.
func TestAnchoredSkillPositiveTwin(t *testing.T) {
	o, _, env := anchoredSkillWorld(t, p0Script(), nil)
	res, err := o.SubmitTask(env)
	if err != nil {
		t.Fatalf("the legitimate skill instance must be admitted under the anchored deployment: %v", err)
	}
	if res.Status != state.StatusCompleted {
		t.Fatalf("expected a completed walk, got %+v", res)
	}
	man, err := o.root.ReadManifest(res.TaskID)
	if err != nil {
		t.Fatal(err)
	}
	// D-SA-5: the selector admission used, beside the attribution.
	if man.GovernedHashes["skill"] != "investigate-cve@1" || man.GovernedHashes["origin:skill"] != "investigate-cve@1" {
		t.Fatalf("selector/attribution missing from the record: %v", man.GovernedHashes)
	}
	if man.GovernedHashes["deployment_anchor"] == "unanchored" || man.GovernedHashes["deployment_anchor"] == "" {
		t.Fatalf("an anchored admission must record its anchor: %q", man.GovernedHashes["deployment_anchor"])
	}
	// Claim 2 evidence (A-SA-11): the consumed catalog and manifest
	// bytes are in the durable closure — content-addressed, so their
	// ids are the hashes of the governed files themselves.
	evs, err := o.root.ReadEvents(res.TaskID)
	if err != nil {
		t.Fatal(err)
	}
	var objects map[string]string
	for _, ev := range evs {
		var body struct {
			Kind    string            `json:"kind"`
			Objects map[string]string `json:"objects"`
		}
		if json.Unmarshal(ev.Body, &body) == nil && body.Kind == "materialized-governed-artifacts" {
			objects = body.Objects
		}
	}
	catHash, _ := deployment.HashFile(filepath.Join(repoRoot, "policies/skills/catalog.json"))
	manHash, _ := deployment.HashFile(filepath.Join(repoRoot, "policies/skills/investigate-cve/skill.json"))
	if objects["skill_catalog"] != "sha256:"+catHash || objects["skill_manifest"] != "sha256:"+manHash {
		t.Fatalf("consumed Governance bytes not retained: catalog=%q manifest=%q (want %s / %s)", objects["skill_catalog"], objects["skill_manifest"], catHash, manHash)
	}
}

// resealed returns the envelope's commitment with one identity replaced
// and the seal recomputed, so the test exercises the CORRESPONDENCE
// gate rather than tripping the seal.
func resealed(t *testing.T, envPath, field, value string) map[string]string {
	t.Helper()
	env, err := LoadEnvelope(envPath)
	if err != nil {
		t.Fatal(err)
	}
	c := *env.Composition
	switch field {
	case "workflow":
		c.Workflow = value
	case "grant_template":
		c.GrantTemplate = value
	case "procedure":
		c.Procedure = value
	default:
		t.Fatalf("unknown field %s", field)
	}
	return commitmentMap(&c)
}

// Negatives for the correspondence and instantiation gates, each
// refused for ITS reason (the twin discipline). All share the positive
// world; each changes exactly one thing.
func TestAnchoredSkillNegativeTwins(t *testing.T) {
	o, _, env := anchoredSkillWorld(t, p0Script(), nil)
	dir := filepath.Dir(env)
	raw, _ := os.ReadFile(env)
	var doc map[string]any
	if err := json.Unmarshal(raw, &doc); err != nil {
		t.Fatal(err)
	}
	grantPath := doc["grant_path"].(string)
	grantRaw, _ := os.ReadFile(grantPath)
	// A grant variant: mutate the effective grant and re-commit its
	// identity so the SEAL and Claim 1 both pass — only D-SA-4 can
	// refuse it.
	withGrant := func(name string, mutate func(g map[string]any)) string {
		var g map[string]any
		if err := json.Unmarshal(grantRaw, &g); err != nil {
			t.Fatal(err)
		}
		mutate(g)
		gb, _ := json.Marshal(g)
		gp := filepath.Join(dir, name+"-grant.json")
		if err := os.WriteFile(gp, gb, 0o644); err != nil {
			t.Fatal(err)
		}
		e, err := LoadEnvelope(env)
		if err != nil {
			t.Fatal(err)
		}
		c := *e.Composition
		c.Grant = hashBytes(gb)
		return withEnvelopeFields(t, env, map[string]any{"grant_path": gp, "composition": commitmentMap(&c)}, "envelope-"+name+".json")
	}
	entries := func(g map[string]any) []map[string]any {
		var out []map[string]any
		for _, e := range g["entries"].([]any) {
			out = append(out, e.(map[string]any))
		}
		return out
	}
	cases := []struct {
		name string
		env  string
		want string
	}{
		{"wrong skill (unregistered)", withEnvelopeFields(t, env, map[string]any{
			"skill": "no-such-skill@1", "origin": map[string]string{"skill": "no-such-skill@1"}}, "envelope-neg-a.json"),
			"not in the anchored skill allowlist"},
		{"member substituted: workflow", withEnvelopeFields(t, env, map[string]any{
			"composition": resealed(t, env, "workflow", strings.Repeat("ab", 32))}, "envelope-neg-b.json"),
			"composition's workflow is not the workflow that investigate-cve@1 registers"},
		{"member substituted: grant template", withEnvelopeFields(t, env, map[string]any{
			"composition": resealed(t, env, "grant_template", strings.Repeat("cd", 32))}, "envelope-neg-c.json"),
			"composition's grant_template is not the grant_template that investigate-cve@1 registers"},
		{"quota widened", withGrant("neg-d", func(g map[string]any) {
			entries(g)[0]["max_calls"] = 9999
			g["total_max_calls"] = 9999
		}), "quotas only narrow"},
		{"tool added", withGrant("neg-e", func(g map[string]any) {
			g["entries"] = append(g["entries"].([]any), map[string]any{"tool": "verify_report", "max_calls": 1, "workspace": "@workspace"})
		}), "not in the template — the tool set is Skill-fixed"},
		{"tool removed", withGrant("neg-f", func(g map[string]any) {
			g["entries"] = g["entries"].([]any)[1:]
		}), "in the template but not the effective grant"},
		{"template_scope added to an entry", withGrant("neg-g", func(g map[string]any) {
			entries(g)[0]["template_scope"] = []any{"analysis@1"}
		}), "template_scope differs from the template"},
		{"mutating flag flipped", withGrant("neg-h", func(g map[string]any) {
			for _, e := range entries(g) {
				if e["tool"] == "read_file" {
					e["mutating"] = true
				}
			}
		}), "mutating flag differs from the template"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			_, err := o.SubmitTask(c.env)
			if err == nil || !strings.Contains(err.Error(), c.want) {
				t.Fatalf("refused for the wrong reason (want %q): %v", c.want, err)
			}
		})
	}
}

// D-SA-8: a withdrawn skill refuses new admission with the withdrawal
// message; the anchor pins the withdrawn catalog state, so this is the
// registry answering, not a stale runtime.
func TestAnchoredWithdrawnSkillRefused(t *testing.T) {
	// A private catalog root: the real bundle copied, the entry marked
	// withdrawn. The composition bytes are untouched — withdrawal is
	// state on the identity, never mutation of the artifact.
	root := t.TempDir()
	src := filepath.Join(repoRoot, "policies/skills")
	for _, name := range []string{"investigate-cve", "remediate-dependency"} {
		if err := os.MkdirAll(filepath.Join(root, name), 0o755); err != nil {
			t.Fatal(err)
		}
		ents, err := os.ReadDir(filepath.Join(src, name))
		if err != nil {
			t.Fatal(err)
		}
		for _, e := range ents {
			b, err := os.ReadFile(filepath.Join(src, name, e.Name()))
			if err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(root, name, e.Name()), b, 0o644); err != nil {
				t.Fatal(err)
			}
		}
	}
	catRaw, _ := os.ReadFile(filepath.Join(src, "catalog.json"))
	withdrawn := strings.Replace(string(catRaw), `"manifest_path": "investigate-cve/skill.json",
   "state": "active"`, `"manifest_path": "investigate-cve/skill.json",
   "state": "withdrawn"`, 1)
	if withdrawn == string(catRaw) {
		t.Fatal("fixture: could not mark investigate-cve@1 withdrawn")
	}
	catalogPath := filepath.Join(root, "catalog.json")
	if err := os.WriteFile(catalogPath, []byte(withdrawn), 0o644); err != nil {
		t.Fatal(err)
	}
	// Instantiate from the ACTIVE catalog (L9 refuses withdrawn), then
	// submit under an anchor pinning the WITHDRAWN one.
	o, _, env := anchoredSkillWorld(t, p0Script(), func(a map[string]any) {
		h, _ := deployment.HashFile(catalogPath)
		a["skill_catalog"] = h
	})
	o.cfg.SkillCatalogPath = catalogPath
	_, err := o.SubmitTask(env)
	if err == nil || !strings.Contains(err.Error(), "is withdrawn — new instantiation is refused") {
		t.Fatalf("a withdrawn skill must refuse with the withdrawal message: %v", err)
	}
}

// D-SA-6 wall: the composition seal has exactly two consumers —
// checkSeal (envelope.go) and the record (via the envelope bytes). It
// is never compared against a catalog, manifest, or anchor value: no
// non-test file in this package references `.Seal` except envelope.go.
func TestSealHasNoGovernanceConsumer(t *testing.T) {
	fset := token.NewFileSet()
	pkgs, err := parser.ParseDir(fset, ".", func(fi os.FileInfo) bool {
		return !strings.HasSuffix(fi.Name(), "_test.go")
	}, 0)
	if err != nil {
		t.Fatal(err)
	}
	// Type-checked, not textual: the L5 environment also has a Seal
	// method, and a name match would flag the wrong thing. Only a
	// selection on CompositionCommitment is the composition seal.
	for _, pkg := range pkgs {
		var files []*ast.File
		names := map[*ast.File]string{}
		for fname, file := range pkg.Files {
			files = append(files, file)
			names[file] = filepath.Base(fname)
		}
		info := &types.Info{Selections: map[*ast.SelectorExpr]*types.Selection{}}
		conf := types.Config{Importer: importer.ForCompiler(fset, "source", nil)}
		if _, err := conf.Check("orchestration", fset, files, info); err != nil {
			t.Fatalf("type-checking the package: %v", err)
		}
		for sel, selection := range info.Selections {
			if sel.Sel.Name != "Seal" {
				continue
			}
			recv := selection.Recv()
			if ptr, ok := recv.(*types.Pointer); ok {
				recv = ptr.Elem()
			}
			named, ok := recv.(*types.Named)
			if !ok || named.Obj().Name() != "CompositionCommitment" {
				continue
			}
			fname := names[fileOf(files, fset, sel)]
			if fname != "envelope.go" {
				t.Fatalf("%s references the composition seal — the seal has two consumers only (D-SA-6)", fname)
			}
		}
	}
}

func fileOf(files []*ast.File, fset *token.FileSet, n ast.Node) *ast.File {
	for _, f := range files {
		if fset.File(f.Pos()) == fset.File(n.Pos()) {
			return f
		}
	}
	return nil
}

// Register E (SA-M6): the anchored positive path against a REAL local
// model through the unmodified production loop — the first anchored
// skill execution. Skipped without an endpoint. The governed property
// asserted is admission + a typed terminal through governed edges or a
// constitution floor; COMPLETED would assert model behaviour, which is
// not a property of the system (the existing live-register discipline).
func TestLiveAnchoredSkillWalk(t *testing.T) {
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
	o, _, env := anchoredSkillWorld(t, model.NewOllamaChat(endpoint), func(a map[string]any) {
		a["models"] = []any{modelName}
	})
	// The helper's envelope named the scripted model; the live anchor
	// allowlists the real one, and task identity is bound into
	// grant/spec by L9 — so instantiate afresh through L9 with the live
	// model as the Class-4 deployment model, keeping the commitment
	// genuine. The helper's own envelope is unused here.
	_ = env
	f := setup(t, model.NewOllamaChat(endpoint), "")
	base := filepath.Dir(o.cfg.StateRoot)
	envLive := instantiateP0(t, f, "anch-live", skills.Request{
		WallDeadlineS: 300,
		Inputs: map[string]any{
			"cve-id": "CVE-2026-12345", "component": "parser",
			"focus": "Read parser.go, then call declare_done with no arguments. When declare_done is the only tool available, call it immediately.",
		},
		Deployment: skills.Deployment{
			Model: modelName, TurnTimeoutSec: 180,
			RegistryPath:    mustAbs(t, filepath.Join(repoRoot, "policies/tools/registry-v4.json")),
			ExecCeilingPath: o.cfg.ExecCeilingPath,
			StateRoot:       o.cfg.StateRoot, ArtifactDir: filepath.Join(base, "artifacts"), WorkspaceRoot: filepath.Join(base, "workspaces"),
		},
	})
	res, err := o.SubmitTask(envLive)
	if err != nil {
		t.Fatalf("live anchored skill admission failed: %v", err)
	}
	switch res.Status {
	case state.StatusCompleted, state.StatusFailed:
	default:
		t.Fatalf("live anchored walk must end at a typed terminal: %+v", res)
	}
	man, err := o.root.ReadManifest("anch-live")
	if err != nil {
		t.Fatal(err)
	}
	if man.GovernedHashes["skill"] != "investigate-cve@1" || man.GovernedHashes["deployment_anchor"] == "unanchored" {
		t.Fatalf("anchored skill admission not in the record: %v", man.GovernedHashes)
	}
	t.Logf("live anchored skill walk: model=%s status=%s verdict=%s", modelName, res.Status, res.Verdict)
}
