const { desktopEnvironment, launchDesktop } = require("../support/desktop.cjs");
const { expect } = require("@playwright/test");
const fs = require("node:fs"),
    os = require("node:os"),
    path = require("node:path");
const project = path.resolve(__dirname, "../..");
const profile = fs.mkdtempSync(path.join(os.tmpdir(), "arx-auto-continue-ui-"));
const env = desktopEnvironment(profile);
(async () => {
    let app;
    const errors = [];
    async function launch() {
        app = await launchDesktop(project, env);
        const page = await app.firstWindow();
        page.on("pageerror", (e) => errors.push(e.message));
        await page.getByRole("textbox", { name: "Message", exact: true }).waitFor();
        return page;
    }
    async function general(page) {
        await page.getByRole("button", { name: "Settings", exact: true }).click();
        return page
            .getByRole("dialog")
            .getByRole("combobox", { name: "Auto continue", exact: true });
    }
    try {
        let page = await launch();
        let picker = await general(page);
        await expect(picker).toContainText("On");
        await picker.click();
        await page.getByRole("option", { name: "Off", exact: true }).click();
        await expect(picker).toContainText("Off");
        const saved = await page.evaluate(() => window.arxDesktop.request("snapshot"));
        if (saved.settings.auto_continue !== false) throw Error("Preference not saved");
        await app.close();
        app = null;
        page = await launch();
        picker = await general(page);
        await expect(picker).toContainText("Off");
        await picker.click();
        await page.getByRole("option", { name: "On", exact: true }).click();
        await expect(picker).toContainText("On");
        if (errors.length) throw Error(errors.join("\n"));
        console.log(
            "PASS: General Auto continue defaults On and persists Off/On across restart.",
        );
    } finally {
        if (app) await app.close();
        fs.rmSync(profile, { recursive: true, force: true });
    }
})().catch((e) => {
    console.error(e);
    process.exitCode = 1;
});
