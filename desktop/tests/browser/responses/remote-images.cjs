const { expect } = require("@playwright/test");

module.exports = async function (page) {
    await page.evaluate(() =>
        window.feed(
            "illustrated-reply",
            {
                text: "Here is the diagram.\n\n[![Web diagram](https://example.com/diagram.png)](https://example.com/source)\n\n![Portrait](https://example.com/portrait.png) ![Landscape](https://example.com/landscape.png)\n\n![Fourth](https://example.com/landscape.png?v=2)\n\nBroken example:\n\n![Missing image](https://example.com/missing.png)\n\n![Unsafe image](file:///tmp/private.png)",
                status: "done",
            },
            true,
        ),
    );
    const webPreview = page.getByRole("button", {
        name: "Preview Web diagram",
        exact: true,
    });
    await webPreview.scrollIntoViewIfNeeded();
    await expect(webPreview.locator("img")).toBeVisible();
    const replyImage = await webPreview.locator("img").boundingBox();
    const replyButton = await webPreview.boundingBox();
    if (
        Math.abs(replyButton.width - replyImage.width) > 1 ||
        Math.abs(replyButton.height - replyImage.height) > 1
    )
        throw Error("Image preview added an empty frame");
    await expect(
        page.getByRole("link", { name: "Missing image", exact: true }),
    ).toHaveAttribute("href", "https://example.com/missing.png");
    await expect(page.locator(".markdown img[src^='file:']")).toHaveCount(0);
    const row = page.locator(".image-row").filter({ has: webPreview });
    await expect(row.locator(".response-image")).toHaveCount(4);
    const tiles = await row.locator(".response-image").evaluateAll((nodes) =>
        nodes.map((node) => {
            const box = node.getBoundingClientRect();
            return { width: box.width, height: box.height, top: box.top };
        }),
    );
    if (
        tiles.some(
            (tile) =>
                tile.width !== 200 ||
                tile.height !== 156 ||
                Math.abs(tile.top - tiles[0].top) > 1,
        )
    )
        throw Error("Image row has unequal or wrapped thumbnails");
    const overflowing = await row.evaluate(
        (element) => element.scrollWidth > element.clientWidth,
    );
    if (!overflowing) throw Error("Image row should scroll");
    await row.evaluate((element) => {
        element.scrollLeft = element.scrollWidth;
    });
    await expect(
        page.getByRole("button", { name: "Preview Fourth", exact: true }),
    ).toBeVisible();
    if ((await row.evaluate((element) => element.scrollLeft)) <= 0)
        throw Error("Image row did not scroll");
    await page.getByRole("button", { name: "Preview Portrait", exact: true }).click();
    const portrait = await page.locator(".image-preview-full").boundingBox();
    if (portrait.height <= portrait.width)
        throw Error("Preview did not preserve the portrait shape");
    await page.keyboard.press("Escape");
    await row.evaluate((element) => {
        element.scrollLeft = 0;
    });
    await webPreview.click();
    await expect(
        page.getByRole("dialog", { name: "Preview Web diagram", exact: true }),
    ).toBeVisible();
    await page.screenshot({ path: "/private/tmp/arx-web-image-preview.png" });
    await page.keyboard.press("Escape");
    await expect(webPreview).toBeFocused();
    await page.screenshot({ path: "/private/tmp/arx-web-image-reply.png" });
};
