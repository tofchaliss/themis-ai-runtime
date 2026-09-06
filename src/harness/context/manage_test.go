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

// HIGH-A regression: a managed set cannot be re-managed — re-entry
// would erase the omitted_for_capacity marker (silent degradation).
func TestManageReentryRefused(t *testing.T) {
	c := loadTestContract(t, dropContract)
	g := pressureGathered(t, c, 4000, 400)
	pol := mgmtPolicy(t, `{"version":1,"name":"tight","budget":300,"drop_order":["source-files"],"rank_keys":["size_asc"],"dedup":"none"}`)
	managed, _, err := Manage(pol, g)
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := Manage(pol, managed); !errors.Is(err, ErrPolicyInvalid) {
		t.Fatalf("re-managing a managed set must refuse: %v", err)
	}
	// Compose still accepts the managed set.
	set, ipol := eisFixture(t)
	if _, err := Compose(set, ipol, managed); err != nil {
		t.Fatal(err)
	}
}

// HIGH-B: every rank key, a two-key chain, and the hash tiebreak.
func TestRankKeys(t *testing.T) {
	refs := []ItemRef{
		{Kind: "b", Version: "2", Hash: "cc", Size: 10},
		{Kind: "a", Version: "1", Hash: "bb", Size: 30},
		{Kind: "a", Version: "3", Hash: "aa", Size: 30},
	}
	order := func(keys []string) string {
		ranked := rankForDrop(refs, keys)
		var hs []string
		for _, r := range ranked {
			hs = append(hs, r.ref.Hash)
		}
		return strings.Join(hs, ",")
	}
	if got := order([]string{"size_asc"}); got != "cc,aa,bb" { // ties by hash
		t.Fatalf("size_asc: %s", got)
	}
	if got := order([]string{"size_desc"}); got != "aa,bb,cc" {
		t.Fatalf("size_desc: %s", got)
	}
	if got := order([]string{"kind"}); got != "aa,bb,cc" { // a<b; ties by hash
		t.Fatalf("kind: %s", got)
	}
	if got := order([]string{"kind", "size_asc"}); got != "aa,bb,cc" { // kind then size
		t.Fatalf("chain: %s", got)
	}
	// Identical refs: terminal index tiebreak keeps it total + stable.
	same := []ItemRef{{Hash: "xx", Size: 1}, {Hash: "xx", Size: 1}}
	ranked := rankForDrop(same, nil)
	if ranked[0].idx != 0 || ranked[1].idx != 1 {
		t.Fatalf("index tiebreak: %+v", ranked)
	}
}

// MEDIUM-C: bijection under an ACTUAL dedup+drop pass, and the fifth
// state on the fully-dropped deduped slot; MEDIUM-E trace-state
// assertions.
func TestManageDedupDropBijection(t *testing.T) {
	const phantomContract2 = `{
	  "version": 1,
	  "workflow": "phantom2",
	  "sensitivity_ceiling": "internal",
	  "slots": [
	    {"name": "big", "kind": "big", "requirement": "required", "classes": ["external-untrusted"]},
	    {"name": "dups", "kind": "file:*", "requirement": "optional", "classes": ["external-untrusted"], "droppable": true}
	  ]
	}`
	c := loadTestContract(t, phantomContract2)
	root := t.TempDir()
	writeFile(t, root, "a.go", strings.Repeat("x", 4000))
	writeFile(t, root, "b.go", strings.Repeat("x", 4000))
	asg := []Assignment{
		{Slot: "big", Source: inlineSrc(ContextItem{Kind: "big", Evidence: []byte(strings.Repeat("y", 10400))})},
		{Slot: "dups", Source: Source{Name: "workspace", Kind: KindFilesystem, Authority: AuthorityExternalUntrusted,
			Sensitivity: SensitivityPublic, Author: "repository", Root: root, Paths: []string{"a.go", "b.go"}}},
	}
	g, err := Gather(c, asg)
	if err != nil {
		t.Fatal(err)
	}
	// Real usage post-dedup = 2600 + 1000 = 3600; budget 3000 forces a
	// real drop of the deduped item → 2600 fits.
	pol := mgmtPolicy(t, `{"version":1,"name":"b3000","budget":3000,"drop_order":["dups"],"dedup":"within-class"}`)
	managed, trace, err := Manage(pol, g)
	if err != nil {
		t.Fatal(err)
	}
	var dups SlotState
	for _, s := range managed.Slots {
		if s.Slot == "dups" {
			dups = s
		}
	}
	if dups.Availability != AvailabilityOmittedCapacity || dups.Delivery != DeliveryOmittedCapacity {
		t.Fatalf("fully dropped deduped slot must carry the fifth state: %+v", dups)
	}
	if dups.SourceStatus != SourceAvailable {
		t.Fatal("source availability must survive capacity omission (source ≠ delivery)")
	}
	if len(dups.Items) != 0 || len(managed.items["dups"]) != 0 {
		t.Fatal("omitted slot carries no refs or items")
	}
	if trace.BudgetUsed != 2600 {
		t.Fatalf("real accounting must survive dedup+drop: used=%d", trace.BudgetUsed)
	}
	drops := 0
	for _, d := range trace.Decisions {
		if d.Action == ActionDropped {
			drops++
		}
	}
	if drops != 1 {
		t.Fatalf("exactly one real drop expected: %d", drops)
	}
}

// MEDIUM-D: multi-source collapse — provenance multiplicity across
// DISTINCT sources, two collapse groups, deterministic ordering incl.
// the Authority tiebreak.
func TestManageDedupMultiSource(t *testing.T) {
	const multiContract = `{
	  "version": 1,
	  "workflow": "multi",
	  "sensitivity_ceiling": "internal",
	  "slots": [
	    {"name": "vex", "kind": "vex", "requirement": "optional", "classes": ["governed-external", "external-untrusted"]}
	  ]
	}`
	c := loadTestContract(t, multiContract)
	same := []byte("VEX: not_affected\n")
	other := []byte("VEX: affected\n")
	hashSame, hashOther := evidenceHash(same), evidenceHash(other)
	mkItem := func(class AuthorityClass, src string, body []byte) ContextItem {
		return ContextItem{Kind: "vex", Authority: class, Hash: evidenceHash(body), Evidence: body,
			Provenance: Provenance{Origin: "external", Source: src, Author: "vendor"}}
	}
	cross := &Gathered{Contract: c, validated: true,
		Slots: []SlotState{{Slot: "vex", Kind: "vex", Requirement: SlotOptional,
			Availability: AvailabilityDelivered, Delivery: DeliveryDelivered, SourceStatus: SourceAvailable,
			Items: []ItemRef{
				{Slot: "vex", Kind: "vex", Authority: AuthorityExternalUntrusted, Hash: hashSame, Size: len(same)},
				{Slot: "vex", Kind: "vex", Authority: AuthorityExternalUntrusted, Hash: hashSame, Size: len(same)},
				{Slot: "vex", Kind: "vex", Authority: AuthorityGovernedExternal, Hash: hashSame, Size: len(same)},
				{Slot: "vex", Kind: "vex", Authority: AuthorityGovernedExternal, Hash: hashSame, Size: len(same)},
				{Slot: "vex", Kind: "vex", Authority: AuthorityExternalUntrusted, Hash: hashOther, Size: len(other)},
				{Slot: "vex", Kind: "vex", Authority: AuthorityExternalUntrusted, Hash: hashOther, Size: len(other)},
			}}},
		items: map[string][]ContextItem{"vex": {
			mkItem(AuthorityExternalUntrusted, "feed-a", same),
			mkItem(AuthorityExternalUntrusted, "feed-b", same),
			mkItem(AuthorityGovernedExternal, "ingest-a", same),
			mkItem(AuthorityGovernedExternal, "ingest-b", same),
			mkItem(AuthorityExternalUntrusted, "feed-a", other),
			mkItem(AuthorityExternalUntrusted, "feed-b", other),
		}}}
	pol := mgmtPolicy(t, `{"version":1,"name":"m","budget":100000,"drop_order":[],"dedup":"within-class"}`)
	managed, trace, err := Manage(pol, cross)
	if err != nil {
		t.Fatal(err)
	}
	// Same bytes: untrusted pair collapses, governed pair collapses,
	// cross-class copies both survive; other bytes collapse separately.
	if len(managed.items["vex"]) != 3 {
		t.Fatalf("want 3 delivered (2 classes of same + 1 of other): %d", len(managed.items["vex"]))
	}
	if len(trace.Collapses) != 3 {
		t.Fatalf("want 3 collapse groups: %+v", trace.Collapses)
	}
	// Deterministic order incl. Authority tiebreak on the same-hash tie.
	for i := 0; i < len(trace.Collapses)-1; i++ {
		a, b := trace.Collapses[i], trace.Collapses[i+1]
		if a.KeptHash == b.KeptHash && a.Authority >= b.Authority {
			t.Fatalf("Authority tiebreak violated: %+v", trace.Collapses)
		}
	}
	// Distinct sources retained per group.
	for _, ref := range trace.Collapses {
		if len(ref.Collapsed) != 2 || ref.Collapsed[0].Source == ref.Collapsed[1].Source {
			t.Fatalf("multiplicity must record distinct suppliers: %+v", ref)
		}
	}
	for i := 0; i < 3; i++ {
		m2, t2, err := Manage(pol, cross)
		if err != nil || !reflect.DeepEqual(t2.Collapses, trace.Collapses) || !reflect.DeepEqual(m2.Slots, managed.Slots) {
			t.Fatal("multi-source collapse must be deterministic")
		}
	}
}

// LOW: estimator pinned exactly; empty drop-order slot under pressure
// is skipped; policy loader edges.
func TestEstimatorAndEdges(t *testing.T) {
	for n, want := range map[int]int{0: 0, 1: 1, 4: 1, 5: 2, 4000: 1000} {
		if got := estimateTokens(n); got != want {
			t.Fatalf("estimateTokens(%d)=%d want %d", n, got, want)
		}
	}
	// Pressure fixture's exact accounting: 100 (400b file) + 5 + 6 = 111.
	c := loadTestContract(t, dropContract)
	g := pressureGathered(t, c, 4000, 400)
	pol := mgmtPolicy(t, `{"version":1,"name":"tight","budget":300,"drop_order":["source-files"],"rank_keys":["size_asc"],"dedup":"none"}`)
	_, trace, err := Manage(pol, g)
	if err != nil {
		t.Fatal(err)
	}
	if trace.BudgetUsed != 111 {
		t.Fatalf("estimator drift: used=%d want 111", trace.BudgetUsed)
	}
	// Unavailable optional slot named in drop order: skipped, no panic.
	c2 := loadTestContract(t, dropContract)
	root := t.TempDir()
	asg := []Assignment{
		{Slot: "finding", Source: themisSrc(map[string]string{"finding": "CVE-2026-12345 OPEN\n"})},
		{Slot: "task-facts", Source: inlineSrc(ContextItem{Kind: "task-facts", Evidence: []byte(strings.Repeat("z", 2000))})},
		{Slot: "source-files", Source: Source{Name: "workspace", Kind: KindFilesystem, Authority: AuthorityExternalUntrusted,
			Sensitivity: SensitivityPublic, Author: "repository", Root: root, Paths: []string{"absent.go"}}},
	}
	g2, err := Gather(c2, asg)
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := Manage(mgmtPolicy(t, `{"version":1,"name":"p","budget":400,"drop_order":["source-files"],"dedup":"none"}`), g2); !errors.Is(err, ErrBudget) {
		t.Fatalf("empty droppable slot cannot save an over-budget set: %v", err)
	}
	// Policy loader edges.
	if _, err := LoadManagementPolicy(filepath.Join(t.TempDir(), "absent.json")); !errors.Is(err, ErrPolicyInvalid) {
		t.Fatal("missing policy file must fail closed")
	}
	for _, body := range []string{
		`{"version":0,"name":"p","budget":10,"drop_order":[],"dedup":"none"}`,
		`{"version":1,"name":"p","budget":10,"drop_order":[""],"dedup":"none"}`,
	} {
		path := filepath.Join(t.TempDir(), "p.json")
		if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
		if _, err := LoadManagementPolicy(path); !errors.Is(err, ErrPolicyInvalid) {
			t.Fatalf("must fail closed: %s", body)
		}
	}
}

// LOW: search — directory symlink refused; oversized files skipped
// deterministically during scan.
func TestSearchScanEdges(t *testing.T) {
	root := t.TempDir()
	outside := t.TempDir()
	writeFile(t, outside, "d/secret.go", "libXYZ secret\n")
	writeFile(t, root, "ok.go", "libXYZ ok\n")
	if err := os.Symlink(filepath.Join(outside, "d"), filepath.Join(root, "dir")); err != nil {
		t.Fatal(err)
	}
	src := Source{Name: "s", Kind: KindSearch, Authority: AuthorityExternalUntrusted,
		Sensitivity: SensitivityPublic, Root: root, Query: "libXYZ"}
	if _, _, err := src.collect(); !errors.Is(err, ErrConfinement) {
		t.Fatalf("directory symlink must refuse: %v", err)
	}
	// Oversized file skipped from scanning; small match still returned.
	root2 := t.TempDir()
	writeFile(t, root2, "big.go", strings.Repeat("libXYZ", (MaxItemBytes/6)+10))
	writeFile(t, root2, "small.go", "libXYZ hit\n")
	src2 := src
	src2.Root = root2
	items, available, err := src2.collect()
	if err != nil || !available || len(items) != 1 || items[0].Kind != "file:small.go" {
		t.Fatalf("oversized files skip, small match survives: %v %v %+v", err, available, items)
	}
}

// Golden managed-payload hash: pins the post-pressure composed record.
const goldenManagedPayloadHash = "e4b4abc9e5add50eac22b5ffab20417c9c5461430f69ea78fd134a99973f281c"

func TestGoldenManagedPayloadHash(t *testing.T) {
	set, ipol := eisFixture(t)
	c := loadTestContract(t, dropContract)
	g := pressureGathered(t, c, 4000, 400)
	pol := mgmtPolicy(t, `{"version":1,"name":"tight","budget":300,"drop_order":["source-files"],"rank_keys":["size_asc"],"dedup":"none"}`)
	managed, _, err := Manage(pol, g)
	if err != nil {
		t.Fatal(err)
	}
	p, err := Compose(set, ipol, managed)
	if err != nil {
		t.Fatal(err)
	}
	if p.PayloadHash != goldenManagedPayloadHash {
		t.Fatalf("managed payload hash drifted:\n got %s\nwant %s", p.PayloadHash, goldenManagedPayloadHash)
	}
}
