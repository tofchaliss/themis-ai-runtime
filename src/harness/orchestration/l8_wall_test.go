package orchestration

// L7-side walls for the L8 seam (D-L8-21 §3, C-L8-11, C-L8-17).

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"reflect"
	"strings"
	"testing"
)

// The Delegator interface carries no conversation: no model.Message,
// ExecutionResponse, or system-message type anywhere in its
// signatures (checked by reflection over the parameter types).
func TestDelegatorInterfaceCarriesNoConversation(t *testing.T) {
	it := reflect.TypeOf((*Delegator)(nil)).Elem()
	if it.NumMethod() != 3 {
		t.Fatalf("Delegator has %d methods; the seam is Registered / Instantiate / Delegate", it.NumMethod())
	}
	var walk func(reflect.Type, string, int)
	seen := map[reflect.Type]bool{}
	walk = func(rt reflect.Type, where string, depth int) {
		if depth > 6 || seen[rt] {
			return
		}
		seen[rt] = true
		name := rt.String()
		for _, bad := range []string{"model.Message", "model.ExecutionResponse", "model.ExecutionRequest"} {
			if strings.Contains(name, bad) {
				t.Errorf("%s reaches %s — the parent conversation must not cross the seam", where, name)
			}
		}
		switch rt.Kind() {
		case reflect.Struct:
			for i := 0; i < rt.NumField(); i++ {
				walk(rt.Field(i).Type, where+"."+rt.Field(i).Name, depth+1)
			}
		case reflect.Ptr, reflect.Slice, reflect.Array, reflect.Map:
			walk(rt.Elem(), where, depth+1)
		case reflect.Func:
			for i := 0; i < rt.NumIn(); i++ {
				walk(rt.In(i), where+"(arg)", depth+1)
			}
		}
	}
	for i := 0; i < it.NumMethod(); i++ {
		m := it.Method(i)
		for j := 0; j < m.Type.NumIn(); j++ {
			walk(m.Type.In(j), m.Name, 0)
		}
	}
}

// The loop's delegation branch appends exactly one tool-role message
// paired by ToolCallID and contains no δ step.
func TestDelegationBranchNeverSteps(t *testing.T) {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "delegation.go", nil, 0)
	if err != nil {
		t.Fatal(err)
	}
	var fn *ast.FuncDecl
	for _, d := range f.Decls {
		if fd, ok := d.(*ast.FuncDecl); ok && fd.Name.Name == "delegate" && fd.Recv != nil {
			fn = fd
		}
	}
	if fn == nil {
		t.Fatal("walk.delegate not found")
	}
	ast.Inspect(fn, func(n ast.Node) bool {
		if c, ok := n.(*ast.CallExpr); ok {
			if sel, ok := c.Fun.(*ast.SelectorExpr); ok {
				switch sel.Sel.Name {
				case "step", "Transition", "AppendEvent", "StoreObject", "Execute":
					t.Errorf("walk.delegate calls %s at %s — the branch delivers, it does not orchestrate", sel.Sel.Name, fset.Position(c.Pos()))
				}
			}
		}
		return true
	})
	// The loop wires the branch to exactly one append and a continue.
	b, err := os.ReadFile("loop.go")
	if err != nil {
		t.Fatal(err)
	}
	src := string(b)
	i := strings.Index(src, "dmsg, derr := w.delegate(")
	if i < 0 {
		t.Fatal("delegation branch not found in loop.go")
	}
	branch := src[i : strings.Index(src[i:], "continue")+i]
	if strings.Count(branch, "conversation = append(conversation, dmsg)") != 1 || strings.Contains(branch, "w.step(") {
		t.Fatalf("delegation branch must append once and never step:\n%s", branch)
	}
}

// controlVerbs and verificationEvents are byte-unchanged by L8: the
// delegation event is not workflow vocabulary.
func TestL8LeavesWorkflowVocabularyUnchanged(t *testing.T) {
	if len(controlVerbs) != 1 || controlVerbs[VerbDeclareDone] != SignalPhaseCompletionRequested {
		t.Fatalf("controlVerbs changed: %v", controlVerbs)
	}
	if len(verificationEvents) != 5 {
		t.Fatalf("verificationEvents changed: %v", verificationEvents)
	}
	for k := range controlVerbs {
		if strings.Contains(k, "delegat") {
			t.Fatal("a delegation control verb exists")
		}
	}
}
