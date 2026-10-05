const { contextBridge, ipcRenderer } = require("electron");

async function request(method, params) {
    const response = await ipcRenderer.invoke("arx:backend-request", method, params);
    if (response.error) throw new Error(response.error);
    return response.result;
}

contextBridge.exposeInMainWorld("arxDesktop", {
    request,
    onChat: (callback) => {
        const listener = (_event, event) => callback(event);
        ipcRenderer.on("arx:chat", listener);
        return () => ipcRenderer.removeListener("arx:chat", listener);
    },
    onPermission: (callback) => {
        const listener = (_event, request) => callback(request);
        ipcRenderer.on("arx:permission", listener);
        return () => ipcRenderer.removeListener("arx:permission", listener);
    },
    onGeneration: (callback) => {
        const listener = (_event, value) => callback(value);
        ipcRenderer.on("arx:generation", listener);
        return () => ipcRenderer.removeListener("arx:generation", listener);
    },
    saveMedia: (id) => ipcRenderer.invoke("arx:save-media", id),
    onLocal: (callback) => {
        const listener = (_event, state) => callback(state);
        ipcRenderer.on("arx:local", listener);
        return () => ipcRenderer.removeListener("arx:local", listener);
    },
    chooseModel: () => ipcRenderer.invoke("arx:choose-model"),
    chooseImages: () => ipcRenderer.invoke("arx:choose-images"),
    readImage: (id) => ipcRenderer.invoke("arx:read-image", id),
    chooseDirectory: () => ipcRenderer.invoke("arx:choose-directory"),
    backendInfo: () => request("status"),
    restartBackend: () => ipcRenderer.invoke("arx:backend-restart"),
    onBackendStatus: (callback) => {
        let active = true;
        let received = false;
        const listener = (_event, status) => {
            received = true;
            if (active) callback(status);
        };
        ipcRenderer.on("arx:backend-status", listener);
        void ipcRenderer
            .invoke("arx:backend-status")
            .then((status) => {
                if (active && !received) callback(status);
            })
            .catch(() => {
                if (active && !received)
                    callback({ state: "error", error: "Could not connect to backend" });
            });
        return () => {
            active = false;
            ipcRenderer.removeListener("arx:backend-status", listener);
        };
    },
    copy: (text) => ipcRenderer.invoke("arx:copy", text),
    openExternal: (url) => ipcRenderer.invoke("arx:open-external", url),
    beginWindowDrag: (point) => ipcRenderer.send("arx:window-drag-start", point),
    moveWindowDrag: (point) => ipcRenderer.send("arx:window-drag-move", point),
    endWindowDrag: () => ipcRenderer.send("arx:window-drag-end"),
    doubleClickTitleBar: () => ipcRenderer.invoke("arx:double-click-titlebar"),
    signalReady: () => ipcRenderer.send("arx:ready"),
    onFullscreenChanged: (callback) => {
        let active = true;
        let received = false;
        const listener = (_event, fullscreen) => {
            received = true;
            if (active) callback(fullscreen === true);
        };
        ipcRenderer.on("arx:fullscreen", listener);
        void ipcRenderer
            .invoke("arx:fullscreen")
            .then((fullscreen) => {
                if (active && !received) callback(fullscreen === true);
            })
            .catch(() => {});
        return () => {
            active = false;
            ipcRenderer.removeListener("arx:fullscreen", listener);
        };
    },
});
