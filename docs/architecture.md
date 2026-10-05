# Architecture

Arx has three cooperating runtime layers. React renders the workspace; Electron owns native capabilities and the Go subprocess; Go owns application behavior, persistent state, inference, tool authorization, and local model/generation control. Python implements generation adapters and reports JSON events to Go.

## Composition points

| Path | Responsibility |
| --- | --- |
| `desktop/main/index.cjs` | Profile, single instance, app/window/backend lifetime, IPC registration |
| `desktop/main/window.cjs` | Sender/frame identity, window controls, page loading and renderer recovery |
| `desktop/preload/index.cjs` | Narrow context-isolated API exposed to React |
| `desktop/renderer/src/app/Startup.tsx` | Backend-ready startup, initial state, errors/retry |
| `desktop/renderer/src/app/App.tsx` | Workspace feature hooks, model selection, settings/shell composition |
| `backend/cmd/arx-desktop/` | Signals, generation supervisor entry, application startup |
| `backend/internal/app/app.go` | Construction, shared state/managers, shutdown |
| `backend/internal/transport/stdio/` | Newline JSON framing, strict handler parameters, responses/events |
| `runtimes/generation/worker.py` | Worker operation/runtime dispatch |

The renderer invokes typed methods through `window.arxDesktop`. Electron validates the native sender and allowlisted method, then writes a request to the Go client. Responses carry matching IDs; events publish chat, permission, local model, or generation updates. stdout belongs to the protocol. Shutdown cancels work and closes the backend before app exit.

## Subsystem ownership

| Subsystem | Authority |
| --- | --- |
| Chat lifecycle | One current reply, send deduplication, user/history preparation, stop/completion/events |
| Agent | Ordered bounded model/tool rounds, streaming records, compaction and continuation |
| Inference | Model/provider resolution and completion wire translation |
| Models | Catalog/library metadata, downloads, runtime load/hold/idle and network discovery |
| Generation | Selected generation models, jobs, request/source limits, Python/runtime supervision |
| Tools and permissions | Prepare the exact action, authorize it, execute/cancel, retain result identity |
| Integrations | MCP configuration/sessions/discovery and skill registration/manifests/activity |
| Storage | Transcript, settings, summaries, saved images and generated artifacts |
| Desktop | Native chooser/clipboard/link/window capabilities and presentation |

## Decisions and reasons

Keep one Go module and one desktop npm package. These match the actual process and build boundaries. Python is shipped runtime source rather than a separate network service.

The settings snapshot/model used by a reply is captured when it begins. Dynamic changes must not silently redirect an active task. Cancellation and resource release remain visible in the reply owner.

Working-directory configuration is a default for resolving paths. Permission policy is a separate access/approval decision; choosing a directory does not itself create a sandbox.

Saved resource IDs are durable references independent of original input files. Attachment and generated artifact IDs have distinct formats and stores.

Keep provider-specific transport behavior below the agent, and feature policy above platform primitives. A filesystem rename helper does not own recovery or model deletion policy.

## Organization and ownership

```text
Arx/
├── backend/
│   ├── cmd/{arx-desktop,arx-contracts}/
│   ├── internal/
│   │   ├── app/                  construction and cross-feature coordination
│   │   ├── chat/                 reply lifecycle and context
│   │   ├── agent/                model/tool rounds and recovery
│   │   ├── inference/            routing, deepseek, chatcompletions
│   │   ├── models/               local GGUF and network model management
│   │   ├── generation/           jobs, model policy, runtime supervision
│   │   ├── tools/                definitions, service and executors
│   │   ├── permissions/          policy and pending review
│   │   ├── integrations/         mcp, skills and enabled catalog
│   │   ├── media/                attachments and generated artifacts
│   │   ├── sessions/             transcript and compaction storage
│   │   ├── settings/             preferences and saved views
│   │   ├── transport/            contract and stdio
│   │   └── platform/             paths, keychain, download, atomicfile
│   └── tests/integration/
├── desktop/
│   ├── main/                     native process, window, backend and IPC
│   ├── preload/                  isolated bridge
│   ├── contracts/                generated wire/method policy, native status
│   ├── renderer/
│   │   ├── index.html
│   │   ├── preview.html
│   │   └── src/
│   │       ├── app/              startup and composition
│   │       ├── features/         chat, sessions, models, generation,
│   │       │                     integrations, permissions, settings
│   │       ├── platform/         desktop, backend and preferences
│   │       ├── shell/            window presentation and sizing
│   │       ├── rich-text/        Markdown, code, links and images
│   │       ├── ui/               shared controls
│   │       ├── styles/           shared visual foundations
│   │       └── preview/          bootstrap, scenarios and fixtures
│   ├── scripts/                  native build, dev and packaging
│   └── tests/{unit,browser,desktop,fixtures,support}/
├── runtimes/generation/           worker, image, video, speech, media,
│                                 requirements, catalog and tests
├── tooling/{architecture,contracts,verification}/
└── docs/                         README hub and ten subject documents
```

The backend groups are `app`, `chat`, `agent`, `inference`, `models`, `generation`, `tools`, `permissions`, `integrations`, `media`, `sessions`, `settings`, `transport`, and `platform`.

`app` constructs services, closes them, and coordinates local inference memory with generation. `chat.Service` owns the reply mutex, cancellation, revision, send preparation and completion. `inference.Router` receives model/validation/completion operations from app and preserves lazy model-manager creation. Callback injection prevents the common inference types from importing adapters that themselves consume those types.

Generation model policy and assets live under `generation/models`, supervision under `generation/runtime`, and reusable transfer and atomic replacement under platform. Integration activity types remain at `integrations`; enabled catalog assembly lives in `integrations/catalog`, which can depend on the MCP/skill managers without creating a parent-package import cycle.

Renderer platform owns native presentation, backend connection and persisted preferences. Shell owns window behavior; ui owns shared controls; rich-text owns Markdown presentation. Features own their screens/state and adjacent styles. Generation has one app-scoped subscription. MCP and skill list, form, detail and request state have separate owners.

Python source lives under `runtimes/generation`, with image/video/speech/media packages and requirements. Build copies retain `runtime/generation` beside the binary. The maintained `preview.html` uses production components with an explicit fake bridge; it contains no copied application tree.

Features must not import app or transport. Platform must not import product workflows. Renderer features must not import app or preview. Policy checks enforce these directions. Small modules belonging to one operation are combined: chat lifecycle/state, send/preparation, reply/title, and app model coordination.

## Verification

Check dependency directions, request/event compatibility, shutdown order, and packed runtime assets. Follow the behavior-specific gates in [chat](chat.md), [tools](tools.md), [generation](generation.md), and [storage](storage.md). Full commands and isolation live in [development](development.md).
