package keychain

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"testing"
)

func TestMigratedProfileKeepsExistingKeychainAccount(t *testing.T) {
	directory := t.TempDir()
	digest := sha256.Sum256([]byte("previous/profile"))
	identity := hex.EncodeToString(digest[:])
	if err := os.WriteFile(filepath.Join(directory, "keychain-account"), []byte(identity), 0600); err != nil {
		t.Fatal(err)
	}
	store, err := New(directory)
	if err != nil || store.account != identity {
		t.Fatalf("Credential identity changed: %v", err)
	}
	if err := os.WriteFile(filepath.Join(directory, "keychain-account"), []byte("invalid"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := New(directory); err == nil {
		t.Fatal("Invalid identity silently replaced")
	}
}
