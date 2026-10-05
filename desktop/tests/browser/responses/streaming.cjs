const { expect } = require("@playwright/test");

module.exports = async function (page) {
    await page.evaluate(() =>
        window.feed("first", {
            reasoning: "Inspect the existing file before making changes.",
            status: "running",
        }),
    );
    const thinking = page.locator(".thinking-header").first();
    await expect(thinking).toHaveAttribute("aria-expanded", "false");
    await expect(thinking.locator(".thinking-label")).toHaveText("Think");
    await expect(thinking.locator(".thinking-preview")).toContainText("Inspect", {
        timeout: 5000,
    });
    await thinking.click();
    await expect(thinking).toHaveAttribute("aria-expanded", "true");
    await thinking.click();
    const prefix = "Before the tool 🌱. ";
    const text =
        prefix + "A smooth response should reveal this burst progressively. ".repeat(20);
    const sizes = await page.evaluate(async (target) => {
        window.feed("first", { text: "Before", status: "running" });
        await new Promise((resolve) => setTimeout(resolve, 100));
        window.feed("first", { text: target, status: "running" });
        await new Promise((resolve) => setTimeout(resolve, 90));
        const early =
            document.querySelector(".message.assistant > .prose")?.textContent.length ||
            0;
        await new Promise((resolve) => setTimeout(resolve, 180));
        const later =
            document.querySelector(".message.assistant > .prose")?.textContent.length ||
            0;
        return { early, later, total: target.length };
    }, text);
    if (!(sizes.early > 0 && sizes.early < sizes.later && sizes.later < sizes.total))
        throw Error("Text did not reveal gradually: " + JSON.stringify(sizes));
    await expect(page.locator(".message.assistant > .prose")).toHaveText(text.trim(), {
        timeout: 5000,
    });
    const call = {
        id: "read",
        name: "files",
        offset: Buffer.byteLength(text),
        reasoning_offset: Buffer.byteLength(
            "Inspect the existing file before making changes.",
        ),
        arguments: '{"operation":"read","path":"/tmp/notes.txt"}',
        status: "running",
        result: "",
        failed: false,
    };
    await page.evaluate(
        (tool) =>
            window.feed("first", {
                tools: [
                    {
                        ...tool,
                        status: "preparing",
                        arguments:
                            '{"operation":"write","path":"/tmp/notes.txt","content":"partial',
                    },
                ],
                status: "running",
            }),
        call,
    );
    await expect(page.locator(".tool-action")).toHaveText("Write file");
    await expect(page.locator(".tool-spinner")).toBeVisible();
    await expect(page.locator(".tool-target")).toHaveText("notes.txt");
    await expect
        .poll(async () =>
            page
                .locator(".tool-reveal")
                .evaluate((element) => element.getBoundingClientRect().height),
        )
        .toBeGreaterThan(45);
    await page.screenshot({ path: "/private/tmp/arx-streaming-tool.png" });

    return { text, call };
};
