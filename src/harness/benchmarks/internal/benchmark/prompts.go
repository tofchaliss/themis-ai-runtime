package benchmark

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"
)

// Prompts loads benchmark prompts with two optional layers on top of
// plain Markdown files:
//
//   - Partials: prompts/partials/*.md are registered as named templates
//     (by filename without extension). A prompt containing template
//     directives is rendered with text/template and may include them
//     via {{template "name"}}. Prompts without directives pass through
//     byte-for-byte.
//
//   - Variants: with a variant set, prompts/variants/<variant>/<file>
//     overrides prompts/<file>, and partials in
//     prompts/variants/<variant>/partials/ override base partials.
//     This enables prompt A/B testing: run the same benchmarks with a
//     variant and compare the scores.
type Prompts struct {
	root     string
	variant  string
	partials *template.Template
}

// NewPrompts creates a loader for the suite root. variant may be empty.
func NewPrompts(root, variant string) (*Prompts, error) {

	if variant != "" {
		variantDir := filepath.Join(root, "prompts", "variants", variant)

		if info, err := os.Stat(variantDir); err != nil || !info.IsDir() {
			return nil, fmt.Errorf(
				"variant %q not found (expected directory %s)",
				variant,
				variantDir,
			)
		}
	}

	partials := template.New("")

	baseDir := filepath.Join(root, "prompts", "partials")
	if err := loadPartials(partials, baseDir); err != nil {
		return nil, err
	}

	if variant != "" {
		variantDir := filepath.Join(
			root, "prompts", "variants", variant, "partials",
		)
		if err := loadPartials(partials, variantDir); err != nil {
			return nil, err
		}
	}

	return &Prompts{
		root:     root,
		variant:  variant,
		partials: partials,
	}, nil
}

// Variant returns the active variant name ("" for the base prompts).
func (p *Prompts) Variant() string {
	return p.variant
}

// Load resolves, and renders if templated, the named prompt file. The
// returned bytes are exactly what should be sent to the model.
func (p *Prompts) Load(name string) ([]byte, error) {

	path := filepath.Join(p.root, "prompts", name)

	if p.variant != "" {
		variantPath := filepath.Join(
			p.root, "prompts", "variants", p.variant, name,
		)
		if _, err := os.Stat(variantPath); err == nil {
			path = variantPath
		}
	}

	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	if !bytes.Contains(raw, []byte("{{")) {
		return raw, nil
	}

	tmpl, err := p.partials.Clone()
	if err != nil {
		return nil, err
	}

	tmpl, err = tmpl.New(name).Parse(string(raw))
	if err != nil {
		return nil, fmt.Errorf("parse prompt template %s: %w", path, err)
	}

	var rendered bytes.Buffer

	if err := tmpl.ExecuteTemplate(&rendered, name, nil); err != nil {
		return nil, fmt.Errorf("render prompt template %s: %w", path, err)
	}

	return rendered.Bytes(), nil
}

// loadPartials registers every *.md file in dir as a template named by
// its filename without extension. A missing dir is fine.
func loadPartials(t *template.Template, dir string) error {

	files, err := filepath.Glob(filepath.Join(dir, "*.md"))
	if err != nil {
		return err
	}

	for _, file := range files {

		content, err := os.ReadFile(file)
		if err != nil {
			return err
		}

		name := strings.TrimSuffix(
			filepath.Base(file),
			filepath.Ext(file),
		)

		if _, err := t.New(name).Parse(string(content)); err != nil {
			return fmt.Errorf("parse partial %s: %w", file, err)
		}
	}

	return nil
}
