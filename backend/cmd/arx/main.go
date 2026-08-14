package main

import (
	"arx/internal/config"
	"arx/internal/llm"
	"arx/internal/tool"
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

func main() {
	if err := config.LoadDotEnv(".env"); err != nil {
		fmt.Fprintln(os.Stderr, "arx: .env:", err)
	}

	models, err := llm.LoadModels(context.Background())
	if err != nil {
		fmt.Fprintln(os.Stderr, "arx: some providers failed:", err)
	}
	for _, m := range models {
		if m.ContextWindow > 0 {
			badge := ""
			if m.Reasoning {
				badge += " R"
			}
			if m.Tools {
				badge += " T"
			}
			fmt.Printf("%-40s %4dk ctx%s\n", m.Spec, m.ContextWindow/1000, badge)
		} else {
			fmt.Println(m.Spec)
		}
	}
	if len(models) == 0 {
		fmt.Fprintln(os.Stderr, "arx: no models found — set DEEPSEEK_API_KEY or OPENAI_API_KEY")
		os.Exit(1)
	}

	// Default model: deepseek when its key exists, else the first
	// discovered model — a default whose provider has no key would 401
	// on every turn right after printing a list of models that work.
	spec := os.Getenv("ARX_MODEL")
	if spec == "" {
		spec = "deepseek/deepseek-v4-flash"
		if p, perr := llm.GetProvider("deepseek"); perr != nil || p.Key() == "" {
			spec = models[0].Spec // non-empty: guarded above
		}
	}
	prof, err := llm.Parse(spec)
	if err != nil {
		fmt.Fprintln(os.Stderr, "arx:", err)
		os.Exit(1)
	}
	tool.Register(tool.Clock)
	msgs := []llm.Message{{Role: "system", Content: "You are arx, a concise assistant."}}

	in := bufio.NewScanner(os.Stdin)
	// Default cap is 64KB per line; a big pasted log would end the REPL.
	in.Buffer(make([]byte, 0, 64*1024), 4*1024*1024)
	for {
		fmt.Print("> ")
		if !in.Scan() {
			break
		}
		text := strings.TrimSpace(in.Text())
		if text == "" {
			continue
		}
		msgs = append(msgs, llm.Message{Role: "user", Content: text})

		done := false
		for step := 0; step < 8 && !done; step++ { // backstop, not a leash
			reply, err := llm.Stream(context.Background(), prof, msgs, tool.Specs(),
				func(s string, thinking bool) {
					if thinking {
						fmt.Print("\033[90m" + s + "\033[0m")
					} else {
						fmt.Print(s)
					}
				})
			if err != nil {
				fmt.Fprintln(os.Stderr, "arx:", err)
				done = true
				continue
			}
			msgs = append(msgs, reply)

			if len(reply.ToolCalls) == 0 {
				fmt.Println()
				done = true
				continue
			}
			for _, tc := range reply.ToolCalls {
				out := runTool(tc)
				fmt.Println("  [" + tc.Function.Name + "] → " + out)
				msgs = append(msgs, llm.Message{
					Role: "tool", ToolCallID: tc.ID, Content: out,
				})
			}
		}
		if !done {
			// The loop ran out of steps while the model was still
			// calling tools; silence here looked exactly like success.
			fmt.Fprintln(os.Stderr, "arx: step limit reached — turn ended without a final answer")
		}
	}
	// Scan() returning false is EOF only when Err() is nil; a too-long
	// line or a read error must not masquerade as a clean ctrl-D.
	if err := in.Err(); err != nil {
		fmt.Fprintln(os.Stderr, "arx: stdin:", err)
		os.Exit(1)
	}
}

func runTool(tc llm.ToolCall) string {
	t, ok := tool.Get(tc.Function.Name)
	if !ok {
		return "error: unknown tool " + tc.Function.Name
	}
	out, err := t.Run(context.Background(), json.RawMessage(tc.Function.Arguments))
	if err != nil {
		return "error: " + err.Error()
	}
	if out == "" {
		// The model can't distinguish "ran, no output" from a dropped
		// result; say it explicitly.
		return "(no output)"
	}
	return out
}
