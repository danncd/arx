package network

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
)

func TestSavedServerRefreshesActiveContextAndLoadedState(t *testing.T) {
	var updated atomic.Bool
	var offline atomic.Bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if offline.Load() {
			w.WriteHeader(503)
			return
		}
		data := fixture
		if updated.Load() {
			data = strings.ReplaceAll(data, "8192", "16384")
		}
		w.Write([]byte(data))
	}))
	defer server.Close()
	manager, _ := Open(t.TempDir())
	connected, err := manager.Connect(context.Background(), server.URL, "PC", "")
	if err != nil {
		t.Fatal(err)
	}
	updated.Store(true)
	state := manager.State(context.Background())
	if state[0].Models[0].Info.ContextWindow != 16384 {
		t.Fatal("State kept stale context")
	}
	info, err := manager.Model(connected.Models[0].Info.ID)
	if err != nil || info.ContextWindow != 16384 {
		t.Fatal("Chat kept stale context", err)
	}
	offline.Store(true)
	state = manager.State(context.Background())
	if state[0].Connected || len(state[0].Models) > 0 {
		t.Fatal("Offline models stayed available")
	}
}
