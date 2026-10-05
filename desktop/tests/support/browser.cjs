const { chromium } = require("@playwright/test");
const path = require("node:path");
const fs = require("node:fs");
async function startBrowser(options = {}) {
    const { createServer } = await import("vite");
    const server = await createServer({
        configFile: path.resolve(__dirname, "../../renderer/vite.config.ts"),
        server: { port: 0, strictPort: false },
        logLevel: "error",
    });
    let browser;
    try {
        await server.listen();
        const chrome =
            process.env.ARX_CHROME_BINARY ||
            "/Applications/Google Chrome.app/Contents/MacOS/Google Chrome";
        browser = await chromium.launch({
            executablePath: fs.existsSync(chrome) ? chrome : undefined,
            headless: true,
        });
        const page = await browser.newPage(options),
            errors = [];
        page.on("pageerror", (error) => errors.push(error.message));
        return { server, browser, page, errors };
    } catch (error) {
        await browser?.close();
        await server.close();
        throw error;
    }
}
module.exports = { startBrowser };
