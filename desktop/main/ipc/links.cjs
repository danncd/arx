const fs = require("node:fs/promises");
const os = require("node:os");
const path = require("node:path");
const { fileURLToPath } = require("node:url");

function resolveLink(raw, directory) {
    if (typeof raw !== "string" || raw.length > 16384 || /[\x00-\x1f\x7f]/.test(raw))
        throw new Error("This link is invalid.");
    let value = raw.trim();
    if (!value) throw new Error("This link is empty.");
    if (value.startsWith("//")) value = `https:${value}`;
    else if (
        !/^[^/]+\.(?:md|txt|json|[cm]?[jt]sx?|py|go|rs|html|css|yaml|yml|toml|sh|pdf|png|jpe?g|svg)(?:[:#]|$)/i.test(
            value,
        ) &&
        /^(?:www\.|[\w-]+(?:\.[\w-]+)+(?::\d+)?(?:\/|$))/.test(value)
    )
        value = `https://${value}`;
    if (/^https?:/i.test(value)) {
        let url;
        try {
            url = new URL(value);
        } catch {
            throw new Error("This website address is incomplete.");
        }
        if (!url.hostname || url.username || url.password)
            throw new Error("This website address is invalid.");
        return { kind: "web", value: url.href };
    }
    if (/^mailto:/i.test(value)) {
        const url = new URL(value);
        if (!url.pathname.includes("@"))
            throw new Error("This email address is incomplete.");
        return { kind: "web", value: url.href };
    }
    if (/^file:/i.test(value)) {
        try {
            value = fileURLToPath(value);
        } catch {
            throw new Error("This file link is invalid.");
        }
    } else if (/^[a-z][a-z\d+.-]*:/i.test(value) && !/:\d+(?::\d+)?$/.test(value)) {
        throw new Error("This type of link is not supported.");
    } else {
        try {
            value = decodeURIComponent(value);
        } catch {
            throw new Error("This file link is invalid.");
        }
    }
    value = value.replace(/(?::\d+(?::\d+)?|#L\d+(?:-L?\d+)?)$/, "");
    if (value.startsWith("~/")) value = path.join(os.homedir(), value.slice(2));
    if (!path.isAbsolute(value)) {
        if (!directory)
            throw new Error("Choose a working directory to locate this file.");
        value = path.resolve(directory, value);
    }
    return { kind: "file", value };
}

function registerLinks(eventWindow, backend, { ipcMain, shell }) {
    ipcMain.handle("arx:open-external", async (event, raw) => {
        if (!eventWindow(event)) throw new Error("Invalid desktop request");
        let target = resolveLink(raw, "/");
        if (target.kind === "file") {
            const snapshot = await backend.request("snapshot");
            target = resolveLink(raw, snapshot.settings.directory);
            try {
                await fs.stat(target.value);
            } catch {
                throw new Error(`File not found: ${path.basename(target.value)}`);
            }
            if (path.extname(target.value).toLowerCase() === ".html") {
                const error = await shell.openPath(target.value);
                if (error) throw new Error("Your browser could not open this file.");
            } else {
                shell.showItemInFolder(target.value);
            }
            return;
        }
        try {
            await shell.openExternal(target.value);
        } catch {
            throw new Error(
                target.value.startsWith("mailto:")
                    ? "No email app could open this address."
                    : "Your browser could not open this address.",
            );
        }
    });
}

module.exports = { registerLinks, resolveLink };
