const { expect } = require("@playwright/test");

module.exports = async function (page) {
    await expect(page.locator(".chat-list .chat-row")).toHaveCount(0);
    await expect(page.locator(".message.user")).toContainText("Check the files");
    await page.evaluate(() => window.renameSession("Organize project files"));
    await expect(page.locator(".chat-list .chat-row")).toHaveCount(1);
    await expect(page.locator(".chat-list .chat-select")).toHaveText(
        "Organize project files",
    );
    await expect(page.locator(".message.user")).toContainText("Check the files");
    await page.evaluate(() => window.renameSession("Check the files"));
};
