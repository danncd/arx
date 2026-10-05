import { test } from "node:test";
import assert from "node:assert/strict";
import { createRequire } from "node:module";
const { resolveLink } = createRequire(import.meta.url)("../../../main/ipc/links.cjs");

test("web and email links accept common model formatting", () => {
    for (const raw of [
        "github.com/danncd",
        "github.com/README.md",
        "www.example.com",
        "//example.com",
        " https://example.com/a?q=b ",
    ])
        assert.match(resolveLink(raw).value, /^https:\/\//);
    assert.equal(
        resolveLink("mailto:hello@example.com").value,
        "mailto:hello@example.com",
    );
});

test("file links retain spaces and resolve against the working directory", () => {
    assert.deepEqual(resolveLink("file:///tmp/a%20b.md"), {
        kind: "file",
        value: "/tmp/a b.md",
    });
    assert.equal(
        resolveLink("src/main.ts:12:4", "/project").value,
        "/project/src/main.ts",
    );
    assert.equal(resolveLink("README.md", "/project").value, "/project/README.md");
    assert.equal(resolveLink("/tmp/file.md#L20-L25").value, "/tmp/file.md");
});

test("unsafe protocols and credentialed URLs are rejected", () => {
    for (const raw of [
        "javascript:alert(1)",
        "data:text/html,hi",
        "https://user:secret@example.com",
        "https://",
        "x\u0000",
    ])
        assert.throws(() => resolveLink(raw));
});
