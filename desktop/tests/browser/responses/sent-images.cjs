const { expect } = require("@playwright/test");

module.exports = async function (page) {
    await page.getByRole("button", { name: "Attach images", exact: true }).click();
    await page.getByRole("textbox", { name: "Message", exact: true }).fill("");
    await expect(
        page.getByRole("button", { name: "Send message", exact: true }),
    ).toBeEnabled();
    await page.getByRole("button", { name: "Send message", exact: true }).click();
    await expect(page.locator(".message.user > .image-attachments img")).toHaveAttribute(
        "alt",
        "sample.png",
    );
    await expect(page.locator(".composer .image-attachments")).toHaveCount(0);
    const sentImage = await page.evaluate(() => window.sentMessage);
    if (sentImage.text !== "" || sentImage.images[0].id !== "a".repeat(64))
        throw Error("Image-only send lost its attachment");
    await expect(
        page.locator(".message.user").last().locator(".user-message"),
    ).toHaveCount(0);
    await page.getByRole("button", { name: "Attach images", exact: true }).click();
    await page
        .getByRole("textbox", { name: "Message", exact: true })
        .fill("Look at this image");
    await page.getByRole("button", { name: "Send message", exact: true }).click();
    const sent = page.locator(".message.user").last();
    await expect(sent.locator(".user-message")).toHaveText("Look at this image");
    await expect(sent.locator(".user-message .image-attachments")).toHaveCount(0);
    const imageBounds = await sent.locator(".image-attachments").boundingBox();
    const textBounds = await sent.locator(".user-message").boundingBox();
    if (imageBounds.y + imageBounds.height >= textBounds.y)
        throw Error("Image overlaps the text bubble");
    const preview = sent.getByRole("button", {
        name: "Preview sample.png",
        exact: true,
    });
    await preview.click();
    await expect(page.getByRole("dialog", { name: "Preview sample.png" })).toBeVisible();
    const expanded = await page.locator(".image-preview-full").boundingBox();
    if (expanded.width <= imageBounds.width) throw Error("Image preview did not expand");
    await page.screenshot({ path: "/private/tmp/arx-image-preview.png" });
    await page.keyboard.press("Escape");
    await expect(page.locator(".image-preview-dialog")).toHaveCount(0);
    await expect(preview).toBeFocused();
    await preview.click();
    await page.getByRole("button", { name: "Close image preview" }).click();
    await expect(page.locator(".image-preview-dialog")).toHaveCount(0);
    await page.setViewportSize({ width: 720, height: 600 });
    await preview.click();
    const dialogBounds = await page.locator(".image-preview-dialog").boundingBox();
    if (
        dialogBounds.x < 0 ||
        dialogBounds.y < 0 ||
        dialogBounds.x + dialogBounds.width > 720 ||
        dialogBounds.y + dialogBounds.height > 600
    )
        throw Error("Preview overflows the window");
    await page.mouse.click(5, 5);
    await expect(page.locator(".image-preview-dialog")).toHaveCount(0);
    await page.setViewportSize({ width: 1120, height: 780 });
    await page.screenshot({ path: "/private/tmp/arx-sent-image-layout.png" });
};
