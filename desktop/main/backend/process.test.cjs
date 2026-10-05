const { test } = require("node:test");
const assert = require("node:assert/strict");
const { once } = require("node:events");
const path = require("node:path");
const fs = require("node:fs");
const os = require("node:os");

function profile(t) {
    const directory = fs.mkdtempSync(path.join(os.tmpdir(), "arx-process-"));
    t.after(() => fs.rmSync(directory, { recursive: true, force: true }));
    return directory;
}
const { Backend } = require("./process.cjs");

function waitFor(backend, state) {
    if (backend.status.state === state) return Promise.resolve();
    return new Promise((resolve, reject) => {
        const timer = setTimeout(() => {
            backend.off("status", listener);
            reject(new Error(`Expected ${state}`));
        }, 5000);
        const listener = (status) => {
            if (status.state !== state) return;
            clearTimeout(timer);
            backend.off("status", listener);
            resolve();
        };
        backend.on("status", listener);
    });
}

const binary = path.resolve(
    __dirname,
    "../../.build/backend",
    process.platform === "win32" ? "arx-desktop.exe" : "arx-desktop",
);

test("starts, handles requests, restarts and shuts down", async (t) => {
    const backend = new Backend(binary, profile(t));
    try {
        backend.start();
        await waitFor(backend, "ready");
        assert.deepEqual(await backend.request("status"), {
            version: "0.1.0",
            execution: "idle",
        });
        const firstPID = backend.child.pid;
        await backend.restart();
        await waitFor(backend, "ready");
        assert.notEqual(backend.child.pid, firstPID);
        await backend.stop();
        assert.equal(backend.child, null);
        assert.equal(backend.status.state, "stopped");
    } finally {
        await backend.stop();
    }
});

test("reports a crash and allows explicit reconnection", async (t) => {
    const backend = new Backend(binary, profile(t));
    try {
        backend.start();
        await waitFor(backend, "ready");
        const closed = once(backend.child, "close");
        backend.child.kill("SIGKILL");
        await closed;
        assert.equal(backend.status.state, "error");
        await assert.rejects(backend.request("status"), /not connected/);
        await backend.restart();
        await waitFor(backend, "ready");
    } finally {
        await backend.stop();
    }
});

test("reports a missing executable without crashing the desktop", async () => {
    const backend = new Backend(binary + "-missing");
    backend.start();
    await waitFor(backend, "error");
    await backend.stop();
    assert.equal(backend.child, null);
});
