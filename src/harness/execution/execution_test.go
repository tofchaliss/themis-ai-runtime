package execution

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
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
	run(repo, "add", ".")
	run(repo, "commit", "-q", "-m", "seed")
	sha := strings.TrimSpace(run(repo, "rev-parse", "HEAD"))
	return root, "themis-demo", sha
}

func testCeiling(t *testing.T, mirrorRoot string) *WorkspaceExecutionCeiling {
	t.Helper()
	body := `{"version":1,"mirror_root":"` + mirrorRoot + `","max_wall_deadline_sec":120,"max_file_bytes":1048576,"max_total_bytes":10485760,"max_file_count":500}`
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
		{"unknown-field", `{"version":1,"mirror_root":"/m","max_wall_deadline_sec":9,"max_file_bytes":1,"max_total_bytes":1,"max_file_count":1,"extra":true}`, "unknown field"},
		{"trailing", `{"version":1,"mirror_root":"/m","max_wall_deadline_sec":9,"max_file_bytes":1,"max_total_bytes":1,"max_file_count":1}{}`, "trailing content"},
		{"no-version", `{"mirror_root":"/m","max_wall_deadline_sec":9,"max_file_bytes":1,"max_total_bytes":1,"max_file_count":1}`, "version required"},
		{"relative-root", `{"version":1,"mirror_root":"mirrors","max_wall_deadline_sec":9,"max_file_bytes":1,"max_total_bytes":1,"max_file_count":1}`, "must be absolute"},
		{"zero-bound", `{"version":1,"mirror_root":"/m","max_wall_deadline_sec":0,"max_file_bytes":1,"max_total_bytes":1,"max_file_count":1}`, "must be positive"},
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
	e := &Env{state: StateActive}
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
	e2 := &Env{state: StateActive}
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
	e3 := &Env{state: StateSealed}
	if err := e3.transition(StateAcknowledged, "x"); !errors.Is(err, ErrLifecycle) {
		t.Fatalf("SEALED->ACKNOWLEDGED must be illegal: %v", err)
	}
	// Teardown() from ACTIVE is refused by the machine: state unchanged.
	eA := &Env{state: StateActive}
	if st := eA.Teardown(); st != StateActive {
		t.Fatalf("Teardown from ACTIVE must be refused, got %s", st)
	}
	// Terminal states have no exits.
	for _, term := range []State{StateDestroyed, StateTeardownAnomalous} {
		e4 := &Env{state: term}
		for _, to := range []State{StateActive, StateTeardown, StateProvisioning, StateEgressing} {
			if err := e4.transition(to, "x"); !errors.Is(err, ErrLifecycle) {
				t.Fatalf("%s must be terminal, allowed -> %s", term, to)
			}
		}
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
	for _, bad := range []string{"https://github.com/x", "git://h/x", "ssh://h/x", "git@github.com:x/y.git", "file://local"} {
		if err := refuseEndpoints([]string{"clone", bad}); !errors.Is(err, ErrEndpoint) {
			t.Errorf("%q must be refused as endpoint-naming", bad)
		}
	}
	if err := refuseEndpoints([]string{"clone", "/local/mirror/repo", "worktree"}); err != nil {
		t.Errorf("plain local paths must pass: %v", err)
	}
}

func TestNoPATHInEnvironment(t *testing.T) {
	e := &Env{homeDir: "/h", tmpDir: "/t"}
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
	if _, err := env.ExecGit(30*time.Second, "status", "--porcelain"); err != nil {
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
	if env != nil {
		if st := env.State(); st != StateDestroyed {
			t.Fatalf("failed provision must reach a verified terminal: %s", st)
		}
	}
	// SHA not in repo: post-checkout failure path.
	bogus := strings.Repeat("12", 20)
	env2, err := p.Provision(ceiling, testSpec(t, "themis-demo", bogus, ""))
	if err == nil {
		t.Fatal("provision at unknown SHA must fail")
	}
	if env2 != nil && env2.State() != StateDestroyed {
		t.Fatalf("failed provision must tear down: %s", env2.State())
	}
	// Traversal repo name: confined before any resource exists.
	s := testSpec(t, "a", sha, "")
	s.Repo = "a/../../outside"
	if _, err := p.Provision(ceiling, s); err == nil || !errors.Is(err, ErrProvision) {
		t.Fatalf("mirror-root escape must refuse: %v", err)
	}
}

func TestExecTimeoutGroupKill(t *testing.T) {
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
	defer func() { _ = env.Seal(SealCallerAbort); env.Teardown() }()
	// A deadline shorter than process spawn: deterministic timeout,
	// group-killed, typed.
	if _, err := env.ExecGit(time.Millisecond, "log"); err == nil || !strings.Contains(err.Error(), "timeout") {
		t.Fatalf("sub-spawn deadline must produce a typed timeout: %v", err)
	}
	tr := env.Trace()
	last := tr.Ops[len(tr.Ops)-1]
	if last.Outcome != "timeout" {
		t.Fatalf("timeout must be audited: %+v", last)
	}
}

func TestTeardownAnomalous(t *testing.T) {
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
	ws := env.Workspace()
	// An unremovable entry: directory without write permission
	// containing a file (unlink requires parent write).
	locked := filepath.Join(ws.Root, "locked")
	if err := os.Mkdir(locked, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(locked, "pin"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(locked, 0o555); err != nil {
		t.Fatal(err)
	}
	defer func() {
		_ = os.Chmod(locked, 0o755)
		_ = os.RemoveAll(env.baseDir)
	}()
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
