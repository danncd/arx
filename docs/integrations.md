# MCPs and skills

MCPs provide operations from configured stdio executables. Skills provide workflows loaded into the current reply. Both retain durable registrations and bounded usage/activity, while permissions remain authoritative.

## Code map

| Path under `backend/internal` | Responsibility |
| --- | --- |
| `integrations/mcp/manager.go` | Registration, snapshots, discovery/actions/calls/activity |
| `integrations/mcp/connection.go` | Retained stdio session, process lifetime, bounded pagination |
| `integrations/mcp/types.go` | Executable configuration, schemas and reviewed read-only tools |
| `integrations/skills/manager.go` | Create/import/edit/remove/read/load, registrations and usage |
| `integrations/skills/files.go` / `types.go` | Manifest validation and bounded reference filesystem access |
| `integrations/skills/memo.go` | Product's default Memo workflow manifest |
| `integrations/usage.go` | Shared activity and detached snapshots |
| `tools/runner/mcp.go` / `skills.go` | Runtime tool schema, permission and per-reply activation |
| `app/integrations.go` | Idle edits and probes; catalog guidance is assembled in integrations/catalog |

Renderer owners are `features/integrations/mcp`, `features/integrations/skills` and `features/integrations/shared`.

## MCP behavior and decisions

Configuration names an executable, argument list and environment values. Execution is direct without a shell. Test connection probes exact unsaved configuration without registering it. Reconnect replaces the live session.

Discovery follows bounded pagination and validates distinct names/schema size. Calls use server/name/argument objects and validate input schemas locally; remote schema references are not fetched.

Sessions are retained across calls and closed on configuration changes, disable/remove, reconnect, failure/cancellation or shutdown. Calls do not automatically retry writes. Process-group cleanup remains part of connection ownership.

Server read-only hints are informative. The user-reviewed allowlist is the authority for selected-folders mode, and unknown calls require review. Review includes server, tool and exact arguments. Configured executables are trusted code and are not confined by the file-tool folder boundary.

`mcps.json` stores registrations and activity. Server counts include executions and denials, excluding discovery; tool counts include executions. Recent activity is capped at 100 entries per server. Environment secrets and raw server logs do not enter activity.

Memo is seeded on first use using its override or installed/project executable candidates. Removing Memo persists across restart. HTTP/OAuth/resource/prompt discovery are outside this stdio implementation.

## Skill behavior and decisions

Import a folder with `SKILL.md`, or create one in Settings. YAML front matter provides a bounded lowercase name and description; the body is workflow guidance. Invalid registrations remain visible and editable.

The model sees enabled metadata first, then uses skills list/load/read. Loading adds the instructions to per-reply system guidance each round; compaction cannot discard them. Usage increments once on the first successful load per reply.

Explicit-only activation requires an invocation in the current user message. Historical instructions do not authorize it. Disabled skills or missing/disabled MCP dependencies cannot load.

References are regular bounded text opened through the registered root. Reject absolute/traversing paths, escaping symlinks, binary files, and files over 64 KiB. Manifests are also bounded to 64 KiB. Imported manifests are edited only through the explicit Settings operation.

Removing registration preserves source files. Profile-created skill folders are user/product data. The default Memo manifest is executable runtime guidance and remains product functionality.

## Organization and ownership

MCP and skill managers live under `integrations/mcp` and `integrations/skills`; parent usage.go owns shared activity types and detached snapshots. Enabled metadata/guidance assembly lives under `integrations/catalog`. Generic file replacement lives in `platform/atomicfile`; managers retain validation, rollback and lifetime ownership.

Renderer MCP and skill features have dedicated list, form, detail and request-state modules. Shared usage/activity/tabs/removal presentation stays together. Mutations refresh their owning dataset. Reading a skill reference, loading detail or validating a manifest does not refresh both datasets. Skills consume MCP metadata explicitly for dependency navigation.

Repository skill references and copied prototypes are preserved in the external archive. Imported profile skills, the default Memo workflow and MCP/skill runtime support remain active product functionality.

## Verification

Go tests cover discovery pagination, detached snapshots, call/session reuse, cancellation, probe exactness, activity persistence, read-only hint distrust, exact approval input, explicit-only activation, once-per-reply usage, manifest edits and reference boundaries.

The real Electron integration flow uses disposable fixtures and does not edit the user's Memo content. Test editor repair, list/detail navigation, dependencies, removal persistence and disable behavior in the maintained feature screens.
