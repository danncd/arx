import type {
    ModelInfo,
    ThinkingCapabilities,
} from "../../../../../contracts/wire.generated";
import { modelDisplayNames } from "./displayNames";
import { ModelMenu } from "./ModelMenu";
import { ThinkingControls } from "./ThinkingControls";

export function ModelPicker({
    models,
    unloadedModels = [],
    capabilities,
    catalogueError,
    model,
    effort,
    disabled,
    onModel,
    onEffort,
    onSettings,
}: {
    models: ModelInfo[];
    unloadedModels?: string[];
    capabilities?: ThinkingCapabilities;
    catalogueError: string;
    model: string;
    effort: string;
    disabled: boolean;
    onModel: (model: string) => Promise<boolean | undefined>;
    onEffort: (effort: string) => Promise<void>;
    onSettings: () => void;
}) {
    if (catalogueError)
        return (
            <button
                type="button"
                className="model-setup-button"
                onClick={onSettings}
                disabled={disabled}
            >
                Choose model
            </button>
        );
    if (models.length === 0) return null;
    const selected = models.find((item) => item.id === model);
    return (
        <ModelMenu
            value={model}
            options={modelDisplayNames(models).map((item) => ({
                value: item.id,
                label: item.name,
                provider: "",
                unloaded: unloadedModels.includes(item.id),
            }))}
            disabled={disabled}
            onChange={onModel}
        >
            <ThinkingControls
                key={model}
                capabilities={capabilities}
                effort={effort}
                disabled={disabled || !selected}
                onChange={onEffort}
            />
        </ModelMenu>
    );
}
