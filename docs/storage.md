# Storage and saved resources

Feature stores own schema validation, recovery and state transitions. Filesystem primitives own writing mechanics. Source folder changes must preserve user profile locations, saved identities and formats.

## Profile layout

Paths below are relative to the selected Arx profile unless marked otherwise.

| Path | Owner and contents |
| --- | --- |
| `settings.json` | Versioned selected run/directory/default and per-chat permissions, idle/continuation preferences |
| `views.json` | UI view state, drafts/startup/sidebar preferences |
| `transcript.jsonl` | Append-only user/assistant/tool/usage chunks with byte offsets and timestamps |
| `summaries/<conversation-hash>.json` | Compaction checkpoint/coverage/digest |
| `attachments/<64-hex-id>.png` | Saved normalized image pixels, identified by content SHA-256 |
| `media/<32-hex-id>/` | Published output file and metadata.json |
| `mcps.json` / `skills.json` | Integration registrations and usage/activity |
| `skills/<id>/SKILL.md` | User/profile-created skill manifests and supporting files |
| `network-models/servers.json` | Saved server address/name/token-presence metadata; tokens live in Keychain |
| `generation/models/library.json` | Installed generation files and selected defaults |
| `generation/jobs/<id>.json` | Persisted job status/progress/output |
| `generation/engine/` | Isolated installed environments/readiness/logs |
| `keychain-account` | Preserved credential account identity when legacy profile migration requires it |
| Local model directory | GGUF library, locks, files, engine, runtime key/record/log; set through ARX_MODELS_DIR |

Electron currently defaults local models to the Arx app-data local-models directory; backend fallback uses the profile if no override is provided. Generation assets are profile-owned. Keep these distinctions.

## Code map

`sessions/storage` owns transcript append/load/index/history/titles/compaction. `settings` owns settings/views and policy persistence. `media/attachments` owns Go saved image normalization/read validation. `media/artifacts` owns staged generated outputs. Model/integration/job stores remain inside their respective owners.

Electron profile migration copies into a temporary sibling, preserves keychain identity and existing destination data, then renames. View-writes coalesces edits but preserves write order; it is not the canonical Go storage layer.

## Persistence decisions

Transcript append serializes a detached chunk, writes and fsyncs before indexing it. A failed append truncates back to the original offset. Recovery retains valid records, records damaged lines, and separates subsequent appends without destroying original bytes.

History/index reconstructs Unicode chunks, reasoning, tools, title, usage and pages. Usage-only entries retain provider accounting. Compaction checkpoints remain separate so summaries cannot overwrite full history.

Settings rejects damaged/future versions rather than overwriting them. Updates commit disk before replacing in-memory state. Settings and compaction sync the containing directory after replacement; other replacement stores sync the file.

Saved attachments are bounded PNGs with a content hash. Go reads validate digest, PNG dimensions and size. Electron retains chooser decoding and hands normalized pixels to the same Go store.

Generated artifacts have random saved IDs, supported formats, bounded nonempty output, metadata and size validation. Publication stages a directory and renames it. Artifact IDs and attachment content IDs are not interchangeable; preserve both schemes and saved links.

## Organization and ownership

`media/attachments` is the canonical saved PNG writer and reader. Electron retains native chooser decoding, the four-image limit and 32 MiB source limit. It normalizes to at most 2048 pixels and 2 MiB, writes a private nonce-named handoff file under profile/attachment-imports, and calls a main-only import operation. Go opens that bounded handoff root, validates the PNG and publishes its unchanged bytes under their SHA-256 ID. Electron deletes the handoff file after the request. Go tool normalization retains its 8 MiB source bound.

Reads go through Go's digest, PNG bounds and size validation. Existing saved PNG IDs remain usable. Large pixel payloads never enter the 1 MiB request frame, and renderer-visible request metadata does not expose handoff paths.

`platform/atomicfile.Replace` creates a same-directory 0600 temporary file, writes, fsyncs, closes and renames it, with optional directory fsync. Settings and compaction require directory fsync; other replacement stores require file fsync. Feature stores own serialization, schema validation, recovery, locks and in-memory rollback. Transcript append and artifact-directory publication remain specialized.

Source organization does not change profile schema versions, credentials, model IDs or saved references. Private profiles, model weights, downloads and runtime state are excluded from source/document archives.

## Verification

Tests cover legacy settings/views, concurrent and failed writes, unsupported versions, view batching, transcript restart/pages/recovery, Unicode offsets, titles, saved compaction, detached records, image digest/bounds, artifact format/size and model path ownership.

Canonical attachment tests include compatibility cases for existing PNG IDs, chooser/tool limits, source decoding, digest mismatch and protocol size. Atomic replacement tests cover private modes and failed-publication cleanup; feature tests cover in-memory rollback. Directory fsync remains an explicit stronger option.
