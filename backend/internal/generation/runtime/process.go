package engines

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

type Request struct {
	Runtime   string   `json:"runtime,omitempty"`
	Operation string   `json:"operation"`
	ModelPath string   `json:"model_path,omitempty"`
	Prompt    string   `json:"prompt,omitempty"`
	Source    string   `json:"source,omitempty"`
	Sources   []string `json:"sources,omitempty"`
	Audio     string   `json:"audio,omitempty"`
	Output    string   `json:"output"`
	Width     int      `json:"width,omitempty"`
	Height    int      `json:"height,omitempty"`
	Steps     int      `json:"steps,omitempty"`
	Frames    int      `json:"frames,omitempty"`
	Seed      int64    `json:"seed"`
	Voice     string   `json:"voice,omitempty"`
	Speed     float64  `json:"speed,omitempty"`
}

type Event struct {
	Event    string  `json:"event"`
	Progress float64 `json:"progress"`
	Detail   string  `json:"detail"`
	MIME     string  `json:"mime"`
	Width    int     `json:"width"`
	Height   int     `json:"height"`
	Duration float64 `json:"duration"`
}

func Run(ctx context.Context, python, worker, logPath string, request Request, progress func(float64, string)) (outputEvent Event, failureErr error) {
	data, err := json.Marshal(request)
	if err != nil {
		return Event{}, err
	}
	if err := os.MkdirAll(filepath.Dir(logPath), 0700); err != nil {
		return Event{}, err
	}
	log, err := os.OpenFile(logPath, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0600)
	if err != nil {
		return Event{}, err
	}
	defer log.Close()
	command, cleanup, err := supervisedCommand(ctx, filepath.Dir(logPath), python, "-u", worker)
	if err != nil {
		return Event{}, err
	}
	defer func() {
		if err := cleanup(); err != nil {
			failureErr = errors.Join(failureErr, fmt.Errorf("Generation worker cleanup: %w", err))
		}
	}()
	command.Stdin = bytes.NewReader(data)
	command.Stderr = log
	output, err := command.StdoutPipe()
	if err != nil {
		return Event{}, err
	}
	if err := command.Start(); err != nil {
		return Event{}, err
	}
	scanner := bufio.NewScanner(output)
	scanner.Buffer(make([]byte, 4096), 1<<20)
	var result Event
	var failure error
	for scanner.Scan() {
		var event Event
		if json.Unmarshal(scanner.Bytes(), &event) != nil {
			log.Write(append(append([]byte{}, scanner.Bytes()...), '\n'))
			continue
		}
		switch event.Event {
		case "progress":
			progress(event.Progress, event.Detail)
		case "complete":
			result = event
		case "error":
			failure = errors.New(event.Detail)
		}
	}
	scanErr := scanner.Err()
	if scanErr != nil {
		command.Cancel()
	}
	waitErr := command.Wait()
	if ctx.Err() != nil {
		return Event{}, ctx.Err()
	}
	if failure != nil {
		return Event{}, failure
	}
	if waitErr != nil {
		return Event{}, fmt.Errorf("Generation worker stopped (%v). Check the generation log", waitErr)
	}
	if scanErr != nil {
		return Event{}, fmt.Errorf("Could not read generation output: %w", scanErr)
	}
	if result.Event != "complete" {
		return Event{}, errors.New("Generation stopped before producing a result. Check the generation log")
	}
	return result, nil
}
