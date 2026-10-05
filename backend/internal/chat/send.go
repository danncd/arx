package chat

import (
	agent "arx/internal/agent"
	contextwindow "arx/internal/agent/context"
	inference "arx/internal/inference"
	attachment "arx/internal/media/attachments"
	model "arx/internal/models"
	session "arx/internal/sessions"
	settings "arx/internal/settings"
	context "context"
	errors "errors"
	strings "strings"
	time "time"
)

func (s *Service) Send(ctx context.Context, input Send) (Run, error) {
	input.Text = strings.TrimSpace(input.Text)
	if input.ID == "" || len(input.ID) > 128 || len(input.Conversation) > 128 || (input.Text == "" && len(input.Images) == 0) || len(input.Text) > 128<<10 {
		return Run{}, errors.New("Enter a message of up to 128 KB")
	}
	s.runMutex.Lock()
	defer s.runMutex.Unlock()
	if s.closed {
		return Run{}, errors.New("Backend is stopping")
	}
	if s.chat.Run.ID == input.ID {
		return s.chat.Run, nil
	}
	if s.runCancel != nil {
		return Run{}, errors.New("Stop the current reply before sending another message")
	}
	saved := s.Preferences.Values()
	prepared, err := s.prepareSend(input, saved)
	if err != nil {
		return Run{}, err
	}
	if prepared.duplicate != nil {
		return *prepared.duplicate, nil
	}
	var release func()
	if inference.IsLocal(saved.Run) {
		manager, err := s.LocalModels()
		if err != nil {
			return Run{}, err
		}
		release, err = manager.Hold(saved.Run.Model)
		if err != nil {
			return Run{}, err
		}
		defer func() {
			if release != nil {
				release()
			}
		}()
	}
	if input.Conversation == "" {
		input.Conversation = agent.ID()
		policy := saved.Permissions
		if input.Permissions != nil {
			policy = *input.Permissions
		}
		if _, err := s.Preferences.SaveChatPermissions(input.Conversation, policy); err != nil {
			return Run{}, err
		}
	}
	compacted, err := s.Sessions.Compaction(input.Conversation)
	if err != nil {
		return Run{}, err
	}
	user := session.TranscriptChunk{
		Images:       input.Images,
		Runtime:      &session.RuntimeContext{Directory: saved.Directory, Date: time.Now().Format("2006-01-02")},
		Conversation: input.Conversation, ID: input.ID, Role: "user", Text: input.Text,
		Status: "done", At: time.Now().UTC().Format(time.RFC3339Nano),
	}
	newConversation := len(prepared.entries) == 0
	if newConversation {
		title := ""
		user.Title = &title
	}
	if err := s.Record(user); err != nil {
		return Run{}, err
	}
	next, err := agent.History([]session.TranscriptChunk{user})
	if err != nil {
		return Run{}, err
	}
	prepared.history = append(prepared.history, next...)
	run := Run{ID: input.ID, Conversation: input.Conversation, State: Running}
	s.chat = ChatEvent{
		Revision: s.chat.Revision, Run: run, Message: &user,
		Conversation: &session.Conversation{ID: input.Conversation, Title: s.Sessions.Conversation(input.Conversation).Title, Updated: user.At, Status: "running"},
	}
	s.startReply(ctx, user, agent.Input{
		Conversation: input.Conversation, Settings: saved, History: prepared.history,
		Model: prepared.model, Compaction: compacted, Usage: contextwindow.Latest(prepared.entries),
	}, newConversation, release)
	release = nil
	return run, nil
}

type preparedSend struct {
	model     model.Info
	entries   []session.TranscriptChunk
	history   []inference.Message
	duplicate *Run
}

func (s *Service) prepareSend(input Send, saved settings.Values) (preparedSend, error) {
	if err := s.Inference.Validate(saved.Run); err != nil {
		return preparedSend{}, err
	}
	info, err := s.Inference.Model(saved.Run)
	if err != nil {
		return preparedSend{}, err
	}
	if inference.IsLocal(saved.Run) {
		manager, err := s.LocalModels()
		if err != nil {
			return preparedSend{}, err
		}
		state := manager.Snapshot()
		if state.Runtime.State != "ready" || state.Runtime.Model != saved.Run.Model {
			return preparedSend{}, errors.New("Load the selected local model before sending")
		}
		if info.Tools == nil || !*info.Tools {
			return preparedSend{}, errors.New("This model does not support Arx tools. Choose another model")
		}
	}
	if _, err := contextwindow.New(info); err != nil {
		return preparedSend{}, err
	}
	if len(input.Images) > attachment.MaxPerMessage {
		return preparedSend{}, errors.New("Attach up to 4 images per message")
	}
	if len(input.Images) > 0 && !info.Vision {
		return preparedSend{}, errors.New("Choose a vision-capable model to use images")
	}
	for _, image := range input.Images {
		if len(image.Name) > 255 {
			return preparedSend{}, errors.New("Image name is too long")
		}
		if _, err := s.Images.Read(image.ID); err != nil {
			return preparedSend{}, err
		}
	}
	entries := s.Sessions.Entries(input.Conversation)
	if input.Conversation != "" && len(entries) == 0 {
		return preparedSend{}, errors.New("Session was not found")
	}
	for _, entry := range entries {
		if entry.ID == input.ID {
			duplicate := Run{ID: input.ID, Conversation: input.Conversation, State: Idle}
			return preparedSend{model: info, entries: entries, duplicate: &duplicate}, nil
		}
	}
	history, err := agent.History(entries)
	if err != nil {
		return preparedSend{}, err
	}
	if !info.Vision {
		for _, message := range history {
			if len(message.Images) > 0 {
				return preparedSend{}, errors.New("This conversation contains images. Select a vision-capable model to continue")
			}
		}
	}
	return preparedSend{model: info, entries: entries, history: history}, nil
}
