package seam

// L10-M4 proofs: the seam composes registry resolution, registered
// canonicalization, and the pure evaluator; the PROPOSED registration
// artifacts are internally consistent (the P0-bundle pattern);
// refusals are pre-instance and typed; hostile report content grades,
// never breaks.

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/tofchaliss/themis/runtime/model"
	"github.com/tofchaliss/themis/tools"
	"github.com/tofchaliss/themis/verification"
)

func repoRoot(t *testing.T) string {
	t.Helper()
	_, file, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(file), "..", "..", "..", "..")
}

func proposedEvaluator(t *testing.T) *Evaluator {
	t.Helper()
	root := repoRoot(t)
	reg, err := tools.LoadRegistry(filepath.Join(root, "policies/tools/registry-v4.proposed.json"))
	if err != nil {
		t.Fatal(err)
	}
	return &Evaluator{
		RegistryPath: filepath.Join(root, "policies/verification/contracts.proposed.json"),
		L4:           reg,
	}
}

func verifyCall(contract, path string) model.ToolCall {
	args, _ := json.Marshal(map[string]string{"contract": contract, "path": path})
	return model.ToolCall{ID: "c1", Name: "verify_report", Arguments: args}
}

const goodReport = `{"finding": "CVE-2026-1 in parser", "remediation": "bumped dep to 2.4.1", "evidence": "go.mod diff", "extra": "fine"}`

func TestProposedBundleIsConsistent(t *testing.T) {
	e := proposedEvaluator(t)

	vo, err := e.EvaluateCall("task-1", verifyCall("report-valid@1", "report.json"), []byte(goodReport), "l4:7", e.L4.Hash)
	if err != nil {
		t.Fatal(err)
	}
	if vo.Refused {
		t.Fatalf("proposed bundle refused its own happy path: %s", vo.RefusalReason)
	}
	if vo.ContractToken != "report-valid@1" || vo.Outcome != "PASS" {
		t.Errorf("got %s/%s", vo.ContractToken, vo.Outcome)
	}
	// The record must be a valid evaluation with class-consistent
	// reason and computed identities.
	var ev verification.Evaluation
	if err := json.Unmarshal(vo.Record, &ev); err != nil {
		t.Fatal(err)
	}
	if err := verification.CheckReasonClass(&ev); err != nil {
		t.Error(err)
	}
	if ev.ContractSHA256 == "" || ev.RawObjectID != hashBytes([]byte(goodReport)) {
		t.Error("record identities must be computed from held bytes")
	}
	if hashBytes(vo.ContractBytes) != ev.ContractSHA256 {
		t.Error("durable contract bytes must hash to the recorded contract identity (no-reopen)")
	}
}

func TestOutcomesAcrossReportShapes(t *testing.T) {
	e := proposedEvaluator(t)

	cases := map[string]struct {
		report string
		want   string
	}{
		"valid report":        {goodReport, "PASS"},
		"missing field":       {`{"finding": "x", "remediation": "y"}`, "FAIL"},
		"empty field":         {`{"finding": "x", "remediation": "y", "evidence": "  "}`, "FAIL"},
		"non-string field":    {`{"finding": 7, "remediation": "y", "evidence": "z"}`, "FAIL"},
		"not json":            {`this is not json`, "FAIL"},
		"json array":          {`[1,2,3]`, "FAIL"},
		"hostile PASS string": {`"PASS"`, "FAIL"},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			vo, err := e.EvaluateCall("t", verifyCall("report-valid@1", "r.json"), []byte(tc.report), "l4:1", e.L4.Hash)
			if err != nil {
				t.Fatal(err)
			}
			if vo.Refused {
				t.Fatalf("refused: %s", vo.RefusalReason)
			}
			if vo.Outcome != tc.want {
				t.Errorf("%s -> %s, want %s", name, vo.Outcome, tc.want)
			}
		})
	}
}

func TestPreInstanceRefusals(t *testing.T) {
	e := proposedEvaluator(t)

	t.Run("no contract named", func(t *testing.T) {
		args, _ := json.Marshal(map[string]string{"path": "r.json"})
		vo, err := e.EvaluateCall("t", model.ToolCall{Name: "verify_report", Arguments: args}, []byte(goodReport), "l4:1", e.L4.Hash)
		if err != nil || !vo.Refused {
			t.Fatalf("must refuse: %v %+v", err, vo)
		}
	})

	t.Run("unregistered contract", func(t *testing.T) {
		vo, err := e.EvaluateCall("t", verifyCall("other-contract@1", "r.json"), []byte(goodReport), "l4:1", e.L4.Hash)
		if err != nil || !vo.Refused {
			t.Fatalf("must refuse: %v %+v", err, vo)
		}
	})

	t.Run("floating reference", func(t *testing.T) {
		vo, err := e.EvaluateCall("t", verifyCall("report-valid", "r.json"), []byte(goodReport), "l4:1", e.L4.Hash)
		if err != nil || !vo.Refused {
			t.Fatalf("must refuse: %v %+v", err, vo)
		}
	})

	t.Run("capability mismatch", func(t *testing.T) {
		call := verifyCall("report-valid@1", "r.json")
		call.Name = "read_file"
		vo, err := e.EvaluateCall("t", call, []byte(goodReport), "l4:1", e.L4.Hash)
		if err != nil || !vo.Refused {
			t.Fatalf("contract bound to verify_report invoked as read_file must refuse: %v %+v", err, vo)
		}
	})

	t.Run("registry-version mismatch fails eligibility", func(t *testing.T) {
		// A contract pinning a different L4 registry version than the
		// one in force must refuse (two-source protection).
		e2 := proposedEvaluator(t)
		e2.L4.Hash = strings.Repeat("00", 32)
		vo, err := e2.EvaluateCall("t", verifyCall("report-valid@1", "r.json"), []byte(goodReport), "l4:1", e2.L4.Hash)
		if err != nil || !vo.Refused {
			t.Fatalf("must refuse on registry drift: %v %+v", err, vo)
		}
	})
}

func TestEmptyEvidenceGradesThroughCanonicalization(t *testing.T) {
	// A successful read of an empty artifact is CONTENT, not absence
	// (close architecture review L-3): it canonicalizes to
	// report_invalid and grades FAIL — verified-negative, honestly.
	e := proposedEvaluator(t)
	vo, err := e.EvaluateCall("t", verifyCall("report-valid@1", "r.json"), nil, "l4:1", e.L4.Hash)
	if err != nil {
		t.Fatal(err)
	}
	if vo.Refused {
		t.Fatalf("empty evidence is an instance fact, not a refusal: %s", vo.RefusalReason)
	}
	if vo.Outcome != "FAIL" {
		t.Errorf("empty capture -> %s, want FAIL via report_invalid", vo.Outcome)
	}
}

func TestMultiSlotContractRefusedBySeam(t *testing.T) {
	// L-2: a registrable multi-slot contract must refuse typed at this
	// seam, never degrade to a misleading INVALID.
	dir := t.TempDir()
	root := repoRoot(t)
	craw := readFileB(t, filepath.Join(root, "policies/verification/report-valid/contract.json"))
	var cdoc map[string]any
	if err := json.Unmarshal(craw, &cdoc); err != nil {
		t.Fatal(err)
	}
	slots := cdoc["evidence"].([]any)
	slots = append(slots, map[string]any{"name": "second", "kind": "artifact", "required": true, "task_bound": true})
	cdoc["evidence"] = slots
	mb, err := json.Marshal(cdoc)
	if err != nil {
		t.Fatal(err)
	}
	multi := string(mb)
	if err := os.MkdirAll(filepath.Join(dir, "report-valid"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "report-valid/contract.json"), []byte(multi), 0644); err != nil {
		t.Fatal(err)
	}
	reg := fmt.Sprintf(`{"version": 1, "entries": [
	  {"name": "report-valid", "version": 1, "contract_sha256": %q,
	   "contract_path": "report-valid/contract.json", "state": "active"}]}`,
		hashBytes([]byte(multi)))
	regPath := filepath.Join(dir, "contracts.json")
	if err := os.WriteFile(regPath, []byte(reg), 0644); err != nil {
		t.Fatal(err)
	}
	e := proposedEvaluator(t)
	e.RegistryPath = regPath
	vo, err := e.EvaluateCall("t", verifyCall("report-valid@1", "r.json"), []byte(goodReport), "l4:1", e.L4.Hash)
	if err != nil || !vo.Refused {
		t.Fatalf("multi-slot contract must refuse typed: %v %+v", err, vo)
	}
}

func readFileB(t *testing.T, path string) []byte {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func TestCanonicalizationIsPure(t *testing.T) {
	// Same bytes, same config -> same canonical result, across
	// repeated invocations (the D-L10-3 registration claim for the
	// in-process canonicalizer).
	for i := 0; i < 5; i++ {
		c1, _ := canonReport([]byte(goodReport), json.RawMessage(`{}`))
		c2, _ := canonReport([]byte(goodReport), json.RawMessage(`{}`))
		if c1 != c2 || c1 != "report_valid" {
			t.Fatalf("canonicalization unstable: %s vs %s", c1, c2)
		}
	}
}

func TestWithdrawnContractRefuses(t *testing.T) {
	dir := t.TempDir()
	root := repoRoot(t)
	craw, err := os.ReadFile(filepath.Join(root, "policies/verification/report-valid/contract.json"))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(dir, "report-valid"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "report-valid/contract.json"), craw, 0644); err != nil {
		t.Fatal(err)
	}
	reg := fmt.Sprintf(`{"version": 1, "entries": [
	  {"name": "report-valid", "version": 1, "contract_sha256": %q,
	   "contract_path": "report-valid/contract.json", "state": "withdrawn"}]}`,
		hashBytes(craw))
	regPath := filepath.Join(dir, "contracts.json")
	if err := os.WriteFile(regPath, []byte(reg), 0644); err != nil {
		t.Fatal(err)
	}
	e := proposedEvaluator(t)
	e.RegistryPath = regPath
	vo, err := e.EvaluateCall("t", verifyCall("report-valid@1", "r.json"), []byte(goodReport), "l4:1", e.L4.Hash)
	if err != nil || !vo.Refused {
		t.Fatalf("withdrawn contract must refuse: %v %+v", err, vo)
	}
}
