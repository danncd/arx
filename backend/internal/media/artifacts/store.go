package artifacts

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"regexp"
)

type Artifact struct {
	ID       string  `json:"id"`
	Name     string  `json:"name"`
	MIME     string  `json:"mime"`
	Size     int64   `json:"size"`
	Width    int     `json:"width,omitempty"`
	Height   int     `json:"height,omitempty"`
	Duration float64 `json:"duration,omitempty"`
}

type Store struct{ Directory string }

var validID = regexp.MustCompile(`^[a-f0-9]{32}$`)
var formats = map[string]string{"image/png": ".png", "audio/wav": ".wav", "video/mp4": ".mp4"}

func (s Store) Save(source string, artifact Artifact) (Artifact, error) {
	extension, ok := formats[artifact.MIME]
	if !ok {
		return Artifact{}, errors.New("Unsupported media format")
	}
	input, err := os.Open(source)
	if err != nil {
		return Artifact{}, err
	}
	defer input.Close()
	info, err := input.Stat()
	if err != nil || !info.Mode().IsRegular() || info.Size() == 0 || info.Size() > 2<<30 {
		return Artifact{}, errors.New("Generated media is empty or exceeds 2 GiB")
	}
	if err := s.verify(input, artifact.MIME); err != nil {
		return Artifact{}, err
	}
	if _, err := input.Seek(0, io.SeekStart); err != nil {
		return Artifact{}, err
	}
	var token [16]byte
	if _, err := rand.Read(token[:]); err != nil {
		return Artifact{}, err
	}
	artifact.ID = hex.EncodeToString(token[:])
	artifact.Name = filepath.Base(artifact.Name)
	if artifact.Name == "." || artifact.Name == string(filepath.Separator) || artifact.Name == "" {
		artifact.Name = "generation" + extension
	}
	root := filepath.Join(s.Directory, "media")
	if err := os.MkdirAll(root, 0700); err != nil {
		return Artifact{}, err
	}
	stage, err := os.MkdirTemp(root, ".saving-")
	if err != nil {
		return Artifact{}, err
	}
	defer os.RemoveAll(stage)
	output, err := os.OpenFile(filepath.Join(stage, "output"+extension), os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		return Artifact{}, err
	}
	size, copyErr := io.Copy(output, io.LimitReader(input, (2<<30)+1))
	closeErr := output.Close()
	if copyErr != nil {
		return Artifact{}, copyErr
	}
	if closeErr != nil {
		return Artifact{}, closeErr
	}
	if size != info.Size() {
		return Artifact{}, errors.New("Generated media changed while saving")
	}
	artifact.Size = size
	encoded, err := json.Marshal(artifact)
	if err != nil {
		return Artifact{}, err
	}
	if err := os.WriteFile(filepath.Join(stage, "metadata.json"), encoded, 0600); err != nil {
		return Artifact{}, err
	}
	if err := os.Rename(stage, filepath.Join(root, artifact.ID)); err != nil {
		return Artifact{}, err
	}
	return artifact, nil
}

func (s Store) Resolve(id string) (Artifact, string, error) {
	if !validID.MatchString(id) {
		return Artifact{}, "", errors.New("Invalid media reference")
	}
	directory := filepath.Join(s.Directory, "media", id)
	data, err := os.ReadFile(filepath.Join(directory, "metadata.json"))
	if err != nil {
		return Artifact{}, "", errors.New("Saved media is unavailable")
	}
	var artifact Artifact
	if json.Unmarshal(data, &artifact) != nil || artifact.ID != id || formats[artifact.MIME] == "" {
		return Artifact{}, "", errors.New("Saved media metadata is invalid")
	}
	path := filepath.Join(directory, "output"+formats[artifact.MIME])
	info, err := os.Lstat(path)
	if err != nil || !info.Mode().IsRegular() || info.Size() != artifact.Size {
		return Artifact{}, "", errors.New("Saved media is missing or damaged")
	}
	return artifact, path, nil
}
