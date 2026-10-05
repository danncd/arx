import type { RunSettings } from "../../../../../contracts/wire.generated";
import type { DiscoveredModel } from "../../../../../contracts/wire.generated";

export function selectRun(models: DiscoveredModel[], run: RunSettings): RunSettings {
    const model = models.find((item) => item.id === run.model) || models[0];
    if (!model) return run;
    const thinking = model.thinking;
    const supported =
        thinking?.efforts.includes(run.effort) ||
        (thinking?.canDisable && run.effort === "none");
    return {
        provider: modelProvider(model.id, model.provider),
        model: model.id,
        effort: supported ? run.effort : thinking?.defaultEffort || "",
    };
}

export function modelProvider(id: string, explicit?: string) {
    if (id.startsWith("local:")) return "local";
    if (id.startsWith("network:")) return "network";
    return explicit === "local" || explicit === "network" ? explicit : "deepseek";
}
export function modelAvailability(
    model: DiscoveredModel | undefined,
    deepseekConnected: boolean,
    runtime: { state: string; model?: string },
): "ready" | "requiresLoad" | "unavailable" {
    if (!model) return "unavailable";
    switch (modelProvider(model.id, model.provider)) {
        case "local":
            return runtime.state === "ready" && runtime.model === model.id
                ? "ready"
                : "requiresLoad";
        case "network":
            return "ready";
        default:
            return deepseekConnected ? "ready" : "unavailable";
    }
}
