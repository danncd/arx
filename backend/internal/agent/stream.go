package agent

import (
	provider "arx/internal/inference"
	"context"
	"time"
)

func (l Loop) stream(ctx context.Context, request provider.Request, record *recorder) (provider.Response, error) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	deltas := make(chan provider.Delta)
	type completion struct {
		response provider.Response
		err      error
	}
	finished := make(chan completion, 1)
	go func() {
		var timing generationTiming
		response, err := l.Complete(ctx, request, func(delta provider.Delta) error {
			timing.observe(delta, time.Now())
			select {
			case deltas <- delta:
				return nil
			case <-ctx.Done():
				return ctx.Err()
			}
		})
		timing.apply(response.Usage, time.Now())
		finished <- completion{response, err}
	}()
	ticker := time.NewTicker(60 * time.Millisecond)
	defer ticker.Stop()
	for {
		var err error
		select {
		case delta := <-deltas:
			err = record.delta(delta)
		case result := <-finished:
			return result.response, result.err
		case <-ticker.C:
			if record.dirty {
				err = record.flush()
			}
		}
		if err != nil {
			cancel()
			<-finished
			return provider.Response{}, err
		}
	}
}
