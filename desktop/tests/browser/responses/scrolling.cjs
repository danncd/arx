const { expect } = require("@playwright/test");

module.exports = async function (page) {
    await page.emulateMedia({ reducedMotion: "reduce" });
    await page.evaluate(() =>
        window.feed("second", {
            text:
                "After the tool.\n\n" +
                "A separate paragraph for scroll testing.\n\n".repeat(50),
            status: "running",
        }),
    );
    await page.locator(".message-scroll").hover();
    await page.mouse.wheel(0, -10000);
    await expect(page.getByRole("button", { name: "Jump to latest" })).toBeVisible();
    await expect
        .poll(() =>
            page.locator(".message-scroll").evaluate((element) => element.scrollTop),
        )
        .toBe(0);
    const before = await page
        .locator(".message-scroll")
        .evaluate((element) => element.scrollTop);
    await page.evaluate(() =>
        window.feed("second", {
            text:
                "After the tool.\n\n" +
                "A separate paragraph for scroll testing.\n\n".repeat(55),
            status: "running",
        }),
    );
    await expect
        .poll(() =>
            page.locator(".message-scroll").evaluate((element) => element.scrollTop),
        )
        .toBe(before);
    await page.getByRole("button", { name: "Jump to latest" }).click();
    await expect
        .poll(() =>
            page
                .locator(".message-scroll")
                .evaluate(
                    (element) =>
                        element.scrollHeight - element.clientHeight - element.scrollTop,
                ),
        )
        .toBeLessThan(2);
};
