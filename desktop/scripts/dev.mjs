import "./build-backend.mjs";
import { spawn } from "node:child_process";
import { createRequire } from "node:module";
import { createServer } from "vite";

const require = createRequire(import.meta.url);
const server = await createServer({ configFile: "renderer/vite.config.ts" });
await server.listen();
server.printUrls();
const environment = { ...process.env, ARX_DEV_URL: "http://127.0.0.1:3006/" };
delete environment.ELECTRON_RUN_AS_NODE;
const desktop = spawn(require("electron"), ["."], { stdio: "inherit", env: environment });
let closing = false;
async function close(code = 0) {
    if (closing) return;
    closing = true;
    desktop.kill();
    await server.close();
    process.exit(code);
}
desktop.on("exit", (code) => void close(code ?? 0));
desktop.on("error", (error) => {
    console.error(error);
    void close(1);
});
process.on("SIGINT", () => void close());
process.on("SIGTERM", () => void close());
