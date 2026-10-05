# Desktop and interface

Electron owns native capabilities and backend lifetime. React owns presentation, drafts, navigation, and feature UI state. Persisted data and permission authority remain in Go.

## Code map

| Path | Responsibility |
| --- | --- |
| `desktop/main/profile.cjs` | Explicit profile, default names, legacy migration, keychain account preservation |
| `desktop/main/window.cjs` | Page/sender validation, sandboxed window, controls, one renderer crash reload |
| `desktop/main/backend/process.cjs` | Spawn, ready/crash state, restart, graceful shutdown |
| `desktop/main/backend/client.cjs` | Request IDs, framing/size limits, timeout/disconnect handling |
| `desktop/main/backend/view-writes.cjs` | Coalesced preference saves with ordered flush |
| `desktop/main/ipc/` | Allowlisted backend/native capabilities and saved media serving |
| `desktop/preload/index.cjs` | Context bridge and event unsubscribe functions |
| `renderer/src/app/` | Startup/initial state and visible application composition |
| `renderer/src/features/sessions/` | History/live merge, selection/drafts, sidebar listing |
| `renderer/src/shell/` | Toolbar/sidebar sizing and native drag behavior |
| `renderer/src/rich-text/` | Code, links, saved/remote images, resource policy |
| `renderer/src/platform/preferences/usePreferences.ts` | Ordered settings writes and view persistence |

## Native boundary

The window uses context isolation, no renderer Node integration, and Electron sandboxing. Native handlers verify the actual BrowserWindow/main frame and expected page URL. Window opens, navigation, and webview attachment are denied.

The backend client caps requests at 1 MiB and response lines at 16 MiB. It matches responses by ID, accepts recognized ready/domain events, and rejects pending requests on disconnect. Timeouts remove the pending entry so late replies cannot settle another request.

View edits coalesce for 200 ms; flushes remain ordered and run before ordinary requests and shutdown. Failed persistence is reported without blocking later requests.

Media serving resolves an artifact through Go, validates saved IDs, and serves GET/HEAD with range support. External links validate supported schemes and resolve file paths through current working-directory settings. Clipboard and choosers remain explicit native capabilities.

## UI state decisions

App is a visible composition point, not a hidden global hook. It wires settings, sessions, chat, approval, attachments, model connections, availability, and settings navigation.

Session requests carry selection tokens and live buffers. Merge uses precise timestamps and Unicode byte offsets. A delayed history page cannot select another chat or overwrite newer text. Clear a draft only if it still matches the text that was sent.

Model menus own their keyboard/focus and navigation stages; a delayed save must not reopen a closed menu. Chat owns scroll placement and release/follow behavior. Response view state is presentation identity; remote resource permissions use the actual conversation identity.

Remote images keep stable URL/conversation dependencies. Streaming reply updates do not discard loaded pixels. In Ask mode, automatic rendering does not fetch a remote image; the Load image action uses the cancellable approval route. See [tools](tools.md).

## Organization and ownership

Shared native presentation lives in `renderer/src/platform/desktop`, backend status in `platform/backend`, and ordered preferences in `platform/preferences`. Shell and rich-text sit beside features; generic controls live under ui. Feature CSS remains adjacent.

App mounts one generation provider. Settings, model catalog/library and tool cards read its context. The owner subscribes once, merges library revisions and job timestamps, invalidates pending fetches on backend state changes, clears old state and fetches the restarted backend. `features/models/catalog.ts` assembles available models without owning process state.

MCP and skill screens own list selection, forms, details and requests inside `features/integrations`. Settings composes those screens. Permission defaults live in `features/permissions`.

Preview bootstrap supplies fixture initial state and a fake bridge to the real App. Production startup loads its state from the native bridge and never imports preview fixtures.

`transport/contract` defines typed parameters/results and method descriptors. `cmd/arx-contracts` generates `desktop/contracts/wire.generated.ts`, `requests.generated.ts` and `methods.generated.cjs`. Renderer visibility, async scheduling, error group and timeout have one descriptor source. Main-only attachment import/read, media.resolve, save_views and shutdown remain excluded from renderer requests. Native declarations stay authored in `renderer/src/desktop.d.ts`; they reuse generated wire types.

## Verification

Subprocess/client/view-write tests cover crash/restart, invalid framing, out-of-order responses, timeouts, ordered edits, and error recovery. Browser suites cover menu anchors/focus/stale saves, streaming/scroll/image stability, model search/deletion, and remote resource approvals.

Fresh Electron checks cover profiles, permissions, images, links, integrations, and continuation. Packaging verifies preload, runtime assets, backend and keychain helper in an isolated candidate.
