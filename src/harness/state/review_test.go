package state

// Regressions from the L6 security, test, and architecture reviews —
// each pins a remediated finding to its exact branch.

import (
	"encoding/json"
	"errors"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tofchaliss/themis/tools"
)

// Security HIGH-1: recovery must never launder a record Verify would
// call corrupt.
func TestRecoveryRefusesLaundering(t *testing.T) {
	r := testRoot(t)
	tr, err := r.CreateTask("t-laund", TaskOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if err := tr.Transition(StatusRunning, "r"); err != nil {
		t.Fatal(err)
	}
	tr.Close()
	// Total stream loss: the manifest claims RUNNING with no record.
	if err := os.Remove(filepath.Join(r.taskDir("t-laund"), "events.log")); err != nil {
		t.Fatal(err)
	}
	if _, err := r.Recover("t-laund"); !errors.Is(err, ErrCorrupt) || !strings.Contains(err.Error(), "recovery refuses") {
		t.Fatalf("stream loss must refuse recovery typed: %v", err)
	}
	if v := r.Verify("t-laund"); v.Verdict != VerdictCorrupt {
		t.Fatalf("stream loss must verify CORRUPT: %+v", v)
	}
	// The stream file must not have been recreated by the refusal.
	if _, err := os.Stat(filepath.Join(r.taskDir("t-laund"), "events.log")); !os.IsNotExist(err) {
		t.Fatal("recovery must never create a stream CreateTask didn't")
	}
}

// Security HIGH-2: the frame cap refuses before any write, typed and
// NON-terminal.
func TestFrameCapRefusesPreWrite(t *testing.T) {
	r := testRoot(t)
	tr, err := r.CreateTask("t-cap", TaskOptions{})
	if err != nil {
		t.Fatal(err)
	}
	huge, _ := json.Marshal(map[string]string{"blob": strings.Repeat("x", maxEventBytes)})
	if _, err := tr.AppendEvent(EvL4Audit, "l4", huge); !errors.Is(err, ErrStream) || !strings.Contains(err.Error(), "frame cap") {
		t.Fatalf("oversized event must refuse typed: %v", err)
	}
	// Nothing was written: the stream is NOT terminal.
	if _, err := tr.AppendEvent(EvL4Audit, "l4", body("small")); err != nil {
		t.Fatalf("pre-write refusal must not terminate the stream: %v", err)
	}
	tr.Close()
}

// Security MED: forged negative EventCount yields a verdict, never a
// verifier panic.
func TestNegativeEventCountIsCorrupt(t *testing.T) {
	r := testRoot(t)
	tr, _ := r.CreateTask("t-neg", TaskOptions{})
	_ = tr.Transition(StatusRunning, "r")
	_ = tr.Transition(StatusCompleted, "done")
	tr.Close()
	dir := r.taskDir("t-neg")
	man, _ := readManifest(dir)
	man.EventCount = -1
	if err := writeManifest(dir, man); err != nil {
		t.Fatal(err)
	}
	if v := r.Verify("t-neg"); v.Verdict != VerdictCorrupt || !strings.Contains(v.Detail, "more events") {
		t.Fatalf("negative count must be CORRUPT, not a panic: %+v", v)
	}
	if scan := r.ScanReachable(); scan.Complete {
		t.Fatal("corrupt count must render the scan incomplete")
	}
}

// Security MED: recovery refuses to race a live writer.
func TestRecoverRefusesLiveWriter(t *testing.T) {
	r := testRoot(t)
	tr, err := r.CreateTask("t-live", TaskOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := r.Recover("t-live"); !errors.Is(err, ErrStream) || !strings.Contains(err.Error(), "live writer") {
		t.Fatalf("recovery must refuse a live writer: %v", err)
	}
	tr.Close()
	out, err := r.Recover("t-live")
	if err != nil || out.Status != StatusFailedPartial {
		t.Fatalf("after Close recovery proceeds: %+v %v", out, err)
	}
}

// Test review HIGH: terminal binding — summary mismatch and count
// overclaim each pinned to their own branch.
func TestTerminalBindingVerified(t *testing.T) {
	r := testRoot(t)
	tr, _ := r.CreateTask("t-bind", TaskOptions{})
	_ = tr.Transition(StatusRunning, "r")
	_ = tr.Transition(StatusCompleted, "done")
	tr.Close()
	dir := r.taskDir("t-bind")
	man, _ := readManifest(dir)
	good := man.StreamSummary
	man.StreamSummary = summaryOver(nil)
	if err := writeManifest(dir, man); err != nil {
		t.Fatal(err)
	}
	if v := r.Verify("t-bind"); v.Verdict != VerdictCorrupt || v.Detail != "stream summary mismatch" {
		t.Fatalf("summary forgery must hit its own branch: %+v", v)
	}
	man.StreamSummary = good
	man.EventCount = 99
	if err := writeManifest(dir, man); err != nil {
		t.Fatal(err)
	}
	if v := r.Verify("t-bind"); v.Verdict != VerdictCorrupt || !strings.Contains(v.Detail, "more events") {
		t.Fatalf("count overclaim must hit its own branch: %+v", v)
	}
}

// Test review HIGH: runtime lifecycle enforcement through the API.
func TestIllegalTransitionsAPI(t *testing.T) {
	r := testRoot(t)
	tr, _ := r.CreateTask("t-ill", TaskOptions{})
	if err := tr.Transition(StatusCompleted, "skip"); !errors.Is(err, ErrLifecycle) || !strings.Contains(err.Error(), "CREATED -> COMPLETED") {
		t.Fatalf("CREATED->COMPLETED must refuse with the edge named: %v", err)
	}
	_ = tr.Transition(StatusRunning, "r")
	_ = tr.Transition(StatusCompleted, "done")
	// Terminal mutation refused.
	if err := tr.Transition(StatusRunning, "resurrect"); !errors.Is(err, ErrLifecycle) {
		t.Fatal("terminal mutation must refuse")
	}
	if err := tr.Transition(StatusFailed, "flip"); !errors.Is(err, ErrLifecycle) {
		t.Fatal("terminal flip must refuse")
	}
	// BindArtifact on a terminal task refused (hermetic; test review).
	id, _ := r.Store().StoreObject(ObjEgressArtifact, []byte("art"))
	if err := tr.BindArtifact(id); !errors.Is(err, ErrLifecycle) {
		t.Fatalf("bind on terminal must refuse: %v", err)
	}
	tr.Close()
}

// Test review HIGH: the FAILED_PARTIAL-without-recovery-event branch.
func TestForgedFailedPartialBranch(t *testing.T) {
	r := testRoot(t)
	tr, _ := r.CreateTask("t-fp", TaskOptions{})
	_ = tr.Transition(StatusRunning, "r")
	tr.Close()
	dir := r.taskDir("t-fp")
	// Append a lifecycle FAILED_PARTIAL WITHOUT a recovery event
	// (in-package forgery through the raw stream).
	parsed, _ := parseStream(filepath.Join(dir, "events.log"))
	st, err := resumeStream(filepath.Join(dir, "events.log"), parsed)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := st.append(EvLifecycle, "forger", lifecycleBody(StatusFailedPartial, "forged"), nil, r.store); err != nil {
		t.Fatal(err)
	}
	st.close()
	reparsed, _ := parseStream(filepath.Join(dir, "events.log"))
	man, _ := readManifest(dir)
	man.Status = StatusFailedPartial
	man.EventCount = int64(len(reparsed.Events))
	man.StreamSummary = reparsed.Summary
	if err := writeManifest(dir, man); err != nil {
		t.Fatal(err)
	}
	if v := r.Verify("t-fp"); v.Verdict != VerdictCorrupt || v.Detail != "FAILED_PARTIAL without a recovery event" {
		t.Fatalf("must hit the dedicated branch: %+v", v)
	}
}

// Test review HIGH: the CreateTask crash window — CREATED event
// committed, manifest never projected; recovery projects attribution
// from the record, inventing nothing.
func TestCreateTaskCrashWindow(t *testing.T) {
	rootDir := filepath.Join(t.TempDir(), "state")
	r, err := OpenRoot(rootDir)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { fault = nil })
	fault = func(p string) error {
		if p == "manifest.pre-write" {
			return errors.New("injected")
		}
		return nil
	}
	opts := TaskOptions{RetryOf: "t-before", GovernedHashes: map[string]string{"grant": "abc"}}
	if _, err := r.CreateTask("t-cw", opts); err == nil {
		t.Fatal("creation must fail at the armed point")
	}
	fault = nil
	r2, err := OpenRoot(rootDir)
	if err != nil {
		t.Fatal(err)
	}
	out, err := r2.Recover("t-cw")
	if err != nil || out.Status != StatusFailedPartial {
		t.Fatalf("create-crash must recover to FAILED_PARTIAL: %+v %v", out, err)
	}
	man, err := r2.ReadManifest("t-cw")
	if err != nil || man.RetryOf != "t-before" || man.GovernedHashes["grant"] != "abc" || man.ConstitutionHash != ConstitutionHash() {
		t.Fatalf("attribution must be projected from the CREATED event, not invented: %+v", man)
	}
	if v := r2.Verify("t-cw"); v.Verdict != VerdictVerified {
		t.Fatalf("recovered creation must verify: %+v", v)
	}
}

// Test review: lifecycle sequence legality — a record that does not
// begin at CREATED is corrupt.
func TestLifecycleSequenceLegality(t *testing.T) {
	r := testRoot(t)
	dir := r.taskDir("t-seq")
	if err := os.Mkdir(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	st, err := openStream(filepath.Join(dir, "events.log"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := st.append(EvLifecycle, "forger", lifecycleBody(StatusRunning, "no-created"), nil, r.store); err != nil {
		t.Fatal(err)
	}
	st.close()
	parsed, _ := parseStream(filepath.Join(dir, "events.log"))
	if err := writeManifest(dir, &Manifest{TaskID: "t-seq", Status: StatusRunning, ConstitutionHash: ConstitutionHash(),
		EventCount: int64(len(parsed.Events)), StreamSummary: parsed.Summary}); err != nil {
		t.Fatal(err)
	}
	if v := r.Verify("t-seq"); v.Verdict != VerdictCorrupt || !strings.Contains(v.Detail, "does not begin at CREATED") {
		t.Fatalf("sequence must begin at CREATED: %+v", v)
	}
}

// Architecture 3A: the manifest's artifact list is a re-derived
// projection; attribution diverging from the CREATED event is corrupt.
func TestProjectionReDerivation(t *testing.T) {
	r := testRoot(t)
	tr, _ := r.CreateTask("t-proj", TaskOptions{GovernedHashes: map[string]string{"g": "1"}})
	art, _ := tr.StoreObject(ObjEgressArtifact, []byte("artifact"))
	if err := tr.BindArtifact(art); err != nil {
		t.Fatal(err)
	}
	_ = tr.Transition(StatusRunning, "r")
	_ = tr.Transition(StatusCompleted, "done")
	tr.Close()
	if v := r.Verify("t-proj"); v.Verdict != VerdictVerified {
		t.Fatalf("bound artifact must verify: %+v", v)
	}
	dir := r.taskDir("t-proj")
	// Artifact address added to the manifest with NO artifact-bound
	// event: projection divergence.
	man, _ := readManifest(dir)
	ghost, _ := r.Store().StoreObject(ObjEgressArtifact, []byte("ghost"))
	man.ArtifactAddrs = append(man.ArtifactAddrs, ghost)
	if err := writeManifest(dir, man); err != nil {
		t.Fatal(err)
	}
	if v := r.Verify("t-proj"); v.Verdict != VerdictCorrupt || !strings.Contains(v.Detail, "artifact projection") {
		t.Fatalf("unbacked artifact address must be corrupt: %+v", v)
	}
	// Governed-hash divergence from the CREATED event.
	man, _ = readManifest(dir)
	man.ArtifactAddrs = man.ArtifactAddrs[:1]
	man.GovernedHashes = map[string]string{"g": "tampered"}
	if err := writeManifest(dir, man); err != nil {
		t.Fatal(err)
	}
	if v := r.Verify("t-proj"); v.Verdict != VerdictCorrupt || !strings.Contains(v.Detail, "governed hashes diverge") {
		t.Fatalf("attribution divergence must be corrupt: %+v", v)
	}
}

// Security MED-LOW: BindArtifact enforces the reference rule at its
// door; EvArtifact is primitive-owned.
func TestBindArtifactDoor(t *testing.T) {
	r := testRoot(t)
	tr, _ := r.CreateTask("t-bindd", TaskOptions{})
	defer tr.Close()
	if err := tr.BindArtifact("garbage-address"); !errors.Is(err, ErrStream) || !strings.Contains(err.Error(), "already-durable") {
		t.Fatalf("garbage artifact address must refuse: %v", err)
	}
	ghost := objectID([]byte("never-stored-artifact"))
	if err := tr.BindArtifact(ghost); !errors.Is(err, ErrStream) {
		t.Fatal("absent artifact address must refuse")
	}
	if _, err := tr.AppendEvent(EvArtifact, "forger", body("x")); !errors.Is(err, ErrConstitution) {
		t.Fatalf("artifact-bound events are primitive-owned: %v", err)
	}
}

// Security INFO: junk between the JSON value and the frame terminator
// is corruption, not a crash artifact.
func TestJunkInsideFrameIsCorrupt(t *testing.T) {
	r := testRoot(t)
	tr, _ := r.CreateTask("t-junk", TaskOptions{})
	_ = tr.Transition(StatusRunning, "r")
	tr.Close()
	stream := filepath.Join(r.taskDir("t-junk"), "events.log")
	entry := `{"seq":2,"class":"l4-audit","writer":"x","ts":"t","body":{},"body_hash":"h"}JUNK`
	f, _ := os.OpenFile(stream, os.O_APPEND|os.O_WRONLY, 0o600)
	f.WriteString(len2(entry) + "\n" + entry + "\n")
	f.Close()
	if v := r.Verify("t-junk"); v.Verdict != VerdictCorrupt || !strings.Contains(v.Detail, "trailing bytes inside its frame") {
		t.Fatalf("in-frame junk must be corrupt: %+v", v)
	}
}

func len2(s string) string {
	b, _ := json.Marshal(len(s))
	return string(b)
}

// Register A (test review MED): the exported API surface is closed —
// every exported identifier is on the allowlist.
func TestExportedAPIClosure(t *testing.T) {
	allow := map[string]bool{
		// Types and their fields are shape, not verbs; functions and
		// methods are the mutation/read surface.
		"OpenRoot": true, "CheckDisjointRoots": true, "ConstitutionHash": true,
		// ValidTaskID exposes the single task-identity predicate so a
		// caller building paths from a task id (L9 instantiation)
		// enforces THIS rule rather than a twin that can drift. Read-
		// only judgment; it grants nothing and mutates nothing.
		"ValidTaskID": true,
		"Root":        true, "TaskRecord": true, "TaskOptions": true, "ObjectStore": true,
		"Event": true, "Ref": true, "Manifest": true, "StatusView": true,
		"VerifyResult": true, "ScanResult": true, "RecoveryOutcome": true,
		"TaskStatus": true, "Verdict": true,
	}
	allowMethods := map[string]bool{
		"Root.CreateTask": true, "Root.Store": true, "Root.Recover": true,
		"Root.Verify": true, "Root.ScanReachable": true, "Root.ReadStatus": true,
		"Root.ReadEvents": true, "Root.Resolve": true, "Root.ReadManifest": true,
		"TaskRecord.AppendEvent": true, "TaskRecord.StoreObject": true,
		"TaskRecord.Transition": true, "TaskRecord.BindArtifact": true,
		"TaskRecord.Close":        true,
		"ObjectStore.StoreObject": true, "ObjectStore.GetObject": true, "ObjectStore.HasObject": true,
	}
	fset := token.NewFileSet()
	pkgs, err := parser.ParseDir(fset, ".", func(fi os.FileInfo) bool {
		return !strings.HasSuffix(fi.Name(), "_test.go")
	}, 0)
	if err != nil {
		t.Fatal(err)
	}
	for _, pkg := range pkgs {
		for _, file := range pkg.Files {
			for _, decl := range file.Decls {
				switch d := decl.(type) {
				case *ast.FuncDecl:
					if !d.Name.IsExported() {
						continue
					}
					name := d.Name.Name
					if d.Recv != nil && len(d.Recv.List) == 1 {
						recv := ""
						switch rt := d.Recv.List[0].Type.(type) {
						case *ast.StarExpr:
							recv = rt.X.(*ast.Ident).Name
						case *ast.Ident:
							recv = rt.Name
						}
						if !allowMethods[recv+"."+name] {
							t.Errorf("unlisted exported method %s.%s — the write/read surface is closed", recv, name)
						}
					} else if !allow[name] {
						t.Errorf("unlisted exported function %s — the surface is closed", name)
					}
				case *ast.GenDecl:
					for _, spec := range d.Specs {
						switch sp := spec.(type) {
						case *ast.TypeSpec:
							if sp.Name.IsExported() && !allow[sp.Name.Name] {
								t.Errorf("unlisted exported type %s", sp.Name.Name)
							}
						case *ast.ValueSpec:
							for _, n := range sp.Names {
								if n.IsExported() && !strings.HasPrefix(n.Name, "Err") &&
									!strings.HasPrefix(n.Name, "Status") && !strings.HasPrefix(n.Name, "Verdict") &&
									!strings.HasPrefix(n.Name, "Obj") && !strings.HasPrefix(n.Name, "Ev") {
									t.Errorf("unlisted exported value %s", n.Name)
								}
							}
						}
					}
				}
			}
		}
	}
}

// Register A (test review MED): no state-plane verb in any shipped
// registry — decoded scan, not a grep.
func TestNoStateVerbInRegistries(t *testing.T) {
	known := map[string]bool{
		"read_file": true, "list_directory": true, "search_code": true,
		"get_finding": true, "get_product": true, "write_file": true, "apply_patch": true,
	}
	for _, reg := range []string{"registry-v1.json", "registry-v2.json"} {
		r, err := tools.LoadRegistry(filepath.Join("..", "..", "..", "policies", "tools", reg))
		if err != nil {
			t.Fatal(err)
		}
		for _, td := range r.Tools {
			if !known[td.Name] {
				t.Errorf("%s: unknown tool %q — a new capability must be reviewed against the no-model-facing-L6 rule", reg, td.Name)
			}
		}
	}
}

// Register C (test review MED): the sweep list and the source's
// fault-point literals must never drift.
func TestFaultPointsSyncWithSource(t *testing.T) {
	inSource := map[string]bool{}
	entries, _ := os.ReadDir(".")
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".go") || strings.HasSuffix(e.Name(), "_test.go") {
			continue
		}
		b, _ := os.ReadFile(e.Name())
		for _, part := range strings.Split(string(b), `faultAt("`)[1:] {
			if i := strings.IndexByte(part, '"'); i > 0 {
				inSource[part[:i]] = true
			}
		}
	}
	swept := map[string]bool{}
	for _, p := range faultPoints {
		swept[p] = true
	}
	// Recovery's crash windows are swept by their dedicated test.
	swept["recovery.pre-truncate"] = true
	swept["recovery.post-truncate"] = true
	for p := range inSource {
		if !swept[p] {
			t.Errorf("fault point %q exists in source but is never swept", p)
		}
	}
	for p := range swept {
		if !inSource[p] {
			t.Errorf("swept point %q does not exist in source", p)
		}
	}
}

// Recovery's own crash windows: a fault between preserve and
// truncate, or after truncate, leaves a state a second recovery
// completes idempotently (security review LOW).
func TestRecoveryCrashWindows(t *testing.T) {
	for _, point := range []string{"recovery.pre-truncate", "recovery.post-truncate"} {
		point := point
		t.Run(point, func(t *testing.T) {
			r := testRoot(t)
			tr, _ := r.CreateTask("t-rw", TaskOptions{})
			_ = tr.Transition(StatusRunning, "r")
			tr.Close()
			stream := filepath.Join(r.taskDir("t-rw"), "events.log")
			f, _ := os.OpenFile(stream, os.O_APPEND|os.O_WRONLY, 0o600)
			f.WriteString("999\n{\"torn")
			f.Close()
			t.Cleanup(func() { fault = nil })
			fault = func(p string) error {
				if p == point {
					return errors.New("injected")
				}
				return nil
			}
			if _, err := r.Recover("t-rw"); err == nil {
				t.Fatal("armed recovery must fail")
			}
			fault = nil
			out, err := r.Recover("t-rw")
			if err != nil || out.Status != StatusFailedPartial {
				t.Fatalf("second recovery must complete: %+v %v", out, err)
			}
			// Exactly one preserved tail; the record verifies.
			matches, _ := filepath.Glob(filepath.Join(r.taskDir("t-rw"), "events.torn.*"))
			if len(matches) != 1 {
				t.Fatalf("exactly one preserved tail expected: %v", matches)
			}
		})
	}
}
