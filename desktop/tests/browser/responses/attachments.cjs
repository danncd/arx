const { expect } = require("@playwright/test");

module.exports = async function (page) {
    await page.getByRole("button", { name: "Attach images", exact: true }).click();
    await expect(page.locator(".composer .image-attachments img")).toHaveAttribute(
        "alt",
        "sample.png",
    );
    const previewBounds = await page
        .locator(".composer .image-attachments")
        .boundingBox();
    const composerBounds = await page.locator(".composer").boundingBox();
    if (
        !previewBounds ||
        !composerBounds ||
        previewBounds.y < composerBounds.y ||
        previewBounds.y + previewBounds.height > composerBounds.y + composerBounds.height
    )
        throw Error("Images are outside the composer");
    await page.screenshot({ path: "/private/tmp/arx-vision-composer.png" });
    await page.getByRole("button", { name: "Remove sample.png", exact: true }).click();
    await expect(page.locator(".composer .image-attachments")).toHaveCount(0);
};
