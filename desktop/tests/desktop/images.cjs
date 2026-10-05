const { desktopEnvironment, launchDesktop } = require("../support/desktop.cjs");
const { expect } = require("@playwright/test");
const fs = require("node:fs");
const os = require("node:os");
const path = require("node:path");
const project = path.resolve(__dirname, "../..");
const profile = fs.mkdtempSync(path.join(os.tmpdir(), "arx-images-"));
const original = path.join(profile, "sample.png");
fs.writeFileSync(
    original,
    Buffer.from(
        "iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mNk+A8AAQUBAScY42YAAAAASUVORK5CYII=",
        "base64",
    ),
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
    try {
        let page = await launch();
        await app.evaluate(({ dialog }, file) => {
            dialog.showOpenDialog = async () => ({
                canceled: false,
                filePaths: [file],
            });
        }, original);
        const images = await page.evaluate(() => window.arxDesktop.chooseImages());
        if (images.length !== 1 || images[0].name !== "sample.png")
            throw Error("Native picker lost image");
        fs.unlinkSync(original);
        await page.evaluate(async (images) => {
            const source = await window.arxDesktop.readImage(images[0].id);
            if (!source.startsWith("data:image/png;base64,"))
                throw Error("Saved image could not be read");
            await window.arxDesktop.request("save_view", {
                key: "images:new",
                value: JSON.stringify(images),
            });
        }, images);
        await page.reload();
        await expect(page.locator(".composer .image-attachments img")).toHaveAttribute(
            "alt",
            "sample.png",
        );
        await page
            .getByRole("button", { name: "Remove sample.png", exact: true })
            .click();
        await expect(page.locator(".composer .image-attachments")).toHaveCount(0);
        await page.reload();
        await expect(page.locator(".composer .image-attachments")).toHaveCount(0);
        await app.close();
        app = null;
        fs.appendFileSync(
            path.join(profile, "transcript.jsonl"),
            JSON.stringify({
                conversation: "images",
                id: "user",
                role: "user",
                images,
                text: "",
                offset: 0,
                status: "done",
                at: new Date().toISOString(),
            }) + "\n",
        );
        page = await launch();
        await expect(
            page.locator(".message.user > .image-attachments img"),
        ).toHaveAttribute("alt", "sample.png");
        const valid = await page
            .locator(".message.user > .image-attachments img")
            .evaluate((image) => image.complete && image.naturalWidth > 0);
        if (!valid) throw Error("Saved chat image did not load");
        if (errors.length) throw Error(errors.join("\n"));
        console.log(
            "PASS: native image import, independent saved copy, draft restoration/removal, saved chat replay.",
        );
    } finally {
        if (app) await app.close();
        fs.rmSync(profile, { recursive: true, force: true });
    }
})().catch((error) => {
    console.error(error);
    process.exitCode = 1;
});
