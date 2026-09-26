package tools

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func scopedGrant(t *testing.T, scope string) *Grant {
	t.Helper()
	body := `{"version":1,"task_id":"T1","total_max_calls":20,"entries":[
	  {"tool":"get_finding","max_calls":3,"themis_scope":[` + scope + `]}
	]}`
	path := filepath.Join(t.TempDir(), "grant.json")
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	g, err := LoadGrant(path)
	if err != nil {
		t.Fatal(err)
	}
	return g
}

// D-I-4: a grant scope of "uuid" authorizes exactly the UUID identity
// shape; a literal prefix keeps its T-M2 meaning; neither widens into
// the other; an empty scope grants nothing. The denial names the scope.
func TestThemisScopeUUIDSyntaxClass(t *testing.T) {
	reg, err := LoadRegistry("../../../policies/tools/registry-v5.json")
	if err != nil {
		t.Fatal(err)
	}
	decide := func(g *Grant, id string) Decision {
		args, _ := json.Marshal(map[string]string{"id": id})
		return Authorize(reg, g, "get_finding", args, CallState{Calls: map[string]int{}})
	}
	uuidGrant := scopedGrant(t, `"uuid"`)
	const id = "b1be6f86-2ecd-451f-9411-95f1f32fd501"
	if d := decide(uuidGrant, id); !d.Allow {
		t.Fatalf("canonical UUID must be in scope: %+v", d)
	}
	for _, bad := range []string{"B1BE6F86-2ECD-451F-9411-95F1F32FD501", "FIND-2026-0001", "b1be6f86-2ecd-451f-9411-95f1f32fd50", "uuid", "b1be6f86-2ecd-451f-9411-95f1f32fd501-x"} {
		if d := decide(uuidGrant, bad); d.Allow || d.Denial != DenialTargetRefused {
			t.Errorf("%q must be refused under a uuid scope: %+v", bad, d)
		}
	}
	prefixGrant := scopedGrant(t, `"FIND-"`)
	if d := decide(prefixGrant, id); d.Allow {
		t.Fatal("a prefix scope must not admit a UUID")
	}
	if d := decide(prefixGrant, "FIND-2026-0001"); !d.Allow {
		t.Fatalf("prefix scope: %+v", d)
	}
	emptyGrant := scopedGrant(t, ``)
	if d := decide(emptyGrant, id); d.Allow {
		t.Fatal("empty scope grants nothing")
	}
}
