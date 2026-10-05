const { expect } = require("@playwright/test");

module.exports = async function (page) {
    const usage = page.locator(".session-usage-trigger");
    await expect(usage).toHaveText("1k");
    await usage.click();
    await expect(page.locator(".session-usage-panel dt")).toHaveText([
        "In",
        "Out",
        "Cache",
        "Hit rate",
    ]);
    await expect(page.locator(".session-usage-panel dd")).toHaveText([
        "800",
        "200",
        "600",
        "75%",
    ]);
    await page.screenshot({ path: "/private/tmp/arx-session-usage.png" });
    await page.evaluate(() => {
        window.tokenUsage = { input: 1000, output: 300, cached: 600, reasoning: 30 };
    });
    await expect(usage).toHaveText("1.3k");
    await expect(page.locator(".session-usage-panel dd")).toHaveText([
        "1,000",
        "300",
        "600",
        "60%",
    ]);
    await page.keyboard.press("Escape");
    await expect(page.locator(".session-usage-panel")).toHaveCount(0);
    await expect(usage).toBeFocused();
    await page.setViewportSize({ width: 720, height: 780 });
    await usage.click();
    const bounds = await page.locator(".session-usage-panel").boundingBox();
    if (!bounds || bounds.x < 0 || bounds.x + bounds.width > 720)
        throw Error("Usage panel overflowed");
    await page.locator(".chat-title").click();
    await expect(page.locator(".session-usage-panel")).toHaveCount(0);
    await page.setViewportSize({ width: 1120, height: 780 });
};
