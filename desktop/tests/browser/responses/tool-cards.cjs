const { expect } = require("@playwright/test");

module.exports = async function (page, { text, call }) {
    await page.evaluate(
        (tool) => window.feed("first", { tools: [tool], status: "running" }),
        call,
    );
    await expect(page.locator(".tool-action")).toHaveText("Read");
    await page.evaluate(
        (tool) =>
            window.feed("first", {
                tools: [
                    {
                        ...tool,
                        status: "done",
                        result: '{"text":"Contents 🌱","failed":false}',
                    },
                ],
                status: "done",
            }),
        call,
    );
    await page.evaluate(() =>
        window.feed("second", {
            text: "After the tool: everything is ready.",
            status: "running",
        }),
    );
    await expect(page.locator(".message.assistant")).toHaveCount(1);
    await expect(page.locator(".agent-label")).toHaveCount(1);
    await expect(page.locator(".message.assistant > .prose").last()).toContainText(
        "After the tool: everything is ready.",
    );
    const order = await page
        .locator(".message.assistant")
        .evaluate((element) =>
            [...element.children]
                .filter(
                    (child) =>
                        child.classList.contains("prose") ||
                        child.classList.contains("tool-reveal"),
                )
                .map((child) =>
                    child.classList.contains("prose")
                        ? child.textContent.trim().slice(0, 14)
                        : "tool",
                ),
        );
    if (
        JSON.stringify(order) !==
        JSON.stringify([text.slice(0, 14), "tool", "After the tool".slice(0, 14)])
    )
        throw Error("Incorrect response order: " + JSON.stringify(order));
    await page.locator(".tool-group-summary").click();
    await page.locator(".tool-file > summary").click();
    await expect(page.getByLabel("Tool output")).toHaveText("Contents 🌱");
    await expect(page.getByLabel("Tool output")).toBeVisible();
    await page.waitForTimeout(250);
    await page.screenshot({ path: "/private/tmp/arx-response-tools.png" });
    const expansion = await page
        .locator(".tool-file")
        .first()
        .evaluate(async (details) => {
            const reveal = details.closest(".tool-reveal");
            const content = reveal.querySelector(".tool-reveal-content");
            const duration = getComputedStyle(
                details,
                "::details-content",
            ).transitionDuration;
            details.querySelector("summary").click();
            const differences = [];
            const started = performance.now();
            while (performance.now() - started < 300) {
                await new Promise(requestAnimationFrame);
                differences.push(
                    Math.abs(
                        reveal.getBoundingClientRect().height -
                            content.getBoundingClientRect().height,
                    ),
                );
            }
            return { duration, maximum: Math.max(...differences) };
        });
    if (
        expansion.maximum > 1 ||
        expansion.duration.split(", ").some((value) => value !== "0.22s")
    )
        throw Error("Nested expansion drift: " + JSON.stringify(expansion));
    await page.locator(".tool-file > summary").click();
    await page.waitForTimeout(250);
    await page.getByRole("button", { name: "In", exact: true }).click();
    await expect(page.getByLabel("Tool input")).toContainText('"operation": "read"');
    await page.evaluate(async () => {
        await Promise.all(
            document
                .getAnimations()
                .filter((animation) =>
                    Number.isFinite(animation.effect?.getComputedTiming().endTime),
                )
                .map((animation) => animation.finished.catch(() => {})),
        );
    });

    await expect
        .poll(() =>
            page
                .locator(".tool-result-height")
                .first()
                .evaluate((element) =>
                    Math.abs(
                        element.getBoundingClientRect().height -
                            element.firstElementChild.getBoundingClientRect().height,
                    ),
                ),
        )
        .toBeLessThan(1);
};
