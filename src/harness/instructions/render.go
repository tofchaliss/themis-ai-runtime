package instructions

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"

	"github.com/tofchaliss/themis/runtime/model"
)

// ErrRenderViolation marks a rendered EIS that failed final
// validation. The final pass detects and refuses delivery — it never
// rewrites, reorders, or recomposes the instruction set (design §6,
// Q-L1-3).
var ErrRenderViolation = errors.New("rendered EIS failed final validation")

// RoleID is the protected neutral role preamble (DEC-06); it renders
// first, outside every section.
const RoleID = "harness.system.role"

// sectionOrder fixes the render headings (design D-L1-6). The
// renderer owns structural furniture ONLY — headings, delimiters,
// ordering. Any delivered text carrying a rule lives in an
// instruction file; adding directive language here fails review
// (design §6, Q-L1-8 permanent review rule).
var sectionOrder = []struct {
	heading    string
	categories []Category
}{
	{"## Safety", []Category{CategorySafety}},
	{"## Themis principles", []Category{CategoryThemis}},
	{"## Engineering", []Category{CategoryEngineering}},
	{"## Procedure", []Category{CategoryHarness, CategorySkill}},
	{"## Task", []Category{CategoryTask}},
}

// Render produces the deterministic instruction text: role preamble
// first, then verbatim bodies grouped under fixed headings in
// resolved order. It requires the same validated policy that judged
// the resolution (hash-matched) and runs the final validation pass on
// the exact bytes that will be delivered: size cap, secret scan over
// the full text, and directive patterns over the adjacency of
// untrusted bodies. Returns the text and its SHA-256 — what was
// scanned = what the model receives.
func (e *EffectiveSet) Render(p *Policy) (text string, hash string, err error) {
	if p == nil || p.Hash == "" {
		return "", "", fmt.Errorf("%w: %w: render requires the validated pattern policy", ErrRenderViolation, ErrPolicyInvalid)
	}
	if p.Hash != e.PolicyHash {
		return "", "", fmt.Errorf("%w: %w: render policy %s differs from resolving policy %s",
			ErrRenderViolation, ErrPolicyInvalid, p.Hash, e.PolicyHash)
	}
	// Structural pre-pass. An empty set is not a deliverable
	// environment; every body must belong to a known section (silent
	// omission would break "what was scanned = what the model
	// received" in the omission direction); and H2 lines are reserved
	// for renderer furniture — a body opening a "## " line would
	// counterfeit section structure, so it refuses delivery at any
	// scope.
	if len(e.Instructions) == 0 {
		return "", "", fmt.Errorf("%w: empty instruction set is not a deliverable environment", ErrRenderViolation)
	}
	for _, inst := range e.Instructions {
		if inst.ID != RoleID && !categoryKnown(inst.Category) {
			return "", "", fmt.Errorf("%w: instruction %q has no render section for category %q",
				ErrRenderViolation, inst.ID, inst.Category)
		}
		for _, line := range strings.Split(inst.Body, "\n") {
			if strings.HasPrefix(line, "## ") {
				return "", "", fmt.Errorf("%w: instruction %q body contains a reserved heading line %q — H2 furniture is renderer-owned",
					ErrRenderViolation, inst.ID, line)
			}
		}
	}
	var b strings.Builder
	writeBody := func(body string) {
		b.WriteString(body)
		if !strings.HasSuffix(body, "\n") {
			b.WriteString("\n")
		}
	}
	for _, inst := range e.Instructions {
		if inst.ID == RoleID {
			writeBody(inst.Body)
			break
		}
	}
	for _, sec := range sectionOrder {
		wrote := false
		for _, inst := range e.Instructions {
			if inst.ID == RoleID || !categoryIn(inst.Category, sec.categories) {
				continue
			}
			if !wrote {
				b.WriteString("\n" + sec.heading + "\n")
				wrote = true
			} else {
				b.WriteString("\n")
			}
			writeBody(inst.Body)
		}
	}
	text = b.String()

	// Final validation pass — on the delivered bytes, detect and
	// refuse only; no rewriting.
	if len(text) > MaxEISBytes {
		return "", "", fmt.Errorf("%w: %w: rendered EIS is %d bytes (cap %d)", ErrRenderViolation, ErrTooLarge, len(text), MaxEISBytes)
	}
	for _, pat := range secretPatterns {
		if pat.MatchString(text) {
			return "", "", fmt.Errorf("%w: %w: rendered text matches %q", ErrRenderViolation, ErrSecretDetected, pat.String())
		}
	}
	// Directive patterns over the rendered adjacency of untrusted
	// bodies only: trusted bodies are reviewed content and would
	// false-positive (e.g. "never expose credentials"); the
	// composition risk the grill identified lives where untrusted
	// fragments become adjacent.
	var untrusted []string
	for _, inst := range e.Instructions {
		if !trustedKinds[inst.Scope] {
			untrusted = append(untrusted, inst.Body)
		}
	}
	if len(untrusted) > 0 {
		if id, _ := p.match(strings.Join(untrusted, "\n")); id != "" {
			return "", "", fmt.Errorf("%w: untrusted bodies compose into directive pattern %q at render", ErrRenderViolation, id)
		}
	}
	sum := sha256.Sum256([]byte(text))
	return text, hex.EncodeToString(sum[:]), nil
}

// SystemMessage renders the EIS as the system message for Context
// Delivery (L2) to compose into the model payload — a convenience for
// that composition, not a delivery path: L1 never calls the model.
func (e *EffectiveSet) SystemMessage(p *Policy) (model.Message, string, error) {
	text, hash, err := e.Render(p)
	if err != nil {
		return model.Message{}, "", err
	}
	return model.Message{Role: model.RoleSystem, Content: text}, hash, nil
}

func categoryIn(c Category, set []Category) bool {
	for _, s := range set {
		if c == s {
			return true
		}
	}
	return false
}

func categoryKnown(c Category) bool {
	for _, sec := range sectionOrder {
		if categoryIn(c, sec.categories) {
			return true
		}
	}
	return false
}
