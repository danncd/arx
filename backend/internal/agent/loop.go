package agent

import (
	compactor "arx/internal/agent/compaction"
	contextwindow "arx/internal/agent/context"
	provider "arx/internal/inference"
	model "arx/internal/models"
	session "arx/internal/sessions"
	"arx/internal/settings"
	tool "arx/internal/tools"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
)

type Input struct {
	IntegrationGuidance string
	GenerationGuidance  string
	Usage               *session.ContextUsage
	Model               model.Info
	Compaction          session.Compaction
	Conversation        string
	Settings            settings.Values
	History             []provider.Message
}

type Loop struct {
	Guidance       func() string
	Recovering     func(string, int)
	SaveCompaction func(session.Compaction) error
	Compacting     func(bool)
	Complete       func(context.Context, provider.Request, func(provider.Delta) error) (provider.Response, error)
	Tool           func(context.Context, tool.Call) tool.Result
	Save           func(session.TranscriptChunk) error
	Emit           func(session.TranscriptChunk)
}

func ID() string {
	var value [16]byte
	if _, err := rand.Read(value[:]); err != nil {
		panic(err)
	}
	return hex.EncodeToString(value[:])
}

func (l Loop) Run(ctx context.Context, input Input) error {
	manager := &compactor.Compactor{
		Usage:       input.Usage,
		AutoRecover: input.Settings.AutoContinue,
		Recovery:    l.Recovering,
		Model:       input.Model,
		State:       input.Compaction,
		Complete: func(ctx context.Context, request provider.Request, emit func(provider.Delta) error) (provider.Response, error) {
			response, err := l.Complete(ctx, request, emit)
			if response.Usage != nil {
				if saveErr := l.Save(session.TranscriptChunk{Conversation: input.Conversation, ID: ID(), Role: "usage", Provider: input.Settings.Run.Provider, Usage: response.Usage, Status: "done"}); saveErr != nil {
					return response, saveErr
				}
			}
			return response, err
		},
		Save:   l.SaveCompaction,
		Active: l.Compacting,
	}
	request := Request(input.Settings, input.History, input.Model)
	AddGenerationGuidance(&request, input.GenerationGuidance)
	AddGenerationGuidance(&request, input.IntegrationGuidance)
	messages := request.Messages
	runSettings := input.Settings.Run
	emptyAnswerRetry := false
	maxRounds := 32
	if input.Settings.AutoContinue {
		maxRounds = 128
	}
	for round := 0; round < maxRounds; round++ {
		record := &recorder{message: session.TranscriptChunk{Conversation: input.Conversation, ID: ID(), Role: "assistant", Provider: input.Settings.Run.Provider, Model: input.Settings.Run.Model, Status: "running"}, save: l.Save, emit: l.Emit}
		if err := record.flush(); err != nil {
			return err
		}
		err := l.step(ctx, runSettings, &messages, record, manager)
		if err != nil {
			if errors.Is(err, provider.ErrOutputLimit) {
				record.message.Interruption = interruptedOutput(record.message.Tools)
				if ctx.Err() == nil && round < maxRounds-1 && manager.Recover("output") {
					record.message.Status = "done"
					if saveErr := record.flush(); saveErr != nil {
						return saveErr
					}
					retained, historyErr := History([]session.TranscriptChunk{record.message})
					if historyErr != nil {
						return historyErr
					}
					messages = append(messages, retained...)
					if input.Model.Thinking != nil && input.Model.Thinking.CanDisable {
						runSettings.Effort = "none"
					}
					continue
				}
			}
			record.message.Status, record.message.Failed, record.message.Reason = "failed", true, err.Error()
			if errors.Is(err, context.Canceled) {
				record.message.Status, record.message.Failed, record.message.Reason = "cancelled", false, "Reply stopped"
			}
			if saveErr := record.flush(); saveErr != nil {
				return fmt.Errorf("Could not save reply: %w", saveErr)
			}
			return err
		}
		if len(record.message.Tools) == 0 {
			if strings.TrimSpace(record.message.Text) == "" {
				if !emptyAnswerRetry && round < maxRounds-1 {
					emptyAnswerRetry = true
					if input.Model.Thinking != nil && input.Model.Thinking.CanDisable {
						runSettings.Effort = "none"
					}
					messages = append(messages, provider.Message{Role: "user", Content: "Please finish the previous request with a direct answer. If another tool is needed, call it."})
					continue
				}
				err = errors.New("The model stopped after thinking without giving an answer")
				record.message.Status, record.message.Failed, record.message.Reason = "failed", true, err.Error()
				if saveErr := record.flush(); saveErr != nil {
					return fmt.Errorf("Could not save reply: %w", saveErr)
				}
				return err
			}
			return nil
		}
		if (round+1)%32 == 0 {
			if input.Settings.AutoContinue && round < maxRounds-1 {
				manager.Continue()
				if l.Recovering != nil {
					l.Recovering("tool_limit", manager.Continuation)
				}
				continue
			}
			record.message.Status, record.message.Failed, record.message.Reason = "failed", true, "Tool limit reached. Send a message to continue."
			if input.Settings.AutoContinue {
				record.message.Reason = "Auto continue stopped after three continuations. Send a message to continue."
			}
			return record.flush()
		}
	}
	return nil
}

func (l Loop) step(ctx context.Context, settings settings.Run, messages *[]provider.Message, record *recorder, manager *compactor.Compactor) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	roundMessages := append([]provider.Message{}, (*messages)...)
	if l.Guidance != nil && len(roundMessages) > 0 {
		roundMessages[0].Content += l.Guidance()
	}
	if manager.Continuation > 0 && len(roundMessages) > 0 {
		roundMessages[0].Content += "\nContinue the current user task from the saved progress. Completed tool actions remain completed; do not repeat them. Finish when the requested outcome is achieved."
	}
	request, err := manager.Prepare(ctx, provider.Request{Model: settings.Model, Effort: settings.Effort, Messages: roundMessages, Tools: tool.Definitions()})
	if err != nil {
		return err
	}
	response, err := l.stream(ctx, request, record)
	forced := false
	for errors.Is(err, provider.ErrContextLength) && ctx.Err() == nil && record.message.Text == "" && record.message.Reasoning == "" && len(record.message.Tools) == 0 {
		if manager.AutoRecover {
			if !manager.Recover("context") {
				break
			}
		} else if forced {
			break
		}
		forced = true
		manager.Force = true
		request, err = manager.Prepare(ctx, provider.Request{Model: settings.Model, Effort: settings.Effort, Messages: roundMessages, Tools: tool.Definitions()})
		manager.Force = false
		if err == nil {
			response, err = l.stream(ctx, request, record)
		}
	}
	record.message.Usage = response.Usage
	if err != nil {
		return err
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	manager.Progress()
	if response.Usage != nil {
		manager.Usage = contextwindow.Measure(request, response.Usage.Input)
		record.message.Context = manager.Usage
	}
	if !strings.HasPrefix(response.Message.Content, record.message.Text) || !strings.HasPrefix(response.Message.Reasoning, record.message.Reasoning) {
		return errors.New("Provider reply did not match its stream")
	}
	record.message.Text, record.message.Reasoning = response.Message.Content, response.Message.Reasoning
	record.message.Round = true
	*messages = append(*messages, response.Message)
	record.completeTools(response.Message.Calls)
	if err := record.flush(); err != nil {
		return err
	}
	for index, call := range response.Message.Calls {
		if err := ctx.Err(); err != nil {
			return err
		}
		record.message.Tools[index].Status = "running"
		if err := record.flush(); err != nil {
			return err
		}
		result := l.Tool(ctx, tool.Call{ID: call.ID, Name: call.Function.Name, Arguments: json.RawMessage(call.Function.Arguments)})
		if len(result.Images) > 0 && !manager.Model.Vision {
			result = tool.Result{Text: "This model cannot inspect images. Choose a vision-capable model.", Failed: true}
		}
		serialized, err := json.Marshal(result)
		if err != nil {
			return err
		}
		record.message.Tools[index].Status = "done"
		record.message.Tools[index].Result = string(serialized)
		record.message.Tools[index].Failed = result.Failed
		record.message.Tools[index].Truncated = result.Truncated
		if err := record.flush(); err != nil {
			return err
		}
		*messages = append(*messages, provider.ToolMessage(call.ID, string(serialized)))
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	record.message.Status = "done"
	return record.flush()
}
