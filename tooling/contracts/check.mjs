import { readFile, writeFile } from "node:fs/promises";
import { spawnSync } from "node:child_process";
import { fileURLToPath } from "node:url";
const backend = fileURLToPath(new URL("../../backend", import.meta.url));
if (process.argv.includes("--tidy")) {
    const result = spawnSync(process.env.ARX_GO_BINARY || "go", ["mod", "tidy"], {
        cwd: backend,
        stdio: "inherit",
    });
    if (result.error) throw result.error;
    if (result.status) process.exit(result.status);
    const file = new URL("../../backend/go.mod", import.meta.url);
    const source = await readFile(file, "utf8");
    await writeFile(file, source.replace(/ \/\/ indirect/g, ""));
} else {
    const args = [
        "run",
        "./cmd/arx-contracts",
        ...(process.argv.includes("--write") ? [] : ["--check"]),
    ];
    const result = spawnSync(process.env.ARX_GO_BINARY || "go", args, {
        cwd: backend,
        stdio: "inherit",
    });
    if (result.error) throw result.error;
    process.exitCode = result.status || 0;
}
