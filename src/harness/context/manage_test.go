package context

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

// Contract with a droppable optional slot for pressure scenarios.
const dropContract = `{
  "version": 1,
  "workflow": "cve-analysis-pressure",
  "sensitivity_ceiling": "internal",
  "slots": [
    {"name": "finding", "kind": "finding", "requirement": "required", "classes": ["governed-record"]},
    {"name": "task-facts", "kind": "task-facts", "requirement": "required", "classes": ["external-untrusted"]},
    {"name": "source-files", "kind": "file:*", "requirement": "optional", "classes": ["external-untrusted"], "droppable": true}
  ]
}`

func mgmtPolicy(t *testing.T, body string) *ManagementPolicy {
	t.Helper()
	path := filepath.Join(t.TempDir(), "mp.json")
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	p, err := LoadManagementPolicy(path)
	if err != nil {
		t.Fatal(err)
	}
	return p
}

func pressureGathered(t *testing.T, c *Contract, fileSizes ...int) *Gathered {
	t.Helper()
	root := t.TempDir()
	var paths []string
	for i, n := range fileSizes {
		name := string(rune('a'+i)) + ".go"
		writeFile(t, root, name, strings.Repeat("x", n))
		paths = append(paths, name)
	}
	asg := []Assignment{
		{Slot: "finding", Source: themisSrc(map[string]string{"finding": "CVE-2026-12345 OPEN\n"})},
		{Slot: "task-facts", Source: inlineSrc(ContextItem{Kind: "task-facts", Evidence: []byte("component libXYZ 1.4.2\n")})},
		{Slot: "source-files", Source: Source{Name: "workspace", Kind: KindFilesystem, Authority: AuthorityExternalUntrusted,
			Sensitivity: SensitivityPublic, Author: "repository", Root: root, Paths: paths}},
	}
	g, err := Gather(c, asg)
	if err != nil {
		t.Fatal(err)
	}
	return g
}

func TestManagementPolicyFailsClosed(t *testing.T) {
	cases := []struct {
		name string
		body string
	}{
		{"invalid json", `{`},
		{"unknown field", `{"version":1,"name":"p","budget":10,"drop_order":[],"dedup":"none","extra":1}`},
		{"trailing", `{"version":1,"name":"p","budget":10,"drop_order":[],"dedup":"none"} X`},
		{"zero budget", `{"version":1,"name":"p","budget":0,"drop_order":[],"dedup":"none"}`},
		{"no name", `{"version":1,"name":"","budget":10,"drop_order":[],"dedup":"none"}`},
		{"bad dedup", `{"version":1,"name":"p","budget":10,"drop_order":[],"dedup":"semantic"}`},
		{"duplicate drop slot", `{"version":1,"name":"p","budget":10,"drop_order":["a","a"],"dedup":"none"}`},
		{"illegal rank key", `{"version":1,"name":"p","budget":10,"drop_order":[],"rank_keys":["relevance"],"dedup":"none"}`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "p.json")
			if err := os.WriteFile(path, []byte(tc.body), 0o644); err != nil {
				t.Fatal(err)
			}
			if _, err := LoadManagementPolicy(path); !errors.Is(err, ErrPolicyInvalid) {
				t.Fatalf("want ErrPolicyInvalid, got %v", err)
			}
		})
	}
}

// No pressure ⇒ byte-identical passthrough: L3 does not quietly
// transform context when capacity isn't binding (Q-L3-9).
func TestManageNoPressurePassthrough(t *testing.T) {
	c := loadTestContract(t, dropContract)
	g := pressureGathered(t, c, 100, 100)
	pol := mgmtPolicy(t, `{"version":1,"name":"roomy","budget":100000,"drop_order":["source-files"],"dedup":"none"}`)
	managed, trace, err := Manage(pol, g)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(managed.items, g.items) {
		t.Fatal("no-pressure passthrough must keep every item byte-identical")
	}
	for i, s := range managed.Slots {
		if s.Omitted || s.Availability != g.Slots[i].Availability {
			t.Fatalf("no-pressure slot state changed: %+v", s)
		}
	}
	if trace.PolicyHash != pol.Hash || trace.Estimator != estimatorVersion || trace.BudgetUsed <= 0 {
		t.Fatalf("trace incomplete: %+v", trace)
	}
	// Compose after passthrough: identical to composing the original.
	set, ipol := eisFixture(t)
	p1, err := Compose(set, ipol, g)
	if err != nil {
		t.Fatal(err)
	}
	p2, err := Compose(set, ipol, managed)
	if err != nil {
		t.Fatal(err)
	}
	if p1.Messages[1].Content != p2.Messages[1].Content {
		t.Fatal("passthrough must not change the composed view")
	}
}

// Pressure drops in policy order, worst-ranked first, and the model
// sees the typed state-only marker.
func TestManagePressureDropsAndMarker(t *testing.T) {
	c := loadTestContract(t, dropContract)
	// Two files: 4000 and 400 bytes → 1000 + 100 tokens; finding+facts ≈ 11 tokens.
	g := pressureGathered(t, c, 4000, 400)
	pol := mgmtPolicy(t, `{"version":1,"name":"tight","budget":300,"drop_order":["source-files"],"rank_keys":["size_asc"],"dedup":"none"}`)
	managed, trace, err := Manage(pol, g)
	if err != nil {
		t.Fatal(err)
	}
	// size_asc ranking: the big file is worst-ranked, dropped first;
	// the small file fits within 300 tokens.
	var files SlotState
	for _, s := range managed.Slots {
		if s.Slot == "source-files" {
			files = s
		}
	}
	if !files.Omitted || files.Availability != AvailabilityDelivered || len(files.Items) != 1 {
		t.Fatalf("partial omission state wrong: %+v", files)
	}
	if files.Items[0].Size != 400 {
		t.Fatalf("size_asc must drop the large item first: %+v", files.Items)
	}
	dropped := 0
	for _, d := range trace.Decisions {
		if d.Action == ActionDropped {
			dropped++
			if d.Slot != "source-files" {
				t.Fatalf("only the droppable slot may lose items: %+v", d)
			}
		}
	}
	if dropped != 1 {
		t.Fatalf("want exactly one drop, got %d", dropped)
	}
	// Model-visible marker: state only, no counts.
	set, ipol := eisFixture(t)
	p, err := Compose(set, ipol, managed)
	if err != nil {
		t.Fatal(err)
	}
	user := p.Messages[1].Content
	if !strings.Contains(user, "[slot: source-files | availability: delivered | omitted_for_capacity]") {
		t.Fatal("partial-omission marker missing")
	}
	if strings.Contains(user, "omitted_count") || strings.Contains(user, "1 of 2") {
		t.Fatal("counts must stay trace-only")
	}
}

// A slot losing every item renders as the fifth availability state.
func TestManageFullSlotOmission(t *testing.T) {
	c := loadTestContract(t, dropContract)
	g := pressureGathered(t, c, 4000, 4000)
	pol := mgmtPolicy(t, `{"version":1,"name":"tiny","budget":50,"drop_order":["source-files"],"dedup":"none"}`)
	managed, _, err := Manage(pol, g)
	if err != nil {
		t.Fatal(err)
	}
	set, ipol := eisFixture(t)
	p, err := Compose(set, ipol, managed)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(p.Messages[1].Content, "[slot: source-files | availability: omitted_for_capacity]") {
		t.Fatal("fully omitted slot must render the fifth state")
	}
}

// Undroppable content exceeding the budget fails closed — capacity is
// never authority, and omission never becomes silent degradation.
func TestManageBudgetFailsClosed(t *testing.T) {
	c := loadTestContract(t, dropContract)
	g := pressureGathered(t, c, 400)
	pol := mgmtPolicy(t, `{"version":1,"name":"impossible","budget":3,"drop_order":["source-files"],"dedup":"none"}`)
	managed, trace, err := Manage(pol, g)
	if !errors.Is(err, ErrBudget) || managed != nil || trace != nil {
		t.Fatalf("unmeetable budget must fail closed: %v", err)
	}
}

// Drop order may name only contract-droppable slots: required,
// undroppable-optional, and withheld slots are all rejected.
func TestManageDropOrderContractGate(t *testing.T) {
	c := loadTestContract(t, cveContract) // source-files NOT droppable here
	g := func() *Gathered {
		gg, err := Gather(c, stdAssignmentsFor(t, c))
		if err != nil {
			t.Fatal(err)
		}
		return gg
	}()
	for _, slot := range []string{"finding", "source-files", "enterprise-position", "nonexistent"} {
		pol := mgmtPolicy(t, `{"version":1,"name":"p","budget":100000,"drop_order":["`+slot+`"],"dedup":"none"}`)
		if _, _, err := Manage(pol, g); !errors.Is(err, ErrPolicyInvalid) {
			t.Fatalf("drop-order slot %q must be rejected: %v", slot, err)
		}
	}
	// Nil/unvalidated inputs refuse.
	pol := mgmtPolicy(t, `{"version":1,"name":"p","budget":100000,"drop_order":[],"dedup":"none"}`)
	if _, _, err := Manage(nil, g); !errors.Is(err, ErrPolicyInvalid) {
		t.Fatal("nil policy must refuse")
	}
	if _, _, err := Manage(pol, &Gathered{Contract: c}); !errors.Is(err, ErrPolicyInvalid) {
		t.Fatal("unvalidated gathered must refuse")
	}
}

// Contract droppability validation: required+droppable and
// withheld+droppable are contradictions.
func TestContractDroppableValidation(t *testing.T) {
	for _, body := range []string{
		`{"version":1,"workflow":"w","sensitivity_ceiling":"public","slots":[{"name":"a","kind":"a","requirement":"required","classes":["derived"],"droppable":true}]}`,
		`{"version":1,"workflow":"w","sensitivity_ceiling":"public","slots":[{"name":"a","kind":"a","requirement":"optional","classes":["derived"],"withhold":true,"droppable":true}]}`,
	} {
		path := filepath.Join(t.TempDir(), "c.json")
		if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
		if _, err := LoadContract(path); !errors.Is(err, ErrContractInvalid) {
			t.Fatalf("contradictory disposition must be rejected: %s", body)
		}
	}
}

// Within-class dedup collapses byte-identical items, retains every
// source ref in the trace, and never collapses across classes.
func TestManageDedup(t *testing.T) {
	const dedupContract = `{
	  "version": 1,
	  "workflow": "dedup",
	  "sensitivity_ceiling": "internal",
	  "slots": [
	    {"name": "vex", "kind": "vex", "requirement": "optional",
	     "classes": ["governed-external", "external-untrusted"]}
	  ]
	}`
	c := loadTestContract(t, dedupContract)
	bytesSame := []byte("VEX: not_affected\n")
	// Two untrusted copies (different callers' inline items collapse)
	// plus the same bytes under a governed class via a second slot is
	// not possible in one slot from one source — use one source with
	// duplicate items for within-class, then verify cross-class
	// separation via direct construction.
	asg := []Assignment{{Slot: "vex", Source: inlineSrc(
		ContextItem{Kind: "vex", Evidence: bytesSame},
		ContextItem{Kind: "vex", Evidence: bytesSame},
	)}}
	g, err := Gather(c, asg)
	if err != nil {
		t.Fatal(err)
	}
	pol := mgmtPolicy(t, `{"version":1,"name":"d","budget":100000,"drop_order":[],"dedup":"within-class"}`)
	managed, trace, err := Manage(pol, g)
	if err != nil {
		t.Fatal(err)
	}
	if len(managed.items["vex"]) != 1 {
		t.Fatalf("within-class byte-identical items must collapse: %d", len(managed.items["vex"]))
	}
	if len(trace.Collapses) != 1 || len(trace.Collapses[0].Collapsed) != 2 {
		t.Fatalf("collapse must retain provenance multiplicity: %+v", trace.Collapses)
	}
	if trace.Collapses[0].Collapsed[0].Kind != "vex" {
		t.Fatalf("collapsed items must retain kind/version identity: %+v", trace.Collapses[0])
	}
	// Cross-class: same bytes, different classes — never collapsed.
	hash := evidenceHash(bytesSame)
	cross := &Gathered{Contract: c, validated: true,
		Slots: []SlotState{{Slot: "vex", Kind: "vex", Requirement: SlotOptional,
			Availability: AvailabilityDelivered, Delivery: DeliveryDelivered, SourceStatus: SourceAvailable,
			Items: []ItemRef{
				{Slot: "vex", Kind: "vex", Authority: AuthorityGovernedExternal, Hash: hash, Size: len(bytesSame)},
				{Slot: "vex", Kind: "vex", Authority: AuthorityExternalUntrusted, Hash: hash, Size: len(bytesSame)},
			}}},
		items: map[string][]ContextItem{"vex": {
			{Kind: "vex", Authority: AuthorityGovernedExternal, Hash: hash, Evidence: bytesSame,
				Provenance: Provenance{Origin: "external", Source: "ingested", Author: "vendor"}},
			{Kind: "vex", Authority: AuthorityExternalUntrusted, Hash: hash, Evidence: bytesSame,
				Provenance: Provenance{Origin: "external", Source: "raw", Author: "vendor"}},
		}}}
	managed2, trace2, err := Manage(pol, cross)
	if err != nil {
		t.Fatal(err)
	}
	if len(managed2.items["vex"]) != 2 || len(trace2.Collapses) != 0 {
		t.Fatal("cross-class byte-identical evidence must never collapse")
	}
}

// Determinism: repeated Manage over the same inputs is deeply equal.
func TestManageDeterministic(t *testing.T) {
	c := loadTestContract(t, dropContract)
	g := pressureGathered(t, c, 4000, 400, 900)
	pol := mgmtPolicy(t, `{"version":1,"name":"t","budget":400,"drop_order":["source-files"],"rank_keys":["size_asc"],"dedup":"within-class"}`)
	m1, t1, err := Manage(pol, g)
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 3; i++ {
		m2, t2, err := Manage(pol, g)
		if err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(m1.Slots, m2.Slots) || !reflect.DeepEqual(t1, t2) {
			t.Fatal("Manage must be deterministic")
		}
	}
}

// M3: the lexical search source — confined, classified
// external-untrusted, deterministic sorted results.
func TestSearchSource(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "b/vuln.go", "package b // uses libXYZ parser\n")
	writeFile(t, root, "a/clean.go", "package a\n")
	writeFile(t, root, "a/hit.go", "package a // libXYZ here too\n")
	writeFile(t, root, "notes.txt", "libXYZ mention\n")
	src := Source{Name: "code-search", Kind: KindSearch, Authority: AuthorityExternalUntrusted,
		Sensitivity: SensitivityPublic, Author: "repository", Root: root, Query: "libXYZ", Suffix: ".go"}
	items, available, err := src.collect()
	if err != nil || !available {
		t.Fatalf("search failed: %v", err)
	}
	if len(items) != 2 || items[0].Kind != "file:a/hit.go" || items[1].Kind != "file:b/vuln.go" {
		t.Fatalf("sorted suffix-filtered matches wrong: %+v", items)
	}
	if items[0].Authority != AuthorityExternalUntrusted {
		t.Fatal("search results are external-untrusted")
	}
	// Class constraint: search cannot mint governed classes.
	bad := src
	bad.Authority = AuthorityGovernedRecord
	if err := checkSource(bad); !errors.Is(err, ErrUnrecognizedSource) {
		t.Fatal("search source cannot declare governed authority")
	}
	// Empty query is a registration error.
	noQuery := src
	noQuery.Query = ""
	if _, _, err := noQuery.collect(); !errors.Is(err, ErrUnrecognizedSource) {
		t.Fatal("search without query must be refused")
	}
	// No matches: typed unavailability, not an error.
	miss := src
	miss.Query = "ZZNOHITZZ"
	_, available, err = miss.collect()
	if err != nil || available {
		t.Fatal("no matches must be typed unavailability")
	}
}

// Security-review HIGH-1 regression: phantom drops. Duplicate bytes +
// within-class dedup + a budget between phantom-adjusted and real
// usage must fail closed, and refs must stay bijective with delivered
// items.
func TestManagePhantomDropBudget(t *testing.T) {
	const phantomContract = `{
	  "version": 1,
	  "workflow": "phantom",
	  "sensitivity_ceiling": "internal",
	  "slots": [
	    {"name": "big", "kind": "big", "requirement": "required", "classes": ["external-untrusted"]},
	    {"name": "dups", "kind": "file:*", "requirement": "optional", "classes": ["external-untrusted"], "droppable": true}
	  ]
	}`
	c := loadTestContract(t, phantomContract)
	root := t.TempDir()
	writeFile(t, root, "a.go", strings.Repeat("x", 4000))
	writeFile(t, root, "b.go", strings.Repeat("x", 4000)) // byte-identical: same hash
	asg := []Assignment{
		{Slot: "big", Source: inlineSrc(ContextItem{Kind: "big", Evidence: []byte(strings.Repeat("y", 10400))})}, // 2600 tokens, undroppable
		{Slot: "dups", Source: Source{Name: "workspace", Kind: KindFilesystem, Authority: AuthorityExternalUntrusted,
			Sensitivity: SensitivityPublic, Author: "repository", Root: root, Paths: []string{"a.go", "b.go"}}},
	}
	g, err := Gather(c, asg)
	if err != nil {
		t.Fatal(err)
	}
	// Dedup collapses the duplicates to ONE 1000-token item; real
	// usage = 2600 + 1000 = 3600; after dropping the single real
	// duplicate: 2600 > 2500 budget. Phantom accounting would have
	// subtracted 2000 and wrongly succeeded at 1600.
	pol := mgmtPolicy(t, `{"version":1,"name":"ph","budget":2500,"drop_order":["dups"],"dedup":"within-class"}`)
	managed, trace, err := Manage(pol, g)
	if !errors.Is(err, ErrBudget) || managed != nil || trace != nil {
		t.Fatalf("phantom drops must not defeat the budget: %v", err)
	}
	// Bijection under dedup+drop: with a feasible budget, every ref in
	// every slot state corresponds to a delivered item.
	pol2 := mgmtPolicy(t, `{"version":1,"name":"ok","budget":3700,"drop_order":["dups"],"dedup":"within-class"}`)
	managed2, _, err := Manage(pol2, g)
	if err != nil {
		t.Fatal(err)
	}
	for _, s := range managed2.Slots {
		if s.Availability != AvailabilityDelivered {
			continue
		}
		if len(s.Items) != len(managed2.items[s.Slot]) {
			t.Fatalf("refs and delivered items diverge in slot %s: %d refs, %d items", s.Slot, len(s.Items), len(managed2.items[s.Slot]))
		}
		for i, ref := range s.Items {
			if ref.Hash != managed2.items[s.Slot][i].Hash {
				t.Fatalf("ref/item hash mismatch in slot %s at %d", s.Slot, i)
			}
		}
	}
}

// Security-review HIGH-2 regression: a file symlink inside the search
// root pointing outside must hard-refuse, never deliver outside
// bytes.
func TestSearchSymlinkEscapeRefused(t *testing.T) {
	root := t.TempDir()
	outside := t.TempDir()
	writeFile(t, outside, "secret.txt", "TOPSECRET libXYZ credentials\n")
	writeFile(t, root, "ok.go", "package ok // libXYZ\n")
	if err := os.Symlink(filepath.Join(outside, "secret.txt"), filepath.Join(root, "leak.go")); err != nil {
		t.Fatal(err)
	}
	src := Source{Name: "code-search", Kind: KindSearch, Authority: AuthorityExternalUntrusted,
		Sensitivity: SensitivityPublic, Author: "repository", Root: root, Query: "libXYZ"}
	items, _, err := src.collect()
	if !errors.Is(err, ErrConfinement) {
		t.Fatalf("symlink in search root must refuse hard: %v", err)
	}
	for _, it := range items {
		if strings.Contains(string(it.Evidence), "TOPSECRET") {
			t.Fatal("outside bytes must never be delivered")
		}
	}
}

// Security-review MEDIUM-1 regression: over-cap match sets refuse
// deterministically — no silent truncation into a complete-looking
// result.
func TestSearchOverCapRefuses(t *testing.T) {
	root := t.TempDir()
	for i := 0; i < 70; i++ {
		writeFile(t, root, fmt.Sprintf("f%02d.go", i), "libXYZ\n")
	}
	src := Source{Name: "code-search", Kind: KindSearch, Authority: AuthorityExternalUntrusted,
		Sensitivity: SensitivityPublic, Author: "repository", Root: root, Query: "libXYZ"}
	if _, _, err := src.collect(); !errors.Is(err, ErrContextTooLarge) {
		t.Fatalf("over-cap search must refuse, not truncate: %v", err)
	}
}
