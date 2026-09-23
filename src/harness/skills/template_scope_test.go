package skills

import (
	"reflect"
	"strings"
	"testing"
)

// C-L8-16 F/G: template_scope is a fixed member of the Skill's grant
// template, copied verbatim at instantiation. The closed instantiation
// surface has no field through which a caller could name, narrow, or
// widen it — a structural fact, pinned here so the surface cannot
// quietly grow one.
func TestInstantiationSurfaceHasNoTemplateScope(t *testing.T) {
	rt := reflect.TypeOf(Request{})
	for i := 0; i < rt.NumField(); i++ {
		n := strings.ToLower(rt.Field(i).Name)
		if strings.Contains(n, "template") || strings.Contains(n, "scope") {
			t.Fatalf("Request.%s: the instantiation surface must not expose template scope", rt.Field(i).Name)
		}
	}
	// QuotaOverrides narrows max_calls per tool only; its value type
	// cannot carry a scope.
	f, _ := rt.FieldByName("QuotaOverrides")
	if f.Type.String() != "map[string]int" {
		t.Fatalf("QuotaOverrides must stay tool→max_calls: %s", f.Type)
	}
}
