const { expect } = require("@playwright/test");

module.exports = async function (page) {
    await expect(page.locator(".context-meter")).toHaveAttribute(
        "title",
        "2% of context used",
    );
    await page.locator(".context-meter").click();
    await expect(page.locator(".context-counts")).toHaveText("~18K / 1M");
    await expect(page.locator(".context-what")).toHaveText([
        "System prompt",
        "Tool definitions",
        "Messages",
    ]);
    await page.screenshot({ path: "/private/tmp/arx-context-breakdown.png" });
    await page.keyboard.press("Escape");
    await page.evaluate(() => window.compact(true));
    await expect(page.locator(".working-line .shimmer-text")).toHaveText("Compacting");
    await expect(page.locator(".context-meter")).toHaveAttribute(
        "title",
        "<1% of context used",
    );
    await expect(page.getByRole("button", { name: "Stop reply" })).toBeEnabled();
    await expect(
        page.getByRole("combobox", { name: "Chat model", exact: true }),
    ).toBeEnabled();
    await page.evaluate(() => window.compact(false));
    await expect(page.locator(".working-line .shimmer-text")).toHaveText("Working");
};
