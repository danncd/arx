package main

import (
	"arx/internal/config"
	"arx/internal/llm"
	"context"
	"fmt"
	"os"
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
			fmt.Printf("%-40s %4dk ctx\n", m.Spec, m.ContextWindow/1000)
		} else {
			fmt.Println(m.Spec)
		}
	}
	if len(models) == 0 {
		fmt.Fprintln(os.Stderr, "arx: no models found — set DEEPSEEK_API_KEY or OPENAI_API_KEY")
		os.Exit(1)
	}
}
