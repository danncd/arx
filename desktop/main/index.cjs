const { app, clipboard, ipcMain, Menu, session, shell, protocol } = require("electron");
const { configureProfile } = require("./profile.cjs");
const { registerMedia } = require("./ipc/media.cjs");
const { registerImages } = require("./ipc/images.cjs");
const { registerLinks } = require("./ipc/links.cjs");
const { registerEnvironment } = require("./ipc/environment.cjs");
const { Backend } = require("./backend/process.cjs");
const { registerBackend } = require("./ipc/backend.cjs");
const path = require("node:path");
const {
    createWindow,
    eventWindow,
    getChatWindows,
    installWindowControls,
} = require("./window.cjs");

protocol.registerSchemesAsPrivileged([
    {
        scheme: "arx-media",
        privileges: { standard: true, secure: true, supportFetchAPI: true, stream: true },
    },
]);
app.setName("Arx");
const profile = configureProfile(app);
process.env.ARX_MODELS_DIR ||= path.join(app.getPath("appData"), "Arx", "local-models");

const binary = path.join(
    app.isPackaged ? process.resourcesPath : path.join(app.getAppPath(), ".build"),
    "backend",
    process.platform === "win32" ? "arx-desktop.exe" : "arx-desktop",
);
const backend = new Backend(binary, profile);
let quitting = false;
app.on("before-quit", (event) => {
    if (quitting) return;
    event.preventDefault();
    quitting = true;
    void backend.stop().finally(() => app.quit());
});

if (!app.requestSingleInstanceLock()) {
    app.quit();
} else {
    app.on("second-instance", () => {
        const window = getChatWindows()[0];
        if (window) {
            if (window.isMinimized()) window.restore();
            window.show();
            window.focus();
        }
    });
    app.whenReady()
        .then(() => {
            session.defaultSession.setPermissionRequestHandler(
                (_contents, _permission, respond) => respond(false),
            );
            session.defaultSession.setPermissionCheckHandler(() => false);
            installWindowControls();
            registerBackend(backend, eventWindow, getChatWindows);
            registerEnvironment(eventWindow);
            registerImages(backend, eventWindow, profile);
            registerMedia(backend, eventWindow);
            backend.start();
            ipcMain.handle("arx:copy", (event, text) => {
                if (
                    !eventWindow(event) ||
                    typeof text !== "string" ||
                    text.length > 4_000_000
                )
                    throw new Error("Invalid clipboard request");
                clipboard.writeText(text);
            });
            registerLinks(eventWindow, backend, { ipcMain, shell });
            Menu.setApplicationMenu(
                Menu.buildFromTemplate([
                    { role: "appMenu" },
                    { role: "editMenu" },
                    { role: "viewMenu" },
                    { role: "windowMenu" },
                ]),
            );
            createWindow(app.getAppPath());
            app.on("activate", () => {
                if (!getChatWindows().length) createWindow(app.getAppPath());
            });
        })
        .catch((error) => {
            console.error(error);
            app.exit(1);
        });
    app.on("window-all-closed", () => {
        if (process.platform !== "darwin") app.quit();
    });
}
