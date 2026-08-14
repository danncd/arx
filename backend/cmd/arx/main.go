package main

import (
	"arx/internal/config"
	"arx/internal/llm"
	"context"
	"fmt"
	"os"
)

func main() {

	config.LoadDotEnv(".env")

	models, err := llm.LoadModels(context.Background())

	if err != nil {
		fmt.Fprintln(os.Stderr, "arx: some providers failed:", err)
	}
	for _, m := range models {
		fmt.Println(m.Spec)
		if m.ContextWindow > 0 {
			fmt.Println(m.ContextWindow)
		}
	}
}
