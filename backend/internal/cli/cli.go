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
	"unicode"

	"github.com/charmbracelet/x/term"

	"arx/internal/agent"
	"arx/internal/config"
	"arx/internal/discovery"
	"arx/internal/tool"
)

func isTerminal(file *os.File) bool {
	return term.IsTerminal(file.Fd())
}

func terminalText(s string) string {
	var b strings.Builder
	for _, r := range s {
		if r == '\n' || r == '\t' {
			b.WriteRune(r)
			continue
		}
		if r >= ' ' && (r < 0x7f || r > 0x9f) && !unicode.Is(unicode.Cf, r) {
			b.WriteRune(r)
		}
	}
	return b.String()
}

func terminalLine(s string) string {
	s = terminalText(s)
	s = strings.ReplaceAll(s, "\n", " ")
	return strings.ReplaceAll(s, "\t", " ")
}

func registerTools() {
	tool.Register(tool.Bash)
}

func Run() error {
	judgeEnv := os.Getenv("ARX_JUDGE_MODEL")

	if err := config.LoadDotEnv(".env"); err != nil {
		fmt.Fprintln(os.Stderr, "arx: .env:", terminalText(err.Error()))
	}

	models, err := discovery.LoadModels(context.Background())
	if err != nil {
		fmt.Fprintln(os.Stderr, "arx: some providers failed:", terminalText(err.Error()))
	}
	if len(models) == 0 {
		return errors.New("no models found: set DEEPSEEK_API_KEY or OPENAI_API_KEY")
	}

	prof, err := discovery.ResolveProfile(discovery.DefaultSpec(models), models)
	if err != nil {
		return err
	}
	registerTools()
	ctrl := agent.New(prof, buildSystemPrompt())

	judgeProf := prof
	if judgeEnv != "" {
		jp, err := discovery.ResolveProfile(judgeEnv, models)
		if err != nil {
			fmt.Fprintln(os.Stderr, "arx: ARX_JUDGE_MODEL:", terminalText(err.Error()), "- using the main model")
		} else {
			judgeProf = jp
		}
	}
	judge := agent.NewJudge(judgeProf)

	if isTerminal(os.Stdin) && isTerminal(os.Stdout) {
		return runTUI(ctrl, prof, judge)
	}
	ctrl.SetGate(agent.NewGate(denyApprover{}, judge))

	tty := isTerminal(os.Stdout)
	if tty {
		fmt.Printf("arx - %s/%s · ctrl-d to leave\n", terminalLine(prof.Provider.Name), terminalLine(prof.Model))
	}

	sink := &terminalSink{color: tty}
	in := bufio.NewScanner(os.Stdin)
	in.Buffer(make([]byte, 0, 64*1024), 4*1024*1024)
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
			fmt.Println()
		}
	}
	return in.Err()
}
