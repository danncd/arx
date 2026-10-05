import { CheckIcon } from "@phosphor-icons/react";
import type { LocalModel } from "../../../../../../contracts/wire.generated";

const contextLabel = (value: number) =>
    value >= 1024 ? `${Number((value / 1024).toFixed(1))}K` : String(value);

export function LocalModelContext({ model }: { model: LocalModel }) {
    if (!model.info.contextWindow) return null;
    const loaded = model.status === "loaded";
    return (
        <div className="model-context" title={model.info.contextReason}>
            {loaded && (
                <>
                    <span className="loaded">
                        <CheckIcon size={12} />
                        Loaded
                    </span>
                    <span className="divider">·</span>
                </>
            )}
            <span>
                {contextLabel(model.info.contextWindow)} {loaded ? "active" : "target"}{" "}
                context
            </span>
            {!!model.info.trainedContext && (
                <>
                    <span className="divider">·</span>
                    <span>{contextLabel(model.info.trainedContext)} supported</span>
                </>
            )}
        </div>
    );
}
