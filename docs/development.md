# Development

The project declares Go 1.26 and uses Node 24. Go must be on PATH or selected through `ARX_GO_BINARY`. Install desktop dependencies with `npm ci` in `desktop/`. Dependencies and runtime versions are recorded in the module/lockfile and generation requirements; the module and lockfile remain authoritative.

## Commands

Run these from `desktop/`:

| Command | Purpose |
| --- | --- |
| `npm run dev` | Build the backend, start Vite and Electron |
| `npm run dev:web` | Renderer dev server; the app requires its native bridge |
| `npm run dev:preview` | Maintained production-component preview with scenario data |
| `npm run check` | TypeScript and main/preload/script/tooling syntax |
| `npm run contracts` / `contracts:check` | Generate wire types/method policy or verify drift |
| `npm run check:architecture` | Dependency direction, syntax/comment and docs policy |
| `npm run deps:tidy` | Tidy the Go module and normalize tool annotations |
| `npm run test` | Fresh backend build plus subprocess/renderer unit tests |
| `npm run test:responses` | Streaming response/browser regression flow |
| `npm run test:menus` | Menus, delayed saves, focus and anchors |
| `npm run test:models` | Local model UI, search and deletion behavior |
| `npm run test:generation` | Shared generation state, stale events, restart epochs, and maintained preview |
| `npm run test:safety` | Conversation resource policy and image stability |
| `npm run test:desktop` | Fresh build plus deterministic Electron flows |
| `npm run test:local` | Optional real local model flow |
| `npm run build` | Go/Swift/runtime assets, checks, renderer output |
| `npm run package:mac` | Mac candidate bundle and signature verification |
| `npm run verify` | Go race/vet, Python, Node, browser, and deterministic Electron tests |

`verify` begins with architecture and deterministic contract checks, then uses `ARX_PYTHON_BINARY` or python3. The ordinary deterministic verification does not require downloaded model weights or paid completions.

## Profiles and output isolation

| Variable | Meaning |
| --- | --- |
| `ARX_DEV_PROFILE` | Absolute explicit user/session profile for development or packaged launches |
| `ARX_MODELS_DIR` | Local GGUF library/runtime location; isolate alongside the test profile |
| `ARX_PACKAGE_OUT` | Candidate package destination |
| `ARX_GO_BINARY` | Go executable |
| `ARX_PYTHON_BINARY` | Python executable for verification |
| `ARX_TEST_MODEL` / `ARX_TEST_PROJECTOR` | Optional real GGUF/vision input |
| `ARX_TEST_NETWORK_URL` | Optional real network model server |
| `ARX_CODEX_BINARY` | Optional external Codex compatibility executable |
| `ARX_MEMO_MCP_PATH` | Explicit Memo integration executable |

Normal app identity, production/development profile names, keychain account identity, saved model IDs, and transcript formats are compatibility boundaries. Test harnesses use disposable directories. Keep the installed app until a candidate package is checked.

## Build assets

`scripts/build-backend.mjs` builds `cmd/arx-desktop` into `desktop/.build/backend`, compiles the Swift keychain helper on macOS, and copies `runtimes/` into the backend output. Vite emits `desktop/.build/renderer`. Packaging stages main, preload, generated contracts and renderer and adds backend assets as resources.

Python source-copy and test paths use runtimes/generation; installed workers still resolve runtime/generation. BrowserWindow and package staging use desktop/preload/index.cjs. Verification tooling lives under tooling/verification; desktop npm commands remain the entry points.

## Documentation and source policy

Developer explanations and decisions live in the eleven documents linked from [README](README.md). Archive earlier readmes/instructions/history externally. Maintained prose should not appear in source comments.

The root README is the project introduction, setup, and checks. It links here for detailed documentation; other Markdown files belong in docs.

Use syntax-aware cleanup; preserve strings, runtime prompt/schema content, fixture text, and upstream legal notices. Use npm run deps:tidy to normalize go.mod tool annotations after dependency maintenance. Generated contracts must also be comment-free.

Policy checks cover imports, comments, docs location/links and contract drift. Fresh packaging checks runtime assets and native bridge files. These checks are policy tooling; they should not introduce a new test framework or repository-wide build abstraction.

## Verification decisions

Run the affected checks once, then expand to full verification before accepting a migration. Preserve meaningful tests for cancellation, stale events, identity, persistence, and recovery. A typecheck does not prove a safe process or source-folder move.

Optional checks may skip when their explicit inputs are missing. Record a skip instead of claiming real inference or compatibility passed. Use a freshly built backend for Electron integration so results can be attributed to source.

Use a fresh npm run verify and npm run build for acceptance. Candidate packages must include the new preload/contracts, Swift helper, Go binary and complete Python package/requirements tree. Optional paid providers, real downloaded models and external Codex checks remain separately enabled.
