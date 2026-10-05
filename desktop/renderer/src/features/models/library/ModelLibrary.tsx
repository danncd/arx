import { GenerationModelRow } from "../../generation/models/GenerationModelRow";
import { useGeneration } from "../../generation/state/useGeneration";
import { LocalModelRow } from "../providers/local/LocalModelRow";
import type { LocalModelsState } from "../providers/local/useLocalModels";

export function ModelLibrary({
    local,
    running,
}: {
    local: LocalModelsState;
    running: boolean;
}) {
    const generation = useGeneration();
    const empty = !local.models.length && !generation.library.models.length;
    return (
        <>
            {empty && !generation.error && (
                <div className="local-empty" role="status">
                    {generation.loading ? "Loading models…" : "No models downloaded."}
                </div>
            )}
            {local.models.map((model) => (
                <LocalModelRow
                    key={model.id}
                    model={model}
                    local={local}
                    running={running}
                />
            ))}
            {generation.library.models.map(({ model }) => (
                <GenerationModelRow
                    key={model.id}
                    model={model}
                    generation={generation}
                    memory={local.hardware.memory}
                    supported={local.hardware.supported}
                    running={running}
                />
            ))}
            {generation.error && (
                <p className="local-error" role="alert">
                    {generation.error}
                </p>
            )}
        </>
    );
}
