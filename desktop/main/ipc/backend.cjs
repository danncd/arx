const methods = require("../../contracts/methods.generated.cjs");
const { ipcMain } = require("electron");

function registerBackend(backend, eventWindow, getWindows) {
    const validate = (event) => {
        if (!eventWindow(event)) throw new Error("Invalid desktop request");
    };
    ipcMain.handle("arx:backend-status", (event) => {
        validate(event);
        return backend.status;
    });
    ipcMain.handle("arx:backend-request", async (event, method, params) => {
        validate(event);
        if (!methods[method]?.renderer) throw new Error("Unsupported backend request");
        try {
            const result = await backend.request(method, params, methods[method].timeout);
            return { result };
        } catch (error) {
            return { error: error.message || "Backend request failed" };
        }
    });
    ipcMain.handle("arx:backend-restart", (event) => {
        validate(event);
        return backend.restart();
    });
    backend.on("generation", (event) => {
        for (const window of getWindows()) {
            if (!window.isDestroyed()) window.webContents.send("arx:generation", event);
        }
    });
    backend.on("local", (state) => {
        for (const window of getWindows()) {
            if (!window.isDestroyed()) window.webContents.send("arx:local", state);
        }
    });
    backend.on("chat", (event) => {
        for (const window of getWindows()) {
            if (!window.isDestroyed()) window.webContents.send("arx:chat", event);
        }
    });
    backend.on("permission", (request) => {
        for (const window of getWindows()) {
            if (!window.isDestroyed()) window.webContents.send("arx:permission", request);
        }
    });
    backend.on("status", (status) => {
        for (const window of getWindows()) {
            if (!window.isDestroyed())
                window.webContents.send("arx:backend-status", status);
        }
    });
}

module.exports = { registerBackend };
