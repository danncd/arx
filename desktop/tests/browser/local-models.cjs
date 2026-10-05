const { startBrowser } = require("../support/browser.cjs");
const { expect } = require("@playwright/test");
const path = require("node:path");
const menuBackend = require("../fixtures/menu-backend.cjs");

(async () => {
    const { server, browser, page, errors } = await startBrowser({
        viewport: { width: 1120, height: 780 },
    });
    try {
        await page.addInitScript(menuBackend);
        await page.addInitScript(require("../fixtures/local-models-backend.cjs"));
        await page.goto(server.resolvedUrls.local[0]);
        await page.getByRole("button", { name: "Settings", exact: true }).click();
        await page.getByRole("button", { name: "Connections", exact: true }).click();
        await page
            .getByRole("group", { name: "Local models", exact: true })
            .getByRole("button", { name: /^Local models/ })
            .click();
        await expect(page.locator(".model-context")).toContainText("24K active context");
        await expect(page.locator(".model-context")).toContainText("32K supported");
        const trash = page.getByRole("button", {
            name: "Delete model and files",
            exact: true,
        });
        await trash.click();
        await expect(
            page.getByRole("button", { name: "Confirm delete model and files" }),
        ).toBeVisible();
        expect(await page.evaluate(() => window.deletions.length)).toBe(0);
        await page.keyboard.press("Escape");
        await expect(trash).toBeVisible();
        await trash.click();
        await page.getByText("Your models", { exact: true }).click();
        await expect(trash).toBeVisible();
        await trash.click();
        await page
            .getByRole("button", { name: "Confirm delete model and files" })
            .click();
        await expect(page.getByText("No models downloaded.")).toBeVisible();
        expect(await page.evaluate(() => window.deletions)).toEqual([
            { action: "remove", id: "local:test", deleteFiles: true },
        ]);
        await page.getByRole("button", { name: "Browse models", exact: true }).click();
        await expect(page.locator(".local-model-row")).toHaveCount(2);
        await page.getByRole("button", { name: "Load more", exact: true }).click();
        await expect(page.locator(".local-model-row")).toHaveCount(3);
        await expect(
            page.getByRole("button", { name: "Load more", exact: true }),
        ).toHaveCount(0);
        await page.getByRole("combobox", { name: "Model type" }).click();
        await page.getByRole("option", { name: "Vision", exact: true }).click();
        await expect(page.locator(".local-model-row")).toHaveCount(1);
        await page.getByRole("checkbox", { name: "Recommended for this Mac" }).check();
        await expect(page.locator(".local-model-row")).toHaveCount(1);
        await page.getByRole("combobox", { name: "Sort models" }).click();
        await page.getByRole("option", { name: "Most liked", exact: true }).click();
        await expect
            .poll(() => page.evaluate(() => window.searchCalls.at(-1)?.sort))
            .toBe("likes");
        await expect(
            page.getByRole("button", { name: "Load more", exact: true }),
        ).toBeVisible();
        await page.getByRole("combobox", { name: "Model type" }).click();
        await page.getByRole("option", { name: "All types", exact: true }).click();
        const search = page.getByRole("searchbox", { name: "Search models" });
        await search.fill("slow");
        await expect
            .poll(() => page.evaluate(() => window.searchCalls.at(-1)?.query))
            .toBe("slow");
        await search.fill("fast");
        await expect(page.locator(".local-model-title")).toHaveText("fast");
        await page.waitForTimeout(1100);
        await expect(page.locator(".local-model-title")).toHaveText("fast");
        expect(errors).toEqual([]);
        console.log(
            "Passed: context labels, two-click deletion, blur/Escape reset, pagination deduplication, filters, sorting and stale search protection.",
        );
    } finally {
        await browser?.close();
        await server.close();
    }
})().catch((error) => {
    console.error(error);
    process.exitCode = 1;
});
