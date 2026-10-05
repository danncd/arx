package skills

import (
	"os"
	"path/filepath"
	"testing"
)

const valid = "---\nname: test-skill\ndescription: Review a project.\n---\n\nRead and review the project.\n"

func TestCreateImportEditAndUsage(t *testing.T) {
	directory := t.TempDir()
	m, err := Open(directory)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = m.Save(Edit{ID: "test", Instructions: valid, Enabled: true, Activation: "auto"}); err != nil {
		t.Fatal(err)
	}
	d, err := m.Load("test", "chat", true)
	if err != nil || d.Name != "test-skill" {
		t.Fatal(d, err)
	}
	if _, err = m.Load("test", "chat", false); err != nil {
		t.Fatal(err)
	}
	m, err = Open(directory)
	if err != nil {
		t.Fatal(err)
	}
	d, err = m.Detail("test")
	if err != nil || d.Usage.Count != 1 {
		t.Fatal(d, err)
	}
	if err = os.WriteFile(filepath.Join(d.Path, "SKILL.md"), []byte("invalid"), 0600); err != nil {
		t.Fatal(err)
	}
	d, err = m.Detail("test")
	if err != nil || d.Error == "" {
		t.Fatal("invalid source cannot be repaired", d, err)
	}
	if _, err = m.Load("test", "chat", true); err == nil {
		t.Fatal("invalid skill loaded")
	}
	if _, err = m.Save(Edit{ID: "test", Instructions: valid, Enabled: true, Activation: "explicit"}); err != nil {
		t.Fatal(err)
	}
	if _, err = m.Remove("test"); err != nil {
		t.Fatal(err)
	}
	if _, err = os.Stat(filepath.Join(d.Path, "SKILL.md")); err != nil {
		t.Fatal("remove deleted source")
	}
	if _, err = m.Save(Edit{ID: "import", Path: d.Path, Enabled: true, Activation: "auto"}); err != nil {
		t.Fatal(err)
	}
	if _, err = m.Save(Edit{ID: "duplicate", Path: d.Path, Enabled: true, Activation: "auto"}); err == nil {
		t.Fatal("duplicate folder accepted")
	}
}
func TestReferenceBoundaries(t *testing.T) {
	m, err := Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	d, err := m.Detail("memo")
	if err != nil {
		t.Fatal(err)
	}
	outside := filepath.Join(t.TempDir(), "secret")
	os.WriteFile(outside, []byte("secret"), 0600)
	os.Symlink(outside, filepath.Join(d.Path, "escape"))
	for _, path := range []string{"../../secret", outside, "escape"} {
		if _, err = m.Read("memo", path); err == nil {
			t.Fatal("escaping reference read", path)
		}
	}
	if _, err = m.Save(Edit{ID: "../bad", Instructions: valid, Activation: "auto"}); err == nil {
		t.Fatal("path traversal ID accepted")
	}
}
func TestManifestValidation(t *testing.T) {
	for _, body := range []string{"text", "---\nname: missing\n---\nbody", "---\nname: ../bad\ndescription: hi\n---\nbody", "---\nname: valid\ndescription: hi\n---\n"} {
		if _, _, err := Validate(body); err == nil {
			t.Fatal("invalid manifest accepted", body)
		}
	}
	if _, _, err := Validate(valid); err != nil {
		t.Fatal(err)
	}
}
