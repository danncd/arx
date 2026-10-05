package main

import (
	engines "arx/internal/generation/runtime"
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	if engines.SupervisorMain() {
		return
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	go func() {
		<-ctx.Done()
		os.Stdin.Close()
	}()
	if err := run(ctx, os.Stdin, os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
