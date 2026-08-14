package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadDotEnv(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".env")
	content := "# comment\n\nDEEPSEEK_API_KEY=sk-from-file\nQUOTED=\"hello\"\nALREADY_SET=from-file\n"
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}

	t.Setenv("DEEPSEEK_API_KEY", "") // simulate unset
	t.Setenv("QUOTED", "")
	t.Setenv("ALREADY_SET", "from-env") // the override that must survive

	if err := LoadDotEnv(path); err != nil {
		t.Fatalf("LoadDotEnv: %v", err)
	}
	if got := os.Getenv("DEEPSEEK_API_KEY"); got != "sk-from-file" {
		t.Errorf("file value not loaded: %q", got)
	}
	if got := os.Getenv("QUOTED"); got != "hello" {
		t.Errorf("quotes not trimmed: %q", got)
	}
	if got := os.Getenv("ALREADY_SET"); got != "from-env" {
		t.Errorf("env override lost to the file: %q", got)
	}
}

func TestLoadDotEnvMissingFileIsFine(t *testing.T) {
	if err := LoadDotEnv(filepath.Join(t.TempDir(), "nope.env")); err != nil {
		t.Fatalf("missing file must be silent, got: %v", err)
	}
}
