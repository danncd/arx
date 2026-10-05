import { test } from "node:test";
import assert from "node:assert/strict";
import { responseTurns } from "../../../renderer/src/features/chat/response/turns.ts";
import { responseGroups } from "../../../renderer/src/features/chat/response/responseGroups.ts";
import {
    toolStatus,
    toolOutput,
    groupCalls,
} from "../../../renderer/src/features/chat/tools/toolPresentation.ts";

const bytes = (text) => Buffer.byteLength(text);
const call = (id, text, reasoning, name = "files") => ({
    id,
    name,
    offset: bytes(text),
    reasoning_offset: bytes(reasoning),
    arguments: '{"operation":"read","path":"notes.txt"}',
    status: "done",
    result: "Read result",
    failed: false,
});

test("renders provider rounds as one turn with tools between the text they interrupted", () => {
    const messages = [
        { id: "u", role: "user", text: "Read and check" },
        {
            id: "a",
            role: "assistant",
            text: "Before 🌱",
            reasoning: "First thought",
            tools: [call("read", "Before 🌱", "First thought")],
            at: "2026-09-26T12:00:00Z",
        },
        {
            id: "b",
            role: "assistant",
            text: "After reading",
            reasoning: "Second thought",
            tools: [call("check", "After reading", "Second thought", "bash")],
            at: "2026-09-26T12:00:01Z",
        },
        {
            id: "c",
            role: "assistant",
            text: "Final answer",
            reasoning: "",
            status: "done",
            at: "2026-09-26T12:00:02Z",
        },
    ];
    const original = structuredClone(messages);
    const turns = responseTurns(messages);
    assert.equal(turns.length, 2);
    assert.deepEqual(turns[1].ids, ["a", "b", "c"]);
    const response = responseGroups(turns[1].message);
    assert.deepEqual(
        response.groups.map((group) => [
            group.text.trim(),
            group.thought.trim(),
            group.tools.map((tool) => tool.id),
        ]),
        [
            ["Before 🌱", "First thought", ["read"]],
            ["After reading", "Second thought", ["check"]],
        ],
    );
    assert.equal(response.text.trim(), "Final answer");
    assert.deepEqual(messages, original);
});

test("groups adjacent calls without moving them above earlier text", () => {
    const response = responseGroups({
        text: "Intro 🌱\nFinish",
        reasoning: "",
        tools: [call("a", "Intro 🌱", ""), call("b", "Intro 🌱", "")],
    });
    assert.equal(response.groups.length, 1);
    assert.equal(response.groups[0].text, "Intro 🌱");
    assert.equal(response.groups[0].tools.length, 2);
    assert.equal(response.text, "\nFinish");
});

test("failed and interrupted calls never display success", () => {
    assert.equal(toolStatus([{ status: "done", failed: true }], false).label, "Failed");
    assert.equal(
        toolStatus([{ status: "running", failed: false }], false).label,
        "Stopped",
    );
    assert.equal(toolStatus([{ status: "running", failed: false }], true).active, true);
    assert.equal(
        toolOutput(
            { result: '{"text":"Readable result","failed":false}', status: "done" },
            false,
        ),
        "Readable result",
    );
});

test("incoming image calls stay grouped while their operation is streaming", () => {
    const first = {
        ...call("image-1", "", "", "web"),
        arguments: '{"operation":"image","url":"https://example.com/1.png"}',
    };
    for (const argumentsText of ["", '{"operation":', '{"operation":"image","url":']) {
        const incoming = {
            ...first,
            id: "image-2",
            status: "preparing",
            arguments: argumentsText,
        };
        assert.deepEqual(groupCalls([first, incoming]), [[first, incoming]]);
    }
    const search = {
        ...first,
        id: "search",
        status: "preparing",
        arguments: '{"operation":"search","query":',
    };
    assert.deepEqual(groupCalls([first, search]), [[first], [search]]);
    const files = {
        ...first,
        id: "files",
        name: "files",
        status: "preparing",
        arguments: "",
    };
    assert.deepEqual(groupCalls([first, files]), [[first], [files]]);
});

test("turn speed combines decode intervals without tool wait time", () => {
    const messages = [
        {
            id: "one",
            role: "assistant",
            text: "",
            at: "2026-09-27T12:00:00Z",
            usage: { input: 20, output: 101, generation: { tokens: 100, seconds: 2 } },
        },
        {
            id: "two",
            role: "assistant",
            text: "Done",
            at: "2026-09-27T12:01:00Z",
            usage: { input: 40, output: 51, generation: { tokens: 50, seconds: 1 } },
        },
    ];
    assert.deepEqual(responseTurns(messages)[0].message.usage.generation, {
        tokens: 150,
        seconds: 3,
    });
    delete messages[0].usage.generation;
    assert.equal(responseTurns(messages)[0].message.usage.generation, undefined);
});
