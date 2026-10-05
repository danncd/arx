package settings

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"sync"
	"testing"
)

func TestLegacySettingsAndViewsSurviveUpdates(t *testing.T) {
	directory := t.TempDir()
	for name, body := range map[string]string{
		"settings.json": `{"version":1,"run":{"model":"deepseek-chat","effort":"high"}}`,
		"views.json":    `{"version":1,"views":{"startup-mode":"new","unknown-old-preference":"keep"}}`,
	} {
		if err := os.WriteFile(filepath.Join(directory, name), []byte(body), 0600); err != nil {
			t.Fatal(err)
		}
	}
	store, err := Open(directory)
	if err != nil {
		t.Fatal(err)
	}
	if store.Values().Run.Model != "deepseek-chat" || store.Views()["startup-mode"] != "new" {
		t.Fatal("legacy settings not read")
	}
	if err := store.SaveView("draft:new", "Draft 🌱"); err != nil {
		t.Fatal(err)
	}
	run := Run{Model: "deepseek-reasoner", Effort: "medium"}
	if _, err := store.Configure(run, directory); err != nil {
		t.Fatal(err)
	}
	resolved, _ := filepath.EvalSymlinks(directory)
	reopened, err := Open(directory)
	if err != nil {
		t.Fatal(err)
	}
	if reopened.Values().Run != run || reopened.Values().Directory != resolved || reopened.Views()["draft:new"] != "Draft 🌱" || reopened.Views()["unknown-old-preference"] != "keep" {
		t.Fatal("settings not restored")
	}
	for _, name := range []string{"settings.json", "views.json"} {
		info, err := os.Stat(filepath.Join(directory, name))
		if err != nil || info.Mode().Perm() != 0600 {
			t.Fatalf("permissions %s: %v", name, err)
		}
	}
}

func TestConcurrentViewSavesDoNotLoseValues(t *testing.T) {
	directory := t.TempDir()
	store, err := Open(directory)
	if err != nil {
		t.Fatal(err)
	}
	var group sync.WaitGroup
	for index := 0; index < 20; index++ {
		group.Go(func() {
			if err := store.SaveView(fmt.Sprint(index), "saved"); err != nil {
				t.Error(err)
			}
		})
	}
	group.Wait()
	reopened, err := Open(directory)
	if err != nil || len(reopened.Views()) != 20 {
		t.Fatalf("lost preferences: %v", err)
	}
}

func TestFailedSaveDoesNotChangeMemory(t *testing.T) {
	directory := t.TempDir()
	store, err := Open(directory)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(directory, "views.json"), 0700); err != nil {
		t.Fatal(err)
	}
	if err := store.SaveView("startup-mode", "new"); err == nil {
		t.Fatal("save should fail")
	}
	if store.Views()["startup-mode"] != "" {
		t.Fatal("failed save changed memory")
	}
	previous := store.Values()
	if _, err := store.Configure(Run{Model: "new"}, filepath.Join(directory, "missing")); err == nil {
		t.Fatal("missing directory accepted")
	}
	if !reflect.DeepEqual(store.Values(), previous) {
		t.Fatal("invalid directory changed settings")
	}
}

func TestDamagedOrFutureSettingsAreNotOverwritten(t *testing.T) {
	for _, body := range []string{`{"version":2,"run":{}}`, `{"version":1,`} {
		directory := t.TempDir()
		path := filepath.Join(directory, "settings.json")
		os.WriteFile(path, []byte(body), 0600)
		if _, err := Open(directory); err == nil {
			t.Fatal("invalid settings accepted")
		}
		after, _ := os.ReadFile(path)
		if string(after) != body {
			t.Fatal("settings overwritten")
		}
	}
}

func TestLocalIdleTimeoutPersists(t *testing.T) {
	directory := t.TempDir()
	store, err := Open(directory)
	if err != nil {
		t.Fatal(err)
	}
	if store.Values().LocalIdleMinutes != 5 {
		t.Fatal("Expected a five-minute default")
	}
	if _, err := store.ConfigureLocalIdleMinutes(10); err != nil {
		t.Fatal(err)
	}
	if _, err := store.ConfigureLocalIdleMinutes(3); err == nil {
		t.Fatal("Invalid timeout accepted")
	}
	reopened, err := Open(directory)
	if err != nil {
		t.Fatal(err)
	}
	if reopened.Values().LocalIdleMinutes != 10 {
		t.Fatal("Timeout was not restored")
	}
}
