package main

import (
	contextwindow "arx/internal/agent/context"
	"arx/internal/app"
	"arx/internal/chat"
	"arx/internal/generation/jobs"
	generation "arx/internal/generation/models"
	"arx/internal/inference/deepseek"
	"arx/internal/integrations"
	"arx/internal/integrations/mcp"
	"arx/internal/integrations/skills"
	"arx/internal/media/artifacts"
	"arx/internal/media/attachments"
	"arx/internal/models"
	"arx/internal/models/local"
	"arx/internal/models/local/catalog"
	"arx/internal/models/local/library"
	"arx/internal/models/network"
	"arx/internal/permissions"
	"arx/internal/sessions"
	"arx/internal/settings"
	"arx/internal/tools"
	"arx/internal/transport/contract"
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"regexp"
	"sort"
	"strings"
	"time"
)

type alias struct {
	Name string
	Type reflect.Type
}

func named[T any](name string) alias { return alias{name, reflect.TypeFor[T]()} }

var aliases = []alias{
	named[permissions.Mode]("PermissionMode"), named[permissions.Policy]("PermissionPolicy"), named[permissions.Request]("PermissionRequest"), named[permissions.Action]("PermissionAction"), named[tools.Result]("ToolResult"),
	named[attachments.Image]("ImageAttachment"), named[models.Info]("DiscoveredModel"), named[models.Thinking]("ThinkingCapabilities"), named[deepseek.Connection]("Connection"),
	named[settings.Run]("RunSettings"), named[settings.Values]("SavedSettings"), named[sessions.TranscriptChunk]("TranscriptChunk"), named[sessions.ToolRecord]("ToolRecord"), named[sessions.ChunkUsage]("ChunkUsage"), named[sessions.GenerationUsage]("GenerationUsage"), named[sessions.Conversation]("Conversation"), named[app.History]("HistoryPage"), named[app.Snapshot]("Snapshot"), named[chat.Run]("ChatRun"), named[chat.State]("ChatState"), named[chat.ChatEvent]("ChatEvent"), named[chat.ContextReport]("ContextReport"), named[contextwindow.Parts]("ContextParts"),
	named[local.State]("LocalState"), named[library.Entry]("LocalModel"), named[catalog.Model]("SearchModel"), named[catalog.SearchPage]("SearchPage"), named[catalog.Repository]("Repository"), named[catalog.Variant]("Variant"), named[network.Server]("NetworkServer"),
	named[generation.Model]("GenerationModel"), named[generation.Entry]("GenerationEntry"), named[generation.State]("GenerationLibrary"), named[jobs.Job]("GenerationJob"), named[artifacts.Artifact]("MediaArtifact"), named[contract.GenerationState]("GenerationState"), named[app.GenerationEvent]("GenerationEvent"),
	named[integrations.Activity]("IntegrationActivity"), named[integrations.Usage]("IntegrationUsage"), named[mcp.Configuration]("MCPConfiguration"), named[mcp.Tool]("MCPTool"), named[mcp.Server]("MCPServer"), named[skills.Skill]("Skill"), named[skills.Detail]("SkillDetail"), named[skills.Edit]("SkillEdit"),
}
var names = map[reflect.Type]string{}

func ts(t reflect.Type, defining bool) string {
	if !defining {
		if name, ok := names[t]; ok {
			return name
		}
	}
	if t == reflect.TypeFor[time.Time]() {
		return "string"
	}
	if t == reflect.TypeFor[json.RawMessage]() {
		return "unknown"
	}
	if t == reflect.TypeFor[permissions.Mode]() {
		return `"ask" | "folders" | "full"`
	}
	if t == reflect.TypeFor[chat.State]() {
		return `"idle" | "running" | "stopping"`
	}
	switch t.Kind() {
	case reflect.Bool:
		return "boolean"
	case reflect.String:
		return "string"
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64, reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Float32, reflect.Float64:
		return "number"
	case reflect.Interface:
		return "unknown"
	case reflect.Pointer:
		return ts(t.Elem(), false) + " | null"
	case reflect.Slice, reflect.Array:
		if t.Elem().Kind() == reflect.Uint8 {
			return "string"
		}
		return "Array<" + ts(t.Elem(), false) + ">"
	case reflect.Map:
		return "Record<string, " + ts(t.Elem(), false) + ">"
	case reflect.Struct:
		fields := []string{}
		for i := 0; i < t.NumField(); i++ {
			f := t.Field(i)
			if f.PkgPath != "" {
				continue
			}
			tag := f.Tag.Get("json")
			if tag == "-" {
				continue
			}
			if f.Anonymous && tag == "" {
				fields = append(fields, strings.TrimSuffix(strings.TrimPrefix(ts(f.Type, true), "{ "), " }"))
				continue
			}
			key, options, _ := strings.Cut(tag, ",")
			if key == "" {
				key = f.Name
			}
			optional := strings.Contains(options, "omitempty") || strings.HasSuffix(t.Name(), "Params") && f.Type.Kind() == reflect.Pointer
			value := ts(f.Type, false)
			if optional && f.Type.Kind() == reflect.Pointer {
				value = ts(f.Type.Elem(), false)
			}
			if t == reflect.TypeFor[sessions.TranscriptChunk]() && key == "tools" {
				value += " | null"
			}
			if t == reflect.TypeFor[sessions.ToolRecord]() && (key == "offset" || key == "reasoning_offset" || key == "added" || key == "removed" || key == "truncated") {
				optional = true
			}
			if key == "provider" && (t == reflect.TypeFor[settings.Run]() || t == reflect.TypeFor[models.Info]()) {
				value = `"deepseek" | "local" | "network"`
			}
			if key == "activation" && (t == reflect.TypeFor[skills.Skill]() || t == reflect.TypeFor[skills.Edit]()) {
				value = `"auto" | "explicit"`
			}
			if key == "policy" && t == reflect.TypeFor[mcp.Configuration]() {
				value = `"ask" | "changes"`
			}
			if key == "category" && t == reflect.TypeFor[generation.Model]() {
				value = "GenerationCategory"
			}
			if key == "mime" && t == reflect.TypeFor[artifacts.Artifact]() {
				value = `"image/png" | "audio/wav" | "video/mp4"`
			}
			if key == "gated" && t == reflect.TypeFor[catalog.Model]() {
				value = "boolean | string"
				optional = true
			}
			mark := ""
			if optional {
				mark = "?"
			}
			fields = append(fields, fmt.Sprintf("%q%s: %s;", key, mark, value))
		}
		if len(fields) == 0 {
			return "Record<string, never>"
		}
		return "{ " + strings.Join(fields, " ") + " }"
	default:
		panic("Unsupported wire type: " + t.String())
	}
}
func main() {
	check := flag.Bool("check", false, "")
	out := flag.String("out", "../desktop/contracts", "")
	flag.Parse()
	for _, a := range aliases {
		names[a.Type] = a.Name
	}
	sort.Slice(aliases, func(i, j int) bool { return aliases[i].Name < aliases[j].Name })
	var wire strings.Builder
	wire.WriteString("export type GenerationCategory = \"image\" | \"video\" | \"speech\";\nexport type ModelInfo = Pick<DiscoveredModel, \"id\" | \"name\">;\n")
	for _, a := range aliases {
		fmt.Fprintf(&wire, "export type %s = %s;\n", a.Name, ts(a.Type, true))
	}
	var requests strings.Builder
	requests.WriteString("import type * as Wire from \"./wire.generated\";\nexport type Requests = {\n")
	metadata := map[string]any{}
	for _, m := range contract.Methods {
		metadata[m.Name] = map[string]any{"renderer": m.Renderer, "async": m.Async, "timeout": m.Timeout, "group": m.Group, "error": m.Error}
		if !m.Renderer {
			continue
		}
		param := "undefined"
		if m.Params != reflect.TypeFor[contract.NoParams]() {
			param = ts(m.Params, false)
		}
		result := ts(m.Result, false)
		for _, alternative := range m.Alternatives {
			result += " | " + ts(alternative, false)
		}
		for _, a := range aliases {
			param = regexp.MustCompile(`\b`+a.Name+`\b`).ReplaceAllString(param, "Wire."+a.Name)
			result = regexp.MustCompile(`\b`+a.Name+`\b`).ReplaceAllString(result, "Wire."+a.Name)
		}
		param = strings.ReplaceAll(param, "GenerationCategory", "Wire.GenerationCategory")
		fmt.Fprintf(&requests, "%q: { params: %s; result: %s };\n", m.Name, param, result)
	}
	requests.WriteString("};\n")
	jsonData, err := json.MarshalIndent(metadata, "", "  ")
	if err != nil {
		panic(err)
	}
	files := map[string][]byte{"wire.generated.ts": []byte(wire.String()), "requests.generated.ts": []byte(requests.String()), "methods.generated.cjs": append(append([]byte("module.exports = "), jsonData...), []byte(";\n")...)}
	if err := os.MkdirAll(*out, 0755); err != nil {
		panic(err)
	}
	for name, data := range files {
		command := exec.Command("node", "../tooling/contracts/format.mjs", name)
		command.Stdin = bytes.NewReader(data)
		formatted, err := command.Output()
		if err != nil {
			panic(err)
		}
		data = formatted
		path := filepath.Join(*out, name)
		if *check {
			held, err := os.ReadFile(path)
			if err != nil || !bytes.Equal(held, data) {
				fmt.Fprintln(os.Stderr, "Generated contract drift:", path)
				os.Exit(1)
			}
		} else if err := os.WriteFile(path, data, 0644); err != nil {
			panic(err)
		}
	}
}
