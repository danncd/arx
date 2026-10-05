import { spawnSync } from "node:child_process";
import { fileURLToPath } from "node:url";
const desktop = fileURLToPath(new URL("../../desktop", import.meta.url));
const backend = fileURLToPath(new URL("../../backend", import.meta.url));
function run(command, args, cwd = desktop) {
    const result = spawnSync(command, args, {
        cwd,
        stdio: "inherit",
        env: process.env,
    });
    if (result.error) throw result.error;
    if (result.status !== 0) process.exit(result.status || 1);
}
run("npm", ["run", "check:architecture"]);
run("npm", ["run", "contracts:check"]);
const go = process.env.ARX_GO_BINARY || "go";
run(go, ["test", "-race", "-count=1", "./..."], backend);
run(go, ["vet", "./..."], backend);
run(
    process.env.ARX_PYTHON_BINARY || "python3",
    ["-B", "-m", "unittest", "discover", "-s", "../runtimes/generation/tests", "-v"],
    backend,
);
for (const script of [
    "test",
    "test:responses",
    "test:menus",
    "test:models",
    "test:safety",
    "test:generation",
    "test:desktop",
])
    run("npm", ["run", script]);
