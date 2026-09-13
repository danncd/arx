package agent

import (
	"context"
	"encoding/json"
	"sync"
)

type Request struct {
	Tool   string
	Args   string
	Intent string
	Note   string
}

type Outcome int

const (
	Once Outcome = iota
	AlwaysSession
	Reject
)

type Approver interface {
	Ask(ctx context.Context, req Request) (Outcome, error)
}

type Judge interface {
	Judge(ctx context.Context, req Request) (allow bool, reason string, err error)
}

type Gate struct {
	approver Approver
	judge    Judge

	mu    sync.Mutex
	allow map[string]bool
}

func NewGate(approver Approver, judge Judge) *Gate {
	return &Gate{approver: approver, judge: judge, allow: map[string]bool{}}
}

type Decision struct {
	Allow  bool
	Reason string
}

func (g *Gate) Authorize(ctx context.Context, req Request, mutating bool) (bool, string, error) {
	out, err := g.AuthorizeAll(ctx, []Request{req}, []bool{mutating})
	if err != nil {
		return false, "", err
	}
	return out[0].Allow, out[0].Reason, nil
}

func (g *Gate) AuthorizeAll(ctx context.Context, reqs []Request, mutating []bool) ([]Decision, error) {
	out := make([]Decision, len(reqs))
	notes := make([]string, len(reqs))
	var judgeable []int

	for i, req := range reqs {
		if !mutating[i] {
			out[i] = Decision{Allow: true}
			continue
		}
		if hit, why := hardDeny(req); hit {
			notes[i] = "blocked: " + why
			continue
		}
		if g.allowed(allowKey(req)) {
			out[i] = Decision{Allow: true}
			continue
		}
		judgeable = append(judgeable, i)
	}

	if len(judgeable) > 0 {
		if g.judge == nil {
			for _, i := range judgeable {
				notes[i] = "no judge configured"
			}
		} else {
			dec, err := g.judgeAll(ctx, reqs, judgeable)
			if err != nil {
				return nil, err
			}
			for _, i := range judgeable {
				if dec[i].Allow {
					out[i] = Decision{Allow: true}
					continue
				}
				notes[i] = dec[i].Reason
			}
		}
	}

	for i := range reqs {
		if out[i].Allow || notes[i] == "" {
			continue
		}
		req := reqs[i]
		req.Note = notes[i]
		allowed, reason, err := g.prompt(ctx, req, allowKey(req))
		if err != nil {
			return nil, err
		}
		out[i] = Decision{Allow: allowed, Reason: reason}
	}
	return out, nil
}

func (g *Gate) judgeAll(ctx context.Context, reqs []Request, idx []int) ([]Decision, error) {
	dec := make([]Decision, len(reqs))
	var wg sync.WaitGroup
	for _, i := range idx {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			allow, reason, err := g.judge.Judge(ctx, reqs[i])
			switch {
			case err != nil:
				dec[i] = Decision{Reason: "judge unavailable"}
			case allow:
				dec[i] = Decision{Allow: true}
			default:
				dec[i] = Decision{Reason: "judge: " + reason}
			}
		}(i)
	}
	wg.Wait()
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return dec, nil
}

func (g *Gate) prompt(ctx context.Context, req Request, key string) (bool, string, error) {
	if g.approver == nil {
		return false, "no approval channel", nil
	}
	outcome, err := g.approver.Ask(ctx, req)
	if err != nil {
		return false, "", err
	}
	switch outcome {
	case Once:
		return true, "", nil
	case AlwaysSession:
		g.remember(key)
		return true, "", nil
	}
	if req.Note != "" {
		return false, req.Note, nil
	}
	return false, "user denied", nil
}

func (g *Gate) allowed(key string) bool {
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.allow[key]
}

func (g *Gate) remember(key string) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.allow[key] = true
}

func allowKey(req Request) string {
	var v any
	if json.Unmarshal([]byte(req.Args), &v) == nil {
		if b, err := json.Marshal(v); err == nil {
			return req.Tool + "\x00" + string(b)
		}
	}
	return req.Tool + "\x00" + req.Args
}
