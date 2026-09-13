# Arx

A terminal AI coding agent in Go. It streams completions from DeepSeek or OpenAI, gives the model one tool (`bash`), and runs a command only if it clears an automatic permission gate. Handing a model a shell is easy; letting it use one safely is the interesting part.

<!--
A GIF of a real task, with the approval card appearing, is worth more than
everything below. Record with asciinema or terminalizer, export to docs/demo.gif,
then delete this comment and add:
![Arx running a task](docs/demo.gif)
-->

```bash
git clone https://github.com/danncd/arx.git && cd arx/backend
export DEEPSEEK_API_KEY=sk-...
go run ./cmd/arx
```

Go 1.26.6 or newer. `OPENAI_API_KEY` works too.

## The permission gate

Only `bash` is mutating, so read-only work never interrupts. A mutating call is checked in four layers, cheapest first.

**Hard deny.** Nothing can override these, not even your own approval: privilege escalation, piping a download into an interpreter, `eval`, `rm -rf` outside the workspace, raw device writes, recursive `chmod` on `/` or `~`, and touches of `.ssh`, `.aws`, `.kube` or `.git/hooks`. The matcher resolves wrappers (`env`, `command`, `xargs`), strips variable assignments and walks statement separators, so `env FOO=1 sudo rm -rf /` is caught rather than only the bare form.

**Session memory.** "Always this session" keys on the canonicalized arguments, so reordering JSON keys does not re-prompt.

**An LLM judge.** A second model allows only calls that are safe, contained, reversible and on-goal, denying when unsure. Command text is untrusted, since a repo file can get text into it, so the goal and action are fenced with a fresh nonce per judgement. An error or unparseable answer counts as denial.

**You.** A card offering allow once, always this session, or deny. Enter is dead for the first 250 ms so a keystroke in flight cannot approve by accident.

With no terminal to ask, the fallback denies; only the judge can approve. That is what lets Arx run unattended.

## Also worth knowing

- **The shell gets a scrubbed environment.** Only `HOME`, `PATH`, `SHELL`, `TERM`, `TMPDIR` and the locale variables survive, so an `AWS_SECRET_ACCESS_KEY` in your shell is invisible to a command the model wrote. Commands expecting an injected token fail by design.
- **Commands run in their own process group,** killed as a group on timeout, with output capped at 64 KB. Streams are validated rather than trusted: invalid UTF-8, idle stalls and a finish reason that contradicts the tool calls are all rejected, and a failed stream keeps what was already said.

## Configuration

| Variable | Effect |
| --- | --- |
| `DEEPSEEK_API_KEY` | Key for the DeepSeek provider |
| `OPENAI_API_KEY` | Key for the OpenAI provider |
| `ARX_MODEL` | Default model as `provider/model` |
| `ARX_JUDGE_MODEL` | Model used to judge commands; defaults to the main model |

A `.env` in the working directory is read and will not override variables already set.

## Tests

`cd backend && go test ./...` runs 136 tests over the gate, the hard-deny rules, stream assembly and model discovery. Nothing calls a live API.
