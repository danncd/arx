const { test } = require("node:test");
const assert = require("node:assert/strict");
const { ViewWrites } = require("./view-writes.cjs");

test("rapid edits persist their latest values in one write", async () => {
    const writes = [];
    const queue = new ViewWrites(async (values) => writes.push(values));
    const pending = [
        queue.set("draft:a", "h"),
        queue.set("draft:a", "hello"),
        queue.set("draft:b", "world"),
    ];
    await queue.flush();
    await Promise.all(pending);
    assert.deepEqual(writes, [{ "draft:a": "hello", "draft:b": "world" }]);
});

test("edits during a write remain ordered and flush before close", async () => {
    const writes = [];
    let release;
    const gate = new Promise((resolve) => {
        release = resolve;
    });
    const queue = new ViewWrites(async (values) => {
        await gate;
        writes.push(values);
    });
    const first = queue.set("draft:a", "first");
    const saving = queue.flush();
    const second = queue.set("draft:a", "second");
    const closing = queue.flush();
    release();
    await Promise.all([first, second, saving, closing]);
    assert.deepEqual(writes, [{ "draft:a": "first" }, { "draft:a": "second" }]);
});

test("a failed save is reported without blocking subsequent requests", async () => {
    let failing = true;
    const queue = new ViewWrites(async () => {
        if (failing) throw Error("disk full");
    });
    const saved = assert.rejects(queue.set("draft:a", "first"), /disk full/);
    await assert.rejects(queue.flush(), /disk full/);
    await saved;
    await queue.flush();
    failing = false;
    const next = queue.set("draft:a", "retry");
    await queue.flush();
    await next;
});
