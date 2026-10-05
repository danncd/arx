package jobs

import (
	"arx/internal/media/artifacts"
	"context"
	"errors"
	"testing"
)

func TestCancellationPersistsAndDoesNotPublishOutput(t *testing.T) {
	directory := t.TempDir()
	manager, err := Open(directory)
	if err != nil {
		t.Fatal(err)
	}
	started := make(chan struct{})
	done := make(chan error, 1)
	go func() {
		_, err := manager.Run(context.Background(), Job{Operation: "image"}, func(ctx context.Context, update func(float64, string)) (artifacts.Artifact, error) {
			update(0.2, "Generating")
			close(started)
			<-ctx.Done()
			return artifacts.Artifact{ID: "partial"}, ctx.Err()
		})
		done <- err
	}()
	<-started
	if !manager.Cancel("") {
		t.Fatal("cancel failed")
	}
	if !errors.Is(<-done, context.Canceled) {
		t.Fatal("missing cancellation")
	}
	restored, err := Open(directory)
	if err != nil {
		t.Fatal(err)
	}
	job := restored.Snapshot()[0]
	if job.State != "cancelled" || job.Output != nil {
		t.Fatalf("unexpected job: %#v", job)
	}
}

func TestRecoverInterruptedJob(t *testing.T) {
	store := Store{Directory: t.TempDir()}
	if err := store.Save(Job{ID: "0123456789abcdef0123456789abcdef", State: "running"}); err != nil {
		t.Fatal(err)
	}
	jobs, err := store.Recover()
	if err != nil || len(jobs) != 1 || jobs[0].State != "interrupted" {
		t.Fatalf("recovery: %#v %v", jobs, err)
	}
}
