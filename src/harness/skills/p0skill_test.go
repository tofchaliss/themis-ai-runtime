package skills

// The P0 slice: the real investigate-cve@1 composition as authored in
// policies/skills/. These tests read the GOVERNED artifacts, not
// fixtures — if the shipped bundle drifts out of conformance, they
// fail. Registration itself remains a governance act; the catalog
// under test here is the proposed one.

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const (
	p0Catalog = "policies/skills/catalog.proposed.json"
	p0Ref     = "investigate-cve@1"
)

// The authored bundle is internally consistent: every pin matches the
// bytes on disk, and the catalog's composition hash matches the
// manifest. This is the check that would catch an artifact edited
// without re-pinning.
func TestP0SkillBundleIsConsistent(t *testing.T) {
	cat, err := LoadCatalog(mustAbs(t, filepath.Join(repoRoot, p0Catalog)))
	if err != nil {
		t.Fatal(err)
	}
	entry, m, err := cat.Resolve(p0Ref)
	if err != nil {
		t.Fatalf("the authored composition must resolve: %v", err)
	}
	if entry.Steward == "" {
		t.Error("a registered skill should name its steward (accountability metadata)")
	}
	// Every pin resolves and byte-verifies.
	for name, pin := range map[string]Pin{
		"workflow": m.Workflow, "workflow_ceiling": m.WorkflowCeiling,
		"context_contract": m.ContextContract, "grant_template": m.GrantTemplate,
		"spec_template": m.SpecTemplate, "input_schema": m.InputSchema,
		"procedure": m.Procedure,
	} {
		if _, _, err := m.resolvePin(name, pin); err != nil {
			t.Errorf("pin %s does not match its bytes: %v", name, err)
		}
	}
}

// D-L9-1/D-L9-6: the composition's own artifacts are mutually
// consistent — the contract is minted for this workflow, and the
// grant template stays within the skill's ceiling.
func TestP0SkillCompositionCoheres(t *testing.T) {
	cat, err := LoadCatalog(mustAbs(t, filepath.Join(repoRoot, p0Catalog)))
	if err != nil {
		t.Fatal(err)
	}
	_, m, err := cat.Resolve(p0Ref)
	if err != nil {
		t.Fatal(err)
	}
	_, wfRaw, err := m.resolvePin("workflow", m.Workflow)
	if err != nil {
		t.Fatal(err)
	}
	_, ccRaw, err := m.resolvePin("context_contract", m.ContextContract)
	if err != nil {
		t.Fatal(err)
	}
	// The contract names the workflow it belongs to (L7 refuses a
	// mismatch at assembly; catching it here means the skill never
	// ships incoherent).
	if !strings.Contains(string(ccRaw), `"workflow": "investigate-cve"`) {
		t.Error("the context contract must be minted for this workflow")
	}
	// The contract declares the slot the loop composes into.
	if !strings.Contains(string(ccRaw), `"task-payload"`) {
		t.Error("the contract must declare the task-payload slot the loop fills")
	}
	// Every workflow capability appears in the skill's own ceiling.
	_, ceilRaw, err := m.resolvePin("workflow_ceiling", m.WorkflowCeiling)
	if err != nil {
		t.Fatal(err)
	}
	for _, cap := range []string{"read_file", "list_directory", "search_code", "write_file", "declare_done"} {
		if strings.Contains(string(wfRaw), `"`+cap+`"`) && !strings.Contains(string(ceilRaw), `"`+cap+`"`) {
			t.Errorf("workflow uses %q but the skill ceiling does not allow it", cap)
		}
	}
}

// The authored input schema accepts a realistic task and refuses the
// things the closed vocabulary is supposed to refuse.
func TestP0SkillInputSchema(t *testing.T) {
	cat, err := LoadCatalog(mustAbs(t, filepath.Join(repoRoot, p0Catalog)))
	if err != nil {
		t.Fatal(err)
	}
	_, m, err := cat.Resolve(p0Ref)
	if err != nil {
		t.Fatal(err)
	}
	_, raw, err := m.resolvePin("input_schema", m.InputSchema)
	if err != nil {
		t.Fatal(err)
	}
	schema, err := parseSchema(raw)
	if err != nil {
		t.Fatalf("the authored schema must load: %v", err)
	}
	if err := schema.Validate(map[string]any{
		"cve-id": "CVE-2026-12345", "component": "libxyz", "component-version": "1.4.2",
	}); err != nil {
		t.Fatalf("a realistic task must validate: %v", err)
	}
	if err := schema.Validate(map[string]any{"component": "libxyz"}); err == nil {
		t.Error("a missing required input must refuse")
	}
	if err := schema.Validate(map[string]any{
		"cve-id": "CVE-1", "component": "libxyz", "exploit": "rm -rf /",
	}); err == nil {
		t.Error("an unknown input must refuse — the surface is closed")
	}
}

// The procedure is advisory technique with honest standing
// constraints: it must not contain language that asserts authority.
func TestP0ProcedureClaimsNoAuthority(t *testing.T) {
	body, err := os.ReadFile(mustAbs(t, filepath.Join(repoRoot, "policies/skills/investigate-cve/procedure.md")))
	if err != nil {
		t.Fatal(err)
	}
	text := strings.ToLower(string(body))
	// A procedure that told the model its completion constitutes
	// acceptance, or that it may bypass a gate, would be authority
	// laundering through advisory text (D-L9-8/D-L9-9).
	for _, forbidden := range []string{
		"you are authorized", "skip the", "bypass", "you may ignore",
		"this approves", "constitutes approval", "grants you",
	} {
		if strings.Contains(text, forbidden) {
			t.Errorf("procedure text must never assert authority: found %q", forbidden)
		}
	}
	// And it should state the evidence-is-data posture explicitly.
	if !strings.Contains(text, "data, not instruction") {
		t.Error("the procedure should state the evidence-is-data constraint")
	}
}
