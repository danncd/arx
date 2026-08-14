package main

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

// terminalSink renders a turn for the terminal: thinking dim, answers
// plain, tool runs bracketed.
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

func main() {
	if err := config.LoadDotEnv(".env"); err != nil {
		fmt.Fprintln(os.Stderr, "arx: .env:", err)
	}

	models, err := llm.LoadModels(context.Background())
	if err != nil {
		fmt.Fprintln(os.Stderr, "arx: some providers failed:", err)
	}
	printCatalog(models)
	if len(models) == 0 {
		fmt.Fprintln(os.Stderr, "arx: no models found — set DEEPSEEK_API_KEY or OPENAI_API_KEY")
		os.Exit(1)
	}

	prof, err := llm.Parse(defaultSpec(models))
	if err != nil {
		fmt.Fprintln(os.Stderr, "arx:", err)
		os.Exit(1)
	}
	tool.Register(tool.Clock)
	ctrl := agent.New(prof, "You are arx, a concise assistant.")

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
		switch err := ctrl.RunTurn(context.Background(), text, terminalSink{}); {
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
	if err := in.Err(); err != nil {
		fmt.Fprintln(os.Stderr, "arx: stdin:", err)
		os.Exit(1)
	}
}

// defaultSpec picks the model when ARX_MODEL doesn't: deepseek when
// its key exists, else the first discovered model — a default whose
// provider has no key would 401 on every turn right after printing a
// list of models that work.
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
