const { protocol, ipcMain, dialog } = require("electron");
const { createReadStream } = require("node:fs");
const { copyFile } = require("node:fs/promises");
const { Readable } = require("node:stream");

function registerMedia(backend, eventWindow) {
    protocol.handle("arx-media", async (request) => {
        try {
            const address = new URL(request.url);
            const id = address.pathname.slice(1);
            if (address.hostname !== "artifact" || !/^[a-f0-9]{32}$/.test(id))
                return new Response(null, { status: 404 });
            if (!["GET", "HEAD"].includes(request.method))
                return new Response(null, { status: 405 });
            const { artifact, path } = await backend.request("media.resolve", { id });
            const headers = {
                "Content-Type": artifact.mime,
                "Accept-Ranges": "bytes",
                "Cache-Control": "private, max-age=31536000, immutable",
            };
            let start = 0;
            let end = artifact.size - 1;
            const range = request.headers.get("range");
            if (range) {
                const match = /^bytes=(\d*)-(\d*)$/.exec(range);
                if (!match || (!match[1] && !match[2]))
                    return new Response(null, { status: 416 });
                if (!match[1]) start = Math.max(0, artifact.size - Number(match[2]));
                else {
                    start = Number(match[1]);
                    if (match[2]) end = Math.min(end, Number(match[2]));
                }
                if (
                    !Number.isSafeInteger(start) ||
                    !Number.isSafeInteger(end) ||
                    start > end ||
                    start >= artifact.size
                )
                    return new Response(null, {
                        status: 416,
                        headers: { "Content-Range": `bytes */${artifact.size}` },
                    });
                headers["Content-Range"] = `bytes ${start}-${end}/${artifact.size}`;
            }
            headers["Content-Length"] = String(end - start + 1);
            const stream =
                request.method === "HEAD"
                    ? null
                    : Readable.toWeb(createReadStream(path, { start, end }));
            return new Response(stream, { status: range ? 206 : 200, headers });
        } catch {
            return new Response(null, { status: 404 });
        }
    });
    ipcMain.handle("arx:save-media", async (event, id) => {
        const window = eventWindow(event);
        if (!window || typeof id !== "string" || !/^[a-f0-9]{32}$/.test(id))
            throw new Error("Invalid media request");
        const { artifact, path } = await backend.request("media.resolve", { id });
        const selected = await dialog.showSaveDialog(window, {
            defaultPath: artifact.name,
        });
        if (selected.canceled || !selected.filePath) return false;
        await copyFile(path, selected.filePath);
        return true;
    });
}

module.exports = { registerMedia };
