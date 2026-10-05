const { startBrowser } = require("../support/browser.cjs");
const { expect } = require("@playwright/test");
const path = require("node:path");
const fs = require("node:fs");

(async () => {
    const { server, browser, page, errors } = await startBrowser({
        viewport: { width: 1120, height: 780 },
    });
    try {
        await page.addInitScript(require("../fixtures/response-backend.cjs"), {
            pendingTitle: true,
        });
        await page.goto(server.resolvedUrls.local[0]);
        await page.locator(".working-line").waitFor();
        await require("./responses/title.cjs")(page);
        await require("./responses/attachments.cjs")(page);
        await require("./responses/usage.cjs")(page);
        await require("./responses/context.cjs")(page);
        const stream = await require("./responses/streaming.cjs")(page);
        await require("./responses/tool-cards.cjs")(page, stream);
        await require("./responses/completion.cjs")(page);
        await require("./responses/scrolling.cjs")(page);
        await require("./responses/web-tools.cjs")(page);
        await require("./responses/settings.cjs")(page);
        await require("./responses/sent-images.cjs")(page);
        await require("./responses/remote-images.cjs")(page);
        await require("./responses/image-tools.cjs")(page);
        if (errors.length) throw Error(errors.join("\n"));
        await page.screenshot({ path: "/private/tmp/arx-response-restored.png" });
        console.log(
            "PASS: paced streaming, compact thinking preview, expansion, one turn, chronological tool cards, readable In/Out, completion footer, reduced motion, scroll release and follow.",
        );
    } finally {
        if (browser) await browser.close();
        await server.close();
    }
})().catch((error) => {
    console.error(error);
    process.exit(1);
});
