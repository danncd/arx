package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

/* Clears a variable for one test. */

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

/* Keeps dotenv values literal. */

func TestLoadDotEnvValuesAreLiteral(t *testing.T) {
	path := writeEnv(t, "ESCAPED=\"p\\\"q\"\nHASH=sk-abc#not-a-comment\nexport\tTABBED=works\n")
	for _, k := range []string{"ESCAPED", "HASH", "TABBED"} {
		unset(t, k)
	}
	if err := LoadDotEnv(path); err != nil {
		t.Fatal(err)
	}
	// One balanced pair stripped; the backslash escape stays literal.
	if got := os.Getenv("ESCAPED"); got != `p\"q` {
		t.Errorf("escapes must stay literal: %q", got)
	}
	if got := os.Getenv("HASH"); got != "sk-abc#not-a-comment" {
		t.Errorf("inline # must not be treated as a comment: %q", got)
	}
	if got := os.Getenv("TABBED"); got != "works" {
		t.Errorf("export<TAB>KEY line not handled: %q", got)
	}
	if _, set := os.LookupEnv("export\tTABBED"); set {
		t.Error("a garbage variable named with the export prefix was created")
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

func TestLoadDotEnvReturnsSetError(t *testing.T) {
	path := writeEnv(t, "BAD\x00KEY=value\n")
	if err := LoadDotEnv(path); err == nil || !strings.Contains(err.Error(), "BAD") {
		t.Fatalf("invalid environment key error = %v", err)
	}
}
