package cli

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/charmbracelet/x/term"

	"arx/internal/agent"
	"arx/internal/config"
	"arx/internal/llm"
	"arx/internal/tool"
)

/* Every conversation starts here. */

const systemPrompt = "You are arx, a concise assistant."

/* Reports whether a file is attached to a terminal. */

func isTerminal(file *os.File) bool {
	return term.IsTerminal(file.Fd())
}

func terminalText(s string) string {
	var b strings.Builder
	for _, r := range s {
		if r == '\n' || r == '\t' || r >= ' ' && (r < 0x7f || r > 0x9f) {
			b.WriteRune(r)
		}
	}
	return b.String()
}

/* Streams plain-mode output. */

type terminalSink struct {
	color  bool // escape codes stay out of pipes and files
	answer strings.Builder
}

func (t *terminalSink) Token(s string, thinking bool) {
	s = terminalText(s)
	if !t.color {
		if !thinking {
			t.answer.WriteString(s)
		}
		return
	}
	if thinking && t.color {
		fmt.Print("\033[90m" + s + "\033[0m")
	} else {
		fmt.Print(s)
	}
}

func (t *terminalSink) ToolResult(name, out string, _ time.Duration, _ bool) {
	if !t.color {
		t.answer.Reset()
		return
	}
	fmt.Println("  [" + terminalLine(name) + "] → " + terminalText(out))
}

func (t *terminalSink) finish(success bool) {
	if !t.color {
		if success {
			fmt.Print(t.answer.String())
		}
		t.answer.Reset()
	}
}

/* Starts the terminal UI or the plain pipe loop. */

func Run() error {
	if err := config.LoadDotEnv(".env"); err != nil {
		fmt.Fprintln(os.Stderr, "arx: .env:", terminalText(err.Error()))
	}

	models, err := llm.LoadModels(context.Background())
	if err != nil {
		fmt.Fprintln(os.Stderr, "arx: some providers failed:", terminalText(err.Error()))
	}
	if len(models) == 0 {
		return errors.New("no models found: set DEEPSEEK_API_KEY or OPENAI_API_KEY")
	}

	prof, err := resolveProfile(defaultSpec(models), models)
	if err != nil {
		return err
	}
	registerTools()
	ctrl := agent.New(prof, systemPrompt)

	// Pipes keep the plain loop for scripts.
	if isTerminal(os.Stdin) && isTerminal(os.Stdout) {
		return runTUI(ctrl, prof)
	}
	// A piped stdout wants bare answers: no banner, prompt, or color.
	tty := isTerminal(os.Stdout)
	if tty {
		fmt.Printf("arx - %s/%s · ctrl-d to leave\n", terminalLine(prof.Provider.Name), terminalLine(prof.Model))
	}

	sink := &terminalSink{color: tty}
	in := bufio.NewScanner(os.Stdin)
	in.Buffer(make([]byte, 0, 64*1024), 4*1024*1024) // large pastes stay valid input
	for {
		if tty {
			fmt.Print("> ")
		}
		if !in.Scan() {
			break
		}
		text := strings.TrimSpace(in.Text())
		if text == "" {
			continue
		}
		turnCtx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
		err := ctrl.RunTurn(turnCtx, text, sink)
		interrupted := errors.Is(turnCtx.Err(), context.Canceled)
		stop()
		sink.finish(err == nil)
		switch {
		case interrupted && errors.Is(err, context.Canceled):
			return nil
		case errors.Is(err, agent.ErrStepLimit):
			fmt.Fprintln(os.Stderr, "arx: step limit reached before a final answer")
		case err != nil:
			fmt.Fprintln(os.Stderr, "arx:", terminalText(err.Error()))
		default:
			fmt.Println() // close the streamed line
		}
	}
	// Scan() false is EOF only when Err() is nil.
	return in.Err()
}

func terminalLine(s string) string {
	s = terminalText(s)
	s = strings.ReplaceAll(s, "\n", " ")
	return strings.ReplaceAll(s, "\t", " ")
}

func registerTools() {
	tool.Register(tool.ReadFile)
	tool.Register(tool.Fetch)
	tool.Register(tool.Bash)
}

/* Adds catalog capabilities to a parsed profile. */

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

/* Picks ARX_MODEL, DeepSeek, or the first discovered model. */

func defaultSpec(models []llm.ModelInfo) string {
	if spec := os.Getenv("ARX_MODEL"); spec != "" {
		return spec
	}
	if p, err := llm.GetProvider("deepseek"); err == nil && p.Key() != "" {
		for _, m := range models {
			if m.Spec == "deepseek/deepseek-v4-flash" {
				return m.Spec
			}
		}
	}
	return models[0].Spec // Run checks that the catalog is not empty
}
