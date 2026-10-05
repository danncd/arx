package runner

import (
	permission "arx/internal/permissions"
	tool "arx/internal/tools"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestApprovalGateCancelAndSingleAction(t *testing.T) {
	root := t.TempDir()
	manager := &permission.Manager{}
	runner := New(manager, nil)
	events := make(chan *permission.Request, 4)
	manager.Subscribe(func(request *permission.Request) { events <- request })
	configuration := func() (string, permission.Policy) { return root, permission.Policy{Mode: permission.Ask} }
	raw, _ := json.Marshal(map[string]string{"operation": "write", "path": "file", "content": "saved"})
	done := make(chan error, 1)
	go func() {
		_, err := runner.Run(context.Background(), tool.Call{ID: "one", Name: "files", Arguments: raw}, configuration)
		done <- err
	}()
	select {
	case <-events:
	case <-time.After(time.Second):
		t.Fatal("no approval")
	}
	if _, err := os.Stat(filepath.Join(root, "file")); !os.IsNotExist(err) {
		t.Fatal("file written before approval")
	}
	if err := runner.Idle(func() error { return nil }); err == nil {
		t.Fatal("permissions changed during pending action")
	}
	if _, err := runner.Run(context.Background(), tool.Call{ID: "two", Name: "files", Arguments: raw}, configuration); err == nil {
		t.Fatal("concurrent tool allowed")
	}
	if !runner.Cancel("one") {
		t.Fatal("pending action not cancelled")
	}
	if err := <-done; err != context.Canceled {
		t.Fatalf("cancel: %v", err)
	}
	if manager.Pending() != nil {
		t.Fatal("approval survived cancellation")
	}
	if err := runner.Idle(func() error { return nil }); err != nil {
		t.Fatal(err)
	}
}
