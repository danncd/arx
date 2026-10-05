# Arx documentation

Arx is an Electron/React desktop assistant with a Go service and supervised Python generation workers.

These documents describe the maintained implementation and its decisions. Code maps use accepted source locations. Historical audits, prototypes, repository skills, and earlier documentation are preserved outside Arx.

## Documents

| Document | Brief |
| --- | --- |
| [Architecture](architecture.md) | Processes, subsystem ownership, dependency rules, composition, and repository layout |
| [Development](development.md) | Setup, commands, isolated profiles, building, packaging, checks, and documentation policy |
| [Desktop](desktop.md) | Electron/window/profile, bridge capabilities, renderer composition, state and presentation |
| [Chat](chat.md) | Send/stop, agent rounds, history, context, compaction, continuation, and response rendering |
| [Models](models.md) | Local GGUF libraries, capability/context information, downloads, idling, and network discovery |
| [Providers](providers.md) | Model identity, completion routing, streaming, provider-specific images/thinking/errors |
| [Tools](tools.md) | Tool definitions, preparation, approval, permission modes, filesystem/web/Bash execution |
| [Integrations](integrations.md) | MCP registration/sessions, skills/manifests, activation, activity, and Settings flows |
| [Generation](generation.md) | Model defaults, requests, media jobs, memory handoff, worker installation and recovery |
| [Storage](storage.md) | Profile layout, settings/history/summaries, saved images/artifacts, durable updates |

## Reading routes

- New to Arx: architecture → desktop → chat.
- Adding or repairing a model: models → providers → development.
- Changing an operation or approval: tools → integrations → storage.
- Working on media creation: generation → models → storage.
- Diagnosing stale UI or a restart: desktop → chat → storage.

Decisions are recorded in their owning document with their reason and verification. Link to another owner instead of duplicating its rules. Explain current behavior directly; keep obsolete plans, audits, imported skills, and progress diaries in the external archive.

Detailed developer documentation lives here. The root [README](../README.md) gives a short project introduction and setup. Source identifiers, state, and tests express implementation behavior without explanatory comments. Executable prompts, schemas, user-imported skill manifests, and legal notices retain their runtime or asset ownership; they are documented here.
