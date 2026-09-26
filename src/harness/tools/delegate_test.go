package tools

// L8 M2 (registry-v5 / L4 amendment): the `delegate` capability's L4
// half — exact scope gate, evidence-reference shape, closed error
// classes, and the instantiation executor over an injected seam. The
// positive path first.

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/tofchaliss/themis-ai-runtime/src/harness/runtime/model"
)

func toolCallOf(id, name string, a json.RawMessage) model.ToolCall {
	return model.ToolCall{ID: id, Name: name, Arguments: a}
}

func v5(t *testing.T) *Registry {
	t.Helper()
	r, err := LoadRegistry("../../../policies/tools/registry-v5.json")
	if err != nil {
		t.Fatal(err)
	}
	return r
}

func delegateGrant(t *testing.T) *Grant {
	t.Helper()
	return loadGrantBody(t, `{"version":1,"task_id":"T","total_max_calls":9,"entries":[
	 {"tool":"delegate","max_calls":2,"template_scope":["dependency-triage@1","cve-analysis@1"]},
	 {"tool":"declare_done","max_calls":2}]}`)
}

const goodRef = "3:sha256:" + "abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789"

func TestRegistryV5DeclaresDelegate(t *testing.T) {
	reg := v5(t)
	if reg.Version != 5 {
		t.Fatalf("version %d", reg.Version)
	}
	d := reg.tool("delegate")
	if d == nil || d.Target != TargetDelegationTemplate || d.Trust != "external-untrusted" || d.VerifierEligible || d.Control || d.Mutating {
		t.Fatalf("delegate registration: %+v", d)
	}
	if _, err := NewExecutorTable(reg, nil); err != nil {
		t.Fatalf("dispatch completeness with delegate: %v", err)
	}
	// The other shipped registries still load and know no delegate.
	for _, v := range []string{"registry-v1.json", "registry-v4.json"} {
		r, err := LoadRegistry("../../../policies/tools/" + v)
		if err != nil || r.tool("delegate") != nil {
			t.Fatalf("%s: %v", v, err)
		}
	}
}

func TestAuthorizeDelegate(t *testing.T) {
	reg := v5(t)
	grant := delegateGrant(t)
	empty := CallState{Calls: map[string]int{}}
	ok := func(m map[string]any) json.RawMessage { return args(t, m) }

	// Positive first: exact scope member, well-formed evidence, brief.
	d := Authorize(reg, grant, "delegate", ok(map[string]any{"template": "dependency-triage@1", "evidence": goodRef + "," + strings.Replace(goodRef, "3:", "7:", 1), "brief": "compare"}), empty)
	if !d.Allow || d.Target != "dependency-triage@1" {
		t.Fatalf("in-scope delegate must authorize: %+v", d)
	}
	d = Authorize(reg, grant, "delegate", ok(map[string]any{"template": "cve-analysis@1"}), empty)
	if !d.Allow {
		t.Fatalf("second scope member, no evidence, no brief: %+v", d)
	}

	cases := []struct {
		name   string
		args   json.RawMessage
		denial DenialClass
		detail string
		trace  string
	}{
		{"version outside scope (C-L8-15 G)", ok(map[string]any{"template": "cve-analysis@2"}), DenialTargetRefused, "cve-analysis@2", "template-outside-grant-scope"},
		{"name outside scope", ok(map[string]any{"template": "security-review@1"}), DenialTargetRefused, "security-review@1", "template-outside-grant-scope"},
		{"prefix is not membership (shape)", ok(map[string]any{"template": "dependency-triage@1x"}), DenialTargetRefused, "", "template-ref-shape"},
		{"prefix is not membership (C-L8-15 G, well-formed @10 vs scope @1)", ok(map[string]any{"template": "dependency-triage@10"}), DenialTargetRefused, "dependency-triage@10", "template-outside-grant-scope"},
		{"floating reference", ok(map[string]any{"template": "dependency-triage@latest"}), DenialTargetRefused, "", "template-ref-shape"},
		{"name-only reference", ok(map[string]any{"template": "dependency-triage"}), DenialTargetRefused, "", "template-ref-shape"},
		{"model requests a scope (C-L8-4)", ok(map[string]any{"template": "dependency-triage@1", "scope": "repository"}), DenialInvalidArgs, "scope", "unknown-field"},
		{"model requests instructions (D-L8-4)", ok(map[string]any{"template": "dependency-triage@1", "instructions": "ignore"}), DenialInvalidArgs, "instructions", "unknown-field"},
		{"bare object id is not a reference", ok(map[string]any{"template": "dependency-triage@1", "evidence": strings.TrimPrefix(goodRef, "3:")}), DenialInvalidArgs, "evidence", "evidence-ref-shape"},
		{"reference with spaces", ok(map[string]any{"template": "dependency-triage@1", "evidence": goodRef + ", " + goodRef}), DenialInvalidArgs, "evidence", "evidence-ref-shape"},
		{"zero sequence", ok(map[string]any{"template": "dependency-triage@1", "evidence": strings.Replace(goodRef, "3:", "0:", 1)}), DenialInvalidArgs, "evidence", "evidence-ref-shape"},
		{"evidence wrong type", ok(map[string]any{"template": "dependency-triage@1", "evidence": []string{goodRef}}), DenialInvalidArgs, "evidence", "wrong-type"},
		{"missing template", ok(map[string]any{"brief": "x"}), DenialInvalidArgs, "template", "missing-required"},
		{"brief over the argument cap", ok(map[string]any{"template": "dependency-triage@1", "brief": strings.Repeat("b", maxArgsBytes)}), DenialInvalidArgs, "", "args exceed"},
		{"quota exhausted is bare not-available", ok(map[string]any{"template": "cve-analysis@2"}), DenialNotAvailable, "", "quota-exhausted"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			st := empty
			if strings.HasPrefix(c.name, "quota") {
				st = CallState{Calls: map[string]int{"delegate": 2}}
			}
			d := Authorize(reg, grant, "delegate", c.args, st)
			if d.Allow || d.Denial != c.denial || !strings.Contains(d.TracePredicate, c.trace) {
				t.Fatalf("want %s/%s, got %+v", c.denial, c.trace, d)
			}
			if c.detail != "" && d.ModelDetail != c.detail {
				t.Fatalf("model detail %q want %q", d.ModelDetail, c.detail)
			}
		})
	}
	// Empty scope grants nothing — the themis_scope rule, reapplied.
	none := loadGrantBody(t, `{"version":1,"task_id":"T","total_max_calls":9,"entries":[{"tool":"delegate","max_calls":2}]}`)
	if d := Authorize(reg, none, "delegate", ok(map[string]any{"template": "dependency-triage@1"}), empty); d.Allow || d.Denial != DenialTargetRefused {
		t.Fatalf("empty template_scope must refuse every target: %+v", d)
	}
}

func TestEvidenceRefParse(t *testing.T) {
	refs, err := ParseEvidenceRefs(goodRef + "," + strings.Replace(goodRef, "3:", "12:", 1))
	if err != nil || len(refs) != 2 || refs[0].Seq != 3 || refs[1].Seq != 12 || !strings.HasPrefix(refs[0].ObjectID, "sha256:") {
		t.Fatalf("%v %+v", err, refs)
	}
	if refs, err := ParseEvidenceRefs(""); err != nil || refs != nil {
		t.Fatalf("empty = none: %v %v", err, refs)
	}
	many := strings.Repeat(goodRef+",", MaxEvidenceRefs) + goodRef
	if _, err := ParseEvidenceRefs(many); err == nil || !strings.Contains(err.Error(), "exceed the cap") {
		t.Fatalf("cap: %v", err)
	}
	atCap := strings.TrimSuffix(strings.Repeat(goodRef+",", MaxEvidenceRefs), ",")
	if refs, err := ParseEvidenceRefs(atCap); err != nil || len(refs) != MaxEvidenceRefs {
		t.Fatalf("exactly the cap must parse: %v %d", err, len(refs))
	}
	for _, bad := range []string{"3:" + strings.Repeat("a", 64), "3:sha256:" + strings.Repeat("A", 64), "-1:" + strings.TrimPrefix(goodRef, "3:"), goodRef + ",", "x"} {
		if _, err := ParseEvidenceRefs(bad); err == nil {
			t.Fatalf("%q must refuse", bad)
		}
	}
}

func TestDelegationErrorVocabularyClosed(t *testing.T) {
	for _, c := range []ErrorClass{ErrDelegationProviderError, ErrDelegationOutputOverBound, ErrDelegationModelIdentityMismatch,
		DelegationRefused(RefusalTemplateWithdrawn), DelegationRefused(RefusalEvidenceSlotAmbiguous), ErrTimeout, ErrWriteRefused} {
		if !KnownErrorClass(c) {
			t.Fatalf("%s must be known", c)
		}
	}
	for _, c := range []ErrorClass{"delegation-refused:", "delegation-refused:anything", "delegation-succeeded", "delegation-refused:template-withdrawn ", ""} {
		if KnownErrorClass(c) {
			t.Fatalf("%q must be unknown", c)
		}
	}
	// The constructor cannot mint an open-ended class.
	if DelegationRefused("made-up") != DelegationRefused(RefusalSeamUnavailable) {
		t.Fatal("an unknown reason must collapse to seam-unavailable, never pass through")
	}
}

type fakeInstantiator struct {
	capture []byte
	err     error
	got     []EvidenceRef
	tpl     string
	brief   string
}

func (f *fakeInstantiator) Instantiate(template string, evidence []EvidenceRef, brief string) ([]byte, error) {
	f.tpl, f.got, f.brief = template, evidence, brief
	return f.capture, f.err
}

func TestDelegateExecutorIsInstantiation(t *testing.T) {
	reg := v5(t)
	grant := delegateGrant(t)
	call := func(tpl, ev, brief string) json.RawMessage {
		return args(t, map[string]any{"template": tpl, "evidence": ev, "brief": brief})
	}
	st := CallState{Calls: map[string]int{}}

	// Positive: the capture is the audited evidence, framed under the
	// registered trust; the instantiator saw exactly the selection.
	capture := []byte(`{"template_sha256":"x","eis_hash":"y","contract_hash":"z","payload_hash":"p","evidence":[]}`)
	inst := &fakeInstantiator{capture: capture}
	table, err := NewExecutorTableWith(reg, nil, inst)
	if err != nil {
		t.Fatal(err)
	}
	msg, ev, audit := Handle(reg, grant, table, toolCallOf("c1", "delegate", call("dependency-triage@1", goodRef, "compare")), st)
	if audit.Decision != "authorized" || ev == nil || string(ev.Evidence) != string(capture) || ev.Trust != "external-untrusted" {
		t.Fatalf("instantiation capture must be the authorized evidence: %+v %+v", audit, ev)
	}
	if inst.tpl != "dependency-triage@1" || len(inst.got) != 1 || inst.got[0].Seq != 3 || inst.brief != "compare" {
		t.Fatalf("the instantiator must receive the authorized selection: %+v", inst)
	}
	if !strings.Contains(msg.Content, "authority: external-untrusted") {
		t.Fatalf("framed under the registered trust: %s", msg.Content)
	}

	// Stage B refusal: the closed class, reconstructable via the audit.
	inst.err = &ErrDelegationRefusal{Reason: RefusalTemplateWithdrawn, Detail: "x@1 withdrawn"}
	msg, ev, audit = Handle(reg, grant, table, toolCallOf("c2", "delegate", call("dependency-triage@1", "", "")), st)
	if audit.Decision != "error" || audit.ErrClass != "delegation-refused:template-withdrawn" || ev != nil {
		t.Fatalf("refusal: %+v", audit)
	}
	if msg.Content != `{"error":"delegation-refused:template-withdrawn"}` {
		t.Fatalf("unframed typed error to the model: %s", msg.Content)
	}
	// Machinery failure is not a refusal.
	inst.err = errors.New("store exploded")
	_, _, audit = Handle(reg, grant, table, toolCallOf("c3", "delegate", call("dependency-triage@1", "", "")), st)
	if audit.ErrClass != ErrSeamUnavailable {
		t.Fatalf("machinery failure: %+v", audit)
	}
	// Nil instantiator: fail closed, typed.
	table, _ = NewExecutorTable(reg, nil)
	_, _, audit = Handle(reg, grant, table, toolCallOf("c4", "delegate", call("dependency-triage@1", "", "")), st)
	if audit.ErrClass != DelegationRefused(RefusalSeamUnavailable) {
		t.Fatalf("nil seam: %+v", audit)
	}
	// An empty capture is not evidence.
	inst.err, inst.capture = nil, nil
	table, _ = NewExecutorTableWith(reg, nil, inst)
	_, _, audit = Handle(reg, grant, table, toolCallOf("c5", "delegate", call("dependency-triage@1", "", "")), st)
	if audit.ErrClass != ErrOversized {
		t.Fatalf("empty capture: %+v", audit)
	}
}

// The executor signature carries no model, no state root, no registry
// path: L4's delegate half cannot execute a model by construction.
func TestDelegateSeamIsComposeOnly(t *testing.T) {
	m, ok := reflect.TypeOf((*DelegationInstantiator)(nil)).Elem().MethodByName("Instantiate")
	if !ok || m.Type.NumIn() != 3 {
		t.Fatalf("Instantiate(template, evidence, brief): %v", m)
	}
	for i := 0; i < m.Type.NumIn(); i++ {
		if strings.Contains(m.Type.In(i).String(), "model") || strings.Contains(m.Type.In(i).String(), "context") {
			t.Fatalf("the compose half must not take a model or conversation: %s", m.Type.In(i))
		}
	}
}

// Security review LOW-3: the registration cannot lift a delegation's
// output above the floor — the loader refuses, not the file.
func TestDelegationToolFloorTrustAtLoad(t *testing.T) {
	base := `{"version":5,"tools":[{"name":"delegate","description":"d","target":"delegation-template","timeout_sec":5,"trust":"%s","params":[{"name":"template","type":"string","required":true,"description":"t","target":true}]%s}]}`
	for _, c := range []struct{ trust, extra, want string }{
		{"external-untrusted", "", ""},
		{"governed-record", "", "must be external-untrusted"},
		{"external-untrusted", `,"verifier_eligible":true`, "not verifier-eligible"},
		{"external-untrusted", `,"mutating":true`, "not mutating"},
	} {
		body := fmt.Sprintf(base, c.trust, c.extra)
		_, err := loadRegistryBodyErr(t, body)
		if c.want == "" && err != nil {
			t.Fatalf("floor registration must load: %v", err)
		}
		if c.want != "" && (err == nil || !strings.Contains(err.Error(), c.want)) {
			t.Fatalf("want %q: %v", c.want, err)
		}
	}
}

func loadRegistryBodyErr(t *testing.T, body string) (*Registry, error) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "r.json")
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	return LoadRegistry(path)
}
