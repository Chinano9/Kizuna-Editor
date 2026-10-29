package database

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolveDBPath(t *testing.T) {
	t.Run("resolves standard db path", func(t *testing.T) {
		path, err := ResolveDBPath("KizunaTest", "test.db")
		if err != nil {
			t.Fatalf("expected no error, got: %v", err)
		}
		if filepath.Base(path) != "test.db" {
			t.Errorf("expected file name 'test.db', got '%s'", filepath.Base(path))
		}
		dir := filepath.Base(filepath.Dir(path))
		if dir != "KizunaTest" {
			t.Errorf("expected parent dir 'KizunaTest', got '%s'", dir)
		}
	})

	t.Run("uses defaults when empty args provided", func(t *testing.T) {
		path, err := ResolveDBPath("", "")
		if err != nil {
			t.Fatalf("expected no error, got: %v", err)
		}
		if filepath.Base(path) != DefaultDBName {
			t.Errorf("expected default DB name '%s', got '%s'", DefaultDBName, filepath.Base(path))
		}
	})
}

func TestEnsureDirectory(t *testing.T) {
	tempDir := t.TempDir()
	nestedPath := filepath.Join(tempDir, "deep", "nested", "kizuna.db")

	err := EnsureDirectory(nestedPath)
	if err != nil {
		t.Fatalf("failed to create directory: %v", err)
	}

	info, err := os.Stat(filepath.Dir(nestedPath))
	if err != nil {
		t.Fatalf("directory was not created: %v", err)
	}
	if !info.IsDir() {
		t.Errorf("expected path to be a directory")
	}
}
