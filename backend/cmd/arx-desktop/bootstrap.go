package main

import (
	"context"
	"errors"
	"io"
	"os"

	service "arx/internal/app"
	"arx/internal/transport/stdio"
)

func run(ctx context.Context, input io.Reader, output io.Writer) error {
	directory := os.Getenv("ARX_STATE")
	if directory == "" {
		return errors.New("ARX_STATE is required")
	}
	service, err := service.Open(directory)
	if err != nil {
		return err
	}
	defer service.Close()
	return stdio.NewServer(service, output).Serve(ctx, input)
}
