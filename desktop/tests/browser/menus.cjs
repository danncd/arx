const { startBrowser } = require("../support/browser.cjs");
const { expect } = require("@playwright/test");
const path = require("node:path");
const fs = require("node:fs");
const menuBackend = require("../fixtures/menu-backend.cjs");

async function trace(page, selector, anchor, gap) {
    await page.evaluate(
        ({ selector, anchor, gap }) => {
            window.menuFrames = [];
            window.traceMenus = true;
            const frame = () => {
                if (!window.traceMenus) return;
                const node = document.querySelector(selector);
                const button = document.querySelector(anchor);
                if (node && button) {
                    const bounds = node.getBoundingClientRect();
                    const target = button.getBoundingClientRect();
                    window.menuFrames.push({
                        text: node.textContent,
                        offset: Math.min(
                            Math.abs(bounds.bottom + gap - target.top),
                            Math.abs(bounds.top - gap - target.bottom),
                        ),
                    });
                }
                requestAnimationFrame(frame);
            };
            requestAnimationFrame(frame);
        },
        { selector, anchor, gap },
    );
}

async function stopTrace(page) {
    return page.evaluate(() => {
        window.traceMenus = false;
        return window.menuFrames;
    });
}

(async () => {
    const { server, browser, page, errors } = await startBrowser({
        viewport: { width: 1120, height: 780 },
    });
    try {
        await page.addInitScript(menuBackend);
        await page.goto(server.resolvedUrls.local[0]);
        const trigger = page.getByRole("combobox", { name: "Chat model" });
        await trigger.waitFor();
        await trace(page, ".model-popup", '[aria-label="Chat model"]', 6);
        await trigger.click();
        await page.getByRole("button", { name: "Choose model", exact: true }).click();
        await page.getByRole("option", { name: "Second model", exact: true }).click();
        await expect(page.locator(".model-current")).toContainText("Second model");
        const frames = await stopTrace(page);
        expect(frames.length).toBeGreaterThan(5);
        expect(frames.every((frame) => frame.offset < 1)).toBe(true);
        const listIndex = frames.findIndex((frame) => frame.text.startsWith("Models"));
        expect(listIndex).toBeGreaterThanOrEqual(0);
        expect(
            frames
                .slice(listIndex)
                .every((frame) => !frame.text.startsWith("First model")),
        ).toBe(true);
        await page.getByRole("button", { name: "Choose model", exact: true }).click();
        await page.evaluate(() => {
            window.failSelection = true;
        });
        await page.getByRole("option", { name: "First model", exact: true }).click();
        await expect(
            page.getByRole("listbox", { name: "Models", exact: true }),
        ).toBeVisible();
        await expect(trigger).toContainText("Second model");
        await page.keyboard.press("Escape");
        await page.keyboard.press("Escape");
        await expect(trigger).toBeFocused();
        await expect(page.locator(".model-popup")).toHaveCount(0);
        await page.evaluate(() => {
            window.failSelection = false;
        });
        await trigger.click();
        await page.setViewportSize({ width: 760, height: 520 });
        await trace(page, ".model-popup", '[aria-label="Chat model"]', 6);
        await page.getByRole("button", { name: "Choose model", exact: true }).click();
        await page.getByRole("option", { name: "First model", exact: true }).click();
        await expect(
            page.getByRole("switch", { name: "Thinking", exact: true }),
        ).toBeVisible();
        expect((await stopTrace(page)).every((frame) => frame.offset < 1)).toBe(true);
        await page.keyboard.press("Escape");
        await page.locator(".context-meter").click();
        await trace(page, ".context-panel", ".context-meter", 8);
        await page
            .locator(".context-panel .context-legend")
            .evaluate((node) => node.remove());
        await page.waitForTimeout(80);
        expect((await stopTrace(page)).every((frame) => frame.offset < 1)).toBe(true);
        await page.keyboard.press("Escape");
        await page.getByRole("combobox", { name: "Permissions", exact: true }).click();
        await expect(
            page.getByRole("listbox", { name: "Permissions", exact: true }),
        ).toBeVisible();
        await page.keyboard.press("ArrowDown");
        await page.keyboard.press("Escape");
        await expect(
            page.getByRole("combobox", { name: "Permissions", exact: true }),
        ).toBeFocused();
        await page.setViewportSize({ width: 1120, height: 780 });
        await page.getByRole("button", { name: "Settings", exact: true }).click();
        await page.getByRole("button", { name: "Permissions", exact: true }).click();
        const dialog = page.getByRole("dialog", { name: "Settings", exact: true });
        await dialog
            .getByRole("combobox", { name: "Permission mode", exact: true })
            .click();
        await page.keyboard.press("Escape");
        await expect(dialog).toBeVisible();
        await page.getByRole("button", { name: "Connections", exact: true }).click();
        await page
            .getByRole("group", { name: "Local models", exact: true })
            .getByRole("button", { name: /^Local models/ })
            .click();
        await page.getByRole("button", { name: "Browse models", exact: true }).click();
        const variant = page.getByRole("combobox", { name: "Model variant" });
        await variant.click();
        const variants = page.getByRole("listbox", { name: "Model variant" });
        const before = await variants.boundingBox();
        await page.keyboard.press("End");
        await page.waitForTimeout(80);
        const after = await variants.boundingBox();
        expect(after).toEqual(before);
        expect(after.height).toBeLessThanOrEqual(200);
        expect(await variants.evaluate((node) => node.scrollTop)).toBeGreaterThan(0);
        await page.keyboard.press("Enter");
        await expect(variant).toContainText("Q19");
        await expect(dialog).toBeVisible();
        await page.getByRole("button", { name: "Close settings", exact: true }).click();
        expect(errors).toEqual([]);
        console.log(
            "Menus: stable anchors, delayed/failed selection, keyboard focus, context resize, permissions and scrollable variants passed",
        );
    } finally {
        await browser?.close();
        await server.close();
    }
})().catch((error) => {
    console.error(error);
    process.exitCode = 1;
});
