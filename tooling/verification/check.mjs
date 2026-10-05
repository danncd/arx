import { readdir } from "node:fs/promises";
import { spawnSync } from "node:child_process";
import { fileURLToPath } from "node:url";

for (const directory of ["main", "preload", "scripts", "contracts", "../tooling"]) {
    const root = new URL(`../../desktop/${directory}/`, import.meta.url);
    for (const file of await readdir(root, { recursive: true })) {
        if (!/\.(cjs|mjs)$/.test(file)) continue;
        const result = spawnSync(
            process.execPath,
            ["--check", fileURLToPath(new URL(file, root))],
            { stdio: "inherit" },
        );
        if (result.error) throw result.error;
        if (result.status !== 0) process.exit(result.status ?? 1);
    }
}
