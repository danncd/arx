package stdio

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"sync"

	service "arx/internal/app"
	localruntime "arx/internal/models/local"
	permission "arx/internal/permissions"
	protocol "arx/internal/transport/contract"
)

type Server struct {
	service *service.Service
	writer  *writer
}

func NewServer(app *service.Service, output io.Writer) *Server {
	server := &Server{service: app, writer: newWriter(output)}
	app.SubscribePermissions(func(request *permission.Request) {
		server.writer.send(protocol.Event{Event: "permission", Data: request})
	})
	app.SubscribeChat(func(event service.ChatEvent) { server.writer.send(protocol.Event{Event: "chat", Data: event}) })
	app.SubscribeLocal(func(state localruntime.State) { server.writer.send(protocol.Event{Event: "local", Data: state}) })
	app.SubscribeGeneration(func(event service.GenerationEvent) {
		server.writer.send(protocol.Event{Event: "generation", Data: event})
	})
	return server
}

func (s *Server) Serve(ctx context.Context, input io.Reader) error {
	ctx, cancel := context.WithCancel(ctx)
	var pending sync.WaitGroup
	defer pending.Wait()
	defer cancel()
	if err := s.writer.send(protocol.Event{Event: "ready", Data: protocol.Ready{Protocol: protocol.Version}}); err != nil {
		return err
	}
	scanner := bufio.NewScanner(input)
	scanner.Buffer(make([]byte, 4096), protocol.MaxMessageBytes+1)
	for scanner.Scan() {
		if ctx.Err() != nil {
			return nil
		}
		line := scanner.Bytes()
		if len(bytes.TrimSpace(line)) == 0 {
			continue
		}
		var request protocol.Request
		if err := json.Unmarshal(line, &request); err != nil || request.ID == "" || len(request.ID) > 128 {
			if err := s.writer.send(protocol.Response{ID: "", Error: &protocol.Error{Code: "invalid_request", Message: "Invalid request"}}); err != nil {
				return err
			}
			continue
		}
		descriptor, _ := protocol.Lookup(request.Method)
		if descriptor.Async {
			pending.Go(func() {
				response, _ := s.dispatch(ctx, request)
				if err := s.writer.send(response); err != nil {
					cancel()
				}
			})
			continue
		}
		response, shutdown := s.dispatch(ctx, request)
		if err := s.writer.send(response); err != nil {
			return err
		}
		if shutdown {
			return nil
		}
	}
	if ctx.Err() != nil {
		return nil
	}
	if err := scanner.Err(); err != nil {
		return fmt.Errorf("read desktop requests: %w", err)
	}
	return nil
}
