package instructions

import (
	"errors"
	"fmt"
)

// Size caps (design §6, Q-L1-6): deterministic validation constants,
// fail closed at load and render; revisited when Layer 3 owns token
// budgets.
const (
	MaxInstructionBytes = 8 * 1024
	MaxEISBytes         = 64 * 1024
)

// Sentinel errors. Every load/validation failure wraps exactly one of
// these; a caller distinguishing abort classes matches with errors.Is.
var (
	ErrUnrecognizedSource   = errors.New("unrecognized instruction source")
	ErrSourceUnavailable    = errors.New("registered instruction source unavailable")
	ErrProtectedDeclaration = errors.New("untrusted source cannot declare a protected instruction")
	ErrMalformed            = errors.New("malformed instruction file")
	ErrBadID                = errors.New("invalid instruction id")
	ErrUnknownNamespace     = errors.New("unknown instruction namespace")
	ErrNamespaceViolation   = errors.New("namespace not owned by declaring source")
	ErrScopeMismatch        = errors.New("instruction scope illegal for source")
	ErrDuplicateID          = errors.New("duplicate instruction id")
	ErrEmptyBody            = errors.New("empty instruction body")
	ErrSecretDetected       = errors.New("secret material in instruction body")
	ErrTooLarge             = errors.New("instruction exceeds size cap")
)

// Source is a registered root or payload explicitly recognized as
// feeding instructions (design D-L1-3): content the agent merely
// encounters is data, never a source. Exactly one of Root or Inline
// is set.
type Source struct {
	Kind   Scope
	Root   string        // directory of *.md files, or
	Inline []Instruction // task payload (ScopeTask)
}

// recognizedKinds are the source kinds v1 loads. ScopeRepository and
// ScopeDirectory await Layer 5 pinned-ref provenance; ScopeSkill
// awaits Layer 9 registered skill identity (design §6, Q-L1-1;
// tasks.md §7). Passing an unrecognized kind is a hard error, never a
// silent skip.
var recognizedKinds = map[Scope]bool{
	ScopeHarnessSafety: true,
	ScopeHarnessSystem: true,
	ScopeThemisDomain:  true,
	ScopeTask:          true,
}

// trustedKinds mark sources whose failure is a trusted-configuration
// failure (abort: no EIS). Violations by untrusted sources are
// rejected and recorded while resolution continues (design §6,
// Q-L1-6 dividing line).
var trustedKinds = map[Scope]bool{
	ScopeHarnessSafety: true,
	ScopeHarnessSystem: true,
	ScopeThemisDomain:  true,
}

func checkSource(s Source) error {
	if !recognizedKinds[s.Kind] {
		return fmt.Errorf("%w: kind %s", ErrUnrecognizedSource, s.Kind)
	}
	if (s.Root == "") == (s.Inline == nil) {
		return fmt.Errorf("%w: source %s must set exactly one of Root or Inline", ErrUnrecognizedSource, s.Kind)
	}
	if s.Inline != nil && s.Kind != ScopeTask {
		return fmt.Errorf("%w: inline instructions are the task payload only, got %s", ErrUnrecognizedSource, s.Kind)
	}
	return nil
}
