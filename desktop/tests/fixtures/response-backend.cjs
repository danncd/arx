module.exports = function responseBackend({ pendingTitle = false } = {}) {
    const at = new Date().toISOString();
    const chunk = (id, role, text) => ({
        conversation: "session",
        id,
        role,
        offset: 0,
        text,
        reasoning: "",
        reasoning_offset: 0,
        tools: null,
        usage: null,
        status: "done",
        failed: false,
        reason: "",
        at,
    });
    const chunks = [
        chunk("user", "user", "Check the files"),
        chunk("first", "assistant", ""),
    ];
    const settings = {
        version: 1,
        run: { model: "test", effort: "low" },
        directory: "/tmp",
        permissions: { mode: "ask", roots: [] },
    };
    const summary = {
        id: "session",
        title: pendingTitle ? "" : "Check the files",
        updated: at,
        status: "running",
    };
    let current = {
        revision: 1,
        run: { id: "run", conversation: "session", state: "running" },
        message: chunks[1],
        conversation: summary,
    };
    const listeners = new Set();
    window.feed = (id, values, done = false) => {
        const existing = chunks.find((entry) => entry.id === id);
        const message = {
            ...(existing || chunk(id, "assistant", "")),
            ...values,
            at: new Date().toISOString(),
        };
        if (existing) chunks[chunks.indexOf(existing)] = message;
        else chunks.push(message);
        current = {
            revision: current.revision + 1,
            run: {
                id: "run",
                conversation: "session",
                state: done ? "idle" : "running",
            },
            message,
            conversation: {
                ...summary,
                updated: message.at,
                status: done ? "idle" : "running",
            },
        };
        for (const listener of listeners) listener(current);
    };
    window.compact = (active) => {
        current = {
            ...current,
            compacting: active,
            revision: current.revision + 1,
        };
        for (const listener of listeners) listener(current);
    };
    window.renameSession = (title) => {
        summary.title = title;
        current = {
            ...current,
            revision: current.revision + 1,
            conversation: { ...current.conversation, title },
        };
        for (const listener of listeners) listener(current);
    };
    window.tokenUsage = { input: 800, output: 200, cached: 600, reasoning: 20 };
    const image = { id: "a".repeat(64), name: "sample.png" };
    const canvas = document.createElement("canvas");
    canvas.width = 800;
    canvas.height = 500;
    const drawing = canvas.getContext("2d");
    drawing.fillStyle = "#e8ded2";
    drawing.fillRect(0, 0, 800, 500);
    drawing.fillStyle = "#aebdb5";
    drawing.fillRect(400, 0, 400, 500);
    drawing.fillStyle = "#262626";
    drawing.font = "28px sans-serif";
    drawing.fillText("Image preview", 48, 72);
    const imageData = canvas.toDataURL();
    canvas.width = 240;
    canvas.height = 700;
    drawing.fillStyle = "#c4b8d6";
    drawing.fillRect(0, 0, 240, 700);
    const portraitData = canvas.toDataURL();
    window.arxDesktop = {
        chooseImages: async () => [image],
        readImage: async () => imageData,
        request: async (method, params) => {
            if (method === "network.state") return [];
            if (method === "generation.state")
                return {
                    library: { models: [], defaults: {}, revision: 0 },
                    jobs: [],
                    catalog: [],
                };
            if (method === "snapshot")
                return {
                    conversations: [summary],
                    settings,
                    views: {},
                    recovered: false,
                };
            if (method === "history")
                return {
                    chunks: structuredClone(chunks),
                    before: "",
                    more: false,
                };
            if (method === "chat.state") return structuredClone(current);
            if (method === "chat.context")
                return {
                    usage: window.tokenUsage,
                    known: true,
                    used: current.compacting ? 2400 : 18000,
                    limit: 1000000,
                    basis: "upper-bound",
                    parts: current.compacting
                        ? {
                              system: 400,
                              tools: 600,
                              messages: 1000,
                              summary: 400,
                          }
                        : {
                              system: 1200,
                              tools: 2800,
                              messages: 14000,
                              summary: 0,
                          },
                };
            if (method === "permissions.pending") return { pending: null };
            if (method === "local.state")
                return {
                    revision: 0,
                    hardware: { name: "Test Mac", memory: 16e9, supported: true },
                    models: [],
                    runtime: { state: "stopped", received: 0, total: 0 },
                };
            if (method === "deepseek.status")
                return {
                    configured: true,
                    connected: true,
                    models: [
                        {
                            id: "test",
                            contextWindow: 1000000,
                            name: "Test",
                            vision: true,
                            thinking: {
                                efforts: ["low"],
                                defaultEffort: "low",
                                canDisable: true,
                            },
                        },
                    ],
                };
            if (method === "web.image") {
                if (params.url.includes("portrait.png")) return portraitData;
                if (params.url.includes("landscape.png")) return imageData;
                if (params.url === "https://example.com/diagram.png") return imageData;
                throw Error("Image is unavailable");
            }
            if (method === "web.icon")
                return params.url.includes("example.com")
                    ? "data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mNk+A8AAQUBAScY42YAAAAASUVORK5CYII="
                    : "";
            if (method === "chat.send") {
                window.sentMessage = params;
                window.feed(
                    params.id,
                    {
                        role: "user",
                        text: params.text,
                        images: params.images,
                        status: "done",
                    },
                    true,
                );
                return { id: params.id, conversation: "session", state: "idle" };
            }
            if (method === "save_view") return {};
            if (method === "configure") return { ...settings, ...params };
            throw Error("Unexpected request: " + method);
        },
        onGeneration: () => () => {},
        onChat: (callback) => {
            listeners.add(callback);
            return () => listeners.delete(callback);
        },
        onPermission: () => () => {},
        onBackendStatus: (callback) => {
            callback({ state: "ready", error: null });
            return () => {};
        },
        onFullscreenChanged: () => () => {},
        signalReady: () => {},
    };
};
