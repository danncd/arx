package engine

import (
	model "arx/internal/models"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"
)

type Server struct {
	recordPath string
	keyPath    string
	mutex      sync.Mutex
	command    *exec.Cmd
	done       chan struct{}
	url        string
	key        string
	info       model.Info
}

func (s *Server) Start(ctx context.Context, binary, path, projector string, info model.Info, logs string) (model.Info, error) {
	s.Stop()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return info, err
	}
	port := listener.Addr().(*net.TCPAddr).Port
	listener.Close()
	var secret [32]byte
	if _, err := rand.Read(secret[:]); err != nil {
		return info, err
	}
	key := hex.EncodeToString(secret[:])
	keyFile, err := os.CreateTemp(filepath.Dir(logs), ".runtime-key-")
	if err != nil {
		return info, err
	}
	keyPath := keyFile.Name()

	if _, err = keyFile.WriteString(key); err != nil {
		keyFile.Close()
		os.Remove(keyPath)
		return info, err
	}
	keyFile.Close()
	args := []string{"--model", path, "--alias", "arx-local", "--host", "127.0.0.1", "--port", fmt.Sprint(port), "--ctx-size", fmt.Sprint(info.ContextWindow), "--parallel", "1", "--jinja", "--reasoning-format", "deepseek", "--api-key-file", keyPath, "--no-ui"}
	if projector != "" {
		args = append(args, "--mmproj", projector)
	}
	output, err := os.OpenFile(logs, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0600)
	if err != nil {
		os.Remove(keyPath)
		return info, err
	}
	command := exec.Command(binary, args...)
	command.Stdout = output
	command.Stderr = output
	command.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	if err := command.Start(); err != nil {
		os.Remove(keyPath)
		output.Close()
		return info, err
	}
	recordPath := filepath.Join(filepath.Dir(logs), "runtime-process.json")
	record, _ := json.Marshal(processRecord{PID: command.Process.Pid, Binary: binary, KeyFile: keyPath})
	if err := os.WriteFile(recordPath, record, 0600); err != nil {
		command.Process.Kill()
		command.Wait()
		output.Close()
		os.Remove(keyPath)
		return info, err
	}
	done := make(chan struct{})
	s.mutex.Lock()
	s.recordPath = recordPath
	s.keyPath = keyPath
	s.command = command
	s.done = done
	s.key = key
	s.url = fmt.Sprintf("http://127.0.0.1:%d", port)
	s.info = model.Info{}
	s.mutex.Unlock()
	go func() { command.Wait(); output.Close(); close(done) }()
	timer := time.NewTimer(3 * time.Minute)
	defer timer.Stop()
	ticker := time.NewTicker(200 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			s.Stop()
			return info, ctx.Err()
		case <-timer.C:
			s.Stop()
			return info, errors.New("Model loading timed out")
		case <-done:
			s.Stop()
			return info, loadFailure(logs)
		case <-ticker.C:
			props, err := s.properties(ctx)
			if err != nil {
				continue
			}
			if props.Settings.Context <= 0 {
				continue
			}
			info.ContextWindow = props.Settings.Context
			info.MaxOutputTokens = min(32768, info.ContextWindow/2)
			info.Vision = props.Modalities.Vision
			if supported, ok := props.Capabilities["supports_tools"]; ok {
				info.Tools = &supported
			}
			template := props.Template
			canDisable := strings.Contains(template, "enable_thinking")
			if canDisable || strings.Contains(template, "<think>") {
				info.Thinking = &model.Thinking{Efforts: []string{"default"}, DefaultEffort: "default", DefaultEnabled: true, CanDisable: canDisable, Source: "runtime"}
			}
			info.CapabilitySource = "runtime"
			s.mutex.Lock()
			s.info = info
			s.mutex.Unlock()
			return info, nil
		}
	}
}

type properties struct {
	Settings struct {
		Context int `json:"n_ctx"`
	} `json:"default_generation_settings"`
	Modalities struct {
		Vision bool `json:"vision"`
	} `json:"modalities"`
	Capabilities map[string]bool `json:"chat_template_caps"`
	Template     string          `json:"chat_template"`
}
