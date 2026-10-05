const { dialog, ipcMain } = require("electron");

function registerEnvironment(eventWindow) {
    ipcMain.handle("arx:choose-model", async (event) => {
        const window = eventWindow(event);
        if (!window) throw new Error("Invalid desktop request");
        const result = await dialog.showOpenDialog(window, {
            title: "Import model file",
            properties: ["openFile"],
            filters: [{ name: "GGUF models", extensions: ["gguf"] }],
        });
        return result.canceled ? null : result.filePaths[0] || null;
    });
    ipcMain.handle("arx:choose-directory", async (event) => {
        const window = eventWindow(event);
        if (!window) throw new Error("Invalid desktop request");
        const result = await dialog.showOpenDialog(window, {
            title: "Working directory",
            properties: ["openDirectory"],
        });
        return result.canceled ? null : result.filePaths[0] || null;
    });
}

module.exports = { registerEnvironment };
