const { expect } = require("@playwright/test");

module.exports = async function (page) {
    await page.evaluate(() =>
        window.feed(
            "web-result",
            {
                text: "Found a page.",
                status: "done",
                tools: [
                    {
                        id: "web-search",
                        name: "web",
                        offset: 13,
                        reasoning_offset: 0,
                        status: "done",
                        arguments: '{"operation":"search","query":"example"}',
                        result: JSON.stringify({
                            text: "Search results",
                            sources: [
                                {
                                    url: "https://example.com/page",
                                    title: "Example page",
                                    snippet: "A source-specific search excerpt.",
                                },
                                {
                                    url: "https://missing.example/page",
                                    title: "No icon page",
                                },
                            ],
                        }),
                        failed: false,
                    },
                ],
            },
            true,
        ),
    );
    await expect(page.locator(".tool-group-summary")).toHaveCount(2);
    await page.locator(".tool-group-summary").last().click();
    await expect(page.locator(".web-site img.web-site-icon")).toHaveCount(1);
    await expect(page.locator(".web-site svg.web-site-icon")).toHaveCount(1);
    await expect(page.locator(".web-site img.web-site-icon")).toBeVisible();
    const searchGroup = page.locator(".tool-group").last();
    await expect(searchGroup.locator(".tool-file")).toHaveCount(2);
    await expect(searchGroup.locator(".tool-group-summary .web-site-icon")).toHaveCount(
        0,
    );
    const site = searchGroup.locator(".web-site").first();
    await expect(site.locator("summary")).toContainText("example.com");
    await site.locator("summary").click();
    await expect(site.getByLabel("Tool output")).toHaveText(
        "A source-specific search excerpt.",
    );
    await expect(site.locator(".web-page-output > a")).toHaveAttribute(
        "href",
        "https://example.com/page",
    );
    await site.getByRole("button", { name: "In", exact: true }).click();
    await expect(site.getByLabel("Tool input")).toContainText('"query": "example"');
    await site.getByRole("button", { name: "Out", exact: true }).click();
    await page.evaluate(() =>
        window.feed(
            "fetch-result",
            {
                text: "Read the page.",
                status: "done",
                tools: [
                    {
                        id: "web-fetch",
                        name: "web",
                        offset: 14,
                        reasoning_offset: 0,
                        status: "done",
                        arguments: JSON.stringify({
                            operation: "fetch",
                            url: "https://example.com/page",
                        }),
                        result: JSON.stringify({
                            text: "Source: https://example.com/page\n\nThe fetched page content.",
                            truncated: false,
                            sources: [
                                {
                                    url: "https://example.com/page",
                                    title: "Fetched page",
                                },
                            ],
                        }),
                        failed: false,
                    },
                ],
            },
            true,
        ),
    );
    await expect(page.locator(".tool-group-summary")).toHaveCount(3);
    const fetchGroup = page.locator(".tool-group").last();
    await expect(fetchGroup.locator(".tool-group-summary")).toContainText(
        "Read page · example.com",
    );
    await fetchGroup.locator(".tool-group-summary").click();
    await expect(fetchGroup.locator(".tool-file")).toHaveCount(1);
    await fetchGroup.locator(".web-site > summary").click();
    await expect(fetchGroup.getByLabel("Tool output")).toHaveText(
        "The fetched page content.",
    );
    await expect(fetchGroup.locator(".web-source-note")).toHaveText(
        "End of retained page",
    );
};
