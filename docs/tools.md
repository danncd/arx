# Tools and permissions

Tools prepare a concrete action, authorize that action, then execute it with cancellation. Permission review is tied to an opaque pending identity rather than a provider tool-call ID.

## Code map

| Path under `backend/internal` | Responsibility |
| --- | --- |
| `tools/definitions.go` / `generation.go` | Executable tool descriptions and JSON schemas |
| `tools/types.go` | Calls, prepared action/run/close hooks, saved results |
| `tools/runner/runner.go` | One active action, preparation/authorization/execution/cancellation/image save |
| `tools/runner/mcp.go` / `skills.go` | Integration tool schemas, approval, skill scope |
| `tools/files/` | Bounded reads/search and atomic reviewed changes |
| `tools/web/` | Public-network fetch/search/image/icon and extraction/cache |
| `tools/bash/` | Bounded shell output, timeout/process group and folder-mode sandbox |
| `permissions/` | Policy/root validation and pending review identity |
| `tools/service.go` / `tools/generation_runner.go` | Conversation policy and generation action preparation |

Renderer owners are `features/permissions`, `features/chat/tools` and `rich-text/ResourcePolicy.tsx`.

## Permission behavior

| Mode | Current behavior |
| --- | --- |
| Ask (`ask`) | Review ordinary reads, writes, web and commands |
| Selected folders (`folders`) | Reads ordinarily proceed; file writes require reviewed roots; Bash uses its separate sandbox profile |
| Bypass (`full`) | Explicit policy bypasses pending action review |

MCP calls may force Ask unless Bypass applies. User-reviewed read-only allowlists, not server hints, determine whether a call can proceed in selected-folders mode. Skill/catalog discovery is metadata; skill guidance never grants access.

Default policy and per-chat policy persist independently. A new chat captures the selected/default policy. Existing chat permission edits require known conversation identity and idle reply ownership. The configured working directory is a path default, not an access boundary.

## Execution invariants and reasons

The runner has one active action and separate cancellation identity. Prepared actions retain the exact approved input and resource boundary. Pending approvals get new random identities; cancelling one cannot authorize its successor.

File edits read existing content and construct a before/after action. After approval, the tool rereads it and rejects stale content instead of overwriting a later change. Writes use a filesystem-root boundary, reject inappropriate targets and symlink escapes, and publish with temporary-file rename. Keep that workflow cohesive.

Bash in selected-folders mode uses macOS sandbox-exec with allowed writes to selected roots and a temporary directory, and restricted networking. Commands use bounded output and terminate their process group on timeout/cancel. Public network access uses the web tool.

Web requests validate schemes/credentials, resolved IPs, redirects, response size and media format. Public web access rejects private/local addresses; network model connections are a separate capability intentionally able to reach configured local servers. Do not merge their HTTP policies.

Search chooses usable bounded results, preserves source links/excerpts, uses fallback providers/cache, and reports challenge/empty/unrelated output instead of inventing successful evidence.

## Rendering resources

Automatic remote images/icons require an existing permissive conversation. Ask renders a Load image action without fetching. Manual load goes through tools.run and approval, supports cancellation on navigation, and saves pixels for later reads.

Conversation identity is authoritative; response view keys are presentation-only. Saved `arx-image` and `arx-media` references remain distinct.

## Organization and ownership

The tools service owns known-conversation checks, per-chat/default policy selection, remote-resource prechecks and result/error composition. Its executor interface is implemented by the runner without making tool definitions depend on their implementing subpackage.

Generation and combine are prepared through the same runner as files/web/Bash/MCP/skills. The active call, pending authorization, cancellation and idle guard therefore cover generation too. The generation service retains durable job and supervisor ownership. Cancellation is checked after approval and before executing the prepared operation.

Integration edits require reply and tool idleness. Remote resource checks remain above the web client. Tool schemas are executable specifications; permissions come from the current policy and reviewed action, never documentation or historical summaries.

## Verification

Tests cover exact approval identity/arguments, stop-before-write, denial/retry, stale edits, symlink/hardlink boundaries, output/time limits, child cleanup, public-network redirects/IPs, image/media validation, search challenges/fallback and unknown-chat resource refusal.

Browser safety tests cover zero automatic Ask fetches, manual review/denial/retry, navigation cancellation and stable loaded images during streaming.
