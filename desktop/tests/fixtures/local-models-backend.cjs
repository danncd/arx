module.exports = function localModelsBackend() {
    const original = window.arxDesktop.request;
    window.searchCalls = [];
    window.deletions = [];
    const state = {
        revision: 1,
        hardware: { name: "Apple M2 Pro", memory: 16 * 2 ** 30, supported: true },
        runtime: { state: "ready", model: "local:test", received: 0, total: 0 },
        models: [
            {
                id: "local:test",
                name: "Qwen3 Test",
                imported: false,
                size: 1e9,
                received: 1e9,
                status: "loaded",
                info: {
                    id: "local:test",
                    name: "Qwen3 Test",
                    contextWindow: 24576,
                    trainedContext: 32768,
                    tools: true,
                    capabilitySource: "runtime",
                },
            },
        ],
    };
    const first = {
        id: "Qwen/Text",
        downloads: 100,
        pipeline_tag: "text-generation",
        tags: ["reasoning"],
    };
    const second = {
        id: "google/Vision",
        downloads: 90,
        pipeline_tag: "image-text-to-text",
    };
    const third = { id: "microsoft/Phi", downloads: 80, pipeline_tag: "text-generation" };
    window.arxDesktop.request = async (method, params) => {
        if (method === "local.state") return structuredClone(state);
        if (method === "local.action") {
            window.deletions.push(params);
            state.models = [];
            state.runtime = { state: "stopped", received: 0, total: 0 };
            state.revision++;
            return structuredClone(state);
        }
        if (method === "local.search") {
            window.searchCalls.push(params);
            if (params.query) {
                await new Promise((resolve) =>
                    setTimeout(resolve, params.query === "slow" ? 1000 : 20),
                );
                return { models: [{ ...first, id: "test/" + params.query }], next: "" };
            }
            if (params.cursor) return { models: [second, third], next: "" };
            return { models: [first, second], next: "page2" };
        }
        return original(method, params);
    };
};
