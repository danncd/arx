import type { SearchModel } from "../../../../../../contracts/wire.generated";
import type { DiscoveredModel } from "../../../../../../contracts/wire.generated";

export function catalogCapabilities(model: SearchModel) {
    const tags = new Set(model.tags || []);
    const vision =
        model.pipeline_tag === "image-text-to-text" || tags.has("image-text-to-text");
    const text =
        vision ||
        ["text-generation", "conversational"].includes(model.pipeline_tag || "") ||
        tags.has("text-generation");
    const thinking = ["reasoning", "thinking", "chain-of-thought"].some((tag) =>
        tags.has(tag),
    );
    return { vision, text, thinking };
}

export function modelBadges(model?: SearchModel, info?: DiscoveredModel) {
    const capabilities = model
        ? catalogCapabilities(model)
        : { text: true, vision: !!info?.vision, thinking: !!info?.thinking };
    const badges = [];
    if (capabilities.text) badges.push("Text");
    if (capabilities.vision) badges.push("Vision");
    if (capabilities.thinking) badges.push("Thinking");
    if (!badges.length) badges.push("Capabilities unknown");
    return badges;
}

export function modelBrand(source: string) {
    const families: [RegExp, string][] = [
        [/deepseek/i, "deepseek-ai"],
        [/qwen/i, "Qwen"],
        [/gemma/i, "google"],
        [/llama/i, "meta-llama"],
        [/mistral|mixtral|ministral|devstral/i, "mistralai"],
        [/(?:^|[\/_.-])phi[\d_.-]/i, "microsoft"],
    ];
    return families.find(([pattern]) => pattern.test(source))?.[1];
}
