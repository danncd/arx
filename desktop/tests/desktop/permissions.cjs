const { desktopEnvironment, launchDesktop } = require("../support/desktop.cjs");
const { expect } = require("@playwright/test");
const fs = require("node:fs");
const os = require("node:os");
const path = require("node:path");
const project = path.resolve(__dirname, "../..");
const profile = fs.mkdtempSync(path.join(os.tmpdir(), "arx-permissions-ui-"));
const workspace = fs.mkdtempSync(path.join(os.tmpdir(), "arx-tools-work-"));
const outside = fs.mkdtempSync(path.join(os.tmpdir(), "arx-tools-outside-"));
fs.writeFileSync(
    path.join(profile, "settings.json"),
    JSON.stringify({
        version: 1,
        run: { model: "", effort: "" },
        directory: fs.realpathSync(workspace),
    }),
);
fs.writeFileSync(
    path.join(profile, "transcript.jsonl"),
    JSON.stringify({
        conversation: "permissions-chat",
        id: "user",
        role: "user",
        text: "Permission checks",
        offset: 0,
        reasoning_offset: 0,
        status: "done",
        at: new Date().toISOString(),
    }) + "\n",
);
const env = desktopEnvironment(profile);
(async () => {
    let app;
    const errors = [];
    const launch = async () => {
        app = await launchDesktop(project, env);
        const page = await app.firstWindow();
        page.on("pageerror", (error) => errors.push(error.message));
        await page.getByRole("textbox", { name: "Message", exact: true }).waitFor();
        return page;
    };
    const setDefault = async (page, mode) => {
        await page.getByRole("button", { name: "Settings", exact: true }).click();
        await page.getByRole("button", { name: "Permissions", exact: true }).click();
        await page.getByRole("combobox", { name: "Permission mode" }).click();
        await page.getByRole("option", { name: mode, exact: true }).click();
        await expect(
            page.getByRole("combobox", { name: "Permission mode" }),
        ).toContainText(mode);
        await page.getByRole("button", { name: "Close settings", exact: true }).click();
    };
    const setMode = async (page, mode) => {
        const picker = page.getByRole("combobox", { name: "Permissions", exact: true });
        await picker.click();
        await page.getByRole("option", { name: mode, exact: true }).click();
        await expect(picker).toContainText(mode);
    };
    const start = async (page, id, name, args) =>
        page.evaluate(
            ({ id, name, args }) => {
                window.toolResult = null;
                void window.arxDesktop
                    .request("tools.run", {
                        id,
                        name,
                        arguments: args,
                        conversation: "permissions-chat",
                    })
                    .then((result) => {
                        window.toolResult = result;
                    });
            },
            { id, name, args },
        );
    const result = async (page) => {
        await expect.poll(() => page.evaluate(() => window.toolResult)).not.toBeNull();
        return page.evaluate(() => window.toolResult);
    };
    try {
        let page = await launch();
        const picker = page.getByRole("combobox", { name: "Permissions", exact: true });
        await picker.click();
        const menu = page.getByRole("listbox", { name: "Permissions", exact: true });
        const triggerBox = await picker.boundingBox();
        const menuBox = await menu.boundingBox();
        if (menuBox.y + menuBox.height > triggerBox.y)
            throw Error("Permissions menu did not open upward");
        await page.getByRole("option", { name: "Auto", exact: true }).click();
        await expect(picker).toContainText("Auto");
        const auto = await page.evaluate(() => window.arxDesktop.request("snapshot"));
        if (
            auto.settings.chat_permissions["permissions-chat"].mode !== "folders" ||
            auto.settings.chat_permissions["permissions-chat"].roots.length !== 1
        )
            throw Error("Auto did not select the working folder");
        await picker.click();
        await page.getByRole("option", { name: "Bypass", exact: true }).click();
        await expect(picker).toContainText("Bypass");
        if (
            (await page.evaluate(() => window.arxDesktop.request("snapshot"))).settings
                .chat_permissions["permissions-chat"].mode !== "full"
        )
            throw Error("Bypass did not enable full access");
        await picker.click();
        await page.getByRole("option", { name: "Ask", exact: true }).click();
        await expect(picker).toContainText("Ask");
        await start(page, "deny", "files", {
            operation: "write",
            path: "denied.txt",
            content: "Do not write",
        });
        await page.getByRole("region", { name: "Permission request" }).waitFor();
        const approvalBox = await page.locator(".approval").boundingBox();
        if (approvalBox.height > 80) throw Error("Approval is not compact");
        await page.getByRole("button", { name: "Review changes", exact: true }).click();
        await expect(page.locator(".approval pre")).toHaveText("Do not write");
        if (fs.existsSync(path.join(workspace, "denied.txt")))
            throw Error("Write occurred before approval");
        await page.getByRole("button", { name: "Deny", exact: true }).click();
        if (!(await result(page)).failed) throw Error("Denied write succeeded");
        await start(page, "allow", "files", {
            operation: "write",
            path: "allowed.txt",
            content: "Approved text",
        });
        await page.getByRole("button", { name: "Allow once", exact: true }).click();
        if (
            (await result(page)).failed ||
            fs.readFileSync(path.join(workspace, "allowed.txt"), "utf8") !==
                "Approved text"
        )
            throw Error("Approved write failed");
        await start(page, "read", "files", { operation: "read", path: "allowed.txt" });
        await expect(
            page.getByRole("heading", { name: "Read file?", exact: true }),
        ).toBeVisible();
        await page.getByRole("button", { name: "Allow once", exact: true }).click();
        if (!(await result(page)).text.includes("Approved text"))
            throw Error("Approved read failed");
        await expect(
            page.getByRole("region", { name: "Permission request" }),
        ).toHaveCount(0);
        await start(page, "search", "web", {
            operation: "search",
            query: "Queens College",
        });
        await expect(
            page.getByRole("heading", { name: "Search web?", exact: true }),
        ).toBeVisible();
        await expect(page.locator(".approval-label p")).toHaveText("Queens College");
        await page.getByRole("button", { name: "Deny", exact: true }).click();
        if (!(await result(page)).failed) throw Error("Denied search succeeded");
        await start(page, "reload", "files", {
            operation: "write",
            path: "reload.txt",
            content: "Pending",
        });
        await page.getByRole("region", { name: "Permission request" }).waitFor();
        await page.reload();
        await page.getByRole("button", { name: "Deny", exact: true }).click();
        await expect(
            page.getByRole("region", { name: "Permission request" }),
        ).toHaveCount(0);
        if (fs.existsSync(path.join(workspace, "reload.txt")))
            throw Error("Reload approved pending write");
        await start(page, "bash", "bash", { command: "printf approved-command" });
        await expect(
            page.getByRole("heading", { name: "Run command?", exact: true }),
        ).toBeVisible();
        await page.getByRole("button", { name: "Allow once", exact: true }).click();
        if ((await result(page)).text !== "approved-command")
            throw Error("Command output missing");
        await setMode(page, "Auto");
        await start(page, "outside", "files", {
            operation: "write",
            path: path.join(outside, "blocked"),
            content: "blocked",
        });
        if (!(await result(page)).failed || fs.existsSync(path.join(outside, "blocked")))
            throw Error("Outside file allowed");
        await start(page, "inside", "files", {
            operation: "write",
            path: "inside.txt",
            content: "allowed",
        });
        if ((await result(page)).failed) throw Error("Allowed folder write failed");
        await expect(
            page.getByRole("region", { name: "Permission request" }),
        ).toHaveCount(0);
        await start(page, "bash-outside", "bash", {
            command: "printf blocked > " + JSON.stringify(path.join(outside, "blocked")),
        });
        if (!(await result(page)).failed || fs.existsSync(path.join(outside, "blocked")))
            throw Error("Bash sandbox escape");
        await start(page, "cancel", "bash", {
            command: "printf started > marker; sleep 20",
            timeout_seconds: 30,
        });
        await expect.poll(() => fs.existsSync(path.join(workspace, "marker"))).toBe(true);
        await page.evaluate(() =>
            window.arxDesktop.request("tools.cancel", { id: "cancel" }),
        );
        if (!(await result(page)).failed) throw Error("Cancelled command succeeded");
        await setMode(page, "Bypass");
        await start(page, "full", "files", {
            operation: "write",
            path: path.join(outside, "allowed"),
            content: "full access",
        });
        if ((await result(page)).failed) throw Error("Full access write failed");
        await setDefault(page, "Ask");
        await expect(picker).toContainText("Bypass");
        await expect(picker.locator(".select-option-title svg")).toHaveCount(1);
        await expect(picker).toHaveCSS("color", "rgb(164, 62, 53)");
        await app.close();
        app = null;
        page = await launch();
        await page.getByRole("button", { name: "Settings", exact: true }).click();
        await page.getByRole("button", { name: "Permissions", exact: true }).click();
        await expect(
            page.getByRole("combobox", { name: "Permission mode" }),
        ).toContainText("Ask");
        await page.getByRole("button", { name: "Close settings", exact: true }).click();
        await page
            .getByRole("button", { name: /New chat/ })
            .first()
            .click();
        await expect(
            page.getByRole("combobox", { name: "Permissions", exact: true }),
        ).toContainText("Ask");
        await setDefault(page, "Bypass");
        await expect(
            page.getByRole("combobox", { name: "Permissions", exact: true }),
        ).toContainText("Bypass");
        await page.getByRole("combobox", { name: "Permissions", exact: true }).click();
        await expect(page.locator(".select-menu-title")).toHaveText("Permissions mode");
        await expect(
            page
                .getByRole("listbox", { name: "Permissions", exact: true })
                .locator(".select-option-title svg"),
        ).toHaveCount(3);
        await page.getByRole("option", { name: "Ask", exact: true }).click();
        await page
            .getByRole("button", { name: "Permission checks", exact: true })
            .click();
        await expect(
            page.getByRole("combobox", { name: "Permissions", exact: true }),
        ).toContainText("Bypass");
        await setDefault(page, "Auto");
        await expect(
            page.getByRole("combobox", { name: "Permissions", exact: true }),
        ).toContainText("Bypass");
        await page
            .getByRole("button", { name: /New chat/ })
            .first()
            .click();
        await expect(
            page.getByRole("combobox", { name: "Permissions", exact: true }),
        ).toContainText("Auto");
        if (errors.length) throw Error(errors.join("\n"));
        console.log(
            "PASS: approval review, deny/allow once, read and web approvals, reload pending approval, Bash approval, folder restrictions, cancellation, full access, preference persistence",
        );
    } finally {
        if (app) await app.close();
        for (const directory of [profile, workspace, outside])
            fs.rmSync(directory, { recursive: true, force: true });
    }
})().catch((error) => {
    console.error(error);
    process.exit(1);
});
