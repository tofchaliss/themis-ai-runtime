package execution

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

// --- helpers ---------------------------------------------------------

func writeTemp(t *testing.T, name, content string) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return p
}

func gitBin(t *testing.T) string {
	t.Helper()
	for _, p := range []string{"/usr/bin/git", "/opt/homebrew/bin/git"} {
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	t.Skip("no pinned git binary on this host")
	return ""
}

// mkMirror creates a local mirror repository with one sentinel commit
// and returns (mirrorRoot, repoName, headSHA).
func mkMirror(t *testing.T) (string, string, string) {
	t.Helper()
	git := gitBin(t)
	root := t.TempDir()
	repo := filepath.Join(root, "themis-demo")
	run := func(dir string, args ...string) string {
		t.Helper()
		cmd := exec.Command(git, append([]string{"-c", "user.name=t", "-c", "user.email=t@t"}, args...)...)
		cmd.Dir = dir
		cmd.Env = append(os.Environ(), "GIT_CONFIG_GLOBAL=/dev/null", "GIT_CONFIG_SYSTEM=/dev/null")
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("git %v: %v: %s", args, err, out)
		}
		return string(out)
	}
	if err := os.Mkdir(repo, 0o755); err != nil {
		t.Fatal(err)
	}
	run(repo, "init", "-q")
	if err := os.WriteFile(filepath.Join(repo, "parser.go"), []byte("package parser // SENTINEL-L5\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(repo, "README.md"), []byte("# demo\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	run(repo, "add", ".")
	run(repo, "commit", "-q", "-m", "seed")
	sha := strings.TrimSpace(run(repo, "rev-parse", "HEAD"))
	return root, "themis-demo", sha
}

const ceilingBounds = `"max_wall_deadline_sec":120,"max_file_bytes":1048576,"max_total_bytes":10485760,"max_file_count":500,"max_mem_bytes":1073741824,"max_cpu_time_sec":600,"max_proc_count":64`

func testCeiling(t *testing.T, mirrorRoot string) *WorkspaceExecutionCeiling {
	t.Helper()
	body := `{"version":1,"mirror_root":"` + mirrorRoot + `",` + ceilingBounds + `}`
	c, err := LoadCeiling(writeTemp(t, "ceiling.json", body))
	if err != nil {
		t.Fatal(err)
	}
	return c
}

func testSpec(t *testing.T, repo, sha, extraLimits string) *ProvisionSpec {
	t.Helper()
	limits := `{"dimension":"wall_deadline_s","value":60}`
	if extraLimits != "" {
		limits += "," + extraLimits
	}
	body := `{"version":1,"task_id":"T-L5","repo":"` + repo + `","pinned_sha":"` + sha + `","limits":[` + limits + `]}`
	s, err := parseSpec([]byte(body), "test")
	if err != nil {
		t.Fatal(err)
	}
	return s
}

// --- governed artifacts: fail closed, branch-asserted ---------------

func TestCeilingFailsClosed(t *testing.T) {
	cases := []struct{ name, body, wantErr string }{
		{"unknown-field", `{"version":1,"mirror_root":"/m",` + ceilingBounds + `,"extra":true}`, "unknown field"},
		{"trailing", `{"version":1,"mirror_root":"/m",` + ceilingBounds + `}{}`, "trailing content"},
		{"no-version", `{"mirror_root":"/m",` + ceilingBounds + `}`, "version required"},
		{"relative-root", `{"version":1,"mirror_root":"mirrors",` + ceilingBounds + `}`, "must be absolute"},
		{"zero-bound", `{"version":1,"mirror_root":"/m","max_wall_deadline_sec":0,"max_file_bytes":1,"max_total_bytes":1,"max_file_count":1,"max_mem_bytes":1,"max_cpu_time_sec":1,"max_proc_count":1}`, "must be positive"},
		{"missing-new-bound", `{"version":1,"mirror_root":"/m","max_wall_deadline_sec":9,"max_file_bytes":1,"max_total_bytes":1,"max_file_count":1}`, "must be positive"},
	}
	for _, c := range cases {
		_, err := LoadCeiling(writeTemp(t, c.name+".json", c.body))
		if err == nil || !errors.Is(err, ErrCeilingInvalid) || !strings.Contains(err.Error(), c.wantErr) {
			t.Errorf("%s: want %q in ErrCeilingInvalid, got %v", c.name, c.wantErr, err)
		}
	}
	if _, err := LoadCeiling(filepath.Join(t.TempDir(), "absent.json")); !errors.Is(err, ErrCeilingInvalid) {
		t.Errorf("missing file must fail closed: %v", err)
	}
}

func TestSpecFailsClosed(t *testing.T) {
	sha := strings.Repeat("ab", 20)
	cases := []struct{ name, body, wantErr string }{
		{"branch-not-pin", `{"version":1,"task_id":"T","repo":"r","pinned_sha":"main","limits":[{"dimension":"wall_deadline_s","value":5}]}`, "branch names are not pins"},
		{"short-sha", `{"version":1,"task_id":"T","repo":"r","pinned_sha":"abc123","limits":[{"dimension":"wall_deadline_s","value":5}]}`, "40-hex"},
		{"url-repo", `{"version":1,"task_id":"T","repo":"https://evil/x","pinned_sha":"` + sha + `","limits":[{"dimension":"wall_deadline_s","value":5}]}`, "bad repository name"},
		{"unknown-dim", `{"version":1,"task_id":"T","repo":"r","pinned_sha":"` + sha + `","limits":[{"dimension":"cpu","value":5},{"dimension":"wall_deadline_s","value":5}]}`, "unknown limit dimension"},
		{"dup-dim", `{"version":1,"task_id":"T","repo":"r","pinned_sha":"` + sha + `","limits":[{"dimension":"wall_deadline_s","value":5},{"dimension":"wall_deadline_s","value":6}]}`, "duplicate limit dimension"},
		{"zero-value", `{"version":1,"task_id":"T","repo":"r","pinned_sha":"` + sha + `","limits":[{"dimension":"wall_deadline_s","value":0}]}`, "must be positive"},
		{"bad-strength", `{"version":1,"task_id":"T","repo":"r","pinned_sha":"` + sha + `","limits":[{"dimension":"wall_deadline_s","value":5},{"dimension":"mem_bytes","value":5,"strength":"supported"}]}`, "unknown strength"},
		{"no-deadline", `{"version":1,"task_id":"T","repo":"r","pinned_sha":"` + sha + `","limits":[{"dimension":"mem_bytes","value":5,"strength":"observed"}]}`, "wall_deadline_s is mandatory"},
		{"observed-deadline", `{"version":1,"task_id":"T","repo":"r","pinned_sha":"` + sha + `","limits":[{"dimension":"wall_deadline_s","value":5,"strength":"observed"}]}`, "enforced strength"},
		{"unknown-field", `{"version":1,"task_id":"T","repo":"r","pinned_sha":"` + sha + `","network":"all","limits":[{"dimension":"wall_deadline_s","value":5}]}`, "unknown field"},
		{"dotdot-repo", `{"version":1,"task_id":"T","repo":"a/../../x","pinned_sha":"` + sha + `","limits":[{"dimension":"wall_deadline_s","value":5}]}`, "traversal segment in repository name"},
	}
	for _, c := range cases {
		_, err := parseSpec([]byte(c.body), c.name)
		if err == nil || !errors.Is(err, ErrSpecInvalid) || !strings.Contains(err.Error(), c.wantErr) {
			t.Errorf("%s: want %q in ErrSpecInvalid, got %v", c.name, c.wantErr, err)
		}
	}
}

func TestSpecCeilingContainment(t *testing.T) {
	c := testCeiling(t, "/m")
	sha := strings.Repeat("ab", 20)
	s := testSpec(t, "r", sha, "")
	s.Limits[0].Value = 999 // > ceiling 120
	if err := s.ValidateAgainst(c); err == nil || !strings.Contains(err.Error(), "exceeds ceiling") {
		t.Fatalf("spec above ceiling must refuse: %v", err)
	}
	s.Limits[0].Value = 60
	if err := s.ValidateAgainst(c); err != nil {
		t.Fatalf("contained spec must pass: %v", err)
	}
	// file_bytes and disk_bytes narrow, never widen.
	s2 := testSpec(t, "r", sha, `{"dimension":"file_bytes","value":99999999,"strength":"observed"}`)
	if err := s2.ValidateAgainst(c); err == nil || !strings.Contains(err.Error(), "file_bytes") {
		t.Fatalf("file_bytes above ceiling must refuse: %v", err)
	}
	s3 := testSpec(t, "r", sha, `{"dimension":"disk_bytes","value":99999999999,"strength":"observed"}`)
	if err := s3.ValidateAgainst(c); err == nil || !strings.Contains(err.Error(), "disk_bytes") {
		t.Fatalf("disk_bytes above ceiling must refuse: %v", err)
	}
	// Containment is total over the closed vocabulary: mem_bytes too.
	s4 := testSpec(t, "r", sha, `{"dimension":"mem_bytes","value":9999999999999,"strength":"observed"}`)
	if err := s4.ValidateAgainst(c); err == nil || !strings.Contains(err.Error(), "mem_bytes") {
		t.Fatalf("mem_bytes above ceiling must refuse: %v", err)
	}
}

func TestLoadSpecFromFile(t *testing.T) {
	sha := strings.Repeat("ab", 20)
	body := `{"version":1,"task_id":"T","repo":"r","pinned_sha":"` + sha + `","limits":[{"dimension":"wall_deadline_s","value":5}]}`
	s, err := LoadSpec(writeTemp(t, "spec.json", body))
	if err != nil || s.Hash == "" || s.Limits[0].Strength != StrengthEnforced {
		t.Fatalf("file spec must load with hash and default-enforced strength: %+v %v", s, err)
	}
	if _, err := LoadSpec(filepath.Join(t.TempDir(), "absent.json")); !errors.Is(err, ErrSpecInvalid) {
		t.Fatalf("missing spec file must fail closed: %v", err)
	}
}

// --- admission: observation is not enforcement ----------------------

func TestAdmissionFailsClosed(t *testing.T) {
	decl := LocalDeclaration()
	sha := strings.Repeat("ab", 20)

	// enforced mem_bytes required; local only observes -> refuse.
	s := testSpec(t, "r", sha, `{"dimension":"mem_bytes","value":1048576,"strength":"enforced"}`)
	if err := Admit(decl, s); err == nil || !errors.Is(err, ErrAdmission) || !strings.Contains(err.Error(), "observation is not enforcement") {
		t.Fatalf("observed provider must not satisfy enforced requirement: %v", err)
	}
	// dimension the provider makes no claim about -> refuse.
	s = testSpec(t, "r", sha, `{"dimension":"proc_count","value":4,"strength":"observed"}`)
	if err := Admit(decl, s); err == nil || !strings.Contains(err.Error(), "makes no claim") {
		t.Fatalf("unclaimed dimension must refuse: %v", err)
	}
	// observed-strength requirement satisfied by observed declaration.
	s = testSpec(t, "r", sha, `{"dimension":"mem_bytes","value":1048576,"strength":"observed"}`)
	if err := Admit(decl, s); err != nil {
		t.Fatalf("observed requirement vs observed declaration must admit: %v", err)
	}
	// default-enforced deadline vs enforced declaration.
	s = testSpec(t, "r", sha, "")
	if err := Admit(decl, s); err != nil {
		t.Fatalf("enforced deadline must admit: %v", err)
	}
	// the local declaration never claims cpu_time_s or proc_count.
	if _, ok := decl.Limits[DimCPUTimeS]; ok {
		t.Fatal("local provider must not claim cpu_time_s without an enforcement mechanism")
	}
	if _, ok := decl.Limits[DimProcCount]; ok {
		t.Fatal("proc_count is deliberately absent under inherited identity")
	}
	if decl.ProcessIdentity != IdentityInherited {
		t.Fatal("local provider must declare inherited identity honestly")
	}
}

func TestProviderConstructionFailsClosed(t *testing.T) {
	if _, err := NewLocalProvider("relative/git", "/abs"); err == nil || !strings.Contains(err.Error(), "absolute") {
		t.Fatalf("relative git path must refuse: %v", err)
	}
	if _, err := NewLocalProvider("/usr/bin/git", "rel"); err == nil || !strings.Contains(err.Error(), "absolute") {
		t.Fatalf("relative base dir must refuse: %v", err)
	}
	p, err := NewLocalProvider(gitBin(t), t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if p.Declaration().Name != "local-v1" {
		t.Fatalf("declaration must be exposed: %+v", p.Declaration())
	}
}

// --- lifecycle: invariants by reachability --------------------------

func TestLifecycleReachability(t *testing.T) {
	// No ACTIVE->EGRESSING edge; no ACTIVE->TEARDOWN without seal.
	e := &Env{witness: &recWitness{}, state: StateActive}
	if err := e.transition(StateEgressing, "x"); !errors.Is(err, ErrLifecycle) {
		t.Fatalf("ACTIVE->EGRESSING must be illegal: %v", err)
	}
	if err := e.transition(StateTeardown, "x"); !errors.Is(err, ErrLifecycle) {
		t.Fatalf("ACTIVE->TEARDOWN must be illegal without seal: %v", err)
	}
	// Unknown seal reason refused.
	if err := e.Seal("panic"); !errors.Is(err, ErrLifecycle) {
		t.Fatalf("unknown seal reason must refuse: %v", err)
	}
	// Non-clean seal cannot egress.
	if err := e.Seal(SealFatalBreach); err != nil {
		t.Fatal(err)
	}
	if err := e.beginEgress(); !errors.Is(err, ErrLifecycle) || !strings.Contains(err.Error(), "cleanly sealed") {
		t.Fatalf("non-clean seal must not egress: %v", err)
	}
	// Double seal refused (no SEALED->SEALED edge).
	if err := e.Seal(SealTaskComplete); !errors.Is(err, ErrLifecycle) {
		t.Fatalf("double seal must refuse: %v", err)
	}
	// Clean seal egresses; EGRESSING cannot return to ACTIVE.
	e2 := &Env{witness: &recWitness{}, state: StateActive}
	if e2.CleanlySealed() {
		t.Fatal("ACTIVE is not cleanly sealed")
	}
	if err := e2.Seal(SealTaskComplete); err != nil {
		t.Fatal(err)
	}
	if !e2.CleanlySealed() {
		t.Fatal("task-complete seal must report clean")
	}
	if err := e2.beginEgress(); err != nil {
		t.Fatalf("clean seal must egress: %v", err)
	}
	if err := e2.transition(StateActive, "x"); !errors.Is(err, ErrLifecycle) {
		t.Fatalf("no backward transitions: %v", err)
	}
	// ACKNOWLEDGED without EGRESSING is unreachable.
	e3 := &Env{witness: &recWitness{}, state: StateSealed}
	if err := e3.transition(StateAcknowledged, "x"); !errors.Is(err, ErrLifecycle) {
		t.Fatalf("SEALED->ACKNOWLEDGED must be illegal: %v", err)
	}
	// Teardown() is never refusable in a way that retains a
	// workspace: from ACTIVE it force-seals as caller-abort and
	// proceeds (M1 security review MED-5).
	eA := &Env{witness: &recWitness{}, state: StateActive}
	if st := eA.Teardown(); st != StateDestroyed {
		t.Fatalf("Teardown from ACTIVE must force-seal and destroy, got %s", st)
	}
	if eA.Trace().SealReason != SealCallerAbort {
		t.Fatalf("forced seal must be typed caller-abort: %q", eA.Trace().SealReason)
	}
	// And it is idempotent at a terminal.
	if st := eA.Teardown(); st != StateDestroyed {
		t.Fatalf("Teardown at terminal must stay terminal, got %s", st)
	}
	// Terminal states have no exits.
	for _, term := range []State{StateDestroyed, StateTeardownAnomalous} {
		e4 := &Env{witness: &recWitness{}, state: term}
		for _, to := range []State{StateActive, StateTeardown, StateProvisioning, StateEgressing} {
			if err := e4.transition(to, "x"); !errors.Is(err, ErrLifecycle) {
				t.Fatalf("%s must be terminal, allowed -> %s", term, to)
			}
		}
	}
}

// The FULL edge product (test review MED): every one of the 8×8
// state pairs is asserted against the declared legal set, so any
// added edge — forward or backward — fails this test, making
// "invariants hold by reachability" itself regression-proof.
func TestLifecycleEdgeProductExhaustive(t *testing.T) {
	all := []State{StateProvisioning, StateActive, StateSealed, StateEgressing,
		StateAcknowledged, StateTeardown, StateDestroyed, StateTeardownAnomalous}
	legal := map[State]map[State]bool{
		StateProvisioning: {StateActive: true, StateTeardown: true},
		StateActive:       {StateSealed: true},
		StateSealed:       {StateEgressing: true, StateTeardown: true},
		StateEgressing:    {StateAcknowledged: true, StateTeardown: true},
		StateAcknowledged: {StateTeardown: true},
		StateTeardown:     {StateDestroyed: true, StateTeardownAnomalous: true},
	}
	for _, from := range all {
		for _, to := range all {
			e := &Env{witness: &recWitness{}, state: from}
			err := e.transition(to, "probe")
			if legal[from][to] {
				if err != nil {
					t.Errorf("%s -> %s must be legal: %v", from, to, err)
				}
			} else if !errors.Is(err, ErrLifecycle) {
				t.Errorf("%s -> %s must be illegal", from, to)
			}
		}
	}
}

// Budget drained DURING an op auto-seals with the typed deadline
// reason (test review LOW: the mid-drain branch, not just pre-exec).
//
// The drain is driven through the spawnOverride seam rather than by
// setting the budget to 1ms and hoping git is slower. That hope is the
// assumption that made TestExecTimeoutGroupKill fail on Linux: git
// takes ~6ms to spawn on darwin but under 1ms on a Linux runner, where
// the op would simply SUCCEED and this test's first assertion would
// fail. It passed there only because `time.Since(start)` also counts
// cmd.Start(), a margin of one fork/exec that nothing designed.
//
// A child outliving the budget by 100x makes the drain a consequence
// of the budget, which is what this test is about, rather than of host
// process-spawn latency, which it is not.
func TestBudgetMidDrainAutoSeals(t *testing.T) {
	if spawnOverride != nil {
		t.Fatal("spawnOverride must be nil in production — a test leaked the seam")
	}
	shBin, err := exec.LookPath("sh")
	if err != nil {
		t.Skipf("no sh: %v", err)
	}
	sleepBin, err := exec.LookPath("sleep")
	if err != nil {
		t.Skipf("no sleep: %v", err)
	}
	mirrorRoot, repo, sha := mkMirror(t)
	p, err := NewLocalProvider(gitBin(t), t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	env, err := p.Provision(testCeiling(t, mirrorRoot), testSpec(t, repo, sha, ""))
	if err != nil {
		t.Fatal(err)
	}
	mustWitness(t, env)
	defer env.Teardown()

	spawnOverride = func() (string, []string) { return shBin, []string{"-c", sleepBin + " 5"} }
	t.Cleanup(func() { spawnOverride = nil })

	const budget = 50 * time.Millisecond
	env.mu.Lock()
	env.remaining = budget // positive: the pre-exec check must pass
	env.mu.Unlock()

	start := time.Now()
	if _, err := env.ExecGit(30*time.Second, "log"); err == nil {
		t.Fatal("draining op must fail")
	}
	// The REMAINING budget bounds the op, not the requested 30s.
	if el := time.Since(start); el > 3*time.Second {
		t.Fatalf("the remaining budget must cap the effective deadline: op took %s", el)
	}
	if env.State() != StateSealed || env.Trace().SealReason != SealDeadline {
		t.Fatalf("mid-op drain must auto-seal env-deadline: %s %q", env.State(), env.Trace().SealReason)
	}
	// Pin the branch. Pre-exec exhaustion (TestBudgetExhaustionSeals)
	// seals with the SAME reason but records no op at all, so without
	// this the two branches are indistinguishable and a regression
	// collapsing mid-drain into pre-exec would go unnoticed.
	ops := env.Trace().Ops
	if len(ops) == 0 || ops[len(ops)-1].Outcome != "timeout" {
		t.Fatalf("the drain must be attributable to an op that actually ran: %+v", ops)
	}
}

// --- attestation and endpoint refusal -------------------------------

func TestAttestation(t *testing.T) {
	dir := t.TempDir()
	mk := func(name string, mode os.FileMode) string {
		p := filepath.Join(dir, name)
		if err := os.WriteFile(p, []byte("#!/bin/sh\n"), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.Chmod(p, mode); err != nil {
			t.Fatal(err)
		}
		return p
	}
	if _, err := attestBinary(mk("world", 0o757)); !errors.Is(err, ErrAttestation) || !strings.Contains(err.Error(), "world-writable") {
		t.Fatalf("world-writable binary must refuse: %v", err)
	}
	if _, err := attestBinary(mk("suid", 0o755|os.ModeSetuid)); !errors.Is(err, ErrAttestation) || !strings.Contains(err.Error(), "setuid") {
		t.Fatalf("setuid binary must refuse: %v", err)
	}
	ok := mk("plain", 0o755)
	link := filepath.Join(dir, "link")
	if err := os.Symlink(ok, link); err != nil {
		t.Fatal(err)
	}
	if _, err := attestBinary(link); !errors.Is(err, ErrAttestation) || !strings.Contains(err.Error(), "symlink") {
		t.Fatalf("symlinked binary must refuse: %v", err)
	}
	att, err := attestBinary(ok)
	if err != nil || att.Digest == "" || att.Mode == "" {
		t.Fatalf("clean binary must attest with digest+mode: %+v %v", att, err)
	}
}

func TestEndpointRefusal(t *testing.T) {
	bad := []string{
		"https://github.com/x", "git://h/x", "ssh://h/x", "file://local", // schemes
		"git@github.com:x/y.git", // scp with user
		"example.com:repo",       // scp without user — git's own heuristic (MED-1)
		"host:path",              // ditto
		"ext::sh -c evil",        // protocol.ext transport helper (MED-1)
		"fd::17",                 // transport-helper syntax
		// Transport-helper syntax behind a PATH-LIKE prefix. These are
		// the only cases the scheme/"::" check catches alone: the
		// colon-heuristic below it does not fire, because the text
		// before the first colon contains "/". Without them the first
		// check is never the control that refuses, and the mutation
		// pass of 2026-09-14 showed it could be deleted with this test
		// still green — while the L5 traceability cites this test as
		// its evidence. git reads <transport>::<address>, so
		// "/abs/path::evil" names a remote helper.
		"/local/mirror/repo::evil",
		"./dir/x://y",
	}
	for _, a := range bad {
		if err := refuseEndpoints([]string{"clone", a}); !errors.Is(err, ErrEndpoint) {
			t.Errorf("%q must be refused as endpoint-naming", a)
		}
	}
	// Plain local paths pass; a colon after the first slash is a
	// local filename, not a remote.
	for _, ok := range []string{"/local/mirror/repo", "worktree", "dir/file:name"} {
		if err := refuseEndpoints([]string{"clone", ok}); err != nil {
			t.Errorf("%q must pass: %v", ok, err)
		}
	}
}

// The ACTIVE seam is a closed vocabulary, not a general git CLI
// (MED-2): unknown subcommands and ALL caller flags refuse typed.
func TestExecVocabularyClosed(t *testing.T) {
	e := &Env{witness: &recWitness{}, state: StateActive, remaining: time.Minute}
	for _, sub := range []string{"clone", "fetch", "push", "config", "submodule", "remote"} {
		if _, err := e.ExecGit(time.Second, sub); err == nil || !strings.Contains(err.Error(), "outside the active invocation vocabulary") {
			t.Errorf("subcommand %q must be outside the vocabulary: %v", sub, err)
		}
	}
	for _, flag := range []string{"-c", "--git-dir=/x", "--exec-path=/x", "--upload-pack=/x", "-C", "--porcelain"} {
		if _, err := e.ExecGit(time.Second, "status", flag); err == nil || !strings.Contains(err.Error(), "not caller vocabulary") {
			t.Errorf("flag %q must be refused: %v", flag, err)
		}
	}
}

// SEALED means stable by mechanism: Seal refuses while an execution
// is in flight (MED-4).
func TestSealRefusesInflight(t *testing.T) {
	e := &Env{witness: &recWitness{}, state: StateActive, inflight: 1}
	if err := e.Seal(SealTaskComplete); !errors.Is(err, ErrLifecycle) || !strings.Contains(err.Error(), "in flight") {
		t.Fatalf("seal with in-flight execution must refuse: %v", err)
	}
	e.inflight = 0
	if err := e.Seal(SealTaskComplete); err != nil {
		t.Fatalf("seal with no in-flight execution must pass: %v", err)
	}
}

// The wall-clock budget is enforced at the envelope: exhaustion
// seals the environment with the typed deadline reason (MED-3).
func TestBudgetExhaustionSeals(t *testing.T) {
	e := &Env{witness: &recWitness{}, state: StateActive, remaining: 0}
	if _, err := e.ExecGit(time.Second, "status"); err == nil || !strings.Contains(err.Error(), "budget exhausted") {
		t.Fatalf("exhausted budget must refuse typed: %v", err)
	}
	if e.State() != StateSealed || e.Trace().SealReason != SealDeadline {
		t.Fatalf("exhaustion must seal with env-deadline: %s %q", e.State(), e.Trace().SealReason)
	}
}

// The trace is a copy: consumers cannot mutate the provider
// declaration or recorded argv through it (LOW).
func TestTraceIsDeepCopy(t *testing.T) {
	e := &Env{witness: &recWitness{}, state: StateActive}
	e.trace.Provider = LocalDeclaration()
	e.trace.Ops = []OpRecord{{Argv: []string{"status"}}}
	e.trace.Transitions = []Transition{{From: StateProvisioning, To: StateActive, Reason: "provisioned"}}
	tr := e.Trace()
	tr.Provider.Limits[DimWallDeadlineS] = StrengthObserved
	tr.Ops[0].Argv[0] = "mutated"
	tr.Transitions[0].Reason = "forged"
	tr2 := e.Trace()
	if tr2.Provider.Limits[DimWallDeadlineS] != StrengthEnforced || tr2.Ops[0].Argv[0] != "status" ||
		tr2.Transitions[0].Reason != "provisioned" {
		t.Fatal("trace must be a deep copy")
	}
}

func TestNoPATHInEnvironment(t *testing.T) {
	e := &Env{witness: &recWitness{}, homeDir: "/h", tmpDir: "/t"}
	for _, kv := range e.allowEnv() {
		if strings.HasPrefix(kv, "PATH=") {
			t.Fatal("the execution environment must not contain PATH")
		}
	}
}

// --- local provider: provision -> execute -> seal -> teardown -------

func TestLocalProvisionLifecycle(t *testing.T) {
	mirrorRoot, repo, sha := mkMirror(t)
	ceiling := testCeiling(t, mirrorRoot)
	spec := testSpec(t, repo, sha, "")
	p, err := NewLocalProvider(gitBin(t), t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	env, err := p.Provision(ceiling, spec)
	if err != nil {
		t.Fatalf("provision failed: %v", err)
	}
	mustWitness(t, env)
	if env.State() != StateActive {
		t.Fatalf("expected ACTIVE, got %s", env.State())
	}
	ws := env.Workspace()
	if ws.PinnedSHA != sha || ws.Root == "" {
		t.Fatalf("workspace identity incomplete: %+v", ws)
	}
	// The sentinel is checked out at the pin.
	b, err := os.ReadFile(filepath.Join(ws.Root, "parser.go"))
	if err != nil || !strings.Contains(string(b), "SENTINEL-L5") {
		t.Fatalf("checkout content missing: %v", err)
	}
	// Active-phase execution works.
	if _, err := env.ExecGit(30*time.Second, "status"); err != nil {
		t.Fatalf("active exec failed: %v", err)
	}
	// Trace carries attestation, hashes, and provisioning ops.
	tr := env.Trace()
	if tr.Binary.Digest == "" || tr.CeilingHash == "" || tr.SpecHash == "" || len(tr.Ops) < 3 {
		t.Fatalf("trace incomplete: %+v", tr)
	}
	for _, op := range tr.Ops[:3] {
		if op.Phase != "provision" || op.Outcome != "ok" {
			t.Fatalf("provisioning ops must be audited ok: %+v", op)
		}
	}
	// Seal, then execution is refused typed.
	if err := env.Seal(SealTaskComplete); err != nil {
		t.Fatal(err)
	}
	if _, err := env.ExecGit(time.Second, "status"); !errors.Is(err, ErrLifecycle) || !strings.Contains(err.Error(), "SEALED") {
		t.Fatalf("post-seal execution must refuse typed: %v", err)
	}
	// Teardown from a live non-teardown-adjacent state is refused by
	// the machine (SEALED->TEARDOWN is legal; ACTIVE->TEARDOWN was
	// covered above) — here: legal path to verified DESTROYED.
	if st := env.Teardown(); st != StateDestroyed {
		t.Fatalf("expected DESTROYED, got %s", st)
	}
	if _, err := os.Stat(ws.Root); !os.IsNotExist(err) {
		t.Fatal("worktree must be gone after verified teardown")
	}
	if !env.Trace().TeardownVerified {
		t.Fatal("teardown verification must be recorded")
	}
}

func TestProvisionFailurePaths(t *testing.T) {
	mirrorRoot, _, sha := mkMirror(t)
	ceiling := testCeiling(t, mirrorRoot)
	p, err := NewLocalProvider(gitBin(t), t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	// Nonexistent repository: fails, tears down, trace records why.
	env, err := p.Provision(ceiling, testSpec(t, "no-such-repo", sha, ""))
	if err == nil {
		t.Fatal("provision of nonexistent repo must fail")
	}
	if env == nil {
		t.Fatal("post-resource provision failure must return the env with its trace")
	}
	if st := env.State(); st != StateDestroyed {
		t.Fatalf("failed provision must reach a verified terminal: %s", st)
	}
	// SHA not in repo: post-checkout failure path.
	bogus := strings.Repeat("12", 20)
	env2, err := p.Provision(ceiling, testSpec(t, "themis-demo", bogus, ""))
	if err == nil {
		t.Fatal("provision at unknown SHA must fail")
	}
	if env2 == nil || env2.State() != StateDestroyed {
		t.Fatalf("failed provision must tear down with a trace: %+v", env2)
	}
	// Traversal repo name: confined before any resource exists.
	s := testSpec(t, "a", sha, "")
	s.Repo = "a/../../outside"
	if _, err := p.Provision(ceiling, s); err == nil || !errors.Is(err, ErrProvision) {
		t.Fatalf("mirror-root escape must refuse: %v", err)
	}
}

// TestExecTimeoutGroupKill proves bounded termination at the
// subprocess tier (Q-L5-2/6): at the effective deadline the whole
// execution-owned process group dies, not merely the direct child,
// and the timeout is typed and audited.
//
// The child is substituted through spawnOverride rather than relying
// on git being slower than the deadline. That assumption held on
// darwin (git log ~6ms vs a 1ms deadline) and failed on Linux (<1ms),
// where the call simply succeeded — so this control went unproven on
// the platform deployments run on. The substitute also forks a
// GRANDCHILD, which the single git process never did: killing only
// the direct child leaves the grandchild alive to create the marker.
func TestExecTimeoutGroupKill(t *testing.T) {
	if spawnOverride != nil {
		t.Fatal("spawnOverride must be nil in production — a test leaked the seam")
	}
	shBin, err := exec.LookPath("sh")
	if err != nil {
		t.Skipf("no sh: %v", err)
	}
	sleepBin, err := exec.LookPath("sleep")
	if err != nil {
		t.Skipf("no sleep: %v", err)
	}
	mirrorRoot, repo, sha := mkMirror(t)
	ceiling := testCeiling(t, mirrorRoot)
	p, err := NewLocalProvider(gitBin(t), t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	env, err := p.Provision(ceiling, testSpec(t, repo, sha, ""))
	if err != nil {
		t.Fatal(err)
	}
	mustWitness(t, env)
	defer func() { _ = env.Seal(SealCallerAbort); env.Teardown() }()

	// The grandchild would create the marker well after the deadline
	// but well before this test checks; the parent blocks far beyond
	// both. The environment carries no PATH, so every binary is
	// absolute and only shell builtins (`:` and redirection) are used
	// for the marker itself.
	marker := filepath.Join(t.TempDir(), "grandchild-survived")
	script := "( " + sleepBin + " 0.5; : > '" + marker + "' ) & " + sleepBin + " 30"
	spawnOverride = func() (string, []string) { return shBin, []string{"-c", script} }
	t.Cleanup(func() { spawnOverride = nil })

	start := time.Now()
	if _, err := env.ExecGit(100*time.Millisecond, "log"); err == nil || !strings.Contains(err.Error(), "timeout") {
		t.Fatalf("deadline must produce a typed timeout: %v", err)
	}
	if el := time.Since(start); el > 5*time.Second {
		t.Fatalf("the deadline did not bound the call: returned after %s", el)
	}
	tr := env.Trace()
	last := tr.Ops[len(tr.Ops)-1]
	if last.Outcome != "timeout" {
		t.Fatalf("timeout must be audited: %+v", last)
	}
	// Group kill, not child kill: the grandchild must never run.
	time.Sleep(2 * time.Second)
	if _, err := os.Stat(marker); !os.IsNotExist(err) {
		t.Fatalf("grandchild outlived the deadline — the execution-owned process group was not killed (stat: %v)", err)
	}
}

// TestObservedRSSIsInBytes: the observed max-RSS accounting is in
// BYTES on every platform. getrusage reports ru_maxrss in bytes on
// darwin and in KILOBYTES on Linux, so local.go multiplies by 1024
// there — a platform branch nothing discriminated.
//
// TestEgressMemObservedGate cannot: its bound is mem_bytes=1, which
// every observation breaches, so a missing conversion (1024x low) or a
// doubled one (1024x high) passed identically. L5 refused "the 'we
// support memory limits' lie" in writing; an observation wrong by
// three orders of magnitude is the same kind of dishonesty, so the
// magnitude needs an assertion of its own.
//
// The bounds are deliberately loose. They are not a claim about git's
// footprint — they are the widest window that still separates bytes
// from kilobytes.
func TestObservedRSSIsInBytes(t *testing.T) {
	mirrorRoot, repo, sha := mkMirror(t)
	p, err := NewLocalProvider(gitBin(t), t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	env, err := p.Provision(testCeiling(t, mirrorRoot), testSpec(t, repo, sha, ""))
	if err != nil {
		t.Fatal(err)
	}
	mustWitness(t, env)
	defer func() { _ = env.Seal(SealCallerAbort); env.Teardown() }()

	var peak int64
	for _, op := range env.Trace().Ops {
		if op.MaxRSSByte > peak {
			peak = op.MaxRSSByte
		}
	}
	t.Logf("observed peak RSS across provisioning ops: %d bytes (%.1f MiB) on %s",
		peak, float64(peak)/(1<<20), runtime.GOOS)
	// 1 MiB floor: below it the value is kilobytes. 1 GiB ceiling:
	// git on this single-commit mirror observes a few MiB, so 1 GiB is
	// generous for a real process yet still catches a doubled
	// conversion, which lands three orders of magnitude above it.
	const floorByte = int64(1) << 20
	const capByte = int64(1) << 30
	if peak < floorByte {
		t.Fatalf("observed RSS %d is implausible for a real git process — the platform unit conversion is missing (value looks like KILOBYTES)", peak)
	}
	if peak > capByte {
		t.Fatalf("observed RSS %d exceeds any plausible git footprint — the unit conversion was applied where it should not be", peak)
	}
}

// A teardown that cannot verify host state is TEARDOWN_ANOMALOUS,
// typed in the trace, and never a false DESTROYED (Q-L5-12, M1 MED-5).
//
// Two fixtures make removal fail, because the single darwin-only one
// this test shipped with left the whole clause unevidenced on Linux —
// the platform deployments run on. `unremovable-parent` is portable
// and is the primary case; `immutable-flag` keeps the original darwin
// mechanism, which is a genuinely different failure (the flag survives
// teardown's permission-restore walk).
func TestTeardownAnomalous(t *testing.T) {
	// Both fixtures are permission/flag based and void under root,
	// which cannot remove the ability to remove (same convention as
	// the egress fixtures).
	if os.Getuid() == 0 {
		t.Skip("removal-blocking fixtures are void under root")
	}
	assertAnomalous := func(t *testing.T, env *Env) {
		t.Helper()
		if err := env.Seal(SealCallerAbort); err != nil {
			t.Fatal(err)
		}
		if st := env.Teardown(); st != StateTeardownAnomalous {
			t.Fatalf("unverifiable teardown must be TEARDOWN_ANOMALOUS, got %s", st)
		}
		tr := env.Trace()
		if tr.TeardownVerified {
			t.Fatal("anomalous teardown must not claim verification")
		}
		final := tr.Transitions[len(tr.Transitions)-1]
		if final.To != StateTeardownAnomalous || !strings.Contains(final.Reason, "host-state-unverified") {
			t.Fatalf("anomaly must be typed in the trace: %+v", final)
		}
	}

	// Portable: the environment's PARENT is made non-writable, so
	// unlinking baseDir itself fails. teardown's restore walk is
	// scoped to baseDir and never chmods its parent, so — unlike a
	// 0555 directory inside the workspace — this survives the walk on
	// every platform.
	t.Run("unremovable-parent", func(t *testing.T) {
		mirrorRoot, repo, sha := mkMirror(t)
		providerDir := t.TempDir()
		p, err := NewLocalProvider(gitBin(t), providerDir)
		if err != nil {
			t.Fatal(err)
		}
		env, err := p.Provision(testCeiling(t, mirrorRoot), testSpec(t, repo, sha, ""))
		if err != nil {
			t.Fatal(err)
		}
		mustWitness(t, env)
		if err := os.Chmod(providerDir, 0o555); err != nil {
			t.Fatal(err)
		}
		defer func() {
			_ = os.Chmod(providerDir, 0o755)
			_ = os.RemoveAll(env.baseDir)
		}()
		assertAnomalous(t, env)
	})

	// darwin: the immutable flag survives the permission-restore walk
	// (a plain 0555 dir would not — the walk exists so the OS-level
	// seal can be undone before RemoveAll).
	t.Run("immutable-flag", func(t *testing.T) {
		if runtime.GOOS != "darwin" {
			t.Skip("uchg immutable-flag fixture is darwin-specific")
		}
		mirrorRoot, repo, sha := mkMirror(t)
		p, err := NewLocalProvider(gitBin(t), t.TempDir())
		if err != nil {
			t.Fatal(err)
		}
		env, err := p.Provision(testCeiling(t, mirrorRoot), testSpec(t, repo, sha, ""))
		if err != nil {
			t.Fatal(err)
		}
		mustWitness(t, env)
		pinned := filepath.Join(env.Workspace().Root, "pinned")
		if err := os.WriteFile(pinned, []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
		if out, err := exec.Command("/usr/bin/chflags", "uchg", pinned).CombinedOutput(); err != nil {
			t.Skipf("cannot set uchg: %v %s", err, out)
		}
		defer func() {
			_ = exec.Command("/usr/bin/chflags", "nouchg", pinned).Run()
			_ = os.RemoveAll(env.baseDir)
		}()
		assertAnomalous(t, env)
	})
}

// Q-L5-3 post-condition: after `checkout --detach <pin>` reports
// success, the workspace IS at the pin — asserted by rev-parse, never
// assumed. TestProvisionFailurePaths covers a checkout that FAILS
// (unknown SHA); it cannot reach this branch, which is the case where
// the checkout succeeded and HEAD still is not the pin. Suppressing the
// post-condition left the whole package green.
//
// The reachable instance is an uppercase pin. Git resolves object names
// case-insensitively, so `checkout --detach <UPPERCASE>` succeeds while
// rev-parse reports the canonical lowercase — a genuine "checkout
// succeeded, HEAD != pin" that needs no fault injection.
//
// What it stands for and what it does not: it exercises the predicate,
// not a hostile mirror. A governed spec cannot carry an uppercase pin
// (parseSpec requires ^[0-9a-f]{40}$), so this is constructed in-package
// the same way the traversal case below constructs s.Repo. The value is
// that the post-condition is now known to fire and to tear down, rather
// than assumed to.
func TestProvisionPostConditionRefusesHeadNotAtPin(t *testing.T) {
	mirrorRoot, _, sha := mkMirror(t)
	ceiling := testCeiling(t, mirrorRoot)
	p, err := NewLocalProvider(gitBin(t), t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	// Premise: the same pin in canonical form provisions cleanly, so
	// nothing but the post-condition can explain a refusal below.
	ok, err := p.Provision(ceiling, testSpec(t, "themis-demo", sha, ""))
	if err != nil {
		t.Fatalf("the canonical pin must provision, or this proves nothing: %v", err)
	}
	mustWitness(t, ok)
	ok.Teardown()

	s := testSpec(t, "themis-demo", sha, "")
	s.PinnedSHA = strings.ToUpper(sha)
	env, err := p.Provision(ceiling, s)
	if err == nil {
		t.Fatal("provisioning returned success with HEAD not at the pinned string")
	}
	if !errors.Is(err, ErrProvision) {
		t.Errorf("a failed post-condition is a provisioning failure, got %v", err)
	}
	if !strings.Contains(err.Error(), "post-condition failed") {
		t.Fatalf("refused for the wrong reason — the checkout itself must have succeeded: %v", err)
	}
	if env == nil {
		t.Fatal("post-resource provision failure must return the env with its trace")
	}
	if st := env.State(); st != StateDestroyed {
		t.Fatalf("a failed post-condition must tear down: %s", st)
	}
}

// Q-L5-6 non-elevation floor. The harness refuses to construct a
// provider when its effective identity differs from its real one —
// setuid or setgid — because every execution it launches would inherit
// that context. The guard survived the 2026-09-14 mutation pass: a
// process cannot lower and restore its own euid to exercise this in
// place, and a seam that let it would be a seam into the privilege
// floor itself.
//
// Both halves are independent. setuid raises the effective USER,
// setgid the effective GROUP, and a binary may carry either alone — a
// check comparing only uids would wave a setgid-elevated process
// straight through.
func TestNonElevationFloor(t *testing.T) {
	for what, id := range map[string][4]int{
		"setuid — effective user raised":  {0, 1000, 1000, 1000},
		"setgid — effective group raised": {1000, 1000, 0, 1000},
		"both raised":                     {0, 1000, 0, 1000},
		"dropped below real user":         {1000, 0, 1000, 1000},
	} {
		err := refuseElevation(id[0], id[1], id[2], id[3])
		if err == nil {
			t.Errorf("%s: an elevated execution context was ACCEPTED (euid=%d uid=%d egid=%d gid=%d)",
				what, id[0], id[1], id[2], id[3])
			continue
		}
		if !errors.Is(err, ErrAttestation) {
			t.Errorf("%s: refused as %v, want an attestation refusal", what, err)
		}
		if !strings.Contains(err.Error(), "refusing elevated execution context") {
			t.Errorf("%s: refused for the wrong reason: %v", what, err)
		}
	}
	// Premise: matching identities pass, or the floor refuses every
	// process and the cases above prove nothing.
	for what, id := range map[string][4]int{
		"ordinary user": {1000, 1000, 1000, 1000},
		"root as root":  {0, 0, 0, 0},
	} {
		if err := refuseElevation(id[0], id[1], id[2], id[3]); err != nil {
			t.Errorf("%s: a non-elevated context was refused: %v", what, err)
		}
	}
	// And the floor is actually wired: the running process is
	// non-elevated (the test suite would not be runnable otherwise), so
	// construction must reach past it.
	if _, err := NewLocalProvider(gitBin(t), t.TempDir()); err != nil {
		t.Fatalf("a non-elevated process must construct a provider: %v", err)
	}
}

// Attestation covers symlink, setuid/setgid, world-writable file and
// world-writable directory. The "regular file" guard is separate and
// was untested: a directory, FIFO or device node is not a symlink and
// carries none of those modes, so every other check waves it through
// and only this one speaks. Pinning a directory as the git binary is an
// operator error that must fail at construction, not at the first exec.
func TestAttestationRefusesNonRegularFile(t *testing.T) {
	dir := t.TempDir()
	sub := filepath.Join(dir, "not-a-binary")
	if err := os.Mkdir(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	_, err := attestBinary(sub)
	if err == nil {
		t.Fatal("a directory attested as the pinned executable")
	}
	if !errors.Is(err, ErrAttestation) {
		t.Errorf("refused as %v, want an attestation refusal", err)
	}
	if !strings.Contains(err.Error(), "not a regular file") {
		t.Fatalf("refused for the wrong reason — no other attestation check applies to a directory: %v", err)
	}
	// The same refusal must reach the provider constructor.
	if _, perr := NewLocalProvider(sub, t.TempDir()); perr == nil ||
		!strings.Contains(perr.Error(), "not a regular file") {
		t.Fatalf("a provider was constructed over a non-regular git path: %v", perr)
	}
}

// Budget exhaustion is refused TWICE with the same words — once at the
// ExecGit envelope, once inside runGit after the remaining budget caps
// the caller's deadline. TestBudgetExhaustionSeals covers the envelope,
// and with that guard suppressed the inner one produces an identical
// message, so neither was evidenced. Both survived the mutation pass.
//
// They are not redundant, and their side effects are what separate
// them: the envelope check SEALS the environment with the typed
// deadline reason, while the inner check records a budget-exhausted op
// in the trace and leaves the state alone. An environment that ran out
// mid-operation is in a different condition from one asked to start
// work it could never finish.
func TestBudgetExhaustionAtBothDoors(t *testing.T) {
	t.Run("envelope: no budget to begin with — seals", func(t *testing.T) {
		e := &Env{witness: &recWitness{}, state: StateActive, remaining: 0}
		_, err := e.ExecGit(time.Second, "status")
		if err == nil || !strings.Contains(err.Error(), "budget exhausted") {
			t.Fatalf("exhausted budget must refuse typed: %v", err)
		}
		if e.State() != StateSealed || e.Trace().SealReason != SealDeadline {
			t.Fatalf("the envelope check must seal with env-deadline: %s %q", e.State(), e.Trace().SealReason)
		}
		// Nothing was attempted, so nothing is recorded as an op.
		for _, op := range e.Trace().Ops {
			if op.Outcome == "budget-exhausted" {
				t.Error("the envelope refusal recorded an op — no execution was attempted")
			}
		}
	})

	t.Run("inner: budget vanishes before spawn — records, does not seal", func(t *testing.T) {
		// A positive remaining budget passes the envelope, then the
		// caller's deadline is capped to it. Driving the remaining
		// budget to zero between the two is what the inner guard is
		// for; here it is set directly, which is the same state.
		//
		// The provider is real so that suppressing the guard produces a
		// FAILED ASSERTION rather than a nil-pointer panic: a test must
		// fail by the thing it claims to check.
		p, perr := NewLocalProvider(gitBin(t), t.TempDir())
		if perr != nil {
			t.Fatal(perr)
		}
		e := &Env{witness: &recWitness{}, state: StateActive, remaining: 0, provider: p}
		_, err := e.runGit("active", time.Second, t.TempDir(), "status")
		if err == nil || !strings.Contains(err.Error(), "budget exhausted") {
			t.Fatalf("a non-positive effective deadline must refuse typed: %v", err)
		}
		if e.State() == StateSealed {
			t.Error("the inner check must not seal — sealing is the envelope's act")
		}
		found := false
		for _, op := range e.Trace().Ops {
			if op.Outcome == "budget-exhausted" && op.Exit == -1 {
				found = true
			}
		}
		if !found {
			t.Errorf("the inner refusal must be recorded as a budget-exhausted op: %+v", e.Trace().Ops)
		}
	})
}

// D-SA-4 spec relation: subject fields substituted, wall deadline
// narrows, every other limit equal; an absent template strength means
// the loader's default (enforced).
func TestSpecInstantiates(t *testing.T) {
	tpl := []byte(`{"version":1,"task_id":"@task_id","repo":"@repo","pinned_sha":"@pinned_sha","limits":[{"dimension":"wall_deadline_s","value":600}]}`)
	mk := func(wall int64, extra string) *ProvisionSpec {
		s := &ProvisionSpec{Version: 1, TaskID: "T", Repo: "demo", PinnedSHA: strings.Repeat("a", 40),
			Limits: []LimitReq{{Dimension: DimWallDeadlineS, Value: wall, Strength: StrengthEnforced}}}
		if extra != "" {
			s.Limits = append(s.Limits, LimitReq{Dimension: extra, Value: 1, Strength: StrengthEnforced})
		}
		return s
	}
	if err := SpecInstantiates(mk(90, ""), tpl); err != nil {
		t.Fatalf("narrowed deadline must instantiate: %v", err)
	}
	if err := SpecInstantiates(mk(600, ""), tpl); err != nil {
		t.Fatalf("equal deadline must instantiate: %v", err)
	}
	for name, c := range map[string]struct {
		spec *ProvisionSpec
		want string
	}{
		"deadline widened": {mk(601, ""), "only narrows"},
		"limit added":      {mk(90, DimFileBytes), "limits are Skill-fixed"},
	} {
		if err := SpecInstantiates(c.spec, tpl); err == nil || !strings.Contains(err.Error(), c.want) {
			t.Fatalf("%s: refused for the wrong reason (want %q): %v", name, c.want, err)
		}
	}
	literal := []byte(`{"version":1,"task_id":"T","repo":"demo","pinned_sha":"` + strings.Repeat("a", 40) + `","limits":[{"dimension":"wall_deadline_s","value":600}]}`)
	if err := SpecInstantiates(mk(90, ""), literal); err == nil || !strings.Contains(err.Error(), "placeholder") {
		t.Fatalf("a literal template must refuse: %v", err)
	}
}
