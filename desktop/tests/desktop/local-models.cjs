const { desktopEnvironment, launchDesktop } = require("../support/desktop.cjs");
const { expect } = require("@playwright/test");
const fs = require("node:fs");
const os = require("node:os");
const path = require("node:path");
const project = path.resolve(__dirname, "../..");
const profile = fs.mkdtempSync(path.join(os.tmpdir(), "arx-local-ui-"));
const modelDirectory = path.join(profile, "models");
const source = process.env.ARX_TEST_MODEL;
if (!source) {
    console.log(
        "SKIP: set ARX_TEST_MODEL to a tool-capable GGUF file for local inference checks",
    );
    process.exit(0);
}
if (!path.isAbsolute(source) || !fs.existsSync(source))
    throw Error("ARX_TEST_MODEL must be an existing absolute path");
fs.mkdirSync(modelDirectory, { recursive: true });
if (process.env.ARX_TEST_ENGINE_DIR) {
    fs.cpSync(process.env.ARX_TEST_ENGINE_DIR, path.join(modelDirectory, "engine"), {
        recursive: true,
    });
}
const disposableModel = path.join(profile, path.basename(source));
fs.copyFileSync(source, disposableModel);
const env = {
    ...process.env,
    ELECTRON_RUN_AS_NODE: "",
    ARX_DEV_PROFILE: profile,
    ARX_MODELS_DIR: modelDirectory,
};
delete env.ARX_DEV_URL;
(async () => {
    let app;
    const errors = [];
    try {
        app = await launchDesktop(project, env);
        const page = await app.firstWindow();
        page.on("pageerror", (error) => errors.push(error.message));
        await page.getByRole("textbox", { name: "Message", exact: true }).waitFor();
        await page.getByRole("button", { name: "Settings", exact: true }).click();
        await page.getByRole("button", { name: "Connections", exact: true }).click();
        await page
            .getByRole("group", { name: "Local models", exact: true })
            .getByRole("button", { name: /^Local models/ })
            .click();
        await expect(
            page.getByRole("button", { name: "Import model file", exact: true }),
        ).toBeVisible();
        await expect(page.getByText("No models downloaded.")).toBeVisible();
        await page.screenshot({ path: "/private/tmp/arx-local-real-settings.png" });
        await page.getByRole("button", { name: "Browse models", exact: true }).click();
        await page
            .getByRole("searchbox", { name: "Search models" })
            .fill("Qwen3-0.6B-GGUF");
        await expect(
            page.locator(".local-model-title").filter({ hasText: "Qwen3-0.6B" }).first(),
        ).toBeVisible({ timeout: 45000 });
        await expect(
            page.getByRole("button", { name: "Download", exact: true }).first(),
        ).toBeEnabled({ timeout: 45000 });
        await page.screenshot({ path: "/private/tmp/arx-local-real-browser.png" });
        await page
            .getByRole("button", { name: "Back to connections", exact: true })
            .click();
        await app.evaluate(({ ipcMain }, source) => {
            ipcMain.removeHandler("arx:choose-model");
            ipcMain.handle("arx:choose-model", async () => source);
        }, disposableModel);
        await page
            .getByRole("group", { name: "Local models", exact: true })
            .getByRole("button", { name: /^Local models/ })
            .click();
        await page
            .getByRole("button", { name: "Import model file", exact: true })
            .click();
        await page.getByRole("button", { name: "Load", exact: true }).click();
        await expect(
            page.getByRole("button", { name: "Unload", exact: true }),
        ).toBeVisible({ timeout: 60000 });
        await page.screenshot({ path: "/private/tmp/arx-local-real-loaded.png" });
        const local = await page.evaluate(() => window.arxDesktop.request("local.state"));
        const model = local.models[0];
        console.log(
            `Loaded context: ${model.info.contextWindow} of ${model.info.trainedContext} supported`,
        );
        if (model.info.contextWindow !== model.info.trainedContext * 0.75)
            throw Error("Fixture should fit at 75% context");
        if (!model.info.tools || model.info.contextWindow < 4096)
            throw Error("Loaded capabilities missing");
        await page.evaluate(async (model) => {
            const snapshot = await window.arxDesktop.request("snapshot");
            await window.arxDesktop.request("configure", {
                directory: snapshot.settings.directory,
                run: { provider: "local", model: model.id, effort: "none" },
            });
        }, model);
        await page.reload();
        await page.getByRole("textbox", { name: "Message", exact: true }).waitFor();
        await expect(
            page.getByRole("combobox", { name: "Chat model", exact: true }),
        ).toContainText(model.name);
        await page
            .getByRole("textbox", { name: "Message", exact: true })
            .fill("Reply with just the word ready.");
        await page.getByRole("button", { name: "Send message", exact: true }).click();
        await expect(page.locator(".message.assistant .prose").last()).toContainText(
            /ready/i,
            { timeout: 60000 },
        );
        await expect
            .poll(() =>
                page.evaluate(
                    async () => (await window.arxDesktop.request("chat.state")).run.state,
                ),
            )
            .toBe("idle");
        await page.getByRole("button", { name: /Session token usage/ }).click();
        await expect(
            page
                .getByRole("dialog", { name: "Session token usage" })
                .locator(".session-usage-row")
                .filter({ hasText: "Cache" })
                .locator("dd"),
        ).toHaveText("—");
        await page.keyboard.press("Escape");
        await page.screenshot({ path: "/private/tmp/arx-local-real-chat.png" });
        const note = path.join(profile, "local-tool-check.txt");
        fs.writeFileSync(note, "arx-local-tool-pass", "utf8");
        await page.evaluate(async () => {
            await window.arxDesktop.request("permissions.configure", {
                conversation: "",
                mode: "ask",
                roots: [],
            });
        });
        await page.getByRole("button", { name: "New chat", exact: true }).first().click();
        await page.getByRole("combobox", { name: "Permissions", exact: true }).click();
        await page.getByRole("option", { name: "Ask", exact: true }).click();
        await page.evaluate((note) => {
            window.localPermissionResult = null;
            void window.arxDesktop
                .request("tools.run", {
                    id: "local-read-check",
                    name: "files",
                    arguments: { operation: "read", path: note },
                    conversation: "",
                })
                .then((result) => {
                    window.localPermissionResult = result;
                });
        }, note);
        await expect(
            page.getByRole("region", { name: "Permission request" }),
        ).toBeVisible({ timeout: 10000 });
        await page.getByRole("button", { name: "Allow once", exact: true }).click();
        await expect
            .poll(() => page.evaluate(() => JSON.stringify(window.localPermissionResult)))
            .toContain("arx-local-tool-pass");
        await page.getByRole("button", { name: "Settings", exact: true }).click();
        await page.getByRole("button", { name: "Connections", exact: true }).click();
        await page
            .getByRole("group", { name: "Local models", exact: true })
            .getByRole("button", { name: /^Local models/ })
            .click();
        await page.getByRole("button", { name: "Unload", exact: true }).click();
        await expect(
            page.getByRole("button", { name: "Load", exact: true }),
        ).toBeVisible();
        await page
            .getByRole("button", { name: "Delete model and files", exact: true })
            .click();
        if (!fs.existsSync(disposableModel)) throw Error("First click deleted the file");
        await page
            .getByRole("button", {
                name: "Confirm delete model and files",
                exact: true,
            })
            .click();
        await expect(page.getByText("No models downloaded.")).toBeVisible();
        if (fs.existsSync(disposableModel) || !fs.existsSync(source))
            throw Error("Incorrect imported-file deletion");
        if (errors.length) throw Error(errors.join("\n"));
        console.log(
            "Passed: native import, live Hugging Face search, engine load, capability discovery, local-only chat, streaming, usage, no UI script errors.",
        );
    } catch (error) {
        if (app) {
            const page = await app.firstWindow();
            console.error(await page.locator("body").innerText());
            await page.screenshot({ path: "/private/tmp/arx-local-failure.png" });
        }
        throw error;
    } finally {
        if (app) await app.close();
    }
})();
