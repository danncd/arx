package agent

import (
	contextwindow "arx/internal/agent/context"
	attachment "arx/internal/media/attachments"
	session "arx/internal/sessions"
	"arx/internal/sessions/storage"
	"reflect"
	"testing"
)

func TestSavedImagesReplayAndCount(t *testing.T) {
	directory := t.TempDir()
	store, err := storage.Open(directory)
	if err != nil {
		t.Fatal(err)
	}
	images := []attachment.Image{{ID: "image-id", Name: "test.png"}}
	if err := store.Append(session.TranscriptChunk{Conversation: "chat", ID: "user", Role: "user", Images: images}); err != nil {
		t.Fatal(err)
	}
	store.Close()
	store, err = storage.Open(directory)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	history, err := History(store.Entries("chat"))
	if err != nil || len(history) != 1 || !reflect.DeepEqual(history[0].Images, images) {
		t.Fatal("images lost after restart", err)
	}
	withImages := contextwindow.Messages(history)
	history[0].Images = nil
	if withImages-contextwindow.Messages(history) != 1024 {
		t.Fatal("image token estimate is missing")
	}
	if store.Conversations()[0].Title != "Image" {
		t.Fatal("image-only chat lost its title")
	}
}
