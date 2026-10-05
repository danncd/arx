package chat

import (
	inference "arx/internal/inference"
	attachments "arx/internal/media/attachments"
	local "arx/internal/models/local"
	permissions "arx/internal/permissions"
	sessions "arx/internal/sessions"
	storage "arx/internal/sessions/storage"
	settings "arx/internal/settings"
	tools "arx/internal/tools"
	context "context"
	errors "errors"
	sync "sync"
)

type Service struct {
	Preferences         *settings.Store
	Sessions            *storage.Store
	Images              attachments.Store
	Inference           *inference.Router
	LocalModels         func() (*local.Manager, error)
	GenerationGuidance  func() string
	IntegrationGuidance func() string
	Tool                func(context.Context, string, string, tools.Call) tools.Result
	runMutex            sync.Mutex
	runCancel           context.CancelFunc
	runDone             chan struct{}
	chat                ChatEvent
	chatNotify          func(ChatEvent)
	closed              bool
}

func New() *Service                                            { return &Service{chat: ChatEvent{Run: Run{State: Idle}}} }
func (s *Service) Record(entry sessions.TranscriptChunk) error { return s.Sessions.Append(entry) }
func (s *Service) WithState(change func(bool, string, bool) error) error {
	s.runMutex.Lock()
	defer s.runMutex.Unlock()
	return change(s.runCancel != nil, s.chat.Run.Conversation, s.closed)
}
func (s *Service) Closed() bool {
	s.runMutex.Lock()
	defer s.runMutex.Unlock()
	return s.closed
}
func (s *Service) ConfigureChatPermissions(conversation string, policy permissions.Policy) (settings.Values, error) {
	var saved settings.Values
	err := s.WithState(func(active bool, current string, closed bool) error {
		if len(s.Sessions.Entries(conversation)) == 0 {
			return errors.New("Session was not found")
		}
		if active && current == conversation {
			return errors.New("Stop the current reply before changing its permissions")
		}
		var err error
		saved, err = s.Preferences.SaveChatPermissions(conversation, policy)
		return err
	})
	return saved, err
}

type Run struct {
	ID           string `json:"id"`
	Conversation string `json:"conversation"`
	State        State  `json:"state"`
}
type Recovery struct {
	Kind    string `json:"kind"`
	Attempt int    `json:"attempt"`
	Limit   int    `json:"limit"`
}
type ChatEvent struct {
	Recovery     *Recovery                 `json:"recovery,omitempty"`
	Compacting   bool                      `json:"compacting,omitempty"`
	Revision     uint64                    `json:"revision"`
	Run          Run                       `json:"run"`
	Message      *sessions.TranscriptChunk `json:"message,omitempty"`
	Conversation *sessions.Conversation    `json:"conversation,omitempty"`
	Error        string                    `json:"error,omitempty"`
}
type Send struct {
	Images       []attachments.Image `json:"images,omitempty"`
	Permissions  *permissions.Policy `json:"permissions,omitempty"`
	ID           string              `json:"id"`
	Conversation string              `json:"conversation"`
	Text         string              `json:"text"`
}

type State string

const (
	Idle     State = "idle"
	Running  State = "running"
	Stopping State = "stopping"
)

func (s *Service) ChatState() ChatEvent {
	s.runMutex.Lock()
	defer s.runMutex.Unlock()
	return s.chat
}
func (s *Service) SubscribeChat(notify func(ChatEvent)) {
	s.runMutex.Lock()
	defer s.runMutex.Unlock()
	s.chatNotify = notify
}
func (s *Service) publishLocked() {
	s.chat.Revision++
	if s.chatNotify != nil {
		s.chatNotify(s.chat)
	}
}
func (s *Service) chatMessage(message sessions.TranscriptChunk) {
	s.runMutex.Lock()
	defer s.runMutex.Unlock()
	s.chat.Message = &message
	summary := *s.chat.Conversation
	summary.Updated = message.At
	s.chat.Conversation = &summary
	s.publishLocked()
}
func (s *Service) chatRecovering(kind string, attempt int) {
	s.runMutex.Lock()
	defer s.runMutex.Unlock()
	s.chat.Recovery = &Recovery{Kind: kind, Attempt: attempt, Limit: 3}
	s.publishLocked()
}

func (s *Service) chatCompacting(active bool) {
	s.runMutex.Lock()
	defer s.runMutex.Unlock()
	s.chat.Compacting = active
	s.publishLocked()
}
func (s *Service) StopReply(id string) bool {
	s.runMutex.Lock()
	defer s.runMutex.Unlock()
	if s.runCancel == nil || s.chat.Run.ID != id {
		return false
	}
	s.chat.Run.State = Stopping
	s.runCancel()
	s.publishLocked()
	return true
}
func (s *Service) Close() {
	s.runMutex.Lock()
	s.closed = true
	if s.runCancel != nil {
		s.runCancel()
	}
	done := s.runDone
	s.runMutex.Unlock()
	if done != nil {
		<-done
	}
}
