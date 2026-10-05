const { desktopEnvironment, launchDesktop } = require("../support/desktop.cjs");
const { expect } = require("@playwright/test");
const fs = require("node:fs"),
    os = require("node:os"),
    path = require("node:path");
const project = path.resolve(__dirname, "../.."),
    profile = fs.mkdtempSync(path.join(os.tmpdir(), "arx-integrations-"));
const executable = path.join(profile, "fixture-mcp");
fs.writeFileSync(
    executable,
    `#!/usr/bin/python3
import json,sys
for line in sys.stdin:
 r=json.loads(line)
 if 'id' not in r: continue
 if r['method']=='initialize': result={'protocolVersion':'2025-11-25','capabilities':{'tools':{}},'serverInfo':{'name':'test','version':'1'}}
 elif r['method']=='tools/list': result={'tools':[{'name':'edit_note','description':'Edit a note','inputSchema':{'type':'object','properties':{'noteId':{'type':'string'}},'required':['noteId']},'annotations':{'readOnlyHint':False}}]}
 elif r['method']=='tools/call': result={'content':[{'type':'text','text':'updated'}]}
 else: result={}
 print(json.dumps({'jsonrpc':'2.0','id':r['id'],'result':result}),flush=True)
`,
    { mode: 0o700 },
);
const env = desktopEnvironment(profile);
(async () => {
    let app;
    const errors = [];
    async function launch() {
        app = await launchDesktop(project, env);
        const p = await app.firstWindow();
        p.on("pageerror", (e) => errors.push(e.message));
        await p.getByRole("textbox", { name: "Message", exact: true }).waitFor();
        return p;
    }
    async function request(p, method, params) {
        return p.evaluate(
            ({ method, params }) => window.arxDesktop.request(method, params),
            { method, params },
        );
    }
    try {
        let page = await launch();
        await page.getByRole("button", { name: "Settings", exact: true }).click();
        const dialog = page.getByRole("dialog");
        const normalTop = await dialog
            .locator(".settings-pane h3")
            .evaluate((h) => h.getBoundingClientRect().top);
        await dialog.getByRole("button", { name: "MCPs", exact: true }).click();
        if (
            (await dialog
                .locator(".settings-pane h3")
                .evaluate((h) => h.getBoundingClientRect().top)) !== normalTop
        )
            throw Error("MCP heading misaligned");
        await dialog.getByRole("button", { name: "Add server…", exact: true }).click();
        await dialog.getByLabel("Name", { exact: true }).fill("Test notes");
        await dialog.getByLabel("Server ID", { exact: true }).fill("test-notes");
        await dialog.getByLabel("Executable", { exact: true }).fill(executable);
        await dialog
            .getByRole("button", { name: "Test connection", exact: true })
            .click();
        await expect(dialog.getByRole("status")).toContainText("Connection test passed");
        await dialog.getByRole("button", { name: "Add server", exact: true }).click();
        await expect(dialog.locator(".integration-heading h3")).toHaveText("Test notes");
        await dialog.getByRole("button", { name: "Reconnect", exact: true }).click();
        await expect(dialog.locator(".integration-heading")).toContainText("Connected");
        await dialog.getByRole("button", { name: "Tools", exact: true }).click();
        await dialog.getByRole("button", { name: "edit_note", exact: true }).click();
        await expect(dialog.locator(".integration-code")).toContainText("noteId");
        await dialog.getByRole("button", { name: "Skills", exact: true }).click();
        await expect(dialog.locator(".settings-pane h3")).toHaveText("Skills");
        if (
            (await dialog
                .locator(".settings-pane h3")
                .evaluate((h) => h.getBoundingClientRect().top)) !== normalTop
        )
            throw Error("Skill heading misaligned");
        await dialog.getByRole("button", { name: "Add skill…", exact: true }).click();
        await dialog.getByLabel("Skill ID", { exact: true }).fill("review");
        await dialog
            .getByRole("textbox", { name: "SKILL.md", exact: true })
            .fill(
                "---\nname: review\ndescription: Review code changes.\n---\n\nRead changed files before reporting findings.\n",
            );
        await dialog.getByRole("button", { name: "Validate", exact: true }).click();
        await expect(dialog.getByRole("status")).toContainText("Valid skill");
        await dialog.getByRole("button", { name: "Create skill", exact: true }).click();
        await expect(dialog.locator(".integration-heading h3")).toHaveText("review");
        await dialog.getByRole("button", { name: "Files", exact: true }).click();
        await expect(dialog.locator(".directory-value")).toContainText("skills/review");
        await dialog.getByRole("button", { name: "Instructions", exact: true }).click();
        await dialog
            .getByRole("textbox", { name: "SKILL.md", exact: true })
            .fill(
                "---\nname: review\ndescription: Review code changes.\n---\n\nCheck tests as well as changed files.\n",
            );
        await dialog.getByRole("button", { name: "Save changes", exact: true }).click();
        await expect(
            dialog.getByRole("textbox", { name: "SKILL.md", exact: true }),
        ).toHaveValue(/Check tests/);
        await dialog.getByRole("button", { name: "Close settings", exact: true }).click();
        await request(page, "permissions.configure", {
            mode: "folders",
            roots: [profile],
        });
        await page.evaluate(() => {
            window.integrationCall = window.arxDesktop.request("tools.run", {
                id: "integration-denied",
                name: "mcp",
                arguments: {
                    operation: "call",
                    server: "test-notes",
                    name: "edit_note",
                    arguments: { noteId: "note-42" },
                },
            });
        });
        await expect(
            page.getByRole("region", { name: "Permission request" }),
        ).toContainText("Call MCP tool?");
        await page.getByRole("button", { name: "Review changes", exact: true }).click();
        await expect(page.locator(".approval-details")).toContainText("test-notes");
        await expect(page.locator(".approval-details")).toContainText("note-42");
        await page.getByRole("button", { name: "Deny", exact: true }).click();
        const denied = await page.evaluate(() => window.integrationCall);
        if (!denied.failed) throw Error("Denied call succeeded");
        await request(page, "permissions.configure", { mode: "full", roots: [] });
        const result = await request(page, "tools.run", {
            id: "integration-call",
            name: "mcp",
            arguments: {
                operation: "call",
                server: "test-notes",
                name: "edit_note",
                arguments: { noteId: "note-42" },
            },
        });
        if (result.failed || result.text !== "updated") throw Error("MCP call failed");
        let states = await request(page, "mcp.state");
        if (states.find((s) => s.id === "test-notes").usage.count !== 2)
            throw Error("Usage wrong");
        await app.close();
        app = null;
        page = await launch();
        states = await request(page, "mcp.state");
        if (states.find((s) => s.id === "test-notes").usage.count !== 2)
            throw Error("Usage lost after restart");
        const skills = await request(page, "skills.state");
        if (!skills.some((s) => s.id === "review"))
            throw Error("Skill lost after restart");
        await page.getByRole("button", { name: "Settings", exact: true }).click();
        await page
            .getByRole("dialog")
            .getByRole("button", { name: "Skills", exact: true })
            .click();
        await page.getByRole("button", { name: /review.*0 times used/ }).click();
        await page.getByRole("button", { name: "Remove skill…", exact: true }).click();
        await page.getByRole("button", { name: "Remove", exact: true }).click();
        await expect(page.getByRole("dialog").locator(".settings-pane h3")).toHaveText(
            "Skills",
        );
        if (!fs.existsSync(path.join(profile, "skills", "review", "SKILL.md")))
            throw Error("Removal deleted source");
        if (errors.length) throw Error(errors.join("\n"));
        console.log(
            "PASS: real Settings CRUD, aligned headings, MCP schemas/approvals/calls, durable usage, skill validation/editing, and source-preserving removal.",
        );
    } finally {
        if (app) await app.close();
        fs.rmSync(profile, { recursive: true, force: true });
    }
})().catch((e) => {
    console.error(e);
    process.exitCode = 1;
});
