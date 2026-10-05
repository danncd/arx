import { LocalModelProgress } from "../../models/providers/local/LocalModelProgress";
import { formatSize } from "../../models/format";
import { CheckIcon, WarningCircleIcon } from "@phosphor-icons/react";
import { useState } from "react";
import type { GenerationModel } from "../../../../../contracts/wire.generated";
import { LocalModelIdentity } from "../../models/providers/local/LocalModelIdentity";
import { DeleteModelButton } from "../../models/providers/local/DeleteModelButton";
import type { Generation } from "../state/useGeneration";

const operations: Record<string, string> = {
    image: "Text to image",
    "image-edit": "Image editing",
    speech: "Text to speech",
    video: "Text to video",
    "image-to-video": "Image to video",
};

export function GenerationModelRow({
    model,
    generation,
    memory,
    supported,
    running,
}: {
    model: GenerationModel;
    generation: Generation;
    memory: number;
    supported: boolean;
    running: boolean;
}) {
    const [details, setDetails] = useState(false);
    const [busy, setBusy] = useState(false);
    const entry = generation.library.models.find((entry) => entry.model.id === model.id);
    const downloading = entry?.status === "downloading";
    const installed = entry?.status === "installed";
    const recommended = !model.experimental && supported && memory >= model.minimumMemory;
    const act = async (action: string) => {
        setBusy(true);
        try {
            await generation.action(action, model.id);
        } finally {
            setBusy(false);
        }
    };
    return (
        <div className="local-model-row">
            <div className="local-model-top">
                <LocalModelIdentity
                    name={model.name}
                    source={model.repository}
                    subtitle={`${model.company} · ${formatSize(model.size)}`}
                    badges={model.operations.map(
                        (operation) => operations[operation] || operation,
                    )}
                    hideTools
                />
                <div className="local-model-actions">
                    {installed ? (
                        <>
                            <span className="connection-status generation-installed">
                                <CheckIcon size={12} /> Installed
                            </span>
                            <DeleteModelButton
                                disabled={running || busy}
                                onDelete={() => act("remove")}
                            />
                        </>
                    ) : (
                        <button
                            type="button"
                            className="local-button"
                            disabled={!supported || busy}
                            onClick={() => void act(downloading ? "pause" : "download")}
                        >
                            {busy
                                ? "Updating…"
                                : downloading
                                  ? "Pause"
                                  : entry?.status === "paused"
                                    ? "Resume"
                                    : "Download"}
                        </button>
                    )}
                </div>
            </div>
            <button
                type="button"
                className={`local-fit ${recommended ? "" : "caution"}`}
                aria-expanded={details}
                onClick={() => setDetails(!details)}
            >
                {recommended ? <CheckIcon size={12} /> : <WarningCircleIcon size={12} />}
                {model.experimental
                    ? "Experimental"
                    : recommended
                      ? "Recommended"
                      : "Not recommended"}
            </button>
            {details && (
                <p className="local-detail">
                    {Math.round(model.minimumMemory / 2 ** 30)} GB memory recommended.{" "}
                    {model.category === "video"
                        ? "Video generation is slower and uses substantially more memory."
                        : "The chat model is released while generating."}
                </p>
            )}
            {(downloading || entry?.status === "paused") && (
                <LocalModelProgress
                    label={downloading ? "Downloading…" : "Paused"}
                    received={entry.received}
                    total={model.size}
                />
            )}
            {entry?.error && (
                <p role="alert" className="local-error">
                    {entry.error}
                </p>
            )}
        </div>
    );
}
