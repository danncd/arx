import { test } from "node:test";
import assert from "node:assert/strict";
import { formatTokens } from "../../../renderer/src/features/chat/context/contextReading.ts";

test("context token labels cross from K to M at the rounded boundary", () => {
    assert.equal(formatTokens(999_499), "999K");
    assert.equal(formatTokens(999_500), "1M");
    assert.equal(formatTokens(1_000_000), "1M");
});
