import { test } from "node:test";
import assert from "node:assert/strict";
import { webSources } from "../../../renderer/src/features/chat/tools/webSources.ts";
import {
    groupLabel,
    toolLabel,
} from "../../../renderer/src/features/chat/tools/toolPresentation.ts";

const tool = (operation, result, overrides = {}) => ({
    id: "web",
    name: "web",
    status: "done",
    failed: false,
    arguments: JSON.stringify({
        operation,
        url: "https://www.example.com/page",
        query: "example",
    }),
    result: JSON.stringify(result),
    ...overrides,
});

test("website rows retain individual search excerpts and reject unsafe addresses", () => {
    const sources = webSources(
        tool("search", {
            sources: [
                {
                    url: "https://example.com/page",
                    title: "Example",
                    snippet: "Search excerpt",
                },
                { url: "javascript:alert(1)", title: "Invalid" },
                { url: "https://user:password@example.com", title: "Credentials" },
                null,
            ],
        }),
    );
    assert.equal(sources.length, 1);
    assert.equal(sources[0].domain, "example.com");
    assert.equal(sources[0].snippet, "Search excerpt");
    assert.equal(sources[0].text, "");
});

test("fetch rows show retained content with an accurate truncation state", () => {
    const result = {
        text: "Source: https://www.example.com/page\n\nPage body",
        truncated: false,
    };
    const fetch = tool("fetch", result);
    assert.equal(webSources(fetch)[0].text, "Page body");
    assert.equal(webSources(fetch)[0].complete, true);
    assert.equal(
        webSources(tool("fetch", { ...result, truncated: true }))[0].complete,
        false,
    );
    assert.equal(
        webSources(tool("fetch", { ...result, truncated: undefined }))[0].complete,
        false,
    );
    assert.equal(toolLabel(fetch).target, "example.com");
});

test("pending, failed, and malformed results keep the generic tool details", () => {
    const result = { sources: [{ url: "https://example.com", title: "Example" }] };
    for (const overrides of [
        { status: "running" },
        { failed: true },
        { result: "broken" },
    ])
        assert.deepEqual(webSources(tool("search", result, overrides)), []);
    assert.deepEqual(webSources(tool("fetch", {})), []);
    assert.deepEqual(webSources(tool("search", { ...result, failed: true })), []);
});

test("older saved search text still supplies website rows", () => {
    const sources = webSources(
        tool("search", { text: "Example\nhttps://example.com/page\nSaved excerpt" }),
    );
    assert.equal(sources[0].title, "Example");
    assert.equal(sources[0].snippet, "Saved excerpt");
});

test("web groups use search and page counts", () => {
    const search = tool("search", {});
    const fetch = tool("fetch", {});
    assert.equal(groupLabel([search, search], [2, 3]).target, "2 searches · 5 results");
    assert.equal(groupLabel([fetch, fetch], [1, 1]).action, "Read pages");
    assert.equal(groupLabel([fetch, fetch], [1, 1]).target, "2 page reads");
});
