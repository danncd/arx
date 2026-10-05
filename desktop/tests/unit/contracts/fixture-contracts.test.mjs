import { test } from "node:test";
import assert from "node:assert/strict";
import fixture from "../../fixtures/menu-backend.cjs";
test("browser fixture supplies current startup contracts and rejects unexpected methods", async () => {
    globalThis.window = {};
    try {
        fixture();
        const request = window.arxDesktop.request;
        assert.ok(Array.isArray(await request("network.state")));
        const generation = await request("generation.state");
        assert.ok(Array.isArray(generation.library.models));
        assert.ok(Array.isArray(generation.jobs));
        assert.ok(Array.isArray(generation.catalog));
        await assert.rejects(request("unexpected.method"), /Unexpected fixture request/);
    } finally {
        delete globalThis.window;
    }
});
