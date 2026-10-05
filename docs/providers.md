# Inference providers

The agent uses common request/message/delta types while provider adapters own their wire behavior. Model management discovers and loads models; completion adapters translate the request and stream.

## Code map

| Path under `backend/internal` | Responsibility |
| --- | --- |
| `inference/router.go` | Provider identity, validation, model information and completion dispatch |
| `inference/types.go` | Requests/messages/tool calls/deltas/responses and limit errors |
| `inference/chatcompletions/stream.go` | SSE decoding, reasoning/tools, usage and completion/limit errors |
| `inference/deepseek/` | Cloud metadata, completion request, thinking and image translation |
| `inference/chatcompletions/` | Shared local and network chat-completions adapter |
| `inference/deepseek/connection.go` | Credential/discovery connection |
| `models/local/runtime.go` | Local completion and runtime information |
| `models/network/provider.go` | Network model/instance resolution and token use |

## Identity and capabilities

IDs beginning `local:` and `network:` route to their corresponding provider. Other saved legacy models route to DeepSeek. Conflicting explicit provider identity is rejected. Preserve those IDs through physical folder changes.

Model information supplies context/output/tools/vision and thinking choices. Supported disabled/default thinking settings are selected from actual capabilities. Do not invent effort levels when metadata is missing.

Local completion uses alias `arx-local` and cache-prompt support. Network completion uses the actual loaded instance/model key, omits the local cache flag, and maps thinking effort. Backend routing remains authoritative.

## Streaming and error decisions

The shared decoder preserves fragmented text/reasoning/tool arguments and reported usage. Tool arguments may appear in the UI while streaming, but execution waits for a complete valid call. Truncated/malformed streams and unfinished calls cannot report normal completion.

Output-limit and context-length errors are typed so the agent can recover selectively. Provider rejection, authentication, invalid settings, cancellation, and storage failures remain distinct. Error messages do not echo arbitrary remote response bodies or credentials.

Reported zero cached tokens differs from missing usage metadata. Full input includes cached input; decode speed requires measurable output/time rather than fabricated timing.

Cancellation closes the active HTTP request and prevents another model/tool round. Redirect rules protect credential forwarding.

## Image and thinking differences

DeepSeek handles its supported vision model IDs, tool image call IDs, and provider-specific reasoning fields. Image payload preparation may resize previews to fit the wire budget while stored pixels remain available.

The local/network adapter validates vision capability and saved input budget. Image pixels from tool results become a user content block after the complete tool-result batch. This placement differs from the DeepSeek tool-role representation.

Keep both behaviors: sharing SSE or transport construction does not justify a universal message serializer. Thinking enable/disable flags also differ across local chat templates, network reasoning_effort, and DeepSeek.

## Organization and ownership

`inference` owns completion vocabulary, provider identity, validation and injected routing operations. DeepSeek discovery/credentials and its adapter live together under `inference/deepseek`, with explicit NewClient and NewConnection constructors. The local/network adapter lives under `inference/chatcompletions` with LocalWire and NetworkWire profiles.

Adapters share SSE decoding and the redirect-denying HTTP client constructor. DeepSeek keeps its 15-second discovery and ten-minute completion budgets; chat-completions retains its fifteen-minute request budget. Message serialization, image placement, limits, error mapping and thinking flags remain provider-specific.

The router uses credential-free completion callbacks. AuthenticatedCompleter represents DeepSeek's credential-bearing discovery client separately. Common inference vocabulary still uses session usage and saved attachment references so source moves do not alter saved data.

## Verification

Wire tests cover real model IDs, thinking fields, tool schema wrapping, text/reasoning/tool fragments, malformed/truncated streams, cancellation, context/output errors, redirect rules, image resizing/budgets and tool image placement.

External Codex checks under `backend/tests/integration/codex` are optional compatibility investigations, not an active Arx provider implementation. Keep their distinction visible.
