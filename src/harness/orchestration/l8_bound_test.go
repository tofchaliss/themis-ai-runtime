package orchestration

// Register D (C-L8-19 I): the static execution bound from loaded
// artifacts — parent turns ≤ W, delegations ≤ min(D, G) ≤ M, every
// model execution one or the other.

import (
	"path/filepath"
	"testing"

	"github.com/tofchaliss/themis/tools"
)

func TestStaticExecutionBoundWithDelegate(t *testing.T) {
	reg, err := tools.LoadRegistry(filepath.Join(repoRoot, "policies/tools/registry-v5.json"))
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	cpath := writeJSON(t, dir, "c.json", `{"version":1,"allowed_tools":["read_file","delegate","declare_done"],"max_total_calls":30,"max_walk_length":200,"max_turns_per_phase":10}`)
	c, err := LoadWorkflowCeiling(cpath)
	if err != nil {
		t.Fatal(err)
	}
	wf, err := LoadWorkflow(writeJSON(t, dir, "wf.json", `{
 "version":1,"name":"w","initial":"A",
 "declared_events":["turn-no-action","turn-provider-error","turns-exhausted","tool-error","signal:phase-completion-requested"],
 "phases":[{"name":"A","capabilities":["read_file","delegate","declare_done"],"max_model_turns":8,"edges":[
    {"on":"signal:phase-completion-requested","to":"@complete"},
    {"on":"turn-no-action","to":"@stay","counter":3,"exhausted_to":"@fail"},
    {"on":"turn-provider-error","to":"@fail"},
    {"on":"tool-error","to":"@stay","counter":5,"exhausted_to":"@fail"},
    {"on":"turns-exhausted","to":"@fail"}]}]}`), c)
	if err != nil {
		t.Fatal(err)
	}
	grant := func(d, total int) *tools.Grant {
		return &tools.Grant{Version: 1, TaskID: "T", TotalMaxCalls: total, Entries: []tools.GrantEntry{
			{Tool: "read_file", MaxCalls: 8, Workspace: "/w"},
			{Tool: "delegate", MaxCalls: d, TemplateScope: []string{"dependency-triage@1"}},
			{Tool: "declare_done", MaxCalls: 6}}}
	}
	b := executionBound(wf, c, grant(3, 30), reg)
	if b.Turns != wf.WorstCaseLen || b.Delegations != 3 || b.Executions != wf.WorstCaseLen+3 {
		t.Fatalf("%+v", b)
	}
	if b.Executions > b.WalkCeiling+b.CallCeiling {
		t.Fatalf("bound violated: %+v", b)
	}
	// min(D, G): a delegate cap above the grant total is bounded by
	// the total.
	if b := executionBound(wf, c, grant(50, 30), reg); b.Delegations != 30 {
		t.Fatalf("delegations must be min(D, G): %+v", b)
	}
	// No delegate entry: no delegation term.
	g := grant(3, 30)
	g.Entries = g.Entries[:1]
	if b := executionBound(wf, c, g, reg); b.Delegations != 0 || b.Executions != wf.WorstCaseLen {
		t.Fatalf("%+v", b)
	}
}
