import type { InitialState } from "../features/sessions/types";
import type { Requests } from "../../../contracts/requests.generated";
import type { GenerationEvent, GenerationState } from "../../../contracts/wire.generated";
import { initialSidebarLayout } from "../shell/useSidebar";
import { previews } from "./fixtures/sessions";
import catalogData from "../../../../runtimes/generation/catalog.json";

export function installScenario(): InitialState {
    const selected = previews[0];
    if (!selected) throw new Error("Preview scenarios are missing");
    const snapshot: InitialState["snapshot"] = {
        conversations: previews.map((s) => ({
            id: s.id,
            title: s.title,
            updated: s.updatedAt,
            status: s.status,
        })),
        settings: {
            version: 1,
            run: { model: "preview", effort: "medium" },
            directory: "",
            local_idle_minutes: 5,
            auto_continue: true,
            permissions: { mode: "ask", roots: [] },
        },
        views: {},
        recovered: false,
    };
    const model = {
        id: "preview",
        name: "Preview model",
        vision: true,
        contextWindow: 65536,
        maxOutputTokens: 8192,
        thinking: {
            efforts: ["medium"],
            defaultEffort: "medium",
            defaultEnabled: true,
            canDisable: true,
            source: "preview",
        },
    };
    const catalog = catalogData.map((model) => ({
        ...model,
        size: model.files.reduce((total, file) => total + file.size, 0),
    })) as GenerationState["catalog"];
    let generation: GenerationState = {
        library: {
            revision: 1,
            models: catalog
                .slice(0, 3)
                .map((model) => ({ model, status: "installed", received: model.size })),
            defaults: {},
        },
        jobs: [],
        catalog,
    };
    const listeners = new Set<(event: GenerationEvent) => void>();
    const noSubscription = () => () => {};
    window.arxDesktop = {
        request: async <K extends keyof Requests>(
            method: K,
            params: Requests[K]["params"],
        ): Promise<Requests[K]["result"]> => {
            let result: unknown;
            switch (method) {
                case "snapshot":
                    result = snapshot;
                    break;
                case "history": {
                    const id = (params as Requests["history"]["params"]).conversation;
                    const s = previews.find((s) => s.id === id);
                    result = {
                        chunks:
                            s?.messages.map((m) => ({
                                conversation: id,
                                id: m.id,
                                role: m.role,
                                text: m.text,
                                reasoning: "",
                                offset: 0,
                                reasoning_offset: 0,
                                tools: null,
                                usage: null,
                                status: "done",
                                failed: false,
                                reason: "",
                                at: s.updatedAt,
                            })) ?? [],
                        before: "",
                        more: false,
                    };
                    break;
                }
                case "configure":
                    snapshot.settings = {
                        ...snapshot.settings,
                        ...(params as Requests["configure"]["params"]),
                    };
                    result = snapshot.settings;
                    break;
                case "save_view": {
                    const p = params as Requests["save_view"]["params"];
                    snapshot.views[p.key] = p.value;
                    result = {};
                    break;
                }
                case "chat.state":
                    result = {
                        revision: 0,
                        run: { id: "", conversation: "", state: "idle" },
                    };
                    break;
                case "chat.context":
                    result = {
                        known: false,
                        used: 0,
                        limit: 0,
                        basis: "",
                        parts: { system: 0, tools: 0, messages: 0, summary: 0 },
                        usage: { input: 0, output: 0, cached: 0, reasoning: 0 },
                    };
                    break;
                case "permissions.pending":
                    result = { pending: null };
                    break;
                case "deepseek.status":
                case "deepseek.refresh":
                    result = {
                        configured: true,
                        connected: true,
                        models: [model],
                        error: "",
                    };
                    break;
                case "local.state":
                    result = {
                        revision: 0,
                        hardware: {
                            name: "Preview Mac",
                            memory: 16 * 2 ** 30,
                            supported: true,
                        },
                        models: [],
                        runtime: { state: "idle", received: 0, total: 0 },
                    };
                    break;
                case "network.state":
                case "network.scan":
                case "mcp.state":
                case "skills.state":
                    result = [];
                    break;
                case "generation.state":
                    result = generation;
                    break;
                case "generation.configure": {
                    const p = params as Requests["generation.configure"]["params"];
                    generation = {
                        ...generation,
                        library: {
                            ...generation.library,
                            revision: generation.library.revision + 1,
                            defaults: {
                                ...generation.library.defaults,
                                [p.category]: p.id,
                            },
                        },
                    };
                    listeners.forEach((l) => l({ library: generation.library }));
                    result = generation.library;
                    break;
                }
                case "generation.action": {
                    const p = params as Requests["generation.action"]["params"];
                    const model = catalog.find((m) => m.id === p.id);
                    if (model) {
                        const models = generation.library.models.filter(
                            (m) => m.model.id !== p.id,
                        );
                        if (p.action !== "remove")
                            models.push({
                                model,
                                status: "installed",
                                received: model.size,
                            });
                        generation = {
                            ...generation,
                            library: {
                                ...generation.library,
                                revision: generation.library.revision + 1,
                                models,
                            },
                        };
                        listeners.forEach((l) => l({ library: generation.library }));
                    }
                    result = generation.library;
                    break;
                }
                default:
                    throw new Error(
                        "This preview operation requires the desktop backend",
                    );
            }
            return result as Requests[K]["result"];
        },
        onBackendStatus: (callback) => {
            callback({ state: "ready", error: null });
            return () => {};
        },
        onGeneration: (callback) => {
            listeners.add(callback);
            return () => {
                listeners.delete(callback);
            };
        },
        onChat: noSubscription,
        onPermission: noSubscription,
        onLocal: noSubscription,
        onFullscreenChanged: noSubscription,
        chooseDirectory: async () => null,
        chooseModel: async () => null,
        chooseImages: async () => [],
        readImage: async () => "",
        saveMedia: async () => false,
        backendInfo: async () => ({ version: "preview", execution: "idle" }),
        restartBackend: async () => ({ state: "ready", error: null }),
        copy: async (text) => {
            await navigator.clipboard.writeText(text);
        },
        openExternal: async () => {},
        beginWindowDrag: () => {},
        moveWindowDrag: () => {},
        endWindowDrag: () => {},
        doubleClickTitleBar: async () => {},
        signalReady: () => {},
    };
    return {
        snapshot,
        selected: selected.id,
        session: selected,
        before: "",
        more: false,
        layout: initialSidebarLayout,
    };
}
