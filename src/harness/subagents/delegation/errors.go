// Package delegation is Layer 8's governed artifact family: the
// delegation-template registry and the template manifest loader
// (openspec/changes/archive/2026-09-23-layer-08-subagents, D-L8-5/6). A delegation template is
// a distinct Governance-registered artifact — not a Skill, not a Skill
// member, never executable. It defines only the bounded composition
// available to one isolated, tool-less L8 reasoning execution: a
// pinned L2 context contract, an EIS carry-over filter, at most one
// pinned skill-scope instruction file, a bounded brief slot, and an
// output bound.
//
// This package only READS. There is deliberately no write API
// anywhere in it (the L9 catalog / L10 registry wall, reapplied), so
// "machinery must not self-register" holds structurally; the AST wall
// in the tests keeps it so.
package delegation

import "errors"

var (
	// ErrRegistry: the registry file is not a valid governed registry.
	ErrRegistry = errors.New("invalid delegation-template registry")
	// ErrTemplate: a template manifest or one of its pins is invalid.
	ErrTemplate = errors.New("invalid delegation template")
	// ErrResolve: an exact reference did not resolve to an executable
	// registration in the loaded registry state.
	ErrResolve = errors.New("delegation template resolution refused")
	// ErrEvent: an l8-delegation body that violates its own closure.
	ErrEvent = errors.New("invalid l8-delegation record")
	// ErrWithdrawn: the reference names a withdrawn registration
	// (wraps ErrResolve). ErrHashMismatch: the manifest bytes are not
	// the registered bytes (wraps ErrResolve). Typed so consumers
	// classify without reading messages.
	ErrWithdrawn    = errors.New("template withdrawn")
	ErrHashMismatch = errors.New("template-hash-mismatch")
)
