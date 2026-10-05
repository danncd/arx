const { dialog, ipcMain, nativeImage } = require("electron");
const fs = require("node:fs/promises");
const path = require("node:path");
const { randomBytes } = require("node:crypto");

function registerImages(backend, eventWindow, profile) {
    const directory = path.join(profile, "attachment-imports");
    const validate = (event) => {
        if (!eventWindow(event)) throw new Error("Invalid desktop request");
    };
    ipcMain.handle("arx:choose-images", async (event) => {
        validate(event);
        const result = await dialog.showOpenDialog(eventWindow(event), {
            title: "Attach images",
            properties: ["openFile", "multiSelections"],
            filters: [
                { name: "Images", extensions: ["png", "jpg", "jpeg", "webp", "gif"] },
            ],
        });
        if (result.canceled) return [];
        if (result.filePaths.length > 4)
            throw new Error("Attach up to 4 images per message");
        const prepared = [];
        for (const file of result.filePaths) {
            const stat = await fs.stat(file);
            if (!stat.isFile() || stat.size > 32 * 1024 * 1024)
                throw new Error("Choose an image smaller than 32 MB");
            let image = nativeImage.createFromPath(file);
            if (image.isEmpty()) throw new Error("This image format could not be opened");
            const size = image.getSize();
            if (Math.max(size.width, size.height) > 8192)
                throw new Error("Choose an image smaller than 8192 pixels per side");
            if (Math.max(size.width, size.height) > 2048) {
                image = image.resize(
                    size.width >= size.height ? { width: 2048 } : { height: 2048 },
                );
            }
            let data = image.toPNG();
            while (data.length > 2 * 1024 * 1024) {
                image = image.resize({
                    width: Math.max(1, Math.floor(image.getSize().width * 0.8)),
                });
                data = image.toPNG();
            }
            prepared.push({
                name: path.basename(file).slice(0, 255),
                data,
            });
        }
        await fs.mkdir(directory, { recursive: true, mode: 0o700 });
        const attachments = [];
        for (const image of prepared) {
            const file = path.join(directory, `${randomBytes(16).toString("hex")}.png`);
            try {
                await fs.writeFile(file, image.data, { mode: 0o600, flag: "wx" });
                attachments.push(
                    await backend.request("attachments.import", {
                        path: file,
                        name: image.name,
                    }),
                );
            } finally {
                await fs.rm(file, { force: true });
            }
        }
        return attachments;
    });
    ipcMain.handle("arx:read-image", async (event, id) => {
        validate(event);
        if (typeof id !== "string" || !/^[a-f0-9]{64}$/.test(id))
            throw new Error("Invalid image attachment");
        return backend.request("attachments.read", { id });
    });
}

module.exports = { registerImages };
