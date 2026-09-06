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
	"regexp"
	"strings"
	"syscall"
	"time"

	hctx "github.com/tofchaliss/themis/context"
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
			DimWallDeadlineS: StrengthEnforced, // group kill at deadline
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

// Declaration returns the provider's property declaration.
func (p *LocalProvider) Declaration() ProviderDeclaration { return p.decl }

// attestBinary records the executable's identity and refuses
// privilege-carrying or tamper-exposed binaries: setuid/setgid,
// world-writable file, world-writable containing directory
// (Q-L5-6). The digest is evidence of what was executed, not proof
// of trustworthy behavior.
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

// endpointShape refuses argv values that name a network endpoint: URL
// schemes and scp-style remotes. This is the deterministically
// testable half of denied-by-construction (Q-L5-7): a network request
// dies as a typed refusal at argv construction, never as a downstream
// git network error. Defense in depth — the invocation vocabulary
// never constructs such an argument.
var scpLike = regexp.MustCompile(`^[^/@]+@[^/@]+:`)

func refuseEndpoints(argv []string) error {
	for _, a := range argv {
		if strings.Contains(a, "://") || scpLike.MatchString(a) {
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
	mirror, err := hctx.ConfinePath(ceiling.MirrorRoot, spec.Repo)
	if err != nil {
		return nil, fmt.Errorf("%w: repository %q: %v", ErrProvision, spec.Repo, err)
	}
	att, err := attestBinary(p.GitPath)
	if err != nil {
		return nil, err
	}

	e := &Env{
		state:    StateProvisioning,
		provider: p,
		trace: Trace{
			CeilingHash: ceiling.Hash,
			SpecHash:    spec.Hash,
			Provider:    p.decl,
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

	deadline, _ := spec.Limit(DimWallDeadlineS)
	// Provisioning is execution (Q-L5-3): the same neutralized,
	// audited invocation path as active-phase ops.
	if _, err := e.runGit("provision", time.Duration(deadline.Value)*time.Second,
		p.BaseDir, "clone", "--no-hardlinks", mirror, worktree); err != nil {
		return fail(err)
	}
	if _, err := e.runGit("provision", time.Duration(deadline.Value)*time.Second,
		worktree, "-c", "advice.detachedHead=false", "checkout", "--detach", spec.PinnedSHA); err != nil {
		return fail(err)
	}
	// Post-condition verification (Q-L5-3): the checkout IS at the
	// pinned SHA — asserted, not assumed.
	head, err := e.runGit("provision", time.Duration(deadline.Value)*time.Second,
		worktree, "rev-parse", "HEAD")
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

// ExecGit runs one active-phase git operation inside the environment.
// Refused unless the environment is ACTIVE — a sealed environment
// refuses all further executions, typed (Q-L5-12).
func (e *Env) ExecGit(deadline time.Duration, args ...string) (string, error) {
	e.mu.Lock()
	if e.state != StateActive {
		st := e.state
		e.mu.Unlock()
		return "", fmt.Errorf("%w: execution refused in state %s", ErrLifecycle, st)
	}
	root := e.trace.Workspace.Root
	e.mu.Unlock()
	return e.runGit("active", deadline, root, args...)
}

// runGit is the single subprocess invocation path: pinned absolute
// binary, endpoint-refused argv, hooks neutralized, empty allowlist
// environment, execution-owned process group, group-killed at
// deadline. There is no second, more convenient path.
func (e *Env) runGit(phase string, deadline time.Duration, dir string, args ...string) (string, error) {
	argv := append([]string{"-c", "core.hooksPath=" + e.hooksDir}, args...)
	if err := refuseEndpoints(argv); err != nil {
		e.record(OpRecord{Phase: phase, Argv: argv, Exit: -1, Outcome: "endpoint-refused"})
		return "", err
	}
	cmd := exec.Command(e.provider.GitPath, argv...)
	cmd.Dir = dir
	cmd.Env = e.allowEnv()
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	var out, errb bytes.Buffer
	cmd.Stdout, cmd.Stderr = &out, &errb

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
	rec := OpRecord{Phase: phase, Argv: argv, Exit: cmd.ProcessState.ExitCode()}
	if ru, ok := cmd.ProcessState.SysUsage().(*syscall.Rusage); ok {
		rec.MaxRSSByte = int64(ru.Maxrss) // observed accounting, never a bound
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

// Teardown drives SEALED/ACKNOWLEDGED (or failed states already in
// TEARDOWN) to a verified terminal. DESTROYED requires the
// host-cleanliness assertions to pass; anything else is
// TEARDOWN_ANOMALOUS — uncertainty is never converted into a success
// assertion (Q-L5-12).
func (e *Env) Teardown() State {
	e.mu.Lock()
	if e.state != StateTeardown {
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
