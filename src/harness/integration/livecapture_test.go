package integration

// Live capture: one remediate-dependency walk under the test anchor with a
// REAL model endpoint, capturing every provider request and raw response
// to a directory (the record plane stores a projection of the model turn,
// never the raw provider bytes — D-L7-11 reconstructs what the MODEL SAW,
// not what the provider returned). Diagnostic only; skipped unless
// LIVECAPTURE_DIR is set. Never an evidence path: nothing here is anchored
// by a production deployment. Added 2026-09-27 for the I-M5 finding.

import (
	stdctx "context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/tofchaliss/themis-ai-runtime/src/harness/runtime/model"
	"github.com/tofchaliss/themis-ai-runtime/src/harness/skills"
)

type loggingModel struct {
	inner model.Interface
	dir   string
	n     int
}

func (l *loggingModel) Name() string { return l.inner.Name() }
func (l *loggingModel) Execute(ctx stdctx.Context, req model.ExecutionRequest) (*model.ExecutionResponse, error) {
	l.n++
	rb, _ := json.MarshalIndent(req, "", " ")
	_ = os.WriteFile(filepath.Join(l.dir, fmt.Sprintf("turn-%02d-request.json", l.n)), rb, 0o644)
	resp, err := l.inner.Execute(ctx, req)
	if err != nil {
		_ = os.WriteFile(filepath.Join(l.dir, fmt.Sprintf("turn-%02d-error.txt", l.n)), []byte(err.Error()), 0o644)
		return resp, err
	}
	_ = os.WriteFile(filepath.Join(l.dir, fmt.Sprintf("turn-%02d-raw.json", l.n)), resp.Provenance.Raw, 0o644)
	return resp, err
}

func TestLiveCapture(t *testing.T) {
	dir := os.Getenv("LIVECAPTURE_DIR")
	if dir == "" {
		t.Skip("LIVECAPTURE_DIR unset")
	}
	name := os.Getenv("LIVECAPTURE_MODEL")
	if name == "" {
		name = "qwen2.5:7b"
	}
	skillName := os.Getenv("LIVECAPTURE_SKILL")
	if skillName == "" {
		skillName = "remediate-dependency@6"
	}
	lm := &loggingModel{inner: model.NewOllamaChat("http://localhost:11434"), dir: dir}
	w, err := newWorld(t, lm, worldOpts{withDoor: true, serveFinding: true}, func(a map[string]any) { a["models"] = []any{name}; a["skills"] = []any{skillName} })
	if err != nil {
		t.Fatal(err)
	}
	_ = os.MkdirAll(filepath.Join(w.base, "state"), 0o755)
	path, err := skills.Instantiate(filepath.Join(w.root, "policies/skills/catalog.json"), skillName, skills.Request{
		TaskID: "capture-1", Repo: "demo-vuln-app", PinnedSHA: w.sha, Commission: "c0a8e3d6-5b1e-4f5a-9d3e-2b4c6a8e0f12",
		Inputs:        map[string]any{"finding": demoFindingID, "dependency": "vulnerable-dep", "advisory": "ADV-2026-1", "target-version": "v2"},
		WallDeadlineS: 540,
		Deployment: skills.Deployment{Model: name, TurnTimeoutSec: 180,
			RegistryPath: filepath.Join(w.root, "policies/tools/registry-v5.json"), ExecCeilingPath: filepath.Join(w.base, "execution-ceiling.json"),
			StateRoot: filepath.Join(w.base, "state"), ArtifactDir: filepath.Join(w.base, "artifacts"), WorkspaceRoot: filepath.Join(w.base, "provider")},
		OutDir: filepath.Join(w.base, "envelopes"),
	})
	if err != nil {
		t.Fatal(err)
	}
	res, err := w.o.SubmitTask(path)
	t.Logf("result: %+v err=%v", res, err)
	evs, _ := w.sroot.ReadEvents("capture-1")
	for _, e := range evs {
		b := string(e.Body)
		if len(b) > 200 {
			b = b[:200]
		}
		if e.Class == "l5-op" || e.Class == "l2-delivery" {
			continue
		}
		t.Logf("%d %s %s", e.Seq, e.Class, b)
	}
}
