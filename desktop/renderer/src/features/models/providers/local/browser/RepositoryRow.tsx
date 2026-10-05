import { LocalModelIdentity } from "../LocalModelIdentity";
import { modelBadges } from "../capabilities";
import { CheckIcon, WarningCircleIcon } from "@phosphor-icons/react";
import { useEffect, useState } from "react";
import { Select } from "../../../../../ui/select/Select";
import type { LocalModelsState } from "../useLocalModels";
import { formatSize } from "../../../format";
import {
    type Repository,
    type SearchModel,
} from "../../../../../../../contracts/wire.generated";
import { LocalModelRow } from "../LocalModelRow";

export function RepositoryRow({
    model,
    local,
    recommended,
    running,
    onMatch,
}: {
    model: SearchModel;
    local: LocalModelsState;
    recommended: boolean;
    running: boolean;
    onMatch: (id: string, match: boolean | undefined) => void;
}) {
    const [repository, setRepository] = useState<Repository>();
    const [variant, setVariant] = useState("");
    const [error, setError] = useState("");
    const [busy, setBusy] = useState(false);
    const [details, setDetails] = useState(false);
    useEffect(() => {
        let active = true;
        void window.arxDesktop
            ?.request("local.repository", { id: model.id })
            .then((result) => {
                if (!active) return;
                setRepository(result);
                setVariant(
                    (
                        result.variants.find((item) => item.quantization === "Q4_K_M") ||
                        result.variants.find((item) =>
                            item.quantization.startsWith("Q4"),
                        ) ||
                        result.variants[0]
                    )?.id || "",
                );
            })
            .catch((error) => {
                if (active) setError(error.message);
            });
        return () => {
            active = false;
        };
    }, [model.id]);
    const selected = repository?.variants.find((item) => item.id === variant);
    const fit = selected?.fit || {
        label: "Checking compatibility",
        reason: "",
        tone: "caution",
    };
    const matches = selected
        ? fit.label === "Recommended"
        : repository || error
          ? false
          : undefined;
    useEffect(() => {
        onMatch(model.id, matches);
    }, [model.id, matches, onMatch]);
    if (recommended && !matches) return null;
    const installed = local.models.find(
        (item) => item.repository === model.id && item.name === selected?.name,
    );
    if (installed)
        return (
            <>
                <LocalModelRow
                    model={installed}
                    local={local}
                    running={running}
                    visionSupported={modelBadges(model).includes("Vision")}
                />
                {selected?.projector && !installed.projector && (
                    <button
                        type="button"
                        className="local-button"
                        disabled={busy || running || installed.status !== "installed"}
                        onClick={() => {
                            setBusy(true);
                            void window.arxDesktop
                                ?.request("local.download", {
                                    repository: model.id,
                                    variant,
                                })
                                .catch((error) => setError(error.message))
                                .finally(() => setBusy(false));
                        }}
                    >
                        {busy
                            ? "Starting…"
                            : installed.status === "loaded"
                              ? "Unload to add vision"
                              : "Download vision support"}
                    </button>
                )}
                {error && (
                    <p role="alert" className="local-error">
                        {error}
                    </p>
                )}
            </>
        );
    return (
        <div className="local-model-row">
            <div className="local-model-top">
                <LocalModelIdentity
                    name={model.id.split("/").pop() || model.id}
                    source={model.id}
                    badges={modelBadges(model).map((badge) =>
                        badge === "Vision" && repository && !selected?.projector
                            ? "Vision · companion unavailable"
                            : badge,
                    )}
                    subtitle={`${model.id.split("/")[0]}${selected ? ` · ${formatSize(selected.size)}` : repository?.gated ? " · Access required" : repository ? " · No supported GGUF files" : error ? "" : " · Checking files…"}`}
                />
                <button
                    type="button"
                    className="local-button"
                    disabled={!selected || busy || !local.hardware.supported}
                    onClick={() => {
                        if (!selected) return;
                        setBusy(true);
                        setError("");
                        void window.arxDesktop
                            ?.request("local.download", {
                                repository: model.id,
                                variant,
                            })
                            .catch((error) => setError(error.message))
                            .finally(() => setBusy(false));
                    }}
                >
                    {busy ? "Starting…" : "Download"}
                </button>
            </div>
            {selected && (
                <div className="local-variant-row">
                    <button
                        type="button"
                        className={`local-fit ${fit.tone}`}
                        aria-expanded={details}
                        onClick={() => setDetails(!details)}
                    >
                        {fit.tone ? (
                            <WarningCircleIcon size={12} />
                        ) : (
                            <CheckIcon size={12} />
                        )}
                        {fit.label}
                    </button>
                    <Select
                        label="Model variant"
                        value={variant}
                        choices={repository!.variants.map((item) => ({
                            value: item.id,
                            label: `${item.quantization || item.name} · ${formatSize(item.size)}`,
                        }))}
                        onChange={setVariant}
                    />
                </div>
            )}
            {details && (
                <p className="local-detail">
                    {fit.reason}
                    {selected?.projector ? " Includes vision projector." : ""}
                </p>
            )}
            {repository?.gated && (
                <button
                    type="button"
                    className="text-button"
                    onClick={() =>
                        void window.arxDesktop?.openExternal(
                            `https://huggingface.co/${model.id}`,
                        )
                    }
                >
                    View access requirements
                </button>
            )}
            {error && (
                <p className="local-error" role="alert">
                    {error}
                </p>
            )}
        </div>
    );
}
