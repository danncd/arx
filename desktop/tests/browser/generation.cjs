const { startBrowser } = require("../support/browser.cjs");
const { expect } = require("@playwright/test");
const menuBackend = require("../fixtures/menu-backend.cjs");
const catalog = require("../../../runtimes/generation/catalog.json");

(async () => {
    const { server, browser, page, errors } = await startBrowser();
    try {
        await page.addInitScript(menuBackend);
        await page.addInitScript((catalog) => {
            const bridge = window.arxDesktop;
            const request = bridge.request;
            const statuses = new Set();
            const events = new Set();
            const pending = [];
            let calls = 0;
            let defer = false;
            const state = (name, revision) => {
                const model = { ...catalog[0], name, size: 100 };
                return {
                    library: {
                        revision,
                        defaults: { image: model.id },
                        models: [{ model, status: "installed", received: 100 }],
                    },
                    jobs: [],
                    catalog,
                };
            };
            window.generationTest = {
                counts: () => ({ listeners: events.size, calls }),
                restart: (hold) => {
                    defer = hold;
                    statuses.forEach((cb) => cb({ state: "starting", error: null }));
                    statuses.forEach((cb) => cb({ state: "ready", error: null }));
                },
                releaseOld: () =>
                    pending.splice(0).forEach((r) => r(state("Obsolete image", 99))),
                emit: (name, revision) =>
                    events.forEach((cb) =>
                        cb({ library: state(name, revision).library }),
                    ),
            };
            bridge.onGeneration = (callback) => {
                events.add(callback);
                return () => events.delete(callback);
            };
            bridge.onBackendStatus = (callback) => {
                statuses.add(callback);
                callback({ state: "ready", error: null });
                return () => statuses.delete(callback);
            };
            bridge.request = (method, params) => {
                if (method !== "generation.state") return request(method, params);
                calls++;
                return defer
                    ? new Promise((resolve) => pending.push(resolve))
                    : Promise.resolve(
                          state(calls > 2 ? "Restarted image" : "Initial image", 0),
                      );
            };
        }, catalog);
        await page.goto(server.resolvedUrls.local[0]);
        const counts = () => page.evaluate(() => window.generationTest.counts());
        await expect.poll(async () => (await counts()).listeners).toBe(1);
        await page.getByRole("button", { name: "Settings", exact: true }).click();
        await page.getByRole("button", { name: "Generation", exact: true }).click();
        const image = page.getByRole("combobox", { name: "image model", exact: true });
        await expect(image).toContainText("Initial image");
        const before = await counts();
        await page.evaluate(() => window.generationTest.emit("Live image", 5));
        await page.evaluate(() => window.generationTest.emit("Stale image", 1));
        await expect(image).toContainText("Live image");
        await page.getByRole("button", { name: "General", exact: true }).click();
        await page.getByRole("button", { name: "Generation", exact: true }).click();
        expect(await counts()).toEqual(before);
        await page.evaluate(() => window.generationTest.restart(true));
        await expect(page.getByText("Loading models…", { exact: true })).toBeVisible();
        await page.evaluate(() => window.generationTest.restart(false));
        await expect(image).toContainText("Restarted image");
        await page.evaluate(() => window.generationTest.releaseOld());
        await expect(image).toContainText("Restarted image");
        expect((await counts()).listeners).toBe(1);
        const preview = await browser.newPage();
        preview.on("pageerror", (error) => errors.push(error.message));
        await preview.goto(new URL("preview.html", server.resolvedUrls.local[0]).href);
        await preview.getByRole("button", { name: "Settings", exact: true }).click();
        await preview.getByRole("button", { name: "Generation", exact: true }).click();
        await expect(
            preview.getByRole("combobox", { name: /^(image|video|speech) model$/ }),
        ).toHaveCount(3);
        await preview
            .getByRole("button", { name: "Browse image models", exact: true })
            .click();
        await expect(preview.locator(".local-model-row").first()).toBeVisible();
        expect(errors).toEqual([]);
        console.log(
            "PASS: one generation subscription, stale events, restart epochs, shared settings state and maintained preview.",
        );
    } finally {
        await browser.close();
        await server.close();
    }
})().catch((error) => {
    console.error(error);
    process.exitCode = 1;
});
