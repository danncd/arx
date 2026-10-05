package generation

import (
	"arx/internal/generation/jobs"
	models "arx/internal/generation/models"
	engines "arx/internal/generation/runtime"
	"arx/internal/media/artifacts"
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
)

type Service struct {
	Directory string
	Assets    string
	Models    *models.Library
	Jobs      *jobs.Manager
	Artifacts artifacts.Store
	Acquire   func(context.Context) (func() error, error)
}

func Open(directory, assets string) (*Service, error) {
	if err := engines.Recover(filepath.Join(directory, "generation")); err != nil {
		return nil, err
	}
	library, err := models.Open(filepath.Join(directory, "generation", "models"), assets)
	if err != nil {
		return nil, err
	}
	manager, err := jobs.Open(filepath.Join(directory, "generation", "jobs"))
	if err != nil {
		return nil, err
	}
	return &Service{Directory: directory, Assets: assets, Models: library, Jobs: manager, Artifacts: artifacts.Store{Directory: directory}}, nil
}

func (s *Service) Run(ctx context.Context, conversation, call string, request Request) (artifacts.Artifact, error) {
	if err := request.Validate(); err != nil {
		return artifacts.Artifact{}, err
	}
	specification := engines.Request{Operation: request.Operation, Prompt: request.Prompt, Width: request.Width, Height: request.Height, Steps: 4, Frames: request.Frames, Seed: request.Seed, Voice: request.Voice, Speed: request.Speed}
	if request.Operation == "video" {
		specification.Steps = 30
	}
	var model models.Model
	var err error
	if request.Operation != "combine" {
		model, specification.ModelPath, err = s.Models.Resolve(request.Operation)
		if err != nil {
			return artifacts.Artifact{}, err
		}
		if err := configureModel(&request, &specification, model); err != nil {
			return artifacts.Artifact{}, err
		}
	}
	if request.Operation == "combine" {
		source, path, err := s.Artifacts.Resolve(strings.TrimPrefix(request.Source, "arx-media:"))
		if err != nil {
			return artifacts.Artifact{}, err
		}
		if source.MIME != "video/mp4" {
			return artifacts.Artifact{}, errors.New("Choose a video output")
		}
		specification.Source = path
	}
	if request.Operation == "combine" {
		audio, path, err := s.Artifacts.Resolve(strings.TrimPrefix(request.Audio, "arx-media:"))
		if err != nil {
			return artifacts.Artifact{}, err
		}
		if audio.MIME != "audio/wav" {
			return artifacts.Artifact{}, errors.New("Choose a speech output")
		}
		specification.Audio = path
	}
	job, err := s.Jobs.Run(ctx, jobs.Job{Conversation: conversation, ToolCall: call, Operation: request.Operation, Model: model.ID}, func(ctx context.Context, progress func(float64, string)) (output artifacts.Artifact, failure error) {
		if s.Acquire != nil {
			progress(0, "Preparing local memory")
			release, err := s.Acquire(ctx)
			if err != nil {
				return output, err
			}
			defer func() { failure = errors.Join(failure, release()) }()
		}
		python, err := engines.Install(ctx, filepath.Join(s.Directory, "generation", "engine"), s.Assets, request.Operation, model.Runtime, progress)
		if err != nil {
			return output, err
		}
		stage, err := os.MkdirTemp(filepath.Join(s.Directory, "generation"), ".output-")
		if err != nil {
			return output, err
		}
		defer os.RemoveAll(stage)
		if request.Operation == "image" || request.Operation == "video" {
			paths, err := s.imageSources(request, stage)
			if err != nil {
				return output, err
			}
			if len(paths) > 0 {
				specification.Source = paths[0]
			}
			if model.Runtime == "flux2" {
				specification.Sources = paths
			}
		}
		extension := map[string]string{"image": ".png", "speech": ".wav", "video": ".mp4", "combine": ".mp4"}[request.Operation]
		specification.Output = filepath.Join(stage, "output"+extension)
		result, err := engines.Run(ctx, python, filepath.Join(s.Assets, "worker.py"), filepath.Join(s.Directory, "generation", "runtime.log"), specification, progress)
		if err != nil {
			return output, err
		}
		if err := ctx.Err(); err != nil {
			return output, err
		}
		return s.Artifacts.Save(specification.Output, artifacts.Artifact{Name: "generation" + extension, MIME: result.MIME, Width: result.Width, Height: result.Height, Duration: result.Duration})
	})
	if job.Output != nil {
		return *job.Output, err
	}
	return artifacts.Artifact{}, err
}

func (s *Service) Close() { s.Models.Pause(); s.Jobs.Close() }
