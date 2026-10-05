package runner

import (
	"arx/internal/generation"
	mcp "arx/internal/integrations/mcp"
	skills "arx/internal/integrations/skills"
	attachment "arx/internal/media/attachments"
	permission "arx/internal/permissions"
	tool "arx/internal/tools"
	"arx/internal/tools/bash"
	"arx/internal/tools/files"
	"arx/internal/tools/web"
	"context"
	"errors"
	"strings"
	"sync"
)

type Runner struct {
	Generation  func() (*generation.Service, error)
	mcp         *mcp.Manager
	skills      *skills.Manager
	images      *attachment.Store
	mutex       sync.Mutex
	active      string
	cancel      context.CancelFunc
	permissions *permission.Manager
	web         *web.Client
}

func New(permissions *permission.Manager, images *attachment.Store) *Runner {
	return &Runner{images: images, permissions: permissions, web: web.New()}
}

func (r *Runner) Integrations(m *mcp.Manager, s *skills.Manager) { r.mcp = m; r.skills = s }

func (r *Runner) Idle(change func() error) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	if r.active != "" {
		return errors.New("Stop the current action before changing permissions")
	}
	return change()
}

func (r *Runner) Run(ctx context.Context, call tool.Call, configuration func() (string, permission.Policy)) (tool.Result, error) {
	if strings.TrimSpace(call.ID) == "" || len(call.ID) > 128 {
		return tool.Result{}, errors.New("A tool call ID is required")
	}
	r.mutex.Lock()
	if r.active != "" {
		r.mutex.Unlock()
		return tool.Result{}, errors.New("Another action is running")
	}
	ctx, cancel := context.WithCancel(ctx)
	directory, policy := configuration()
	r.active, r.cancel = call.ID, cancel
	r.mutex.Unlock()
	defer func() {
		cancel()
		r.mutex.Lock()
		r.active = ""
		r.cancel = nil
		r.mutex.Unlock()
	}()
	var prepared tool.Prepared
	var err error
	switch call.Name {
	case "generate", "media":
		if r.Generation == nil {
			err = errors.New("Generation is unavailable")
		} else {
			prepared, err = tool.PrepareGeneration(ctx, call, r.Generation)
		}
	case "files":
		prepared, err = files.Prepare(directory, policy, call.Arguments, r.images)
	case "web":
		prepared, err = r.web.Prepare(call.Arguments)
	case "bash":
		prepared, err = bash.Prepare(directory, policy, call.Arguments)
	case "mcp":
		prepared, err = r.prepareMCP(ctx, call.Arguments)
	case "skills":
		prepared, err = r.prepareSkills(call.Arguments)
	default:
		err = errors.New("Unknown tool")
	}
	if err != nil {
		return tool.Result{}, err
	}
	if prepared.Close != nil {
		defer prepared.Close()
	}
	if prepared.RequireApproval && policy.Mode != permission.Full {
		policy.Mode = permission.Ask
	}
	if !prepared.Metadata {
		if err := r.permissions.Authorize(ctx, policy, prepared.Action); err != nil {
			if prepared.Denied != nil && ctx.Err() == nil {
				if saveErr := prepared.Denied(ctx); saveErr != nil {
					return tool.Result{}, errors.Join(err, saveErr)
				}
			}
			return tool.Result{}, err
		}
	}
	if err := ctx.Err(); err != nil {
		return tool.Result{}, err
	}
	result, err := prepared.Run(ctx)
	if err != nil || len(result.ImageData) == 0 {
		return result, err
	}
	if err := ctx.Err(); err != nil {
		return tool.Result{}, err
	}
	if r.images == nil {
		return tool.Result{}, errors.New("Image storage is unavailable")
	}
	saved, err := r.images.Save(result.ImageData, result.ImageName)
	if err != nil {
		return tool.Result{}, err
	}
	result.ImageData, result.ImageName = nil, ""
	result.Images = []attachment.Image{saved}
	return result, ctx.Err()
}

func (r *Runner) Cancel(id string) bool {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	if r.active != id || r.cancel == nil {
		return false
	}
	r.cancel()
	return true
}

func (r *Runner) Stop() {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	if r.cancel != nil {
		r.cancel()
	}
}

func (r *Runner) WebIcon(ctx context.Context, address string) (string, error) {
	return r.web.Icon(ctx, address)
}

func (r *Runner) WebImage(ctx context.Context, address string) (string, error) {
	return r.web.Image(ctx, address)
}
