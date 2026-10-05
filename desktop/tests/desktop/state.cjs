const { desktopEnvironment, launchDesktop } = require("../support/desktop.cjs");
const { expect } = require("@playwright/test");
const fs = require("node:fs");
const os = require("node:os");
const path = require("node:path");
const { execFileSync } = require("node:child_process");
const project = path.resolve(__dirname, "../..");
const profile = fs.mkdtempSync(path.join(os.tmpdir(), "arx-phase2-ui-"));
const chosen = fs.mkdtempSync(path.join(os.tmpdir(), "arx-phase2-directory-"));
const chunk = (id, role, text, extra = {}) => ({
    conversation: "recent",
    id,
    role,
    offset: 0,
    text,
    reasoning: "",
    reasoning_offset: 0,
    tools: null,
    usage: null,
    status: "done",
    failed: false,
    reason: "",
    at: "2026-09-26T12:00:00Z",
    ...extra,
});
const chunks = [
    chunk("old-q", "user", "Older session", {
        conversation: "older",
        at: "2026-09-25T12:00:00Z",
    }),
    chunk("old-a", "assistant", "Older reply", {
        conversation: "older",
        at: "2026-09-25T12:00:01Z",
    }),
];
for (let i = 0; i < 70; i++)
    chunks.push(chunk(`m${i}`, "user", i === 0 ? "Recent session" : `Message ${i}`));
chunks.push(
    chunk("answer", "assistant", "Hello 🌱", {
        reasoning: "Thought 🌱",
        status: "working",
    }),
);
chunks.push(
    chunk("answer", "assistant", " Danny", {
        offset: Buffer.byteLength("Hello 🌱"),
        reasoning_offset: Buffer.byteLength("Thought 🌱"),
        usage: { input: 42, output: 9, cached: 0, reasoning: 3 },
        tools: [
            {
                id: "t",
                name: "files",
                summary: "Read notes",
                status: "done",
                arguments: '{"path":"notes.txt"}',
                result: "Saved tool result",
                failed: false,
            },
        ],
    }),
);
fs.writeFileSync(
    path.join(profile, "transcript.jsonl"),
    chunks.map((v) => JSON.stringify(v)).join("\n") + "\n",
    { mode: 0o600 },
);
fs.writeFileSync(
    path.join(profile, "views.json"),
    JSON.stringify({
        version: 1,
        views: {
            sidebar: JSON.stringify({ open: true, width: 220 }),
            "draft:recent": "Saved draft",
        },
    }),
);
fs.writeFileSync(
    path.join(profile, "settings.json"),
    JSON.stringify({
        version: 1,
        run: { model: "deepseek-chat", effort: "high" },
        directory: os.homedir(),
    }),
);
const env = desktopEnvironment(profile);
(async () => {
    let app;
    const errors = [];
    const launch = async () => {
        app = await launchDesktop(project, env);
        const page = await app.firstWindow();
        page.on("pageerror", (e) => errors.push(e.message));
        await page.getByRole("textbox", { name: "Message", exact: true }).waitFor();
        return page;
    };
    try {
        let page = await launch();
        await expect(page.getByText("Hello 🌱 Danny", { exact: true })).toBeVisible();
        await expect(
            page.getByRole("textbox", { name: "Message", exact: true }),
        ).toHaveValue("Saved draft");
        await expect(page.locator(".app-shell")).toHaveCSS("--sidebar-width", "220px");
        await expect(
            page.getByRole("button", { name: "Send message", exact: true }),
        ).toBeDisabled();
        await page.getByText("Think", { exact: true }).click();
        await expect(page.getByText("Thought 🌱", { exact: true })).toBeVisible();
        await page.locator(".tool-group-summary").click();
        await page.locator(".tool-file > summary").click();
        await expect(page.getByText("Saved tool result", { exact: true })).toBeVisible();
        await page.getByRole("button", { name: "Earlier messages", exact: true }).click();
        await expect(page.getByText("Message 1", { exact: true })).toHaveCount(1);
        await expect(
            page.getByRole("button", { name: "Earlier messages", exact: true }),
        ).toHaveCount(0);
        await page
            .getByRole("textbox", { name: "Message", exact: true })
            .fill("Draft persisted 🌱");
        await page.getByRole("separator", { name: "Sidebar width" }).focus();
        await page.keyboard.press("End");
        const edge = page.getByRole("separator", { name: "Sidebar width" });
        await expect.poll(async () => (await edge.boundingBox()).x).toBe(340);
        await page.waitForTimeout(250);
        const collapseFrom = await edge.boundingBox();
        await page.mouse.move(
            collapseFrom.x + collapseFrom.width / 2,
            collapseFrom.y + 80,
        );
        await page.mouse.down();
        await page.mouse.move(120, collapseFrom.y + 80, { steps: 15 });
        await page.mouse.up();
        await expect(
            page.getByRole("button", { name: "Show sidebar", exact: true }),
        ).toBeVisible();
        await page.getByRole("button", { name: "Show sidebar", exact: true }).click();
        await expect(page.locator(".app-shell")).toHaveCSS("--sidebar-width", "340px");
        await expect.poll(async () => (await edge.boundingBox()).x).toBe(340);
        await page.waitForTimeout(250);
        const narrowFrom = await edge.boundingBox();
        await page.mouse.move(narrowFrom.x + narrowFrom.width / 2, narrowFrom.y + 80);
        await page.mouse.down();
        await page.mouse.move(
            narrowFrom.x + narrowFrom.width / 2 - 135,
            narrowFrom.y + 80,
            { steps: 10 },
        );
        await page.mouse.up();
        const narrowWidth = await edge.getAttribute("aria-valuenow");
        if (Number(narrowWidth) < 200 || Number(narrowWidth) > 220)
            throw Error("Narrow sidebar drag failed");
        await page.getByRole("button", { name: "Hide sidebar", exact: true }).click();
        await page.getByRole("button", { name: "Show sidebar", exact: true }).click();
        await expect(edge).toHaveAttribute("aria-valuenow", narrowWidth);
        await edge.focus();
        await page.keyboard.press("End");
        await page.getByRole("button", { name: "Older session", exact: true }).click();
        await expect(page.getByText("Older reply", { exact: true })).toBeVisible();
        await page
            .getByRole("textbox", { name: "Message", exact: true })
            .fill("Older draft");
        await page.getByRole("button", { name: "Recent session", exact: true }).click();
        await expect(
            page.getByRole("textbox", { name: "Message", exact: true }),
        ).toHaveValue("Draft persisted 🌱");
        await page.getByRole("button", { name: "Settings", exact: true }).click();
        await app.evaluate(({ dialog }, directory) => {
            dialog.showOpenDialog = async () => ({
                canceled: false,
                filePaths: [directory],
            });
        }, chosen);
        await page.getByRole("button", { name: "Choose…", exact: true }).click();
        await expect(page.locator(".directory-value p")).toHaveText(
            fs.realpathSync(chosen),
        );
        await page.getByRole("combobox", { name: "On startup", exact: true }).click();
        await page.getByRole("option", { name: "New session", exact: true }).click();

        await page.getByRole("button", { name: "Close settings", exact: true }).click();
        await page
            .getByRole("button", { name: /New chat/ })
            .first()
            .click();
        await page
            .getByRole("textbox", { name: "Message", exact: true })
            .fill("Unsent new session");
        await app.close();
        app = null;
        page = await launch();
        await expect(
            page.getByRole("textbox", { name: "Message", exact: true }),
        ).toHaveValue("Unsent new session");
        await expect(page.locator(".app-shell")).toHaveCSS("--sidebar-width", "340px");
        await expect(page.getByText("Hello 🌱 Danny", { exact: true })).toHaveCount(0);
        await page.getByRole("button", { name: "Recent session", exact: true }).click();
        await expect(
            page.getByRole("textbox", { name: "Message", exact: true }),
        ).toHaveValue("Draft persisted 🌱");
        const snapshot = await page.evaluate(() => window.arxDesktop.request("snapshot"));
        if (
            snapshot.settings.directory !== fs.realpathSync(chosen) ||
            snapshot.settings.run.effort !== "high"
        )
            throw Error("Settings not restored");
        const child = execFileSync("/bin/ps", ["-axo", "pid,ppid,command"], {
            encoding: "utf8",
        })
            .split("\n")
            .map((l) => l.trim().split(/\s+/))
            .find(
                (p) =>
                    Number(p[1]) === app.process().pid &&
                    p.slice(2).join(" ").endsWith("/.build/backend/arx-desktop"),
            );
        process.kill(Number(child[0]), "SIGKILL");
        await expect(
            page.getByText("Can’t reach backend", { exact: true }),
        ).toBeVisible();
        await page.getByRole("button", { name: "Retry", exact: true }).click();
        await expect(page.getByText("Hello 🌱 Danny", { exact: true })).toBeVisible();
        await expect(
            page.getByRole("textbox", { name: "Message", exact: true }),
        ).toHaveValue("Draft persisted 🌱");
        await page.getByRole("button", { name: "Settings", exact: true }).click();
        await page.getByRole("combobox", { name: "On startup", exact: true }).click();
        await page
            .getByRole("option", { name: "Most recent session", exact: true })
            .click();
        await page.getByRole("button", { name: "Close settings", exact: true }).click();
        await page.reload();
        await expect(page.getByText("Hello 🌱 Danny", { exact: true })).toBeVisible();

        if (errors.length) throw Error(errors.join("\n"));
        console.log(
            "PASS: legacy transcripts, Unicode, thinking/tools/usage, pagination, per-session drafts, startup modes, sidebar width, folder picker, quit/relaunch, reload, crash/retry, no renderer errors",
        );
    } finally {
        if (app) await app.close();
        fs.rmSync(profile, { recursive: true, force: true });
        fs.rmSync(chosen, { recursive: true, force: true });
    }
})().catch((e) => {
    console.error(e);
    process.exit(1);
});
