package permissions

import (
	"context"
	"path/filepath"
	"reflect"
	"testing"
	"time"
)

func TestApprovalIsBoundToOneAction(t *testing.T) {
	manager := &Manager{}
	events := make(chan *Request, 4)
	manager.Subscribe(func(request *Request) { events <- request })
	done := make(chan error, 1)
	go func() {
		done <- manager.Authorize(context.Background(), Policy{Mode: Ask}, Action{Tool: "files", Writes: true})
	}()
	select {
	case request := <-events:
		if request.ID == "" {
			t.Fatal("wrong request")
		}
	case <-time.After(time.Second):
		t.Fatal("approval not requested")
	}
	if err := manager.Respond("different", true); err == nil {
		t.Fatal("wrong approval accepted")
	}
	if err := manager.Respond(manager.Pending().ID, true); err != nil {
		t.Fatal(err)
	}
	if err := <-done; err != nil {
		t.Fatal(err)
	}
	if request := <-events; request != nil {
		t.Fatal("approval not cleared")
	}
	if err := manager.Respond("one", true); err == nil {
		t.Fatal("approval reused")
	}
}

func TestCancellationAndDenialReleasePendingApproval(t *testing.T) {
	for _, cancelled := range []bool{true, false} {
		manager := &Manager{}
		ready := make(chan struct{}, 1)
		manager.Subscribe(func(request *Request) {
			if request != nil {
				ready <- struct{}{}
			}
		})
		ctx, cancel := context.WithCancel(context.Background())
		done := make(chan error, 1)
		go func() { done <- manager.Authorize(ctx, Policy{Mode: Ask}, Action{Tool: "bash"}) }()
		<-ready
		if cancelled {
			cancel()
		} else {
			manager.Respond(manager.Pending().ID, false)
		}
		if err := <-done; err == nil {
			t.Fatal("denied/cancelled action approved")
		}
		cancel()
		if manager.Pending() != nil {
			t.Fatal("stale approval")
		}
	}
}

func TestModesAndFolderBoundary(t *testing.T) {
	manager := &Manager{}
	root := t.TempDir()
	if err := manager.Authorize(context.Background(), Policy{Mode: Folders, Roots: []string{root}}, Action{Tool: "files", Operation: "read"}); err != nil {
		t.Fatal(err)
	}
	for _, mode := range []Mode{Full, Folders} {
		err := manager.Authorize(context.Background(), Policy{Mode: mode, Roots: []string{root}}, Action{Tool: "files", Writes: true, Path: filepath.Join(root, "file")})
		if err != nil {
			t.Fatal(err)
		}
	}
	if err := manager.Authorize(context.Background(), Policy{Mode: Folders, Roots: []string{root}}, Action{Tool: "files", Writes: true, Path: root + "-other/file"}); err == nil {
		t.Fatal("prefix escape accepted")
	}
}

func TestAskWaitsForReadsAndWebRequests(t *testing.T) {
	for _, action := range []Action{
		{Tool: "files", Operation: "read"},
		{Tool: "files", Operation: "search"},
		{Tool: "web", Operation: "search", Query: "Queens College"},
		{Tool: "web", Operation: "fetch", URL: "https://example.com"},
	} {
		t.Run(action.Tool+"-"+action.Operation, func(t *testing.T) {
			manager := &Manager{}
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			pending := make(chan Request, 1)
			manager.Subscribe(func(request *Request) {
				if request != nil {
					pending <- *request
				}
			})
			done := make(chan error, 1)
			go func() { done <- manager.Authorize(ctx, Policy{Mode: Ask}, action) }()
			select {
			case request := <-pending:
				if !reflect.DeepEqual(request.Action, action) {
					t.Fatal("approval lost action details")
				}
			case <-done:
				t.Fatal("Ask allowed action without approval")
			case <-time.After(time.Second):
				t.Fatal("no approval appeared")
			}
			if err := manager.Respond(manager.Pending().ID, false); err != nil {
				t.Fatal(err)
			}
			if err := <-done; err == nil {
				t.Fatal("denied action allowed")
			}
		})
	}
}
