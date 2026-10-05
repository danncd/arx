import { test } from "node:test";
import assert from "node:assert/strict";
import { messagesFromChunks } from "../../../renderer/src/features/sessions/history.ts";

const chunk = (values) => ({
    conversation: "session",
    id: "reply",
    role: "assistant",
    offset: 0,
    text: "",
    reasoning_offset: 0,
    reasoning: "",
    tools: null,
    usage: null,
    status: "done",
    failed: false,
    reason: "",
    at: "2026-09-26T12:00:00Z",
    ...values,
});

test("restores Unicode streams and the latest tool result and usage", () => {
    const tool = { id: "tool", name: "files", status: "running", summary: "Read" };
    const messages = messagesFromChunks([
        chunk({ text: "Hello 🌱", reasoning: "考える", tools: [tool] }),
        chunk({
            offset: Buffer.byteLength("Hello 🌱"),
            reasoning_offset: Buffer.byteLength("考える"),
            text: " Danny",
            tools: [{ ...tool, status: "done", result: "file contents" }],
            usage: { input: 30, output: 12, cached: 10, reasoning: 4 },
        }),
    ]);
    assert.equal(messages.length, 1);
    assert.equal(messages[0].text, "Hello 🌱 Danny");
    assert.equal(messages[0].reasoning, "考える");
    assert.equal(messages[0].tools.length, 1);
    assert.equal(messages[0].tools[0].result, "file contents");
    assert.equal(messages[0].usage.output, 12);
});

test("refuses a damaged transcript instead of silently joining missing text", () => {
    assert.throws(
        () =>
            messagesFromChunks([
                chunk({ text: "Hello" }),
                chunk({ offset: 8, text: "world" }),
            ]),
        /incomplete/,
    );
});

test("reads transcripts written before reasoning was supported", () => {
    const messages = messagesFromChunks([
        { conversation: "session", id: "user", role: "user", offset: 0, text: "Hello" },
    ]);
    assert.equal(messages[0].text, "Hello");
    assert.equal(messages[0].reasoning, "");
});

test("live snapshots cannot overwrite a newer history response", async () => {
    const { mergeMessages } =
        await import("../../../renderer/src/features/sessions/merge.ts");
    const older = {
        id: "reply",
        text: "partial",
        at: "2026-09-26T12:00:00.123Z",
        status: "running",
    };
    const newer = {
        ...older,
        text: "partial complete",
        at: "2026-09-26T12:00:00.123456Z",
        status: "done",
    };
    assert.deepEqual(mergeMessages([newer], [older]), [newer]);
    assert.deepEqual(mergeMessages([older], [newer]), [newer]);
});
