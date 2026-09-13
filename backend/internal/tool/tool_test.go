package tool

import (
	"context"
	"encoding/json"
	"testing"
)

func stub(name string, mutating bool) Tool {
	return Tool{
		Name:        name,
		Description: name,
		Mutating:    mutating,
		Run:         func(context.Context, json.RawMessage) (string, error) { return "", nil },
	}
}

func withRegistry(t *testing.T, tools ...Tool) {
	t.Helper()
	old := registry
	registry = map[string]Tool{}
	for _, tl := range tools {
		Register(tl)
	}
	t.Cleanup(func() { registry = old })
}

func TestRegistryRoundTrip(t *testing.T) {
	withRegistry(t, stub("z", true))
	got, ok := Get("z")
	if !ok || got.Name != "z" {
		t.Fatalf("Get after Register = %+v, %v", got, ok)
	}
	if _, ok := Get("missing"); ok {
		t.Fatal("unknown tool must not resolve")
	}
	if !IsMutating("z") {
		t.Fatal("registered mutating tool reported read-only")
	}
	if IsMutating("missing") {
		t.Fatal("unknown tool must not report mutating")
	}
}

func TestSpecsAreSortedAndShaped(t *testing.T) {
	withRegistry(t, stub("zeta", true), stub("alpha", false))
	specs := Specs()
	if len(specs) != 2 || specs[0].Function.Name != "alpha" || specs[1].Function.Name != "zeta" {
		t.Fatalf("specs = %+v", specs)
	}
	if specs[0].Type != "function" {
		t.Fatalf("spec type = %q, want function", specs[0].Type)
	}
	if len(specs[0].Function.Parameters) == 0 {
		t.Fatalf("nil schema must default to an object: %+v", specs[0])
	}
}

func TestClipUTF8KeepsRuneBoundary(t *testing.T) {
	s, truncated := clipUTF8([]byte("héllo"), 2)
	if !truncated || s != "h" {
		t.Fatalf("clipUTF8 = %q, %v", s, truncated)
	}
	if s, truncated := clipUTF8([]byte("ok"), 8); truncated || s != "ok" {
		t.Fatalf("under-limit clip = %q, %v", s, truncated)
	}
}
