package tools

import (
	permission "arx/internal/permissions"
	"arx/internal/sessions/storage"
	"arx/internal/settings"
	"context"
	"errors"
)

type Executor interface {
	Run(context.Context, Call, func() (string, permission.Policy)) (Result, error)
	Cancel(string) bool
	Idle(func() error) error
	WebIcon(context.Context, string) (string, error)
	WebImage(context.Context, string) (string, error)
}
type Service struct {
	Runner      Executor
	Sessions    *storage.Store
	Preferences *settings.Store
	Permissions *permission.Manager
}

func (s *Service) RunTool(ctx context.Context, call Call) Result {
	return s.RunChatTool(ctx, "", call)
}

func (s *Service) RunChatTool(ctx context.Context, conversation string, call Call) Result {
	if conversation != "" && len(s.Sessions.Entries(conversation)) == 0 {
		return Result{Failed: true, Text: "Session was not found"}
	}
	return s.Run(ctx, conversation, s.Preferences.Values().Directory, call)
}

func (s *Service) Run(ctx context.Context, conversation, directory string, call Call) Result {
	call.Conversation = conversation
	result, err := s.Runner.Run(ctx, call, func() (string, permission.Policy) {
		policy := s.Preferences.Values().Permissions
		if conversation != "" {
			policy = s.Preferences.ChatPermissions(conversation)
		}
		return directory, policy
	})
	return toolResult(result, err)
}

func toolResult(result Result, err error) Result {
	if err != nil {
		result.Failed = true
		if result.Text != "" {
			result.Text += "\n"
		}
		result.Text += err.Error()
	}
	return result
}
func (s *Service) CancelTool(id string) bool {
	return s.Runner.Cancel(id)
}
func (s *Service) PendingPermission() *permission.Request { return s.Permissions.Pending() }
func (s *Service) RespondPermission(id string, allow bool) error {
	return s.Permissions.Respond(id, allow)
}
func (s *Service) SubscribePermissions(notify func(*permission.Request)) {
	s.Permissions.Subscribe(notify)
}
func (s *Service) ConfigurePermissions(policy permission.Policy) (settings.Values, error) {
	var saved settings.Values
	err := s.Runner.Idle(func() error {
		var err error
		saved, err = s.Preferences.SavePermissions(policy)
		return err
	})
	return saved, err
}

func (s *Service) WebIcon(ctx context.Context, conversation, address string) (string, error) {
	if err := s.RemoteResourcePolicy(conversation); err != nil {
		return "", err
	}
	return s.Runner.WebIcon(ctx, address)
}

func (s *Service) WebImage(ctx context.Context, conversation, address string) (string, error) {
	if err := s.RemoteResourcePolicy(conversation); err != nil {
		return "", err
	}
	return s.Runner.WebImage(ctx, address)
}

func (s *Service) RemoteResourcePolicy(conversation string) error {
	if conversation == "" || len(s.Sessions.Entries(conversation)) == 0 {
		return errors.New("Session was not found")
	}
	policy := s.Preferences.ChatPermissions(conversation)
	if policy.Mode != permission.Folders && policy.Mode != permission.Full {
		return errors.New("Approve remote images using the Load image action")
	}
	return nil
}
