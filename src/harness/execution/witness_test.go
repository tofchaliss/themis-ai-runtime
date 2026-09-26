package execution

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// recWitness is the test double for the L5 handle: it records every
// emission in order and can be told to refuse ops.
type recWitness struct {
	items     []string
	refuseOps bool
}

func (w *recWitness) Transition(from, to, reason string) error {
	w.items = append(w.items, "t:"+from+">"+to)
	return nil
}
func (w *recWitness) Op(phase string, argv []string, exit int, outcome string) error {
	if w.refuseOps {
		return errors.New("refused by witness")
	}
	w.items = append(w.items, fmt.Sprintf("op:%s:%s", phase, outcome))
	return nil
}
func (w *recWitness) Egress(outcome, address string, total, count int64) error {
	tag := "egress:" + outcome
	if address != "" {
		tag += ":addr"
	}
	w.items = append(w.items, tag)
	return nil
}

func mustWitness(t *testing.T, env *Env) *recWitness {
	t.Helper()
	if env == nil {
		return nil
	}
	w := &recWitness{}
	if err := env.Attach(w); err != nil {
		t.Fatalf("attach witness: %v", err)
	}
	return w
}

// W-M2, D-W-2/D-W-3: the whole machine and every op are witnessed, in
// order, with the per-edge ordering — provisioning after effect, seal
// after effect, EGRESSING before effect, egress op after the store
// acknowledged and before return, ACKNOWLEDGED after, TEARDOWN before,
// the terminal edge after. Provisioning-time emissions buffered before
// the record existed replay first, in order.
func TestFullMachineWitnessed(t *testing.T) {
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
	if env.Witnessed() {
		t.Fatal("no witness before attach")
	}
	// Unwitnessed: governed work refuses, closed.
	if _, err := env.ExecGit(30*time.Second, "status"); !errors.Is(err, ErrUnwitnessed) {
		t.Fatalf("active op in an unwitnessed env: %v", err)
	}
	if err := env.Seal(SealTaskComplete); !errors.Is(err, ErrUnwitnessed) {
		t.Fatalf("clean seal in an unwitnessed env: %v", err)
	}
	w := mustWitness(t, env)
	if err := env.Attach(w); err == nil {
		t.Fatal("second attach must refuse")
	}
	// Replayed provisioning: the clone ops precede the ACTIVE edge.
	replay := strings.Join(w.items, " ")
	if !strings.Contains(replay, "op:provision:ok") || !strings.HasSuffix(replay, "t:PROVISIONING>ACTIVE") {
		t.Fatalf("replayed provisioning emissions: %v", w.items)
	}
	nProv := len(w.items)
	ws := env.Workspace()
	if err := os.WriteFile(filepath.Join(ws.Root, "parser.go"), []byte("package parser // changed\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := env.ExecGit(30*time.Second, "status"); err != nil {
		t.Fatal(err)
	}
	if err := env.Seal(SealTaskComplete); err != nil {
		t.Fatal(err)
	}
	store, err := NewArtifactStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	addr, err := env.Egress(ceiling, spec, store)
	if err != nil {
		t.Fatal(err)
	}
	if st := env.Teardown(); st != StateDestroyed {
		t.Fatalf("teardown: %s", st)
	}
	// The egress phase runs its own governed git ops (diff, listing);
	// each is witnessed between the EGRESSING edge and the egress
	// acknowledgement. Assert their placement, then the edge order.
	var got []string
	egressOps := 0
	afterEgressing, beforeAck := false, true
	for _, it := range w.items[nProv:] {
		if strings.HasPrefix(it, "op:egress:") {
			egressOps++
			if !afterEgressing || !beforeAck {
				t.Fatalf("egress-phase op outside the EGRESSING…acknowledged window: %v", w.items[nProv:])
			}
			continue
		}
		if it == "t:SEALED>EGRESSING" {
			afterEgressing = true
		}
		if it == "egress:acknowledged:addr" {
			beforeAck = false
		}
		got = append(got, it)
	}
	if egressOps == 0 {
		t.Fatal("egress-phase ops were not witnessed")
	}
	want := []string{"op:active:ok", "t:ACTIVE>SEALED", "t:SEALED>EGRESSING", "egress:acknowledged:addr", "t:EGRESSING>ACKNOWLEDGED", "t:ACKNOWLEDGED>TEARDOWN", "t:TEARDOWN>DESTROYED"}
	if strings.Join(got, "|") != strings.Join(want, "|") {
		t.Fatalf("witness order:\n got %v\nwant %v", got, want)
	}
	if addr == "" || env.Trace().WitnessErr != "" {
		t.Fatalf("addr %q witnessErr %q", addr, env.Trace().WitnessErr)
	}
}

// A refused egress is witnessed with no address and can never read as
// production; teardown edges still follow.
func TestRefusedEgressWitnessed(t *testing.T) {
	env, ceiling, spec, _ := provisioned(t)
	w := env.witness.(*recWitness)
	if err := env.Seal(SealTaskComplete); err != nil {
		t.Fatal(err)
	}
	// A store inside the environment boundary is refused before any read.
	inside, err := NewArtifactStore(filepath.Join(env.baseDir, "store"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := env.Egress(ceiling, spec, inside); err == nil {
		t.Fatal("egress into the environment must refuse")
	}
	items := strings.Join(w.items, " ")
	if !strings.Contains(items, "egress:refused: store inside provider boundary") || strings.Contains(items, ":addr") {
		t.Fatalf("refused egress witness: %v", w.items)
	}
	env.Teardown()
}

// A witness that refuses an op fails the op closed even though the
// command succeeded (the record must hold no gap where an op ran).
func TestWitnessRefusalFailsOpClosed(t *testing.T) {
	env, _, _, _ := provisioned(t)
	w := env.witness.(*recWitness)
	w.refuseOps = true
	out, err := env.ExecGit(30*time.Second, "rev-parse", "HEAD")
	if err == nil || !errors.Is(err, ErrExec) || !strings.Contains(err.Error(), "witness refused") || out != "" {
		t.Fatalf("op with a refusing witness: out=%q err=%v", out, err)
	}
	w.refuseOps = false
	env.Teardown()
}

// Teardown never waits on a witness: an unwitnessed environment (the
// record was never created) tears down, and its edges are simply not
// in any record.
func TestTeardownWithoutWitness(t *testing.T) {
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
	if st := env.Teardown(); st != StateDestroyed {
		t.Fatalf("teardown: %s", st)
	}
	if env.Witnessed() {
		t.Fatal("still unwitnessed")
	}
}
