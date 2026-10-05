# Chat and agent execution

A reply captures model/settings/history, persists its user message, then owns cancellation and resources until it completes. The agent makes ordered bounded model/tool rounds and stores enough history to resume without replaying completed actions.

## Code map

| Path | Responsibility |
| --- | --- |
| `chat/send.go` | Reply lock, deduplication, model hold, input/history validation and user persistence |
| `chat/reply.go` | Async reply, skill scope, title, completion, cancel and model release |
| `chat/service.go` | State, event revision, stop and close |
| `chat/context.go` | Current context report from history/model/saved compaction |
| `agent/loop.go` | Model/tool rounds, continuation and failure handling |
| `agent/record.go` / `stream.go` / `tool_stream.go` | Streamed text/reasoning/tool records |
| `agent/history.go` / `round_history.go` | Replayable provider messages and interrupted-result meaning |
| `agent/context/` / `agent/compaction/` | Budget, usage, retained history, checkpoints and bounded recovery |
| `renderer/src/features/chat/` / `sessions/` | Composer/send guard, transcript rendering, history/live merge |

Backend paths above are under `backend/internal`. Renderer paths are under `desktop`.

## Send and stop flow

Send trims text and validates message/conversation IDs, text length, and attachments. Under one reply lock it rejects a second active reply and returns an existing run for a duplicate request. It validates the selected model and history before recording a new user chunk.

Local inference requires a ready matching model and supported tools. Vision history cannot be continued with a text-only model. A local model hold transfers to the reply goroutine only after startup succeeds.

A new chat gets its own persisted permissions. User chunks retain runtime date/directory. The reply builds a skill scope from explicit invocations in the current user text, names a new conversation, reads generation/integration guidance, then runs the agent.

Stop cancels the current reply and pending work. Partial text remains saved; an unfinished streamed tool call is not executed. Completion releases the local model hold, closes the done channel, and returns chat state to idle.

## Context and continuation decisions

Model limits must be known and at least 4,096 context tokens. The budget reserves output and headroom; it accounts for system/tool definitions, text, images, and reported provider usage. Cached input is part of input context, not subtracted from it.

Compaction retains atomic exchanges and the current user request. Checkpoints are separate from full transcript storage and validated with coverage/digest. A failed or cancelled summary does not advance coverage. Image pressure can trigger compaction independently of token pressure.

Auto continue defaults on and is captured when a reply starts. The loop permits 32 rounds per batch and up to three additional batches, for 128 rounds maximum. Recoverable context/output limits permit up to three attempts per reply. Authentication, validation, storage, permission, and cancellation errors are not context recovery.

Recovery reduces only the model's context copy of oversized tool output/reasoning/images. Exact user instructions, tool arguments, loaded skill guidance, and full stored results remain intact. Summaries can fall back to bounded factual checkpoints after bounded smaller-batch attempts.

Partial output remains progress. Unfinished calls are context only; unknown interrupted tool outcomes require checking before retry. Continuation must not repeat completed actions. Optional thinking may be disabled for an output-limit continuation when supported.

Runtime prompts are product data. Their rationale belongs here and in [tools](tools.md)/[generation](generation.md); removing developer comments does not remove prompt strings.

## Rendering decisions

Join Unicode chunks using stored byte offsets and preserve reasoning/tool boundaries. Display one response turn with tools in chronological position. Completion footers reserve space while streaming. Generation speed uses measured decode intervals and provider output tokens, excluding tool wait.

History/live pages use selection tokens and timestamp comparison; delayed pages do not replace newer reply state. These remain cohesive helpers rather than a universal state framework.

## Organization and ownership

`chat.Service` owns the active reply lock, cancellation/done channel, current chat revision/event and notification. Send and history validation stay together in send.go; reply goroutine and title work stay together in reply.go; lifecycle and event publication stay together in service.go.

App injects settings, session and attachment stores, the inference router, lazy local model access, tool execution and generation/integration guidance. Reply settings/model selection remain fixed at startup. Send preparation runs while the reply mutex is held. The goroutine owns transferred cancellation and local-model release.

App retains explicit cross-feature coordination and typed aliases for its public composition API. WithState executes permission, integration and local model guards under the same reply mutex. Independent locks are not a replacement for this invariant.

## Verification

Existing tests cover duplicate send, active reply rejection, new/default chat policy, stop during approval/compaction, storage failure before side effects, Unicode/reasoning/tool replay, output interruption, recovery bounds, skill guidance retention, context usage after restart, and local-model selection.

Add meaningful coverage for new ownership seams: cancellation racing integration/model edits, exactly-once hold release, and backend restart/state reattachment. Keep existing browser stream/turn/scroll/session tests.
