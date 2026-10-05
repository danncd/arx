package local

import (
	"testing"
	"time"
)

func TestIdleModelUnloadsAfterFiveMinutes(t *testing.T) {
	now := time.Now()
	manager := &Manager{state: State{Runtime: Runtime{State: "ready", Model: "local:test"}}, lastUsed: now}
	if manager.idleExpired(now.Add(idleTimeout - time.Second)) {
		t.Fatal("Model unloaded before the idle timeout")
	}
	if !manager.idleExpired(now.Add(idleTimeout + time.Second)) {
		t.Fatal("Idle model stayed loaded")
	}
	if manager.state.Runtime.State != "stopping" {
		t.Fatal("Model did not enter the stopping state")
	}
}

func TestActiveUseDefersIdleUnload(t *testing.T) {
	now := time.Now()
	manager := &Manager{state: State{Runtime: Runtime{State: "ready", Model: "local:test"}}, lastUsed: now.Add(-idleTimeout)}
	release, err := manager.Hold("local:test")
	if err != nil {
		t.Fatal(err)
	}
	if manager.idleExpired(now.Add(2 * idleTimeout)) {
		t.Fatal("Active model unloaded")
	}
	release()
	release()
	if manager.active != 0 || manager.idleExpired(time.Now().Add(idleTimeout-time.Second)) {
		t.Fatal("Idle time was not reset after use")
	}
	if !manager.idleExpired(time.Now().Add(idleTimeout + time.Second)) {
		t.Fatal("Model stayed loaded after use ended")
	}
}

func TestIdleTimeoutCanChangeWhileLoaded(t *testing.T) {
	now := time.Now()
	manager := &Manager{state: State{Runtime: Runtime{State: "ready", Model: "local:test"}}, lastUsed: now}
	manager.SetIdleMinutes(10)
	if manager.idleExpired(now.Add(9 * time.Minute)) {
		t.Fatal("Model unloaded before the configured timeout")
	}
	if !manager.idleExpired(now.Add(10 * time.Minute)) {
		t.Fatal("Model stayed loaded past the configured timeout")
	}
}
