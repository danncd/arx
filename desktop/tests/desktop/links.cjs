const { desktopEnvironment, launchDesktop } = require("../support/desktop.cjs");
const { expect } = require("@playwright/test");
const fs = require("node:fs");
const os = require("node:os");
const path = require("node:path");
const profile = fs.mkdtempSync(path.join(os.tmpdir(), "arx-links-ui-"));
const file = path.join(profile, "notes.md");
fs.writeFileSync(file, "Link test");
const html = path.join(profile, "game.html");
fs.writeFileSync(html, "<!doctype html><title>Game</title>");
const chunks = [
    {
        conversation: "links",
        id: "user",
        role: "user",
        text: "Links",
        status: "done",
        at: new Date().toISOString(),
    },
    {
        conversation: "links",
        id: "assistant",
        role: "assistant",
        text: `[Website](www.example.com) [Email](mailto:test@example.com) [File](<${file}>) [Game](<${html}>) [Missing](</tmp/arx-file-that-does-not-exist.md>)`,
        status: "done",
        at: new Date().toISOString(),
    },
];
fs.writeFileSync(
    path.join(profile, "transcript.jsonl"),
    chunks.map((item) => JSON.stringify(item)).join("\n") + "\n",
);
const env = desktopEnvironment(profile);
(async () => {
    const app = await launchDesktop(path.resolve(__dirname, "../.."), env);
    try {
        await app.evaluate(({ shell }) => {
            global.openedLinks = [];
            shell.openExternal = async (value) => {
                global.openedLinks.push(value);
            };
            shell.showItemInFolder = (value) => {
                global.openedLinks.push(value);
            };
            shell.openPath = async (value) => {
                global.openedLinks.push(value);
                return "";
            };
        });
        const page = await app.firstWindow();
        await page.getByRole("link", { name: "Website", exact: true }).click();
        await page.getByRole("link", { name: "Email", exact: true }).click();
        await page.getByRole("link", { name: "File", exact: true }).click();
        await page.getByRole("link", { name: "Game", exact: true }).click();
        await expect
            .poll(() => app.evaluate(() => global.openedLinks))
            .toEqual(["https://www.example.com/", "mailto:test@example.com", file, html]);
        await page.getByRole("link", { name: "Missing", exact: true }).click();
        await expect(page.getByRole("alert")).toContainText(
            "File not found: arx-file-that-does-not-exist.md",
        );
        console.log(
            "PASS: Markdown website, email and file links, native dispatch, missing-file feedback",
        );
    } finally {
        await app.close();
    }
})().catch((error) => {
    console.error(error);
    process.exitCode = 1;
});
