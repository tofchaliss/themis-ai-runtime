package execution

import (
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tofchaliss/themis/tools"
)

func provisioned(t *testing.T) (*Env, *WorkspaceExecutionCeiling, *ProvisionSpec, string) {
	t.Helper()
	mirrorRoot, repo, sha := mkMirror(t)
	ceiling := testCeiling(t, mirrorRoot)
	spec := testSpec(t, repo, sha, "")
	p, err := NewLocalProvider(gitBin(t), t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	env, err := p.Provision(ceiling, spec)
	if err != nil {
		t.Fatal(err)
	}
	return env, ceiling, spec, sha
}

// The full egress path: mutate → seal clean → egress → acknowledged
// artifact whose manifest proves exactly what changed against the
// pinned base.
func TestEgressAcknowledged(t *testing.T) {
	env, ceiling, spec, sha := provisioned(t)
	ws := env.Workspace()
	// The task's effect: one modification, one addition, one
	// deletion, one symlink (recorded, never followed).
	if err := os.WriteFile(filepath.Join(ws.Root, "parser.go"), []byte("package parser // CHANGED\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(ws.Root, "new.go"), []byte("package neu\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(filepath.Join(ws.Root, "README.md")); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("/etc/passwd", filepath.Join(ws.Root, "link")); err != nil {
		t.Fatal(err)
	}
	store, err := NewArtifactStore(filepath.Join(t.TempDir(), "store"))
	if err != nil {
		t.Fatal(err)
	}
	if err := env.Seal(SealTaskComplete); err != nil {
		t.Fatal(err)
	}
	addr, err := env.Egress(ceiling, spec, store)
	if err != nil {
		t.Fatalf("egress failed: %v", err)
	}
	if env.State() != StateAcknowledged {
		t.Fatalf("expected ACKNOWLEDGED, got %s", env.State())
	}
	tr := env.Trace()
	if tr.ArtifactAddress != addr || tr.EgressOutcome != "acknowledged" {
		t.Fatalf("trace must bind the artifact address: %+v", tr.EgressOutcome)
	}
	// Retrieve by address; the manifest is self-describing.
	raw, err := store.Get(addr)
	if err != nil {
		t.Fatal(err)
	}
	var m ArtifactManifest
	if err := json.Unmarshal(raw, &m); err != nil {
		t.Fatal(err)
	}
	if m.Base.PinnedSHA != sha || m.Base.Repo != ws.Repo || m.BinaryDigest == "" {
		t.Fatalf("manifest provenance incomplete: %+v", m.Base)
	}
	byPath := map[string]ChangeEntry{}
	for _, c := range m.Changes {
		byPath[c.Path] = c
	}
	mod := byPath["parser.go"]
	if mod.Type != "modified" || mod.OldHash == "" || mod.NewHash == "" || !strings.Contains(mod.Content, "CHANGED") {
		t.Fatalf("modified entry must carry old/new hash + content: %+v", mod)
	}
	add := byPath["new.go"]
	if add.Type != "added" || add.OldHash != "" || add.NewHash == "" {
		t.Fatalf("added entry: %+v", add)
	}
	link := byPath["link"]
	if link.Type != "symlink" || link.Target != "/etc/passwd" || link.Content != "" {
		t.Fatalf("symlink must be recorded, never followed: %+v", link)
	}
	del := byPath["README.md"]
	if del.Type != "deleted" || del.OldHash == "" || del.NewHash != "" || del.Content != "" {
		t.Fatalf("deleted entry must carry only the pinned old hash: %+v", del)
	}
	if env.Teardown() != StateDestroyed {
		t.Fatal("teardown after acknowledgment must destroy")
	}
	// The artifact survives teardown: the store is outside the
	// provider boundary.
	if _, err := store.Get(addr); err != nil {
		t.Fatal("artifact must survive teardown")
	}
}

// A non-clean seal produces no artifact — reachability, not policy.
func TestEgressRequiresCleanSeal(t *testing.T) {
	env, ceiling, spec, _ := provisioned(t)
	store, _ := NewArtifactStore(filepath.Join(t.TempDir(), "s"))
	if err := env.Seal(SealFatalBreach); err != nil {
		t.Fatal(err)
	}
	if _, err := env.Egress(ceiling, spec, store); !errors.Is(err, ErrLifecycle) {
		t.Fatalf("breach seal must not egress: %v", err)
	}
	if env.Teardown() != StateDestroyed {
		t.Fatal("teardown must proceed")
	}
}

// The observed file_bytes dimension gates acceptance deterministically
// (Q-L5-9.5): a spec-narrowed bound breached during execution becomes
// a typed egress refusal.
func TestEgressObservedBreachRefuses(t *testing.T) {
	mirrorRoot, repo, sha := mkMirror(t)
	ceiling := testCeiling(t, mirrorRoot)
	spec := testSpec(t, repo, sha, `{"dimension":"file_bytes","value":8,"strength":"observed"}`)
	p, err := NewLocalProvider(gitBin(t), t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	env, err := p.Provision(ceiling, spec)
	if err != nil {
		t.Fatal(err)
	}
	ws := env.Workspace()
	if err := os.WriteFile(filepath.Join(ws.Root, "big.go"), []byte("well over eight bytes\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	store, _ := NewArtifactStore(filepath.Join(t.TempDir(), "s"))
	if err := env.Seal(SealTaskComplete); err != nil {
		t.Fatal(err)
	}
	if _, err := env.Egress(ceiling, spec, store); !errors.Is(err, ErrEgress) || !strings.Contains(err.Error(), "file_bytes") {
		t.Fatalf("observed breach must refuse egress typed: %v", err)
	}
	tr := env.Trace()
	if tr.ArtifactAddress != "" || !strings.Contains(tr.EgressOutcome, "refused") {
		t.Fatalf("no artifact on refusal; outcome typed: %+v", tr.EgressOutcome)
	}
	if env.Teardown() != StateDestroyed {
		t.Fatal("teardown must proceed after egress refusal")
	}
	if entries, _ := os.ReadDir(store.Dir); len(entries) != 0 {
		t.Fatal("refused egress must persist nothing")
	}
}

// Case-fold regression for the egress-side VCS exclusion (M2/M3
// security review HIGH sibling): .GIT aliases .git on APFS.
func TestVCSComponentCaseFold(t *testing.T) {
	for _, rel := range []string{".GIT/config", ".Git/hooks/x", "a/.GitHub/y", ".gitignore"} {
		if !vcsComponent(rel) {
			t.Errorf("%q must be excluded as a VCS component", rel)
		}
	}
	if vcsComponent("src/main.go") {
		t.Error("ordinary paths must not be excluded")
	}
}

// A staged rename emits a two-field porcelain entry; the parser must
// consume both fields as ONE change (M2/M3 security review MED).
func TestEgressStagedRename(t *testing.T) {
	env, ceiling, spec, _ := provisioned(t)
	ws := env.Workspace()
	if err := os.Rename(filepath.Join(ws.Root, "README.md"), filepath.Join(ws.Root, "RENAMED.md")); err != nil {
		t.Fatal(err)
	}
	// Stage it with raw git (test equipment, outside the vocabulary)
	// so status emits the R two-field form.
	cmd := exec.Command(gitBin(t), "add", "-A")
	cmd.Dir = ws.Root
	cmd.Env = []string{"GIT_CONFIG_GLOBAL=/dev/null", "GIT_CONFIG_SYSTEM=/dev/null"}
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("stage: %v: %s", err, out)
	}
	store, _ := NewArtifactStore(filepath.Join(t.TempDir(), "s"))
	if err := env.Seal(SealTaskComplete); err != nil {
		t.Fatal(err)
	}
	addr, err := env.Egress(ceiling, spec, store)
	if err != nil {
		t.Fatalf("rename egress must succeed: %v", err)
	}
	raw, _ := store.Get(addr)
	var m ArtifactManifest
	if err := json.Unmarshal(raw, &m); err != nil {
		t.Fatal(err)
	}
	var ren *ChangeEntry
	for i := range m.Changes {
		if m.Changes[i].Path == "RENAMED.md" {
			ren = &m.Changes[i]
		}
		if strings.Contains(m.Changes[i].Path, "EADME") && m.Changes[i].Path != "README.md" {
			t.Fatalf("mis-sliced phantom entry: %+v", m.Changes[i])
		}
	}
	if ren == nil || ren.Type != "renamed" || ren.OldPath != "README.md" || ren.OldHash == "" || ren.NewHash == "" {
		t.Fatalf("staged rename must be one typed entry with both paths and hashes: %+v", m.Changes)
	}
	env.Teardown()
}

// The encoded artifact is bounded by the ceiling's total bound —
// source bytes passing the observed check must not smuggle an
// oversized store artifact (M2/M3 security review MED).
func TestEgressEncodedArtifactBound(t *testing.T) {
	mirrorRoot, repo, sha := mkMirror(t)
	body := `{"version":1,"mirror_root":"` + mirrorRoot + `","max_wall_deadline_sec":120,"max_file_bytes":1048576,"max_total_bytes":700,"max_file_count":500,"max_mem_bytes":1073741824,"max_cpu_time_sec":600,"max_proc_count":64}`
	ceiling, err := LoadCeiling(writeTemp(t, "tiny.json", body))
	if err != nil {
		t.Fatal(err)
	}
	spec := testSpec(t, repo, sha, "")
	p, err := NewLocalProvider(gitBin(t), t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	env, err := p.Provision(ceiling, spec)
	if err != nil {
		t.Fatal(err)
	}
	// 600 source bytes: under the 700 total, but the encoded manifest
	// (provenance + hashes + escaped content) exceeds it.
	if err := os.WriteFile(filepath.Join(env.Workspace().Root, "big.txt"), []byte(strings.Repeat("x", 600)), 0o644); err != nil {
		t.Fatal(err)
	}
	store, _ := NewArtifactStore(filepath.Join(t.TempDir(), "s"))
	if err := env.Seal(SealTaskComplete); err != nil {
		t.Fatal(err)
	}
	if _, err := env.Egress(ceiling, spec, store); err == nil || !strings.Contains(err.Error(), "encoded manifest") {
		t.Fatalf("oversized encoded artifact must refuse typed: %v", err)
	}
	if entries, _ := os.ReadDir(store.Dir); len(entries) != 0 {
		t.Fatal("nothing may persist on refusal")
	}
	env.Teardown()
}

// Persistence failure fails closed (Q-L5-11, test review HIGH): no
// acknowledgment, typed outcome, no retained workspace — teardown
// proceeds and the work is lost by design.
func TestEgressPersistenceFailure(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("permission-based fixture is void under root")
	}
	env, ceiling, spec, _ := provisioned(t)
	ws := env.Workspace()
	if err := os.WriteFile(filepath.Join(ws.Root, "change.go"), []byte("package c\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	store, err := NewArtifactStore(filepath.Join(t.TempDir(), "s"))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(store.Dir, 0o555); err != nil { // Put cannot create
		t.Fatal(err)
	}
	defer os.Chmod(store.Dir, 0o755)
	if err := env.Seal(SealTaskComplete); err != nil {
		t.Fatal(err)
	}
	_, err = env.Egress(ceiling, spec, store)
	if !errors.Is(err, ErrPersist) {
		t.Fatalf("store failure must be artifact-persistence-failed: %v", err)
	}
	tr := env.Trace()
	if tr.ArtifactAddress != "" || tr.EgressOutcome != "persistence-failed" {
		t.Fatalf("no acknowledgment on persistence failure: %+v", tr.EgressOutcome)
	}
	if env.State() != StateEgressing {
		t.Fatalf("failure leaves EGRESSING for teardown: %s", env.State())
	}
	if st := env.Teardown(); st != StateDestroyed {
		t.Fatalf("teardown proceeds, workspace not retained: %s", st)
	}
	if _, err := os.Stat(ws.Root); !os.IsNotExist(err) {
		t.Fatal("workspace must not be preserved for recovery")
	}
}

// The remaining observed egress bounds (test review MED): file_count
// and disk_bytes refusals, spec-narrowed.
func TestEgressCountAndTotalBounds(t *testing.T) {
	mirrorRoot, repo, sha := mkMirror(t)
	ceiling := testCeiling(t, mirrorRoot)
	p, err := NewLocalProvider(gitBin(t), t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	// disk_bytes narrowed by spec: two files summing over the bound.
	spec := testSpec(t, repo, sha, `{"dimension":"disk_bytes","value":30,"strength":"observed"}`)
	env, err := p.Provision(ceiling, spec)
	if err != nil {
		t.Fatal(err)
	}
	ws := env.Workspace()
	for _, f := range []string{"a.txt", "b.txt"} {
		if err := os.WriteFile(filepath.Join(ws.Root, f), []byte(strings.Repeat("y", 20)), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	store, _ := NewArtifactStore(filepath.Join(t.TempDir(), "s1"))
	if err := env.Seal(SealTaskComplete); err != nil {
		t.Fatal(err)
	}
	if _, err := env.Egress(ceiling, spec, store); !errors.Is(err, ErrEgress) || !strings.Contains(err.Error(), "disk_bytes") {
		t.Fatalf("total over spec disk_bytes must refuse typed: %v", err)
	}
	env.Teardown()

	// file_count: a tiny ceiling makes three changed files too many.
	body := `{"version":1,"mirror_root":"` + mirrorRoot + `","max_wall_deadline_sec":120,"max_file_bytes":1048576,"max_total_bytes":10485760,"max_file_count":2,"max_mem_bytes":1073741824,"max_cpu_time_sec":600,"max_proc_count":64}`
	smallCeiling, err := LoadCeiling(writeTemp(t, "count.json", body))
	if err != nil {
		t.Fatal(err)
	}
	spec2 := testSpec(t, repo, sha, "")
	env2, err := p.Provision(smallCeiling, spec2)
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range []string{"c.txt", "d.txt", "e.txt"} {
		if err := os.WriteFile(filepath.Join(env2.Workspace().Root, f), []byte("z"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	store2, _ := NewArtifactStore(filepath.Join(t.TempDir(), "s2"))
	if err := env2.Seal(SealTaskComplete); err != nil {
		t.Fatal(err)
	}
	if _, err := env2.Egress(smallCeiling, spec2, store2); !errors.Is(err, ErrEgress) || !strings.Contains(err.Error(), "file_count") {
		t.Fatalf("count over ceiling must refuse typed: %v", err)
	}
	env2.Teardown()
}

// .git*-component paths are excluded-and-noted by the CONTRACT, not
// by git's own listing (Q-L5-10; test review MED).
func TestEgressVCSExcludedAndNoted(t *testing.T) {
	env, ceiling, spec, _ := provisioned(t)
	ws := env.Workspace()
	if err := os.MkdirAll(filepath.Join(ws.Root, ".github"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(ws.Root, ".github", "w.yml"), []byte("on: push\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(ws.Root, "ok.go"), []byte("package ok\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	store, _ := NewArtifactStore(filepath.Join(t.TempDir(), "s"))
	if err := env.Seal(SealTaskComplete); err != nil {
		t.Fatal(err)
	}
	addr, err := env.Egress(ceiling, spec, store)
	if err != nil {
		t.Fatal(err)
	}
	raw, _ := store.Get(addr)
	var m ArtifactManifest
	if err := json.Unmarshal(raw, &m); err != nil {
		t.Fatal(err)
	}
	for _, c := range m.Changes {
		if strings.Contains(c.Path, ".github") {
			t.Fatalf(".git* content must never ship: %+v", c)
		}
	}
	noted := false
	for _, x := range m.ExcludedVCS {
		if strings.Contains(x, ".github") {
			noted = true
		}
	}
	if !noted {
		t.Fatalf("exclusion must be noted: %+v", m.ExcludedVCS)
	}
	env.Teardown()
}

// Store contract: content-addressed, write-once, verify-on-read.
func TestArtifactStoreImmutability(t *testing.T) {
	store, err := NewArtifactStore(filepath.Join(t.TempDir(), "s"))
	if err != nil {
		t.Fatal(err)
	}
	addr, err := store.Put([]byte(`{"a":1}`))
	if err != nil {
		t.Fatal(err)
	}
	// Idempotent: same bytes, same address.
	addr2, err := store.Put([]byte(`{"a":1}`))
	if err != nil || addr2 != addr {
		t.Fatalf("content-addressed Put must be idempotent: %v", err)
	}
	// Different bytes → different address; the original is untouched.
	addr3, err := store.Put([]byte(`{"a":2}`))
	if err != nil || addr3 == addr {
		t.Fatal("different bytes must get a different address")
	}
	// Tampering is unhideable: corrupt bytes fail verify-on-read.
	if err := os.Chmod(filepath.Join(store.Dir, addr+".json"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(store.Dir, addr+".json"), []byte("tampered"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Get(addr); !errors.Is(err, ErrPersist) || !strings.Contains(err.Error(), "do not match") {
		t.Fatalf("tampered artifact must fail verification: %v", err)
	}
	// An occupied address with different bytes refuses.
	if _, err := store.Put([]byte(`{"a":1}`)); !errors.Is(err, ErrPersist) || !strings.Contains(err.Error(), "occupied") {
		t.Fatalf("occupied address with different bytes must refuse: %v", err)
	}
	// Relative store dir refuses.
	if _, err := NewArtifactStore("relative"); !errors.Is(err, ErrPersist) {
		t.Fatal("relative store dir must refuse")
	}
}

// No-push structural proof (D-L5-5): the environment has no path to a
// shared remote — push is outside the exec vocabulary, no push/commit
// capability exists in registry-v2, and the env carries no credential
// variables.
func TestNoPushStructuralProof(t *testing.T) {
	e := &Env{state: StateActive, remaining: 1 << 40}
	for _, sub := range []string{"push", "commit", "fetch", "pull", "remote"} {
		if _, err := e.ExecGit(1, sub); err == nil || !strings.Contains(err.Error(), "outside the active invocation vocabulary") {
			t.Errorf("%q must be outside the vocabulary: %v", sub, err)
		}
	}
	reg, err := tools.LoadRegistry(filepath.Join("..", "..", "..", "policies", "tools", "registry-v2.json"))
	if err != nil {
		t.Fatal(err)
	}
	for _, td := range reg.Tools {
		switch td.Name {
		case "push", "commit", "run_command", "shell", "git_push":
			t.Errorf("registry-v2 must not declare %q", td.Name)
		}
	}
	for _, kv := range (&Env{homeDir: "/h", tmpDir: "/t"}).allowEnv() {
		k := strings.SplitN(kv, "=", 2)[0]
		switch k {
		case "HOME", "TMPDIR", "LC_ALL", "GIT_CONFIG_GLOBAL", "GIT_CONFIG_SYSTEM", "GIT_CONFIG_NOSYSTEM", "GIT_TERMINAL_PROMPT":
		default:
			t.Errorf("unexpected environment variable %q — the allowlist is the contract", k)
		}
	}
}
