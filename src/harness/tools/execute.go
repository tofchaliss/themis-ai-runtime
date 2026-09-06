package tools

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	hctx "github.com/tofchaliss/themis/context"
	"github.com/tofchaliss/themis/runtime/model"
)

// confine delegates to the single confinement implementation (L2's) —
// tool target validation and context gathering must never drift.
func confine(root, rel string) (string, error) {
	return hctx.ConfinePath(root, rel)
}

// ErrorClass is the bounded, deliberately non-topological executor
// error vocabulary (Q-L4-8): authorized operation + world failure.
type ErrorClass string

const (
	ErrFileUnreadable  ErrorClass = "file-unreadable"
	ErrSeamUnavailable ErrorClass = "seam-unavailable"
	ErrOversized       ErrorClass = "oversized"
	ErrTimeout         ErrorClass = "timeout"
)

// DecisionRequiresApproval is the RESERVED governance-era decision
// state (Q-L4-6): nothing in v1 produces it; when the approval channel
// exists it renders to the model as not-available (Q-L4-5 §9). The
// constant exists so the vocabulary reservation is code, not prose.
const DecisionRequiresApproval = "requires-approval"

// Outcome is one executed call's result: evidence bytes under the
// registered trust class, or a typed error — never silent, never
// denial-shaped.
type Outcome struct {
	Evidence []byte
	ErrClass ErrorClass
	// SkippedOversized counts scan candidates omitted by size guards —
	// surfaced in the audit so caps are never silent (F4).
	SkippedOversized int
}

// Executor runs one authorized capability. v1 executors are read-only
// by construction; mutating and shell entries do not exist in the
// table (absent = impossible, not denied).
type Executor func(entry *GrantEntry, args map[string]any, target string) Outcome

// ThemisSeam is the typed read boundary (Q-L2-8 discipline). v1 stub:
// nil seam yields seam-unavailable errors.
type ThemisSeam interface {
	Read(kind, id string) ([]byte, error)
}

// NewExecutorTable builds the dispatch table and fails closed if any
// registry tool lacks an executor — phantom registry entries cannot
// exist (startup completeness check).
func NewExecutorTable(reg *Registry, seam ThemisSeam) (map[string]Executor, error) {
	table := map[string]Executor{
		"read_file":      execReadFile,
		"list_directory": execListDirectory,
		"search_code":    execSearchCode,
		"get_finding":    themisExec(seam, "finding"),
		"get_product":    themisExec(seam, "product"),
		// Mutating executors (registry-v2 era, D-L5-4): present in the
		// table, live only when a registry declares them AND a grant
		// carries the mutating visibility flag.
		"write_file":  execWriteFile,
		"apply_patch": execApplyPatch,
	}
	for _, t := range reg.Tools {
		if _, ok := table[t.Name]; !ok {
			return nil, fmt.Errorf("%w: registry tool %q has no executor", ErrDispatch, t.Name)
		}
	}
	// Prune entries the registry does not declare: the registry is the
	// vocabulary; the table cannot widen it.
	pruned := map[string]Executor{}
	for _, t := range reg.Tools {
		pruned[t.Name] = table[t.Name]
	}
	return pruned, nil
}

const maxToolEvidence = 256 * 1024

func execReadFile(entry *GrantEntry, args map[string]any, target string) Outcome {
	abs, err := confine(entry.Workspace, target)
	if err != nil {
		return Outcome{ErrClass: ErrFileUnreadable}
	}
	b, err := os.ReadFile(abs)
	if err != nil {
		return Outcome{ErrClass: ErrFileUnreadable}
	}
	if len(b) > maxToolEvidence {
		return Outcome{ErrClass: ErrOversized}
	}
	return Outcome{Evidence: b}
}

func execListDirectory(entry *GrantEntry, args map[string]any, target string) Outcome {
	abs, err := confine(entry.Workspace, target)
	if err != nil {
		return Outcome{ErrClass: ErrFileUnreadable}
	}
	entries, err := os.ReadDir(abs)
	if err != nil {
		return Outcome{ErrClass: ErrFileUnreadable}
	}
	var names []string
	total := 0
	for _, e := range entries {
		name := e.Name()
		if e.IsDir() {
			name += "/"
		}
		total += len(name) + 1
		names = append(names, name)
	}
	if total > maxToolEvidence {
		// Evidence-size cap applies to every executor (security review
		// F1): directory contents are external-untrusted and entry
		// count/name length are repo-controlled.
		return Outcome{ErrClass: ErrOversized}
	}
	sort.Strings(names)
	return Outcome{Evidence: []byte(strings.Join(names, "\n") + "\n")}
}

func execSearchCode(entry *GrantEntry, args map[string]any, target string) Outcome {
	query, _ := args["query"].(string)
	rootAbs, err := confine(entry.Workspace, target)
	if err != nil {
		return Outcome{ErrClass: ErrFileUnreadable}
	}
	var hits []string
	skipped := 0
	walkErr := filepath.WalkDir(rootAbs, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		if !d.Type().IsRegular() {
			return fmt.Errorf("non-regular file in workspace")
		}
		if info, err := d.Info(); err != nil || info.Size() > maxToolEvidence {
			skipped++
			return nil
		}
		b, err := os.ReadFile(path)
		if err != nil {
			skipped++
			return nil
		}
		if !strings.Contains(string(b), query) {
			return nil
		}
		rel, err := filepath.Rel(rootAbs, path)
		if err != nil {
			return err
		}
		hits = append(hits, filepath.ToSlash(rel))
		return nil
	})
	if walkErr != nil {
		return Outcome{ErrClass: ErrFileUnreadable}
	}
	sort.Strings(hits)
	if len(hits) > 64 {
		// No silent caps (L3 precedent): over-cap refuses as a typed
		// error the model can react to by narrowing.
		return Outcome{ErrClass: ErrOversized, SkippedOversized: skipped}
	}
	if len(hits) == 0 && skipped > 0 {
		// Empty-but-hidden: matches may exist only in size-skipped
		// files — an adversarial repo must not hide indicators behind
		// padding while the tool reports a clean empty result (F4).
		return Outcome{ErrClass: ErrOversized, SkippedOversized: skipped}
	}
	return Outcome{Evidence: []byte(strings.Join(hits, "\n") + "\n"), SkippedOversized: skipped}
}

func themisExec(seam ThemisSeam, kind string) Executor {
	return func(entry *GrantEntry, args map[string]any, target string) Outcome {
		if seam == nil {
			return Outcome{ErrClass: ErrSeamUnavailable}
		}
		b, err := seam.Read(kind, target)
		if err != nil {
			return Outcome{ErrClass: ErrSeamUnavailable}
		}
		if len(b) > maxToolEvidence {
			return Outcome{ErrClass: ErrOversized}
		}
		return Outcome{Evidence: b}
	}
}

// ToolEvidence is the executed result as L2-disciplined evidence:
// trust class from registration, byte-exact content, hash — ready for
// the tool-execution trace (Q-L2-1 amendment 4 completeness).
type ToolEvidence struct {
	Tool      string
	Kind      string
	Trust     hctx.AuthorityClass
	Producer  string
	Target    string
	Evidence  []byte
	Hash      string
	Mechanism hctx.Mechanism
}

// AuditEvent is emitted for EVERY call — allow, deny, and error alike
// (D-L4-8; the L6 shape).
type AuditEvent struct {
	Tool             string
	ArgsHash         string
	Decision         string // authorized | denied | error
	DenialClass      DenialClass
	ModelDetail      string // exact model-visible denial detail (F2: joint reconstructability)
	TracePredicate   string
	ErrClass         ErrorClass
	Target           string // requested target, recorded on every path incl. denials (F2)
	RegistryHash     string
	GrantHash        string
	ResultHash       string
	SkippedOversized int
	// Timing and executor identity join at the L6 trace-sink era —
	// explicit deferral, not omission (F2).
}

// Handle is the per-call pipeline: authorize → execute → result/denial
// as a tool message + audit event. It is stateless: call counts
// arrive in state (L7-owned). Denial content is deterministic,
// bounded, and class-only (Q-L4-5); results and errors re-enter the
// conversation as data.
func Handle(reg *Registry, grant *Grant, table map[string]Executor, call model.ToolCall, state CallState) (model.Message, *ToolEvidence, AuditEvent) {
	argsSum := hctx.EvidenceHash(call.Arguments)
	audit := AuditEvent{Tool: call.Name, ArgsHash: argsSum, RegistryHash: reg.Hash, GrantHash: grant.Hash}

	d := Authorize(reg, grant, call.Name, call.Arguments, state)
	audit.TracePredicate = d.TracePredicate
	audit.Target = d.RequestedTarget
	if !d.Allow {
		audit.Decision, audit.DenialClass, audit.ModelDetail = "denied", d.Denial, d.ModelDetail
		body := map[string]string{"denial": string(d.Denial)}
		if d.ModelDetail != "" {
			key := "field"
			if d.Denial == DenialTargetRefused {
				key = "target"
			}
			body[key] = d.ModelDetail
		}
		content, _ := json.Marshal(body)
		return model.Message{Role: model.RoleTool, Content: string(content), ToolCallID: call.ID}, nil, audit
	}

	def := reg.tool(call.Name)
	exec, ok := table[call.Name]
	if !ok {
		// Registry/table drift: fail closed with a typed error and a
		// complete audit event, never a panic (F3).
		audit.Decision, audit.ErrClass = "error", ErrSeamUnavailable
		content, _ := json.Marshal(map[string]string{"error": string(ErrSeamUnavailable)})
		return model.Message{Role: model.RoleTool, Content: string(content), ToolCallID: call.ID}, nil, audit
	}
	// Registry timeout enforced per call (arch review F1: an asserted
	// control must exist): the executor runs under its declared
	// deadline; overrun is a typed timeout error. The goroutine may
	// linger until its syscall returns — process-level isolation is
	// the L5 sandbox's job, recorded deferral.
	outCh := make(chan Outcome, 1)
	go func() { outCh <- exec(grant.entry(call.Name), d.Args, d.Target) }()
	var out Outcome
	select {
	case out = <-outCh:
	case <-time.After(time.Duration(def.TimeoutSec) * time.Second):
		out = Outcome{ErrClass: ErrTimeout}
	}
	audit.SkippedOversized = out.SkippedOversized
	if out.ErrClass != "" {
		audit.Decision, audit.ErrClass = "error", out.ErrClass
		content, _ := json.Marshal(map[string]string{"error": string(out.ErrClass)})
		return model.Message{Role: model.RoleTool, Content: string(content), ToolCallID: call.ID}, nil, audit
	}

	ev := &ToolEvidence{
		Tool: call.Name, Kind: "tool:" + call.Name, Trust: def.Trust,
		Producer: call.Name, Target: d.Target, Evidence: out.Evidence,
		Hash: hctx.EvidenceHash(out.Evidence), Mechanism: hctx.MechanismCapabilityFetch,
	}
	audit.Decision, audit.ResultHash = "authorized", ev.Hash
	return model.Message{Role: model.RoleTool, Content: string(out.Evidence), ToolCallID: call.ID}, ev, audit
}
