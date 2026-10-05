package generation

import (
	download "arx/internal/platform/download"
	"context"
	"errors"
	"os"
	"path/filepath"
	"syscall"
	"time"
)

func (l *Library) Download(id string) error {
	l.mutex.Lock()
	defer l.mutex.Unlock()
	if l.cancel != nil {
		return errors.New("Pause the current download first")
	}
	var model Model
	for _, option := range l.catalog {
		if option.ID == id {
			model = option
			break
		}
	}
	if model.ID == "" {
		return errors.New("Unsupported generation model")
	}
	if err := os.MkdirAll(l.directory, 0700); err != nil {
		return err
	}
	index := l.index(id)
	if index >= 0 && l.state.Models[index].Status == "installed" && l.complete(model) {
		return nil
	}
	var remaining int64
	for _, file := range model.Files {
		n, err := download.Remaining(context.Background(), file, filepath.Join(l.directory, model.ID, file.Name))
		if err != nil {
			return err
		}
		remaining += n
	}
	if err := l.checkSpace(remaining); err != nil {
		return err
	}
	entry := Entry{Model: model, Status: "downloading"}
	if index < 0 {
		index = len(l.state.Models)
		l.state.Models = append(l.state.Models, entry)
	} else {
		l.state.Models[index] = entry
	}
	if err := l.saveLocked(); err != nil {
		return err
	}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	l.cancel, l.done = cancel, done
	if l.notify != nil {
		l.state.Revision++
		l.notify(l.snapshotLocked())
	}
	go l.transfer(ctx, model, done)
	return nil
}

func (l *Library) transfer(ctx context.Context, model Model, done chan struct{}) {
	defer close(done)
	var received int64
	var failure error
	last := time.Time{}
	for index, file := range model.Files {
		failure = download.TransferChecked(ctx, file, filepath.Join(l.directory, model.ID, file.Name), func(n int64) {
			if time.Since(last) < 150*time.Millisecond && n != file.Size {
				return
			}
			last = time.Now()
			l.mutex.Lock()
			l.state.Models[l.index(model.ID)].Received = received + n
			if l.notify != nil {
				l.state.Revision++
				l.notify(l.snapshotLocked())
			}
			l.mutex.Unlock()
		}, func(needed int64) error {
			for _, next := range model.Files[index+1:] {
				remaining, err := download.Remaining(ctx, next, filepath.Join(l.directory, model.ID, next.Name))
				if err != nil {
					return err
				}
				needed += remaining
			}
			return l.checkSpace(needed)
		})
		if failure != nil {
			break
		}
		received += file.Size
	}
	l.mutex.Lock()
	defer l.mutex.Unlock()
	held := &l.state.Models[l.index(model.ID)]
	held.Status, held.Received = "installed", received
	if ctx.Err() != nil {
		held.Status = "paused"
	} else if failure != nil {
		held.Status, held.Error = "failed", failure.Error()
	}
	if err := l.saveLocked(); err != nil {
		held.Status, held.Error = "failed", err.Error()
	}
	l.cancel = nil
	if l.notify != nil {
		l.state.Revision++
		l.notify(l.snapshotLocked())
	}
}

func (l *Library) Pause() {
	l.mutex.Lock()
	cancel, done := l.cancel, l.done
	l.mutex.Unlock()
	if cancel != nil {
		cancel()
		<-done
	}
}

func (l *Library) Remove(id string) error {
	l.mutex.Lock()
	defer l.mutex.Unlock()
	if l.cancel != nil {
		return errors.New("Pause the download first")
	}
	index := l.index(id)
	if index < 0 {
		return errors.New("Model was not found")
	}
	if filepath.Base(id) != id || id == "." || id == ".." {
		return errors.New("Invalid model ID")
	}
	if err := os.RemoveAll(filepath.Join(l.directory, id)); err != nil {
		return err
	}
	l.state.Models = append(l.state.Models[:index], l.state.Models[index+1:]...)
	for category, held := range l.state.Defaults {
		if held == id {
			l.state.Defaults[category] = ""
		}
	}
	if err := l.saveLocked(); err != nil {
		return err
	}
	if l.notify != nil {
		l.state.Revision++
		l.notify(l.snapshotLocked())
	}
	return nil
}

const generationDiskMargin = 512 << 20

func (l *Library) checkSpace(remaining int64) error {
	query := l.freeBytes
	if query == nil {
		query = availableGenerationBytes
	}
	available, err := query(l.directory)
	if err != nil {
		return err
	}
	if uint64(remaining)+generationDiskMargin > available {
		return errors.New("Not enough disk space for this model")
	}
	return nil
}
func availableGenerationBytes(directory string) (uint64, error) {
	var disk syscall.Statfs_t
	if err := syscall.Statfs(directory, &disk); err != nil {
		return 0, err
	}
	return disk.Bavail * uint64(disk.Bsize), nil
}
