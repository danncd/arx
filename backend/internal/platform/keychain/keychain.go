package keychain

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"time"
)

type Store interface {
	Load(context.Context) (string, error)
	Save(context.Context, string) error
	Delete(context.Context) error
}

type Keychain struct{ account string }

func New(directory string) (*Keychain, error) {
	digest := sha256.Sum256([]byte(filepath.Clean(directory)))
	account := hex.EncodeToString(digest[:])
	if saved, err := os.ReadFile(filepath.Join(directory, "keychain-account")); err == nil {
		value, decodeErr := hex.DecodeString(string(saved))
		if decodeErr != nil || len(value) != 32 {
			return nil, errors.New("Invalid saved Keychain identity")
		}
		account = string(saved)
	} else if !os.IsNotExist(err) {
		return nil, err
	}
	return &Keychain{account: account}, nil
}

func (k *Keychain) Load(ctx context.Context) (string, error) { return k.request(ctx, "load", "") }
func (k *Keychain) Save(ctx context.Context, key string) error {
	_, err := k.request(ctx, "save", key)
	return err
}
func (k *Keychain) Delete(ctx context.Context) error {
	_, err := k.request(ctx, "delete", "")
	return err
}

func (k *Keychain) request(ctx context.Context, operation, key string) (string, error) {
	if runtime.GOOS != "darwin" {
		return "", errors.New("Secure credential storage currently requires macOS")
	}
	executable, err := os.Executable()
	if err != nil {
		return "", errors.New("Could not locate Keychain helper")
	}
	body, err := json.Marshal(struct {
		Operation string `json:"operation"`
		Account   string `json:"account"`
		Key       string `json:"key,omitempty"`
	}{operation, k.account, key})
	if err != nil {
		return "", err
	}
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	command := exec.CommandContext(ctx, filepath.Join(filepath.Dir(executable), "arx-keychain"))
	command.Stdin = bytes.NewReader(body)
	output, err := command.Output()
	if err != nil {
		return "", errors.New("Could not access macOS Keychain")
	}
	var response struct {
		Key   string `json:"key"`
		Error string `json:"error"`
	}
	if json.Unmarshal(output, &response) != nil {
		return "", errors.New("Invalid Keychain response")
	}
	if response.Error != "" {
		return "", errors.New(response.Error)
	}
	return response.Key, nil
}
