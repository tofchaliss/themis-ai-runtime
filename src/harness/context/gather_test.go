package context

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/tofchaliss/themis/instructions"
)

func writeFile(t *testing.T, dir, name, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(filepath.Join(dir, name)), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

const cveContract = `{
  "version": 1,
  "workflow": "cve-analysis",
  "sensitivity_ceiling": "internal",
  "slots": [
    {"name": "finding", "kind": "finding", "requirement": "required", "classes": ["governed-record"]},
    {"name": "task-facts", "kind": "task-facts", "requirement": "required", "classes": ["external-untrusted"]},
    {"name": "source-files", "kind": "file:*", "requirement": "optional", "classes": ["external-untrusted"]},
    {"name": "enterprise-position", "kind": "enterprise-position", "requirement": "optional",
     "classes": ["governed-record"], "withhold": true}
  ]
}`

func loadTestContract(t *testing.T, body string) *Contract {
	t.Helper()
	path := filepath.Join(t.TempDir(), "contract.json")
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	c, err := LoadContract(path)
	if err != nil {
		t.Fatal(err)
	}
	return c
}

type fakeReader struct{ data map[string]string }

func (f fakeReader) Read(kind string) ([]byte, string, error) {
	v, ok := f.data[kind]
	if !ok {
		return nil, "", errors.New("no record")
	}
	return []byte(v), "v1", nil
}

func inlineSrc(items ...ContextItem) Source {
	return Source{Name: "task-payload", Kind: KindInline, Authority: AuthorityExternalUntrusted,
		Sensitivity: SensitivityPublic, Author: "task-caller", Items: items}
}

func themisSrc(data map[string]string) Source {
	return Source{Name: "themis-api", Kind: KindThemis, Authority: AuthorityGovernedRecord,
		Sensitivity: SensitivityInternal, Author: "themis", Reader: fakeReader{data},
		Items: []ContextItem{{Kind: "finding"}}}
}

func stdAssignments(t *testing.T) (*Contract, []Assignment) {
	t.Helper()
	c := loadTestContract(t, cveContract)
	fsRoot := t.TempDir()
	writeFile(t, fsRoot, "parser.go", "package parser\n")
	return c, []Assignment{
		{Slot: "finding", Source: themisSrc(map[string]string{"finding": "CVE-2026-12345 OPEN\n"})},
		{Slot: "task-facts", Source: inlineSrc(ContextItem{Kind: "task-facts", Evidence: []byte("component libXYZ 1.4.2\n")})},
		{Slot: "source-files", Source: Source{Name: "workspace", Kind: KindFilesystem, Authority: AuthorityExternalUntrusted,
			Sensitivity: SensitivityPublic, Author: "repository", Root: fsRoot, Paths: []string{"parser.go"}}},
	}
}

func TestGatherHappyPath(t *testing.T) {
	c, asg := stdAssignments(t)
	g, err := Gather(c, asg)
	if err != nil {
		t.Fatal(err)
	}
	if len(g.Slots) != 4 {
		t.Fatalf("want 4 slot states, got %d", len(g.Slots))
	}
	byName := map[string]SlotState{}
	for _, s := range g.Slots {
		byName[s.Slot] = s
	}
	if byName["finding"].Availability != AvailabilityDelivered ||
		byName["finding"].Items[0].Authority != AuthorityGovernedRecord ||
		byName["finding"].Items[0].Origin != "themis" {
		t.Fatalf("finding slot wrong: %+v", byName["finding"])
	}
	if byName["enterprise-position"].Availability != AvailabilityWithheld ||
		byName["enterprise-position"].Delivery != DeliveryWithheld {
		t.Fatalf("withheld slot wrong: %+v", byName["enterprise-position"])
	}
	if byName["task-facts"].Items[0].Authority != AuthorityExternalUntrusted {
		t.Fatal("task facts must be external-untrusted")
	}
}

// Classification is stamped from registration: caller-supplied class,
// provenance, producer, sensitivity, and hash on inline items are all
// overwritten. No field determines its own treatment.
func TestClassificationForcedFromRegistration(t *testing.T) {
	c := loadTestContract(t, cveContract)
	forged := ContextItem{Kind: "task-facts", Evidence: []byte("facts\n"),
		Authority: AuthorityGovernedRecord, Hash: "forged", Producer: "forged",
		Provenance:  Provenance{Origin: "themis", Source: "forged", Author: "forged"},
		Sensitivity: SensitivityRestricted}
	asg := []Assignment{
		{Slot: "finding", Source: themisSrc(map[string]string{"finding": "F\n"})},
		{Slot: "task-facts", Source: inlineSrc(forged)},
		{Slot: "source-files", Source: unavailableFS(t)},
	}
	g, err := Gather(c, asg)
	if err != nil {
		t.Fatal(err)
	}
	var ref ItemRef
	for _, s := range g.Slots {
		if s.Slot == "task-facts" {
			ref = s.Items[0]
		}
	}
	if ref.Authority != AuthorityExternalUntrusted || ref.Origin != "external" ||
		ref.Source != "task-payload" || ref.Producer != "task-payload" ||
		ref.Sensitivity != SensitivityPublic || ref.Hash != evidenceHash([]byte("facts\n")) {
		t.Fatalf("forged fields survived: %+v", ref)
	}
}

func unavailableFS(t *testing.T) Source {
	t.Helper()
	return Source{Name: "workspace", Kind: KindFilesystem, Authority: AuthorityExternalUntrusted,
		Sensitivity: SensitivityPublic, Author: "repository", Root: t.TempDir(), Paths: []string{"absent.go"}}
}

// A source kind can only mint the classes its provenance can prove.
func TestSourceClassConstraints(t *testing.T) {
	cases := []Source{
		{Name: "x", Kind: KindInline, Authority: AuthorityGovernedRecord, Sensitivity: SensitivityPublic},
		{Name: "x", Kind: KindInline, Authority: AuthorityDerived, Sensitivity: SensitivityPublic},
		{Name: "x", Kind: KindFilesystem, Authority: AuthorityGovernedExternal, Sensitivity: SensitivityPublic},
		{Name: "x", Kind: KindThemis, Authority: AuthorityExternalUntrusted, Sensitivity: SensitivityPublic},
		{Name: "x", Kind: "database", Authority: AuthorityExternalUntrusted, Sensitivity: SensitivityPublic},
		{Name: "", Kind: KindInline, Authority: AuthorityExternalUntrusted, Sensitivity: SensitivityPublic},
		{Name: "x", Kind: KindInline, Authority: AuthorityExternalUntrusted, Sensitivity: "secret"},
	}
	for i, src := range cases {
		if err := checkSource(src); !errors.Is(err, ErrUnrecognizedSource) {
			t.Fatalf("case %d must be rejected: %v", i, err)
		}
	}
}

func TestGatherFailsClosed(t *testing.T) {
	c := loadTestContract(t, cveContract)
	finding := func() Assignment {
		return Assignment{Slot: "finding", Source: themisSrc(map[string]string{"finding": "F\n"})}
	}
	facts := func() Assignment {
		return Assignment{Slot: "task-facts", Source: inlineSrc(ContextItem{Kind: "task-facts", Evidence: []byte("f\n")})}
	}
	files := func() Assignment { return Assignment{Slot: "source-files", Source: unavailableFS(t)} }

	cases := []struct {
		name    string
		asg     []Assignment
		wantErr error
	}{
		{"unknown slot", append([]Assignment{{Slot: "sbom", Source: inlineSrc()}}, finding(), facts(), files()), ErrPlanOutsideContract},
		{"assignment to withheld slot", []Assignment{finding(), facts(), files(),
			{Slot: "enterprise-position", Source: themisSrc(map[string]string{"enterprise-position": "AFFECTED\n"})}}, ErrPlanOutsideContract},
		{"slot assigned twice", []Assignment{finding(), finding(), facts(), files()}, ErrPlanOutsideContract},
		{"class not permitted by slot", []Assignment{
			{Slot: "finding", Source: Source{Name: "t", Kind: KindThemis, Authority: AuthorityGovernedExternal,
				Sensitivity: SensitivityPublic, Reader: fakeReader{map[string]string{"finding": "F\n"}},
				Items: []ContextItem{{Kind: "finding"}}}}, facts(), files()}, ErrPlanOutsideContract},
		{"sensitivity above ceiling", []Assignment{
			{Slot: "finding", Source: Source{Name: "t", Kind: KindThemis, Authority: AuthorityGovernedRecord,
				Sensitivity: SensitivityRestricted, Reader: fakeReader{map[string]string{"finding": "F\n"}},
				Items: []ContextItem{{Kind: "finding"}}}}, facts(), files()}, ErrSensitivityCeiling},
		{"required slot unavailable", []Assignment{
			{Slot: "finding", Source: themisSrc(map[string]string{})}, facts(), files()}, ErrRequiredMissing},
		{"required slot stub-unwired", []Assignment{
			{Slot: "finding", Source: Source{Name: "t", Kind: KindThemis, Authority: AuthorityGovernedRecord,
				Sensitivity: SensitivityPublic, Items: []ContextItem{{Kind: "finding"}}}}, facts(), files()}, ErrRequiredMissing},
		{"kind mismatch", []Assignment{
			{Slot: "finding", Source: Source{Name: "t", Kind: KindThemis, Authority: AuthorityGovernedRecord,
				Sensitivity: SensitivityPublic, Reader: fakeReader{map[string]string{"sbom": "S\n"}},
				Items: []ContextItem{{Kind: "sbom"}}}}, facts(), files()}, ErrPlanOutsideContract},
		{"uncovered non-withheld slot", []Assignment{finding(), facts()}, ErrPlanOutsideContract},
		{"oversized inline item", []Assignment{finding(),
			{Slot: "task-facts", Source: inlineSrc(ContextItem{Kind: "task-facts", Evidence: make([]byte, MaxItemBytes+1)})},
			files()}, ErrItemTooLarge},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			g, err := Gather(c, tc.asg)
			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("want %v, got %v", tc.wantErr, err)
			}
			if g != nil {
				t.Fatal("fail-closed gather must yield nil")
			}
		})
	}
	if _, err := Gather(nil, nil); !errors.Is(err, ErrContractInvalid) {
		t.Fatal("nil contract must abort")
	}
	if _, err := Gather(&Contract{}, nil); !errors.Is(err, ErrContractInvalid) {
		t.Fatal("unhashed contract must abort")
	}
}

// Optional-missing is typed, never silent; required-missing already
// aborted above. Source-status and delivery-status stay distinct.
func TestOptionalUnavailableTyped(t *testing.T) {
	c, asg := stdAssignments(t)
	asg[2].Source = unavailableFS(t)
	g, err := Gather(c, asg)
	if err != nil {
		t.Fatal(err)
	}
	for _, s := range g.Slots {
		if s.Slot == "source-files" {
			if s.Availability != AvailabilityUnavailable || s.SourceStatus != SourceUnavailable || s.Delivery != DeliveryNone {
				t.Fatalf("optional-missing must be typed: %+v", s)
			}
		}
	}
}

// Intake vs resolution classification: inline (task payload) failures
// are intake; config/trusted failures are resolution.
func TestGatherErrorClassification(t *testing.T) {
	c := loadTestContract(t, cveContract)
	_, err := Gather(c, []Assignment{
		{Slot: "task-facts", Source: inlineSrc(ContextItem{Kind: "task-facts", Evidence: make([]byte, MaxItemBytes+1)})},
	})
	if instructions.StatusOf(err) != instructions.StatusFailedIntake {
		t.Fatalf("oversized task payload must be intake: %v", err)
	}
	_, err = Gather(c, []Assignment{{Slot: "sbom", Source: unavailableFS(t)}})
	if instructions.StatusOf(err) != instructions.StatusFailedResolution {
		t.Fatalf("plan-outside-contract is resolution: %v", err)
	}
}

func TestConfinement(t *testing.T) {
	root := t.TempDir()
	outside := t.TempDir()
	writeFile(t, outside, "secret.txt", "secret\n")
	writeFile(t, root, "ok.txt", "ok\n")
	if err := os.Symlink(filepath.Join(outside, "secret.txt"), filepath.Join(root, "link.txt")); err != nil {
		t.Fatal(err)
	}
	mk := func(paths ...string) Source {
		return Source{Name: "ws", Kind: KindFilesystem, Authority: AuthorityExternalUntrusted,
			Sensitivity: SensitivityPublic, Root: root, Paths: paths}
	}
	for _, tc := range []struct {
		name string
		src  Source
	}{
		{"absolute path", mk(filepath.Join(outside, "secret.txt"))},
		{"dotdot escape", mk("../" + filepath.Base(outside) + "/secret.txt")},
		{"symlink escape", mk("link.txt")},
		{"empty path", mk("")},
		{"no root", Source{Name: "ws", Kind: KindFilesystem, Authority: AuthorityExternalUntrusted,
			Sensitivity: SensitivityPublic, Paths: []string{"ok.txt"}}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, _, err := tc.src.collect()
			if !errors.Is(err, ErrConfinement) {
				t.Fatalf("want ErrConfinement, got %v", err)
			}
		})
	}
	// Legitimate confined read still works.
	items, available, err := mk("ok.txt").collect()
	if err != nil || !available || len(items) != 1 || string(items[0].Evidence) != "ok\n" {
		t.Fatalf("confined read failed: %v %v %+v", err, available, items)
	}
}

func TestContractFailsClosed(t *testing.T) {
	cases := []struct {
		name string
		body string
	}{
		{"invalid json", `{`},
		{"unknown field", `{"version":1,"workflow":"w","sensitivity_ceiling":"public","slots":[{"name":"a","kind":"a","requirement":"required","classes":["derived"]}],"extra":1}`},
		{"trailing content", `{"version":1,"workflow":"w","sensitivity_ceiling":"public","slots":[{"name":"a","kind":"a","requirement":"required","classes":["derived"]}]} X`},
		{"no slots", `{"version":1,"workflow":"w","sensitivity_ceiling":"public","slots":[]}`},
		{"version zero", `{"version":0,"workflow":"w","sensitivity_ceiling":"public","slots":[{"name":"a","kind":"a","requirement":"required","classes":["derived"]}]}`},
		{"duplicate slot", `{"version":1,"workflow":"w","sensitivity_ceiling":"public","slots":[{"name":"a","kind":"a","requirement":"required","classes":["derived"]},{"name":"a","kind":"b","requirement":"optional","classes":["derived"]}]}`},
		{"unknown requirement", `{"version":1,"workflow":"w","sensitivity_ceiling":"public","slots":[{"name":"a","kind":"a","requirement":"mandatory","classes":["derived"]}]}`},
		{"unknown class", `{"version":1,"workflow":"w","sensitivity_ceiling":"public","slots":[{"name":"a","kind":"a","requirement":"required","classes":["tool-output"]}]}`},
		{"no classes", `{"version":1,"workflow":"w","sensitivity_ceiling":"public","slots":[{"name":"a","kind":"a","requirement":"required","classes":[]}]}`},
		{"unknown ceiling", `{"version":1,"workflow":"w","sensitivity_ceiling":"secret","slots":[{"name":"a","kind":"a","requirement":"required","classes":["derived"]}]}`},
		{"required and withheld", `{"version":1,"workflow":"w","sensitivity_ceiling":"public","slots":[{"name":"a","kind":"a","requirement":"required","classes":["derived"],"withhold":true}]}`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "c.json")
			if err := os.WriteFile(path, []byte(tc.body), 0o644); err != nil {
				t.Fatal(err)
			}
			if _, err := LoadContract(path); !errors.Is(err, ErrContractInvalid) {
				t.Fatalf("want ErrContractInvalid, got %v", err)
			}
		})
	}
	// tool-output is transport metadata, never an authority class —
	// pinned above by the "unknown class" case using it.
	if strings.Contains(cveContract, "tool-output") {
		t.Fatal("test contract must not use transport as a class")
	}
}

func TestGatherDeterministic(t *testing.T) {
	c, asg := stdAssignments(t)
	first, err := Gather(c, asg)
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 3; i++ {
		again, err := Gather(c, asg)
		if err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(first.Slots, again.Slots) {
			t.Fatal("gather must be deterministic")
		}
	}
}
