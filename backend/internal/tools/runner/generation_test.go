package runner

import (
	"arx/internal/generation"
	"arx/internal/permissions"
	"arx/internal/tools"
	"context"
	"testing"
	"time"
)

func TestGenerationUsesPendingCancellationAndIdleGuard(t *testing.T) {
	permissionsManager := &permissions.Manager{}
	executor := New(permissionsManager, nil)
	executor.Generation = func() (*generation.Service, error) { return &generation.Service{}, nil }
	pending := make(chan *permissions.Request, 4)
	permissionsManager.Subscribe(func(request *permissions.Request) { pending <- request })
	done := make(chan error, 1)
	go func() {
		_, err := executor.Run(context.Background(), tools.Call{ID: "generation", Name: "generate", Arguments: []byte(`{"operation":"image","prompt":"test","width":512,"height":512}`)}, func() (string, permissions.Policy) { return "", permissions.Policy{Mode: permissions.Ask} })
		done <- err
	}()
	select {
	case request := <-pending:
		if request == nil {
			t.Fatal("missing approval")
		}
	case <-time.After(time.Second):
		t.Fatal("generation skipped approval")
	}
	if err := executor.Idle(func() error { return nil }); err == nil {
		t.Fatal("idle edit allowed during generation approval")
	}
	if !executor.Cancel("generation") {
		t.Fatal("generation cancellation not routed")
	}
	select {
	case err := <-done:
		if err != context.Canceled {
			t.Fatal(err)
		}
	case <-time.After(time.Second):
		t.Fatal("generation did not cancel")
	}
	if permissionsManager.Pending() != nil {
		t.Fatal("pending generation approval survived")
	}
}
