/* Terminal frontend, minimal form: read a line, run a turn, print. */

package cli

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

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

func (terminalSink) ToolResult(name, out string, _ time.Duration) {
	fmt.Println("  [" + name + "] → " + out)
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
	ctrl := agent.New(prof, systemPrompt)

	// A real terminal gets the Bubble Tea UI; pipes keep the plain
	// read-run-print loop so scripting and tests stay possible.
	if isTerminal() {
		return runTUI(ctrl, prof)
	}
	fmt.Printf("arx — %s/%s · ctrl-d to leave\n", prof.Provider.Name, prof.Model)

	in := bufio.NewScanner(os.Stdin)
	in.Buffer(make([]byte, 0, 64*1024), 4*1024*1024) // large pastes stay valid input
	for {
		fmt.Print("> ")
		if !in.Scan() {
			break
		}
		text := strings.TrimSpace(in.Text())
		if text == "" {
			continue
		}
		switch err := ctrl.RunTurn(context.Background(), text, terminalSink{}); {
		case errors.Is(err, agent.ErrStepLimit):
			fmt.Fprintln(os.Stderr, "arx: step limit reached before a final answer")
		case err != nil:
			fmt.Fprintln(os.Stderr, "arx:", err)
		default:
			fmt.Println() // close the streamed line
		}
	}
	// Scan() false is EOF only when Err() is nil.
	return in.Err()
}

/* Parses a spec and copies the catalog's capabilities onto it. */

func resolveProfile(spec string, models []llm.ModelInfo) (llm.Profile, error) {
	prof, err := llm.Parse(spec)
	if err != nil {
		return llm.Profile{}, err
	}
	for _, m := range models {
		if m.Provider == prof.Provider.Name && m.Model == prof.Model {
			prof.Tools = m.Tools
			prof.ToolsKnown = m.ToolsKnown
			// The catalog's completion ceiling beats the parse default.
			if m.MaxOutput > 0 && m.MaxOutput < prof.MaxTokens {
				prof.MaxTokens = m.MaxOutput
			}
			break
		}
	}
	return prof, nil
}

/* Picks the default model: ARX_MODEL, else deepseek, else first found. */

func defaultSpec(models []llm.ModelInfo) string {
	if spec := os.Getenv("ARX_MODEL"); spec != "" {
		return spec
	}
	if p, err := llm.GetProvider("deepseek"); err == nil && p.Key() != "" {
		return "deepseek/deepseek-v4-flash"
	}
	return models[0].Spec // Run checks that the catalog is not empty
}
