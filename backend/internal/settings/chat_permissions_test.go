package settings

import (
	permission "arx/internal/permissions"
	"testing"
)

func TestChatPermissionsAreIndependentAndPersist(t *testing.T) {
	directory := t.TempDir()
	store, err := Open(directory)
	if err != nil {
		t.Fatal(err)
	}
	initial := store.ChatPermissions("new")
	if initial.Mode != permission.Folders || len(initial.Roots) != 1 || initial.Roots[0] != store.Values().Directory {
		t.Fatal("new chat did not default to Auto")
	}
	if _, err := store.SaveChatPermissions("first", permission.Policy{Mode: permission.Ask}); err != nil {
		t.Fatal(err)
	}
	saved, err := store.SaveChatPermissions("second", permission.Policy{Mode: permission.Full})
	if err != nil {
		t.Fatal(err)
	}
	saved.ChatPermissions["first"] = permission.Policy{Mode: permission.Full}
	if store.ChatPermissions("first").Mode != permission.Ask {
		t.Fatal("caller mutated saved policy")
	}
	reopened, err := Open(directory)
	if err != nil {
		t.Fatal(err)
	}
	if reopened.ChatPermissions("first").Mode != permission.Ask || reopened.ChatPermissions("second").Mode != permission.Full || reopened.ChatPermissions("third").Mode != permission.Folders {
		t.Fatal("chat policies leaked or failed to persist")
	}
	if _, err := reopened.SaveChatPermissions("first", permission.Policy{Mode: "invalid"}); err == nil {
		t.Fatal("invalid mode accepted")
	}
	if reopened.ChatPermissions("first").Mode != permission.Ask {
		t.Fatal("failed save changed permissions")
	}
}
