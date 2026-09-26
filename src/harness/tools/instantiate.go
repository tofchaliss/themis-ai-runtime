package tools

// Grant instantiation (D-SA-4, openspec/changes/archive/2026-09-23-l9-l7-skill-admission-
// identity): the deterministic relation between an EFFECTIVE grant and
// the Governance-approved grant TEMPLATE it claims to instantiate. L4
// owns the grant vocabulary, so it owns this relation; L7 applies it at
// anchored assembly and L9 applies it at instantiation. It is grant
// algebra — the grantWithinCeiling shape — never skill semantics.
//
// Field-specific: caller-narrowable numeric fields may narrow within
// bounds; Skill-fixed structural fields must remain EQUAL. In
// particular template_scope is set-equality, not subset: it is not a
// resource quota but the set of delegation templates the reviewed
// composition permits, and letting a caller remove entries would make a
// Class-1 structural field caller-controlled (owner, A-SA-3).

import (
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
	"strings"

	"github.com/tofchaliss/themis-ai-runtime/src/harness/internal/strictjson"
)

// skillRefSyntax: an exact delegation-template reference, name@version
// — the template_scope entry shape (C-L8-15 G: exact equality at L4,
// no prefixes, so the loader refuses any non-exact entry).
var skillRefSyntax = regexp.MustCompile(`^[a-z0-9]+(-[a-z0-9]+)*@[1-9][0-9]*$`)

// grantShape is the grant read WITHOUT binding validation: it accepts
// the "@task_id" and "@workspace" placeholders a template (or an
// envelope grant before L7 binds it) legitimately carries. It is used
// only to compare shapes; it never authorizes anything.
type grantShape struct {
	Version       int          `json:"version"`
	TaskID        string       `json:"task_id"`
	Entries       []GrantEntry `json:"entries"`
	TotalMaxCalls int          `json:"total_max_calls"`
}

func parseGrantShape(label string, raw []byte) (*grantShape, error) {
	if err := strictjson.Check(raw); err != nil {
		return nil, fmt.Errorf("%w: %s: %v", ErrGrantInvalid, label, err)
	}
	dec := json.NewDecoder(strings.NewReader(string(raw)))
	dec.DisallowUnknownFields()
	var g grantShape
	if err := dec.Decode(&g); err != nil {
		return nil, fmt.Errorf("%w: %s: %v", ErrGrantInvalid, label, err)
	}
	if dec.More() {
		return nil, fmt.Errorf("%w: %s: trailing content", ErrGrantInvalid, label)
	}
	if g.Version < 1 || g.TaskID == "" || len(g.Entries) == 0 || g.TotalMaxCalls <= 0 {
		return nil, fmt.Errorf("%w: %s: version, task, entries, and positive total cap required", ErrGrantInvalid, label)
	}
	seen := map[string]bool{}
	for _, e := range g.Entries {
		if e.Tool == "" || seen[e.Tool] {
			return nil, fmt.Errorf("%w: %s: empty or duplicate grant entry %q", ErrGrantInvalid, label, e.Tool)
		}
		seen[e.Tool] = true
		if e.MaxCalls <= 0 {
			return nil, fmt.Errorf("%w: %s: grant %q needs a positive call cap", ErrGrantInvalid, label, e.Tool)
		}
		// A shape is compared BEFORE assembly binds the workspace, so
		// the only values a template or an instance may carry are the
		// placeholder or nothing. A literal path here is never a
		// legitimate instance: it is a caller trying to mint a scope
		// (security review CRITICAL-1).
		if e.Workspace != "" && e.Workspace != "@workspace" {
			return nil, fmt.Errorf("%w: %s: grant %q workspace must be the @workspace placeholder or absent — a literal path is never a template or an instance", ErrGrantInvalid, label, e.Tool)
		}
		if err := checkTemplateScope(e.TemplateScope); err != nil {
			return nil, fmt.Errorf("%w: %s: grant %q: %v", ErrGrantInvalid, label, e.Tool, err)
		}
	}
	return &g, nil
}

// checkTemplateScope: exact references only, no duplicates. Shared by
// LoadGrant and the shape parser so a scope entry the runtime gate
// could never match (a prefix, a range, "latest") is refused at load.
func checkTemplateScope(scope []string) error {
	seen := map[string]bool{}
	for _, s := range scope {
		if !skillRefSyntax.MatchString(s) {
			return fmt.Errorf("template_scope entry %q must be an exact name@version", s)
		}
		if seen[s] {
			return fmt.Errorf("duplicate template_scope entry %q", s)
		}
		seen[s] = true
	}
	return nil
}

// ErrNotInstantiation is the typed refusal for an effective grant that
// is not a legitimate instantiation of its template.
var ErrNotInstantiation = fmt.Errorf("%w: effective grant is not an instantiation of its template", ErrGrantInvalid)

// Instantiates reports whether effective ⊑ template under D-SA-4:
//
//	tool set                equality (no additions, no removals)
//	per-tool max_calls      1 ≤ effective ≤ template
//	total_max_calls         1 ≤ effective ≤ template
//	workspace               equality (@workspace or absent, exactly as
//	                        the template; a literal path is refused at
//	                        parse — binding is assembly's act, after
//	                        this relation has been checked)
//	mutating                equality
//	themis_scope            set equality
//	template_scope          set equality
//	task_id                 effective = taskID where the template has
//	                        @task_id
//
// Each clause carries its own refusal so a suppressed clause lets
// exactly one mutation through (the D-SA-4 twin discipline).
func Instantiates(effectiveRaw, templateRaw []byte, taskID string) error {
	eff, err := parseGrantShape("effective grant", effectiveRaw)
	if err != nil {
		return err
	}
	tpl, err := parseGrantShape("grant template", templateRaw)
	if err != nil {
		return err
	}
	if tpl.TaskID != "@task_id" {
		return fmt.Errorf("%w: the template's task_id must be the @task_id placeholder", ErrNotInstantiation)
	}
	if eff.TaskID != taskID {
		return fmt.Errorf("%w: task_id %q is not the task %q being instantiated", ErrNotInstantiation, eff.TaskID, taskID)
	}
	if eff.Version != tpl.Version {
		return fmt.Errorf("%w: version %d differs from the template's %d", ErrNotInstantiation, eff.Version, tpl.Version)
	}
	if eff.TotalMaxCalls > tpl.TotalMaxCalls {
		return fmt.Errorf("%w: total_max_calls %d exceeds the template's %d — quotas only narrow", ErrNotInstantiation, eff.TotalMaxCalls, tpl.TotalMaxCalls)
	}
	tplEntries := map[string]GrantEntry{}
	for _, e := range tpl.Entries {
		tplEntries[e.Tool] = e
	}
	effEntries := map[string]GrantEntry{}
	for _, e := range eff.Entries {
		effEntries[e.Tool] = e
	}
	for tool := range tplEntries {
		if _, ok := effEntries[tool]; !ok {
			return fmt.Errorf("%w: tool %q is in the template but not the effective grant — the tool set is Skill-fixed", ErrNotInstantiation, tool)
		}
	}
	for tool, e := range effEntries {
		t, ok := tplEntries[tool]
		if !ok {
			return fmt.Errorf("%w: tool %q is not in the template — the tool set is Skill-fixed", ErrNotInstantiation, tool)
		}
		if e.MaxCalls > t.MaxCalls {
			return fmt.Errorf("%w: tool %q max_calls %d exceeds the template's %d — quotas only narrow", ErrNotInstantiation, tool, e.MaxCalls, t.MaxCalls)
		}
		if e.Workspace != t.Workspace {
			return fmt.Errorf("%w: tool %q workspace binding differs from the template — a scope is Skill-fixed", ErrNotInstantiation, tool)
		}
		if e.Mutating != t.Mutating {
			return fmt.Errorf("%w: tool %q mutating flag differs from the template — the review surface is Skill-fixed", ErrNotInstantiation, tool)
		}
		if !sameSet(e.ThemisScope, t.ThemisScope) {
			return fmt.Errorf("%w: tool %q themis_scope differs from the template — authority scope is Skill-fixed", ErrNotInstantiation, tool)
		}
		if !sameSet(e.TemplateScope, t.TemplateScope) {
			return fmt.Errorf("%w: tool %q template_scope differs from the template — it is set-equal, never narrowed or widened by a caller", ErrNotInstantiation, tool)
		}
	}
	return nil
}

// sameSet: order and repetition do not change what a scope permits.
func sameSet(a, b []string) bool {
	norm := func(s []string) []string {
		u := map[string]bool{}
		for _, x := range s {
			u[x] = true
		}
		out := make([]string, 0, len(u))
		for x := range u {
			out = append(out, x)
		}
		sort.Strings(out)
		return out
	}
	na, nb := norm(a), norm(b)
	if len(na) != len(nb) {
		return false
	}
	for i := range na {
		if na[i] != nb[i] {
			return false
		}
	}
	return true
}
