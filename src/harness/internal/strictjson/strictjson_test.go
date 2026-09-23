package strictjson

import (
	"strings"
	"testing"
)

func TestCheck(t *testing.T) {
	cases := []struct{ name, doc, want string }{
		{"exact lowercase keys pass", `{"a":1,"b_c":[{"d":2},{"d":3}],"e":{"f":null}}`, ""},
		{"case variant refused", `{"a":1,"Workspace":"/etc"}`, "not an exact lowercase key"},
		{"nested case variant refused", `{"entries":[{"tool":"x","Workspace":"/etc"}]}`, "not an exact lowercase key"},
		{"duplicate refused", `{"a":1,"a":2}`, "duplicate key"},
		{"nested duplicate refused", `{"a":[{"b":1,"b":2}]}`, "duplicate key"},
		{"same key in sibling objects is fine", `{"a":[{"b":1},{"b":2}]}`, ""},
		{"hyphen refused", `{"a-b":1}`, "not an exact lowercase key"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			err := Check([]byte(c.doc))
			if c.want == "" {
				if err != nil {
					t.Fatalf("must pass: %v", err)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), c.want) {
				t.Fatalf("want %q, got %v", c.want, err)
			}
		})
	}
}
