package tools

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

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
)

// Outcome is one executed call's result: evidence bytes under the
// registered trust class, or a typed error — never silent, never
// denial-shaped.
type Outcome struct {
	Evidence []byte
	ErrClass ErrorClass
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
	for _, e := range entries {
		name := e.Name()
		if e.IsDir() {
			name += "/"
		}
		names = append(names, name)
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
	walkErr := filepath.WalkDir(rootAbs, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		if !d.Type().IsRegular() {
			return fmt.Errorf("non-regular file in workspace")
		}
		if info, err := d.Info(); err != nil || info.Size() > maxToolEvidence {
			return nil
		}
		b, err := os.ReadFile(path)
		if err != nil || !strings.Contains(string(b), query) {
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
		return Outcome{ErrClass: ErrOversized}
	}
	return Outcome{Evidence: []byte(strings.Join(hits, "\n") + "\n")}
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
	Tool           string
	ArgsHash       string
	Decision       string // authorized | denied | error
	DenialClass    DenialClass
	TracePredicate string
	ErrClass       ErrorClass
	Target         string
	RegistryHash   string
	GrantHash      string
	ResultHash     string
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
	if !d.Allow {
		audit.Decision, audit.DenialClass = "denied", d.Denial
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
	out := table[call.Name](grant.entry(call.Name), d.Args, d.Target)
	audit.Target = d.Target
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
