package benchmark

import (
	"os"
	"path/filepath"
	"testing"
)

func writePrompt(t *testing.T, root, rel, content string) {
	t.Helper()

	path := filepath.Join(root, "prompts", rel)

	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
}

func TestPrompts(t *testing.T) {

	t.Run("plain prompt passes through byte-for-byte", func(t *testing.T) {
		root := t.TempDir()
		writePrompt(t, root, "B001.md", "Analyze { this } CVE.\n")

		p, err := NewPrompts(root, "")
		if err != nil {
			t.Fatal(err)
		}

		got, err := p.Load("B001.md")
		if err != nil {
			t.Fatal(err)
		}
		if string(got) != "Analyze { this } CVE.\n" {
			t.Errorf("got %q", got)
		}
	})

	t.Run("templated prompt renders partials", func(t *testing.T) {
		root := t.TempDir()
		writePrompt(t, root, "partials/preamble.md", "Evidence only.")
		writePrompt(t, root, "B001.md", "{{template \"preamble\"}}\n\nTask.")

		p, err := NewPrompts(root, "")
		if err != nil {
			t.Fatal(err)
		}

		got, err := p.Load("B001.md")
		if err != nil {
			t.Fatal(err)
		}
		if string(got) != "Evidence only.\n\nTask." {
			t.Errorf("got %q", got)
		}
	})

	t.Run("variant overrides prompt file", func(t *testing.T) {
		root := t.TempDir()
		writePrompt(t, root, "B001.md", "base prompt")
		writePrompt(t, root, "variants/v2/B001.md", "variant prompt")
		writePrompt(t, root, "B002.md", "untouched")

		p, err := NewPrompts(root, "v2")
		if err != nil {
			t.Fatal(err)
		}

		got, _ := p.Load("B001.md")
		if string(got) != "variant prompt" {
			t.Errorf("B001 = %q, want variant prompt", got)
		}

		// Files not in the variant fall back to base.
		got, _ = p.Load("B002.md")
		if string(got) != "untouched" {
			t.Errorf("B002 = %q, want base fallback", got)
		}
	})

	t.Run("variant overrides partials", func(t *testing.T) {
		root := t.TempDir()
		writePrompt(t, root, "partials/preamble.md", "base preamble")
		writePrompt(t, root, "variants/v2/partials/preamble.md", "v2 preamble")
		writePrompt(t, root, "B001.md", "{{template \"preamble\"}}")

		p, err := NewPrompts(root, "v2")
		if err != nil {
			t.Fatal(err)
		}

		got, err := p.Load("B001.md")
		if err != nil {
			t.Fatal(err)
		}
		if string(got) != "v2 preamble" {
			t.Errorf("got %q, want v2 preamble", got)
		}
	})

	t.Run("unknown variant is an error", func(t *testing.T) {
		root := t.TempDir()
		writePrompt(t, root, "B001.md", "x")

		if _, err := NewPrompts(root, "nope"); err == nil {
			t.Error("expected error for unknown variant")
		}
	})

	t.Run("malformed template is an error", func(t *testing.T) {
		root := t.TempDir()
		writePrompt(t, root, "B001.md", "{{template \"missing\"")

		p, err := NewPrompts(root, "")
		if err != nil {
			t.Fatal(err)
		}

		if _, err := p.Load("B001.md"); err == nil {
			t.Error("expected error for malformed template")
		}
	})

	t.Run("reference to unknown partial is an error", func(t *testing.T) {
		root := t.TempDir()
		writePrompt(t, root, "B001.md", "{{template \"missing\"}}")

		p, err := NewPrompts(root, "")
		if err != nil {
			t.Fatal(err)
		}

		if _, err := p.Load("B001.md"); err == nil {
			t.Error("expected error for unknown partial reference")
		}
	})
}
