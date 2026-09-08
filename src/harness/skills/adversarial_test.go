package skills

// Register B (adversarial): the boundaries must survive hostile input
// and deliberate bypass attempts. Each test names the decision it
// defends and the audit finding it pins, where applicable.

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// D-L9-8 wall 4: identity is the REGISTERED composition hash. A
// byte-identical unregistered copy is data, not the skill.
func TestUnregisteredCopyIsNotTheSkill(t *testing.T) {
	b := newBundle(t)
	// A perfect copy of the bundle, in a catalog that does not list it.
	empty := write(t, t.TempDir(), "catalog.json", `{"version":1,"entries":[]}`)
	req := b.request(t)
	if _, err := Instantiate(empty, "investigate-cve@1", req); !errors.Is(err, ErrResolve) ||
		!strings.Contains(err.Error(), "not registered") {
		t.Fatalf("unregistered skill-shaped artifact must be data, not executable: %v", err)
	}
}

// D-L9-8 wall 4: a catalog claiming a composition hash that does not
// match the manifest bytes is refused — content cannot masquerade.
func TestCompositionHashMismatchRefused(t *testing.T) {
	b := newBundle(t)
	bad := fmt.Sprintf(`{"version":1,"entries":[
	 {"name":"investigate-cve","version":1,"composition_sha256":%q,"manifest_path":"bundle/skill.json","state":"active"}]}`,
		strings.Repeat("c", 64))
	p := write(t, b.catalogDir, "bad-catalog.json", bad)
	c, err := LoadCatalog(p)
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := c.Resolve("investigate-cve@1"); !errors.Is(err, ErrResolve) ||
		!strings.Contains(err.Error(), "composition hash") {
		t.Fatalf("manifest bytes must match the registered hash: %v", err)
	}
}

// D-L9-10: a withdrawn skill refuses NEW instantiation, typed.
func TestWithdrawnRefusesInstantiation(t *testing.T) {
	b := newBundle(t)
	cat := fmt.Sprintf(`{"version":1,"entries":[
	 {"name":"investigate-cve","version":1,"composition_sha256":%q,"manifest_path":"bundle/skill.json","state":"withdrawn"}]}`,
		b.composition)
	p := write(t, b.catalogDir, "withdrawn.json", cat)
	if _, err := Instantiate(p, "investigate-cve@1", b.request(t)); !errors.Is(err, ErrResolve) ||
		!strings.Contains(err.Error(), "withdrawn") {
		t.Fatalf("withdrawn skill must refuse new instantiation: %v", err)
	}
}

// D-L9-11: pinned bytes are verified BEFORE the composition becomes
// executable — tampering with any artifact refuses at resolution.
func TestTamperedArtifactRefused(t *testing.T) {
	for _, artifact := range []string{"workflow.json", "procedure.md", "grant.json", "schema.json"} {
		b := newBundle(t)
		write(t, b.dir, artifact, "tampered\n")
		_, err := Instantiate(b.catalogPath, "investigate-cve@1", b.request(t))
		if !errors.Is(err, ErrResolve) || !strings.Contains(err.Error(), "do not match the reviewed pin") {
			t.Fatalf("tampered %s must refuse: %v", artifact, err)
		}
	}
}

// D-L9-10 (D-10 in the audit): a pin pointing at a symlink must not
// read outside the reviewed bundle.
func TestPinSymlinkEscapeRefused(t *testing.T) {
	b := newBundle(t)
	outside := write(t, t.TempDir(), "elsewhere.md", fxProc)
	link := filepath.Join(b.dir, "procedure.md")
	if err := os.Remove(link); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, link); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	// Even though the target bytes hash correctly, the symlink escapes
	// the bundle and must be refused on confinement grounds.
	if _, err := Instantiate(b.catalogPath, "investigate-cve@1", b.request(t)); err == nil {
		t.Fatal("a pin resolving through a symlink out of the bundle must refuse")
	}
}

// D-L9-5: quotas are upper bounds — narrowing is legal, raising is not.
func TestQuotasNarrowOnly(t *testing.T) {
	b := newBundle(t)
	req := b.request(t)
	req.QuotaOverrides = map[string]int{"read_file": 3}
	envPath, err := Instantiate(b.catalogPath, "investigate-cve@1", req)
	if err != nil {
		t.Fatalf("narrowing must be legal: %v", err)
	}
	grant := readJSON(t, filepath.Join(filepath.Dir(envPath), "t-1-grant.json"))
	entries := grant["entries"].([]any)
	for _, e := range entries {
		entry := e.(map[string]any)
		if entry["tool"] == "read_file" && entry["max_calls"].(float64) != 3 {
			t.Fatalf("quota not narrowed: %v", entry)
		}
	}
	// Raising refuses.
	req2 := b.request(t)
	req2.QuotaOverrides = map[string]int{"read_file": 99}
	if _, err := Instantiate(b.catalogPath, "investigate-cve@1", req2); !errors.Is(err, ErrResolve) ||
		!strings.Contains(err.Error(), "may only narrow") {
		t.Fatalf("raising a quota must refuse: %v", err)
	}
	// An override naming no grant entry refuses (closed surface).
	req3 := b.request(t)
	req3.QuotaOverrides = map[string]int{"write_file": 1}
	if _, err := Instantiate(b.catalogPath, "investigate-cve@1", req3); !errors.Is(err, ErrResolve) ||
		!strings.Contains(err.Error(), "matches no grant entry") {
		t.Fatalf("override for an ungranted tool must refuse: %v", err)
	}
}

// D-L9-5/D-L9-7: the wall deadline narrows only, and max_model_turns
// is NOT on the instantiation surface at all.
func TestWallDeadlineNarrowsOnly(t *testing.T) {
	b := newBundle(t)
	req := b.request(t)
	req.WallDeadlineS = 30
	envPath, err := Instantiate(b.catalogPath, "investigate-cve@1", req)
	if err != nil {
		t.Fatalf("narrowing the deadline must be legal: %v", err)
	}
	spec := readJSON(t, filepath.Join(filepath.Dir(envPath), "t-1-spec.json"))
	limits := spec["limits"].([]any)
	got := limits[0].(map[string]any)["value"].(float64)
	if got != 30 {
		t.Fatalf("deadline not narrowed: %v", got)
	}
	req2 := b.request(t)
	req2.WallDeadlineS = 900
	if _, err := Instantiate(b.catalogPath, "investigate-cve@1", req2); !errors.Is(err, ErrResolve) ||
		!strings.Contains(err.Error(), "may only narrow") {
		t.Fatalf("raising the deadline must refuse: %v", err)
	}
}

// D-1 (audit HIGH): L9 must not emit an envelope L5 will refuse — the
// pinned SHA rule is L5's, not a looser cousin.
func TestPinnedSHAMatchesL5Rule(t *testing.T) {
	b := newBundle(t)
	for _, bad := range []string{"abc1234", "main", "", strings.Repeat("a", 39), strings.Repeat("A", 40)} {
		req := b.request(t)
		req.PinnedSHA = bad
		if _, err := Instantiate(b.catalogPath, "investigate-cve@1", req); !errors.Is(err, ErrResolve) ||
			!strings.Contains(err.Error(), "40-hex") {
			t.Fatalf("pinned_sha %q must refuse at instantiation, not downstream: %v", bad, err)
		}
	}
}

// D-2 (audit): traversal in the repo name dies here, not only at the
// confinement layer.
func TestRepoTraversalRefused(t *testing.T) {
	b := newBundle(t)
	for _, bad := range []string{"../../etc", "a/../../b", "..", ""} {
		req := b.request(t)
		req.Repo = bad
		if _, err := Instantiate(b.catalogPath, "investigate-cve@1", req); !errors.Is(err, ErrResolve) {
			t.Fatalf("repo %q must refuse: %v", bad, err)
		}
	}
}

// D-L9-8 wall 2: the catalog must be disjoint from task-writable roots.
func TestCatalogRootDisjointness(t *testing.T) {
	b := newBundle(t)
	req := b.request(t)
	// Point a task-writable root AT the catalog root: a task holding
	// write_file there could otherwise reach registration.
	req.Deployment.ArtifactDir = b.catalogDir
	if _, err := Instantiate(b.catalogPath, "investigate-cve@1", req); !errors.Is(err, ErrCatalog) ||
		!strings.Contains(err.Error(), "disjoint") {
		t.Fatalf("catalog inside a task-writable root must refuse: %v", err)
	}
	// And with no writable root supplied at all, the check cannot be
	// vacuous — nothing is defaulted.
	req2 := b.request(t)
	req2.Deployment.StateRoot = ""
	req2.Deployment.ArtifactDir = ""
	req2.Deployment.WorkspaceRoot = ""
	if _, err := Instantiate(b.catalogPath, "investigate-cve@1", req2); !errors.Is(err, ErrCatalog) {
		t.Fatalf("disjointness with no comparison root must refuse: %v", err)
	}
}

// D-L9-4: caller input NEVER reaches procedure text. The delivered
// procedure bytes are the reviewed bytes, byte-for-byte.
func TestCallerInputNeverTouchesProcedure(t *testing.T) {
	b := newBundle(t)
	req := b.request(t)
	req.Inputs = map[string]any{"cve-id": "CVE-2026-99999"}
	envPath, err := Instantiate(b.catalogPath, "investigate-cve@1", req)
	if err != nil {
		t.Fatal(err)
	}
	env := readJSON(t, envPath)
	procPath := env["skill_procedure_path"].(string)
	got, err := os.ReadFile(procPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != fxProc {
		t.Fatal("procedure bytes changed — caller input must never be templated into instruction material")
	}
	if strings.Contains(string(got), "CVE-2026-99999") {
		t.Fatal("caller input leaked into procedure text")
	}
	// The input travels in the payload instead, as external-untrusted data.
	if !strings.Contains(env["payload"].(string), "CVE-2026-99999") {
		t.Fatal("input must travel via the payload channel")
	}
}

// D-L9-7: unknown inputs refuse; required inputs are not defaulted.
func TestInputSurfaceClosed(t *testing.T) {
	b := newBundle(t)
	req := b.request(t)
	req.Inputs = map[string]any{"cve-id": "CVE-1", "sneaky": "x"}
	if _, err := Instantiate(b.catalogPath, "investigate-cve@1", req); !errors.Is(err, ErrInputs) ||
		!strings.Contains(err.Error(), "unknown input") {
		t.Fatalf("unknown input must refuse: %v", err)
	}
	req2 := b.request(t)
	req2.Inputs = map[string]any{}
	if _, err := Instantiate(b.catalogPath, "investigate-cve@1", req2); !errors.Is(err, ErrInputs) ||
		!strings.Contains(err.Error(), "nothing is defaulted") {
		t.Fatalf("missing required input must refuse: %v", err)
	}
	req3 := b.request(t)
	req3.Inputs = map[string]any{"cve-id": strings.Repeat("x", 65)}
	if _, err := Instantiate(b.catalogPath, "investigate-cve@1", req3); !errors.Is(err, ErrInputs) {
		t.Fatalf("over-length input must refuse: %v", err)
	}
}

// D-L9-4 / D-L9-7: templates must use placeholders — a literal
// task_id in a reviewed template is refused, so instantiation cannot
// silently execute another task's identity.
func TestTemplatePlaceholdersRequired(t *testing.T) {
	b := newBundle(t)
	literal := strings.Replace(fxGrant, `"@task_id"`, `"t-other"`, 1)
	write(t, b.dir, "grant.json", literal)
	// Re-pin the manifest so only the placeholder rule can fail.
	repin(t, b, "grant_template", "grant.json", literal)
	if _, err := Instantiate(b.catalogPath, "investigate-cve@1", b.request(t)); !errors.Is(err, ErrResolve) ||
		!strings.Contains(err.Error(), "placeholder") {
		t.Fatalf("literal task_id in a template must refuse: %v", err)
	}
}

// D-5 (audit): a substituted artifact that is structurally invalid
// fails HERE, at the substitution site, not later inside SubmitTask.
func TestEffectiveArtifactsRevalidated(t *testing.T) {
	b := newBundle(t)
	broken := `{"version":1,"task_id":"@task_id","total_max_calls":0,"entries":[{"tool":"read_file","max_calls":8,"workspace":"@workspace"}]}`
	write(t, b.dir, "grant.json", broken)
	repin(t, b, "grant_template", "grant.json", broken)
	if _, err := Instantiate(b.catalogPath, "investigate-cve@1", b.request(t)); !errors.Is(err, ErrResolve) ||
		!strings.Contains(err.Error(), "invalid after substitution") {
		t.Fatalf("invalid effective grant must fail at the substitution site: %v", err)
	}
}

// D-L9-7 / D-L9-13: instantiation records BOTH template and effective
// identities, so the record shows what was reviewed and what executed.
func TestOriginCarriesTemplateAndEffectiveHashes(t *testing.T) {
	b := newBundle(t)
	envPath, err := Instantiate(b.catalogPath, "investigate-cve@1", b.request(t))
	if err != nil {
		t.Fatal(err)
	}
	env := readJSON(t, envPath)
	origin, ok := env["origin"].(map[string]any)
	if !ok {
		t.Fatal("skill-instantiated envelopes must carry origin attribution")
	}
	// Every recorded identity must equal the bytes it claims to
	// identify. Asserting non-emptiness let a mutation record the
	// template hash as the effective one, or any 64-hex string as the
	// catalog anchor (test review HIGH: attribution honesty defeated).
	cat, err := LoadCatalog(b.catalogPath)
	if err != nil {
		t.Fatal(err)
	}
	_, m, err := cat.Resolve("investigate-cve@1")
	if err != nil {
		t.Fatal(err)
	}
	outDir := filepath.Dir(envPath)
	recomputed := map[string]string{
		"skill":              "investigate-cve@1",
		"skill_composition":  m.CompositionHash,
		"skill_catalog":      cat.Hash,
		"grant_template":     m.GrantTemplate.SHA256,
		"spec_template":      m.SpecTemplate.SHA256,
		"skill_procedure":    m.Procedure.SHA256,
		"skill_workflow":     m.Workflow.SHA256,
		"grant_instantiated": shaFile(t, filepath.Join(outDir, "t-1-grant.json")),
		"spec_effective":     shaFile(t, filepath.Join(outDir, "t-1-spec.json")),
	}
	for key, want := range recomputed {
		got, _ := origin[key].(string)
		if got != want {
			t.Errorf("origin[%q] = %q, but the bytes it names hash to %q", key, got, want)
		}
	}
	if origin["grant_template"] == origin["grant_instantiated"] {
		t.Fatal("template and instantiated identities must be distinguishable")
	}
}

// --- helpers ---------------------------------------------------------

func readJSON(t *testing.T, path string) map[string]any {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var doc map[string]any
	if err := json.Unmarshal(raw, &doc); err != nil {
		t.Fatal(err)
	}
	return doc
}

// repin rewrites the manifest pin for one artifact and re-registers
// the resulting composition, so a test can exercise a rule OTHER than
// the byte-verification rule.
func repin(t *testing.T, b *bundle, pinField, file, body string) {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join(b.dir, "skill.json"))
	if err != nil {
		t.Fatal(err)
	}
	var doc map[string]any
	if err := json.Unmarshal(raw, &doc); err != nil {
		t.Fatal(err)
	}
	doc[pinField] = map[string]string{"path": file, "sha256": sha([]byte(body))}
	out, _ := json.Marshal(doc)
	write(t, b.dir, "skill.json", string(out))
	cat := fmt.Sprintf(`{"version":1,"entries":[
	 {"name":"investigate-cve","version":1,"composition_sha256":%q,"manifest_path":"bundle/skill.json","state":"active"}]}`,
		sha(out))
	write(t, b.catalogDir, "catalog.json", cat)
	b.composition = sha(out)
}

// shaFile hashes a file's bytes — used to recompute the identity an
// origin field claims, rather than trusting the claim.
func shaFile(t *testing.T, path string) string {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return sha(b)
}

// Final architecture review HIGH: manifest_path was validated only
// lexically, so a symlink inside the catalog root passed and was then
// followed — reading a manifest from outside the root AND setting
// m.Dir to the foreign directory, so every subsequent pin resolved
// confined to the attacker's root. This is the symmetric twin of
// TestPinSymlinkEscapeRefused one level down.
func TestManifestSymlinkEscapeRefused(t *testing.T) {
	b := newBundle(t)
	// A complete, internally valid bundle OUTSIDE the catalog root.
	foreign := t.TempDir()
	for name, body := range map[string]string{
		"workflow.json": fxWorkflow, "ceiling.json": fxCeiling, "contract.json": fxContract,
		"grant.json": fxGrant, "spec.json": fxSpec, "schema.json": fxSchema, "procedure.md": fxProc,
	} {
		write(t, foreign, name, body)
	}
	man := readFileT(t, filepath.Join(b.dir, "skill.json"))
	write(t, foreign, "skill.json", man)

	// Replace the in-root manifest with a symlink to the foreign one.
	link := filepath.Join(b.dir, "skill.json")
	if err := os.Remove(link); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join(foreign, "skill.json"), link); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	cat, err := LoadCatalog(b.catalogPath)
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := cat.Resolve("investigate-cve@1"); err == nil {
		t.Fatal("a manifest_path resolving through a symlink out of the catalog root must refuse — otherwise every pin resolves against the attacker's directory")
	}
}

func readFileT(t *testing.T, p string) string {
	t.Helper()
	b, err := os.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}
