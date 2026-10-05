import { test } from "node:test";
import assert from "node:assert/strict";
import {
    formatUsage,
    cacheHitRate,
} from "../../../renderer/src/features/chat/usage/format.ts";

test("session tokens use compact k/M labels, including billion-size totals", () => {
    for (const [count, text] of [
        [0, "0"],
        [900, "900"],
        [1000, "1k"],
        [1250, "1.3k"],
        [999950, "1M"],
        [1000000, "1M"],
        [1301000000, "1301M"],
    ]) {
        assert.equal(formatUsage(count), text);
    }
});

test("cache hit rate uses all input tokens and handles empty sessions", () => {
    assert.equal(cacheHitRate(800, 600), "75%");
    assert.equal(cacheHitRate(1000, 600), "60%");
    assert.equal(cacheHitRate(300, 100), "33.3%");
    assert.equal(cacheHitRate(1000, 0), "0%");
    assert.equal(cacheHitRate(0, 0), "—");
});

test("a partial cache hit never rounds up to 100 percent", () => {
    assert.equal(cacheHitRate(1000000, 999999), "<100%");
    assert.equal(cacheHitRate(1000000, 1000000), "100%");
});

test("generation speed omits missing timing and formats measured tokens per second", async () => {
    const { formatGenerationSpeed } =
        await import("../../../renderer/src/features/chat/usage/format.ts");
    assert.equal(formatGenerationSpeed(), "");
    assert.equal(formatGenerationSpeed({ tokens: 10, seconds: 0 }), "");
    assert.equal(formatGenerationSpeed({ tokens: 129, seconds: 2 }), "64.5 t/s");
});
