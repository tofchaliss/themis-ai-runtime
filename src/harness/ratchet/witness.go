package ratchet

// Witness verification — D-G2-1 enforcement. Storage proves bytes;
// events prove establishment: an L6 object is a fact of kind F only
// when a committed event of F's minting class names it. The
// fact-kind → witness-class table below is architecture (D-G2-1
// Q-G2-3), implemented as closed reviewed code, never data. The
// benchmark plane's witness is its own admission mechanism — the
// verdict + digest + location predicate (the gatePassed contract) —
// and no L6 event is invented for it (owner instruction at lock).

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/tofchaliss/themis/state"
)

// witnessClasses maps each L6-plane fact kind to the event class
// whose committed events establish it (D-G2-1 Q-G2-2). Model turns
// (EvModelTurn) are deliberately absent: model-authored bytes are
// not a fact kind. L11 artifacts have no witnessing event at all —
// unwitnessed is unestablishable.
var witnessClasses = map[string]string{
	"l10_evaluation_record": state.EvVerification,
	"l6_execution_record":   state.EvL4Audit,
}

// verifyL6Witness resolves the fact's claimed witness in the event
// plane and verifies the full chain: event exists → class matches
// the kind's witnessing class → the event names this ObjectID →
// (optionally) the minting tool matches the selector's pin. Returns
// a refusal-reason detail on failure; empty string = witnessed.
func verifyL6Witness(root *state.Root, source, objectID, taskID string, eventSeq int64, toolPin string) string {
	class, ok := witnessClasses[source]
	if !ok {
		return fmt.Sprintf("source %q has no witnessing event class — not an L6-plane fact kind", source)
	}
	events, err := root.ReadEvents(taskID)
	if err != nil {
		return fmt.Sprintf("witness task %q unreadable in the event plane", taskID)
	}
	for _, ev := range events {
		if ev.Seq != eventSeq {
			continue
		}
		if ev.Class != class {
			return fmt.Sprintf("witness event %s/%d has class %q — kind %q is established only by %q", taskID, eventSeq, ev.Class, source, class)
		}
		switch source {
		case "l10_evaluation_record":
			// The anti-shadowing rule promoted to the general law:
			// the record is the one the event body EXPLICITLY names.
			var body struct {
				Record string `json:"record"`
			}
			if err := json.Unmarshal(ev.Body, &body); err != nil || body.Record != objectID {
				return fmt.Sprintf("witness event %s/%d does not name object %s as its evaluation record", taskID, eventSeq, objectID)
			}
		case "l6_execution_record":
			named := false
			for _, ref := range ev.Refs {
				if ref.ID == objectID {
					named = true
					break
				}
			}
			if !named {
				return fmt.Sprintf("witness event %s/%d does not reference object %s", taskID, eventSeq, objectID)
			}
			if toolPin != "" {
				var body struct {
					Tool string `json:"Tool"`
				}
				if err := json.Unmarshal(ev.Body, &body); err != nil || body.Tool != toolPin {
					return fmt.Sprintf("witness event %s/%d was not minted by the pinned capability %q", taskID, eventSeq, toolPin)
				}
			}
		}
		return "" // witnessed
	}
	return fmt.Sprintf("no committed event at %s/%d — the claimed witness does not exist", taskID, eventSeq)
}

// benchSources: fact kinds established by the benchmark plane's own
// admission mechanism.
var benchSources = map[string]bool{
	"benchmark_validated_score": true,
	"gate_verdict":              true,
}

// benchVerdict mirrors the gate's recorded verdict shape (the
// gatePassed consumption contract — admission re-verified at every
// consumer, the router precedent).
type benchVerdict struct {
	Model        string `json:"model"`
	Current      string `json:"current"`
	Pass         bool   `json:"pass"`
	ScoresDigest string `json:"scores_digest"`
}

// BenchRunDigest fingerprints a run's validation score files exactly
// as the gate does (sorted basenames + contents) — the algorithm is
// a contract with gate.RunDigest / service.gatePassed.
func BenchRunDigest(dir string) (string, error) {
	files, err := filepath.Glob(filepath.Join(dir, "*.json"))
	if err != nil {
		return "", err
	}
	sort.Strings(files)
	h := sha256.New()
	for _, file := range files {
		data, err := os.ReadFile(file)
		if err != nil {
			return "", err
		}
		h.Write([]byte(filepath.Base(file)))
		h.Write([]byte{'\n'})
		h.Write(data)
		h.Write([]byte{'\n'})
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

// verifyBenchWitness applies the external-plane witness: the ref
// must live in the canonical layout, the run's verdict must exist
// with Pass=true, the recorded scores digest must match a fresh
// recomputation over the run directory, and the referenced bytes
// must match the supplied value. Returns (runID, refusalDetail).
//
//	benchmark_validated_score: ref = validation/<date>/<model...>/<file>.json
//	gate_verdict:              ref = gate/<date>/<model...>/verdict.json
func verifyBenchWitness(benchRoot, source, ref string, value []byte) (string, string) {
	if benchRoot == "" {
		return "", "no benchmark root supplied — external-plane facts cannot be witnessed"
	}
	parts := strings.Split(filepath.ToSlash(ref), "/")
	var wantPlane string
	switch source {
	case "benchmark_validated_score":
		wantPlane = "validation"
	case "gate_verdict":
		wantPlane = "gate"
	default:
		return "", fmt.Sprintf("source %q is not a benchmark-plane fact kind", source)
	}
	if len(parts) < 4 || parts[0] != wantPlane {
		return "", fmt.Sprintf("ref %q is outside the canonical %s/<date>/<model>/ layout", ref, wantPlane)
	}
	for _, seg := range parts {
		if seg == "" || seg == "." || seg == ".." {
			return "", fmt.Sprintf("ref %q contains an invalid path segment", ref)
		}
	}
	date := parts[1]
	modelName := strings.Join(parts[2:len(parts)-1], "/")
	file := parts[len(parts)-1]
	if source == "gate_verdict" && file != "verdict.json" {
		return "", "gate_verdict facts reference the verdict file itself"
	}
	// The verdict is the plane's admission record: required for BOTH
	// kinds — an unadmitted (or failing) run establishes nothing.
	vb, err := os.ReadFile(filepath.Join(benchRoot, "gate", date, modelName, "verdict.json"))
	if err != nil {
		return "", fmt.Sprintf("run %s/%s has no recorded verdict — unadmitted runs establish nothing", date, modelName)
	}
	var v benchVerdict
	if err := json.Unmarshal(vb, &v); err != nil {
		return "", "recorded verdict is unreadable"
	}
	if !v.Pass {
		return "", fmt.Sprintf("run %s/%s verdict is not passing — the plane did not admit it", date, modelName)
	}
	if v.Model != modelName || v.Current != date {
		return "", "verdict identity does not match the referenced run location"
	}
	digest, err := BenchRunDigest(filepath.Join(benchRoot, "validation", date, modelName))
	if err != nil {
		return "", "run score files unreadable for digest recomputation"
	}
	if digest != v.ScoresDigest {
		return "", "score files do not match the digest the verdict admitted — rewritten after gating"
	}
	stored, err := os.ReadFile(filepath.Join(benchRoot, filepath.FromSlash(filepath.ToSlash(ref))))
	if err != nil {
		return "", fmt.Sprintf("ref %q did not resolve under the benchmark root", ref)
	}
	if string(stored) != string(value) {
		return "", fmt.Sprintf("supplied value bytes differ from the bytes at %q", ref)
	}
	return "bench:" + date + "/" + modelName, ""
}
