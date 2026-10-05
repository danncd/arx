package settings

import (
	"os"
	"path/filepath"
	"testing"
)

func TestAutoContinueDefaultsOnAndPersistsOffWithoutUnrelatedConfigureResettingIt(t *testing.T) {
	for _, legacy := range []bool{false, true} {
		directory := t.TempDir()
		if legacy {
			os.WriteFile(filepath.Join(directory, "settings.json"), []byte(`{"version":1,"run":{"model":"test","effort":""}}`), 0600)
		}
		store, err := Open(directory)
		if err != nil {
			t.Fatal(err)
		}
		if !store.Values().AutoContinue {
			t.Fatal("default not enabled")
		}
		off := false
		if _, err := store.Configure(store.Values().Run, store.Values().Directory, &off); err != nil {
			t.Fatal(err)
		}
		if _, err := store.Configure(Run{Model: "changed"}, store.Values().Directory); err != nil {
			t.Fatal(err)
		}
		reopened, err := Open(directory)
		if err != nil {
			t.Fatal(err)
		}
		if reopened.Values().AutoContinue {
			t.Fatal("off preference lost")
		}
	}
}
