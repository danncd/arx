import { LocalModelIdentity } from "./LocalModelIdentity";
import { modelBadges } from "./capabilities";
import { DeleteModelButton } from "./DeleteModelButton";
import { LocalModelContext } from "./LocalModelContext";
import { PauseIcon, PlayIcon, XIcon } from "@phosphor-icons/react";
import { useState } from "react";
import type { LocalModel } from "../../../../../../contracts/wire.generated";
import { formatSize } from "../../format";
import type { LocalModelsState } from "./useLocalModels";
import { LocalModelProgress } from "./LocalModelProgress";

export function LocalModelRow({
    model,
    local,
    running,
    visionSupported = false,
}: {
    model: LocalModel;
    local: LocalModelsState;
    running: boolean;
    visionSupported?: boolean;
}) {
    const [busy, setBusy] = useState(false);
    const act = async (action: string, deleteFiles = false) => {
        setBusy(true);
        try {
            await local.action(action, model.id, deleteFiles);
        } catch {
        } finally {
            setBusy(false);
        }
    };
    const loading =
        local.runtime.model === model.id &&
        ["installing", "loading"].includes(local.runtime.state);
    const downloading = ["downloading", "verifying"].includes(model.status);
    return (
        <div className="local-model-row">
            <div className="local-model-top">
                <LocalModelIdentity
                    name={model.name}
                    source={model.repository || model.name}
                    subtitle={`${model.imported ? "Imported" : model.quantization || "GGUF"} · ${formatSize(model.size)}`}
                    badges={[
                        ...modelBadges(undefined, model.info),
                        ...(visionSupported && !model.info.vision
                            ? [
                                  model.projector
                                      ? "Vision · runtime unavailable"
                                      : "Vision · companion required",
                              ]
                            : []),
                    ]}
                    tools={model.info.tools}
                    verified={model.info.capabilitySource === "runtime"}
                />
                <div className="local-model-actions">
                    {downloading && (
                        <button
                            type="button"
                            className="icon-button"
                            title="Pause download"
                            aria-label="Pause download"
                            disabled={busy}
                            onClick={() => void act("pause")}
                        >
                            <PauseIcon size={15} />
                        </button>
                    )}
                    {downloading && (
                        <button
                            type="button"
                            className="icon-button"
                            title="Cancel download"
                            aria-label="Cancel download"
                            disabled={busy}
                            onClick={() => void act("cancel")}
                        >
                            <XIcon size={15} />
                        </button>
                    )}
                    {["paused", "failed"].includes(model.status) && (
                        <button
                            type="button"
                            className="icon-button"
                            title="Resume download"
                            aria-label="Resume download"
                            disabled={busy}
                            onClick={() => void act("resume")}
                        >
                            <PlayIcon size={15} />
                        </button>
                    )}
                    {model.status === "installed" && (
                        <button
                            type="button"
                            className="local-button"
                            disabled={
                                busy ||
                                running ||
                                ["loading", "installing"].includes(local.runtime.state)
                            }
                            onClick={() => {
                                setBusy(true);
                                void local
                                    .ensure(model.id)
                                    .catch(() => {})
                                    .finally(() => setBusy(false));
                            }}
                        >
                            Load
                        </button>
                    )}
                    {model.status === "loaded" && (
                        <button
                            type="button"
                            className="local-button"
                            disabled={busy || running}
                            onClick={() => void act("unload")}
                        >
                            Unload
                        </button>
                    )}
                    {loading && (
                        <button
                            type="button"
                            className="icon-button"
                            aria-label="Cancel loading"
                            title="Cancel loading"
                            disabled={busy}
                            onClick={() => void act("unload")}
                        >
                            <XIcon size={15} />
                        </button>
                    )}
                    {!loading && !downloading && (
                        <DeleteModelButton
                            disabled={busy || running}
                            onDelete={() => act("remove", true)}
                        />
                    )}
                </div>
            </div>
            <LocalModelContext model={model} />
            {(downloading || model.status === "paused") && (
                <LocalModelProgress
                    label={
                        model.status === "paused"
                            ? "Paused"
                            : model.status === "verifying"
                              ? "Verifying…"
                              : "Downloading…"
                    }
                    received={model.received}
                    total={model.size}
                />
            )}
            {loading && (
                <LocalModelProgress
                    label={
                        local.runtime.state === "installing"
                            ? "Installing runtime…"
                            : "Loading model…"
                    }
                    received={local.runtime.total ? local.runtime.received : undefined}
                    total={local.runtime.total || undefined}
                />
            )}
            {model.error && (
                <p className="local-error" role="alert">
                    {model.error}
                </p>
            )}
        </div>
    );
}
