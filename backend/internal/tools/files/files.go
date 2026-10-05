package files

import (
	"arx/internal/media/artifacts"
	attachment "arx/internal/media/attachments"
	permission "arx/internal/permissions"
	environment "arx/internal/platform/paths"
	tool "arx/internal/tools"
	"context"
	"encoding/json"
	"errors"
	"path/filepath"
	"strings"
)

type arguments struct {
	Operation  string  `json:"operation"`
	Path       string  `json:"path"`
	Offset     int     `json:"offset"`
	Limit      int     `json:"limit"`
	Pattern    string  `json:"pattern"`
	Content    *string `json:"content"`
	OldText    *string `json:"old_text"`
	NewText    *string `json:"new_text"`
	ReplaceAll bool    `json:"replace_all"`
}

func Prepare(directory string, policy permission.Policy, raw json.RawMessage, images *attachment.Store) (tool.Prepared, error) {
	var args arguments
	if err := tool.Decode(raw, &args); err != nil {
		return tool.Prepared{}, err
	}
	if args.Operation == "image" && strings.HasPrefix(args.Path, "arx-media:") && images != nil {
		store := artifacts.Store{Directory: images.Directory}
		artifact, path, err := store.Resolve(strings.TrimPrefix(args.Path, "arx-media:"))
		if err != nil || artifact.MIME != "image/png" {
			return tool.Prepared{}, errors.New("Generated image was not found")
		}
		return tool.Prepared{Action: permission.Action{Tool: "files", Operation: "image", Path: path}, Run: func(ctx context.Context) (tool.Result, error) {
			return readImage(ctx, path)
		}}, nil
	}
	if args.Operation == "image" && strings.HasPrefix(args.Path, "arx-image:") {
		id := strings.TrimPrefix(args.Path, "arx-image:")
		return tool.Prepared{Action: permission.Action{Tool: "files", Operation: "image", Path: args.Path}, Run: func(ctx context.Context) (tool.Result, error) {
			return reopenImage(ctx, images, id)
		}}, nil
	}
	path, err := environment.Resolve(directory, args.Path)
	if err != nil {
		return tool.Prepared{}, err
	}
	if args.Offset < 0 || args.Limit < 0 || args.Limit > 500 {
		return tool.Prepared{}, errors.New("Invalid line or result limit")
	}
	if args.Offset == 0 {
		args.Offset = 1
	}
	if args.Limit == 0 {
		args.Limit = 200
	}
	action := permission.Action{Tool: "files", Operation: args.Operation, Path: path}
	switch args.Operation {
	case "image":
		return tool.Prepared{Action: action, Run: func(ctx context.Context) (tool.Result, error) { return readImage(ctx, path) }}, nil
	case "read":
		return tool.Prepared{Action: action, Run: func(ctx context.Context) (tool.Result, error) { return read(ctx, path, args.Offset, args.Limit) }}, nil
	case "list":
		return tool.Prepared{Action: action, Run: func(ctx context.Context) (tool.Result, error) { return list(ctx, path, args.Offset, args.Limit) }}, nil
	case "search":
		return tool.Prepared{Action: action, Run: func(ctx context.Context) (tool.Result, error) { return search(ctx, path, args.Pattern, args.Limit) }}, nil
	case "write", "edit", "mkdir":
		return prepareChange(filepath.Clean(path), policy, args)
	default:
		return tool.Prepared{}, errors.New("Unknown files operation")
	}
}
