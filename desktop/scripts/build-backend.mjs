import { spawn } from "node:child_process";
import { mkdir, cp, rm } from "node:fs/promises";
import { fileURLToPath } from "node:url";

const output = new URL("../.build/backend/", import.meta.url);
await mkdir(output, { recursive: true });
const binary = fileURLToPath(
    new URL(process.platform === "win32" ? "arx-desktop.exe" : "arx-desktop", output),
);
const child = spawn(
    process.env.ARX_GO_BINARY || "go",
    ["build", "-trimpath", "-o", binary, "./cmd/arx-desktop"],
    {
        cwd: fileURLToPath(new URL("../../backend/", import.meta.url)),
        stdio: "inherit",
    },
);
await new Promise((resolve, reject) => {
    child.once("error", () =>
        reject(
            new Error(
                "Go is required to build the backend. Set ARX_GO_BINARY if it is not on PATH.",
            ),
        ),
    );
    child.once("exit", (code) =>
        code === 0 ? resolve() : reject(new Error("Backend build failed")),
    );
});
await import("./build-keychain.mjs");

await rm(new URL("runtime/", output), { recursive: true, force: true });
await cp(new URL("../../runtimes/", import.meta.url), new URL("runtime/", output), {
    recursive: true,
});
