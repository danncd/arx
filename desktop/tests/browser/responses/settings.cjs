const { expect } = require("@playwright/test");

module.exports = async function (page) {
    await page.getByRole("button", { name: "Settings", exact: true }).click();
    await expect(page.locator(".settings-dialog")).toHaveCSS("width", "960px");
    await page.getByRole("button", { name: "Connections", exact: true }).click();
    const provider = page.getByRole("button", { name: "DeepSeek", exact: true });
    await expect(provider).toHaveAttribute("aria-expanded", "false");
    await provider.click();
    await expect(provider).toHaveAttribute("aria-expanded", "true");
    await expect(
        page.getByRole("button", { name: "Change key", exact: true }),
    ).toBeVisible();
    await page.getByRole("button", { name: "Change key", exact: true }).click();
    await expect(page.getByLabel("DeepSeek API key")).toHaveAttribute("type", "password");
    await page.getByRole("button", { name: "Show key", exact: true }).click();
    await expect(page.getByLabel("DeepSeek API key")).toHaveAttribute("type", "text");
    await page.getByRole("button", { name: "Cancel", exact: true }).click();
    await expect(page.getByText("No models downloaded.")).toBeVisible();
    await expect(page.locator(".local-error")).toHaveCount(0);
    await page.screenshot({ path: "/private/tmp/arx-settings-connections.png" });
    await page.getByRole("button", { name: "Permissions", exact: true }).click();
    const mode = page.getByRole("combobox", { name: "Permission mode", exact: true });
    await expect(mode).toContainText("Ask");
    await mode.click();
    await expect(
        page
            .getByRole("listbox", { name: "Permission mode", exact: true })
            .locator(".select-option-title svg"),
    ).toHaveCount(3);
    await page.screenshot({ path: "/private/tmp/arx-settings-permissions.png" });
    await page.keyboard.press("Escape");
    await page.getByRole("button", { name: "Close settings", exact: true }).click();
};
