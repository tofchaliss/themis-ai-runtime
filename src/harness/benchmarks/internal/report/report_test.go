package report

import (
	"os"
	"path/filepath"
	"testing"
)

func TestWrite(t *testing.T) {

	t.Run("simple model name", func(t *testing.T) {
		root := t.TempDir()

		report := Generate("mymodel", "2026-08-19", nil)

		if err := Write(root, "2026-08-19", report); err != nil {
			t.Fatal(err)
		}

		path := filepath.Join(root, "reports", "2026-08-19", "mymodel.md")
		if _, err := os.Stat(path); err != nil {
			t.Errorf("report not written: %v", err)
		}
	})

	t.Run("model name with slashes", func(t *testing.T) {
		root := t.TempDir()

		model := "WhiteRabbitNeo/WhiteRabbitNeo-2.5-Qwen-2.5-Coder-7B"
		report := Generate(model, "2026-08-19", nil)

		if err := Write(root, "2026-08-19", report); err != nil {
			t.Fatal(err)
		}

		path := filepath.Join(root, "reports", "2026-08-19", model+".md")
		if _, err := os.Stat(path); err != nil {
			t.Errorf("report not written: %v", err)
		}
	})
}
