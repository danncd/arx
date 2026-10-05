const { expect } = require("@playwright/test");

module.exports = async function (page) {
    await page.reload();
    await expect(
        page.getByRole("button", { name: "Attach images", exact: true }),
    ).toBeVisible();
    const imageCalls = [];
    let imageCardHeight = 0;
    for (let index = 0; index < 6; index++) {
        const imageCall = {
            id: `view-image-${index}`,
            name: "web",
            offset: 0,
            reasoning_offset: 0,
            arguments: "",
            status: "preparing",
            result: "",
            failed: false,
        };
        imageCalls.push(imageCall);
        for (const argumentsText of [
            "",
            '{"operation":',
            '{"operation":"image","url":"https://example.com/',
            `{"operation":"image","url":"https://example.com/${index}.png"}`,
        ]) {
            imageCall.arguments = argumentsText;
            await page.evaluate(
                (tools) =>
                    window.feed("first", {
                        text: "",
                        reasoning: "",
                        tools,
                        status: "running",
                    }),
                imageCalls,
            );
            await page.waitForTimeout(80);
            await expect(page.locator(".tool-group")).toHaveCount(1);
            await expect(page.locator(".tool-group[open]")).toHaveCount(0);
            const height = await page
                .locator(".tool-activity")
                .evaluate((element) => element.getBoundingClientRect().height);
            if (imageCardHeight && Math.abs(height - imageCardHeight) > 1)
                throw Error("Collapsed image tools changed height while streaming");
            imageCardHeight = height;
        }
    }
    for (const imageCall of imageCalls) {
        imageCall.status = "done";
        imageCall.result = '{"text":"Image loaded","failed":false}';
        await page.evaluate(
            (tools) => window.feed("first", { tools, status: "running" }),
            imageCalls,
        );
        await expect(page.locator(".tool-group")).toHaveCount(1);
        await expect(page.locator(".tool-group[open]")).toHaveCount(0);
    }
    await expect(page.locator(".tool-target")).toHaveText("6 calls");
    await page.locator(".tool-group-summary").click();
    await expect(page.locator(".tool-group[open]")).toHaveCount(1);
    await expect(page.locator(".tool-file")).toHaveCount(6);
    await page.evaluate(
        (tools) => window.feed("first", { tools, status: "done" }, true),
        imageCalls,
    );
    await expect(page.locator(".tool-group[open]")).toHaveCount(1);
};
