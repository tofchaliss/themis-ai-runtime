package instructions

import (
	"fmt"
	"regexp"
	"strings"
)

// The namespace registry is trusted configuration (design §6, Q-L1-7):
// each namespace has exactly one owning source kind, and only that
// owner may declare identities within it. Owner resolution is
// claimed id -> parse namespace -> this registry -> expected owner ->
// compare against the declaring source. An id outside every registered
// namespace is a configuration failure, never inferred (recognition,
// never discovery). skill.* binds to immutable registered skill
// identities when Layer 9 lands; until then no recognized source may
// declare in it.
var namespaceRegistry = []struct {
	prefix string
	owner  Scope
}{
	{"harness.safety.", ScopeHarnessSafety},
	{"harness.system.", ScopeHarnessSystem},
	{"themis.", ScopeThemisDomain},
	{"repo.", ScopeRepository},
	{"skill.", ScopeSkill},
	{"task.", ScopeTask},
}

// idSyntax: lowercase dot-separated segments of letters, digits, and
// interior hyphens; at least two segments (namespace + name).
var idSyntax = regexp.MustCompile(`^[a-z0-9]+(-[a-z0-9]+)*(\.[a-z0-9]+(-[a-z0-9]+)*)+$`)

// NamespaceOwner returns the source kind that owns the id's namespace.
// Identity judgments never inspect bodies: ownership is decided from
// the id string and this registry alone.
func NamespaceOwner(id string) (Scope, error) {
	if !idSyntax.MatchString(id) {
		return 0, fmt.Errorf("%w: %q", ErrBadID, id)
	}
	for _, ns := range namespaceRegistry {
		if strings.HasPrefix(id, ns.prefix) && len(id) > len(ns.prefix) {
			return ns.owner, nil
		}
	}
	return 0, fmt.Errorf("%w: %q", ErrUnknownNamespace, id)
}
