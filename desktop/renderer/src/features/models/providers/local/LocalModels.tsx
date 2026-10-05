import {
    CaretRightIcon,
    CheckIcon,
    CpuIcon,
    FileArrowUpIcon,
    PlusIcon,
} from "@phosphor-icons/react";
import { useId, useState } from "react";
import type { LocalModelsState } from "./useLocalModels";
import { ModelLibrary } from "../../library/ModelLibrary";
import { Select } from "../../../../ui/select/Select";
import "./local-models.css";

export function LocalModels({
    local,
    onBrowse,
    running,
    idleMinutes,
    onIdleMinutesChange,
}: {
    local: LocalModelsState;
    onBrowse: () => void;
    running: boolean;
    idleMinutes: number;
    onIdleMinutesChange: (minutes: number) => void;
}) {
    const [expanded, setExpanded] = useState(false);
    const [error, setError] = useState("");
    const id = useId();
    const importModel = async () => {
        setError("");
        try {
            const path = await window.arxDesktop?.chooseModel();
            if (path) await window.arxDesktop?.request("local.import", { path });
        } catch (error) {
            setError(error instanceof Error ? error.message : "Could not import model");
        }
    };
    return (
        <div className="provider-section" role="group" aria-label="Local models">
            <h4 className="provider-heading">
                <button
                    type="button"
                    className="provider-toggle"
                    aria-expanded={expanded}
                    aria-controls={id}
                    onClick={() => setExpanded(!expanded)}
                >
                    <span className="provider-identity">
                        <CpuIcon className="provider-icon" />
                        <strong>Local models</strong>
                    </span>
                    <span className="provider-tail">
                        {local.runtime.state === "ready" && (
                            <span className="connection-status">
                                <CheckIcon size={12} />
                                Connected
                            </span>
                        )}
                        <CaretRightIcon className="provider-chevron" size={12} />
                    </span>
                </button>
            </h4>
            <div
                className={`provider-body${expanded ? " expanded" : ""}`}
                id={id}
                inert={!expanded}
            >
                <div>
                    <div className="provider-content">
                        <div className="local-machine">
                            {local.hardware.name}
                            {local.hardware.memory > 0 &&
                                ` · ${Math.round(local.hardware.memory / 2 ** 30)} GB unified memory`}
                        </div>
                        <div className="local-idle-setting">
                            <span>Unload after</span>
                            <Select
                                label="Unload local model after"
                                value={String(idleMinutes)}
                                choices={[
                                    { value: "5", label: "5 minutes" },
                                    { value: "10", label: "10 minutes" },
                                    { value: "15", label: "15 minutes" },
                                ]}
                                onChange={(value) => onIdleMinutesChange(Number(value))}
                            />
                        </div>
                        <div className="local-library-heading">
                            <span>Your models</span>
                            <div className="local-model-actions">
                                <button
                                    type="button"
                                    className="local-button"
                                    onClick={onBrowse}
                                >
                                    Browse models
                                    <PlusIcon size={13} />
                                </button>
                                <button
                                    type="button"
                                    className="local-button local-import"
                                    aria-label="Import model file"
                                    title="Import model file"
                                    onClick={() => void importModel()}
                                >
                                    <FileArrowUpIcon size={14} />
                                </button>
                            </div>
                        </div>
                        <ModelLibrary local={local} running={running} />
                        {(error || local.error) && (
                            <p className="local-error" role="alert">
                                {error || local.error}
                            </p>
                        )}
                    </div>
                </div>
            </div>
        </div>
    );
}
