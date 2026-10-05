const { expect } = require("@playwright/test");

module.exports = async function (page) {
    const settledHeight = await page
        .locator(".messages")
        .evaluate((element) => element.getBoundingClientRect().height);
    const workingNode = await page.locator(".working-line").elementHandle();
    await page.evaluate(() =>
        window.feed(
            "second",
            {
                text: "After the tool: everything is ready.",
                status: "done",
                usage: { input: 120, output: 30, cached: 0, reasoning: 4 },
            },
            true,
        ),
    );
    await expect(page.locator(".working-line")).toHaveCount(1);
    await expect(page.locator(".working-line")).toBeHidden();
    await expect(page.locator(".message-footer")).not.toHaveClass(/reserved/);
    const completedHeight = await page
        .locator(".messages")
        .evaluate((element) => element.getBoundingClientRect().height);
    if (Math.abs(completedHeight - settledHeight) > 1) {
        throw Error(
            `Reply shifted when Working disappeared: ${settledHeight} -> ${completedHeight}`,
        );
    }
    if (
        !(await workingNode.evaluate(
            (element) => element === document.querySelector(".working-line"),
        ))
    )
        throw Error("Working row was remounted");
};
