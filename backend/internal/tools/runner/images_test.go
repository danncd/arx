package runner

import (
	attachment "arx/internal/media/attachments"
	permission "arx/internal/permissions"
	tool "arx/internal/tools"
	"bytes"
	"context"
	"encoding/json"
	"image"
	"image/jpeg"
	"image/png"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestImageReadRequiresApprovalAndSavesAnIndependentCopy(t *testing.T) {
	directory := t.TempDir()
	path := filepath.Join(directory, "photo.jpg")
	var data bytes.Buffer
	jpeg.Encode(&data, image.NewRGBA(image.Rect(0, 0, 2400, 1200)), nil)
	os.WriteFile(path, data.Bytes(), 0600)
	images := &attachment.Store{Directory: filepath.Join(directory, "profile")}
	manager := &permission.Manager{}
	runner := New(manager, images)
	pending := make(chan *permission.Request, 2)
	manager.Subscribe(func(request *permission.Request) {
		if request != nil {
			pending <- request
		}
	})
	done := make(chan tool.Result, 1)
	failure := make(chan error, 1)
	args, _ := json.Marshal(map[string]string{"operation": "image", "path": path})
	go func() {
		result, err := runner.Run(context.Background(), tool.Call{ID: "image", Name: "files", Arguments: args}, func() (string, permission.Policy) { return directory, permission.Policy{Mode: permission.Ask} })
		done <- result
		failure <- err
	}()
	select {
	case request := <-pending:
		if request.Action.Operation != "image" || request.Action.Writes {
			t.Fatal("incorrect permission")
		}
	case <-time.After(time.Second):
		t.Fatal("missing approval")
	}
	if _, err := os.Stat(filepath.Join(images.Directory, "attachments")); !os.IsNotExist(err) {
		t.Fatal("image processed before approval")
	}
	if err := manager.Respond(manager.Pending().ID, true); err != nil {
		t.Fatal(err)
	}
	result := <-done
	if err := <-failure; err != nil {
		t.Fatal(err)
	}
	if len(result.Images) != 1 || len(result.ImageData) > 0 {
		t.Fatal("image was not converted to a reference")
	}
	os.Remove(path)
	saved, err := images.Read(result.Images[0].ID)
	if err != nil {
		t.Fatal(err)
	}
	config, err := png.DecodeConfig(bytes.NewReader(saved))
	if err != nil || config.Width != 2048 || config.Height != 1024 {
		t.Fatal("image sizing changed its aspect ratio", config, err)
	}
	encoded, _ := json.Marshal(result)
	if len(encoded) > 2048 {
		t.Fatal("binary image leaked into transcript")
	}
	args, _ = json.Marshal(map[string]string{"operation": "image", "path": "arx-image:" + result.Images[0].ID})
	go func() {
		reopened, err := runner.Run(context.Background(), tool.Call{ID: "reopen", Name: "files", Arguments: args}, func() (string, permission.Policy) { return directory, permission.Policy{Mode: permission.Ask} })
		done <- reopened
		failure <- err
	}()
	select {
	case request := <-pending:
		if request.Action.Operation != "image" {
			t.Fatal("saved image bypassed its approval")
		}
	case <-time.After(time.Second):
		t.Fatal("saved image approval missing")
	}
	if err := manager.Respond(manager.Pending().ID, true); err != nil {
		t.Fatal(err)
	}
	reopened := <-done
	if err := <-failure; err != nil {
		t.Fatal(err)
	}
	if len(reopened.Images) != 1 || reopened.Images[0].ID != result.Images[0].ID {
		t.Fatal("could not reopen saved image after original was removed")
	}
}

func TestDeniedImageReadDoesNotTouchTheFile(t *testing.T) {
	directory := t.TempDir()
	manager := &permission.Manager{}
	runner := New(manager, &attachment.Store{Directory: directory})
	pending := make(chan *permission.Request, 1)
	manager.Subscribe(func(request *permission.Request) {
		if request != nil {
			pending <- request
		}
	})
	done := make(chan error, 1)
	go func() {
		_, err := runner.Run(context.Background(), tool.Call{ID: "denied", Name: "files", Arguments: json.RawMessage(`{"operation":"image","path":"missing.png"}`)}, func() (string, permission.Policy) { return directory, permission.Policy{Mode: permission.Ask} })
		done <- err
	}()
	select {
	case <-pending:
	case <-time.After(time.Second):
		t.Fatal("missing approval")
	}
	manager.Respond(manager.Pending().ID, false)
	if err := <-done; err == nil || err.Error() != "Permission denied" {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(directory, "attachments")); !os.IsNotExist(err) {
		t.Fatal("image saved after denial")
	}
}
