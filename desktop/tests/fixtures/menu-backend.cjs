module.exports = function menuBackend() {
    let settings = {
        version: 1,
        run: { model: "test", effort: "low" },
        directory: "/tmp",
        permissions: { mode: "folders", roots: ["/tmp"] },
    };
    window.arxDesktop = new Proxy(
        {
            onBackendStatus: (cb) => {
                cb({ state: "ready", error: null });
                return () => {};
            },
            request: async (method, params) => {
                if (method === "network.state") return [];
                if (method === "generation.state")
                    return {
                        library: { models: [], defaults: {}, revision: 0 },
                        jobs: [],
                        catalog: [],
                    };
                if (method === "snapshot")
                    return { conversations: [], settings, views: {}, recovered: false };
                if (method === "chat.state")
                    return {
                        revision: 0,
                        run: { id: "", conversation: "", state: "idle" },
                    };
                if (method === "chat.context")
                    return {
                        known: true,
                        used: 2000,
                        limit: 1000000,
                        parts: { system: 200, tools: 500, messages: 1300, summary: 0 },
                        usage: { input: 1000, output: 200, cached: 800 },
                    };
                if (method === "permissions.pending") return { pending: null };
                if (method === "deepseek.status")
                    return {
                        configured: true,
                        connected: true,
                        error: "",
                        models: [
                            {
                                id: "test",
                                name: "First model",
                                contextWindow: 1000000,
                                thinking: {
                                    efforts: ["low", "medium", "high"],
                                    defaultEffort: "low",
                                    defaultEnabled: true,
                                    canDisable: true,
                                },
                            },
                            {
                                id: "second",
                                name: "Second model",
                                contextWindow: 65536,
                            },
                        ],
                    };
                if (method === "local.state")
                    return {
                        revision: 0,
                        hardware: {
                            name: "Apple M2 Pro",
                            memory: 16 * 2 ** 30,
                            supported: true,
                        },
                        models: [],
                        runtime: { state: "stopped", received: 0, total: 0 },
                    };
                if (method === "configure") {
                    if (window.failSelection) throw Error("Save failed");
                    await new Promise((r) => setTimeout(r, 300));
                    settings = { ...settings, ...params };
                    return settings;
                }
                if (method === "permissions.configure") {
                    settings = { ...settings, permissions: params };
                    return settings;
                }
                if (method === "save_view") return {};
                if (method === "local.search")
                    return {
                        models: [{ id: "test/Example-GGUF", downloads: 100 }],
                        next: "",
                    };
                if (method === "local.repository")
                    return {
                        id: params.id,
                        revision: "test",
                        gated: false,
                        variants: Array.from({ length: 20 }, (_, i) => ({
                            id: String(i),
                            name: "model-" + i + ".gguf",
                            quantization: "Q" + i,
                            size: 1000000000,
                            fit: {
                                label: "Recommended",
                                reason: "",
                                tone: "",
                                context: 8192,
                            },
                        })),
                    };
                throw new Error(`Unexpected fixture request: ${method}`);
            },
        },
        { get: (o, p) => o[p] || (() => () => {}) },
    );
};
