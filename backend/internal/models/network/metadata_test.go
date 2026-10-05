package network

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestMetadataSharesConcurrentLookupsAndRefreshesExplicitly(t *testing.T) {
	var requests atomic.Int32
	entered, release := make(chan struct{}), make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if requests.Add(1) == 2 {
			close(entered)
			<-release
		}
		w.Write([]byte(fixture))
	}))
	defer server.Close()
	manager, _ := Open(t.TempDir())
	connected, err := manager.Connect(context.Background(), server.URL, "PC", "")
	if err != nil {
		t.Fatal(err)
	}
	id := connected.Models[0].Info.ID
	var pending sync.WaitGroup
	for range 20 {
		pending.Go(func() {
			info, err := manager.Model(id)
			if err != nil || info.ContextWindow != 8192 {
				t.Errorf("Invalid cached model: %+v %v", info, err)
			}
		})
	}
	<-entered
	close(release)
	pending.Wait()
	if requests.Load() != 2 {
		t.Fatalf("Concurrent lookups sent %d requests", requests.Load())
	}
	for range 10 {
		manager.Model(id)
	}
	if requests.Load() != 2 {
		t.Fatal("Context polling fetched metadata again")
	}
	manager.State(context.Background())
	if requests.Load() != 3 {
		t.Fatal("Explicit refresh did not fetch metadata")
	}
	manager.cacheMutex.Lock()
	manager.cache[connected.ID].fetched = time.Now().Add(-metadataLifetime)
	manager.cacheMutex.Unlock()
	manager.Model(id)
	if requests.Load() != 4 {
		t.Fatal("Expired metadata did not refresh")
	}
}
