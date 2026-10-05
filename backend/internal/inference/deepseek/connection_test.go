package deepseek

import (
	model "arx/internal/models"
	"context"
	"errors"
	"sync"
	"testing"
	"time"
)

type memoryCredentials struct {
	key  string
	fail bool
}

func (s *memoryCredentials) Load(context.Context) (string, error) {
	if s.fail {
		return "", errors.New("Keychain locked")
	}
	return s.key, nil
}
func (s *memoryCredentials) Save(_ context.Context, key string) error {
	if s.fail {
		return errors.New("Keychain locked")
	}
	s.key = key
	return nil
}
func (s *memoryCredentials) Delete(context.Context) error {
	if s.fail {
		return errors.New("Keychain locked")
	}
	s.key = ""
	return nil
}

type discovery struct {
	calls int
	fail  bool
}

func (d *discovery) Models(_ context.Context, key string) ([]model.Info, error) {
	d.calls++
	if key == "invalid" || d.fail {
		return nil, errors.New("Provider rejected request")
	}
	return []model.Info{{ID: "model", Name: "Model", Thinking: &model.Thinking{Efforts: []string{"low", "high"}, DefaultEffort: "high", CanDisable: true}}}, nil
}

func TestConnectionRestoreCacheAndDisconnect(t *testing.T) {
	store := &memoryCredentials{key: "saved"}
	provider := &discovery{}
	connection := NewConnection(store, provider)
	var group sync.WaitGroup
	for range 8 {
		group.Go(func() {
			result, err := connection.Status(context.Background(), false)
			if err != nil || !result.Connected {
				t.Errorf("status: %+v %v", result, err)
			}
		})
	}
	group.Wait()
	if provider.calls != 1 {
		t.Fatalf("discovery called %d times", provider.calls)
	}
	connection.checked = time.Now().Add(-11 * time.Minute)
	if _, err := connection.Status(context.Background(), false); err != nil || provider.calls != 2 {
		t.Fatal("expired catalog not refreshed")
	}
	result, err := connection.Disconnect(context.Background())
	if err != nil || result.Configured || len(result.Models) != 0 || store.key != "" {
		t.Fatalf("disconnect: %+v %v", result, err)
	}
}

func TestFailedReplacementPreservesWorkingKey(t *testing.T) {
	store := &memoryCredentials{}
	provider := &discovery{}
	connection := NewConnection(store, provider)
	if _, err := connection.Connect(context.Background(), "working"); err != nil {
		t.Fatal(err)
	}
	if _, err := connection.Connect(context.Background(), "invalid"); err == nil {
		t.Fatal("bad key accepted")
	}
	if store.key != "working" || connection.key != "working" {
		t.Fatal("failed replacement changed key")
	}
	store.fail = true
	if _, err := connection.Connect(context.Background(), "replacement"); err == nil {
		t.Fatal("Keychain failure ignored")
	}
	if connection.key != "working" {
		t.Fatal("unsaved key became active")
	}
	if _, err := connection.Disconnect(context.Background()); err == nil {
		t.Fatal("failed delete ignored")
	}
	if connection.key != "working" {
		t.Fatal("failed delete disconnected")
	}
}

func TestOfflineRefreshKeepsModelsWithoutClaimingConnection(t *testing.T) {
	store := &memoryCredentials{key: "saved"}
	provider := &discovery{}
	connection := NewConnection(store, provider)
	connection.Status(context.Background(), false)
	provider.fail = true
	result, err := connection.Status(context.Background(), true)
	if err != nil || result.Connected || !result.Configured || len(result.Models) != 1 || result.Error == "" {
		t.Fatalf("offline: %+v %v", result, err)
	}
}

func TestSelectionValidationUsesActualCapabilities(t *testing.T) {
	connection := NewConnection(&memoryCredentials{key: "saved"}, &discovery{})
	connection.Status(context.Background(), false)
	for _, effort := range []string{"low", "high", "none"} {
		if err := connection.Validate("model", effort); err != nil {
			t.Fatal(err)
		}
	}
	for _, pair := range [][2]string{{"missing", "high"}, {"model", "medium"}} {
		if err := connection.Validate(pair[0], pair[1]); err == nil {
			t.Fatal("unsupported selection accepted")
		}
	}
}
