package app

import (
	chat "arx/internal/chat"
	generation "arx/internal/generation"
	deepseek "arx/internal/inference/deepseek"
	mcp "arx/internal/integrations/mcp"
	skills "arx/internal/integrations/skills"
	attachment "arx/internal/media/attachments"
	localruntime "arx/internal/models/local"
	network "arx/internal/models/network"
	permission "arx/internal/permissions"
	credentials "arx/internal/platform/keychain"
	storage "arx/internal/sessions/storage"
	settings "arx/internal/settings"
	toolservice "arx/internal/tools"
	runner "arx/internal/tools/runner"
	sync "sync"
)

type Status struct {
	Version   string
	Execution State
}

type Actions = toolservice.Service

type Service struct {
	*Actions
	mcp                   *mcp.Manager
	skills                *skills.Manager
	networkMutex          sync.Mutex
	network               *network.Manager
	generationMutex       sync.Mutex
	generationNotifyMutex sync.Mutex
	generation            *generation.Service
	generationNotify      func(GenerationEvent)
	directory             string
	localMutex            sync.Mutex
	local                 *localruntime.Manager
	localNotify           func(localruntime.State)
	images                attachment.Store
	*chat.Service
	permissions *permission.Manager
	tools       *runner.Runner
	deepseek    *deepseek.DeepSeek
	sessions    *storage.Store
	preferences *settings.Store
}

func (s *Service) Status() Status {
	return Status{Version: "0.1.0", Execution: s.ChatState().Run.State}
}

func Open(directory string) (*Service, error) {
	preferences, err := settings.Open(directory)
	if err != nil {
		return nil, err
	}
	sessions, err := storage.Open(directory)
	if err != nil {
		return nil, err
	}
	keys, err := credentials.New(directory)
	if err != nil {
		sessions.Close()
		return nil, err
	}
	permissions := &permission.Manager{}
	images := attachment.Store{Directory: directory}
	client := deepseek.NewClient()
	client.ReadImage = images.Read
	connections, err := mcp.Open(directory)
	if err != nil {
		sessions.Close()
		return nil, err
	}
	workflows, err := skills.Open(directory)
	if err != nil {
		connections.Close()
		sessions.Close()
		return nil, err
	}
	tools := runner.New(permissions, &images)
	tools.Integrations(connections, workflows)
	app := &Service{mcp: connections, skills: workflows, directory: directory, images: images, Service: chat.New(), sessions: sessions, preferences: preferences, deepseek: deepseek.NewConnection(keys, client), permissions: permissions, tools: tools}
	app.Service.Preferences, app.Service.Sessions, app.Service.Images = preferences, sessions, images
	app.Service.LocalModels = app.LocalModels
	app.Service.Inference = app.newRouter()
	app.Service.GenerationGuidance, app.Service.IntegrationGuidance = app.generationGuidance, app.integrationGuidance
	app.Actions = &toolservice.Service{Runner: tools, Sessions: sessions, Preferences: preferences, Permissions: permissions}
	tools.Generation = app.Generation
	app.Service.Tool = app.Actions.Run
	return app, nil
}

func (s *Service) Close() error {
	s.Service.Close()
	s.tools.Stop()
	s.mcp.Close()
	s.generationMutex.Lock()
	generation := s.generation
	s.generationMutex.Unlock()
	if generation != nil {
		generation.Close()
	}
	s.localMutex.Lock()
	local := s.local
	s.localMutex.Unlock()
	if local != nil {
		local.Close()
	}
	return s.sessions.Close()
}

type State = chat.State
type ChatEvent = chat.ChatEvent
type Send = chat.Send
type Run = chat.Run
type ContextReport = chat.ContextReport

const (
	Idle     = chat.Idle
	Running  = chat.Running
	Stopping = chat.Stopping
)
