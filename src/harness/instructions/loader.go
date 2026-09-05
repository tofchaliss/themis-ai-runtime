package instructions

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// LoadResult carries the validated instructions, the conflicts
// recorded while loading, and the security-configuration identities
// that judged the load. Load is atomic over trusted configuration:
// any trusted-source failure yields (nil, error) — there is no
// partial, degraded, or fallback instruction environment.
type LoadResult struct {
	Instructions      []Instruction
	Conflicts         []Conflict
	PolicyHash        string      // which pattern policy judged this load
	ExemptionsApplied []Exemption // suppressed detection decisions, on the record
}

// Load reads and validates every registered source under the given
// security configuration. Determinism: root files are discovered in
// sorted path order, bodies are byte-exact, and identical inputs
// produce identical results. Untrusted-content violations (a foreign
// namespace claim, a directive-pattern match) are dropped and
// recorded as Conflicts; everything else fails closed.
func Load(cfg Config, sources ...Source) (*LoadResult, error) {
	if cfg.Policy == nil || cfg.Policy.Hash == "" {
		return nil, fmt.Errorf("%w: no validated pattern policy configured — resolution without the configured detector does not happen", ErrPolicyInvalid)
	}
	if err := cfg.validateExemptions(); err != nil {
		return nil, err
	}
	res := &LoadResult{PolicyHash: cfg.Policy.Hash}
	seen := map[string]string{} // id -> SourceRef
	for _, src := range sources {
		if err := checkSource(src); err != nil {
			return nil, err
		}
		insts, err := loadSource(src)
		if err != nil {
			return nil, err
		}
		// A registered source is expected to contribute; a source with
		// nothing to say is omitted by the caller, so an empty yield is
		// a configuration discrepancy, not a valid quiet state.
		if len(insts) == 0 {
			return nil, fmt.Errorf("%w: source %s yielded no instructions", ErrSourceUnavailable, src.Kind)
		}
		for _, inst := range insts {
			conflict, err := validate(inst, src)
			if err != nil {
				return nil, err
			}
			if conflict != nil {
				// Ownership rejection: the id belongs to another
				// source, so it does not enter duplicate accounting.
				res.Conflicts = append(res.Conflicts, *conflict)
				continue
			}
			// Duplicate accounting happens before pattern checks: a
			// pattern-rejected instruction still legitimately claimed
			// its id, so a decoy body cannot convert a structural
			// duplicate abort into run-continues.
			if prev, dup := seen[inst.ID]; dup {
				return nil, fmt.Errorf("%w: %q declared by %s and %s",
					ErrDuplicateID, inst.ID, prev, inst.SourceRef)
			}
			seen[inst.ID] = inst.SourceRef
			if !trustedKinds[src.Kind] {
				// Directive patterns run over untrusted bodies only:
				// trusted roots are reviewed content, and the corpus
				// protects legitimate phrasing at the scopes where
				// untrusted callers write.
				patternConflict, applied := cfg.checkPatterns(inst)
				res.ExemptionsApplied = append(res.ExemptionsApplied, applied...)
				if patternConflict != nil {
					res.Conflicts = append(res.Conflicts, *patternConflict)
					continue
				}
			}
			res.Instructions = append(res.Instructions, inst)
		}
	}
	return res, nil
}

func loadSource(src Source) ([]Instruction, error) {
	if src.Inline != nil {
		insts := make([]Instruction, len(src.Inline))
		for i, inst := range src.Inline {
			inst.SourceRef = fmt.Sprintf("inline:%s[%d]", src.Kind, i)
			inst.BodyHash = bodyHash(inst.Body)
			insts[i] = inst
		}
		return insts, nil
	}
	var paths []string
	err := filepath.WalkDir(src.Root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() && strings.HasSuffix(path, ".md") {
			paths = append(paths, path)
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("%w: source %s root %q: %v", ErrSourceUnavailable, src.Kind, src.Root, err)
	}
	sort.Strings(paths)
	var insts []Instruction
	for _, path := range paths {
		inst, err := parseFile(path, src.Kind)
		if err != nil {
			return nil, err
		}
		insts = append(insts, inst)
	}
	return insts, nil
}

// parseFile reads one instruction file: a strict frontmatter block
// (deliberately a restricted deterministic subset, not full YAML)
// followed by the byte-exact body. Fail closed: any deviation is a
// hard error naming the file.
func parseFile(path string, kind Scope) (Instruction, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return Instruction{}, fmt.Errorf("%w: %s: %v", ErrSourceUnavailable, path, err)
	}
	content := string(raw)
	const delim = "---\n"
	if !strings.HasPrefix(content, delim) {
		return Instruction{}, fmt.Errorf("%s: %w: missing opening frontmatter delimiter", path, ErrMalformed)
	}
	rest := content[len(delim):]
	end := strings.Index(rest, "\n"+delim)
	var block, body string
	if strings.HasPrefix(rest, delim) { // empty frontmatter block
		block, body = "", rest[len(delim):]
	} else if end >= 0 {
		block, body = rest[:end+1], rest[end+1+len(delim):]
	} else {
		return Instruction{}, fmt.Errorf("%s: %w: unterminated frontmatter", path, ErrMalformed)
	}

	inst := Instruction{Body: body, SourceRef: path, BodyHash: bodyHash(body)}
	fields := map[string]string{}
	for _, line := range strings.Split(strings.TrimSuffix(block, "\n"), "\n") {
		if line == "" && block == "" {
			continue
		}
		key, value, ok := strings.Cut(line, ": ")
		if !ok || key == "" || value == "" || strings.TrimSpace(key) != key {
			return Instruction{}, fmt.Errorf("%s: %w: bad frontmatter line %q", path, ErrMalformed, line)
		}
		if _, dup := fields[key]; dup {
			return Instruction{}, fmt.Errorf("%s: %w: duplicate frontmatter key %q", path, ErrMalformed, key)
		}
		fields[key] = value
	}
	for key, value := range fields {
		switch key {
		case "id":
			inst.ID = value
		case "scope":
			sc, err := ParseScope(value)
			if err != nil {
				return Instruction{}, fmt.Errorf("%s: %w", path, err)
			}
			inst.Scope = sc
		case "category":
			inst.Category = Category(value)
		case "protected":
			switch value {
			case "true":
				inst.Protected = true
			case "false":
			default:
				return Instruction{}, fmt.Errorf("%s: %w: protected must be true or false, got %q", path, ErrMalformed, value)
			}
		default:
			return Instruction{}, fmt.Errorf("%s: %w: unknown frontmatter key %q", path, ErrMalformed, key)
		}
	}
	for _, required := range []string{"id", "scope", "category"} {
		if _, ok := fields[required]; !ok {
			return Instruction{}, fmt.Errorf("%s: %w: missing required frontmatter key %q", path, ErrMalformed, required)
		}
	}
	return inst, nil
}
