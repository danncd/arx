const { BrowserWindow, ipcMain, screen, systemPreferences } = require("electron");
const path = require("node:path");
const { pathToFileURL } = require("node:url");

let pageURL;
let drag = null;
const restoreBounds = new WeakMap();
const chatWindows = new Set();

function getChatWindows() {
    return [...chatWindows].filter((window) => !window.isDestroyed());
}

function eventWindow(event) {
    const window = BrowserWindow.fromWebContents(event.sender);
    if (
        !window ||
        event.senderFrame !== window.webContents.mainFrame ||
        event.senderFrame.url !== pageURL
    )
        return null;
    return window;
}

function validPoint(point) {
    return point && Number.isFinite(point.x) && Number.isFinite(point.y);
}

function installWindowControls() {
    ipcMain.handle(
        "arx:fullscreen",
        (event) => eventWindow(event)?.isFullScreen() ?? false,
    );
    ipcMain.on("arx:window-drag-start", (event, point) => {
        const window = eventWindow(event);
        if (!window || window.isFullScreen() || !validPoint(point)) return;
        drag = { window, sender: event.sender.id, origin: window.getBounds(), point };
    });
    ipcMain.on("arx:window-drag-move", (event, point) => {
        if (
            !drag ||
            drag.sender !== event.sender.id ||
            !eventWindow(event) ||
            !validPoint(point)
        )
            return;
        drag.window.setPosition(
            Math.round(drag.origin.x + point.x - drag.point.x),
            Math.round(drag.origin.y + point.y - drag.point.y),
            false,
        );
    });
    ipcMain.on("arx:window-drag-end", (event) => {
        if (drag?.sender === event.sender.id) drag = null;
    });
    ipcMain.handle("arx:double-click-titlebar", (event) => {
        const window = eventWindow(event);
        if (!window || window.isFullScreen()) return;
        const action = systemPreferences.getUserDefault(
            "AppleActionOnDoubleClick",
            "string",
        );
        if (action === "Minimize") return window.minimize();
        if (action === "None") return;
        const previous = restoreBounds.get(window);
        if (previous) {
            restoreBounds.delete(window);
            window.setBounds(previous, true);
        } else {
            const bounds = window.getBounds();
            restoreBounds.set(window, bounds);
            window.setBounds(screen.getDisplayMatching(bounds).workArea, true);
        }
    });
}

function createWindow(appPath) {
    const file = path.join(appPath, ".build", "renderer", "index.html");
    const development = process.env.ARX_DEV_URL;
    if (development && development !== "http://127.0.0.1:3006/")
        throw new Error("Invalid development address");
    pageURL = development || pathToFileURL(file).href;
    const window = new BrowserWindow({
        title: "Arx",
        width: 1120,
        height: 780,
        minWidth: 720,
        minHeight: 520,
        show: false,
        backgroundColor: "#ffffff",
        titleBarStyle: "hidden",
        trafficLightPosition: { x: 17, y: 20 },
        webPreferences: {
            preload: path.join(__dirname, "../preload/index.cjs"),
            contextIsolation: true,
            nodeIntegration: false,
            sandbox: true,
        },
    });
    chatWindows.add(window);

    let recoveredRenderer = false;
    window.webContents.on("render-process-gone", (_event, details) => {
        if (window.isDestroyed() || recoveredRenderer || details.reason === "clean-exit")
            return;
        recoveredRenderer = true;
        window.webContents.reload();
    });

    window.webContents.on(
        "did-fail-load",
        (_event, code, description, url, isMainFrame) => {
            if (!isMainFrame) return;
            console.error(
                `the renderer failed to load (${code} ${description}) from ${url}`,
            );
        },
    );
    const show = () => {
        if (!window.isDestroyed()) window.show();
    };
    const fallback = setTimeout(show, 2500);
    const ready = (event) => {
        if (eventWindow(event) !== window) return;
        clearTimeout(fallback);
        show();
    };
    ipcMain.on("arx:ready", ready);
    window.webContents.setWindowOpenHandler(() => ({ action: "deny" }));
    window.webContents.on("will-navigate", (event) => event.preventDefault());
    window.webContents.on("will-attach-webview", (event) => event.preventDefault());
    const fullscreenChanged = () => {
        if (drag?.window === window) drag = null;
        window.webContents.send("arx:fullscreen", window.isFullScreen());
    };
    window.on("enter-full-screen", fullscreenChanged);
    window.on("leave-full-screen", fullscreenChanged);
    window.on("blur", () => {
        if (drag?.window === window) drag = null;
    });
    window.on("closed", () => {
        chatWindows.delete(window);
        clearTimeout(fallback);
        ipcMain.removeListener("arx:ready", ready);
        if (drag?.window === window) drag = null;
    });
    if (development) void window.loadURL(development);
    else void window.loadFile(file);
    return window;
}

module.exports = { createWindow, installWindowControls, eventWindow, getChatWindows };
