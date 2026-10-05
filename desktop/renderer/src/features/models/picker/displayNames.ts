import type { ModelInfo } from "../../../../../contracts/wire.generated";

const quantization = /[._ -]((?:I?Q\d(?:_[A-Z0-9]+)*|BF16|F16|F32))$/i;

export function modelDisplayNames(models: ModelInfo[]): ModelInfo[] {
    const labels = models.map((model) => {
        if (!model.id.startsWith("local:"))
            return { model, name: model.name, variant: "" };
        const filename = model.name
            .replace(/\.gguf$/i, "")
            .replace(/-\d{5}-of-\d{5}$/i, "");
        const variant = filename.match(quantization)?.[1] || "";
        const name = filename
            .replace(quantization, "")
            .replace(/[-_]+/g, " ")
            .replace(/\s+/g, " ")
            .trim();
        return { model, name: name || model.name, variant };
    });
    const counts = new Map<string, number>();
    for (const { name } of labels) counts.set(name, (counts.get(name) || 0) + 1);
    return labels.map(({ model, name, variant }) => ({
        ...model,
        name: (counts.get(name) || 0) > 1 && variant ? `${name} · ${variant}` : name,
    }));
}
