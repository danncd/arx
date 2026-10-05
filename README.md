# Arx

A desktop coding agent for macOS built with Go, Electron, React, and TypeScript.

Use DeepSeek, local GGUF models, or LM Studio. Arx can read and edit files, run commands, search the web, and create images, video, and speech locally. MCP servers and skills can be added in Settings.

Tool calls follow the permission mode you select.

## Running locally

Use Node.js 24, Go 1.26, and the Xcode command line tools. Go must be on PATH.

```sh
cd desktop
npm ci
npm run dev
```

Open Settings → Connections to add a DeepSeek key or choose a local or network model.

## Building the app

From `desktop/`:

```sh
npm run package:mac
```

The app is written to the repository's `dist/` folder.

## Project structure

```text
backend/    Chat, models, tools, permissions, and storage
desktop/    Electron app and React interface
runtimes/   Python workers for media generation
tooling/    Contract generation and project checks
docs/       Architecture and feature documentation
```

## Checks

From `desktop/`:

```sh
npm run check
npm run verify
```

See [docs/README.md](docs/README.md) for the full documentation.
