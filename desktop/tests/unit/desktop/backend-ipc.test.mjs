import { test } from "node:test";
import assert from "node:assert/strict";
import { EventEmitter } from "node:events";
import fs from "node:fs";
import vm from "node:vm";
import { createRequire } from "node:module";

const require = createRequire(import.meta.url);

test("network requests reach the backend and status listeners remain valid", async () => {
    const handlers = new Map();
    const module = { exports: {} };
    vm.runInNewContext(
        fs.readFileSync(
            new URL("../../../main/ipc/backend.cjs", import.meta.url),
            "utf8",
        ),
        {
            module,
            require: (name) =>
                name === "../../contracts/methods.generated.cjs"
                    ? require("../../../contracts/methods.generated.cjs")
                    : {
                          ipcMain: {
                              handle: (name, handler) => handlers.set(name, handler),
                          },
                      },
        },
    );
    const backend = new EventEmitter();
    const calls = [];
    backend.request = async (...args) => {
        calls.push(args);
        return [];
    };
    module.exports.registerBackend(
        backend,
        () => true,
        () => [],
    );
    for (const method of [
        "network.state",
        "network.scan",
        "network.cancel",
        "network.connect",
        "network.remove",
    ]) {
        await handlers.get("arx:backend-request")({}, method, {});
        assert.equal(calls.at(-1)[0], method);
        assert.equal(calls.at(-1)[2], 35000);
    }
    assert.equal(backend.listenerCount("status"), 1);
    backend.emit("status", { state: "ready" });
});
