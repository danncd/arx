const { startBrowser } = require("../support/browser.cjs");
const { expect } = require("@playwright/test");
const fs = require("node:fs");
const path = require("node:path");

(async () => {
    const { server, browser, page, errors } = await startBrowser({});
    try {
        await page.addInitScript(require("../fixtures/response-backend.cjs"));
        await page.addInitScript(() => {
            const bridge = window.arxDesktop;
            const request = bridge.request;
            const listeners = new Set();
            let pending;
            window.resourceCalls = [];
            window.resourceFetches = 0;
            window.resourceCancels = 0;
            bridge.onPermission = (callback) => {
                listeners.add(callback);
                return () => listeners.delete(callback);
            };
            const publish = (value) => listeners.forEach((callback) => callback(value));
            bridge.request = async (method, params) => {
                if (method === "snapshot") {
                    const snapshot = await request(method, params);
                    snapshot.settings.run = {
                        model: "network:server:test",
                        provider: "network",
                        effort: "",
                    };
                    snapshot.settings.chat_permissions = {
                        session: { mode: "ask", roots: [] },
                    };
                    return snapshot;
                }
                if (method === "network.state")
                    return [
                        {
                            id: "server",
                            name: "Test server",
                            connected: true,
                            url: "http://localhost:1234",
                            requiresToken: false,
                            models: [
                                {
                                    loaded: true,
                                    key: "test",
                                    info: {
                                        id: "network:server:test",
                                        provider: "network",
                                        name: "Network",
                                        contextWindow: 100000,
                                        vision: true,
                                    },
                                },
                            ],
                        },
                    ];
                if (method === "deepseek.status")
                    return {
                        configured: false,
                        connected: false,
                        models: [],
                        error: "",
                    };
                if (method === "web.image" || method === "web.icon") {
                    window.resourceCalls.push(method);
                    throw Error("Ask rendering must not request remote resources");
                }
                if (method === "tools.run") {
                    window.resourceCalls.push(method);
                    return new Promise((resolve) => {
                        pending = { id: params.id, resolve };
                        publish({
                            id: crypto.randomUUID(),
                            action: {
                                tool: "web",
                                operation: "image",
                                url: params.arguments.url,
                            },
                        });
                    });
                }
                if (method === "permissions.respond") {
                    const held = pending;
                    pending = null;
                    publish(null);
                    if (params.allow) window.resourceFetches++;
                    held.resolve(
                        params.allow
                            ? {
                                  text: "saved",
                                  failed: false,
                                  images: [{ id: "saved" }],
                              }
                            : { text: "Permission denied", failed: true },
                    );
                    return {};
                }
                if (method === "tools.cancel") {
                    if (pending?.id === params.id) {
                        window.resourceCancels++;
                        pending.resolve({ text: "Cancelled", failed: true });
                        pending = null;
                        publish(null);
                    }
                    return { accepted: true };
                }
                return request(method, params);
            };
        });
        await page.goto(server.resolvedUrls.local[0]);
        await page.getByRole("textbox", { name: "Message", exact: true }).waitFor();
        await page.evaluate(() =>
            window.feed("first", { text: "Network ready", status: "done" }, true),
        );
        await page
            .getByRole("textbox", { name: "Message", exact: true })
            .fill("Network-only send");
        await page.getByRole("button", { name: "Send message", exact: true }).click();
        await expect
            .poll(() => page.evaluate(() => window.sentMessage?.text))
            .toBe("Network-only send");
        await page.evaluate(() =>
            window.feed(
                "web-result",
                {
                    text: "Found a page.",
                    status: "done",
                    tools: [
                        {
                            id: "search",
                            name: "web",
                            offset: 13,
                            reasoning_offset: 0,
                            status: "done",
                            arguments: JSON.stringify({
                                operation: "search",
                                query: "example",
                            }),
                            result: JSON.stringify({
                                text: "Search results",
                                sources: [
                                    { url: "https://example.com/page", title: "Example" },
                                ],
                            }),
                            failed: false,
                        },
                    ],
                },
                true,
            ),
        );
        await page.locator(".tool-group-summary").last().click();
        await expect(page.locator(".web-site svg.web-site-icon")).toHaveCount(1);
        await expect(page.locator(".web-site img.web-site-icon")).toHaveCount(0);
        const text = "![Private query](https://example.com/image.png?secret=test)";
        await page.evaluate(
            (text) => window.feed("illustration", { text, status: "done" }, true),
            text,
        );
        await page
            .getByRole("button", { name: "Load image", exact: true })
            .scrollIntoViewIfNeeded();
        await expect(
            page.getByRole("button", { name: "Load image", exact: true }),
        ).toBeVisible();
        await expect.poll(() => page.evaluate(() => window.resourceCalls.length)).toBe(0);
        await page.getByRole("button", { name: "Load image", exact: true }).click();
        await expect(
            page.getByRole("region", { name: "Permission request" }),
        ).toContainText("https://example.com/image.png?secret=test");
        await page.getByRole("button", { name: "Deny", exact: true }).click();
        await expect(
            page.getByRole("button", { name: "Retry image", exact: true }),
        ).toBeVisible();
        await expect.poll(() => page.evaluate(() => window.resourceFetches)).toBe(0);
        await page.getByRole("button", { name: "Retry image", exact: true }).click();
        await page.getByRole("button", { name: "Allow once", exact: true }).click();
        const image = page
            .getByRole("button", { name: "Preview Private query", exact: true })
            .locator("img");
        await expect(image).toBeVisible();
        await image.evaluate((node) => (window.loadedImageNode = node));
        await page.evaluate(
            (text) =>
                window.feed(
                    "illustration",
                    { text: text + "\n\nMore streaming text", status: "done" },
                    true,
                ),
            text,
        );
        await expect(image).toBeVisible();
        if (!(await image.evaluate((node) => node === window.loadedImageNode)))
            throw Error("A chat update remounted the loaded image");
        await page.evaluate(() =>
            window.feed(
                "pending-image",
                { text: "![Next](https://example.com/next.png)", status: "done" },
                true,
            ),
        );
        await page.getByRole("button", { name: "Load image", exact: true }).click();
        await expect(
            page.getByRole("region", { name: "Permission request" }),
        ).toBeVisible();
        await page.getByRole("button", { name: "New chat", exact: true }).first().click();
        await expect.poll(() => page.evaluate(() => window.resourceCancels)).toBe(1);
        await expect(
            page.getByRole("region", { name: "Permission request" }),
        ).toHaveCount(0);
        if (errors.length) throw Error(errors.join("\n"));
        console.log(
            "PASS: network-only send, Ask zero automatic fetches, denial/retry/approval, navigation cancellation, stable loaded image during updates.",
        );
    } finally {
        if (browser) await browser.close();
        await server.close();
    }
})().catch((error) => {
    console.error(error);
    process.exitCode = 1;
});
