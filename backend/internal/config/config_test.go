package config

import (
	"os"
	"path/filepath"
	"testing"
)

// unset truly removes the variable for the test's duration. t.Setenv
// alone cannot do this — it SETS the value (even "") — but calling it
// first registers automatic restoration, after which Unsetenv is safe.
func unset(t *testing.T, key string) {
	t.Helper()
	t.Setenv(key, "sentinel-to-register-cleanup")
	os.Unsetenv(key)
}

func writeEnv(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), ".env")
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestLoadDotEnv(t *testing.T) {
	path := writeEnv(t, "# comment\n\nFROM_FILE=sk-from-file\nQUOTED=\"hello\"\nALREADY_SET=from-file\nSET_EMPTY=from-file\nexport EXPORTED=works\nTRAILING=abc'\n")

	unset(t, "FROM_FILE")
	unset(t, "QUOTED")
	unset(t, "EXPORTED")
	unset(t, "TRAILING")
	t.Setenv("ALREADY_SET", "from-env") // non-empty override must survive
	t.Setenv("SET_EMPTY", "")           // deliberately blanked must STAY blank

	if err := LoadDotEnv(path); err != nil {
		t.Fatalf("LoadDotEnv: %v", err)
	}
	if got := os.Getenv("FROM_FILE"); got != "sk-from-file" {
		t.Errorf("unset var not loaded from file: %q", got)
	}
	if got := os.Getenv("QUOTED"); got != "hello" {
		t.Errorf("balanced quotes not stripped: %q", got)
	}
	if got := os.Getenv("ALREADY_SET"); got != "from-env" {
		t.Errorf("env override lost to the file: %q", got)
	}
	if got, set := os.LookupEnv("SET_EMPTY"); !set || got != "" {
		t.Errorf("explicitly-empty var was overwritten by the file: %q (set=%v)", got, set)
	}
	if got := os.Getenv("EXPORTED"); got != "works" {
		t.Errorf("export-prefixed line not loaded: %q", got)
	}
	if got := os.Getenv("TRAILING"); got != "abc'" {
		t.Errorf("unbalanced trailing quote must be preserved: %q", got)
	}
}

func TestLoadDotEnvDoesNotCorruptQuoteBearingValues(t *testing.T) {
	path := writeEnv(t, "A=He said \"hi\"\nB=p'ass\"word'\nC=\"'inner'\"\n")
	for _, k := range []string{"A", "B", "C"} {
		unset(t, k)
	}
	if err := LoadDotEnv(path); err != nil {
		t.Fatal(err)
	}
	if got := os.Getenv("A"); got != `He said "hi"` {
		t.Errorf("A corrupted: %q", got)
	}
	if got := os.Getenv("B"); got != `p'ass"word'` {
		t.Errorf("B corrupted: %q", got)
	}
	// One balanced pair comes off; the inner quotes stay.
	if got := os.Getenv("C"); got != "'inner'" {
		t.Errorf("C: want one layer stripped, got %q", got)
	}
}

func TestLoadDotEnvMissingFileIsFine(t *testing.T) {
	if err := LoadDotEnv(filepath.Join(t.TempDir(), "nope.env")); err != nil {
		t.Fatalf("missing file must be silent, got: %v", err)
	}
}

func TestLoadDotEnvStripsBOM(t *testing.T) {
	path := writeEnv(t, "\ufeffBOM_KEY=value\n")
	unset(t, "BOM_KEY")
	if err := LoadDotEnv(path); err != nil {
		t.Fatal(err)
	}
	if got := os.Getenv("BOM_KEY"); got != "value" {
		t.Errorf("BOM not stripped, key mangled: %q", got)
	}
}
