package catalog

import (
	"arx/internal/models/local/hardware"
	"arx/internal/models/local/library"
	download "arx/internal/platform/download"
	"context"
	"errors"
	"fmt"
	"net/url"
	"path"
	"regexp"
	"sort"
	"strings"
)

type Variant struct {
	Fit          hardware.Fit    `json:"fit"`
	ID           string          `json:"id"`
	Name         string          `json:"name"`
	Quantization string          `json:"quantization"`
	Size         int64           `json:"size"`
	Files        []download.File `json:"files"`
	Projector    string          `json:"projector,omitempty"`
}
type Repository struct {
	ID       string    `json:"id"`
	Revision string    `json:"revision"`
	Gated    bool      `json:"gated"`
	License  string    `json:"license,omitempty"`
	Variants []Variant `json:"variants"`
}

var splitPattern = regexp.MustCompile(`(?i)-([0-9]{5})-of-([0-9]{5})\.gguf$`)
var quantPattern = regexp.MustCompile(`(?i)(?:IQ[1-4]_[A-Z0-9_]+|Q[2-8]_[A-Z0-9_]+|BF16|F16|F32)`)
var commitPattern = regexp.MustCompile(`^[a-f0-9]{40}$`)

func (c *Client) Repository(ctx context.Context, id string) (Repository, error) {
	result := Repository{ID: id, Variants: []Variant{}}
	if !repositoryPattern.MatchString(id) {
		return result, errors.New("Invalid model repository")
	}
	var metadata struct {
		SHA   string `json:"sha"`
		Task  string `json:"pipeline_tag"`
		Gated any    `json:"gated"`
		Card  struct {
			License string `json:"license"`
		} `json:"cardData"`
	}
	if err := c.get(ctx, "/api/models/"+id, &metadata); err != nil {
		return result, err
	}
	if !chatTask(metadata.Task, id) {
		return result, errors.New("This repository is not a chat model")
	}
	result.Revision = metadata.SHA
	result.License = metadata.Card.License
	result.Gated = metadata.Gated != nil && metadata.Gated != false
	if result.Gated {
		return result, nil
	}
	if !commitPattern.MatchString(result.Revision) {
		return result, errors.New("Model revision is unavailable")
	}
	type treeItem struct {
		Type string `json:"type"`
		Path string `json:"path"`
		Size int64  `json:"size"`
		LFS  *struct {
			OID  string `json:"oid"`
			Size int64  `json:"size"`
		} `json:"lfs"`
	}
	var tree []treeItem
	next := "/api/models/" + id + "/tree/" + result.Revision + "?recursive=true&limit=1000"
	for page := 0; next != ""; page++ {
		if page >= 20 {
			return result, errors.New("This repository contains too many files")
		}
		var items []treeItem
		var err error
		next, err = c.getPage(ctx, next, &items)
		if err != nil {
			return result, err
		}
		tree = append(tree, items...)
	}
	groups := map[string][]download.File{}
	projectors := []download.File{}
	for _, item := range tree {
		lower := strings.ToLower(item.Path)
		if item.Type != "file" || !strings.HasSuffix(lower, ".gguf") || strings.Contains(lower, "draft") || strings.Contains(lower, "dspark") {
			continue
		}
		if path.Clean(item.Path) != item.Path || strings.HasPrefix(item.Path, "/") || strings.Contains(item.Path, "..") || strings.Contains(item.Path, "\\") {
			continue
		}
		escaped := strings.Split(item.Path, "/")
		for i := range escaped {
			escaped[i] = url.PathEscape(escaped[i])
		}
		file := download.File{Name: item.Path, Size: item.Size, URL: c.BaseURL + "/" + id + "/resolve/" + result.Revision + "/" + strings.Join(escaped, "/")}
		if item.LFS != nil {
			file.SHA256 = strings.TrimPrefix(item.LFS.OID, "sha256:")
			file.Size = item.LFS.Size
		}
		if strings.Contains(lower, "mmproj") {
			projectors = append(projectors, file)
			continue
		}
		if library.ValidateChatFile(item.Path) != nil {
			continue
		}
		group := splitPattern.ReplaceAllString(item.Path, ".gguf")
		groups[group] = append(groups[group], file)
	}
	for name, files := range groups {
		sort.Slice(files, func(i, j int) bool { return files[i].Name < files[j].Name })
		if match := splitPattern.FindStringSubmatch(files[0].Name); match != nil {
			var total int
			fmt.Sscanf(match[2], "%d", &total)
			if total != len(files) {
				continue
			}
			complete := true
			for i, file := range files {
				m := splitPattern.FindStringSubmatch(file.Name)
				var part int
				if m == nil {
					complete = false
					break
				}
				fmt.Sscanf(m[1], "%d", &part)
				if part != i+1 || m[2] != match[2] {
					complete = false
					break
				}
			}
			if !complete {
				continue
			}
		}
		variant := Variant{ID: name, Name: path.Base(name), Quantization: strings.ToUpper(quantPattern.FindString(name)), Files: files}
		if projector := chooseProjector(projectors); projector != nil {
			variant.Projector = projector.Name
			variant.Files = append(variant.Files, *projector)
		}
		for _, file := range variant.Files {
			variant.Size += file.Size
		}
		if variant.Size > 0 {
			result.Variants = append(result.Variants, variant)
		}
	}
	sort.Slice(result.Variants, func(i, j int) bool { return result.Variants[i].Size < result.Variants[j].Size })
	return result, nil
}
