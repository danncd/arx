// Package cli is the terminal frontend: discovery, model selection,
// slash commands, and the REPL that drives the agent.
package cli

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	"arx/internal/agent"
	"arx/internal/config"
	"arx/internal/llm"
	"arx/internal/tool"
)

const systemPrompt = "You are arx, a concise assistant."

/* Renders a turn: thinking dim, answers plain, tool runs bracketed. */

type terminalSink struct{}

func (terminalSink) Token(s string, thinking bool) {
	if thinking {
		fmt.Print("\033[90m" + s + "\033[0m")
	} else {
		fmt.Print(s)
	}
}

func (terminalSink) ToolResult(name, out string) {
	fmt.Println("  [" + name + "] → " + out)
}

/* Session state the commands operate on. */

type cli struct {
	models []llm.ModelInfo
	prof   llm.Profile
	ctrl   *agent.Controller
}

func Run() error {
	if err := config.LoadDotEnv(".env"); err != nil {
		fmt.Fprintln(os.Stderr, "arx: .env:", err)
	}

	models, err := llm.LoadModels(context.Background())
	if err != nil {
		fmt.Fprintln(os.Stderr, "arx: some providers failed:", err)
	}
	if len(models) == 0 {
		return errors.New("no models found — set DEEPSEEK_API_KEY or OPENAI_API_KEY")
	}

	prof, err := resolveProfile(defaultSpec(models), models)
	if err != nil {
		return err
	}
	tool.Register(tool.Clock)

	c := &cli{models: models, prof: prof, ctrl: agent.New(prof, systemPrompt)}
	fmt.Printf("arx — %s/%s · %d models · /help for commands\n",
		prof.Provider.Name, prof.Model, len(models))

	in := bufio.NewScanner(os.Stdin)
	in.Buffer(make([]byte, 0, 64*1024), 4*1024*1024) // a >64KB paste must not end the REPL
	for {
		fmt.Print("> ")
		if !in.Scan() {
			break
		}
		text := strings.TrimSpace(in.Text())
		if text == "" {
			continue
		}
		if strings.HasPrefix(text, "/") {
			if quit := c.command(text); quit {
				return nil
			}
			continue
		}
		switch err := c.ctrl.RunTurn(context.Background(), text, terminalSink{}); {
		case errors.Is(err, agent.ErrStepLimit):
			fmt.Fprintln(os.Stderr, "arx: step limit reached — turn ended without a final answer")
		case err != nil:
			fmt.Fprintln(os.Stderr, "arx:", err)
		default:
			fmt.Println() // close the streamed line
		}
	}
	// Scan() false is EOF only when Err() is nil; a too-long line or a
	// read error must not masquerade as a clean ctrl-D.
	return in.Err()
}

/* Dispatches a slash command; reports whether the REPL should quit. */

func (c *cli) command(line string) (quit bool) {
	cmd, arg, _ := strings.Cut(line, " ")
	arg = strings.TrimSpace(arg)
	switch cmd {
	case "/exit", "/quit":
		return true
	case "/help":
		fmt.Print(`/models         list discovered models
/model <spec>   switch model (resets the conversation)
/clear          reset the conversation
/exit           leave
`)
	case "/models":
		printCatalog(c.models)
	case "/clear":
		c.ctrl = agent.New(c.prof, systemPrompt)
		fmt.Println("conversation reset")
	case "/model":
		if arg == "" {
			fmt.Printf("current: %s/%s — usage: /model provider/model\n",
				c.prof.Provider.Name, c.prof.Model)
			return false
		}
		prof, err := resolveProfile(arg, c.models)
		if err != nil {
			fmt.Fprintln(os.Stderr, "arx:", err)
			return false
		}
		c.prof = prof
		c.ctrl = agent.New(prof, systemPrompt)
		fmt.Printf("model set to %s/%s (conversation reset)\n",
			prof.Provider.Name, prof.Model)
	default:
		fmt.Fprintln(os.Stderr, "arx: unknown command "+cmd+" — /help lists commands")
	}
	return false
}

/* Parses a spec and copies the catalog's capability flags onto it. */

func resolveProfile(spec string, models []llm.ModelInfo) (llm.Profile, error) {
	prof, err := llm.Parse(spec)
	if err != nil {
		return llm.Profile{}, err
	}
	for _, m := range models {
		if m.Provider == prof.Provider.Name && m.Model == prof.Model {
			prof.Tools = m.Tools
			prof.ToolsKnown = m.ToolsKnown
			break
		}
	}
	return prof, nil
}

/*
	defaultSpec picks the model when ARX_MODEL doesn't: deepseek when
	its key exists, else the first discovered model. A default whose
	provider has no key would 401 on every turn.
*/

func defaultSpec(models []llm.ModelInfo) string {
	if spec := os.Getenv("ARX_MODEL"); spec != "" {
		return spec
	}
	if p, err := llm.GetProvider("deepseek"); err == nil && p.Key() != "" {
		return "deepseek/deepseek-v4-flash"
	}
	return models[0].Spec // non-empty: guarded by the caller
}

func printCatalog(models []llm.ModelInfo) {
	for _, m := range models {
		if m.ContextWindow == 0 {
			fmt.Println(m.Spec)
			continue
		}
		badge := ""
		if m.Reasoning {
			badge += " R"
		}
		if m.Tools {
			badge += " T"
		}
		fmt.Printf("%-40s %4dk ctx%s\n", m.Spec, m.ContextWindow/1000, badge)
	}
}
