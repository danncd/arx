# Model management

Chat model capabilities and lifecycle differ from generation model assets. The UI can present a combined library, while backend ownership stays explicit.

## Code map

| Path under `backend/internal` | Responsibility |
| --- | --- |
| `models/info.go` | Context/output/tools/vision/thinking capability vocabulary |
| `inference/deepseek/connection.go` | Cloud discovery, connection, key validation and saved selection |
| `models/local/` | Library, import/download, load/unload, holds and idle policy |
| `models/local/catalog/` | Hugging Face search/pagination, shards/variants/projectors |
| `models/local/library/` | GGUF metadata, managed/imported paths, files and persistence |
| `models/local/hardware/` | Hardware fit and local context sizing |
| `models/local/runtime/` | Isolated llama server installation, process, readiness and recovery |
| `platform/download/` | Existing shared verified/resumable transfer primitive |
| `models/network/` | LM Studio server registration, scan/discovery and metadata cache |
| `generation/models/` | Pinned generation catalog/library/defaults/downloads |

Renderer owners are `features/models` and `features/generation`. Provider completion behavior is in [providers](providers.md); generation execution is in [generation](generation.md).

## Local model decisions

The library is exclusive to an Arx instance through a filesystem lock. GGUF inspection reads metadata without loading weights. Helper/projector files are not independent chat models. Variants group complete shards and choose the matching projector.

Managed paths reject traversal/symlink escapes. Imported models preserve original source ownership; removal without file deletion preserves input files. Explicit imported-file deletion refuses shared or escaping files.

Loading installs/starts the local runtime with loopback binding and a temporary API-key file. Runtime properties determine active context/tools/vision/thinking; catalog estimates do not override observed runtime capabilities.

Holds defer idle unload during a reply. Idle timeout is configurable to 5, 10, or 15 minutes. A local selection can be installed and require loading; send orchestration loads it before starting the reply. Stop the reply before load/unload/remove mutations.

Context sizing considers unified memory and KV-cache metadata. Active context and maximum training context are different fields. Unsupported/missing tool or vision support is reported rather than inferred.

## Downloads

Transfer verifies declared size and checksum when available, resumes bounded partials with validated Content-Range, and checks free space before writing. If a server ignores a range and returns a full response, check required space again before truncating the partial.

Remaining-space calculations account for verified complete files and resumable partials. A corrupt complete partial cannot be treated as a valid zero-byte remaining download. Feature policy supplies additional files and disk margin.

Local library restart restores interrupted downloads as paused and loaded/loading entries as installed. Do not delete arbitrary external source files to reconcile that state.

## Network models

Normalize an HTTP(S) server address, optionally ending in /v1. Discovery uses LM Studio's /api/v1/models metadata, filters to LLMs, and maps loaded instances and active context. A network model is selectable only when loaded/connected; local GGUF models use a different load lifecycle.

Tokens are stored in Keychain, not the server JSON. Server IDs derive from the normalized address. Metadata lookups share in-flight work and cache for 30 seconds; explicit state refresh obtains current context/loaded status. The UI also polls saved server state.

## Organization and ownership

Local GGUF and network models remain distinct lifecycle owners. DeepSeek connection lives beside inference; generation catalog/library/defaults live under `generation/models`.

`platform/download` supplies verified/resumable transfer to both model families and installers. Domain-specific disk margins, schema validation and deletion policy stay with the owning library. Transfer checks required space again when a server restarts a range; corrupt complete partials are repaired rather than trusted.

The renderer's available-model selector combines discovered cloud models, connected loaded network instances, and installed/loading local models. Picker labels and availability consume normalized metadata. Backend validation and existing model IDs remain authoritative.

## Verification

Tests cover shard/projector selection, GGUF metadata/cache sizing, import role rejection, managed path boundaries, deletion ownership, library locking, idle/holds, runtime failures/recovery, checksum/range/space handling, network loaded context and concurrent metadata.

Browser tests cover sorting/filtering/pagination/stale searches and explicit deletion. Optional real GGUF, projector, and network checks require supplied inputs.
