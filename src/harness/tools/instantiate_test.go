package tools

import (
	"strings"
	"testing"
)

// D-SA-4 relation table: each clause is coupled to exactly one
// mutation, so suppressing a clause lets exactly its mutation through.
func TestInstantiates(t *testing.T) {
	tpl := `{"version":1,"task_id":"@task_id","total_max_calls":10,"entries":[
	 {"tool":"read_file","max_calls":6,"workspace":"@workspace"},
	 {"tool":"write_file","max_calls":2,"workspace":"@workspace","mutating":true},
	 {"tool":"delegate","max_calls":2,"template_scope":["analysis@1","remediation@1"]},
	 {"tool":"declare_done","max_calls":2}]}`
	ok := `{"version":1,"task_id":"T1","total_max_calls":8,"entries":[
	 {"tool":"read_file","max_calls":3,"workspace":"@workspace"},
	 {"tool":"write_file","max_calls":1,"workspace":"@workspace","mutating":true},
	 {"tool":"delegate","max_calls":2,"template_scope":["remediation@1","analysis@1"]},
	 {"tool":"declare_done","max_calls":2}]}`
	mut := func(from, to string) string { return strings.Replace(ok, from, to, 1) }
	cases := []struct{ name, eff, want string }{
		{"legitimate narrowing (scope set-equal, order irrelevant)", ok, ""},
		{"bound workspace is still an instantiation", mut(`"workspace":"@workspace"},
	 {"tool":"write_file"`, `"workspace":"/abs/ws"},
	 {"tool":"write_file"`), ""},
		{"quota widened", mut(`"max_calls":3`, `"max_calls":7`), "quotas only narrow"},
		{"total widened", mut(`"total_max_calls":8`, `"total_max_calls":11`), "quotas only narrow"},
		{"tool added", mut(`{"tool":"declare_done"`, `{"tool":"search_code","max_calls":1},{"tool":"declare_done"`), "not in the template"},
		{"tool removed", mut(`{"tool":"declare_done","max_calls":2}`, `{"tool":"declare_done","max_calls":2}`+"\x00"), ""}, // placeholder, replaced below
		{"mutating flipped", mut(`"mutating":true`, `"mutating":false`), "mutating flag differs"},
		{"scope widened", mut(`["remediation@1","analysis@1"]`, `["remediation@1","analysis@1","incident-analysis@1"]`), "template_scope differs"},
		{"scope narrowed", mut(`["remediation@1","analysis@1"]`, `["analysis@1"]`), "template_scope differs"},
		{"workspace dropped", mut(`"max_calls":3,"workspace":"@workspace"`, `"max_calls":3`), "workspace binding differs"},
		{"wrong task", mut(`"task_id":"T1"`, `"task_id":"T2"`), "is not the task"},
		{"floating scope entry refused at parse", mut(`"analysis@1"]`, `"analysis@latest"]`), "exact name@version"},
	}
	// "tool removed": drop declare_done entirely.
	removed := strings.Replace(ok, `,
	 {"tool":"declare_done","max_calls":2}`, ``, 1)
	cases[5].eff, cases[5].want = removed, "in the template but not the effective grant"
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			err := Instantiates([]byte(c.eff), []byte(tpl), "T1")
			if c.want == "" {
				if err != nil {
					t.Fatalf("must instantiate: %v", err)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), c.want) {
				t.Fatalf("refused for the wrong reason (want %q): %v", c.want, err)
			}
		})
	}
	// A template that is not a template (literal task_id) is refused
	// as such, never accepted as a trivially-equal instance.
	if err := Instantiates([]byte(ok), []byte(ok), "T1"); err == nil || !strings.Contains(err.Error(), "@task_id placeholder") {
		t.Fatalf("a literal-task template must refuse: %v", err)
	}
}

// LoadGrant refuses a template_scope entry the runtime gate could never
// match exactly (C-L8-15 G).
func TestLoadGrantTemplateScopeRules(t *testing.T) {
	dir := t.TempDir()
	write := func(name, body string) string {
		writeFile(t, dir, name, body)
		return dir + "/" + name
	}
	good := write("g.json", `{"version":1,"task_id":"T","total_max_calls":2,"entries":[{"tool":"delegate","max_calls":2,"template_scope":["analysis@1"]}]}`)
	if _, err := LoadGrant(good); err != nil {
		t.Fatalf("exact scope must load: %v", err)
	}
	for name, body := range map[string]string{
		"floating": `{"version":1,"task_id":"T","total_max_calls":2,"entries":[{"tool":"delegate","max_calls":2,"template_scope":["analysis@latest"]}]}`,
		"dup":      `{"version":1,"task_id":"T","total_max_calls":2,"entries":[{"tool":"delegate","max_calls":2,"template_scope":["analysis@1","analysis@1"]}]}`,
	} {
		if _, err := LoadGrant(write(name+".json", body)); err == nil {
			t.Fatalf("%s template_scope must refuse", name)
		}
	}
}
