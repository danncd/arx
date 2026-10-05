import { spawn } from "node:child_process";
import { stat } from "node:fs/promises";
import { fileURLToPath } from "node:url";

if (process.platform === "darwin") {
    const source = new URL(
        "../../backend/internal/platform/keychain/helper/main.swift",
        import.meta.url,
    );
    const binary = new URL("../.build/backend/arx-keychain", import.meta.url);
    const built = await stat(binary).catch(() => null);
    if (!built || built.mtimeMs < (await stat(source)).mtimeMs) {
        const child = spawn(
            "/usr/bin/xcrun",
            ["swiftc", "-O", fileURLToPath(source), "-o", fileURLToPath(binary)],
            { stdio: "inherit" },
        );
        await new Promise((resolve, reject) => {
            child.once("error", reject);
            child.once("exit", (code) =>
                code === 0
                    ? resolve()
                    : reject(new Error("Keychain helper build failed")),
            );
        });
    }
}
