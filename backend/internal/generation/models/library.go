package generation

import (
	"arx/internal/platform/atomicfile"
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sync"
	"time"
)

type Entry struct {
	Model    Model  `json:"model"`
	Status   string `json:"status"`
	Received int64  `json:"received"`
	Error    string `json:"error,omitempty"`
}

type State struct {
	Revision uint64            `json:"revision"`
	Models   []Entry           `json:"models"`
	Defaults map[string]string `json:"defaults"`
}

type Library struct {
	freeBytes func(string) (uint64, error)
	mutex     sync.Mutex
	directory string
	catalog   []Model
	state     State
	cancel    context.CancelFunc
	done      chan struct{}
	notify    func(State)
}

func Open(directory, assets string) (*Library, error) {
	catalog, err := Catalog(assets)
	if err != nil {
		return nil, err
	}
	library := &Library{directory: directory, catalog: catalog, state: State{Models: []Entry{}, Defaults: map[string]string{}}}
	data, err := os.ReadFile(filepath.Join(directory, "library.json"))
	if err == nil {
		if json.Unmarshal(data, &library.state) != nil {
			return nil, errors.New("Generation library is invalid")
		}
		if err := library.reconcile(); err != nil {
			return nil, err
		}
	} else if !os.IsNotExist(err) {
		return nil, err
	}
	if library.state.Defaults == nil {
		library.state.Defaults = map[string]string{}
	}
	library.state.Revision = uint64(time.Now().UnixMicro())
	return library, nil
}

func (l *Library) Catalog() []Model { return append([]Model{}, l.catalog...) }
func (l *Library) snapshotLocked() State {
	data, _ := json.Marshal(l.state)
	var state State
	json.Unmarshal(data, &state)
	return state
}
func (l *Library) Snapshot() State { l.mutex.Lock(); defer l.mutex.Unlock(); return l.snapshotLocked() }
func (l *Library) Subscribe(notify func(State)) {
	l.mutex.Lock()
	defer l.mutex.Unlock()
	l.notify = notify
}
func (l *Library) index(id string) int {
	for i, entry := range l.state.Models {
		if entry.Model.ID == id {
			return i
		}
	}
	return -1
}
func (l *Library) saveLocked() error {
	if err := os.MkdirAll(l.directory, 0700); err != nil {
		return err
	}
	data, err := json.Marshal(l.state)
	if err != nil {
		return err
	}
	return atomicfile.Replace(filepath.Join(l.directory, "library.json"), data, false)
}

func (l *Library) Configure(category, id string) error {
	l.mutex.Lock()
	defer l.mutex.Unlock()
	if category != "image" && category != "video" && category != "speech" {
		return errors.New("Unknown generation category")
	}
	if id != "" {
		index := l.index(id)
		if index < 0 || l.state.Models[index].Status != "installed" || l.state.Models[index].Model.Category != category {
			return errors.New("Choose an installed model for this category")
		}
	}
	previous := l.state.Defaults[category]
	l.state.Defaults[category] = id
	if err := l.saveLocked(); err != nil {
		l.state.Defaults[category] = previous
		return err
	}
	if l.notify != nil {
		l.state.Revision++
		l.notify(l.snapshotLocked())
	}
	return nil
}
func (l *Library) Resolve(category string) (Model, string, error) {
	l.mutex.Lock()
	defer l.mutex.Unlock()
	index := l.index(l.state.Defaults[category])
	if index < 0 || l.state.Models[index].Status != "installed" {
		return Model{}, "", errors.New("No model is configured. Configure a generation model in Settings, then try again")
	}
	model := l.state.Models[index].Model
	if model.Category != category || !l.complete(model) {
		return Model{}, "", errors.New("Model files are missing. Download again to repair them")
	}
	return model, filepath.Join(l.directory, model.ID), nil
}
