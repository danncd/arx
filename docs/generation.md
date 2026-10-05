# Media generation

Generation resolves the selected installed model, validates its capabilities and sources, runs a persisted cancellable job through a supervised worker, then publishes an independently saved artifact.

## Code map

| Path | Responsibility |
| --- | --- |
| `backend/internal/generation/service.go` | Model/job/source/output orchestration |
| `generation/request.go` / `model.go` / `source.go` | Input bounds, runtime defaults, source validation/staging |
| `generation/jobs/` | Durable job records, progress, cancel/close/recovery |
| `generation/runtime/` | Python environment/install, request/events, watcher and recovery records |
| `backend/internal/generation/models/` | Trusted catalog, model library/defaults and downloads |
| `backend/internal/app/generation.go` | Lazy service, events and local inference memory handoff |
| `tools/generation_runner.go` / `generation/guidance.go` | Authorization and selected-model guidance |
| `runtimes/generation/` | Python adapters, pinned catalog, requirement sets and unit tests |
| `desktop/renderer/src/features/generation/` | Browser/rows/default settings/current state |
| `features/chat/tools/useGenerationProgress.ts` | Live job display in tool cards |

Short backend paths are under `backend/internal`; short renderer paths are under `desktop/renderer/src`.

## Request policy

Operations are image, video, speech and combine. Prompt text is bounded to 16,000 Unicode characters. Dimensions are 128–1024 in multiples of 32; default image size is 512×512 and video is 384×256. General video frame bounds and area/work limits are followed by tighter runtime/model bounds.

`source` and `sources` are exclusive. Image references preserve order. FLUX.2 Klein accepts up to four image-edit references; other current image-edit models accept one. Video sources require image-to-video capability. Saved inputs are resolved/staged without mutating their original copies.

Speech accepts supported model voices and speed bounds; Qwen3 speech currently accepts normal speed only. Combine resolves an existing saved MP4 and WAV and muxes them; it does not provide lip sync.

The trusted catalog has ten entries: five image, four speech, and one AnimateDiff video model. Wan CPP, FastMetal and LTX workers are implemented but have no selectable catalog entry in this catalog. Existing code alone is not evidence that a model is available in Settings.

## Job and memory lifecycle

One generation job can be active. Persist queued state before running; retain progress/detail and completion/failure/cancel state. Cancelling does not publish a new output. Shutdown waits for owned worker/installer cleanup. Restart marks saved queued/running jobs interrupted.

The application can unload a ready local chat model to release memory before generation, then restore it with bounded timeout unless closing. Keep this cross-feature coordination explicit; do not merge local and generation model state into a generic runtime manager.

Artifacts are staged, verified as PNG/WAV/MP4, then published under a durable saved ID. Tool results supply exact `arx-media` references; generated images display using those references, and optional inspection is separate. See [storage](storage.md).

## Worker installation and supervision

Current local generation requires Apple Silicon. Isolated environment selection follows operation/runtime: diffusion, speech, mflux, FastMetal or Wan CPP. Runtime install uses pinned inputs and readiness markers; normal inference workers set offline model-loading flags.

Worker stdin is one JSON request and stdout is newline JSON events. Progress, complete and error events are interpreted by Go; logs are separate from protocol output.

A separate Go watcher receives a parent-liveness pipe and startup barrier. Worker/installer work begins only after a recovery record is saved. Recovery verifies owner/process identity, start time, token, process group and command. Uncertain records remain visible errors and do not justify PID-only killing.

Preserve watcher, startup barrier and cleanup ownership as one cohesive implementation. Python can be busy inside native code, so Python-only parent monitoring is insufficient.

## Organization and ownership

Go generation models live under generation/models and engines under generation/runtime. Python source lives under runtimes/generation, with image/video/speech/media packages and requirements. Packaging preserves runtime/generation beside the backend binary and replaces stale generated copies before copying source.

Shared transfers use platform/download. CapabilitiesFor in generation/models/catalog.go is the Go policy table for steps, default/max frames, pixel limits, image references and fixed speed. Catalog responses expose this metadata. Python keeps defensive input checks.

The renderer's single generation provider merges library revisions and job timestamps, invalidates fetches on disconnect/restart and loads the new backend epoch. Tool cards select jobs by tool call from the same state as Settings/catalog/library.

Supervision keeps its persisted startup barrier and owner pipe together. The owner persists recovery records after Start and before releasing the worker. Only the writer end stays alive during execution. Recovery never treats uncertain process identity as permission to kill; records remain until process-group disappearance is confirmed. Close waits for owned worker and installer cleanup.

## Verification

Go tests cover job cancellation/recovery, source order/limits, runtime-specific bounds, saved output, download remaining space, trusted paths, watcher owner death, unrelated PID protection and start-before-work barrier.

Python tests cover FLUX references and Wan command/progress/failure behavior using fakes. Protocol and generation state tests cover structural and ordering behavior. Real generation requires optional installed models and is separate from structural tests.
