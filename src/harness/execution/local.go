package execution

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"time"

	"github.com/tofchaliss/themis/confine"
)

var (
	ErrProvision   = errors.New("environment-provision-failed")
	ErrAttestation = errors.New("binary attestation refused")
	ErrEndpoint    = errors.New("endpoint-naming input refused")
	ErrExec        = errors.New("environment execution failed")
)

// LocalDeclaration is the v1 local provider's honest property
// declaration. Deviations from the design matrix are refusals to
// overclaim, not omissions:
//   - cpu_time_s is ABSENT (design matrix listed RLIMIT_CPU for the
//     subprocess tier, but Go's exec cannot set a child rlimit
//     portably pre-exec; declaring without a mechanism is the exact
//     lie Q-L5-9 forbids — the dimension joins when the mechanism
//     exists).
//   - proc_count is ABSENT deliberately (RLIMIT_NPROC is per-UID
//     under inherited identity — the mechanism could damage the
//     developer's session; Q-L5-9).
func LocalDeclaration() ProviderDeclaration {
	return ProviderDeclaration{
		Name:            "local-v1",
		ProcessIdentity: IdentityInherited,
		Network:         DeniedByConstruction,
		HostServices:    DeniedByConstruction,
		Termination:     TerminationGroupKill,
		Limits: map[string]Strength{
			DimWallDeadlineS: StrengthEnforced, // env budget + group kill
			DimMemBytes:      StrengthObserved, // post-hoc rusage max-RSS
			DimDiskBytes:     StrengthObserved, // measured at egress
			DimFileBytes:     StrengthObserved, // per-file scan at egress
		},
	}
}

// LocalProvider provisions git-worktree environments on the host,
// under inherited identity (declared, not disguised).
type LocalProvider struct {
	GitPath string // pinned absolute executable path
	BaseDir string // absolute directory environments are created under
	decl    ProviderDeclaration
	binary  BinaryAttestation
}

// NewLocalProvider asserts the non-elevation floor and attests the
// pinned binary once at construction; provisioning re-records the
// attestation per environment.
func NewLocalProvider(gitPath, baseDir string) (*LocalProvider, error) {
	// Non-elevation floor (Q-L5-6): the harness must not be running
	// with effective privilege above its real identity, and must never
	// launch executions through an elevated context.
	if os.Geteuid() != os.Getuid() || os.Getegid() != os.Getgid() {
		return nil, fmt.Errorf("%w: effective identity differs from real identity — refusing elevated execution context", ErrAttestation)
	}
	if !filepath.IsAbs(gitPath) || !filepath.IsAbs(baseDir) {
		return nil, fmt.Errorf("%w: git path and base dir must be absolute", ErrProvision)
	}
	att, err := attestBinary(gitPath)
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(baseDir, 0o755); err != nil {
		return nil, fmt.Errorf("%w: base dir: %v", ErrProvision, err)
	}
	return &LocalProvider{GitPath: gitPath, BaseDir: baseDir, decl: LocalDeclaration(), binary: att}, nil
}

// Declaration returns a copy of the provider's property declaration.
func (p *LocalProvider) Declaration() ProviderDeclaration {
	d := p.decl
	d.Limits = copyLimits(p.decl.Limits)
	return d
}

// attestBinary records the executable's identity and refuses
// privilege-carrying or tamper-exposed binaries: setuid/setgid,
// world-writable file, world-writable containing directory
// (Q-L5-6). The digest is evidence of what was executed, not proof
// of trustworthy behavior; the attest→exec window is a documented
// threat-model residual, not a closed one.
func attestBinary(path string) (BinaryAttestation, error) {
	info, err := os.Lstat(path)
	if err != nil {
		return BinaryAttestation{}, fmt.Errorf("%w: %s: %v", ErrAttestation, path, err)
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return BinaryAttestation{}, fmt.Errorf("%w: %s is a symlink — pin the resolved executable", ErrAttestation, path)
	}
	if !info.Mode().IsRegular() {
		return BinaryAttestation{}, fmt.Errorf("%w: %s is not a regular file", ErrAttestation, path)
	}
	if info.Mode()&(os.ModeSetuid|os.ModeSetgid) != 0 {
		return BinaryAttestation{}, fmt.Errorf("%w: %s carries setuid/setgid — refusing privilege-carrying executable", ErrAttestation, path)
	}
	if info.Mode().Perm()&0o002 != 0 {
		return BinaryAttestation{}, fmt.Errorf("%w: %s is world-writable", ErrAttestation, path)
	}
	dirInfo, err := os.Stat(filepath.Dir(path))
	if err != nil {
		return BinaryAttestation{}, fmt.Errorf("%w: %v", ErrAttestation, err)
	}
	// A world-writable directory without the sticky bit lets any user
	// replace the binary between attestation and exec.
	if dirInfo.Mode().Perm()&0o002 != 0 && dirInfo.Mode()&os.ModeSticky == 0 {
		return BinaryAttestation{}, fmt.Errorf("%w: %s: containing directory is world-writable", ErrAttestation, path)
	}
	f, err := os.Open(path)
	if err != nil {
		return BinaryAttestation{}, fmt.Errorf("%w: %v", ErrAttestation, err)
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return BinaryAttestation{}, fmt.Errorf("%w: %v", ErrAttestation, err)
	}
	return BinaryAttestation{Path: path, Digest: hex.EncodeToString(h.Sum(nil)), Mode: info.Mode().String()}, nil
}

// refuseEndpoints rejects argv values that could name a network
// endpoint or transport helper — the deterministically testable half
// of denied-by-construction (Q-L5-7): a network-shaped input dies as
// a typed refusal at argv construction, never as a downstream git
// network error. Rules (M1 security review MED-1):
//   - "://"        — URL schemes;
//   - "::"         — ext::/transport-helper syntax (arbitrary command
//     execution via protocol.ext);
//   - a ":" with no "/" before it — git's own scp-remote heuristic
//     (host:path, with or without user@).
//
// Defense in depth — the invocation vocabulary never constructs such
// an argument.
func refuseEndpoints(argv []string) error {
	for _, a := range argv {
		if strings.Contains(a, "://") || strings.Contains(a, "::") {
			return fmt.Errorf("%w: %q", ErrEndpoint, a)
		}
		if i := strings.IndexByte(a, ':'); i >= 0 && !strings.Contains(a[:i], "/") {
			return fmt.Errorf("%w: %q", ErrEndpoint, a)
		}
	}
	return nil
}

// Provision walks PROVISIONING for a spec against a ceiling. Any
// failure transitions the environment to TEARDOWN and tears down what
// was partially created; the returned Env (never nil once
// provisioning began) carries the typed trace either way.
func (p *LocalProvider) Provision(ceiling *WorkspaceExecutionCeiling, spec *ProvisionSpec) (*Env, error) {
	// Admission before any resource exists: refuse the environment,
	// never the request.
	if err := spec.ValidateAgainst(ceiling); err != nil {
		return nil, err
	}
	if err := Admit(p.decl, spec); err != nil {
		return nil, err
	}
	// Repository identity resolves under the governed mirror root only
	// (Q-L5-3: local mirrors only); confinement reuses the canonical
	// implementation.
	mirror, err := confine.ResolvePath(ceiling.MirrorRoot, spec.Repo)
	if err != nil {
		return nil, fmt.Errorf("%w: repository %q: %v", ErrProvision, spec.Repo, err)
	}
	att, err := attestBinary(p.GitPath)
	if err != nil {
		return nil, err
	}

	deadline, _ := spec.Limit(DimWallDeadlineS)
	e := &Env{
		state:    StateProvisioning,
		provider: p,
		// One wall-clock budget for the whole environment:
		// provisioning and active ops draw it down together, so the
		// declared wall_deadline_s bounds the envelope, not each call
		// (M1 security review MED-3).
		remaining: time.Duration(deadline.Value) * time.Second,
		trace: Trace{
			CeilingHash: ceiling.Hash,
			SpecHash:    spec.Hash,
			Provider:    p.Declaration(),
			Binary:      att,
			Workspace:   Workspace{Repo: spec.Repo, PinnedSHA: spec.PinnedSHA},
		},
	}
	fail := func(cause error) (*Env, error) {
		_ = e.transition(StateTeardown, "provision-failed: "+cause.Error())
		e.teardown()
		return e, cause
	}

	base, err := os.MkdirTemp(p.BaseDir, "env-")
	if err != nil {
		return fail(fmt.Errorf("%w: %v", ErrProvision, err))
	}
	e.baseDir = base
	e.homeDir = filepath.Join(base, "home")
	e.tmpDir = filepath.Join(base, "tmp")
	e.hooksDir = filepath.Join(base, "nohooks")
	worktree := filepath.Join(base, "worktree")
	for _, d := range []string{e.homeDir, e.tmpDir, e.hooksDir} {
		if err := os.Mkdir(d, 0o755); err != nil {
			return fail(fmt.Errorf("%w: %v", ErrProvision, err))
		}
	}
	e.mu.Lock()
	e.trace.Workspace.Root = worktree
	e.mu.Unlock()

	// Provisioning is execution (Q-L5-3): the same neutralized,
	// audited invocation path as active-phase ops.
	if _, err := e.runGit("provision", e.budget(), p.BaseDir, "clone", "--no-hardlinks", mirror, worktree); err != nil {
		return fail(err)
	}
	if _, err := e.runGit("provision", e.budget(), worktree, "-c", "advice.detachedHead=false", "checkout", "--detach", spec.PinnedSHA); err != nil {
		return fail(err)
	}
	// Post-condition verification (Q-L5-3): the checkout IS at the
	// pinned SHA — asserted, not assumed.
	head, err := e.runGit("provision", e.budget(), worktree, "rev-parse", "HEAD")
	if err != nil {
		return fail(err)
	}
	if got := strings.TrimSpace(head); got != spec.PinnedSHA {
		return fail(fmt.Errorf("%w: post-condition failed: HEAD %q != pinned %q", ErrProvision, got, spec.PinnedSHA))
	}
	if err := e.transition(StateActive, "provisioned"); err != nil {
		return fail(err)
	}
	return e, nil
}

func (e *Env) budget() time.Duration {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.remaining
}

// allowEnv is the entire execution environment: empty by default,
// explicitly declared non-secret variables only, no PATH (Q-L5-5).
// Dangerous variables are not scrubbed — they never exist.
func (e *Env) allowEnv() []string {
	return []string{
		"HOME=" + e.homeDir,
		"TMPDIR=" + e.tmpDir,
		"LC_ALL=C",
		"GIT_CONFIG_GLOBAL=/dev/null",
		"GIT_CONFIG_SYSTEM=/dev/null",
		"GIT_CONFIG_NOSYSTEM=1",
		"GIT_TERMINAL_PROMPT=0",
	}
}

// activeVocabulary is the closed set of git subcommands an ACTIVE
// environment exposes to callers. ExecGit is not a general git seam:
// caller arguments may not be flags, so `-c`, `-C`, `--git-dir`,
// `--exec-path`, `--upload-pack` and every other behavior-altering
// switch is structurally outside the caller vocabulary (M1 security
// review MED-2). The vocabulary widens only by deliberate edit here.
var activeVocabulary = map[string]bool{
	"status": true, "log": true, "diff": true, "show": true,
	"rev-parse": true, "ls-files": true,
}

// ExecGit runs one active-phase git operation inside the environment.
// Refused unless the environment is ACTIVE — a sealed environment
// refuses all further executions, typed (Q-L5-12). The call draws
// down the environment's wall-clock budget; exhaustion seals the
// environment with SealDeadline.
func (e *Env) ExecGit(deadline time.Duration, sub string, args ...string) (string, error) {
	if !activeVocabulary[sub] {
		return "", fmt.Errorf("%w: subcommand %q is outside the active invocation vocabulary", ErrExec, sub)
	}
	for _, a := range args {
		if strings.HasPrefix(a, "-") {
			return "", fmt.Errorf("%w: flag argument %q is not caller vocabulary", ErrExec, a)
		}
	}
	e.mu.Lock()
	if e.state != StateActive {
		st := e.state
		e.mu.Unlock()
		return "", fmt.Errorf("%w: execution refused in state %s", ErrLifecycle, st)
	}
	if e.remaining <= 0 {
		_ = e.sealLocked(SealDeadline)
		e.mu.Unlock()
		return "", fmt.Errorf("%w: environment wall-clock budget exhausted", ErrExec)
	}
	root := e.trace.Workspace.Root
	e.inflight++
	e.mu.Unlock()
	out, err := e.runGit("active", deadline, root, append([]string{sub}, args...)...)
	e.mu.Lock()
	e.inflight--
	if e.remaining <= 0 && e.state == StateActive && e.inflight == 0 {
		_ = e.sealLocked(SealDeadline)
	}
	e.mu.Unlock()
	return out, err
}

// runGit is the single subprocess invocation path: pinned absolute
// binary, endpoint-refused argv, hooks neutralized, empty allowlist
// environment, execution-owned process group, group-killed at the
// effective deadline (min of the requested deadline and the
// environment's remaining budget). There is no second, more
// convenient path.
func (e *Env) runGit(phase string, deadline time.Duration, dir string, args ...string) (string, error) {
	argv := append([]string{"-c", "core.hooksPath=" + e.hooksDir}, args...)
	if err := refuseEndpoints(argv); err != nil {
		e.record(OpRecord{Phase: phase, Argv: argv, Exit: -1, Outcome: "endpoint-refused"})
		return "", err
	}
	if rem := e.budget(); deadline > rem {
		deadline = rem
	}
	if deadline <= 0 {
		e.record(OpRecord{Phase: phase, Argv: argv, Exit: -1, Outcome: "budget-exhausted"})
		return "", fmt.Errorf("%w: environment wall-clock budget exhausted", ErrExec)
	}
	cmd := exec.Command(e.provider.GitPath, argv...)
	cmd.Dir = dir
	cmd.Env = e.allowEnv()
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	var out, errb bytes.Buffer
	cmd.Stdout, cmd.Stderr = &out, &errb

	start := time.Now()
	if err := cmd.Start(); err != nil {
		e.record(OpRecord{Phase: phase, Argv: argv, Exit: -1, Outcome: "start-failed"})
		return "", fmt.Errorf("%w: %v", ErrExec, err)
	}
	done := make(chan error, 1)
	go func() { done <- cmd.Wait() }()
	var werr error
	timedOut := false
	select {
	case werr = <-done:
	case <-time.After(deadline):
		timedOut = true
		// Bounded termination, subprocess tier: the whole
		// execution-owned group, not just the direct child (Q-L5-2/6).
		_ = syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
		werr = <-done
	}
	e.mu.Lock()
	e.remaining -= time.Since(start)
	e.mu.Unlock()

	rec := OpRecord{Phase: phase, Argv: argv, Exit: cmd.ProcessState.ExitCode()}
	if ru, ok := cmd.ProcessState.SysUsage().(*syscall.Rusage); ok {
		rss := int64(ru.Maxrss) // bytes on darwin, kilobytes on linux
		if runtime.GOOS == "linux" {
			rss *= 1024
		}
		rec.MaxRSSByte = rss // observed accounting, never a bound
	}
	switch {
	case timedOut:
		rec.Outcome = "timeout"
		e.record(rec)
		return "", fmt.Errorf("%w: timeout after %s: git %s", ErrExec, deadline, strings.Join(args, " "))
	case werr != nil:
		rec.Outcome = "exit-error"
		e.record(rec)
		return "", fmt.Errorf("%w: git %s: %v: %s", ErrExec, strings.Join(args, " "), werr, firstLineOf(errb.String()))
	default:
		rec.Outcome = "ok"
		e.record(rec)
		return out.String(), nil
	}
}

// sealWorkspace makes the task-content tree read-only (dirs 0555,
// regular files 0444; symlinks untouched — chmod would follow them).
// The .git subtree is skipped: it is provider mechanism state, deny-
// listed from every tool mutation and excluded from egress, and git's
// own read commands need their opportunistic index writes. Errors are
// best-effort here; egress and teardown verification catch what
// matters.
func (e *Env) sealWorkspace() {
	root := e.trace.Workspace.Root
	if root == "" {
		return
	}
	_ = filepath.WalkDir(root, func(p string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() && d.Name() == ".git" && p != root {
			return filepath.SkipDir
		}
		info, ierr := d.Info()
		if ierr != nil || info.Mode()&os.ModeSymlink != 0 {
			return nil
		}
		if d.IsDir() {
			_ = os.Chmod(p, 0o555)
		} else {
			_ = os.Chmod(p, 0o444)
		}
		return nil
	})
}

// unsealWorkspace restores write permission so teardown's RemoveAll
// can do its job; failures surface as TEARDOWN_ANOMALOUS.
func (e *Env) unsealWorkspace() {
	if e.baseDir == "" {
		return
	}
	_ = filepath.WalkDir(e.baseDir, func(p string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		info, ierr := d.Info()
		if ierr != nil || info.Mode()&os.ModeSymlink != 0 {
			return nil
		}
		if d.IsDir() {
			_ = os.Chmod(p, 0o755)
		} else {
			_ = os.Chmod(p, 0o644)
		}
		return nil
	})
}

func (e *Env) record(op OpRecord) {
	e.mu.Lock()
	e.trace.Ops = append(e.trace.Ops, op)
	e.mu.Unlock()
}

func firstLineOf(s string) string {
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		s = s[:i]
	}
	if len(s) > 200 {
		s = s[:200]
	}
	return s
}

// Teardown drives the environment to a verified terminal. It is
// never refusable in a way that retains a workspace (M1 security
// review MED-5): an ACTIVE environment is force-sealed as
// caller-abort first — teardown always occurs (D-L5-9). DESTROYED
// requires the host-cleanliness assertions to pass; anything else is
// TEARDOWN_ANOMALOUS — uncertainty is never converted into a success
// assertion (Q-L5-12).
func (e *Env) Teardown() State {
	e.mu.Lock()
	if e.state == StateActive {
		_ = e.sealLocked(SealCallerAbort)
	}
	switch e.state {
	case StateDestroyed, StateTeardownAnomalous:
		st := e.state
		e.mu.Unlock()
		return st // already terminal — idempotent
	case StateTeardown:
	default:
		if err := e.transitionLocked(StateTeardown, "teardown"); err != nil {
			st := e.state
			e.mu.Unlock()
			return st
		}
	}
	e.mu.Unlock()
	return e.teardown()
}

func (e *Env) teardown() State {
	verified := true
	var removeErr error
	if e.baseDir != "" { // nothing was created before the failure
		e.unsealWorkspace()
		removeErr = os.RemoveAll(e.baseDir)
		verified = removeErr == nil
		if verified {
			if _, err := os.Stat(e.baseDir); !os.IsNotExist(err) {
				verified = false // asserted, not assumed
			}
		}
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	if verified {
		e.trace.TeardownVerified = true
		_ = e.transitionLocked(StateDestroyed, "verified-clean")
		return e.state
	}
	reason := "host-state-unverified"
	if removeErr != nil {
		reason += ": " + removeErr.Error()
	}
	_ = e.transitionLocked(StateTeardownAnomalous, reason)
	return e.state
}
