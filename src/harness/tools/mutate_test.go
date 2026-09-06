package tools

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tofchaliss/themis/runtime/model"
)

func mkCall(name, args string) model.ToolCall {
	return model.ToolCall{ID: "c", Name: name, Arguments: json.RawMessage(args)}
}

func shippedRegistryV2(t *testing.T) *Registry {
	t.Helper()
	r, err := LoadRegistry("../../../policies/tools/registry-v2.json")
	if err != nil {
		t.Fatal(err)
	}
	return r
}

func mutGrant(t *testing.T, ws string, mutating bool) *Grant {
	t.Helper()
	flag := ""
	if mutating {
		flag = `,"mutating":true`
	}
	return loadGrantBody(t, `{"version":1,"task_id":"M3","total_max_calls":50,"entries":[
	  {"tool":"write_file","max_calls":10,"workspace":"`+ws+`"`+flag+`},
	  {"tool":"apply_patch","max_calls":10,"workspace":"`+ws+`"`+flag+`}]}`)
}

// D-L5-4: registry-v2 declares the mutating capabilities; a grant
// without the mutating visibility flag cannot use them, with zero
// model detail (availability-class denial).
func TestMutatingVisibilityRequired(t *testing.T) {
	reg := shippedRegistryV2(t)
	ws := t.TempDir()
	grant := mutGrant(t, ws, false)
	d := Authorize(reg, grant, "write_file", json.RawMessage(`{"path":"a.go","content":"x"}`), CallState{Calls: map[string]int{}})
	if d.Allow || d.Denial != DenialNotAvailable || d.ModelDetail != "" {
		t.Fatalf("mutating tool without visibility flag must be not-available with zero detail: %+v", d)
	}
	if !strings.Contains(d.TracePredicate, "mutating-not-visible") {
		t.Fatalf("trace must carry the exact predicate: %q", d.TracePredicate)
	}
	// With the flag, the same call authorizes (the flag is the only
	// difference).
	d2 := Authorize(reg, mutGrant(t, ws, true), "write_file", json.RawMessage(`{"path":"a.go","content":"x"}`), CallState{Calls: map[string]int{}})
	if !d2.Allow {
		t.Fatalf("mutating-visible grant must authorize: %+v", d2)
	}
}

func TestWriteFileExecutor(t *testing.T) {
	reg := shippedRegistryV2(t)
	ws := t.TempDir()
	writeFile(t, ws, "old.go", "package old\n")
	grant := mutGrant(t, ws, true)
	table, err := NewExecutorTable(reg, nil)
	if err != nil {
		t.Fatal(err)
	}
	state := CallState{Calls: map[string]int{}}
	call := func(name, args string) (string, AuditEvent) {
		msg, _, audit := Handle(reg, grant, table, mkCall(name, args), state)
		return msg.Content, audit
	}

	// Creation: record carries new_hash, no old_hash.
	_, audit := call("write_file", `{"path":"pkg.go","content":"package p\n"}`)
	if audit.Decision != "authorized" {
		t.Fatalf("visible mutating grant must authorize: %+v", audit)
	}
	b, err := os.ReadFile(filepath.Join(ws, "pkg.go"))
	if err != nil || string(b) != "package p\n" {
		t.Fatalf("file must be written: %v", err)
	}
	// Overwrite: record carries old_hash + new_hash.
	_, audit2 := call("write_file", `{"path":"old.go","content":"package new\n"}`)
	if audit2.Decision != "authorized" {
		t.Fatalf("overwrite: %+v", audit2)
	}
	var rec MutationRecord
	msg, _, _ := Handle(reg, grant, table, mkCall("write_file", `{"path":"old.go","content":"package newer\n"}`), state)
	if err := json.Unmarshal([]byte(msg.Content), &rec); err != nil {
		t.Fatalf("result must be a mutation record: %v (%q)", err, msg.Content)
	}
	if rec.OldHash == "" || rec.NewHash == "" || rec.Op != "write" {
		t.Fatalf("mutation record must carry old/new hashes: %+v", rec)
	}

	// Refusals: .git*, symlink parent, missing parent — all typed
	// write-refused errors, never partial writes.
	for _, bad := range []string{`{"path":".git/config","content":"x"}`, `{"path":".gitignore","content":"x"}`, `{"path":"no/such/dir/f.go","content":"x"}`} {
		_, _, a := Handle(reg, grant, table, mkCall("write_file", bad), state)
		if a.Decision != "error" || a.ErrClass != ErrWriteRefused {
			t.Fatalf("%s must be write-refused: %+v", bad, a)
		}
	}
	outside := t.TempDir()
	if err := os.Symlink(outside, filepath.Join(ws, "esc")); err != nil {
		t.Fatal(err)
	}
	_, _, a := Handle(reg, grant, table, mkCall("write_file", `{"path":"esc/f.go","content":"x"}`), state)
	if a.Decision != "error" || a.ErrClass != ErrWriteRefused {
		t.Fatalf("symlinked-parent write must be refused: %+v", a)
	}
	if _, err := os.Stat(filepath.Join(outside, "f.go")); !os.IsNotExist(err) {
		t.Fatal("nothing may land outside the workspace")
	}
}

func TestApplyPatchTransactional(t *testing.T) {
	reg := shippedRegistryV2(t)
	ws := t.TempDir()
	writeFile(t, ws, "a.go", "package a\n")
	writeFile(t, ws, "b.go", "package b\n")
	grant := mutGrant(t, ws, true)
	table, err := NewExecutorTable(reg, nil)
	if err != nil {
		t.Fatal(err)
	}
	state := CallState{Calls: map[string]int{}}
	apply := func(patch string) AuditEvent {
		raw, _ := json.Marshal(map[string]string{"patch": patch})
		_, _, audit := Handle(reg, grant, table, mkCall("apply_patch", string(raw)), state)
		return audit
	}

	// Valid multi-op transaction.
	audit := apply(`{"ops":[
	  {"op":"write","path":"c.go","content":"package c\n"},
	  {"op":"rename","from":"b.go","to":"renamed.go"},
	  {"op":"delete","path":"a.go"}]}`)
	if audit.Decision != "authorized" {
		t.Fatalf("valid patch must apply: %+v", audit)
	}
	if _, err := os.Stat(filepath.Join(ws, "c.go")); err != nil {
		t.Fatal("write op must land")
	}
	if _, err := os.Stat(filepath.Join(ws, "renamed.go")); err != nil {
		t.Fatal("rename op must land")
	}
	if _, err := os.Stat(filepath.Join(ws, "a.go")); !os.IsNotExist(err) {
		t.Fatal("delete op must land")
	}

	// Invalid op anywhere ⇒ NOTHING applies (validation precedes
	// mutation).
	audit = apply(`{"ops":[
	  {"op":"write","path":"d.go","content":"package d\n"},
	  {"op":"delete","path":"ghost.go"}]}`)
	if audit.Decision != "error" || audit.ErrClass != ErrWriteRefused {
		t.Fatalf("patch with invalid op must refuse: %+v", audit)
	}
	if _, err := os.Stat(filepath.Join(ws, "d.go")); !os.IsNotExist(err) {
		t.Fatal("transaction must apply NOTHING on failure")
	}
	// Rename into .git*, unknown op, unknown field: refused whole.
	for _, bad := range []string{
		`{"ops":[{"op":"rename","from":"c.go","to":".git/hooks/x"}]}`,
		`{"ops":[{"op":"chmod","path":"c.go"}]}`,
		`{"ops":[{"op":"write","path":"e.go","content":"x","mode":"0777"}]}`,
	} {
		if a := apply(bad); a.Decision != "error" || a.ErrClass != ErrWriteRefused {
			t.Fatalf("%s must refuse: %+v", bad, a)
		}
	}
	if _, err := os.Stat(filepath.Join(ws, "c.go")); err != nil {
		t.Fatal("refused patches must leave prior state intact")
	}
}
