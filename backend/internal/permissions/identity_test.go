package permissions

import (
	"context"
	"testing"
)

func TestCancelledApprovalCannotAuthorizeItsSuccessor(t *testing.T) {
	manager := &Manager{}
	requests := make(chan *Request, 4)
	manager.Subscribe(func(r *Request) {
		if r != nil {
			requests <- r
		}
	})
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() {
		done <- manager.Authorize(ctx, Policy{Mode: Ask}, Action{Tool: "web", URL: "https://first.example"})
	}()
	old := <-requests
	cancel()
	<-done
	go func() {
		done <- manager.Authorize(context.Background(), Policy{Mode: Ask}, Action{Tool: "web", URL: "https://second.example"})
	}()
	next := <-requests
	if old.ID == next.ID {
		t.Fatal("approval identity reused")
	}
	if err := manager.Respond(old.ID, true); err == nil {
		t.Fatal("stale approval accepted")
	}
	if err := manager.Respond(next.ID, false); err != nil {
		t.Fatal(err)
	}
	if err := <-done; err == nil {
		t.Fatal("denial ignored")
	}
}
