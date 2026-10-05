package settings

import "testing"

func TestViewBatchIsAtomicAndPreservesOtherDrafts(t *testing.T) {
	store, err := Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if err := store.SaveViews(map[string]string{"draft:a": "one", "draft:b": "two"}); err != nil {
		t.Fatal(err)
	}
	if err := store.SaveViews(map[string]string{"draft:a": "changed", "": "invalid"}); err == nil {
		t.Fatal("Invalid batch accepted")
	}
	if store.Views()["draft:a"] != "one" {
		t.Fatal("Failed batch partially applied")
	}
	if err := store.SaveViews(map[string]string{"draft:a": "", "sidebar": "saved"}); err != nil {
		t.Fatal(err)
	}
	views := store.Views()
	if views["draft:a"] != "" || views["draft:b"] != "two" || views["sidebar"] != "saved" {
		t.Fatal(views)
	}
}
