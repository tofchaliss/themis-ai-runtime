package tools

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
)

// DenialClass is the model-visible denial vocabulary (Q-L4-5, locked):
// four classes; unknown ∪ not-granted ∪ quota-exhausted collapse to
// not-available; full mechanics are trace-only.
type DenialClass string

const (
	DenialNone          DenialClass = ""
	DenialNotAvailable  DenialClass = "not-available"
	DenialInvalidArgs   DenialClass = "invalid-args"
	DenialTargetRefused DenialClass = "target-refused"
)

// Decision is the authorization outcome. ModelDetail is the bounded,
// model-visible fragment (field name or canonicalized target echo —
// only information the model already supplied or already holds via
// the ToolDef). TracePredicate names the exact failed check and never
// reaches the model.
type Decision struct {
	Allow           bool
	Denial          DenialClass
	ModelDetail     string
	TracePredicate  string
	Tool            string
	Args            map[string]any // strictly validated, only when Allow
	Target          string         // validated target instance, only when Allow
	RequestedTarget string         // model-supplied target echo, every path (F2)
	RegistryHash    string
	GrantHash       string
}

// CallState is L7-supplied execution history: L4 owns no counters
// (Q-L4-7 — Authorize stays pure and input-bounded).
type CallState struct {
	Calls map[string]int // per-tool completed calls
	Total int
}

const maxArgsBytes = 16 * 1024
const maxTargetEcho = 256

var themisIDSyntax = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._:-]{0,127}$`)

// Authorize implements the locked decision table. Check order is the
// anti-oracle invariant (Q-L4-5 §3): availability (registry ∩ grant ∩
// quota) is established BEFORE any argument inspection, so an
// unavailable capability reveals nothing about schemas, existence, or
// limits. Purity, amended per architecture review F5: Authorize is
// input-bounded EXCEPT workspace target confinement, which necessarily
// consults live filesystem state (symlink resolution) — lexical-only
// confinement would be bypassable; TOCTOU is the L5 sandbox's problem.
func Authorize(reg *Registry, grant *Grant, toolName string, rawArgs json.RawMessage, state CallState) Decision {
	d := Decision{Tool: toolName, RegistryHash: reg.Hash, GrantHash: grant.Hash}
	deny := func(class DenialClass, detail, predicate string) Decision {
		d.Allow, d.Denial, d.ModelDetail, d.TracePredicate = false, class, detail, predicate
		return d
	}

	// 1. Availability: registry ∩ grant ∩ quota → not-available.
	def := reg.tool(toolName)
	if def == nil {
		return deny(DenialNotAvailable, "", "unknown-tool: not in registry")
	}
	entry := grant.entry(toolName)
	if entry == nil {
		return deny(DenialNotAvailable, "", "not-granted: absent from execution grant")
	}
	if def.Mutating && !entry.Mutating {
		// A mutating capability granted without the explicit mutating
		// visibility flag is not granted (D-L5-4): the grant is the
		// review surface, and invisible write authority must be
		// structurally impossible. Availability-class denial: zero
		// model detail.
		return deny(DenialNotAvailable, "", "mutating-not-visible: grant lacks mutating flag for mutating tool")
	}
	if state.Calls[toolName] >= entry.MaxCalls {
		return deny(DenialNotAvailable, "", fmt.Sprintf("quota-exhausted: tool cap %d reached", entry.MaxCalls))
	}
	if state.Total >= grant.TotalMaxCalls {
		return deny(DenialNotAvailable, "", fmt.Sprintf("quota-exhausted: total cap %d reached", grant.TotalMaxCalls))
	}

	// 2. Strict argument validation (only now — the capability is
	// available, so its schema is already model-known via the ToolDef).
	if len(rawArgs) > maxArgsBytes {
		return deny(DenialInvalidArgs, "", fmt.Sprintf("args exceed %d bytes", maxArgsBytes))
	}
	args := map[string]any{}
	if len(rawArgs) > 0 {
		decoder := json.NewDecoder(strings.NewReader(string(rawArgs)))
		decoder.UseNumber()
		if err := decoder.Decode(&args); err != nil {
			return deny(DenialInvalidArgs, "", "args not a JSON object: "+err.Error())
		}
		if decoder.More() {
			return deny(DenialInvalidArgs, "", "trailing content after args object")
		}
	}
	byName := map[string]ParamDef{}
	for _, p := range def.Params {
		byName[p.Name] = p
	}
	for k := range args {
		if _, ok := byName[k]; !ok {
			// Unknown field ⇒ whole-call rejection, recorded (the L1
			// Scenario-3 rule: no silent stripping, ever).
			return deny(DenialInvalidArgs, bound(k), "unknown-field: "+k)
		}
	}
	for _, p := range def.Params {
		v, present := args[p.Name]
		if !present {
			if p.Required {
				return deny(DenialInvalidArgs, p.Name, "missing-required: "+p.Name)
			}
			continue
		}
		if !typeOK(p.Type, v) {
			return deny(DenialInvalidArgs, p.Name, fmt.Sprintf("wrong-type: %s wants %s", p.Name, p.Type))
		}
	}

	// 3. Target validation: class rule (registry) × instance scope
	// (grant). The model supplies the requested target; the grant
	// supplies the binding; the model cannot override the binding.
	target := ""
	for _, p := range def.Params {
		if p.Target {
			if s, ok := args[p.Name].(string); ok {
				target = s
			}
		}
	}
	d.RequestedTarget = bound(target)
	switch def.Target {
	case TargetWorkspacePath:
		if entry.Workspace == "" {
			// Operator/config error, not a model target fault: the
			// capability is effectively unusable in this execution —
			// actionable state is not-available (F9).
			return deny(DenialNotAvailable, "", "grant-missing-workspace-binding")
		}
		if _, err := confine(entry.Workspace, target); err != nil {
			return deny(DenialTargetRefused, bound(target), "confinement: "+err.Error())
		}
	case TargetThemisID:
		if !themisIDSyntax.MatchString(target) {
			return deny(DenialTargetRefused, bound(target), "themis-id-shape")
		}
		inScope := len(entry.ThemisScope) == 0 && false // empty scope grants nothing
		for _, prefix := range entry.ThemisScope {
			if strings.HasPrefix(target, prefix) {
				inScope = true
				break
			}
		}
		if !inScope {
			return deny(DenialTargetRefused, bound(target), "themis-id-outside-grant-scope")
		}
	case TargetNone:
	}

	d.Allow, d.Args, d.Target = true, args, target
	d.TracePredicate = "authorized"
	return d
}

func typeOK(t ParamType, v any) bool {
	switch t {
	case ParamString:
		_, ok := v.(string)
		return ok
	case ParamInteger:
		n, ok := v.(json.Number)
		if !ok {
			return false
		}
		_, err := n.Int64()
		return err == nil
	case ParamBoolean:
		_, ok := v.(bool)
		return ok
	}
	return false
}

// bound truncates a model-supplied echo to the locked disclosure
// bound; it never adds information the model did not send.
func bound(s string) string {
	if len(s) > maxTargetEcho {
		return s[:maxTargetEcho]
	}
	return s
}
